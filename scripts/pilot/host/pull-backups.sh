#!/bin/sh
# Pull the host's encrypted backup staging to the operator's machine
# (phase 10 CP4, BRIEF §3D: laptop-pull — the VPS holds no credential to
# reach the operator; the operator reaches the VPS). Runs ON THE OPERATOR'S
# MACHINE, via its LaunchAgent (com.roadbook.pilot.pull) or by hand.
#
#   ROADBOOK_HOST=<tailnet address> scripts/pilot/host/pull-backups.sh
#
# Copies ~/backups/*.tar.age (ciphertext only — the age secret never
# touches the host, so what transits here is unreadable in transit and at
# rest on the host) into the same iCloud directory the laptop-era chain
# proved, under host/. rsync makes re-runs incremental; archives are
# timestamped and never overwritten. Transport is Tailscale SSH: if the
# tailnet asks for its periodic operator re-auth, an unattended run fails
# and logs — a later interactive ssh surfaces the prompt, and the next
# pull catches up (archives accumulate on the host meanwhile).
set -eu
export PATH=/opt/homebrew/bin:/usr/local/bin:$PATH  # launchd-safe

HOST=${ROADBOOK_HOST:?set ROADBOOK_HOST to the host tailnet address}
DEST=${ROADBOOK_BACKUP_DIR:-"$HOME/Library/Mobile Documents/com~apple~CloudDocs/RoadbookBackups"}/host

mkdir -p "$DEST"
rsync -av --ignore-existing "ubuntu@$HOST:backups/" "$DEST/"
echo "pull complete: $(ls "$DEST" | wc -l | tr -d ' ') archives in $DEST"
