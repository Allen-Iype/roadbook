# Phase 18 — Places: what a day can say — brief

Status: written 2026-11-11; **choices A–D decided the same day (§4); gate
passed. No code exists.** Sections 3–6 were rewritten after the decisions;
§2 is the evidence as measured, including the lists that were not chosen.

The phase 16 review found the day narrative honest but empty: "1 stop ·
Stayed 14 h" names nowhere, because nothing Roadbook holds names a place.
This phase lets a day say which regions it crossed and lets a stay say
where it was.

The measurements below changed the plan written on 2026-10-29
(`docs/PLAN.md`, phase 18) in four ways:

1. **Natural Earth's populated places are too coarse to name stops.** On
   the maintainer's archive the median stop is 39.5 km from the nearest
   Natural Earth place and the median overnight stop 68.0 km. This is the
   condition phase 16's decision named as the one that would change its
   mind (`docs/phase-16/DECISIONS.md`, 2026-10-29).
2. **OpenStreetMap's cities and towns name stops best** of the lists small
   enough to ship: a named place within 10 km of 86–89 % of stops, against
   59–88 % for GeoNames and 25–27 % for Natural Earth (§2.5).
3. **Stops without a recorded position can be named from Google's visit
   record:** 15–19 % of stops have no position, and 97 % of those overlap
   a visit that carries one (§2.3).
4. **The list lives in Postgres, not in Go memory**, so that later work —
   a larger list with villages, or anything a traveller records about a
   town — has a row to point at (§4D).

The whole comparison is also kept on one page: `COMPARISON.md`.

## 1. Concepts this phase introduces

### Gazetteer, and reverse geocoding against one

**What.** A gazetteer is a table of named places with a coordinate each:
"Ísafjörður, 66.075, −23.124, population 2,534". Reverse geocoding is the
question asked the other way round from a search box: given a coordinate,
what is it called? Against a gazetteer the answer is the nearest row and
the distance to it. Nothing here knows a town's outline, so the honest
sentence is never "in Ísafjörður" — it is "near Ísafjörður", or "16 km
from Ísafjörður".

**How it differs from what Roadbook already does.** Countries and regions
(phases 2 and 14) are *polygons*: a point is inside Vestfirðir or it is
not, and PostGIS answers with `ST_Contains` (`internal/store/states.go`).
Towns in a gazetteer are *points*: the question is "which is nearest, and
how far", not containment.

**The alternative kind of geocoder.** A service such as Nominatim holds
the whole of OpenStreetMap and answers with the village, hamlet or
neighbourhood the point actually lies in. It is far more precise, and it
is a network service: every lookup sends a coordinate to someone else's
machine. Roadbook already calls it in one place — the name suggestion on
the confirm step (`internal/suggest/nominatim.go`, opt-in through
`ROADBOOK_GEOCODER`, one request per click).

### Why the source of a name is recorded

A name printed beside a stay is an inference, like a routed line: it comes
from a table, a distance rule, and sometimes from a position Google
supplied rather than one the phone traced. Invariant 8 applies to the map;
the same reasoning applies here. Every served name carries how far away
the named place is, which table or service named it, and which position
was looked up. The web decides the wording from those facts; it never
receives a bare string.

### A stop's position, and the visit record

**What a stop is today.** `journey.findStops`
(`internal/journey/journey.go:508`) reports a stop wherever two consecutive
Google *activities* leave a pause of five minutes or more
(`MinStopDwellSeconds`). Its position is the mean of the path points
recorded inside the pause. When no point fell inside it, the stop has a
time span and no position, and the trip page prints "position not
recorded".

**What a visit is.** Google's export also carries *visits*: a time span
and one coordinate each (`domain.Visit`). Detection uses them (home bases,
the destination dwell). Journey assembly does not read them at all.

**Why this matters now.** A stop without a position cannot be named. The
visit record overlapping it usually has the position (§2.3). The phase
uses it for naming only — the stop's own `Loc`, the map and the golden
files are unchanged — and says so on the row.

### Nearest-neighbour search in PostGIS

**What.** The list becomes a table, `places`, one row per town with its
position. "Which row is nearest to this point?" is a *nearest-neighbour*
(KNN) query: `ORDER BY position <-> point LIMIT 1`. The `<->` operator
means "distance between", and with a GiST index on the column — the same
kind of index `states` and `countries` already have — PostGIS walks the
index outward from the point and stops at the first row, without measuring
all 113,584.

