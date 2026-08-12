#!/bin/sh
# Reset or retire a per-tester instance (phase 8 BRIEF §3C — the handover
# rules as procedure).
#
#   scripts/pilot/reset-instance.sh <slug>            # wipe data, keep slot
#   scripts/pilot/reset-instance.sh <slug> --retire   # wipe + free the slot
#
# Reset is VOLUME-level (`down -v`): database, photo thumbnails, and
# retained uploads all go structurally — never row deletes, which can
# silently miss a file. Before running it for a handover: tell the outgoing
# tester, and remind them to keep their own export file (the instance's
# retained copy may be their only one).
#
# Plain reset keeps the instance directory, credential, and funnel listener
# (same tester, fresh data). --retire also removes the Caddy block, turns
# the funnel listener off, and deletes the instance directory: the slot is
# free and the old link+credential pair is dead. A NEW tester must always
# get a fresh stamp (new-instance.sh) — never an old credential.
set -eu

[ $# -ge 1 ] || { echo "usage: $0 <slug> [--retire]" >&2; exit 2; }
SLUG=$1
RETIRE=${2:-}

REPO=$(cd "$(dirname "$0")/../.." && pwd)
PILOT_DIR=${ROADBOOK_PILOT_DIR:-$REPO/docs/private/pilot}
INST=$PILOT_DIR/instances/$SLUG
PROJECT=roadbook-$SLUG

[ -d "$INST" ] || { echo "no such instance: $INST" >&2; exit 1; }

echo "== $PROJECT: down -v (volumes wiped)"
docker compose -p "$PROJECT" --env-file "$INST/.env" \
  -f "$REPO/compose.yaml" -f "$REPO/scripts/pilot/compose.instance.yaml" \
  down -v

if [ "$RETIRE" = "--retire" ]; then
  LISTENER=$(sed -n 's/^http:\/\/:\([0-9]*\) {.*/\1/p' "$INST/caddy.conf")
  case "$LISTENER" in
    8100) FUNNEL=443 ;;
    8101) FUNNEL=8443 ;;
    8102) FUNNEL=10000 ;;
    *) FUNNEL= ;;
  esac
  rm -rf "$INST"
  caddy reload --config /opt/homebrew/etc/Caddyfile --adapter caddyfile
  if [ -n "$FUNNEL" ]; then
    tailscale funnel --https="$FUNNEL" off || true
  fi
  echo "== retired: slot freed, credential dead, funnel $FUNNEL off"
  echo "Update the ledger (holder gone, date)."
else
  echo "== reset: volumes gone; stack left DOWN. Bring it back with:"
  echo "   docker compose -p $PROJECT --env-file $INST/.env \\"
  echo "     -f compose.yaml -f scripts/pilot/compose.instance.yaml up -d"
  echo "Same tester continues with the same link and credential."
fi
