package detect_test

// Regression tests against the reference detector's recorded output. The inputs
// and expected files live in data/, which is private and gitignored; on a clone
// without it these tests skip. The expected files are produced by the reference
// implementation itself (prototype/detect_fixture.py writes
// data/fixture-candidates.json; a copy of it pointed at the full archive writes
// data/archive-candidates.json), so this file compares Go against Python
// field-for-field without embedding any real value in committed code.

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"testing"
	"time"

	"roadbook/internal/detect"
	"roadbook/internal/domain"
	"roadbook/internal/timeline"
)

func TestFixtureRegression(t *testing.T) {
	compareWithReference(t,
		"../../data/fixture-2025-04.segments.timeline.json",
		"../../data/fixture-candidates.json")
}

func TestArchiveRegression(t *testing.T) {
	compareWithReference(t,
		"../../data/Timeline.json",
		"../../data/archive-candidates.json")
}

type expected struct {
	Params map[string]float64 `json:"params"`
	Bases  []struct {
		LL   []float64 `json:"ll"`
		From string    `json:"from"`
		To   string    `json:"to"`
		N    int       `json:"n"`
	} `json:"bases"`
	Candidates []struct {
		Start   string         `json:"start"`
		End     string         `json:"end"`
		Days    float64        `json:"days"`
		DestKm  int            `json:"dest_km"`
		TrackKm int            `json:"track_km"`
		Stops   int            `json:"stops"`
		Dest    []float64      `json:"dest"`
		Modes   map[string]int `json:"modes"`
		Repeat  int            `json:"repeat"`
	} `json:"candidates"`
}

func compareWithReference(t *testing.T, srcPath, expPath string) {
	t.Helper()
	src, err := os.ReadFile(srcPath)
	if errors.Is(err, fs.ErrNotExist) {
		t.Skipf("private dataset %s not present; skipping", srcPath)
	}
	if err != nil {
		t.Fatal(err)
	}
	expRaw, err := os.ReadFile(expPath)
	if errors.Is(err, fs.ErrNotExist) {
		t.Skipf("reference output %s not present; regenerate with the prototype", expPath)
	}
	if err != nil {
		t.Fatal(err)
	}
	var exp expected
	if err := json.Unmarshal(expRaw, &exp); err != nil {
		t.Fatal(err)
	}

	p := detect.DefaultParams()
	// Guard against a stale expected file: it must have been produced with the
	// same parameters we are about to run.
	for name, want := range map[string]float64{
		"NEAR_M": p.NearM, "FAR_KM": p.FarKm, "MIN_OBS": float64(p.MinObs),
		"MIN_HRS": p.MinHrs, "MIN_DWELL_MIN": p.MinDwellMin, "MAX_KMH": p.MaxKmh,
	} {
		if got, ok := exp.Params[name]; !ok || got != want {
			t.Fatalf("expected file %s was produced with %s=%v, defaults use %v — regenerate it",
				expPath, name, exp.Params[name], want)
		}
	}

	obs, _, err := timeline.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	res := detect.Run(obs, p)

	if len(res.Bases) != len(exp.Bases) {
		t.Fatalf("bases = %d, want %d", len(res.Bases), len(exp.Bases))
	}
	for i, want := range exp.Bases {
		got := res.Bases[i]
		if got.Center.Lat != want.LL[0] || got.Center.Lon != want.LL[1] {
			t.Errorf("base %d center = (%v,%v), want (%v,%v)", i, got.Center.Lat, got.Center.Lon, want.LL[0], want.LL[1])
		}
		if f := got.From.Format("2006-01-02"); f != want.From {
			t.Errorf("base %d from = %s, want %s", i, f, want.From)
		}
		if f := got.To.Format("2006-01-02"); f != want.To {
			t.Errorf("base %d to = %s, want %s", i, f, want.To)
		}
		if got.N != want.N {
			t.Errorf("base %d n = %d, want %d", i, got.N, want.N)
		}
	}

	if len(res.Candidates) != len(exp.Candidates) {
		t.Fatalf("candidates = %d, want %d", len(res.Candidates), len(exp.Candidates))
	}
	for i, want := range exp.Candidates {
		got := res.Candidates[i]
		if !got.Start.Equal(pyTime(t, want.Start)) || !got.End.Equal(pyTime(t, want.End)) {
			t.Errorf("candidate %d span = %v..%v, want %s..%s", i, got.Start, got.End, want.Start, want.End)
		}
		if got.Days != want.Days || got.DestKm != want.DestKm || got.TrackKm != want.TrackKm ||
			got.Stops != want.Stops || got.Repeat != want.Repeat {
			t.Errorf("candidate %d metrics = {days %v dest %d track %d stops %d rpt %d}, want {days %v dest %d track %d stops %d rpt %d}",
				i, got.Days, got.DestKm, got.TrackKm, got.Stops, got.Repeat,
				want.Days, want.DestKm, want.TrackKm, want.Stops, want.Repeat)
		}
		if got.Dest.Lat != want.Dest[0] || got.Dest.Lon != want.Dest[1] {
			t.Errorf("candidate %d dest = (%v,%v), want (%v,%v)", i, got.Dest.Lat, got.Dest.Lon, want.Dest[0], want.Dest[1])
		}
		gotModes := map[string]int{}
		for _, m := range got.Modes {
			gotModes[m.Mode] = m.N
		}
		if len(gotModes) != len(want.Modes) {
			t.Errorf("candidate %d modes = %v, want %v", i, gotModes, want.Modes)
		} else {
			for mode, n := range want.Modes {
				if gotModes[mode] != n {
					t.Errorf("candidate %d mode %s = %d, want %d", i, mode, gotModes[mode], n)
				}
			}
		}
	}
}

// pyTime parses Python's str(datetime) form, with and without microseconds.
func pyTime(t *testing.T, s string) time.Time {
	t.Helper()
	for _, layout := range []string{"2006-01-02 15:04:05-07:00", "2006-01-02 15:04:05.999999-07:00"} {
		if ts, err := time.Parse(layout, s); err == nil {
			return ts
		}
	}
	t.Fatalf("unparseable timestamp in expected file: %q", s)
	return time.Time{}
}

var _ = domain.LatLng{} // keep the import if the comparison helpers change
