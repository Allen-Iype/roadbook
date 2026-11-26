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

## 2026-11-24 — The guard is 1.5

- **Chosen:** `MATCH_MAX_RATIO` defaults to 1.5. The maintainer compared
  the same trips of the archive at 1.3 and 1.5 on two private previews
  over one database and preferred 1.5. Over the archive's 67 trips, 80%
  of recorded km are drawn along roads at 1.3 and 84% at 1.5; the roads
  drawn are 1.08 and 1.09 times their point-to-point length. (Read from
  the served journeys after `roadbook route -all`; `roadbook journey
  -candidate N -match-max-ratio R` prints the same per trip. These are
  with train, subway, ferry and flight stretches skipped.) 29 of the 67
  trips are identical at both values.
- **Rejected:** 1.3, the brief's default. Stricter, and it left
  stretches straight that at 1.5 follow the road the maintainer took.
- **Would change our mind:** a trip where a line at 1.5 leaves the road
  travelled for a side road. Changing it costs no new requests: the
  guard is applied on reading.

## 2026-11-24 — The guard is a server setting

- **Chosen:** `roadbook serve -match-max-ratio` /
  `ROADBOOK_MATCH_MAX_RATIO`, and the same flag on `roadbook journey`.
  Built so two previews could serve one database at two values; kept
  because the value is one judged by eye, and a self-hoster may judge
  differently. The value used is served with every journey.
- **Rejected:** a per-account setting. Nothing asks for it.
- **Would change our mind:** nobody ever sets it.

## 2026-11-24 — One drawing rule on the web

- **Chosen:** `legLine` in `web/lib/geo.ts` decides the line for a leg
  and is used by the maps, the thumbnails and the exported image. A
  recorded stretch with a road is drawn along it and joined to its own
  first and last recorded positions, so the line stays continuous with
  the gaps either side. The leg's kind is not changed.
- **Rejected:** a second rule in the image export; drawing the road
  alone, which leaves breaks where the road starts away from the point.
- **Would change our mind:** the joins reading as spurs on real trips.

## 2026-11-24 — The wording behind "Observed"

- **Chosen:** the note and the how-it's-built page say: where the
  positions lie on a road the line follows it, where no road fits it
  joins them point to point, and the positions are never moved. The
  legend's own wording ("the phone recorded it") stays. `DESIGN.md` §6
  carries the dated amendment.
- **Rejected:** leaving "the line joins them exactly as recorded",
  which is no longer true.
- **Would change our mind:** a reader taking the solid line for a
  recording of the road itself.

## 2026-11-24 — Found, not acted on: flights and trains in gaps

- **Found:** on a trip made by two flights and a train, every gap was
  routed along roads. A gap is a flight only above `AIR_SPEED_MIN_KMH`
  between its ends, and a silence that includes time at an airport
  falls under it; a gap on a train is routed with the driving profile.
  The export's own mode labels are not read for gaps. This is gap
  routing as built in phase 3, not matching.
- **Open:** whether gaps should read the labels (flying: an arc; train,
  subway, ferry: left unrouted), and whether in this phase. To be
  measured before it is decided.

## 2026-11-25 — A gap the export labels a flight is a flight

- **Chosen:** in `Assemble`, a gap under the speed line is air when the
  activity overlapping it longest is `FLYING` and it is at least
  `AIR_LABEL_MIN_KM` (100) long. Additive: the speed rule is unchanged,
  and it still catches flights the labels miss. Measured (BRIEF §7): 2
  gaps on the archive and 3 on the iPhone export, 6,268 km that were
  drawn as road; the minimum is flat from 50 to 200 km. On the archive's
  67 trips, served from one database before and after: air 12,588 →
  14,135 km, and the trip that raised it goes from 144% over Google's
  ground distance to 6%. Both journey goldens and every pinned count are
  unchanged.
- **Rejected:** the label alone (five short gaps on each long file carry
  it: the ground at either end of a flight); lowering
  `AIR_SPEED_MIN_KMH` (ground transport reaches it).
