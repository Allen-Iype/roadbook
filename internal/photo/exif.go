package photo

import (
	"encoding/binary"
	"strconv"
	"strings"
	"time"
)

// ExtractEXIF walks a JPEG's metadata and returns whatever it holds of the
// seven tags this product consumes (BRIEF §3C): GPS latitude/longitude and
// their hemisphere refs, the GPS date and time stamps, DateTimeOriginal,
// OffsetTimeOriginal, and Orientation. It never fails: EXIF is offset-based
// and files in the wild violate the specification routinely, so every read
// is bounds-checked and any malformation — truncated segment, out-of-range
// offset, zero-denominator rational — yields absence, not an error. The
// caller has already established via Sniff that data is a JPEG.
//
// The walk: JPEG segments to APP1 ("Exif\0\0"), the embedded TIFF header
// (byte order, IFD0 offset), IFD0 for Orientation and the two pointer tags,
// then the Exif IFD (times) and GPS IFD (position). The IFD1 chain
// (embedded thumbnail) is deliberately not followed — nothing there is
// consumed, and not walking it is one less way for a hostile file to send
// the parser on a tour.
func ExtractEXIF(data []byte) Meta {
	var m Meta
	tiff, ok := findEXIFTIFF(data)
	if !ok {
		return m
	}
	parseTIFF(tiff, &m)
	return m
}

// findEXIFTIFF scans JPEG segments for the APP1 Exif payload and returns the
// embedded TIFF blob.
func findEXIFTIFF(data []byte) ([]byte, bool) {
	const (
		markerAPP1 = 0xE1
		markerSOS  = 0xDA
		markerEOI  = 0xD9
	)
	i := 2 // past SOI
	for i+4 <= len(data) {
		if data[i] != 0xFF {
			return nil, false // lost sync; stop rather than guess
		}
		marker := data[i+1]
		switch {
		case marker == 0xFF: // padding
			i++
			continue
		case marker == markerSOS || marker == markerEOI:
			return nil, false // image data begins; no EXIF ahead of it
		case marker == 0x01 || (marker >= 0xD0 && marker <= 0xD7):
			i += 2 // standalone marker, no length
			continue
		}
		length := int(binary.BigEndian.Uint16(data[i+2 : i+4]))
		if length < 2 || i+2+length > len(data) {
			return nil, false // declared length runs past EOF: malformation
		}
		payload := data[i+4 : i+2+length]
		if marker == markerAPP1 && len(payload) > 6 && string(payload[:6]) == "Exif\x00\x00" {
			return payload[6:], true
		}
		i += 2 + length
	}
	return nil, false
}

// tiffReader wraps the TIFF blob with its declared byte order and
// bounds-checked reads. All offsets in the blob are relative to its start.
type tiffReader struct {
	data []byte
	bo   binary.ByteOrder
}

func (r tiffReader) u16(off int) (uint16, bool) {
	if off < 0 || off+2 > len(r.data) {
		return 0, false
	}
	return r.bo.Uint16(r.data[off : off+2]), true
}

func (r tiffReader) u32(off int) (uint32, bool) {
	if off < 0 || off+4 > len(r.data) {
		return 0, false
	}
	return r.bo.Uint32(r.data[off : off+4]), true
}

func (r tiffReader) bytes(off, n int) ([]byte, bool) {
	if off < 0 || n < 0 || off+n > len(r.data) {
		return nil, false
	}
	return r.data[off : off+n], true
}

// field types the consumed tags use
const (
	typeASCII    = 2
	typeShort    = 3
	typeLong     = 4
	typeRational = 5
)

var typeSize = map[uint16]int{1: 1, typeASCII: 1, typeShort: 2, typeLong: 4, typeRational: 8, 7: 1, 9: 4, 10: 8}

// entry is one 12-byte IFD entry, with its value located (inline or offset).
type entry struct {
	typ uint16
	cnt int
	val []byte
}

