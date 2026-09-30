package handoff

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"apexclaude/internal/fm"
)

// initGitRepo creates a minimal git repo with one commit in dir.
// Returns false if git is not available, caller should t.Skip.
func initGitRepo(t *testing.T, dir string) bool {
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
			"HOME="+t.TempDir(), // isolate global config
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-b", "main")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test")
	// write a file so we can commit
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "README.md")
	run("commit", "-m", "initial commit")
	return true
}

// makeCommit adds another commit to the repo so HEAD advances.
func makeCommit(t *testing.T, dir string) {
	t.Helper()
	f := filepath.Join(dir, "extra.md")
	if err := os.WriteFile(f, []byte("extra\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("add", "extra.md")
	run("commit", "-m", "second commit")
}

// --- Checkpoint 1 & 2: Scan ---

func TestScanGitFields(t *testing.T) {
	root := t.TempDir()
	if !initGitRepo(t, root) {
		t.Skip("git not available")
	}

	s, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}
	if s.Branch == "" {
		t.Error("Branch should be non-empty in a git repo")
	}
	if s.Head == "" {
		t.Error("Head should be non-empty in a git repo")
	}
	if len(s.Head) > 12 {
		t.Errorf("Head should be short sha, got %q (len %d)", s.Head, len(s.Head))
	}
	if s.LastCommit == "" {
		t.Error("LastCommit should be non-empty")
	}
}

func TestScanNonRepo(t *testing.T) {
	root := t.TempDir()
	// no git init — should not error
	s, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan outside repo should not error, got: %v", err)
	}
	// git fields should be zero values
	if s.Branch != "" || s.Head != "" {
		t.Errorf("expected zero git fields outside repo, got branch=%q head=%q", s.Branch, s.Head)
	}
}

func TestScanNonGitFields(t *testing.T) {
	root := t.TempDir()
	// No followups, reminders, health file → zero/defaults expected
	s, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}
	// OpenFollowups should be 0 when none exist
	if s.OpenFollowups != 0 {
		t.Errorf("expected OpenFollowups=0, got %d", s.OpenFollowups)
	}
	// DueReminders should be 0 when none exist
	if s.DueReminders != 0 {
		t.Errorf("expected DueReminders=0, got %d", s.DueReminders)
	}
	// Health should be -1 when file absent
	if s.Health != -1 {
		t.Errorf("expected Health=-1, got %d", s.Health)
	}
}

// --- Checkpoint 3: Render ---

// --- Report: the deterministic half the model composes from ---

func TestReportRendersEveryScannedFact(t *testing.T) {
	s := State{
		Branch: "feat/x", Head: "abc1234", Dirty: true,
		Staged: []string{"a.go", "b.go"}, LastCommit: "feat: land the thing",
		OpenFollowups: 7, DueReminders: 2, Health: 88,
		SignalsStale: true, SignalsReason: "manifests changed since last scan — re-scan",
		BriefPath: "/tmp/x/BRIEF.md",
	}
	out := Report(s)
	// Every field of State must reach the report — the old Render dropped eight
	// of them silently, which is the regression this test exists to prevent.
	for _, want := range []string{
		"branch:", "feat/x",
		"head:", "abc1234",
		"dirty:", "yes",
		"staged:", "a.go, b.go",
		"last-commit:", "feat: land the thing",
		"followups:", "7 open",
		"reminders:", "2 due",
		"health:", "88",
		"signals:", "STALE", "manifests changed",
		"brief:", "/tmp/x/BRIEF.md",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("report missing %q\n%s", want, out)
		}
	}
}

func TestReportZeroState(t *testing.T) {
	out := Report(State{Health: -1})
	for _, want := range []string{
		"branch:       (none)",
		"dirty:        no",
		"staged:       (none)",
		"followups:    0 open",
		"reminders:    0 due",
		"health:       unset",
		"signals:      fresh",
		"brief:        (none)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("report missing %q\n%s", want, out)
		}
	}
}

func TestReportWritesNothing(t *testing.T) {
	root := t.TempDir()
	_ = Report(State{Health: -1})
	if _, err := os.Stat(Path(root)); !os.IsNotExist(err) {
		t.Errorf("Report must not create %s", Path(root))
	}
}

