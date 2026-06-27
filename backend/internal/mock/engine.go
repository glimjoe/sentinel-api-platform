// Package mock — engine (Phase 2 M1).
//
// The engine is the public entry point for /mock/:projectSlug/*path. It wires
// the matcher, extractor, and varbag behind a single Serve(*gin.Context) method.
//
// Per-request flow (plan §6.6 + ADR-0007):
//  1. Extract projectSlug from the URL and the X-Mock-Session header.
//  2. Look up the project by slug (404 if missing).
//  3. Load enabled rules for the project + load APIs for path/method mapping.
//  4. Filter candidates by method and path (the matcher scores per-field but
//     does not check method/path itself).
//  5. matcher.PickBest scores; engine maps ErrBadMatchJSON → 422, no match → 404.
//  6. extractor.Apply writes any `{{var}}` sources to the varbag.
//  7. varbag.Substitute rewrites the response body in place.
//  8. Apply response headers, status, optional delay.
//  9. TODO(Phase 2 M2): record a mock_hits row + increment hit_count.
package mock

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/oklog/ulid/v2"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/errs"
)

// ProjectFinder is the project-lookup contract the engine needs.
type ProjectFinder interface {
	FindBySlug(ctx context.Context, slug string) (*model.Project, error)
}

// RuleFinder is the rule-list contract the engine needs.
type RuleFinder interface {
	ListByProject(ctx context.Context, projectID string) ([]*model.MockRule, error)
}

// APIFinder is the API-list contract the engine needs to map rules to their
// parent API's method+path.
type APIFinder interface {
	ListByProject(ctx context.Context, projectID string) ([]*model.API, error)
}

// HitRecorder is the (currently unused) mock_hits write contract. The engine
// TODO-comment calls into this; M2 wires the implementation.
type HitRecorder interface {
	Record(ctx context.Context, hit *model.MockHit) error
}

// Engine is the public /mock/:slug/*path handler. Stateless across requests
// beyond the dependencies it holds.
type Engine struct {
	projects  ProjectFinder
	apis      APIFinder
	rules     RuleFinder
	varbag    VarBag
	matcher   *Matcher
	extractor *Extractor
	// recorder is nil for M1 (no hit recording yet). M2 will inject a real
	// implementation backed by MockHitRepo.
	recorder HitRecorder
}

// NewEngine constructs an Engine. The recorder argument is optional for M1 —
// pass nil to skip hit recording.
func NewEngine(
	projects ProjectFinder,
	apis APIFinder,
	rules RuleFinder,
	varbag VarBag,
	matcher *Matcher,
	extractor *Extractor,
	recorder HitRecorder,
) *Engine {
	return &Engine{
		projects:  projects,
		apis:      apis,
		rules:     rules,
		varbag:    varbag,
		matcher:   matcher,
		extractor: extractor,
		recorder:  recorder,
	}
}

