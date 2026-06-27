# CLAUDE.md — Sentinel project guide for AI assistants

> This file is the operating manual for any AI agent (Claude Code, Copilot, etc.) working on this repo. Read it before making changes. It encodes conventions, structure, and AI workflow expectations.

## Project at a glance

- **Purpose:** Contract-driven API testing platform. OpenAPI → mock server + test runner + failure attribution.
- **Owner:** yangwei (full-stack job-hunt portfolio piece)
- **Phase:** see `/.claude/plans/expressive-forging-hare.md` (or current plan file)
- **Status convention:** every PR / commit should reference the phase it advances.

## Reading order for new sessions

1. `/.claude/plans/<latest>.md` — current plan, phase breakdown, DoD
2. `README.md` — project surface
3. `docs/adr/` — read the 5 ADRs at minimum
4. `docs/ai-workflow/README.md` — how AI is used in this repo
5. Module-specific CLAUDE.md if it exists (none yet; create one when a module gets > 1000 LoC)

## Conventions

- **Git:** Conventional Commits, branch prefix `feat/`. AI-assisted commits must end with `AI-Assisted: [claude-code|copilot|manual]`.
- **Go:** `gofmt` + `goimports`. GORM models carry json tags. Errors are wrapped with `fmt.Errorf("...: %w", err)`. Domain errors live in `internal/pkg/errs`.
- **Vue:** TypeScript strict mode. Pinia for state, Element Plus for UI primitives. No `any` unless annotated with `// @ts-expect-error` and a reason.
- **Paths:** use `path/filepath` in Go (forward slashes internally). In bash scripts, always use `"$REPO_ROOT"` quoted.
- **Env vars:** read in `internal/pkg/config` only; never `os.Getenv` scattered through code.

## Build / test commands

```bash
make install            # one-time setup
make doctor             # health check
make dev                # start backend + frontend
make test               # all Go tests
make test-coverage      # coverage report (HTML in backend/coverage.html)
make e2e                # Playwright (requires `make dev` running)
make lint               # vet + eslint
make migrate            # apply DB migrations
make seed               # load demo data
```

## When making changes

1. **Before writing code:** update or add an ADR in `docs/adr/` if the change is architectural. Format: `NNNN-short-slug.md` (see ADR-0001 template).
2. **Test-first for new logic:** write the test, see it fail, then implement. Coverage floor: 80% on `service/`, `repository/`, `mock/`, `contract/`, `runner/`.
3. **Migrations:** add a new file in `backend/migrations/NNNN_description.sql`. Never edit a checked-in migration.
4. **Commit message format:**
   ```
   <type>(<scope>): <subject>

   <body explaining why, not what>

   AI-Assisted: claude-code
   AI-Model: claude-sonnet-4-6
   ```
5. **Never** commit secrets, `.env`, or `coverage.out`. `.gitignore` already covers these — verify with `git status --ignored` if unsure.

## AI integration

- AI features (LLM calls) live in `backend/internal/ai/`. They are **never** called at import time; only via service handlers.
- The default provider in dev is `mock` (deterministic, zero cost). Real providers require an API key in `.env`.
- The AI module is feature-flagged with `AI_ENABLED=false` to fully hide the buttons and 503 the endpoints.
- All prompt templates live in `docs/ai-workflow/prompts/*.md` — never hard-code long prompts in Go source.

## Prompt patterns

When asking AI to write code, use the structure:
1. **Goal** — one sentence.
2. **Context** — file paths, current state, what already exists.
3. **Constraints** — naming, style, must-not-touch files, library version pins.
4. **Output** — expected function signature, test names, or PR diff.
5. **Verification** — how you'll know it works (test command, curl call).

Example prompt template: `docs/ai-workflow/prompts/code-gen.md`.

## Out of scope (do not add without asking)

- New language runtimes (we chose Go + Node).
- New UI libraries beyond Element Plus + ECharts + Monaco.
- Container infrastructure (no Docker, no k8s).
- Authentication providers other than local JWT (no OAuth/SAML until post-launch).
- New database engines (we chose MySQL 8).
- WebSocket / gRPC support (out of v1).
- Multi-tenant isolation (single-org per deployment until v1.1).

## When stuck

1. Read the relevant ADR first. Most decisions are pre-made.
2. Read the 95KB plan in `/tmp/plan-agent-output.md` (sibling of the active plan) for deep implementation details.
3. Search the codebase with `grep -rn` rather than guessing.
4. If still unclear, **ask the user before guessing** (per user-global rule 10: strict confirmation).

## Commit signature

When AI is the author or co-author of code, the commit's trailer must include:
```
AI-Assisted: claude-code
```
This is non-negotiable. The user will use this tag in interviews to show AI integration as a *deliberate workflow*, not a black box.
