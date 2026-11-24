package route_test

// Phase 23 (#9): a recorded stretch drawn along the road its points lie on.
// These tests were written before the code they test. They pin, in order:
// which stretches are asked about at all, that the cache key is the request,
// what the null and OSRM matchers answer, and — the part the product's
// honesty rests on — what ApplyMatches may and may not change on a journey.

import (
	"context"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"roadbook/internal/domain"
	"roadbook/internal/journey"
	"roadbook/internal/route"
)

// matchJourney is assembled by the real Assemble, like gapJourney: a
// four-point recorded stretch of about 3.3 km, a silence (unknown gap), then
// a two-point recorded stretch of about 110 m — under MATCH_MIN_M.
func matchJourney(t *testing.T) journey.Journey {
	t.Helper()
	obs := domain.Observations{
		Points: []domain.PathPoint{
			{Time: at(0), Loc: &domain.LatLng{Lat: 10.00, Lon: 70.0}},
			{Time: at(60), Loc: &domain.LatLng{Lat: 10.01, Lon: 70.0}},
			{Time: at(120), Loc: &domain.LatLng{Lat: 10.02, Lon: 70.0}},
			{Time: at(180), Loc: &domain.LatLng{Lat: 10.03, Lon: 70.0}},
			// 30 min of silence over ~11 km: an unknown gap.
			{Time: at(180 + 1800), Loc: &domain.LatLng{Lat: 10.13, Lon: 70.0}},
			{Time: at(180 + 1800 + 60), Loc: &domain.LatLng{Lat: 10.131, Lon: 70.0}},
		},
	}
	j := journey.Assemble(obs, t0, at(4*3600), journey.DefaultParams())
	var shape []string
	for _, l := range j.Legs {
		shape = append(shape, string(l.Kind))
	}
	if strings.Join(shape, " ") != "observed gap observed" {
		t.Fatalf("fixture legs = %v, want [observed gap observed]", shape)
	}
	if n := len(j.Legs[0].Points); n != 4 {
		t.Fatalf("first stretch has %d points, want 4", n)
	}
	return j
}

func TestDefaultMatchParams(t *testing.T) {
	p := route.DefaultMatchParams()
	if p.RadiusM != 100 || p.MaxRatio != 1.5 || p.MinM != 200 {
		t.Errorf("defaults = %+v, want radius 100 m, ratio 1.5, minimum 200 m (phase 23 BRIEF §4 B, C; DECISIONS 2026-11-24)", p)
	}
}

func TestTracesAskOnlyAboutRecordedStretchesLongEnough(t *testing.T) {
	j := matchJourney(t)
	p := route.DefaultMatchParams()

	traces := route.Traces(j, nil, "driving", p)
	if len(traces) != 1 {
		t.Fatalf("traces = %d, want 1: the 3.3 km stretch only — never a gap, never the 110 m stretch under MATCH_MIN_M", len(traces))
	}
	tr := traces[0]
	if len(tr.Points) != 4 {
		t.Fatalf("trace carries %d points, want all 4 of the stretch", len(tr.Points))
	}
	// The request is the recorded points at five decimals with their times.
	if tr.Points[1].LatE5 != 1001000 || tr.Points[1].LonE5 != 7000000 || tr.Points[1].Unix != at(60).Unix() {
		t.Errorf("point 1 = %+v, want (1001000, 7000000) at %d", tr.Points[1], at(60).Unix())
	}
	if got := tr.Points[1].Loc(); got.Lat != 10.01 || got.Lon != 70.0 {
		t.Errorf("Loc() = %+v, want the rounded coordinate", got)
	}

	// Lowering the minimum brings the short stretch in: the threshold is the
	// parameter, not a literal.
	p.MinM = 50
	if n := len(route.Traces(j, nil, "driving", p)); n != 2 {
		t.Errorf("with MATCH_MIN_M 50: %d traces, want 2", n)
	}
}

