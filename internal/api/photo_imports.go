package api

import (
	"context"
	"fmt"
	"io"
	"log"

	"roadbook/internal/photo"
	"roadbook/internal/photosource"
	"roadbook/internal/store"
)

// MaxPhotoImportFiles caps one photo-batch request. A named parameter, not a
// judgment on camera rolls: bigger rolls upload in several batches, and the
// import path is idempotent so overlap between batches costs nothing.
const MaxPhotoImportFiles = 2000

// UploadPhotoImport is the photo-ingestion path (phase 11 BRIEF §4C/§4D):
// parts stream through metadata extraction one file at a time — scan, cut a
// thumbnail while the bytes are still in hand (JPEG only; HEIC pixels are
// not decodable here), then let the bytes go. Originals never touch disk.
// After the stream, sidecar pairing and fix resolution run over the scanned
// batch, thumbnails for the usable photos are written before their rows
// (file-before-row, the phase-4 rule), and the whole batch lands in one
// transaction. The import row is completed before the response; detection
// follows in the background exactly like the Timeline path.
func (s *Server) UploadPhotoImport(ctx context.Context, req UploadPhotoImportRequestObject) (UploadPhotoImportResponseObject, error) {
	if !s.importMu.TryLock() {
		return UploadPhotoImport409JSONResponse{Error: "an import is already running on this instance — wait for it to finish and try again"}, nil
	}
	handed := false
	defer func() {
		if !handed {
			s.importMu.Unlock()
		}
	}()

	label := "photo upload"
	var scans []photosource.Scanned
	type thumbData struct {
		bytes []byte
		w, h  int
	}
	thumbs := map[string]thumbData{} // content hash → thumbnail, fix files decided later

	for {
		part, err := req.Body.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			// A transport failure mid-stream: no row exists yet, nothing is
			// recorded — the caller retries the whole batch.
			return nil, fmt.Errorf("reading multipart stream: %w", err)
		}
		switch part.FormName() {
		case "label":
			b, err := io.ReadAll(io.LimitReader(part, 1024))
			if err == nil && len(b) > 0 {
				label = string(b)
			}
		case "file":
			name := part.FileName()
			if len(scans) >= MaxPhotoImportFiles {
				scans = append(scans, photosource.Scanned{Name: name})
				_, _ = io.Copy(io.Discard, part)
				continue
			}
			data, err := io.ReadAll(io.LimitReader(part, MaxUploadBytes+1))
			if err != nil {
				return nil, fmt.Errorf("reading part %q: %w", name, err)
			}
			if len(data) > MaxUploadBytes {
				// Per-file verdict, not a request-level 413: one oversized
				// file must not sink 800 good ones.
				scans = append(scans, photosource.Scanned{Name: name})
				continue
			}
			sc := photosource.ScanFile(name, data)
			if sc.IsPhoto && sc.ThumbDecodable {
				if tb, w, h, err := photo.Thumbnail(data, sc.Orientation, photo.DefaultThumbMaxPx, photo.DefaultThumbQuality); err == nil {
					thumbs[sc.ContentHash] = thumbData{tb, w, h}
				}
			}
			scans = append(scans, sc)
		default:
			_, _ = io.Copy(io.Discard, part)
		}
	}
	if len(scans) == 0 {
		return UploadPhotoImport400JSONResponse{Error: "the request carried no files — send photos (and their sidecars) as repeated `file` parts"}, nil
	}

	_, results, _ := photosource.Resolve(scans) // fixes travel per-result into items below

	files := make([]PhotoImportFile, len(results))
	var items []store.PhotoIngest
	for i, r := range results {
		files[i] = PhotoImportFile{File: r.Name, Status: PhotoImportFileStatus(r.Verdict)}
		if r.Verdict == photosource.VerdictUnsupported {
			files[i].Message = strp(r.Message)
		}
		// The zero Scanned from the size/count caps resolves as unsupported
		// with no message — give those their real reason.
		if r.Verdict == photosource.VerdictUnsupported && r.Message == "unrecognised input" {
			files[i].Message = strp(fmt.Sprintf("not imported — over the %d MB per-file limit or the %d-file per-request limit; send it in a batch of its own", MaxUploadBytes>>20, MaxPhotoImportFiles))
		}
		if r.Verdict != photosource.VerdictFix {
			continue
		}
		rec := store.PhotoRecord{
			ContentHash:  r.ContentHash,
			OriginalName: r.Name,
			TimeSource:   r.TimeSource,
			PosSource:    r.PosSource,
		}
		if tb, ok := thumbs[r.ContentHash]; ok {
			// File before row (the phase-4 rule): a mid-failure leaves an
			// unreachable file, never a row pointing at nothing.
			if err := s.Photos.WriteThumb(r.ContentHash, tb.bytes); err != nil {
				return nil, err
			}
			rec.ThumbW, rec.ThumbH = tb.w, tb.h
		}
		items = append(items, store.PhotoIngest{Fix: *r.Fix, Record: rec})
	}

	importID, err := s.Store.BeginImport(ctx, label, nil, nil)
	if err != nil {
		return nil, err
	}
	if _, err := s.Store.ImportPhotos(ctx, importID, items); err != nil {
		if ferr := s.Store.FailImport(ctx, importID, "photos", "storing the photo batch failed: "+err.Error()); ferr != nil {
			log.Printf("photo import %d: recording failure: %v", importID, ferr)
		}
		return nil, err
	}

	handed = true
	go func() {
		defer s.importMu.Unlock()
		defer func() {
			if r := recover(); r != nil {
				log.Printf("photo import %d: detect panic: %v", importID, r)
			}
		}()
		s.runAutoDetect(context.Background(), importID)
	}()

	row, err := s.Store.GetImport(ctx, importID)
	if err != nil || row == nil {
		return nil, fmt.Errorf("reading back import %d: %w", importID, err)
	}
	return UploadPhotoImport202JSONResponse(PhotoImportResult{
		Import: toAPIImport(*row),
		Files:  files,
	}), nil
}
