// Command gen regenerates every fixture in testdata/photos/. All coordinates
// and times are fabricated (12.3456°N 45.6789°E is open ocean); nothing here
// derives from real data — the data-safety rule for committed fixtures
// (docs/phase-4/BRIEF.md §4). Run from the repository root:
//
//	go run ./testdata/photos/gen
//
// The EXIF blocks are hand-assembled TIFF structures, which is the point:
// the parser's regression fixtures should exercise the exact byte layout the
// specification describes (both byte orders) plus the malformations the
// defensive paths exist for (truncated APP1, out-of-bounds IFD offset,
// zero-denominator rational).
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"time"
)

const outDir = "testdata/photos"

func main() {
	if _, err := os.Stat(outDir); err != nil {
		fmt.Fprintln(os.Stderr, "gen: run from the repository root:", err)
		os.Exit(1)
	}
	le, be := binary.ByteOrder(binary.LittleEndian), binary.ByteOrder(binary.BigEndian)

	// gps_full: every consumed tag, little-endian. Wall 21:15:03 at +05:30
	// against GPS 15:45:03 UTC — the pair the offset derivation resolves.
	writeJPEG("gps_full.jpg", buildTIFF(le, tiffSpec{
		orientation: 6,
		dateTime:    "2026:07:27 21:15:03",
		offsetTime:  "+05:30",
		gps: &gpsSpec{
			latRef: "N", lat: [3][2]uint32{{12, 1}, {20, 1}, {4416, 100}},
			lonRef: "E", lon: [3][2]uint32{{45, 1}, {40, 1}, {4404, 100}},
			dateStamp: "2026:07:27", timeStamp: [3][2]uint32{{15, 1}, {45, 1}, {3, 1}},
		},
	}))

	// bigendian: position only, MM byte order, southern/western hemispheres.
	// Orientation is deliberately LONG-typed, not the specified SHORT —
	// Xiaomi writes it that way in the field, and the parser must read both.
	writeJPEG("bigendian.jpg", buildTIFF(be, tiffSpec{
		orientation: 3, orientationLong: true,
		gps: &gpsSpec{
			latRef: "S", lat: [3][2]uint32{{12, 1}, {20, 1}, {4416, 100}},
			lonRef: "W", lon: [3][2]uint32{{45, 1}, {40, 1}, {4404, 100}},
		},
	}))

	// wall_only: a bare DateTimeOriginal — the pre-2016 Android common case.
	writeJPEG("wall_only.jpg", buildTIFF(le, tiffSpec{dateTime: "2026:07:27 09:30:00"}))

	// offset_time: wall clock plus explicit OffsetTimeOriginal, no GPS.
	writeJPEG("offset_time.jpg", buildTIFF(le, tiffSpec{
		dateTime: "2026:07:27 09:30:00", offsetTime: "+02:00",
	}))

	// no_meta: a JPEG with no APP1 at all.
	writeJPEG("no_meta.jpg", nil)

	// zero_denom: GPS rationals with zero denominators (position must be
	// dropped) beside a valid DateTimeOriginal (which must survive).
	writeJPEG("zero_denom.jpg", buildTIFF(le, tiffSpec{
		dateTime: "2026:07:27 11:00:00",
		gps: &gpsSpec{
			latRef: "N", lat: [3][2]uint32{{12, 0}, {20, 0}, {4416, 0}},
			lonRef: "E", lon: [3][2]uint32{{45, 0}, {40, 0}, {4404, 0}},
		},
	}))

	// trunc_app1: APP1 declares a length running past end of file.
	write("trunc_app1.jpg", []byte{
		0xFF, 0xD8, // SOI
		0xFF, 0xE1, 0xFF, 0xFF, // APP1, declared length 65535 — far past EOF
		'E', 'x', 'i', 'f', 0, 0,
		'I', 'I', 42, 0,
	})

	// bad_ifd_offset: valid TIFF header whose IFD0 offset points past EOF.
	badTIFF := &bytes.Buffer{}
	badTIFF.WriteString("II")
	binary.Write(badTIFF, le, uint16(42))
	binary.Write(badTIFF, le, uint32(0xFFFFFF00))
	writeJPEG("bad_ifd_offset.jpg", badTIFF.Bytes())

	// Sniff-taxonomy stubs: just enough magic bytes to classify.
	pngBuf := &bytes.Buffer{}
	png.Encode(pngBuf, image.NewRGBA(image.Rect(0, 0, 1, 1)))
	write("sample.png", pngBuf.Bytes())
	write("sample.heic", ftypStub("heic"))
	write("sample.mp4", ftypStub("isom"))
	write("sample.webp", append([]byte("RIFF\x10\x00\x00\x00WEBP"), []byte("VP8 ")...))

	// Sidecars. gps_full's sidecar carries the same fabricated instant and a
	// deliberately-nearby position, so tests can prove EXIF wins (§3D).
	utc := time.Date(2026, 7, 27, 15, 45, 3, 0, time.UTC).Unix()
	writeJSON("gps_full.jpg.json", map[string]any{
		"title":          "gps_full.jpg",
		"photoTakenTime": map[string]any{"timestamp": fmt.Sprint(utc), "formatted": "27 Jul 2026, 15:45:03 UTC"},
		"creationTime":   map[string]any{"timestamp": fmt.Sprint(utc + 86400)},
		"geoData":        map[string]any{"latitude": 12.3450, "longitude": 45.6780, "altitude": 0.0},
	})
	writeJSON("wall_only.jpg.json", map[string]any{
		"title":          "wall_only.jpg",
		"photoTakenTime": map[string]any{"timestamp": fmt.Sprint(time.Date(2026, 7, 27, 4, 0, 0, 0, time.UTC).Unix())},
		"geoData":        map[string]any{"latitude": 12.34, "longitude": 45.67},
	})
	// zero_geo: geoData zeroed, geoDataExif still holding the reading — the
	// fallback DECISIONS.md records. Newer-style truncatable sidecar name.
	writeJSON("zero_geo.jpg.supplemental-metadata.json", map[string]any{
		"title":       "zero_geo.jpg",
		"geoData":     map[string]any{"latitude": 0.0, "longitude": 0.0},
		"geoDataExif": map[string]any{"latitude": 12.3456, "longitude": 45.6789},
	})
	writeJSON("not_sidecar.json", map[string]any{"foo": 1})

	fmt.Println("gen: fixtures written to", outDir)
}

