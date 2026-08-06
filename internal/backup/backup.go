// Package backup writes and restores the product backup (phase 5 BRIEF §3D):
// one tar.gz holding exactly the irreplaceable set — decisions, photo rows,
// and thumbnail bytes (the original photo was discarded at upload, so the
// thumbnail is the copy) — plus a manifest. Everything else in the database
// regenerates from the user's own export files and is deliberately absent.
//
// Rows are archived by durable identity, never by row id: decisions by their
// anchor, photos by content hash, and a photo references its decision by the
// anchor tuple. Restore merges by those identities into any instance, empty
// or not; the overlap is skipped and reported, never reconciled. Restored
// decisions whose candidates do not exist yet are orphans — the state the
// API has reported since phase 1 — and the user's next import + detection
// re-attaches them.
package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"time"

	"roadbook/internal/domain"
	"roadbook/internal/store"
)

// FormatVersion is the archive format this binary writes and the highest it
// restores. It moves only when the layout or semantics change.
const FormatVersion = 1

type Manifest struct {
	FormatVersion int       `json:"format_version"`
	SchemaVersion int64     `json:"schema_version"`
	CreatedAt     time.Time `json:"created_at"`
	Decisions     int       `json:"decisions"`
	Photos        int       `json:"photos"`
	Thumbnails    int       `json:"thumbnails"`
}

// anchorRef is a decision's durable identity — the join key between
// decisions.json and photos.json.
type anchorRef struct {
	SpanStart time.Time `json:"anchor_span_start"`
	SpanEnd   time.Time `json:"anchor_span_end"`
	DestLat   float64   `json:"anchor_dest_lat"`
	DestLon   float64   `json:"anchor_dest_lon"`
}

func (a anchorRef) key() string {
	// RFC3339Nano of the UTC instant: the identity is the instant, not the
	// civil rendering it arrived in.
	return fmt.Sprintf("%s|%s|%v|%v",
		a.SpanStart.UTC().Format(time.RFC3339Nano), a.SpanEnd.UTC().Format(time.RFC3339Nano),
		a.DestLat, a.DestLon)
}

