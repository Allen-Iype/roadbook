#!/bin/sh
# Rotate a tester's basic-auth credential in place on the VPS host (phase 8
# BRIEF §1.4, unchanged rule): at every handover, and immediately on any
# suspected leak. The link stays; the old password stops working the moment
# Caddy reloads.
#
#   scripts/pilot/host/rotate-credential.sh <slug>
set -eu

[ $# -eq 1 ] || { echo "usage: $0 <slug>" >&2; exit 2; }
SLUG=$1

PILOT_DIR=${ROADBOOK_PILOT_DIR:-$HOME/pilot}
INST=$PILOT_DIR/instances/$SLUG

[ -f "$INST/caddy.conf" ] || { echo "no such instance: $INST" >&2; exit 1; }

PASSWORD=$(openssl rand -base64 15 | tr '+/' '-_')
HASH=$(caddy hash-password --plaintext "$PASSWORD")

# The basic_auth block holds one line per accepted username variant
# (new-instance.sh writes four — keyboard-forgiving spellings of the same
# user). Replace the hash on every one; usernames stay as written.
awk -v hash="$HASH" '
  $NF ~ /^\$2[aby]?\$/ { $NF = hash; print "\t\t" $0; next }
  { print }
' "$INST/caddy.conf" > "$INST/caddy.conf.new"
mv "$INST/caddy.conf.new" "$INST/caddy.conf"

sudo systemctl reload caddy

cat <<EOF
== credential rotated for $SLUG ==
username:  $SLUG
password:  $PASSWORD

Old password is dead. Update the ledger, then send the new pair over the
same channel as the link.
EOF
