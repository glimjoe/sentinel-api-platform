// Package mock — rule matcher (Phase 2 M1).
//
// The matcher scores each candidate rule against an incoming request and the
// engine picks the highest scorer. The algorithm follows plan §6.6:
//
//   - Method MUST match for a rule to be considered. The engine drops rules
//     whose method doesn't match before passing the survivors to the matcher.
//   - Each declared field in `match_json` is scored by its match style:
//     exact (+50), regex (+30), or type-only (+10).
//   - Same total score → lower `priority` wins.
//   - Same score + same priority → lexicographically smallest rule ID wins,
//     and the engine surfaces this via the `X-Mock-Match-Reason: tie-break-by-id`
//     response header so debugging is straightforward.
//
// `match_json` schema (per ADR-0007 + plan §6.6 + architect review):
//
//	{
//	  "query":   {"status": "available"},                  // exact string
//	  "headers": {"X-Trace-Id": {"regex": "^req-[0-9]+$"}}, // regex via object
//	  "body":    {"user": {"type": "object"}}              // type-only
//	}
//
// A field whose value is a plain string is treated as exact. A value that is
// an object with a `regex` key uses regex matching. A value with a `type` key
// is type-only (matches if the request value is of the named JSON type). Any
// other shape returns a 422-friendly error — we do NOT silently skip the rule
// (architect review rule of 2026-06-27).
package mock

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
)

// Match scores from plan §6.6. The method is implicitly gated upstream so it
// is not represented here — if a rule gets to scoring, method matched.
const (
	scoreExact = 50
	scoreRegex = 30
	scoreType  = 10
)

// ErrBadMatchJSON is returned by PickBest when a stored rule has invalid
// match_json. The engine maps this to 422 per the architect review rule.
var ErrBadMatchJSON = errors.New("bad match_json")

// MatchRequest is the flattened request the matcher scores rules against.
type MatchRequest struct {
	Method  string
	Path    string
	Query   url.Values
	Headers http.Header
	Body    []byte
}

// ScoredRule pairs a rule with its match result. Reason is one of:
// "exact", "regex", "type-only", "tie-break-by-id".
type ScoredRule struct {
	Rule   *model.MockRule
	Score  int
	Reason string
}

// Matcher scores rules. It is stateless after construction.
type Matcher struct{}

// NewMatcher constructs a Matcher.
func NewMatcher() *Matcher { return &Matcher{} }

// matchSpec is the decoded shape of a `match_json` column.
type matchSpec struct {
	Query   map[string]json.RawMessage `json:"query,omitempty"`
	Headers map[string]json.RawMessage `json:"headers,omitempty"`
	Body    json.RawMessage            `json:"body,omitempty"`
}

// fieldSpec is the per-field shape. Exactly one of Literal/Regex/TypeOf is
// set. An empty value (empty string for Literal) is treated as "field absent"
// because the matcher should not score a declared-but-empty field.
type fieldSpec struct {
	Literal string
	Regex   string
	TypeOf  string
}

// decodeFieldSpec returns the (spec, err) for a single field. err is non-nil
// only when the shape is unrecognised — that maps to 422.
func decodeFieldSpec(raw json.RawMessage) (fieldSpec, error) {
	if len(raw) == 0 {
		return fieldSpec{}, errors.New("empty match value")
	}
	// Plain string → exact.
	if raw[0] == '"' {
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return fieldSpec{}, fmt.Errorf("decode exact string: %w", err)
		}
		return fieldSpec{Literal: s}, nil
	}
	// Object → either regex or type.
	var obj struct {
		Regex string `json:"regex"`
		Type  string `json:"type"`
	}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return fieldSpec{}, fmt.Errorf("decode match object: %w", err)
	}
	switch {
	case obj.Regex != "":
		if _, err := regexp.Compile(obj.Regex); err != nil {
			return fieldSpec{}, fmt.Errorf("invalid regex %q: %w", obj.Regex, err)
		}
		return fieldSpec{Regex: obj.Regex}, nil
	case obj.Type != "":
		return fieldSpec{TypeOf: obj.Type}, nil
	default:
		return fieldSpec{}, errors.New("match object must have `regex` or `type`")
	}
}

