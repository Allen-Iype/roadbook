"""Phase 18 brief measurement: trim the Overpass city and town output
(osm_world.sh) to name, lat, lon, population, -, -, class. The name is
`name` when it is Latin script, else `name:en` when that is; a place with
neither is dropped and counted."""
import os, re, sys, unicodedata

d = sys.argv[1]

def latin(s):
    return all("LATIN" in unicodedata.name(ch, "") for ch in s if ch.isalpha())

kept, stats = [], {}
for c in ("city", "town"):
    n = en = unnamed = nonlatin = 0
    for line in open(os.path.join(d, "osmworld", c + ".tsv"), encoding="utf-8"):
        f = line.rstrip("\n").split("\t")
        if len(f) < 7:
            continue
        n += 1
        _, lat, lon, name, name_en, place, pop = f[:7]
        if not name or not latin(name):
            if name_en and latin(name_en):
                name, en = name_en, en + 1
            elif not name:
                unnamed += 1
                continue
            else:
                nonlatin += 1
                continue
        pop = re.sub(r"\D", "", pop) or "0"
        kept.append("%s\t%.5f\t%.5f\t%s\t\t\t%s\n" % (name, float(lat), float(lon), pop, place))
    print("%s: %d nodes; name:en used %d; dropped: %d unnamed, %d with no Latin-script name" % (c, n, en, unnamed, nonlatin))
open(os.path.join(d, "osm_ct_world.tsv"), "w", encoding="utf-8").writelines(kept)
print("osm_ct_world.tsv: %d places" % len(kept))
