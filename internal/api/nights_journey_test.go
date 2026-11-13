package api_test

// Nights on the served journey (phase 19, #14): the demo's six nights,
// pinned through the same upload path a user takes, with the contract's
// promises about the shape — one entry per night, a stop index exactly on
// a night at a stop, hours absent on an unknown one, and the parameters
// echoed.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"roadbook/internal/api"
)

func TestDemoNights(t *testing.T) {
	ts, _, _ := newUploadServer(t)
	body, ctype := multipartUpload(t, demoExport)
	resp := postUpload(t, ts, body, ctype)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("demo upload: HTTP %d", resp.StatusCode)
	}
	accepted := decodeJSON[api.Import](t, resp.Body)
	resp.Body.Close()
	pollImport(t, ts, accepted.Id, func(i api.Import) bool {
		return i.Status == "completed" && i.DetectStatus != nil && *i.DetectStatus == "completed"
	})
	var list api.CandidateList
	fetchJSON(t, ts.URL+"/candidates", &list)

	var got []string
	for n, c := range list.Candidates {
		url := fmt.Sprintf("%s/candidates/%d/journey", ts.URL, c.Id)
		var j api.Journey
		fetchJSON(t, url, &j)

		if want := (api.NightParams{StartHour: 22, EndHour: 6, MinHours: 3}); j.NightParams != want {
			t.Errorf("trip %d: night_params = %+v, want %+v", n+1, j.NightParams, want)
		}
		if len(j.Nights) != j.Summary.CivilDays-1 {
			t.Errorf("trip %d: %d nights for %d civil days; want days-1", n+1, len(j.Nights), j.Summary.CivilDays)
		}
		for _, night := range j.Nights {
			line := fmt.Sprintf("trip %d  %s  %s", n+1, night.Date, night.Kind)
			switch night.Kind {
			case api.NightKindStop:
				if night.StopIndex == nil || night.Hours == nil {
					t.Fatalf("%s: a night at a stop must carry stop_index and hours", line)
				}
				if *night.StopIndex < 0 || *night.StopIndex >= len(j.Stops) {
					t.Fatalf("%s: stop_index %d is outside the journey's %d stops", line, *night.StopIndex, len(j.Stops))
				}
				line += fmt.Sprintf("  stop %d  %.1f h", *night.StopIndex+1, *night.Hours)
				if p := j.Stops[*night.StopIndex].Place; p != nil {
					line += "  " + p.Name
				}
			case api.NightKindMoving:
				if night.StopIndex != nil || night.Hours == nil {
					t.Fatalf("%s: a night on the move carries hours and no stop_index", line)
				}
			default:
				if night.StopIndex != nil || night.Hours != nil {
					t.Fatalf("%s: an unknown night carries neither stop_index nor hours", line)
				}
			}
			got = append(got, line)
		}
	}
	// The places table is empty in this server, so no town is appended:
	// the pin is the nights themselves. The towns are TestDemoStopsAreNamed's.
	want := []string{
		"trip 1  2026-04-24  stop  stop 3  8.0 h",
		"trip 1  2026-04-25  stop  stop 6  8.0 h",
		"trip 2  2026-05-22  stop  stop 1  8.0 h",
		"trip 2  2026-05-23  stop  stop 1  8.0 h",
		"trip 3  2026-06-12  stop  stop 3  8.0 h",
		"trip 3  2026-06-13  stop  stop 5  8.0 h",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("demo nights:\n got %q\nwant %q", got, want)
	}
}

// A day trip's list is on the wire as [], never absent: the page reads the
// length.
func TestNightsEmptyListIsSerialised(t *testing.T) {
	b, err := json.Marshal(api.Journey{Nights: []api.Night{}})
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	if string(raw["nights"]) != "[]" {
		t.Errorf("nights = %s, want []", raw["nights"])
	}
}
