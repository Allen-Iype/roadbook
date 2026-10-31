#!/bin/sh
# Switch on the host front's request log, masked and short-lived (phase 15
# CP4, 2026-10-31).
#
#   scripts/pilot/host/setup-access-log.sh
#
# What it answers: how many people open the landing, how many tap through
# to sign-in, how many come out the other side, how many reach an upload.
# The users table answers only the last three; the first two happen before
# an account exists, so only the front can see them. funnel.py reads what
# this writes.
#
# What it keeps: one JSON line per request — time, host, method, path,
# status, size, referer, user agent — with the address masked by Caddy
# BEFORE it is written (last octet of IPv4, last 80 bits of IPv6) and the
# Authorization, Cookie and Set-Cookie headers deleted. Nothing in the
# file names a person, nothing is joined to an account, and nothing lives
# past seven days: logrotate rotates daily and keeps seven. What outlives
# the log is totals: a timer runs funnel.py --record every morning and
# appends yesterday's counts (no address, no agent, just numbers per day
# and per source tag) to /srv/pilot/front/funnel.csv and sources.csv —
# the trend line a campaign is read against. The public page
# (landing/how-its-built.html, "Where your data lives") states both in
# one sentence; keep the three in step.
#
# front.env may set ROADBOOK_FUNNEL_TZ (e.g. Asia/Kolkata) — the day
# boundary the nightly row uses; UTC otherwise.
#
# Caddy's `log` is a site-level directive, so it cannot ride inside the
# host-matched `handle` snippets the other scripts write. It lives in its
# own snippet imported at site level — the same once-only operator edit
# of /etc/caddy/Caddyfile that deploy-landing.sh asks for, and this script
# checks for it the same way. Caddy's own size-based rolling is disabled:
# traffic here is a few lines a day and a size roll would never fire, so
# "seven days" would not hold. logrotate's copytruncate is safe with
# Caddy's writer (it appends).
set -eu

PILOT_DIR=${ROADBOOK_PILOT_DIR:-/srv/pilot}
FRONT=$PILOT_DIR/front
LOG_DIR=/var/log/caddy
SNIPPET=$FRONT/access-log.conf
HERE=$(cd "$(dirname "$0")" && pwd)
[ -f "$PILOT_DIR/front.env" ] && . "$PILOT_DIR/front.env"
FUNNEL_TZ=${ROADBOOK_FUNNEL_TZ:-UTC}

sudo mkdir -p "$LOG_DIR"
sudo chown caddy:caddy "$LOG_DIR"
sudo chmod 750 "$LOG_DIR"
sudo mkdir -p "$FRONT"
sudo chmod 755 "$FRONT"

# The masks are applied by Caddy before the write — the unmasked address
# never touches disk. Both fields: remote_ip is the TCP peer, client_ip is
# the same here (no proxy in front) but logged separately.
sudo tee "$SNIPPET" >/dev/null <<CADDY
# Written by setup-access-log.sh — do not edit by hand.
log front {
	output file $LOG_DIR/access.log {
		roll_disabled
		mode 0640
	}
	format filter {
		request>remote_ip ip_mask {
			ipv4 24
			ipv6 48
		}
		request>client_ip ip_mask {
			ipv4 24
			ipv6 48
		}
		request>headers>Authorization delete
		request>headers>Cookie delete
		resp_headers>Set-Cookie delete
	}
}
CADDY
sudo chmod 644 "$SNIPPET"

sudo tee /etc/logrotate.d/caddy-access >/dev/null <<ROTATE
# Written by setup-access-log.sh — the seven-day promise lives here.
$LOG_DIR/access.log {
	daily
	rotate 7
	missingok
	notifempty
	copytruncate
	compress
	delaycompress
}
ROTATE
sudo chmod 644 /etc/logrotate.d/caddy-access

# The nightly totals. 01:30 UTC: after logrotate's daily run (00:00 + up
# to an hour) and, for any Asian or European day boundary, after
# yesterday has ended. Persistent=true catches a night the host slept
# through (reboots are 23:30 UTC). Root reads the 0640 log.
sudo tee /etc/systemd/system/roadbook-funnel.service >/dev/null <<UNIT
[Unit]
Description=Roadbook front funnel — append yesterday's totals
After=caddy.service

[Service]
Type=oneshot
ExecStart=/usr/bin/python3 -I $HERE/funnel.py --log-dir $LOG_DIR --tz $FUNNEL_TZ --record $FRONT
UNIT
sudo tee /etc/systemd/system/roadbook-funnel.timer >/dev/null <<UNIT
[Unit]
Description=Roadbook front funnel, nightly

[Timer]
OnCalendar=*-*-* 01:30:00 UTC
Persistent=true

[Install]
WantedBy=timers.target
UNIT
sudo systemctl daemon-reload
sudo systemctl enable --now roadbook-funnel.timer >/dev/null

if ! grep -q "import $SNIPPET" /etc/caddy/Caddyfile; then
  echo "" >&2
  echo "ACTION NEEDED: /etc/caddy/Caddyfile does not import the access-log snippet." >&2
  echo "Add this line inside the site block, next to the other imports:" >&2
  echo "" >&2
  echo "    import $SNIPPET" >&2
  echo "" >&2
  echo "then re-run this script." >&2
  exit 1
fi

# Validate as the caddy user (a root validate lies about readability), then
# reload. logrotate is dry-run so a typo in the rotate file fails here, not
# at 00:00.
sudo -u caddy caddy validate --config /etc/caddy/Caddyfile
sudo logrotate -d /etc/logrotate.d/caddy-access >/dev/null 2>&1 || { echo "logrotate config invalid" >&2; exit 1; }
sudo systemctl reload caddy
echo "access log on: $LOG_DIR/access.log (masked, 7 days); totals nightly → $FRONT/funnel.csv (tz $FUNNEL_TZ)."
echo "Count now: sudo python3 $HERE/funnel.py --tz $FUNNEL_TZ"
