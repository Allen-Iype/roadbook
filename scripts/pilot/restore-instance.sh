#!/bin/sh
# Restore an encrypted pilot backup into an instance (phase 8 BRIEF §3D).
#
#   scripts/pilot/restore-instance.sh <slug|demo> <archive.tar.age>
#
# Decrypts with the operator key, runs `roadbook restore` (decisions,
# photos, thumbnails — merge-by-durable-identity, skip-overlap-and-report),
# and unpacks the uploads tar into the uploads volume (existing files kept:
# content-hash names make same-name same-content). Restored decisions with
# no matching candidate sit as orphans until data is imported and detection
# runs — that is the designed re-attachment path, not an error (phase 5).
# The target stack must be up.
set -eu
export PATH=/opt/homebrew/bin:/usr/local/bin:$PATH

[ $# -eq 2 ] || { echo "usage: $0 <slug|demo> <archive.tar.age>" >&2; exit 2; }
NAME=$1
ARCHIVE=$2

REPO=$(cd "$(dirname "$0")/../.." && pwd)
PILOT_DIR=${ROADBOOK_PILOT_DIR:-$REPO/docs/private/pilot}
KEY=$PILOT_DIR/keys/backup.key
[ -f "$ARCHIVE" ] || { echo "no such archive: $ARCHIVE" >&2; exit 1; }

if [ "$NAME" = demo ]; then
  PROJECT=roadbook
  set -- -f "$REPO/compose.yaml"
else
  INST=$PILOT_DIR/instances/$NAME
  [ -d "$INST" ] || { echo "no such instance: $INST" >&2; exit 1; }
  PROJECT=roadbook-$NAME
  set -- --env-file "$INST/.env" -f "$REPO/compose.yaml" -f "$REPO/scripts/pilot/compose.instance.yaml"
fi

STAGE=$(mktemp -d)
trap 'rm -rf "$STAGE"' EXIT

age -d -i "$KEY" "$ARCHIVE" | tar xf - -C "$STAGE"

docker compose -p "$PROJECT" "$@" run --rm --no-deps -v "$STAGE:/backup" api \
  sh -c "roadbook restore -src /backup/roadbook.tar.gz && tar xzf /backup/uploads.tar.gz -C /uploads"
echo "$PROJECT: restore complete (orphaned decisions re-attach after import + detection)"
