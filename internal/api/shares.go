package api

import (
	"bytes"
	"context"

	"roadbook/internal/auth"
	"roadbook/internal/journey"
	"roadbook/internal/store"
)

// Share links (phase 13 CP4). Three owner-side operations behind the
// session (create, list, revoke) and three token-side reads that are
// exempt from it: the token is the credential. Everything the shared view
// returns is read AS THE LINK'S OWNER — the requester has no identity here
// and needs none — and scoped to exactly the one adventure the link names.

// noindex is the robots directive every shared response carries: a share
// link is for the people it was sent to, never for a crawler that found it.
const noindex = "noindex, nofollow"

func (s *Server) CreateShareLink(ctx context.Context, req CreateShareLinkRequestObject) (CreateShareLinkResponseObject, error) {
	cand, err := s.Store.LatestCandidate(ctx, s.currentUser(ctx), req.Id)
	if err != nil {
		return nil, err
	}
	if cand == nil {
		return CreateShareLink404JSONResponse{Error: "no such candidate in the latest run — re-detection may have replaced it; reload the list"}, nil
	}
	dec, err := s.confirmedDecisionFor(ctx, cand.ID)
	if err != nil {
		return nil, err
	}
	if dec == nil {
		return CreateShareLink409JSONResponse{Error: "only confirmed adventures can be shared — confirm this candidate first"}, nil
	}
	// The same 128-bit generator and the same hash-only storage as sessions:
	// one token discipline for both credentials the server mints.
	raw, hash, err := auth.NewSessionToken()
	if err != nil {
		return nil, err
	}
	row, err := s.Store.CreateShareLink(ctx, s.currentUser(ctx), dec.ID, hash)
	if err != nil {
		return nil, err
	}
	return CreateShareLink201JSONResponse{Id: row.ID, CreatedAt: row.CreatedAt, Token: raw}, nil
}

func (s *Server) ListShareLinks(ctx context.Context, req ListShareLinksRequestObject) (ListShareLinksResponseObject, error) {
	cand, err := s.Store.LatestCandidate(ctx, s.currentUser(ctx), req.Id)
	if err != nil {
		return nil, err
	}
	if cand == nil {
		return ListShareLinks404JSONResponse{Error: "no such candidate in the latest run — re-detection may have replaced it; reload the list"}, nil
	}
	dec, err := s.confirmedDecisionFor(ctx, cand.ID)
	if err != nil {
		return nil, err
	}
	if dec == nil {
		return ListShareLinks409JSONResponse{Error: "only confirmed adventures can be shared — confirm this candidate first"}, nil
	}
	rows, err := s.Store.ListShareLinks(ctx, s.currentUser(ctx), dec.ID)
	if err != nil {
		return nil, err
	}
	out := ShareLinkList{Shares: make([]ShareLink, 0, len(rows))}
	for _, r := range rows {
		out.Shares = append(out.Shares, ShareLink{Id: r.ID, CreatedAt: r.CreatedAt})
	}
	return ListShareLinks200JSONResponse(out), nil
}

func (s *Server) RevokeShareLink(ctx context.Context, req RevokeShareLinkRequestObject) (RevokeShareLinkResponseObject, error) {
	ok, err := s.Store.DeleteShareLink(ctx, s.currentUser(ctx), req.Id)
	if err != nil {
		return nil, err
	}
	if !ok {
		return RevokeShareLink404JSONResponse{Error: "no such share link"}, nil
	}
	return RevokeShareLink204Response{}, nil
}

// sharedTarget resolves a raw token to the adventure it opens: the link
// row, the owner's currently-matched candidate, and the confirmed decision.
// Every way a link can fail to open — unknown, revoked, decision dismissed
// since, decision orphaned by re-detection, owner has no run — returns
// (nil, nil, nil, nil): the caller answers one 404 and the outside learns
// nothing about which it was.
func (s *Server) sharedTarget(ctx context.Context, token string) (*store.ShareLinkRow, *store.CandidateRow, *store.DecisionRow, error) {
	link, err := s.Store.ResolveShareLink(ctx, auth.HashToken(token))
	if err != nil || link == nil {
		return nil, nil, nil, err
	}
	_, cands, decs, matched, err := s.matchedStateFor(ctx, link.UserID)
	if err != nil {
		return nil, nil, nil, err
	}
	var dec *store.DecisionRow
	for i := range decs {
		if decs[i].ID == link.DecisionID && decs[i].Action == "confirmed" {
			dec = &decs[i]
		}
	}
	if dec == nil {
		return nil, nil, nil, nil
	}
	for i := range cands {
		if did, ok := matched[cands[i].ID]; ok && did == dec.ID {
			return link, &cands[i], dec, nil
		}
	}
	return nil, nil, nil, nil
}

