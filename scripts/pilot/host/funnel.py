#!/usr/bin/env python3
"""The front's funnel, counted from the masked request log (phase 15 CP4).

    sudo python3 scripts/pilot/host/funnel.py [--days 14] [--tz Asia/Kolkata]
                                             [--log-dir /var/log/caddy]
    sudo python3 scripts/pilot/host/funnel.py --tz Asia/Kolkata --record /srv/pilot/front
    sudo python3 scripts/pilot/host/funnel.py --tz Asia/Kolkata --record /srv/pilot/front --recount

Reads access.log and its rotated siblings (access.log.1, access.log.*.gz —
setup-access-log.sh writes them, logrotate keeps seven days) and prints
one row per day:

    landing     the landing page served (www, GET /), bots and scanners excluded
    browsers    of those, loads followed within two minutes by a font or image
                fetch from the same address — a browser that drew the page
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
    bots        link previews, crawlers, monitors (user agent) and scanners (an
                address that probed /wp-…, /.env, /.git and the like that day),
                excluded above
    people      distinct masked addresses on www+app that day, bots excluded
    back        of those, addresses already seen on an earlier day in the window

and, under the table, landing arrivals per source tag: the `s=` query on
a link you post (`www.<domain>/?s=ig-bio`, `?s=ig-story-nov3`), or a
`utm_source`/`utm_campaign` pair if a tool insists. Tag every link you
put anywhere; untagged arrivals print as "(none)". The tag survives only
the landing hit — there is no cookie to carry it through sign-in — so a
campaign is read as its landing count against that day's sign-ins.

"landing" and "browsers" are an upper and a lower bound on real visits.
Landing still includes anything that fetches the page with a browser-like
user agent and never probes. Browsers misses a real visitor whose fonts
and screenshot were already cached (the landing is cached for an hour, so
a second look within the hour fetches nothing more). The truth sits
between the two, and on a quiet front it sits near "browsers".

`--record DIR` is the nightly mode (setup-access-log.sh installs the
timer): it appends YESTERDAY's row to DIR/funnel.csv and its sources to
DIR/sources.csv, once (a day already present is skipped), and prints
nothing; "back" there means seen in the seven days before. The log is kept seven days; these counts are kept for good —
they hold no address, only totals per day.

`--failed-from SLUG` (nightly mode only; the timer passes `app`) adds a
`failed` column: upload attempts that failed that day, read from that
stack's database through accounts.sh --failed-by-day. The front's log
cannot see it — an upload that fails still returns normally — and without
it a run of failures is found only when a person reports one. If the
database cannot be read the cell is left blank, never zero.

`--record DIR --recount` replaces, instead of skipping, the rows of every
whole day the log still covers — for when the counting rule itself changed
(2026-11-12: monitors and scanners were being counted as landing visits).
Days older than the log cannot be recounted and stay as first recorded; a
column added after a day was recorded is blank for that day, not zero.
For a recounted day "back" looks only as far back as the log reaches.

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
    r"censys|zgrab|masscan|nmap|expanse|palo alto|scan|monitor|sitewatch",
    re.IGNORECASE,
)
ASSET = re.compile(r"^/(_next/|fonts/|shots/|favicon|apple-touch-icon|\.well-known/|robots\.txt|sitemap)")
# Paths nothing here serves and no visitor asks for: the stock probes for
# other people's software. An address that asks for one is a scanner for
# that day, whatever user agent it wears on its other requests.
PROBE = re.compile(
    r"^/(wp-|wordpress|\.env|\.git|\.aws|\.ssh|\.vscode|\.DS_Store|phpmyadmin|pma/|"
    r"cgi-bin/|vendor/|actuator|xmlrpc|owa/|autodiscover|boaform|HNAP1|solr/|"
    r"config\.|backup|admin/)|\.(php|asp|aspx|jsp|cgi|sql|bak)$",
    re.IGNORECASE,
)
# What a browser fetches after the landing page, and a page-fetcher does not.
LANDING_ASSET = re.compile(r"^/(fonts/|shots/)")
BROWSER_WINDOW_S = 120
INSIDE = re.compile(r"^/(adventures|candidates|imports|adventure/[^/]+|shared/[^/]+)$")

# Recorded nightly but not printed by the table: it comes from the app's
# database, not from the log (see --failed-from).
RECORD_ONLY = ["failed"]
COLUMNS = ["landing", "browsers", "ig app", "built", "make mine", "started", "signed in", "refused",
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


def address(rec):
    req = rec.get("request") or {}
    return req.get("client_ip") or req.get("remote_ip") or "?"


def records(files, first, today, tz):
    """Every access line in the window → (day, ts, record); plus unreadable count."""
    out, unreadable, earliest = [], 0, None
    for f in files:
        with open_any(f) as fh:
            for line in fh:
                try:
                    rec = json.loads(line)
                except ValueError:
                    unreadable += 1
                    continue
                if not isinstance(rec, dict):
                    unreadable += 1
                    continue
                if rec.get("logger") not in (None, "http.log.access", "http.log.access.front"):
                    continue
                ts = rec.get("ts")
                if not isinstance(ts, (int, float)):
                    continue
                if earliest is None or ts < earliest:
                    earliest = ts
                day = dt.datetime.fromtimestamp(ts, tz).date()
                if day < first or day > today:
                    continue
                out.append((day, ts, rec))
    return out, unreadable, earliest


def count(files, first, today, tz):
    """Every file → per-day column counts, people sets, per-day source counts,
    unreadable lines, and the earliest timestamp any file holds."""
    recs, unreadable, earliest = records(files, first, today, tz)
    # First pass: who probed, on which day — on any host, including the bare
    # address, which is where most scanners arrive.
    scanners = defaultdict(set)
    for day, _ts, rec in recs:
        uri = (rec.get("request") or {}).get("uri", "/")
        if PROBE.search(urlsplit(uri).path or "/"):
            scanners[day].add(address(rec))

    counts = defaultdict(lambda: defaultdict(int))
    people = defaultdict(set)   # day → masked addresses (www+app, not bots)
    sources = defaultdict(lambda: defaultdict(int))
    landings = defaultdict(list)  # (day, address) → times the landing was served
    assets = defaultdict(list)    # address → times a font or image was served
    for day, ts, rec in recs:
        r, col, is_bot, page, source, ig = classify(rec)
        if r is None:
            continue
        addr = address(rec)
        if is_bot or addr in scanners[day]:
            if page or col:
                counts[day]["bots"] += 1
            continue
        if r == "landing":
            req = rec.get("request") or {}
            path = urlsplit(req.get("uri", "/")).path
            if LANDING_ASSET.match(path) and int(rec.get("status", 0)) in (200, 304):
                assets[addr].append(ts)
        if col:
            counts[day][col] += 1
        if col == "landing":
            sources[day][source] += 1
            landings[(day, addr)].append(ts)
            if ig:
                counts[day]["ig app"] += 1
        if r in ("landing", "app") and (page or col):
            people[day].add(addr)
    for (day, addr), times in landings.items():
        fetched = assets.get(addr, [])
        counts[day]["browsers"] += sum(
            1 for t in times if any(t <= a <= t + BROWSER_WINDOW_S for a in fetched))
    seen = set()
    for day in sorted(people):
        counts[day]["people"] = len(people[day])
        counts[day]["back"] = len(people[day] & seen)
        seen |= people[day]
    return counts, people, sources, unreadable, earliest


def failed_by_day(slug):
    """day (ISO) → failed upload attempts, from the stack's database; None
    when it cannot be read (stack down, no docker) — unknown, not zero."""
    import subprocess
    script = os.path.join(os.path.dirname(os.path.abspath(__file__)), "accounts.sh")
    try:
        out = subprocess.run(
            ["sh", script, "--failed-by-day", slug], stdin=subprocess.DEVNULL,
            capture_output=True, text=True, timeout=60, check=True).stdout
    except (OSError, subprocess.SubprocessError) as e:
        print(f"failed uploads not read for {slug}: {e}", file=sys.stderr)
        return None
    found = {}
    for row in out.splitlines():
        day, _, n = row.partition("|")
        if n.strip().isdigit():
            found[day.strip()] = int(n)
    return found


def record(args, tz):
    """Nightly: append yesterday once to funnel.csv and sources.csv under
    --record. With --recount: replace every whole day the log covers."""
    import csv
    yesterday = dt.datetime.now(tz).date() - dt.timedelta(days=1)
    files = log_files(args.log_dir)
    # Count the week before as well, so "back" means "seen in the previous
    # seven days" rather than "seen earlier yesterday".
    counts, _people, sources, _unreadable, earliest = count(
        files, yesterday - dt.timedelta(days=7), yesterday, tz)
    days = [yesterday]
    if args.recount and earliest is not None:
        # The day the log begins on is partial; every day after it is whole.
        first_whole = dt.datetime.fromtimestamp(earliest, tz).date() + dt.timedelta(days=1)
        days = [first_whole + dt.timedelta(days=i)
                for i in range((yesterday - first_whole).days + 1)]
    os.makedirs(args.record, exist_ok=True)
    fpath = os.path.join(args.record, "funnel.csv")
    spath = os.path.join(args.record, "sources.csv")

    # funnel.csv is read whole and written whole: a few hundred short rows,
    # and it lets a column added later take its place in the header with
    # the older days left blank — unknown, not zero.
    failed = failed_by_day(args.failed_from) if args.failed_from else None
    rows = {}
    if os.path.exists(fpath):
        with open(fpath, newline="") as fh:
            for old in csv.DictReader(fh):
                if old.get("day"):
                    rows[old["day"]] = old
    changed = False
    for d in days:
        day = d.isoformat()
        if day in rows and not args.recount:
            continue
        row = counts.get(d, {})
        rows[day] = {"day": day, **{c: row.get(c, 0) for c in COLUMNS}}
        # A day with no failures is absent from the query's answer: zero.
        # A database that was not asked, or did not answer: blank.
        rows[day]["failed"] = "" if failed is None else failed.get(day, 0)
        changed = True
    if not changed:
        return
    tmp = fpath + ".tmp"
    with open(tmp, "w", newline="") as fh:
        w = csv.DictWriter(fh, fieldnames=["day"] + COLUMNS + RECORD_ONLY,
                           restval="", extrasaction="ignore")
        w.writeheader()
        for day in sorted(rows):
            w.writerow(rows[day])
    os.replace(tmp, fpath)

    written = {d.isoformat() for d in days}
    kept = []
    if os.path.exists(spath):
        with open(spath, newline="") as fh:
            body = list(csv.reader(fh))[1:]
        already = {r[0] for r in body if r}
        if not args.recount:
            written -= already
        kept = [r for r in body if r and r[0] not in written]
    for d in days:
        if d.isoformat() not in written:
            continue
        for src, n in sorted(sources.get(d, {}).items(), key=lambda kv: -kv[1]):
            kept.append([d.isoformat(), src, str(n)])
    kept.sort(key=lambda r: r[0])  # stable: a day's sources stay in count order
    tmp = spath + ".tmp"
    with open(tmp, "w", newline="") as fh:
        w = csv.writer(fh)
        w.writerow(["day", "source", "landing"])
        w.writerows(kept)
    os.replace(tmp, spath)


def main():
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    ap.add_argument("--days", type=int, default=14, help="window ending today (default 14)")
    ap.add_argument("--tz", default=None, help="day boundary, e.g. Asia/Kolkata (default: local)")
    ap.add_argument("--log-dir", default="/var/log/caddy")
    ap.add_argument("--record", metavar="DIR", default=None,
                    help="nightly mode: append yesterday to DIR/funnel.csv and DIR/sources.csv, print nothing")
    ap.add_argument("--failed-from", metavar="SLUG", default=None,
                    help="with --record: add the day's failed uploads from that stack's database")
    ap.add_argument("--recount", action="store_true",
                    help="with --record: replace the rows of every whole day the log covers")
    args = ap.parse_args()

    tz = None
    if args.tz:
        from zoneinfo import ZoneInfo
        tz = ZoneInfo(args.tz)
    if args.recount and not args.record:
        ap.error("--recount goes with --record DIR")
    if args.failed_from and not args.record:
        ap.error("--failed-from goes with --record DIR")
    if args.record:
        record(args, tz)
        return
    today = dt.datetime.now(tz).date()
    first = today - dt.timedelta(days=args.days - 1)

    files = log_files(args.log_dir)
    if not files:
        sys.exit(f"no access.log under {args.log_dir} — run setup-access-log.sh first")
    counts, people, sources, unreadable, _earliest = count(files, first, today, tz)

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
