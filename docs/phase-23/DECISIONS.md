# Phase 23 — decisions

Three lines each: what was chosen, what was rejected, what would change
our mind. Written as each was made.

## 2026-11-16 — No third line kind; invariant 6 amended

- **Chosen:** a recorded stretch may be drawn along the road its points
  lie on and stays "Observed". Recorded points are never altered or
  discarded; where no road fits, the stretch is drawn point to point;
  gaps remain their own kind. The maintainer's reasoning: the points
  say the traveller was there, and joining two of them by a road
  instead of a straight line misleads nobody. `CLAUDE.md` invariant 6
  carries the new wording.
- **Rejected:** a third, visibly distinct "snapped" kind (first
  recommended, withdrawn: the straight line between two points was
  already a guess); dropping the distinction for gaps too (nothing was
  recorded there, and the road chosen is one of several).
- **Would change our mind:** readers taking a matched line for a
  recorded track in a way that matters to them.

## 2026-11-19 — The brief's five choices, as recommended

- **Chosen:** A, each recorded stretch is matched as a whole. B, a
  matched line is kept when it is under 1.3 times point to point by
  default, and the value is fixed by the maintainer at the web stop
  after looking at their own trips at 1.3 and 1.5. C, a 100 m search
  radius. D, a trip's kilometres and pace stay computed from the
  recorded points; the road length appears only under "How this route
  was drawn". E, built for wherever the routing batch runs, with a
  router on the hosted instance as its own checkpoint at the end,
  measured before anything is promised.
- **Rejected:** routing hop by hop (1.45 times point to point on the
  fixture against 1.21); the matcher's own confidence as the guard
  (near zero throughout); a 250 m radius (single requests ran for
  minutes); the public routing server for hosted users (a trip is
  hundreds of requests, and positions would leave the service).
- **Would change our mind:** STOP 1's comparison with Google's own
  activity distances, if matched lengths disagree with them more than
  point-to-point lengths do.

## 2026-11-19 — The cache key is the request: a hash of the points sent

- **Chosen:** a match is cached under a SHA-256 of exactly what the
  matcher was sent — each point's coordinate rounded to five decimals and
  its time in whole seconds — with the profile and the search radius. A
  hit is therefore exact, as with the routing cache's rounded endpoint
  pair. The guard is not in the key: it is applied on reading.
- **Rejected:** keying by trip and leg number (a re-detection that moves
  a trip's edge renumbers every leg and would orphan every answer);
  keying by the first and last point only (two stretches can share ends
  and differ between them).
- **Would change our mind:** a re-import that shifts recorded points by
  less than a metre and so re-asks everything; not seen.

## 2026-11-19 — One stretch, one line; both ends placed

- **Chosen:** when the matcher can only fit a stretch as several
  disconnected pieces, that is cached as no match. When it fits one line
  but leaves the first or last point off the road, the row is kept with
  that fact (`ends_placed`), and the line is not drawn: it would stop
  short of the stretch, and being shorter than the points it would pass
  the length guard. A point left off in the middle does not stop the
  line being drawn.
- **Rejected:** drawing the pieces separately with straight lines
  between (three treatments inside one "recorded" stretch, none of them
  saying which is which); a lower length bound instead of the end rule
  (a road honestly shorter than noisy points would be refused).
- **Would change our mind:** the measurement showing a large share of
  matches lost to an unplaced end point; then extend the line to the
  recorded end point instead of refusing it.

## 2026-11-19 — A second function beside `Apply`, and one caller-facing wrapper

- **Chosen:** `route.ApplyMatches` is its own pure function; `route.Apply`
  is untouched, so its tests still pin gap routing exactly as before.
  `api.ApplyCaches` runs both and is what the page's handler and
  `roadbook journey -candidate` call, so the two cannot disagree.
- **Rejected:** widening `Apply`'s signature (every existing test and
  caller changes for a behaviour none of them is about).
- **Would change our mind:** a third cache.

## 2026-11-19 — The match cache is shared between accounts, like the route cache

- **Chosen:** `match_cache` has no owner column, following migration 11's
  decision for `route_cache`: a row can only be found by a request that
  holds the same points. Deleting an account therefore does not delete
  the road lines matched for its trips, exactly as it does not delete
  routed gaps today.
- **Rejected:** nothing yet — this follows the precedent and is put to
  the maintainer at STOP 1 as an open point, because a matched line is
  more of a trip than a routed gap between two rounded ends is.
- **Would change our mind:** the maintainer's answer; and in any case a
  router on the hosted instance (STOP 3) must not go live before it is
  settled, since until then both tables are empty there.

## 2026-11-19 — Matching is on by default in `roadbook route`

- **Chosen:** `-match` defaults to true; `-match=false` routes gaps only.
  `-match-radius-m` and `-match-min-m` are flags; the served journey
  reads answers at the default radius. Run counts for matching are new
  columns on `route_runs`.
- **Rejected:** off by default (the feature would exist only for those
  who read the flags).
- **Would change our mind:** the public demonstration server. A trip is
  tens to hundreds of matching requests where it was a handful of
  routes, and that server is the default `-router-url`. Put to the
  maintainer at STOP 1.

