# Contributing to Sentinel

Thanks for your interest. Sentinel is currently a single-author portfolio project, but the conventions below are written as if a team were contributing.

## Ground rules

- **Code quality is non-negotiable.** This project is the user's primary job-search artifact. Every commit is part of that story.
- **Tests come with the code.** Coverage floor: 80% on `service/`, `repository/`, `mock/`, `contract/`, `runner/`.
- **Documentation is part of the deliverable.** New features update `README.md`, relevant ADRs, and `docs/ai-workflow/`.
- **AI assistance is welcome but must be transparent.** Every AI-assisted commit ends with `AI-Assisted: [tool]`.

## Workflow

1. **Branch:** `feat/<short-slug>` (e.g. `feat/mock-engine`).
2. **Develop in small commits** — each commit should be reviewable in 5 minutes.
3. **Pre-commit checklist:**
   - [ ] `make lint` passes
   - [ ] `make test-unit` passes
   - [ ] Migration added (if schema changed)
   - [ ] ADR added/updated (if architecture changed)
   - [ ] Commit message follows Conventional Commits
4. **Push and review** (self-review is fine for solo work).
5. **Merge to `main`** with a descriptive squash message.

## Conventional Commits

```
<type>(<scope>): <subject>

<body explaining why, not what>

<footer with AI-Assisted tag and any issue refs>
```

**Types:** `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `perf`, `build`, `ci`.

**Scopes (in this repo):** `backend`, `frontend`, `mock`, `contract`, `runner`, `ai`, `db`, `auth`, `docs`, `ci`, `scripts`.

**Example:**
```
feat(mock): add variable extraction across requests

Allow a mock rule to extract a value from a response body (JSONPath
or regex) and inject it into subsequent requests via {{var}} syntax.
This unlocks chained auth flows (e.g. login → use token).

AI-Assisted: claude-code
AI-Model: claude-sonnet-4-6
```

## Coding standards

### Go

- `gofmt` (run `make format`).
- Errors are wrapped with `fmt.Errorf("doing X: %w", err)`. Never `log.Fatal` in library code; only `cmd/*` may exit.
- Prefer `context.Context` as the first parameter of any function that does I/O.
- Use the `internal/` directory to keep public surface area zero.
- Domain errors: define typed errors in `internal/pkg/errs` and map to HTTP in middleware.

### Vue / TypeScript

- TypeScript strict mode on (`"strict": true` in `tsconfig.json`).
- Components: PascalCase files, `<script setup lang="ts">`.
- Pinia stores: one file per resource (`stores/auth.ts`, `stores/project.ts`).
- Composables: `use<Thing>` naming, return refs not reactive objects.
- Imports: external first, then `@/`, then relative. Sort alphabetically within groups.

### Database

- All IDs are `CHAR(26)` ULIDs.
- All timestamps are `DATETIME(3)` UTC.
- All `created_at` / `updated_at` columns are managed by GORM.
- Soft deletes via `gorm.DeletedAt` for `users`, `projects`, `apis`, `test_cases`, `test_runs`.

## Testing

- **Unit tests:** `*_test.go` next to source. Use `testify/require` for fail-fast.
- **Integration tests:** `backend/tests/integration/*_test.go` with build tag `//go:build integration`. Require real MySQL+Redis.
- **E2E tests:** `frontend/tests/e2e/*.spec.ts`. Each test is one user journey, not one feature.

## Pull request template

A PR description should answer:
- What does this change?
- Why is it needed?
- How was it verified? (test commands run, manual checks, screenshots)
- Any follow-up work?

## Releases (post-launch)

Semantic versioning: `MAJOR.MINOR.PATCH`.
- `MAJOR`: breaking API change.
- `MINOR`: new feature, backward-compatible.
- `PATCH`: bug fix.

Changelog: `CHANGELOG.md` (generated in Phase 5).