// ValidateMatchJSON parses match_json. Returns nil on valid, otherwise a
// descriptive error that the engine maps to 422 (no silent skip).
func (m *Matcher) ValidateMatchJSON(b []byte) error {
	if len(bytes.TrimSpace(b)) == 0 {
		return errors.New("match_json is empty")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return fmt.Errorf("match_json: not a JSON object: %w", err)
	}
	for k := range raw {
		switch k {
		case "query", "headers", "body":
		default:
			return fmt.Errorf("match_json: unknown key %q (allowed: query, headers, body)", k)
		}
	}
	var spec matchSpec
	if err := json.Unmarshal(b, &spec); err != nil {
		return fmt.Errorf("match_json: not a JSON object: %w", err)
	}
	for k, v := range spec.Query {
		if _, err := decodeFieldSpec(v); err != nil {
			return fmt.Errorf("match_json.query[%s]: %w", k, err)
		}
	}
	for k, v := range spec.Headers {
		if _, err := decodeFieldSpec(v); err != nil {
			return fmt.Errorf("match_json.headers[%s]: %w", k, err)
		}
	}
	if len(spec.Body) > 0 {
		switch spec.Body[0] {
		case '{', '[':
			// ok — object or array
		default:
			return errors.New("match_json.body: must be a JSON object or array")
		}
	}
	return nil
}

// scoreField applies one declared field to the request. The match style was
// already decided by decodeFieldSpec; this function only compares.
func scoreField(spec fieldSpec, requestValue string) (int, string) {
	switch {
	case spec.Literal != "":
		if spec.Literal == requestValue {
			return scoreExact, "exact"
		}
	case spec.Regex != "":
		re := regexp.MustCompile(spec.Regex)
		if re.MatchString(requestValue) {
			return scoreRegex, "regex"
		}
	case spec.TypeOf != "":
		if typeMatches(spec.TypeOf, requestValue) {
			return scoreType, "type-only"
		}
	}
	return 0, ""
}

// typeMatches checks that requestValue parses as the named JSON type. For
// "string" we accept any non-empty value; for the rest we round-trip through
// json.Unmarshal.
func typeMatches(typ, requestValue string) bool {
	switch typ {
	case "string":
		return requestValue != ""
	case "number":
		var f float64
		return json.Unmarshal([]byte(requestValue), &f) == nil
	case "boolean":
		var b bool
		return json.Unmarshal([]byte(requestValue), &b) == nil
	case "null":
		return requestValue == "" || requestValue == "null"
	case "array":
		var a []any
		return json.Unmarshal([]byte(requestValue), &a) == nil
	case "object":
		var o map[string]any
		return json.Unmarshal([]byte(requestValue), &o) == nil
	}
	return false
}

