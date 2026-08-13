#!/bin/sh
# Back up the demo project and every stamped instance (phase 8 BRIEF §3D).
# The nightly cron entry runs this (RUNBOOK section 8); running it by hand
# any time is equally fine — backups are timestamped, never overwritten.
set -u
export PATH=/opt/homebrew/bin:/usr/local/bin:$PATH  # cron-safe

REPO=$(cd "$(dirname "$0")/../.." && pwd)
PILOT_DIR=${ROADBOOK_PILOT_DIR:-$REPO/docs/private/pilot}

FAIL=0
"$REPO/scripts/pilot/backup-instance.sh" demo || FAIL=1
for d in "$PILOT_DIR"/instances/*/; do
  [ -d "$d" ] || continue
  "$REPO/scripts/pilot/backup-instance.sh" "$(basename "$d")" || FAIL=1
done
exit $FAIL
