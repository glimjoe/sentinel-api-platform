package ai

import (
	"context"
	"encoding/json"
	"fmt"
)

// PriorityItem is one priority suggestion.
type PriorityItem struct {
	CaseID    string `json:"case_id"`
	Priority  string `json:"priority"`
	Reasoning string `json:"reasoning"`
}

// Prioritizer provides AI-powered priority suggestions.
type Prioritizer struct{ engine *Engine }

// NewPrioritizer creates a Prioritizer.
func NewPrioritizer(engine *Engine) *Prioritizer { return &Prioritizer{engine: engine} }

// Prioritize suggests priorities for test cases.
func (p *Prioritizer) Prioritize(ctx context.Context, casesJSON string) ([]PriorityItem, error) {
	msg := Message{Role: "user", Content: "Test cases:\n" + casesJSON}
	resp, err := p.engine.call(ctx, "prioritization", promptPrioritization, []Message{msg})
	if err != nil {
		return nil, err
	}
	var result struct{ Priorities []PriorityItem `json:"priorities"` }
	if err := json.Unmarshal([]byte(resp.Content), &result); err != nil {
		return nil, fmt.Errorf("parse prioritization response: %w", err)
	}
	return result.Priorities, nil
}
