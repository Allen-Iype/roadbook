package api

import (
	"context"
	"testing"
)

// Deletion refuses while an import holds the lock: the goroutine would
// write rows after the deletion otherwise.
func TestDeleteMyDataGuard(t *testing.T) {
	s := &Server{}
	s.importMu.Lock()
	defer s.importMu.Unlock()
	resp, err := s.DeleteMyData(context.Background(), DeleteMyDataRequestObject{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := resp.(DeleteMyData409JSONResponse); !ok {
		t.Fatalf("got %T, want 409 while an import holds the lock", resp)
	}
}
