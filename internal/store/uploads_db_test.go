package store_test

import (
	"context"
	"os"
	"testing"

	"roadbook/internal/store"
	"roadbook/internal/store/storetest"
)

// TestUploadImportBookkeeping covers the store half of the upload path
// (phase 7 BRIEF §1.2, migration 00009): the running row carries its
// content hash, the startup sweep finalises orphaned running rows as
// failed, and detect_status round-trips without touching status.
func TestUploadImportBookkeeping(t *testing.T) {
	s := storetest.Open(t)
	ctx := context.Background()

	id, err := s.BeginUploadImport(ctx, "Timeline.json", "cafe0000")
	if err != nil {
		t.Fatal(err)
	}
	row, err := s.GetImport(ctx, id)
	if err != nil || row == nil {
		t.Fatalf("GetImport: %v, %v", row, err)
	}
	if row.Status != "running" || row.ContentHash == nil || *row.ContentHash != "cafe0000" {
		t.Errorf("running row = %+v — want status running with its content hash", row)
	}
	if row.WindowStart != nil || row.WindowEnd != nil {
		t.Errorf("upload import has a window: %+v — uploads import the whole file", row)
	}

	// The sweep: a crash left this row running; startup marks it failed and
	// says why. A completed CLI-style row must be untouched.
	cliID, err := s.BeginImport(ctx, "cli.json", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.FailImport(ctx, cliID, "gzip", "already finalised"); err != nil {
		t.Fatal(err)
	}
	swept, err := s.SweepRunningImports(ctx, "interrupted by a server restart")
	if err != nil {
		t.Fatal(err)
	}
	if swept != 1 {
		t.Fatalf("swept %d rows, want exactly the one running row", swept)
	}
	row, _ = s.GetImport(ctx, id)
	if row.Status != "failed" || row.Error == nil || *row.Error != "interrupted by a server restart" {
		t.Errorf("swept row = %+v — want failed with the interruption message", row)
	}
	cli, _ := s.GetImport(ctx, cliID)
	if cli.Error == nil || *cli.Error != "already finalised" {
		t.Errorf("sweep touched an already-finalised row: %+v", cli)
	}

	// detect_status is its own channel: setting it never changes status.
	for _, st := range []string{"running", "completed"} {
		if err := s.SetImportDetectStatus(ctx, id, st); err != nil {
			t.Fatal(err)
		}
	}
	row, _ = s.GetImport(ctx, id)
	if row.DetectStatus == nil || *row.DetectStatus != "completed" {
		t.Errorf("detect_status = %v, want completed", row.DetectStatus)
	}
	if row.Status != "failed" {
		t.Errorf("status = %q — detect_status must never touch status", row.Status)
	}

	// GetImport for an id that never existed is nil, not an error.
	missing, err := s.GetImport(ctx, 999999)
	if err != nil || missing != nil {
		t.Errorf("missing import = %v, %v — want nil, nil", missing, err)
	}
}

// TestUploadFilesLifecycle pins the temp→promote→dedupe file contract
// (BRIEF §3C): promotion is a rename, identical bytes land on one file,
// and Remove tolerates absence.
func TestUploadFilesLifecycle(t *testing.T) {
	f := store.UploadFiles{Dir: t.TempDir()}
	if err := f.Init(); err != nil {
		t.Fatal(err)
	}

	write := func(content string) string {
		tmp, err := f.CreateTemp()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := tmp.WriteString(content); err != nil {
			t.Fatal(err)
		}
		tmp.Close()
		return tmp.Name()
	}

	p1 := write("export bytes")
	kept, err := f.Promote(p1, "aaaa")
	if err != nil {
		t.Fatal(err)
	}
	if kept != f.Path("aaaa") {
		t.Errorf("promoted to %q, want %q", kept, f.Path("aaaa"))
	}

	// Same hash again: the temp is discarded, the original file stands.
	p2 := write("export bytes")
	if _, err := f.Promote(p2, "aaaa"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Promote(write("other"), "bbbb"); err != nil {
		t.Fatal(err)
	}

	// Exactly the two retained files remain — no temp litter.
	entries, err := os.ReadDir(f.Dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}
		t.Errorf("uploads dir holds %v, want the two promoted files only", names)
	}

	if err := f.Remove("never-existed.tmp"); err != nil {
		t.Errorf("Remove of an absent file = %v, want nil — absence is the goal", err)
	}
}
