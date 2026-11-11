# Phase 18 — which list of places: the comparison

Recorded 2026-11-11. This is the evidence behind the choice of list
(DECISIONS.md, same date), on one page. The brief's §2 has the same
figures with their method; the scripts are in `measure/`.

**The question.** For each stop on a trip, how far is the nearest named
place in a given list? A list is useful when that distance is small
enough for "near X" to be true.

**The stops.** Every stop with a recorded position, on the trips detection
finds with default parameters:

| dataset | trips | stops |
|---|---|---|
| maintainer's archive (Android, 8 years) | 66 | 1,875 |
| Android fixture (16 months) | 18 | 485 |
| a tester's iPhone export | 60 | 744 |
| demo (fictional, committed) | 3 | 4 |

The private datasets are in `data/`; only counts and distances were read
out of them.

## 1. The lists

| list | source and licence | places worldwide | places in India | gzipped size | built for this comparison |
|---|---|---|---|---|---|
| Natural Earth 10m populated places | Natural Earth, public domain | 7,342 | 214 | 0.14 MB | yes |
| GeoNames `cities15000` | GeoNames, CC BY 4.0 | 34,214 | 3,788 | 0.66 MB | yes |
| GeoNames `cities5000` | GeoNames, CC BY 4.0 | 70,026 | 6,595 | 1.31 MB | yes |
| GeoNames `cities1000` | GeoNames, CC BY 4.0 | 171,832 | 7,240 | 3.03 MB | yes |
| OSM cities + towns | OpenStreetMap, ODbL | 113,584 | 4,616 | 1.90 MB | yes, worldwide |
| OSM cities + towns + villages | OpenStreetMap, ODbL | about 1.77 million nodes (OSM's tag statistics) | 182,742 | 2.48 MB for India and Iceland; about 24 MB worldwide, estimated | India and Iceland only |
| OSM, seven classes (adds hamlets and parts of towns) | OpenStreetMap, ODbL | not counted | 298,409 | 4.08 MB for India and Iceland | India and Iceland only |

A GeoNames list "N" holds places with a population over N plus the seats
of administrative divisions. OSM classes are tags mappers apply
(`place=city`, `town`, `village`, …), not population cut-offs.

## 2. Worldwide lists, every stop

| dataset | list | median | p90 | within 10 km | within 25 km |
|---|---|---|---|---|---|
| archive, 1,875 | Natural Earth | 39.5 km | 70.1 km | 25 % | 42 % |
| | GeoNames `cities15000` | 7.0 km | 18.9 km | 57 % | 99 % |
| | GeoNames `cities5000` | 6.3 km | 18.6 km | 59 % | 100 % |
| | GeoNames `cities1000` | 6.3 km | 18.6 km | 59 % | 100 % |
| | **OSM cities + towns** | 5.0 km | 11.8 km | 86 % | 100 % |
| archive, overnight stops, 371 | Natural Earth | 68.0 km | 68.6 km | 14 % | 30 % |
| | GeoNames `cities5000` | 16.2 km | 16.8 km | 40 % | 100 % |
| | **OSM cities + towns** | 5.3 km | 11.8 km | 88 % | 100 % |
| Android fixture, 485 | Natural Earth | 52.5 km | 72.0 km | 27 % | 36 % |
| | GeoNames `cities5000` | 6.9 km | 19.6 km | 61 % | 100 % |
| | **OSM cities + towns** | 4.8 km | 10.7 km | 89 % | 100 % |
| iPhone export, 744 | Natural Earth | 33.7 km | 69.5 km | 27 % | 39 % |
| | GeoNames `cities15000` | 2.6 km | 14.3 km | 83 % | 96 % |
| | GeoNames `cities5000` | 2.0 km | 11.5 km | 88 % | 98 % |
| | GeoNames `cities1000` | 2.0 km | 10.1 km | 89 % | 99 % |
| | **OSM cities + towns** | 2.4 km | 10.5 km | 89 % | 100 % |
| demo, 4 | Natural Earth | 1.4 km | 4.0 km | 100 % | 100 % |
| | GeoNames `cities5000` | 2.6 km | 2.7 km | 100 % | 100 % |
| | **OSM cities + towns** | 2.4 km | 2.7 km | 100 % | 100 % |

"Overnight" is a stop that covers 02:00 on its own clock.

## 3. The denser OSM tiers, stops in India and Iceland

The village and seven-class lists were built from two country extracts,
so this table is held to stops whose nearest GeoNames place is in India
or Iceland: all of the fixture and iPhone stops, 1,836 of the archive's.

| dataset | list | median | p90 | within 5 km | within 10 km |
|---|---|---|---|---|---|
| archive, 1,836 | GeoNames `cities5000` | 6.74 km | 18.6 km | 39 % | 58 % |
| | OSM cities + towns | 5.08 km | 11.7 km | 49 % | 87 % |
| | OSM + villages | 2.28 km | 4.2 km | 93 % | 99 % |
| | OSM, seven classes | 0.61 km | 1.5 km | 100 % | 100 % |
| archive, overnight, 364 | GeoNames `cities5000` | 16.24 km | 16.8 km | 23 % | 39 % |
| | OSM cities + towns | 5.30 km | 11.4 km | 27 % | 88 % |
| | OSM + villages | 3.68 km | 4.5 km | 92 % | 100 % |
| | OSM, seven classes | 1.05 km | 1.6 km | 100 % | 100 % |
| Android fixture, 485 | GeoNames `cities5000` | 6.92 km | 19.6 km | 40 % | 61 % |
| | OSM cities + towns | 4.81 km | 10.8 km | 54 % | 88 % |
| | OSM + villages | 1.66 km | 4.1 km | 95 % | 98 % |
| | OSM, seven classes | 0.62 km | 1.7 km | 100 % | 100 % |
| iPhone export, 744 | GeoNames `cities5000` | 2.02 km | 11.5 km | 74 % | 88 % |
| | OSM cities + towns | 2.26 km | 9.6 km | 74 % | 91 % |
| | OSM + villages | 0.99 km | 3.4 km | 98 % | 100 % |
| | OSM, seven classes | 0.53 km | 1.6 km | 99 % | 100 % |

## 4. What it showed

- Natural Earth, the list the plan named, puts a place within 25 km of
  fewer than half of real stops.
- Among GeoNames lists, size barely matters on the Android data: India's
  count hardly grows below a population of 5,000. No GeoNames list knows
  the place the archive's overnight stops cluster at; it stays 16 km away
  at every size.
- OSM's cities and towns — fewer Indian places than GeoNames
  `cities5000` — put a place within 10 km of 86–89 % of stops, and move
  the archive's overnight stops from 16 km away to 5 km.
- Villages bring 92–98 % of stops within 5 km, at roughly twelve times the
  file size worldwide (estimated, not built).
- The demo cannot tell the lists apart: its four located stops are near a
  place in all of them.

## 5. What was decided

OSM cities and towns, embedded, loaded into a Postgres `places` table.
Villages are a later data step, decided after the names have been read on
real trips. Nominatim is not used. (DECISIONS.md, 2026-11-11.)

## 6. Not measured

- The worldwide size and coverage of the village list.
- Any stop outside India and Iceland against the village tiers (39 of the
  archive's stops).
- Whether the nearest place is the *right* name — the one the traveller
  would use. Distance is a proxy; STOP 1's reading of real trips is the
  test.
- Places tagged on OSM ways and relations and not as nodes (taginfo counts
  563 cities and 4,637 towns as ways; most also have a node).

## 7. Reproducing it

`measure/fetch.sh` (Natural Earth and GeoNames), `measure/osm_world.sh`
(OSM cities and towns, two Overpass queries), `measure/osm_trim.py` (the
country-extract tiers; its header has the `ogr2ogr` command), then
`measure/coverage_test.go.txt` copied into `internal/journey`, run with
`GAZ_DIR`, and removed. GeoNames and OpenStreetMap change daily, so
counts drift from those recorded here (all fetched 2026-11-11; the
country extracts are of 2026-08-04 and 2026-08-07).
