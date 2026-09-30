package repo

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// tempDir resolves symlinks so paths compare equal on macOS, where TMPDIR sits under /var -> /private/var.
func tempDir(t *testing.T) string {
	d, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func newRepo(t *testing.T) string {
	d := tempDir(t)
	git(t, d, "init", "-q")
	git(t, d, "commit", "-q", "--allow-empty", "-m", "init")
	return d
}

func TestMainFromSubdirIsCheckoutRoot(t *testing.T) {
	r := newRepo(t)
	sub := filepath.Join(r, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := Main(sub)
	if err != nil || got != r {
		t.Fatalf("Main(sub) = %q, %v; want %q", got, err, r)
	}
}

// State belongs to the main checkout: an Agent working in a worktree must still read and write the main .claude/.
func TestMainFromWorktreeIsMainCheckout(t *testing.T) {
	r := newRepo(t)
	wt := filepath.Join(r, ".claude", "worktrees", "feat")
	git(t, r, "worktree", "add", "-q", "-b", "feat", wt)
	got, err := Main(wt)
	if err != nil || got != r {
		t.Fatalf("Main(worktree) = %q, %v; want %q", got, err, r)
	}
}

func TestMainOutsideGitIsDir(t *testing.T) {
	d := tempDir(t)
	got, err := Main(d)
	if err != nil || got != d {
		t.Fatalf("Main(non-git) = %q, %v; want %q", got, err, d)
	}
}

// A worktree whose main checkout moved must fail: falling back to the worktree would write state that is lost when
// the worktree is removed.
func TestMainFromOrphanedWorktreeFails(t *testing.T) {
	r := newRepo(t)
	wt := filepath.Join(tempDir(t), "wt")
	git(t, r, "worktree", "add", "-q", "-b", "feat", wt)
	if err := os.Rename(r, r+"-moved"); err != nil {
		t.Fatal(err)
	}
	if got, err := Main(wt); err == nil {
		t.Fatalf("Main(orphaned worktree) = %q, nil; want an error", got)
	}
}

// With --separate-git-dir the common dir is not named .git; a linked worktree then has no main checkout to resolve to.
func TestMainFromSeparateGitDirWorktreeFails(t *testing.T) {
	r, store := tempDir(t), filepath.Join(tempDir(t), "store.git")
	git(t, r, "init", "-q", "--separate-git-dir", store)
	git(t, r, "commit", "-q", "--allow-empty", "-m", "init")
	if got, err := Main(r); err != nil || got != r {
		t.Fatalf("Main(main checkout) = %q, %v; want %q", got, err, r)
	}
	wt := filepath.Join(tempDir(t), "wt")
	git(t, r, "worktree", "add", "-q", "-b", "feat", wt)
	if got, err := Main(wt); err == nil {
		t.Fatalf("Main(worktree of separate git dir) = %q, nil; want an error", got)
	}
}

func checkout(t *testing.T, dir string) (string, error) {
	t.Helper()
	main, err := Main(dir)
	if err != nil {
		t.Fatal(err)
	}
	return Checkout(dir, main)
}

func TestCheckoutNames(t *testing.T) {
	r := newRepo(t)
	wt := filepath.Join(tempDir(t), "wt-feat")
	git(t, r, "worktree", "add", "-q", "-b", "feat", wt)
	sub := filepath.Join(wt, "x", "y")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	for dir, want := range map[string]string{r: "main", wt: "wt-feat", sub: "wt-feat", tempDir(t): "main"} {
		if got, err := checkout(t, dir); err != nil || got != want {
			t.Errorf("Checkout(%s) = %q, %v; want %q", dir, got, err, want)
		}
	}
}

// A worktree directory named main would share the main checkout's graph.
func TestCheckoutNamedMainFails(t *testing.T) {
	r := newRepo(t)
	wt := filepath.Join(tempDir(t), "main")
	git(t, r, "worktree", "add", "-q", "-b", "feat", wt)
	if got, err := checkout(t, wt); err == nil {
		t.Fatalf("Checkout(worktree named main) = %q, nil; want an error", got)
	}
}

// Graphs are keyed by worktree directory name, so two live worktrees with one name would share a graph.
func TestCheckoutDuplicateNameFails(t *testing.T) {
	r := newRepo(t)
	a, b := filepath.Join(tempDir(t), "feat"), filepath.Join(tempDir(t), "feat")
	git(t, r, "worktree", "add", "-q", "-b", "a", a)
	git(t, r, "worktree", "add", "-q", "-b", "b", b)
	if got, err := checkout(t, a); err == nil {
		t.Fatalf("Checkout(duplicate name) = %q, nil; want an error", got)
	}
}

// git keeps an entry for a worktree whose directory was deleted until `git worktree prune`; only directories on disk
// count, except a locked worktree, which git keeps too.
func TestLiveWorktreesKeepsOnDiskAndLocked(t *testing.T) {
	r := newRepo(t)
	keep, gone, locked := filepath.Join(tempDir(t), "keep"), filepath.Join(tempDir(t), "gone"), filepath.Join(tempDir(t), "locked")
	git(t, r, "worktree", "add", "-q", "-b", "a", keep)
	git(t, r, "worktree", "add", "-q", "-b", "b", gone)
	git(t, r, "worktree", "add", "-q", "--lock", "-b", "c", locked)
	for _, d := range []string{gone, locked} {
		if err := os.RemoveAll(d); err != nil {
			t.Fatal(err)
		}
	}
	got, err := LiveWorktrees(r)
	if err != nil || strings.Join(got, ",") != "keep,locked" {
		t.Fatalf("LiveWorktrees = %v, %v; want [keep locked]", got, err)
	}
	if got, err := LiveWorktrees(tempDir(t)); err != nil || len(got) != 0 {
		t.Fatalf("LiveWorktrees(non-git) = %v, %v; want none", got, err)
	}
}

// Liveness only feeds a delete; when a directory cannot be checked, keeping its graph is the safe answer.
func TestLiveWorktreesKeepsUnreadable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	r := newRepo(t)
	parent := tempDir(t)
	git(t, r, "worktree", "add", "-q", "-b", "a", filepath.Join(parent, "hidden"))
	if err := os.Chmod(parent, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(parent, 0o755) })
	got, err := LiveWorktrees(r)
	if err != nil || strings.Join(got, ",") != "hidden" {
		t.Fatalf("LiveWorktrees = %v, %v; want [hidden]", got, err)
	}
}
