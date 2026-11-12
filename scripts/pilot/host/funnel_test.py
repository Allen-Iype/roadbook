#!/usr/bin/env python3
"""Tests for funnel.py on a made-up access log (no real request is in here).

    python3 scripts/pilot/host/funnel_test.py

Standard library only, like the script. Addresses are documentation
ranges (RFC 5737); hosts are example.org.
"""
import csv
import datetime as dt
import importlib.util
import json
import os
import tempfile
import unittest

HERE = os.path.dirname(os.path.abspath(__file__))
spec = importlib.util.spec_from_file_location("funnel", os.path.join(HERE, "funnel.py"))
funnel = importlib.util.module_from_spec(spec)
spec.loader.exec_module(funnel)

UTC = dt.timezone.utc
BROWSER = "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) Safari/604.1"
MONITOR = "Mozilla/5.0 (compatible; sitewatch/1.0; contract-address monitor)"


def at(day, h, m=0, s=0):
    return dt.datetime(2026, 11, day, h, m, s, tzinfo=UTC).timestamp()


def line(ts, addr, uri, ua=BROWSER, host="www.example.org", status=200, method="GET"):
    return json.dumps({
        "logger": "http.log.access.front", "ts": ts, "status": status,
        "request": {"host": host, "method": method, "uri": uri, "client_ip": addr,
                    "headers": {"User-Agent": [ua]}},
    })


def read_csv(directory, name):
    with open(os.path.join(directory, name), newline="") as fh:
        return list(csv.reader(fh))


def funnel_rows(directory):
    body = read_csv(directory, "funnel.csv")
    return {r[0]: dict(zip(body[0], r)) for r in body[1:]}


def write_log(directory, lines):
    with open(os.path.join(directory, "access.log"), "w") as fh:
        fh.write("\n".join(lines) + "\n")


class Counting(unittest.TestCase):
    def count(self, lines, first=5, last=6):
        with tempfile.TemporaryDirectory() as d:
            write_log(d, lines)
            counts, people, sources, unreadable, earliest = funnel.count(
                funnel.log_files(d), dt.date(2026, 11, first), dt.date(2026, 11, last), UTC)
        return counts

    def test_a_browser_is_a_landing_and_a_browser(self):
        c = self.count([
            line(at(5, 9), "192.0.2.0", "/?s=ig-bio"),
            line(at(5, 9, 0, 1), "192.0.2.0", "/fonts/body.woff2"),
        ])[dt.date(2026, 11, 5)]
        self.assertEqual((c["landing"], c["browsers"], c["bots"], c["people"]), (1, 1, 0, 1))

    def test_a_page_fetch_without_assets_is_a_landing_only(self):
        c = self.count([line(at(5, 9), "192.0.2.0", "/")])[dt.date(2026, 11, 5)]
        self.assertEqual((c["landing"], c["browsers"]), (1, 0))

    def test_an_asset_after_the_window_does_not_count(self):
        c = self.count([
            line(at(5, 9), "192.0.2.0", "/"),
            line(at(5, 9, 2, 1), "192.0.2.0", "/shots/app.png"),
        ])[dt.date(2026, 11, 5)]
        self.assertEqual(c["browsers"], 0)

    def test_another_address_fetching_assets_does_not_count(self):
        c = self.count([
            line(at(5, 9), "192.0.2.0", "/"),
            line(at(5, 9, 0, 1), "198.51.100.0", "/fonts/body.woff2"),
        ])[dt.date(2026, 11, 5)]
        self.assertEqual(c["browsers"], 0)

    def test_the_monitor_is_a_bot(self):
        c = self.count([line(at(5, h), "203.0.113.0", "/", ua=MONITOR) for h in range(6)])
        c = c[dt.date(2026, 11, 5)]
        self.assertEqual((c["landing"], c["bots"], c["people"]), (0, 6, 0))

    def test_a_prober_is_a_scanner_for_the_day_even_before_it_probes(self):
        c = self.count([
            line(at(5, 9), "203.0.113.0", "/"),                       # browser-like, earlier
            line(at(5, 10), "203.0.113.0", "/wp-admin/install.php", status=404),
            line(at(5, 11), "203.0.113.0", "/.env", host="203.0.113.9", status=404),
            line(at(6, 9), "203.0.113.0", "/"),                       # next day, no probe
        ])
        five, six = c[dt.date(2026, 11, 5)], c[dt.date(2026, 11, 6)]
        self.assertEqual((five["landing"], five["bots"], five["people"]), (0, 1, 0))
        self.assertEqual((six["landing"], six["people"]), (1, 1))

    def test_a_probe_on_the_bare_address_marks_the_scanner(self):
        c = self.count([
            line(at(5, 9), "203.0.113.0", "/.git/config", host="203.0.113.9", status=404),
            line(at(5, 10), "203.0.113.0", "/"),
        ])[dt.date(2026, 11, 5)]
        self.assertEqual((c["landing"], c["bots"]), (0, 1))

    def test_the_apps_own_paths_are_not_probes(self):
        for path in ("/", "/signin", "/welcome", "/adventure/12", "/api/auth/callback",
                     "/api/imports", "/how-its-built.html", "/privacy.html",
                     "/.well-known/security.txt", "/shots/app.png", "/_next/static/a.js"):
            self.assertIsNone(funnel.PROBE.search(path), path)


