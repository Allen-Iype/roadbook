package places

import (
	"crypto/sha256"
	"fmt"
	"math"
	"strings"
	"testing"
	"unicode"
)

// The embedded list is reference data the repository commits; this pins
// what the loader feeds the database, offline, in every `make test` run —
// the bytes (the package doc's checksum), the row count, key uniqueness,
// the name rule, and the two classes.
func TestBundled(t *testing.T) {
	const wantSum = "5caa978dcd0705cae4c4d558c92dcd3d055d0f0733bb357013b8031bbc265151"
	if got := fmt.Sprintf("%x", sha256.Sum256(bundled)); got != wantSum {
		t.Fatalf("embedded list sha256 = %s, want %s — rebuilt? update the package doc and this pin together", got, wantSum)
	}

	list, err := Bundled()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(list), 113584; got != want {
		t.Fatalf("bundled places = %d, want %d", got, want)
	}

	byName := map[string][]Place{}
	classes := map[string]int{}
	for _, p := range list {
		classes[p.Class]++
		if !strings.HasPrefix(p.ID, "n") {
			t.Fatalf("id %q is not a node", p.ID)
		}
		for _, r := range p.Name {
			if unicode.IsLetter(r) && !unicode.Is(unicode.Latin, r) {
				t.Fatalf("%s: name %q is not Latin script", p.ID, p.Name)
			}
		}
		switch p.Name {
		case "Ísafjörður", "Reykjavík", "Munnar":
			byName[p.Name] = append(byName[p.Name], p)
		}
	}
	if classes["city"] != 11788 || classes["town"] != 101796 || len(classes) != 2 {
		t.Errorf("classes = %v, want city 11788, town 101796 and nothing else", classes)
	}

	// The road-sign name with its diacritics, the class OpenStreetMap gives
	// it, and a position in the right part of the world (within a degree: the
	// point of the check is that latitude and longitude are not swapped).
	for _, w := range []struct {
		name, class string
		lat, lon    float64
	}{
		{"Ísafjörður", "town", 66, -23},
		{"Reykjavík", "city", 64, -22},
		{"Munnar", "town", 10, 77},
	} {
		found := false
		for _, p := range byName[w.name] {
			if p.Class == w.class && math.Abs(p.Lat-w.lat) < 1 && math.Abs(p.Lon-w.lon) < 1 {
				found = true
			}
		}
		if !found {
			t.Errorf("%s (%s) near %v, %v not in the list: %+v", w.name, w.class, w.lat, w.lon, byName[w.name])
		}
	}
}

func TestParseRejectsBrokenLists(t *testing.T) {
	good := "n1\tAlpha\ttown\t1200\t10.5\t20.25\nn2\tBeta\tcity\t0\t-33\t151\n"
	list, err := Parse(strings.NewReader(good))
	if err != nil || len(list) != 2 {
		t.Fatalf("good list: %v, %d rows", err, len(list))
	}
	if list[0] != (Place{ID: "n1", Name: "Alpha", Class: "town", Population: 1200, Lat: 10.5, Lon: 20.25}) {
		t.Errorf("first row = %+v", list[0])
	}

	for name, bad := range map[string]string{
		"empty":             "",
		"five fields":       "n1\tAlpha\ttown\t1200\t10.5\n",
		"no name":           "n1\t\ttown\t0\t10\t20\n",
		"duplicate id":      "n1\tAlpha\ttown\t0\t10\t20\nn1\tBeta\ttown\t0\t11\t21\n",
		"latitude over 90":  "n1\tAlpha\ttown\t0\t91\t20\n",
		"longitude swapped": "n1\tAlpha\ttown\t0\t20\t191\n",
		"population text":   "n1\tAlpha\ttown\tmany\t10\t20\n",
	} {
		if _, err := Parse(strings.NewReader(bad)); err == nil {
			t.Errorf("%s: want an error, got none", name)
		}
	}
}
