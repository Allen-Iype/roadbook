package photosource

import (
	"os"
	"path/filepath"
	"testing"

	"roadbook/internal/domain"
)

// The committed phase-4 fixtures exercise every verdict: they were designed
// to cover the EXIF/sidecar matrix, and the batch parser must classify each
// one exactly.
func loadFixtures(t *testing.T) []File {
	t.Helper()
	dir := filepath.Join("..", "..", "testdata", "photos")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var files []File
	for _, e := range entries {
		if e.IsDir() { // skip the generator source
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, File{Name: e.Name(), Data: data})
	}
	return files
}

func TestParseFilesFixtureMatrix(t *testing.T) {
	obs, results, st := ParseFiles(loadFixtures(t))

	want := map[string]Verdict{
		"bad_ifd_offset.jpg": VerdictNoPosition,
		"bigendian.jpg":      VerdictNoTime,
		"gps_full.jpg":       VerdictFix,
		"gps_full.heic":      VerdictFix, // the same TIFF via the HEIF walk
		"gps_full.jpg.json":  VerdictSidecarPaired,
		"no_meta.jpg":        VerdictNoPosition,
		"not_sidecar.json":   VerdictUnsupported,
		"offset_time.jpg":    VerdictNoPosition,
		"sample.heic":        VerdictNoPosition, // accepted HEIF, no metadata inside
		"sample.mp4":         VerdictUnsupported,
		"sample.png":         VerdictUnsupported,
		"sample.webp":        VerdictUnsupported,
		"trunc_app1.jpg":     VerdictNoPosition,
		"trunc_meta.heic":    VerdictNoPosition, // malformation is absence
		"wall_only.jpg":      VerdictFix,        // position and instant arrive via its sidecar
		"wall_only.jpg.json": VerdictSidecarPaired,
		"zero_denom.jpg":     VerdictNoPosition,
		"zero_geo.jpg.supplemental-metadata.json": VerdictSidecarUnpaired,
	}
	if len(results) != len(want) {
		t.Fatalf("got %d results, want %d — fixture set changed; update this table", len(results), len(want))
	}
	for _, r := range results {
		w, ok := want[r.Name]
		if !ok {
			t.Errorf("unexpected file %s", r.Name)
			continue
		}
		if r.Verdict != w {
			t.Errorf("%s: verdict %s, want %s (%s)", r.Name, r.Verdict, w, r.Message)
		}
		if r.Verdict == VerdictUnsupported && r.Message == "" {
			t.Errorf("%s: unsupported with no actionable message", r.Name)
		}
	}

	if st != (Stats{Photos: 11, Fixes: 3, NoPosition: 7, NoTime: 1,
		SidecarsPaired: 2, SidecarsUnpaired: 1, Unsupported: 4}) {
		t.Errorf("stats = %+v", st)
	}

	// Fixes emit only raw positions — invariant 4: nothing photo-shaped
	// escapes, and no other observation class is invented.
	if len(obs.Visits)+len(obs.Activities)+len(obs.Points) != 0 {
		t.Error("photo batch emitted non-fix observations")
	}
	if len(obs.RawPositions) != 3 {
		t.Fatalf("got %d fixes, want 3", len(obs.RawPositions))
	}
	for _, rp := range obs.RawPositions {
		if rp.Source != domain.SourcePhoto {
			t.Errorf("fix source = %q, want %q", rp.Source, domain.SourcePhoto)
		}
		if rp.Loc == nil {
			t.Error("fix with nil location")
		}
	}
}

func TestParseFilesGPSFixDetail(t *testing.T) {
	obs, _, _ := ParseFiles(loadFixtures(t))
	// gps_full.jpg: the GPS clock gives the instant (15:45:03 UTC, the
	// sidecar's photoTakenTime agrees: 1785167103) and the wall−GPS
	// derivation gives +05:30 — the fix's civil zone must carry it so
	// downstream civil-date logic works in the photo's local calendar.
	var got *domain.RawPosition
	for i := range obs.RawPositions {
		if obs.RawPositions[i].Loc.Lat > 12.3 && obs.RawPositions[i].Loc.Lat < 12.4 &&
			obs.RawPositions[i].Loc.Lon > 45.6 && obs.RawPositions[i].Loc.Lon < 45.7 {
			if obs.RawPositions[i].Time.Unix() == 1785167103 {
				got = &obs.RawPositions[i]
			}
		}
	}
	if got == nil {
		t.Fatal("gps_full fix not found")
	}
	if _, off := got.Time.Zone(); off != 5*3600+1800 {
		t.Errorf("fix zone offset = %d, want +05:30", off)
	}
}
