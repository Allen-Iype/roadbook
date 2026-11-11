// Command gen rebuilds the embedded list of places,
// internal/places/osm_places_city_town.tsv.gz, from OpenStreetMap.
//
// Run from the repository root:
//
//	go run ./internal/places/gen -dir /some/dir/outside/the/repo
//
// It asks a public Overpass server for every node tagged place=city, then
// every node tagged place=town (two queries; a combined one timed out when
// this was first built — cities answered in about 3 minutes, towns in about
// 8), keeps each raw answer in -dir as city.tsv and town.tsv, and writes the
// trimmed, sorted, gzipped list. With -cached it skips the network and
// rebuilds from the two files already in -dir, which is how the committed
// file can be re-derived from the same raw answers.
//
// OpenStreetMap changes by the minute, so a fresh fetch never reproduces the
// committed bytes: the committed file is the record, and the package doc of
// internal/places states the query date, the row count and the checksum.
// From the same raw answers the output IS byte-identical — rows are sorted
// by id and the gzip header carries no name or timestamp.
//
// Data © OpenStreetMap contributors, Open Database License (ODbL) 1.0.
package main

import (
	"bufio"
	"compress/gzip"
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	overpassURL = "https://overpass-api.de/api/interpreter"
	userAgent   = "roadbook-places-gen/1 (github.com/Allen-Iype/roadbook)"
	// One row per node: id, lat, lon, name, name:en, place, population.
	queryFmt = `[out:csv(::id,::lat,::lon,name,"name:en",place,population;false;"\t")][timeout:800];node[place=%s];out;`
)

var classes = []string{"city", "town"}

type row struct {
	id          int64
	name, class string
	lat, lon    float64
	population  int64
}

func main() {
	dir := flag.String("dir", "", "working directory for the raw Overpass answers (outside the repository)")
	cached := flag.Bool("cached", false, "do not fetch; rebuild from city.tsv and town.tsv already in -dir")
	out := flag.String("out", "internal/places/osm_places_city_town.tsv.gz", "output file")
	flag.Parse()
	if *dir == "" {
		fmt.Fprintln(os.Stderr, "gen: -dir is required")
		os.Exit(2)
	}
	if err := run(*dir, *out, *cached); err != nil {
		fmt.Fprintln(os.Stderr, "gen:", err)
		os.Exit(1)
	}
}

func run(dir, out string, cached bool) error {
	var rows []row
	for _, c := range classes {
		raw := filepath.Join(dir, c+".tsv")
		if !cached {
			fmt.Printf("fetching place=%s …\n", c)
			if err := fetch(c, raw); err != nil {
				return err
			}
		}
		got, st, err := trim(raw, c)
		if err != nil {
			return err
		}
		fmt.Printf("%s: %d nodes; name:en used for %d; left out: %d unnamed, %d with no Latin-script name\n",
			c, st.nodes, st.english, st.unnamed, st.nonLatin)
		rows = append(rows, got...)
	}
	sort.Slice(rows, func(i, k int) bool { return rows[i].id < rows[k].id })

	f, err := os.Create(out)
	if err != nil {
		return err
	}
	h := sha256.New()
	gz, err := gzip.NewWriterLevel(io.MultiWriter(f, h), gzip.BestCompression)
	if err != nil {
		return err
	}
	w := bufio.NewWriter(gz)
	for _, r := range rows {
		// Five decimals is about a metre — far below what "near" means here,
		// and a fifth smaller than OpenStreetMap's seven once compressed.
		fmt.Fprintf(w, "n%d\t%s\t%s\t%d\t%.5f\t%.5f\n", r.id, r.name, r.class, r.population, r.lat, r.lon)
	}
	if err := w.Flush(); err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	fi, err := os.Stat(out)
	if err != nil {
		return err
	}
	fmt.Printf("%s: %d places, %d bytes, sha256 %x\n", out, len(rows), fi.Size(), h.Sum(nil))
	return nil
}

func fetch(class, dst string) error {
	body := url.Values{"data": {fmt.Sprintf(queryFmt, class)}}.Encode()
	req, err := http.NewRequest(http.MethodPost, overpassURL, strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", userAgent)
	resp, err := (&http.Client{Timeout: 15 * time.Minute}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("overpass: HTTP %d for place=%s", resp.StatusCode, class)
	}
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

type stats struct{ nodes, english, unnamed, nonLatin int }

// trim applies the name rule: the `name` tag when every letter in it is
// Latin script — the name on the road sign, diacritics kept, the rule
// internal/states applies to regions — else `name:en` when that is Latin;
// a node with neither is left out and counted. The page prints names in
// one script, and a name the reader cannot read names nothing for them.
func trim(path, class string) ([]row, stats, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, stats{}, err
	}
	defer f.Close()
	var out []row
	var st stats
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		c := strings.Split(sc.Text(), "\t")
		if len(c) < 7 {
			continue
		}
		id, err := strconv.ParseInt(c[0], 10, 64)
		if err != nil {
			return nil, st, fmt.Errorf("%s: bad node id %q", path, c[0])
		}
		st.nodes++
		name, en := clean(c[3]), clean(c[4])
		switch {
		case name != "" && latin(name):
		case en != "" && latin(en):
			name = en
			st.english++
		case name == "":
			st.unnamed++
			continue
		default:
			st.nonLatin++
			continue
		}
		lat, err := strconv.ParseFloat(c[1], 64)
		if err != nil {
			return nil, st, fmt.Errorf("%s: node %d: bad latitude %q", path, id, c[1])
		}
		lon, err := strconv.ParseFloat(c[2], 64)
		if err != nil {
			return nil, st, fmt.Errorf("%s: node %d: bad longitude %q", path, id, c[2])
		}
		out = append(out, row{id: id, name: name, class: class, lat: lat, lon: lon, population: population(c[6])})
	}
	return out, st, sc.Err()
}

func clean(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func latin(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) && !unicode.Is(unicode.Latin, r) {
			return false
		}
	}
	return true
}

// population reads the tag's digits ("12 442 373", "1,200") and gives 0 for
// anything else: the tag is free text, and the column only ever breaks ties.
func population(s string) int64 {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == ',' || r == '.' || r == ' ':
		default:
			return 0
		}
	}
	n, err := strconv.ParseInt(b.String(), 10, 64)
	if err != nil || n > 100_000_000 {
		return 0
	}
	return n
}
