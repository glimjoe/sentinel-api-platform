package ai

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestCache(t *testing.T) (*Cache, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return NewCache(rdb, 60*time.Second), mr
}

func TestCache_SetGet(t *testing.T) {
	c, _ := newTestCache(t)
	ctx := context.Background()
	req := &ProviderRequest{SystemPrompt: "test attribution engine"}

	resp := &ProviderResponse{Content: "cached-result", Model: "mock", InputTokens: 10, OutputTokens: 5}
	if err := c.Set(ctx, req, resp); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, ok := c.Get(ctx, req)
	if !ok {
		t.Fatal("cache miss after Set")
	}
	if got.Content != "cached-result" {
		t.Errorf("Content = %q, want cached-result", got.Content)
	}
}

func TestCache_HitStable(t *testing.T) {
	c, _ := newTestCache(t)
	ctx := context.Background()
	req := &ProviderRequest{SystemPrompt: "attribution", Messages: []Message{{Role: "user", Content: "test"}}}

	resp := &ProviderResponse{Content: "x", Model: "mock"}
	if err := c.Set(ctx, req, resp); err != nil {
		t.Fatalf("Set: %v", err)
	}

	req2 := &ProviderRequest{SystemPrompt: "attribution", Messages: []Message{{Role: "user", Content: "test"}}}
	if _, ok := c.Get(ctx, req2); !ok {
		t.Error("expected cache hit for identical prompt")
	}
}

func TestCache_MissDifferent(t *testing.T) {
	c, _ := newTestCache(t)
	ctx := context.Background()

	req := &ProviderRequest{SystemPrompt: "attribution"}
	c.Set(ctx, req, &ProviderResponse{Content: "x", Model: "mock"})

	diff := &ProviderRequest{SystemPrompt: "completion"}
	if _, ok := c.Get(ctx, diff); ok {
		t.Error("expected cache miss for different prompt")
	}
}

func TestCache_MissEmpty(t *testing.T) {
	c, _ := newTestCache(t)
	ctx := context.Background()
	if _, ok := c.Get(ctx, &ProviderRequest{SystemPrompt: "never-set"}); ok {
		t.Error("expected cache miss for unset key")
	}
}
