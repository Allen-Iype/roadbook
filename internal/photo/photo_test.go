package photo_test

import (
	"testing"
	"time"

	"roadbook/internal/domain"
	"roadbook/internal/photo"
)

func TestResolveTimeLadder(t *testing.T) {
	utc := func(h, m, s int) *time.Time {
		t := time.Date(2026, 7, 27, h, m, s, 0, time.UTC)
		return &t
	}
	wall := &photo.CivilTime{Year: 2026, Month: time.July, Day: 27, Hour: 21, Min: 15, Sec: 3}
	offIST := 19800

	cases := []struct {
		name       string
		m          photo.Meta
		fallback   int
		haveFB     bool
		wantOK     bool
		wantSource photo.TimeSource
		wantTime   time.Time
		wantOffset int
	}{
		{
			name:       "gps wins, offset derived from wall minus gps",
			m:          photo.Meta{GPSTime: utc(15, 45, 3), Wall: wall},
			wantOK:     true,
			wantSource: photo.TimeGPS,
			wantTime:   *utc(15, 45, 3),
			wantOffset: 19800,
		},
		{
			name: "derivation beats the explicit tag when both exist",
			m: photo.Meta{GPSTime: utc(15, 45, 3), Wall: wall,
				WallOffsetSec: intp(7200)},
			wantOK: true, wantSource: photo.TimeGPS,
			wantTime: *utc(15, 45, 3), wantOffset: 19800,
		},
		{
			name:       "gps with no wall clock falls back to the explicit tag",
			m:          photo.Meta{GPSTime: utc(15, 45, 3), WallOffsetSec: intp(7200)},
			wantOK:     true,
			wantSource: photo.TimeGPS,
			wantTime:   *utc(15, 45, 3),
			wantOffset: 7200,
		},
		{
			name:       "wall plus explicit offset",
			m:          photo.Meta{Wall: wall, WallOffsetSec: &offIST},
			wantOK:     true,
			wantSource: photo.TimeEXIFOffset,
			wantTime:   *utc(15, 45, 3), // 21:15:03+05:30
			wantOffset: 19800,
		},
		{
			name:       "sidecar epoch when exif has no offset evidence",
			m:          photo.Meta{SidecarTime: utc(15, 45, 3), Wall: wall},
			wantOK:     true,
			wantSource: photo.TimeSidecar,
			wantTime:   *utc(15, 45, 3),
			wantOffset: 19800, // derived: wall − sidecar instant
		},
		{
			name:       "bare wall clock resolves only with the adventure fallback",
			m:          photo.Meta{Wall: wall},
			fallback:   19800,
			haveFB:     true,
			wantOK:     true,
			wantSource: photo.TimeEXIFLocal,
			wantTime:   *utc(15, 45, 3),
			wantOffset: 19800,
		},
		{
			name:   "bare wall clock without fallback is unplaced",
			m:      photo.Meta{Wall: wall},
			wantOK: false, wantSource: photo.TimeNone,
		},
		{
			name:   "nothing resolves to nothing",
			m:      photo.Meta{},
			wantOK: false, wantSource: photo.TimeNone,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := photo.ResolveTime(tc.m, tc.fallback, tc.haveFB)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if got.Source != tc.wantSource {
				t.Errorf("source = %q, want %q", got.Source, tc.wantSource)
			}
			if !tc.wantOK {
				return
			}
			if !got.Time.Equal(tc.wantTime) {
				t.Errorf("time = %v, want %v", got.Time, tc.wantTime)
			}
			if got.OffsetSec != tc.wantOffset {
				t.Errorf("offset = %d, want %d", got.OffsetSec, tc.wantOffset)
			}
		})
	}
}

func TestResolveTimeRejectsWildDerivedOffset(t *testing.T) {
	// A camera whose wall clock is a day off the GPS clock: the derivation
	// lands outside ±14 h and must be rejected, falling to the next source.
	wall := &photo.CivilTime{Year: 2026, Month: time.July, Day: 29, Hour: 21, Min: 15, Sec: 3}
	gps := time.Date(2026, 7, 27, 15, 45, 3, 0, time.UTC)
	got, ok := photo.ResolveTime(photo.Meta{GPSTime: &gps, Wall: wall, WallOffsetSec: intp(3600)}, 0, false)
	if !ok || got.Source != photo.TimeGPS {
		t.Fatalf("resolve = %+v ok=%v, want gps rung", got, ok)
	}
	if got.OffsetSec != 3600 {
		t.Errorf("offset = %d, want 3600 (wild derivation rejected, explicit tag used)", got.OffsetSec)
	}
}

func TestMergeSidecarPrecedence(t *testing.T) {
	exifPos := &domain.LatLng{Lat: 12.3456, Lon: 45.6789}
	sidecarPos := &domain.LatLng{Lat: 12.3450, Lon: 45.6780}
	taken := time.Date(2026, 7, 27, 15, 45, 3, 0, time.UTC)

	t.Run("exif position wins", func(t *testing.T) {
		m := photo.MergeSidecar(
			photo.Meta{Pos: exifPos, PosSource: photo.PosEXIF},
			photo.Sidecar{Pos: sidecarPos, TakenTime: &taken},
		)
		if m.Pos != exifPos || m.PosSource != photo.PosEXIF {
			t.Errorf("pos = %+v source %q, want the EXIF reading kept", m.Pos, m.PosSource)
		}
		if m.SidecarTime == nil || !m.SidecarTime.Equal(taken) {
			t.Errorf("sidecar time = %v, want %v joined regardless", m.SidecarTime, taken)
		}
	})

	t.Run("sidecar fills absence", func(t *testing.T) {
		m := photo.MergeSidecar(photo.Meta{}, photo.Sidecar{Pos: sidecarPos})
		if m.Pos != sidecarPos || m.PosSource != photo.PosSidecar {
			t.Errorf("pos = %+v source %q, want the sidecar fill", m.Pos, m.PosSource)
		}
	})
}

func intp(v int) *int { return &v }