func TestMatchKeyIsTheRequest(t *testing.T) {
	j := matchJourney(t)
	p := route.DefaultMatchParams()
	leg := j.Legs[0]

	base, ok := route.TraceFor(leg, "", "driving", p)
	if !ok {
		t.Fatal("the 3.3 km stretch must be eligible")
	}
	again, _ := route.TraceFor(leg, "", "driving", p)
	if base.Key != again.Key {
		t.Error("the same stretch must give the same key every time")
	}
	if base.Key.Profile != "driving" || base.Key.RadiusM != 100 {
		t.Errorf("key = %+v, want profile driving, radius 100", base.Key)
	}
	if len(base.Key.PointsHash) != 64 {
		t.Errorf("points hash = %q, want 64 hex characters (SHA-256)", base.Key.PointsHash)
	}

	differs := func(name string, mutate func(*journey.Leg, *string, *route.MatchParams)) {
		l := leg
		l.Points = append([]journey.TimedPoint(nil), leg.Points...)
		profile, mp := "driving", p
		mutate(&l, &profile, &mp)
		got, ok := route.TraceFor(l, "", profile, mp)
		if !ok {
			t.Fatalf("%s: stretch became ineligible", name)
		}
		if got.Key == base.Key {
			t.Errorf("%s: must not share a cache key with the original request", name)
		}
	}
	differs("a point moved by 2 m", func(l *journey.Leg, _ *string, _ *route.MatchParams) {
		l.Points[2].Loc.Lat += 0.00002
	})
	differs("a point's time moved by a second", func(l *journey.Leg, _ *string, _ *route.MatchParams) {
		l.Points[2].Time = l.Points[2].Time.Add(1e9)
	})
	differs("a different profile", func(_ *journey.Leg, profile *string, _ *route.MatchParams) {
		*profile = "cycling"
	})
	differs("a different search radius", func(_ *journey.Leg, _ *string, mp *route.MatchParams) {
		mp.RadiusM = 50
	})

	// The guard is applied on reading, so it is not part of the question:
	// changing it must reuse every cached answer (BRIEF §3).
	mp := p
	mp.MaxRatio = 1.5
	if got, _ := route.TraceFor(leg, "", "driving", mp); got.Key != base.Key {
		t.Error("MATCH_MAX_RATIO must not be part of the cache key")
	}
}

func TestTraceTimesStrictlyIncrease(t *testing.T) {
	// OSRM rejects a trace whose timestamps do not strictly increase. Two
	// points in the same second are sent one second apart; the recorded
	// points themselves are not touched.
	leg := journey.Leg{
		Kind:       journey.LegObserved,
		DistanceKm: 2.2,
		Points: []journey.TimedPoint{
			{Time: at(0), Loc: domain.LatLng{Lat: 10.00, Lon: 70}},
			{Time: at(0), Loc: domain.LatLng{Lat: 10.01, Lon: 70}},
			{Time: at(0), Loc: domain.LatLng{Lat: 10.02, Lon: 70}},
		},
	}
	tr, ok := route.TraceFor(leg, "", "driving", route.DefaultMatchParams())
	if !ok {
		t.Fatal("eligible stretch refused")
	}
	u := at(0).Unix()
	if tr.Points[0].Unix != u || tr.Points[1].Unix != u+1 || tr.Points[2].Unix != u+2 {
		t.Errorf("times sent = %d, %d, %d; want %d, %d, %d", tr.Points[0].Unix, tr.Points[1].Unix, tr.Points[2].Unix, u, u+1, u+2)
	}
	if !leg.Points[1].Time.Equal(at(0)) {
		t.Error("TraceFor altered a recorded point's time")
	}
}

func TestNullMatchesNothing(t *testing.T) {
	var m route.Matcher = route.Null{}
	_, err := m.Match(context.Background(), []route.TracePoint{{LatE5: 1, LonE5: 1, Unix: 1}, {LatE5: 2, LonE5: 2, Unix: 2}}, 100)
	if err != route.ErrNoMatch {
		t.Fatalf("err = %v, want ErrNoMatch", err)
	}
}