## 2026-11-19 — The web fixtures were regenerated, not edited

- **Chosen:** the three demo journeys under `web/lib/fixtures/` were
  served again from a scratch instance holding only the demo, by the
  recipe in that folder's README. Compared key by key with the old
  files: identical, plus `match_params`, `snapped_observed_km` and
  `snapped_road_km`.
- **Rejected:** adding the three fields by hand.
- **Would change our mind:** nothing.

## 2026-11-23 — What the three datasets make of the matcher as built

Not a choice; the measurement the checkpoint owed, recorded where the
decisions below rest on it. `measure/built_test.go.txt`, local OSRM,
India data, 100 m, every mode matched, counts only.

| at guard 1.3 | Android fixture | archive | iPhone export |
|---|---|---|---|
| stretches drawn along roads | 513 | 1,988 | 1,014 |
| share of recorded km | 78% | 83% | 75% |
| road length ÷ point-to-point | 1.08 | 1.08 | 1.10 |
| largest stretch, points | 144 | 221 | 238 |
| against Google's distances, today | +12.0% | +8.0% | −2.8% |
| against Google's distances, along roads | +17.6% | +12.4% | +0.3% |
| requests over 5 s | 69 | 252 | 129 |
| time for the whole history | 27 min | 86 min | 36 min |

The comparison with Google is without gap routing, so it shows matching
alone. On two datasets of three the road lengths move away from Google's
figures, not toward them. That is the "would change our mind" of choice
D read the other way round: it supports keeping every distance from the
recorded points.

The brief's §2.2 figures (842 matched, 82% at 1.3) were taken with a
looser rule than was built: the built rule also refuses a line whose
first or last point the matcher did not place.

## 2026-11-23 — Train, subway, ferry and flight stretches are not matched

- **Chosen:** `MATCH_SKIP_MODES` (`IN_TRAIN`, `IN_SUBWAY`, `IN_FERRY`,
  `FLYING`), served with the journey. A stretch takes the label of the
  activity that overlaps it the longest (`route.LegMode`); a stretch
  with one of these labels is not asked about, and not drawn along a
  road even when the cache holds an answer. A stretch with no label is
  matched as before.
- **Why:** the matcher has one profile, driving. At guard 1.3 the
  stretches labelled train pass it for 69% (iPhone export), 81%
  (archive) and 100% (fixture) of their kilometres, at about 1.2 times
  point to point: a railway drawn on the road beside it. On the iPhone
  export that is 3,744 km. Taking the four labels out of the table
  above leaves 75%, 80% and 62% of recorded kilometres along roads
  (by subtraction from the per-mode rows, not a new run).
- **Rejected:** leaving it to the guard (it does not separate them);
  a lower guard for these modes (no value does: train roads are 1.12 to
  1.25 times at every guard measured); a rail profile (another prepared
  dataset per region, and not in this phase).
- **Would change our mind:** a report of a road trip left point to
  point because Google called it a train; or a rail profile.

## 2026-11-23 — Matching is off against the public demonstration server

- **Chosen:** when `-match` is not given, `roadbook route` matches
  unless the router is the public demonstration server
  (`route.MatchByDefault`, by host name). It says so when it skips.
  `-match` given explicitly is obeyed either way. Supersedes the
  default in the 2026-11-19 entry.
- **Why:** a gap request carries two ends; a matching request carries
  every recorded point of a stretch, to a third party, tens to hundreds
  of times a trip, against a courtesy service.
- **Rejected:** refusing outright (a self-hoster may decide for
  themselves); off everywhere (the 2026-11-19 reason stands for one's
  own router).
- **Would change our mind:** nothing foreseen.

## 2026-11-23 — The shared cache and account deletion: before STOP 3

- **Chosen:** recorded, not built. `match_cache`, like `route_cache`,
  is not tied to an account, so deleting an account leaves the road
  lines matched for it. The hosted instance has no router and both
  tables are empty there. This is settled before a router goes live on
  it (STOP 3), not before.
- **Rejected:** building account-scoped caches now, ahead of the
  decision that would need them.
- **Would change our mind:** STOP 3 going ahead.

## 2026-11-23 — Slow matching requests are accepted for the batch

- **Chosen:** nothing built. About one request in ten takes over 5
  seconds and the slowest over a minute; a whole history takes 27 to 86
  minutes, once, after which every answer is cached. The slow ones are
  sparse stretches, a few points kilometres apart; timestamps and the
  `gaps` option were ruled out as the cause. Not explained further. The
  brief's "several minutes" for the fixture was not reproduced.
- **Rejected:** a smaller radius for sparse stretches, untested.
- **Would change our mind:** STOP 3, where the same requests would run
  on a host with less memory; measured there before anything is
  promised.

## 2026-11-23 — One constant added to the web fixtures by hand

- **Chosen:** `match_params.skip_modes` was written into the three demo
  journeys directly. It is a constant echoed from the defaults, and the
  API test pins the served value.
- **Rejected:** a scratch instance for one constant. The 2026-11-19
  entry's rule stands for anything computed.
- **Would change our mind:** any fixture difference found at STOP 2,
  when the fixtures are served again anyway.
