package main

import (
	"testing"

	"apexclaude/internal/followups"
)

// addEntry runs `apex followups add <args>` against a fresh repo root and
// returns the exit code and the entries on disk afterwards.
func addEntry(t *testing.T, args ...string) (int, []followups.Entry) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("APEX_REPO", root)
	code := runFollowups(append([]string{"add"}, args...))
	list, err := followups.List(root)
	if err != nil {
		t.Fatal(err)
	}
	return code, list
}

func TestFollowupsAddPositionalStillWorks(t *testing.T) {
	code, list := addEntry(t, "fix the thing", "plan", "risk", "review", "pass")
	if code != 0 || len(list) != 1 {
		t.Fatalf("code=%d entries=%d, want 0 and 1", code, len(list))
	}
	e := list[0]
	if e.Title != "fix the thing" || e.Kind != "plan" || e.Severity != "risk" || e.Origin != "review pass" {
		t.Errorf("got %+v", e)
	}
}

func TestFollowupsAddFlags(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"space-separated", []string{"CI is ubuntu-only", "--kind", "finding", "--severity", "nit", "--origin", "cli review"}},
		{"equals", []string{"--kind=finding", "--severity=nit", "--origin=cli review", "CI is ubuntu-only"}},
		{"flags before title", []string{"--severity", "nit", "--kind", "finding", "CI is ubuntu-only", "--origin", "cli review"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			code, list := addEntry(t, tc.args...)
			if code != 0 || len(list) != 1 {
				t.Fatalf("code=%d entries=%d, want 0 and 1", code, len(list))
			}
			e := list[0]
			if e.Title != "CI is ubuntu-only" || e.Kind != "finding" || e.Severity != "nit" || e.Origin != "cli review" {
				t.Errorf("got %+v", e)
			}
		})
	}
}

// The exact invocation that produced the malformed 008 on master: flag-style
// args parsed positionally put "--kind" in the title and "--severity" in the
// severity. Anything that looks like a flag and is not a known one must be
// refused, never stored.
func TestFollowupsAddRejectsFlagLikeGarbage(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"unknown flag", []string{"a title", "--sev", "nit"}},
		{"flag missing value", []string{"a title", "--kind"}},
		{"flag as title", []string{"--kind"}},
		{"dash title", []string{"-x"}},
		{"bad kind", []string{"a title", "--kind", "bogus"}},
		{"no title", []string{"--kind", "plan"}},
		{"two titles", []string{"one", "--kind", "plan", "two", "three", "four", "five"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			code, list := addEntry(t, tc.args...)
			if code != 2 {
				t.Errorf("exit = %d, want 2 (usage)", code)
			}
			if len(list) != 0 {
				t.Errorf("wrote %d entries on a usage error: %+v", len(list), list)
			}
		})
	}
}
