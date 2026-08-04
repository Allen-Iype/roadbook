package route_test

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"roadbook/internal/domain"
	"roadbook/internal/journey"
	"roadbook/internal/route"
)

var t0 = time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)

func at(sec int) time.Time { return t0.Add(time.Duration(sec) * time.Second) }

func TestKeyRounding(t *testing.T) {
	// 4-decimal fixed point, exact by construction — the float-equality trap
	// the integer representation exists to avoid (BRIEF §3C).
	k := route.KeyFor(
		domain.LatLng{Lat: 11.62495, Lon: 76.15391},
		domain.LatLng{Lat: -0.00004, Lon: -76.98765},
		"driving",
	)
	if k.FromLatE4 != 116250 || k.FromLonE4 != 761539 {
		t.Errorf("from = (%d, %d), want (116250, 761539)", k.FromLatE4, k.FromLonE4)
	}
	if k.ToLatE4 != 0 || k.ToLonE4 != -769877 {
		t.Errorf("to = (%d, %d), want (0, -769877)", k.ToLatE4, k.ToLonE4)
	}
	// The key's coordinates are the request: they round-trip exactly.
	if got := k.From(); got.Lat != 11.625 || got.Lon != 76.1539 {
		t.Errorf("From() = %+v, want the rounded coordinate", got)
	}
	// Nearby endpoints coalesce to one key (both sides of each value round
	// to the same e4 integer); direction never does.
	same := route.KeyFor(domain.LatLng{Lat: 11.62498, Lon: 76.15393}, domain.LatLng{Lat: 0.00001, Lon: -76.98766}, "driving")
	if same != k {
		t.Error("endpoints within rounding distance should share a key")
	}
	rev := route.KeyFor(k.To(), k.From(), "driving")
	if rev == k {
		t.Error("reversed endpoints must not share a key: one-way roads exist")
	}
	if route.KeyFor(k.From(), k.To(), "cycling") == k {
		t.Error("a different profile must not share a key")
	}
}

func TestNullFillsNothing(t *testing.T) {
	_, err := route.Null{}.Route(context.Background(), domain.LatLng{Lat: 1, Lon: 1}, domain.LatLng{Lat: 2, Lon: 2})
	if err != route.ErrNoRoute {
		t.Fatalf("err = %v, want ErrNoRoute", err)
	}
}

// gapJourney assembles a journey with one observed run, one ground-speed
// unknown gap, and one air gap — through the real Assemble, so this test
// consumes exactly what Apply consumes in production.
func gapJourney(t *testing.T) journey.Journey {
	t.Helper()
	obs := domain.Observations{
		Points: []domain.PathPoint{
			{Time: at(0), Loc: &domain.LatLng{Lat: 10.0, Lon: 70.0}},
			{Time: at(60), Loc: &domain.LatLng{Lat: 10.01, Lon: 70.0}},
			// 30 min silence over ~11 km: unknown gap.
			{Time: at(60 + 1800), Loc: &domain.LatLng{Lat: 10.11, Lon: 70.0}},
			// 1 h silence over ~5°: air gap (~556 km/h).
			{Time: at(60 + 1800 + 3600), Loc: &domain.LatLng{Lat: 15.11, Lon: 70.0}},
		},
	}
	j := journey.Assemble(obs, t0, at(4*3600), journey.DefaultParams())
	kinds := []journey.GapKind{}
	for _, l := range j.Legs {
		if l.Kind == journey.LegGap {
			kinds = append(kinds, l.GapKind)
		}
	}
	if len(kinds) != 2 || kinds[0] != journey.GapUnknown || kinds[1] != journey.GapAir {
		t.Fatalf("fixture legs wrong: gap kinds = %v, want [unknown air]", kinds)
	}
	return j
}

func TestUnknownKeysSkipsAirAndObserved(t *testing.T) {
	j := gapJourney(t)
	keys := route.UnknownKeys(j, "driving")
	if len(keys) != 1 {
		t.Fatalf("keys = %d, want 1 (the unknown gap only — never air, never observed)", len(keys))
	}
	want := route.KeyFor(domain.LatLng{Lat: 10.01, Lon: 70.0}, domain.LatLng{Lat: 10.11, Lon: 70.0}, "driving")
	if keys[0] != want {
		t.Errorf("key = %+v, want %+v", keys[0], want)
	}
}

