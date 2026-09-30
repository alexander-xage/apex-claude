package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"apexclaude/internal/version"
)

// setArtifactRoot points layout.ArtifactRoot() at root for the duration of
// the test — the same CLAUDE_PLUGIN_ROOT seam internal/doctor's tests use
// (internal/doctor/doctor_test.go:31). It works for a loose root too:
// ArtifactRoot() consults CLAUDE_PLUGIN_ROOT unconditionally, before any
// dev-vs-loose inference, so it is a clean override point regardless of
// which layout the fixture represents.
func setArtifactRoot(t *testing.T, root string) {
	t.Helper()
	t.Setenv("CLAUDE_PLUGIN_ROOT", root)
}

func binName() string {
	if runtime.GOOS == "windows" {
		return "apex.exe"
	}
	return "apex"
}

func writeFixtureFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// newLooseFixtureRoot builds a minimal loose-install root: no
// .claude-plugin dir (so layout.IsLooseInstall is true), the artifact
// surface apply touches, and a wired settings.json.
func newLooseFixtureRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFixtureFile(t, filepath.Join(root, "commands", "ax-plan.md"), "old plan\n")
	writeFixtureFile(t, filepath.Join(root, "agents", "ax-builder.md"), "old builder\n")
	writeFixtureFile(t, filepath.Join(root, "output-styles", "apex.md"), "old style\n")
	writeFixtureFile(t, filepath.Join(root, "skills", "ax-tdd", "SKILL.md"), "old skill\n")
	writeFixtureFile(t, filepath.Join(root, "bin", binName()), "old-binary-bytes\n")
	binPath := strings.ReplaceAll(filepath.Join(root, "bin", binName()), `\`, `\\`)
	settings := `{"hooks":{"SessionStart":[{"hooks":[{"command":"` + binPath + ` hooks session-start"}]}]}}`
	writeFixtureFile(t, filepath.Join(root, "settings.json"), settings)
	return root
}

func buildFixtureZip(t *testing.T, dir, name string, files map[string]string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	for entry, content := range files {
		w, err := zw.Create(entry)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func fixtureZipSHA256(t *testing.T, path string) string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// newFixtureUpdateServer serves a crafted bundle zip + SHA256SUMS at
// <base>/<tag>/<asset> and <base>/<tag>/SHA256SUMS, and — when latestTag is
// non-empty — a latest-release redirect at the server root.
func newFixtureUpdateServer(t *testing.T, tag, asset, zipPath, sums, latestTag string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/"+tag+"/"+asset, func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, zipPath)
	})
	mux.HandleFunc("/"+tag+"/SHA256SUMS", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(sums))
	})
	if latestTag != "" {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Location", "https://github.com/FNGApex/apex-claude/releases/tag/"+latestTag)
			w.WriteHeader(http.StatusFound)
		})
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func assetNameForTest() string {
	return "apex-claude-" + runtime.GOOS + "-" + runtime.GOARCH + ".zip"
}

// --- guard -------------------------------------------------------------

func TestRunUpdateDevLayoutGuardExitsTwo(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".claude-plugin"), 0o755); err != nil {
		t.Fatal(err)
	}
	setArtifactRoot(t, root)

	code, errOut := captureStderr(t, func() int { return runUpdate(nil) })
	if code != 2 {
		t.Fatalf("want exit 2 on dev layout, got %d", code)
	}
	if !strings.Contains(errOut, "dev layout") {
		t.Errorf("expected a dev-layout diagnostic, got %q", errOut)
	}
}

// --- apply happy path ----------------------------------------------------

func TestRunUpdateApplyHappyPathExitZero(t *testing.T) {
	isolateCache(t)
	root := newLooseFixtureRoot(t)
	setArtifactRoot(t, root)

	dlDir := t.TempDir()
	tag := "v1.2.3"
	asset := assetNameForTest()
	zipPath := buildFixtureZip(t, dlDir, "bundle.zip", map[string]string{
		"commands/ax-plan.md":    "new plan\n",
		"agents/ax-builder.md":   "new builder\n",
		"output-styles/apex.md":  "new style\n",
		"skills/ax-tdd/SKILL.md": "new skill\n",
		binName():                "new-binary-bytes\n",
	})
	sums := fixtureZipSHA256(t, zipPath) + "  " + asset + "\n"
	srv := newFixtureUpdateServer(t, tag, asset, zipPath, sums, "")
	t.Setenv("APEX_UPDATE_BASE_URL", srv.URL)

	code, out := captureStdout(t, func() int { return runUpdate([]string{"--to", tag}) })
	if code != 0 {
		t.Fatalf("want exit 0, got %d (out=%q)", code, out)
	}
	if !strings.Contains(out, "updated") || !strings.Contains(out, tag) {
		t.Errorf("expected an 'updated ... -> v1.2.3' message, got %q", out)
	}

	got, err := os.ReadFile(filepath.Join(root, "commands", "ax-plan.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new plan\n" {
		t.Errorf("artifact not replaced: got %q", got)
	}
}

func TestRunUpdateApplyCorruptedHashExitsOne(t *testing.T) {
	isolateCache(t)
	root := newLooseFixtureRoot(t)
	setArtifactRoot(t, root)

	dlDir := t.TempDir()
	tag := "v1.2.3"
	asset := assetNameForTest()
	zipPath := buildFixtureZip(t, dlDir, "bundle.zip", map[string]string{
		"commands/ax-plan.md": "new plan\n",
	})
	sums := "0000000000000000000000000000000000000000000000000000000000000000  " + asset + "\n"
	srv := newFixtureUpdateServer(t, tag, asset, zipPath, sums, "")
	t.Setenv("APEX_UPDATE_BASE_URL", srv.URL)

	code, errOut := captureStderr(t, func() int { return runUpdate([]string{"--to", tag}) })
	if code != 1 {
		t.Fatalf("want exit 1, got %d (err=%q)", code, errOut)
	}

	got, err := os.ReadFile(filepath.Join(root, "commands", "ax-plan.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "old plan\n" {
		t.Errorf("root should be untouched on verify failure, got %q", got)
	}
}

func TestRunUpdateApplySameVersionNoOpExitsZero(t *testing.T) {
	isolateCache(t)
	root := newLooseFixtureRoot(t)
	setArtifactRoot(t, root)

	cur := "v" + version.Version
	srv := newFixtureUpdateServer(t, cur, assetNameForTest(), "", "", cur)
	t.Setenv("APEX_UPDATE_BASE_URL", srv.URL)

	code, out := captureStdout(t, func() int { return runUpdate(nil) })
	if code != 0 {
		t.Fatalf("want exit 0, got %d (out=%q)", code, out)
	}
	if !strings.Contains(out, "up to date") {
		t.Errorf("expected an up-to-date message, got %q", out)
	}

	got, err := os.ReadFile(filepath.Join(root, "commands", "ax-plan.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "old plan\n" {
		t.Errorf("root should be untouched on a no-op, got %q", got)
	}
}

func TestRunUpdateApplyRejectsMalformedTo(t *testing.T) {
	code := runUpdate([]string{"--to", "../../evil"})
	if code != 2 {
		t.Fatalf("want usage exit 2 for malformed --to, got %d", code)
	}
}

func TestRunUpdateApplyRejectsUnknownArgs(t *testing.T) {
	code := runUpdate([]string{"--to", "v1.2.3", "stray"})
	if code != 2 {
		t.Fatalf("want usage exit 2 for stray argument, got %d", code)
	}
}
