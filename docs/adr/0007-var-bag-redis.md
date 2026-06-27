# ADR-0007: Mock variable bag — Redis-backed

**Status:** Accepted (2026-06-27)
**Phase:** 2 (required at Mock engine MVP)

## Context

- Plan §6.6 mock engine includes an `extractor` that pulls variables from response bodies (JSONPath/regex) for use in subsequent requests via `{{var}}` placeholders.
- The mock endpoint `ANY /mock/:projectSlug/*path` (§8) is a public, stateless HTTP route — any given user's session spans multiple HTTP requests with no in-protocol correlation.
- Without a persistent bag, three failure modes appear:
  1. **Process restart** wipes extracted variables mid-flow, breaking "next request sends `Authorization: Bearer {{login_token}}`".
  2. **Horizontal scale** — the runner pulls an extra backend binary per §14.3 answer 4; an in-process map diverges across replicas.
  3. **Single-process + multi-tenant** — multiple users hitting `/mock/<slug>/*` simultaneously would see each other's tokens.

The architect review (2026-06-27) flagged this as **high-severity**; user decision on 2026-06-27 was "直接上 Redis" (skip the single-process-memory MVP and go straight to Redis).

## Decision

Store the variable bag in **Redis**, scoped by an explicit **`session_id`** that the client passes between requests:

- **Key shape**: `mock:vars:{projectSlug}:{sessionId}` — Redis HASH.
  - field: variable name (e.g. `login_token`)
  - value: extracted string
- **Session ID lifecycle**:
  - Created on the **first** mock request that omits `X-Mock-Session`; the server returns `X-Mock-Session: <ulid>` in the response so the client can persist it.
  - TTL: 30 minutes sliding (refresh on every read/write); cleared by explicit `DELETE /mock/:slug/session` (not in MVP) or natural expiry.
- **Client contract**:
  - Frontend's `MockConsole` view (§7.1) generates and stores the session ID in component state.
  - API clients (curl, Postman) must propagate `X-Mock-Session` across calls or pass `-H "X-Mock-Session: $(uuidgen)"` once and reuse.
- **Extraction source-of-truth**: `mock_rules.extractor_json` — a small JSON: `{"path": "$.data.token", "as": "login_token", "from": "response.body"}`. Multiple extractors per rule allowed; processed in declared order.
- **Substitution**: `{{var}}` substitution happens **after** rule matching and **before** the response body is returned, for both response body and (if the rule has one) request templating in chained rules.
- **Conflict rule**: later extractor wins (last-write-wins per field within the same request).
- **Eviction**: at TTL expiry; no explicit LRU.

## Consequences

**Positive**
- Multi-process and multi-replica deployments just work — Redis is the shared store.
- Process restart does not invalidate active sessions.
- Cross-tenant isolation is enforced by the key prefix `mock:vars:{projectSlug}`.
- TTL bounds memory growth even when clients abandon sessions.

**Negative**
- Adds one Redis round-trip per mock request with `{{var}}` references (cache lookup) and one write per extraction. Acceptable: a single hash GET is sub-millisecond on local Redis.
- Clients must opt in to session continuity (no implicit per-IP session); this is a small UX cost but matches how test suites naturally work (each suite gets a fresh session ID).
- `miniredis/v2` is already used in tests; this design does not require pub/sub from it, so the existing test double works.

## Alternatives considered

- **In-process `sync.Map`** — simplest, but fails the three failure modes above. Rejected per user decision.
- **MySQL table `mock_vars`** — durable but slower than Redis for hot path; Redis is already required. Rejected.
- **Cookie-based session** — adds cookie parsing to a public mock endpoint; cookies don't survive curl scripts well and complicate `MockConsole` UX. Rejected.
- **JWT carrying the bag** — bag grows unbounded; tokens become fat. Rejected.

## Notes

- A new file `internal/mock/varbag.go` owns Redis I/O. The interface is `VarBag.Get(ctx, slug, session) / .Set(ctx, slug, session, name, value) / .Substitute(ctx, slug, session, template)`.
- `internal/mock/engine.go` calls `VarBag` after rule matching but before returning the response.
- When `AI_ENABLED=false` and the project hasn't configured extraction rules, the bag stays empty; the `{{var}}` substitution is a no-op. The MockConsole must show a banner if a referenced variable is missing.
- One unit test asserts that two processes (same Redis, different Go test binaries via `t.Parallel` + helper goroutine) see the same bag — proving the cross-process property.