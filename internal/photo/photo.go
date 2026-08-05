// Package photo extracts position and capture time from photographs and
// produces their thumbnails. It is a parser package in the mould of
// internal/timeline: pure functions over bytes, emitting the types defined
// here and nothing else — no EXIF tag, IFD, or sidecar JSON shape escapes
// (CLAUDE.md invariant 4). Malformation is treated as absence, never as a
// crash: the product answer to "no GPS block" and to "GPS offset points past
// end of file" is the same photo, honestly unplaced.
//
// Two sources feed one Meta: the image's own EXIF block (ExtractEXIF) and a
// Google Photos Takeout sidecar JSON (ParseSidecar), merged by MergeSidecar
// under the precedence the phase brief fixes — EXIF wins for position, the
// sidecar fills gaps (docs/phase-4/BRIEF.md §3D); capture time resolves down
// an explicit ladder recorded per photo (§3E).
package photo

import (
	"time"

	"roadbook/internal/domain"
)

// PosSource records which reading produced a photo's position (BRIEF §3D).
type PosSource string

const (
	PosEXIF    PosSource = "exif"    // the camera's own GPS block
	PosSidecar PosSource = "sidecar" // Takeout geoData / geoDataExif
	PosNone    PosSource = "none"
)

// TimeSource records which rung of the resolution ladder produced a photo's
// instant (BRIEF §3E) — strongest first.
type TimeSource string

const (
	TimeGPS        TimeSource = "gps"         // GPS date+time stamps, defined UTC
	TimeEXIFOffset TimeSource = "exif_offset" // DateTimeOriginal + OffsetTimeOriginal
	TimeSidecar    TimeSource = "sidecar"     // Takeout photoTakenTime (epoch)
	TimeEXIFLocal  TimeSource = "exif_local"  // DateTimeOriginal in the adventure's offset
	TimeNone       TimeSource = "none"
)

// CivilTime is a wall-clock reading with no UTC offset — what EXIF
// DateTimeOriginal actually is. Kept as its own type rather than a time.Time
// in a fake location precisely so nothing downstream can mistake it for an
// instant.
type CivilTime struct {
	Year, Day, Hour, Min, Sec int
	Month                     time.Month
}

// In returns the instant this wall-clock reading denotes under the given
// UTC offset.
func (c CivilTime) In(offsetSec int) time.Time {
	loc := time.FixedZone("", offsetSec)
	return time.Date(c.Year, c.Month, c.Day, c.Hour, c.Min, c.Sec, 0, loc)
}

// Meta is everything extraction learned about one photograph. Pointer fields
// are nil when the source did not speak.
type Meta struct {
	Pos       *domain.LatLng
	PosSource PosSource

	GPSTime       *time.Time // UTC instant from the GPS receiver's own clock
	Wall          *CivilTime // DateTimeOriginal, offsetless
	WallOffsetSec *int       // OffsetTimeOriginal, parsed to seconds
	SidecarTime   *time.Time // photoTakenTime (or creationTime) from a sidecar

	Orientation int // EXIF orientation 1–8; 0 = absent (treat as 1)
}

// ResolvedTime is one photo's capture instant plus the provenance that
// placement and display need (BRIEF §3B: every instant states which reading
// produced it).
type ResolvedTime struct {
	Time      time.Time
	OffsetSec int // civil offset for display; see offset preference below
	Source    TimeSource
}

const (
	offsetRound = 900       // real UTC offsets are 15-minute multiples
	offsetMax   = 14 * 3600 // ±14 h bounds every real offset (UTC+14 exists)
)

// ResolveTime walks the ladder of BRIEF §3E: GPS UTC, then explicit EXIF
// offset, then sidecar epoch, then — only when the caller can supply the
// adventure's own offset (haveFallback) — wall time interpreted locally.
// A photo that reaches no rung returns ok=false: stored, shown, unplaced.
//
// The display offset prefers, in order: the wall−GPS derivation (both are
// measurements in the same file; docs/phase-4/DECISIONS.md), the explicit
// OffsetTimeOriginal tag, the caller's fallback, zero.
func ResolveTime(m Meta, fallbackOffsetSec int, haveFallback bool) (ResolvedTime, bool) {
	displayOffset := func(instant time.Time) int {
		if m.Wall != nil {
			if off, ok := deriveOffset(*m.Wall, instant); ok {
				return off
			}
		}
		if m.WallOffsetSec != nil {
			return *m.WallOffsetSec
		}
		if haveFallback {
			return fallbackOffsetSec
		}
		return 0
	}

	switch {
	case m.GPSTime != nil:
		return ResolvedTime{Time: *m.GPSTime, OffsetSec: displayOffset(*m.GPSTime), Source: TimeGPS}, true
	case m.Wall != nil && m.WallOffsetSec != nil:
		return ResolvedTime{Time: m.Wall.In(*m.WallOffsetSec), OffsetSec: *m.WallOffsetSec, Source: TimeEXIFOffset}, true
	case m.SidecarTime != nil:
		return ResolvedTime{Time: *m.SidecarTime, OffsetSec: displayOffset(*m.SidecarTime), Source: TimeSidecar}, true
	case m.Wall != nil && haveFallback:
		return ResolvedTime{Time: m.Wall.In(fallbackOffsetSec), OffsetSec: fallbackOffsetSec, Source: TimeEXIFLocal}, true
	}
	return ResolvedTime{Source: TimeNone}, false
}

// deriveOffset recovers the camera's civil offset from its two clocks: the
// wall reading minus the UTC instant, rounded to the nearest 15 minutes.
// Rejected outside ±14 h — then the two clocks simply disagree (a camera set
// to the wrong date) and the derivation would be noise, not an offset.
func deriveOffset(wall CivilTime, instant time.Time) (int, bool) {
	diff := wall.In(0).Sub(instant)
	sec := int(diff / time.Second)
	rounded := ((sec + offsetRound/2*sign(sec)) / offsetRound) * offsetRound
	if rounded < -offsetMax || rounded > offsetMax {
		return 0, false
	}
	return rounded, true
}

func sign(n int) int {
	if n < 0 {
		return -1
	}
	return 1
}

// MergeSidecar folds a paired sidecar into an image's Meta under BRIEF §3D:
// EXIF wins for position, the sidecar fills absence; the sidecar's capture
// time joins the ladder at its own rung regardless.
func MergeSidecar(m Meta, s Sidecar) Meta {
	if m.Pos == nil && s.Pos != nil {
		m.Pos = s.Pos
		m.PosSource = PosSidecar
	}
	if s.TakenTime != nil {
		m.SidecarTime = s.TakenTime
	}
	return m
}

// validPos applies the shared "Null Island means unset" rule: exact (0,0) is
// absence, not a position — the same rejection the anomaly filters apply to
// raw fixes — and out-of-range coordinates are malformation, hence absence.
func validPos(lat, lon float64) *domain.LatLng {
	if lat == 0 && lon == 0 {
		return nil
	}
	if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return nil
	}
	return &domain.LatLng{Lat: lat, Lon: lon}
}
