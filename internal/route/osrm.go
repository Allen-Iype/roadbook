package route

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"roadbook/internal/domain"
)

// PublicOSRMURL is the OSRM project's demo server — a courtesy service, so
// the batch command spaces requests when pointed at it (BRIEF §3G). A
// self-hoster substitutes their own instance URL and drops the interval.
const PublicOSRMURL = "https://router.project-osrm.org"

// OSRM is the one network Router: the same client serves a self-hosted
// instance and the public endpoint — they differ in URL and politeness, not
// protocol (BRIEF §1.1 states this honestly: two configurations, one type).
// It is also the one network Matcher (phase 23): OSRM answers /route and
// /match from the same prepared data.
type OSRM struct {
	BaseURL string
	Profile string
	client  *http.Client
	// matchClient waits longer: a match weighs every point of a stretch
	// against its neighbours, and a long stretch takes seconds where a
	// route takes milliseconds. A timeout is operational — reported, not
	// cached, asked again next run.
	matchClient *http.Client
}

func NewOSRM(baseURL, profile string) *OSRM {
	return &OSRM{
		BaseURL:     baseURL,
		Profile:     profile,
		client:      &http.Client{Timeout: 10 * time.Second},
		matchClient: &http.Client{Timeout: 2 * time.Minute},
	}
}

// osrmResponse is the wire shape; it never escapes this file.
type osrmResponse struct {
	Code   string `json:"code"`
	Routes []struct {
		Geometry struct {
			Coordinates [][]float64 `json:"coordinates"` // GeoJSON: [lon, lat]
		} `json:"geometry"`
		DistanceM float64 `json:"distance"`
		DurationS float64 `json:"duration"`
	} `json:"routes"`
}

func (o *OSRM) Route(ctx context.Context, from, to domain.LatLng) (Route, error) {
	// OSRM speaks lon,lat in the path; the repo speaks lat-first everywhere
	// behind the API, so the flip happens here and nowhere else in Go.
	url := fmt.Sprintf("%s/route/v1/%s/%.4f,%.4f;%.4f,%.4f?overview=full&geometries=geojson&alternatives=false&steps=false",
		o.BaseURL, o.Profile, from.Lon, from.Lat, to.Lon, to.Lat)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Route{}, err
	}
	req.Header.Set("User-Agent", "roadbook (batch route reconstruction)")

	resp, err := o.client.Do(req)
	if err != nil {
		return Route{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return Route{}, err
	}
	// OSRM reports request-level problems as 4xx with a code in the body;
	// decode before judging the status so the message is specific.
	var out osrmResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return Route{}, fmt.Errorf("osrm: HTTP %d, undecodable body: %w", resp.StatusCode, err)
	}
	switch out.Code {
	case "Ok":
		// fall through
	case "NoRoute", "NoSegment":
		// A data answer: the network cannot connect (or even snap) these
		// points. Cacheable; the gap stays visibly unknown.
		return Route{}, ErrNoRoute
	default:
		return Route{}, fmt.Errorf("osrm: HTTP %d code %q", resp.StatusCode, out.Code)
	}
	if len(out.Routes) == 0 {
		return Route{}, fmt.Errorf("osrm: code Ok but no routes")
	}
	r := out.Routes[0]
	pts := make([]domain.LatLng, 0, len(r.Geometry.Coordinates))
	for _, c := range r.Geometry.Coordinates {
		if len(c) < 2 {
			return Route{}, fmt.Errorf("osrm: malformed coordinate in geometry")
		}
		pts = append(pts, domain.LatLng{Lat: c[1], Lon: c[0]})
	}
	return Route{Points: pts, DistanceM: r.DistanceM, DurationS: r.DurationS}, nil
}

