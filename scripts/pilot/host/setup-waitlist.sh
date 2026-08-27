#!/bin/sh
# Install (or refresh) the waitlist intake service on the host (phase 12
# CP2). Idempotent: safe to re-run after a git pull that changed
# waitlistd.py — systemd restarts on the new file.
#
#   scripts/pilot/host/setup-waitlist.sh
#
# The entries file lives under /srv/pilot/waitlist (0700, ubuntu) and is
# deliberately NOT covered by the nightly backup — the page promises the
# address lives in one file, deletable absolutely (DECISIONS 2026-08-27).
# Reading the list: `cat /srv/pilot/waitlist/entries.log` — one
# tab-separated line per entry, UTC timestamp + address.
set -eu

REPO=$(cd "$(dirname "$0")/../../.." && pwd)
PILOT_DIR=${ROADBOOK_PILOT_DIR:-/srv/pilot}

sudo mkdir -p "$PILOT_DIR/waitlist"
sudo chown ubuntu:ubuntu "$PILOT_DIR/waitlist"
sudo chmod 700 "$PILOT_DIR/waitlist"

sudo cp "$REPO/scripts/pilot/host/roadbook-waitlist.service" /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now roadbook-waitlist.service
sudo systemctl restart roadbook-waitlist.service
sleep 1
systemctl is-active roadbook-waitlist.service
echo "waitlist intake on 127.0.0.1:9310 -> $PILOT_DIR/waitlist/entries.log"