// ScanWritesNothing guards the property /ax-resume depends on: scan is safe to
// run while a handoff doc is being consumed, because it never touches disk.
func TestScanWritesNothing(t *testing.T) {
	root := t.TempDir()
	seedDoc(t, root, "deadbee")
	before := readDoc(t, root)
	if _, err := Scan(root); err != nil {
		t.Fatal(err)
	}
	if after := readDoc(t, root); after != before {
		t.Errorf("Scan mutated the active doc\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

// --- the model-authored document: Path / Status / Archive ---

// seedDoc writes a handoff doc the way the MODEL does — the binary no longer
// has a writer, so tests author the document directly.
func seedDoc(t *testing.T, root, head string) string {
	t.Helper()
	dir := filepath.Join(root, ".claude", "project")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	doc := fm.Render(
		[]string{"mode", "created", "branch", "head", "health", "status"},
		map[string]string{
			"mode": "graceful", "created": "2026-06-19T12:00:00Z",
			"branch": "main", "head": head, "health": "88", "status": "open",
		},
		"## Shipped\n\nthe thing\n\n## Next\n\nthe next thing\n")
	if err := os.WriteFile(Path(root), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	return doc
}

func readDoc(t *testing.T, root string) string {
	t.Helper()
	b, err := os.ReadFile(Path(root))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestPath(t *testing.T) {
	root := t.TempDir()
	p := Path(root)
	want := filepath.Join(root, ".claude", "project", "handoff.md")
	if p != want {
		t.Errorf("Path = %q, want %q", p, want)
	}
}

func TestStatusAbsent(t *testing.T) {
	root := t.TempDir()
	if code := Status(root); code != 1 {
		t.Errorf("Status with no doc = %d, want 1", code)
	}
}

func TestStatusFresh(t *testing.T) {
	root := t.TempDir()
	if !initGitRepo(t, root) {
		t.Skip("git not available")
	}
	live, _ := gitHead(root)
	seedDoc(t, root, live)
	if code := Status(root); code != 0 {
		t.Errorf("Status with matching head = %d, want 0", code)
	}
}

func TestStatusStale(t *testing.T) {
	root := t.TempDir()
	if !initGitRepo(t, root) {
		t.Skip("git not available")
	}
	live, _ := gitHead(root)
	seedDoc(t, root, live)
	makeCommit(t, root)
	if code := Status(root); code != 2 {
		t.Errorf("Status after HEAD moved = %d, want 2", code)
	}
}

func TestArchiveMovesDocAndMarksConsumed(t *testing.T) {
	root := t.TempDir()
	seedDoc(t, root, "abc1234")

	id, err := Archive(root)
	if err != nil {
		t.Fatal(err)
	}
	if id != "001" {
		t.Errorf("first archive id = %q, want 001", id)
	}
	if _, err := os.Stat(Path(root)); !os.IsNotExist(err) {
		t.Error("active doc should be gone after Archive")
	}

	b, err := os.ReadFile(filepath.Join(root, ".claude", "project", "handoffs", "001.md"))
	if err != nil {
		t.Fatal(err)
	}
	meta, body := fm.Parse(string(b))
	if meta["status"] != "consumed" {
		t.Errorf("archived status = %q, want consumed", meta["status"])
	}
	if meta["head"] != "abc1234" {
		t.Errorf("archived head = %q, want abc1234", meta["head"])
	}
	// The model's narrative must survive the archive round-trip.
	if !strings.Contains(body, "the next thing") {
		t.Errorf("archive dropped the model's body\n%s", body)
	}
}

func TestArchiveNextID(t *testing.T) {
	root := t.TempDir()
	seedDoc(t, root, "aaa1111")
	id1, err := Archive(root)
	if err != nil {
		t.Fatal(err)
	}
	seedDoc(t, root, "bbb2222")
	id2, err := Archive(root)
	if err != nil {
		t.Fatal(err)
	}
	if id1 != "001" || id2 != "002" {
		t.Errorf("archive ids = %q, %q; want 001, 002", id1, id2)
	}
}

func TestArchiveAbsent(t *testing.T) {
	root := t.TempDir()
	if _, err := Archive(root); err == nil {
		t.Error("Archive with no active doc should error")
	}
}
