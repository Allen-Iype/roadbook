#!/bin/sh
# Phase 18 brief measurement: every OpenStreetMap node tagged place=city or
# place=town, worldwide, from a public Overpass server — two queries, one
# per class (a single combined query timed out on 2026-11-11; cities took
# 3 minutes, towns 8).
#   sh docs/phase-18/measure/osm_world.sh "$DIR"     # the fetch.sh directory
# Writes $DIR/osm_ct_world.tsv in the shape trim.py writes. OSM changes by
# the minute, so counts drift from the brief's.
set -eu
D="$1"; mkdir -p "$D/osmworld"
for c in city town; do
  curl -sSf -m 850 -A "roadbook-places-build/0.1 (github.com/Allen-Iype/roadbook)" \
    -o "$D/osmworld/$c.tsv" \
    --data-urlencode "data=[out:csv(::id,::lat,::lon,name,\"name:en\",place,population;false;\"\\t\")][timeout:800];node[place=$c];out;" \
    https://overpass-api.de/api/interpreter
done
python3 -I "$(dirname "$0")/osm_world_trim.py" "$D"
echo "gzip -9n: $(gzip -9nc "$D/osm_ct_world.tsv" | wc -c) bytes"
