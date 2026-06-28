package ai

import (
	"context"
	"encoding/json"
	"fmt"
)

// AttributionResult is the parsed output from the attributor.
type AttributionResult struct {
	Analysis     string  `json:"analysis"`
	RootCause    string  `json:"root_cause"`
	Confidence   float64 `json:"confidence"`
	SuggestedFix string  `json:"suggested_fix,omitempty"`
}

// Attributor provides failure attribution.
type Attributor struct{ engine *Engine }

// NewAttributor creates an Attributor backed by the engine.
func NewAttributor(engine *Engine) *Attributor { return &Attributor{engine: engine} }

// Attribute analyzes a test result and returns attribution.
func (a *Attributor) Attribute(ctx context.Context, resultJSON string) (*AttributionResult, error) {
	msg := Message{Role: "user", Content: "Test result:\n" + resultJSON}
	resp, err := a.engine.call(ctx, "attribution", promptAttribution, []Message{msg})
	if err != nil {
		return nil, err
	}
	var ar AttributionResult
	if err := json.Unmarshal([]byte(resp.Content), &ar); err != nil {
		return nil, fmt.Errorf("parse attribution response: %w", err)
	}
	return &ar, nil
}