// snapped is a road line a little longer than the first stretch's 3.3 km.
var snapped = []domain.LatLng{{Lat: 10.0001, Lon: 70.0002}, {Lat: 10.015, Lon: 70.003}, {Lat: 10.0299, Lon: 70.0001}}

func TestApplyMatches(t *testing.T) {
	j := matchJourney(t)
	p := route.DefaultMatchParams()
	tr := route.Traces(j, nil, "driving", p)[0]
	chordKm := j.Legs[0].DistanceKm
	lookup := map[route.MatchKey]route.CachedMatch{
		tr.Key: {Status: route.StatusMatched, Points: snapped, DistanceM: chordKm * 1000 * 1.2, EndsPlaced: true},
	}

	out := route.ApplyMatches(j, nil, "driving", p, lookup)

	got := out.Legs[0]
	if got.Kind != journey.LegObserved || got.GapKind != journey.GapNone {
		t.Errorf("kind = %s/%q; a stretch drawn along the road stays recorded (invariant 6)", got.Kind, got.GapKind)
	}
	if len(got.SnappedPoints) != 3 {
		t.Fatalf("snapped geometry = %d points, want 3", len(got.SnappedPoints))
	}
	if math.Abs(got.SnappedKm-chordKm*1.2) > 1e-9 {
		t.Errorf("leg SnappedKm = %.4f, want the road length %.4f", got.SnappedKm, chordKm*1.2)
	}
	// Recorded points are never altered or discarded (invariant 6).
	if len(got.Points) != 4 {
		t.Fatalf("leg has %d recorded points after ApplyMatches, want 4", len(got.Points))
	}
	for i := range got.Points {
		if got.Points[i] != j.Legs[0].Points[i] {
			t.Errorf("recorded point %d changed", i)
		}
	}
	// Distances stay computed from the recorded points (BRIEF §4 D).
	if got.DistanceKm != chordKm {
		t.Errorf("leg DistanceKm = %.4f, want %.4f unchanged", got.DistanceKm, chordKm)
	}
	if out.TotalKm != j.TotalKm || out.ObservedKm != j.ObservedKm || out.InferredKm != j.InferredKm ||
		out.UnknownKm != j.UnknownKm || out.RoutedKm != j.RoutedKm || out.GroundKm() != j.GroundKm() {
		t.Error("a journey distance changed under ApplyMatches; every one of them stays from the recorded points")
	}
	// The two reported figures: how much recorded distance follows a road,
	// and how long that road is.
	if math.Abs(out.SnappedObservedKm-chordKm) > 1e-9 || math.Abs(out.SnappedRoadKm-chordKm*1.2) > 1e-9 {
		t.Errorf("journey snapped = %.4f recorded km along %.4f road km, want %.4f along %.4f",
			out.SnappedObservedKm, out.SnappedRoadKm, chordKm, chordKm*1.2)
	}
	// The gap and the short stretch pass through untouched.
	if out.Legs[1].SnappedPoints != nil || out.Legs[2].SnappedPoints != nil {
		t.Error("only the matched stretch may carry a snapped geometry")
	}
	if out.Legs[1].Kind != journey.LegGap || out.Legs[1].GapKind != journey.GapUnknown {
		t.Error("a gap remains its own kind (invariant 6)")
	}
	// The input journey was not mutated.
	if j.Legs[0].SnappedPoints != nil || j.SnappedObservedKm != 0 {
		t.Fatal("ApplyMatches mutated its input journey")
	}
}

