package photo_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"roadbook/internal/photo"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "photos", name))
	if err != nil {
		t.Fatalf("reading fixture %s: %v (regenerate with: go run ./testdata/photos/gen)", name, err)
	}
	return data
}

const coordTol = 1e-6 // rationals are exact hundredths of arcseconds; 1e-6° ≈ 0.1 m

func TestExtractEXIFFull(t *testing.T) {
	m := photo.ExtractEXIF(fixture(t, "gps_full.jpg"))

	if m.Pos == nil {
		t.Fatal("no position extracted")
	}
	if diff := m.Pos.Lat - 12.3456; diff > coordTol || diff < -coordTol {
		t.Errorf("lat = %v, want 12.3456", m.Pos.Lat)
	}
	if diff := m.Pos.Lon - 45.6789; diff > coordTol || diff < -coordTol {
		t.Errorf("lon = %v, want 45.6789", m.Pos.Lon)
	}
	if m.PosSource != photo.PosEXIF {
		t.Errorf("pos source = %q, want exif", m.PosSource)
	}
	if m.Orientation != 6 {
		t.Errorf("orientation = %d, want 6", m.Orientation)
	}
	if m.Wall == nil || *m.Wall != (photo.CivilTime{Year: 2026, Month: time.July, Day: 27, Hour: 21, Min: 15, Sec: 3}) {
		t.Errorf("wall = %+v, want 2026-07-27 21:15:03", m.Wall)
	}
	if m.WallOffsetSec == nil || *m.WallOffsetSec != 19800 {
		t.Errorf("wall offset = %v, want 19800 (+05:30)", m.WallOffsetSec)
	}
	wantGPS := time.Date(2026, 7, 27, 15, 45, 3, 0, time.UTC)
	if m.GPSTime == nil || !m.GPSTime.Equal(wantGPS) {
		t.Errorf("gps time = %v, want %v", m.GPSTime, wantGPS)
	}
}

func TestExtractEXIFBigEndian(t *testing.T) {
	m := photo.ExtractEXIF(fixture(t, "bigendian.jpg"))
	if m.Pos == nil {
		t.Fatal("no position extracted from big-endian TIFF")
	}
	if diff := m.Pos.Lat - (-12.3456); diff > coordTol || diff < -coordTol {
		t.Errorf("lat = %v, want -12.3456 (S ref negates)", m.Pos.Lat)
	}
	if diff := m.Pos.Lon - (-45.6789); diff > coordTol || diff < -coordTol {
		t.Errorf("lon = %v, want -45.6789 (W ref negates)", m.Pos.Lon)
	}
	if m.GPSTime != nil {
		t.Errorf("gps time = %v, want nil (fixture has no date/time stamps)", m.GPSTime)
	}
	if m.Orientation != 3 {
		t.Errorf("orientation = %d, want 3 — the fixture writes the tag LONG-typed (the Xiaomi field variant), which must parse like the specified SHORT", m.Orientation)
	}
}

func TestExtractEXIFPartialAndMalformed(t *testing.T) {
	cases := []struct {
		name     string
		wantWall bool
		wantPos  bool
	}{
		{"wall_only.jpg", true, false},
		{"offset_time.jpg", true, false},
		{"no_meta.jpg", false, false},
		// Malformation is absence, never a crash: a zero-denominator
		// rational voids the position but not the valid time beside it;
		// truncation and wild offsets void everything.
		{"zero_denom.jpg", true, false},
		{"trunc_app1.jpg", false, false},
		{"bad_ifd_offset.jpg", false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := photo.ExtractEXIF(fixture(t, tc.name))
			if got := m.Wall != nil; got != tc.wantWall {
				t.Errorf("wall present = %v, want %v", got, tc.wantWall)
			}
			if got := m.Pos != nil; got != tc.wantPos {
				t.Errorf("pos present = %v, want %v", got, tc.wantPos)
			}
		})
	}
}

// TestExtractEXIFNeverPanics feeds the parser every truncation of a valid
// file and every single-byte corruption of its metadata region. The parser's
// contract is that malformation yields absence — this is the test that the
// bounds-checking holds everywhere, not just on the malformations we thought
// to craft.
func TestExtractEXIFNeverPanics(t *testing.T) {
	data := fixture(t, "gps_full.jpg")
	for i := 0; i <= len(data); i++ {
		photo.ExtractEXIF(data[:i])
	}
	region := min(len(data), 700) // APP1 sits at the front
	for i := range region {
		for _, b := range []byte{0x00, 0xFF} {
			mut := append([]byte(nil), data...)
			mut[i] = b
			photo.ExtractEXIF(mut)
		}
	}
}
