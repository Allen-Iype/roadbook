#!/usr/bin/env python3
"""The front's funnel, counted from the masked request log (phase 15 CP4).

    sudo python3 scripts/pilot/host/funnel.py [--days 14] [--tz Asia/Kolkata]
                                             [--log-dir /var/log/caddy]
    sudo python3 scripts/pilot/host/funnel.py --tz Asia/Kolkata --record /srv/pilot/front

Reads access.log and its rotated siblings (access.log.1, access.log.*.gz —
setup-access-log.sh writes them, logrotate keeps seven days) and prints
one row per day:

    landing     the landing page served (www, GET /)
    ig app      of those, opened inside the Instagram app's own browser
    built       the "how it's built" page
    make mine   the app's sign-in page (www's button lands here)
    started     Google sign-in begun (/api/auth/signin)
    signed in   callback came back with a session (302 anywhere but /signin)
    refused     callback bounced to /signin (cap full, or an error)
    welcome     the welcome page, the first thing after sign-in
    inside      any other page behind the door (map, shelf, candidates, imports)
    uploads     POST /api/imports or /api/imports/photos — an upload begun
    demo        the demo instance's front page served past its basic auth
    bots        link previews, crawlers and scanners (user agent), excluded above
    people      distinct masked addresses on www+app that day, bots excluded
    back        of those, addresses already seen on an earlier day in the window

and, under the table, landing arrivals per source tag: the `s=` query on
a link you post (`www.<domain>/?s=ig-bio`, `?s=ig-story-nov3`), or a
`utm_source`/`utm_campaign` pair if a tool insists. Tag every link you
put anywhere; untagged arrivals print as "(none)". The tag survives only
the landing hit — there is no cookie to carry it through sign-in — so a
campaign is read as its landing count against that day's sign-ins.

`--record DIR` is the nightly mode (setup-access-log.sh installs the
timer): it appends YESTERDAY's row to DIR/funnel.csv and its sources to
DIR/sources.csv, once (a day already present is skipped), and prints
nothing; "back" there means seen in the seven days before. The log is kept seven days; these counts are kept for good —
they hold no address, only totals per day.

Only full page loads count as pages: Next's client navigations and link
prefetches (an `RSC` header or a `_rsc` query) and asset requests
(/_next/, fonts, shots, favicons) are skipped. A WhatsApp or Telegram
share fetches the page once to draw its preview, which is why bots are
split out rather than folded into "landing".

Caveats, so the figures are read right: an address is a /24 (IPv4) or /48
(IPv6), so two people on one carrier range are one "person", and one
person on wifi then cellular is two. Good enough to see whether five
friends became three arrivals; not a figure to publish. Standard library
only — the host has python3 and nothing else is wanted on it.
"""
import argparse
import datetime as dt
import glob
import gzip
import json
import os
import re
import sys
from collections import defaultdict
from urllib.parse import urlsplit

BOT_UA = re.compile(
    r"bot|crawl|spider|slurp|preview|whatsapp|telegram|facebookexternalhit|"
    r"twitter|slack|discord|skype|curl/|wget/|python-requests|go-http-client|"
    r"censys|zgrab|masscan|nmap|expanse|palo alto|scan",
    re.IGNORECASE,
)
ASSET = re.compile(r"^/(_next/|fonts/|shots/|favicon|apple-touch-icon|\.well-known/|robots\.txt|sitemap)")
INSIDE = re.compile(r"^/(adventures|candidates|imports|adventure/[^/]+|shared/[^/]+)$")

COLUMNS = ["landing", "ig app", "built", "make mine", "started", "signed in", "refused",
           "welcome", "inside", "uploads", "demo", "bots", "people", "back"]
IG_UA = re.compile(r"Instagram", re.IGNORECASE)


def open_any(path):
    return gzip.open(path, "rt", encoding="utf-8", errors="replace") if path.endswith(".gz") \
        else open(path, encoding="utf-8", errors="replace")


def log_files(log_dir):
    files = [os.path.join(log_dir, "access.log")]
    files += sorted(glob.glob(os.path.join(log_dir, "access.log.*")))
    return [f for f in files if os.path.isfile(f)]


