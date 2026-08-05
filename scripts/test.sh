#!/bin/sh
# Runs the test suite with a database for the store's DB-backed tests
# (internal/store/storetest). The tests run by default — skipping is the
# exception, not the norm (docs/phase-4/DECISIONS.md). Resolution order:
#
#   1. ROADBOOK_TEST_DB already set        — use it as the admin URL.
#   2. A local Postgres answering on 5432  — postgres://localhost/postgres.
#   3. Docker available                    — compose.test.yaml's testdb.
#   4. None of the above                   — DB tests skip, visibly.
#
# The admin URL is only used to CREATE/DROP scratch databases named
# roadbook_test_*; nothing existing is touched.
set -e
cd "$(dirname "$0")/.."

if [ -z "$ROADBOOK_TEST_DB" ]; then
  if command -v pg_isready >/dev/null 2>&1 && pg_isready -q 2>/dev/null; then
    ROADBOOK_TEST_DB="postgres://localhost/postgres"
  elif command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
    docker compose -f compose.test.yaml up -d --wait testdb
    ROADBOOK_TEST_DB="postgres://roadbook:roadbook@localhost:55432/postgres"
  else
    echo "warning: no test database (no local Postgres, no Docker) — DB-backed store tests will SKIP" >&2
  fi
fi
export ROADBOOK_TEST_DB

exec go test ./cmd/... ./internal/... ./migrations/... "$@"
