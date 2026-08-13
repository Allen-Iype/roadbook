#!/bin/sh
# Back up one pilot instance (phase 8 BRIEF §3D): `roadbook backup` for the
# durable identities (decisions, photos, thumbnails) PLUS a tar of the
# uploads volume — retained exports are hard-to-replace data the backup
# format does not carry (phase 7 LOG, carried tension). The pair is bundled,
# age-encrypted to the operator key, and written to the offsite directory;
# only ciphertext ever leaves this machine.
#
#   scripts/pilot/backup-instance.sh <slug|demo>
#
#   demo     the base compose project (`roadbook`)
#   <slug>   a stamped instance (docs/private/pilot/instances/<slug>/)
#
# Destination: $ROADBOOK_BACKUP_DIR, default
# "$HOME/Library/Mobile Documents/com~apple~CloudDocs/RoadbookBackups"
# (iCloud Drive). Key: docs/private/pilot/keys/backup.key — its PUBLIC half
# encrypts here; the SECRET half is needed only at restore, and a copy of
# the key file must live OFF this machine (password manager), or the
# backups die with the laptop they were meant to survive.
#
# Skips (with a message, exit 0) when the instance's stack is not running:
# `compose run` would boot a downed database only to back up its emptiness.
set -eu
export PATH=/opt/homebrew/bin:/usr/local/bin:$PATH  # cron-safe

[ $# -eq 1 ] || { echo "usage: $0 <slug|demo>" >&2; exit 2; }
NAME=$1

REPO=$(cd "$(dirname "$0")/../.." && pwd)
PILOT_DIR=${ROADBOOK_PILOT_DIR:-$REPO/docs/private/pilot}
DEST=${ROADBOOK_BACKUP_DIR:-"$HOME/Library/Mobile Documents/com~apple~CloudDocs/RoadbookBackups"}
KEY=$PILOT_DIR/keys/backup.key

RECIPIENT=$(sed -n 's/^# public key: //p' "$KEY")
[ -n "$RECIPIENT" ] || { echo "no public key found in $KEY" >&2; exit 1; }

if [ "$NAME" = demo ]; then
  PROJECT=roadbook
  set -- -f "$REPO/compose.yaml"
else
  INST=$PILOT_DIR/instances/$NAME
  [ -d "$INST" ] || { echo "no such instance: $INST" >&2; exit 1; }
  PROJECT=roadbook-$NAME
  set -- --env-file "$INST/.env" -f "$REPO/compose.yaml" -f "$REPO/scripts/pilot/compose.instance.yaml"
fi

if [ -z "$(docker ps -q --filter "name=${PROJECT}-db-1" 2>/dev/null)" ]; then
  echo "$PROJECT: stack not running — skipped (nothing new to back up while down)"
  exit 0
fi

STAGE=$(mktemp -d)
trap 'rm -rf "$STAGE"' EXIT

docker compose -p "$PROJECT" "$@" run --rm --no-deps -v "$STAGE:/backup" api \
  sh -c "roadbook backup -out /backup/roadbook.tar.gz && tar czf /backup/uploads.tar.gz -C /uploads ."

mkdir -p "$DEST"
OUT=$DEST/$NAME-$(date +%Y%m%d-%H%M%S).tar.age
tar cf - -C "$STAGE" roadbook.tar.gz uploads.tar.gz | age -r "$RECIPIENT" -o "$OUT"
echo "$PROJECT: backed up -> $OUT ($(du -h "$OUT" | cut -f1 | tr -d ' '))"
