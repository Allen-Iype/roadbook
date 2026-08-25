package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"roadbook/internal/detect"
	"roadbook/internal/store"
	"roadbook/internal/timeline"
)

// MaxImportUploadBytes caps one uploaded export. Named parameter (the
// photos MaxUploadBytes discipline), generous because an archive's size is
// not the user's fault (phase 7 BRIEF §3B).
const MaxImportUploadBytes = 2 << 30 // 2 GiB

// sniffHeadBytes matches the parser's own head window (Parse peeks 4096).
const sniffHeadBytes = 4096

// UploadImport is the browser import path (phase 7 BRIEF §§1.1–1.3): stream
// the file to disk hashing as it goes, sniff synchronously so a wrong file
// is answered while the user is looking, then import in the background with
// the imports row as the only status channel. No import row exists until
// the upload has fully arrived and passed the sniff — an interrupted upload
// leaves nothing behind.
func (s *Server) UploadImport(ctx context.Context, req UploadImportRequestObject) (UploadImportResponseObject, error) {
	// One import at a time (BRIEF §1.2): a semaphore, not a scheduler. The
	// lock is released by the background goroutine when the import
	// finalises, or below on every path that never starts one.
	if !s.importMu.TryLock() {
		return UploadImport409JSONResponse{Error: "an import is already running — watch it on the imports page and retry when it finishes"}, nil
	}
	handed := false // set true once the goroutine owns the unlock
	defer func() {
		if !handed {
			s.importMu.Unlock()
		}
	}()

	label := ""
	var tmpPath string
	var contentHash string
	uploaded := false
	// Any exit before promotion removes the temp file — absence is the goal.
	defer func() {
		if tmpPath != "" {
			if err := s.Uploads.Remove(tmpPath); err != nil {
				log.Printf("upload: removing temp file: %v", err)
			}
		}
	}()

	for {
		part, err := req.Body.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			// The stream broke mid-request (client abort, malformed body).
			// The deferred cleanup leaves nothing behind.
			return nil, fmt.Errorf("reading upload: %w", err)
		}
		switch part.FormName() {
		case "label":
			b, err := io.ReadAll(io.LimitReader(part, 1024))
			if err != nil {
				return nil, fmt.Errorf("reading label: %w", err)
			}
			label = strings.TrimSpace(string(b))
		case "file":
			if label == "" {
				label = part.FileName()
			}
			tmp, err := s.Uploads.CreateTemp()
			if err != nil {
				return nil, err
			}
			tmpPath = tmp.Name()
			h := sha256.New()
			n, err := io.Copy(io.MultiWriter(tmp, h), io.LimitReader(part, MaxImportUploadBytes+1))
			if cerr := tmp.Close(); err == nil {
				err = cerr
			}
			if err != nil {
				return nil, fmt.Errorf("storing upload: %w", err)
			}
			if n > MaxImportUploadBytes {
				return UploadImport413JSONResponse{Error: fmt.Sprintf(
					"the file exceeds the %d MB upload limit", MaxImportUploadBytes>>20)}, nil
			}
			contentHash = hex.EncodeToString(h.Sum(nil))
			uploaded = true
		}
	}
	if !uploaded {
		return UploadImport400JSONResponse{Error: "the request carried no file part — choose a Timeline export and try again"}, nil
	}

	// Synchronous sniff (BRIEF §1.3): the head of the fully-written file,
	// answered in this response so a wrong file redirects immediately. The
	// label is evidence for the front door's walkthrough mapping; rejected
	// bytes are not retained.
	head, err := readHead(tmpPath, sniffHeadBytes)
	if err != nil {
		return nil, err
	}
	if ue := timeline.Sniff(head); ue != nil {
		resp := UploadImport400JSONResponse{Error: ue.Message}
		if ue.Kind != "" {
			resp.DetectedFormat = &ue.Kind
		}
		return resp, nil
	}

	retained, err := s.Uploads.Promote(tmpPath, contentHash)
	if err != nil {
		return nil, err
	}
	tmpPath = "" // promoted: nothing to clean

	if label == "" {
		label = "browser upload"
	}
	importID, err := s.Store.BeginUploadImport(ctx, label, contentHash)
	if err != nil {
		return nil, err
	}

	handed = true
	go s.runUploadImport(importID, retained)

	row, err := s.Store.GetImport(ctx, importID)
	if err != nil || row == nil {
		// The row was just inserted; a read miss here is a real failure.
		return nil, fmt.Errorf("reading back import %d: %w", importID, err)
	}
	return UploadImport202JSONResponse(toAPIImport(*row)), nil
}

