package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"roadbook/internal/api"
	"roadbook/internal/detect"
	"roadbook/internal/places"
	"roadbook/internal/states"
	"roadbook/internal/store"
	"roadbook/internal/store/storetest"
	"roadbook/internal/suggest"
)

// The demo's stops, named (phase 18, #16). The demo is fiction over public
// geography (testdata/demo/gen), so its names can be committed and pinned:
// this is the test that says what a fresh instance's demo pages print, and
// it runs the whole path — upload, detection, the journey handler, the
// places table loaded from the embedded list.
//
// It also holds the two states around naming: before `roadbook places` has
// run no stop carries a place (and nothing errors), and a stop with no
// points of its own is named from Google's visit record and says so.
func TestDemoStopsAreNamed(t *testing.T) {
	ts, s, _ := newUploadServer(t)
	ctx := context.Background()

	body, ctype := multipartUpload(t, demoExport)
	resp := postUpload(t, ts, body, ctype)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("demo upload: HTTP %d", resp.StatusCode)
	}
	accepted := decodeJSON[api.Import](t, resp.Body)
	resp.Body.Close()
	pollImport(t, ts, accepted.Id, func(i api.Import) bool {
		return i.Status == "completed" && i.DetectStatus != nil && *i.DetectStatus == "completed"
	})

	var list api.CandidateList
	fetchJSON(t, ts.URL+"/candidates", &list)
	if len(list.Candidates) != 3 {
		t.Fatalf("demo candidates = %d, want 3", len(list.Candidates))
	}
	journeyOf := func(id int64) api.Journey {
		var j api.Journey
		fetchJSON(t, fmt.Sprintf("%s/candidates/%d/journey", ts.URL, id), &j)
		return j
	}

	// Before the list is loaded: positions are known, names are not.
	for _, c := range list.Candidates {
		j := journeyOf(c.Id)
		if j.PlaceParams.NearKm != 10 || j.PlaceParams.MaxKm != 25 {
			t.Fatalf("place_params = %+v", j.PlaceParams)
		}
		for i, st := range j.Stops {
			if st.Place != nil {
				t.Fatalf("trip %d stop %d named with an empty places table: %+v", c.Id, i, st.Place)
			}
		}
	}

	bundled, err := places.Bundled()
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ReplacePlaces(ctx, bundled, places.SourceOSM); err != nil {
		t.Fatal(err)
	}

	// One line per stop: wording, name, position kind. Runs of the same
	// line are folded ("×3") to keep the pin readable.
	var got []string
	recorded, visit, none := 0, 0, 0
	for n, c := range list.Candidates {
		j := journeyOf(c.Id)
		var lines []string
		for _, st := range j.Stops {
			line := "—"
			if st.NamedFrom == nil {
				none++
			} else if *st.NamedFrom == api.Visit {
				visit++
			} else {
				recorded++
			}
			if st.Place != nil {
				if st.NamedFrom == nil {
					t.Fatalf("a place without a position kind: %+v", st)
				}
				if st.Place.Source != places.SourceOSM {
					t.Fatalf("source = %q", st.Place.Source)
				}
				if st.Place.Near != (st.Place.DistanceKm <= j.PlaceParams.NearKm) || st.Place.DistanceKm > j.PlaceParams.MaxKm {
					t.Fatalf("band disagrees with the echoed parameters: %+v", st.Place)
				}
				word := "near"
				if !st.Place.Near {
					word = "from"
				}
				line = fmt.Sprintf("%s %s (%s)", word, st.Place.Name, *st.NamedFrom)
			}
			lines = append(lines, line)
		}
		got = append(got, fmt.Sprintf("trip %d: %s", n+1, strings.Join(fold(lines), " · ")))
	}
	t.Logf("position kinds over the demo's stops: %d recorded, %d visit, %d none", recorded, visit, none)
	for _, g := range got {
		t.Log(g)
	}

	want := demoStopNames
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("demo stop names moved.\n got:\n  %s\nwant:\n  %s", strings.Join(got, "\n  "), strings.Join(want, "\n  "))
	}
	// The brief's measurement (§2.3), held here: 4 of the demo's 15 stops
	// have their own position, the other 11 are covered by a visit record.
	if recorded != 4 || visit != 11 || none != 0 {
		t.Errorf("position kinds = %d recorded, %d visit, %d none; want 4, 11, 0", recorded, visit, none)
	}
}

