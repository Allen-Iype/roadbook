package route

// Map matching (phase 23, #9): a recorded stretch may be drawn along the
// road its points lie on (invariant 6, as amended 2026-11-16).
//
// Routing and matching are different questions. Routing asks for the best
// road from A to B and knows nothing in between. Matching is handed a
// sequence of noisy positions with their times and asks which roads it was
// most likely travelling along; every point is weighed against its
// neighbours, so one position on the wrong side road does not drag the line
// there and back. For a run of recorded points, matching is the right
// question (measured in docs/phase-23/BRIEF.md §2.2: routing each hop on
// its own came back 1.45 times point to point, matching 1.21).
//
// The shape repeats the routing seam's: a narrow interface, a null
// implementation, the OSRM client; answers fetched by the batch command
// and cached; ApplyMatches, pure, lays a cached answer alongside a leg's
// own points. What a match may never do is in ApplyMatches' comment.

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"math"
	"net/url"
	"slices"
	"strings"
	"time"

	"roadbook/internal/domain"
	"roadbook/internal/journey"
)

// ErrNoMatch is a data answer, like ErrNoRoute: no single road line fits
// these points within the search radius. It is cached so re-runs don't
// re-ask. Anything else is operational and never cached.
var ErrNoMatch = errors.New("no road fits these points")

// MatchParams are matching's named thresholds (invariant 3), served with
// every journey.
type MatchParams struct {
	// RadiusM: how far from a recorded point the matcher may look for its
	// road. 100 m placed more points and failed less often than 50 m at the
	// same overall length; 250 m ran for minutes per request (BRIEF §2.2).
	// Part of the question, so part of the cache key.
	RadiusM float64 `json:"match_radius_m"`
	// MaxRatio is the guard: a matched line is drawn only when it is under
	// this many times the stretch's point-to-point length. Positions are
	// mostly WiFi and cell, and a matcher handed one 300 m off the road
	// will sometimes take a side road to reach it; the result is far longer
	// than the points justify. Applied when a cached answer is READ, never
	// when it is written, so changing it costs no new requests. 0 draws
	// nothing along roads.
	MaxRatio float64 `json:"match_max_ratio"`
	// MinM: a stretch shorter than this, point to point, is not asked
	// about — at that length the road and the straight line are the same
	// line on any map.
	MinM float64 `json:"match_min_m"`
	// SkipModes: a stretch whose mode label (LegMode) is one of these is
	// not asked about and not drawn along a road. The matcher has one
	// profile, driving; a railway mostly runs beside a road, so a train
	// stretch passes the length guard and would be drawn on that road
	// (measured: 69% to 100% of train kilometres at the default guard,
	// docs/phase-23/DECISIONS.md). The label is Google's guess; a wrong
	// one costs a road stretch its road line, which is today's drawing.
	// Applied when asking and again when reading.
	SkipModes []string `json:"match_skip_modes"`
}

func DefaultMatchParams() MatchParams {
	return MatchParams{RadiusM: 100, MaxRatio: 1.3, MinM: 200,
		SkipModes: []string{"IN_TRAIN", "IN_SUBWAY", "IN_FERRY", "FLYING"}}
}

// LegMode is the source's mode label for a stretch: the mode of the
// activity that overlaps it in time the longest, the earlier one on a tie,
// "" when none overlaps. Legs carry no mode of their own (Assemble builds
// them from points), so it is read from the same activities the journey
// was assembled from. Pure.
func LegMode(l journey.Leg, acts []domain.Activity) string {
	best, bestD := "", time.Duration(0)
	var bestStart time.Time
	for _, a := range acts {
		st, en := a.Start, a.End
		if l.Start().After(st) {
			st = l.Start()
		}
		if l.End().Before(en) {
			en = l.End()
		}
		d := en.Sub(st)
		if d > bestD || (d == bestD && d > 0 && a.Start.Before(bestStart)) {
			best, bestD, bestStart = a.Mode, d, a.Start
		}
	}
	return best
}

// MatchByDefault says whether the batch command matches when -match is not
// given. A matching request carries every recorded point of a stretch,
// where a gap request carries two ends, so against the public
// demonstration server it is off unless asked for.
func MatchByDefault(routerURL string) bool {
	pub, _ := url.Parse(PublicOSRMURL)
	u, err := url.Parse(strings.TrimSpace(routerURL))
	if err != nil {
		return true
	}
	return !strings.EqualFold(u.Hostname(), pub.Hostname())
}

// TracePoint is one point of a matching request: the recorded coordinate
// rounded to five decimals (about 1 m — far below the source's accuracy)
// and its time in whole seconds. Fixed-point integers for the reason Key
// uses them: the request must be reproducible exactly, and hashed.
type TracePoint struct {
	LatE5 int32
	LonE5 int32
	Unix  int64
}

func e5(deg float64) int32 { return int32(math.Round(deg * 1e5)) }

// Loc is the rounded coordinate — what a Matcher is actually asked about.
func (p TracePoint) Loc() domain.LatLng {
	return domain.LatLng{Lat: float64(p.LatE5) / 1e5, Lon: float64(p.LonE5) / 1e5}
}

// Match is a matcher's answer in domain terms.
type Match struct {
	Points    []domain.LatLng
	DistanceM float64
	// Unplaced counts request points the matcher left off the road as
	// outliers. EndsPlaced is false when the first or the last of them was
	// one: the line then stops short of the stretch, which ApplyMatches
	// refuses to draw.
	Unplaced   int
	EndsPlaced bool
}

