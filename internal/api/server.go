// Package api implements the generated StrictServerInterface (gen.go, produced
// from api/openapi.yaml — never edited by hand). Handlers translate between
// store rows and contract types and run candidate↔decision matching; business
// rules live in internal/detect, persistence in internal/store.
package api

import (
	"context"
	"encoding/json"
	"strings"

	"roadbook/internal/detect"
	"roadbook/internal/domain"
	"roadbook/internal/journey"
	"roadbook/internal/store"
	"roadbook/internal/suggest"
)

type Server struct {
	Store       *store.Store
	MatchParams detect.MatchParams
	Suggester   suggest.Suggester
}

var _ StrictServerInterface = (*Server)(nil)

func (s *Server) GetHealth(ctx context.Context, _ GetHealthRequestObject) (GetHealthResponseObject, error) {
	return GetHealth200JSONResponse{Status: "ok"}, nil
}

func (s *Server) ListCandidates(ctx context.Context, _ ListCandidatesRequestObject) (ListCandidatesResponseObject, error) {
	run, cands, decs, matched, err := s.matchedState(ctx)
	if err != nil {
		return nil, err
	}

	resp := CandidateList{
		Candidates:        []Candidate{},
		OrphanedDecisions: []Decision{},
	}
	if run != nil {
		params := map[string]any{}
		if err := json.Unmarshal(run.Params, &params); err != nil {
			return nil, err
		}
		resp.Run = &Run{Id: run.ID, RanAt: run.RanAt, Params: params, OutliersDropped: run.OutliersDropped}
	}

	decByID := make(map[int64]store.DecisionRow, len(decs))
	for _, d := range decs {
		decByID[d.ID] = d
	}
	attachedDec := map[int64]bool{}
	for _, c := range cands {
		ac := toAPICandidate(c)
		if did, ok := matched[c.ID]; ok {
			d := toAPIDecision(decByID[did])
			ac.Decision = &d
			attachedDec[did] = true
		}
		resp.Candidates = append(resp.Candidates, ac)
	}
	for _, d := range decs {
		if !attachedDec[d.ID] {
			resp.OrphanedDecisions = append(resp.OrphanedDecisions, toAPIDecision(d))
		}
	}
	return ListCandidates200JSONResponse(resp), nil
}

