package api

import (
	"bytes"
	"context"
	"mime/multipart"
	"testing"
)

// TestUploadImportGuard pins the one-import-at-a-time rule (BRIEF §1.2)
// deterministically: with the lock held — as it is for the whole life of a
// running import — a second upload answers 409 before reading anything.
// Internal test: the lock is deliberately unexported, so holding it without
// racing a real import is only possible from inside the package.
func TestUploadImportGuard(t *testing.T) {
	s := &Server{}
	s.importMu.Lock()
	defer s.importMu.Unlock()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, _ := w.CreateFormFile("file", "Timeline.json")
	fw.Write([]byte(`{"semanticSegments":[]}`))
	w.Close()

	resp, err := s.UploadImport(context.Background(), UploadImportRequestObject{
		Body: multipart.NewReader(&buf, w.Boundary()),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := resp.(UploadImport409JSONResponse); !ok {
		t.Fatalf("got %T, want 409 while an import holds the lock", resp)
	}
}