func TestApplyMatchesGuards(t *testing.T) {
	j := matchJourney(t)
	p := route.DefaultMatchParams()
	tr := route.Traces(j, nil, "driving", p)[0]
	chordM := j.Legs[0].DistanceKm * 1000
	matched := func(ratio float64) route.CachedMatch {
		return route.CachedMatch{Status: route.StatusMatched, Points: snapped, DistanceM: chordM * ratio, EndsPlaced: true}
	}

	cases := []struct {
		name   string
		params func(route.MatchParams) route.MatchParams
		cached *route.CachedMatch
		want   bool
	}{
		{name: "nothing cached", want: false},
		{name: "a remembered no_match", cached: &route.CachedMatch{Status: route.StatusNoMatch}, want: false},
		{name: "road 1.49 times point to point", cached: ptr(matched(1.49)), want: true},
		{name: "road exactly 1.5 times: kept only when under", cached: ptr(matched(1.5)), want: false},
		{name: "road twice as long: a side-road detour", cached: ptr(matched(2)), want: false},
		{
			name:   "the same detour under a guard of 2.5: the guard is read, not stored",
			params: func(p route.MatchParams) route.MatchParams { p.MaxRatio = 2.5; return p },
			cached: ptr(matched(2)), want: true,
		},
		{
			name: "first or last point not placed on a road: the line would stop short",
			cached: &route.CachedMatch{Status: route.StatusMatched, Points: snapped, DistanceM: chordM * 0.6,
				Unplaced: 1, EndsPlaced: false},
			want: false,
		},
		{
			name: "a middle point not placed: the line still spans the stretch",
			cached: &route.CachedMatch{Status: route.StatusMatched, Points: snapped, DistanceM: chordM * 1.1,
				Unplaced: 1, EndsPlaced: true},
			want: true,
		},
		{
			name:   "a guard of 0 switches the drawing off",
			params: func(p route.MatchParams) route.MatchParams { p.MaxRatio = 0; return p },
			cached: ptr(matched(1.0)), want: false,
		},
	}
	for _, c := range cases {
		mp := p
		if c.params != nil {
			mp = c.params(p)
		}
		lookup := map[route.MatchKey]route.CachedMatch{}
		if c.cached != nil {
			lookup[tr.Key] = *c.cached
		}
		out := route.ApplyMatches(j, nil, "driving", mp, lookup)
		got := out.Legs[0].SnappedPoints != nil
		if got != c.want {
			t.Errorf("%s: drawn along the road = %v, want %v", c.name, got, c.want)
		}
		if !got && (out.SnappedObservedKm != 0 || out.SnappedRoadKm != 0 || out.Legs[0].SnappedKm != 0) {
			t.Errorf("%s: snapped figures are non-zero for a stretch drawn point to point", c.name)
		}
	}
}

func ptr[T any](v T) *T { return &v }

func TestOSRMMatch(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		w.Write([]byte(`{"code":"Ok",
			"matchings":[{"geometry":{"coordinates":[[70.0002,10.0001],[70.003,10.015],[70.0001,10.0299]],"type":"LineString"},"distance":3601.5,"confidence":0.04}],
			"tracepoints":[{"location":[70.0002,10.0001]},null,{"location":[70.0001,10.0299]}]}`))
	}))
	defer srv.Close()

	var m route.Matcher = route.NewOSRM(srv.URL, "driving")
	got, err := m.Match(context.Background(), []route.TracePoint{
		{LatE5: 1000000, LonE5: 7000000, Unix: 1000},
		{LatE5: 1001500, LonE5: 7000123, Unix: 1060},
		{LatE5: 1003000, LonE5: 7000000, Unix: 1120},
	}, 100)
	if err != nil {
		t.Fatal(err)
	}
	// lon,lat on the wire, five decimals; lat-first in the answer.
	if gotPath != "/match/v1/driving/70.00000,10.00000;70.00123,10.01500;70.00000,10.03000" {
		t.Errorf("request path = %s", gotPath)
	}
	for _, want := range []string{"timestamps=1000;1060;1120", "radiuses=100;100;100", "gaps=ignore", "tidy=false", "overview=full", "geometries=geojson"} {
		if !strings.Contains(gotQuery, want) {
			t.Errorf("query %q lacks %q", gotQuery, want)
		}
	}
	if len(got.Points) != 3 || got.Points[0] != (domain.LatLng{Lat: 10.0001, Lon: 70.0002}) {
		t.Errorf("points = %+v", got.Points)
	}
	if got.DistanceM != 3601.5 {
		t.Errorf("distance = %.1f, want 3601.5", got.DistanceM)
	}
	if got.Unplaced != 1 || !got.EndsPlaced {
		t.Errorf("unplaced = %d, ends placed = %v; want 1 (the middle point) and true", got.Unplaced, got.EndsPlaced)
	}
}

