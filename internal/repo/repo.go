// Package repo resolves where protocol state lives for a working directory.
package repo

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

// Main returns the main checkout that owns dir: the directory whose .claude/ holds every Agent's state, even when dir
// is inside a linked worktree. Outside a git repository it returns dir itself.
func Main(dir string) (string, error) {
	common, err := gitOut(dir, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if errors.Is(err, errNotRepo) {
		return filepath.Abs(dir)
	}
	if err != nil {
		return "", err
	}
	if filepath.Base(common) == ".git" {
		return filepath.Dir(common), nil
	}
	// Submodules, --separate-git-dir and bare repos keep the common dir elsewhere. The toplevel is only the main
	// checkout when dir is not a linked worktree; for a linked worktree there is no main checkout to find.
	gitDir, err := gitOut(dir, "rev-parse", "--path-format=absolute", "--git-dir")
	if err != nil {
		return "", err
	}
	if gitDir != common {
		return "", fmt.Errorf("%s is a linked worktree of a repository with no main checkout (%s)", dir, common)
	}
	return gitOut(dir, "rev-parse", "--show-toplevel")
}

var errNotRepo = errors.New("not a git repository")

func gitOut(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	// Git localizes its messages; the "outside any repo" check below matches the English text.
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		// Only a search that found no repo at all means "outside git". A worktree whose main checkout moved also says
		// "not a git repository", naming its dangling gitdir, and must fail rather than fall back to dir.
		if strings.Contains(stderr.String(), "not a git repository (or any") {
			return "", errNotRepo
		}
		return "", fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(string(out)), nil
}

// Checkout names the checkout dir belongs to, given its main checkout: "main" for the main checkout or outside git,
// the worktree's directory name otherwise. It errors when that name is "main" or another live worktree shares it.
func Checkout(dir, main string) (string, error) {
	top, err := gitOut(dir, "rev-parse", "--show-toplevel")
	if errors.Is(err, errNotRepo) {
		return "main", nil
	}
	if err != nil {
		return "", err
	}
	if top == main {
		return "main", nil
	}
	name := filepath.Base(top)
	if name == "main" {
		return "", fmt.Errorf("worktree %s is named main, which is reserved for the main checkout", top)
	}
	live, err := LiveWorktrees(dir)
	if err != nil {
		return "", err
	}
	n := 0
	for _, w := range live {
		if w == name {
			n++
		}
	}
	if n > 1 {
		return "", fmt.Errorf("%d live worktrees are named %s; rename one so each keeps its own graph", n, name)
	}
	return name, nil
}

// LiveWorktrees returns the directory names of the linked worktrees that exist on disk. Only a directory known to be
// missing counts as gone: an unreadable one, or a locked one (git keeps it; it may sit on unmounted media), is live.
func LiveWorktrees(dir string) ([]string, error) {
	out, err := gitOut(dir, "worktree", "list", "--porcelain")
	if errors.Is(err, errNotRepo) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var names []string
	for i, block := range strings.Split(out, "\n\n") {
		lines := strings.Split(block, "\n")
		path, ok := strings.CutPrefix(lines[0], "worktree ")
		if !ok || i == 0 {
			continue
		}
		_, err := os.Stat(path)
		if !errors.Is(err, fs.ErrNotExist) || slices.ContainsFunc(lines, func(l string) bool { return l == "locked" || strings.HasPrefix(l, "locked ") }) {
			names = append(names, filepath.Base(path))
		}
	}
	return names, nil
}
