package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"roadbook/internal/domain"
	"roadbook/internal/photo"
)

// runPhoto is the parser made visible: `roadbook photo -inspect FILE` prints
// exactly what extraction sees — the sniff verdict, each metadata reading
// with its source, and the resolved instant — so every future "why didn't my
// photo place?" is answerable from the terminal (BRIEF §2). It writes no
// files; output goes to the terminal only.
func runPhoto(args []string) error {
	fs := flag.NewFlagSet("photo", flag.ExitOnError)
	inspect := fs.String("inspect", "", "photo or sidecar file to inspect")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *inspect == "" {
		return errors.New("photo: -inspect FILE is required")
	}

	data, err := os.ReadFile(*inspect)
	if err != nil {
		return err
	}
	name := filepath.Base(*inspect)

	kind, rej := photo.Sniff(data)
	if rej != nil {
		fmt.Printf("%s: rejected (%s)\n  %s\n", name, rej.Kind, rej.Message)
		return nil
	}

	switch kind {
	case photo.KindSidecar:
		return inspectSidecar(name, data)
	case photo.KindJPEG:
		return inspectJPEG(*inspect, name, data)
	}
	return nil
}

func inspectSidecar(name string, data []byte) error {
	s, err := photo.ParseSidecar(data)
	if err != nil {
		fmt.Printf("%s: JSON, but not a usable sidecar\n  %v\n", name, err)
		return nil
	}
	fmt.Printf("%s: Takeout sidecar\n", name)
	fmt.Printf("  describes    %s\n", orAbsent(s.Title))
	if pair := photo.SidecarPairName(name); pair != "" {
		fmt.Printf("  pairs with   %s (by filename)\n", pair)
	}
	if s.TakenTime != nil {
		fmt.Printf("  taken        %s\n", s.TakenTime.Format(time.RFC3339))
	} else {
		fmt.Printf("  taken        absent\n")
	}
	fmt.Printf("  position     %s\n", posString(s.Pos, ""))
	return nil
}

func inspectJPEG(path, name string, data []byte) error {
	m := photo.ExtractEXIF(data)

	// A Takeout sidecar living beside the file joins in, exactly as an
	// upload pairing would.
	sidecarNote := ""
	if scPath, scData := adjacentSidecar(path); scData != nil {
		if s, err := photo.ParseSidecar(scData); err == nil {
			hadEXIFPos := m.Pos != nil
			m = photo.MergeSidecar(m, s)
			sidecarNote = filepath.Base(scPath)
			if hadEXIFPos && s.Pos != nil {
				sidecarNote += " (its position ignored — the EXIF reading wins)"
			}
		}
	}

	fmt.Printf("%s: JPEG\n", name)
	fmt.Printf("  position     %s\n", posString(m.Pos, string(m.PosSource)))
	if m.GPSTime != nil {
		fmt.Printf("  gps clock    %s (UTC by definition)\n", m.GPSTime.Format(time.RFC3339))
	} else {
		fmt.Printf("  gps clock    absent\n")
	}
	if m.Wall != nil {
		fmt.Printf("  wall clock   %04d-%02d-%02d %02d:%02d:%02d (no timezone — this is what EXIF stores)\n",
			m.Wall.Year, m.Wall.Month, m.Wall.Day, m.Wall.Hour, m.Wall.Min, m.Wall.Sec)
	} else {
		fmt.Printf("  wall clock   absent\n")
	}
	if m.WallOffsetSec != nil {
		fmt.Printf("  wall offset  %s (OffsetTimeOriginal)\n", offsetString(*m.WallOffsetSec))
	}
	if m.SidecarTime != nil {
		fmt.Printf("  sidecar      %s, taken %s\n", sidecarNote, m.SidecarTime.Format(time.RFC3339))
	} else if sidecarNote != "" {
		fmt.Printf("  sidecar      %s (no capture time in it)\n", sidecarNote)
	}

	if rt, ok := photo.ResolveTime(m, 0, false); ok {
		fmt.Printf("  resolved     %s  display offset %s  source=%s\n",
			rt.Time.Format(time.RFC3339), offsetString(rt.OffsetSec), rt.Source)
	} else if m.Wall != nil {
		fmt.Printf("  resolved     wall clock only — resolves against the adventure's offset at placement (source=exif_local)\n")
	} else {
		fmt.Printf("  resolved     no usable capture time — the photo would be stored but unplaced\n")
	}

	if m.Orientation != 0 {
		fmt.Printf("  orientation  %d%s\n", m.Orientation, orientationNote(m.Orientation))
	}

	thumb, w, h, err := photo.Thumbnail(data, m.Orientation, photo.DefaultThumbMaxPx, photo.DefaultThumbQuality)
	if err != nil {
		fmt.Printf("  thumbnail    FAILED: %v\n", err)
		return nil
	}
	fmt.Printf("  thumbnail    %d×%d px, %.1f KB (max edge %d, metadata stripped)\n",
		w, h, float64(len(thumb))/1024, photo.DefaultThumbMaxPx)
	return nil
}

// adjacentSidecar looks for the Takeout sidecar naming patterns beside the
// image, including Takeout's arbitrary truncations of "supplemental-metadata".
func adjacentSidecar(imagePath string) (string, []byte) {
	if data, err := os.ReadFile(imagePath + ".json"); err == nil {
		return imagePath + ".json", data
	}
	matches, _ := filepath.Glob(imagePath + ".suppl*.json")
	for _, m := range matches {
		if data, err := os.ReadFile(m); err == nil {
			return m, data
		}
	}
	return "", nil
}

func posString(p *domain.LatLng, source string) string {
	if p == nil {
		return "absent"
	}
	s := fmt.Sprintf("%.6f, %.6f", p.Lat, p.Lon)
	if source != "" && source != string(photo.PosNone) {
		s += "  (" + source + ")"
	}
	return s
}

func offsetString(sec int) string {
	sign := "+"
	if sec < 0 {
		sign, sec = "-", -sec
	}
	return fmt.Sprintf("%s%02d:%02d", sign, sec/3600, sec%3600/60)
}

func orAbsent(s string) string {
	if strings.TrimSpace(s) == "" {
		return "absent"
	}
	return s
}

func orientationNote(o int) string {
	notes := map[int]string{
		2: " (mirrored)", 3: " (rotate 180°)", 4: " (mirrored vertically)",
		5: " (transposed)", 6: " (rotate 90° CW)", 7: " (transverse)", 8: " (rotate 270° CW)",
	}
	return notes[o]
}
