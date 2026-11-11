# Phase 18 — Places: decisions

Written as decisions are made. Three lines each: what was chosen, what was
rejected, what would change our mind.

## 2026-11-11 — Stop names come from the offline list only; Nominatim stays open

- **Chosen:** this phase names stops from the embedded list alone. The
  `Namer` interface and the `source` field on a served name ship, so a
  service-backed namer can be added without rework. If Roadbook gains
  users and is going well, that namer is built against a Nominatim server
  the project runs itself. Replaces the 2026-10-29 decision
  (`docs/phase-16/DECISIONS.md`) to build both sources in this phase.
- **Rejected:** an opt-in batch pass against the public Nominatim server
  now — with a denser list embedded it adds precision inside 25 km rather
  than coverage (BRIEF §2.2), it needs a cache table, a batch command and
  rate limiting, and on the hosted instance it would send other people's
  stop coordinates to a third party while the public page says the export
  goes "to the server and nowhere else"; deciding at STOP 2 after reading
  the offline names (the brief's recommendation — the maintainer settled
  it at the gate instead).
- **Would change our mind:** real users, and "N km from X" rows reading
  badly enough on their trips to justify running a geocoder; then a
  self-run Nominatim, its own brief.

## 2026-11-11 — A stop with no position of its own is named from the visit record

- **Chosen:** when a stop has no recorded position, the located Google
  visit overlapping it longest supplies the position for the name and the
  region, and the row says so. The stop's own `Loc`, the stop list, the
  map and the golden files do not change. Evidence: 506 of 521
  position-less stops on the private datasets overlap a located visit;
  where both positions exist they are a median 0.06–0.11 km apart
  (BRIEF §2.3).
- **Rejected:** leaving those stops unnamed (the demo would name 4 of its
  15); drawing the stop marker at the visit position (changes what the
  map claims — a decision about visits in assembly, not about names).
- **Would change our mind:** names from visit positions turning out wrong
  on trips the maintainer remembers, at STOP 1.

## 2026-11-11 — Choices A and D reopened: OpenStreetMap measured as the list

- **Chosen:** before deciding which list to ship, an OpenStreetMap-derived
  list was measured on the same stops (BRIEF §2.5). It names stops
  markedly better than GeoNames at every tier, so the brief's
  recommendation (GeoNames `cities5000`, held in Go memory) is withdrawn
  pending the maintainer's decision on A and D together.
- **Rejected:** deciding A on the GeoNames figures alone.
- **Would change our mind:** not applicable — this entry records that the
  question is open.

## 2026-11-11 — The list is OpenStreetMap's cities and towns, embedded

- **Chosen:** OpenStreetMap nodes tagged `place=city` or `place=town`,
  worldwide, shipped inside Roadbook: 113,584 places, 1,902,986 bytes as
  measured for the brief. A named place lies within 10 km of 86–89 % of
  stops on the three private datasets and within 25 km of all of them
  (BRIEF §2.5, `COMPARISON.md`). Accepted with it: a second named
  exception to the 1 MB rule, ODbL (credit, and the derived list offered
  under the same terms through the committed build script and file), no
  pinned upstream, and Latin-script names only.
