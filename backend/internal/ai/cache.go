package ai

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache provides Redis-backed prompt-response caching.
type Cache struct {
	rdb redis.UniversalClient
	ttl time.Duration
}

// NewCache creates a Cache with the given Redis client and TTL.
func NewCache(rdb redis.UniversalClient, ttl time.Duration) *Cache {
	return &Cache{rdb: rdb, ttl: ttl}
}

// Get returns a cached response for the request, or (nil, false) on miss.
func (c *Cache) Get(ctx context.Context, req *ProviderRequest) (*ProviderResponse, bool) {
	key := "ai:cache:" + hashPrompt(req)
	data, err := c.rdb.Get(ctx, key).Bytes()
	if err != nil {
		return nil, false
	}
	var resp ProviderResponse
	if json.Unmarshal(data, &resp) != nil {
		return nil, false
	}
	return &resp, true
}

// Set stores a response for the given request in the cache.
func (c *Cache) Set(ctx context.Context, req *ProviderRequest, resp *ProviderResponse) error {
	key := "ai:cache:" + hashPrompt(req)
	data, _ := json.Marshal(resp)
	return c.rdb.Set(ctx, key, data, c.ttl).Err()
}

func hashPrompt(req *ProviderRequest) string {
	h := sha256.New()
	h.Write([]byte(req.SystemPrompt))
	for _, m := range req.Messages {
		h.Write([]byte(m.Role))
		h.Write([]byte(m.Content))
	}
	// Include parameter variations in the cache key.
	h.Write([]byte(fmt.Sprintf("|mt=%d|t=%.2f", req.MaxTokens, req.Temperature)))
	return hex.EncodeToString(h.Sum(nil))
}