// baseImage is a 64×48 gradient — visibly asymmetric in both axes so
// orientation tests can tell rotation from reflection by pixel colour.
func baseImage() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 64, 48))
	for y := range 48 {
		for x := range 64 {
			img.SetRGBA(x, y, color.RGBA{R: uint8(x * 4), G: uint8(y * 5), B: 128, A: 255})
		}
	}
	return img
}

// writeJPEG encodes the base image and, when tiff is non-nil, splices an
// EXIF APP1 segment immediately after SOI — where cameras put it.
func writeJPEG(name string, tiff []byte) {
	buf := &bytes.Buffer{}
	if err := jpeg.Encode(buf, baseImage(), &jpeg.Options{Quality: 90}); err != nil {
		panic(err)
	}
	encoded := buf.Bytes()
	if tiff == nil {
		write(name, encoded)
		return
	}
	payload := append([]byte("Exif\x00\x00"), tiff...)
	app1 := &bytes.Buffer{}
	app1.Write([]byte{0xFF, 0xE1})
	binary.Write(app1, binary.BigEndian, uint16(len(payload)+2))
	app1.Write(payload)

	out := &bytes.Buffer{}
	out.Write(encoded[:2]) // SOI
	out.Write(app1.Bytes())
	out.Write(encoded[2:])
	write(name, out.Bytes())
}

func write(name string, data []byte) {
	path := filepath.Join(outDir, name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		panic(err)
	}
	fmt.Printf("  %-44s %5d bytes\n", name, len(data))
}

func writeJSON(name string, v any) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	write(name, append(data, '\n'))
}

func ftypStub(brand string) []byte {
	out := []byte{0, 0, 0, 0x18}
	out = append(out, []byte("ftyp")...)
	out = append(out, []byte(brand)...)
	out = append(out, make([]byte, 12)...)
	return out
}

// --- TIFF assembly -----------------------------------------------------

type tiffSpec struct {
	orientation     int
	orientationLong bool // write the tag as LONG (the Xiaomi field variant)
	dateTime        string
	offsetTime      string
	gps             *gpsSpec
}

type gpsSpec struct {
	latRef, lonRef string
	lat, lon       [3][2]uint32
	dateStamp      string
	timeStamp      [3][2]uint32
}

type entryDef struct {
	tag, typ uint16
	count    uint32
	value    []byte // raw value bytes; placed inline if ≤4, else in overflow
}

const (
	tASCII    = 2
	tShort    = 3
	tLong     = 4
	tRational = 5
)

func ascii(s string) []byte { return append([]byte(s), 0) }

func shortVal(bo binary.ByteOrder, v uint16) []byte {
	b := make([]byte, 2)
	bo.PutUint16(b, v)
	return b
}

func longVal(bo binary.ByteOrder, v uint32) []byte {
	b := make([]byte, 4)
	bo.PutUint32(b, v)
	return b
}

func rationals(bo binary.ByteOrder, pairs [3][2]uint32) []byte {
	b := make([]byte, 0, 24)
	for _, p := range pairs {
		b = append(b, longVal(bo, p[0])...)
		b = append(b, longVal(bo, p[1])...)
	}
	return b
}

