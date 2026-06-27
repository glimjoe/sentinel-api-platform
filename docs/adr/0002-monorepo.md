# ADR-0002: Repository layout — monorepo

**Status:** Accepted (2026-06-27)
**Phase:** 0

## Context

- The reference project `~/new1/` uses a monorepo with `frontend/` + `packages/<pkg>/` (Python installable packages with `src/` layout).
- Frontend and backend evolve together; many TypeScript types mirror Go structs.
- A single repo simplifies the CV narrative: one project, one README, one commit graph.

## Decision

Use a **monorepo** with two top-level trees: `backend/` (Go) and `frontend/` (Vue 3). Shared assets at the root:
- `docs/` (ADRs, AI workflow, design)
- `scripts/` (install, dev, seed, doctor, build)
- `Makefile` (entry point for all common operations)
- `.env.example`, `.gitignore`, `.editorconfig`, `CLAUDE.md`, `CONTRIBUTING.md`

## Consequences

**Positive**
- Atomic changes across frontend + backend (one PR, one CI run).
- Single `git log` for the full project history — easier to tell a coherent story.
- One CI pipeline covers both.
- Shared `Makefile` is the single entry point for newcomers.

**Negative**
- Larger checkout than split repos (mitigated: `node_modules` and `bin/` are git-ignored).
- Risk of coupling — mitigated by the `backend/` ↔ `frontend/` boundary enforced by directory structure (no cross-imports).

## Alternatives considered

- **Two repos with git submodules** — too much friction for the marginal benefit.
- **Polyrepo with separate CI** — splits the narrative for recruiters reviewing the project. Rejected.

## Notes

- If the project grew past 50 contributors, consider splitting `frontend/` and `backend/` into separate repos with API contract tests between them.
