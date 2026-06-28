// Package contract — runtime JSON Schema validation.
package contract

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/xeipuuv/gojsonschema"
)

// ValidationError holds a single schema validation issue.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidateBody checks the given JSON body against a JSON Schema document.
// Returns nil if the body passes validation.
func ValidateBody(schemaRaw, body []byte) ([]ValidationError, error) {
	schemaLoader := gojsonschema.NewBytesLoader(schemaRaw)
	docLoader := gojsonschema.NewBytesLoader(body)
	result, err := gojsonschema.Validate(schemaLoader, docLoader)
	if err != nil {
		return nil, fmt.Errorf("schema validation: %w", err)
	}
	if result.Valid() {
		return nil, nil
	}
	errs := make([]ValidationError, 0, len(result.Errors()))
	for _, e := range result.Errors() {
		errs = append(errs, ValidationError{
			Field:   e.Field(),
			Message: strings.TrimPrefix(e.String(), e.Field()+": "),
		})
	}
	return errs, nil
}

// IsValidSchema checks whether raw is a valid JSON Schema document.
func IsValidSchema(raw []byte) bool {
	loader := gojsonschema.NewBytesLoader(raw)
	_, err := gojsonschema.NewSchema(loader)
	return err == nil
}

// SchemaFromJSON extracts a JSON Schema from an OpenAPI schema object.
func SchemaFromJSON(raw json.RawMessage) ([]byte, error) {
	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		return nil, fmt.Errorf("parse schema: %w", err)
	}
	if _, ok := schema["$schema"]; !ok {
		schema["$schema"] = "http://json-schema.org/draft-07/schema#"
	}
	return json.Marshal(schema)
}