// walkIFD reads the directory at off and returns the consumed entries by tag.
// Entry counts are capped: a directory claiming thousands of entries is
// either malformed or hostile, and this parser wants seven tags.
func walkIFD(r tiffReader, off int, want map[uint16]bool) map[uint16]entry {
	const maxEntries = 512
	out := map[uint16]entry{}
	count, ok := r.u16(off)
	if !ok {
		return out
	}
	n := min(int(count), maxEntries)
	for i := range n {
		base := off + 2 + i*12
		tag, ok1 := r.u16(base)
		typ, ok2 := r.u16(base + 2)
		cnt32, ok3 := r.u32(base + 4)
		if !ok1 || !ok2 || !ok3 || !want[tag] {
			continue
		}
		size, known := typeSize[typ]
		if !known {
			continue
		}
		cnt := int(cnt32)
		if cnt < 0 || cnt > 1<<16 {
			continue
		}
		total := size * cnt
		var val []byte
		if total <= 4 {
			val, _ = r.bytes(base+8, total)
		} else {
			valOff, ok := r.u32(base + 8)
			if !ok {
				continue
			}
			val, _ = r.bytes(int(valOff), total)
		}
		if val == nil {
			continue
		}
		out[tag] = entry{typ: typ, cnt: cnt, val: val}
	}
	return out
}

// Consumed tag numbers.
const (
	tagOrientation      = 0x0112
	tagExifIFD          = 0x8769
	tagGPSIFD           = 0x8825
	tagDateTimeOriginal = 0x9003
	tagOffsetTimeOrig   = 0x9011
	tagGPSLatRef        = 0x0001
	tagGPSLat           = 0x0002
	tagGPSLonRef        = 0x0003
	tagGPSLon           = 0x0004
	tagGPSTimeStamp     = 0x0007
	tagGPSDateStamp     = 0x001D
)

func parseTIFF(tiff []byte, m *Meta) {
	if len(tiff) < 8 {
		return
	}
	var bo binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		bo = binary.LittleEndian
	case "MM":
		bo = binary.BigEndian
	default:
		return
	}
	r := tiffReader{data: tiff, bo: bo}
	if magic, ok := r.u16(2); !ok || magic != 42 {
		return
	}
	ifd0Off, ok := r.u32(4)
	if !ok {
		return
	}

	ifd0 := walkIFD(r, int(ifd0Off), map[uint16]bool{
		tagOrientation: true, tagExifIFD: true, tagGPSIFD: true,
	})
	// Orientation is SHORT by specification, but real writers disagree —
	// Xiaomi stores it as LONG (found on the maintainer's own phone) — so
	// both integer types are accepted.
	if e, ok := ifd0[tagOrientation]; ok && e.cnt >= 1 {
		if v, ok := uintValue(r.bo, e); ok && v >= 1 && v <= 8 {
			m.Orientation = int(v)
		}
	}

	if e, ok := ifd0[tagExifIFD]; ok {
		if off, ok := pointerValue(r.bo, e); ok {
			parseExifIFD(r, off, m)
		}
	}
	if e, ok := ifd0[tagGPSIFD]; ok {
		if off, ok := pointerValue(r.bo, e); ok {
			parseGPSIFD(r, off, m)
		}
	}
}

func parseExifIFD(r tiffReader, off int, m *Meta) {
	ifd := walkIFD(r, off, map[uint16]bool{tagDateTimeOriginal: true, tagOffsetTimeOrig: true})
	if e, ok := ifd[tagDateTimeOriginal]; ok {
		if ct, ok := parseCivil(asciiValue(e)); ok {
			m.Wall = &ct
		}
	}
	if e, ok := ifd[tagOffsetTimeOrig]; ok {
		if sec, ok := parseUTCOffset(asciiValue(e)); ok {
			m.WallOffsetSec = &sec
		}
	}
}

func parseGPSIFD(r tiffReader, off int, m *Meta) {
	ifd := walkIFD(r, off, map[uint16]bool{
		tagGPSLatRef: true, tagGPSLat: true, tagGPSLonRef: true, tagGPSLon: true,
		tagGPSTimeStamp: true, tagGPSDateStamp: true,
	})

	lat, latOK := dms(r.bo, ifd[tagGPSLat])
	lon, lonOK := dms(r.bo, ifd[tagGPSLon])
	if latOK && lonOK {
		if strings.HasPrefix(asciiValue(ifd[tagGPSLatRef]), "S") {
			lat = -lat
		}
		if strings.HasPrefix(asciiValue(ifd[tagGPSLonRef]), "W") {
			lon = -lon
		}
		if p := validPos(lat, lon); p != nil {
			m.Pos = p
			m.PosSource = PosEXIF
		}
	}

	// GPS time needs both halves: the date stamp string and the three
	// time-of-day rationals. Either alone cannot make an instant.
	date := asciiValue(ifd[tagGPSDateStamp])
	if e, ok := ifd[tagGPSTimeStamp]; ok && date != "" && e.typ == typeRational && e.cnt >= 3 {
		h, ok1 := rational(r.bo, e.val, 0)
		mi, ok2 := rational(r.bo, e.val, 1)
		s, ok3 := rational(r.bo, e.val, 2)
		// Guard nonsense times — the rationals are unvalidated bytes.
		if ok1 && ok2 && ok3 && h < 24 && mi < 60 && s < 61 {
			if ct, ok := parseCivil(date + " 00:00:00"); ok {
				t := time.Date(ct.Year, ct.Month, ct.Day, 0, 0, 0, 0, time.UTC).
					Add(time.Duration((h*3600 + mi*60 + s) * float64(time.Second)))
				m.GPSTime = &t
			}
		}
	}
}

