package api_test

// Phase 23 (#9): a recorded stretch drawn along the road, on the served
// journey. Written before the handler code. The demo is uploaded through
// the path a user takes; one stretch's answer is then put in the match
// cache by hand — the server never dials a router — and the journey is
// read back.

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"roadbook/internal/api"
	"roadbook/internal/detect"
	"roadbook/internal/domain"
	"roadbook/internal/journey"
	"roadbook/internal/route"
)

func TestServedJourneyCarriesSnappedGeometry(t *testing.T) {
	ts, s, _ := newUploadServer(t)
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
	url := fmt.Sprintf("%s/candidates/%d/journey", ts.URL, list.Candidates[0].Id)

	// Before anything is cached: today's drawing, with the parameters served.
	var before api.Journey
	fetchJSON(t, url, &before)
	if mp := before.MatchParams; mp.RadiusM != 100 || mp.MaxRatio != 1.5 || mp.MinM != 200 ||
		strings.Join(mp.SkipModes, " ") != "IN_TRAIN IN_SUBWAY IN_FERRY FLYING" {
		t.Errorf("match_params = %+v, want 100 m, 1.5, 200 m and the four skipped modes", mp)
	}
	if before.SnappedObservedKm != 0 || before.SnappedRoadKm != 0 {
		t.Errorf("empty cache: snapped %.1f / %.1f km, want 0 / 0", before.SnappedObservedKm, before.SnappedRoadKm)
	}
	for i, l := range before.Legs {
		if l.SnappedPoints != nil || l.SnappedKm != nil {
			t.Fatalf("leg %d carries a snapped geometry with an empty cache", i)
		}
	}

	// The first recorded stretch long enough to be asked about, rebuilt from
	// what the API served: the cache key is the request, so the served
	// points are enough to find it.
	p := route.DefaultMatchParams()
	target := -1
	var tr route.Trace
	for i, l := range before.Legs {
		if l.Kind != api.LegKindObserved {
			continue
		}
		leg := journey.Leg{Kind: journey.LegObserved, DistanceKm: l.DistanceKm}
		for _, pt := range l.Points {
			leg.Points = append(leg.Points, journey.TimedPoint{Time: pt.T, Loc: domain.LatLng{Lat: pt.Lat, Lon: pt.Lon}})
		}
		if got, ok := route.TraceFor(leg, "", api.RouteProfile, p); ok {
			target, tr = i, got
			break
		}
	}
	if target < 0 {
		t.Fatal("the demo's first trip has no recorded stretch long enough to match")
	}
	chordKm := before.Legs[target].DistanceKm
	first, last := before.Legs[target].Points[0], before.Legs[target].Points[len(before.Legs[target].Points)-1]
	road := []domain.LatLng{{Lat: first.Lat, Lon: first.Lon}, {Lat: (first.Lat + last.Lat) / 2, Lon: (first.Lon+last.Lon)/2 + 0.001}, {Lat: last.Lat, Lon: last.Lon}}
	save := func(ratio float64) {
		t.Helper()
		if err := s.SaveMatch(context.Background(), "self", tr, route.CachedMatch{
			Status: route.StatusMatched, Points: road, DistanceM: chordKm * 1000 * ratio, EndsPlaced: true,
		}, "test", ""); err != nil {
			t.Fatal(err)
		}
	}

	save(1.2)
	var after api.Journey
	fetchJSON(t, url, &after)
	got := after.Legs[target]
	if got.Kind != api.LegKindObserved || got.GapKind != nil {
		t.Errorf("the snapped leg is served as %s (gap_kind %v); it stays observed", got.Kind, got.GapKind)
	}
	if got.SnappedPoints == nil || len(*got.SnappedPoints) != 3 {
		t.Fatalf("snapped_points = %v, want the 3 cached road points", got.SnappedPoints)
	}
	if (*got.SnappedPoints)[1].Lat != road[1].Lat || (*got.SnappedPoints)[1].Lon != road[1].Lon {
		t.Errorf("snapped_points[1] = %+v, want %+v", (*got.SnappedPoints)[1], road[1])
	}
	if got.SnappedKm == nil || math.Abs(*got.SnappedKm-chordKm*1.2) > 1e-9 {
		t.Errorf("snapped_km = %v, want %.4f", got.SnappedKm, chordKm*1.2)
	}
	if len(got.Points) != len(before.Legs[target].Points) || got.DistanceKm != chordKm {
		t.Error("the leg's recorded points or its distance changed")
	}
	if math.Abs(after.SnappedObservedKm-chordKm) > 1e-9 || math.Abs(after.SnappedRoadKm-chordKm*1.2) > 1e-9 {
		t.Errorf("journey snapped = %.4f recorded km along %.4f road km, want %.4f along %.4f",
			after.SnappedObservedKm, after.SnappedRoadKm, chordKm, chordKm*1.2)
	}
	// Every figure computed from recorded points is as it was (BRIEF §4 D).
	if after.TotalKm != before.TotalKm || after.ObservedKm != before.ObservedKm || after.GroundKm != before.GroundKm ||
		!reflect.DeepEqual(after.Summary, before.Summary) || !reflect.DeepEqual(after.DivergencePct, before.DivergencePct) {
		t.Error("a headline figure moved when a stretch was drawn along the road")
	}
	for i, l := range after.Legs {
		if i != target && l.SnappedPoints != nil {
			t.Errorf("leg %d carries a snapped geometry nobody cached", i)
		}
	}

	// The guard is applied when reading: the same cache row at twice the
	// point-to-point length is not drawn.
	save(2.0)
	var guarded api.Journey
	fetchJSON(t, url, &guarded)
	if guarded.Legs[target].SnappedPoints != nil || guarded.SnappedObservedKm != 0 {
		t.Error("a match twice as long as its points was served; the guard must refuse it on reading")
	}

	// The guard is a setting of the server, not of the cache: a second
	// server over the same database with a looser guard draws that same
	// row, and says which guard it used.
	loose := route.DefaultMatchParams()
	loose.MaxRatio = 2.5
	srv2 := &api.Server{Store: s, MatchParams: detect.DefaultMatchParams(), RoadMatch: &loose}
	ts2 := httptest.NewServer(api.HandlerFromMux(api.NewStrictHandler(srv2, []api.StrictMiddlewareFunc{srv2.AuthMiddleware}), http.NewServeMux()))
	defer ts2.Close()
	var wide api.Journey
	fetchJSON(t, fmt.Sprintf("%s/candidates/%d/journey", ts2.URL, list.Candidates[0].Id), &wide)
	if wide.MatchParams.MaxRatio != 2.5 {
		t.Errorf("match_params.max_ratio = %v, want the server's 2.5", wide.MatchParams.MaxRatio)
	}
	if wide.Legs[target].SnappedPoints == nil {
		t.Error("with the guard at 2.5 the row at 2.0 times was not drawn")
	}
}
