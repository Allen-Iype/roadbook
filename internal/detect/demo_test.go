package detect_test

import (
	"os"
	"testing"

	"roadbook/internal/detect"
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
