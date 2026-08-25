// Package photosource turns a batch of photograph files into domain
// observations: each photo that yields both a position and a capture instant
// becomes one timestamped fix (domain.RawPosition, Source = SourcePhoto), and
// nothing else escapes (CLAUDE.md invariant 4 — detection and journey
// assembly never learn that photos exist). It composes the phase-4 photo
// parser (EXIF, Takeout sidecars, the time-resolution ladder) into the
// batch shape ingestion needs, with a per-file verdict for every input —
// a camera roll is many files, and all-or-nothing reporting would hide
// which ones carried no evidence.
package photosource

import (
	"time"

	"roadbook/internal/domain"
	"roadbook/internal/photo"
)

// fixedZone names nothing: the offset is the entire content, and a fabricated
// zone name would imply knowledge the photo does not carry.
func fixedZone(offsetSec int) *time.Location { return time.FixedZone("", offsetSec) }

// File is one named input file. The caller owns transport (a directory walk,
// a multipart upload); this package is pure over bytes.
type File struct {
	Name string
	Data []byte
}

// Verdict says what one file contributed.
type Verdict string

const (
	VerdictFix             Verdict = "fix"              // position + instant extracted; one fix emitted
	VerdictNoPosition      Verdict = "no_position"      // no usable position in EXIF or sidecar
	VerdictNoTime          Verdict = "no_time"          // position but no resolvable instant
	VerdictSidecarPaired   Verdict = "sidecar_paired"   // consumed as metadata for its photo
	VerdictSidecarUnpaired Verdict = "sidecar_unpaired" // no matching photo in the batch
	VerdictUnsupported     Verdict = "unsupported"      // rejected by the photo sniffer
)

// FileResult is one file's verdict. Message carries the sniffer's actionable
// rejection text for VerdictUnsupported and is empty otherwise.
type FileResult struct {
	Name    string
	Verdict Verdict
	Message string
}

// Stats summarises one batch.
type Stats struct {
	Photos           int // image files recognised
	Fixes            int // fixes emitted (photos with position and instant)
	NoPosition       int
	NoTime           int
	SidecarsPaired   int
	SidecarsUnpaired int
	Unsupported      int
}

// ParseFiles processes one batch. Sidecars pair to photos by Takeout filename
// convention, falling back to the sidecar's own title (the same two-step the
// phase-4 upload endpoint uses); EXIF wins position, the sidecar fills
// absence, and capture time resolves down the phase-4 ladder with no local
// fallback offset — at import time there is no adventure whose offset could
// stand in, so a wall-clock-only photo is honestly unplaceable (VerdictNoTime)
// rather than guessed. Fixes carry their instant in the photo's own resolved
// UTC offset so downstream civil-date logic (home eras, day slicing) works in
// the photo's local calendar.
func ParseFiles(files []File) (domain.Observations, []FileResult, Stats) {
	type entry struct {
		kind    photo.Kind
		rej     *photo.UnsupportedError
		meta    photo.Meta
		sidecar *photo.Sidecar
	}
	entries := make([]entry, len(files))
	results := make([]FileResult, len(files))
	byName := map[string]int{} // image filename → index

	// Pass 1 — classify every file; index images by name.
	for i, f := range files {
		results[i].Name = f.Name
		kind, rej := photo.Sniff(f.Data)
		entries[i] = entry{kind: kind, rej: rej}
		switch {
		case rej != nil:
			results[i].Verdict = VerdictUnsupported
			results[i].Message = rej.Message
		case kind == photo.KindJPEG:
			entries[i].meta = photo.ExtractEXIF(f.Data)
			byName[f.Name] = i
		}
	}

	// Pass 2 — pair sidecars: filename convention first, title fallback.
	for i, f := range files {
		if entries[i].kind != photo.KindSidecar {
			continue
		}
		sc, err := photo.ParseSidecar(f.Data)
		if err != nil {
			results[i].Verdict = VerdictUnsupported
			results[i].Message = err.Error()
			continue
		}
		target := photo.SidecarPairName(f.Name)
		idx, ok := byName[target]
		if !ok && sc.Title != "" {
			idx, ok = byName[sc.Title]
		}
		if !ok {
			results[i].Verdict = VerdictSidecarUnpaired
			continue
		}
		entries[idx].sidecar = &sc
		results[i].Verdict = VerdictSidecarPaired
	}

	// Pass 3 — resolve each photo to a fix, or say why not.
	var obs domain.Observations
	var st Stats
	for i := range files {
		e := &entries[i]
		switch {
		case e.rej != nil:
			st.Unsupported++
			continue
		case e.kind == photo.KindSidecar:
			switch results[i].Verdict {
			case VerdictSidecarPaired:
				st.SidecarsPaired++
			case VerdictSidecarUnpaired:
				st.SidecarsUnpaired++
			default: // JSON that would not parse as a sidecar
				st.Unsupported++
			}
			continue
		}
		st.Photos++
		m := e.meta
		if e.sidecar != nil {
			m = photo.MergeSidecar(m, *e.sidecar)
		}
		if m.Pos == nil {
			results[i].Verdict = VerdictNoPosition
			st.NoPosition++
			continue
		}
		rt, ok := photo.ResolveTime(m, 0, false)
		if !ok {
			results[i].Verdict = VerdictNoTime
			st.NoTime++
			continue
		}
		results[i].Verdict = VerdictFix
		st.Fixes++
		obs.RawPositions = append(obs.RawPositions, domain.RawPosition{
			Time:   rt.Time.In(fixedZone(rt.OffsetSec)),
			Loc:    m.Pos,
			Source: domain.SourcePhoto,
		})
	}
	return obs, results, st
}
