package contract

import (
	"encoding/json"
	"testing"
)

func TestValidateBody_Valid(t *testing.T) {
	schema := json.RawMessage(`{"type":"object","properties":{"ok":{"type":"boolean"}},"required":["ok"]}`)
	body := []byte(`{"ok":true}`)
	errs, err := ValidateBody(schema, body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("expected 0 errors, got %v", errs)
	}
}

func TestValidateBody_Invalid(t *testing.T) {
	schema := json.RawMessage(`{"type":"object","properties":{"ok":{"type":"boolean"}},"required":["ok"]}`)
	body := []byte(`{"not_ok":1}`)
	errs, err := ValidateBody(schema, body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("expected validation errors for missing required field")
	}
}

func TestIsValidSchema_Valid(t *testing.T) {
	if !IsValidSchema([]byte(`{"type":"object"}`)) {
		t.Error("expected valid schema")
	}
}

func TestIsValidSchema_Invalid(t *testing.T) {
	if IsValidSchema([]byte(`not json schema`)) {
		t.Error("expected invalid schema")
	}
}

func TestSchemaFromJSON_AddsDollarSchema(t *testing.T) {
	schema, err := SchemaFromJSON(json.RawMessage(`{"type":"string"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !json.Valid(schema) {
		t.Error("output is not valid JSON")
	}
}

func TestSchemaFromJSON_InvalidJSON(t *testing.T) {
	_, err := SchemaFromJSON(json.RawMessage(`not json`))
	if err == nil {
		t.Error("expected error for invalid JSON input")
	}
}

func TestValidateBody_InvalidSchema(t *testing.T) {
	schema := json.RawMessage(`{"type":"object","properties":{"ok":{"type":"boolean"}}}`)
	body := []byte(`{"ok":true}`)
	_, err := ValidateBody(schema, body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Test with a malformed schema that gojsonschema itself rejects.
	_, err = ValidateBody(json.RawMessage(`{invalid schema`), body)
	if err == nil {
		t.Error("expected system error for invalid schema JSON")
	}
}
