package layout

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLooksLikeArtifactRoot(t *testing.T) {
	empty := t.TempDir()
	if LooksLikeArtifactRoot(empty) {
		t.Error("empty dir must not look like an artifact root")
	}
	loose := t.TempDir()
	os.MkdirAll(filepath.Join(loose, "commands"), 0o755)
	if !LooksLikeArtifactRoot(loose) {
		t.Error("dir with commands/ should look like an artifact root (loose layout)")
	}
	dev := t.TempDir()
	os.MkdirAll(filepath.Join(dev, ".claude-plugin"), 0o755)
	if !LooksLikeArtifactRoot(dev) {
		t.Error("dir with .claude-plugin/ should look like an artifact root (dev layout)")
	}
}

// Windows PowerShell 5.1's `Set-Content -Encoding UTF8` prefixes a BOM, and
// encoding/json rejects a leading BOM as invalid JSON. A settings.json written
// by an older install.ps1 must still read as wired, or doctor fails every
// Windows loose install and `apex update` warns the hooks are gone.
func TestApexHooksWiredToleratesUTF8BOM(t *testing.T) {
	root := t.TempDir()
	body := `{"hooks":{"SessionStart":[{"hooks":[{"command":"C:/x/bin/apex.exe hooks session-start"}]}]}}`
	if err := os.WriteFile(filepath.Join(root, "settings.json"), append([]byte("\xEF\xBB\xBF"), body...), 0o644); err != nil {
		t.Fatal(err)
	}
	if !ApexHooksWired(root) {
		t.Error("BOM-prefixed settings.json with an apex hook must read as wired")
	}
}

func TestApexHooksWiredPlain(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "settings.json"),
		[]byte(`{"hooks":{"SessionStart":[{"hooks":[{"command":"/x/apex hooks session-start"}]}]}}`), 0o644)
	if !ApexHooksWired(root) {
		t.Error("plain settings.json with an apex hook must read as wired")
	}
	os.WriteFile(filepath.Join(root, "settings.json"), []byte(`{"hooks":{}}`), 0o644)
	if ApexHooksWired(root) {
		t.Error("no apex hook must read as unwired")
	}
}
