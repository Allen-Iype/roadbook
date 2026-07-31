package detect

import (
	"testing"
	"time"

	"roadbook/internal/domain"
)

// Synthetic data only, as in detect_test.go.

func TestMatch(t *testing.T) {
	day := func(d int) time.Time { return time.Date(2025, 3, d, 0, 0, 0, 0, time.UTC) }
	dest := domain.LatLng{Lat: 11, Lon: 20}
	nearDest := domain.LatLng{Lat: 11.1, Lon: 20} // ≈11 km away — inside MATCH_KM
	farDest := domain.LatLng{Lat: 12, Lon: 20}    // ≈111 km away — outside MATCH_KM

	cases := []struct {
		name  string
		cands []SpanRef
		decs  []Anchor
		want  map[int64]int64
	}{
		{
			name:  "identical span and destination match",
			cands: []SpanRef{{ID: 1, Start: day(1), End: day(8), Dest: dest}},
			decs:  []Anchor{{ID: 10, Start: day(1), End: day(8), Dest: dest, CreatedAt: day(9)}},
			want:  map[int64]int64{1: 10},
		},
		{
			name: "shifted boundaries still match — the bug 4 re-import case",
			// The anchor was recorded on a truncated 4-day span; re-import with
			// a wider window extends the candidate days earlier. Overlap and a
			// nearby destination carry the decision across.
			cands: []SpanRef{{ID: 1, Start: day(1), End: day(12), Dest: nearDest}},
			decs:  []Anchor{{ID: 10, Start: day(5), End: day(12), Dest: dest, CreatedAt: day(13)}},
			want:  map[int64]int64{1: 10},
		},
		{
			name:  "no time overlap means no match, even at the same destination",
			cands: []SpanRef{{ID: 1, Start: day(1), End: day(5), Dest: dest}},
			decs:  []Anchor{{ID: 10, Start: day(10), End: day(15), Dest: dest, CreatedAt: day(16)}},
			want:  map[int64]int64{},
		},
		{
			name:  "destination beyond MATCH_KM means no match, even with overlap",
			cands: []SpanRef{{ID: 1, Start: day(1), End: day(8), Dest: farDest}},
			decs:  []Anchor{{ID: 10, Start: day(1), End: day(8), Dest: dest, CreatedAt: day(9)}},
			want:  map[int64]int64{},
		},
		{
			name: "split: the decision follows the better-overlapping half",
			// One decided 10-day span re-detects as two candidates; the second
			// covers more of the anchor. The first half re-asks for triage.
			cands: []SpanRef{
				{ID: 1, Start: day(1), End: day(4), Dest: dest},
				{ID: 2, Start: day(4), End: day(11), Dest: nearDest},
			},
			decs: []Anchor{{ID: 10, Start: day(1), End: day(11), Dest: dest, CreatedAt: day(12)}},
			want: map[int64]int64{2: 10},
		},
		{
			name:  "merge: one decision attaches, the other becomes an orphan",
			cands: []SpanRef{{ID: 1, Start: day(1), End: day(11), Dest: dest}},
			decs: []Anchor{
				{ID: 10, Start: day(1), End: day(4), Dest: dest, CreatedAt: day(5)},
				{ID: 11, Start: day(5), End: day(11), Dest: nearDest, CreatedAt: day(12)},
			},
			want: map[int64]int64{1: 11}, // longer overlap wins; 10 is orphaned, not lost
		},
		{
			name: "repeat trips to the same destination pair up by time, not place",
			cands: []SpanRef{
				{ID: 1, Start: day(1), End: day(5), Dest: dest},
				{ID: 2, Start: day(20), End: day(25), Dest: nearDest},
			},
			decs: []Anchor{
				{ID: 11, Start: day(20), End: day(25), Dest: nearDest, CreatedAt: day(26)},
				{ID: 10, Start: day(1), End: day(5), Dest: dest, CreatedAt: day(6)},
			},
			want: map[int64]int64{1: 10, 2: 11},
		},
		{
			name: "equal overlap ties break deterministically by earlier candidate",
			cands: []SpanRef{
				{ID: 1, Start: day(1), End: day(5), Dest: dest},
				{ID: 2, Start: day(1), End: day(5), Dest: nearDest},
			},
			decs: []Anchor{{ID: 10, Start: day(1), End: day(5), Dest: dest, CreatedAt: day(6)}},
			want: map[int64]int64{1: 10},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Match(tc.cands, tc.decs, DefaultMatchParams())
			if len(got) != len(tc.want) {
				t.Fatalf("matched %d pairs (%v), want %d (%v)", len(got), got, len(tc.want), tc.want)
			}
			for cid, did := range tc.want {
				if got[cid] != did {
					t.Errorf("candidate %d matched decision %d, want %d", cid, got[cid], did)
				}
			}
		})
	}
}