// pointerValue reads a LONG-typed IFD pointer entry.
func pointerValue(bo binary.ByteOrder, e entry) (int, bool) {
	if e.typ != typeLong || e.cnt < 1 || len(e.val) < 4 {
		return 0, false
	}
	return int(bo.Uint32(e.val[:4])), true
}

// uintValue reads a SHORT- or LONG-typed entry's first value.
func uintValue(bo binary.ByteOrder, e entry) (uint32, bool) {
	switch {
	case e.typ == typeShort && len(e.val) >= 2:
		return uint32(bo.Uint16(e.val[:2])), true
	case e.typ == typeLong && len(e.val) >= 4:
		return bo.Uint32(e.val[:4]), true
	}
	return 0, false
}

// rational reads the i-th RATIONAL (two uint32s) from a value blob. A zero
// denominator is malformation → absent, per the defensive-parse rule.
func rational(bo binary.ByteOrder, val []byte, i int) (float64, bool) {
	off := i * 8
	if off+8 > len(val) {
		return 0, false
	}
	num := bo.Uint32(val[off : off+4])
	den := bo.Uint32(val[off+4 : off+8])
	if den == 0 {
		return 0, false
	}
	return float64(num) / float64(den), true
}

// dms converts a 3-rational degrees/minutes/seconds entry to decimal degrees.
func dms(bo binary.ByteOrder, e entry) (float64, bool) {
	if e.typ != typeRational || e.cnt < 3 {
		return 0, false
	}
	d, ok1 := rational(bo, e.val, 0)
	m, ok2 := rational(bo, e.val, 1)
	s, ok3 := rational(bo, e.val, 2)
	if !ok1 || !ok2 || !ok3 {
		return 0, false
	}
	return d + m/60 + s/3600, true
}

// asciiValue returns an ASCII entry's string, trimmed of the NUL terminator
// the format requires and the padding some writers add.
func asciiValue(e entry) string {
	if e.typ != typeASCII {
		return ""
	}
	return strings.TrimRight(string(e.val), "\x00 ")
}

// parseCivil parses "YYYY:MM:DD HH:MM:SS" (the EXIF convention; some writers
// use '-' in the date) into a CivilTime.
func parseCivil(s string) (CivilTime, bool) {
	s = strings.TrimSpace(s)
	if len(s) != 19 {
		return CivilTime{}, false
	}
	norm := strings.ReplaceAll(s[:10], "-", ":") + s[10:]
	atoi := func(part string) (int, bool) {
		n, err := strconv.Atoi(part)
		return n, err == nil
	}
	y, ok1 := atoi(norm[0:4])
	mo, ok2 := atoi(norm[5:7])
	d, ok3 := atoi(norm[8:10])
	h, ok4 := atoi(norm[11:13])
	mi, ok5 := atoi(norm[14:16])
	sec, ok6 := atoi(norm[17:19])
	if !ok1 || !ok2 || !ok3 || !ok4 || !ok5 || !ok6 {
		return CivilTime{}, false
	}
	if y < 1000 || mo < 1 || mo > 12 || d < 1 || d > 31 || h > 23 || mi > 59 || sec > 60 {
		return CivilTime{}, false
	}
	return CivilTime{Year: y, Month: time.Month(mo), Day: d, Hour: h, Min: mi, Sec: sec}, true
}

// parseUTCOffset parses "+05:30", "-08:00", or "Z" to seconds.
func parseUTCOffset(s string) (int, bool) {
	s = strings.TrimSpace(s)
	if s == "Z" {
		return 0, true
	}
	if len(s) != 6 || (s[0] != '+' && s[0] != '-') || s[3] != ':' {
		return 0, false
	}
	h, err1 := strconv.Atoi(s[1:3])
	m, err2 := strconv.Atoi(s[4:6])
	if err1 != nil || err2 != nil || h > 14 || m > 59 {
		return 0, false
	}
	sec := h*3600 + m*60
	if s[0] == '-' {
		sec = -sec
	}
	return sec, true
}
