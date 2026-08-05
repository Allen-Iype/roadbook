// Package timeline parses Google Timeline exports into domain types. No type in
// this package escapes it (CLAUDE.md invariant 4). Parsing is defensive
// throughout: Google changes this schema without announcement, so a malformed
// segment is counted and skipped, never fatal — but a file that is not the
// supported export at all is rejected with a message that says what it is and
// what to do instead (the failure taxonomy; PLAN.md phase 1 checkpoint 5).
package timeline

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"roadbook/internal/domain"
)

// PhoneTimelineFormat is the stable slug for the supported input, the current
// on-device phone export. It shares a namespace with UnsupportedInputError.Kind:
// together they are the sniffer's format taxonomy, recorded per import in
// imports.detected_format (phase 5 BRIEF §3B) so format populations are
// queryable — the label is evidence, the Message is prose.
const PhoneTimelineFormat = "phone-timeline"

// Stats reports what one parse saw. Skipped counts segments or points that were
// present but unusable (bad shape, missing/invalid timestamps). Format is the
// recognised input's stable slug (PhoneTimelineFormat on success — the only
// format that parses today).
type Stats struct {
	Visits       int
	Activities   int
	Points       int
	RawPositions int
	Skipped      int
	Format       string
}

// UnsupportedInputError is a recognised-but-wrong input. Message is written for
// the person who chose the file: what this file is, and what to do instead.
type UnsupportedInputError struct {
	Kind    string // stable slug: "gzip", "records-json", "my-activity", …
	Message string
}

func (e *UnsupportedInputError) Error() string { return e.Message }

// exportHint is the one true way to produce the supported export. Timeline
// moved on-device in 2024; this path is where every message points.
const exportHint = "export Timeline data on the phone: Settings → Location → Location Services → Timeline → Export Timeline data"

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

// Parse reads a Timeline export as a stream: constant memory regardless of
// file size (real Takeouts reach gigabytes). It returns every visit, activity,
// and path point in source-file order — including ones whose coordinates could
// not be parsed (Loc/From/To nil), because downstream logic needs their
// timeline positions even without a location.
func Parse(r io.Reader) (domain.Observations, Stats, error) {
	var obs domain.Observations
	var st Stats

	br := bufio.NewReaderSize(r, 64*1024)
	head, err := br.Peek(4096)
	if len(head) == 0 {
		if err != nil && err != io.EOF {
			return obs, st, err
		}
		return obs, st, &UnsupportedInputError{Kind: "empty", Message: "the file is empty; " + exportHint}
	}
	if ue := sniff(head); ue != nil {
		return obs, st, ue
	}
	if bytes.HasPrefix(head, []byte{0xEF, 0xBB, 0xBF}) {
		br.Discard(3) // UTF-8 BOM would trip the JSON decoder
	}

	dec := json.NewDecoder(br)
	if err := expectDelim(dec, '{'); err != nil {
		return obs, st, err
	}
	var topKeys []string
	found := false
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return obs, st, truncated(err)
		}
		key, ok := keyTok.(string)
		if !ok {
			return obs, st, malformed(fmt.Errorf("expected object key, got %v", keyTok))
		}
		topKeys = append(topKeys, key)
		switch key {
		case "semanticSegments":
			found = true
			if err := expectDelim(dec, '['); err != nil {
				return obs, st, err
			}
			for dec.More() {
				var seg rawSegment
				if err := dec.Decode(&seg); err != nil {
					// A type mismatch consumes the value and leaves the decoder
					// usable: count and continue (defensive-parse rule). A syntax
					// error or EOF means the stream itself is broken: stop.
					var typeErr *json.UnmarshalTypeError
					if errors.As(err, &typeErr) {
						st.Skipped++
						continue
					}
					return obs, st, truncated(err)
				}
				processSegment(seg, &obs, &st)
			}
			if _, err := dec.Token(); err != nil { // consume ']'
				return obs, st, truncated(err)
			}
		case "rawSignals":
			if err := parseRawSignals(dec, &obs, &st); err != nil {
				return obs, st, err
			}
		default:
			if err := skipValue(dec); err != nil {
				return obs, st, truncated(err)
			}
		}
	}
	if _, err := dec.Token(); err != nil { // consume '}'
		return obs, st, truncated(err)
	}

	if !found {
		return obs, st, &UnsupportedInputError{
			Kind: "json-unrecognised",
			Message: fmt.Sprintf(
				"JSON, but with no semanticSegments section — top-level keys found: %s; this is not a phone Timeline export; %s",
				strings.Join(topKeys, ", "), exportHint),
		}
	}
	st.Format = PhoneTimelineFormat
	return obs, st, nil
}

