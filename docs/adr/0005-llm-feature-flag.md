# ADR-0005: AI / LLM — feature-flagged, mock-by-default in dev

**Status:** Accepted (2026-06-27)
**Phase:** 4 (informs Phase 1 architecture)

## Context

- AI features are a JD requirement (responsibility #4 and #6) but are a recurring cost.
- During development, LLM calls slow the loop and burn tokens.
- The AI module is *one feature* of Sentinel, not the project's main value prop.
- The user wants to demonstrate "graceful degradation" in interviews — what happens when the LLM is down, out of budget, or disabled?

## Decision

Treat the LLM as a **feature-flagged, swappable component**:
- `AI_ENABLED` (bool): master switch; when `false`, AI UI is hidden and `/api/v1/ai/*` returns 503
- `AI_PROVIDER`: `mock` (default, zero cost) / `noop` (disabled) / `anthropic` / `openai`
- API keys read from env: `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`
- Model selection: `AI_MODEL_ATTRIBUTOR`, `AI_MODEL_COMPLETER`, `AI_MODEL_PRIORITIZER` — swappable per feature
- Cost guard: `internal/ai/guard.go` enforces `AI_DAILY_LIMIT_USD` and `AI_MONTHLY_LIMIT_USD` from the `ai_budget` table
- Response cache: Redis keyed by SHA-256 of the prompt, TTL `AI_CACHE_TTL_SECONDS` (default 1h)
- All long prompts live in `docs/ai-workflow/prompts/*.md`, not in Go source
- Mock provider is **deterministic** (no randomness) — enables reliable tests

## Consequences

**Positive**
- Zero-cost development; no API key required to run the project.
- The project works fully without LLM — AI is an enhancer, not a dependency.
- Cost guardrails prevent runaway spend if a developer forgets to disable it.
- Graceful degradation demonstrable in interviews.
- Prompt template versioning is independent of code versioning (no PR needed to tweak a prompt).

**Negative**
- Tests must cover both paths (mock and real) — mitigated by the `Client` interface and a small test matrix.
- Cached responses may go stale when the LLM improves; accept the staleness as a cost-saving feature.

## Alternatives considered

- **Always-on LLM** — would burn tokens and break the "works without API key" property. Rejected.
- **No LLM at all** — fails JD requirement. Rejected.
- **Embed a small local model (e.g. ggml)** — would add GBs of dependencies and degrade the user experience. Rejected for v1.

## Notes

- The mock provider's behavior is asserted in `backend/internal/ai/mock_client_test.go` and used by all E2E flows.
- A future "model upgrade" workflow: bump `AI_MODEL_ATTRIBUTOR`, monitor the `ai_usage` table, retire the old model after 30 days.