type archivedDecision struct {
	Action    string    `json:"action"`
	Name      *string   `json:"name,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	anchorRef
}

type archivedPhoto struct {
	ContentHash    string     `json:"content_hash"`
	OriginalName   string     `json:"original_name"`
	TakenAt        *time.Time `json:"taken_at,omitempty"`
	TakenOffsetSec *int       `json:"taken_offset_sec,omitempty"`
	TimeSource     string     `json:"time_source"`
	Lat            *float64   `json:"lat,omitempty"`
	Lon            *float64   `json:"lon,omitempty"`
	PosSource      string     `json:"pos_source"`
	ThumbW         int        `json:"thumb_w"`
	ThumbH         int        `json:"thumb_h"`
	UploadedAt     time.Time  `json:"uploaded_at"`
	Decision       anchorRef  `json:"decision"`
}

// Write produces the archive. now is passed in rather than read here so the
// caller owns the clock. A photo row whose thumbnail file is missing on disk
// is excluded from the archive with a warning: a row without its file is a
// permanently broken image (the phase 4 delete-order reasoning), and a backup
// must not preserve a state it would be a bug to create.
func Write(ctx context.Context, s *store.Store, files store.PhotoFiles, w io.Writer, now time.Time) (Manifest, []string, error) {
	var warnings []string

	decs, err := s.ListDecisions(ctx)
	if err != nil {
		return Manifest{}, nil, err
	}
	photos, err := s.ListAllPhotos(ctx)
	if err != nil {
		return Manifest{}, nil, err
	}
	schema, err := s.SchemaVersion(ctx)
	if err != nil {
		return Manifest{}, nil, err
	}

	anchorByID := make(map[int64]anchorRef, len(decs))
	outDecs := make([]archivedDecision, 0, len(decs))
	for _, d := range decs {
		ref := anchorRef{SpanStart: d.AnchorStart, SpanEnd: d.AnchorEnd, DestLat: d.AnchorDest.Lat, DestLon: d.AnchorDest.Lon}
		anchorByID[d.ID] = ref
		outDecs = append(outDecs, archivedDecision{
			Action: d.Action, Name: d.Name, CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt, anchorRef: ref,
		})
	}

	outPhotos := make([]archivedPhoto, 0, len(photos))
	thumbs := map[string][]byte{}
	for _, p := range photos {
		ref, ok := anchorByID[p.DecisionID]
		if !ok {
			// FK makes this impossible; check anyway rather than archive a
			// dangling reference.
			return Manifest{}, nil, fmt.Errorf("photo %d references decision %d which was not listed", p.ID, p.DecisionID)
		}
		jpeg, err := files.ReadThumb(p.ContentHash)
		if err != nil {
			warnings = append(warnings,
				fmt.Sprintf("photo %q (%s): thumbnail file missing — excluded from the archive", p.OriginalName, p.ContentHash))
			continue
		}
		thumbs[p.ContentHash] = jpeg
		outPhotos = append(outPhotos, archivedPhoto{
			ContentHash: p.ContentHash, OriginalName: p.OriginalName,
			TakenAt: p.TakenAt, TakenOffsetSec: p.TakenOffsetSec, TimeSource: p.TimeSource,
			Lat: p.Lat, Lon: p.Lon, PosSource: p.PosSource,
			ThumbW: p.ThumbW, ThumbH: p.ThumbH, UploadedAt: p.UploadedAt,
			Decision: ref,
		})
	}

	man := Manifest{
		FormatVersion: FormatVersion,
		SchemaVersion: schema,
		CreatedAt:     now,
		Decisions:     len(outDecs),
		Photos:        len(outPhotos),
		Thumbnails:    len(thumbs),
	}

	gz := gzip.NewWriter(w)
	tw := tar.NewWriter(gz)
	writeEntry := func(name string, data []byte) error {
		if err := tw.WriteHeader(&tar.Header{
			Name: name, Mode: 0o644, Size: int64(len(data)), ModTime: now.UTC(),
		}); err != nil {
			return err
		}
		_, err := tw.Write(data)
		return err
	}
	writeJSON := func(name string, v any) error {
		data, err := json.MarshalIndent(v, "", " ")
		if err != nil {
			return err
		}
		return writeEntry(name, append(data, '\n'))
	}

	if err := writeJSON("manifest.json", man); err != nil {
		return man, warnings, err
	}
	if err := writeJSON("decisions.json", outDecs); err != nil {
		return man, warnings, err
	}
	if err := writeJSON("photos.json", outPhotos); err != nil {
		return man, warnings, err
	}
	hashes := make([]string, 0, len(thumbs))
	for h := range thumbs {
		hashes = append(hashes, h)
	}
	sort.Strings(hashes) // deterministic archive order
	for _, h := range hashes {
		if err := writeEntry("thumbnails/"+h+".jpg", thumbs[h]); err != nil {
			return man, warnings, err
		}
	}
	if err := tw.Close(); err != nil {
		return man, warnings, err
	}
	return man, warnings, gz.Close()
}

// Report is what one restore did — every count printed, nothing silent.
type Report struct {
	Manifest          Manifest
	DecisionsRestored int
	DecisionsSkipped  int // identity already present — left untouched
	PhotosRestored    int
	PhotosSkipped     int
	ThumbsWritten     int
	ThumbsExisting    int
	// Photos excluded because no thumbnail was available in the archive or
	// on disk — restoring the row alone would create a permanently broken
	// image.
	MissingThumb []string
}

// Restore merges an archive into the connected instance. Decisions land
// first; each photo's thumbnail file is written before its row (the mirror
// of delete's row-first ordering: both fail toward a sweepable orphan file,
// never a row without its file).
func Restore(ctx context.Context, s *store.Store, files store.PhotoFiles, r io.Reader) (Report, error) {
	var rep Report

	gz, err := gzip.NewReader(r)
	if err != nil {
		return rep, fmt.Errorf("not a roadbook backup (gzip): %w", err)
	}
	tr := tar.NewReader(gz)

	var haveManifest bool
	var decs []archivedDecision
	var photos []archivedPhoto
	thumbInArchive := map[string]bool{}
	idByAnchor := map[string]int64{}

	restoreDecisions := func() error {
		for _, d := range decs {
			row := store.DecisionRow{
				Action: d.Action, Name: d.Name,
				AnchorStart: d.SpanStart, AnchorEnd: d.SpanEnd,
				AnchorDest: domain.LatLng{Lat: d.DestLat, Lon: d.DestLon},
				CreatedAt:  d.CreatedAt, UpdatedAt: d.UpdatedAt,
			}
			id, inserted, err := s.RestoreDecision(ctx, row)
			if err != nil {
				return err
			}
			idByAnchor[d.key()] = id
			if inserted {
				rep.DecisionsRestored++
			} else {
				rep.DecisionsSkipped++
			}
		}
		return nil
	}

	restorePhoto := func(p archivedPhoto) error {
		id, ok := idByAnchor[p.Decision.key()]
		if !ok {
			return fmt.Errorf("photo %q references a decision absent from the archive", p.OriginalName)
		}
		inserted, err := s.RestorePhoto(ctx, store.PhotoRow{
			DecisionID: id, ContentHash: p.ContentHash, OriginalName: p.OriginalName,
			TakenAt: p.TakenAt, TakenOffsetSec: p.TakenOffsetSec, TimeSource: p.TimeSource,
			Lat: p.Lat, Lon: p.Lon, PosSource: p.PosSource,
			ThumbW: p.ThumbW, ThumbH: p.ThumbH, UploadedAt: p.UploadedAt,
		})
		if err != nil {
			return err
		}
		if inserted {
			rep.PhotosRestored++
		} else {
			rep.PhotosSkipped++
		}
		return nil
	}

	photoByHash := map[string]archivedPhoto{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return rep, fmt.Errorf("reading archive: %w", err)
		}
		switch {
		case hdr.Name == "manifest.json":
			if err := json.NewDecoder(tr).Decode(&rep.Manifest); err != nil {
				return rep, fmt.Errorf("manifest: %w", err)
			}
			if rep.Manifest.FormatVersion > FormatVersion {
				return rep, fmt.Errorf("archive format v%d is newer than this binary understands (v%d) — upgrade roadbook first",
					rep.Manifest.FormatVersion, FormatVersion)
			}
			haveManifest = true
		case hdr.Name == "decisions.json":
			if !haveManifest {
				return rep, fmt.Errorf("malformed archive: decisions.json before manifest.json")
			}
			if err := json.NewDecoder(tr).Decode(&decs); err != nil {
				return rep, fmt.Errorf("decisions.json: %w", err)
			}
			if err := restoreDecisions(); err != nil {
				return rep, err
			}
		case hdr.Name == "photos.json":
			if err := json.NewDecoder(tr).Decode(&photos); err != nil {
				return rep, fmt.Errorf("photos.json: %w", err)
			}
			for _, p := range photos {
				photoByHash[p.ContentHash] = p
			}
		case len(hdr.Name) > len("thumbnails/") && hdr.Name[:len("thumbnails/")] == "thumbnails/":
			hash := hdr.Name[len("thumbnails/") : len(hdr.Name)-len(".jpg")]
			p, ok := photoByHash[hash]
			if !ok {
				// A thumbnail with no row is inert; note nothing.
				continue
			}
			thumbInArchive[hash] = true
			if _, err := files.ReadThumb(hash); err == nil {
				rep.ThumbsExisting++
			} else {
				jpeg, err := io.ReadAll(tr)
				if err != nil {
					return rep, fmt.Errorf("thumbnail %s: %w", hash, err)
				}
				if err := files.WriteThumb(hash, jpeg); err != nil {
					return rep, err
				}
				rep.ThumbsWritten++
			}
			if err := restorePhoto(p); err != nil {
				return rep, err
			}
		}
	}
	if !haveManifest {
		return rep, fmt.Errorf("not a roadbook backup: no manifest.json")
	}

	// Rows whose thumbnails the archive did not carry: restore only if the
	// file already exists here; otherwise the row would be a broken image.
	for _, p := range photos {
		if thumbInArchive[p.ContentHash] {
			continue
		}
		if _, err := files.ReadThumb(p.ContentHash); err == nil {
			rep.ThumbsExisting++
			if err := restorePhoto(p); err != nil {
				return rep, err
			}
		} else {
			rep.MissingThumb = append(rep.MissingThumb, p.OriginalName)
		}
	}
	return rep, nil
}
