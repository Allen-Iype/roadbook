package photo_test

import (
	"strings"
	"testing"

	"roadbook/internal/photo"
)

func TestSniff(t *testing.T) {
	cases := []struct {
		file     string
		wantKind photo.Kind // "" = rejected
		rejKind  string
	}{
		{"gps_full.jpg", photo.KindJPEG, ""},
		{"no_meta.jpg", photo.KindJPEG, ""},
		{"gps_full.jpg.json", photo.KindSidecar, ""},
		{"not_sidecar.json", photo.KindSidecar, ""}, // sniff says JSON; ParseSidecar gives the verdict
		{"sample.png", "", "png"},
		{"sample.heic", photo.KindHEIC, ""}, // accepted since phase 11: metadata extraction
		{"gps_full.heic", photo.KindHEIC, ""},
		{"trunc_meta.heic", photo.KindHEIC, ""}, // sniff says HEIF; extraction gives absence
		{"sample.mp4", "", "video"},
		{"sample.webp", "", "webp"},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			kind, rej := photo.Sniff(fixture(t, tc.file))
			if tc.wantKind != "" {
				if rej != nil {
					t.Fatalf("rejected %q: %s", rej.Kind, rej.Message)
				}
				if kind != tc.wantKind {
					t.Errorf("kind = %q, want %q", kind, tc.wantKind)
				}
				return
			}
			if rej == nil {
				t.Fatalf("accepted as %q, want rejection %q", kind, tc.rejKind)
			}
			if rej.Kind != tc.rejKind {
				t.Errorf("rejection kind = %q, want %q", rej.Kind, tc.rejKind)
			}
			if rej.Message == "" {
				t.Error("rejection carries no message; the taxonomy's point is actionable messages")
			}
		})
	}
}

func TestSniffSynthetic(t *testing.T) {
	cases := []struct {
		name    string
		data    []byte
		rejKind string
	}{
		{"empty", nil, "empty"},
		{"gzip", []byte{0x1f, 0x8b, 8, 0}, "gzip"},
		{"zip", []byte("PK\x03\x04...."), "zip"},
		{"pdf", []byte("%PDF-1.7"), "pdf"},
		{"html", []byte("<!doctype html><html>"), "html"},
		{"tiff-raw", []byte("II*\x00\x08\x00\x00\x00"), "tiff"},
		{"gif", []byte("GIF89a??"), "gif"},
		{"text", []byte("hello, not a photo"), "unrecognised"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, rej := photo.Sniff(tc.data)
			if rej == nil {
				t.Fatal("accepted, want rejection")
			}
			if rej.Kind != tc.rejKind {
				t.Errorf("kind = %q, want %q (message: %s)", rej.Kind, tc.rejKind, rej.Message)
			}
		})
	}
}

func TestSniffAVIFMessageIsActionable(t *testing.T) {
	// HEIC itself is accepted since phase 11; AVIF — same container family,
	// but a web re-encode — stays rejected, and the message must still say
	// the way forward.
	avif := []byte{0, 0, 0, 0x18}
	avif = append(avif, []byte("ftypavif")...)
	avif = append(avif, make([]byte, 12)...)
	_, rej := photo.Sniff(avif)
	if rej == nil || !strings.Contains(rej.Message, "JPEG") {
		t.Errorf("AVIF rejection must tell the user the way forward; got: %v", rej)
	}
}
