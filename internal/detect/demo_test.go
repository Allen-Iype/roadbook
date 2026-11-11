package detect_test

import (
	"os"
	"reflect"
	"testing"
	"time"

	"roadbook/internal/detect"
	"roadbook/internal/domain"
	"roadbook/internal/timeline"
)

// TestDemoDataset pins detection over the committed demo (phase 5 BRIEF
// §3C) — the only dataset whose numbers the README may state (invariant
// 13), so this test is what makes those numbers regression-held. Ungated:
// the demo is public, committed bytes; reproduce with
//
//	go run ./testdata/demo/gen
//	roadbook detect -src testdata/demo/demo.json
func TestDemoDataset(t *testing.T) {
	f, err := os.Open("../../testdata/demo/demo.json")
	if err != nil {
		t.Fatalf("the demo dataset is committed and must exist: %v", err)
	}
	defer f.Close()

	obs, st, err := timeline.Parse(f)
	if err != nil {
		t.Fatal(err)
	}
	if st.Visits != 253 || st.Activities != 166 || st.Points != 301 || st.RawPositions != 0 || st.Skipped != 0 {
		t.Errorf("parse counts = %d/%d/%d/%d (%d skipped), want 253/166/301/0 (0 skipped)",
			st.Visits, st.Activities, st.Points, st.RawPositions, st.Skipped)
	}

	res := detect.Run(obs, detect.DefaultParams())
	if len(res.Bases) != 1 {
		t.Fatalf("home bases = %d, want 1 (Reykjavík)", len(res.Bases))
	}
	if res.OutliersDropped != 0 {
		t.Errorf("outliers dropped = %d, want 0", res.OutliersDropped)
	}
	if len(res.Candidates) != 3 {
		t.Fatalf("candidates = %d, want 3 (Höfn, Ísafjörður, Akureyri) — the Borgarnes day trip and the commute noise must not appear", len(res.Candidates))
	}

	// The three scripted adventures, by farthest dwelt destination; the
	// Borgarnes day trip (44 km, under FAR) proves absence by the count
	// above.
	wantDest := []struct {
		date string
		km   int
	}{
		{"2026-04-24", 325}, // south coast to Höfn — dense
		{"2026-05-22", 222}, // Ísafjörður — sparse
		{"2026-06-12", 249}, // Akureyri — by air
	}
	for i, w := range wantDest {
		c := res.Candidates[i]
		if got := c.Start.Format("2006-01-02"); got != w.date {
			t.Errorf("candidate %d start = %s, want %s", i+1, got, w.date)
		}
		if c.DestKm != w.km {
			t.Errorf("candidate %d dest_km = %d, want %d", i+1, c.DestKm, w.km)
		}
		if c.StartTruncated || c.EndTruncated {
			t.Errorf("candidate %d truncated (%v/%v) — the demo window fully contains its journeys", i+1, c.StartTruncated, c.EndTruncated)
		}
	}
}

// TestDemoDatasetIOS holds the iPhone-shaped copy of the demo (phase 17,
// issue #4; written by the same generator) to the Android file's result.
// Visits and activities must parse identically; path points carry whole-
// minute offsets on iPhone, so each may sit up to 30 s from its twin, and
// detection must still find the same bases, the same three trips, the same
// destinations and stops.
func TestDemoDatasetIOS(t *testing.T) {
	parse := func(path string) (domain.Observations, timeline.Stats) {
		t.Helper()
		f, err := os.Open(path)
		if err != nil {
			t.Fatalf("the demo datasets are committed and must exist: %v", err)
		}
		defer f.Close()
		obs, st, err := timeline.Parse(f)
		if err != nil {
			t.Fatal(err)
		}
		return obs, st
	}
	android, stA := parse("../../testdata/demo/demo.json")
	ios, stI := parse("../../testdata/demo/demo-ios.json")

	if stI.Format != timeline.PhoneTimelineIOSFormat || stA.Format != timeline.PhoneTimelineFormat {
		t.Errorf("formats = %q / %q", stA.Format, stI.Format)
	}
	if stI.Visits != stA.Visits || stI.Activities != stA.Activities || stI.Points != stA.Points || stI.Skipped != 0 {
		t.Fatalf("iPhone counts = %d/%d/%d (%d skipped), Android %d/%d/%d",
			stI.Visits, stI.Activities, stI.Points, stI.Skipped, stA.Visits, stA.Activities, stA.Points)
	}
	if !reflect.DeepEqual(ios.Visits, android.Visits) {
		t.Error("visits differ between the iPhone and Android shapes")
	}
	if !reflect.DeepEqual(ios.Activities, android.Activities) {
		t.Error("activities differ between the iPhone and Android shapes")
	}
	for i := range android.Points {
		a, b := android.Points[i], ios.Points[i]
		if d := a.Time.Sub(b.Time); d > 30*time.Second || d < -30*time.Second || !reflect.DeepEqual(a.Loc, b.Loc) {
			t.Fatalf("point %d: Android %v %v, iPhone %v %v", i, a.Time, a.Loc, b.Time, b.Loc)
		}
	}

	p := detect.DefaultParams()
	ra, ri := detect.Run(android, p), detect.Run(ios, p)
	if len(ri.Bases) != len(ra.Bases) || ri.OutliersDropped != ra.OutliersDropped || len(ri.Candidates) != len(ra.Candidates) {
		t.Fatalf("iPhone bases/outliers/candidates = %d/%d/%d, Android %d/%d/%d",
			len(ri.Bases), ri.OutliersDropped, len(ri.Candidates), len(ra.Bases), ra.OutliersDropped, len(ra.Candidates))
	}
	for i := range ra.Candidates {
		a, b := ra.Candidates[i], ri.Candidates[i]
		if a.Start.Format("2006-01-02") != b.Start.Format("2006-01-02") || a.DestKm != b.DestKm || a.Stops != b.Stops || a.Dest != b.Dest {
			t.Errorf("candidate %d: Android {%s %d km, %d stops}, iPhone {%s %d km, %d stops}",
				i+1, a.Start.Format("2006-01-02"), a.DestKm, a.Stops, b.Start.Format("2006-01-02"), b.DestKm, b.Stops)
		}
	}
}
