// Package photosource turns a batch of photograph files into domain
// observations: each photo that yields both a position and a capture instant
// becomes one timestamped fix (domain.RawPosition, Source = SourcePhoto), and
// nothing else escapes (CLAUDE.md invariant 4 — detection and journey
// assembly never learn that photos exist). It composes the phase-4 photo
// parser (EXIF, Takeout sidecars, the time-resolution ladder) into the
// batch shape ingestion needs, with a per-file verdict for every input —
// a camera roll is many files, and all-or-nothing reporting would hide
// which ones carried no evidence.
//
// The API is two-phase for the server's sake: ScanFile consumes one file's
// bytes and keeps only extracted metadata, so an upload handler can process
// parts as they stream and discard each file before the next arrives (§4D:
// originals are never retained); Resolve then pairs sidecars and produces
// the fixes over the whole scanned batch, because pairing is inherently a
// batch operation. ParseFiles composes the two for callers that hold the
// batch anyway (the CLI, tests).
package photosource

import (
	"crypto/sha256"
	"fmt"
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
// rejection text for VerdictUnsupported and is empty otherwise. For
// VerdictFix the provenance fields are set — everything a photo record
// (BRIEF §4D) stores about this photo, so the caller never re-parses:
// ContentHash (sha256 hex of the file bytes; the record's identity and its
// thumbnail's filename), the emitted Fix, the position and time sources,
// the EXIF orientation, and whether the pixels are decodable here
// (JPEG yes, HEIC no — that flag is what "thumbnail where decodable" reads).
type FileResult struct {
	Name    string
	Verdict Verdict
	Message string

	ContentHash    string
	Fix            *domain.RawPosition
	PosSource      string
	TimeSource     string
	Orientation    int
	ThumbDecodable bool
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

// Scanned is one file's extracted metadata — everything Resolve needs, with
// the bytes gone. IsPhoto and the exported provenance fields let an upload
// handler decide (while it still holds the bytes) whether to cut a
// thumbnail; everything else feeds Resolve.
type Scanned struct {
	Name           string
	ContentHash    string // sha256 hex; photos only
	Orientation    int
	ThumbDecodable bool // JPEG: pixels decodable here; HEIC: not
	IsPhoto        bool

	meta    photo.Meta
	sidecar *photo.Sidecar
	rej     *photo.UnsupportedError
	kind    photo.Kind
}

// ScanFile consumes one file's bytes: sniff, extract, hash. The returned
// Scanned holds no reference to data.
func ScanFile(name string, data []byte) Scanned {
	s := Scanned{Name: name}
	kind, rej := photo.Sniff(data)
	s.kind = kind
	if rej != nil {
		s.rej = rej
		return s
	}
	switch kind {
	case photo.KindJPEG:
		s.meta = photo.ExtractEXIF(data)
		s.IsPhoto, s.ThumbDecodable = true, true
	case photo.KindHEIC:
		s.meta = photo.ExtractHEIF(data)
		s.IsPhoto = true
	case photo.KindSidecar:
		sc, err := photo.ParseSidecar(data)
		if err != nil {
			s.rej = &photo.UnsupportedError{Kind: "sidecar", Message: err.Error()}
			return s
		}
		s.sidecar = &sc
	}
	if s.IsPhoto {
		s.ContentHash = fmt.Sprintf("%x", sha256.Sum256(data))
		s.Orientation = s.meta.Orientation
	}
	return s
}

// Resolve pairs sidecars to photos over the whole scanned batch and produces
// the fixes. Sidecars pair by Takeout filename convention, falling back to
// the sidecar's own title (the same two-step the phase-4 upload endpoint
// uses); EXIF wins position, the sidecar fills absence, and capture time
// resolves down the phase-4 ladder with no local fallback offset — at import
// time there is no adventure whose offset could stand in, so a
// wall-clock-only photo is honestly unplaceable (VerdictNoTime) rather than
// guessed. Fixes carry their instant in the photo's own resolved UTC offset
// so downstream civil-date logic (home eras, day slicing) works in the
// photo's local calendar.
func Resolve(scans []Scanned) (domain.Observations, []FileResult, Stats) {
	results := make([]FileResult, len(scans))
	byName := map[string]int{}
	for i, sc := range scans {
		results[i].Name = sc.Name
		if sc.IsPhoto {
			byName[sc.Name] = i
		}
	}

	// Pair sidecars: filename convention first, title fallback.
	pairedSidecar := make([]*photo.Sidecar, len(scans))
	for i, sc := range scans {
		if sc.sidecar == nil {
			continue
		}
		target := photo.SidecarPairName(sc.Name)
		idx, ok := byName[target]
		if !ok && sc.sidecar.Title != "" {
			idx, ok = byName[sc.sidecar.Title]
		}
		if !ok {
			results[i].Verdict = VerdictSidecarUnpaired
			continue
		}
		pairedSidecar[idx] = sc.sidecar
		results[i].Verdict = VerdictSidecarPaired
	}

	var obs domain.Observations
	var st Stats
	for i, sc := range scans {
		switch {
		case sc.rej != nil:
			results[i].Verdict = VerdictUnsupported
			results[i].Message = sc.rej.Message
			st.Unsupported++
			continue
		case sc.sidecar != nil:
			switch results[i].Verdict {
			case VerdictSidecarPaired:
				st.SidecarsPaired++
			default:
				st.SidecarsUnpaired++
			}
			continue
		case !sc.IsPhoto:
			// Unreachable today: every sniffed kind is photo, sidecar, or
			// rejection. Kept as a visible verdict rather than a panic.
			results[i].Verdict = VerdictUnsupported
			results[i].Message = "unrecognised input"
			st.Unsupported++
			continue
		}
		st.Photos++
		m := sc.meta
		if pairedSidecar[i] != nil {
			m = photo.MergeSidecar(m, *pairedSidecar[i])
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
		fix := domain.RawPosition{
			Time:   rt.Time.In(fixedZone(rt.OffsetSec)),
			Loc:    m.Pos,
			Source: domain.SourcePhoto,
		}
		results[i].Verdict = VerdictFix
		results[i].ContentHash = sc.ContentHash
		results[i].Fix = &fix
		results[i].PosSource = string(m.PosSource)
		results[i].TimeSource = string(rt.Source)
		results[i].Orientation = sc.Orientation
		results[i].ThumbDecodable = sc.ThumbDecodable
		st.Fixes++
		obs.RawPositions = append(obs.RawPositions, fix)
	}
	return obs, results, st
}

// ParseFiles processes one whole batch: ScanFile per file, then Resolve.
func ParseFiles(files []File) (domain.Observations, []FileResult, Stats) {
	scans := make([]Scanned, len(files))
	for i, f := range files {
		scans[i] = ScanFile(f.Name, f.Data)
	}
	return Resolve(scans)
}
