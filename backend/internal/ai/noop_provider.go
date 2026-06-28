package ai

import "context"

// NoopProvider returns ErrAIDisabled on every call — used when AI is off.
type NoopProvider struct{}

func (n *NoopProvider) Name() string { return "noop" }

func (n *NoopProvider) Complete(_ context.Context, _ *ProviderRequest) (*ProviderResponse, error) {
	return nil, ErrAIDisabled
}
