#!/usr/bin/env bash
# Sentinel installer — idempotent. Safe to re-run.
# Phase 0 scope: verify environment only. Real module installs land in Phase 1.

set -euo pipefail

# Always run from repo root, regardless of where this script is invoked.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

# ─── Colors (only when interactive) ────────────────────────────────────────────
if [[ -t 1 ]]; then
  C_OK="\033[32m"; C_WARN="\033[33m"; C_ERR="\033[31m"; C_RST="\033[0m"; C_DIM="\033[2m"
else
  C_OK=""; C_WARN=""; C_ERR=""; C_RST=""; C_DIM=""
fi

info() { printf "${C_DIM}[install]${C_RST} %s\n" "$*"; }
ok()   { printf "${C_OK}✓${C_RST} %s\n" "$*"; }
warn() { printf "${C_WARN}!${C_RST} %s\n" "$*"; }
die()  { printf "${C_ERR}✗${C_RST} %s\n" "$*" >&2; exit 1; }

# ─── 1. Environment binary checks ─────────────────────────────────────────────
info "Checking environment..."

command -v go      >/dev/null 2>&1 || die "go not found. Install Go 1.22+ (apt: golang-go)."
command -v node    >/dev/null 2>&1 || die "node not found. Install Node 18+."
command -v npm     >/dev/null 2>&1 || die "npm not found."
command -v mysql   >/dev/null 2>&1 || die "mysql client not found. apt install mysql-client (or mysql-server)."
command -v redis-cli >/dev/null 2>&1 || die "redis-cli not found. apt install redis-tools (or redis-server)."

GO_VERSION="$(go version | awk '{print $3}' | sed 's/go//')"
GO_MAJOR="$(echo "$GO_VERSION" | cut -d. -f1)"
GO_MINOR="$(echo "$GO_VERSION" | cut -d. -f2)"
if [[ "$GO_MAJOR" -lt 1 ]] || { [[ "$GO_MAJOR" -eq 1 ]] && [[ "$GO_MINOR" -lt 22 ]]; }; then
  die "Go >= 1.22 required (found $GO_VERSION)."
fi
ok "go $GO_VERSION"

NODE_VERSION="$(node -v | tr -d 'v')"
NODE_MAJOR="$(echo "$NODE_VERSION" | cut -d. -f1)"
if [[ "$NODE_MAJOR" -lt 18 ]]; then
  die "Node >= 18 required (found $NODE_VERSION)."
fi
ok "node $NODE_VERSION"
ok "npm $(npm -v)"

# ─── 2. .env presence ─────────────────────────────────────────────────────────
if [[ ! -f "$REPO_ROOT/.env" ]]; then
  warn ".env not found. Copying from .env.example..."
  cp "$REPO_ROOT/.env.example" "$REPO_ROOT/.env"
  warn "Edit .env to set real JWT secrets before going to production."
else
  ok ".env present"
fi

# ─── 3. Service reachability (uses .env values) ───────────────────────────────
# shellcheck disable=SC1091
set -a; source "$REPO_ROOT/.env"; set +a

MYSQL_HOST="${MYSQL_HOST:-127.0.0.1}"
MYSQL_PORT="${MYSQL_PORT:-3307}"
MYSQL_USER="${MYSQL_USER:-root}"
MYSQL_PASSWORD="${MYSQL_PASSWORD:-}"

if mysql -h "$MYSQL_HOST" -P "$MYSQL_PORT" -u "$MYSQL_USER" -p"$MYSQL_PASSWORD" -e "SELECT 1" >/dev/null 2>&1; then
  ok "MySQL $MYSQL_HOST:$MYSQL_PORT reachable"
else
  warn "MySQL $MYSQL_HOST:$MYSQL_PORT not reachable. Run: sudo systemctl restart mysql"
fi

if redis-cli -h "${REDIS_HOST:-127.0.0.1}" -p "${REDIS_PORT:-6379}" ping 2>/dev/null | grep -q PONG; then
  ok "Redis ${REDIS_HOST:-127.0.0.1}:${REDIS_PORT:-6379} reachable"
else
  warn "Redis not reachable. Run: sudo systemctl restart redis-server"
fi

# ─── 4. Module skeletons (created in Phase 1, verified here) ──────────────────
for d in backend frontend; do
  if [[ -d "$REPO_ROOT/$d" ]]; then
    ok "$d/ directory present"
  else
    info "$d/ directory will be created in Phase 1"
  fi
done

info "Phase 0 install complete. See Makefile for next steps: 'make doctor', 'make dev'."
