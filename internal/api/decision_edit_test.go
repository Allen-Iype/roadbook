package api_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"roadbook/internal/store"
)

// Editing a kept trip's name and notes (phase 17, issue #3) through the
// generated mux: the edit lands, the owner's list carries the notes, the
// shared view never does, and only a kept trip is editable.

func patchJSON(t *testing.T, url string, body any) (int, []byte) {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, out
}

func TestEditDecision(t *testing.T) {
	ts, s, photosDir := newShareServer(t, nil)
	c := http.DefaultClient
	confirmedID, undecidedID, _, _, _ := seedShareable(t, s, photosDir, store.SelfUser)
	url := func(id int64) string { return ts.URL + "/candidates/" + itoa(id) + "/decision" }
	const notes = "Rain on day two.\n\n  The pass was closed; we went round."

	t.Run("name and notes land, trimmed, nothing else changes", func(t *testing.T) {
		code, body := patchJSON(t, url(confirmedID), map[string]any{"name": "  Hill country  ", "notes": notes + "  \n"})
		if code != 200 {
			t.Fatalf("status %d: %s", code, body)
		}
		var d struct {
			Action string  `json:"action"`
			Name   string  `json:"name"`
			Notes  *string `json:"notes"`
		}
		if err := json.Unmarshal(body, &d); err != nil {
			t.Fatal(err)
		}
		if d.Action != "confirmed" || d.Name != "Hill country" || d.Notes == nil || *d.Notes != notes {
			t.Errorf("got %+v", d)
		}
	})

	t.Run("the owner's list carries the notes", func(t *testing.T) {
		resp, body := do(t, c, http.MethodGet, ts.URL+"/candidates", nil)
		if resp.StatusCode != 200 || !strings.Contains(string(body), "The pass was closed") {
			t.Errorf("list status %d, notes absent", resp.StatusCode)
		}
	})

	t.Run("the shared view never carries them", func(t *testing.T) {
		resp, body := do(t, c, http.MethodPost, ts.URL+"/candidates/"+itoa(confirmedID)+"/shares", nil)
		if resp.StatusCode != 201 && resp.StatusCode != 200 {
			t.Fatalf("mint: %d %s", resp.StatusCode, body)
		}
		var created struct {
			Token string `json:"token"`
		}
		if err := json.Unmarshal(body, &created); err != nil {
			t.Fatal(err)
		}
		resp, body = do(t, c, http.MethodGet, ts.URL+"/shared/"+created.Token, nil)
		if resp.StatusCode != 200 {
			t.Fatalf("shared view: %d", resp.StatusCode)
		}
		if strings.Contains(string(body), "The pass was closed") || strings.Contains(string(body), `"notes"`) {
			t.Error("the shared view carries the owner's notes")
		}
		if !strings.Contains(string(body), "Hill country") {
			t.Error("the shared view lost the edited name")
		}
	})

	t.Run("blank notes clear them", func(t *testing.T) {
		code, body := patchJSON(t, url(confirmedID), map[string]any{"name": "Hill country", "notes": " \n "})
		if code != 200 || strings.Contains(string(body), `"notes"`) {
			t.Errorf("status %d, body %s — want notes absent", code, body)
		}
	})

	t.Run("limits are counted in characters, not bytes", func(t *testing.T) {
		// 120 two-byte characters (240 bytes) is within the limit.
		if code, body := patchJSON(t, url(confirmedID), map[string]any{"name": strings.Repeat("é", 120)}); code != 200 {
			t.Errorf("120 characters: %d %s", code, body)
		}
		if code, _ := patchJSON(t, url(confirmedID), map[string]any{"name": strings.Repeat("é", 121)}); code != 400 {
			t.Errorf("121 characters: %d — want 400", code)
		}
		if code, _ := patchJSON(t, url(confirmedID), map[string]any{"name": "x", "notes": strings.Repeat("ü", 4000)}); code != 200 {
			t.Errorf("4,000-character notes: %d — want 200", code)
		}
		if code, _ := patchJSON(t, url(confirmedID), map[string]any{"name": "x", "notes": strings.Repeat("ü", 4001)}); code != 400 {
			t.Errorf("4,001-character notes: %d — want 400", code)
		}
	})

	t.Run("a blank name is refused", func(t *testing.T) {
		if code, _ := patchJSON(t, url(confirmedID), map[string]any{"name": "   "}); code != 400 {
			t.Errorf("status %d — want 400", code)
		}
	})

	t.Run("only a kept trip is editable", func(t *testing.T) {
		if code, _ := patchJSON(t, url(undecidedID), map[string]any{"name": "Not kept"}); code != 404 {
			t.Errorf("undecided: %d — want 404", code)
		}
		if code, _ := patchJSON(t, url(999999), map[string]any{"name": "Nowhere"}); code != 404 {
			t.Errorf("unknown candidate: %d — want 404", code)
		}
	})

	t.Run("a re-decide keeps the notes", func(t *testing.T) {
		if code, _ := patchJSON(t, url(confirmedID), map[string]any{"name": "Hill country", "notes": notes}); code != 200 {
			t.Fatal("setting notes failed")
		}
		req, _ := http.NewRequest(http.MethodPost, url(confirmedID), strings.NewReader(`{"action":"confirmed","name":"Hills again"}`))
		req.Header.Set("Content-Type", "application/json")
		resp, err := c.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != 200 || !strings.Contains(string(body), "The pass was closed") {
			t.Errorf("re-decide: %d %s — want the notes kept", resp.StatusCode, body)
		}
	})

	t.Run("confirming with an over-long name is refused too", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, url(confirmedID), strings.NewReader(`{"action":"confirmed","name":"`+strings.Repeat("a", 121)+`"}`))
		req.Header.Set("Content-Type", "application/json")
		resp, err := c.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != 400 {
			t.Errorf("status %d — want 400", resp.StatusCode)
		}
	})
}