func (s *Server) GetSharedAdventure(ctx context.Context, req GetSharedAdventureRequestObject) (GetSharedAdventureResponseObject, error) {
	link, cand, dec, err := s.sharedTarget(ctx, req.Token)
	if err != nil {
		return nil, err
	}
	if link == nil {
		return GetSharedAdventure404JSONResponse{Error: "this link does not open anything — it may have been revoked"}, nil
	}
	j, _, err := s.journeyFor(ctx, link.UserID, cand)
	if err != nil {
		return nil, err
	}
	rows, err := s.Store.ListPhotos(ctx, link.UserID, dec.ID)
	if err != nil {
		return nil, err
	}
	photos, err := s.placedPhotos(ctx, link.UserID, cand, rows)
	if err != nil {
		return nil, err
	}
	imported, err := s.placedImportPhotos(ctx, link.UserID, cand)
	if err != nil {
		return nil, err
	}
	name := ""
	if dec.Name != nil {
		name = *dec.Name
	}
	robots := noindex
	return GetSharedAdventure200JSONResponse{
		Headers: GetSharedAdventure200ResponseHeaders{XRobotsTag: &robots},
		Body: SharedAdventure{
			Name:           name,
			SpanStart:      cand.SpanStart,
			SpanEnd:        cand.SpanEnd,
			StartTruncated: cand.StartTruncated,
			EndTruncated:   cand.EndTruncated,
			Journey:        j,
			Photos:         photos.Photos,
			ImportPhotos:   imported.Photos,
			Params:         map[string]any{"photo_far_warn_m": journey.DefaultPhotoFarWarnM},
		},
	}, nil
}

func (s *Server) GetSharedPhotoThumbnail(ctx context.Context, req GetSharedPhotoThumbnailRequestObject) (GetSharedPhotoThumbnailResponseObject, error) {
	link, _, dec, err := s.sharedTarget(ctx, req.Token)
	if err != nil {
		return nil, err
	}
	if link == nil {
		return GetSharedPhotoThumbnail404JSONResponse{Error: "this link does not open anything"}, nil
	}
	p, err := s.Store.GetPhoto(ctx, link.UserID, req.Id)
	if err != nil {
		return nil, err
	}
	// The token grants one adventure: a photo of the owner's OTHER
	// adventures is as absent as anyone else's.
	if p == nil || p.DecisionID != dec.ID {
		return GetSharedPhotoThumbnail404JSONResponse{Error: "no such photo on this adventure"}, nil
	}
	data, err := s.Photos.ReadThumb(p.ContentHash)
	if err != nil {
		return GetSharedPhotoThumbnail404JSONResponse{Error: "thumbnail file missing"}, nil
	}
	return GetSharedPhotoThumbnail200ImagejpegResponse{Body: bytes.NewReader(data), ContentLength: int64(len(data))}, nil
}

func (s *Server) GetSharedImportPhotoThumbnail(ctx context.Context, req GetSharedImportPhotoThumbnailRequestObject) (GetSharedImportPhotoThumbnailResponseObject, error) {
	link, cand, _, err := s.sharedTarget(ctx, req.Token)
	if err != nil {
		return nil, err
	}
	if link == nil {
		return GetSharedImportPhotoThumbnail404JSONResponse{Error: "this link does not open anything"}, nil
	}
	r, err := s.Store.GetPhotoRecord(ctx, link.UserID, req.Id)
	if err != nil {
		return nil, err
	}
	// Same span join the shared view lists by: a record outside the
	// adventure's window is not on this adventure.
	if r == nil || r.ThumbW == 0 || r.TakenAt.Before(cand.SpanStart) || r.TakenAt.After(cand.SpanEnd) {
		return GetSharedImportPhotoThumbnail404JSONResponse{Error: "no such photo on this adventure"}, nil
	}
	data, err := s.Photos.ReadThumb(r.ContentHash)
	if err != nil {
		return GetSharedImportPhotoThumbnail404JSONResponse{Error: "thumbnail file missing"}, nil
	}
	return GetSharedImportPhotoThumbnail200ImagejpegResponse{Body: bytes.NewReader(data), ContentLength: int64(len(data))}, nil
}
