package api

import (
	"bytes"
	"context"

	"roadbook/internal/domain"
	"roadbook/internal/journey"
	"roadbook/internal/store"
)

// The record→adventure relation is a read-time span join (DECISIONS
// 2026-08-26): the records returned are exactly those whose capture instant
// falls inside the candidate's span, computed per request and never stored.
// Records ride imports, so re-detection has nothing to orphan; the same
// placement that positions attached photos positions these, against the same
// assembled, route-applied journey the page draws.
func (s *Server) ListCandidateImportPhotos(ctx context.Context, req ListCandidateImportPhotosRequestObject) (ListCandidateImportPhotosResponseObject, error) {
	cand, err := s.Store.LatestCandidate(ctx, s.currentUser(ctx), req.Id)
	if err != nil {
		return nil, err
	}
	if cand == nil {
		return ListCandidateImportPhotos404JSONResponse{Error: "no such candidate in the latest run — re-detection may have replaced it; reload the list"}, nil
	}
	out, err := s.placedImportPhotos(ctx, s.currentUser(ctx), cand)
	if err != nil {
		return nil, err
	}
	return ListCandidateImportPhotos200JSONResponse(out), nil
}

// placedImportPhotos span-joins the owner's records to the candidate and
// places them against the assembled journey. Shared with the shared view
// (CP4).
func (s *Server) placedImportPhotos(ctx context.Context, userID string, cand *store.CandidateRow) (ImportPhotoList, error) {
	recs, err := s.Store.ListPhotoRecordsInSpan(ctx, userID, cand.SpanStart, cand.SpanEnd)
	if err != nil {
		return ImportPhotoList{}, err
	}

	out := ImportPhotoList{
		Photos: make([]ImportPhoto, len(recs)),
		Params: map[string]any{"photo_far_warn_m": journey.DefaultPhotoFarWarnM},
	}
	var j journey.Journey
	var haveJourney bool
	if len(recs) > 0 {
		j, _, err = s.assembledJourney(ctx, userID, cand)
		if err != nil {
			return ImportPhotoList{}, err
		}
		haveJourney = true
	}
	for i, r := range recs {
		ap := ImportPhoto{
			Id:             r.ID,
			ImportId:       r.ImportID,
			OriginalName:   r.OriginalName,
			TakenAt:        r.TakenAt,
			TakenOffsetSec: r.TakenOffsetSec,
			TimeSource:     ImportPhotoTimeSource(r.TimeSource),
			Pos:            LatLng{Lat: r.Lat, Lon: r.Lon},
			PosSource:      ImportPhotoPosSource(r.PosSource),
			ThumbW:         r.ThumbW,
			ThumbH:         r.ThumbH,
			UploadedAt:     r.UploadedAt,
		}
		if haveJourney {
			pos := domain.LatLng{Lat: r.Lat, Lon: r.Lon}
			if p, ok := journey.PlacePhoto(j, r.TakenAt, pos, journey.DefaultPhotoFarWarnM); ok {
				pk := ImportPhotoPlaceKind(p.Kind)
				ap.PlaceKind = &pk
				if p.Kind == journey.PlaceStop {
					ap.StopIndex = &p.StopIndex
				} else {
					ap.LegIndex = &p.LegIndex
				}
				if p.HasDistance {
					d := p.DistanceM
					ap.DistanceFromRouteM = &d
					flagged := p.Flagged
					ap.FarFlagged = &flagged
				}
			}
		}
		out.Photos[i] = ap
	}
	return out, nil
}

func (s *Server) GetImportPhotoThumbnail(ctx context.Context, req GetImportPhotoThumbnailRequestObject) (GetImportPhotoThumbnailResponseObject, error) {
	r, err := s.Store.GetPhotoRecord(ctx, s.currentUser(ctx), req.Id)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return GetImportPhotoThumbnail404JSONResponse{Error: "no such photo record"}, nil
	}
	if r.ThumbW == 0 {
		// HEIC at ingest: metadata readable, pixels not decodable — the
		// record exists with no thumbnail, and that is its honest state.
		return GetImportPhotoThumbnail404JSONResponse{Error: "this record has no thumbnail — its photo format's pixels are not decodable here"}, nil
	}
	data, err := s.Photos.ReadThumb(r.ContentHash)
	if err != nil {
		return GetImportPhotoThumbnail404JSONResponse{Error: "thumbnail file missing from the photos directory — re-import the photo to restore it"}, nil
	}
	return GetImportPhotoThumbnail200ImagejpegResponse{
		Body:          bytes.NewReader(data),
		ContentLength: int64(len(data)),
	}, nil
}
