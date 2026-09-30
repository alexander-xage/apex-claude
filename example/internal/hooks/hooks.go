// Package hooks implements the SessionStart hook payload: it surfaces project
// nudges (stale signals, and later due reminders) as additionalContext. Emits
// nothing when there is nothing pending.
package hooks

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"apexclaude/internal/layout"
	"apexclaude/internal/proj"
	"apexclaude/internal/reminder"
	"apexclaude/internal/signals"
	"apexclaude/internal/update"
	"apexclaude/internal/version"
)

// spawnCheck starts a detached `apex update check --quiet` and returns
// immediately without waiting on it: Start (not Run) + Process.Release, nil
// stdio. Best-effort — a resolution or spawn failure is silently swallowed,
// matching the hook's zero-network, always-return-0 contract. A package-level
// var so tests can substitute a no-op and assert whether it was invoked
// instead of actually spawning a process.
var spawnCheck = func() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	cmd := exec.Command(exe, "update", "check", "--quiet")
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return
	}
	_ = cmd.Process.Release()
}

// SessionStart writes a SessionStart hook payload to w. Returns 0 always (a hook
// failure should never block a session). No output when nothing is pending.
func SessionStart(args []string, w io.Writer) int {
	root := proj.Root()

	// Best-effort Windows .old sweep — reuses internal/update's sweep
	// (exported for this exact purpose) rather than duplicating it here.
	update.RemoveStaleWindowsOld(layout.ArtifactRoot())

	var nudges []string

	if os.Getenv("APEX_NO_UPDATE_CHECK") == "" {
		cache := update.ReadCache(update.CachePath())
		if update.ValidTag(cache.Latest) && update.Compare(cache.Latest, "v"+version.Version) > 0 {
			nudges = append(nudges, fmt.Sprintf("Apex update available: v%s → %s — run 'apex update'", version.Version, cache.Latest))
		}
		// update.Stale treats a never-populated (missing) cache as stale too,
		// so a single check covers both the "no cache yet" and "cache older
		// than TTL" cases the brief calls out separately.
		if update.Stale(cache) {
			spawnCheck()
		}
	}

	if code, reason := signals.Stale(root); code == 1 {
		nudges = append(nudges, "Project signals stale: "+reason)
	} else if code != 0 {
		nudges = append(nudges, "Project signals check failed: "+reason)
	}
	if due, _ := reminder.Due(root, time.Now()); len(due) > 0 {
		for _, r := range due {
			nudges = append(nudges, "Reminder due: "+r.Text)
		}
	}

	if len(nudges) == 0 {
		return 0
	}

	ctx := "Apex session start:\n- " + strings.Join(nudges, "\n- ")
	out := map[string]any{
		"hookSpecificOutput": map[string]any{
			"hookEventName":     "SessionStart",
			"additionalContext": ctx,
		},
	}
	b, _ := json.Marshal(out)
	fmt.Fprintln(w, string(b))
	return 0
}
