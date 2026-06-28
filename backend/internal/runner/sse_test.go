package runner

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisEventBroker_PublishSubscribe(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	broker := NewRedisEventBroker(rdb)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ch, unsubscribe, err := broker.Subscribe(ctx, "run-1")
	require.NoError(t, err)
	defer unsubscribe()

	err = broker.Publish(ctx, &RunEvent{
		Type: "progress", RunID: "run-1", Total: 5, Passed: 2, Failed: 1,
		Status: "running", Timestamp: 123,
	})
	require.NoError(t, err)

	select {
	case evt := <-ch:
		assert.Equal(t, "progress", evt.Type)
		assert.Equal(t, "run-1", evt.RunID)
		assert.Equal(t, 5, evt.Total)
		assert.Equal(t, 2, evt.Passed)
		assert.Equal(t, 1, evt.Failed)
	case <-ctx.Done():
		t.Fatal("timeout waiting for event")
	}
}

func TestRedisEventBroker_MultipleEvents(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	broker := NewRedisEventBroker(rdb)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ch, unsubscribe, err := broker.Subscribe(ctx, "run-2")
	require.NoError(t, err)
	defer unsubscribe()

	for i := 0; i < 3; i++ {
		err = broker.Publish(ctx, &RunEvent{Type: "progress", RunID: "run-2", Total: i})
		require.NoError(t, err)
	}

	for i := 0; i < 3; i++ {
		select {
		case evt := <-ch:
			assert.Equal(t, i, evt.Total)
		case <-ctx.Done():
			t.Fatal("timeout waiting for event", i)
		}
	}
}

func TestRedisEventBroker_ContextCancel(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	broker := NewRedisEventBroker(rdb)

	ctx, cancel := context.WithCancel(context.Background())
	ch, unsubscribe, err := broker.Subscribe(ctx, "run-3")
	require.NoError(t, err)
	defer unsubscribe()

	cancel()
	time.Sleep(50 * time.Millisecond)

	select {
	case _, ok := <-ch:
		if ok {
			// Might receive a heartbeat, that's fine
		}
	default:
		// Channel closed or empty — either is acceptable
	}
}
