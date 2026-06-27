# Sentinel

> Contract-driven API testing platform. Upload an OpenAPI spec → get a dynamic mock server, a regression test runner, and failure attribution reports. AI module is opt-in.

**Status:** Phase 0 (scaffolding). See `docs/design/DESIGN.md` (Phase 5) for the full architecture.

## Why

Engineering teams lose hours to fragmented tooling — Postman for ad-hoc requests, Swagger for docs, Mockaroo for static mocks, JMeter for load, pytest for unit tests, k6 for chaos. None of them close the **contract → test → report** loop. Sentinel does.

## What it does

```
OpenAPI spec ──▶ register APIs ──▶ configure mock rules ──▶ author/AI-generate test cases
   │                                                                    │
   ▼                                                                    ▼
   /mock/:slug/*path ◀── dynamic mock server                  regression run
                                                                    │
                                                                    ▼
                                                          failure attribution (LLM or heuristic)
                                                                    │
                                                                    ▼
                                                              report + audit log
```

## Tech stack

- **Backend:** Go 1.22 + Gin + GORM + MySQL 8 + Redis 7
- **Frontend:** Vue 3 + TypeScript + Vite + Element Plus + Pinia
- **Testing:** Go `testing` + testify; Vitest; Playwright
- **AI:** Anthropic Claude API (with `mock` provider for zero-cost dev)
- **Deployment:** Bare-metal (no Docker) — `scripts/start_all.sh` + optional systemd

## Quick start (development)

Prerequisites: Go 1.22+, Node 18+, MySQL 8, Redis 7.

```bash
git clone https://github.com/glimjoe/sentinel-api-platform.git
cd sentinel
cp .env.example .env
make install       # install Go modules + npm packages + run migrations
make seed          # load demo data (admin@sentinel.local / admin123)
make dev           # starts backend on :8081, frontend on :5180
```

Open <http://localhost:5180>.

## Documentation

- **Architecture decision records:** `docs/adr/`
- **AI workflow / methodology:** `docs/ai-workflow/`
- **Full design doc (zh, written in Phase 5):** `docs/design/DESIGN.md`
- **OpenAPI export (auto-generated):** `docs/api/openapi.yaml`

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md). Conventional commits required. AI-assisted commits must be tagged — see `docs/ai-workflow/README.md`.

## License

[MIT](./LICENSE)

## Project meta

- **Owner:** glimjoe
- **Target role:** Full-stack engineer
- **Phases:** 6 (see `$HOME/.claude/plans/expressive-forging-hare.md`)
- **Total estimated effort:** 7–8 weeks
