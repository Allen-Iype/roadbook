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
