package signals

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanShowStale(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}

	p, err := Scan(root)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("scan file missing: %v", err)
	}

	body, err := Show(root)
	if err != nil {
		t.Fatalf("show: %v", err)
	}
	if !strings.Contains(body, "Go") {
		t.Errorf("expected Go in signals, got:\n%s", body)
	}
	if !strings.Contains(body, "src/") {
		t.Errorf("expected src/ dir in signals, got:\n%s", body)
	}

	if code, _ := Stale(root); code != 0 {
		t.Errorf("fresh after scan: got code %d", code)
	}

	// changing a manifest invalidates the fingerprint
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module x\nrequire y v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, _ := Stale(root); code != 1 {
		t.Errorf("expected stale(1) after manifest change, got %d", code)
	}
}

func TestStaleMissing(t *testing.T) {
	if code, _ := Stale(t.TempDir()); code != 1 {
		t.Errorf("missing signals should be stale(1), got %d", code)
	}
}
