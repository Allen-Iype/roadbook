#!/bin/sh
# Stamp a per-tester Roadbook instance (phase 8 BRIEF §3C).
#
#   scripts/pilot/new-instance.sh <slug> <slot>
#
#   slug   short lowercase name for the tester (letters/digits/dashes);
#          becomes compose project roadbook-<slug>. Never a real full name —
#          it appears in `docker ps` and the private ledger only.
#   slot   1, 2 or 3 — which of the three funnel listeners this instance
#          occupies (BRIEF §1.5):
#            slot 1: web 3010, Caddy 8100, funnel 443   (link has no port)
#            slot 2: web 3011, Caddy 8101, funnel 8443
#            slot 3: web 3012, Caddy 8102, funnel 10000
#
# What it does: writes the instance .env (random Postgres password) and a
# Caddy listener block with a fresh basic-auth credential under
# $ROADBOOK_PILOT_DIR/instances/<slug>/, brings the stack up, reloads
# Caddy, enables the funnel listener, and prints the link + credential for
# the handover message. Idempotence: refuses to reuse an existing instance
# directory or an occupied slot — reset or retire first (reset-instance.sh).
#
# Everything it writes lands under ROADBOOK_PILOT_DIR (default
# docs/private/pilot — gitignored). Nothing here may be committed.
set -eu

usage() { echo "usage: $0 <slug> <slot 1|2|3>" >&2; exit 2; }
[ $# -eq 2 ] || usage
SLUG=$1
SLOT=$2
case "$SLUG" in (*[!a-z0-9-]*|'') echo "slug must be lowercase letters/digits/dashes" >&2; exit 2;; esac
case "$SLOT" in
  1) WEB_PORT=3010; LISTENER=8100; FUNNEL=443 ;;
  2) WEB_PORT=3011; LISTENER=8101; FUNNEL=8443 ;;
  3) WEB_PORT=3012; LISTENER=8102; FUNNEL=10000 ;;
  *) usage ;;
esac

REPO=$(cd "$(dirname "$0")/../.." && pwd)
PILOT_DIR=${ROADBOOK_PILOT_DIR:-$REPO/docs/private/pilot}
INST=$PILOT_DIR/instances/$SLUG
PROJECT=roadbook-$SLUG

[ -d "$INST" ] && { echo "instance dir already exists: $INST (reset or retire first)" >&2; exit 1; }
if grep -qs ":$LISTENER {" "$PILOT_DIR"/instances/*/caddy.conf 2>/dev/null; then
  echo "slot $SLOT (listener $LISTENER) is already occupied — retire its holder first" >&2
  exit 1
fi

# Secrets. The password is printed once and recorded in the ledger by the
# operator; the Caddy file holds only the bcrypt hash.
PASSWORD=$(openssl rand -base64 15 | tr '+/' '-_')
HASH=$(caddy hash-password --plaintext "$PASSWORD")
PGPASS=$(openssl rand -hex 16)

mkdir -p "$INST"
cat > "$INST/.env" <<EOF
# Instance $SLUG (slot $SLOT) — generated $(date +%F). Private, never commit.
ROADBOOK_WEB_PORT=$WEB_PORT
POSTGRES_PASSWORD=$PGPASS
EOF

# The username is an address, not a secret — the password guards the
# instance. Phone keyboards auto-capitalize and append spaces (measured on
# Android AND iOS in this pilot), so accept those variants up front rather
# than teaching every tester to fight their keyboard.
CAP=$(printf '%s' "$SLUG" | awk '{print toupper(substr($0,1,1)) substr($0,2)}')
cat > "$INST/caddy.conf" <<EOF
# Instance $SLUG (slot $SLOT) — generated $(date +%F). Private, never commit.
# Port-only address + loopback bind: see scripts/pilot/Caddyfile.template.
http://:$LISTENER {
	bind 127.0.0.1
	basic_auth {
		$SLUG $HASH
		$CAP $HASH
		"$SLUG " $HASH
		"$CAP " $HASH
	}
	header X-Robots-Tag "noindex, nofollow"
	log {
		output file /opt/homebrew/var/log/roadbook-pilot-$LISTENER.log
	}
	reverse_proxy 127.0.0.1:$WEB_PORT
}
EOF

echo "== bringing up $PROJECT (web on 127.0.0.1:$WEB_PORT)"
docker compose -p "$PROJECT" --env-file "$INST/.env" \
  -f "$REPO/compose.yaml" -f "$REPO/scripts/pilot/compose.instance.yaml" \
  up -d --build

echo "== reloading Caddy"
caddy reload --config /opt/homebrew/etc/Caddyfile --adapter caddyfile

echo "== enabling funnel listener $FUNNEL -> $LISTENER"
tailscale funnel --bg --https="$FUNNEL" "$LISTENER"

HOST=$(tailscale status --json | python3 -c 'import json,sys; print(json.load(sys.stdin)["Self"]["DNSName"].rstrip("."))')
[ "$FUNNEL" = 443 ] && LINK="https://$HOST" || LINK="https://$HOST:$FUNNEL"

cat <<EOF

== instance $SLUG is up ==
link:      $LINK
username:  $SLUG
password:  $PASSWORD

Record these in the ledger (docs/private/pilot/LEDGER.md) now, then build
the handover message from RUNBOOK section 3. Two lines the message must
keep: copy the password rather than typing it, and no space after the
username (phone keyboards add one).
EOF
