#!/usr/bin/env bash
# Stop all Sentinel services started by start_all.sh.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
PID_DIR="$REPO_ROOT/.pids"

C_OK="\033[32m"; C_WARN="\033[33m"; C_RST="\033[0m"
[[ -t 1 ]] || { C_OK=""; C_WARN=""; C_RST=""; }

stopped=0

for svc in backend frontend; do
  pidfile="$PID_DIR/${svc}.pid"
  if [[ -f "$pidfile" ]]; then
    pid=$(cat "$pidfile")
    if kill -0 "$pid" 2>/dev/null; then
      kill "$pid" 2>/dev/null || true
      printf "${C_OK}[stop]${C_RST} %s (pid %s) terminated\n" "$svc" "$pid"
      stopped=$((stopped + 1))
    else
      printf "${C_WARN}[stop]${C_RST} %s pid %s not running (stale pid)\n" "$svc" "$pid"
    fi
    rm -f "$pidfile"
  fi
done

# Fallback: kill processes on our ports.
APP_PORT="${APP_PORT:-8081}"
FRONTEND_PORT="${FRONTEND_PORT:-5180}"

for port in "$APP_PORT" "$FRONTEND_PORT"; do
  if pids=$(lsof -ti "tcp:$port" 2>/dev/null); then
    for p in $pids; do
      kill "$p" 2>/dev/null || true
      printf "${C_OK}[stop]${C_RST} port %s freed (pid %s)\n" "$port" "$p"
      stopped=$((stopped + 1))
    done
  fi
done

[[ $stopped -eq 0 ]] && printf "${C_WARN}[stop]${C_RST} nothing was running\n"
