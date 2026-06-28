#!/usr/bin/env bash
# Start Sentinel backend (Go) and frontend (Vite) in background.
# Idempotent: refuses to re-start if already running.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

# shellcheck disable=SC1091
[[ -f "$REPO_ROOT/.env" ]] && set -a && source "$REPO_ROOT/.env" && set +a

APP_PORT="${APP_PORT:-8081}"
FRONTEND_PORT="${FRONTEND_PORT:-5180}"
PID_DIR="$REPO_ROOT/.pids"
mkdir -p "$PID_DIR"

C_OK="\033[32m"; C_WARN="\033[33m"; C_RST="\033[0m"
[[ -t 1 ]] || { C_OK=""; C_WARN=""; C_RST=""; }

cleanup() {
  printf "\n${C_WARN}[start]${C_RST} shutting down...\n"
  bash "$SCRIPT_DIR/stop_all.sh"
}
trap cleanup EXIT INT TERM

# ── Backend ───────────────────────────────────────────────────────────
if [[ -f "$PID_DIR/backend.pid" ]] && kill -0 "$(cat "$PID_DIR/backend.pid")" 2>/dev/null; then
  printf "${C_WARN}[start]${C_RST} backend already running (pid %s)\n" "$(cat "$PID_DIR/backend.pid")"
else
  printf "${C_OK}[start]${C_RST} starting backend on :%s...\n" "$APP_PORT"
  cd "$REPO_ROOT/backend"
  go run ./cmd/server &
  echo $! > "$PID_DIR/backend.pid"
  cd "$REPO_ROOT"
  for _ in $(seq 1 30); do
    if curl -s "http://localhost:$APP_PORT/api/v1/healthz" >/dev/null 2>&1; then
      printf "${C_OK}[start]${C_RST} backend ready (pid %s)\n" "$(cat "$PID_DIR/backend.pid")"
      break
    fi
    sleep 1
  done
fi

# ── Frontend ──────────────────────────────────────────────────────────
if [[ -f "$PID_DIR/frontend.pid" ]] && kill -0 "$(cat "$PID_DIR/frontend.pid")" 2>/dev/null; then
  printf "${C_WARN}[start]${C_RST} frontend already running (pid %s)\n" "$(cat "$PID_DIR/frontend.pid")"
else
  printf "${C_OK}[start]${C_RST} starting frontend on :%s...\n" "$FRONTEND_PORT"
  cd "$REPO_ROOT/frontend"
  npm run dev &
  echo $! > "$PID_DIR/frontend.pid"
  cd "$REPO_ROOT"
  for _ in $(seq 1 30); do
    if curl -s "http://localhost:$FRONTEND_PORT" >/dev/null 2>&1; then
      printf "${C_OK}[start]${C_RST} frontend ready (pid %s)\n" "$(cat "$PID_DIR/frontend.pid")"
      break
    fi
    sleep 1
  done
fi

printf "${C_OK}[start]${C_RST} all services running. Press Ctrl-C to stop.\n"
printf "  backend:  http://localhost:%s\n" "$APP_PORT"
printf "  frontend: http://localhost:%s\n" "$FRONTEND_PORT"
wait