// Matcher answers one question: which road do these timed points lie on?
type Matcher interface {
	Match(ctx context.Context, trace []TracePoint, radiusM int) (Match, error)
}

// Match on the null implementation matches nothing: every recorded stretch
// stays drawn point to point, which is today's drawing (invariant 7).
func (Null) Match(context.Context, []TracePoint, int) (Match, error) {
	return Match{}, ErrNoMatch
}

// MatchKey identifies one cached matching request: a SHA-256 over the
// request's points, the profile and the search radius. As with Key, the key
// IS the request — the hash covers exactly the rounded coordinates and
// times a Matcher is sent — so a hit is exact. The guard (MaxRatio) is
// deliberately absent: it is applied on reading.
type MatchKey struct {
	PointsHash string
	Profile    string
	RadiusM    int32
}

// Trace is one matching request with its key.
type Trace struct {
	Key    MatchKey
	Points []TracePoint
}

// TraceFor builds the request for one leg, or reports that the leg is not
// asked about: only a recorded stretch of two or more points and at least
// MinM, whose mode (LegMode; "" for none) is not in SkipModes, is. Times are made strictly increasing in the request (the matcher
// requires it; Assemble's thinning already spaces points, so this is a
// guarantee, not a routine correction) — in the request only, the leg's
// points are not touched.
func TraceFor(l journey.Leg, mode string, profile string, p MatchParams) (Trace, bool) {
	if l.Kind != journey.LegObserved || len(l.Points) < 2 || l.DistanceKm*1000 < p.MinM {
		return Trace{}, false
	}
	if mode != "" && slices.Contains(p.SkipModes, mode) {
		return Trace{}, false
	}
	pts := make([]TracePoint, len(l.Points))
	h := sha256.New()
	var buf [16]byte
	var last int64
	for i, pt := range l.Points {
		u := pt.Time.Unix()
		if i > 0 && u <= last {
			u = last + 1
		}
		last = u
		pts[i] = TracePoint{LatE5: e5(pt.Loc.Lat), LonE5: e5(pt.Loc.Lon), Unix: u}
		binary.BigEndian.PutUint32(buf[0:], uint32(pts[i].LatE5))
		binary.BigEndian.PutUint32(buf[4:], uint32(pts[i].LonE5))
		binary.BigEndian.PutUint64(buf[8:], uint64(u))
		h.Write(buf[:])
	}
	return Trace{
		Key: MatchKey{
			PointsHash: hex.EncodeToString(h.Sum(nil)),
			Profile:    profile,
			RadiusM:    int32(math.Round(p.RadiusM)),
		},
		Points: pts,
	}, true
}

// Traces collects the matching requests for a journey's recorded stretches,
// deduped, in first-appearance order. acts are the activities the journey
// was assembled from; they supply each stretch's mode.
func Traces(j journey.Journey, acts []domain.Activity, profile string, p MatchParams) []Trace {
	seen := map[MatchKey]bool{}
	var out []Trace
	for _, l := range j.Legs {
		tr, ok := TraceFor(l, LegMode(l, acts), profile, p)
		if !ok || seen[tr.Key] {
			continue
		}
		seen[tr.Key] = true
		out = append(out, tr)
	}
	return out
}

// Match-cache statuses. StatusNoMatch is the remembered negative.
const (
	StatusMatched = "matched"
	StatusNoMatch = "no_match"
)

// CachedMatch is one match_cache row in domain terms.
type CachedMatch struct {
	Status     string
	Points     []domain.LatLng
	DistanceM  float64
	Unplaced   int
	EndsPlaced bool
}

// ApplyMatches lays cached road geometry alongside a journey's recorded
// stretches. A stretch gains SnappedPoints and SnappedKm when all of these
// hold: its mode is not in SkipModes; a matched answer is cached for
// exactly its points; the matcher
// placed both its first and its last point (otherwise the line stops short
// of the stretch); and the road is under MaxRatio times the stretch's
// point-to-point length.
//
// What it never does, which is the amended invariant 6: it does not alter,
// drop or reorder a recorded point; it does not change a leg's kind — the
// stretch stays observed, and a gap stays a gap and is not looked at; it
// does not change DistanceKm or any journey distance, all of which stay
// computed from the recorded points (BRIEF §4 D). A stretch with no
// answer, or one refused by the guard, is left exactly as Assemble made it
// and is drawn point to point.
//
// Pure, like Apply: the lookup map is the only bridge to the cache, and
// the input journey is not mutated.
func ApplyMatches(j journey.Journey, acts []domain.Activity, profile string, p MatchParams, lookup map[MatchKey]CachedMatch) journey.Journey {
	legs := make([]journey.Leg, len(j.Legs))
	copy(legs, j.Legs)
	j.Legs = legs

	j.SnappedObservedKm = 0
	j.SnappedRoadKm = 0
	for i, l := range j.Legs {
		tr, ok := TraceFor(l, LegMode(l, acts), profile, p)
		if !ok {
			continue
		}
		c, ok := lookup[tr.Key]
		if !ok || c.Status != StatusMatched || !c.EndsPlaced || len(c.Points) < 2 {
			continue
		}
		roadKm := c.DistanceM / 1000
		if !(roadKm < p.MaxRatio*l.DistanceKm) {
			continue
		}
		l.SnappedPoints = c.Points
		l.SnappedKm = roadKm
		j.Legs[i] = l
		j.SnappedObservedKm += l.DistanceKm
		j.SnappedRoadKm += roadKm
	}
	return j
}
