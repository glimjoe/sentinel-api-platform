package ai

import (
	"context"
	"testing"
)

func TestMockProvider_Complete_Attribution(t *testing.T) {
	p := &MockProvider{}
	resp, err := p.Complete(context.Background(), &ProviderRequest{
		SystemPrompt: "You are a failure attribution engine. Analyze the test result.",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content == "" {
		t.Error("expected non-empty content for attribution")
	}
	if resp.InputTokens == 0 || resp.OutputTokens == 0 {
		t.Error("mock should track token counts")
	}
	if resp.Model != "mock" {
		t.Errorf("model = %q, want mock", resp.Model)
	}
}

func TestMockProvider_Complete_Completion(t *testing.T) {
	p := &MockProvider{}
	resp, err := p.Complete(context.Background(), &ProviderRequest{
		SystemPrompt: "You are a test case completion engine. Generate test cases.",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content == "" {
		t.Error("expected non-empty content for completion")
	}
}

func TestMockProvider_Complete_Prioritization(t *testing.T) {
	p := &MockProvider{}
	resp, err := p.Complete(context.Background(), &ProviderRequest{
		SystemPrompt: "You are a priority suggestion engine. Reorder these test cases.",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content == "" {
		t.Error("expected non-empty content for prioritization")
	}
}

func TestMockProvider_Complete_Default(t *testing.T) {
	p := &MockProvider{}
	resp, err := p.Complete(context.Background(), &ProviderRequest{
		SystemPrompt: "What is 2+2?",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content == "" {
		t.Error("expected fallback content for unknown prompt")
	}
}

func TestNoopProvider_Complete(t *testing.T) {
	p := &NoopProvider{}
	_, err := p.Complete(context.Background(), &ProviderRequest{})
	if err != ErrAIDisabled {
		t.Errorf("expected ErrAIDisabled, got %v", err)
	}
}

func TestNewProvider_Factory(t *testing.T) {
	tests := []struct {
		provider string
		wantName string
	}{
		{"mock", "mock"},
		{"noop", "noop"},
	}
	for _, tc := range tests {
		t.Run(tc.provider, func(t *testing.T) {
			p := NewProvider(tc.provider, "", "")
			if p.Name() != tc.wantName {
				t.Errorf("Name() = %q, want %q", p.Name(), tc.wantName)
			}
		})
	}
}
