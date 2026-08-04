# OSRM self-hosting runbook

Every step here is run by the maintainer, by hand. Nothing in this
repository downloads, builds, or serves any of it — the repo's rule (phase 3
BRIEF §4): no fetching at build, install, or run time. The product is fully
usable without any of this; the null router leaves gaps visibly unknown,
which is the designed degraded state, not an error.

OSRM answers routing queries against a preprocessed snapshot of
OpenStreetMap. The preprocessing is expensive (minutes to tens of minutes,
RAM in the gigabytes); the queries afterwards are milliseconds. That
asymmetry is why routing is a batch step: OSRM needs to run only while
`roadbook route` runs, and can then be shut down — the cache in Postgres
holds the answers permanently.

All files live under `data/osrm/` — inside the wholesale-gitignored `data/`
directory, so nothing can leak into a commit. Everything here regenerates
from the `.osm.pbf`, and the `.osm.pbf` re-downloads from its source; unlike
the Timeline exports, none of it is irreplaceable.

## 1. Get an extract

Geofabrik publishes regional OSM extracts, updated daily. The candidates in
this dataset are all within India:

```
mkdir -p data/osrm
curl -o data/osrm/india-latest.osm.pbf \
  https://download.geofabrik.de/asia/india-latest.osm.pbf
```

Roughly 1.5 GB. **Note the download date** — it names the snapshot, and
`roadbook route -dataset india-YYYYMMDD` records it with every cached
answer (BRIEF §3C: without it, a divergence puzzle later cannot say which
map it was routed against).

## 2. Preprocess (once per extract)

Two equivalent paths; the brew one matches how this project already installs
things (postgresql@18, postgis). Pipeline is MLD (partition/customize —
faster preprocessing than contraction, ideal for a single-user instance).

**Via Homebrew:**

```
brew install osrm-backend
cd data/osrm
osrm-extract -p "$(brew --prefix)/share/osrm-backend/profiles/car.lua" india-latest.osm.pbf
osrm-partition india-latest.osrm
osrm-customize india-latest.osrm
```

(If the profile path differs on your install:
`find "$(brew --prefix)" -name car.lua` — it ships with the formula.)

**Via Docker (alternative):**

```
cd data/osrm
docker run -t -v "$PWD:/data" ghcr.io/project-osrm/osrm-backend \
  osrm-extract -p /opt/car.lua /data/india-latest.osm.pbf
docker run -t -v "$PWD:/data" ghcr.io/project-osrm/osrm-backend \
  osrm-partition /data/india-latest.osrm
docker run -t -v "$PWD:/data" ghcr.io/project-osrm/osrm-backend \
  osrm-customize /data/india-latest.osrm
```

Expectations for India on a 24 GB machine: `osrm-extract` is the long, hungry
step (tens of minutes, peak RAM around 10 GB); partition and customize are
minutes. Output is a family of `india-latest.osrm.*` files totalling several
GB beside the pbf. `osrm-extract` reports "To prepare the data for routing,
run: ..." on success.

## 3. Serve, route, shut down

```
osrm-routed --algorithm mld data/osrm/india-latest.osrm
```

(or the Docker equivalent with `-p 5000:5000`). Listens on port 5000.

Two snags seen in practice: the dataset path is resolved relative to the
current directory — "Missing/Broken File" for files that exist means the
path is wrong from where you stand (use an absolute path; note also that no
bare `.osrm` file exists in modern OSRM, the name is only the prefix of the
`.osrm.*` family). And on macOS, port 5000 is held by the AirPlay Receiver
("Address already in use") — pass `-p 5001` and point `-router-url` at it. Then,
from the repository root, with the API's DATABASE_URL:

```
roadbook route -router osrm -router-url http://localhost:5000 \
  -interval 0 -dataset india-YYYYMMDD
```

`-interval 0` because politeness spacing exists for the public demo, not
your own machine. Default scope is the confirmed adventures of the latest
run; `-all` widens, `-candidate N` narrows, `-refresh` re-asks cached pairs
(the path for "I built a newer extract").

When the run reports its totals, stop `osrm-routed` (Ctrl-C). The answers
are in Postgres; the map, the API, and `roadbook journey -candidate N` read
only the cache from here on. OSRM does not need to exist again until the
next batch.

## 4. What "no_route" means here

Expect some. OSM coverage is patchy exactly where adventures happen — rural
and mountain roads — and a pair OSRM cannot connect is cached as `no_route`
and stays a dashed grey unknown gap on the map. That is designed behaviour
(BRIEF §1.3, PLAN): the road network being incomplete is information, and
the honest rendering of it is the product working, not failing. After an OSM
mapping improvement, `-refresh` retries the negatives.