// GetCandidateJourney assembles the candidate's span on demand — legs are
// derived data over immutable observations, never persisted (BRIEF §3B), so a
// parameter change is free and nothing can drift out of date.
func (s *Server) GetCandidateJourney(ctx context.Context, req GetCandidateJourneyRequestObject) (GetCandidateJourneyResponseObject, error) {
	cand, err := s.Store.LatestCandidate(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if cand == nil {
		return GetCandidateJourney404JSONResponse{Error: "no such candidate in the latest run — re-detection may have replaced it; reload the list"}, nil
	}
	obs, err := s.Store.LoadJourneyInputs(ctx, cand.SpanStart, cand.SpanEnd)
	if err != nil {
		return nil, err
	}
	j := journey.Assemble(obs, cand.SpanStart, cand.SpanEnd, journey.DefaultParams())
	out, err := toAPIJourney(j)
	if err != nil {
		return nil, err
	}

	// Countries crossed: every point the route draws, against the loaded
	// polygons, in one query (BRIEF §1.4). Gap-leg endpoints duplicate the
	// neighbouring observed points; the query's GROUP BY absorbs that.
	var pts []domain.LatLng
	for _, l := range j.Legs {
		for _, p := range l.Points {
			pts = append(pts, p.Loc)
		}
	}
	crossed, err := s.Store.CountriesForPoints(ctx, pts)
	if err != nil {
		return nil, err
	}
	out.Countries = make([]Country, 0, len(crossed))
	for _, c := range crossed {
		out.Countries = append(out.Countries, Country{IsoCode: c.ISOCode, Name: c.Name})
	}
	return GetCandidateJourney200JSONResponse(out), nil
}

func toAPIJourney(j journey.Journey) (Journey, error) {
	// Round-trip params through JSON so the response echoes exactly the named
	// parameters that produced the assembly (invariant 3).
	raw, err := json.Marshal(j.Params)
	if err != nil {
		return Journey{}, err
	}
	params := map[string]any{}
	if err := json.Unmarshal(raw, &params); err != nil {
		return Journey{}, err
	}

	out := Journey{
		WindowStart:     j.WindowStart,
		WindowEnd:       j.WindowEnd,
		Params:          params,
		Legs:            make([]Leg, 0, len(j.Legs)),
		Stops:           make([]Stop, 0, len(j.Stops)),
		TotalKm:         j.TotalKm,
		ObservedKm:      j.ObservedKm,
		InferredKm:      j.InferredKm,
		GoogleKm:        j.GoogleDistanceKm,
		MergedPoints:    j.MergedPoints(),
		TracePointsKept: j.TracePointsKept,
		RawPointsKept:   j.RawPointsKept,
	}
	for _, l := range j.Legs {
		al := Leg{
			Kind:       LegKind(l.Kind),
			Points:     make([]TimedPoint, len(l.Points)),
			DistanceKm: l.DistanceKm,
			Start:      l.Start(),
			End:        l.End(),
		}
		if l.Kind == journey.LegGap {
			gk := LegGapKind(l.GapKind)
			al.GapKind = &gk
		}
		for i, pt := range l.Points {
			al.Points[i] = TimedPoint{T: pt.Time, Lat: pt.Loc.Lat, Lon: pt.Loc.Lon}
		}
		out.Legs = append(out.Legs, al)
	}
	for _, st := range j.Stops {
		out.Stops = append(out.Stops, Stop{
			Start:          st.Start,
			End:            st.End,
			Loc:            LatLng{Lat: st.Loc.Lat, Lon: st.Loc.Lon},
			Points:         st.Points,
			DisplacementKm: st.DisplacementKm,
		})
	}
	return out, nil
}

// SuggestCandidateName is a one-shot lookup through the configured Suggester
// (BRIEF §1.7). The suggestion prefills the confirm step's name input and is
// never applied automatically; the null suggester's empty answer renders as
// exactly today's empty input.
func (s *Server) SuggestCandidateName(ctx context.Context, req SuggestCandidateNameRequestObject) (SuggestCandidateNameResponseObject, error) {
	cand, err := s.Store.LatestCandidate(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if cand == nil {
		return SuggestCandidateName404JSONResponse{Error: "no such candidate in the latest run — re-detection may have replaced it; reload the list"}, nil
	}
	sug, err := s.Suggester.Suggest(ctx, cand.Dest)
	if err != nil {
		// The seam degrades visibly (invariant 7's shape): a reachable API
		// with an unreachable geocoder says so instead of pretending the
		// null answer.
		return SuggestCandidateName502JSONResponse{Error: err.Error()}, nil
	}
	out := NameSuggestion{Source: sug.Source}
	if sug.Name != "" {
		out.Name = &sug.Name
	}
	return SuggestCandidateName200JSONResponse(out), nil
}

func (s *Server) DecideCandidate(ctx context.Context, req DecideCandidateRequestObject) (DecideCandidateResponseObject, error) {
	action := DecisionAction(req.Body.Action)
	if !action.Valid() {
		return DecideCandidate400JSONResponse{Error: "action must be 'confirmed' or 'dismissed'"}, nil
	}
	var name *string
	if action == DecisionActionConfirmed {
		trimmed := ""
		if req.Body.Name != nil {
			trimmed = strings.TrimSpace(*req.Body.Name)
		}
		if trimmed == "" {
			return DecideCandidate400JSONResponse{Error: "confirming requires a non-empty name"}, nil
		}
		name = &trimmed
	}

	cand, err := s.Store.LatestCandidate(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if cand == nil {
		return DecideCandidate404JSONResponse{Error: "no such candidate in the latest run — re-detection may have replaced it; reload the list"}, nil
	}

	// If this candidate already has a matched decision, the user is re-deciding:
	// update it in place and refresh its anchor to what they are looking at now.
	// Otherwise this is a fresh decision.
	_, _, _, matched, err := s.matchedState(ctx)
	if err != nil {
		return nil, err
	}
	var row store.DecisionRow
	if did, ok := matched[cand.ID]; ok {
		row, err = s.Store.UpdateDecision(ctx, did, string(action), name, *cand)
	} else {
		row, err = s.Store.InsertDecision(ctx, string(action), name, *cand)
	}
	if err != nil {
		return nil, err
	}
	return DecideCandidate200JSONResponse(toAPIDecision(row)), nil
}

// matchedState loads the latest run, its candidates, all decisions, and the
// recomputed candidate→decision association (BRIEF §3.1: matching is derived,
// never stored).
func (s *Server) matchedState(ctx context.Context) (*store.Run, []store.CandidateRow, []store.DecisionRow, map[int64]int64, error) {
	run, cands, err := s.Store.LatestRun(ctx)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	decs, err := s.Store.ListDecisions(ctx)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	refs := make([]detect.SpanRef, len(cands))
	for i, c := range cands {
		refs[i] = detect.SpanRef{ID: c.ID, Start: c.SpanStart, End: c.SpanEnd, Dest: c.Dest}
	}
	anchors := make([]detect.Anchor, len(decs))
	for i, d := range decs {
		anchors[i] = detect.Anchor{ID: d.ID, Start: d.AnchorStart, End: d.AnchorEnd, Dest: d.AnchorDest, CreatedAt: d.CreatedAt}
	}
	return run, cands, decs, detect.Match(refs, anchors, s.MatchParams), nil
}

func toAPICandidate(c store.CandidateRow) Candidate {
	modes := make([]ModeCount, len(c.Modes))
	for i, m := range c.Modes {
		modes[i] = ModeCount{Mode: m.Mode, N: m.N}
	}
	out := Candidate{
		Id:             c.ID,
		SpanStart:      c.SpanStart,
		SpanEnd:        c.SpanEnd,
		Days:           c.Days,
		Dest:           LatLng{Lat: c.Dest.Lat, Lon: c.Dest.Lon},
		DestKm:         c.DestKm,
		TrackKm:        c.TrackKm,
		Stops:          c.Stops,
		Repeat:         c.Repeat,
		ObsCount:       c.ObsCount,
		StartTruncated: c.StartTruncated,
		EndTruncated:   c.EndTruncated,
		Modes:          modes,
		Score:          c.Score,
	}
	// Both stay absent (not zero, not empty) on rows from pre-scoring runs.
	if c.ScoreBreakdown != nil {
		comps := make([]ScoreComponent, len(c.ScoreBreakdown))
		for i, sc := range c.ScoreBreakdown {
			comps[i] = ScoreComponent{
				Name: sc.Name, Present: sc.Present, Weight: sc.Weight,
				Raw: sc.Raw, Unit: sc.Unit, Normalized: sc.Normalized,
				Contribution: sc.Contribution,
			}
		}
		out.ScoreBreakdown = &comps
	}
	return out
}

func toAPIDecision(d store.DecisionRow) Decision {
	return Decision{Id: d.ID, Action: DecisionAction(d.Action), Name: d.Name, UpdatedAt: d.UpdatedAt}
}
