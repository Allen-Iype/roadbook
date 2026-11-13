// Package api implements the generated StrictServerInterface (gen.go, produced
// from api/openapi.yaml — never edited by hand). Handlers translate between
// store rows and contract types and run candidate↔decision matching; business
// rules live in internal/detect, persistence in internal/store.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"roadbook/internal/auth"
	"roadbook/internal/detect"
	"roadbook/internal/domain"
	"roadbook/internal/journey"
	"roadbook/internal/route"
	"roadbook/internal/store"
	"roadbook/internal/suggest"
)

// RouteProfile is the routing profile journeys are served with. One profile
// today; the cache key carries it so a second (cycling, walking) is a new
// key space, not a migration.
const RouteProfile = "driving"

type Server struct {
	Store       *store.Store
	MatchParams detect.MatchParams
	Suggester   suggest.Suggester
	// Photos is the thumbnail directory — the filesystem half of the photos
	// stratum, reachable only through this process (phase 4 BRIEF §1.3).
	Photos store.PhotoFiles
	// Uploads is the retained-exports directory (phase 7 BRIEF §3C) —
	// real location history, same safety class as data/.
	Uploads store.UploadFiles
	// importMu is the one-import-at-a-time guard (phase 7 BRIEF §1.2):
	// held from an accepted upload until its background import finalises.
	importMu sync.Mutex
	// Auth is the OIDC service in multi-user mode; nil is mode off — the
	// authless single-user reference (phase 13 BRIEF §2), in which no auth
	// surface exists and every request is store.SelfUser.
	Auth *auth.Service
	// UserCap is the account limit in multi-user mode (phase 15 BRIEF §9):
	// 0 is unlimited; otherwise a first sign-in is refused once this many
	// OIDC accounts exist, and the sign-in page shows its "full" state.
	// Returning accounts are never refused. Unread when Auth is nil.
	UserCap int
}

var _ StrictServerInterface = (*Server)(nil)

func (s *Server) GetHealth(ctx context.Context, _ GetHealthRequestObject) (GetHealthResponseObject, error) {
	// Readiness, not liveness: compose gates dependent services on this
	// answer (phase 5 BRIEF §3A), so a listening socket with a dead database
	// behind it must say 503, not ok.
	if err := s.Store.Ping(ctx); err != nil {
		return GetHealth503JSONResponse{Error: "database unreachable: " + err.Error()}, nil
	}
	return GetHealth200JSONResponse{Status: "ok"}, nil
}

func (s *Server) ListImports(ctx context.Context, _ ListImportsRequestObject) (ListImportsResponseObject, error) {
	rows, err := s.Store.ListImports(ctx, s.currentUser(ctx))
	if err != nil {
		return nil, err
	}
	resp := ImportList{Imports: []Import{}}
	for _, r := range rows {
		resp.Imports = append(resp.Imports, toAPIImport(r))
	}
	return ListImports200JSONResponse(resp), nil
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
	cand, err := s.Store.LatestCandidate(ctx, s.currentUser(ctx), req.Id)
	if err != nil {
		return nil, err
	}
	if cand == nil {
		return GetCandidateJourney404JSONResponse{Error: "this trip isn't in your list any more — reload the page"}, nil
	}
	out, _, err := s.journeyFor(ctx, s.currentUser(ctx), cand)
	if err != nil {
		return nil, err
	}
	return GetCandidateJourney200JSONResponse(out), nil
}