def header(req, name):
    # Caddy logs headers under their canonical names as lists.
    h = req.get("headers") or {}
    for k, v in h.items():
        if k.lower() == name.lower():
            return v[0] if isinstance(v, list) and v else (v if isinstance(v, str) else "")
    return ""


def role(host):
    host = (host or "").split(":")[0].lower()
    for prefix, r in (("www.", "landing"), ("app.", "app"), ("demo.", "demo")):
        if host.startswith(prefix):
            return r
    return None


def source_of(query):
    """The tag on a posted link: `s=`, else utm_source[/utm_campaign]."""
    from urllib.parse import parse_qs
    q = parse_qs(query, keep_blank_values=False)
    if q.get("s"):
        return q["s"][0][:40]
    if q.get("utm_source"):
        tag = q["utm_source"][0]
        if q.get("utm_campaign"):
            tag += "/" + q["utm_campaign"][0]
        return tag[:40]
    return "(none)"


def classify(rec):
    """One request → (role, column or None, is_bot, is_page_load, source, ig)."""
    req = rec.get("request") or {}
    r = role(req.get("host"))
    if r is None:
        return None, None, False, False, None, False
    method = req.get("method", "")
    uri = req.get("uri", "/")
    parts = urlsplit(uri)
    path, query = parts.path or "/", parts.query
    status = int(rec.get("status", 0))
    ua = header(req, "User-Agent")
    is_bot = bool(BOT_UA.search(ua))
    if ASSET.match(path):
        return r, None, is_bot, False, None, False
    if header(req, "RSC") or "_rsc=" in query or header(req, "Next-Router-Prefetch"):
        return r, None, is_bot, False, None, False
    ok = status in (200, 304)
    col = None
    source = None
    ig = False
    if r == "landing":
        if method == "GET" and path in ("/", "/index.html") and ok:
            col = "landing"
            source = source_of(query)
            ig = bool(IG_UA.search(ua))
        elif method == "GET" and path == "/how-its-built.html" and ok:
            col = "built"
    elif r == "app":
        if method == "GET" and path == "/signin" and ok:
            col = "make mine"
        elif method == "GET" and path == "/api/auth/signin" and status in (302, 303, 307):
            col = "started"
        elif method == "GET" and path == "/api/auth/callback" and status in (302, 303, 307):
            loc = ((rec.get("resp_headers") or {}).get("Location") or [""])[0]
            col = "refused" if loc.startswith("/signin") else "signed in"
        elif method == "GET" and path == "/welcome" and ok:
            col = "welcome"
        elif method == "GET" and (path == "/" or INSIDE.match(path)) and ok:
            col = "inside"
        elif method == "POST" and path in ("/api/imports", "/api/imports/photos"):
            col = "uploads"
    elif r == "demo":
        if method == "GET" and path == "/" and ok:
            col = "demo"
    return r, col, is_bot, method == "GET" and ok, source, ig


def count(files, first, today, tz):
    """Every file → per-day column counts, people sets, per-day source counts."""
    counts = defaultdict(lambda: defaultdict(int))
    people = defaultdict(set)   # day → masked addresses (www+app, not bots)
    sources = defaultdict(lambda: defaultdict(int))
    unreadable = 0
    for f in files:
        with open_any(f) as fh:
            for line in fh:
                try:
                    rec = json.loads(line)
                except ValueError:
                    unreadable += 1
                    continue
                if rec.get("logger") not in (None, "http.log.access", "http.log.access.front"):
                    continue
                ts = rec.get("ts")
                if not isinstance(ts, (int, float)):
                    continue
                day = dt.datetime.fromtimestamp(ts, tz).date()
                if day < first or day > today:
                    continue
                r, col, is_bot, page, source, ig = classify(rec)
                if r is None:
                    continue
                if is_bot:
                    if page or col:
                        counts[day]["bots"] += 1
                    continue
                if col:
                    counts[day][col] += 1
                if col == "landing":
                    sources[day][source] += 1
                    if ig:
                        counts[day]["ig app"] += 1
                if r in ("landing", "app") and (page or col):
                    req = rec.get("request") or {}
                    addr = req.get("client_ip") or req.get("remote_ip") or "?"
                    people[day].add(addr)
    seen = set()
    for day in sorted(people):
        counts[day]["people"] = len(people[day])
        counts[day]["back"] = len(people[day] & seen)
        seen |= people[day]
    return counts, people, sources, unreadable