func fold(lines []string) []string {
	var out []string
	for i := 0; i < len(lines); {
		k := i
		for k < len(lines) && lines[k] == lines[i] {
			k++
		}
		if k-i > 1 {
			out = append(out, fmt.Sprintf("%s ×%d", lines[i], k-i))
		} else {
			out = append(out, lines[i])
		}
		i = k
	}
	return out
}

func fetchJSON(t *testing.T, url string, into any) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s: HTTP %d", url, resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(into); err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
}

// What the demo prints, trip by trip in the API's order. "(visit)" marks a
// stop named from Google's visit record. The four unnamed stops of trip 1
// are real: they lie along Iceland's south coast, more than 25 km from any
// place OpenStreetMap tags as a city or a town — the gap a list with
// villages would close (docs/phase-18/DECISIONS.md, "Villages wait").
var demoStopNames = []string{
	"trip 1: from Hvolsvöllur (visit) · — ×4 · near Höfn (visit) · near Selfoss (visit)",
	"trip 2: near Ísafjörður (visit)",
	"trip 3: near Reykjanesbær (recorded) · near Akureyri (recorded) · near Akureyri (visit) · from Akureyri (visit) · near Akureyri (visit) · near Akureyri (recorded) · near Reykjanesbær (recorded)",
}

// The demo's regions, leg by leg and stop by stop (phase 18, #17): what a
// day's line will be built from. Iceland's regions only are loaded — the
// demo never leaves the country, and the full file is the states package's
// own test. Before the load every leg carries an empty list, never an
// absent one.
func TestDemoRegionsPerLegAndStop(t *testing.T) {
	ts, s, _ := newUploadServer(t)
	ctx := context.Background()

	body, ctype := multipartUpload(t, demoExport)
	resp := postUpload(t, ts, body, ctype)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("demo upload: HTTP %d", resp.StatusCode)
	}
	accepted := decodeJSON[api.Import](t, resp.Body)
	resp.Body.Close()
	pollImport(t, ts, accepted.Id, func(i api.Import) bool {
		return i.Status == "completed" && i.DetectStatus != nil && *i.DetectStatus == "completed"
	})
	var list api.CandidateList
	fetchJSON(t, ts.URL+"/candidates", &list)

	// Raw JSON for the unloaded check: "states":[] must be on the wire.
	for _, c := range list.Candidates {
		var raw struct {
			Legs []map[string]json.RawMessage `json:"legs"`
		}
		fetchJSON(t, fmt.Sprintf("%s/candidates/%d/journey", ts.URL, c.Id), &raw)
		for i, l := range raw.Legs {
			if string(l["states"]) != "[]" {
				t.Fatalf("trip %d leg %d, unloaded table: states = %s, want []", c.Id, i, l["states"])
			}
		}
	}

	all, err := states.Bundled()
	if err != nil {
		t.Fatal(err)
	}
	var iceland []states.State
	for _, st := range all {
		if st.CountryCode == "IS" {
			iceland = append(iceland, st)
		}
	}
	if err := s.ReplaceStates(ctx, iceland); err != nil {
		t.Fatal(err)
	}

	var got []string
	for n, c := range list.Candidates {
		var j api.Journey
		fetchJSON(t, fmt.Sprintf("%s/candidates/%d/journey", ts.URL, c.Id), &j)

		var legs, stops []string
		tripLevel := map[string]bool{}
		for _, st := range j.States {
			tripLevel[st.Code] = true
		}
		for _, l := range j.Legs {
			var names []string
			seen := map[string]bool{}
			for _, st := range l.States {
				if seen[st.Code] {
					t.Fatalf("trip %d: a leg lists %s twice", n+1, st.Name)
				}
				seen[st.Code] = true
				// A leg's regions come from the same points as the trip's
				// list, so they can never name one the cover does not.
				if !tripLevel[st.Code] {
					t.Fatalf("trip %d: leg region %s is not in the trip's list", n+1, st.Name)
				}
				names = append(names, st.Name)
			}
			line := "—"
			if len(names) > 0 {
				line = strings.Join(names, " → ")
			}
			legs = append(legs, line)
		}
		for _, st := range j.Stops {
			line := "—"
			if st.State != nil {
				line = st.State.Name
			}
			stops = append(stops, line)
		}
		got = append(got,
			fmt.Sprintf("trip %d legs: %s", n+1, strings.Join(fold(legs), " | ")),
			fmt.Sprintf("trip %d stops: %s", n+1, strings.Join(fold(stops), " | ")))
	}
	for _, g := range got {
		t.Log(g)
	}
	if strings.Join(got, "\n") != strings.Join(demoRegions, "\n") {
		t.Errorf("demo regions moved.\n got:\n  %s\nwant:\n  %s", strings.Join(got, "\n  "), strings.Join(demoRegions, "\n  "))
	}
}

