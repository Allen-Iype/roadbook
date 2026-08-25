// Command gen regenerates every fixture in testdata/photos/. All coordinates
// and times are fabricated (12.3456°N 45.6789°E is open ocean; the corpus
// uses the demo dataset's fictional Reykjavík persona); nothing here derives
// from real data — the data-safety rule for committed fixtures
// (docs/phase-4/BRIEF.md §4). Run from the repository root:
//
//	go run ./testdata/photos/gen           # the per-file parser fixtures
//	go run ./testdata/photos/gen -corpus   # testdata/photos/corpus/ (phase 11)
//
// The EXIF blocks are hand-assembled TIFF structures, which is the point:
// the parser's regression fixtures should exercise the exact byte layout the
// specification describes (both byte orders) plus the malformations the
// defensive paths exist for (truncated APP1, out-of-bounds IFD offset,
// zero-denominator rational). The corpus is a plausible camera roll for the
// stay-point synthesis regression: deterministic — running it twice writes
// identical bytes.
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"time"
)

const outDir = "testdata/photos"

func main() {
	corpus := flag.Bool("corpus", false, "write the phase-11 synthesis corpus instead of the parser fixtures")
	flag.Parse()
	if _, err := os.Stat(outDir); err != nil {
		fmt.Fprintln(os.Stderr, "gen: run from the repository root:", err)
		os.Exit(1)
	}
	if *corpus {
		writeCorpus()
		return
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

// --- Phase 11 synthesis corpus ------------------------------------------
//
// A fictional persona's camera roll, Apr–Jun 2026, on the demo dataset's
// Reykjavík geography (all fictional; the persona photographs their evenings
// at home and two trips). Everything is UTC+0 — each photo carries a GPS
// clock and a matching wall clock, so the derived offset is zero. The
// storyline the synthesis regression pins:
//
//   - 16 home evenings, two shots ~40 min apart each: recurring ≥30-min
//     stays across ≥8 distinct days ⇒ the one home base, from synthetic
//     evidence alone (no visit segments exist anywhere in a photo batch).
//   - A dense Höfn trip (Jun 5–7): in-transit shots plus ≥60-min evening
//     stays ⇒ a candidate with dwells, transit fixes, and a stay under the
//     destination-dwell bar (Jökulsárlón, 20 min) that must count as
//     observations but never as a destination.
//   - A sparse Akureyri weekend (May 9–10): the minimum honest evidence —
//     one ≥60-min stay plus a handful of fixes ⇒ a second candidate.
//   - One photo with stripped EXIF whose Takeout sidecar restores position
//     and time, and two wall-clock-only photos that stay honestly unplaced.

const corpusDir = "testdata/photos/corpus"

type shot struct {
	name     string
	lat, lon float64
	t        time.Time
}

func at(y int, m time.Month, d, hh, mm int) time.Time {
	return time.Date(y, m, d, hh, mm, 0, 0, time.UTC)
}

func writeCorpus() {
	if err := os.MkdirAll(corpusDir, 0o755); err != nil {
		panic(err)
	}
	homeLat, homeLon := 64.1466, -21.9426 // Reykjavík (demo persona)

	var shots []shot
	// Home evenings: two shots, 19:00 and 19:40, deterministic ~25 m scatter.
	homeDays := []struct {
		m time.Month
		d int
	}{
		{time.April, 5}, {time.April, 8}, {time.April, 12}, {time.April, 15},
		{time.April, 19}, {time.April, 22}, {time.April, 26}, {time.April, 29},
		{time.May, 3}, {time.May, 8}, {time.May, 13}, {time.May, 21},
		{time.June, 1}, {time.June, 4}, {time.June, 7}, {time.June, 20},
	}
	for i, hd := range homeDays {
		tag := fmt.Sprintf("home_%02d%s", hd.d, map[time.Month]string{time.April: "apr", time.May: "may", time.June: "jun"}[hd.m])
		jitter := 0.0002 * float64(i%3) // 0, ~22 m, ~44 m — same place, honest scatter
		shots = append(shots,
			shot{tag + "_a.jpg", homeLat + jitter, homeLon, at(2026, hd.m, hd.d, 19, 0)},
			shot{tag + "_b.jpg", homeLat, homeLon - jitter, at(2026, hd.m, hd.d, 19, 40)},
		)
	}
	// Dense trip: Höfn, Jun 5–7.
	shots = append(shots,
		shot{"trip1_selfoss.jpg", 63.9339, -20.9971, at(2026, time.June, 5, 11, 0)},
		shot{"trip1_vik.jpg", 63.4194, -19.0060, at(2026, time.June, 5, 13, 0)},
		shot{"trip1_jokulsarlon_a.jpg", 64.0784, -16.2306, at(2026, time.June, 5, 16, 0)},
		shot{"trip1_jokulsarlon_b.jpg", 64.0786, -16.2300, at(2026, time.June, 5, 16, 20)},
		shot{"trip1_hofn_eve_a.jpg", 64.2539, -15.2082, at(2026, time.June, 5, 19, 0)},
		shot{"trip1_hofn_eve_b.jpg", 64.2540, -15.2080, at(2026, time.June, 5, 19, 30)},
		shot{"trip1_hofn_eve_c.jpg", 64.2538, -15.2085, at(2026, time.June, 5, 20, 10)},
		shot{"trip1_hofn_morning_a.jpg", 64.2539, -15.2082, at(2026, time.June, 6, 10, 0)},
		shot{"trip1_hofn_morning_b.jpg", 64.2541, -15.2079, at(2026, time.June, 6, 10, 45)},
		shot{"trip1_stokksnes_a.jpg", 64.2444, -14.9784, at(2026, time.June, 6, 15, 0)},
		shot{"trip1_stokksnes_b.jpg", 64.2446, -14.9780, at(2026, time.June, 6, 16, 10)},
		shot{"trip1_hofn_night_a.jpg", 64.2539, -15.2082, at(2026, time.June, 6, 20, 0)},
		shot{"trip1_hofn_night_b.jpg", 64.2540, -15.2083, at(2026, time.June, 6, 20, 35)},
		shot{"trip1_vik_return.jpg", 63.4194, -19.0060, at(2026, time.June, 7, 13, 0)},
	)
	// Sparse trip: Akureyri, May 9–10.
	shots = append(shots,
		shot{"trip2_borgarnes.jpg", 64.5380, -21.9210, at(2026, time.May, 9, 15, 0)},
		shot{"trip2_akureyri_eve_a.jpg", 65.6835, -18.1002, at(2026, time.May, 9, 18, 0)},
		shot{"trip2_akureyri_eve_b.jpg", 65.6836, -18.1000, at(2026, time.May, 9, 19, 10)},
		shot{"trip2_akureyri_morn_a.jpg", 65.6835, -18.1002, at(2026, time.May, 10, 11, 0)},
		shot{"trip2_akureyri_morn_b.jpg", 65.6837, -18.1005, at(2026, time.May, 10, 11, 40)},
		shot{"trip2_blonduos.jpg", 65.6600, -20.2800, at(2026, time.May, 10, 14, 0)},
	)

	le := binary.ByteOrder(binary.LittleEndian)
	for _, s := range shots {
		writeJPEGTo(corpusDir, s.name, buildTIFF(le, tiffSpec{
			dateTime: s.t.Format("2006:01:02 15:04:05"),
			gps:      gpsFromDeg(s.lat, s.lon, s.t),
		}))
	}

	// A photo whose EXIF was stripped (a messenger copy): the Takeout
	// sidecar restores position and time — the sidecar path, in-corpus.
	writeJPEGTo(corpusDir, "trip1_hofn_market.jpg", nil)
	writeJSONTo(corpusDir, "trip1_hofn_market.jpg.json", map[string]any{
		"title":          "trip1_hofn_market.jpg",
		"photoTakenTime": map[string]any{"timestamp": fmt.Sprint(at(2026, time.June, 6, 12, 0).Unix())},
		"geoData":        map[string]any{"latitude": 64.2536, "longitude": -15.2090},
	})

	// Wall-clock-only photos: no position anywhere — honestly unplaced.
	writeJPEGTo(corpusDir, "unplaced_a.jpg", buildTIFF(le, tiffSpec{dateTime: "2026:05:15 10:00:00"}))
	writeJPEGTo(corpusDir, "unplaced_b.jpg", buildTIFF(le, tiffSpec{dateTime: "2026:05:16 11:30:00"}))

	fmt.Println("gen: corpus written to", corpusDir)
}

// gpsFromDeg converts decimal degrees and a UTC instant into the EXIF GPS
// block: DMS rationals at 1/100-arcsecond precision (~0.3 m) plus the GPS
// date and time stamps.
func gpsFromDeg(lat, lon float64, t time.Time) *gpsSpec {
	dms := func(deg float64) [3][2]uint32 {
		a := math.Abs(deg)
		d := math.Floor(a)
		m := math.Floor((a - d) * 60)
		s := math.Round(((a-d)*60 - m) * 60 * 100)
		return [3][2]uint32{{uint32(d), 1}, {uint32(m), 1}, {uint32(s), 100}}
	}
	ref := func(deg float64, pos, neg string) string {
		if deg < 0 {
			return neg
		}
		return pos
	}
	return &gpsSpec{
		latRef: ref(lat, "N", "S"), lat: dms(lat),
		lonRef: ref(lon, "E", "W"), lon: dms(lon),
		dateStamp: t.Format("2006:01:02"),
		timeStamp: [3][2]uint32{{uint32(t.Hour()), 1}, {uint32(t.Minute()), 1}, {uint32(t.Second()), 1}},
	}
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
func writeJPEG(name string, tiff []byte) { writeJPEGTo(outDir, name, tiff) }

func writeJPEGTo(dir, name string, tiff []byte) {
	buf := &bytes.Buffer{}
	if err := jpeg.Encode(buf, baseImage(), &jpeg.Options{Quality: 90}); err != nil {
		panic(err)
	}
	encoded := buf.Bytes()
	if tiff == nil {
		writeTo(dir, name, encoded)
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
	writeTo(dir, name, out.Bytes())
}

func write(name string, data []byte) { writeTo(outDir, name, data) }

func writeTo(dir, name string, data []byte) {
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		panic(err)
	}
	fmt.Printf("  %-44s %5d bytes\n", name, len(data))
}

func writeJSON(name string, v any) { writeJSONTo(outDir, name, v) }

func writeJSONTo(dir, name string, v any) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	writeTo(dir, name, append(data, '\n'))
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
