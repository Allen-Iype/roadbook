// Package countries parses Natural Earth admin-0 country polygons into the
// rows the countries table stores. A 1:110m file is embedded so `roadbook
// countries` works with no download and no arguments (BRIEF §3E: no network
// fetch at build or install time); a higher-resolution Natural Earth file can
// be supplied from disk and parses identically, because the property names
// are the same at every Natural Earth scale.
package countries

import (
	"bufio"
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
)

// Natural Earth admin-0 countries at 1:110m, verbatim from the upstream
// repository (public domain), gzipped: 209 KB committed against the 1 MB
// rule. 110m is adequate for country-level attribution of adventure-scale
// journeys; border-adjacent misattribution is possible and the UI labels the
// result as derived. Upgrading resolution is a data swap via -src, not a
// schema change.
//
//go:embed ne_110m_admin_0_countries.geojson.gz
var bundled []byte

// Country is one polygon row: an ISO-style code, a display name, and the
// feature's GeoJSON geometry verbatim (PostGIS parses it on insert).
type Country struct {
	ISOCode  string
	Name     string
	GeomJSON json.RawMessage
}

// Bundled parses the embedded 1:110m file.
func Bundled() ([]Country, error) {
	return Parse(bytes.NewReader(bundled))
}

// Parse reads a Natural Earth admin-0 GeoJSON FeatureCollection, gzipped or
// plain (sniffed from the first two bytes).
//
// The code for each country prefers ISO_A2_EH over ISO_A2: Natural Earth
// leaves ISO_A2 as "-99" for a handful of countries (France and Norway among
// them, because of their dependent territories), and ISO_A2_EH is its own
// corrected column. Territories with no ISO code at all (Northern Cyprus,
// Somaliland) fall back to Natural Earth's ADM0_A3 identifier so they remain
// attributable rather than silently dropped.
func Parse(r io.Reader) ([]Country, error) {
	br := bufio.NewReader(r)
	if magic, err := br.Peek(2); err == nil && magic[0] == 0x1f && magic[1] == 0x8b {
		gz, err := gzip.NewReader(br)
		if err != nil {
			return nil, fmt.Errorf("countries: %w", err)
		}
		defer gz.Close()
		return parseGeoJSON(gz)
	}
	return parseGeoJSON(br)
}

func parseGeoJSON(r io.Reader) ([]Country, error) {
	var fc struct {
		Type     string `json:"type"`
		Features []struct {
			Properties struct {
				Admin   string `json:"ADMIN"`
				ISOA2   string `json:"ISO_A2"`
				ISOA2EH string `json:"ISO_A2_EH"`
				ADM0A3  string `json:"ADM0_A3"`
			} `json:"properties"`
			Geometry json.RawMessage `json:"geometry"`
		} `json:"features"`
	}
	if err := json.NewDecoder(r).Decode(&fc); err != nil {
		return nil, fmt.Errorf("countries: not parseable as GeoJSON: %w", err)
	}
	if fc.Type != "FeatureCollection" || len(fc.Features) == 0 {
		return nil, fmt.Errorf("countries: not a GeoJSON FeatureCollection with features")
	}

	out := make([]Country, 0, len(fc.Features))
	seen := make(map[string]string, len(fc.Features))
	for _, f := range fc.Features {
		p := f.Properties
		code := isoCode(p.ISOA2EH, p.ISOA2, p.ADM0A3)
		if code == "" {
			return nil, fmt.Errorf("countries: feature %q has no usable code (ISO_A2_EH=%q ISO_A2=%q ADM0_A3=%q)",
				p.Admin, p.ISOA2EH, p.ISOA2, p.ADM0A3)
		}
		if p.Admin == "" {
			return nil, fmt.Errorf("countries: feature with code %q has no ADMIN name", code)
		}
		if other, dup := seen[code]; dup {
			return nil, fmt.Errorf("countries: code %q claimed by both %q and %q", code, other, p.Admin)
		}
		seen[code] = p.Admin
		out = append(out, Country{ISOCode: code, Name: p.Admin, GeomJSON: f.Geometry})
	}
	return out, nil
}

// isoCode returns the first candidate that looks like a real code: two ASCII
// letters (ISO 3166-1 alpha-2 shaped — Kosovo's XK is user-assigned, not
// ISO-official, and is kept), else Natural Earth's three-letter ADM0_A3.
// "-99" is Natural Earth's null and never passes.
func isoCode(candidates ...string) string {
	for i, c := range candidates {
		if i == len(candidates)-1 {
			if len(c) == 3 && alpha(c) {
				return c
			}
			continue
		}
		if len(c) == 2 && alpha(c) {
			return c
		}
	}
	return ""
}

func alpha(s string) bool {
	for _, r := range s {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}
