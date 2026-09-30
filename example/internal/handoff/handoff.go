// Package handoff provides session continuity for the apex backbone.
//
// The binary reports; the model writes. Scan collects repo state into a State
// struct and Report renders it as a plain-text fact table — that is the whole
// deterministic half. Composing .claude/project/handoff.md, frontmatter
// included, is the model's job (see commands/ax-handoff.md). The binary still
// owns the two mechanical operations on an existing doc: Status (staleness
// routing) and Archive (move to .claude/project/handoffs/<id>.md).
package handoff

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"apexclaude/internal/fm"
	"apexclaude/internal/followups"
	"apexclaude/internal/health"
	"apexclaude/internal/reminder"
	"apexclaude/internal/signals"
)

// State captures all facts needed for a handoff document.
type State struct {
	// git facts
	Branch     string
	Head       string // short sha
	Dirty      bool
	Staged     []string
	LastCommit string

	// non-git facts
	OpenFollowups int
	DueReminders  int
	Health        int // -1 if unset
	SignalsStale  bool
	SignalsReason string
	BriefPath     string
}

// Scan collects all handoff facts for the given repo root.
// Git facts come from isolated helpers; non-git facts come from library calls.
// Tolerate non-repo roots: git helpers return zero values + nil errors.
func Scan(root string) (State, error) {
	var s State
	s.Health = -1

	// git facts (each helper is individually tolerant of non-repo dirs)
	s.Branch, _ = gitBranch(root)
	s.Head, _ = gitHead(root)
	s.Dirty, _ = gitDirty(root)
	s.Staged, _ = gitStaged(root)
	s.LastCommit, _ = gitLastCommit(root)

	// followups
	entries, err := followups.List(root)
	if err == nil {
		s.OpenFollowups = len(entries)
	}

	// reminders
	now := time.Now()
	due, err := reminder.Due(root, now)
	if err == nil {
		s.DueReminders = len(due)
	}

	// health
	score, _, err := health.Show(root)
	if err == nil {
		s.Health = score
	}

	// signals staleness
	code, reason := signals.Stale(root)
	s.SignalsStale = code != 0
	s.SignalsReason = reason

	// active scratchpad BRIEF.md
	briefPath := filepath.Join(root, ".claude", ".scratchpad")
	entries2, _ := os.ReadDir(briefPath)
	for _, e := range entries2 {
		if !e.IsDir() {
			continue
		}
		candidate := filepath.Join(briefPath, e.Name(), "BRIEF.md")
		if _, err := os.Stat(candidate); err == nil {
			s.BriefPath = candidate
			break
		}
	}

	return s, nil
}

// Report renders a scanned State as a plain-text fact table for stdout. This is
// the input the model composes the handoff document from; the binary never
// writes that document itself.
func Report(s State) string {
	yn := func(b bool) string {
		if b {
			return "yes"
		}
		return "no"
	}
	orNone := func(v string) string {
		if v == "" {
			return "(none)"
		}
		return v
	}

	staged := "(none)"
	if len(s.Staged) > 0 {
		staged = strings.Join(s.Staged, ", ")
	}
	health := "unset"
	if s.Health >= 0 {
		health = strconv.Itoa(s.Health)
	}
	sig := "fresh"
	if s.SignalsStale {
		sig = "STALE — " + s.SignalsReason
	}

	var b strings.Builder
	row := func(k, v string) { fmt.Fprintf(&b, "%-13s %s\n", k+":", v) }
	row("branch", orNone(s.Branch))
	row("head", orNone(s.Head))
	row("dirty", yn(s.Dirty))
	row("staged", staged)
	row("last-commit", orNone(s.LastCommit))
	row("followups", fmt.Sprintf("%d open", s.OpenFollowups))
	row("reminders", fmt.Sprintf("%d due", s.DueReminders))
	row("health", health)
	row("signals", sig)
	row("brief", orNone(s.BriefPath))
	return b.String()
}

// Path returns the on-disk path for the active handoff document.
func Path(root string) string {
	return filepath.Join(root, ".claude", "project", "handoff.md")
}

// Status returns:
//
//	1  — no active doc present
//	0  — active doc present, head matches live HEAD
//	2  — active doc present, head differs from live HEAD
func Status(root string) int {
	data, err := os.ReadFile(Path(root))
	if os.IsNotExist(err) {
		return 1
	}
	if err != nil {
		return 1
	}
	meta, _ := fm.Parse(string(data))
	recorded := meta["head"]

	live, err := gitHead(root)
	if err != nil {
		// outside a repo or git absent — can't compare
		return 1
	}
	if recorded == live {
		return 0
	}
	return 2
}

// Archive moves the active handoff doc to .claude/project/handoffs/<id>.md,
// sets its frontmatter status to consumed, and returns the assigned id.
// id is the next %03d integer over existing files in the handoffs dir.
func Archive(root string) (string, error) {
	src := Path(root)
	data, err := os.ReadFile(src)
	if err != nil {
		return "", fmt.Errorf("handoff Archive: read active: %w", err)
	}

	archiveDir := filepath.Join(root, ".claude", "project", "handoffs")
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		return "", fmt.Errorf("handoff Archive: mkdir: %w", err)
	}

	id := nextArchiveID(archiveDir)

	// rewrite frontmatter with status: consumed
	meta, body := fm.Parse(string(data))
	meta["status"] = "consumed"
	order := []string{"mode", "created", "branch", "head", "health", "status"}
	updated := fm.Render(order, meta, body)

	dst := filepath.Join(archiveDir, id+".md")
	if err := os.WriteFile(dst, []byte(updated), 0o644); err != nil {
		return "", fmt.Errorf("handoff Archive: write archive: %w", err)
	}

	if err := os.Remove(src); err != nil {
		return "", fmt.Errorf("handoff Archive: remove active: %w", err)
	}

	return id, nil
}

// nextArchiveID returns the next %03d id over existing NNN.md files in dir.
// Mirrors internal/followups nextID.
func nextArchiveID(dir string) string {
	max := 0
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		var n int
		if _, err := fmt.Sscanf(strings.TrimSuffix(e.Name(), ".md"), "%d", &n); err == nil && n > max {
			max = n
		}
	}
	return fmt.Sprintf("%03d", max+1)
}

// ---- git helpers ----
// Each helper runs a git sub-command in dir and returns the result.
// On any error (non-repo, git absent, etc.) they return ("", nil) or (false, nil)
// so callers can treat them as zero values without hard-failing.

func gitBranch(dir string) (string, error) {
	out, err := gitRun(dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", nil
	}
	return strings.TrimSpace(out), nil
}

func gitHead(dir string) (string, error) {
	out, err := gitRun(dir, "rev-parse", "--short", "HEAD")
	if err != nil {
		return "", nil
	}
	return strings.TrimSpace(out), nil
}

func gitDirty(dir string) (bool, error) {
	out, err := gitRun(dir, "status", "--porcelain")
	if err != nil {
		return false, nil
	}
	return strings.TrimSpace(out) != "", nil
}

func gitStaged(dir string) ([]string, error) {
	out, err := gitRun(dir, "diff", "--name-only", "--cached")
	if err != nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(out)
	if trimmed == "" {
		return nil, nil
	}
	return strings.Split(trimmed, "\n"), nil
}

func gitLastCommit(dir string) (string, error) {
	out, err := gitRun(dir, "log", "-1", "--pretty=%s")
	if err != nil {
		return "", nil
	}
	return strings.TrimSpace(out), nil
}

// gitRun executes git with args in dir and returns stdout.
// Returns ("", error) on failure.
func gitRun(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