**Geography, not geometry.** PostGIS has two position types. `geometry`
treats latitude and longitude as flat x and y, so "nearest" is nearest in
degrees — wrong away from the equator, where a degree of longitude is
shorter than a degree of latitude (45 km against 111 km at Ísafjörður).
`geography` measures on the globe, in metres. The `places` column is
`geography`, so the nearest row is the nearest town on the ground and the
distance comes back in metres from the same query. The existing tables
use `geometry` because containment does not have this problem.

**One query per trip.** All of a trip's stop positions go in as two
arrays, and a `LATERAL` join runs the nearest-row search once per
position — the shape `StatesForPoints` already uses for regions, with the
join condition swapped for the nearest-row search.

**Rejected:** holding the list in Go memory with a grid index (the first
draft of this brief: no migration and no load step, but nothing for later
data to reference and no SQL over places — §4D); nearest on `geometry`
(wrong at high latitudes).

## 2. What was measured

Every figure in this section comes from `docs/phase-18/measure/`
(`fetch.sh` downloads and trims the candidate tables into a directory
outside the repository; `coverage_test.go.txt` is copied into
`internal/journey`, run, and removed — its header has the three commands).
It prints counts and distance percentiles only, never a coordinate or a
name. The private rows need the files in `data/`; the demo row runs on a
clean clone. GeoNames republishes daily, so its counts will drift from
these (downloaded 2026-11-11).

### 2.1 The candidate tables

| table | licence | places | India | Iceland | trimmed, `gzip -9n` |
|---|---|---|---|---|---|
| Natural Earth 10m populated places | public domain | 7,342 | 214 | 9 | 142,566 bytes |
| GeoNames `cities15000` | CC BY 4.0 | 34,214 | 3,788 | 6 | 664,949 bytes |
| GeoNames `cities5000` | CC BY 4.0 | 70,026 | 6,595 | 14 | 1,312,748 bytes |
| GeoNames `cities1000` | CC BY 4.0 | 171,832 | 7,240 | 35 | 3,026,895 bytes |
| GeoNames `cities500` | CC BY 4.0 | 236,997 | 7,324 | 50 | 4,030,633 bytes |

"Trimmed" is seven columns (name, latitude, longitude, population,
country, admin-1 code, feature code). The upstream GeoNames zips carry
every alternate spelling and are 3.4 MB (`cities15000`) to 13.9 MB
(`cities500`). A GeoNames table "N" holds places with a population over N
plus the seats of administrative divisions.

### 2.2 Distance from a stop to the nearest named place

Stops with a recorded position, on the trips detection finds with default
parameters. "Overnight" is a stop that covers 02:00 on its own clock.

| dataset | stops | table | median | p90 | within 10 km | within 25 km |
|---|---|---|---|---|---|---|
| maintainer's archive (66 trips) | 1,875 | Natural Earth | 39.5 km | 70.1 km | 25 % | 42 % |
| | | `cities15000` | 7.0 km | 18.9 km | 57 % | 99 % |
| | | `cities5000` | 6.3 km | 18.6 km | 59 % | 100 % |
| | | `cities1000` | 6.3 km | 18.6 km | 59 % | 100 % |
| — overnight only | 371 | Natural Earth | 68.0 km | 68.6 km | 14 % | 30 % |
| | | `cities5000` | 16.2 km | 16.8 km | 40 % | 100 % |
| Android fixture (18 trips) | 485 | Natural Earth | 52.5 km | 72.0 km | 27 % | 36 % |
| | | `cities5000` | 6.9 km | 19.6 km | 61 % | 100 % |
| tester's iPhone export (60 trips) | 744 | Natural Earth | 33.7 km | 69.5 km | 27 % | 39 % |
| | | `cities15000` | 2.6 km | 14.3 km | 83 % | 96 % |
| | | `cities5000` | 2.0 km | 11.5 km | 88 % | 98 % |
| | | `cities1000` | 2.0 km | 10.1 km | 89 % | 99 % |
| demo (3 trips) | 4 | Natural Earth | 1.4 km | 4.0 km | 100 % | 100 % |
| | | `cities5000` | 2.6 km | 2.7 km | 100 % | 100 % |

What the table says:

