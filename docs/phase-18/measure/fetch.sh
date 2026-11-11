#!/bin/sh
# Phase 18 brief measurement, step 1: fetch the candidate gazetteers into a
# directory OUTSIDE the repository and trim them to one table each.
#   sh docs/phase-18/measure/fetch.sh /some/empty/dir
# Natural Earth is pinned to a commit; GeoNames publishes a fresh dump daily,
# so its counts drift slightly from the brief's (downloaded 2026-11-11).
set -eu
D="$1"; mkdir -p "$D/ne" "$D/gn"
C=117488dc884bad03366ff727eca013e434615127
curl -sSLf -o "$D/ne/ne_10m_populated_places.geojson" \
  "https://raw.githubusercontent.com/nvkelso/natural-earth-vector/$C/geojson/ne_10m_populated_places.geojson"
for f in cities500 cities1000 cities5000 cities15000; do
  curl -sSLf -o "$D/gn/$f.zip" "https://download.geonames.org/export/dump/$f.zip"
done
python3 -I "$(dirname "$0")/trim.py" "$D"
for f in "$D"/*.tsv; do
  echo "$(basename "$f"): $(wc -l <"$f") places, $(wc -c <"$f") bytes, gzip -9n $(gzip -9nc "$f" | wc -c) bytes"
done
