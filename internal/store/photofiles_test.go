package store_test

import (
	"os"
	"path/filepath"
	"testing"

	"roadbook/internal/store"
)

// PhotoFiles needs no database: it is the filesystem half of the photos
// stratum, tested against a real temp directory.
func TestPhotoFiles(t *testing.T) {
	f := store.PhotoFiles{Dir: filepath.Join(t.TempDir(), "photos")}
	if err := f.Init(); err != nil {
		t.Fatal(err)
	}

	const hash = "abc123"
	data := []byte{0xFF, 0xD8, 0xFF, 0x01, 0x02}

	if err := f.WriteThumb(hash, data); err != nil {
		t.Fatal(err)
	}
	got, err := f.ReadThumb(hash)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(data) {
		t.Errorf("read back %v, want %v", got, data)
	}

	// Overwrite is idempotent — same hash means same bytes by construction.
	if err := f.WriteThumb(hash, data); err != nil {
		t.Errorf("overwrite: %v", err)
	}

	if err := f.DeleteThumb(hash); err != nil {
		t.Fatal(err)
	}
	if _, err := f.ReadThumb(hash); !os.IsNotExist(err) {
		t.Errorf("read after delete: %v, want not-exist", err)
	}
	// Deleting what is already gone is success: the row is authoritative and
	// already deleted; a second delete has nothing left to do.
	if err := f.DeleteThumb(hash); err != nil {
		t.Errorf("second delete: %v, want nil", err)
	}
}

func TestPhotoFilesInitCreatesNestedDir(t *testing.T) {
	f := store.PhotoFiles{Dir: filepath.Join(t.TempDir(), "a", "b", "photos")}
	if err := f.Init(); err != nil {
		t.Fatal(err)
	}
	if err := f.WriteThumb("h", []byte{1}); err != nil {
		t.Fatal(err)
	}
}
