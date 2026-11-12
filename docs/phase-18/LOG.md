# Phase 18 — log: places

What the phase does, what broke, and why each fix took the form it did.
BRIEF.md and DECISIONS.md are the other two artifacts; COMPARISON.md holds
the measurement that chose the list. This closes the set.

## What the phase does

A trip's page can now say where a day was. Before this phase nothing the
system held named a place: the export carries coordinates only, and
countries and regions were served once per trip.

- **#15 — a list of towns in the database.** Every OpenStreetMap node
  tagged as a city or a town, worldwide: 113,584 places in one embedded
  file of 2,396,071 bytes (`internal/places`), built by the command kept
  beside it (`go run ./internal/places/gen`). Migration 16 adds the
  `places` table with a nearest-place index, and `roadbook places
  -if-empty` loads it at start-up beside countries and states.
- **#16 — a stay says where it was.** Every stop is served with its
  nearest listed town, the distance, and whose position was looked up
  (`Stop.place`, `Stop.named_from`). Within 10 km the wording is "near
  X"; up to 25 km it is "N km from X"; beyond that the stop has no name.
  Both distances are named parameters and are served with the journey. A
  stop with no recorded position of its own is named from the position in
  Google's record of the overlapping visit. Journey assembly is untouched.
- **#17 — a day can say which regions it crossed.** Every leg is served
  with the regions its recorded points fall in, in order, and every stop
  with its region (`Leg.states`, `Stop.state`).
- **#18 — the trip-name suggestion works without a service.** The nearest
  listed town to a trip's destination is the default suggestion
  (`ROADBOOK_GEOCODER=places`). `none` and `nominatim` remain; only
  `nominatim` reaches the network.
- **#19 — the trip page says it.** Each day line opens with its regions,
  each stay row ends with its town (once per run of stays that share it),
  a name that rests on Google's visit record is a taught term, and the
  OpenStreetMap credit sits in "How this route was drawn", on the
  landing's how-it's-built page and in the README.

`roadbook journey -candidate N` prints every served name and region
(invariant 13).

By checkpoint:

- **STOP 0 — brief, decisions, comparison** (2e1e1cf).
- **STOP 1 — Go:** #15 (0050a12), #16 (61d6304), #17 (b5e9a77), #18
  (e409607), then the maintainer read one of their own trips through the
  command.
- **STOP 2 — web:** #19 (fe13988).

## What was measured before anything was built

The plan named Natural Earth's populated places as the list. Measured on
the maintainer's archive, three lists were compared by how many stops
have a listed place within 10 km (COMPARISON.md, scripts in `measure/`):

| list | stops within 10 km |
|---|---|
| Natural Earth populated places | 25 % |
| GeoNames cities5000 | 59 % |
| OpenStreetMap city + town | 86 % |

The measurement changed the phase: the list became OpenStreetMap's, the
Nominatim pass was dropped (the hosted instance would have sent other
people's stop positions to a third party), and the list moved from Go
memory into a table.

## What was verified

- **`make test` with the database suites at every commit.** All detection
  pins held: fixture 18/1/32, archive 66/4/238, demo 3/1/0, both journey
  goldens byte-identical.
- **The demo's names and regions are pinned** through the running API
  (`internal/api/places_journey_test.go`): 4 stops named from a recorded
  position and 11 from the visit record, and the three suggestions Höfn,
  Ísafjörður and Akureyri.
- **Private data is counted, never printed**
  (`internal/api/places_coverage_test.go`; skips when `data/` is absent).
  On the maintainer's archive:

  | | count |
  |---|---|
  | stops | 2,227 |
  | position recorded / from the visit record / none | 1,875 / 340 / 12 |
  | "near X" / "N km from X" / no town within 25 km | 1,912 / 302 / 1 |
  | legs | 9,622 |
  | legs in no region / legs crossing a border | 47 / 176 |
  | stops in no region | 23 |

  The same test runs on the Android fixture and on a tester's iPhone
  export.
- **Web:** tsc clean, lint 0 errors, vitest 145 (the demo's nine day
  lines pinned in `web/lib/day-places.test.ts`), Playwright 133 passed
  and 26 skipped across 390, 768 and 1280 against the demo.
- **Read by the maintainer:** one three-day trip of their own, 19 of 19
  stops named, the towns matching memory.

## What broke, and the shape of each fix

1. **One Overpass query for cities and towns together timed out.** The
   build command asks twice, once per tag, and merges. Cities take about
   three minutes and towns about eight.
2. **The embedded file came out larger than the measured one.** The
   measurement list had no ids and was 1.90 MB; with OpenStreetMap ids and
   seven-decimal coordinates it was 2.63 MB. Coordinates were rounded to
   five decimals (about a metre), which brought it to 2,396,071 bytes. The
   ids stayed: they are what lets a later record point at a place. The
   issue text was corrected to the real size.
3. **Two new required fields broke the web's type-check.** `Leg.states`
   and `Journey.place_params` are required in the contract, so every
   hand-built leg in the web tests and the three captured demo journeys
   stopped compiling. The fixtures were recaptured from a scratch API and
   the test literals gained `states: []`. Required was kept: an absent
   list and an empty one would otherwise mean the same thing to the page.
4. **The demo's south coast has stays with no name.** Four stops are more
   than 25 km from any city or town. They are left unnamed; the page does
   not reach for a farther town. This is the evidence for the village
   list (#13).
5. **Points just off a coast fall in no region.** The region outlines are
   simplified, so 7 of the demo's 33 legs and 4 of its 15 stops have none
   (under 1 % on the archive). A nearest-region rule was not built: it
   would also change which regions a trip's cover lists, and that is the
   maintainer's decision.
6. **A phone-width stay row ended on a dangling dot.** The town is a
   block of its own so it can wrap as a unit, and the dot before it was
   left behind at the line's end. The dot moved inside the block. That
   swallowed the space before it (leading white space inside an
   inline-block collapses), so the space sits outside the block.

## Carried out of the phase

- **Villages (#13), the coast rule and the two distances** wait for what
  testers say once names are on their pages (DECISIONS, STOP 1 review).
- **A blank stay row is ambiguous.** It means "same town as the row
  above" inside a run and "no town" outside one. Left as built; the
  alternative is a name on every row.
- **"km from home" is a straight line** to the farthest place stayed. The
  maintainer remembered the one-way road distance, which is longer. The
  wording stays for now.
- **Nights (#14)** — backlog, from the competitive study.
- **A self-run Nominatim** stays open for when there are users to justify
  it; `source` on every served name keeps a second source additive.
- **Host deploy after the merge:** app and demo stacks rebuilt (the API
  applies migration 16; start-up loads the places list), the backup
  script with the places exclusion pulled, the landing redeployed for the
  how-it's-built paragraph, checked from outside. A host `.env` that sets
  `ROADBOOK_GEOCODER=none` keeps suggestions off until the line is
  removed.
- **Scratch:** local databases `roadbook_p18_review` and
  `roadbook_p18_web`, and the review servers on 8098 and 3022, are
  removed at the close.
