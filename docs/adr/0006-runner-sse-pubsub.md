# ADR-0006: Runner SSE — Redis pub/sub fanout

**Status:** Accepted (2026-06-27)
**Phase:** 2 (informs Phase 3 implementation)

## Context

- The test runner (`internal/runner/runner.go`) runs in a single goroutine per `test_runs` row and may execute dozens of cases over seconds-to-minutes.
- The frontend (`/projects/:pid/runs/:runId`, §7.1) needs real-time per-case progress via SSE (`EventSource`).
- The backend is bare-metal (`scripts/start_all.sh`, no k8s); the user wants a single-binary deployable runner but **also** wants Phase 5 horizontal scale (per §14.3 "扩到 100 人怎么改").
- The plan §6.8 mentions "SSE 实时流（Redis pub/sub）" but does not specify keys, payload schema, reconnect, or backpressure.

If the runner writes SSE directly to the response writer, the design breaks the moment a second process is involved (the user already plans for that in §14.3 answer 4: "runner 拆独立二进制"). Pub/sub decouples runner from listener.

## Decision

Use **Redis Pub/Sub** as the fanout bus between runner and SSE handlers:

- **Channel naming**: `sse:run:{runId}` — one channel per active run.
- **Payload schema** (JSON, validated in code):
  ```json
  {
    "seq": 42,
    "ts": "2026-06-27T15:00:00.123Z",
    "case_id": "01HXY...",
    "case_name": "POST /orders 201",
    "status": "pass" | "fail" | "error" | "skip",
    "duration_ms": 87,
    "summary": "expected 201, got 500"
  }
  ```
  Plus terminal events: `run.started`, `run.completed`, `run.cancelled`.
- **Sequence number** (`seq`) is a per-run monotonic counter assigned at publish time; clients use it to dedupe and detect gaps.
- **Subscription lifetime**: SSE handler subscribes on request start, unsubscribes on disconnect. Runner publishes each case event then `test_results` row insert.
- **Replay buffer**: when a subscriber joins mid-run, send the last 16 events from `sse:run:{runId}:buffer` (Redis LIST, capped with `LTRIM`).
- **Heartbeat**: server publishes `{"ts":...}` to `sse:run:{runId}:heartbeat` every 15s; SSE handler forwards as a comment line (`:`) so the connection survives proxies that close idle sockets.
- **Reconnect**: client uses `EventSource` natively; on resume it sends `Last-Event-ID` (the `seq`), and the server replays from buffer. If buffer is exhausted, server sends `run.resync` terminal event so the client does a full `GET /runs/:id/results` refresh.
- **Backpressure**: if a subscriber's outbound channel is full, the SSE handler closes the connection (status 503 + reason in log); the client retries with exponential backoff.

## Consequences

**Positive**
- Runner process can scale horizontally; SSE handler can be a separate binary without code changes.
- Reconnect is safe (sequence-based, no full restart needed for transient blips).
- Buffer of 16 covers typical transient drops (mobile network blip, dev tools reload).
- Heartbeat prevents silent disconnects behind reverse proxies.

**Negative**
- Adds a Redis dependency to the SSE path (already a hard dep per plan §1, but worth noting).
- Buffer + heartbeat add ~3 round-trips/sec per active run; negligible at the stated scale (<100 concurrent runs).
- The replay buffer is best-effort; if Redis evicts under memory pressure, late joiners get `run.resync` and must do a full refresh. Acceptable.

## Alternatives considered

- **SSE from runner → in-process channel only** — simpler, breaks the moment runner and listener are in different processes. Rejected.
- **WebSocket** — bidirectional, but the user only needs server→client push; SSE + reconnect is enough and survives more proxy setups. Rejected (also flagged out-of-scope per CLAUDE.md).
- **Long polling** — wastes connections, slower perceived latency. Rejected.
- **Server-Sent Events via Postgres `LISTEN/NOTIFY`** — possible but Postgres isn't in the stack (ADR-0003 chose MySQL). Rejected.

## Notes

- The Redis dependency is non-negotiable for Phase 3+; the user's `make install` already provisions Redis on port 6380.
- The `miniredis/v2` test double does **not** support pub/sub in all versions; if it lags, the runner test will use a real Redis container via `make test-integration`.
- A new file `internal/runner/sse_bus.go` will own the publish side; `internal/api/test_run.go` owns the subscribe side.