// runUploadImport is the background half (BRIEF §1.2): parse the retained
// file, insert, then auto-detect with default parameters (BRIEF §3D). It
// runs on a background context — the upload response has long returned —
// and owns the single-import lock. Failures land on the imports row, the
// only status channel; log lines carry messages, never content.
func (s *Server) runUploadImport(importID int64, path string) {
	ctx := context.Background()
	defer s.importMu.Unlock()
	defer func() {
		// The parser is defensive, but this goroutine eats untrusted input;
		// a panic must become a visible failed import, not a dead serve.
		if r := recover(); r != nil {
			log.Printf("import %d: panic: %v", importID, r)
			if err := s.Store.FailImport(ctx, importID, "", "internal error during import — the server log has details"); err != nil {
				log.Printf("import %d: recording panic failure: %v", importID, err)
			}
		}
	}()

	fail := func(format, msg string) {
		if err := s.Store.FailImport(ctx, importID, format, msg); err != nil {
			log.Printf("import %d: recording failure: %v", importID, err)
		}
	}

	f, err := os.Open(path)
	if err != nil {
		fail("", "the stored upload could not be reopened: "+err.Error())
		return
	}
	defer f.Close()

	obs, st, err := timeline.Parse(f)
	if err != nil {
		// Mid-parse rejections (truncated, json-unrecognised) land here —
		// the asynchronous half of the taxonomy (BRIEF §1.3).
		kind := ""
		var ue *timeline.UnsupportedInputError
		if errors.As(err, &ue) {
			kind = ue.Kind
		}
		fail(kind, err.Error())
		return
	}
	if _, err := s.Store.ImportObservations(ctx, importID, st.Format, obs, st.Skipped); err != nil {
		fail(st.Format, "storing observations failed: "+err.Error())
		return
	}

	s.runAutoDetect(ctx, importID)
}

// runAutoDetect runs detection with default parameters after a successful
// import, reporting through detect_status only — a detect failure never
// marks the import failed (BRIEF §3D). Shared by the Timeline and photo
// upload paths.
func (s *Server) runAutoDetect(ctx context.Context, importID int64) {
	setDetect := func(status string) {
		if err := s.Store.SetImportDetectStatus(ctx, importID, status); err != nil {
			log.Printf("import %d: recording detect status %q: %v", importID, status, err)
		}
	}
	setDetect("running")
	all, err := s.Store.LoadObservations(ctx)
	if err != nil {
		log.Printf("import %d: auto-detect load: %v", importID, err)
		setDetect("failed")
		return
	}
	p := detect.DefaultParams()
	res := detect.Run(all, p)
	if _, err := s.Store.SaveRun(ctx, p, res); err != nil {
		log.Printf("import %d: auto-detect save: %v", importID, err)
		setDetect("failed")
		return
	}
	setDetect("completed")
}

func (s *Server) GetImport(ctx context.Context, req GetImportRequestObject) (GetImportResponseObject, error) {
	row, err := s.Store.GetImport(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return GetImport404JSONResponse{Error: "no such import"}, nil
	}
	return GetImport200JSONResponse(toAPIImport(*row)), nil
}

func toAPIImport(r store.ImportRow) Import {
	out := Import{
		Id:             r.ID,
		SourceLabel:    r.SourceLabel,
		ImportedAt:     r.ImportedAt,
		WindowStart:    r.WindowStart,
		WindowEnd:      r.WindowEnd,
		Visits:         r.Visits,
		Activities:     r.Activities,
		Points:         r.Points,
		RawPositions:   r.RawPositions,
		Skipped:        r.Skipped,
		Status:         ImportStatus(r.Status),
		Error:          r.Error,
		DetectedFormat: r.DetectedFormat,
		ContentHash:    r.ContentHash,
		Inserted:       r.Inserted,
	}
	if r.DetectStatus != nil {
		ds := ImportDetectStatus(*r.DetectStatus)
		out.DetectStatus = &ds
	}
	return out
}

func readHead(path string, n int) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	buf := make([]byte, n)
	m, err := io.ReadFull(f, buf)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return nil, err
	}
	return buf[:m], nil
}
