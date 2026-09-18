package api

import (
	"context"
	"log"
	"os"
)

// Self-serve deletion (phase 13 CP5). Rows first, in one store transaction,
// then files — the order that fails toward an orphaned file on disk, never
// toward a row whose file is gone (the phase-4 photos rule, applied to the
// whole account). The import lock is taken for the duration so no import
// goroutine can write rows behind the deletion.
func (s *Server) DeleteMyData(ctx context.Context, _ DeleteMyDataRequestObject) (DeleteMyDataResponseObject, error) {
	if !s.importMu.TryLock() {
		return DeleteMyData409JSONResponse{Error: "an import is in progress — wait for it to finish, then delete"}, nil
	}
	defer s.importMu.Unlock()

	// Mode off keeps the 'self' row (every table's DEFAULT names it); mode
	// oidc forgets the account, so the next sign-in creates a fresh user.
	rep, err := s.Store.DeleteUserData(ctx, s.currentUser(ctx), s.Auth != nil)
	if err != nil {
		return nil, err
	}
	out := DeletionReport{Rows: map[string]int64{}}
	for table, n := range rep.Rows {
		out.Rows[table] = n
	}
	for _, h := range rep.OrphanThumbs {
		if err := s.Photos.DeleteThumb(h); err != nil {
			// The rows are already gone; a leftover file is sweepable
			// garbage, not a leak of anything reachable. Log and count
			// what did go.
			log.Printf("deleting thumbnail %s: %v", h, err)
			continue
		}
		out.ThumbnailsRemoved++
	}
	for _, h := range rep.OrphanUploads {
		if err := os.Remove(s.Uploads.Path(h)); err != nil && !os.IsNotExist(err) {
			log.Printf("deleting upload %s: %v", h, err)
			continue
		}
		out.UploadsRemoved++
	}
	resp := DeleteMyData200JSONResponse{Body: out}
	if s.Auth != nil {
		sc := s.Auth.ClearSessionCookie().String()
		resp.Headers.SetCookie = &sc
	}
	return resp, nil
}
