// Package mock — variable bag (ADR-0007).
//
// The VarBag stores named values across requests within a mock session. It is
// backed by Redis (HASH at `mock:vars:{slug}:{sessionId}`) so that multi-replica
// and multi-process deployments all see the same bag. The TTL is 30 minutes
// sliding — every read or write resets it, so an active session never expires
// mid-flow; an abandoned session is reaped by Redis automatically.
//
// Per ADR-0007 §"Conflict rule", last-write-wins per field. Per §"Substitution",
// unknown `{{var}}` references are left intact (NOT replaced with empty) so
// the MockConsole can show a banner pointing at the missing name.
package mock

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// varTTL is the sliding window. ADR-0007 specifies 30 minutes.
const varTTL = 30 * time.Minute

// VarBag is the per-session, per-project variable storage. The engine calls
// Set when the extractor finds a `{{var}}` source, and Substitute before
// returning a response body. GetAll is exposed for the MockConsole (M3).
type VarBag interface {
	// Get returns the value for name and whether it exists.
	Get(ctx context.Context, slug, session, name string) (string, bool, error)

	// Set writes name=value and refreshes the TTL. Errors if the write fails.
	Set(ctx context.Context, slug, session, name, value string) error

	// GetAll returns the entire bag as a map. Empty map (not nil) when no keys
	// exist. Returns an error only on transport failure — a missing key is
	// represented by an empty map, not an error.
	GetAll(ctx context.Context, slug, session string) (map[string]string, error)

	// Substitute replaces every `{{name}}` token in template with the bag
	// value. Tokens whose name is not in the bag are left intact so the
	// MockConsole can flag them. The TTL is refreshed on each lookup.
	Substitute(ctx context.Context, slug, session, template string) (string, error)
}

// RedisVarBag is the production VarBag, backed by Redis.
type RedisVarBag struct {
	client *redis.Client
}

// NewRedisVarBag constructs a RedisVarBag from a connected *redis.Client.
func NewRedisVarBag(c *redis.Client) *RedisVarBag {
	return &RedisVarBag{client: c}
}

// varKey is the canonical key shape `mock:vars:{slug}:{sessionId}`.
func varKey(slug, session string) string {
	return "mock:vars:" + slug + ":" + session
}

// Get implements VarBag.Get. Uses HGET which is a single round-trip.
func (b *RedisVarBag) Get(ctx context.Context, slug, session, name string) (string, bool, error) {
	v, err := b.client.HGet(ctx, varKey(slug, session), name).Result()
	switch {
	case errors.Is(err, redis.Nil):
		return "", false, nil
	case err != nil:
		return "", false, fmt.Errorf("varbag hget: %w", err)
	}
	// Best-effort TTL refresh. Failure here is non-fatal — the data was read
	// successfully, and the next write or a separate housekeeping sweep will
	// refresh the TTL anyway.
	_ = b.client.Expire(ctx, varKey(slug, session), varTTL).Err()
	return v, true, nil
}

// Set implements VarBag.Set. Uses HSET + EXPIRE in a pipeline (one round-trip).
func (b *RedisVarBag) Set(ctx context.Context, slug, session, name, value string) error {
	key := varKey(slug, session)
	pipe := b.client.Pipeline()
	pipe.HSet(ctx, key, name, value)
	pipe.Expire(ctx, key, varTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("varbag hset+expire: %w", err)
	}
	return nil
}

// GetAll implements VarBag.GetAll. Uses HGETALL.
func (b *RedisVarBag) GetAll(ctx context.Context, slug, session string) (map[string]string, error) {
	m, err := b.client.HGetAll(ctx, varKey(slug, session)).Result()
	if err != nil {
		return nil, fmt.Errorf("varbag hgetall: %w", err)
	}
	if len(m) == 0 {
		return map[string]string{}, nil
	}
	// Best-effort TTL refresh — see Get for rationale.
	_ = b.client.Expire(ctx, varKey(slug, session), varTTL).Err()
	return m, nil
}

// Substitute implements VarBag.Substitute. The regex matches `{{name}}` where
// name is anything that is not whitespace and not `}`. The match is non-greedy
// so `{{a}}{{b}}` produces two captures, not one giant one.
var substituteRE = regexp.MustCompile(`\{\{\s*([^\s}]+?)\s*\}\}`)

func (b *RedisVarBag) Substitute(ctx context.Context, slug, session, template string) (string, error) {
	matches := substituteRE.FindAllStringSubmatchIndex(template, -1)
	if len(matches) == 0 {
		return template, nil
	}

	// Collect unique var names to minimise round-trips. The bag is a HASH so
	// we can fetch them in one MGET-like call... actually HGETALL is fine for
	// the whole hash (typical bags have < 20 entries).
	bag, err := b.GetAll(ctx, slug, session)
	if err != nil {
		return "", err
	}

	// Walk the template left-to-right and rebuild. We use a strings.Builder
	// with manual copy of unmatched segments to avoid a second regex pass.
	var out strings.Builder
	out.Grow(len(template))
	cursor := 0
	for _, m := range matches {
		fullStart, fullEnd := m[0], m[1]
		nameStart, nameEnd := m[2], m[3]
		// Copy the segment before this match.
		out.WriteString(template[cursor:fullStart])
		name := template[nameStart:nameEnd]
		if v, ok := bag[name]; ok {
			out.WriteString(v)
		} else {
			// Leave the original `{{name}}` token intact per ADR-0007.
			out.WriteString(template[fullStart:fullEnd])
		}
		cursor = fullEnd
	}
	out.WriteString(template[cursor:])
	return out.String(), nil
}
