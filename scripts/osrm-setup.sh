#!/bin/sh
# osrm-setup.sh — download and preprocess an OSM extract for the optional
# routing profile (phase 5 BRIEF §3E; spec: docs/phase-3/OSRM.md).
#
# Run BY AN OPERATOR, BY HAND, from the repository root:
#
#   scripts/osrm-setup.sh europe/iceland
#
# Nothing in this repository calls it — not the Dockerfiles, not compose,
# not make, not any install path. The region is a required argument by
# design: the product makes no assumption about where you live (CLAUDE.md
# invariant 9), so it cannot pick an extract for you. Region names are
# Geofabrik paths — browse https://download.geofabrik.de to find yours;
# smaller is better (the extract must only cover where your adventures are).
#
# Everything lands in data/osrm/ (gitignored). All of it regenerates from
# the .osm.pbf, and the .osm.pbf re-downloads from Geofabrik; unlike your
# Timeline exports, none of it is irreplaceable.
#
# Preprocessing runs in the same OSRM image the compose profile serves
# with, so the toolchain and the server always match. Expect the extract
# step to be the hungry one: minutes and a few GB of RAM for a small
# country, tens of minutes and ~10 GB for something India-sized.
set -eu

if [ $# -ne 1 ]; then
  echo "usage: scripts/osrm-setup.sh <geofabrik-region>" >&2
  echo "  e.g. scripts/osrm-setup.sh europe/iceland" >&2
  echo "  regions: https://download.geofabrik.de" >&2
  exit 2
fi
region=$1

if [ ! -f go.mod ] || [ ! -d scripts ]; then
  echo "osrm-setup: run from the repository root" >&2
  exit 2
fi
if ! command -v docker >/dev/null 2>&1; then
  echo "osrm-setup: docker is required (it runs the OSRM preprocessing image)" >&2
  exit 2
fi

base=$(basename "$region")-latest
pbf="data/osrm/$base.osm.pbf"
mkdir -p data/osrm

if [ -f "$pbf" ]; then
  echo "using existing $pbf (delete it first to fetch a fresh snapshot)"
else
  echo "downloading https://download.geofabrik.de/$region-latest.osm.pbf ..."
  curl -fL -o "$pbf" "https://download.geofabrik.de/$region-latest.osm.pbf"
fi

# The dataset name records WHICH map snapshot answered every cached route
# (phase 3 BRIEF §3C): without it, a divergence puzzle later cannot say
# what it was routed against. Stamped from the pbf file's own date.
snapshot=$(date -r "$pbf" +%Y%m%d 2>/dev/null || stat -c %y "$pbf" | cut -c1-10 | tr -d -)
dataset="$(basename "$region")-$snapshot"

# MLD pipeline (extract → partition → customize), per OSRM.md §2. The
# extract step is skipped if its output family already exists.
run() {
  docker run --rm -t -v "$PWD/data/osrm:/data" ghcr.io/project-osrm/osrm-backend "$@"
}
if [ -f "data/osrm/$base.osrm.partition" ]; then
  echo "data/osrm/$base.osrm.* already preprocessed (delete the family to redo)"
else
  run osrm-extract -p /opt/car.lua "/data/$base.osm.pbf"
  run osrm-partition "/data/$base.osrm"
  run osrm-customize "/data/$base.osrm"
fi

cat <<EOF

done. dataset: $dataset

serve it (from the repository root):
  OSRM_DATA=$base docker compose --profile routing up -d osrm

route the confirmed adventures against it:
  docker compose run --rm api roadbook route -router osrm \\
    -router-url http://osrm:5000 -interval 0 -dataset $dataset

then stop it — the answers are cached in Postgres and the running
application never needs OSRM again until the next batch:
  docker compose --profile routing down
EOF
