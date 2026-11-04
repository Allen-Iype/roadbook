package timeline

// Synthetic export snippets only — no real data.

import (
	"errors"
	"strings"
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
	obs, st, err := Parse(strings.NewReader(sample))
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

// One case per failure-taxonomy kind: every known wrong input is rejected with
// a message that says what it is. Heads are synthetic minimal stand-ins.
func TestParseRejectsKnownWrongInputs(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		wantKind string
	}{
		{"gzip archive", "\x1f\x8b\x08\x00rest", "gzip"},
		{"zip archive (Takeout download)", "PK\x03\x04rest", "zip"},
		{"PDF", "%PDF-1.7 blah", "pdf"},
		{"JPEG photo", "\xff\xd8\xff\xe0JFIF", "image"},
		{"HEIC photo", "\x00\x00\x00\x18ftypheic", "image"},
		{"archive_browser.html", "<!DOCTYPE html><html><body>Takeout</body>", "html"},
		{"KML from the old Timeline UI", `<?xml version="1.0"?><kml xmlns="x"><Document/></kml>`, "kml"},
		{"encrypted/binary blob", "\x00\x01\x02\x03garbage\x05", "binary"},
		{"plain text", "hello, this is notes.txt", "not-json"},
		{"empty file", "", "empty"},
		{"Semantic Location History (old Takeout)", `{"timelineObjects": [{"placeVisit": {}}]}`, "semantic-history"},
		{"Records.json (old Takeout)", `{"locations": [{"latitudeE7": 100000000, "longitudeE7": 200000000}]}`, "records-json"},
		{"My Activity export", `[{"header": "Maps", "title": "Searched", "titleUrl": "https://x"}]`, "my-activity"},
		{"JSON but not a Timeline export", `{"settings": {}, "devices": []}`, "json-unrecognised"},
		{"truncated mid-array", `{"semanticSegments": [{"startTime": "2025-01-01T10:00:00.000+05:30",`, "truncated"},
		{"JSON array of non-segments", `[1, 2, {"name": "x"}]`, "json-unrecognised"},
		{"empty JSON array", `[]`, "json-unrecognised"},
		{"truncated iPhone array", `[{"startTime": "2025-01-01T10:00:00.000+05:30",`, "truncated"},
		{"a JSON string, not a container", `"geo:1,2"`, "not-json"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := Parse(strings.NewReader(tc.input))
			var ue *UnsupportedInputError
			if !errors.As(err, &ue) {
				t.Fatalf("err = %v, want UnsupportedInputError", err)
			}
			if ue.Kind != tc.wantKind {
				t.Errorf("kind = %q (%s), want %q", ue.Kind, ue.Message, tc.wantKind)
			}
			if ue.Message == "" {
				t.Error("message must never be empty — it is the user-facing explanation")
			}
		})
	}
}

