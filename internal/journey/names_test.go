package journey

import (
	"reflect"
	"testing"
	"time"

	"roadbook/internal/domain"
)

func TestStopPositions(t *testing.T) {
	at := func(h, m int) time.Time { return time.Date(2026, 5, 2, h, m, 0, 0, time.UTC) }
	loc := func(lat, lon float64) *domain.LatLng { return &domain.LatLng{Lat: lat, Lon: lon} }

	j := Journey{Stops: []Stop{
		// 0: has its own points — visits are ignored even when one overlaps.
		{Start: at(8, 0), End: at(9, 0), Loc: domain.LatLng{Lat: 1, Lon: 1}, Points: 4},
		// 1: no points; two visits overlap, the longer overlap wins.
		{Start: at(10, 0), End: at(12, 0)},
		// 2: no points; only an unlocated visit overlaps — nothing to name from.
		{Start: at(13, 0), End: at(14, 0)},
		// 3: no points; a visit that merely touches the stop's edge is not an overlap.
		{Start: at(15, 0), End: at(16, 0)},
		// 4: no points; two visits overlap equally — the earlier one wins.
		{Start: at(18, 0), End: at(20, 0)},
	}}
	visits := []domain.Visit{
		{Start: at(7, 30), End: at(9, 30), Loc: loc(50, 50)}, // over stop 0
		{Start: at(9, 45), End: at(10, 20), Loc: loc(2, 2)},  // 20 min of stop 1
		{Start: at(10, 30), End: at(12, 30), Loc: loc(3, 3)}, // 90 min of stop 1
		{Start: at(13, 0), End: at(14, 0), Loc: nil},         // stop 2, no coordinate
		{Start: at(16, 0), End: at(17, 0), Loc: loc(4, 4)},   // touches stop 3's end
		{Start: at(19, 0), End: at(20, 30), Loc: loc(6, 6)},  // 60 min of stop 4, later
		{Start: at(17, 30), End: at(19, 0), Loc: loc(5, 5)},  // 60 min of stop 4, earlier
	}
	before := append([]domain.Visit(nil), visits...)

	got := StopPositions(j, visits)
	want := []StopPosition{
		{Loc: domain.LatLng{Lat: 1, Lon: 1}, Kind: PositionRecorded, OK: true},
		{Loc: domain.LatLng{Lat: 3, Lon: 3}, Kind: PositionVisit, OK: true},
		{},
		{},
		{Loc: domain.LatLng{Lat: 5, Lon: 5}, Kind: PositionVisit, OK: true},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("StopPositions =\n %+v\nwant\n %+v", got, want)
	}
	if !reflect.DeepEqual(visits, before) {
		t.Fatal("StopPositions altered its input")
	}

	// Input order must not matter (the tie rule is by time, not position).
	rev := make([]domain.Visit, len(visits))
	for i, v := range visits {
		rev[len(visits)-1-i] = v
	}
	if got := StopPositions(j, rev); !reflect.DeepEqual(got, want) {
		t.Fatalf("reversed visits changed the answer: %+v", got)
	}
}

func TestNameStops(t *testing.T) {
	p := PlaceParams{NearKm: 10, MaxKm: 25}
	got := NameStops([]NearestPlace{
		{Found: true, Name: "Alpha", Source: "osm", DistanceKm: 0.4},
		{Found: true, Name: "Beta", Source: "osm", DistanceKm: 10},    // the near edge is near
		{Found: true, Name: "Gamma", Source: "osm", DistanceKm: 10.1}, // just past it: "N km from"
		{Found: true, Name: "Delta", Source: "osm", DistanceKm: 25},   // the far edge is still named
		{Found: true, Name: "Epsilon", Source: "osm", DistanceKm: 25.1},
		{}, // no position, or an empty table
	}, p)
	want := []*StopName{
		{Name: "Alpha", Source: "osm", DistanceKm: 0.4, Near: true},
		{Name: "Beta", Source: "osm", DistanceKm: 10, Near: true},
		{Name: "Gamma", Source: "osm", DistanceKm: 10.1, Near: false},
		{Name: "Delta", Source: "osm", DistanceKm: 25, Near: false},
		nil,
		nil,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NameStops = %+v, want %+v", got, want)
	}
	if d := DefaultPlaceParams(); d.NearKm != 10 || d.MaxKm != 25 {
		t.Errorf("default place parameters moved: %+v", d)
	}
}
