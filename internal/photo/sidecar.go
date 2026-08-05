package photo

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"roadbook/internal/domain"
)

// Sidecar is what a Google Photos Takeout JSON sidecar contributes: which
// image it describes, a capture instant, a position. Takeout writes one such
// file beside each exported photo.
type Sidecar struct {
	Title     string     // the image filename the sidecar describes
	TakenTime *time.Time // photoTakenTime, falling back to creationTime (BRIEF §3D)
	Pos       *domain.LatLng
}

// rawSidecar is the Takeout shape. It stays inside this package
// (invariant 4). Timestamps are epoch seconds as strings; geoData is present
// but zeroed when Google holds no position.
type rawSidecar struct {
	Title          string      `json:"title"`
	PhotoTakenTime *rawEpoch   `json:"photoTakenTime"`
	CreationTime   *rawEpoch   `json:"creationTime"`
	GeoData        *rawGeoData `json:"geoData"`
	GeoDataExif    *rawGeoData `json:"geoDataExif"`
}

type rawEpoch struct {
	Timestamp string `json:"timestamp"`
}

type rawGeoData struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// ErrNotSidecar reports JSON that is not a Takeout photo sidecar.
var ErrNotSidecar = errors.New("JSON, but not a Google Photos Takeout sidecar (no title, photoTakenTime, or geoData)")

// ParseSidecar reads one Takeout sidecar. Position: geoData first, then
// geoDataExif (Takeout emits a zeroed geoData while geoDataExif still holds
// the camera's reading — docs/phase-4/DECISIONS.md); exact (0,0) means
// absent. Time: photoTakenTime is capture time; creationTime — upload-to-
// Google time — only as fallback (the PLAN correction recorded in BRIEF §3D).
func ParseSidecar(data []byte) (Sidecar, error) {
	var raw rawSidecar
	if err := json.Unmarshal(data, &raw); err != nil {
		return Sidecar{}, err
	}
	if raw.Title == "" && raw.PhotoTakenTime == nil && raw.GeoData == nil {
		return Sidecar{}, ErrNotSidecar
	}

	s := Sidecar{Title: raw.Title}
	if t, ok := epochTime(raw.PhotoTakenTime); ok {
		s.TakenTime = t
	} else if t, ok := epochTime(raw.CreationTime); ok {
		s.TakenTime = t
	}
	if raw.GeoData != nil {
		s.Pos = validPos(raw.GeoData.Latitude, raw.GeoData.Longitude)
	}
	if s.Pos == nil && raw.GeoDataExif != nil {
		s.Pos = validPos(raw.GeoDataExif.Latitude, raw.GeoDataExif.Longitude)
	}
	return s, nil
}

func epochTime(e *rawEpoch) (*time.Time, bool) {
	if e == nil {
		return nil, false
	}
	sec, err := strconv.ParseInt(e.Timestamp, 10, 64)
	if err != nil || sec <= 0 {
		return nil, false
	}
	t := time.Unix(sec, 0).UTC()
	return &t, true
}

// SidecarPairName returns the image filename a sidecar file describes, by
// its name alone: Takeout names sidecars "IMG.jpg.json" (older exports) or
// "IMG.jpg.supplemental-metadata.json" (newer) — and truncates long names at
// arbitrary points ("….supplemental-metad.json"), so the marker is matched
// by prefix, not from a suffix list. Returns "" when the name carries no
// pairing information — the caller then falls back to the sidecar's own
// Title field.
func SidecarPairName(filename string) string {
	lower := strings.ToLower(filename)
	if !strings.HasSuffix(lower, ".json") {
		return ""
	}
	base := filename[:len(filename)-len(".json")]
	if idx := strings.LastIndex(strings.ToLower(base), ".suppl"); idx > 0 {
		base = base[:idx]
	}
	if base != "" && strings.Contains(base, ".") {
		return base
	}
	return ""
}
