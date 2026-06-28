package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RunEvent is emitted during test execution for SSE clients.
type RunEvent struct {
	Type      string `json:"type"`      // "progress", "complete", "heartbeat"
	RunID     string `json:"run_id"`
	Total     int    `json:"total"`
	Passed    int    `json:"passed"`
	Failed    int    `json:"failed"`
	Errored   int    `json:"errored"`
	Skipped   int    `json:"skipped"`
	Status    string `json:"status,omitempty"` // "running", "success", "failed"
	Timestamp int64  `json:"ts"`
}

// EventPublisher pushes run events to subscribers (Redis pub/sub).
type EventPublisher interface {
	Publish(ctx context.Context, evt *RunEvent) error
}

// EventSubscriber returns a channel of RunEvents for a given run.
type EventSubscriber interface {
	Subscribe(ctx context.Context, runID string) (<-chan *RunEvent, func(), error)
}

// ---------------------------------------------------------------------------
// Redis implementation
// ---------------------------------------------------------------------------

const runEventChannel = "run:events"

type RedisEventBroker struct{ rdb *redis.Client }

func NewRedisEventBroker(rdb *redis.Client) *RedisEventBroker {
	return &RedisEventBroker{rdb: rdb}
}

func (b *RedisEventBroker) Publish(ctx context.Context, evt *RunEvent) error {
	data, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	return b.rdb.Publish(ctx, runEventChannel+":"+evt.RunID, data).Err()
}

func (b *RedisEventBroker) Subscribe(ctx context.Context, runID string) (<-chan *RunEvent, func(), error) {
	pubsub := b.rdb.Subscribe(ctx, runEventChannel+":"+runID)
	ch := make(chan *RunEvent, 16)

	go func() {
		defer pubsub.Close()
		defer close(ch)

		// Heartbeat ticker — 15s per ADR-0006
		heartbeat := time.NewTicker(15 * time.Second)
		defer heartbeat.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-heartbeat.C:
				select {
				case ch <- &RunEvent{Type: "heartbeat", RunID: runID}:
				default:
				}
			case msg, ok := <-pubsub.Channel():
				if !ok {
					return
				}
				var evt RunEvent
				if err := json.Unmarshal([]byte(msg.Payload), &evt); err != nil {
					continue
				}
				select {
				case ch <- &evt:
				default:
					// Drop if buffer full (client is slow)
				}
			}
		}
	}()

	cancel := func() { pubsub.Close() }
	return ch, cancel, nil
}
