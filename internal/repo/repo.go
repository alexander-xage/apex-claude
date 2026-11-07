// Package repo resolves where protocol state lives for a working directory.
package repo

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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

// ponytail: any WORD-123 in the branch counts, so release-2 reads as a ticket; match git.md's ticket prefix if that bites.
var ticketRe = regexp.MustCompile(`(?i)\b[a-z][a-z0-9]+-[0-9]+`)

// Ticket names the ticket dir's checked-out branch works on: the ticket id in the branch, upper-cased (feat/eng-42-x
// gives ENG-42), or else the branch itself with / as -. It is "" when HEAD is detached or dir is outside git.
func Ticket(dir string) (string, error) {
	branch, err := gitOut(dir, "branch", "--show-current")
	if errors.Is(err, errNotRepo) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if t := ticketRe.FindString(branch); t != "" {
		return strings.ToUpper(t), nil
	}
	return strings.ReplaceAll(branch, "/", "-"), nil
}
