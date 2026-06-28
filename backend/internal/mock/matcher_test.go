package mock

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
)

// newRule is a small helper to build MockRule with sane defaults for testing.
// The `method` parameter is ignored at the model level — the engine pre-filters
// by method before calling PickBest — but it's kept in the signature so tests
// read like the calling code.
func newRule(id, _method string, match string, priority int) *model.MockRule {
	return &model.MockRule{
		ID:        id,
		APIID:     "api-" + id,
		Name:      "rule-" + id,
		MatchJSON: []byte(match),
		Priority:  priority,
		Enabled:   true,
	}
}

func newReq(method string, query map[string]string, headers map[string]string, body string) MatchRequest {
	q := url.Values{}
	for k, v := range query {
		q.Set(k, v)
	}
	h := http.Header{}
	for k, v := range headers {
		h.Set(k, v)
	}
	return MatchRequest{
		Method:  method,
		Query:   q,
		Headers: h,
		Body:    []byte(body),
	}
}

func TestMatcher_ExactQueryWins(t *testing.T) {
	m := NewMatcher()
	rule := newRule("01", "GET", `{"query":{"status":"available"}}`, 100)
	req := newReq("GET", map[string]string{"status": "available"}, nil, "")

	scored, err := m.PickBest(req, []*model.MockRule{rule})
	require.NoError(t, err)
	require.NotNil(t, scored)
	assert.Equal(t, 50, scored.Score)
	assert.Equal(t, "exact", scored.Reason)
}

func TestMatcher_MethodFilteringIsEnginesJob(t *testing.T) {
	// Method pre-filtering is the engine's responsibility (the engine joins
	// on apis.method when loading candidates). The matcher scores whatever
	// candidates it is given — it does not filter by method. This test
	// documents the contract: passing a rule with `match` that hits, the
	// matcher returns a score regardless of the request method.
	m := NewMatcher()
	rule := newRule("01", "GET", `{"query":{"status":"available"}}`, 100)
	req := newReq("POST", map[string]string{"status": "available"}, nil, "") // POST request, GET rule

	scored, err := m.PickBest(req, []*model.MockRule{rule})
	require.NoError(t, err)
	require.NotNil(t, scored, "matcher does not filter by method; that's the engine's job")
}

func TestMatcher_NoMatchReturnsNil(t *testing.T) {
	m := NewMatcher()
	rule := newRule("01", "GET", `{"query":{"status":"available"}}`, 100)
	req := newReq("GET", map[string]string{"status": "pending"}, nil, "")

	scored, err := m.PickBest(req, []*model.MockRule{rule})
	require.NoError(t, err)
	assert.Nil(t, scored)
}

func TestMatcher_PriorityWinsOnEqualScore(t *testing.T) {
	m := NewMatcher()
	low := newRule("01", "GET", `{"query":{"status":"available"}}`, 200) // higher priority number
	hi := newRule("02", "GET", `{"query":{"status":"available"}}`, 100)  // lower = wins
	req := newReq("GET", map[string]string{"status": "available"}, nil, "")

	scored, err := m.PickBest(req, []*model.MockRule{low, hi})
	require.NoError(t, err)
	require.NotNil(t, scored)
	assert.Equal(t, "02", scored.Rule.ID, "lower priority number wins on equal score")
}

func TestMatcher_TieBreakByID(t *testing.T) {
	m := NewMatcher()
	a := newRule("99", "GET", `{"query":{"status":"available"}}`, 100)
	b := newRule("01", "GET", `{"query":{"status":"available"}}`, 100) // same priority, lower ID
	req := newReq("GET", map[string]string{"status": "available"}, nil, "")

	scored, err := m.PickBest(req, []*model.MockRule{a, b})
	require.NoError(t, err)
	require.NotNil(t, scored)
	assert.Equal(t, "01", scored.Rule.ID, "lexicographically smallest ID wins")
	assert.Equal(t, "tie-break-by-id", scored.Reason, "engine will echo this header")
}

func TestMatcher_RegexMatch(t *testing.T) {
	m := NewMatcher()
	rule := newRule("01", "GET", `{"query":{"q":{"regex":"^hello.*"}}}`, 100)
	req := newReq("GET", map[string]string{"q": "hello world"}, nil, "")

	scored, err := m.PickBest(req, []*model.MockRule{rule})
	require.NoError(t, err)
	require.NotNil(t, scored)
	assert.Equal(t, 30, scored.Score)
	assert.Equal(t, "regex", scored.Reason)
}

func TestMatcher_TypeOnlyMatch(t *testing.T) {
	m := NewMatcher()
	rule := newRule("01", "GET", `{"query":{"id":{"type":"number"}}}`, 100)
	req := newReq("GET", map[string]string{"id": "42"}, nil, "")

	scored, err := m.PickBest(req, []*model.MockRule{rule})
	require.NoError(t, err)
	require.NotNil(t, scored)
	assert.Equal(t, 10, scored.Score)
	assert.Equal(t, "type-only", scored.Reason)
}