// osrmMatchResponse is the /match wire shape; it never escapes this file.
// Tracepoints has one entry per request point, null where the matcher left
// the point off the road as an outlier.
type osrmMatchResponse struct {
	Code      string `json:"code"`
	Matchings []struct {
		Geometry struct {
			Coordinates [][]float64 `json:"coordinates"` // GeoJSON: [lon, lat]
		} `json:"geometry"`
		DistanceM float64 `json:"distance"`
	} `json:"matchings"`
	Tracepoints []*struct{} `json:"tracepoints"`
}

// Match asks OSRM's map-matching service which road a stretch lies on.
//
// The request, option by option: `timestamps` gives the matcher each
// point's time, so it can reject a road that would need an impossible
// speed; `radiuses` is how far from each point it may look; `gaps=ignore`
// stops it cutting the trace at long time gaps (a recorded stretch has
// none longer than the gap threshold by construction — cutting is
// Assemble's job, already done); `tidy=false` keeps every point in the
// request, so tracepoints line up with them one for one;
// `overview=full&geometries=geojson` returns the whole road line as
// coordinates.
//
// One stretch, one line: when the matcher returns more than one matching it
// could not connect the points along one road, and that is ErrNoMatch — a
// data answer. Its own `confidence` figure is not read: measured near zero
// throughout (BRIEF §2.2), it separates nothing.
func (o *OSRM) Match(ctx context.Context, trace []TracePoint, radiusM int) (Match, error) {
	coords := make([]string, len(trace))
	times := make([]string, len(trace))
	radii := make([]string, len(trace))
	r := strconv.Itoa(radiusM)
	for i, p := range trace {
		loc := p.Loc()
		coords[i] = fmt.Sprintf("%.5f,%.5f", loc.Lon, loc.Lat)
		times[i] = strconv.FormatInt(p.Unix, 10)
		radii[i] = r
	}
	url := fmt.Sprintf("%s/match/v1/%s/%s?timestamps=%s&radiuses=%s&gaps=ignore&tidy=false&overview=full&geometries=geojson",
		o.BaseURL, o.Profile, strings.Join(coords, ";"), strings.Join(times, ";"), strings.Join(radii, ";"))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Match{}, err
	}
	req.Header.Set("User-Agent", "roadbook (batch route reconstruction)")

	resp, err := o.matchClient.Do(req)
	if err != nil {
		return Match{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return Match{}, err
	}
	var out osrmMatchResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return Match{}, fmt.Errorf("osrm: HTTP %d, undecodable body: %w", resp.StatusCode, err)
	}
	switch out.Code {
	case "Ok":
		// fall through
	case "NoMatch", "NoSegment":
		return Match{}, ErrNoMatch
	case "TooBig":
		// The server's own limit (osrm-routed --max-matching-size, 100
		// points unless raised), not a fact about the road: never cached.
		return Match{}, fmt.Errorf("osrm: %d points is more than this server matches in one request (raise osrm-routed --max-matching-size)", len(trace))
	default:
		return Match{}, fmt.Errorf("osrm: HTTP %d code %q", resp.StatusCode, out.Code)
	}
	if len(out.Matchings) != 1 {
		return Match{}, ErrNoMatch
	}
	if len(out.Tracepoints) != len(trace) {
		return Match{}, fmt.Errorf("osrm: %d tracepoints for %d points", len(out.Tracepoints), len(trace))
	}
	m := Match{DistanceM: out.Matchings[0].DistanceM}
	for _, tp := range out.Tracepoints {
		if tp == nil {
			m.Unplaced++
		}
	}
	m.EndsPlaced = out.Tracepoints[0] != nil && out.Tracepoints[len(out.Tracepoints)-1] != nil
	m.Points = make([]domain.LatLng, 0, len(out.Matchings[0].Geometry.Coordinates))
	for _, c := range out.Matchings[0].Geometry.Coordinates {
		if len(c) < 2 {
			return Match{}, fmt.Errorf("osrm: malformed coordinate in geometry")
		}
		m.Points = append(m.Points, domain.LatLng{Lat: c[1], Lon: c[0]})
	}
	return m, nil
}
