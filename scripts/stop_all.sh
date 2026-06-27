#!/usr/bin/env bash
# Stop all Sentinel services started by start_all.sh.
# Phase 0 stub: noop.

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

C_OK="\033[32m"; C_RST="\033[0m"
[[ -t 1 ]] || { C_OK=""; C_RST=""; }

printf "${C_OK}[stop]${C_RST} Sentinel shutdown\n"

# Future (Phase 1+): read pid files and kill.
#   for pidfile in backend/.pid frontend/.dev.pid; do
#     [[ -f "$REPO_ROOT/$pidfile" ]] && kill "$(cat "$REPO_ROOT/$pidfile")" 2>/dev/null || true
#     rm -f "$REPO_ROOT/$pidfile"
#   done

printf "${C_OK}[stop]${C_RST} Phase 0 stub: nothing to stop.\n"
