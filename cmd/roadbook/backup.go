package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"roadbook/internal/backup"
	"roadbook/internal/store"
)

func photosDirFlag(fs *flag.FlagSet) *string {
	return fs.String("photos-dir", envOr("ROADBOOK_PHOTOS_DIR", "data/photos"),
		"thumbnail directory (default $ROADBOOK_PHOTOS_DIR)")
}

func runBackup(args []string) error {
	fs := flag.NewFlagSet("backup", flag.ExitOnError)
	db := dbFlag(fs)
	out := fs.String("out", "", "archive to write (required; refuses to overwrite)")
	photosDir := photosDirFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *out == "" {
		fs.Usage()
		return fmt.Errorf("-out is required")
	}

	// O_EXCL: a backup is the last copy of irreplaceable data — never
	// silently replace one. Pick a new name (date-stamped, say) instead.
	f, err := os.OpenFile(*out, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("%s already exists — refusing to overwrite a backup; choose a new name", *out)
		}
		return err
	}
	defer f.Close()

	ctx := context.Background()
	s, err := openStore(ctx, *db)
	if err != nil {
		return err
	}
	defer s.Close()

	man, warnings, err := backup.Write(ctx, s, store.PhotoFiles{Dir: *photosDir}, f, time.Now())
	if err != nil {
		// A partial archive must not look like a backup.
		f.Close()
		os.Remove(*out)
		return err
	}
	for _, w := range warnings {
		fmt.Fprintln(os.Stderr, "warning:", w)
	}
	fmt.Printf("backup %s: %d decisions, %d photos, %d thumbnails (schema %d)\n",
		*out, man.Decisions, man.Photos, man.Thumbnails, man.SchemaVersion)
	return nil
}

func runRestore(args []string) error {
	fs := flag.NewFlagSet("restore", flag.ExitOnError)
	db := dbFlag(fs)
	src := fs.String("src", "", "archive to restore (required)")
	photosDir := photosDirFlag(fs)
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
	defer f.Close()

	ctx := context.Background()
	s, err := openStore(ctx, *db)
	if err != nil {
		return err
	}
	defer s.Close()

	files := store.PhotoFiles{Dir: *photosDir}
	if err := files.Init(); err != nil {
		return err
	}

	rep, err := backup.Restore(ctx, s, files, f)
	if err != nil {
		return err
	}
	fmt.Printf("restored from %s (written %s, schema %d):\n", *src,
		rep.Manifest.CreatedAt.Format("2006-01-02 15:04"), rep.Manifest.SchemaVersion)
	fmt.Printf("  decisions: %d restored, %d already present\n", rep.DecisionsRestored, rep.DecisionsSkipped)
	fmt.Printf("  photos:    %d restored, %d already present\n", rep.PhotosRestored, rep.PhotosSkipped)
	fmt.Printf("  thumbnails: %d written, %d already present\n", rep.ThumbsWritten, rep.ThumbsExisting)
	for _, name := range rep.MissingThumb {
		fmt.Fprintf(os.Stderr, "warning: photo %q: no thumbnail in the archive or on disk — row not restored\n", name)
	}
	if rep.DecisionsRestored > 0 {
		fmt.Println("restored decisions attach to candidates at the next import + detection (they are orphans until then)")
	}
	return nil
}
