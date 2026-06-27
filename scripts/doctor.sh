#!/usr/bin/env bash
# Doctor: verify environment health. Exit non-zero on any failed check.
# Used by `make doctor` and in CI sanity scripts.

set -uo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

C_OK="\033[32m"; C_ERR="\033[31m"; C_RST="\033[0m"
[[ -t 1 ]] || { C_OK=""; C_ERR=""; C_RST=""; }

FAIL=0
ok()  { printf "${C_OK}✓${C_RST} %s\n" "$*"; }
bad() { printf "${C_ERR}✗${C_RST} %s\n" "$*"; FAIL=$((FAIL+1)); }

# 1. Required binaries
for bin in go node npm mysql redis-cli git make; do
  command -v "$bin" >/dev/null 2>&1 && ok "$bin: $(command -v "$bin")" || bad "$bin: NOT FOUND"
done

# 2. Versions
go version  2>/dev/null | sed 's/^/  /'
node -v     2>/dev/null | sed 's/^/  /'

# 3. Git config
git config --global user.name  >/dev/null 2>&1 && ok "git user.name: $(git config --global user.name)"  || bad "git user.name not set"
git config --global user.email >/dev/null 2>&1 && ok "git user.email: $(git config --global user.email)" || bad "git user.email not set"

# 4. .env
[[ -f "$REPO_ROOT/.env" ]] && ok ".env present" || bad ".env missing (run scripts/install.sh)"

# 5. MySQL / Redis reachability
if [[ -f "$REPO_ROOT/.env" ]]; then
  set -a; source "$REPO_ROOT/.env"; set +a
  if mysql -h "${MYSQL_HOST:-127.0.0.1}" -P "${MYSQL_PORT:-3307}" -u "${MYSQL_USER:-root}" -p"${MYSQL_PASSWORD:-}" -e "SELECT 1" >/dev/null 2>&1; then
    ok "MySQL ${MYSQL_HOST}:${MYSQL_PORT} reachable"
  else
    bad "MySQL ${MYSQL_HOST}:${MYSQL_PORT} unreachable"
  fi
  if redis-cli -h "${REDIS_HOST:-127.0.0.1}" -p "${REDIS_PORT:-6379}" ping 2>/dev/null | grep -q PONG; then
    ok "Redis ${REDIS_HOST}:${REDIS_PORT} reachable"
  else
    bad "Redis ${REDIS_HOST}:${REDIS_PORT} unreachable"
  fi
fi

# 6. Port summary
for port in 8081 5180 3307 6379; do
  if ss -tln 2>/dev/null | grep -qE ":${port} "; then
    printf "  port %s: in use\n" "$port"
  fi
done

if [[ $FAIL -gt 0 ]]; then
  echo
  echo "${C_ERR}${FAIL} check(s) failed.${C_RST}"
  exit 1
fi
echo
echo "${C_OK}All checks passed.${C_RST}"
