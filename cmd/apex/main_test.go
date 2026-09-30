package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func setup(t *testing.T) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	skill := filepath.Join(root, ".claude", "skills", "dev")
	if err := os.MkdirAll(skill, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skill, "SKILL.md"), []byte("---\nname: dev\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func apex(t *testing.T, dir string, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := run(dir, args, &out, &errb)
	return code, out.String(), errb.String()
}

// The model writes flags after positionals; stdlib flag parsing alone would silently drop them.
func TestFlagsAfterPositionals(t *testing.T) {
	root := setup(t)
	code, out, errs := apex(t, root, "add", "task", "wire the hook", "--agent", "dev")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errs)
	}
	want := filepath.Join(root, ".claude", "ledger", "dev", "001.md")
	if strings.TrimSpace(out) != want {
		t.Fatalf("add printed %q, want %q", out, want)
	}
}

func TestExitCodes(t *testing.T) {
	root := setup(t)
	apex(t, root, "add", "task", "a", "--agent", "dev")
	cases := []struct {
		args []string
		code int
	}{
		{[]string{}, 64},
		{[]string{"frobnicate"}, 64},
		{[]string{"list"}, 64}, // no default agent
		{[]string{"list", "--agent", "dev", "--bogus"}, 64},
		{[]string{"start", "../x", "--agent", "dev"}, 64},
		{[]string{"start", "9", "--agent", "dev"}, 1},
		{[]string{"list", "--agent", "ghost"}, 1},
		{[]string{"start", "1", "--agent", "dev"}, 0},
		{[]string{"start", "1", "--agent", "dev"}, 3},
		{[]string{"close", "1", "--agent", "dev"}, 64}, // reason required
		{[]string{"close", "1", "shipped", "--agent", "dev"}, 0},
		{[]string{"reopen", "1", "--agent", "dev"}, 0},
	}
	for _, c := range cases {
		if code, _, errs := apex(t, root, c.args...); code != c.code {
			t.Errorf("apex %v: exit %d, want %d (%s)", c.args, code, c.code, strings.TrimSpace(errs))
		}
	}
}

func TestListFiltersAndCounts(t *testing.T) {
	root := setup(t)
	apex(t, root, "add", "task", "alpha", "--agent", "dev")
	apex(t, root, "add", "followup", "beta", "--agent", "dev")
	apex(t, root, "close", "1", "done", "--agent", "dev")

	_, out, _ := apex(t, root, "list", "--agent", "dev")
	if strings.Contains(out, "alpha") || !strings.Contains(out, "beta") {
		t.Fatalf("default list must hide closed items:\n%s", out)
	}
	if !strings.Contains(out, "1 open, 0 in-progress, 1 closed") {
		t.Fatalf("missing counts line:\n%s", out)
	}
	_, out, _ = apex(t, root, "list", "--agent", "dev", "--status", "all", "--kind", "task")
	if !strings.Contains(out, "alpha") || strings.Contains(out, "beta") {
		t.Fatalf("--status all --kind task:\n%s", out)
	}
}

// Every Agent's state lives in the main checkout, so a call from a subdirectory must reach the same ledger.
func TestRunFromSubdirUsesMainCheckout(t *testing.T) {
	root := setup(t)
	if out, err := exec.Command("git", "init", "-q", root).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	sub := filepath.Join(root, "pkg")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if code, _, errs := apex(t, root, "add", "task", "a", "--agent", "dev"); code != 0 {
		t.Fatal(errs)
	}
	_, out, _ := apex(t, sub, "list", "--agent", "dev")
	if !strings.Contains(out, "001") {
		t.Fatalf("list from subdir missed the main checkout's ledger:\n%s", out)
	}
}

// Usage is judged before the disk is read, so the same bad call exits 64 whatever the ledger holds.
func TestUsageBeatsDiskState(t *testing.T) {
	root := setup(t)
	apex(t, root, "add", "task", "a", "--agent", "dev")
	if err := os.WriteFile(filepath.Join(root, ".claude", "ledger", "dev", "002.md"), []byte("junk"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"start", "abc", "--agent", "ghost"},
		{"add", "bogus", "t", "--agent", "ghost"},
		{"list", "--status", "bogus", "--agent", "dev"},
		{"list", "--kind", "taks", "--agent", "dev"},
		{"list", "--title", "x", "--agent", "dev"},
		{"start", "1", "--status", "open", "--agent", "dev"},
		{"close", "1", "a\nb", "--agent", "dev"},
	} {
		if code, _, errs := apex(t, root, args...); code != 64 {
			t.Errorf("apex %q: exit %d, want 64 (%s)", args, code, strings.TrimSpace(errs))
		}
	}
	if code, _, _ := apex(t, root, "list", "--agent", "dev"); code != 4 {
		t.Errorf("list over a corrupt item: exit %d, want 4", code)
	}
}

func TestEnvironmentErrorExits2(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	root := setup(t)
	apex(t, root, "add", "task", "a", "--agent", "dev")
	dir := filepath.Join(root, ".claude", "ledger", "dev")
	if err := os.Chmod(dir, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })
	if code, _, _ := apex(t, root, "list", "--agent", "dev"); code != 2 {
		t.Errorf("unreadable ledger: exit %d, want 2", code)
	}
}

func TestDashTitleAfterDoubleDash(t *testing.T) {
	root := setup(t)
	if code, _, errs := apex(t, root, "add", "task", "--agent", "dev", "--", "-starts with dash"); code != 0 {
		t.Fatalf("title after --: exit %d (%s)", code, errs)
	}
	if _, out, _ := apex(t, root, "list", "--agent", "dev"); !strings.Contains(out, "-starts with dash") {
		t.Fatalf("dash title missing:\n%s", out)
	}
}

func TestHelpInsideVerb(t *testing.T) {
	root := setup(t)
	if code, out, _ := apex(t, root, "list", "-h"); code != 0 || !strings.Contains(out, "usage:") {
		t.Fatalf("list -h: exit %d, out %q", code, out)
	}
}

func TestCountsFollowKindFilter(t *testing.T) {
	root := setup(t)
	apex(t, root, "add", "task", "a", "--agent", "dev")
	apex(t, root, "add", "followup", "b", "--agent", "dev")
	if _, out, _ := apex(t, root, "list", "--agent", "dev", "--kind", "followup"); !strings.Contains(out, "1 open, 0 in-progress, 0 closed") {
		t.Fatalf("counts ignore --kind:\n%s", out)
	}
}

// Where the main checkout cannot be resolved, only the check before repo.Main turns a bad agent name into 64.
func TestBadAgentIsUsageEvenWhenGitFails(t *testing.T) {
	r, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	wt := filepath.Join(t.TempDir(), "wt")
	for _, args := range [][]string{{"init", "-q"}, {"-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "--allow-empty", "-m", "i"}, {"worktree", "add", "-q", "-b", "f", wt}} {
		if out, err := exec.Command("git", append([]string{"-C", r}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	if err := os.Rename(r, r+"-moved"); err != nil {
		t.Fatal(err)
	}
	if code, _, errs := apex(t, wt, "start", "1", "--agent", "Dev"); code != 64 {
		t.Fatalf("exit %d, want 64 (%s)", code, errs)
	}
}
