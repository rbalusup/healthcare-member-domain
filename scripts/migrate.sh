#!/usr/bin/env bash
# migrate.sh — Runs database migrations using golang-migrate.
#
# Prerequisites:
#   go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
#
# Usage:
#   ./scripts/migrate.sh up           # Apply all pending migrations
#   ./scripts/migrate.sh down         # Roll back the last migration
#   ./scripts/migrate.sh down N       # Roll back N migrations
#   ./scripts/migrate.sh version      # Show current migration version
#   ./scripts/migrate.sh force V      # Force set migration version (use with care)
#
# Environment variables (can also come from .env):
#   MEMBER_DB_HOST, MEMBER_DB_PORT, MEMBER_DB_USER, MEMBER_DB_PASSWORD, MEMBER_DB_NAME

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MIGRATIONS_DIR="${REPO_ROOT}/migrations"

# Load .env if present
if [[ -f "${REPO_ROOT}/.env" ]]; then
  set -a
  # shellcheck source=/dev/null
  source "${REPO_ROOT}/.env"
  set +a
fi

DB_HOST="${MEMBER_DB_HOST:-localhost}"
DB_PORT="${MEMBER_DB_PORT:-5432}"
DB_USER="${MEMBER_DB_USER:-member_user}"
DB_PASS="${MEMBER_DB_PASSWORD:-member_pass}"
DB_NAME="${MEMBER_DB_NAME:-member_db}"
SSL="${MEMBER_DB_SSL_MODE:-disable}"

DB_URL="postgres://${DB_USER}:${DB_PASS}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${SSL}"

COMMAND="${1:-up}"

case "${COMMAND}" in
  up)
    echo "▶ Applying migrations..."
    migrate -database "${DB_URL}" -path "${MIGRATIONS_DIR}" up
    echo "✔ Migrations applied."
    ;;
  down)
    STEPS="${2:-1}"
    echo "▶ Rolling back ${STEPS} migration(s)..."
    migrate -database "${DB_URL}" -path "${MIGRATIONS_DIR}" down "${STEPS}"
    echo "✔ Rollback complete."
    ;;
  version)
    migrate -database "${DB_URL}" -path "${MIGRATIONS_DIR}" version
    ;;
  force)
    VERSION="${2:?'Usage: migrate.sh force <version>'}"
    echo "⚠ Forcing migration version to ${VERSION}..."
    migrate -database "${DB_URL}" -path "${MIGRATIONS_DIR}" force "${VERSION}"
    ;;
  *)
    echo "Unknown command: ${COMMAND}"
    echo "Usage: $0 [up|down [N]|version|force V]"
    exit 1
    ;;
esac