- Natural Earth names fewer than half of real stops within 25 km. "Near a
  city 68 km away" is not a name for where someone slept.
- Any GeoNames table names nearly every stop within 25 km. Between
  `cities5000` and `cities1000` the Android figures are identical — India's
  count barely grows below 5,000 (6,595 → 7,240) — and the iPhone figures
  differ by one to two points.
- The archive's overnight stops sit 16 km from the nearest GeoNames place
  at *every* table size. No offline table knows the village; only a
  service does. Whatever is embedded, there is a band of stops for which
  the honest words are "16 km from X", not "near X".
- The demo's four located stops are close to a place in every table, so
  the demo cannot tell the tables apart.

### 2.3 Stops without a position, and visits

| dataset | stops | without a position | of those, overlapping a located visit |
|---|---|---|---|
| archive | 2,227 | 352 | 340 |
| Android fixture | 574 | 89 | 89 |
| iPhone export | 824 | 80 | 77 |
| demo | 15 | 11 | 11 |

Where a stop has both its own position and an overlapping visit, the two
agree: median 0.06–0.11 km apart, within 1 km for 87–94 % of pairs (3,083
pairs across the private datasets; p90 0.75 km on Android, 2.75 km on
iPhone).

The demo is the sharp case: 11 of its 15 stops have no position. Without
the visit fallback the public demo would show names on four rows.

### 2.4 How many names a trip carries

Counting each stop that can be named (own position or visit), a trip has a
median of 10–29 and up to 212. Consecutive stops that resolve to the same
name (a week at one house is dozens of stops) collapse to a median of
6–12 runs per trip, up to 86. A name on every stay row would repeat one
town many times a day; §3 states the rule.

### 2.5 OpenStreetMap as the list (added 2026-11-11, at the maintainer's ask)

OpenStreetMap tags settlements by class: `place=city`, `town`, `village`,
`hamlet`, and parts of towns (`suburb`, `neighbourhood`, `quarter`). There
is no ready-made worldwide file of them; they are filtered out of an
extract. This measurement used the two country extracts already on the
maintainer's machine for routing (India of 2026-08-04, 1.7 GB; Iceland of
2026-08-07), filtered with `ogr2ogr` and `measure/osm_trim.py`. Places
whose name is not in Latin script use `name:en`; 85,593 Indian entries
with neither were dropped.

Because the OSM lists cover two countries, the comparison is held to
stops whose nearest GeoNames place is in India or Iceland: all of the
fixture, iPhone and demo stops, 1,836 of the archive's 1,875.

| dataset | list | places (India) | median | p90 | within 5 km | within 10 km |
|---|---|---|---|---|---|---|
| archive, 1,836 stops | GeoNames `cities5000` | 6,595 | 6.74 km | 18.6 km | 39 % | 58 % |
| | OSM city + town | 4,616 | 5.08 km | 11.7 km | 49 % | 87 % |
| | OSM city + town + village | 182,742 | 2.28 km | 4.2 km | 93 % | 99 % |
| | OSM, all seven classes | 298,409 | 0.61 km | 1.5 km | 100 % | 100 % |
| — overnight, 364 | GeoNames `cities5000` | | 16.24 km | 16.8 km | 23 % | 39 % |
| | OSM city + town | | 5.30 km | 11.4 km | 27 % | 88 % |
| | OSM city + town + village | | 3.68 km | 4.5 km | 92 % | 100 % |
| | OSM, all seven classes | | 1.05 km | 1.6 km | 100 % | 100 % |
| Android fixture, 485 | GeoNames `cities5000` | | 6.92 km | 19.6 km | 40 % | 61 % |
| | OSM city + town | | 4.81 km | 10.8 km | 54 % | 88 % |
| | OSM city + town + village | | 1.66 km | 4.1 km | 95 % | 98 % |
| iPhone export, 744 | GeoNames `cities5000` | | 2.02 km | 11.5 km | 74 % | 88 % |
| | OSM city + town | | 2.26 km | 9.6 km | 74 % | 91 % |
| | OSM city + town + village | | 0.99 km | 3.4 km | 98 % | 100 % |

Size, trimmed and gzipped, India and Iceland together: city + town 74,428
bytes (4,658 places); with villages 2,481,418 bytes (182,823); all seven
classes 4,080,800 bytes (298,681).

