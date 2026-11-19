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
