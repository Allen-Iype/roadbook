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
