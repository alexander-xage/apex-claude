package hooks

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"apexclaude/internal/update"
	"apexclaude/internal/version"
)

// isolateCache redirects os.UserCacheDir to a fresh temp dir so tests never
// touch the real machine's update-check.json (mirrors cmd/apex's helper).
func isolateCache(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("LocalAppData", dir)   // windows: os.UserCacheDir()
	t.Setenv("XDG_CACHE_HOME", dir) // linux: os.UserCacheDir()
	t.Setenv("HOME", dir)           // darwin fallback: os.UserCacheDir()
}

// noSpawn swaps spawnCheck for a no-op that records whether it was invoked,
// restoring the original on cleanup. Tests use this to assert the detached
// refresh was (or wasn't) attempted without actually spawning a process.
func noSpawn(t *testing.T) *bool {
	t.Helper()
	called := false
	orig := spawnCheck
	spawnCheck = func() { called = true }
	t.Cleanup(func() { spawnCheck = orig })
	return &called
}

// A missing signals file is code 1 (stale) — the nudge must surface it.
func TestSessionStartSurfacesStaleSignals(t *testing.T) {
	// Without this the real spawnCheck runs: os.Executable() is the test
	// binary, so it re-execs hooks.test, which ignores the unknown args and
	// re-runs this whole suite — forking exponentially until the machine is
	// out of processes. Every test touching SessionStart must substitute it.
	noSpawn(t)
	root := t.TempDir()
	t.Setenv("APEX_REPO", root)

	var buf bytes.Buffer
	if code := SessionStart(nil, &buf); code != 0 {
		t.Fatalf("hook must never fail the session, got %d", code)
	}
	if !strings.Contains(buf.String(), "Project signals stale") {
		t.Errorf("expected stale-signals nudge\n%s", buf.String())
	}
}

// A signals file that can't be read is code 2 (error) — that must be surfaced
// too, not swallowed. A directory at the file's path forces the read error.
func TestSessionStartSurfacesSignalsError(t *testing.T) {
	// Without this the real spawnCheck runs: os.Executable() is the test
	// binary, so it re-execs hooks.test, which ignores the unknown args and
	// re-runs this whole suite — forking exponentially until the machine is
	// out of processes. Every test touching SessionStart must substitute it.
	noSpawn(t)
	root := t.TempDir()
	t.Setenv("APEX_REPO", root)
	if err := os.MkdirAll(filepath.Join(root, ".claude", "project", "deterministic-signals.md"), 0o755); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if code := SessionStart(nil, &buf); code != 0 {
		t.Fatalf("hook must never fail the session, got %d", code)
	}
	if !strings.Contains(buf.String(), "Project signals check failed") {
		t.Errorf("expected signals-error nudge\n%s", buf.String())
	}
}

// --- update nudge (checkpoint 5) ----------------------------------------

func TestSessionStartNudgesOnNewerCachedVersion(t *testing.T) {
	root := t.TempDir()
	t.Setenv("APEX_REPO", root)
	isolateCache(t)
	noSpawn(t)

	if err := update.WriteCache(update.CachePath(), update.Cache{
		CheckedAt: time.Now().UTC().Format(time.RFC3339),
		Latest:    "v9.9.9",
	}); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if code := SessionStart(nil, &buf); code != 0 {
		t.Fatalf("hook must never fail the session, got %d", code)
	}
	if !strings.Contains(buf.String(), "Apex update available: v"+version.Version+" → v9.9.9 — run 'apex update'") {
		t.Errorf("expected update nudge\n%s", buf.String())
	}
}

func TestSessionStartNoNudgeOnEqualOrOlderCachedVersion(t *testing.T) {
	// Equal to the running version, and strictly older.
	for _, latest := range []string{"v" + version.Version, "v0.0.1"} {
		t.Run(latest, func(t *testing.T) {
			root := t.TempDir()
			t.Setenv("APEX_REPO", root)
			isolateCache(t)
			noSpawn(t)

			if err := update.WriteCache(update.CachePath(), update.Cache{
				CheckedAt: time.Now().UTC().Format(time.RFC3339),
				Latest:    latest,
			}); err != nil {
				t.Fatal(err)
			}

			var buf bytes.Buffer
			if code := SessionStart(nil, &buf); code != 0 {
				t.Fatalf("hook must never fail the session, got %d", code)
			}
			if strings.Contains(buf.String(), "Apex update available") {
				t.Errorf("expected no update nudge\n%s", buf.String())
			}
		})
	}
}

func TestSessionStartToleratesMalformedCachedLatest(t *testing.T) {
	for _, latest := range []string{"banana", "v1.2", ""} {
		t.Run(latest, func(t *testing.T) {
			root := t.TempDir()
			t.Setenv("APEX_REPO", root)
			isolateCache(t)
			noSpawn(t)

			if err := update.WriteCache(update.CachePath(), update.Cache{
				CheckedAt: time.Now().UTC().Format(time.RFC3339),
				Latest:    latest,
			}); err != nil {
				t.Fatal(err)
			}

			var buf bytes.Buffer
			if code := SessionStart(nil, &buf); code != 0 {
				t.Fatalf("hook must never fail the session, got %d", code)
			}
			if strings.Contains(buf.String(), "Apex update available") {
				t.Errorf("malformed cached latest must not nudge\n%s", buf.String())
			}
		})
	}
}

func TestSessionStartOptOutSkipsNudgeAndSpawn(t *testing.T) {
	root := t.TempDir()
	t.Setenv("APEX_REPO", root)
	t.Setenv("APEX_NO_UPDATE_CHECK", "1")
	isolateCache(t)
	called := noSpawn(t)

	if err := update.WriteCache(update.CachePath(), update.Cache{
		CheckedAt: time.Now().UTC().Format(time.RFC3339),
		Latest:    "v9.9.9",
	}); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if code := SessionStart(nil, &buf); code != 0 {
		t.Fatalf("hook must never fail the session, got %d", code)
	}
	if strings.Contains(buf.String(), "Apex update available") {
		t.Errorf("opt-out must suppress the nudge\n%s", buf.String())
	}
	if *called {
		t.Errorf("opt-out must suppress the detached spawn")
	}
}

func TestSessionStartSpawnsOnStaleOrMissingCache(t *testing.T) {
	t.Run("missing cache", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("APEX_REPO", root)
		isolateCache(t)
		called := noSpawn(t)

		var buf bytes.Buffer
		SessionStart(nil, &buf)
		if !*called {
			t.Error("expected spawn attempt on missing cache")
		}
	})

	t.Run("stale cache", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("APEX_REPO", root)
		isolateCache(t)
		called := noSpawn(t)

		if err := update.WriteCache(update.CachePath(), update.Cache{
			CheckedAt: time.Now().UTC().Add(-48 * time.Hour).Format(time.RFC3339),
			Latest:    "v" + version.Version,
		}); err != nil {
			t.Fatal(err)
		}

		var buf bytes.Buffer
		SessionStart(nil, &buf)
		if !*called {
			t.Error("expected spawn attempt on stale cache")
		}
	})
}

func TestSessionStartNoSpawnOnFreshCache(t *testing.T) {
	root := t.TempDir()
	t.Setenv("APEX_REPO", root)
	isolateCache(t)
	called := noSpawn(t)

	if err := update.WriteCache(update.CachePath(), update.Cache{
		CheckedAt: time.Now().UTC().Format(time.RFC3339),
		Latest:    "v" + version.Version,
	}); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	SessionStart(nil, &buf)
	if *called {
		t.Error("fresh cache must not trigger a spawn")
	}
}
