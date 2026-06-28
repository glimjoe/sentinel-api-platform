package runner

import (
	"encoding/json"
	"testing"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
)

func TestAssert_StatusMatch(t *testing.T) {
	tc := &model.TestCase{ExpectedStatus: 200}
	f := Assert(tc, 200, nil, nil)
	if len(f) != 0 {
		t.Errorf("expected 0 failures, got %d: %v", len(f), f)
	}
}

func TestAssert_StatusMismatch(t *testing.T) {
	tc := &model.TestCase{ExpectedStatus: 200}
	f := Assert(tc, 404, nil, nil)
	if len(f) == 0 || f[0].Type != "status" {
		t.Errorf("expected status failure, got %v", f)
	}
}

func TestAssert_ExactBody(t *testing.T) {
	expected := json.RawMessage(`{"ok":true}`)
	tc := &model.TestCase{ExpectedBodyMatch: "exact", ExpectedBodyJSON: expected}
	f := Assert(tc, 200, nil, []byte(`{"ok":true}`))
	if len(f) != 0 {
		t.Errorf("expected 0 failures, got %v", f)
	}
}

func TestAssert_ExactBodyMismatch(t *testing.T) {
	expected := json.RawMessage(`{"ok":true}`)
	tc := &model.TestCase{ExpectedBodyMatch: "exact", ExpectedBodyJSON: expected}
	f := Assert(tc, 200, nil, []byte(`{"ok":false}`))
	if len(f) == 0 {
		t.Error("expected body.exact failure")
	}
}

func TestAssert_Contains(t *testing.T) {
	tc := &model.TestCase{ExpectedBodyMatch: "contains", ExpectedBodyJSON: []byte(`hello`)}
	f := Assert(tc, 200, nil, []byte(`{"msg":"hello world"}`))
	if len(f) != 0 {
		t.Errorf("expected 0 failures, got %v", f)
	}
}

func TestAssert_Regex(t *testing.T) {
	tc := &model.TestCase{ExpectedBodyMatch: "regex", ExpectedBodyPattern: `^\d{3}$`}
	f := Assert(tc, 200, nil, []byte(`200`))
	if len(f) != 0 {
		t.Errorf("expected 0 failures, got %v", f)
	}
}

func TestAssert_RegexMismatch(t *testing.T) {
	tc := &model.TestCase{ExpectedBodyMatch: "regex", ExpectedBodyPattern: `^\d{3}$`}
	f := Assert(tc, 200, nil, []byte(`abcd`))
	if len(f) == 0 {
		t.Error("expected regex failure")
	}
}

func TestAssert_Advanced_Status(t *testing.T) {
	assertions := json.RawMessage(`[{"type":"status","expect":200}]`)
	tc := &model.TestCase{AssertionsJSON: assertions, ExpectedStatus: 0}
	f := Assert(tc, 404, nil, nil)
	if len(f) == 0 {
		t.Error("expected status failure from advanced assertions")
	}
}

func TestAssert_JSONPath(t *testing.T) {
	spec := json.RawMessage(`{"path":"$.name","expect":"pets"}`)
	tc := &model.TestCase{ExpectedBodyMatch: "jsonpath", ExpectedBodyJSON: spec}
	f := Assert(tc, 200, nil, []byte(`{"name":"pets","count":5}`))
	if len(f) != 0 {
		t.Errorf("expected 0 failures, got %v", f)
	}
}

func TestAssert_JSONPathMismatch(t *testing.T) {
	spec := json.RawMessage(`{"path":"$.name","expect":"dogs"}`)
	tc := &model.TestCase{ExpectedBodyMatch: "jsonpath", ExpectedBodyJSON: spec}
	f := Assert(tc, 200, nil, []byte(`{"name":"pets"}`))
	if len(f) == 0 {
		t.Error("expected jsonpath mismatch")
	}
}

func TestAssert_JSONSchema(t *testing.T) {
	schema := json.RawMessage(`{"type":"object","properties":{"ok":{"type":"boolean"}},"required":["ok"]}`)
	tc := &model.TestCase{ExpectedBodyMatch: "schema", ExpectedBodyJSON: schema}
	f := Assert(tc, 200, nil, []byte(`{"ok":true}`))
	if len(f) != 0 {
		t.Errorf("expected 0 failures, got %v", f)
	}
}

