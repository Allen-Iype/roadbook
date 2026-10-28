#!/bin/sh
# The nightly durability job for the host (phase 15 CP3, BRIEF §10): for
# every stamped instance, stage a whole-stack bundle (backup-stack.sh),
# compare its two fingerprints (database / files) with the last ones
# pushed, and — only for a part that changed — encrypt that part to the
# operator's PUBLIC age key and push it to the off-host bucket. Then tell
# the dead-man's switch what happened: one ping, one body, one line per
# stack. Two parts because the database is small and changes on every
# sign-in while the files are large and change only on an upload: a
# quiet night costs megabytes, not the whole archive (the free tier is
# 10 GB; the LEDGER tracks it).
#
#   scripts/pilot/host/backup-nightly.sh          # the systemd timer runs this
#
# Environment (the unit loads /srv/pilot/backup.env, 0600, never committed):
#   ROADBOOK_BACKUP_REMOTE   rclone destination, e.g. r2:<bucket>
#   ROADBOOK_PING_URL        the check's ping URL; /start and /fail are
#                            derived from it. Unset = no pings (printed).
#   RCLONE_CONFIG_R2_*       rclone's env-style remote definition for R2
#                            (type s3, provider Cloudflare, access key,
#                            secret, endpoint, no_check_bucket) — no
#                            rclone.conf file exists on the host.
#
# State under $ROADBOOK_BACKUP_DIR (default /srv/pilot/backups):
#   <slug>.db.fingerprint, <slug>.files.fingerprint
#                        each written only after its push returned clean,
#                        so a failed push is retried the next night
#   stage/               scratch, emptied every run
#
# Nothing readable leaves this script: the bundle is encrypted before it
# is pushed, the ciphertext is deleted locally once the push succeeds,
# and the ping body carries slugs and sizes only. The age SECRET never
# touches the host — it cannot read what it produces (phase 10 §1.5,
# unchanged).
set -u

REPO=$(cd "$(dirname "$0")/../../.." && pwd)
PILOT_DIR=${ROADBOOK_PILOT_DIR:-$HOME/pilot}
BDIR=${ROADBOOK_BACKUP_DIR:-/srv/pilot/backups}
REMOTE=${ROADBOOK_BACKUP_REMOTE:-}
PING=${ROADBOOK_PING_URL:-}

ping() {  # ping <suffix> <body>
  [ -n "$PING" ] || { echo "(no ping url) $1: $2"; return 0; }
  curl -fsS -m 10 --retry 3 -o /dev/null --data-raw "$2" "$PING$1" \
    || echo "ping $1 failed" >&2
}

RECIPIENT=$(tr -d '[:space:]' < "$PILOT_DIR/keys/backup.pub" 2>/dev/null || true)
case "$RECIPIENT" in
  age1*) ;;
  *) echo "no age recipient in $PILOT_DIR/keys/backup.pub" >&2; ping /fail "no age recipient on host"; exit 1 ;;
esac
[ -n "$REMOTE" ] || { echo "ROADBOOK_BACKUP_REMOTE unset" >&2; ping /fail "no backup remote configured"; exit 1; }

ping /start ""
mkdir -p "$BDIR/stage"
BODY=""
FAIL=0

for d in "$PILOT_DIR"/instances/*/; do
  [ -d "$d" ] || continue
  SLUG=$(basename "$d")
  STAGE=$BDIR/stage/$SLUG
  ERR=$BDIR/stage/$SLUG.err      # outside the stage dir, which must start empty
  rm -rf "$STAGE" "$ERR"; mkdir -p "$STAGE"

  if ! FP=$("$REPO/scripts/pilot/host/backup-stack.sh" "$SLUG" "$STAGE" 2>"$ERR"); then
    rc=$?
    if [ "$rc" -eq 3 ]; then
      LINE="$SLUG: stack down, nothing to back up"
    else
      LINE="$SLUG: FAILED staging ($(head -c 300 "$ERR" | tr '\n' ' '))"
      FAIL=1
    fi
    echo "$LINE"; BODY="$BODY$LINE
"; rm -rf "$STAGE" "$ERR"; continue
  fi

  STAMP=$(date -u +%Y%m%dT%H%M%SZ)
  LINE="$SLUG:"
  for PART in db files; do
    NOW=$(printf '%s\n' "$FP" | sed -n "s/^$PART //p")
    LAST=$(cat "$BDIR/$SLUG.$PART.fingerprint" 2>/dev/null || true)
    if [ -n "$NOW" ] && [ "$NOW" = "$LAST" ]; then
      LINE="$LINE $PART unchanged;"
      continue
    fi
    case $PART in
      db)    MEMBERS="db.sql.gz fingerprint" ;;
      files) MEMBERS="photos.tar uploads.tar fingerprint" ;;
    esac
    NAME=$SLUG-$PART-$STAMP.tar.age
    # shellcheck disable=SC2086  # MEMBERS is a deliberate word list
    if tar cf - -C "$STAGE" $MEMBERS | age -r "$RECIPIENT" -o "$STAGE/$NAME" \
       && rclone copy --no-traverse --retries 3 "$STAGE/$NAME" "$REMOTE/$SLUG/" 2>"$ERR"; then
      SIZE=$(du -h "$STAGE/$NAME" | cut -f1 | tr -d ' ')
      printf '%s\n' "$NOW" > "$BDIR/$SLUG.$PART.fingerprint"
      LINE="$LINE pushed $NAME ($SIZE);"
    else
      LINE="$LINE FAILED $PART push ($(head -c 200 "$ERR" | tr '\n' ' '));"
      FAIL=1
    fi
    rm -f "$STAGE/$NAME"
  done
  case "$LINE" in *pushed*|*FAILED*) ;; *) LINE="$SLUG: unchanged, skipped" ;; esac
  echo "$LINE"; BODY="$BODY$LINE
"
  rm -rf "$STAGE" "$ERR"
done

if [ "$FAIL" -eq 0 ]; then
  ping "" "$BODY"
else
  ping /fail "$BODY"
fi
exit $FAIL