// ifdSize is the serialized size of an IFD: count, entries, next-IFD
// pointer, then overflow values.
func ifdSize(entries []entryDef) int {
	size := 2 + len(entries)*12 + 4
	for _, e := range entries {
		if len(e.value) > 4 {
			size += len(e.value)
		}
	}
	return size
}

// buildIFD serializes an IFD at absolute offset base within the TIFF blob.
func buildIFD(bo binary.ByteOrder, base int, entries []entryDef) []byte {
	buf := &bytes.Buffer{}
	binary.Write(buf, bo, uint16(len(entries)))
	overflowOff := base + 2 + len(entries)*12 + 4
	overflow := &bytes.Buffer{}
	for _, e := range entries {
		binary.Write(buf, bo, e.tag)
		binary.Write(buf, bo, e.typ)
		binary.Write(buf, bo, e.count)
		if len(e.value) <= 4 {
			padded := make([]byte, 4)
			copy(padded, e.value)
			buf.Write(padded)
		} else {
			binary.Write(buf, bo, uint32(overflowOff+overflow.Len()))
			overflow.Write(e.value)
		}
	}
	buf.Write(longVal(bo, 0)) // no next IFD
	buf.Write(overflow.Bytes())
	return buf.Bytes()
}

func buildTIFF(bo binary.ByteOrder, spec tiffSpec) []byte {
	// Sub-IFD entries first: their sizes fix where each directory lands.
	var exifEntries, gpsEntries []entryDef
	if spec.dateTime != "" {
		exifEntries = append(exifEntries, entryDef{0x9003, tASCII, uint32(len(spec.dateTime) + 1), ascii(spec.dateTime)})
	}
	if spec.offsetTime != "" {
		exifEntries = append(exifEntries, entryDef{0x9011, tASCII, uint32(len(spec.offsetTime) + 1), ascii(spec.offsetTime)})
	}
	if g := spec.gps; g != nil {
		gpsEntries = append(gpsEntries,
			entryDef{0x0001, tASCII, 2, ascii(g.latRef)},
			entryDef{0x0002, tRational, 3, rationals(bo, g.lat)},
			entryDef{0x0003, tASCII, 2, ascii(g.lonRef)},
			entryDef{0x0004, tRational, 3, rationals(bo, g.lon)},
		)
		if g.dateStamp != "" {
			gpsEntries = append(gpsEntries,
				entryDef{0x0007, tRational, 3, rationals(bo, g.timeStamp)},
				entryDef{0x001D, tASCII, uint32(len(g.dateStamp) + 1), ascii(g.dateStamp)},
			)
		}
	}

	var ifd0 []entryDef
	if spec.orientation != 0 {
		if spec.orientationLong {
			ifd0 = append(ifd0, entryDef{0x0112, tLong, 1, longVal(bo, uint32(spec.orientation))})
		} else {
			ifd0 = append(ifd0, entryDef{0x0112, tShort, 1, shortVal(bo, uint16(spec.orientation))})
		}
	}
	// Placeholder pointer entries so IFD0's size is final before the
	// sub-IFD offsets are known.
	exifPtrIdx, gpsPtrIdx := -1, -1
	if len(exifEntries) > 0 {
		exifPtrIdx = len(ifd0)
		ifd0 = append(ifd0, entryDef{0x8769, tLong, 1, longVal(bo, 0)})
	}
	if len(gpsEntries) > 0 {
		gpsPtrIdx = len(ifd0)
		ifd0 = append(ifd0, entryDef{0x8825, tLong, 1, longVal(bo, 0)})
	}

	ifd0Base := 8
	exifBase := ifd0Base + ifdSize(ifd0)
	gpsBase := exifBase
	if len(exifEntries) > 0 {
		gpsBase = exifBase + ifdSize(exifEntries)
	}
	if exifPtrIdx >= 0 {
		ifd0[exifPtrIdx].value = longVal(bo, uint32(exifBase))
	}
	if gpsPtrIdx >= 0 {
		ifd0[gpsPtrIdx].value = longVal(bo, uint32(gpsBase))
	}

	buf := &bytes.Buffer{}
	if bo == binary.ByteOrder(binary.LittleEndian) {
		buf.WriteString("II")
	} else {
		buf.WriteString("MM")
	}
	binary.Write(buf, bo, uint16(42))
	binary.Write(buf, bo, uint32(ifd0Base))
	buf.Write(buildIFD(bo, ifd0Base, ifd0))
	if len(exifEntries) > 0 {
		buf.Write(buildIFD(bo, exifBase, exifEntries))
	}
	if len(gpsEntries) > 0 {
		buf.Write(buildIFD(bo, gpsBase, gpsEntries))
	}
	return buf.Bytes()
}