- **Would change our mind:** a gap labelled flying that was not a
  flight and is 100 km or longer.

## 2026-11-25 — A gap labelled train, subway or ferry is not routed

- **Chosen:** `route.UnknownKeys` and `route.Apply` take the activities
  and the skipped modes; a gap whose label is in `MATCH_SKIP_MODES` is
  not asked about and a cached road is not drawn on it. It stays
  "unknown": a dashed straight line. On the archive: routed 41,826 →
  36,253 km, unknown 0 → 3,051 km; trips flagged for diverging from
  Google 40 → 39.
- **Rejected:** a rail profile (a second router dataset; a later piece
  of work); leaving the gap on the road, which draws a road nobody
  took.
- **Would change our mind:** a rail router being added, at which point
  these gaps have somewhere to go.

## 2026-11-25 — The parameter keeps its name

- **Chosen:** `MATCH_SKIP_MODES` and `match_params.skip_modes` now
  govern gaps as well as recorded stretches, under the same names. The
  brief's addendum proposed `ROAD_SKIP_MODES`.
- **Rejected:** the rename: it changes a served field and the recorded
  parameters of earlier runs two days after they were introduced, for a
  name.
- **Would change our mind:** the two uses needing different lists.

## 2026-11-25 — `LegMode` moved into `journey`

- **Chosen:** the longest-overlap rule lives in `internal/journey`,
  because `Assemble` now needs it and `route` imports `journey`, not
  the reverse. `route.LegMode` remains as a call through.
- **Rejected:** a second copy of the rule.

## 2026-11-25 — The road caches belong to an account

- **Chosen:** migration 19 gives `route_cache` and `match_cache` an
  owner. A row is read only by the account it was made for, two
  accounts asking the same question each keep their own answer, and
  `DeleteUserData` removes both tables' rows with the account
  (`UserDataTables`). This reverses phase 13's decision that
  `route_cache` stays global "so one user's routing warms another's",
  and the 2026-11-19 entry here that followed it for `match_cache`.
  Little is lost: a match row is keyed by a hash of the exact recorded
  points, a route row by the two ends of one gap rounded to about 11 m,
  so two accounts rarely ask the same question.
- **Rejected:** keeping the caches shared and saying so on the privacy
  page. What a person contributes on purpose for others may one day be
  kept after they leave, stated when they give it; a road derived from
  their history for their own page is not that. Also rejected:
  reference counting shared rows, which is more machinery for the same
  result.
- **Would change our mind:** nothing foreseen.
- **Open, for STOP 3:** `roadbook route` reads and writes the `self`
  account only. On an instance with accounts it needs to run for each
  account before any hosted user sees a routed gap.

## 2026-11-26 — STOP 3: the hosted router is not tried in this phase

- **Measured, on a laptop:** the India dataset is 19 GB on disk and the
  router loaded normally holds 13.2 GB; the hosted machine has 7.9 GB.
  In the router's file-mapped mode (`--mmap`) the same history, 3,938
  gaps and 2,778 stretches, was answered identically with 0.65 GB of
  the server's own memory and at most 2.7 GB of the files resident.
  Wall time for the full run was not obtained (the machine slept); two
  trips of 4 and 429 gaps took 10 and 58 seconds.
- **Chosen:** stop there. Memory is no longer the obstacle; the trial
  on the host itself (disk speed, a memory cap beside the app) is not
  run now, by the maintainer's decision. The file-mapped mode is
  documented for self-hosters. Hosted users keep straight gaps and
  point-to-point recorded lines; phase 15 §3F stays open.
- **Rejected for now:** a larger plan, a smaller extract, another
  engine (Valhalla is tiled and has map matching; not measured).
- **Still needed before a hosted router:** the trial on the host, and
  `roadbook route` running for every account, not `self` only.
- **Would change our mind:** the maintainer taking up the trial; a
  hosted user asking for road-following lines again.

