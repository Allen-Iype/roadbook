package timeline

// Synthetic export snippets only — no real data.

import (
	"testing"
	"time"
)

const sample = `{
 "semanticSegments": [
  {
   "startTime": "2025-01-01T10:00:00.000+05:30",
   "endTime": "2025-01-01T12:00:00.000+05:30",
   "visit": {
    "topCandidate": {
     "placeLocation": {"latLng": "10.5°, 20.5°"},
     "semanticType": "INFERRED_HOME"
    }
   }
  },
  {
   "startTime": "2025-01-01T12:00:00.000+05:30",
   "endTime": "2025-01-01T13:00:00.000+05:30",
   "activity": {
    "start": {"latLng": "10.5°, 20.5°"},
    "end": {"latLng": "10.6°, 20.6°"},
    "distanceMeters": 15000.0,
    "topCandidate": {"type": "IN_PASSENGER_VEHICLE", "probability": 0.0}
   }
  },
  {
   "startTime": "2025-01-01T13:00:00.000+05:30",
   "endTime": "2025-01-01T14:00:00.000+05:30",
   "timelinePath": [
    {"point": "10.7°, 20.7°", "time": "2025-01-01T13:15:00.000+05:30"},
    {"point": "not a coordinate", "time": "2025-01-01T13:30:00.000+05:30"},
    {"point": "10.8°, 20.8°"}
   ]
  },
  {
   "startTime": "garbage",
   "endTime": "2025-01-01T15:00:00.000+05:30",
   "visit": {"topCandidate": {"placeLocation": {"latLng": "1.0°, 2.0°"}}}
  },
  {
   "startTime": "2025-01-02T10:00:00.000+05:30",
   "endTime": "2025-01-02T11:00:00.000+05:30",
   "timelineMemory": {"trip": {"distanceFromOriginKms": 100}}
  }
 ]
}`

func TestParse(t *testing.T) {
	obs, st, err := Parse([]byte(sample))
	if err != nil {
		t.Fatal(err)
	}
	if st.Visits != 1 || st.Activities != 1 || st.Points != 2 {
		t.Fatalf("stats = %+v, want 1 visit, 1 activity, 2 points", st)
	}
	// Skipped: the garbage-timestamp visit and the timeless path point. The
	// timelineMemory segment is ignored, not skipped.
	if st.Skipped != 2 {
		t.Errorf("skipped = %d, want 2", st.Skipped)
	}

	v := obs.Visits[0]
	if v.Loc == nil || v.Loc.Lat != 10.5 || v.Loc.Lon != 20.5 {
		t.Errorf("visit loc = %+v, want (10.5, 20.5)", v.Loc)
	}
	if v.SemanticType != "INFERRED_HOME" {
		t.Errorf("semanticType = %q", v.SemanticType)
	}
	// The +05:30 offset must survive parsing: home-base eras depend on civil
	// dates taken in the timestamp's own zone.
	if _, off := v.Start.Zone(); off != 5*3600+1800 {
		t.Errorf("visit start offset = %d, want +05:30", off)
	}
	if !v.Start.Equal(time.Date(2025, 1, 1, 4, 30, 0, 0, time.UTC)) {
		t.Errorf("visit start = %v", v.Start)
	}

	a := obs.Activities[0]
	if a.From == nil || a.To == nil || a.DistanceM != 15000 || a.Mode != "IN_PASSENGER_VEHICLE" {
		t.Errorf("activity = %+v", a)
	}

	if obs.Points[0].Loc == nil || obs.Points[0].Loc.Lat != 10.7 {
		t.Errorf("point 0 = %+v", obs.Points[0])
	}
	// A bad coordinate with a good timestamp is kept with a nil Loc, matching
	// the reference detector's accounting.
	if obs.Points[1].Loc != nil {
		t.Errorf("point 1 loc = %+v, want nil", obs.Points[1].Loc)
	}
}

func TestProbe(t *testing.T) {
	paths, err := Probe([]byte(sample))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]int{
		"semanticSegments[].visit.topCandidate.placeLocation.latLng": 2,
		"semanticSegments[].timelinePath[].point":                    3,
		"semanticSegments[].startTime":                               5,
	}
	got := map[string]int{}
	for _, pc := range paths {
		got[pc.Path] = pc.Count
	}
	for p, n := range want {
		if got[p] != n {
			t.Errorf("path %s = %d, want %d", p, got[p], n)
		}
	}
}
