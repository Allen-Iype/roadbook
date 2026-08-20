#!/bin/sh
# Stamp a per-tester Roadbook instance on the VPS host (phase 10 CP2;
# topology decisions in docs/phase-10/DECISIONS.md 2026-08-20).
#
#   scripts/pilot/host/new-instance.sh <slug>
#
#   slug   short lowercase name for the tester (letters/digits/dashes);
#          becomes compose project roadbook-<slug> AND the subdomain
#          <slug>.$ROADBOOK_DOMAIN. Never a real full name — it appears in
#          `docker ps`, DNS queries, and the private ledger only.
#
# The host front is one wildcard TLS site (Caddy, DNS-01 cert): routing is
# by hostname, so there is no slot argument and no port ceiling — the
# laptop front's three funnel ports are why the old script had slots. The
# web port is allocated sequentially from 3010 by scanning existing
# instance .env files; it is loopback-only and appears in exactly two
# places (the instance .env and its Caddy snippet).
#
# What it does: writes the instance .env (random Postgres password) and a
# host-matched Caddy handle block with a fresh basic-auth credential under
# $ROADBOOK_PILOT_DIR/instances/<slug>/, brings the stack up, reloads
# Caddy, and prints the link + credential for the handover message.
# Idempotence: refuses an existing instance directory — reset or retire
# first (reset-instance.sh).
#
# Requires: $ROADBOOK_PILOT_DIR/front.env defining ROADBOOK_DOMAIN (the
# base domain is a parameter — the repo is public, the real name lives in
# the private ledger). Nothing this script writes may be committed.
set -eu

usage() { echo "usage: $0 <slug>" >&2; exit 2; }
[ $# -eq 1 ] || usage
SLUG=$1
case "$SLUG" in (*[!a-z0-9-]*|'') echo "slug must be lowercase letters/digits/dashes" >&2; exit 2;; esac

REPO=$(cd "$(dirname "$0")/../../.." && pwd)
PILOT_DIR=${ROADBOOK_PILOT_DIR:-$HOME/pilot}
[ -f "$PILOT_DIR/front.env" ] || { echo "missing $PILOT_DIR/front.env (must define ROADBOOK_DOMAIN)" >&2; exit 1; }
. "$PILOT_DIR/front.env"
[ -n "${ROADBOOK_DOMAIN:-}" ] || { echo "ROADBOOK_DOMAIN not set in $PILOT_DIR/front.env" >&2; exit 1; }

INST=$PILOT_DIR/instances/$SLUG
PROJECT=roadbook-$SLUG
[ -d "$INST" ] && { echo "instance dir already exists: $INST (reset or retire first)" >&2; exit 1; }

# Next free loopback web port, starting at 3010.
PORT=3010
for f in "$PILOT_DIR"/instances/*/.env; do
  [ -f "$f" ] || continue
  p=$(sed -n 's/^ROADBOOK_WEB_PORT=//p' "$f")
  [ -n "$p" ] && [ "$p" -ge "$PORT" ] && PORT=$((p + 1))
done

# Secrets. The password is printed once and recorded in the ledger by the
# operator; the Caddy snippet holds only the bcrypt hash.
PASSWORD=$(openssl rand -base64 15 | tr '+/' '-_')
HASH=$(caddy hash-password --plaintext "$PASSWORD")
PGPASS=$(openssl rand -hex 16)

mkdir -p "$INST"
# The pilot tree lives at /srv/pilot (~/pilot is a symlink): the systemd
# Caddy runs as user `caddy`, which cannot traverse /home/ubuntu (750) —
# a glob import that matches unreadable files silently matches NOTHING and
# every subdomain 404s (found live at CP2). Dirs are 755 so caddy can read
# snippets; the .env (Postgres password) is 600, ubuntu-only.
cat > "$INST/.env" <<EOF
# Instance $SLUG — generated $(date +%F). Private, never commit.
ROADBOOK_WEB_PORT=$PORT
POSTGRES_PASSWORD=$PGPASS
ROADBOOK_INSTANCE_LABEL=$SLUG
EOF
chmod 600 "$INST/.env"

# The username is an address, not a secret — the password guards the
# instance. Phone keyboards auto-capitalize and append spaces (measured on
# Android AND iOS in this pilot), so accept those variants up front rather
# than teaching every tester to fight their keyboard.
CAP=$(printf '%s' "$SLUG" | awk '{print toupper(substr($0,1,1)) substr($0,2)}')
cat > "$INST/caddy.conf" <<EOF
# Instance $SLUG — generated $(date +%F). Private, never commit.
# Imported inside the wildcard site: see scripts/pilot/host/caddy-instance.template.
@$SLUG host $SLUG.$ROADBOOK_DOMAIN
handle @$SLUG {
	basic_auth {
		$SLUG $HASH
		$CAP $HASH
		"$SLUG " $HASH
		"$CAP " $HASH
	}
	header X-Robots-Tag "noindex, nofollow"
	reverse_proxy 127.0.0.1:$PORT
}
EOF

echo "== bringing up $PROJECT (web on 127.0.0.1:$PORT)"
docker compose -p "$PROJECT" --env-file "$INST/.env" \
  -f "$REPO/compose.yaml" -f "$REPO/scripts/pilot/compose.instance.yaml" \
  up -d --build

echo "== reloading Caddy"
sudo systemctl reload caddy

cat <<EOF

== instance $SLUG is up ==
link:      https://$SLUG.$ROADBOOK_DOMAIN
username:  $SLUG
password:  $PASSWORD

Record these in the ledger (docs/private/pilot/LEDGER.md on the laptop)
now, then build the handover message from the runbook. Two lines the
message must keep: copy the password rather than typing it, and no space
after the username (phone keyboards add one).
EOF
