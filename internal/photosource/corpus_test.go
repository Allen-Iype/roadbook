package photosource

import (
	"os"
	"path/filepath"
	"testing"

	"roadbook/internal/detect"
)

// The corpus regression: the committed synthetic camera roll
// (testdata/photos/corpus, written by `go run ./testdata/photos/gen -corpus`)
// must parse and detect to exactly these pinned values with default
// parameters. This is the phase-11 analogue of the demo-dataset regression —
// ungated, so it runs on every `make test` with no private data anywhere.
func TestCorpusDetection(t *testing.T) {
	dir := filepath.Join("..", "..", "testdata", "photos", "corpus")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var files []File
	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, File{Name: e.Name(), Data: data})
	}

	obs, _, st := ParseFiles(files)
	if st != (Stats{Photos: 57, Fixes: 55, NoPosition: 2, SidecarsPaired: 1}) {
		t.Fatalf("corpus parse stats = %+v", st)
	}

	res := detect.Run(obs, detect.DefaultParams())

	if len(res.Bases) != 1 {
		t.Fatalf("bases = %d, want 1 (synthetic home evidence)", len(res.Bases))
	}
	b := res.Bases[0]
	if b.Center.Lat != 64.1467 || b.Center.Lon != -21.9427 || b.N != 16 {
		t.Errorf("home base = (%.6f, %.6f) n=%d, want Reykjavík (64.1467, -21.9427) n=16",
			b.Center.Lat, b.Center.Lon, b.N)
	}
	if res.OutliersDropped != 0 {
		t.Errorf("outliers dropped = %d, want 0", res.OutliersDropped)
	}
	if len(res.Candidates) != 2 {
		t.Fatalf("candidates = %d, want 2 (Akureyri, Höfn): %+v", len(res.Candidates), res.Candidates)
	}

	// Chronological: the sparse Akureyri weekend, then the dense Höfn trip.
	// Höfn's destination is Stokksnes — the farthest place *dwelt* (337 km),
	// not the hotel (332 km): the rule from bug 3, holding on photo data.
	// Its three stops: the Höfn evening stay, Stokksnes, and the morning
	// stay that only clears MinDwellMin because the sidecar-restored market
	// fix extends it to two hours — the sidecar path earning its keep.
	type pin struct {
		days             float64
		destKm, stops    int
		obsCount         int
		destLat, destLon float64
	}
	// The two HEIC shots land inside existing stays: one more observation in
	// each span, everything else — stays, destinations, days — unmoved.
	want := []pin{
		{days: 1.0, destKm: 249, stops: 1, obsCount: 9, destLat: 65.6835, destLon: -18.1001},
		{days: 2.1, destKm: 337, stops: 3, obsCount: 20, destLat: 64.2445, destLon: -14.9782},
	}
	for i, w := range want {
		c := res.Candidates[i]
		if c.Days != w.days || c.DestKm != w.destKm || c.Stops != w.stops || c.ObsCount != w.obsCount ||
			c.Dest.Lat != w.destLat || c.Dest.Lon != w.destLon {
			t.Errorf("candidate %d = days=%.1f dest=%d stops=%d obs=%d (%.4f, %.4f), want %+v",
				i, c.Days, c.DestKm, c.Stops, c.ObsCount, c.Dest.Lat, c.Dest.Lon, w)
		}
		if c.TrackKm != 0 {
			t.Errorf("candidate %d TrackKm = %d, want 0 — photo data has no activities", i, c.TrackKm)
		}
		if c.StartTruncated || c.EndTruncated {
			t.Errorf("candidate %d truncated (%v, %v) — home evenings bound both trips", i, c.StartTruncated, c.EndTruncated)
		}
	}
}
