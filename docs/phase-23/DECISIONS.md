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
