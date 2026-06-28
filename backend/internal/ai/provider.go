// Package ai — AI module (Phase 4).
package ai

import (
	"context"
	"errors"
)

// Sentinel errors for AI operations.
var (
	ErrAIDisabled            = errors.New("ai: AI is disabled")
	ErrDailyBudgetExceeded   = errors.New("ai: daily budget exceeded")
	ErrMonthlyBudgetExceeded = errors.New("ai: monthly budget exceeded")
)

// Provider abstracts an LLM backend (mock, anthropic, openai, noop).
type Provider interface {
	Complete(ctx context.Context, req *ProviderRequest) (*ProviderResponse, error)
	Name() string
}

// ProviderRequest is the canonical request shape for all providers.
type ProviderRequest struct {
	SystemPrompt string
	Messages     []Message
	MaxTokens    int
	Temperature  float64
	Function     string // "attribution", "completion", "prioritization"
}

// Message represents a chat message.
type Message struct {
	Role    string
	Content string
}

// ProviderResponse is the canonical response shape from all providers.
type ProviderResponse struct {
	Content      string
	Model        string
	InputTokens  int
	OutputTokens int
}

// NewProvider returns a provider for the given type.
func NewProvider(provider, anthropicKey, openaiKey string) Provider {
	switch provider {
	case "mock":
		return &MockProvider{}
	case "anthropic":
		return &NoopProvider{} // deferred
	case "openai":
		return &NoopProvider{} // deferred
	default:
		return &NoopProvider{}
	}
}