func TestAssert_JSONSchemaMismatch(t *testing.T) {
	schema := json.RawMessage(`{"type":"object","properties":{"ok":{"type":"boolean"}},"required":["ok"]}`)
	tc := &model.TestCase{ExpectedBodyMatch: "schema", ExpectedBodyJSON: schema}
	f := Assert(tc, 200, nil, []byte(`{"not_ok":1}`))
	if len(f) == 0 {
		t.Error("expected schema failure")
	}
}

func TestAssert_AdvancedHeader(t *testing.T) {
	assertions := json.RawMessage(`[{"type":"header","name":"x-request-id","expect":"abc123"}]`)
	tc := &model.TestCase{AssertionsJSON: assertions}
	headers := map[string][]string{"X-Request-Id": {"abc123"}}
	f := Assert(tc, 200, headers, nil)
	if len(f) != 0 {
		t.Errorf("expected 0 failures, got %v", f)
	}
}

func TestAssert_AdvancedHeaderMismatch(t *testing.T) {
	assertions := json.RawMessage(`[{"type":"header","name":"x-request-id","expect":"abc123"}]`)
	tc := &model.TestCase{AssertionsJSON: assertions}
	headers := map[string][]string{"X-Request-Id": {"xyz789"}}
	f := Assert(tc, 200, headers, nil)
	if len(f) == 0 {
		t.Error("expected header failure")
	}
}

func TestAssert_AdvancedInvalidJSON(t *testing.T) {
	assertions := json.RawMessage(`not-json`)
	tc := &model.TestCase{AssertionsJSON: assertions}
	f := Assert(tc, 200, nil, nil)
	if len(f) == 0 || f[0].Type != "assertions.parse" {
		t.Errorf("expected assertions.parse failure, got %v", f)
	}
}

func TestAssert_BodyUnknownType(t *testing.T) {
	tc := &model.TestCase{ExpectedBodyMatch: "unknown_mode", ExpectedBodyJSON: []byte(`x`)}
	f := Assert(tc, 200, nil, []byte(`x`))
	if len(f) == 0 || f[0].Type != "body.unknown" {
		t.Errorf("expected body.unknown failure, got %v", f)
	}
}

func TestAssert_JSONPathArrayIndex(t *testing.T) {
	spec := json.RawMessage(`{"path":"$.items[1].name","expect":"bar"}`)
	tc := &model.TestCase{ExpectedBodyMatch: "jsonpath", ExpectedBodyJSON: spec}
	f := Assert(tc, 200, nil, []byte(`{"items":[{"name":"foo"},{"name":"bar"}]}`))
	if len(f) != 0 {
		t.Errorf("expected 0 failures, got %v", f)
	}
}

func TestAssert_JSONPathArrayIndexOutOfRange(t *testing.T) {
	spec := json.RawMessage(`{"path":"$.items[99].name","expect":"bar"}`)
	tc := &model.TestCase{ExpectedBodyMatch: "jsonpath", ExpectedBodyJSON: spec}
	f := Assert(tc, 200, nil, []byte(`{"items":[{"name":"foo"}]}`))
	if len(f) == 0 {
		t.Error("expected jsonpath failure for out-of-range index")
	}
}

func TestAssert_InvalidRegex(t *testing.T) {
	tc := &model.TestCase{ExpectedBodyMatch: "regex", ExpectedBodyPattern: `[invalid`}
	f := Assert(tc, 200, nil, []byte(`test`))
	if len(f) == 0 || f[0].Type != "body.regex" {
		t.Errorf("expected body.regex failure, got %v", f)
	}
}

func TestAssert_RegexFallbackToBodyJSON(t *testing.T) {
	tc := &model.TestCase{ExpectedBodyMatch: "regex", ExpectedBodyJSON: []byte(`^\d+$`)}
	f := Assert(tc, 200, nil, []byte(`42`))
	if len(f) != 0 {
		t.Errorf("expected 0 failures (regex from body_json), got %v", f)
	}
}

func TestJSONPathLookup_NavigateNonObject(t *testing.T) {
	_, err := jsonPathLookup([]byte(`"not an object"`), "$.somekey")
	if err == nil {
		t.Error("expected error navigating into non-object")
	}
}

func TestJSONPathLookup_KeyNotFound(t *testing.T) {
	_, err := jsonPathLookup([]byte(`{"a":1}`), "$.b")
	if err == nil {
		t.Error("expected error for missing key")
	}
}

func TestJSONPathLookup_NotArray(t *testing.T) {
	_, err := jsonPathLookup([]byte(`{"a":"not_array"}`), "$.a[0]")
	if err == nil {
		t.Error("expected error for indexing non-array")
	}
}
