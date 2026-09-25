// Package states parses Natural Earth admin-1 polygons — states, provinces,
// regions, départements: each country's first subdivision — into the rows
// the states table stores, the way internal/countries does for admin-0.
// One scale exists worldwide, 1:10m (the 1:50m file covers nine countries
// and the 1:110m file the United States alone — invariant 9 rules both out),
// so that file is embedded verbatim and gzipped: 12,210,503 bytes, the one
// named exception to the repository's 1 MB rule (CLAUDE.md, data safety;
// phase 14 BRIEF §3B).
//
// Provenance — the embedded file is byte-for-byte upstream:
//
//	repository  github.com/nvkelso/natural-earth-vector (public domain)
//	path        geojson/ne_10m_admin_1_states_provinces.geojson
//	commit      117488dc884bad03366ff727eca013e434615127 ("v5.1.0 packaging")
//	sha256      22d0e3ad85eb3e27f17cabf8ba2d50e554fbc27a87796ff891d958185da62fb5 (raw, 40,726,851 bytes)
//	gzip -9n    7549c975a3db831fd0b10ad95f6194e44fe92be2a0cb82d2877a403da26642e5 (12,210,503 bytes)
//
// Reproduction: download the file at that commit, `gzip -9n` it (no name or
// timestamp in the header, so the archive is reproducible), compare.
package states

import (
	"bufio"
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
)

//go:embed ne_10m_admin_1_states_provinces.geojson.gz
var bundled []byte

// State is one admin-1 polygon row.
type State struct {
	// Code is Natural Earth's adm1_code, unique across the file (the
	// ISO 3166-2 column is not: eight codes repeat). It is the row's key.
	Code string
	// ISO3166_2 is the subdivision's ISO 3166-2 code where Natural Earth
	// carries one ("IS-4", "IN-KL"); shown nowhere today, kept for the day
	// a consumer wants a standard code.
	ISO3166_2 string
	// Name is the local Latin-script name with its diacritics — "Vestfirðir",
	// not the file's English "Westfjords"; "Suðurland", not "Southern". The
	// name a traveller saw on the road sign is the name the plate prints.
	// Natural Earth's `name` column is Latin-script for every feature (the
	// native-script names live in per-language columns), so no font issue
	// follows. `name_en` is the fallback for a feature whose `name` is
	// empty — in practice none: the seven features with no `name` have no
	// `name_en` either and are dropped (see Parse).
	Name string
	// CountryCode aligns with the countries table's iso_code: the feature's
	// two-letter iso_a2 when it is one, else Natural Earth's ADM0_A3 — the
	// same fallback the countries loader applies, so Northern Cyprus's
	// regions carry CYN exactly as its country row does.
	CountryCode string
	GeomJSON    json.RawMessage
}

// Bundled parses the embedded 1:10m file.
func Bundled() ([]State, error) {
	return Parse(bytes.NewReader(bundled))
}

// Parse reads a Natural Earth admin-1 GeoJSON FeatureCollection, gzipped or
// plain (sniffed from the first two bytes). Features without any name are
// dropped rather than loaded: the upstream file carries seven placeholder
// features (adm1_code "ATA+99?", "RUS+99?", …) with null `name` and
// `name_en`; a polygon nobody can be told they crossed attributes to
// nothing useful, and a loud row would misstate the data.
func Parse(r io.Reader) ([]State, error) {
	br := bufio.NewReader(r)
	if magic, err := br.Peek(2); err == nil && magic[0] == 0x1f && magic[1] == 0x8b {
		gz, err := gzip.NewReader(br)
		if err != nil {
			return nil, fmt.Errorf("states: %w", err)
		}
		defer gz.Close()
		return parseGeoJSON(gz)
	}
	return parseGeoJSON(br)
}

func parseGeoJSON(r io.Reader) ([]State, error) {
	var fc struct {
		Type     string `json:"type"`
		Features []struct {
			Properties struct {
				Adm1Code string `json:"adm1_code"`
				ISO31662 string `json:"iso_3166_2"`
				Name     string `json:"name"`
				NameEn   string `json:"name_en"`
				ISOA2    string `json:"iso_a2"`
				ADM0A3   string `json:"adm0_a3"`
			} `json:"properties"`
			Geometry json.RawMessage `json:"geometry"`
		} `json:"features"`
	}
	if err := json.NewDecoder(r).Decode(&fc); err != nil {
		return nil, fmt.Errorf("states: not parseable as GeoJSON: %w", err)
	}
	if fc.Type != "FeatureCollection" || len(fc.Features) == 0 {
		return nil, fmt.Errorf("states: not a GeoJSON FeatureCollection with features")
	}

	out := make([]State, 0, len(fc.Features))
	seen := make(map[string]bool, len(fc.Features))
	for _, f := range fc.Features {
		p := f.Properties
		name := p.Name
		if name == "" {
			name = p.NameEn
		}
		if name == "" {
			continue // placeholder feature, see Parse
		}
		if p.Adm1Code == "" {
			return nil, fmt.Errorf("states: feature %q has no adm1_code", name)
		}
		if seen[p.Adm1Code] {
			return nil, fmt.Errorf("states: adm1_code %q appears twice", p.Adm1Code)
		}
		seen[p.Adm1Code] = true
		cc := countryCode(p.ISOA2, p.ADM0A3)
		if cc == "" {
			return nil, fmt.Errorf("states: feature %q (%s) has no usable country code (iso_a2=%q adm0_a3=%q)",
				name, p.Adm1Code, p.ISOA2, p.ADM0A3)
		}
		if len(f.Geometry) == 0 || string(f.Geometry) == "null" {
			return nil, fmt.Errorf("states: feature %q (%s) has no geometry", name, p.Adm1Code)
		}
		out = append(out, State{
			Code: p.Adm1Code, ISO3166_2: p.ISO31662, Name: name,
			CountryCode: cc, GeomJSON: f.Geometry,
		})
	}
	return out, nil
}

// countryCode mirrors countries.isoCode: a two-letter upper-case code passes,
// else the three-letter ADM0_A3. Natural Earth's nulls here are "-1" (twelve
// features: disputed or unclaimed areas such as Kashmir, the Indian Ocean
// territories, Somaliland) and never pass.
func countryCode(isoA2, adm0A3 string) string {
	if len(isoA2) == 2 && alpha(isoA2) {
		return isoA2
	}
	if len(adm0A3) == 3 && alpha(adm0A3) {
		return adm0A3
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
