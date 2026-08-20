#!/bin/sh
# Back up one pilot instance on the VPS host (phase 10, BRIEF §1.5/§3D):
# `roadbook backup` for the durable identities (decisions, photos,
# thumbnails) PLUS a tar of the uploads volume — retained exports are
# hard-to-replace data the backup format does not carry (phase 7 LOG,
# carried tension). The pair is bundled and age-encrypted to the operator
# key's PUBLIC half; the host can produce ciphertext it can never read.
#
#   scripts/pilot/host/backup-instance.sh <slug>
#
# Destination: $ROADBOOK_BACKUP_DIR, default $HOME/backups — a staging
# directory the laptop PULLS from (CP4 wires the pull + retention); only
# ciphertext ever sits there. Recipient: $ROADBOOK_PILOT_DIR/keys/backup.pub,
# a file holding the age recipient string ("age1..."). The SECRET half
# lives on the laptop and in the operator's password manager, never here —
# a compromised host must not be able to read backup history.
#
# Skips (with a message, exit 0) when the instance's stack is not running:
# `compose run` would boot a downed database only to back up its emptiness.
set -eu

[ $# -eq 1 ] || { echo "usage: $0 <slug>" >&2; exit 2; }
SLUG=$1

REPO=$(cd "$(dirname "$0")/../../.." && pwd)
PILOT_DIR=${ROADBOOK_PILOT_DIR:-$HOME/pilot}
DEST=${ROADBOOK_BACKUP_DIR:-$HOME/backups}

RECIPIENT=$(cat "$PILOT_DIR/keys/backup.pub" 2>/dev/null | tr -d '[:space:]')
case "$RECIPIENT" in
  age1*) ;;
  *) echo "no age recipient in $PILOT_DIR/keys/backup.pub" >&2; exit 1 ;;
esac

INST=$PILOT_DIR/instances/$SLUG
[ -d "$INST" ] || { echo "no such instance: $INST" >&2; exit 1; }
PROJECT=roadbook-$SLUG

if [ -z "$(docker ps -q --filter "name=${PROJECT}-db-1" 2>/dev/null)" ]; then
  echo "$PROJECT: stack not running — skipped (nothing new to back up while down)"
  exit 0
fi

STAGE=$(mktemp -d)
trap 'rm -rf "$STAGE"' EXIT
# The api container runs as its own nonroot user and Linux bind mounts are
# literal uids (no Docker Desktop mapping): a 700 mktemp dir is unwritable
# from inside. World-writable is fine for a throwaway staging dir.
chmod 777 "$STAGE"

docker compose -p "$PROJECT" --env-file "$INST/.env" \
  -f "$REPO/compose.yaml" -f "$REPO/scripts/pilot/compose.instance.yaml" \
  run --rm --no-deps -v "$STAGE:/backup" api \
  sh -c "roadbook backup -out /backup/roadbook.tar.gz && tar czf /backup/uploads.tar.gz -C /uploads ."

mkdir -p "$DEST"
OUT=$DEST/$SLUG-$(date +%Y%m%d-%H%M%S).tar.age
tar cf - -C "$STAGE" roadbook.tar.gz uploads.tar.gz | age -r "$RECIPIENT" -o "$OUT"
echo "$PROJECT: backed up -> $OUT ($(du -h "$OUT" | cut -f1 | tr -d ' '))"
