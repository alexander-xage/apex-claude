package followups

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAddListRenderClose(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 6, 17, 0, 0, 0, 0, time.UTC)

	fid, err := Add(root, "drop the bash guard", "finding", "risk", "review", "body", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Add(root, "wiki subsystem", "plan", "", "design", "deferred", now); err != nil {
		t.Fatal(err)
	}

	list, _ := List(root)
	if len(list) != 2 {
		t.Fatalf("List len=%d, want 2", len(list))
	}

	idx, err := os.ReadFile(filepath.Join(root, ".claude", "project", "followups", "INDEX.md"))
	if err != nil {
		t.Fatalf("INDEX.md missing: %v", err)
	}
	body := string(idx)
	if !strings.Contains(body, "📋 plans") || !strings.Contains(body, "wiki subsystem") {
		t.Errorf("INDEX missing plans section:\n%s", body)
	}
	if !strings.Contains(body, "drop the bash guard") {
		t.Errorf("INDEX missing finding:\n%s", body)
	}

	if err := Close(root, fid, "done"); err != nil {
		t.Fatal(err)
	}
	if list, _ := List(root); len(list) != 1 {
		t.Fatalf("after close len=%d, want 1", len(list))
	}
	closed, err := os.ReadFile(filepath.Join(root, ".claude", "project", "followups", "CLOSED.md"))
	if err != nil || !strings.Contains(string(closed), "drop the bash guard") {
		t.Errorf("CLOSED.md missing entry: %v\n%s", err, closed)
	}
}

// --- id permanence: a closed id must never be reissued ---

func TestNextIDSkipsClosedIDs(t *testing.T) {
	root := t.TempDir()
	now := time.Now()

	for _, title := range []string{"first", "second", "third"} {
		if _, err := Add(root, title, "finding", "minor", "test", "", now); err != nil {
			t.Fatal(err)
		}
	}
	if err := Close(root, "003", "done"); err != nil {
		t.Fatal(err)
	}

	// 003 is retired. The next add must not reuse it, or CLOSED.md would
	// attribute "third (done)" to a different, open entry.
	id, err := Add(root, "brand new", "finding", "blocker", "test", "", now)
	if err != nil {
		t.Fatal(err)
	}
	if id == "003" {
		t.Fatal("reissued closed id 003")
	}
	if id != "004" {
		t.Errorf("next id = %q, want 004", id)
	}
}

// Closing the highest ids in a row must keep advancing, not walk backwards.
func TestNextIDAfterClosingEveryEntry(t *testing.T) {
	root := t.TempDir()
	now := time.Now()

	for i := 0; i < 3; i++ {
		if _, err := Add(root, "x", "finding", "nit", "test", "", now); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []string{"001", "002", "003"} {
		if err := Close(root, id, "done"); err != nil {
			t.Fatal(err)
		}
	}
	if list, _ := List(root); len(list) != 0 {
		t.Fatalf("expected an empty ledger, got %d entries", len(list))
	}

	id, err := Add(root, "after the purge", "finding", "nit", "test", "", now)
	if err != nil {
		t.Fatal(err)
	}
	if id != "004" {
		t.Errorf("next id after closing everything = %q, want 004", id)
	}
}

// The live high-water mark still wins when it is above the ledger's.
func TestNextIDUsesLiveHighWaterMark(t *testing.T) {
	root := t.TempDir()
	now := time.Now()

	for i := 0; i < 5; i++ {
		if _, err := Add(root, "x", "finding", "nit", "test", "", now); err != nil {
			t.Fatal(err)
		}
	}
	if err := Close(root, "002", "done"); err != nil { // a gap in the middle
		t.Fatal(err)
	}

	id, err := Add(root, "next", "finding", "nit", "test", "", now)
	if err != nil {
		t.Fatal(err)
	}
	if id != "006" {
		t.Errorf("next id = %q, want 006 (005 is still live)", id)
	}
}

// A CLOSED.md that predates or outlives the entry files still anchors the id.
func TestNextIDReadsLedgerWithNoLiveEntries(t *testing.T) {
	root := t.TempDir()
	d := filepath.Join(root, ".claude", "project", "followups")
	if err := os.MkdirAll(d, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d, closedFile),
		[]byte("- 007 — old thing (done)\n- 012 — newer thing (done)\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	id, err := Add(root, "fresh", "finding", "nit", "test", "", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if id != "013" {
		t.Errorf("next id = %q, want 013", id)
	}
}
