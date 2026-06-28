package ai

import (
	"context"
	"fmt"
	"math"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/id"
)

// Engine orchestrates AI operations: provider + cache + guard.
type Engine struct {
	provider Provider
	cache    *Cache
	guard    *Guard
	MaxTokens   int
	Temperature float64
}

// NewEngine constructs an Engine.
func NewEngine(provider Provider, cache *Cache, guard *Guard, maxTokens int, temperature float64) *Engine {
	return &Engine{
		provider:    provider,
		cache:       cache,
		guard:       guard,
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}
}

// call runs the full pipeline: guard → cache → provider → cache → record.
func (e *Engine) call(ctx context.Context, function string, systemPrompt string, messages []Message) (*ProviderResponse, error) {
	if err := e.guard.Allow(ctx); err != nil {
		return nil, err
	}

	req := &ProviderRequest{
		SystemPrompt: systemPrompt,
		Messages:     messages,
		MaxTokens:    e.MaxTokens,
		Temperature:  e.Temperature,
	}

	if e.cache != nil {
		if resp, ok := e.cache.Get(ctx, req); ok {
			return resp, nil
		}
	}

	resp, err := e.provider.Complete(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("ai: provider: %w", err)
	}

	if e.cache != nil {
		_ = e.cache.Set(ctx, req, resp)
	}

	cost := estimateCost(resp.Model, resp.InputTokens, resp.OutputTokens)
	_ = e.guard.Record(ctx, &model.AiUsage{
		ID:               id.New(),
		Model:            resp.Model,
		Function:         function,
		PromptTokens:     resp.InputTokens,
		CompletionTokens: resp.OutputTokens,
		CostUSD:          cost,
	})

	return resp, nil
}

var modelPricing = map[string]struct{ input, output float64 }{
	"mock":                {0, 0},
	"claude-sonnet-4-6":   {3.0 / 1_000_000, 15.0 / 1_000_000},
	"claude-opus-4-8":     {15.0 / 1_000_000, 75.0 / 1_000_000},
	"gpt-4o":              {2.5 / 1_000_000, 10.0 / 1_000_000},
	"gpt-4o-mini":         {0.15 / 1_000_000, 0.60 / 1_000_000},
}

func estimateCost(modelStr string, inTokens, outTokens int) float64 {
	p, ok := modelPricing[modelStr]
	if !ok {
		p = modelPricing["gpt-4o"]
	}
	cost := float64(inTokens)*p.input + float64(outTokens)*p.output
	return math.Round(cost*1e8) / 1e8
}
