"""Phase 18 brief measurement: trim OpenStreetMap place points to the same
table shape as trim.py. Input: CSVs written by

  ogr2ogr -f CSV india.csv india-latest.osm.pbf -lco GEOMETRY=AS_XY \
    -sql "SELECT name, place, other_tags FROM points WHERE place IS NOT NULL"

(Geofabrik country extracts; the brief used India of 2026-08-04 and Iceland
of 2026-08-07). Writes three tiers into the gazetteer directory:
  osm_ct.tsv    place = city | town
  osm_ctv.tsv   place = city | town | village
  osm_all.tsv   the above + hamlet | suburb | neighbourhood | quarter
The name is `name:en` when the `name` tag is not Latin script, else `name`;
a place with neither is dropped and counted."""
import csv, os, re, sys, unicodedata

csv.field_size_limit(1 << 30)
out_dir, inputs = sys.argv[1], sys.argv[2:]
CT = {"city", "town"}
CTV = CT | {"village"}
ALL = CTV | {"hamlet", "suburb", "neighbourhood", "quarter"}

def latin(s):
    return all("LATIN" in unicodedata.name(ch, "") for ch in s if ch.isalpha())

def tag(tags, k):
    m = re.search(r'"%s"=>"((?:[^"\\]|\\.)*)"' % re.escape(k), tags or "")
    return m.group(1) if m else ""

rows = {"osm_ct": [], "osm_ctv": [], "osm_all": []}
for path in inputs:
    classes, unnamed, swapped = {}, 0, 0
    for r in csv.DictReader(open(path, encoding="utf-8", newline="")):
        p = r["place"]
        if p not in ALL:
            continue
        name = r["name"]
        if not name or not latin(name):
            en = tag(r["other_tags"], "name:en")
            if en and latin(en):
                name, swapped = en, swapped + 1
            else:
                unnamed += 1
                continue
        classes[p] = classes.get(p, 0) + 1
        pop = re.sub(r"\D", "", tag(r["other_tags"], "population")) or "0"
        line = "%s\t%.5f\t%.5f\t%s\t\t\t%s\n" % (name.replace("\t", " "), float(r["Y"]), float(r["X"]), pop, p)
        rows["osm_all"].append(line)
        if p in CTV:
            rows["osm_ctv"].append(line)
        if p in CT:
            rows["osm_ct"].append(line)
    print(os.path.basename(path), dict(sorted(classes.items())), "name:en used", swapped, "dropped, no Latin name", unnamed)
for k, v in rows.items():
    open(os.path.join(out_dir, k + ".tsv"), "w", encoding="utf-8").writelines(v)
    print(k, len(v), "places")
