#!/bin/sh
# Restore a pilot backup into an instance on the VPS host (phase 10,
# BRIEF §1.5/§3D).
#
#   scripts/pilot/host/restore-instance.sh <slug> <bundle.tar>
#   scripts/pilot/host/restore-instance.sh <slug> -            # tar on stdin
#
# Takes PLAINTEXT tar — the host never holds the age secret, so decryption
# happens on the laptop, which pipes the plaintext over tailnet SSH
# (WireGuard-encrypted in transit):
#
#   age -d -i docs/private/pilot/keys/backup.key <archive.tar.age> | \
#     ssh ubuntu@<host> '~/roadbook/scripts/pilot/host/restore-instance.sh <slug> -'
#
# The bundle is what backup-instance.sh stages: roadbook.tar.gz (decisions,
# photos, thumbnails — merge-by-durable-identity, skip-overlap-and-report)
# and uploads.tar.gz (unpacked into the uploads volume; existing files
# kept — content-hash names make same-name same-content). Restored
# decisions with no matching candidate sit as orphans until data is
# imported and detection runs — that is the designed re-attachment path,
# not an error (phase 5). The target stack must be up.
set -eu

[ $# -eq 2 ] || { echo "usage: $0 <slug> <bundle.tar | ->" >&2; exit 2; }
SLUG=$1
ARCHIVE=$2

REPO=$(cd "$(dirname "$0")/../../.." && pwd)
PILOT_DIR=${ROADBOOK_PILOT_DIR:-$HOME/pilot}
INST=$PILOT_DIR/instances/$SLUG
[ -d "$INST" ] || { echo "no such instance: $INST" >&2; exit 1; }
PROJECT=roadbook-$SLUG

STAGE=$(mktemp -d)
trap 'rm -rf "$STAGE"' EXIT
# Literal-uid bind mount (see backup-instance.sh): the container must be
# able to read the staged bundle from its own nonroot user.
chmod 755 "$STAGE"

if [ "$ARCHIVE" = - ]; then
  tar xf - -C "$STAGE"
else
  [ -f "$ARCHIVE" ] || { echo "no such archive: $ARCHIVE" >&2; exit 1; }
  tar xf "$ARCHIVE" -C "$STAGE"
fi
[ -f "$STAGE/roadbook.tar.gz" ] || { echo "bundle missing roadbook.tar.gz (is this a decrypted backup bundle?)" >&2; exit 1; }

docker compose -p "$PROJECT" --env-file "$INST/.env" \
  -f "$REPO/compose.yaml" -f "$REPO/scripts/pilot/compose.instance.yaml" \
  run --rm --no-deps -v "$STAGE:/backup" api \
  sh -c "roadbook restore -src /backup/roadbook.tar.gz && tar xzf /backup/uploads.tar.gz -C /uploads"
echo "$PROJECT: restore complete (orphaned decisions re-attach after import + detection)"
