package repo

import (
	"os"
	"os/exec"
	"path/filepath"
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
