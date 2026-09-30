package update

import (
	"archive/zip"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"apexclaude/internal/version"
)

// --- fixture helpers -------------------------------------------------------

// isolateApplyCache redirects os.UserCacheDir to a fresh temp dir so Apply's
// cache rewrite never touches the real machine's update-check.json.
func isolateApplyCache(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("LocalAppData", dir)   // windows
	t.Setenv("XDG_CACHE_HOME", dir) // linux
	t.Setenv("HOME", dir)           // darwin fallback
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// newFixtureRoot builds a temp loose-install root carrying the artifact
// surface Apply touches: commands/, agents/, output-styles/, a
// skills/ax-tdd dir (with a stale extra file to prove wholesale-replace),
// bin/apex(.exe), and a settings.json wiring an apex hook.
func newFixtureRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "commands", "ax-plan.md"), "old plan\n")
	mustWriteFile(t, filepath.Join(root, "commands", "my-command.md"), "unrelated command\n")
	mustWriteFile(t, filepath.Join(root, "commands", "ax-retired.md"), "command a later release dropped\n")
	mustWriteFile(t, filepath.Join(root, "agents", "ax-builder.md"), "old builder\n")
	mustWriteFile(t, filepath.Join(root, "output-styles", "apex.md"), "old style\n")
	mustWriteFile(t, filepath.Join(root, "skills", "ax-tdd", "SKILL.md"), "old skill\n")
	mustWriteFile(t, filepath.Join(root, "skills", "ax-tdd", "stale.txt"), "stale leftover\n")
	mustWriteFile(t, filepath.Join(root, "bin", binaryName()), "old-binary-bytes\n")
	binPath := strings.ReplaceAll(filepath.Join(root, "bin", binaryName()), `\`, `\\`)
	settings := `{"hooks":{"SessionStart":[{"hooks":[{"command":"` + binPath + ` hooks session-start"}]}]}}`
	mustWriteFile(t, filepath.Join(root, "settings.json"), settings)
	return root
}

// buildBundleZip zips files (archive-path -> content) into a new zip under
// dir/name and returns its path.
func buildBundleZip(t *testing.T, dir, name string, files map[string]string) string {
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

// newUpdateServer serves a crafted bundle zip + SHA256SUMS at
// <base>/<tag>/<asset> and <base>/<tag>/SHA256SUMS, and (optionally) a
// latest-release redirect to latestTag at the server root — the same
// APEX_UPDATE_BASE_URL seam covers both LatestTag and Apply's downloads.
func newUpdateServer(t *testing.T, tag, asset, zipPath, sums, latestTag string) *httptest.Server {
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

func sumsLine(t *testing.T, zipPath, asset string) string {
	t.Helper()
	sum, err := sha256File(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	return sum + "  " + asset + "\n"
}

// snapshot captures the content of every regular file under root, keyed by
// path relative to root, so a test can assert "nothing touched on disk".
func snapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		b, _ := os.ReadFile(path)
		out[rel] = string(b)
		return nil
	})
	return out
}

func assertSnapshotUnchanged(t *testing.T, root string, before map[string]string) {
	t.Helper()
	after := snapshot(t, root)
	if len(before) != len(after) {
		t.Fatalf("file count changed: before=%d after=%d", len(before), len(after))
	}
	for k, v := range before {
		if after[k] != v {
			t.Errorf("file %q changed: before=%q after=%q", k, v, after[k])
		}
	}
}

// --- Apply: happy path -----------------------------------------------------

func TestApplyHappyPathReplacesArtifactsAndBinary(t *testing.T) {
	isolateApplyCache(t)
	root := newFixtureRoot(t)
	dlDir := t.TempDir()

	tag := "v1.2.3"
	asset := assetName()
	files := map[string]string{
		"commands/ax-plan.md":    "new plan\n",
		"agents/ax-builder.md":   "new builder\n",
		"output-styles/apex.md":  "new style\n",
		"skills/ax-tdd/SKILL.md": "new skill\n",
		binaryName():             "new-binary-bytes\n",
	}
	// bin/<name> lives at the zip root per the bundle layout contract.
	zipPath := buildBundleZip(t, dlDir, "bundle.zip", files)
	sums := sumsLine(t, zipPath, asset)
	srv := newUpdateServer(t, tag, asset, zipPath, sums, "")
	t.Setenv("APEX_UPDATE_BASE_URL", srv.URL)

	res, err := Apply(root, tag)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	if res.UpToDate {
		t.Error("expected UpToDate=false for a real update")
	}
	if res.New != tag {
		t.Errorf("New = %q, want %q", res.New, tag)
	}
	if res.Old != "v"+version.Version {
		t.Errorf("Old = %q, want v%s", res.Old, version.Version)
	}

	// Artifacts overwritten.
	assertFileContent(t, filepath.Join(root, "commands", "ax-plan.md"), "new plan\n")
	assertFileContent(t, filepath.Join(root, "agents", "ax-builder.md"), "new builder\n")
	assertFileContent(t, filepath.Join(root, "output-styles", "apex.md"), "new style\n")
	// A user's own (non-ax-) command is left alone...
	assertFileContent(t, filepath.Join(root, "commands", "my-command.md"), "unrelated command\n")
	// ...but an Apex command the new release no longer ships is pruned.
	if _, err := os.Stat(filepath.Join(root, "commands", "ax-retired.md")); !os.IsNotExist(err) {
		t.Errorf("ax-retired.md should be pruned: the bundle no longer ships it (stat err = %v)", err)
	}

	// Skills wholesale-replaced: stale.txt gone, SKILL.md fresh.
	assertFileContent(t, filepath.Join(root, "skills", "ax-tdd", "SKILL.md"), "new skill\n")
	if _, err := os.Stat(filepath.Join(root, "skills", "ax-tdd", "stale.txt")); !os.IsNotExist(err) {
		t.Errorf("stale.txt should be gone after wholesale skill replace, stat err = %v", err)
	}

	// Binary replaced.
	assertFileContent(t, filepath.Join(root, "bin", binaryName()), "new-binary-bytes\n")
	if runtime.GOOS == "windows" {
		// The rename dance leaves .old behind on success — it still backs
		// the (simulated) running process; cleanup happens at the start of
		// the NEXT apply.
		assertFileContent(t, filepath.Join(root, "bin", "apex.exe.old"), "old-binary-bytes\n")
	} else {
		if _, err := os.Stat(filepath.Join(root, "bin", ".apex.new")); !os.IsNotExist(err) {
			t.Errorf(".apex.new staging file should not survive a successful swap, stat err = %v", err)
		}
	}

	// Cache rewritten as up-to-date.
	c := ReadCache(CachePath())
	if c.Latest != tag {
		t.Errorf("cache Latest = %q, want %q", c.Latest, tag)
	}
	if c.CheckedAt == "" {
		t.Error("cache CheckedAt should be stamped")
	}
}

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(got) != want {
		t.Errorf("%s = %q, want %q", path, got, want)
	}
}

// --- Apply: verify failures leave root untouched ---------------------------

func TestApplyCorruptedHashFailsAndLeavesRootUntouched(t *testing.T) {
	isolateApplyCache(t)
	root := newFixtureRoot(t)
	before := snapshot(t, root)
	dlDir := t.TempDir()

	tag := "v1.2.3"
	asset := assetName()
	zipPath := buildBundleZip(t, dlDir, "bundle.zip", map[string]string{
		"commands/ax-plan.md": "new plan\n",
	})
	// Deliberately wrong hash.
	sums := "0000000000000000000000000000000000000000000000000000000000000000  " + asset + "\n"
	srv := newUpdateServer(t, tag, asset, zipPath, sums, "")
	t.Setenv("APEX_UPDATE_BASE_URL", srv.URL)

	_, err := Apply(root, tag)
	if err == nil {
		t.Fatal("expected an error for a corrupted/mismatched checksum")
	}
	if !strings.Contains(err.Error(), "checksum") {
		t.Errorf("expected a checksum-mismatch error, got: %v", err)
	}
	assertSnapshotUnchanged(t, root, before)
}

func TestApplyMissingSHA256SUMSLineFails(t *testing.T) {
	isolateApplyCache(t)
	root := newFixtureRoot(t)
	before := snapshot(t, root)
	dlDir := t.TempDir()

	tag := "v1.2.3"
	asset := assetName()
	zipPath := buildBundleZip(t, dlDir, "bundle.zip", map[string]string{
		"commands/ax-plan.md": "new plan\n",
	})
	// SHA256SUMS present but names a different asset entirely.
	sums := "abc123  apex-claude-someother-arch.zip\n"
	srv := newUpdateServer(t, tag, asset, zipPath, sums, "")
	t.Setenv("APEX_UPDATE_BASE_URL", srv.URL)

	_, err := Apply(root, tag)
	if err == nil {
		t.Fatal("expected an error for a missing SHA256SUMS line")
	}
	assertSnapshotUnchanged(t, root, before)
}

// --- Apply: --to pin vs. no-op --------------------------------------------

func TestApplySameVersionNoOpWithoutTo(t *testing.T) {
	isolateApplyCache(t)
	root := newFixtureRoot(t)
	before := snapshot(t, root)

	cur := "v" + version.Version
	srv := newUpdateServer(t, cur, assetName(), "", "", cur) // only the "/" redirect matters here
	t.Setenv("APEX_UPDATE_BASE_URL", srv.URL)

	res, err := Apply(root, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.UpToDate {
		t.Error("expected UpToDate=true when latest == running version and no --to given")
	}
	if res.Old != cur || res.New != cur {
		t.Errorf("Old/New = %q/%q, want %q/%q", res.Old, res.New, cur, cur)
	}
	assertSnapshotUnchanged(t, root, before)
}

func TestApplyExplicitToPinAppliesEvenAtSameVersion(t *testing.T) {
	isolateApplyCache(t)
	root := newFixtureRoot(t)
	dlDir := t.TempDir()

	cur := "v" + version.Version
	asset := assetName()
	files := map[string]string{
		"commands/ax-plan.md":    "reinstalled plan\n",
		"agents/ax-builder.md":   "reinstalled builder\n",
		"output-styles/apex.md":  "reinstalled style\n",
		"skills/ax-tdd/SKILL.md": "reinstalled skill\n",
		binaryName():             "reinstalled-binary\n",
	}
	zipPath := buildBundleZip(t, dlDir, "bundle.zip", files)
	sums := sumsLine(t, zipPath, asset)
	srv := newUpdateServer(t, cur, asset, zipPath, sums, "")
	t.Setenv("APEX_UPDATE_BASE_URL", srv.URL)

	res, err := Apply(root, cur)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.UpToDate {
		t.Error("an explicit --to pin must not short-circuit as up-to-date, even at the same version")
	}
	assertFileContent(t, filepath.Join(root, "commands", "ax-plan.md"), "reinstalled plan\n")
}

// --- extract: zip-slip guard ------------------------------------------------

func TestExtractZipRejectsPathTraversalEntry(t *testing.T) {
	dlDir := t.TempDir()
	zipPath := buildBundleZip(t, dlDir, "evil.zip", map[string]string{
		"../evil.txt":         "escaped!\n",
		"commands/ax-plan.md": "fine\n",
	})
	destDir := t.TempDir()

	err := extractZip(zipPath, destDir)
	if err == nil {
		t.Fatal("expected an error for a zip-slip entry")
	}
	if !strings.Contains(err.Error(), "escapes extract root") {
		t.Errorf("expected an escapes-extract-root error, got: %v", err)
	}
	// Nothing should have escaped destDir's parent.
	if _, statErr := os.Stat(filepath.Join(filepath.Dir(destDir), "evil.txt")); !os.IsNotExist(statErr) {
		t.Error("zip-slip entry was written outside the extract root")
	}
}

func TestApplyZipSlipBundleFailsAndLeavesRootUntouched(t *testing.T) {
	isolateApplyCache(t)
	root := newFixtureRoot(t)
	before := snapshot(t, root)
	dlDir := t.TempDir()

	tag := "v1.2.3"
	asset := assetName()
	zipPath := buildBundleZip(t, dlDir, "bundle.zip", map[string]string{
		"../evil.txt":         "escaped!\n",
		"commands/ax-plan.md": "fine\n",
	})
	sums := sumsLine(t, zipPath, asset)
	srv := newUpdateServer(t, tag, asset, zipPath, sums, "")
	t.Setenv("APEX_UPDATE_BASE_URL", srv.URL)

	_, err := Apply(root, tag)
	if err == nil {
		t.Fatal("expected an error for a zip-slip bundle")
	}
	assertSnapshotUnchanged(t, root, before)
}

// --- Windows rename-dance specifics -----------------------------------------

func TestSwapBinaryWindowsRollbackOnWriteFailure(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows rename dance only exercised on windows")
	}
	root := t.TempDir()
	dst := filepath.Join(root, "bin", "apex.exe")
	mustWriteFile(t, dst, "original\n")

	srcDir := t.TempDir()
	src := filepath.Join(srcDir, "apex.exe")
	mustWriteFile(t, src, "new-binary\n")

	orig := writeBinary
	writeBinary = func(string, string, os.FileMode) error {
		return errors.New("simulated write failure")
	}
	t.Cleanup(func() { writeBinary = orig })

	if err := swapBinary(srcDir, root); err == nil {
		t.Fatal("expected an error from the simulated write failure")
	}

	assertFileContent(t, dst, "original\n")
	if _, err := os.Stat(dst + ".old"); !os.IsNotExist(err) {
		t.Errorf("rollback should remove the .old leftover, stat err = %v", err)
	}
}

func TestApplyRemovesStaleWindowsOldAtStart(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("apex.exe.old cleanup only applies on windows")
	}
	isolateApplyCache(t)
	root := newFixtureRoot(t)
	staleOld := filepath.Join(root, "bin", "apex.exe.old")
	mustWriteFile(t, staleOld, "leftover-from-a-prior-interrupted-update\n")

	dlDir := t.TempDir()
	tag := "v1.2.3"
	asset := assetName()
	zipPath := buildBundleZip(t, dlDir, "bundle.zip", map[string]string{
		"commands/ax-plan.md":    "new plan\n",
		"agents/ax-builder.md":   "new builder\n",
		"output-styles/apex.md":  "new style\n",
		"skills/ax-tdd/SKILL.md": "new skill\n",
		binaryName():             "new-binary-bytes\n",
	})
	sums := sumsLine(t, zipPath, asset)
	srv := newUpdateServer(t, tag, asset, zipPath, sums, "")
	t.Setenv("APEX_UPDATE_BASE_URL", srv.URL)

	if _, err := Apply(root, tag); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The stale leftover from BEFORE this run must have been cleaned at the
	// start; the .old present now belongs to THIS run's dance (the fixture's
	// original binary bytes), not the pre-seeded garbage.
	got, err := os.ReadFile(staleOld)
	if err != nil {
		t.Fatalf("read .old: %v", err)
	}
	if string(got) == "leftover-from-a-prior-interrupted-update\n" {
		t.Error("stale .old from a prior interrupted update was not cleaned at the start of Apply")
	}
	if string(got) != "old-binary-bytes\n" {
		t.Errorf(".old = %q, want the fixture's original binary bytes", got)
	}
}

// The stale-.old sweep must run before the up-to-date early return — an
// already-current install is exactly when a leftover .old lingers longest.
func TestApplyRemovesStaleOldWhenUpToDate(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-only rename-dance artifact")
	}
	root := t.TempDir()
	old := filepath.Join(root, "bin", "apex.exe.old")
	if err := os.MkdirAll(filepath.Dir(old), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(old, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "https://github.com/x/y/releases/tag/v"+version.Version) // latest == current
		w.WriteHeader(http.StatusFound)
	}))
	defer srv.Close()
	t.Setenv("APEX_UPDATE_BASE_URL", srv.URL)

	res, err := Apply(root, "")
	if err != nil || !res.UpToDate {
		t.Fatalf("want up-to-date no-op, got res=%+v err=%v", res, err)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Error("stale apex.exe.old must be swept even on an up-to-date run")
	}
}

// A malformed --to must be rejected before any network or disk activity —
// the raw string would otherwise be concatenated into the download URL.
func TestApplyRejectsMalformedToTag(t *testing.T) {
	root := t.TempDir()
	t.Setenv("APEX_UPDATE_BASE_URL", "http://127.0.0.1:1") // unreachable: proves no fetch happens
	for _, bad := range []string{"../../evil", "v1.2", "1.2.3", "v1.2.3/../x", "latest"} {
		_, err := Apply(root, bad)
		if err == nil || !strings.Contains(err.Error(), "invalid tag") {
			t.Errorf("Apply(root, %q) must fail tag validation (before any fetch), got %v", bad, err)
		}
	}
}

// A failed rename must not strand the staged .apex.new next to the binary.
func TestSwapBinaryUnixCleansStagedFileOnRenameFailure(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src-binary")
	if err := os.WriteFile(src, []byte("new"), 0o755); err != nil {
		t.Fatal(err)
	}
	// dst is a non-empty directory: os.Rename(file, non-empty dir) fails on
	// every platform, forcing the post-stage failure path.
	dst := filepath.Join(dir, "bin", "apex")
	if err := os.MkdirAll(filepath.Join(dst, "occupied"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := swapBinaryUnix(src, dst); err == nil {
		t.Fatal("want rename failure")
	}
	if _, err := os.Stat(filepath.Join(dir, "bin", ".apex.new")); !os.IsNotExist(err) {
		t.Error("staged .apex.new must be removed when the rename fails")
	}
}

// A release that drops a command, agent or skill must remove it from the
// install on update. Overwrite-only semantics left every cut ax-* artifact on
// disk forever, still listed in Claude Code's slash menu. ax-* is Apex's
// namespace (uninstall already deletes the whole prefix); anything else in
// those directories belongs to the user and must survive.
func TestApplyArtifactsPrunesDroppedApexArtifacts(t *testing.T) {
	extract, root := t.TempDir(), t.TempDir()
	write := func(base, rel, body string) {
		t.Helper()
		p := filepath.Join(base, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// New bundle: one command, one agent, one skill.
	write(extract, "commands/ax-keep.md", "new")
	write(extract, "agents/ax-keep.md", "new")
	write(extract, "skills/ax-keep/SKILL.md", "new")
	write(extract, "output-styles/apex.md", "new")
	// Existing install: the kept artifacts (old), dropped ones, and user files.
	write(root, "commands/ax-keep.md", "old")
	write(root, "commands/ax-dropped.md", "old")
	write(root, "commands/my-own.md", "user")
	write(root, "agents/ax-keep.md", "old")
	write(root, "agents/ax-dropped.md", "old")
	write(root, "agents/my-agent.md", "user")
	write(root, "skills/ax-keep/SKILL.md", "old")
	write(root, "skills/ax-dropped/SKILL.md", "old")
	write(root, "skills/my-skill/SKILL.md", "user")

	if err := applyArtifacts(extract, root); err != nil {
		t.Fatal(err)
	}

	for _, rel := range []string{"commands/ax-dropped.md", "agents/ax-dropped.md", "skills/ax-dropped"} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); !os.IsNotExist(err) {
			t.Errorf("%s should have been pruned", rel)
		}
	}
	for rel, want := range map[string]string{
		"commands/ax-keep.md": "new", "agents/ax-keep.md": "new", "skills/ax-keep/SKILL.md": "new",
		"commands/my-own.md": "user", "agents/my-agent.md": "user", "skills/my-skill/SKILL.md": "user",
	} {
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil || string(b) != want {
			t.Errorf("%s = %q (err %v), want %q", rel, b, err, want)
		}
	}
}

// A bundle missing a whole artifact directory is malformed, not a release that
// deleted everything. Pruning keys on what the bundle ships, so an absent or
// empty directory must never wipe the installed set.
func TestApplyArtifactsDoesNotPruneWhenBundleLacksDirectory(t *testing.T) {
	extract, root := t.TempDir(), t.TempDir()
	os.MkdirAll(filepath.Join(extract, "agents"), 0o755) // present but empty
	for _, rel := range []string{"commands/ax-a.md", "agents/ax-b.md", "skills/ax-c/SKILL.md"} {
		p := filepath.Join(root, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte("old"), 0o644)
	}
	if err := applyArtifacts(extract, root); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"commands/ax-a.md", "agents/ax-b.md", "skills/ax-c/SKILL.md"} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			t.Errorf("%s was removed by a bundle that shipped no replacement set", rel)
		}
	}
}
