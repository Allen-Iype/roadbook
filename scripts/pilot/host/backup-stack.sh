#!/bin/sh
# Stage a whole-stack backup bundle for one stamped instance (phase 15
# CP3, BRIEF §10). The unit is the STACK, not the person: one `pg_dump` of
# the one database plus a tar of each file volume — the only complete
# picture of a shared instance (users, sessions, share links, photo
# records, decisions, retained exports, thumbnails).
#
#   scripts/pilot/host/backup-stack.sh <slug> <stage-dir>
#
# Writes into <stage-dir> (must exist, must be empty):
#   db.sql.gz     plain-format dump, data of `countries` and `states`
#                 excluded (regenerable: the api refills them at startup
#                 with -if-empty; their schema and the goose version table
#                 are kept)
#   photos.tar    the photos volume (thumbnails)
#   uploads.tar   the uploads volume (retained exports)
#   fingerprint   two lines, `db <sha256>` and `files <sha256>`: the
#                 first over the UNcompressed dump, the second over a
#                 listing (path, size, mtime) of both volumes. Any row
#                 change moves the first, any file change the second;
#                 backup-nightly.sh pushes each PART only when its own
#                 line differs from the last pushed one — the database is
#                 small and changes on every sign-in, the files are large
#                 and change only when someone uploads, so the two travel
#                 separately and the bucket stays inside its free 10 GB.
#
# Prints the two fingerprint lines on stdout. Exit 3 (with a message) when the
# stack is not running — `compose run` would boot a downed database only
# to dump its emptiness, so a down stack is "nothing to back up", which
# the caller reports rather than silently skips.
#
# pg_dump runs INSIDE the db container (PostgreSQL's own client, same
# major version as the server — nothing is installed on the host); the
# volume tars run inside a throwaway api container as its nonroot user,
# which owns the files (Linux bind mounts are literal uids).
set -eu

[ $# -eq 2 ] || { echo "usage: $0 <slug> <stage-dir>" >&2; exit 2; }
SLUG=$1
STAGE=$2

REPO=$(cd "$(dirname "$0")/../../.." && pwd)
PILOT_DIR=${ROADBOOK_PILOT_DIR:-$HOME/pilot}
INST=$PILOT_DIR/instances/$SLUG
PROJECT=roadbook-$SLUG

[ -d "$INST" ] || { echo "no such instance: $INST" >&2; exit 1; }
[ -d "$STAGE" ] || { echo "stage dir missing: $STAGE" >&2; exit 1; }
[ -z "$(ls -A "$STAGE")" ] || { echo "stage dir not empty: $STAGE" >&2; exit 1; }

compose() {
  docker compose --progress quiet -p "$PROJECT" --env-file "$INST/.env" \
    -f "$REPO/compose.yaml" -f "$REPO/scripts/pilot/compose.instance.yaml" "$@"
}

if [ -z "$(docker ps -q --filter "name=^${PROJECT}-db-1$" --filter status=running)" ]; then
  echo "$PROJECT: stack not running" >&2
  exit 3
fi

# The api container runs as its own nonroot user; a 700 mktemp dir is
# unwritable from inside (literal uids). The stage dir is the caller's —
# it opens it just enough for the container user to write.
chmod 777 "$STAGE"

# 1. Database. Plain format so the dump's bytes are a function of the data
#    alone (custom format embeds a timestamp in its header); no owner or
#    privilege statements (one role, same name everywhere).
compose exec -T db pg_dump -U roadbook -d roadbook \
  --format=plain --no-owner --no-privileges \
  --exclude-table-data=countries --exclude-table-data=states \
  > "$STAGE/db.sql" </dev/null

# 2. Volumes, as the user that owns them.
compose run --rm --no-deps -T -v "$STAGE:/stage" api sh -c '
  tar cf /stage/photos.tar  -C /photos  . &&
  tar cf /stage/uploads.tar -C /uploads . &&
  { find /photos /uploads -type f -exec stat -c "%n %s %Y" {} + | sort; } > /stage/listing
' >/dev/null </dev/null   # compose exec/run forward stdin: never this script's

# 3. Fingerprint over the uncompressed dump + listing; then compress.
#    pg_dump (17.6+/18) opens and closes every plain dump with
#    `\restrict <random token>` / `\unrestrict <token>` guard lines — a
#    fresh random token per run, so two dumps of identical data differ in
#    exactly those lines. They are kept in the stored dump (psql honours
#    them on restore) and left out of the fingerprint.
DBFP=$(grep -v '^\\\(un\)\?restrict ' "$STAGE/db.sql" | sha256sum | cut -d' ' -f1)
FILESFP=$(sha256sum < "$STAGE/listing" | cut -d' ' -f1)
printf 'db %s\nfiles %s\n' "$DBFP" "$FILESFP" > "$STAGE/fingerprint"
gzip -n "$STAGE/db.sql"
rm -f "$STAGE/listing"

cat "$STAGE/fingerprint"
