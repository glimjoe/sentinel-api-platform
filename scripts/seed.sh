#!/usr/bin/env bash
# Load demo data: admin user, sample project, sample APIs/cases/rules.
# Phase 0 stub: prints what will be seeded. Real implementation in Phase 1.

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

C_OK="\033[32m"; C_RST="\033[0m"
[[ -t 1 ]] || { C_OK=""; C_RST=""; }

printf "${C_OK}[seed]${C_RST} Sentinel demo data (Phase 0 stub — real seeding lands in Phase 1)\n"
printf "  • admin user:  admin@sentinel.local / admin123\n"
printf "  • demo user:   demo@sentinel.local  / demo123\n"
printf "  • project:     Petstore (slug: petstore)\n"
printf "  • APIs:        GET/POST /pet, GET /pet/findByStatus, …\n"
printf "  • mock rules:  2 examples\n"
printf "  • test cases:  2 examples (1 pass, 1 fail-on-purpose for AI demo)\n"
