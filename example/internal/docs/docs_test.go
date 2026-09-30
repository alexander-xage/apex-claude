package docs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanStale(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "guide.md"), []byte("guide\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Scan(root); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if code, _ := Stale(root); code != 0 {
		t.Errorf("fresh after scan: got %d", code)
	}

	// adding a doc surface invalidates the fingerprint
	if err := os.WriteFile(filepath.Join(root, "docs", "new.md"), []byte("new\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, _ := Stale(root); code != 1 {
		t.Errorf("expected stale(1) after new doc, got %d", code)
	}
}

func TestStaleMissing(t *testing.T) {
	if code, _ := Stale(t.TempDir()); code != 1 {
		t.Errorf("missing cache should be stale(1), got %d", code)
	}
}