// journeyFor is the full contract-shaped journey for one candidate: the
// assembled, route-applied legs plus the two derived lines the page prints
// beside them — the source-asserted mode breakdown and the countries
// crossed. One function because two consumers render the same plate: the
// owner's page (GetCandidateJourney) and the shared view (CP4), and a
// stranger must see exactly what the owner sees, legs and kinds included.
func (s *Server) journeyFor(ctx context.Context, userID string, cand *store.CandidateRow) (Journey, journey.Journey, error) {
	j, obs, err := s.assembledJourney(ctx, userID, cand)
	if err != nil {
		return Journey{}, j, err
	}
	out, err := toAPIJourney(j)
	if err != nil {
		return Journey{}, j, err
	}

	// Source-asserted mode figures (phase 11 §6.2): absent — not empty —
	// when the window holds no activities at all (photo-sourced journeys),
	// so the display can say "no mode record" instead of zeros.
	if len(obs.Activities) > 0 {
		bd := journey.ModeBreakdown(obs.Activities, cand.SpanStart, cand.SpanEnd)
		apiBd := make([]ModeKm, len(bd))
		for i, m := range bd {
			apiBd[i] = ModeKm{Mode: m.Mode, Km: m.Km, Hours: m.Hours}
		}
		out.ModeBreakdown = &apiBd
	}

	// Where each stop was (phase 18): the nearest listed town, when one is
	// close enough to say. Absent, never empty, on a stop that goes unnamed;
	// the thresholds are echoed so the wording can be explained afterwards.
	pp := journey.DefaultPlaceParams()
	out.PlaceParams = PlaceParams{NearKm: pp.NearKm, MaxKm: pp.MaxKm}
	positions, names, err := StopNames(ctx, s.Store, j, obs.Visits, pp)
	if err != nil {
		return Journey{}, j, err
	}
	for i := range out.Stops {
		if positions[i].OK {
			from := StopNamedFrom(positions[i].Kind)
			out.Stops[i].NamedFrom = &from
		}
		if n := names[i]; n != nil {
			out.Stops[i].Place = &StopPlace{Name: n.Name, DistanceKm: n.DistanceKm, Source: n.Source, Near: n.Near}
		}
	}

	// Where each night was spent (phase 19): one entry per night, in order,
	// so the list's length is always summary.civil_days - 1. Empty, never
	// absent, on a day trip. A night names nothing itself — one at a stop
	// points at the stop, which carries its town above.
	np := journey.DefaultNightParams()
	out.NightParams = NightParams{StartHour: np.StartHour, EndHour: np.EndHour, MinHours: np.MinHours}
	nights := journey.Nights(j, np)
	out.Nights = make([]Night, len(nights))
	for i, n := range nights {
		out.Nights[i] = Night{Date: n.Date, Kind: NightKind(n.Kind)}
		if n.Kind == journey.NightStop {
			idx := n.StopIndex
			out.Nights[i].StopIndex = &idx
		}
		if n.Kind != journey.NightUnknown {
			h := n.Hours
			out.Nights[i].Hours = &h
		}
	}

	// Which regions each leg and each stop was in (phase 18, #17). Every
	// leg carries a list — empty, never absent, like the trip-level list
	// below; a stop's region is absent when its position is in none.
	legStates, stopStates, err := Regions(ctx, s.Store, j, positions)
	if err != nil {
		return Journey{}, j, err
	}
	for i := range out.Legs {
		out.Legs[i].States = make([]State, 0, len(legStates[i]))
		for _, r := range legStates[i] {
			out.Legs[i].States = append(out.Legs[i].States, State{Code: r.Code, Name: r.Name, CountryCode: r.CountryCode})
		}
	}
	for i := range out.Stops {
		if r := stopStates[i]; r != nil {
			out.Stops[i].State = &State{Code: r.Code, Name: r.Name, CountryCode: r.CountryCode}
		}
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
		return Journey{}, j, err
	}
	out.Countries = make([]Country, 0, len(crossed))
	for _, c := range crossed {
		out.Countries = append(out.Countries, Country{IsoCode: c.ISOCode, Name: c.Name})
	}
	// Admin-1 regions the same way (phase 14 CP1). Empty, never absent:
	// an unloaded states table reads as "no regions" on every surface.
	regions, err := s.Store.StatesForPoints(ctx, pts)
	if err != nil {
		return Journey{}, j, err
	}
	out.States = make([]State, 0, len(regions))
	for _, r := range regions {
		out.States = append(out.States, State{Code: r.Code, Name: r.Name, CountryCode: r.CountryCode})
	}
	return out, j, nil
}