// scoreRule returns the total score for rule against req. If match_json is
// invalid, ErrBadMatchJSON is returned (wrapped) so the engine can 422.
//
// An empty match_json — either literally `""` / whitespace, or a parsed
// `{}` with no declared fields — is treated as a "match any" fallback: it
// scores 1 (the minimum positive score) and the engine still applies
// priority+id tie-break across multiple catchalls. This lets users write a
// default response without enumerating query/header/body fields.
func (m *Matcher) scoreRule(req MatchRequest, rule *model.MockRule) (int, string, error) {
	if len(bytes.TrimSpace(rule.MatchJSON)) == 0 {
		return 1, "catchall", nil
	}
	if err := m.ValidateMatchJSON(rule.MatchJSON); err != nil {
		return 0, "", fmt.Errorf("%w: rule %s: %s", ErrBadMatchJSON, rule.ID, err)
	}
	var spec matchSpec
	if err := json.Unmarshal(rule.MatchJSON, &spec); err != nil {
		return 0, "", fmt.Errorf("%w: rule %s: %s", ErrBadMatchJSON, rule.ID, err)
	}

	// No declared fields → catchall.
	if len(spec.Query) == 0 && len(spec.Headers) == 0 && len(spec.Body) == 0 {
		return 1, "catchall", nil
	}

	total := 0
	bestReason := "exact" // default; overwritten if a regex/type match contributes

	for k, raw := range spec.Query {
		fs, _ := decodeFieldSpec(raw) // validated above
		s, reason := scoreField(fs, req.Query.Get(k))
		total += s
		if reason != "" && reason != "exact" {
			bestReason = reason
		}
	}
	for k, raw := range spec.Headers {
		fs, _ := decodeFieldSpec(raw)
		s, reason := scoreField(fs, req.Headers.Get(k))
		total += s
		if reason != "" && reason != "exact" {
			bestReason = reason
		}
	}
	if len(spec.Body) > 0 {
		s, reason, err := scoreBodyField(spec.Body, req.Body)
		if err != nil {
			return 0, "", fmt.Errorf("%w: rule %s: %s", ErrBadMatchJSON, rule.ID, err)
		}
		total += s
		if reason != "" && reason != "exact" {
			bestReason = reason
		}
	}
	if total == 0 {
		// A non-empty spec that scores 0 on every field = "doesn't match".
		return 0, "", nil
	}
	return total, bestReason, nil
}

// scoreBodyField handles body matching. Body values can be exact JSON or
// a {type:"..."} or {regex:"..."} wrapper.
func scoreBodyField(raw json.RawMessage, requestBody []byte) (int, string, error) {
	fs, err := decodeFieldSpec(raw)
	if err == nil && (fs.Literal != "" || fs.Regex != "" || fs.TypeOf != "") {
		s, reason := scoreField(fs, string(requestBody))
		return s, reason, nil
	}
	// Treat as exact JSON: parse request body and rule body, deep-equal.
	var want, got any
	if err := json.Unmarshal(raw, &want); err != nil {
		return 0, "", fmt.Errorf("body not JSON: %w", err)
	}
	if len(requestBody) == 0 {
		return 0, "", nil
	}
	if err := json.Unmarshal(requestBody, &got); err != nil {
		return 0, "", nil // non-JSON request body never matches an exact JSON rule
	}
	wj, _ := json.Marshal(want)
	gj, _ := json.Marshal(got)
	if bytes.Equal(wj, gj) {
		return scoreExact, "exact", nil
	}
	return 0, "", nil
}

// PickBest scores the candidates and returns the best match. The engine is
// responsible for pre-filtering candidates by HTTP method (and path) before
// passing them in — mock_rules does not carry a method column; the join is
// done in repository.ListByProject. Returns (nil, nil) if no candidate
// matched; (nil, ErrBadMatchJSON-wrapped error) if a stored rule's match_json
// is invalid (engine → 422).
//
// Tie-break per plan §6.6 + 2026-06-27 ADR edit:
//  1. Highest total score.
//  2. Lowest priority value.
//  3. Lexicographically smallest rule id (with X-Mock-Match-Reason = "tie-break-by-id").
func (m *Matcher) PickBest(req MatchRequest, candidates []*model.MockRule) (*ScoredRule, error) {
	var best *ScoredRule
	for _, r := range candidates {
		score, reason, err := m.scoreRule(req, r)
		if err != nil {
			return nil, err
		}
		if score == 0 {
			continue
		}
		cand := &ScoredRule{Rule: r, Score: score, Reason: reason}
		if best == nil {
			best = cand
			continue
		}
		switch {
		case cand.Score > best.Score:
			best = cand
		case cand.Score == best.Score && cand.Rule.Priority < best.Rule.Priority:
			best = cand
		case cand.Score == best.Score && cand.Rule.Priority == best.Rule.Priority &&
			cand.Rule.ID < best.Rule.ID:
			best = cand
			best.Reason = "tie-break-by-id"
		}
	}
	return best, nil
}
