# ADR-0001: Backend language — Go 1.22 + Gin

**Status:** Accepted (2026-06-27)
**Phase:** 0 (decision pre-implementation)

## Context

- The user is a Python-first test/dev engineer transitioning to full-stack, and **wants to add Go to the resume** for marketability.
- The target JD ("any of Java/Node/Python/Go") accepts Go, and the rest of the project (GitHub, Cloudflare, Kubernetes, infrastructure-as-code) increasingly skews Go.
- Three candidates: Go + Gin, Java + Spring Boot, Python + FastAPI.
- The reference project `~/new1/` is Python+FastAPI; using Python again would be the fastest but would not advance the user's learning goal.

## Decision

Use **Go 1.22 + Gin** for the backend, with the following libraries pinned:
- HTTP: `gin-gonic/gin` v1.10.0
- ORM: `gorm.io/gorm` v1.25 + `gorm.io/driver/mysql`
- Config: `spf13/viper` v1.19
- Logging: `go.uber.org/zap` v1.27
- Validation: `go-playground/validator/v10`
- Test: `stretchr/testify`, `alicebob/miniredis/v2`, `golang/mock`

## Consequences

**Positive**
- Single-binary deployment (no runtime dependencies on the target box) — matches the no-Docker environment.
- Goroutines make the test runner's parallel mode straightforward.
- Strong static typing and small memory footprint; `go test -cover` is first-class.
- Adds a JD-relevant language to the user's resume.

**Negative**
- Smaller standard library for OpenAPI parsing; relies on `github.com/getkin/kin-openapi`.
- GORM is good but has footguns (N+1, soft deletes); user must learn to read generated SQL.
- User is learning the language; first-week velocity will be lower than Python.

## Alternatives considered

- **Java + Spring Boot** — heavier, slower JVM startup, overkill at this scale. Would add Java (which user already knows) but not a *new* language.
- **Python + FastAPI** — fastest to write, but fails the user's explicit "learn Go" goal. Rejected.

## Notes

- If Go learning friction exceeds 1 week, revisit at end of Phase 1.
- The AI module uses an LLM client abstraction so swapping Go SDKs (anthropic-sdk-go vs go-openai) costs little.
