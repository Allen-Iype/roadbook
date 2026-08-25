package photo_test

import (
	"reflect"
	"testing"

	"roadbook/internal/photo"
)

// The HEIC walk must read exactly what the JPEG walk reads: gps_full.heic
// wraps the same TIFF blob as gps_full.jpg, so the two Metas must be
// identical field for field.
func TestExtractHEIFMatchesJPEG(t *testing.T) {
	fromJPEG := photo.ExtractEXIF(fixture(t, "gps_full.jpg"))
	fromHEIC := photo.ExtractHEIF(fixture(t, "gps_full.heic"))
	if !reflect.DeepEqual(fromJPEG, fromHEIC) {
		t.Errorf("HEIC meta differs from JPEG meta:\n jpeg: %+v\n heic: %+v", fromJPEG, fromHEIC)
	}
	if fromHEIC.Pos == nil || fromHEIC.GPSTime == nil {
		t.Fatal("gps_full.heic must yield position and GPS time")
	}
}

// Malformation is absence, never a crash — the exif.go posture, held by the
// box walk: a bare ftyp stub, a meta box declaring a size past EOF, and
// every truncation of a valid file must all return quietly.
func TestExtractHEIFMalformation(t *testing.T) {
	empty := photo.Meta{}
	if got := photo.ExtractHEIF(fixture(t, "sample.heic")); !reflect.DeepEqual(got, empty) {
		t.Errorf("bare ftyp stub yielded %+v, want absence", got)
	}
	if got := photo.ExtractHEIF(fixture(t, "trunc_meta.heic")); !reflect.DeepEqual(got, empty) {
		t.Errorf("truncated meta box yielded %+v, want absence", got)
	}

	valid := fixture(t, "gps_full.heic")
	for n := range valid {
		photo.ExtractHEIF(valid[:n]) // must not panic; any result is fine
	}
	// Corrupt every byte in turn — same sweep the JPEG parser survives.
	for i := range valid {
		mutated := make([]byte, len(valid))
		copy(mutated, valid)
		mutated[i] ^= 0xFF
		photo.ExtractHEIF(mutated)
	}
}