func processSegment(seg rawSegment, obs *domain.Observations, st *Stats) {
	switch {
	case seg.Visit != nil:
		start, err1 := parseTime(seg.StartTime)
		end, err2 := parseTime(seg.EndTime)
		if err1 != nil || err2 != nil {
			st.Skipped++
			return
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
			return
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

// rawSignal is one rawSignals array entry. Only position records carry
// observations; the other kinds (activityRecord, wifiScan, …) are recognised
// by their absence of a position key and ignored.
type rawSignal struct {
	Position *struct {
		LatLng         json.RawMessage `json:"LatLng"`
		AccuracyMeters float64         `json:"accuracyMeters"`
		Source         string          `json:"source"`
		Timestamp      string          `json:"timestamp"`
	} `json:"position"`
}

// parseRawSignals streams the rawSignals array with the same defensive rules
// as semanticSegments: a malformed entry is counted and skipped, a broken
// stream is fatal.
func parseRawSignals(dec *json.Decoder, obs *domain.Observations, st *Stats) error {
	if err := expectDelim(dec, '['); err != nil {
		return err
	}
	for dec.More() {
		var sig rawSignal
		if err := dec.Decode(&sig); err != nil {
			var typeErr *json.UnmarshalTypeError
			if errors.As(err, &typeErr) {
				st.Skipped++
				continue
			}
			return truncated(err)
		}
		if sig.Position == nil {
			continue
		}
		t, err := parseTime(sig.Position.Timestamp)
		if err != nil {
			st.Skipped++
			continue
		}
		obs.RawPositions = append(obs.RawPositions, domain.RawPosition{
			Time:      t,
			Loc:       parseLoc(sig.Position.LatLng),
			AccuracyM: sig.Position.AccuracyMeters,
			Source:    sig.Position.Source,
		})
		st.RawPositions++
	}
	if _, err := dec.Token(); err != nil { // consume ']'
		return truncated(err)
	}
	return nil
}

// sniff classifies the head of the file. It rejects only inputs that are
// *definitely* something else; anything merely unrecognised falls through to
// the streaming parser, whose verdict (with the actual top-level keys) is
// authoritative. The rules are a port of field-tested detection from the
// Dawarich comparison (docs/feature-comparison-dawarich-roadbook.md §1.1).
func sniff(head []byte) *UnsupportedInputError {
	reject := func(kind, what string) *UnsupportedInputError {
		return &UnsupportedInputError{Kind: kind, Message: what + "; " + exportHint}
	}

	// Binary container and document magics first.
	switch {
	case bytes.HasPrefix(head, []byte{0x1f, 0x8b}):
		return reject("gzip", "this is a gzip archive, not a Timeline JSON file — decompress it and import the JSON inside")
	case bytes.HasPrefix(head, []byte("PK\x03\x04")):
		return reject("zip", "this is a ZIP archive (a Takeout download?) — extract it and import the Timeline JSON file inside")
	case bytes.HasPrefix(head, []byte("%PDF")):
		return reject("pdf", "this is a PDF, not a location export")
	case bytes.HasPrefix(head, []byte{0xFF, 0xD8, 0xFF}), bytes.HasPrefix(head, []byte("\x89PNG")),
		len(head) > 11 && bytes.Equal(head[4:8], []byte("ftyp")):
		return reject("image", "this is an image file, not a location export")
	}

	trimmed := bytes.TrimLeft(head, " \t\r\n\xef\xbb\xbf")
	lower := bytes.ToLower(trimmed)

	switch {
	case bytes.HasPrefix(lower, []byte("<!doctype html")), bytes.HasPrefix(lower, []byte("<html")):
		return reject("html", "this is a web page — Takeout's archive_browser.html is a table of contents, not data; import the Timeline JSON file instead")
	case bytes.HasPrefix(trimmed, []byte("<?xml")):
		if bytes.Contains(lower, []byte("<kml")) {
			return reject("kml", "this is KML from the old Timeline interface, which Roadbook does not read")
		}
		return reject("xml", "this is XML, not a Timeline JSON export")
	}

	if len(trimmed) > 0 && trimmed[0] != '{' && trimmed[0] != '[' {
		if looksBinary(head) {
			return reject("binary", "unreadable binary data — possibly an encrypted Timeline backup, which cannot be imported; instead")
		}
		return reject("not-json", fmt.Sprintf("not JSON (the file starts with %q)", preview(trimmed)))
	}

	// JSON: recognise the known wrong products by their distinctive markers.
	s := string(head)
	switch {
	case strings.Contains(s, `"semanticSegments"`), strings.Contains(s, `"rawSignals"`):
		return nil // the supported phone export
	case strings.Contains(s, `"timelineObjects"`):
		return reject("semantic-history",
			"this is Semantic Location History from an old Google Takeout (monthly timelineObjects files) — not supported; for current data")
	case strings.Contains(s, `"latitudeE7"`), strings.Contains(s, `"longitudeE7"`):
		return reject("records-json",
			"this is Records.json from an old Google Takeout (raw location samples) — not supported; for current data")
	case trimmed[0] == '[' && strings.Contains(s, `"titleUrl"`):
		return reject("my-activity",
			"this is a Google My Activity export — a different Takeout product with no location timeline in it")
	}
	// JSON with no marker in the first 4 KB: let the streaming parser decide.
	return nil
}

func looksBinary(head []byte) bool {
	n := min(len(head), 512)
	for _, b := range head[:n] {
		if b < 0x09 || (b > 0x0d && b < 0x20) {
			return true
		}
	}
	return false
}

func preview(b []byte) string {
	n := min(len(b), 24)
	return string(b[:n])
}

// skipValue consumes exactly one JSON value from the decoder without building
// it in memory — this is what keeps memory constant while skipping sections
// like rawSignals that can dwarf the one we read.
func skipValue(dec *json.Decoder) error {
	t, err := dec.Token()
	if err != nil {
		return err
	}
	if d, ok := t.(json.Delim); ok && (d == '{' || d == '[') {
		depth := 1
		for depth > 0 {
			t, err := dec.Token()
			if err != nil {
				return err
			}
			if d, ok := t.(json.Delim); ok {
				switch d {
				case '{', '[':
					depth++
				case '}', ']':
					depth--
				}
			}
		}
	}
	return nil
}

func expectDelim(dec *json.Decoder, want json.Delim) error {
	t, err := dec.Token()
	if err != nil {
		return truncated(err)
	}
	if d, ok := t.(json.Delim); !ok || d != want {
		return malformed(fmt.Errorf("expected %q, got %v", want, t))
	}
	return nil
}

func truncated(err error) error {
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return &UnsupportedInputError{
			Kind:    "truncated",
			Message: "the file ends mid-record — the export or the copy of it was cut short; re-export or re-copy the file",
		}
	}
	return malformed(err)
}

func malformed(err error) error {
	return &UnsupportedInputError{
		Kind:    "malformed-json",
		Message: fmt.Sprintf("the file is not valid JSON (%v); if it was edited or converted, start from a fresh export — %s", err, exportHint),
	}
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
