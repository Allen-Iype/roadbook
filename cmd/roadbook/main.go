// Command roadbook is the pipeline entry point: one binary, subcommands per
// stage. Phase 1 checkpoint 1 ships detect and probe; import (Postgres) and
// serve (HTTP API) arrive with checkpoint 2.
package main

import (
	"context"
	"encoding/json"
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
	"roadbook/internal/store"
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
  roadbook countries [-src <admin-0 geojson[.gz]>] [-db url]
  roadbook detect  (-src <export.json> | -db url) [threshold flags] [-json out.json]
  roadbook journey -src <export.json> -from <RFC3339> -to <RFC3339> [threshold flags]
  roadbook serve   [-db url] [-addr :8080]
  roadbook probe   -src <timeline export.json>

'migrate' applies embedded schema migrations.
'import' parses an export and stores observations idempotently.
'countries' loads country polygons for point-in-polygon attribution: the
bundled Natural Earth 1:110m set by default, or a higher-resolution admin-0
file from disk via -src. Replaces the table wholesale; never fetches.
'detect' finds adventure candidates: from a file (prints only) or from the
database (persists a run and its candidates, then prints).
'journey' assembles one window of observations into observed and gap legs and
prints the reconstruction — the command behind every journey figure.
'serve' runs the HTTP API.
'probe' reports every JSON key path in an export and its frequency; diff two
probes to spot unannounced schema changes.

-db defaults to $DATABASE_URL.`)
}

// dbFlag wires -db with $DATABASE_URL as the default. Configuration is via
// environment (docs/PLAN.md phase 5); the flag exists for one-off overrides.
func dbFlag(fs *flag.FlagSet) *string {
	return fs.String("db", os.Getenv("DATABASE_URL"), "Postgres URL (default $DATABASE_URL)")
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

	f, err := os.Open(*src)
	if err != nil {
		return err
	}
	obs, st, err := timeline.Parse(f)
	f.Close()
	if err != nil {
		return fmt.Errorf("cannot import %s: %w", filepath.Base(*src), err)
	}
	fmt.Printf("parsed %s: %d visits, %d activities, %d path points, %d raw positions (%d skipped)\n",
		filepath.Base(*src), st.Visits, st.Activities, st.Points, st.RawPositions, st.Skipped)

	winStart, err := parseDateFlag(*from)
	if err != nil {
		return fmt.Errorf("-from: %w", err)
	}
	winEnd, err := parseDateFlag(*to)
	if err != nil {
		return fmt.Errorf("-to: %w", err)
	}
	if winStart != nil || winEnd != nil {
		obs = filterWindow(obs, winStart, winEnd)
		fmt.Printf("window filter kept %d visits, %d activities, %d path points, %d raw positions\n",
			len(obs.Visits), len(obs.Activities), len(obs.Points), len(obs.RawPositions))
	}

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
	res, err := s.ImportObservations(ctx, lbl, winStart, winEnd, obs, st.Skipped)
	if err != nil {
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
	if err := fs.Parse(args); err != nil {
		return err
	}
	ctx := context.Background()
	s, err := openStore(ctx, *db)
	if err != nil {
		return err
	}
	defer s.Close()

	srv := &api.Server{Store: s, MatchParams: detect.DefaultMatchParams()}
	handler := api.HandlerFromMux(api.NewStrictHandler(srv, nil), http.NewServeMux())
	fmt.Printf("roadbook API listening on %s\n", *addr)
	return http.ListenAndServe(*addr, handler)
}

func runDetect(args []string) error {
	fs := flag.NewFlagSet("detect", flag.ExitOnError)
	src := fs.String("src", "", "path to a Timeline export (file mode: print only)")
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
	if err := fs.Parse(args); err != nil {
		return err
	}
	if (*src == "") == !*useDB {
		fs.Usage()
		return fmt.Errorf("exactly one of -src (file mode) or -from-db (database mode) is required")
	}

	var obs domain.Observations
	var s *store.Store
	ctx := context.Background()
	if *useDB {
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
	fmt.Printf("%3s %-11s %5s %8s %6s %5s %4s %-5s %s\n",
		"#", "start", "days", "dest_km", "track", "stops", "rpt", "trunc", "dest")
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
		fmt.Printf("%3d %-11s %5.1f %8d %6d %5d %4d %-5s [%.4f, %.4f]\n",
			i+1, c.Start.Format("2006-01-02"), c.Days, c.DestKm, c.TrackKm,
			c.Stops, c.Repeat, trunc, c.Dest.Lat, c.Dest.Lon)
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
	src := fs.String("src", "", "path to a Timeline export (required)")
	from := fs.String("from", "", "window start, RFC3339 (required)")
	to := fs.String("to", "", "window end, RFC3339 (required)")
	p := journey.DefaultParams()
	fs.Float64Var(&p.GapThresholdMinutes, "gap-min", p.GapThresholdMinutes,
		"silence longer than this becomes a gap leg, minutes")
	fs.Float64Var(&p.ThinSpacingSeconds, "thin-sec", p.ThinSpacingSeconds,
		"minimum spacing between kept points, seconds")
	fs.Float64Var(&p.MinStopDwellSeconds, "stop-sec", p.MinStopDwellSeconds,
		"minimum activity pause reported as a stop, seconds")
	fs.Float64Var(&p.MaxAccuracyM, "max-acc-m", p.MaxAccuracyM,
		"exclude raw positions with worse reported accuracy, metres (0 = off)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *src == "" || *from == "" || *to == "" {
		fs.Usage()
		return fmt.Errorf("-src, -from and -to are required")
	}
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

	j := journey.Assemble(obs, winStart, winEnd, p)

	fmt.Printf("\nwindow %s .. %s (%.1f h)\n",
		j.WindowStart.Format(time.RFC3339), j.WindowEnd.Format(time.RFC3339),
		j.WindowEnd.Sub(j.WindowStart).Hours())
	fmt.Printf("points: %d trace + %d raw in window -> %d kept (%d trace + %d raw)\n",
		j.TracePointsInWindow, j.RawPointsInWindow, j.MergedPoints(),
		j.TracePointsKept, j.RawPointsKept)
	obsPct, infPct := 0.0, 0.0
	if j.TotalKm > 0 {
		obsPct = j.ObservedKm / j.TotalKm * 100
		infPct = j.InferredKm / j.TotalKm * 100
	}
	fmt.Printf("distance %.1f km: observed %.1f (%.1f%%) + inferred %.1f (%.1f%%)\n",
		j.TotalKm, j.ObservedKm, obsPct, j.InferredKm, infPct)
	if j.GoogleDistanceKm > 0 {
		fmt.Printf("google's own figure %.1f km (%+.1f%%)\n",
			j.GoogleDistanceKm, (j.TotalKm-j.GoogleDistanceKm)/j.GoogleDistanceKm*100)
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
