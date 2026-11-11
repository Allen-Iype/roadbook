// Package places parses the list of named settlements a stop is named from —
// the rows the places table stores — the way internal/states does for
// regions. A row is a point with a name: "Ísafjörður, town, 66.07, −23.12".
// Which row is nearest to a stop, and how far, is the store's question
// (PostGIS, a nearest-neighbour index); this package only reads the list.
//
// The embedded list is every OpenStreetMap node tagged place=city or
// place=town, worldwide (phase 18 BRIEF §4A; the lists it was chosen over
// are in docs/phase-18/COMPARISON.md). At 2,396,071 bytes it is the second
// named exception to the repository's 1 MB rule (CLAUDE.md, data safety).
//
// Provenance — weaker than internal/states, and stated as such. That file
// is byte-for-byte a file at a named upstream commit. OpenStreetMap has no
// such thing: it changes by the minute, so the committed file is the record
// and a fresh fetch will not reproduce it.
//
//	source      OpenStreetMap, via the public Overpass API (overpass-api.de)
//	queries     node[place=city]; node[place=town]   (gen/gen.go)
//	fetched     2026-11-11
//	raw         city.tsv  11,856 nodes  sha256 f076974f454440a4c474532c67ee277d120db78f75a0a8354b632dc8adcfb7e5
//	            town.tsv 103,561 nodes  sha256 b7bf604131814b0b9efdb43528e10c00b833b7ac64d86d286b0c6f10926e48d7
//	left out    725 nodes with no name; 1,108 with no Latin-script name
//	embedded    113,584 places, 2,396,071 bytes
//	sha256      5caa978dcd0705cae4c4d558c92dcd3d055d0f0733bb357013b8031bbc265151
//
// Rebuilding: `go run ./internal/places/gen -dir <dir>` fetches and builds;
// with -cached it rebuilds from raw answers already in <dir>, and from the
// same raw answers the output is byte-identical (rows sorted by id, no name
// or timestamp in the gzip header).
//
// Licence: © OpenStreetMap contributors. The data is available under the
// Open Database License (ODbL) 1.0, https://opendatacommons.org/licenses/odbl/ .
// The embedded file is a database derived from OpenStreetMap and is offered
// under the same licence; gen/gen.go is the method that produced it.
package places

import (
	"bufio"
	"bytes"
	"compress/gzip"
	_ "embed"
	"fmt"
	"io"
	"strconv"
	"strings"
)

//go:embed osm_places_city_town.tsv.gz
var bundled []byte

// SourceOSM is the source recorded on every row of the embedded list, and
// served with every name drawn from it (invariant 8's reasoning applied to
// words: a name is an inference, and says where it came from).
const SourceOSM = "osm"

// Place is one row of the list.
type Place struct {
	// ID is the row's key: the OpenStreetMap element, type letter first —
	// "n60741085" is node 60741085. The letter keeps the key unique should a
	// later list also carry ways ("w…") or relations ("r…"), whose numbers
	// overlap the nodes'.
	ID string
	// Name is the `name` tag when it is in Latin script — the name on the
	// road sign, diacritics kept ("Ísafjörður") — else `name:en`. The rule
	// is applied when the list is built (gen/gen.go), not here.
	Name string
	// Class is OpenStreetMap's place value: "city" or "town" in the
	// embedded list. Mappers assign it; it is not a population cut-off.
	Class string
	// Population is the `population` tag where it is a plain number, else 0.
	// Free text upstream; nothing depends on it being present.
	Population int64
	Lat, Lon   float64
}

// Bundled parses the embedded list.
func Bundled() ([]Place, error) {
	return Parse(bytes.NewReader(bundled))
}

// Parse reads a list in the embedded file's format, gzipped or plain
// (sniffed from the first two bytes): one place per line, six tab-separated
// fields — id, name, class, population, latitude, longitude. A larger list
// (villages) loads through the same door: `roadbook places -src <file>`.
//
// Strict on purpose. The file is reference data built by a tool, so a
// malformed line is a broken build, not something to skip quietly.
func Parse(r io.Reader) ([]Place, error) {
	br := bufio.NewReader(r)
	if magic, err := br.Peek(2); err == nil && magic[0] == 0x1f && magic[1] == 0x8b {
		gz, err := gzip.NewReader(br)
		if err != nil {
			return nil, fmt.Errorf("places: %w", err)
		}
		defer gz.Close()
		return parseTSV(gz)
	}
	return parseTSV(br)
}

func parseTSV(r io.Reader) ([]Place, error) {
	var out []Place
	seen := make(map[string]bool)
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for n := 1; sc.Scan(); n++ {
		line := sc.Text()
		if line == "" {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) != 6 {
			return nil, fmt.Errorf("places: line %d has %d fields, want 6", n, len(f))
		}
		p := Place{ID: f[0], Name: f[1], Class: f[2]}
		if p.ID == "" || p.Name == "" || p.Class == "" {
			return nil, fmt.Errorf("places: line %d has an empty id, name or class", n)
		}
		if seen[p.ID] {
			return nil, fmt.Errorf("places: id %q appears twice", p.ID)
		}
		seen[p.ID] = true
		var err error
		if p.Population, err = strconv.ParseInt(f[3], 10, 64); err != nil || p.Population < 0 {
			return nil, fmt.Errorf("places: line %d: population %q is not a number", n, f[3])
		}
		if p.Lat, err = strconv.ParseFloat(f[4], 64); err != nil || p.Lat < -90 || p.Lat > 90 {
			return nil, fmt.Errorf("places: line %d: latitude %q out of range", n, f[4])
		}
		if p.Lon, err = strconv.ParseFloat(f[5], 64); err != nil || p.Lon < -180 || p.Lon > 180 {
			return nil, fmt.Errorf("places: line %d: longitude %q out of range", n, f[5])
		}
		out = append(out, p)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("places: %w", err)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("places: the list is empty")
	}
	return out, nil
}