// What the demo carries. A dash is a leg or stop whose points fall in no
// region: Natural Earth's 1:10m coastline runs a few hundred metres inside
// the real one in places, so a harbour town or a shoreline road can sit
// outside every polygon (the caution TestStatesForPoints already records).
// Trip 2 shows the cost: its Westfjords legs are single recorded positions
// on the shore and carry no region, while its one stop, a little inland,
// does. Recorded as found (docs/phase-18/DECISIONS.md); whether a point
// just off a coast should take the nearest region is a decision of its own.
var demoRegions = []string{
	"trip 1 legs: Suðurland ×5 | — | Suðurland → Austurland | Austurland ×4 | — | Austurland → Suðurland | Suðurland ×2",
	"trip 1 stops: Suðurland ×2 | — | Austurland ×2 | — | Suðurland",
	"trip 2 legs: — ×3 | Vesturland ×2",
	"trip 2 stops: Vestfirðir",
	"trip 3 legs: Suðurnes ×4 | — | Norðurland eystra ×5 | — | Suðurnes ×2",
	"trip 3 stops: Suðurnes | — | Norðurland eystra ×3 | — | Suðurnes",
}

// The trip-name suggestion, offline (phase 18, #18): with the places table
// loaded, the default suggester answers with the nearest listed town to
// each demo trip's destination; before the load it answers with nothing and
// no error — the confirm step then simply has no prefill, as it always has
// on an instance without a geocoder.
func TestDemoNameSuggestionsFromPlaces(t *testing.T) {
	s := storetest.Open(t)
	ctx := context.Background()
	uploads := store.UploadFiles{Dir: t.TempDir()}
	if err := uploads.Init(); err != nil {
		t.Fatal(err)
	}
	srv := &api.Server{Store: s, MatchParams: detect.DefaultMatchParams(), Uploads: uploads, Suggester: suggest.NewPlaces(s)}
	ts := httptest.NewServer(api.HandlerFromMux(api.NewStrictHandler(srv, []api.StrictMiddlewareFunc{srv.AuthMiddleware}), http.NewServeMux()))
	t.Cleanup(ts.Close)

	body, ctype := multipartUpload(t, demoExport)
	resp := postUpload(t, ts, body, ctype)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("demo upload: HTTP %d", resp.StatusCode)
	}
	accepted := decodeJSON[api.Import](t, resp.Body)
	resp.Body.Close()
	pollImport(t, ts, accepted.Id, func(i api.Import) bool {
		return i.Status == "completed" && i.DetectStatus != nil && *i.DetectStatus == "completed"
	})
	var list api.CandidateList
	fetchJSON(t, ts.URL+"/candidates", &list)

	suggestions := func() []string {
		var out []string
		for _, c := range list.Candidates {
			var sug api.NameSuggestion
			fetchJSON(t, fmt.Sprintf("%s/candidates/%d/name-suggestion", ts.URL, c.Id), &sug)
			name := "—"
			if sug.Name != nil {
				name = *sug.Name
			}
			out = append(out, fmt.Sprintf("%s (%s)", name, sug.Source))
		}
		return out
	}

	if got, want := strings.Join(suggestions(), " · "), "— (places) · — (places) · — (places)"; got != want {
		t.Errorf("unloaded table: %s, want %s", got, want)
	}

	bundled, err := places.Bundled()
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ReplacePlaces(ctx, bundled, places.SourceOSM); err != nil {
		t.Fatal(err)
	}
	got := strings.Join(suggestions(), " · ")
	t.Log(got)
	if got != demoSuggestions {
		t.Errorf("demo suggestions moved: %s, want %s", got, demoSuggestions)
	}
}

// The nearest listed town to each demo trip's destination, in the API's order.
const demoSuggestions = "Höfn (osm) · Ísafjörður (osm) · Akureyri (osm)"
