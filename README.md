# Sentinel

> Contract-driven API testing platform. Upload an OpenAPI spec → get a dynamic mock server, a regression test runner, and failure attribution reports. AI module is opt-in.

**Status:** Phase 5b (quality & polish).

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    User (Browser :5180)                  │
└─────────────────────┬───────────────────────────────────┘
                      │
          ┌───────────▼───────────┐
          │   Vue 3 SPA (Vite)    │
          │   Element Plus + Pinia│
          └───────────┬───────────┘
                      │ /api/v1 (proxy)
          ┌───────────▼───────────┐
          │   Go + Gin :8081      │
          │   ┌───────────────┐   │
          │   │ API handlers   │   │
          │   ├───────────────┤   │
          │   │ Services       │   │
          │   ├───────────────┤   │
          │   │ Repository     │   │
          │   └───────┬───────┘   │
          └───────────┼───────────┘
                      │
        ┌─────────────┼─────────────┐
   ┌────▼────┐  ┌─────▼────┐  ┌─────▼─────┐
   │ MySQL 8 │  │ Redis 7  │  │ AI Provider│
   │  :3307  │  │  :6380   │  │ (optional) │
   └─────────┘  └──────────┘  └───────────┘
```

**Core flow:** OpenAPI spec → register APIs → configure mock rules → author test cases → regression run → failure attribution (LLM or heuristic) → report + audit log.

## Tech stack

- **Backend:** Go 1.22 + Gin + GORM + MySQL 8 + Redis 7
- **Frontend:** Vue 3 + TypeScript + Vite + Element Plus + Pinia
- **Testing:** Go `testing` + testify; Vitest; Playwright
- **AI:** Anthropic Claude API (with `mock` provider for zero-cost dev)
- **Deployment:** Bare-metal — `scripts/start_all.sh` + optional systemd

## Quick start

Prerequisites: Go 1.22+, Node 18+, MySQL 8, Redis 7.

```bash
git clone https://github.com/glimjoe/sentinel-api-platform.git
cd sentinel
cp .env.example .env
make install       # Go modules + npm packages + migrations
make seed          # demo data (admin@sentinel.local / admin123)
make dev           # backend :8081, frontend :5180
```

Open <http://localhost:5180>.

## Demo

```bash
# Generate a demo GIF walkthrough (requires ImageMagick):
bash scripts/start_all.sh
make seed
npx tsx scripts/demo-screenshots.ts
convert -delay 150 -loop 0 demo-*.png demo.gif
```

> Run `make demo-gif` after starting the stack to capture a walkthrough GIF.

## Make targets

| Command | Description |
|---|---|
| `make install` | Install deps + run migrations |
| `make dev` | Start backend + frontend (Ctrl-C to stop) |
| `make stop` | Stop running services |
| `make test` | All Go tests (unit + integration) |
| `make test-coverage` | Coverage report (HTML at backend/coverage.html) |
| `make e2e` | Playwright E2E tests |
| `make lint` | Go vet + ESLint |
| `make migrate` | Apply DB migrations |
| `make seed` | Load demo data |
| `make doctor` | Verify environment |
| `bash scripts/dump.sh` | MySQL backup → `sentinel_YYYYMMDD_HHMMSS.sql.gz` |

## Test coverage

| Package | Coverage |
|---|---|
| service | 80.0% |
| repository | 82.0% |
| mock | 82.2% |
| contract | 91.9% |
| runner | 86.9% |
| **Total** | **83.5%** |

## API overview

```
Auth:     POST /api/v1/auth/{register,login,refresh,logout}   GET /api/v1/auth/me
Projects: GET|POST /api/v1/projects   GET|PATCH|DELETE /api/v1/projects/:pid
APIs:     GET|POST /api/v1/projects/:pid/apis   POST .../import-openapi
Mock:     POST /api/v1/rules   GET /api/v1/apis/:apiId/rules
Mock srv: ANY /mock/:projectSlug/*path
Cases:    GET|POST /api/v1/projects/:pid/cases
Runs:     POST /api/v1/projects/:pid/runs   POST .../start   GET .../stream (SSE)
AI:       POST /api/v1/projects/:pid/ai/{attribution,complete,prioritize}
```

## Documentation

- **ADRs:** `docs/adr/` (8 decision records)
- **Design:** `docs/design/DESIGN.md` (C4 + sequence diagrams)
- **AI workflow:** `docs/ai-workflow/`
- **API spec:** `docs/api/openapi.yaml`

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md). Conventional commits required.

## License

[MIT](./LICENSE)

---

**Owner:** glimjoe | **Phases:** 6 (0–5a complete, 5b in progress)