def record(args, tz):
    """Nightly: append yesterday once to funnel.csv and sources.csv under --record."""
    import csv
    yesterday = dt.datetime.now(tz).date() - dt.timedelta(days=1)
    files = log_files(args.log_dir)
    # Count the week before as well, so "back" means "seen in the previous
    # seven days" rather than "seen earlier yesterday".
    counts, _people, sources, _unreadable = count(
        files, yesterday - dt.timedelta(days=7), yesterday, tz)
    row = counts.get(yesterday, {})
    os.makedirs(args.record, exist_ok=True)
    fpath = os.path.join(args.record, "funnel.csv")
    spath = os.path.join(args.record, "sources.csv")
    day = yesterday.isoformat()
    if os.path.exists(fpath) and any(l.startswith(day + ",") for l in open(fpath)):
        return
    new = not os.path.exists(fpath) or os.path.getsize(fpath) == 0
    with open(fpath, "a", newline="") as fh:
        w = csv.writer(fh)
        if new:
            w.writerow(["day"] + COLUMNS)
        w.writerow([day] + [row.get(c, 0) for c in COLUMNS])
    new = not os.path.exists(spath) or os.path.getsize(spath) == 0
    with open(spath, "a", newline="") as fh:
        w = csv.writer(fh)
        if new:
            w.writerow(["day", "source", "landing"])
        for src, n in sorted(sources.get(yesterday, {}).items(), key=lambda kv: -kv[1]):
            w.writerow([day, src, n])


def main():
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    ap.add_argument("--days", type=int, default=14, help="window ending today (default 14)")
    ap.add_argument("--tz", default=None, help="day boundary, e.g. Asia/Kolkata (default: local)")
    ap.add_argument("--log-dir", default="/var/log/caddy")
    ap.add_argument("--record", metavar="DIR", default=None,
                    help="nightly mode: append yesterday to DIR/funnel.csv and DIR/sources.csv, print nothing")
    args = ap.parse_args()

    tz = None
    if args.tz:
        from zoneinfo import ZoneInfo
        tz = ZoneInfo(args.tz)
    if args.record:
        record(args, tz)
        return
    today = dt.datetime.now(tz).date()
    first = today - dt.timedelta(days=args.days - 1)

    files = log_files(args.log_dir)
    if not files:
        sys.exit(f"no access.log under {args.log_dir} — run setup-access-log.sh first")
    counts, people, sources, unreadable = count(files, first, today, tz)

    width = max(len(c) for c in COLUMNS)
    head = "day         " + " ".join(c.rjust(width) for c in COLUMNS)
    print(head)
    print("-" * len(head))
    total = defaultdict(int)
    for i in range(args.days):
        day = first + dt.timedelta(days=i)
        row = counts.get(day, {})
        if not row:
            continue
        print(day.isoformat() + "  " + " ".join(str(row.get(c, 0)).rjust(width) for c in COLUMNS))
        for c in COLUMNS:
            total[c] += row.get(c, 0)
    print("-" * len(head))
    all_people = set().union(*people.values()) if people else set()
    total["people"] = len(all_people)
    total["back"] = sum(1 for p in all_people if sum(p in s for s in people.values()) > 1)
    print("total       " + " ".join(str(total.get(c, 0)).rjust(width) for c in COLUMNS))
    by_source = defaultdict(int)
    for day_sources in sources.values():
        for src, n in day_sources.items():
            by_source[src] += n
    if by_source:
        print("\nlanding by source tag (the s= on the link you posted):")
        for src, n in sorted(by_source.items(), key=lambda kv: -kv[1]):
            print(f"  {src:<40} {n:>6}")
    print(f"\n{len(files)} file(s), window {first} → {today}"
          + (f", tz {args.tz}" if args.tz else ", local time")
          + (f", {unreadable} unreadable line(s)" if unreadable else "")
          + ". 'back' in the total row = addresses seen on more than one day.")


if __name__ == "__main__":
    main()
