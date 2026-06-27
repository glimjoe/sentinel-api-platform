# ─── Sentinel Makefile ─────────────────────────────────────────────────────────
# Run from repo root. Self-documenting: `make help`.

# --- Detect OS / shell ------------------------------------------------------------
SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c

# --- Variables (override on command line, e.g. `make dev PORT=9090`) ------------
APP_PORT      ?= 8081
FRONTEND_PORT ?= 5180
GO_BIN        ?= go
NPM_BIN       ?= npm

# --- Phony targets ---------------------------------------------------------------
.PHONY: help install dev build test test-unit test-integration test-coverage \
        e2e lint format migrate seed clean stop logs doctor

# --- Help (default) --------------------------------------------------------------
help: ## Show this help message
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# --- Setup -----------------------------------------------------------------------
install: ## Install all dependencies (Go modules + npm packages) and run migrations
	@bash scripts/install.sh

doctor: ## Verify environment (go, node, mysql, redis, ports)
	@bash scripts/doctor.sh

# --- Development -----------------------------------------------------------------
dev: ## Start backend + frontend (foreground; Ctrl-C to stop)
	@bash scripts/start_all.sh

stop: ## Stop all running services started by `make dev`
	@bash scripts/stop_all.sh

build: ## Build production binaries (backend binary + frontend bundle)
	@bash scripts/build.sh

# --- Database --------------------------------------------------------------------
migrate: ## Apply database migrations
	@cd backend && $(GO_BIN) run ./cmd/migrate up

migrate-down: ## Roll back the most recent migration
	@cd backend && $(GO_BIN) run ./cmd/migrate down

migrate-status: ## Show migration status
	@cd backend && $(GO_BIN) run ./cmd/migrate status

seed: ## Load demo data (admin user, sample project, APIs, cases, mock rules)
	@bash scripts/seed.sh

# --- Testing ---------------------------------------------------------------------
test: test-unit test-integration ## Run all Go tests (unit + integration)

test-unit: ## Run Go unit tests only
	@cd backend && $(GO_BIN) test -short -race -count=1 ./...

test-integration: ## Run Go integration tests (requires MySQL+Redis on default ports)
	@cd backend && $(GO_BIN) test -race -count=1 -tags=integration ./tests/...

test-coverage: ## Run Go tests with coverage report
	@cd backend && $(GO_BIN) test -short -race -count=1 -coverprofile=coverage.out -covermode=atomic ./...
	@cd backend && $(GO_BIN) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: backend/coverage.html"

e2e: ## Run Playwright E2E tests (requires `make dev` running)
	@cd frontend && $(NPM_BIN) run test:e2e

frontend-test: ## Run Vue unit tests (Vitest)
	@cd frontend && $(NPM_BIN) run test:unit

# --- Quality ---------------------------------------------------------------------
lint: ## Run all linters (Go: golangci-lint; Vue: eslint)
	@cd backend && $(GO_BIN) vet ./...
	@cd frontend && $(NPM_BIN) run lint

format: ## Auto-format all code (Go: gofmt; Vue: prettier)
	@cd backend && $(GO_BIN) fmt ./...
	@cd frontend && $(NPM_BIN) run format

# --- Operations ------------------------------------------------------------------
logs: ## Tail backend logs
	@tail -n 200 -f backend/storage/logs/app.log 2>/dev/null || echo "no log file yet"

clean: ## Remove build artifacts and local caches (keeps .env and DB)
	@rm -rf backend/bin backend/coverage.out backend/coverage.html
	@rm -rf frontend/dist frontend/coverage
	@rm -rf backend/storage/.tmp
	@echo "Cleaned."