- **Rejected:** Natural Earth populated places, the plan of 2026-10-29
  (25–27 % within 10 km; the median overnight stop on the archive 68 km
  from a listed place); GeoNames `cities5000`, the brief's first
  recommendation (59–88 % within 10 km, a second licence and a second
  credit line, where OpenStreetMap's credit is already on every map);
  Nominatim alone (no names without a service).
- **Would change our mind:** names reading wrong on remembered trips at
  STOP 1 in a way a different class mix would fix; a licence problem with
  shipping an ODbL extract in an AGPL repository that the notice does not
  settle.

## 2026-11-11 — Villages wait until city and town names have been read

- **Chosen:** the worldwide village list is not built in this phase. The
  loader accepts a larger file in the same format from the start, so
  adding it later is a data step. Decided after STOP 1, on the
  maintainer's reading of real trips.
- **Rejected:** building it now — on India it brings 92–98 % of stops
  within 5 km, but worldwide it is an estimated 24 MB that needs the
  whole-world OSM file filtered, too large for git, and its distribution
  (a download, not a committed file) is a design question of its own.
- **Would change our mind:** "N km from X" rows at STOP 1 reading badly
  enough that the phase should not close without villages.

## 2026-11-11 — The list lives in a Postgres table, not in Go memory

- **Chosen:** a `places` table (migration 16) with a `geography` position
  and a GiST index, loaded by `roadbook places -if-empty` at startup on
  the `countries`/`states` pattern; lookups are one nearest-neighbour
  query per trip. Decided on the maintainer's question: later work that
  records something about a town, a road or a route gets a row to point
  at, questions across trips become SQL, and a village-sized list is a
  table rather than a slice. A later row that references a place keeps
  the id and the name as written, not a foreign key, because the table
  is replaced wholesale.
- **Rejected:** Go memory with a grid index (the brief's first
  recommendation — no migration, no load step, unit tests without a
  database; nothing for later data to reference). With one
  implementation behind the store, the `Namer` interface named in the
  entry above on Nominatim is not built either; the `source` field on a
  served name is what keeps a second source additive.
- **Would change our mind:** the startup load proving slow or fragile on
  the small host; then the table stays and the load moves to a one-time
  operator step.

## 2026-11-11 — Gate: STOP 0 passed

- **Chosen:** the brief stands as rewritten after the four decisions
  above (A OpenStreetMap cities and towns, B visit fallback, C offline
  only, D Postgres). Two build stops, Go then web, on a `phase-18` branch
  with a milestone and issues; the maintainer commits at each STOP.
- **Rejected:** starting code before the list comparison was finished —
  the first recommendation would have shipped the weaker list.
- **Would change our mind:** not applicable.

## 2026-11-11 — #15: the list file carries OpenStreetMap ids, and is 2.4 MB for it

- **Chosen:** each row is `id, name, class, population, latitude,
  longitude`, the id written with OpenStreetMap's type letter
  ("n60741085"), positions to five decimals (about a metre). 113,584
  rows, 2,396,071 bytes — against the 1,902,986 measured for the brief,
  whose trial list had no ids or class. The id is the table's key, and
  what a later row about a place would keep.
- **Rejected:** no ids, a serial key (smaller by about 0.4 MB, but
  reloading the list would renumber every place, and nothing could refer
  to one across loads); the bare node number (a later list with ways or
  relations would collide — their numbers overlap); seven decimals as
  OpenStreetMap gives them (2,628,644 bytes for precision no "near" can
  use).
- **Would change our mind:** nothing in view; the file is already an
  exception to the 1 MB rule at either size.

## 2026-11-11 — #15: the list is built by a Go command kept beside it

- **Chosen:** `go run ./internal/places/gen -dir <dir>` — two Overpass
  queries (cities, towns), the raw answers kept in `<dir>`, the name rule
  (Latin-script `name`, else `name:en`, else left out and counted), rows
  sorted by id, gzip with no name or timestamp. With `-cached` it rebuilds
  from raw answers already fetched, byte-identically; a fresh fetch never
  will, because OpenStreetMap moves. The package doc records the fetch
  date, the raw answers' checksums and the embedded file's, and
  `TestBundled` pins the embedded checksum and the row count.
- **Rejected:** the shell-and-Python scripts used for the brief's
  measurement (a second language for one build step, and Python's
  "is this Latin?" test differs in detail from Go's); committing the raw
  answers (6.8 MB more, for a re-derivation nobody needs to repeat
  often); filtering the whole-world file (tens of GB for 115,417 nodes).
- **Would change our mind:** the public Overpass server refusing these
  queries; then the whole-world file, filtered once.

## 2026-11-11 — #15: one statement loads the list; `-if-empty` counts before it reads

- **Chosen:** `ReplacePlaces` sends the columns as six arrays and inserts
  with `unnest` in a single statement inside the delete-and-replace
  transaction: 113,584 rows in about a second on the test database.
  `roadbook places -if-empty` counts the table before it decompresses the
  list, so every start after the first costs one query.
- **Rejected:** one INSERT per row as `ReplaceStates` does (fine for
  4,589 rows; a round trip per place here, at every fresh instance's
  first start); pgx's COPY (the position column is built by a PostGIS
  function, which COPY cannot call).
- **Would change our mind:** a village-sized list making the single
  statement's arrays too large; then the same statement in batches.

## 2026-11-11 — #16: the position kind sits on the stop, and the wording band is served

- **Chosen:** `Stop.named_from` (`recorded` | `visit`) says whose
  position was looked up, on the stop itself; `Stop.place` carries `name`,
  `distance_km`, `source` and `near`. `near` is decided in Go from the
  echoed `place_params` (10 km and 25 km), so the page, the command and
  any later surface word the same stop the same way.
- **Rejected:** the brief's shape, `place.position` — the region (#17) is
  looked up at the same position and can exist where no place is within
  the limit, so the kind cannot live inside `place`; leaving the band to
  the web (two implementations of one comparison, and the kind of logic
  the Next layer is not to hold).
- **Would change our mind:** nothing in view.

## 2026-11-11 — #16: the page and the command name stops through one function

- **Chosen:** `api.StopNames` — positions (pure), one nearest-place query,
  the naming rule (pure) — called by the journey handler and by `roadbook
  journey -candidate N`. The journey loader now also reads the visits
  overlapping the window; `Assemble` still ignores them, and the goldens
  and the three reference pins are unmoved.
- **Rejected:** a separate visits query in the handler (two loaders for
  one window); naming inside `Assemble` (it would need the database, and
  the golden contract pins its output).
- **Would change our mind:** nothing in view.

## 2026-11-11 — #16: what the first run showed

- **Chosen (recorded, not decided):** with the embedded list and default
  parameters, of stops that have a position: fixture 506 near, 67 "N km
  from", 1 unnamed of 574; archive 1,912 / 302 / 1 of 2,215 (12 more have
  no position at all); iPhone export 735 / 84 / 2 of 821 (3 with none)
  — `TestStopNamesOn…` in `internal/api`, counts only. The demo names 11
  of its 15 stops: four stops of its south-coast trip are more than 25 km
  from any OpenStreetMap city or town (`TestDemoStopsAreNamed`). The
  brief's demo row (§2.2) measured only the four stops with a recorded
  position and so missed this.
