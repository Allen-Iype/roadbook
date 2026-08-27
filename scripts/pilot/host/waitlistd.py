#!/usr/bin/env python3
"""Roadbook waitlist intake (phase 12 CP2).

One job: accept the landing page's waitlist form and append the entry to
one file. Deliberately tiny and stdlib-only — the host has no Go
toolchain, and a build step for sixty lines would be machinery. This is
operator machinery, not product code (PRODUCT.md boundary); the product
never reads this file.

Shape:
  POST /waitlist  (urlencoded: email, website=honeypot)
      -> 303 /joined.html   on accept (and on honeypot hits, which are
                               silently dropped — a bot gets no signal)
      -> 303 /oops.html     on a value that cannot be an email address
  anything else   -> 404/405

Custody, as promised on the page: the entry file holds a UTC timestamp
and the address, nothing else. The client IP is used only for an
in-memory rate limit (Caddy's rate-limit plugin is not in our build) and
is never written. The file is 0600, lives at $WAITLIST_FILE, and is
deliberately excluded from nightly backups so "deleted on request" is
absolute — one file, one place.

Runs on loopback behind Caddy (reverse_proxy /waitlist). systemd unit:
scripts/pilot/host/roadbook-waitlist.service; setup-waitlist.sh installs.
"""

import fcntl
import os
import re
import time
import urllib.parse
from datetime import datetime, timezone
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

ADDR = ("127.0.0.1", int(os.environ.get("WAITLIST_PORT", "9310")))
FILE = os.environ.get("WAITLIST_FILE", "/srv/pilot/waitlist/entries.log")
MAX_BODY = 4096
RATE_MAX = 5            # accepted writes per IP...
RATE_WINDOW = 3600      # ...per hour; in memory only, resets on restart

# Loose on purpose: the entry email is the real verification (a bad
# address never receives entry, self-correcting). We only refuse what
# cannot be an address or would corrupt a line-oriented log.
EMAIL_RE = re.compile(r"^[^@\s]+@[^@\s]+\.[^@\s]+$")

_hits: dict[str, list[float]] = {}


def _allowed(ip: str) -> bool:
    now = time.monotonic()
    hits = [t for t in _hits.get(ip, []) if now - t < RATE_WINDOW]
    if len(hits) >= RATE_MAX:
        _hits[ip] = hits
        return False
    hits.append(now)
    _hits[ip] = hits
    return True


class Handler(BaseHTTPRequestHandler):
    server_version = "roadbook-waitlist"

    def _redirect(self, where: str) -> None:
        self.send_response(303)
        self.send_header("Location", where)
        self.send_header("Content-Length", "0")
        self.end_headers()

    def do_POST(self) -> None:  # noqa: N802 (stdlib naming)
        if self.path != "/waitlist":
            self.send_error(404)
            return
        try:
            length = min(int(self.headers.get("Content-Length", "0")), MAX_BODY)
            form = urllib.parse.parse_qs(
                self.rfile.read(length).decode("utf-8", "replace")
            )
        except (ValueError, OSError):
            self._redirect("/oops.html")
            return

        # Honeypot: the CSS-hidden field a person never sees. A filled one
        # is a bot; answer success and store nothing.
        if form.get("website", [""])[0].strip():
            self._redirect("/joined.html")
            return

        email = form.get("email", [""])[0].strip()
        if len(email) > 254 or not EMAIL_RE.match(email):
            self._redirect("/oops.html")
            return

        # Caddy fronts us; the first X-Forwarded-For hop is the client.
        ip = (self.headers.get("X-Forwarded-For") or
              self.client_address[0]).split(",")[0].strip()
        if not _allowed(ip):
            # Over the limit: no write, but no signal either.
            self._redirect("/joined.html")
            return

        line = "%s\t%s\n" % (
            datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"), email)
        fd = os.open(FILE, os.O_WRONLY | os.O_APPEND | os.O_CREAT, 0o600)
        try:
            with os.fdopen(fd, "a") as f:
                fcntl.flock(f, fcntl.LOCK_EX)
                f.write(line)
        except OSError:
            self._redirect("/oops.html")
            return
        self._redirect("/joined.html")

    def do_GET(self) -> None:  # noqa: N802
        # Nothing to read here, ever — a browser landing on the endpoint
        # goes back to the page.
        self._redirect("/")

    def log_message(self, *args) -> None:
        # Quiet by design: request logs would collect IPs and addresses
        # the custody promise says we do not keep.
        pass


if __name__ == "__main__":
    os.makedirs(os.path.dirname(FILE), mode=0o700, exist_ok=True)
    ThreadingHTTPServer(ADDR, Handler).serve_forever()