// The supported export must parse even when semanticSegments is not the first
// key and the file carries large sections we skip (rawSignals), and a UTF-8
// BOM must not trip the decoder.
func TestParseSkipsOtherSectionsAndBOM(t *testing.T) {
	input := "\xef\xbb\xbf" + `{
	 "rawSignals": [{"position": {"point": "1.0°, 2.0°"}}, {"wifi": {"scan": [1,2,3]}}],
	 "userLocationProfile": {"frequentPlaces": [{"placeLocation": "3.0°, 4.0°"}]},
	 "semanticSegments": [
	  {"startTime": "2025-01-01T10:00:00.000+05:30",
	   "endTime": "2025-01-01T12:00:00.000+05:30",
	   "visit": {"topCandidate": {"placeLocation": {"latLng": "10.5°, 20.5°"}, "semanticType": "UNKNOWN"}}}
	 ]
	}`
	obs, st, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if st.Visits != 1 || len(obs.Visits) != 1 {
		t.Fatalf("stats = %+v, want exactly 1 visit", st)
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

// The iPhone app's shape of the same export (phase 17, issue #4): a top-level
// array, geo: positions, numbers as strings, path points timed in minutes
// after the segment's start, Google's enum values spelled in words.
const sampleIOS = `[
 {
  "startTime": "2025-01-01T10:00:00.000+05:30",
  "endTime": "2025-01-01T12:00:00.000+05:30",
  "visit": {
   "hierarchyLevel": "0",
   "probability": "0.85",
   "topCandidate": {
    "placeID": "ChIJxyz",
    "semanticType": "Inferred Home",
    "probability": "0.6",
    "placeLocation": "geo:10.500000,20.500000"
   }
  }
 },
 {
  "startTime": "2025-01-01T12:00:00.000+05:30",
  "endTime": "2025-01-01T13:00:00.000+05:30",
  "activity": {
   "start": "geo:10.500000,20.500000",
   "end": "geo:10.600000,20.600000",
   "distanceMeters": "15000.5",
   "topCandidate": {"type": "in passenger vehicle", "probability": "0.000000"}
  }
 },
 {
  "startTime": "2025-01-01T07:30:00Z",
  "endTime": "2025-01-01T09:30:00Z",
  "timelinePath": [
   {"point": "geo:10.700000,20.700000", "durationMinutesOffsetFromStartTime": "15"},
   {"point": "geo:10.800000,20.800000", "durationMinutesOffsetFromStartTime": 30},
   {"point": "geo:10.900000,20.900000", "durationMinutesOffsetFromStartTime": "121"},
   {"point": "geo:11.000000,21.000000", "durationMinutesOffsetFromStartTime": "-1"},
   {"point": "geo:11.100000,21.100000"}
  ]
 },
 {
  "startTime": "2025-01-01T14:00:00.000+05:30",
  "endTime": "2025-01-01T15:00:00.000+05:30",
  "activity": {
   "start": "geo:10.6,20.6",
   "end": "geo:10.7,20.7",
   "distanceMeters": "not a number",
   "topCandidate": {"type": "flying"}
  }
 },
 {
  "startTime": "2025-01-02T10:00:00.000+05:30",
  "endTime": "2025-01-02T11:00:00.000+05:30",
  "timelineMemory": {"destinations": [{"identifier": "x"}], "distanceFromOriginKms": "100"}
 }
]`

func TestParseIOS(t *testing.T) {
	obs, st, err := Parse(strings.NewReader(sampleIOS))
	if err != nil {
		t.Fatal(err)
	}
	if st.Format != PhoneTimelineIOSFormat {
		t.Errorf("format = %q, want %q", st.Format, PhoneTimelineIOSFormat)
	}
	// Visit + activity parse; the activity with an unreadable distance is a
	// type error, counted. Path: 2 points kept; past-end, negative and
	// untimed points skipped.
	if st.Visits != 1 || st.Activities != 1 || st.Points != 2 || st.Skipped != 4 {
		t.Fatalf("stats = %+v, want 1 visit, 1 activity, 2 points, 4 skipped", st)
	}

	v := obs.Visits[0]
	if v.SemanticType != "INFERRED_HOME" {
		t.Errorf("semanticType = %q, want the Android spelling INFERRED_HOME", v.SemanticType)
	}
	if v.Loc == nil || v.Loc.Lat != 10.5 || v.Loc.Lon != 20.5 {
		t.Errorf("visit loc = %v, want (10.5, 20.5)", v.Loc)
	}
	if _, off := v.Start.Zone(); off != 19800 {
		t.Errorf("visit offset = %d s, want +05:30 kept", off)
	}

	a := obs.Activities[0]
	if a.Mode != "IN_PASSENGER_VEHICLE" || a.DistanceM != 15000.5 {
		t.Errorf("activity = %q %v m, want IN_PASSENGER_VEHICLE 15000.5 m", a.Mode, a.DistanceM)
	}
	if a.From == nil || a.To == nil || a.To.Lat != 10.6 {
		t.Errorf("activity ends = %v → %v", a.From, a.To)
	}

	// Offsets resolve against startTime in startTime's own zone (UTC here).
	want := []time.Time{
		time.Date(2025, 1, 1, 7, 45, 0, 0, time.UTC),
		time.Date(2025, 1, 1, 8, 0, 0, 0, time.UTC),
	}
	for i, w := range want {
		if !obs.Points[i].Time.Equal(w) {
			t.Errorf("point %d time = %v, want %v", i, obs.Points[i].Time, w)
		}
	}
	if obs.Points[1].Loc == nil || obs.Points[1].Loc.Lat != 10.8 {
		t.Errorf("point 1 loc = %v, want (10.8, 20.8)", obs.Points[1].Loc)
	}
}

// Normalising Google's spellings must leave the Android export as it was.
func TestNormaliseKind(t *testing.T) {
	for in, want := range map[string]string{
		"INFERRED_HOME":        "INFERRED_HOME",
		"Inferred Home":        "INFERRED_HOME",
		"Searched Address":     "SEARCHED_ADDRESS",
		"in passenger vehicle": "IN_PASSENGER_VEHICLE",
		"flying":               "FLYING",
		"":                     "",
	} {
		if got := normaliseKind(in); got != want {
			t.Errorf("normaliseKind(%q) = %q, want %q", in, got, want)
		}
	}
}
