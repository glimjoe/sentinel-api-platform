package mock

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestBag spins up an in-memory Redis (miniredis) and returns a RedisVarBag
// pointed at it. Cleanup is automatic via t.Cleanup.
func newTestBag(t *testing.T) (*RedisVarBag, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	c := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = c.Close() })

	// Quick connectivity sanity check; miniredis always accepts the connection
	// so a failure here means the test setup itself is broken.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, c.Ping(ctx).Err(), "miniredis ping failed")

	return NewRedisVarBag(c), mr
}

func TestVarBag_SetGet_RoundTrip(t *testing.T) {
	bag, _ := newTestBag(t)
	ctx := context.Background()

	require.NoError(t, bag.Set(ctx, "petstore", "sess1", "login_token", "abc123"))

	got, ok, err := bag.Get(ctx, "petstore", "sess1", "login_token")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "abc123", got)
}

func TestVarBag_Get_MissingKey(t *testing.T) {
	bag, _ := newTestBag(t)
	ctx := context.Background()

	got, ok, err := bag.Get(ctx, "petstore", "sess1", "does_not_exist")
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Empty(t, got)
}

func TestVarBag_GetAll_EmptyReturnsMapNotNil(t *testing.T) {
	bag, _ := newTestBag(t)
	ctx := context.Background()

	m, err := bag.GetAll(ctx, "petstore", "sess1")
	require.NoError(t, err)
	assert.NotNil(t, m, "empty bag should be empty map, not nil")
	assert.Empty(t, m)
}

func TestVarBag_GetAll_AfterSets(t *testing.T) {
	bag, _ := newTestBag(t)
	ctx := context.Background()

	require.NoError(t, bag.Set(ctx, "petstore", "sess1", "a", "1"))
	require.NoError(t, bag.Set(ctx, "petstore", "sess1", "b", "2"))

	m, err := bag.GetAll(ctx, "petstore", "sess1")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"a": "1", "b": "2"}, m)
}

func TestVarBag_Set_RefreshesTTL(t *testing.T) {
	bag, mr := newTestBag(t)
	ctx := context.Background()

	require.NoError(t, bag.Set(ctx, "petstore", "sess1", "k", "v"))
	first := mr.TTL(varKey("petstore", "sess1"))
	assert.Greater(t, first, time.Duration(0), "TTL should be set after Set")

	// Sleep past the original TTL, then Set again. After the second Set the
	// TTL should be back near full window — proving it's sliding, not fixed.
	mr.FastForward(2 * varTTL)
	require.NoError(t, bag.Set(ctx, "petstore", "sess1", "k", "v2"))
	after := mr.TTL(varKey("petstore", "sess1"))
	assert.Greater(t, after, varTTL-5*time.Second, "TTL should be refreshed by second Set")
}

func TestVarBag_Substitute_ReplacesKnown(t *testing.T) {
	bag, _ := newTestBag(t)
	ctx := context.Background()

	require.NoError(t, bag.Set(ctx, "petstore", "sess1", "user_id", "42"))
	require.NoError(t, bag.Set(ctx, "petstore", "sess1", "role", "admin"))

	out, err := bag.Substitute(ctx, "petstore", "sess1", `{"id":{{user_id}},"role":"{{role}}"}`)
	require.NoError(t, err)
	assert.Equal(t, `{"id":42,"role":"admin"}`, out)
}

func TestVarBag_Substitute_LeavesUnknownIntact(t *testing.T) {
	bag, _ := newTestBag(t)
	ctx := context.Background()

	// {{missing}} is not in the bag — must remain literal per ADR-0007.
	out, err := bag.Substitute(ctx, "petstore", "sess1", `value={{missing}}`)
	require.NoError(t, err)
	assert.Equal(t, `value={{missing}}`, out)
}

func TestVarBag_Substitute_MixedKnownAndUnknown(t *testing.T) {
	bag, _ := newTestBag(t)
	ctx := context.Background()

	require.NoError(t, bag.Set(ctx, "petstore", "sess1", "a", "1"))

	out, err := bag.Substitute(ctx, "petstore", "sess1", `a={{a}} b={{b}} c={{a}}`)
	require.NoError(t, err)
	assert.Equal(t, `a=1 b={{b}} c=1`, out)
}

func TestVarBag_Substitute_NoTokensReturnsTemplate(t *testing.T) {
	bag, _ := newTestBag(t)
	ctx := context.Background()

	out, err := bag.Substitute(ctx, "petstore", "sess1", `no tokens here`)
	require.NoError(t, err)
	assert.Equal(t, `no tokens here`, out)
}

func TestVarBag_SessionIsolation(t *testing.T) {
	bag, _ := newTestBag(t)
	ctx := context.Background()

	require.NoError(t, bag.Set(ctx, "petstore", "sess-A", "k", "alpha"))
	require.NoError(t, bag.Set(ctx, "petstore", "sess-B", "k", "beta"))

	gotA, _, err := bag.Get(ctx, "petstore", "sess-A", "k")
	require.NoError(t, err)
	gotB, _, err := bag.Get(ctx, "petstore", "sess-B", "k")
	require.NoError(t, err)
	assert.Equal(t, "alpha", gotA)
	assert.Equal(t, "beta", gotB)
}

func TestVarBag_SlugIsolation(t *testing.T) {
	bag, _ := newTestBag(t)
	ctx := context.Background()

	// Same session id but different slugs must NOT see each other's vars.
	require.NoError(t, bag.Set(ctx, "petstore", "sess-1", "k", "pet"))
	require.NoError(t, bag.Set(ctx, "orders", "sess-1", "k", "ord"))

	gotP, _, err := bag.Get(ctx, "petstore", "sess-1", "k")
	require.NoError(t, err)
	gotO, _, err := bag.Get(ctx, "orders", "sess-1", "k")
	require.NoError(t, err)
	assert.Equal(t, "pet", gotP)
	assert.Equal(t, "ord", gotO)
}
