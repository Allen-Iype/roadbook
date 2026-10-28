#!/bin/sh
# Restore a whole-stack bundle into a stamped instance on the host (phase
# 15 CP3, BRIEF §10). The inverse of backup-stack.sh: the database is
# dropped and recreated from the dump; both file volumes are unpacked.
#
#   scripts/pilot/host/restore-stack.sh <slug> -            # plaintext tar on stdin
#   scripts/pilot/host/restore-stack.sh <slug> <bundle.tar> [--force]
#
# The host never holds the age secret, so decryption happens on the
# laptop, which pipes the plaintext over ssh (encrypted in transit):
#
#   { age -d -i docs/private/pilot/keys/backup.key <slug>-db-<T>.tar.age;
#     age -d -i docs/private/pilot/keys/backup.key <slug>-files-<T>.tar.age; } | \
#     ssh ubuntu@<host> '~/roadbook/scripts/pilot/host/restore-stack.sh <slug> -'
#
# The nightly job pushes TWO archives per stack — `<slug>-db-<time>` (the
# dump) and `<slug>-files-<time>` (both volumes) — each only when its own
# part changed, so the newest of each may carry different times. Decrypt
# the newest of both and concatenate them on stdin (tar reads
# concatenated archives one after the other); a db archive alone is
# accepted too and restores the database only.
#
# The target stack must exist (stamped) and be up. It is meant to be EMPTY
# — a fresh scratch stack for a drill, or a fresh stamp after a loss. A
# stack that already holds accounts or candidates is refused unless
# --force is given, because the restore REPLACES its database wholesale.
#
# Sequence: stop api+web (nothing may hold the database) → drop + create
# the database → psql the dump (ON_ERROR_STOP: a partial restore is an
# error, not a silent half) → unpack both volumes as the api's own user
# (existing files are overwritten; content-hash names make same-name
# same-content) → bring the stack up. The api's startup refills the
# excluded `countries`/`states` tables with -if-empty.
set -eu

[ $# -ge 2 ] || { echo "usage: $0 <slug> <bundle.tar | -> [--force]" >&2; exit 2; }
SLUG=$1
ARCHIVE=$2
FORCE=${3:-}

REPO=$(cd "$(dirname "$0")/../../.." && pwd)
PILOT_DIR=${ROADBOOK_PILOT_DIR:-$HOME/pilot}
INST=$PILOT_DIR/instances/$SLUG
PROJECT=roadbook-$SLUG
[ -d "$INST" ] || { echo "no such instance: $INST" >&2; exit 1; }

compose() {
  docker compose -p "$PROJECT" --env-file "$INST/.env" \
    -f "$REPO/compose.yaml" -f "$REPO/scripts/pilot/compose.instance.yaml" "$@"
}
# `compose exec -T` still forwards stdin, so every call that is not the
# dump load reads from /dev/null — otherwise the bundle on stdin is eaten
# by the first query (found on the first drill).
psql_db() { compose exec -T db psql -v ON_ERROR_STOP=1 -q -U roadbook "$@" </dev/null; }
psql_load() { compose exec -T db psql -v ON_ERROR_STOP=1 -q -U roadbook "$@"; }

STAGE=$(mktemp -d)
trap 'rm -rf "$STAGE"' EXIT
chmod 755 "$STAGE"   # literal-uid bind mount: the container user must read it

# The bundle is read before anything else touches stdin.
if [ "$ARCHIVE" = - ]; then
  tar xf - -i -C "$STAGE"     # -i: read on past the end of one archive into the next
else
  [ -f "$ARCHIVE" ] || { echo "no such archive: $ARCHIVE" >&2; exit 1; }
  tar xf "$ARCHIVE" -i -C "$STAGE"
fi
[ -f "$STAGE/db.sql.gz" ] || { echo "bundle has no db.sql.gz (is this a decrypted db archive?)" >&2; exit 1; }
HAVE_FILES=1
[ -f "$STAGE/photos.tar" ] && [ -f "$STAGE/uploads.tar" ] || { HAVE_FILES=0; echo "note: no files archive on stdin — restoring the database only" >&2; }
chmod 644 "$STAGE"/*

[ -n "$(docker ps -q --filter "name=^${PROJECT}-db-1$" --filter status=running)" ] \
  || { echo "$PROJECT: stack not running (bring it up first)" >&2; exit 1; }

# Refuse to replace a populated stack without --force.
POP=$(psql_db -d roadbook -tAc "select (select count(*) from users where id <> 'self') + (select count(*) from candidates)" 2>/dev/null || echo 0)
if [ "${POP:-0}" -gt 0 ] && [ "$FORCE" != "--force" ]; then
  echo "$PROJECT holds data (accounts/candidates: $POP); refusing to replace it. Add --force to restore anyway." >&2
  exit 1
fi

echo "== $PROJECT: stopping api and web"
compose stop api web >/dev/null </dev/null

echo "== recreating the database"
psql_db -d postgres -c "DROP DATABASE IF EXISTS roadbook"
psql_db -d postgres -c "CREATE DATABASE roadbook OWNER roadbook"
gunzip -c "$STAGE/db.sql.gz" | psql_load -d roadbook >/dev/null

if [ "$HAVE_FILES" -eq 1 ]; then   # `if`, never `[ ] &&` under set -e (deploy-landing's lesson)
  echo "== unpacking volumes"
  compose run --rm --no-deps -T -v "$STAGE:/stage:ro" api sh -c '
    tar xf /stage/photos.tar  -C /photos &&
    tar xf /stage/uploads.tar -C /uploads
  ' >/dev/null </dev/null
fi

echo "== bringing the stack up"
compose up -d >/dev/null </dev/null

echo "== restored. Counts:"
psql_db -d roadbook -tA -c "select 'users '||count(*) from users union all select 'imports '||count(*) from imports union all select 'candidates '||count(*) from candidates union all select 'decisions '||count(*) from decisions union all select 'raw_positions '||count(*) from raw_positions union all select 'photo_records '||count(*) from photo_records union all select 'share_links '||count(*) from share_links"
echo "(fingerprint of the bundle: $(cat "$STAGE/fingerprint" 2>/dev/null || echo unknown))"
