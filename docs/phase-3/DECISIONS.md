# Phase 3 decisions

Three lines each: what was chosen, what was rejected, what would change our
mind. Written as decisions are made. The brief's §3 recommendations were
approved as written at Gate 1 (2026-08-04) and are not restated here; entries
below record decisions made during implementation.

## 2026-08-04 — the air golden fixture is the 30 Apr 2026 flight window, anonymised into a new frame

Chosen: `journey-30apr2026.anon.json` extracts the out-and-back flight
adventure (two air legs; Google labels only one activity FLYING — the case
that proves speed-over-mode), longitudes shifted by a constant that is
recorded nowhere in the repository, placeIds redacted to "ANON", latitudes
and timestamps real so every distance and duration equals the measurement.
Rejected: the 27jul fixture's documented frame (its CONTRACT table exposes
enough to invert the shift; a second fixture should not inherit that),
reusing candidate 63's window (a 19.5-day span for one flight — five times
the bytes for a weaker case), and classifying from the FLYING mode (this
very fixture shows it missing half the flights).
Would change our mind: needing cross-fixture geometry consistency (both
fixtures in one frame) — nothing foreseeable wants that.

## 2026-08-04 — OSRM setup automation is captured for phase 5, not built here

Chosen: checkpoint 3 keeps extract acquisition and preprocessing as the
maintainer's manual steps per the approved brief; the hosted-deployment
want (an OSRM compose profile + an operator-run setup script, spec'd by
docs/phase-3/OSRM.md) is recorded in PLAN's phase 5 features and BRIEF §7.
Rejected: writing scripts/osrm-setup.sh now — it front-runs phase 5's
compose file and contradicts the Gate 1 brief mid-checkpoint; also
rejected: OSRM on the serve host as a requirement (serve reads only the
cache; the batch runs wherever OSRM is, against any reachable database).
Would change our mind: phase 5 arriving — the script lands there against
the real compose file, explicitly invoked, never at build or install time.

## 2026-08-04 — teleport rejection at assembly: cluster-level, bridge-conditioned

Chosen: assembly rejects a cluster of consecutive points (within
ClusterRadiusM, default 1000 m) iff an entry-or-exit implied speed exceeds
MaxSpeedKmh (default 900, the detector's constant) AND bridging its
neighbours directly is plausible; edge durations floor at
ThinSpacingSeconds because trace timestamps are minute-truncated (a
same-minute 130 m step is quantization, not teleportation — a false
positive caught by the 30 Apr golden during implementation). Flagged as
RejectedSpeed, rows untouched. Found because the maintainer spotted a
straight green line on the map: a mislocated geolocation-database entry
placed six fixes ~297 km away across three days (candidate 62), and
assembly had no speed filter — only detection did.
Rejected: the detector's point-wise both-neighbours rule and Dawarich's
speed sandwich (a run's interior is self-consistent — both pass runs
wholesale); speed-from-last-accepted anchoring (an overnight silence
launders the run's entry speed); either-side point-level rejection (kills
real flight endpoints).
Would change our mind on the bridge condition: a real teleport whose
neighbours are also implausibly far apart — the rule then keeps it,
conservatively, and the map shows the contradiction rather than an
invented resolution; that case appearing in practice reopens this entry.
