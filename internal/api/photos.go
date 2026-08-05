package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"time"

	"roadbook/internal/photo"
	"roadbook/internal/store"
)

// Upload limits — named parameters (invariant 3's discipline), enforced here
// in Go; the Next-side body limit merely accommodates them (BRIEF §1.3).
const (
	MaxUploadBytes = 25 << 20 // per file
	MaxUploadFiles = 50       // per request
)

// uploadEntry is one multipart part, buffered. Originals live only in this
// buffer for the life of the request — they are never written to disk
// (BRIEF §3B: the thumbnail is the only artifact). A non-empty reject means
// the reader already refused it (size or count limit) and data is nil.
type uploadEntry struct {
	name   string
	data   []byte
	reject string
}

func (s *Server) UploadCandidatePhotos(ctx context.Context, req UploadCandidatePhotosRequestObject) (UploadCandidatePhotosResponseObject, error) {
	cand, err := s.Store.LatestCandidate(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if cand == nil {
		return UploadCandidatePhotos404JSONResponse{Error: "no such candidate in the latest run — re-detection may have replaced it; reload the list"}, nil
	}
	dec, err := s.confirmedDecisionFor(ctx, cand.ID)
	if err != nil {
		return nil, err
	}
	if dec == nil {
		return UploadCandidatePhotos409JSONResponse{Error: "photos attach to confirmed adventures — confirm this candidate first"}, nil
	}

	entries, err := readParts(req.Body)
	if err != nil {
		return nil, err
	}

	// Pass 1 — classify every part and pair sidecars to images by filename,
	// falling back to the sidecar's own title field (BRIEF §3D).
	type classified struct {
		entry   uploadEntry
		kind    photo.Kind
		rej     *photo.UnsupportedError
		sidecar *photo.Sidecar
	}
	parts := make([]classified, len(entries))
	imageIndex := map[string]int{}
	for i, e := range entries {
		c := classified{entry: e}
		if e.reject == "" {
			c.kind, c.rej = photo.Sniff(e.data)
			if c.kind == photo.KindJPEG {
				imageIndex[e.name] = i
			}
		}
		parts[i] = c
	}
	sidecarFor := map[int]photo.Sidecar{} // image part index → its sidecar
	pairedTo := map[int]string{}          // sidecar part index → image name
	for i, c := range parts {
		if c.kind != photo.KindSidecar {
			continue
		}
		sc, err := photo.ParseSidecar(c.entry.data)
		if err != nil {
			parts[i].rej = &photo.UnsupportedError{Kind: "sidecar", Message: err.Error()}
			continue
		}
		parts[i].sidecar = &sc
		target := photo.SidecarPairName(c.entry.name)
		if _, ok := imageIndex[target]; !ok && sc.Title != "" {
			target = sc.Title
		}
		if idx, ok := imageIndex[target]; ok {
			sidecarFor[idx] = sc
			pairedTo[i] = target
		}
	}

	// Pass 2 — process in upload order, one result per file.
	results := make([]PhotoUploadResult, 0, len(parts))
	for i, c := range parts {
		res := PhotoUploadResult{File: c.entry.name}
		switch {
		case c.entry.reject != "":
			res.Status = Rejected
			res.Reason = strp(c.entry.reject)
		case c.rej != nil:
			res.Status = Rejected
			res.Reason = strp(c.rej.Message)
		case c.kind == photo.KindSidecar:
			if target, ok := pairedTo[i]; ok {
				res.Status = SidecarPaired
				res.PairedWith = strp(target)
			} else {
				res.Status = SidecarUnpaired
				res.Reason = strp("names no image in this upload — include the photo it describes in the same request")
			}
		default: // a JPEG
			var sc *photo.Sidecar
			if v, ok := sidecarFor[i]; ok {
				sc = &v
			}
			row, inserted, err := s.storePhoto(ctx, dec.ID, offsetOf(cand.SpanStart), c.entry, sc)
			if err != nil {
				if ue, ok := err.(*photo.UnsupportedError); ok {
					res.Status = Rejected
					res.Reason = strp(ue.Message)
					break
				}
				return nil, err
			}
			ap := toAPIPhoto(row)
			res.Photo = &ap
			if inserted {
				res.Status = Accepted
			} else {
				res.Status = Duplicate
			}
		}
		results = append(results, res)
	}
	return UploadCandidatePhotos200JSONResponse(PhotoUploadResults{Results: results}), nil
}

// storePhoto runs one image through extraction and persists it: thumbnail
// file first, then the row — a mid-failure leaves unreachable garbage,
// never a row pointing at a missing file (BRIEF §3B).
func (s *Server) storePhoto(ctx context.Context, decisionID int64, spanOffsetSec int, e uploadEntry, sc *photo.Sidecar) (store.PhotoRow, bool, error) {
	m := photo.ExtractEXIF(e.data)
	if sc != nil {
		m = photo.MergeSidecar(m, *sc)
	}

	thumb, w, h, err := photo.Thumbnail(e.data, m.Orientation, photo.DefaultThumbMaxPx, photo.DefaultThumbQuality)
	if err != nil {
		return store.PhotoRow{}, false, &photo.UnsupportedError{Kind: "undecodable",
			Message: "the image could not be decoded — it may be corrupted or a format masquerading as JPEG"}
	}

	sum := sha256.Sum256(e.data)
	hash := hex.EncodeToString(sum[:])

	row := store.PhotoRow{
		DecisionID:   decisionID,
		ContentHash:  hash,
		OriginalName: e.name,
		TimeSource:   string(photo.TimeNone),
		PosSource:    string(photo.PosNone),
		ThumbW:       w,
		ThumbH:       h,
	}
	// Rung 4 of the time ladder resolves against the adventure's own civil
	// offset (BRIEF §3E) — available here because the upload names the
	// candidate; recorded as exif_local, the stated weakest source.
	if rt, ok := photo.ResolveTime(m, spanOffsetSec, true); ok {
		t, off := rt.Time, rt.OffsetSec
		row.TakenAt = &t
		row.TakenOffsetSec = &off
		row.TimeSource = string(rt.Source)
	}
	if m.Pos != nil {
		lat, lon := m.Pos.Lat, m.Pos.Lon
		row.Lat, row.Lon = &lat, &lon
		row.PosSource = string(m.PosSource)
	}

	if err := s.Photos.WriteThumb(hash, thumb); err != nil {
		return store.PhotoRow{}, false, err
	}
	return s.Store.InsertPhoto(ctx, row)
}

func (s *Server) ListCandidatePhotos(ctx context.Context, req ListCandidatePhotosRequestObject) (ListCandidatePhotosResponseObject, error) {
	cand, err := s.Store.LatestCandidate(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if cand == nil {
		return ListCandidatePhotos404JSONResponse{Error: "no such candidate in the latest run — re-detection may have replaced it; reload the list"}, nil
	}
	dec, err := s.confirmedDecisionFor(ctx, cand.ID)
	if err != nil {
		return nil, err
	}
	if dec == nil {
		return ListCandidatePhotos409JSONResponse{Error: "photos attach to confirmed adventures — confirm this candidate first"}, nil
	}
	rows, err := s.Store.ListPhotos(ctx, dec.ID)
	if err != nil {
		return nil, err
	}
	out := PhotoList{Photos: make([]Photo, len(rows))}
	for i, r := range rows {
		out.Photos[i] = toAPIPhoto(r)
	}
	return ListCandidatePhotos200JSONResponse(out), nil
}

func (s *Server) GetPhotoThumbnail(ctx context.Context, req GetPhotoThumbnailRequestObject) (GetPhotoThumbnailResponseObject, error) {
	p, err := s.Store.GetPhoto(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return GetPhotoThumbnail404JSONResponse{Error: "no such photo"}, nil
	}
	data, err := s.Photos.ReadThumb(p.ContentHash)
	if err != nil {
		// The row exists but its file is gone — say so plainly rather than
		// pretending the photo does not exist.
		return GetPhotoThumbnail404JSONResponse{Error: "thumbnail file missing from the photos directory — the photo must be re-uploaded"}, nil
	}
	return GetPhotoThumbnail200ImagejpegResponse{
		Body:          bytes.NewReader(data),
		ContentLength: int64(len(data)),
	}, nil
}

func (s *Server) DeletePhoto(ctx context.Context, req DeletePhotoRequestObject) (DeletePhotoResponseObject, error) {
	p, err := s.Store.GetPhoto(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return DeletePhoto404JSONResponse{Error: "no such photo"}, nil
	}
	// Row first, then file (BRIEF §3B): fail toward unreachable garbage.
	if _, err := s.Store.DeletePhoto(ctx, p.ID); err != nil {
		return nil, err
	}
	if err := s.Photos.DeleteThumb(p.ContentHash); err != nil {
		return nil, err
	}
	return DeletePhoto204Response{}, nil
}

// confirmedDecisionFor resolves a candidate to its matched decision iff that
// decision is confirmed — the only rows photos may attach to (BRIEF §3A).
func (s *Server) confirmedDecisionFor(ctx context.Context, candID int64) (*store.DecisionRow, error) {
	_, _, decs, matched, err := s.matchedState(ctx)
	if err != nil {
		return nil, err
	}
	did, ok := matched[candID]
	if !ok {
		return nil, nil
	}
	for _, d := range decs {
		if d.ID == did && d.Action == "confirmed" {
			return &d, nil
		}
	}
	return nil, nil
}

// readParts buffers the multipart stream: filenames and bytes, with the
// per-file size cap applied while reading (an oversized part is drained and
// marked, never buffered past the limit) and the file-count cap on parts
// carrying filenames. Parts without a filename (plain form fields) are
// skipped. Originals exist only in these buffers, for the life of the
// request.
func readParts(r *multipart.Reader) ([]uploadEntry, error) {
	var entries []uploadEntry
	for {
		part, err := r.NextPart()
		if err == io.EOF {
			return entries, nil
		}
		if err != nil {
			return nil, err
		}
		name := part.FileName()
		if name == "" {
			part.Close()
			continue
		}
		e := uploadEntry{name: name}
		switch {
		case len(entries) >= MaxUploadFiles:
			io.Copy(io.Discard, part)
			e.reject = fmt.Sprintf("over the %d-files-per-request limit — upload the rest in another request", MaxUploadFiles)
		default:
			data, err := io.ReadAll(io.LimitReader(part, MaxUploadBytes+1))
			if err != nil {
				part.Close()
				return nil, err
			}
			if len(data) > MaxUploadBytes {
				io.Copy(io.Discard, part) // drain so the stream advances
				e.reject = fmt.Sprintf("larger than the %d MB per-file limit", MaxUploadBytes>>20)
			} else {
				e.data = data
			}
		}
		part.Close()
		entries = append(entries, e)
	}
}

func toAPIPhoto(r store.PhotoRow) Photo {
	p := Photo{
		Id:           r.ID,
		OriginalName: r.OriginalName,
		TimeSource:   PhotoTimeSource(r.TimeSource),
		PosSource:    PhotoPosSource(r.PosSource),
		ThumbW:       r.ThumbW,
		ThumbH:       r.ThumbH,
		UploadedAt:   r.UploadedAt,
	}
	if r.TakenAt != nil {
		p.TakenAt = r.TakenAt
		p.TakenOffsetSec = r.TakenOffsetSec
	}
	if r.Lat != nil && r.Lon != nil {
		p.Pos = &LatLng{Lat: *r.Lat, Lon: *r.Lon}
	}
	return p
}

func offsetOf(t time.Time) int {
	_, off := t.Zone()
	return off
}

func strp(s string) *string { return &s }
