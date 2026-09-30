package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"apexclaude/internal/handoff"
)

// initTestGitRepo creates a minimal git repo with one commit in dir.
// Returns false if git is not available; caller should t.Skip.
func initTestGitRepo(t *testing.T, dir string) bool {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		return false
	}
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_CONFIG_NOSYSTEM=1",
			"HOME="+t.TempDir(),
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-b", "main")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "README.md")
	run("commit", "-m", "initial commit")
	return true
}

// makeTestCommit adds another commit so HEAD advances.
func makeTestCommit(t *testing.T, dir string) {
	t.Helper()
	f := filepath.Join(dir, "extra.md")
	if err := os.WriteFile(f, []byte("extra\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_CONFIG_NOSYSTEM=1",
			"HOME="+t.TempDir(),
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("add", "extra.md")
	run("commit", "-m", "second commit")
}

// seedHandoffDoc writes a handoff doc the way the MODEL does — the binary has
// no writer, so the CLI tests author the document directly.
func seedHandoffDoc(t *testing.T, root, head string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(handoff.Path(root)), 0o755); err != nil {
		t.Fatal(err)
	}
	doc := "---\nmode: graceful\ncreated: 2026-06-19T12:00:00Z\nbranch: main\nhead: " +
		head + "\nhealth: 88\nstatus: open\n---\n\n## Shipped\n\nthe thing\n"
	if err := os.WriteFile(handoff.Path(root), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
}

func gitShortHead(t *testing.T, dir string) string {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}

func TestHandoffStatusAbsent(t *testing.T) {
	root := t.TempDir()
	t.Setenv("APEX_REPO", root)

	if code := runHandoff([]string{"status"}); code != 1 {
		t.Errorf("status absent: got %d want 1", code)
	}
}

// scan is a READ-ONLY reporter: it must print facts and create nothing.
func TestHandoffScanWritesNothing(t *testing.T) {
	root := t.TempDir()
	if !initTestGitRepo(t, root) {
		t.Skip("git not available")
	}
	t.Setenv("APEX_REPO", root)

	if code := runHandoff([]string{"scan"}); code != 0 {
		t.Fatalf("scan returned %d want 0", code)
	}
	if _, err := os.Stat(handoff.Path(root)); !os.IsNotExist(err) {
		t.Error("scan must not create the handoff doc — the model writes it")
	}
}

// scan takes no mode argument now; the mode is the model's to choose.
func TestHandoffScanRejectsExtraArg(t *testing.T) {
	root := t.TempDir()
	t.Setenv("APEX_REPO", root)

	if code := runHandoff([]string{"scan", "urgent"}); code != 2 {
		t.Errorf("scan with a mode arg: got %d want 2", code)
	}
}

func TestHandoffStatusFresh(t *testing.T) {
	root := t.TempDir()
	if !initTestGitRepo(t, root) {
		t.Skip("git not available")
	}
	t.Setenv("APEX_REPO", root)
	seedHandoffDoc(t, root, gitShortHead(t, root))

	if code := runHandoff([]string{"status"}); code != 0 {
		t.Errorf("status fresh: got %d want 0", code)
	}
}

func TestHandoffStatusStaleAfterCommit(t *testing.T) {
	root := t.TempDir()
	if !initTestGitRepo(t, root) {
		t.Skip("git not available")
	}
	t.Setenv("APEX_REPO", root)
	seedHandoffDoc(t, root, gitShortHead(t, root))

	makeTestCommit(t, root)

	if code := runHandoff([]string{"status"}); code != 2 {
		t.Errorf("status stale: got %d want 2", code)
	}
}

func TestHandoffArchive(t *testing.T) {
	root := t.TempDir()
	t.Setenv("APEX_REPO", root)
	seedHandoffDoc(t, root, "abc1234")

	if code := runHandoff([]string{"archive"}); code != 0 {
		t.Errorf("archive returned %d want 0", code)
	}
	if _, err := os.Stat(handoff.Path(root)); !os.IsNotExist(err) {
		t.Error("active doc should be removed after archive")
	}

	entries, err := os.ReadDir(filepath.Join(root, ".claude", "project", "handoffs"))
	if err != nil {
		t.Fatalf("handoffs dir missing: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 archived file, got %d", len(entries))
	}
}

func TestHandoffArchiveNothingToArchive(t *testing.T) {
	root := t.TempDir()
	t.Setenv("APEX_REPO", root)

	if code := runHandoff([]string{"archive"}); code != 1 {
		t.Errorf("archive with nothing: got %d want 1", code)
	}
}

func TestHandoffUnknownSubcommand(t *testing.T) {
	t.Setenv("APEX_REPO", t.TempDir())
	if code := runHandoff([]string{"bogus"}); code != 2 {
		t.Errorf("unknown sub: got %d want 2", code)
	}
}

func TestHandoffNoSubcommand(t *testing.T) {
	t.Setenv("APEX_REPO", t.TempDir())
	if code := runHandoff([]string{}); code != 2 {
		t.Errorf("empty args: got %d want 2", code)
	}
}