// assembledJourney is the one journey pipeline both consumers run —
// GetCandidateJourney for the page, photo placement for the strip and map —
// so a photo's distance and the drawn route can never disagree: they read
// the same assembly and the same routing cache (phase 3 BRIEF §1.2: the
// serve binary never dials a router; a gap with no cached answer stays
// unknown and renders as such).
// assembledJourney also hands back the window's raw observations: the journey
// handler derives the source-asserted mode breakdown from their activities —
// deliberately outside Assemble, whose output the golden contract pins.
func (s *Server) assembledJourney(ctx context.Context, userID string, cand *store.CandidateRow) (journey.Journey, domain.Observations, error) {
	obs, err := s.Store.LoadJourneyInputs(ctx, userID, cand.SpanStart, cand.SpanEnd)
	if err != nil {
		return journey.Journey{}, obs, err
	}
	j := journey.Assemble(obs, cand.SpanStart, cand.SpanEnd, journey.DefaultParams())
	lookup, err := s.Store.LookupRoutes(ctx, route.UnknownKeys(j, RouteProfile))
	if err != nil {
		return journey.Journey{}, obs, err
	}
	return route.Apply(j, RouteProfile, lookup), obs, nil
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
		AirKm:           j.AirKm,
		UnknownKm:       j.UnknownKm,
		RoutedKm:        j.RoutedKm,
		GroundKm:        j.GroundKm(),
		GoogleGroundKm:  j.GoogleGroundKm,
		GoogleKm:        j.GoogleDistanceKm,
		MergedPoints:    j.MergedPoints(),
		TracePointsKept: j.TracePointsKept,
		RawPointsKept:   j.RawPointsKept,
	}
	// Both stay absent when Google recorded no ground distance — nothing to
	// compare is not zero divergence.
	if pct, ok := j.DivergencePct(); ok {
		out.DivergencePct = &pct
		flagged := j.DivergenceFlagged()
		out.DivergenceFlagged = &flagged
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
		if l.GapKind == journey.GapRoad {
			pts := make([]LatLng, len(l.RoutedPoints))
			for i, p := range l.RoutedPoints {
				pts[i] = LatLng{Lat: p.Lat, Lon: p.Lon}
			}
			al.RoutedPoints = &pts
			rk := l.RoutedKm
			al.RoutedKm = &rk
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
	// The summary block (phase 14 CP2): derived beside the assembly, the
	// same function the CLI prints from. Pace stays absent, not zero, when
	// no observed leg has a duration.
	sum := journey.Summarize(j)
	out.Summary = JourneySummary{
		SpanHours: sum.SpanHours, CivilDays: sum.CivilDays,
		Stops: sum.Stops, DwellHours: sum.DwellHours,
	}
	if sum.PaceOK {
		oh, pace := sum.ObservedHours, sum.ObservedPaceKmh
		out.Summary.ObservedHours = &oh
		out.Summary.ObservedPaceKmh = &pace
	}
	return out, nil
}

// SuggestCandidateName is a one-shot lookup through the configured Suggester
// (BRIEF §1.7). The suggestion prefills the confirm step's name input and is
// never applied automatically; the null suggester's empty answer renders as
// exactly today's empty input.
func (s *Server) SuggestCandidateName(ctx context.Context, req SuggestCandidateNameRequestObject) (SuggestCandidateNameResponseObject, error) {
	cand, err := s.Store.LatestCandidate(ctx, s.currentUser(ctx), req.Id)
	if err != nil {
		return nil, err
	}
	if cand == nil {
		return SuggestCandidateName404JSONResponse{Error: "this trip isn't in your list any more — reload the page"}, nil
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
		if n := utf8.RuneCountInString(trimmed); n > maxNameRunes {
			return DecideCandidate400JSONResponse{Error: fmt.Sprintf("the name is %d characters; the most is %d", n, maxNameRunes)}, nil
		}
		name = &trimmed
	}

	cand, err := s.Store.LatestCandidate(ctx, s.currentUser(ctx), req.Id)
	if err != nil {
		return nil, err
	}
	if cand == nil {
		return DecideCandidate404JSONResponse{Error: "this trip isn't in your list any more — reload the page"}, nil
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
		row, err = s.Store.UpdateDecision(ctx, s.currentUser(ctx), did, string(action), name, *cand)
	} else {
		row, err = s.Store.InsertDecision(ctx, s.currentUser(ctx), string(action), name, *cand)
	}
	if err != nil {
		return nil, err
	}
	return DecideCandidate200JSONResponse(toAPIDecision(row)), nil
}

// Limits on the owner's words (phase 17, #3), counted in characters
// (runes), as the spec's maxLength counts them. Generous for a title and a
// few paragraphs; small enough that the candidate list, which carries
// every decision, stays a few kilobytes per trip at most.
const (
	maxNameRunes  = 120
	maxNotesRunes = 4000
)

// EditDecision changes a kept trip's name and notes (phase 17, #3) and
// nothing else. The candidate id resolves to its matched decision exactly
// as DecideCandidate resolves it; only a confirmed decision is editable.
func (s *Server) EditDecision(ctx context.Context, req EditDecisionRequestObject) (EditDecisionResponseObject, error) {
	name := strings.TrimSpace(req.Body.Name)
	if name == "" {
		return EditDecision400JSONResponse{Error: "a trip needs a name"}, nil
	}
	if n := utf8.RuneCountInString(name); n > maxNameRunes {
		return EditDecision400JSONResponse{Error: fmt.Sprintf("the name is %d characters; the most is %d", n, maxNameRunes)}, nil
	}
	var notes *string
	if req.Body.Notes != nil {
		// Trailing space goes; the owner's line breaks and indentation
		// inside the text stay as typed.
		if t := strings.TrimRightFunc(*req.Body.Notes, unicode.IsSpace); strings.TrimSpace(t) != "" {
			if n := utf8.RuneCountInString(t); n > maxNotesRunes {
				return EditDecision400JSONResponse{Error: fmt.Sprintf("the notes are %d characters; the most is %d", n, maxNotesRunes)}, nil
			}
			notes = &t
		}
	}

	user := s.currentUser(ctx)
	cand, err := s.Store.LatestCandidate(ctx, user, req.Id)
	if err != nil {
		return nil, err
	}
	if cand == nil {
		return EditDecision404JSONResponse{Error: "this trip isn't in your list any more — reload the page"}, nil
	}
	_, _, _, matched, err := s.matchedState(ctx)
	if err != nil {
		return nil, err
	}
	did, ok := matched[cand.ID]
	if !ok {
		return EditDecision404JSONResponse{Error: "only a kept trip has a name to change — keep it first"}, nil
	}
	row, ok, err := s.Store.EditDecisionText(ctx, user, did, name, notes)
	if err != nil {
		return nil, err
	}
	if !ok {
		return EditDecision404JSONResponse{Error: "only a kept trip has a name to change — keep it first"}, nil
	}
	return EditDecision200JSONResponse(toAPIDecision(row)), nil
}

// DecideCandidatesBulk is the atomic bulk-triage write (phase 11 §6.1):
// every item validated up front against one matchedState snapshot, then one
// store transaction. Per-item semantics — anchoring, re-decide-in-place —
// are exactly DecideCandidate's; the batch adds nothing but the transaction.
func (s *Server) DecideCandidatesBulk(ctx context.Context, req DecideCandidatesBulkRequestObject) (DecideCandidatesBulkResponseObject, error) {
	items := req.Body.Decisions
	if len(items) == 0 {
		return DecideCandidatesBulk400JSONResponse{Error: "decisions must not be empty"}, nil
	}

	_, cands, _, matched, err := s.matchedState(ctx)
	if err != nil {
		return nil, err
	}
	byID := make(map[int64]*store.CandidateRow, len(cands))
	for i := range cands {
		byID[cands[i].ID] = &cands[i]
	}

	seen := make(map[int64]bool, len(items))
	var missing []string
	bulk := make([]store.BulkDecision, 0, len(items))
	for _, it := range items {
		action := DecisionAction(it.Action)
		if !action.Valid() {
			return DecideCandidatesBulk400JSONResponse{Error: fmt.Sprintf("candidate %d: action must be 'confirmed' or 'dismissed'", it.Id)}, nil
		}
		var name *string
		if action == DecisionActionConfirmed {
			trimmed := ""
			if it.Name != nil {
				trimmed = strings.TrimSpace(*it.Name)
			}
			if trimmed == "" {
				return DecideCandidatesBulk400JSONResponse{Error: fmt.Sprintf("candidate %d: confirming requires a non-empty name", it.Id)}, nil
			}
			if n := utf8.RuneCountInString(trimmed); n > maxNameRunes {
				return DecideCandidatesBulk400JSONResponse{Error: fmt.Sprintf("candidate %d: the name is %d characters; the most is %d", it.Id, n, maxNameRunes)}, nil
			}
			name = &trimmed
		}
		if seen[it.Id] {
			return DecideCandidatesBulk400JSONResponse{Error: fmt.Sprintf("candidate %d appears twice — one decision per candidate", it.Id)}, nil
		}
		seen[it.Id] = true

		cand, ok := byID[it.Id]
		if !ok {
			missing = append(missing, fmt.Sprintf("%d", it.Id))
			continue
		}
		b := store.BulkDecision{Anchor: *cand, Action: string(action), Name: name}
		if did, ok := matched[cand.ID]; ok {
			id := did
			b.UpdateID = &id
		}
		bulk = append(bulk, b)
	}
	if len(missing) > 0 {
		return DecideCandidatesBulk404JSONResponse{Error: "these trips aren't in your list any more — reload the page: " + strings.Join(missing, ", ")}, nil
	}

	if err := s.Store.DecideBulk(ctx, s.currentUser(ctx), bulk); err != nil {
		return nil, err
	}
	return DecideCandidatesBulk200JSONResponse(BulkDecisionResult{Decided: len(bulk)}), nil
}

// matchedState loads the latest run, its candidates, all decisions, and the
// recomputed candidate→decision association (BRIEF §3.1: matching is derived,
// never stored).
func (s *Server) matchedState(ctx context.Context) (*store.Run, []store.CandidateRow, []store.DecisionRow, map[int64]int64, error) {
	return s.matchedStateFor(ctx, s.currentUser(ctx))
}

// matchedStateFor is matchedState for an explicit owner — the shared view
// (CP4) reads as the link's owner, not as the requester, who has no session.
func (s *Server) matchedStateFor(ctx context.Context, userID string) (*store.Run, []store.CandidateRow, []store.DecisionRow, map[int64]int64, error) {
	run, cands, err := s.Store.LatestRun(ctx, userID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	decs, err := s.Store.ListDecisions(ctx, userID)
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
	return Decision{Id: d.ID, Action: DecisionAction(d.Action), Name: d.Name, Notes: d.Notes, UpdatedAt: d.UpdatedAt}
}
