#!/bin/sh
# Deploy the public landing page to the host front (phase 12 CP1).
#
#   scripts/pilot/host/deploy-landing.sh
#
# Stages /srv/pilot/landing/site/ from the repo checkout — landing/index.html
# plus the two asset directories the page references but git does not
# duplicate (fonts from web/app/fonts, screenshots from docs/screens) — and
# writes the Caddy snippet routing www.$ROADBOOK_DOMAIN to it, credential-
# free: this is the one public, unauthenticated surface, and it is static
# files only. The wildcard certificate covers www.<domain>; the bare apex
# would need its own certificate entry, which is why the landing lives on a
# subdomain (phase 12 BRIEF §1.4).
#
# The Caddyfile must import the snippet inside the wildcard site block,
# BEFORE the 404 fallback, alongside the instances import:
#
#   import /srv/pilot/landing/caddy.conf
#
# This script checks for that line and prints it if missing — it never
# edits /etc/caddy/Caddyfile itself (one deliberate operator edit, once).
#
# Phase-10 lessons applied here: everything staged is made world-readable
# (systemd's caddy cannot read operator-umask files, and an unreadable
# import 404s the whole front silently), and validation runs AS THE CADDY
# USER — a root validate lies about readability.
set -eu

REPO=$(cd "$(dirname "$0")/../../.." && pwd)
PILOT_DIR=${ROADBOOK_PILOT_DIR:-/srv/pilot}
[ -f "$PILOT_DIR/front.env" ] || { echo "missing $PILOT_DIR/front.env (must define ROADBOOK_DOMAIN)" >&2; exit 1; }
. "$PILOT_DIR/front.env"
[ -n "${ROADBOOK_DOMAIN:-}" ] || { echo "ROADBOOK_DOMAIN not set in $PILOT_DIR/front.env" >&2; exit 1; }

LANDING=$PILOT_DIR/landing
SITE=$LANDING/site

# Stage into a fresh directory, then swap: the front never serves a
# half-copied tree.
STAGE=$(mktemp -d "$LANDING.stage.XXXXXX")
trap 'rm -rf "$STAGE"' EXIT
mkdir -p "$STAGE/fonts" "$STAGE/shots"
cp "$REPO/landing/index.html" "$REPO/landing/joined.html" "$REPO/landing/oops.html" "$STAGE/"
cp -R "$REPO/web/app/fonts/source-serif-4" "$STAGE/fonts/source-serif-4"
cp -R "$REPO/web/app/fonts/ibm-plex-mono" "$STAGE/fonts/ibm-plex-mono"
# Screenshot mapping — the page's <img> names on the left, the committed
# demo captures on the right. Update both together.
cp "$REPO/docs/screens/phase9-cp4-home.png"        "$STAGE/shots/life-map.png"
cp "$REPO/docs/screens/phase9-cp4-adventure-2.png" "$STAGE/shots/adventure.png"
cp "$REPO/docs/screens/phase9-cp4-adventures.png"  "$STAGE/shots/atlas.png"

# World-readable throughout: caddy runs as its own user and bind/disk
# permissions are literal (phase 10 CP2).
find "$STAGE" -type d -exec chmod 755 {} +
find "$STAGE" -type f -exec chmod 644 {} +

mkdir -p "$LANDING"
if [ -d "$SITE" ]; then
  OLD=$(mktemp -d "$LANDING.old.XXXXXX")
  mv "$SITE" "$OLD/site"
fi
mv "$STAGE" "$SITE"
trap - EXIT
if [ -n "${OLD:-}" ]; then rm -rf "$OLD"; fi
chmod 755 "$LANDING" "$SITE"

# The Caddy snippet. Host-matched handle block, same shape as the instance
# snippets; no auth, no robots header — this page is the public face and
# is meant to be found. The one dynamic route is the waitlist form's POST,
# proxied to the loopback intake service (setup-waitlist.sh installs it).
cat > "$LANDING/caddy.conf" <<EOF
@landing host www.$ROADBOOK_DOMAIN
handle @landing {
	reverse_proxy /waitlist 127.0.0.1:9310
	root * $SITE
	file_server
	header Cache-Control "public, max-age=3600"
}
EOF
chmod 644 "$LANDING/caddy.conf"

if ! grep -q "import $LANDING/caddy.conf" /etc/caddy/Caddyfile; then
  echo "" >&2
  echo "ACTION NEEDED: /etc/caddy/Caddyfile does not import the landing snippet." >&2
  echo "Add this line inside the wildcard site block, next to the instances" >&2
  echo "import and before the 404 fallback:" >&2
  echo "" >&2
  echo "    import $LANDING/caddy.conf" >&2
  echo "" >&2
  echo "then re-run this script." >&2
  exit 1
fi

# Validate as the caddy user (root validate lies about readability), then
# reload.
sudo -u caddy caddy validate --config /etc/caddy/Caddyfile
sudo systemctl reload caddy
echo "landing deployed: https://www.$ROADBOOK_DOMAIN"
