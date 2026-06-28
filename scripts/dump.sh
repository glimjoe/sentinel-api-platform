#!/usr/bin/env bash
# MySQL backup for Sentinel.
# Output: sentinel_YYYYMMDD_HHMMSS.sql.gz in repo root.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

# shellcheck disable=SC1091
[[ -f "$REPO_ROOT/.env" ]] && set -a && source "$REPO_ROOT/.env" && set +a

MYSQL_HOST="${MYSQL_HOST:-127.0.0.1}"
MYSQL_PORT="${MYSQL_PORT:-3307}"
MYSQL_USER="${MYSQL_USER:-root}"
MYSQL_PASSWORD="${MYSQL_PASSWORD:-}"
MYSQL_DATABASE="${MYSQL_DATABASE:-sentinel}"

TIMESTAMP=$(date +%Y%m%d_%H%M%S)
OUTFILE="$REPO_ROOT/sentinel_${TIMESTAMP}.sql.gz"

C_OK="\033[32m"; C_ERR="\033[31m"; C_RST="\033[0m"
[[ -t 1 ]] || { C_OK=""; C_ERR=""; C_RST=""; }

printf "Dumping %s@%s:%s/%s → %s\n" \
  "$MYSQL_USER" "$MYSQL_HOST" "$MYSQL_PORT" "$MYSQL_DATABASE" "$OUTFILE"

mysqldump \
  -h "$MYSQL_HOST" -P "$MYSQL_PORT" \
  -u "$MYSQL_USER" -p"$MYSQL_PASSWORD" \
  --single-transaction --routines --triggers \
  --default-character-set=utf8mb4 \
  "$MYSQL_DATABASE" \
  | gzip > "$OUTFILE"

printf "${C_OK}[dump]${C_RST} %s (%s bytes)\n" \
  "$(basename "$OUTFILE")" "$(wc -c < "$OUTFILE" | tr -d ' ')"
