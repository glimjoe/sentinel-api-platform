#!/usr/bin/env bash
# Build production artifacts: backend binary + frontend bundle.
# Phase 0 stub: backend/ and frontend/ not present yet.

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

C_WARN="\033[33m"; C_RST="\033[0m"
[[ -t 1 ]] || { C_WARN=""; C_RST=""; }

printf "${C_WARN}[build]${C_RST} Phase 0 stub — no artifacts to build. Phase 1+ will produce:\n"
printf "  • backend/bin/sentinel          (Go static binary)\n"
printf "  • frontend/dist/                (Vite production bundle)\n"
