# Phase 23 — log: recorded stretches drawn along the road

What the phase does, what broke, and why each fix took the form it did.
BRIEF.md and DECISIONS.md are the other two artifacts; this closes the set.

## What the phase does

A recorded stretch is drawn along the road its points lie on (#9). It is
still "Observed": the charter decision of 2026-11-16 amended invariant 6
and refused a third line kind. Recorded points are never moved, and every
distance is still measured between them.

- **A matcher beside the router** (`internal/route/match.go`). `Matcher`
  has one method: timed points in, a road line and its length out, or "no
  match". OSRM's `/match` implements it; the null router matches nothing.
- **The match cache** (migration 18). Keyed by a SHA-256 of the request:
  the stretch's points, the profile and the search radius. `roadbook
  route` fills it; `roadbook serve` only reads it.
- **The guard is applied on reading.** A matched road is drawn only when
  it is under `MATCH_MAX_RATIO` (1.5) times the point-to-point length and
  both ends of the stretch were placed. Changing the ratio needs no new
  requests. It is a server setting (`-match-max-ratio`,
  `ROADBOOK_MATCH_MAX_RATIO`) and is served with every journey.
- **Modes that are not on roads.** A stretch or a gap whose longest
  overlapping activity is labelled train, subway, ferry or flight
  (`MATCH_SKIP_MODES`) is never drawn along a road. A gap labelled a
  flight and at least `AIR_LABEL_MIN_KM` (100) long is drawn as a flight
  even under the speed line.
- **The web** has one drawing rule (`legLine`, `web/lib/geo.ts`) for the
  maps, the thumbnails and the exported image, and one new line in "How
  this route was drawn".
- **The road caches belong to an account** (migration 19) and are deleted
  with it.

By checkpoint:

- **STOP 0 — brief** (3ebe0dc), invariant 6 amended (b3a2ef0).
- **STOP 1 — Go** (8bbd90a, 83aeb94).
- **STOP 2 — web, and the guard chosen by eye** (896a68d).
- **Addendum — flights and trains in gaps** (476f37e).
- **Owned caches**, then **STOP 3** measured and not taken further.

## What was measured

Counts and distances only; the files under `measure/` print them.

- **The need** (`hops_test.go.txt`): about half of recorded distance lies
  in hops of more than 5 km on all three real datasets.
- **Matching as built, guard 1.3** (`built_test.go.txt`): 78%, 83% and
  75% of recorded km drawn along roads on the fixture, the archive and
  the iPhone export, the roads 1.08 to 1.10 times point to point. Set
  against Google's own distances, drawing along roads moved the total
  away from Google on two of the three, which is why distances stay
  measured between recorded points.
- **The guard**: over the archive's 67 trips, 80% of recorded km at 1.3
  and 84% at 1.5 (train, subway, ferry and flight stretches skipped); 29
  trips are identical at both.
- **Gaps by label** (`gapmodes_test.go.txt`): 2 and 3 long gaps labelled
  flying but drawn as road on the archive and the iPhone export; 84 and
  134 gaps labelled train, subway or ferry sent to the road router.
  After the fix, on the archive: air 12,588 → 14,135 km, routed 41,826 →
  36,253 km, unknown 0 → 3,051 km.
- **The router's memory**: 13.2 GB loaded normally for the India
  dataset; in file-mapped mode 0.65 GB of its own with at most 2.7 GB of
  the files resident, for the same answers.

## What was verified

- `make test` with the database suites: every pinned count and both
  journey goldens unchanged throughout.
- Table tests written first for the guard, the ends rule, the mode rule,
  skipped modes, the flight-by-label rule, unrouted rail gaps, the owned
  caches and deletion.
- Web: tsc clean, lint 0 errors, vitest 166. Playwright 139 passed, 26
  skipped, against a demo instance, before the gap fix; the only web
  change after it is one sentence of wording. The demo has no stretch
  that follows a road, so that run shows nothing broke and does not show
  the new line.
- The three web fixtures regenerated from a fresh demo instance twice.
- Read in a browser by the maintainer on the archive, at two guards.

## What broke, and the shape of each fix

1. **The brief's fixture figures did not survive the build.** 842
   matched and 82% at 1.3 came from a looser rule than the one built
   (the ends rule, the skipped modes). The decisions carry the figures
   as built.
2. **Some match requests are slow.** About one in ten takes over five
   seconds and a whole history takes 27 to 86 minutes, once. Timestamps
   and the `gaps` option were ruled out; the cause is inside the
   matcher and was not found. Accepted for a batch.
3. **Trains were drawn on the highway beside them.** Found by asking
   before the web stop; the per-mode table showed rail stretches passing
   the guard. Fixed with a named list of modes, not a ratio.
4. **The router's default refuses long stretches** (100 points). The
   compose file and the OSRM notes pass `--max-matching-size 2000`.
5. **Flights and trains in gaps were drawn along roads.** Found by the
   maintainer at STOP 2 on one trip. Not matching: gap routing since
   phase 3, which read only speed. A flight's silence includes time at
   the airport. Fixed by reading the export's label for gaps too; a rail
   gap stays a dashed line until there is a rail router.
6. **A cache nobody could delete.** `match_cache` followed
   `route_cache` in being global. Both now have an owner, because a road
   derived from a person's history is part of it.
7. **A decision about fixtures was broken once.** One constant was
   added to the web fixtures by hand against the rule that they are
   regenerated; recorded at the time, and the later regeneration showed
   them identical.

## Carried out of the phase

- **The hosted instance has no router.** Memory is no longer the
  obstacle; the trial on the host was not run. Before it can be:
  `roadbook route` must run for every account. Phase 15 §3F stays open,
  and the tester who raised #9 does not see the change.
- **A rail router** for train gaps and stretches.
- **Slow match requests**, unexplained.
- **Hops of more than 50 km inside a recorded stretch** on the iPhone
  export (BRIEF §2.1).
- **Host deploy after the merge:** app and demo stacks rebuilt;
  migrations 18 and 19 apply at start-up; landing redeployed for the
  how-it's-built sentences.