**The worldwide cities-and-towns list was then built** (`measure/
osm_world.sh`: two queries to a public Overpass server on 2026-11-11, 11
minutes together). 11,856 city nodes and 103,561 town nodes; 725 had no
name and 1,108 no Latin-script name or `name:en`, leaving **113,584
places, 1,902,986 bytes** trimmed and gzipped. Against every located stop,
with no country restriction:

| dataset | list | median | p90 | within 10 km | within 25 km |
|---|---|---|---|---|---|
| archive, 1,875 stops | GeoNames `cities5000` | 6.3 km | 18.6 km | 59 % | 100 % |
| | OSM cities + towns, world | 5.0 km | 11.8 km | 86 % | 100 % |
| — overnight, 371 | GeoNames `cities5000` | 16.2 km | 16.8 km | 40 % | 100 % |
| | OSM cities + towns, world | 5.3 km | 11.8 km | 88 % | 100 % |
| Android fixture, 485 | GeoNames `cities5000` | 6.9 km | 19.6 km | 61 % | 100 % |
| | OSM cities + towns, world | 4.8 km | 10.7 km | 89 % | 100 % |
| iPhone export, 744 | GeoNames `cities5000` | 2.0 km | 11.5 km | 88 % | 98 % |
| | OSM cities + towns, world | 2.4 km | 10.5 km | 89 % | 100 % |
| demo, 4 | GeoNames `cities5000` | 2.6 km | 2.7 km | 100 % | 100 % |
| | OSM cities + towns, world | 2.4 km | 2.7 km | 100 % | 100 % |

The worldwide list **with villages was not built or measured**.
OpenStreetMap's tag statistics (taginfo, 2026-11-11) count 1,655,066
village nodes and 2,002,327 hamlet nodes; scaling India's bytes per place
gives an *estimate* of about 24 MB with villages. Building it means
filtering the whole-world file (the "planet", tens of GB) once.

What this says:

- OSM's cities and towns alone — fewer places in India than GeoNames
  lists — name stops better than GeoNames: 87–91 % within 10 km against
  58–88 %, and the archive's overnight stops move from 16 km away to 5 km.
- With villages, 92–98 % of stops have a named place within 5 km. This is
  the tier that removes the "16 km from X" rows.
- The licence is ODbL: credit, and a derived list must be offered under
  the same terms (a committed build script and a published file meet
  that). The credit line "© OpenStreetMap contributors" is already on
  every Roadbook map, because the map tiles are OSM data.

Choices A and D were reopened on this evidence and decided the same day
(§4, `DECISIONS.md`).

## 3. What gets built

One migration, one command, no network call at run time.

### Go

- **`internal/places`** (new): the embedded list — OpenStreetMap city and
  town nodes worldwide, 113,584 rows, 2,396,071 bytes gzipped as built
  (the 1.90 MB measured in §2.5 carried no ids) — and
  its parser, on the pattern of `internal/states`. Columns: OSM node id,
  name, class (`city` | `town`), population where tagged, latitude,
  longitude. The name is the `name` tag when it is Latin script, else
  `name:en`; a node with neither is left out. The build script is
  committed beside it (the two Overpass queries and the trim), and the
  package doc records the query date, the row count and the file's
  sha256. OpenStreetMap changes by the minute, so re-running the script
  does not reproduce the bytes: the committed file is the record. This is
  weaker than `internal/states`, which is byte-for-byte a file at a named
  upstream commit, and the package doc says so.
- **`migrations/00016_places.sql`**: `places (id bigint primary key, name,
  class, population, source, geog geography(Point, 4326))` with a GiST
  index. Reference data like `countries` and `states`: loaded wholesale,
  never accumulated, the same for every account.