// Serve handles one /mock/:projectSlug/*path request.
func (e *Engine) Serve(c *gin.Context) {
	ctx := c.Request.Context()
	slug := c.Param("projectSlug")
	if slug == "" {
		// Should be impossible because of the route pattern, but guard anyway.
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing projectSlug"})
		return
	}
	sessionID := e.sessionID(c)

	// 1. Project lookup.
	project, err := e.projects.FindBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// 2. Load candidates. We need both rules (for matching) and APIs (for
	// method+path mapping). Two queries is fine for M1; the engine is the only
	// reader and this is a single-digit-millisecond path.
	rules, err := e.rules.ListByProject(ctx, project.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	apis, err := e.apis.ListByProject(ctx, project.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// 3. Build apiID → API map and filter rules by method+path.
	// The route pattern is /mock/:projectSlug/*path — gin puts the wildcard
	// into Param("path") (without the leading slash). Re-prepend "/" so
	// "pet/findByStatus" → "/pet/findByStatus" to match the API.path column.
	requestPath := "/" + strings.TrimPrefix(c.Param("path"), "/")
	apiByID := make(map[string]*model.API, len(apis))
	for _, a := range apis {
		apiByID[a.ID] = a
	}
	candidates := make([]*model.MockRule, 0, len(rules))
	for _, r := range rules {
		api, ok := apiByID[r.APIID]
		if !ok {
			continue // orphaned rule (FK enforced but defensive)
		}
		if api.Method != c.Request.Method {
			continue
		}
		if api.Path != requestPath {
			continue
		}
		candidates = append(candidates, r)
	}

	// 4. Match.
	scored, err := e.matcher.PickBest(MatchRequest{
		Method:  c.Request.Method,
		Path:    requestPath,
		Query:   c.Request.URL.Query(),
		Headers: c.Request.Header,
		Body:    readBody(c),
	}, candidates)
	if err != nil {
		if errors.Is(err, ErrBadMatchJSON) {
			// Per architect review: bad match_json → 422 (not silent skip).
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	if scored == nil {
		// 404 with structured error so the MockConsole can show a useful message.
		c.JSON(http.StatusNotFound, gin.H{
			"error":  "no mock rule matched the request",
			"slug":   slug,
			"path":   requestPath,
			"method": c.Request.Method,
		})
		return
	}

	// 5. Optional delay before responding (drives the M2 timing graph).
	if scored.Rule.DelayMs > 0 {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(scored.Rule.DelayMs) * time.Millisecond):
		}
	}

	// 6. Extract → write to varbag. This happens BEFORE substitute so the same
	// response can both feed and consume the bag.
	if len(scored.Rule.ExtractorJSON) > 0 {
		if err := e.extractor.Apply(ctx, slug, sessionID,
			scored.Rule.ExtractorJSON, scored.Rule.ResponseBodyJSON, e.varbag); err != nil {
			// Extraction errors are config errors (bad extractor_json). Log via
			// gin's error field and continue — the user still gets a response.
			_ = c.Error(err)
		}
	}

	// 7. Substitute {{var}} in the response body.
	body, err := e.substituteBody(ctx, slug, sessionID, scored.Rule.ResponseBodyJSON)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// 8. Apply response headers.
	var respHeaders map[string]string
	if len(scored.Rule.ResponseHeadersJSON) > 0 {
		_ = json.Unmarshal(scored.Rule.ResponseHeadersJSON, &respHeaders)
		for k, v := range respHeaders {
			c.Header(k, v)
		}
	}
	c.Header("X-Mock-Session", sessionID)
	c.Header("X-Mock-Match-Reason", scored.Reason)
	c.Header("X-Mock-Rule-Id", scored.Rule.ID)

	// 9. Write response.
	status := scored.Rule.ResponseStatus
	if status == 0 {
		status = http.StatusOK
	}
	c.Data(status, "application/json; charset=utf-8", body)

	// 10. TODO(Phase 2 M2): record a mock_hits row + bump hit_count.
	// The recorder dependency is nil in M1; this is a no-op.
	if e.recorder != nil {
		_ = e.recorder.Record(ctx, &model.MockHit{
			ID:             ulid.Make().String(),
			MockRuleID:     scored.Rule.ID,
			RequestMethod:  c.Request.Method,
			RequestPath:    requestPath,
			ResponseStatus: status,
			DurationMs:     0, // TODO: measure in M2
			CreatedAt:      time.Now().UTC(),
		})
	}
}

// sessionID returns the X-Mock-Session header value, generating a fresh ULID
// if absent. Always non-empty.
func (e *Engine) sessionID(c *gin.Context) string {
	s := c.GetHeader("X-Mock-Session")
	if s == "" {
		s = ulid.Make().String()
		c.Header("X-Mock-Session-Generated", "true")
	}
	return s
}

// readBody returns up to 64 KB of the request body for matching. The body is
// not closed here because gin owns its lifecycle; the http server will close
// it when the handler returns. We do drain a small buffer so subsequent reads
// (e.g. for logging) see a consistent position.
func readBody(c *gin.Context) []byte {
	if c.Request.Body == nil {
		return nil
	}
	const maxBodyForMatching = 64 * 1024
	buf := make([]byte, maxBodyForMatching)
	n, _ := c.Request.Body.Read(buf)
	if n == 0 {
		return nil
	}
	// Drain whatever else is there so the connection can be reused. Discard
	// the result — we already have what we need.
	_, _ = io.Copy(io.Discard, c.Request.Body)
	return buf[:n]
}

// substituteBody returns the response body bytes with `{{var}}` tokens
// replaced from the varbag. A nil/empty body is a no-op.
func (e *Engine) substituteBody(ctx context.Context, slug, session string, raw []byte) ([]byte, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	out, err := e.varbag.Substitute(ctx, slug, session, string(raw))
	if err != nil {
		return nil, fmt.Errorf("substitute: %w", err)
	}
	return []byte(out), nil
}
