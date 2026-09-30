package doctor

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeArtifacts lays down the shared artifact surface both layouts require:
// one output-style, agent, command, and a skill dir with a SKILL.md.
func writeArtifacts(t *testing.T, root string) {
	t.Helper()
	must := func(p, body string) {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	must(filepath.Join(root, "output-styles", "apex.md"), "# style")
	must(filepath.Join(root, "agents", "ax-x.md"), "# agent")
	must(filepath.Join(root, "commands", "ax-x.md"), "# command")
	must(filepath.Join(root, "skills", "ax-x", "SKILL.md"), "# skill")
}

func run(t *testing.T, root string) (int, string) {
	t.Helper()
	t.Setenv("CLAUDE_PLUGIN_ROOT", root)
	var buf bytes.Buffer
	code := Run(&buf)
	return code, buf.String()
}

func TestLooseInstallPasses(t *testing.T) {
	root := t.TempDir()
	writeArtifacts(t, root)
	os.WriteFile(filepath.Join(root, "settings.json"),
		[]byte(`{"hooks":{"SessionStart":[{"hooks":[{"command":"/x/apex hooks session-start"}]}]}}`), 0o644)

	code, out := run(t, root)
	if code != 0 {
		t.Fatalf("want pass, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "loose install") {
		t.Errorf("expected loose-install layout label\n%s", out)
	}
	// Loose layout must NOT enforce the plugin-era contract.
	if strings.Contains(out, "plugin.json") {
		t.Errorf("loose install should not check plugin.json\n%s", out)
	}
	if !strings.Contains(out, "apex hooks wired in settings.json") {
		t.Errorf("expected the settings.json hook-wiring check\n%s", out)
	}
}

// A Windows install wires `apex.exe hooks ...`; the detector must not require
// the bare `apex hooks` form or it false-negatives on every Windows machine.
func TestLooseInstallPassesWithWindowsExeHook(t *testing.T) {
	root := t.TempDir()
	writeArtifacts(t, root)
	os.WriteFile(filepath.Join(root, "settings.json"),
		[]byte(`{"hooks":{"SessionStart":[{"hooks":[{"command":"C:\\x\\apex.exe hooks session-start"}]}]}}`), 0o644)

	code, out := run(t, root)
	if code != 0 {
		t.Fatalf("want pass with apex.exe hook, got %d\n%s", code, out)
	}
}

func TestLooseInstallFailsWithoutWiredHooks(t *testing.T) {
	root := t.TempDir()
	writeArtifacts(t, root)
	os.WriteFile(filepath.Join(root, "settings.json"), []byte(`{"hooks":{}}`), 0o644)

	code, out := run(t, root)
	if code != 1 {
		t.Fatalf("want failure when hooks unwired, got %d\n%s", code, out)
	}
}

func TestLooseInstallFailsWithoutSettingsFile(t *testing.T) {
	root := t.TempDir()
	writeArtifacts(t, root)
	// No settings.json at all — apexHooksWired must treat the read error as unwired.

	code, _ := run(t, root)
	if code != 1 {
		t.Fatalf("want failure when settings.json is absent, got %d", code)
	}
}

func TestDevLayoutChecksManifest(t *testing.T) {
	root := t.TempDir()
	writeArtifacts(t, root)
	os.MkdirAll(filepath.Join(root, ".claude-plugin"), 0o755)
	os.WriteFile(filepath.Join(root, ".claude-plugin", "plugin.json"), []byte(`{"name":"x"}`), 0o644)
	os.MkdirAll(filepath.Join(root, "hooks"), 0o755)
	os.WriteFile(filepath.Join(root, "hooks", "hooks.json"), []byte(`{}`), 0o644)

	code, out := run(t, root)
	if code != 0 {
		t.Fatalf("want pass, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "dev/plugin layout") {
		t.Errorf("expected dev/plugin layout label\n%s", out)
	}
	if !strings.Contains(out, "plugin.json is valid JSON") {
		t.Errorf("dev layout should check plugin.json\n%s", out)
	}
}

func TestDevLayoutFailsOnBadManifest(t *testing.T) {
	root := t.TempDir()
	writeArtifacts(t, root)
	os.MkdirAll(filepath.Join(root, ".claude-plugin"), 0o755)
	os.WriteFile(filepath.Join(root, ".claude-plugin", "plugin.json"), []byte(`{bad json`), 0o644)
	// Valid hooks.json so the bad manifest is the *only* failing check.
	os.MkdirAll(filepath.Join(root, "hooks"), 0o755)
	os.WriteFile(filepath.Join(root, "hooks", "hooks.json"), []byte(`{}`), 0o644)

	code, _ := run(t, root)
	if code != 1 {
		t.Fatalf("want failure on invalid plugin.json, got %d", code)
	}
}

func TestDirOnPathTrailingSlashAndCase(t *testing.T) {
	dir := t.TempDir()
	sep := string(os.PathListSeparator)

	// Trailing separator on the PATH entry must still match.
	entry := dir + string(os.PathSeparator)
	if !dirOnPath(dir, "/somewhere/else"+sep+entry) {
		t.Errorf("trailing-separator PATH entry should match %q", dir)
	}
	if dirOnPath(dir, "/somewhere/else"+sep+"/not/it") {
		t.Error("unrelated PATH must not match")
	}
	if dirOnPath(dir, "") {
		t.Error("empty PATH must not match")
	}
}

func TestDirOnPathResolvesSymlinks(t *testing.T) {
	real := t.TempDir()
	link := filepath.Join(t.TempDir(), "bin-link")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	// PATH carries the symlink; the binary reports the real dir. Must match.
	if !dirOnPath(real, link) {
		t.Error("symlinked PATH entry should match the resolved dir")
	}
	// And the inverse: binary dir is the symlink, PATH has the real path.
	if !dirOnPath(link, real) {
		t.Error("real PATH entry should match the symlinked dir")
	}
}

// --- followup 001: hooks.json must not point at a binary that was never built ---

// devRoot lays down a dev/plugin layout whose hooks.json references the
// gitignored bin/apex, with no binary present.
func devRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeArtifacts(t, root)
	os.MkdirAll(filepath.Join(root, ".claude-plugin"), 0o755)
	os.WriteFile(filepath.Join(root, ".claude-plugin", "plugin.json"), []byte(`{"name":"x"}`), 0o644)
	os.MkdirAll(filepath.Join(root, "hooks"), 0o755)
	os.WriteFile(filepath.Join(root, "hooks", "hooks.json"), []byte(
		`{"hooks":{"SessionStart":[{"hooks":[{"type":"command","command":"${CLAUDE_PLUGIN_ROOT}/bin/apex hooks session-start"}]}]}}`), 0o644)
	return root
}

func TestDevLayoutFailsWhenHookBinaryMissing(t *testing.T) {
	root := devRoot(t)
	code, out := run(t, root)
	if code != 1 {
		t.Fatalf("want failure when bin/apex is absent, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "hooks.json targets exist") {
		t.Errorf("expected a hook-target failure\n%s", out)
	}
	if !strings.Contains(out, "make build") {
		t.Errorf("failure should point at the fix\n%s", out)
	}
}

func TestDevLayoutPassesWhenHookBinaryBuilt(t *testing.T) {
	root := devRoot(t)
	os.MkdirAll(filepath.Join(root, "bin"), 0o755)
	os.WriteFile(filepath.Join(root, "bin", "apex"), []byte("#!/bin/sh\n"), 0o755)

	if code, out := run(t, root); code != 0 {
		t.Fatalf("want pass once the binary exists, got %d\n%s", code, out)
	}
}

// A Windows checkout builds apex.exe; the same hooks.json entry must satisfy it.
func TestDevLayoutAcceptsExeSuffix(t *testing.T) {
	root := devRoot(t)
	os.MkdirAll(filepath.Join(root, "bin"), 0o755)
	os.WriteFile(filepath.Join(root, "bin", "apex.exe"), []byte("MZ"), 0o644)

	if code, out := run(t, root); code != 0 {
		t.Fatalf("want pass with apex.exe, got %d\n%s", code, out)
	}
}