- **Rejected:** widening `max_km` to name those four — "40 km from" a
  town is not where the stop was.
- **Would change our mind:** this is evidence for the village list
  (#13), to be weighed at the STOP 1 review with the maintainer's own
  trips.

## 2026-11-11 — #17: regions ride on each leg and stop as whole objects, each once per leg

- **Chosen:** `Leg.states` is a list of `State` objects (code, name,
  country) — every region the leg's own timestamped points fall in, each
  once, in order of first appearance; always present, empty when there is
  none. `Stop.state` is the region of the position the stop was named
  from. One ungrouped containment query for all of a trip's leg points
  and stop positions (`store.StateForEachPoint`), shaped by `api.Regions`,
  which the handler and `roadbook journey -candidate N` both call.
- **Rejected:** region codes pointing into `Journey.states` (a stop named
  from a visit position can be in a region no leg point touched, so the
  code would point at nothing); the full sequence of region changes along
  a leg (positions scatter across a border that follows a river or a
  ridge, and the list would flicker A, B, A, B); regions from routed
  geometry (inference does not testify to where a person was — the
  trip-level rule, kept); deriving the trip-level list from the per-point
  answers to save a query (the cover's list is pinned as it is).
- **Would change our mind:** the day line at STOP 2 needing the order of
  crossings within one long leg; then a run-collapsed sequence with a
  minimum run length.

## 2026-11-11 — #17: what the first run showed — coasts

- **Chosen (recorded, not decided):** a leg or stop whose points fall in
  no admin-1 polygon carries no region. On the private datasets that is
  rare: 47 of 9,622 legs and 23 positioned stops on the archive, 28 of
  2,594 and 15 on the fixture, 22 of 3,872 and 15 on the iPhone export
  (`TestStopNamesOn…`, counts only). On the demo it is 7 of 33 legs and 4
  of 15 stops (`TestDemoRegionsPerLegAndStop`): Natural Earth's 1:10m
  coastline runs inside the real one in places, the demo is a coastal
  country, and its Westfjords trip — single recorded positions on the
  shore — has three legs with no region while its one stop, slightly
  inland, is in Vestfirðir. This is the same effect the phase 14 test
  noted for the trip-level list.
- **Rejected for now:** giving a point just off a coast the nearest
  region within a small named distance. It would fix the demo's day
  lines, but it also changes what a trip's cover lists unless the two
  rules are allowed to differ, and that is a decision for the maintainer,
  not a detail of this issue.
- **Would change our mind:** the demo's day lines reading wrong at STOP 2
  — then the nearest-region rule, for the cover and the day together,
  with its distance as a named parameter.

## 2026-11-11 — #18: the trip-name suggestion defaults to the places table

- **Chosen:** `suggest.Places` — the nearest listed town to the trip's
  destination, within the same 25 km limit that names a stop — is the
  default suggester (`ROADBOOK_GEOCODER=places`, in the binary and in
  compose). It asks the same `PlacesForPoints` the stops use, so a trip is
  never offered a name its own page would not print. `none` and
  `nominatim` remain; only `nominatim` reaches the network. The demo's
  three trips are suggested Höfn, Ísafjörður and Akureyri
  (`TestDemoNameSuggestionsFromPlaces`). No change to the web: the
  confirm step and bulk keep already handle a suggestion being present or
  absent.
- **Rejected:** leaving the default at `none` and making places opt-in
  (every instance already carries the list; an empty name box where a
  name is known helps nobody); composing a longer name here ("Trip to
  Höfn") — the suggestion is prefill the traveller edits, and wording
  belongs to the page.
- **Would change our mind:** bulk keep producing rows of identical names
  (several trips to one town) that testers find confusing; then the web
  adds the month to a repeated suggestion.

## 2026-11-11 — STOP 1 review: the list and the two distances stand

- **Chosen:** after the maintainer read one of their own trips through
  `roadbook journey -candidate` (19 of 19 stops named, 17 from a recorded
  position and 2 from the visit record), the town names were accepted as
  matching memory. Cities and towns, "near" within 10 km and named within
  25 km, go to the web half unchanged.
- **Rejected for now:** adding villages (#13), a nearest-region rule for
  points just off a coast, and retuning the two distances. Each waits for
  what testers say once names are on the page, not for more reading by
  the maintainer.
- **Would change our mind:** testers reporting stays with no name or a
  name they do not recognise (villages), day lines that miss a region
  they were in (coast rule), or "near" read as wrong at the far end of
  its band (the 10 km parameter).
- **Noted, outside this phase:** a trip's "km from home" is a straight
  line to the farthest place stayed, and the maintainer remembered the
  one-way road distance, which is longer. A wording question for the trip
  page, to be tracked on its own.
