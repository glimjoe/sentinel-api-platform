# ADR-0003: Database — MySQL 8

**Status:** Accepted (2026-06-27)
**Phase:** 0

## Context

- MySQL 8 is available via `apt` on Ubuntu 24.04; PostgreSQL is not pre-installed.
- The data model is simple relational: 13 tables, mostly FK relationships, with JSON columns for OpenAPI payloads and match rules.
- JSON Schema validation is done in Go (`gojsonschema`); we don't need Postgres's JSONB query operators.

## Decision

Use **MySQL 8** with:
- charset `utf8mb4`, collation `utf8mb4_unicode_ci`
- engine `InnoDB`
- all IDs as `CHAR(26)` ULIDs (no native UUID; we accept the string storage cost)
- all timestamps as `DATETIME(3)` UTC, managed by GORM
- composite indexes for hot paths (see `backend/migrations/0002_indexes.sql`)

Listen on **port 3307** to avoid conflicts with the default MySQL install on 3306 and with `~/new1/`.

## Consequences

**Positive**
- One-step install via `apt install mysql-server`.
- Familiar SQL dialect; well-documented GORM driver.
- JSON column type sufficient for OpenAPI spec storage (read/write as opaque blobs).

**Negative**
- No native JSON path operators like Postgres. We compensate with explicit JSON Schema validation in Go.
- No native UUID type; `CHAR(26)` is a few bytes larger per row.
- No partial indexes; we use composite indexes instead.

## Alternatives considered

- **PostgreSQL** — would give better JSON ops and native UUID, but adds install step and is not on the box. Rejected for now; revisit if performance becomes an issue.
- **SQLite** — no shared-write semantics; ruled out for any multi-process scenario.
