#!/bin/sh
# Reset or retire a per-tester instance on the VPS host (phase 10 CP2 —
# phase 8's handover rules as procedure, re-addressed for the wildcard
# front).
#
#   scripts/pilot/host/reset-instance.sh <slug>            # wipe data, keep link
#   scripts/pilot/host/reset-instance.sh <slug> --retire   # wipe + kill subdomain
#
# Reset is VOLUME-level (`down -v`): database, photo thumbnails, and
# retained uploads all go structurally — never row deletes, which can
# silently miss a file. Before running it for a handover: tell the outgoing
# tester, and remind them to keep their own export file (the instance's
# retained copy may be their only one).
#
# Plain reset keeps the instance directory, credential, and subdomain
# (same tester, fresh data). --retire also removes the Caddy snippet and
# deletes the instance directory: the subdomain answers 404 (the wildcard
# fallback) and the old link+credential pair is dead. A NEW tester must
# always get a fresh stamp (new-instance.sh) — never an old credential.
set -eu

[ $# -ge 1 ] || { echo "usage: $0 <slug> [--retire]" >&2; exit 2; }
SLUG=$1
RETIRE=${2:-}

REPO=$(cd "$(dirname "$0")/../../.." && pwd)
PILOT_DIR=${ROADBOOK_PILOT_DIR:-$HOME/pilot}
INST=$PILOT_DIR/instances/$SLUG
PROJECT=roadbook-$SLUG

[ -d "$INST" ] || { echo "no such instance: $INST" >&2; exit 1; }

echo "== $PROJECT: down -v (volumes wiped)"
docker compose -p "$PROJECT" --env-file "$INST/.env" \
  -f "$REPO/compose.yaml" -f "$REPO/scripts/pilot/compose.instance.yaml" \
  down -v

if [ "$RETIRE" = "--retire" ]; then
  rm -rf "$INST"
  sudo systemctl reload caddy
  echo "== retired: subdomain dead (wildcard 404), credential dead"
  echo "Update the ledger (holder gone, date)."
else
  echo "== reset: volumes gone; stack left DOWN. Bring it back with:"
  echo "   docker compose -p $PROJECT --env-file $INST/.env \\"
  echo "     -f compose.yaml -f scripts/pilot/compose.instance.yaml up -d"
  echo "Same tester continues with the same link and credential."
fi