func TestApply(t *testing.T) {
	j := gapJourney(t)
	key := route.UnknownKeys(j, "driving")[0]
	road := []domain.LatLng{{Lat: 10.01, Lon: 70.0}, {Lat: 10.05, Lon: 70.02}, {Lat: 10.11, Lon: 70.0}}
	lookup := map[route.Key]route.Cached{
		key: {Status: route.StatusRouted, Points: road, DistanceM: 13200, DurationS: 900},
	}

	out := route.Apply(j, "driving", lookup)

	var gotRoad, gotAir *journey.Leg
	for i := range out.Legs {
		l := &out.Legs[i]
		switch l.GapKind {
		case journey.GapRoad:
			gotRoad = l
		case journey.GapAir:
			gotAir = l
		}
	}
	if gotRoad == nil {
		t.Fatal("no road leg after Apply with a routed cache entry")
	}
	if len(gotRoad.Points) != 2 {
		t.Errorf("road leg carries %d timestamped points, want exactly 2 — routed geometry rides alongside", len(gotRoad.Points))
	}
	if len(gotRoad.RoutedPoints) != 3 || gotRoad.RoutedKm != 13.2 {
		t.Errorf("routed geometry = %d pts / %.1f km, want 3 / 13.2", len(gotRoad.RoutedPoints), gotRoad.RoutedKm)
	}
	if gotAir == nil || gotAir.RoutedPoints != nil {
		t.Error("the air leg must pass through untouched — flights are never routed")
	}
	if out.RoutedKm != 13.2 {
		t.Errorf("RoutedKm = %.1f, want 13.2", out.RoutedKm)
	}
	if out.UnknownKm != 0 {
		t.Errorf("UnknownKm = %.4f, want 0 (the only unknown gap was claimed)", out.UnknownKm)
	}
	// Chord aggregates keep their pinned meanings: routing changes nothing.
	if out.TotalKm != j.TotalKm || out.InferredKm != j.InferredKm || out.AirKm != j.AirKm {
		t.Error("chord aggregates changed under Apply; they are pinned semantics")
	}

	// The input journey was not mutated (invariant 2's discipline).
	for _, l := range j.Legs {
		if l.GapKind == journey.GapRoad || l.RoutedPoints != nil {
			t.Fatal("Apply mutated its input journey")
		}
	}
}

func TestApplyWithoutAnswersLeavesEverythingUnknown(t *testing.T) {
	j := gapJourney(t)
	unknownChord := j.UnknownKm

	// Empty cache, and separately a cached negative: both leave the gap
	// unknown — the designed degraded state, not an error (BRIEF §1.3).
	for name, lookup := range map[string]map[route.Key]route.Cached{
		"empty cache": {},
		"cached no_route": {
			route.UnknownKeys(j, "driving")[0]: {Status: route.StatusNoRoute},
		},
	} {
		out := route.Apply(j, "driving", lookup)
		for _, l := range out.Legs {
			if l.GapKind == journey.GapRoad {
				t.Errorf("%s: produced a road leg from nothing", name)
			}
		}
		if out.RoutedKm != 0 || math.Abs(out.UnknownKm-unknownChord) > 1e-12 {
			t.Errorf("%s: routed %.1f / unknown %.4f, want 0 / %.4f", name, out.RoutedKm, out.UnknownKm, unknownChord)
		}
	}
}

func TestOSRMClient(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Write([]byte(`{"code":"Ok","routes":[{"geometry":{"coordinates":[[70.0,10.01],[70.02,10.05],[70.0,10.11]],"type":"LineString"},"distance":13200.5,"duration":901.2}]}`))
	}))
	defer srv.Close()

	r, err := route.NewOSRM(srv.URL, "driving").Route(context.Background(),
		domain.LatLng{Lat: 10.01, Lon: 70.0}, domain.LatLng{Lat: 10.11, Lon: 70.0})
	if err != nil {
		t.Fatal(err)
	}
	// lon,lat on the wire (the one flip in Go); lat-first in the answer.
	if gotPath != "/route/v1/driving/70.0000,10.0100;70.0000,10.1100" {
		t.Errorf("request path = %s", gotPath)
	}
	if len(r.Points) != 3 || r.Points[0] != (domain.LatLng{Lat: 10.01, Lon: 70.0}) {
		t.Errorf("points = %+v", r.Points)
	}
	if r.DistanceM != 13200.5 || r.DurationS != 901.2 {
		t.Errorf("distance/duration = %.1f / %.1f", r.DistanceM, r.DurationS)
	}
}

func TestOSRMNoRouteAndErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"code":"NoRoute"}`))
	}))
	defer srv.Close()
	_, err := route.NewOSRM(srv.URL, "driving").Route(context.Background(), domain.LatLng{Lat: 1, Lon: 1}, domain.LatLng{Lat: 2, Lon: 2})
	if err != route.ErrNoRoute {
		t.Fatalf("err = %v, want ErrNoRoute (a cacheable data answer)", err)
	}

	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"code":"TooBig"}`))
	}))
	defer down.Close()
	_, err = route.NewOSRM(down.URL, "driving").Route(context.Background(), domain.LatLng{Lat: 1, Lon: 1}, domain.LatLng{Lat: 2, Lon: 2})
	if err == nil || err == route.ErrNoRoute {
		t.Fatalf("err = %v, want an operational error distinct from ErrNoRoute (never cached)", err)
	}
}
