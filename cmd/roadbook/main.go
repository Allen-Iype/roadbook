// Command roadbook is the pipeline entry point: one binary, subcommands per
// stage. Phase 1 checkpoint 1 ships detect and probe; import (Postgres) and
// serve (HTTP API) arrive with checkpoint 2.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"roadbook/internal/api"
	"roadbook/internal/countries"
	"roadbook/internal/detect"
	"roadbook/internal/domain"
	"roadbook/internal/journey"
	"roadbook/internal/photosource"
	"roadbook/internal/route"
	"roadbook/internal/store"
	"roadbook/internal/suggest"
	"roadbook/internal/timeline"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "detect":
		err = runDetect(os.Args[2:])
	case "journey":
		err = runJourney(os.Args[2:])
	case "route":
		err = runRoute(os.Args[2:])
	case "photo":
		err = runPhoto(os.Args[2:])
	case "probe":
		err = runProbe(os.Args[2:])
	case "migrate":
		err = runMigrate(os.Args[2:])
	case "import":
		err = runImport(os.Args[2:])
	case "countries":
		err = runCountries(os.Args[2:])
	case "serve":
		err = runServe(os.Args[2:])
	case "backup":
		err = runBackup(os.Args[2:])
	case "restore":
		err = runRestore(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "roadbook:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage:
  roadbook migrate [-db url]
  roadbook import  -src <timeline export.json> [-db url] [-from date] [-to date] [-label name]
  roadbook countries [-src <admin-0 geojson[.gz]>] [-if-empty] [-db url]
  roadbook detect  (-src <export.json> | -db url) [threshold flags] [-json out.json]
  roadbook journey (-src <export.json> -from <RFC3339> -to <RFC3339> | -candidate id [-db url]) [threshold flags]
  roadbook route   [-db url] [-router none|osrm] [-router-url url] [-profile driving] [-interval 1s] [-dataset name] [-all | -candidate id] [-refresh]
  roadbook serve   [-db url] [-addr :8080]
  roadbook backup  -out <file.tar.gz> [-db url] [-photos-dir dir]
  roadbook restore -src <file.tar.gz> [-db url] [-photos-dir dir]
  roadbook photo   -inspect <photo.jpg | sidecar.json>
  roadbook probe   -src <timeline export.json>

'migrate' applies embedded schema migrations.
'import' parses an export and stores observations idempotently.
'countries' loads country polygons for point-in-polygon attribution: the
bundled Natural Earth 1:110m set by default, or a higher-resolution admin-0
file from disk via -src. Replaces the table wholesale; never fetches.
'detect' finds adventure candidates: from a file (prints only) or from the
database (persists a run and its candidates, then prints).
'journey' assembles one window of observations into observed and gap legs and
prints the reconstruction — the command behind every journey figure. With
-candidate it reads the candidate's span from the database and applies the
routing cache, exactly as the API does; with -src it is pure file-in.
'route' fills unknown gaps from a router, in batch, into the route_cache
table — the only network step in routing, and entirely optional: with the
default null router it inventories the gaps and fills nothing. Scope
defaults to confirmed adventures of the latest run.
'serve' runs the HTTP API.
'backup' writes the irreplaceable set — decisions, photo rows, thumbnail
files — as one archive. Everything else regenerates from your export files.
'restore' merges an archive into the connected instance by durable identity
(decision anchor, photo content hash); overlap is skipped and reported.
Restored decisions attach to candidates at the next import + detection.
'photo' inspects one photo or Takeout sidecar: the format verdict, every
metadata reading with its source, the resolved capture instant, and the
thumbnail that would be produced. Writes nothing; a sidecar named beside the
photo joins in automatically.
'probe' reports every JSON key path in an export and its frequency; diff two
probes to spot unannounced schema changes.

-db defaults to $DATABASE_URL.`)
}

// dbFlag wires -db with $DATABASE_URL as the default. Configuration is via
// environment (docs/PLAN.md phase 5); the flag exists for one-off overrides.
func dbFlag(fs *flag.FlagSet) *string {
	return fs.String("db", os.Getenv("DATABASE_URL"), "Postgres URL (default $DATABASE_URL)")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func openStore(ctx context.Context, dbURL string) (*store.Store, error) {
	if dbURL == "" {
		return nil, fmt.Errorf("no database configured: set DATABASE_URL or pass -db")
	}
	return store.Open(ctx, dbURL)
}

func runMigrate(args []string) error {
	fs := flag.NewFlagSet("migrate", flag.ExitOnError)
	db := dbFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *db == "" {
		return fmt.Errorf("no database configured: set DATABASE_URL or pass -db")
	}
	n, err := store.Migrate(context.Background(), *db)
	if err != nil {
		return err
	}
	fmt.Printf("applied %d migration(s)\n", n)
	return nil
}

func runImport(args []string) error {
	fs := flag.NewFlagSet("import", flag.ExitOnError)
	src := fs.String("src", "", "path to a Timeline export (required)")
	db := dbFlag(fs)
	from := fs.String("from", "", "only import observations starting on/after this date (YYYY-MM-DD, UTC)")
	to := fs.String("to", "", "only import observations starting before this date (YYYY-MM-DD, UTC)")
	label := fs.String("label", "", "provenance label for this import (default: source file name)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *src == "" {
		fs.Usage()
		return fmt.Errorf("-src is required")
	}

	winStart, err := parseDateFlag(*from)
	if err != nil {
		return fmt.Errorf("-from: %w", err)
	}
	winEnd, err := parseDateFlag(*to)
	if err != nil {
		return fmt.Errorf("-to: %w", err)
	}

	// The imports row is written before the parse (status 'running') and
	// finalised after it, so a failed import is visible in the product, with
	// the sniffer's format label recorded queryably beside the prose message
	// (phase 5 BRIEF §3B). Only a file that cannot be opened at all fails
	// before the row exists.
	f, err := os.Open(*src)
	if err != nil {
		return err
	}
	defer f.Close()

	ctx := context.Background()
	s, err := openStore(ctx, *db)
	if err != nil {
		return err
	}
	defer s.Close()

	lbl := *label
	if lbl == "" {
		lbl = filepath.Base(*src)
	}
	importID, err := s.BeginImport(ctx, lbl, winStart, winEnd)
	if err != nil {
		return err
	}

	obs, st, err := timeline.Parse(f)
	if err != nil {
		var ue *timeline.UnsupportedInputError
		kind := ""
		if errors.As(err, &ue) {
			kind = ue.Kind
		}
		if ferr := s.FailImport(ctx, importID, kind, err.Error()); ferr != nil {
			return fmt.Errorf("cannot import %s: %w (and recording the failure also failed: %v)", filepath.Base(*src), err, ferr)
		}
		return fmt.Errorf("cannot import %s: %w", filepath.Base(*src), err)
	}
	fmt.Printf("parsed %s: %d visits, %d activities, %d path points, %d raw positions (%d skipped)\n",
		filepath.Base(*src), st.Visits, st.Activities, st.Points, st.RawPositions, st.Skipped)

	if winStart != nil || winEnd != nil {
		obs = filterWindow(obs, winStart, winEnd)
		fmt.Printf("window filter kept %d visits, %d activities, %d path points, %d raw positions\n",
			len(obs.Visits), len(obs.Activities), len(obs.Points), len(obs.RawPositions))
	}

	res, err := s.ImportObservations(ctx, importID, st.Format, obs, st.Skipped)
	if err != nil {
		if ferr := s.FailImport(ctx, importID, st.Format, err.Error()); ferr != nil {
			return fmt.Errorf("import failed: %w (and recording the failure also failed: %v)", err, ferr)
		}
		return err
	}
	fmt.Printf("import %d: %d observations, %d new (%d already present)\n",
		res.ImportID, res.Parsed, res.Inserted, res.Parsed-res.Inserted)
	return nil
}

func parseDateFlag(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// filterWindow keeps observations whose start falls in [from, to). Inputs are
// immutable — this builds new slices, it never mutates.
func filterWindow(obs domain.Observations, from, to *time.Time) domain.Observations {
	in := func(t time.Time) bool {
		if from != nil && t.Before(*from) {
			return false
		}
		if to != nil && !t.Before(*to) {
			return false
		}
		return true
	}
	var out domain.Observations
	for _, v := range obs.Visits {
		if in(v.Start) {
			out.Visits = append(out.Visits, v)
		}
	}
	for _, a := range obs.Activities {
		if in(a.Start) {
			out.Activities = append(out.Activities, a)
		}
	}
	for _, p := range obs.Points {
		if in(p.Time) {
			out.Points = append(out.Points, p)
		}
	}
	for _, rp := range obs.RawPositions {
		if in(rp.Time) {
			out.RawPositions = append(out.RawPositions, rp)
		}
	}
	return out
}

// runCountries loads country polygons for point-in-polygon attribution. The
// default source is the file embedded in internal/countries — no network
// fetch at any time (BRIEF §3E); -src accepts a higher-resolution Natural
// Earth admin-0 file from disk, gzipped or plain.
func runCountries(args []string) error {
	fs := flag.NewFlagSet("countries", flag.ExitOnError)
	src := fs.String("src", "", "Natural Earth admin-0 GeoJSON, .geojson or .gz (default: bundled 1:110m)")
	ifEmpty := fs.Bool("if-empty", false, "load only when the countries table is empty (startup-safe: never overwrites an existing load)")
	db := dbFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	var list []countries.Country
	label := "bundled Natural Earth 1:110m"
	if *src == "" {
		var err error
		list, err = countries.Bundled()
		if err != nil {
			return err
		}
	} else {
		f, err := os.Open(*src)
		if err != nil {
			return err
		}
		list, err = countries.Parse(f)
		f.Close()
		if err != nil {
			return fmt.Errorf("cannot load %s: %w", filepath.Base(*src), err)
		}
		label = filepath.Base(*src)
	}

	ctx := context.Background()
	s, err := openStore(ctx, *db)
	if err != nil {
		return err
	}
	defer s.Close()
	if *ifEmpty {
		n, err := s.CountCountries(ctx)
		if err != nil {
			return err
		}
		if n > 0 {
			fmt.Printf("countries table already holds %d rows; -if-empty leaves it untouched\n", n)
			return nil
		}
	}
	if err := s.ReplaceCountries(ctx, list); err != nil {
		return err
	}
	fmt.Printf("loaded %d countries from %s\n", len(list), label)
	return nil
}

func runServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	db := dbFlag(fs)
	addr := fs.String("addr", ":8080", "listen address")
	geocoder := fs.String("geocoder", envOr("ROADBOOK_GEOCODER", "none"),
		"name suggester: none | nominatim (default $ROADBOOK_GEOCODER)")
	nominatimURL := fs.String("nominatim-url", envOr("ROADBOOK_NOMINATIM_URL", "https://nominatim.openstreetmap.org"),
		"Nominatim base URL (default $ROADBOOK_NOMINATIM_URL)")
	photosDir := fs.String("photos-dir", envOr("ROADBOOK_PHOTOS_DIR", "data/photos"),
		"thumbnail directory (default $ROADBOOK_PHOTOS_DIR) — under gitignored data/ by default; photos are user data")
	uploadsDir := fs.String("uploads-dir", envOr("ROADBOOK_UPLOADS_DIR", "data/uploads"),
		"retained-uploads directory (default $ROADBOOK_UPLOADS_DIR) — under gitignored data/ by default; exports are real location history")
	if err := fs.Parse(args); err != nil {
		return err
	}

	// The Suggester seam (BRIEF §1.7): null by default — a self-hosted
	// product makes no surprise network calls; the geocoder is opt-in.
	var sug suggest.Suggester
	switch *geocoder {
	case "none":
		sug = suggest.Null{}
	case "nominatim":
		sug = suggest.NewNominatim(*nominatimURL)
	default:
		return fmt.Errorf("unknown -geocoder %q: use none or nominatim", *geocoder)
	}

	ctx := context.Background()
	s, err := openStore(ctx, *db)
	if err != nil {
		return err
	}
	defer s.Close()

	photos := store.PhotoFiles{Dir: *photosDir}
	if err := photos.Init(); err != nil {
		return fmt.Errorf("photos directory: %w", err)
	}
	uploads := store.UploadFiles{Dir: *uploadsDir}
	if err := uploads.Init(); err != nil {
		return fmt.Errorf("uploads directory: %w", err)
	}

	// A 'running' import whose goroutine died with the last process would
	// say running forever; finalise it visibly (phase 7 BRIEF §1.2). The
	// retained file makes retry cheap.
	if swept, err := s.SweepRunningImports(ctx,
		"interrupted by a server restart — upload the file again to retry"); err != nil {
		return fmt.Errorf("sweeping interrupted imports: %w", err)
	} else if swept > 0 {
		fmt.Printf("marked %d interrupted import(s) failed\n", swept)
	}

	srv := &api.Server{Store: s, MatchParams: detect.DefaultMatchParams(), Suggester: sug, Photos: photos, Uploads: uploads}
	handler := api.HandlerFromMux(api.NewStrictHandler(srv, nil), http.NewServeMux())
	fmt.Printf("roadbook API listening on %s\n", *addr)
	return http.ListenAndServe(*addr, handler)
}

func runDetect(args []string) error {
	fs := flag.NewFlagSet("detect", flag.ExitOnError)
	src := fs.String("src", "", "path to a Timeline export (file mode: print only)")
	photosDir := fs.String("photos", "", "path to a directory of geotagged photos (file mode: print only)")
	db := dbFlag(fs)
	useDB := fs.Bool("from-db", false, "detect over the database's observations and persist the run")
	jsonOut := fs.String("json", "", "also write full results as JSON to this path")
	p := detect.DefaultParams()
	fs.Float64Var(&p.NearM, "near-m", p.NearM, "'away' threshold from every active home base, metres")
	fs.Float64Var(&p.FarKm, "far-km", p.FarKm, "minimum distance of the dwelt destination, km")
	fs.IntVar(&p.MinObs, "min-obs", p.MinObs, "minimum observations in a span")
	fs.Float64Var(&p.MinHrs, "min-hrs", p.MinHrs, "minimum span duration, hours")
	fs.Float64Var(&p.MinDwellMin, "min-dwell-min", p.MinDwellMin, "minimum stay to count as dwelling, minutes")
	fs.Float64Var(&p.MaxKmh, "max-kmh", p.MaxKmh, "implied-speed outlier threshold, km/h")
	fs.Float64Var(&p.Score.WeightDistance, "score-w-distance", p.Score.WeightDistance, "score weight: distance from home")
	fs.Float64Var(&p.Score.WeightDwell, "score-w-dwell", p.Score.WeightDwell, "score weight: destination dwell")
	fs.Float64Var(&p.Score.WeightDensity, "score-w-density", p.Score.WeightDensity, "score weight: observation density")
	fs.Float64Var(&p.Score.WeightDuration, "score-w-duration", p.Score.WeightDuration, "score weight: span duration")
	fs.Float64Var(&p.Score.DistanceFullKm, "score-distance-full-km", p.Score.DistanceFullKm, "destination distance saturating its score component, km")
	fs.Float64Var(&p.Score.DwellFullHrs, "score-dwell-full-hrs", p.Score.DwellFullHrs, "destination dwell saturating its score component, hours")
	fs.Float64Var(&p.Score.DensityFullPerDay, "score-density-full", p.Score.DensityFullPerDay, "observations/day saturating its score component")
	fs.Float64Var(&p.Score.DurationFullDays, "score-duration-full-days", p.Score.DurationFullDays, "span days saturating its score component")
	fs.Float64Var(&p.Score.DestRadiusKm, "score-dest-radius-km", p.Score.DestRadiusKm, "dwells within this of the destination count as destination dwell, km")
	fs.Float64Var(&p.Synth.StayRadiusM, "stay-radius-m", p.Synth.StayRadiusM, "stay-point synthesis: a photo fix within this of the open stay's centroid extends it, metres")
	fs.Float64Var(&p.Synth.StayMinMin, "stay-min-min", p.Synth.StayMinMin, "stay-point synthesis: discard stays shorter than this, minutes")
	fs.Float64Var(&p.Synth.StayMaxGapMin, "stay-max-gap-min", p.Synth.StayMaxGapMin, "stay-point synthesis: a silence longer than this closes the open stay, minutes")
	fs.IntVar(&p.Synth.HomeMinDays, "home-min-days", p.Synth.HomeMinDays, "synthetic home evidence must recur across at least this many distinct days")
	fs.Float64Var(&p.Bases.GridPerDeg, "base-grid-per-deg", p.Bases.GridPerDeg, "home derivation: evidence grid cells per degree")
	fs.Float64Var(&p.Bases.MergeM, "base-merge-m", p.Bases.MergeM, "home derivation: merge clusters whose medians sit within this, metres")
	fs.IntVar(&p.Bases.MinVisits, "base-min-visits", p.Bases.MinVisits, "home derivation: minimum evidence count per base")
	fs.IntVar(&p.Bases.EraPadDays, "base-era-pad-days", p.Bases.EraPadDays, "home derivation: era padding around first..last evidence, days")
	if err := fs.Parse(args); err != nil {
		return err
	}
	modes := 0
	for _, on := range []bool{*src != "", *photosDir != "", *useDB} {
		if on {
			modes++
		}
	}
	if modes != 1 {
		fs.Usage()
		return fmt.Errorf("exactly one of -src (file mode), -photos (photo-directory mode), or -from-db (database mode) is required")
	}

	var obs domain.Observations
	var s *store.Store
	ctx := context.Background()
	if *photosDir != "" {
		entries, err := os.ReadDir(*photosDir)
		if err != nil {
			return err
		}
		var files []photosource.File
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			data, err := os.ReadFile(filepath.Join(*photosDir, e.Name()))
			if err != nil {
				return err
			}
			files = append(files, photosource.File{Name: e.Name(), Data: data})
		}
		var st photosource.Stats
		var results []photosource.FileResult
		obs, results, st = photosource.ParseFiles(files)
		fmt.Printf("parsed %d photos from %s: %d fixes, %d without position, %d without time, %d sidecars paired (%d unpaired), %d unsupported\n",
			st.Photos, filepath.Base(*photosDir), st.Fixes, st.NoPosition, st.NoTime,
			st.SidecarsPaired, st.SidecarsUnpaired, st.Unsupported)
		for _, r := range results {
			if r.Verdict == photosource.VerdictUnsupported {
				fmt.Printf("  %s: %s\n", r.Name, r.Message)
			}
		}
	} else if *useDB {
		var err error
		s, err = openStore(ctx, *db)
		if err != nil {
			return err
		}
		defer s.Close()
		obs, err = s.LoadObservations(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("loaded from database: %d visits, %d activities, %d path points\n",
			len(obs.Visits), len(obs.Activities), len(obs.Points))
	} else {
		f, err := os.Open(*src)
		if err != nil {
			return err
		}
		var st timeline.Stats
		obs, st, err = timeline.Parse(f)
		f.Close()
		if err != nil {
			return fmt.Errorf("cannot read %s: %w", filepath.Base(*src), err)
		}
		fmt.Printf("parsed %s: %d visits, %d activities, %d path points",
			filepath.Base(*src), st.Visits, st.Activities, st.Points)
		if st.Skipped > 0 {
			fmt.Printf(" (%d segments skipped)", st.Skipped)
		}
		fmt.Println()
	}

	res := detect.Run(obs, p)

	if s != nil {
		runID, err := s.SaveRun(ctx, p, res)
		if err != nil {
			return err
		}
		fmt.Printf("saved detection run %d (%d candidates); decisions re-match on read\n",
			runID, len(res.Candidates))
	}

	fmt.Printf("outliers dropped %d | home bases %d:", res.OutliersDropped, len(res.Bases))
	for i, b := range res.Bases {
		if i > 0 {
			fmt.Print(",")
		}
		fmt.Printf(" (%.3f,%.3f) n=%d active %s..%s",
			b.Center.Lat, b.Center.Lon, b.N, b.From.Format("2006-01-02"), b.To.Format("2006-01-02"))
	}
	fmt.Println()

	fmt.Printf("\n=== %d candidates ===\n", len(res.Candidates))
	fmt.Printf("%3s %-11s %5s %8s %6s %5s %4s %5s %-5s %s\n",
		"#", "start", "days", "dest_km", "track", "stops", "rpt", "score", "trunc", "dest")
	for i, c := range res.Candidates {
		trunc := ""
		if c.StartTruncated {
			trunc += "start"
		}
		if c.EndTruncated {
			if trunc != "" {
				trunc += "+"
			}
			trunc += "end"
		}
		fmt.Printf("%3d %-11s %5.1f %8d %6d %5d %4d %5.1f %-5s [%.4f, %.4f]\n",
			i+1, c.Start.Format("2006-01-02"), c.Days, c.DestKm, c.TrackKm,
			c.Stops, c.Repeat, c.Score, trunc, c.Dest.Lat, c.Dest.Lon)
	}

	if *jsonOut != "" {
		out := struct {
			Params detect.Params `json:"params"`
			detect.Result
		}{p, res}
		b, err := json.MarshalIndent(out, "", " ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(*jsonOut, b, 0o644); err != nil {
			return err
		}
		fmt.Printf("\nwrote %s\n", *jsonOut)
	}
	return nil
}

// runJourney is the reproduction command for journey figures (CLAUDE.md
// invariant 13): the golden fixture's numbers come from
//
//	roadbook journey -src testdata/journey-27jul2026.anon.json \
//	  -from 2026-07-27T19:46:35+05:30 -to 2026-07-28T07:12:22+05:30
func runJourney(args []string) error {
	fs := flag.NewFlagSet("journey", flag.ExitOnError)
	src := fs.String("src", "", "path to a Timeline export (file mode)")
	from := fs.String("from", "", "window start, RFC3339 (file mode)")
	to := fs.String("to", "", "window end, RFC3339 (file mode)")
	candidateID := fs.Int64("candidate", 0, "assemble a candidate's span from the database, cache-applied (DB mode)")
	db := dbFlag(fs)
	p := journey.DefaultParams()
	fs.Float64Var(&p.GapThresholdMinutes, "gap-min", p.GapThresholdMinutes,
		"silence longer than this becomes a gap leg, minutes")
	fs.Float64Var(&p.ThinSpacingSeconds, "thin-sec", p.ThinSpacingSeconds,
		"minimum spacing between kept points, seconds")
	fs.Float64Var(&p.MinStopDwellSeconds, "stop-sec", p.MinStopDwellSeconds,
		"minimum activity pause reported as a stop, seconds")
	fs.Float64Var(&p.MaxAccuracyM, "max-acc-m", p.MaxAccuracyM,
		"exclude raw positions with worse reported accuracy, metres (0 = off)")
	fs.Float64Var(&p.AirSpeedMinKmh, "air-kmh", p.AirSpeedMinKmh,
		"a gap implying at least this speed classifies as air (0 = off)")
	fs.Float64Var(&p.MaxSpeedKmh, "max-kmh", p.MaxSpeedKmh,
		"teleport rejection: cluster edge speed above this is impossible (0 = off)")
	fs.Float64Var(&p.ClusterRadiusM, "cluster-m", p.ClusterRadiusM,
		"teleport rejection: cluster radius, metres")
	if err := fs.Parse(args); err != nil {
		return err
	}

	var j journey.Journey
	// The window's activities, kept for the source-asserted mode breakdown —
	// computed outside Assemble (the golden contract pins assembly's output).
	var acts []domain.Activity
	switch {
	case *candidateID != 0:
		// DB mode mirrors the API handler exactly — Assemble, then apply the
		// routing cache — so CLI and page cannot disagree about the same
		// candidate. The serve rule holds here too: the cache is read, no
		// router is ever dialled.
		ctx := context.Background()
		s, err := openStore(ctx, *db)
		if err != nil {
			return err
		}
		defer s.Close()
		cand, err := s.LatestCandidate(ctx, *candidateID)
		if err != nil {
			return err
		}
		if cand == nil {
			return fmt.Errorf("candidate %d is not in the latest run", *candidateID)
		}
		obs, err := s.LoadJourneyInputs(ctx, cand.SpanStart, cand.SpanEnd)
		if err != nil {
			return err
		}
		j = journey.Assemble(obs, cand.SpanStart, cand.SpanEnd, p)
		acts = obs.Activities
		lookup, err := s.LookupRoutes(ctx, route.UnknownKeys(j, api.RouteProfile))
		if err != nil {
			return err
		}
		j = route.Apply(j, api.RouteProfile, lookup)
	case *src != "" && *from != "" && *to != "":
		winStart, err := time.Parse(time.RFC3339, *from)
		if err != nil {
			return fmt.Errorf("-from: %w", err)
		}
		winEnd, err := time.Parse(time.RFC3339, *to)
		if err != nil {
			return fmt.Errorf("-to: %w", err)
		}
		f, err := os.Open(*src)
		if err != nil {
			return err
		}
		obs, st, err := timeline.Parse(f)
		f.Close()
		if err != nil {
			return fmt.Errorf("cannot read %s: %w", filepath.Base(*src), err)
		}
		fmt.Printf("parsed %s: %d visits, %d activities, %d path points, %d raw positions\n",
			filepath.Base(*src), st.Visits, st.Activities, st.Points, st.RawPositions)
		j = journey.Assemble(obs, winStart, winEnd, p)
		acts = obs.Activities
	default:
		fs.Usage()
		return fmt.Errorf("either -candidate id, or -src with -from and -to")
	}

	fmt.Printf("\nwindow %s .. %s (%.1f h)\n",
		j.WindowStart.Format(time.RFC3339), j.WindowEnd.Format(time.RFC3339),
		j.WindowEnd.Sub(j.WindowStart).Hours())
	fmt.Printf("points: %d trace + %d raw in window -> %d kept (%d trace + %d raw)\n",
		j.TracePointsInWindow, j.RawPointsInWindow, j.MergedPoints(),
		j.TracePointsKept, j.RawPointsKept)
	if j.RejectedNullIsland+j.RejectedAccuracy+j.RejectedSpeed > 0 {
		fmt.Printf("anomalies excluded from assembly (rows untouched): %d null-island, %d accuracy, %d teleport\n",
			j.RejectedNullIsland, j.RejectedAccuracy, j.RejectedSpeed)
	}
	obsPct, infPct := 0.0, 0.0
	if j.TotalKm > 0 {
		obsPct = j.ObservedKm / j.TotalKm * 100
		infPct = j.InferredKm / j.TotalKm * 100
	}
	fmt.Printf("distance %.1f km: observed %.1f (%.1f%%) + inferred %.1f (%.1f%%)\n",
		j.TotalKm, j.ObservedKm, obsPct, j.InferredKm, infPct)
	if j.AirKm > 0 {
		fmt.Printf("of the inferred, %.1f km is air (great-circle; excluded from road validation)\n", j.AirKm)
	}
	if j.RoutedKm > 0 {
		fmt.Printf("routed roads cover %.1f km (cache); %.1f km of gaps stay unknown\n", j.RoutedKm, j.UnknownKm)
	}
	if j.GoogleDistanceKm > 0 {
		fmt.Printf("google's own figure %.1f km total\n", j.GoogleDistanceKm)
	}
	// Source-asserted per-mode figures (phase 11 §6.2) — the reproduction
	// command for the adventure page's mode line. Absent activities (a
	// photo-sourced journey) print as the absence they are, never zeros.
	if bd := journey.ModeBreakdown(acts, j.WindowStart, j.WindowEnd); len(bd) > 0 {
		fmt.Printf("modes, source-asserted (Google's labels, guesses):")
		for i, m := range bd {
			sep := " "
			if i > 0 {
				sep = " · "
			}
			fmt.Printf("%s%s %.1f km", sep, m.Mode, m.Km)
		}
		fmt.Println()
	} else {
		fmt.Println("no mode record — the window's evidence carries no activity data")
	}
	if pct, ok := j.DivergencePct(); ok {
		flag := ""
		if j.DivergenceFlagged() {
			flag = fmt.Sprintf("  FLAGGED (warn at %.0f%%)", j.Params.DivergenceWarnPct)
		}
		fmt.Printf("ground validation: %.1f km reconstructed vs %.1f km google ground (%+.1f%%)%s\n",
			j.GroundKm(), j.GoogleGroundKm, pct, flag)
	}

	fmt.Printf("\n=== %d legs ===\n", len(j.Legs))
	fmt.Printf("%3s %-8s %-7s %-9s %-9s %7s %6s %s\n",
		"#", "kind", "gap", "start", "end", "min", "km", "pts")
	for i, l := range j.Legs {
		fmt.Printf("%3d %-8s %-7s %-9s %-9s %7.1f %6.1f %d\n",
			i+1, l.Kind, l.GapKind,
			l.Start().Format("15:04:05"), l.End().Format("15:04:05"),
			l.End().Sub(l.Start()).Minutes(), l.DistanceKm, len(l.Points))
	}

	fmt.Printf("\n=== %d stops ===\n", len(j.Stops))
	for i, s := range j.Stops {
		fmt.Printf("%3d %s .. %s (%.0f min) displacement %.2f km, %d points, near [%.4f, %.4f]\n",
			i+1, s.Start.Format("15:04:05"), s.End.Format("15:04:05"),
			s.End.Sub(s.Start).Minutes(), s.DisplacementKm, s.Points, s.Loc.Lat, s.Loc.Lon)
	}
	return nil
}

// runRoute is the batch routing step (phase 3 BRIEF §1.2, §3G): enumerate
// journeys, collect their unknown gaps, consult the cache, ask the router
// only what the cache cannot answer, persist every data answer, record the
// run. The serve binary never routes — this command is the one place the
// product touches a routing service, and with the default null router it
// touches nothing at all.
func runRoute(args []string) error {
	fs := flag.NewFlagSet("route", flag.ExitOnError)
	db := dbFlag(fs)
	routerName := fs.String("router", envOr("ROADBOOK_ROUTER", "none"),
		"router: none | osrm (default $ROADBOOK_ROUTER)")
	routerURL := fs.String("router-url", envOr("ROADBOOK_ROUTER_URL", route.PublicOSRMURL),
		"OSRM base URL (default $ROADBOOK_ROUTER_URL, else the public demo)")
	profile := fs.String("profile", api.RouteProfile, "routing profile")
	interval := fs.Duration("interval", time.Second,
		"minimum spacing between router requests — courtesy for the public demo; 0 against localhost")
	dataset := fs.String("dataset", "",
		"OSM snapshot identity recorded with each answer (e.g. the extract filename)")
	all := fs.Bool("all", false, "route every candidate of the latest run, not just confirmed ones")
	candidateID := fs.Int64("candidate", 0, "route one candidate by id (any decision state)")
	refresh := fs.Bool("refresh", false, "re-ask the router even for cached pairs and replace the rows")
	if err := fs.Parse(args); err != nil {
		return err
	}

	var router route.Router
	switch *routerName {
	case "none":
		router = route.Null{}
	case "osrm":
		router = route.NewOSRM(*routerURL, *profile)
	default:
		return fmt.Errorf("unknown -router %q: use none or osrm", *routerName)
	}

	ctx := context.Background()
	s, err := openStore(ctx, *db)
	if err != nil {
		return err
	}
	defer s.Close()

	run, cands, err := s.LatestRun(ctx)
	if err != nil {
		return err
	}
	if run == nil {
		return fmt.Errorf("no detection run in the database — run 'roadbook detect' first")
	}

	// Scope: one candidate > all > confirmed (the default — confirmed
	// adventures are the product; routing dismissable rows multiplies load).
	decs, err := s.ListDecisions(ctx)
	if err != nil {
		return err
	}
	refs := make([]detect.SpanRef, len(cands))
	for i, c := range cands {
		refs[i] = detect.SpanRef{ID: c.ID, Start: c.SpanStart, End: c.SpanEnd, Dest: c.Dest}
	}
	anchors := make([]detect.Anchor, len(decs))
	for i, d := range decs {
		anchors[i] = detect.Anchor{ID: d.ID, Start: d.AnchorStart, End: d.AnchorEnd, Dest: d.AnchorDest, CreatedAt: d.CreatedAt}
	}
	matched := detect.Match(refs, anchors, detect.DefaultMatchParams())
	decByID := make(map[int64]store.DecisionRow, len(decs))
	for _, d := range decs {
		decByID[d.ID] = d
	}

	type target struct {
		cand store.CandidateRow
		name string
	}
	var targets []target
	for _, c := range cands {
		name := ""
		confirmed := false
		if did, ok := matched[c.ID]; ok {
			d := decByID[did]
			confirmed = d.Action == "confirmed"
			if d.Name != nil {
				name = *d.Name
			}
		}
		switch {
		case *candidateID != 0:
			if c.ID == *candidateID {
				targets = append(targets, target{c, name})
			}
		case *all || confirmed:
			targets = append(targets, target{c, name})
		}
	}
	if *candidateID != 0 && len(targets) == 0 {
		return fmt.Errorf("candidate %d is not in the latest run", *candidateID)
	}
	fmt.Printf("run %d: %d candidates, routing %d (scope: %s)\n",
		run.ID, len(cands), len(targets), scopeName(*candidateID, *all))

	// Collect every unknown-gap key across the scope, deduped in order; a
	// pair routed once serves every journey that contains it.
	params := journey.DefaultParams()
	perTarget := make([][]route.Key, len(targets))
	var keys []route.Key
	seen := map[route.Key]bool{}
	for i, tg := range targets {
		obs, err := s.LoadJourneyInputs(ctx, tg.cand.SpanStart, tg.cand.SpanEnd)
		if err != nil {
			return err
		}
		j := journey.Assemble(obs, tg.cand.SpanStart, tg.cand.SpanEnd, params)
		perTarget[i] = route.UnknownKeys(j, *profile)
		for _, k := range perTarget[i] {
			if !seen[k] {
				seen[k] = true
				keys = append(keys, k)
			}
		}
	}

	cached, err := s.LookupRoutes(ctx, keys)
	if err != nil {
		return err
	}

	var counts store.RouteRunCounts
	counts.GapsFound = len(keys)
	status := make(map[route.Key]string, len(keys))
	var toAsk []route.Key
	for _, k := range keys {
		if c, ok := cached[k]; ok && !*refresh {
			counts.CacheHits++
			status[k] = c.Status + " (cached)"
			continue
		}
		toAsk = append(toAsk, k)
	}

	if _, isNull := router.(route.Null); isNull && len(toAsk) > 0 {
		fmt.Printf("null router: %d uncached pair(s) stay unknown — run with -router osrm to fill them\n", len(toAsk))
	} else {
		for i, k := range toAsk {
			if i > 0 && *interval > 0 {
				time.Sleep(*interval)
			}
			r, err := router.Route(ctx, k.From(), k.To())
			switch {
			case err == nil:
				if err := s.SaveRoute(ctx, k, route.Cached{
					Status: route.StatusRouted, Points: r.Points,
					DistanceM: r.DistanceM, DurationS: r.DurationS,
				}, *routerName+" "+*routerURL, *dataset); err != nil {
					return err
				}
				counts.Routed++
				status[k] = "routed"
			case err == route.ErrNoRoute:
				if err := s.SaveRoute(ctx, k, route.Cached{Status: route.StatusNoRoute},
					*routerName+" "+*routerURL, *dataset); err != nil {
					return err
				}
				counts.NoRoute++
				status[k] = "no_route"
			default:
				// Operational, not a data answer: report, never cache.
				counts.Failures++
				status[k] = "failed"
				fmt.Printf("  FAILED (%.4f,%.4f)->(%.4f,%.4f): %v\n",
					k.From().Lat, k.From().Lon, k.To().Lat, k.To().Lon, err)
			}
		}
	}

	for i, tg := range targets {
		tally := map[string]int{}
		for _, k := range perTarget[i] {
			st := status[k]
			if st == "" {
				st = "unknown"
			}
			tally[st]++
		}
		label := fmt.Sprintf("candidate %d", tg.cand.ID)
		if tg.name != "" {
			label += fmt.Sprintf(" %q", tg.name)
		}
		fmt.Printf("  %s: %d unknown gap(s) — %s\n", label, len(perTarget[i]), tallyString(tally))
	}
	fmt.Printf("totals: %d unique pair(s) | %d cached | %d routed | %d no_route | %d failed\n",
		counts.GapsFound, counts.CacheHits, counts.Routed, counts.NoRoute, counts.Failures)

	runID, err := s.InsertRouteRun(ctx, *routerName+" "+*routerURL, *dataset, map[string]any{
		"profile": *profile, "interval": interval.String(), "refresh": *refresh,
		"scope": scopeName(*candidateID, *all), "journey_params": params,
	}, counts)
	if err != nil {
		return err
	}
	fmt.Printf("recorded route run %d\n", runID)
	return nil
}

func scopeName(candidateID int64, all bool) string {
	switch {
	case candidateID != 0:
		return fmt.Sprintf("candidate %d", candidateID)
	case all:
		return "all"
	default:
		return "confirmed"
	}
}

func tallyString(t map[string]int) string {
	if len(t) == 0 {
		return "none"
	}
	order := []string{"routed", "routed (cached)", "no_route", "no_route (cached)", "failed", "unknown"}
	out := ""
	for _, k := range order {
		if t[k] > 0 {
			if out != "" {
				out += ", "
			}
			out += fmt.Sprintf("%d %s", t[k], k)
		}
	}
	return out
}

func runProbe(args []string) error {
	fs := flag.NewFlagSet("probe", flag.ExitOnError)
	src := fs.String("src", "", "path to a Timeline export (required)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *src == "" {
		fs.Usage()
		return fmt.Errorf("-src is required")
	}
	data, err := os.ReadFile(*src)
	if err != nil {
		return err
	}
	paths, err := timeline.Probe(data)
	if err != nil {
		return err
	}
	for _, pc := range paths {
		fmt.Printf("%8d  %s\n", pc.Count, pc.Path)
	}
	return nil
}
