#!/usr/bin/env bash
# Start backend (Go) and frontend (Vite dev) in background.
# Idempotent: refuses to start if ports are already bound by another sentinel.
# Phase 0 stub: warns and exits 0 because backend/frontend code is not yet present.

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

# shellcheck disable=SC1091
[[ -f "$REPO_ROOT/.env" ]] && set -a && source "$REPO_ROOT/.env" && set +a

APP_PORT="${APP_PORT:-8081}"
FRONTEND_PORT="${FRONTEND_PORT:-5180}"

C_OK="\033[32m"; C_WARN="\033[33m"; C_RST="\033[0m"
[[ -t 1 ]] || { C_OK=""; C_WARN=""; C_RST=""; }

printf "${C_OK}[start]${C_RST} Sentinel dev launcher\n"
printf "  backend:  :%s  (Go, scaffolded in Phase 1)\n" "$APP_PORT"
printf "  frontend: :%s  (Vite, scaffolded in Phase 1)\n" "$FRONTEND_PORT"
printf "  data:     MySQL :%s, Redis :%s\n" "${MYSQL_PORT:-3307}" "${REDIS_PORT:-6379}"

if [[ ! -d "$REPO_ROOT/backend" ]] || [[ ! -d "$REPO_ROOT/frontend" ]]; then
  printf "${C_WARN}[start]${C_RST} backend/ or frontend/ not yet present. Skipping. Phase 0 stub.\n"
  exit 0
fi

# Future (Phase 1+):
#   cd backend && go run ./cmd/server >"$REPO_ROOT/backend/storage/logs/app.log" 2>&1 & echo $! >"$REPO_ROOT/backend/.pid"
#   cd frontend && npm run dev >"$REPO_ROOT/frontend/.dev.log" 2>&1 & echo $! >"$REPO_ROOT/frontend/.dev.pid"
#   trap cleanup EXIT
