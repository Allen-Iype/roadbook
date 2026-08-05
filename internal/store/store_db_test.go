package store_test

import (
	"context"
	"testing"
	"time"

	"roadbook/internal/detect"
	"roadbook/internal/domain"
	"roadbook/internal/store"
	"roadbook/internal/store/storetest"
)

var ist = time.FixedZone("", 19800) // +05:30, the source data's usual offset

func at(y int, mo time.Month, d, h int) time.Time {
	return time.Date(y, mo, d, h, 0, 0, 0, ist)
}

// candidate builds a synthetic detect.Candidate. Coordinates are fabricated
// (the testdata/photos convention); nothing here derives from real data.
func candidate(start, end time.Time, dest domain.LatLng) detect.Candidate {
	return detect.Candidate{
		Start: start, End: end,
		Days: end.Sub(start).Hours() / 24,
		Dest: dest, DestKm: 300, TrackKm: 700, Stops: 2, ObsCount: 40,
		Modes: []detect.ModeCount{{Mode: "IN_BUS", N: 3}},
	}
}

func saveRun(t *testing.T, s *store.Store, cands ...detect.Candidate) []store.CandidateRow {
	t.Helper()
	ctx := context.Background()
	_, err := s.SaveRun(ctx, detect.DefaultParams(), detect.Result{
		Bases: []detect.Base{}, Candidates: cands,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, rows, err := s.LatestRun(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != len(cands) {
		t.Fatalf("latest run has %d candidates, want %d", len(rows), len(cands))
	}
	return rows
}

// match reruns exactly the association the API computes: SpanRefs from the
// stored candidates, Anchors from the stored decisions, detect.Match.
func match(cands []store.CandidateRow, decs []store.DecisionRow) map[int64]int64 {
	refs := make([]detect.SpanRef, len(cands))
	for i, c := range cands {
		refs[i] = detect.SpanRef{ID: c.ID, Start: c.SpanStart, End: c.SpanEnd, Dest: c.Dest}
	}
	anchors := make([]detect.Anchor, len(decs))
	for i, d := range decs {
		anchors[i] = detect.Anchor{ID: d.ID, Start: d.AnchorStart, End: d.AnchorEnd, Dest: d.AnchorDest, CreatedAt: d.CreatedAt}
	}
	return detect.Match(refs, anchors, detect.DefaultMatchParams())
}

// TestDecisionsSurviveRedetection is the invariant the two-table design
// exists for: candidates are disposable and renumber on every run; decisions
// are user data and must re-attach to the right candidate by anchor. A
// regression here silently destroys the user's triage rather than erroring —
// which is why it is asserted against the real tables, not a mock.
func TestDecisionsSurviveRedetection(t *testing.T) {
	s := storetest.Open(t)
	ctx := context.Background()

	// Run 1: two adventures close in time — A and B — plus C, undecided.
	destA := domain.LatLng{Lat: 12.30, Lon: 45.60}
	destB := domain.LatLng{Lat: 12.90, Lon: 45.10} // ~85 km from A: distinct, but both within MatchKm of a sloppy anchor
	destC := domain.LatLng{Lat: 15.00, Lon: 48.00}
	run1 := saveRun(t, s,
		candidate(at(2026, 3, 1, 8), at(2026, 3, 4, 20), destA),
		candidate(at(2026, 3, 5, 8), at(2026, 3, 8, 20), destB),
		candidate(at(2026, 4, 1, 8), at(2026, 4, 3, 20), destC),
	)

	nameA := "Trip A"
	decA, err := s.InsertDecision(ctx, "confirmed", &nameA, run1[0])
	if err != nil {
		t.Fatal(err)
	}
	decB, err := s.InsertDecision(ctx, "dismissed", nil, run1[1])
	if err != nil {
		t.Fatal(err)
	}

	// Offsets must survive the round trip: matching compares instants, and
	// detection's civil-date logic needs the writer's offset back.
	if got := run1[0].SpanStart; !got.Equal(at(2026, 3, 1, 8)) {
		t.Errorf("span start round-trip: %v, want %v", got, at(2026, 3, 1, 8))
	}
	if _, off := run1[0].SpanStart.Zone(); off != 19800 {
		t.Errorf("span offset round-trip: %d, want 19800", off)
	}

	// Run 2: different parameters moved every boundary by hours and each
	// destination by a few km; candidates renumbered (new rows, new ids).
	shift := 5 * time.Hour
	nudge := func(p domain.LatLng) domain.LatLng { return domain.LatLng{Lat: p.Lat + 0.05, Lon: p.Lon - 0.03} } // ~7 km
	run2 := saveRun(t, s,
		candidate(at(2026, 3, 1, 8).Add(shift), at(2026, 3, 4, 20).Add(-shift), nudge(destA)),
		candidate(at(2026, 3, 5, 8).Add(-shift), at(2026, 3, 8, 20).Add(shift), nudge(destB)),
		candidate(at(2026, 4, 1, 8), at(2026, 4, 3, 20), destC),
		candidate(at(2026, 5, 1, 8), at(2026, 5, 2, 20), domain.LatLng{Lat: 16, Lon: 49}), // new
	)

	decs, err := s.ListDecisions(ctx)
	if err != nil {
		t.Fatal(err)
	}
	m := match(run2, decs)
	if m[run2[0].ID] != decA.ID {
		t.Errorf("decision A attached to candidate %d's decision %d, want A' (%d→%d)", run2[0].ID, m[run2[0].ID], run2[0].ID, decA.ID)
	}
	if m[run2[1].ID] != decB.ID {
		t.Errorf("decision B did not re-attach to B': got %v", m)
	}
	if _, ok := m[run2[2].ID]; ok {
		t.Errorf("undecided candidate C acquired a decision: %v", m)
	}
	if _, ok := m[run2[3].ID]; ok {
		t.Errorf("new candidate D acquired a decision: %v", m)
	}

	// Run 3: A's span no longer exists (out of tolerance). The decision must
	// orphan visibly — never attach to the nearest wrong candidate.
	run3 := saveRun(t, s,
		candidate(at(2026, 6, 1, 8), at(2026, 6, 4, 20), destA), // same place, months later
		candidate(at(2026, 3, 5, 8), at(2026, 3, 8, 20), destB),
	)
	m = match(run3, decs)
	for candID, decID := range m {
		if decID == decA.ID {
			t.Errorf("decision A attached to candidate %d despite no overlapping span — must orphan instead", candID)
		}
	}
	if m[run3[1].ID] != decB.ID {
		t.Errorf("decision B lost its attachment in run 3: %v", m)
	}
}

func TestImportIdempotency(t *testing.T) {
	s := storetest.Open(t)
	ctx := context.Background()

	loc := func(lat, lon float64) *domain.LatLng { return &domain.LatLng{Lat: lat, Lon: lon} }
	obs := domain.Observations{
		Visits: []domain.Visit{
			{Start: at(2026, 3, 1, 9), End: at(2026, 3, 1, 11), Loc: loc(12.1, 45.1), SemanticType: "INFERRED_HOME"},
			{Start: at(2026, 3, 2, 9), End: at(2026, 3, 2, 10), Loc: loc(12.2, 45.2)},
		},
		Activities: []domain.Activity{
			{Start: at(2026, 3, 1, 11), End: at(2026, 3, 1, 12), From: loc(12.1, 45.1), To: loc(12.2, 45.2), DistanceM: 15000, Mode: "IN_BUS"},
		},
		Points: []domain.PathPoint{
			{Time: at(2026, 3, 1, 11), Loc: loc(12.15, 45.15)},
			{Time: at(2026, 3, 1, 12), Loc: loc(12.18, 45.18)},
		},
		RawPositions: []domain.RawPosition{
			{Time: at(2026, 3, 1, 11), Loc: loc(12.16, 45.16), AccuracyM: 16, Source: "GPS"},
		},
	}

	id1, err := s.BeginImport(ctx, "test", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	first, err := s.ImportObservations(ctx, id1, "phone-timeline", obs, 0)
	if err != nil {
		t.Fatal(err)
	}
	if first.Parsed != 6 || first.Inserted != 6 {
		t.Errorf("first import parsed/inserted = %d/%d, want 6/6", first.Parsed, first.Inserted)
	}

	id2, err := s.BeginImport(ctx, "test-again", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.ImportObservations(ctx, id2, "phone-timeline", obs, 0)
	if err != nil {
		t.Fatal(err)
	}
	if second.Inserted != 0 {
		t.Errorf("re-import inserted %d rows, want 0 — idempotency is the point", second.Inserted)
	}

	back, err := s.LoadObservations(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(back.Visits) != 2 || len(back.Activities) != 1 || len(back.Points) != 2 || len(back.RawPositions) != 1 {
		t.Errorf("loaded %d/%d/%d/%d rows, want 2/1/2/1",
			len(back.Visits), len(back.Activities), len(back.Points), len(back.RawPositions))
	}
}

// TestImportBookkeeping pins the phase 5 lifecycle (BRIEF §3B): the row exists
// as 'running' before observations land, finalises to 'completed' with the
// sniffer's format label, and a failure records the label queryably beside the
// prose message — the label, not the message, is the legacy-trigger evidence.
func TestImportBookkeeping(t *testing.T) {
	s := storetest.Open(t)
	ctx := context.Background()

	okID, err := s.BeginImport(ctx, "good.json", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.ImportObservations(ctx, okID, "phone-timeline", domain.Observations{
		Visits: []domain.Visit{{Start: at(2026, 3, 1, 9), End: at(2026, 3, 1, 10)}},
	}, 2); err != nil {
		t.Fatal(err)
	}

	failID, err := s.BeginImport(ctx, "old-takeout.json", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.FailImport(ctx, failID, "records-json", "this is Records.json — not supported"); err != nil {
		t.Fatal(err)
	}

	// A failure before the input was recognised stores NULL, not "".
	blindID, err := s.BeginImport(ctx, "garbage.bin", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.FailImport(ctx, blindID, "", "not JSON"); err != nil {
		t.Fatal(err)
	}

	rows, err := s.ListImports(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("ListImports returned %d rows, want 3", len(rows))
	}
	byID := map[int64]store.ImportRow{}
	for _, r := range rows {
		byID[r.ID] = r
	}
	ok := byID[okID]
	if ok.Status != "completed" || ok.DetectedFormat == nil || *ok.DetectedFormat != "phone-timeline" || ok.Visits != 1 || ok.Skipped != 2 {
		t.Errorf("completed row = %+v, want completed/phone-timeline with counters 1 visit, 2 skipped", ok)
	}
	fl := byID[failID]
	if fl.Status != "failed" || fl.DetectedFormat == nil || *fl.DetectedFormat != "records-json" || fl.Error == nil {
		t.Errorf("failed row = %+v, want failed/records-json with an error message", fl)
	}
	bl := byID[blindID]
	if bl.Status != "failed" || bl.DetectedFormat != nil {
		t.Errorf("unrecognised-input row = %+v, want failed with NULL detected_format", bl)
	}

	// Finalising a row that is not running is a loud error, not a silent no-op.
	if _, err := s.ImportObservations(ctx, failID, "phone-timeline", domain.Observations{}, 0); err == nil {
		t.Error("ImportObservations on a failed row succeeded — want an error")
	}
}

func TestPhotoRoundTrip(t *testing.T) {
	s := storetest.Open(t)
	ctx := context.Background()

	rows := saveRun(t, s, candidate(at(2026, 3, 1, 8), at(2026, 3, 4, 20), domain.LatLng{Lat: 12.3, Lon: 45.6}))
	name := "Trip"
	dec, err := s.InsertDecision(ctx, "confirmed", &name, rows[0])
	if err != nil {
		t.Fatal(err)
	}

	taken := at(2026, 3, 2, 14)
	off := 19800
	lat, lon := 12.3456, 45.6789
	full := store.PhotoRow{
		DecisionID: dec.ID, ContentHash: "hash-full", OriginalName: "a.jpg",
		TakenAt: &taken, TakenOffsetSec: &off, TimeSource: "gps",
		Lat: &lat, Lon: &lon, PosSource: "exif", ThumbW: 512, ThumbH: 384,
	}
	stored, inserted, err := s.InsertPhoto(ctx, full)
	if err != nil {
		t.Fatal(err)
	}
	if !inserted {
		t.Fatal("first insert reported duplicate")
	}
	if !stored.TakenAt.Equal(taken) {
		t.Errorf("taken_at round-trip: %v, want %v", stored.TakenAt, taken)
	}
	if _, gotOff := stored.TakenAt.Zone(); gotOff != off {
		t.Errorf("taken_at offset restored as %d, want %d — display needs the civil offset back", gotOff, off)
	}

	// The duplicate path: same bytes, different name and adventure — the
	// original row wins, unchanged. (ON CONFLICT DO NOTHING returns no row;
	// the refetch must find the original, not error.)
	dup := full
	dup.OriginalName = "renamed.jpg"
	got, inserted, err := s.InsertPhoto(ctx, dup)
	if err != nil {
		t.Fatal(err)
	}
	if inserted || got.ID != stored.ID || got.OriginalName != "a.jpg" {
		t.Errorf("duplicate insert: inserted=%v id=%d name=%q, want the untouched original", inserted, got.ID, got.OriginalName)
	}

	// A metadata-free photo: every nullable stays null, sources say none.
	bare := store.PhotoRow{
		DecisionID: dec.ID, ContentHash: "hash-bare", OriginalName: "b.jpg",
		TimeSource: "none", PosSource: "none", ThumbW: 100, ThumbH: 80,
	}
	if _, _, err := s.InsertPhoto(ctx, bare); err != nil {
		t.Fatal(err)
	}
	list, err := s.ListPhotos(ctx, dec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].ID != stored.ID {
		t.Fatalf("list = %d rows, want 2 in upload order", len(list))
	}
	if b := list[1]; b.TakenAt != nil || b.TakenOffsetSec != nil || b.Lat != nil || b.Lon != nil {
		t.Errorf("bare photo grew values: %+v", b)
	}

	// Get / delete / gone.
	p, err := s.GetPhoto(ctx, stored.ID)
	if err != nil || p == nil || p.ContentHash != "hash-full" {
		t.Fatalf("GetPhoto = %+v, %v", p, err)
	}
	okDel, err := s.DeletePhoto(ctx, stored.ID)
	if err != nil || !okDel {
		t.Fatalf("delete: %v ok=%v", err, okDel)
	}
	p, err = s.GetPhoto(ctx, stored.ID)
	if err != nil || p != nil {
		t.Errorf("photo still readable after delete: %+v, %v", p, err)
	}
	okDel, err = s.DeletePhoto(ctx, stored.ID)
	if err != nil || okDel {
		t.Errorf("second delete reported ok=%v, want false", okDel)
	}
}