func TestMatcher_HeaderCaseInsensitive(t *testing.T) {
	m := NewMatcher()
	rule := newRule("01", "GET", `{"headers":{"X-Trace-Id":"abc"}}`, 100)
	req := newReq("GET", nil, map[string]string{"x-trace-id": "abc"}, "") // lowercase

	scored, err := m.PickBest(req, []*model.MockRule{rule})
	require.NoError(t, err)
	require.NotNil(t, scored)
	assert.Equal(t, 50, scored.Score)
}

func TestMatcher_BodyExactMatch(t *testing.T) {
	m := NewMatcher()
	rule := newRule("01", "POST", `{"body":{"user":{"name":"alice"}}}`, 100)
	req := newReq("POST", nil, nil, `{"user":{"name":"alice"}}`)

	scored, err := m.PickBest(req, []*model.MockRule{rule})
	require.NoError(t, err)
	require.NotNil(t, scored)
	assert.Equal(t, 50, scored.Score)
}

func TestMatcher_BodyNonMatchingJSONDoesNotScore(t *testing.T) {
	m := NewMatcher()
	rule := newRule("01", "POST", `{"body":{"user":{"name":"alice"}}}`, 100)
	req := newReq("POST", nil, nil, `{"user":{"name":"bob"}}`)

	scored, err := m.PickBest(req, []*model.MockRule{rule})
	require.NoError(t, err)
	assert.Nil(t, scored)
}

func TestMatcher_MultipleFieldsSumScore(t *testing.T) {
	m := NewMatcher()
	rule := newRule("01", "GET", `{"query":{"a":"1","b":"2"}}`, 100)
	req := newReq("GET", map[string]string{"a": "1", "b": "2", "c": "3"}, nil, "")

	scored, err := m.PickBest(req, []*model.MockRule{rule})
	require.NoError(t, err)
	require.NotNil(t, scored)
	assert.Equal(t, 100, scored.Score, "two exact fields = 50 + 50")
}

// ─── ValidateMatchJSON ────────────────────────────────────────────────────────

func TestValidateMatchJSON_HappyPath(t *testing.T) {
	m := NewMatcher()
	require.NoError(t, m.ValidateMatchJSON([]byte(`{"query":{"a":"1"}}`)))
	require.NoError(t, m.ValidateMatchJSON([]byte(`{"headers":{"K":{"regex":"^v"}}, "body":{"x":1}}`)))
}

func TestValidateMatchJSON_RejectsEmpty(t *testing.T) {
	m := NewMatcher()
	err := m.ValidateMatchJSON([]byte(``))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestValidateMatchJSON_RejectsInvalidJSON(t *testing.T) {
	m := NewMatcher()
	err := m.ValidateMatchJSON([]byte(`{"query": "not-an-object"}`))
	require.Error(t, err)
}

func TestValidateMatchJSON_RejectsUnknownKey(t *testing.T) {
	m := NewMatcher()
	err := m.ValidateMatchJSON([]byte(`{"quey":{"a":"1"}}`)) // typo
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown key")
}

func TestValidateMatchJSON_RejectsBadRegex(t *testing.T) {
	m := NewMatcher()
	err := m.ValidateMatchJSON([]byte(`{"query":{"q":{"regex":"("}}}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid regex")
}

func TestValidateMatchJSON_RejectsEmptyObjectSpec(t *testing.T) {
	m := NewMatcher()
	// Object with neither regex nor type → 422 per architect review.
	err := m.ValidateMatchJSON([]byte(`{"query":{"q":{}}}`))
	require.Error(t, err)
}

func TestMatcher_BadMatchJSON_ReturnsErrBadMatchJSON(t *testing.T) {
	m := NewMatcher()
	rule := newRule("01", "GET", `{"query":{"q":{}}}`, 100) // empty spec
	req := newReq("GET", map[string]string{"q": "v"}, nil, "")

	_, err := m.PickBest(req, []*model.MockRule{rule})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "bad match_json"),
		"engine looks for this exact phrase to decide between 404 and 422")
}

func TestMatcher_EmptyMatchJSON_ActsAsCatchall(t *testing.T) {
	m := NewMatcher()
	// Empty match_json is a "match any" fallback: scores 1 (minimum positive)
	// so the engine still applies priority+id tie-break. This is what lets
	// users configure a default response without enumerating query/header fields.
	rule := newRule("01", "GET", `{}`, 100)
	req := newReq("GET", nil, nil, "")

	scored, err := m.PickBest(req, []*model.MockRule{rule})
	require.NoError(t, err)
	require.NotNil(t, scored)
	assert.Equal(t, 1, scored.Score)
	assert.Equal(t, "catchall", scored.Reason)
}

func TestMatcher_TypeMatches_AllTypes(t *testing.T) {
	tests := []struct {
		typ, val string
		match    bool
	}{
		{"string", "hello", true},
		{"string", "", false},
		{"number", "42", true},
		{"number", "abc", false},
		{"boolean", "true", true},
		{"boolean", "nope", false},
		{"null", "", true},
		{"null", "null", true},
		{"null", "something", false},
		{"array", `[1,2]`, true},
		{"array", `not array`, false},
		{"object", `{"a":1}`, true},
		{"object", `not object`, false},
		{"unknown", "x", false},
	}
	for _, tt := range tests {
		t.Run(tt.typ+"="+tt.val, func(t *testing.T) {
			got := typeMatches(tt.typ, tt.val)
			assert.Equal(t, tt.match, got)
		})
	}
}
