package photo_test

import (
	"bytes"
	"image/jpeg"
	"testing"

	"roadbook/internal/photo"
)

func TestThumbnailScalesAndOrients(t *testing.T) {
	// The fixture is 64×48 with orientation 6 (rotate 90° CW): the longest
	// edge scales 64→32 giving 32×24, and orientation swaps it to 24×32.
	data := fixture(t, "gps_full.jpg")
	thumb, w, h, err := photo.Thumbnail(data, 6, 32, photo.DefaultThumbQuality)
	if err != nil {
		t.Fatal(err)
	}
	if w != 24 || h != 32 {
		t.Errorf("dims = %d×%d, want 24×32 (scaled then rotated)", w, h)
	}
	img, err := jpeg.Decode(bytes.NewReader(thumb))
	if err != nil {
		t.Fatalf("thumbnail does not decode: %v", err)
	}
	if b := img.Bounds(); b.Dx() != w || b.Dy() != h {
		t.Errorf("encoded dims %d×%d disagree with reported %d×%d", b.Dx(), b.Dy(), w, h)
	}
}

func TestThumbnailRotationMovesPixels(t *testing.T) {
	// The base gradient brightens green downward. After a 90° CW rotation,
	// the source's bottom-left (bright green) lands top-left, and the
	// source's top-left (dark) lands top-right — distinguishable even
	// through JPEG loss.
	data := fixture(t, "no_meta.jpg")
	thumb, _, _, err := photo.Thumbnail(data, 6, 32, 90)
	if err != nil {
		t.Fatal(err)
	}
	img, err := jpeg.Decode(bytes.NewReader(thumb))
	if err != nil {
		t.Fatal(err)
	}
	_, gTL, _, _ := img.At(1, 1).RGBA()
	_, gTR, _, _ := img.At(img.Bounds().Dx()-2, 1).RGBA()
	if gTL <= gTR+0x2000 {
		t.Errorf("top-left green %d not clearly brighter than top-right %d — rotation did not happen", gTL, gTR)
	}
}

func TestThumbnailNeverUpscales(t *testing.T) {
	data := fixture(t, "no_meta.jpg") // 64×48
	_, w, h, err := photo.Thumbnail(data, 0, 4096, photo.DefaultThumbQuality)
	if err != nil {
		t.Fatal(err)
	}
	if w != 64 || h != 48 {
		t.Errorf("dims = %d×%d, want the original 64×48 (no upscaling)", w, h)
	}
}

func TestThumbnailStripsMetadata(t *testing.T) {
	data := fixture(t, "gps_full.jpg")
	if !bytes.Contains(data, []byte("Exif\x00\x00")) {
		t.Fatal("fixture unexpectedly has no EXIF to strip")
	}
	thumb, _, _, err := photo.Thumbnail(data, 1, 32, photo.DefaultThumbQuality)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(thumb, []byte("Exif\x00\x00")) {
		t.Error("thumbnail still contains an EXIF block — the privacy property of BRIEF §1.2 is broken")
	}
}

func TestThumbnailRejectsNonImage(t *testing.T) {
	if _, _, _, err := photo.Thumbnail([]byte("not a jpeg"), 0, 32, 80); err == nil {
		t.Error("want an error for undecodable input")
	}
}
