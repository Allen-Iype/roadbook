package journey_test

import (
	"testing"
	"time"

	"roadbook/internal/domain"
	"roadbook/internal/geo"
	"roadbook/internal/journey"
)

// The synthetic journey: an observed leg, a stop, a routed gap, an air gap,
// and an unknown gap, laid out on fabricated open-ocean coordinates along
// lat 12, lon 45.0→45.4 (~43 km east-west; 0.01° lat ≈ 1.11 km).
func placeFixture() journey.Journey {
	at := func(h, m int) time.Time {
		return time.Date(2026, 3, 1, h, m, 0, 0, time.UTC)
	}
	tp := func(h, m int, lat, lon float64) journey.TimedPoint {
		return journey.TimedPoint{Time: at(h, m), Loc: domain.LatLng{Lat: lat, Lon: lon}}
	}
	return journey.Journey{
		WindowStart: at(8, 0),
		WindowEnd:   at(16, 0),
		Legs: []journey.Leg{
			{ // observed: straight run along lat 12
				Kind: journey.LegObserved,
				Points: []journey.TimedPoint{
					tp(8, 0, 12, 45.00), tp(8, 30, 12, 45.05), tp(9, 0, 12, 45.10),
				},
			},
			{ // routed gap: endpoints at the chord, routed polyline bows north
				Kind: journey.LegGap, GapKind: journey.GapRoad,
				Points: []journey.TimedPoint{tp(9, 0, 12, 45.10), tp(10, 0, 12, 45.20)},
				RoutedPoints: []domain.LatLng{
					{Lat: 12, Lon: 45.10}, {Lat: 12.03, Lon: 45.15}, {Lat: 12, Lon: 45.20},
				},
			},
			{ // air gap
				Kind: journey.LegGap, GapKind: journey.GapAir,
				Points: []journey.TimedPoint{tp(10, 0, 12, 45.20), tp(11, 0, 12, 45.30)},
			},
			{ // unknown gap
				Kind: journey.LegGap, GapKind: journey.GapUnknown,
				Points: []journey.TimedPoint{tp(12, 0, 12, 45.30), tp(13, 0, 12, 45.40)},
			},
		},
		Stops: []journey.Stop{
			// The halt at 11:00–12:00 sits between the air and unknown legs.
			{Start: at(11, 0), End: at(12, 0), Loc: domain.LatLng{Lat: 12, Lon: 45.30}},
		},
	}
}

func TestPlacePhoto(t *testing.T) {
	j := placeFixture()
	at := func(h, m int) time.Time { return time.Date(2026, 3, 1, h, m, 0, 0, time.UTC) }
	warn := journey.DefaultPhotoFarWarnM

	cases := []struct {
		name     string
		t        time.Time
		pos      domain.LatLng
		wantOK   bool
		wantKind journey.PlaceKind
		wantFlag bool
	}{
		{"on the observed track", at(8, 15), domain.LatLng{Lat: 12, Lon: 45.03}, true, journey.PlaceObserved, false},
		{"near the observed track", at(8, 15), domain.LatLng{Lat: 12.005, Lon: 45.03}, true, journey.PlaceObserved, false},  // ~550 m
		{"far off the observed track", at(8, 15), domain.LatLng{Lat: 12.05, Lon: 45.03}, true, journey.PlaceObserved, true}, // ~5.5 km
		// Against the routed gap the distance is to the ROUTED polyline, not
		// the chord: a photo on the bowed road is close, one on the chord
		// midpoint is ~2 km from the bow but must measure against the road.
		{"on the routed road", at(9, 30), domain.LatLng{Lat: 12.03, Lon: 45.15}, true, journey.PlaceRoad, false},
		{"far south of the routed road", at(9, 30), domain.LatLng{Lat: 11.97, Lon: 45.15}, true, journey.PlaceRoad, true},
		{"during the air leg — never flagged", at(10, 30), domain.LatLng{Lat: 20, Lon: 60}, true, journey.PlaceAir, false},
		{"during the stop, at it", at(11, 30), domain.LatLng{Lat: 12.001, Lon: 45.30}, true, journey.PlaceStop, false},
		{"during the stop, far away", at(11, 30), domain.LatLng{Lat: 12.1, Lon: 45.30}, true, journey.PlaceStop, true},
		{"during the unknown gap, on the chord", at(12, 30), domain.LatLng{Lat: 12, Lon: 45.35}, true, journey.PlaceUnknown, false},
		{"before the journey", at(7, 0), domain.LatLng{Lat: 12, Lon: 45}, false, "", false},
		{"in the 11:00 boundary the stop wins over the air leg", at(11, 0), domain.LatLng{Lat: 12, Lon: 45.30}, true, journey.PlaceStop, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, ok := journey.PlacePhoto(j, tc.t, tc.pos, warn)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v (placement %+v)", ok, tc.wantOK, p)
			}
			if !ok {
				return
			}
			if p.Kind != tc.wantKind {
				t.Errorf("kind = %q, want %q", p.Kind, tc.wantKind)
			}
			if p.Flagged != tc.wantFlag {
				t.Errorf("flagged = %v (distance %.0f m), want %v", p.Flagged, p.DistanceM, tc.wantFlag)
			}
			if p.Kind == journey.PlaceAir && p.HasDistance {
				t.Error("air placement carries a distance — §3G excludes it")
			}
			if p.Kind != journey.PlaceAir && !p.HasDistance {
				t.Error("non-air placement carries no distance")
			}
		})
	}
}

func TestPointToPolylineM(t *testing.T) {
	line := []domain.LatLng{{Lat: 12, Lon: 45.0}, {Lat: 12, Lon: 45.1}}
	// A point due north of the segment's midpoint: distance is the
	// perpendicular, ~1.11 km per 0.01° of latitude.
	d := geo.PointToPolylineM(domain.LatLng{Lat: 12.01, Lon: 45.05}, line)
	if d < 1050 || d > 1180 {
		t.Errorf("perpendicular distance = %.0f m, want ≈1112 m", d)
	}
	// A point beyond the end measures to the endpoint, not the infinite line.
	end := geo.PointToPolylineM(domain.LatLng{Lat: 12, Lon: 45.2}, line)
	want := geo.HaversineM(domain.LatLng{Lat: 12, Lon: 45.2}, domain.LatLng{Lat: 12, Lon: 45.1})
	if diff := end - want; diff > 1 || diff < -1 {
		t.Errorf("beyond-end distance = %.1f m, want endpoint distance %.1f m", end, want)
	}
	// Single-point polyline.
	single := geo.PointToPolylineM(domain.LatLng{Lat: 12.01, Lon: 45}, []domain.LatLng{{Lat: 12, Lon: 45}})
	if single < 1050 || single > 1180 {
		t.Errorf("single-point distance = %.0f m, want ≈1112 m", single)
	}
}