class Recording(unittest.TestCase):
    """--record against a log whose 'yesterday' is pinned by the test."""

    def run_record(self, d, recount):
        class Args:
            pass
        a = Args()
        a.log_dir, a.record, a.recount = d, d, recount
        funnel.record(a, UTC)

    def setUp(self):
        self.today = dt.datetime.now(UTC).date()

    def ts(self, days_ago, h):
        day = self.today - dt.timedelta(days=days_ago)
        return dt.datetime(day.year, day.month, day.day, h, tzinfo=UTC).timestamp()

    def day(self, days_ago):
        return (self.today - dt.timedelta(days=days_ago)).isoformat()

    def test_an_old_file_gains_the_new_column_blank_and_recount_replaces(self):
        old_columns = [c for c in funnel.COLUMNS if c != "browsers"]
        with tempfile.TemporaryDirectory() as d:
            # The log begins three days ago (a partial day), then two whole days.
            write_log(d, [
                line(self.ts(3, 20), "192.0.2.0", "/"),
                line(self.ts(2, 9), "192.0.2.0", "/"),
                line(self.ts(2, 9) + 1, "192.0.2.0", "/fonts/body.woff2"),
                line(self.ts(2, 10), "203.0.113.0", "/", ua=MONITOR),
                line(self.ts(1, 9), "198.51.100.0", "/?s=ig-bio"),
            ])
            # funnel.csv as the old script left it: no browsers column, the
            # monitor counted as a visit, and one day older than the log.
            with open(os.path.join(d, "funnel.csv"), "w", newline="") as fh:
                w = csv.writer(fh)
                w.writerow(["day"] + old_columns)
                w.writerow([self.day(9)] + [7] * len(old_columns))
                w.writerow([self.day(2)] + [2] * len(old_columns))
            with open(os.path.join(d, "sources.csv"), "w", newline="") as fh:
                w = csv.writer(fh)
                w.writerow(["day", "source", "landing"])
                w.writerow([self.day(9), "(none)", 7])
                w.writerow([self.day(2), "(none)", 2])

            self.run_record(d, recount=False)   # nightly: adds yesterday only
            rows = funnel_rows(d)
            self.assertEqual(list(rows), [self.day(9), self.day(2), self.day(1)])
            self.assertEqual(rows[self.day(9)]["browsers"], "")   # unknown, not zero
            self.assertEqual(rows[self.day(9)]["landing"], "7")
            self.assertEqual(rows[self.day(2)]["landing"], "2")   # untouched
            self.assertEqual(rows[self.day(1)]["landing"], "1")

            self.run_record(d, recount=False)   # idempotent
            self.assertEqual(len(read_csv(d, "funnel.csv")), 4)

            self.run_record(d, recount=True)    # whole days in the log are replaced
            rows = funnel_rows(d)
            self.assertEqual(rows[self.day(2)]["landing"], "1")
            self.assertEqual(rows[self.day(2)]["browsers"], "1")
            self.assertEqual(rows[self.day(2)]["bots"], "1")
            self.assertEqual(rows[self.day(9)]["landing"], "7")   # older than the log: kept
            self.assertNotIn(self.day(3), rows)                    # partial day: not written
            src = read_csv(d, "sources.csv")[1:]
            self.assertEqual(src, [[self.day(9), "(none)", "7"],
                                   [self.day(2), "(none)", "1"],
                                   [self.day(1), "ig-bio", "1"]])


if __name__ == "__main__":
    unittest.main()
