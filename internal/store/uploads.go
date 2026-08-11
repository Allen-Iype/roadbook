package store

import (
	"context"
	"os"
	"path/filepath"
)

// UploadFiles is the uploads directory: retained Timeline exports, named by
// content hash (phase 7 BRIEF §3C). An export can be irreplaceable — its
// rawSignals window expires at the source — so the instance keeps what it
// was given; re-uploading identical bytes lands on the same name and stores
// nothing new. Same shape as PhotoFiles: the filesystem half of a stratum,
// reachable only through this process.
type UploadFiles struct {
	Dir string
}

// Init creates the directory. Called once at startup by serve.
func (f UploadFiles) Init() error {
	return os.MkdirAll(f.Dir, 0o755)
}

// Path is the retained file's location for a content hash. Exports are
// stored as .json regardless of the uploaded name — the sniffer, not the
// filename, decided what the bytes are.
func (f UploadFiles) Path(contentHash string) string {
	return filepath.Join(f.Dir, contentHash+".json")
}

// CreateTemp opens a temp file in the uploads directory for a streaming
// upload. Same-directory placement makes Promote an atomic rename, never a
// cross-device copy. The caller must either Promote it or Remove it — an
// interrupted upload leaves nothing behind (BRIEF §1.2).
func (f UploadFiles) CreateTemp() (*os.File, error) {
	return os.CreateTemp(f.Dir, "upload-*.tmp")
}

// Promote renames a fully-written temp file to its content-hash name.
// If identical bytes are already retained, the temp is discarded and the
// existing file stands — hash collisions of different content are not a
// practical concern, and same-hash-same-content makes overwrite and skip
// equivalent; skipping preserves the original file's timestamps.
func (f UploadFiles) Promote(tmpPath, contentHash string) (retained string, err error) {
	dst := f.Path(contentHash)
	if _, statErr := os.Stat(dst); statErr == nil {
		return dst, os.Remove(tmpPath)
	}
	return dst, os.Rename(tmpPath, dst)
}

// Remove deletes a temp file after a failed or rejected upload. Already
// gone is success — the goal is absence.
func (f UploadFiles) Remove(tmpPath string) error {
	err := os.Remove(tmpPath)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// BeginUploadImport records an upload-path attempt, status 'running', with
// the retained file's content hash (migration 00009). Upload imports have
// no date window: the whole file imports, as the front door promises.
func (s *Store) BeginUploadImport(ctx context.Context, label, contentHash string) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `INSERT INTO imports (source_label, visits, activities, points, raw_positions, skipped, status, content_hash)
		VALUES ($1,0,0,0,0,0,'running',$2) RETURNING id`,
		label, contentHash).Scan(&id)
	return id, err
}

// GetImport is the polling read (BRIEF §1.2). Nil means no such import.
func (s *Store) GetImport(ctx context.Context, id int64) (*ImportRow, error) {
	rows, err := s.pool.Query(ctx, importSelect+` WHERE id = $1`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, rows.Err()
	}
	r, err := scanImport(rows)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// SweepRunningImports finalises every 'running' import as failed — the
// serve-startup answer to a crash mid-import (BRIEF §1.2): the goroutine
// died with the process, and a row that says running forever is a lie. The
// known edge is accepted and documented: a CLI import in flight at serve
// startup would be swept too; the CLI operator is watching a terminal.
func (s *Store) SweepRunningImports(ctx context.Context, errMsg string) (int64, error) {
	ct, err := s.pool.Exec(ctx,
		`UPDATE imports SET status = 'failed', error = $1 WHERE status = 'running'`, errMsg)
	if err != nil {
		return 0, err
	}
	return ct.RowsAffected(), nil
}

// SetImportDetectStatus records the auto-detect phase on the import row
// (BRIEF §3D): running before detection starts, completed/failed after. It
// never touches status — a detection failure must not mark the import
// failed.
func (s *Store) SetImportDetectStatus(ctx context.Context, importID int64, status string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE imports SET detect_status = $2 WHERE id = $1`, importID, status)
	return err
}
