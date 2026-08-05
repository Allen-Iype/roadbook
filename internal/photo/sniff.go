package photo

import (
	"bytes"
	"fmt"
)

// Kind is what Sniff decided an upload is. Only two kinds proceed; everything
// else is an UnsupportedError with an actionable message, extending the
// phase-1 input taxonomy to uploads (BRIEF §3C).
type Kind string

const (
	KindJPEG    Kind = "jpeg"
	KindSidecar Kind = "sidecar" // a Takeout JSON sidecar, to be paired with an image
)

// UnsupportedError mirrors timeline.UnsupportedInputError: a machine-readable
// kind plus a message that tells the user what the file actually is and what
// to do instead.
type UnsupportedError struct {
	Kind    string
	Message string
}

func (e *UnsupportedError) Error() string { return e.Message }

// Sniff classifies an uploaded file by its magic bytes. It rejects only what
// is definitely something else; JPEG structure problems past the magic are
// the extractor's to survive (malformation is absence, never a crash).
func Sniff(data []byte) (Kind, *UnsupportedError) {
	reject := func(kind, what string) (Kind, *UnsupportedError) {
		return "", &UnsupportedError{Kind: kind, Message: what}
	}

	if len(data) < 4 {
		return reject("empty", "the file is empty or too short to be a photo")
	}

	switch {
	case bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF}):
		return KindJPEG, nil

	// ISO base media container (HEIC, MP4, MOV …): size box then "ftyp" and a
	// four-byte brand.
	case len(data) >= 12 && bytes.Equal(data[4:8], []byte("ftyp")):
		brand := string(data[8:12])
		switch brand {
		case "heic", "heix", "heim", "heis", "hevc", "hevx", "hevm", "hevs", "mif1", "msf1", "avif":
			return reject("heic", "HEIC/HEIF (the iPhone default format) is not supported — convert to JPEG, or download the photo from Google Photos, which serves JPEG")
		case "qt  ":
			return reject("video", "this is a video (QuickTime), which the product excludes — photos only")
		default:
			// mp42, isom, iso2, avc1, 3gp*, M4V … — all video containers.
			return reject("video", "this is a video, which the product excludes — photos only")
		}

	case bytes.HasPrefix(data, []byte("\x89PNG")):
		return reject("png", "PNG carries no camera position metadata in practice — screenshots and edited exports lose it; upload the original JPEG")
	case bytes.HasPrefix(data, []byte("GIF8")):
		return reject("gif", "this is a GIF, not a camera photo — upload the original JPEG")
	case len(data) >= 12 && bytes.HasPrefix(data, []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WEBP")):
		return reject("webp", "WebP is a re-encoded web format that has lost its camera metadata — upload the original JPEG")
	case bytes.HasPrefix(data, []byte("II*\x00")), bytes.HasPrefix(data, []byte("MM\x00*")):
		return reject("tiff", "this is a TIFF or camera RAW file — convert to JPEG (the camera or phone will have made one alongside it)")

	case bytes.HasPrefix(data, []byte{0x1f, 0x8b}):
		return reject("gzip", "this is a gzip archive, not a photo — decompress it and upload the photos inside")
	case bytes.HasPrefix(data, []byte("PK\x03\x04")):
		return reject("zip", "this is a ZIP archive (a Takeout download?) — extract it and upload the photos inside")
	case bytes.HasPrefix(data, []byte("%PDF")):
		return reject("pdf", "this is a PDF, not a photo")
	}

	trimmed := bytes.TrimLeft(data, " \t\r\n\xef\xbb\xbf")
	lower := bytes.ToLower(trimmed)
	switch {
	case bytes.HasPrefix(lower, []byte("<!doctype html")), bytes.HasPrefix(lower, []byte("<html")):
		return reject("html", "this is a web page, not a photo")
	case len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '['):
		return KindSidecar, nil
	}

	return reject("unrecognised", fmt.Sprintf("not a format Roadbook reads (the file starts with %q) — photos are JPEG, metadata sidecars are Takeout JSON", preview(trimmed)))
}

func preview(b []byte) string {
	n := min(len(b), 24)
	return string(b[:n])
}
