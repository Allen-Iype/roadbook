package states

import "testing"

// The bundled file is reference data the repository commits; this pins what
// the loader feeds the database, offline, in every `make test` run — the
// feature count after the placeholder drop, key uniqueness, the name rule,
// and every class of country-code quirk.
func TestBundled(t *testing.T) {
	list, err := Bundled()
	if err != nil {
		t.Fatal(err)
	}
	// Natural Earth 1:10m admin-1 at the pinned commit has 4,596 features;
	// seven carry no name at all and are dropped (Parse).
	if got, want := len(list), 4589; got != want {
		t.Fatalf("bundled states = %d, want %d", got, want)
	}

	byCode := make(map[string]State, len(list))
	for _, s := range list {
		if _, dup := byCode[s.Code]; dup {
			t.Fatalf("adm1_code %q twice", s.Code)
		}
		byCode[s.Code] = s
		if s.Name == "" || s.CountryCode == "" || len(s.GeomJSON) == 0 {
			t.Errorf("state %q incomplete: %+v", s.Code, s)
		}
	}

	// By ISO 3166-2 where unique, checking the local-name rule (Vestfirðir,
	// not "Westfjords"; Suðurland, not "Southern"), a plain state, and the
	// ADM0_A3 fallback for a territory whose iso_a2 is Natural Earth's "-1".
	type want struct{ name, country string }
	wants := map[string]want{
		"IS-4":  {"Vestfirðir", "IS"},
		"IS-8":  {"Suðurland", "IS"},
		"IN-KL": {"Kerala", "IN"},
	}
	byISO := map[string]State{}
	for _, s := range list {
		if _, ok := wants[s.ISO3166_2]; ok {
			byISO[s.ISO3166_2] = s
		}
	}
	for iso, w := range wants {
		s, ok := byISO[iso]
		if !ok {
			t.Errorf("%s missing", iso)
			continue
		}
		if s.Name != w.name || s.CountryCode != w.country {
			t.Errorf("%s = %q/%q, want %q/%q", iso, s.Name, s.CountryCode, w.name, w.country)
		}
	}

	var cyn, sol int
	for _, s := range list {
		switch s.CountryCode {
		case "CYN":
			cyn++
		case "SOL":
			sol++
		}
	}
	if cyn == 0 || sol == 0 {
		t.Errorf("ADM0_A3 fallback: CYN=%d SOL=%d, want both > 0", cyn, sol)
	}
	for _, s := range list {
		if s.CountryCode == "-1" || s.CountryCode == "-99" {
			t.Errorf("Natural Earth null leaked as a country code: %+v", s)
		}
	}
}
