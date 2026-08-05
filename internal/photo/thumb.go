package photo

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"

	"golang.org/x/image/draw"
)

// Thumbnail parameters. Named, not magic numbers in the algorithm body —
// invariant 3's discipline, cheap here (BRIEF §1.2). 512 px is comfortably
// larger than any rendering on the page; quality 80 keeps files in the tens
// of kilobytes.
const (
	DefaultThumbMaxPx   = 512
	DefaultThumbQuality = 80
)

// Thumbnail decodes a JPEG, bakes the EXIF orientation into the pixels,
// scales the longest edge down to maxPx (never up), and re-encodes. The
// output is a fresh JPEG with no metadata block at all — re-encoding pixel
// data strips EXIF by construction, which is the privacy property BRIEF
// §1.2 relies on: the served file is pixels only, position lives in
// Postgres. Returns the encoded bytes and the final displayed dimensions
// (post-orientation).
//
// orientation is the EXIF value 1–8; 0 (absent) is treated as 1. Catmull-Rom
// is the slowest, highest-quality scaler in x/image/draw; for a batch of
// tens of photos it costs milliseconds per image.
func Thumbnail(data []byte, orientation, maxPx, quality int) ([]byte, int, int, error) {
	if maxPx <= 0 {
		return nil, 0, 0, fmt.Errorf("thumbnail: maxPx must be positive, got %d", maxPx)
	}
	src, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("thumbnail: decoding image: %w", err)
	}

	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w == 0 || h == 0 {
		return nil, 0, 0, fmt.Errorf("thumbnail: image has zero dimension")
	}

	// Scale so the longest edge is maxPx, preserving aspect; never upscale.
	scale := 1.0
	if longest := max(w, h); longest > maxPx {
		scale = float64(maxPx) / float64(longest)
	}
	dw := max(1, int(float64(w)*scale+0.5))
	dh := max(1, int(float64(h)*scale+0.5))

	scaled := image.NewRGBA(image.Rect(0, 0, dw, dh))
	draw.CatmullRom.Scale(scaled, scaled.Bounds(), src, b, draw.Src, nil)

	oriented := orient(scaled, orientation)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, oriented, &jpeg.Options{Quality: quality}); err != nil {
		return nil, 0, 0, fmt.Errorf("thumbnail: encoding: %w", err)
	}
	ob := oriented.Bounds()
	return buf.Bytes(), ob.Dx(), ob.Dy(), nil
}

// orient bakes an EXIF orientation into the pixels, because the pipeline
// strips metadata: a thumbnail with neither pixels rotated nor a surviving
// tag would render sideways forever (BRIEF §1.2). The eight EXIF values map
// to combinations of transpose, mirror, and rotation; each target pixel is
// filled from its source position. Runs on the already-scaled image, so the
// work is bounded by the thumbnail size, not the original.
func orient(src *image.RGBA, orientation int) *image.RGBA {
	if orientation <= 1 || orientation > 8 {
		return src
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()

	// Orientations 5–8 swap the axes.
	dw, dh := w, h
	if orientation >= 5 {
		dw, dh = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))

	for y := range h {
		for x := range w {
			var dx, dy int
			switch orientation {
			case 2: // mirror horizontal
				dx, dy = w-1-x, y
			case 3: // rotate 180°
				dx, dy = w-1-x, h-1-y
			case 4: // mirror vertical
				dx, dy = x, h-1-y
			case 5: // transpose (mirror + 90° CW)
				dx, dy = y, x
			case 6: // rotate 90° CW
				dx, dy = h-1-y, x
			case 7: // transverse (mirror + 270° CW)
				dx, dy = h-1-y, w-1-x
			case 8: // rotate 270° CW
				dx, dy = y, w-1-x
			}
			dst.SetRGBA(dx, dy, src.RGBAAt(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}
