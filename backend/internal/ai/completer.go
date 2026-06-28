package ai

import (
	"context"
	"encoding/json"
	"fmt"
)

// GeneratedCase is one AI-suggested test case.
type GeneratedCase struct {
	Name                string `json:"name"`
	Method              string `json:"method"`
	Path                string `json:"path"`
	ExpectedStatus      int    `json:"expected_status"`
	ExpectedBodyMatch   string `json:"expected_body_match,omitempty"`
	ExpectedBodyPattern string `json:"expected_body_pattern,omitempty"`
}

// Completer provides AI-powered test case generation.
type Completer struct{ engine *Engine }

// NewCompleter creates a Completer.
func NewCompleter(engine *Engine) *Completer { return &Completer{engine: engine} }

// Complete generates test cases from API specs.
func (c *Completer) Complete(ctx context.Context, apiSpecsJSON, existingCasesJSON string) ([]GeneratedCase, error) {
	content := "API specifications:\n" + apiSpecsJSON
	if existingCasesJSON != "" && existingCasesJSON != "[]" {
		content += "\n\nExisting test cases:\n" + existingCasesJSON
	}
	msg := Message{Role: "user", Content: content}
	resp, err := c.engine.call(ctx, "completion", promptCompletion, []Message{msg})
	if err != nil {
		return nil, err
	}
	var result struct{ TestCases []GeneratedCase `json:"test_cases"` }
	if err := json.Unmarshal([]byte(resp.Content), &result); err != nil {
		return nil, fmt.Errorf("parse completion response: %w", err)
	}
	return result.TestCases, nil
}