func TestOSRMMatchAnswersAndErrors(t *testing.T) {
	trace := []route.TracePoint{{LatE5: 1, LonE5: 1, Unix: 1}, {LatE5: 2, LonE5: 2, Unix: 2}}
	ask := func(status int, body string) (route.Match, error) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			w.Write([]byte(body))
		}))
		defer srv.Close()
		return route.NewOSRM(srv.URL, "driving").Match(context.Background(), trace, 100)
	}

	// Data answers, remembered in the cache.
	for name, body := range map[string]string{
		"no road within the radius": `{"code":"NoMatch"}`,
		"no segment":                `{"code":"NoSegment"}`,
		"the matcher split the stretch in two: not one line along one road": `{"code":"Ok",
			"matchings":[{"geometry":{"coordinates":[[1,1],[1,2]]},"distance":10},{"geometry":{"coordinates":[[2,2],[2,3]]},"distance":10}],
			"tracepoints":[{},{}]}`,
	} {
		if _, err := ask(http.StatusOK, body); err != route.ErrNoMatch {
			t.Errorf("%s: err = %v, want ErrNoMatch", name, err)
		}
	}

	// A first point left off the road is still an answer, and says so.
	got, err := ask(http.StatusOK, `{"code":"Ok",
		"matchings":[{"geometry":{"coordinates":[[1,1],[1,2]]},"distance":10}],
		"tracepoints":[null,{}]}`)
	if err != nil || got.EndsPlaced || got.Unplaced != 1 {
		t.Errorf("first point unplaced: err %v, ends placed %v, unplaced %d; want nil, false, 1", err, got.EndsPlaced, got.Unplaced)
	}

	// Operational: reported, never cached. A stretch with more points than
	// the server accepts is one of these — it is the server's setting, not
	// a fact about the road.
	for name, body := range map[string]string{
		"too many points for this server": `{"code":"TooBig"}`,
		"not JSON at all":                 `<html>bad gateway</html>`,
	} {
		_, err := ask(http.StatusBadRequest, body)
		if err == nil || errors.Is(err, route.ErrNoMatch) {
			t.Errorf("%s: err = %v, want an operational error distinct from ErrNoMatch", name, err)
		}
	}
}

// --- Modes that do not travel on roads (phase 23, decided 2026-11-23) ---
//
// Written before the code. Measured: at the default guard, 69% to 100% of
// the kilometres Google labels as train pass it and would be drawn on the
// road beside the railway. The matcher has one profile, driving, so a
// stretch whose mode is in MATCH_SKIP_MODES is not asked about and not
// drawn along a road, whatever the cache holds.

// act is an activity over [fromS, toS] seconds of the fixture clock.
func act(mode string, fromS, toS int) domain.Activity {
	return domain.Activity{Start: at(fromS), End: at(toS), Mode: mode}
}

func TestDefaultSkipModes(t *testing.T) {
	got := strings.Join(route.DefaultMatchParams().SkipModes, " ")
	if got != "IN_TRAIN IN_SUBWAY IN_FERRY FLYING" {
		t.Errorf("MATCH_SKIP_MODES = %q, want the four measured labels that are not road travel", got)
	}
}

