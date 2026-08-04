package route

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
type OSRM struct {
	BaseURL string
	Profile string
	client  *http.Client
}

func NewOSRM(baseURL, profile string) *OSRM {
	return &OSRM{
		BaseURL: baseURL,
		Profile: profile,
		client:  &http.Client{Timeout: 10 * time.Second},
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
