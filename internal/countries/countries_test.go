package countries

import "testing"

// The bundled file is reference data the repository commits; this pins what
// the loader will actually feed the database, offline, in every `make test`
// run (no data/ gate — the file ships with the package).
func TestBundled(t *testing.T) {
	list, err := Bundled()
	if err != nil {
		t.Fatal(err)
	}
	// Natural Earth 1:110m admin-0 has exactly 177 country features.
	if got, want := len(list), 177; got != want {
		t.Fatalf("bundled countries = %d, want %d", got, want)
	}

	byCode := make(map[string]Country, len(list))
	for _, c := range list {
		byCode[c.ISOCode] = c
	}

	// Straightforward codes, plus every class of Natural Earth code quirk:
	// FR and NO carry ISO_A2 "-99" and recover via ISO_A2_EH; XK (Kosovo) is
	// a user-assigned alpha-2; CYN and SOL have no alpha-2 at all and fall
	// back to ADM0_A3.
	want := map[string]string{
		"IN":  "India",
		"NP":  "Nepal",
		"FR":  "France",
		"NO":  "Norway",
		"XK":  "Kosovo",
		"CYN": "Northern Cyprus",
		"SOL": "Somaliland",
	}
	for code, name := range want {
		c, ok := byCode[code]
		if !ok {
			t.Errorf("code %s missing", code)
			continue
		}
		if c.Name != name {
			t.Errorf("code %s = %q, want %q", code, c.Name, name)
		}
		if len(c.GeomJSON) == 0 {
			t.Errorf("code %s has empty geometry", code)
		}
	}
}
