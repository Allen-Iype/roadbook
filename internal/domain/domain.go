// Package domain holds the types the rest of the system exchanges. Parsers emit
// these and nothing else (CLAUDE.md invariant 4): detection, storage, and the API
// all speak domain types, so adding another source format later touches only its
// parser package.
package domain

import "time"

// LatLng is a WGS84 coordinate in decimal degrees.
type LatLng struct {
	Lat float64
	Lon float64
}

// Visit is a stay at one place. Loc is nil when the export omitted or mangled the
// coordinate; such visits are kept (they still occupy their position in the
// timeline) and skipped where a location is required.
type Visit struct {
	Start        time.Time
	End          time.Time
	Loc          *LatLng
	SemanticType string // INFERRED_HOME, INFERRED_WORK, SEARCHED_ADDRESS, UNKNOWN, or ""
}

// Activity is a movement between two endpoints.
type Activity struct {
	Start     time.Time
	End       time.Time
	From      *LatLng
	To        *LatLng
	DistanceM float64
	Mode      string // Google's guess (IN_PASSENGER_VEHICLE, FLYING, …); unreliable at extremes
}

// PathPoint is one timestamped position from a timelinePath segment.
type PathPoint struct {
	Time time.Time
	Loc  *LatLng
}

// RawPosition is one position fix from the export's rawSignals section — the
// only data in the export that carries a reported accuracy. An export holds
// only the ~30 days before it was taken, so most of the timeline has none.
type RawPosition struct {
	Time      time.Time
	Loc       *LatLng
	AccuracyM float64 // reported horizontal accuracy in metres; 0 when absent
	Source    string  // WIFI, CELL, WIFI_ONLY, GPS, or "" when absent
}

// Observations is everything parsed from one source, in source-file order.
// Slices are treated as immutable after parsing (CLAUDE.md invariant 2).
type Observations struct {
	Visits       []Visit
	Activities   []Activity
	Points       []PathPoint
	RawPositions []RawPosition
}
