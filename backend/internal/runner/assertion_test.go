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
