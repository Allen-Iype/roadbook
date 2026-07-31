// Package timeline parses Google Timeline exports into domain types. No type in
// this package escapes it (CLAUDE.md invariant 4). Parsing is defensive
// throughout: Google changes this schema without announcement, so a malformed
// segment is counted and skipped, never fatal.
package timeline

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"roadbook/internal/domain"
)

// Stats reports what one parse saw. Skipped counts segments or points that were
// present but unusable (bad JSON, missing/invalid timestamps).
type Stats struct {
	Visits     int
	Activities int
	Points     int
	Skipped    int
}

type rawFile struct {
	SemanticSegments []json.RawMessage `json:"semanticSegments"`
}

type rawSegment struct {
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
	Visit     *struct {
		TopCandidate struct {
			PlaceLocation json.RawMessage `json:"placeLocation"`
			SemanticType  string          `json:"semanticType"`
		} `json:"topCandidate"`
	} `json:"visit"`
	Activity *struct {
		Start          json.RawMessage `json:"start"`
		End            json.RawMessage `json:"end"`
		DistanceMeters float64         `json:"distanceMeters"`
		TopCandidate   struct {
			Type string `json:"type"`
		} `json:"topCandidate"`
	} `json:"activity"`
	TimelinePath []struct {
		Point json.RawMessage `json:"point"`
		Time  string          `json:"time"`
	} `json:"timelinePath"`
}

// Parse reads a Timeline export. It returns every visit, activity, and path point
// in source-file order — including visits and activities whose coordinates could
// not be parsed (Loc/From/To nil), because downstream logic needs their timeline
// positions even without a location.
func Parse(data []byte) (domain.Observations, Stats, error) {
	var f rawFile
	if err := json.Unmarshal(data, &f); err != nil {
		return domain.Observations{}, Stats{}, fmt.Errorf("timeline: not a Timeline export: %w", err)
	}
	var obs domain.Observations
	var st Stats
	for _, raw := range f.SemanticSegments {
		var seg rawSegment
		if err := json.Unmarshal(raw, &seg); err != nil {
			st.Skipped++
			continue
		}
		switch {
		case seg.Visit != nil:
			start, err1 := parseTime(seg.StartTime)
			end, err2 := parseTime(seg.EndTime)
			if err1 != nil || err2 != nil {
				st.Skipped++
				continue
			}
			obs.Visits = append(obs.Visits, domain.Visit{
				Start:        start,
				End:          end,
				Loc:          parseLoc(seg.Visit.TopCandidate.PlaceLocation),
				SemanticType: seg.Visit.TopCandidate.SemanticType,
			})
			st.Visits++
		case seg.Activity != nil:
			start, err1 := parseTime(seg.StartTime)
			end, err2 := parseTime(seg.EndTime)
			if err1 != nil || err2 != nil {
				st.Skipped++
				continue
			}
			obs.Activities = append(obs.Activities, domain.Activity{
				Start:     start,
				End:       end,
				From:      parseLoc(seg.Activity.Start),
				To:        parseLoc(seg.Activity.End),
				DistanceM: seg.Activity.DistanceMeters,
				Mode:      seg.Activity.TopCandidate.Type,
			})
			st.Activities++
		case seg.TimelinePath != nil:
			for _, pt := range seg.TimelinePath {
				if pt.Time == "" {
					st.Skipped++
					continue
				}
				t, err := parseTime(pt.Time)
				if err != nil {
					st.Skipped++
					continue
				}
				obs.Points = append(obs.Points, domain.PathPoint{Time: t, Loc: parseLoc(pt.Point)})
				st.Points++
			}
		}
		// Other segment kinds (timelineMemory, …) are ignored, matching the
		// reference detector.
	}
	return obs, st, nil
}

func parseTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}

// parseLoc accepts the two shapes coordinates take in the export: a bare string
// `"17.0°, 78.0°"` or an object `{"latLng": "17.0°, 78.0°"}`. Anything else
// yields nil.
func parseLoc(raw json.RawMessage) *domain.LatLng {
	if len(raw) == 0 {
		return nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		var obj struct {
			LatLng string `json:"latLng"`
		}
		if err := json.Unmarshal(raw, &obj); err != nil || obj.LatLng == "" {
			return nil
		}
		s = obj.LatLng
	}
	parts := strings.Split(strings.ReplaceAll(s, "°", ""), ",")
	if len(parts) != 2 {
		return nil
	}
	lat, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	lon, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err1 != nil || err2 != nil {
		return nil
	}
	return &domain.LatLng{Lat: lat, Lon: lon}
}