- **`roadbook places [-src file] [-if-empty]`**: loads the embedded list,
  or a file in the same format. Compose's startup line gains `roadbook
  places -if-empty` beside the `countries` and `states` steps. The
  `-src` form is what a later, larger list (villages) uses — a data step,
  no code change.
- **`store.PlacesForPoints`**: the nearest place and its distance for each
  position, one query (§1). An empty table answers with nothing, and the
  page shows no names — the same visible degradation as regions before
  `roadbook states` has run.
- **`journey.StopPositions`** (new, pure, outside `Assemble`, so the golden
  files and the reference pins do not move): for each stop, the position
  to look up — its own when it has one, else that of the located visit
  overlapping it longest — and which of the two it is. Testable without a
  database.
- **Names and regions in the handler**, beside the existing countries and
  states step (`internal/api/server.go`): each stop gets its nearest place
  when that lies within `PlaceMaxKm`, and its region; each leg gets the
  ordered list of regions its points cross (a second, ungrouped form of
  `StatesForPoints`). Named parameters, echoed in the response
  (invariant 3): `PlaceNearKm` 10, `PlaceMaxKm` 25 — starting values to be
  read against real trips at STOP 1.
- **`api/openapi.yaml`**: `Stop` gains optional `named_from` (`recorded` |
  `visit`), optional `place` — `name`, `distance_km`, `source` (`osm`),
  `near` — and optional `state`; `Leg` gains `states`; the journey echoes
  the two parameters as `place_params` (as built, #16: the position kind
  moved onto the stop and the wording band is served — DECISIONS). Go and TypeScript regenerated. The shared view carries the
  same `Journey` schema, so it inherits.
- **The name suggestion** (confirm step): the default suggester changes
  from "none" to the places table, through the existing `internal/suggest`
  seam, so every instance prefills a trip's name with no service. No UI
  change.
- **CLI**: `roadbook journey -candidate N` prints, per stop, the name,
  distance, source and position kind, and per leg its regions
  (invariant 13).
- **Backup and deletion.** The host backup excludes the table's data
  beside `countries` and `states` (`scripts/pilot/host/backup-stack.sh`);
  restore reloads it at startup. Delete-my-data does not touch it: it
  holds nothing of anyone's.
- **Tests** (`make test`, database suites): the loader's row count and
  checksum; nearest-place cases on fictional coordinates, one at high
  latitude, one in mid-ocean (nothing within the limit); the demo's stop
  names and day regions pinned; `Assemble` output byte-identical before
  and after; a private test in the pattern of `needles_test.go` giving
  the share of stops named, per band, per private dataset — the
  reproducible source for the figures the LOG will quote.

Not built: a `Namer` interface. The first draft proposed one as the seam
for a later service. With one implementation, behind the store like
regions, an interface would have nothing to choose between; the `source`
field on every served name is what keeps a second source additive.

### Web

- **The day line** gains the regions phrase in the slot phase 16 left
  (`web/components/adventure/day-narrative.tsx`): the regions of the
  day's legs and stops, in order, de-duplicated — one region is just its
  name; several read as a sequence. A pure helper in `lib/` with vitest
  cases; exact wording settled at STOP 2 on real pages.
- **The stay row** gains the name after the duration: "Stayed 14 h · near
  Ísafjörður" within `PlaceNearKm`; "Stayed 14 h · 16 km from
  Ísafjörður" between that and `PlaceMaxKm`; nothing beyond. A stay named
  from a visit position keeps its hollow marker and swaps "position not
  recorded" for a taught term (the existing `Term` control) saying the
  place comes from Google's visit record.
- **Repetition.** Within a day, a name prints on the first stay of a run
  of consecutive stays that share it (§2.4).
- **Credit.** "Place names © OpenStreetMap contributors" in the page-foot
  disclosure "How this route was drawn", on the landing's how-it's-built
  page, and in the README beside the Natural Earth credit. The map itself
  already carries the OpenStreetMap credit for its tiles.
- Playwright on the demo at 390/768/1280: a stay row shows a name; the
  day line shows a region; no horizontal scroll.

## 4. The choices, as decided (2026-11-11)

Each is recorded in `DECISIONS.md` with what was rejected and what would
change it.

### A. Which list — OpenStreetMap cities and towns, embedded

| list | licence | places | size | named place within 10 km (archive) |
|---|---|---|---|---|
| Natural Earth (the plan) | public domain | 7,342 | 0.14 MB | 25 % |
| GeoNames `cities5000` (first recommendation) | CC BY 4.0 | 70,026 | 1.31 MB | 59 % |
| **OSM cities + towns (chosen)** | ODbL | 113,584 | 1.90 MB | 86 % |
| OSM with villages (later) | ODbL | about 1.77 million | about 24 MB, estimated | 99 % on India; not built worldwide |

Costs accepted:

- **A second named exception to CLAUDE.md's 1 MB rule**, by the phase 14
  procedure (the maintainer's edit, when the file exists at STOP 1).
- **ODbL.** Credit wherever names show, and the derived list offered under
  the same terms: the committed build script and the committed file meet
  that. A notice goes in the package doc and the README.
- **No pinned upstream** (§3).
- **Latin-script names only**: 1,108 of 115,417 nodes are left out for
  having neither a Latin `name` nor `name:en`.

**Villages are not in this phase.** The loader takes a larger file from
the start. Whether to build the worldwide village list (the whole-world
OSM file, filtered once) is decided after the maintainer has read city
and town names on remembered trips at STOP 1.

### B. Stops without a position — named from the visit record

Yes, for the name and the region only. 506 of 521 position-less stops
across the three private datasets overlap a located visit, and where both
positions exist they agree to about 100 m (§2.3). PLAN.md excluded
"visits as stops … unless the brief takes them on evidence"; this is that
evidence, and the scope taken is the narrowest: the stop list, the stop's
own `Loc`, the map and the goldens do not change. The row says which kind
of position was named.

### C. Nominatim — not in this phase

Offline list only. The service stays open: if Roadbook gains users and is
going well, a geocoder is added against a Nominatim server the project
runs itself, so no coordinate leaves the operator's machines. The public
server is not used for stops — on the hosted instance that would send
other people's stop coordinates to a third party, while the public page
says "Your export streams to the server and nowhere else"
(`landing/how-its-built.html`). This replaces the 2026-10-29 decision to
build both sources in this phase.

### D. Where the list lives — a `places` table in Postgres

The first draft recommended Go memory: no migration, no load step, plain
unit tests. The maintainer's question decided it the other way: if
travellers later record something about a town, a road or a route, will
the list being in Postgres help? It will.

- What a traveller records is their data and lives in Postgres either
  way; with the list beside it, their row can point at a place row.
- "Every trip that stopped here" and "the towns stopped in most" become
  SQL over tables that already hold the trips.
- A later list with villages is millions of rows — a table with an index,
  not a slice loaded at every start.

Cost accepted: a migration and a startup load step, on the pattern
`countries` and `states` already follow. One caution recorded for that
later work: the table is replaced wholesale when a new list is loaded, so
a traveller's row should keep the place's id *and* its name as written at
the time, not a foreign key — the note `states` already carries about
`country_code`.

## 5. What is not in this phase

- Google's `placeId` (paid API, terms) — unchanged from PLAN.md.
- Any network call from `serve`; Nominatim (§4C).
- The village list (§4A).
- Anything a traveller records about a town, road or route. §4D makes
  room for it; it is not designed here.
- Nights ("3 nights away: 2 near A") and home-from-nights (competitive
  study P2). Stop names are what that sentence needs; counting nights is
  its own pass with its own parameters, and it touches detection's score.
  Proposed as the next Go item after this phase, issue to be filed.
- Town labels on the map and the downloaded image (study P10). It needs
  these names first and a label-collision policy of its own.
- Named destinations on the review cards and the trips shelf. A separate
  small change once the names have been read on the trip page.
- Visits as stops in assembly; "farthest that day"; instantaneous speed.
- Names in other scripts or languages.

## 6. Checkpoints

- **STOP 0 — this brief.** Passed 2026-11-11 with A–D decided. Branch
  `phase-18`; milestone "Phase 18": #15 the list, table and loader, #16
  stop names, #17 regions per leg and stop, #18 the offline name
  suggestion, #19 the web. Backlog: #13 villages, #14 nights.
- **STOP 1 — Go.** The list and its build script, migration 16, the
  loader, the lookup, stop positions, regions per leg and stop, the
  contract, the CLI print, the suggester default. `make test` (goldens
  byte-identical; fixture 18/1/32, archive 66/4/238, demo 3/1/0 unmoved;
  migration up and down). Review: the maintainer reads `roadbook journey
  -candidate N` for remembered trips and judges the names, the two
  distance bands, and whether villages are wanted. The 1 MB exception is
  written into CLAUDE.md here. No web change yet.
- **STOP 2 — web.** Day line, stay row, credit, tests, on the scratch
  stack with the demo; phone review.
- **Close.** LOG.md; the README and how-it's-built credits; PLAN.md's
  phase 18 text corrected to what was built; PR; host rebuild (migration
  16 applies and the list loads at startup; the backup script's exclusion
  goes out with it).
