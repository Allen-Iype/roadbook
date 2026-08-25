package store_test

import (
	"context"
	"testing"
	"time"

	"roadbook/internal/domain"
	"roadbook/internal/store"
	"roadbook/internal/store/storetest"
)

// TestImportPhotos covers the photo-ingestion store path (phase 11,
// migration 00010): fixes land in the observation stratum, records pair to
// them by fix hash, the imports row is finalised, and a duplicate batch is
// fully idempotent — 0 new fixes, 0 new records.
func TestImportPhotos(t *testing.T) {
	s := storetest.Open(t)
	ctx := context.Background()

	loc := func(lat, lon float64) *domain.LatLng { return &domain.LatLng{Lat: lat, Lon: lon} }
	zone := time.FixedZone("", 0)
	items := []store.PhotoIngest{
		{
			Fix: domain.RawPosition{Time: time.Date(2026, 6, 5, 19, 0, 0, 0, zone),
				Loc: loc(64.2539, -15.2082), Source: domain.SourcePhoto},
			Record: store.PhotoRecord{ContentHash: "aa01", OriginalName: "eve_a.jpg",
				TimeSource: "gps", PosSource: "exif", ThumbW: 512, ThumbH: 384},
		},
		{
			Fix: domain.RawPosition{Time: time.Date(2026, 6, 5, 19, 50, 0, 0, zone),
				Loc: loc(64.2541, -15.2081), Source: domain.SourcePhoto},
			Record: store.PhotoRecord{ContentHash: "aa02", OriginalName: "eve_d.heic",
				TimeSource: "gps", PosSource: "exif"}, // no thumbnail: HEIC
		},
	}

	importID, err := s.BeginImport(ctx, "photo batch", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err := s.ImportPhotos(ctx, importID, items)
	if err != nil {
		t.Fatal(err)
	}
	if res.Parsed != 2 || res.Inserted != 2 {
		t.Errorf("first import = %+v, want parsed 2 inserted 2", res)
	}

	row, err := s.GetImport(ctx, importID)
	if err != nil || row == nil {
		t.Fatalf("GetImport: %v, %v", row, err)
	}
	if row.Status != "completed" || row.RawPositions != 2 ||
		row.DetectedFormat == nil || *row.DetectedFormat != "photos" {
		t.Errorf("imports row = %+v — want completed, 2 raw positions, format photos", row)
	}

	recs, err := s.ListPhotoRecords(ctx, importID)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 2 {
		t.Fatalf("got %d records, want 2", len(recs))
	}
	if recs[0].Lat != 64.2539 || recs[0].TakenOffsetSec != 0 || recs[0].ThumbW != 512 {
		t.Errorf("record 0 = %+v", recs[0])
	}
	if recs[1].ThumbW != 0 || recs[1].ThumbH != 0 {
		t.Errorf("HEIC record has thumbnail dims: %+v", recs[1])
	}

	// The observation stratum received the fixes — and detection sees them
	// as photo-sourced.
	obs, err := s.LoadObservations(ctx)
	if err != nil {
		t.Fatal(err)
	}
	photoFixes := 0
	for _, rp := range obs.RawPositions {
		if rp.Source == domain.SourcePhoto {
			photoFixes++
		}
	}
	if photoFixes != 2 {
		t.Errorf("observation stratum holds %d photo fixes, want 2", photoFixes)
	}

	// Duplicate batch: full idempotency, records included.
	dupID, err := s.BeginImport(ctx, "photo batch again", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	dup, err := s.ImportPhotos(ctx, dupID, items)
	if err != nil {
		t.Fatal(err)
	}
	if dup.Inserted != 0 {
		t.Errorf("duplicate batch inserted %d fixes, want 0", dup.Inserted)
	}
	dupRecs, err := s.ListPhotoRecords(ctx, dupID)
	if err != nil {
		t.Fatal(err)
	}
	if len(dupRecs) != 0 {
		t.Errorf("duplicate batch created %d records under the new import, want 0 (hash-unique)", len(dupRecs))
	}
}
