package photo_test

import (
	"errors"
	"testing"
	"time"

	"roadbook/internal/photo"
)

func TestParseSidecarFull(t *testing.T) {
	s, err := photo.ParseSidecar(fixture(t, "gps_full.jpg.json"))
	if err != nil {
		t.Fatal(err)
	}
	if s.Title != "gps_full.jpg" {
		t.Errorf("title = %q", s.Title)
	}
	want := time.Date(2026, 7, 27, 15, 45, 3, 0, time.UTC)
	if s.TakenTime == nil || !s.TakenTime.Equal(want) {
		t.Errorf("taken = %v, want %v (photoTakenTime, not creationTime — the fixture's creationTime is a day later)", s.TakenTime, want)
	}
	if s.Pos == nil || s.Pos.Lat != 12.3450 || s.Pos.Lon != 45.6780 {
		t.Errorf("pos = %+v, want (12.3450, 45.6780)", s.Pos)
	}
}

func TestParseSidecarZeroGeoFallsBackToGeoDataExif(t *testing.T) {
	s, err := photo.ParseSidecar(fixture(t, "zero_geo.jpg.supplemental-metadata.json"))
	if err != nil {
		t.Fatal(err)
	}
	if s.Pos == nil || s.Pos.Lat != 12.3456 || s.Pos.Lon != 45.6789 {
		t.Errorf("pos = %+v, want geoDataExif's reading ((0,0) geoData means absent)", s.Pos)
	}
}

func TestParseSidecarRejectsForeignJSON(t *testing.T) {
	_, err := photo.ParseSidecar(fixture(t, "not_sidecar.json"))
	if !errors.Is(err, photo.ErrNotSidecar) {
		t.Errorf("err = %v, want ErrNotSidecar", err)
	}
}

func TestSidecarPairName(t *testing.T) {
	cases := []struct{ in, want string }{
		{"IMG_1234.jpg.json", "IMG_1234.jpg"},
		{"IMG_1234.jpg.supplemental-metadata.json", "IMG_1234.jpg"},
		// Takeout truncates long sidecar names at arbitrary points.
		{"very-long-photo-name.jpg.supplemental-metad.json", "very-long-photo-name.jpg"},
		{"holiday.jpeg.suppl.json", "holiday.jpeg"},
		{"metadata.json", ""},    // no embedded image name
		{"IMG_1234.jpg", ""},     // not a sidecar name at all
		{"print-order.json", ""}, // Takeout's own non-photo JSON
	}
	for _, tc := range cases {
		if got := photo.SidecarPairName(tc.in); got != tc.want {
			t.Errorf("SidecarPairName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
