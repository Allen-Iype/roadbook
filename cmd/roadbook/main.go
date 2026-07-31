// Command roadbook is the pipeline entry point: one binary, subcommands per
// stage. Phase 1 checkpoint 1 ships detect and probe; import (Postgres) and
// serve (HTTP API) arrive with checkpoint 2.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"roadbook/internal/detect"
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
	case "probe":
		err = runProbe(os.Args[2:])
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
  roadbook detect -src <timeline export.json> [threshold flags] [-json out.json]
  roadbook probe  -src <timeline export.json>

'detect' parses a Google Timeline export and prints ranked adventure candidates.
'probe' reports every JSON key path in an export and its frequency; diff two
probes to spot unannounced schema changes.`)
}

func runDetect(args []string) error {
	fs := flag.NewFlagSet("detect", flag.ExitOnError)
	src := fs.String("src", "", "path to a Timeline export (required)")
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
	if *src == "" {
		fs.Usage()
		return fmt.Errorf("-src is required")
	}

	data, err := os.ReadFile(*src)
	if err != nil {
		return err
	}
	obs, st, err := timeline.Parse(data)
	if err != nil {
		return err
	}
	fmt.Printf("parsed %s: %d visits, %d activities, %d path points",
		filepath.Base(*src), st.Visits, st.Activities, st.Points)
	if st.Skipped > 0 {
		fmt.Printf(" (%d segments skipped)", st.Skipped)
	}
	fmt.Println()

	res := detect.Run(obs, p)

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