func TestLegMode(t *testing.T) {
	j := matchJourney(t)
	first := j.Legs[0] // recorded from 0 s to 180 s
	cases := []struct {
		name string
		acts []domain.Activity
		want string
	}{
		{"no activity at all", nil, ""},
		{"an activity that does not overlap the stretch", []domain.Activity{act("IN_TRAIN", 600, 900)}, ""},
		{"an activity that only touches its end", []domain.Activity{act("IN_TRAIN", 180, 900)}, ""},
		{"the one overlapping activity", []domain.Activity{act("IN_BUS", -60, 400)}, "IN_BUS"},
		{"the longer overlap wins", []domain.Activity{act("WALKING", 0, 50), act("IN_TRAIN", 50, 180)}, "IN_TRAIN"},
		{"the longer overlap wins whatever the order", []domain.Activity{act("IN_TRAIN", 50, 180), act("WALKING", 0, 50)}, "IN_TRAIN"},
		{"equal overlaps: the earlier activity", []domain.Activity{act("IN_TRAIN", 90, 180), act("WALKING", 0, 90)}, "WALKING"},
	}
	for _, tc := range cases {
		if got := route.LegMode(first, tc.acts); got != tc.want {
			t.Errorf("%s: mode = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestSkippedModesAreNotAskedAbout(t *testing.T) {
	j := matchJourney(t)
	p := route.DefaultMatchParams()
	train := []domain.Activity{act("IN_TRAIN", 0, 180)}
	bus := []domain.Activity{act("IN_BUS", 0, 180)}

	if n := len(route.Traces(j, train, "driving", p)); n != 0 {
		t.Errorf("a stretch labelled train: %d traces, want 0", n)
	}
	if n := len(route.Traces(j, bus, "driving", p)); n != 1 {
		t.Errorf("a stretch labelled bus: %d traces, want 1", n)
	}
	if n := len(route.Traces(j, nil, "driving", p)); n != 1 {
		t.Errorf("a stretch with no label: %d traces, want 1 — no label is not a reason to skip", n)
	}
	// The list is the parameter: emptied, the train stretch is asked about;
	// with the bus added, the bus stretch is not.
	p.SkipModes = nil
	if n := len(route.Traces(j, train, "driving", p)); n != 1 {
		t.Errorf("empty MATCH_SKIP_MODES: %d traces for the train stretch, want 1", n)
	}
	p.SkipModes = []string{"IN_BUS"}
	if n := len(route.Traces(j, bus, "driving", p)); n != 0 {
		t.Errorf("MATCH_SKIP_MODES = IN_BUS: %d traces for the bus stretch, want 0", n)
	}
}

// The skip is applied on reading too: an answer already in the cache (from
// before the rule, or from a run with another list) is not drawn.
func TestSkippedModesAreNotDrawnFromTheCache(t *testing.T) {
	j := matchJourney(t)
	p := route.DefaultMatchParams()
	tr := route.Traces(j, nil, "driving", p)[0]
	lookup := map[route.MatchKey]route.CachedMatch{
		tr.Key: {Status: route.StatusMatched, Points: snapped, DistanceM: j.Legs[0].DistanceKm * 1000 * 1.1, EndsPlaced: true},
	}

	out := route.ApplyMatches(j, []domain.Activity{act("IN_TRAIN", 0, 180)}, "driving", p, lookup)
	if out.Legs[0].SnappedPoints != nil || out.SnappedObservedKm != 0 || out.SnappedRoadKm != 0 {
		t.Errorf("a train stretch was drawn along a road from the cache: %d points, %.2f km", len(out.Legs[0].SnappedPoints), out.SnappedObservedKm)
	}
	out = route.ApplyMatches(j, []domain.Activity{act("IN_BUS", 0, 180)}, "driving", p, lookup)
	if out.Legs[0].SnappedPoints == nil {
		t.Error("a bus stretch with a passing answer in the cache was not drawn along the road")
	}
}

// Matching sends every recorded point of a stretch, where gap routing
// sends two ends. Against the public demonstration server it is therefore
// off unless asked for.
func TestMatchByDefault(t *testing.T) {
	cases := []struct {
		url  string
		want bool
	}{
		{route.PublicOSRMURL, false},
		{route.PublicOSRMURL + "/", false},
		{"http://router.project-osrm.org", false},
		{"https://ROUTER.project-osrm.org", false},
		{"http://127.0.0.1:5001", true},
		{"http://osrm:5000", true},
		{"https://osrm.example.org/", true},
		{"not a url", true},
	}
	for _, tc := range cases {
		if got := route.MatchByDefault(tc.url); got != tc.want {
			t.Errorf("MatchByDefault(%q) = %v, want %v", tc.url, got, tc.want)
		}
	}
}
