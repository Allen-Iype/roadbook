"""Phase 18 brief measurement, step 2: trim each gazetteer to a tab-separated
table — name, lat, lon, population, country, admin-1 code, feature code.
Called by fetch.sh; prints places per file and for four sample countries."""
import json, os, sys, zipfile

g = sys.argv[1]
SAMPLE = ("IN", "IS", "US", "DE")

d = json.load(open(os.path.join(g, "ne", "ne_10m_populated_places.geojson"), encoding="utf-8"))
with open(os.path.join(g, "ne_10m_populated_places.tsv"), "w", encoding="utf-8") as out:
    cc = {}
    for ft in d["features"]:
        p = ft["properties"]
        lon, lat = ft["geometry"]["coordinates"][:2]
        out.write("%s\t%.5f\t%.5f\t%s\t%s\t\t\n" % (p.get("NAME") or "", lat, lon, p.get("POP_MAX") or 0, p.get("ISO_A2") or ""))
        cc[p.get("ISO_A2")] = cc.get(p.get("ISO_A2"), 0) + 1
    print("ne_10m_populated_places", len(d["features"]), {k: cc.get(k, 0) for k in SAMPLE})

for f in ("cities500", "cities1000", "cities5000", "cities15000"):
    raw = zipfile.ZipFile(os.path.join(g, "gn", f + ".zip")).read(f + ".txt").decode("utf-8")
    with open(os.path.join(g, f + ".tsv"), "w", encoding="utf-8") as out:
        n, cc = 0, {}
        for line in raw.split("\n"):
            if not line:
                continue
            c = line.split("\t")
            out.write("\t".join((c[1], c[4], c[5], c[14], c[8], c[10], c[7])) + "\n")
            n += 1
            cc[c[8]] = cc.get(c[8], 0) + 1
        print(f, n, {k: cc.get(k, 0) for k in SAMPLE})
