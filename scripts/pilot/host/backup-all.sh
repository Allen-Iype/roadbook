#!/bin/sh
# Back up every stamped instance on the VPS host (phase 10, BRIEF §3D).
# CP4's nightly systemd timer runs this; running it by hand any time is
# equally fine — backups are timestamped, never overwritten. On the host
# there is no special-cased demo project: the demo is a stamped instance
# like any other.
set -u

REPO=$(cd "$(dirname "$0")/../../.." && pwd)
PILOT_DIR=${ROADBOOK_PILOT_DIR:-$HOME/pilot}

FAIL=0
for d in "$PILOT_DIR"/instances/*/; do
  [ -d "$d" ] || continue
  "$REPO/scripts/pilot/host/backup-instance.sh" "$(basename "$d")" || FAIL=1
done
exit $FAIL
