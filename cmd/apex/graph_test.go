package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// 7 is no apex exit code, so seeing it proves graphify's code passed through.
func fakeGraphify(t *testing.T) {
	t.Helper()
	fakeGraphifyScript(t, "#!/bin/sh\necho \"OUT=$GRAPHIFY_OUT\"\nprintf 'ARG[%s]\\n' \"$@\"\nexit 7\n")
}

func fakeGraphifyScript(t *testing.T, script string) {
	t.Helper()
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "graphify"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func gitRepo(t *testing.T) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	gitRun(t, r, "init", "-q")
	gitRun(t, r, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "--allow-empty", "-m", "i")
	return r
}

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v %s", args, err, out)
	}
}

func TestGraphPassesArgsAndSetsOut(t *testing.T) {
	fakeGraphify(t)
	r := gitRepo(t)
	code, out, _ := apex(t, r, "graph", "query", "x y", "--budget", "5", "-h", "--agent", "dev", "--")
	if code != 7 {
		t.Fatalf("exit %d, want graphify's 7", code)
	}
	want := filepath.Join(r, ".claude", "graphify", "main")
	args := "ARG[query]\nARG[x y]\nARG[--budget]\nARG[5]\nARG[-h]\nARG[--agent]\nARG[dev]\nARG[--]\n"
	if !strings.Contains(out, "OUT="+want+"\n") || !strings.Contains(out, args) {
		t.Fatalf("output:\n%s\nwant OUT=%s and args unchanged", out, want)
	}
	if fi, err := os.Stat(want); err != nil || !fi.IsDir() {
		t.Fatalf("graph dir not created: %v", err)
	}
}

func TestGraphFromWorktreeUsesItsOwnDir(t *testing.T) {
	fakeGraphify(t)
	r := gitRepo(t)
	wt := filepath.Join(r, ".claude", "worktrees", "feat")
	gitRun(t, r, "worktree", "add", "-q", "-b", "feat", wt)
	_, out, _ := apex(t, wt, "graph", "update", ".")
	if want := "OUT=" + filepath.Join(r, ".claude", "graphify", "feat") + "\n"; !strings.Contains(out, want) {
		t.Fatalf("output:\n%s\nwant %s", out, want)
	}
}

func TestGraphPrunesGraphsOfRemovedWorktrees(t *testing.T) {
	fakeGraphify(t)
	r := gitRepo(t)
	keep, gone := filepath.Join(t.TempDir(), "keep"), filepath.Join(t.TempDir(), "gone")
	gitRun(t, r, "worktree", "add", "-q", "-b", "a", keep)
	gitRun(t, r, "worktree", "add", "-q", "-b", "b", gone)
	graphs := filepath.Join(r, ".claude", "graphify")
	for _, d := range []string{"main", "keep", "gone", "stray"} {
		if err := os.MkdirAll(filepath.Join(graphs, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.RemoveAll(gone); err != nil {
		t.Fatal(err)
	}
	apex(t, r, "graph", "update", ".")
	entries, _ := os.ReadDir(graphs)
	var got []string
	for _, e := range entries {
		got = append(got, e.Name())
	}
	if strings.Join(got, ",") != "keep,main" {
		t.Fatalf("graphs after prune = %v, want [keep main]", got)
	}
}

func TestGraphWithoutGraphifyExits2(t *testing.T) {
	r := gitRepo(t)
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	if err := os.Symlink(gitPath, filepath.Join(bin, "git")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	if code, _, errs := apex(t, r, "graph", "update", "."); code != 2 || !strings.Contains(errs, "graphify") {
		t.Fatalf("exit %d (%s), want 2 naming graphify", code, errs)
	}
	if _, err := os.Stat(filepath.Join(r, ".claude")); !os.IsNotExist(err) {
		t.Fatalf("a failed lookup left .claude/ behind: %v", err)
	}
}

// A signal-killed graphify must not look like an apex environment error (2); the shell convention is 128+signal.
func TestGraphSignalExit(t *testing.T) {
	fakeGraphifyScript(t, "#!/bin/sh\nkill -TERM $$\n")
	r := gitRepo(t)
	if code, _, _ := apex(t, r, "graph", "update", "."); code != 128+15 {
		t.Fatalf("exit %d, want 143", code)
	}
}

// graphify must see the exclusion before it scans, and apex must not leave an untracked file in the checkout (that
// would block `git worktree remove`).
func TestGraphExcludesClaudeViaBuildConfig(t *testing.T) {
	fakeGraphifyScript(t, "#!/bin/sh\ncat \"$GRAPHIFY_OUT/.graphify_build.json\"\n")
	r := gitRepo(t)
	out := filepath.Join(r, ".claude", "graphify", "main")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, ".graphify_build.json"), []byte(`{"excludes": ["vendor"], "gitignore": false}`), 0o644); err != nil {
		t.Fatal(err)
	}
	code, stdout, errs := apex(t, r, "graph", "update", ".")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errs)
	}
	for _, want := range []string{`"vendor"`, `".claude"`, `"gitignore":false`} {
		if !strings.Contains(stdout, want) {
			t.Errorf("build config seen by graphify = %s; missing %s", stdout, want)
		}
	}
	if out, _ := exec.Command("git", "-C", r, "status", "--porcelain", "--untracked-files=all", "--", ":!.claude/graphify").Output(); len(out) != 0 {
		t.Errorf("apex graph left files in the checkout:\n%s", out)
	}
	code, _, _ = apex(t, r, "graph", "update", ".")
	data, _ := os.ReadFile(filepath.Join(out, ".graphify_build.json"))
	if code != 0 || strings.Count(string(data), ".claude") != 1 {
		t.Errorf("second run: exit %d, config %s; want .claude listed once", code, data)
	}
}

// graphify ignores a config it cannot parse, which would silently drop the exclusion.
func TestGraphRejectsCorruptBuildConfig(t *testing.T) {
	fakeGraphify(t)
	r := gitRepo(t)
	out := filepath.Join(r, ".claude", "graphify", "main")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"{not json", "null", "[]", `"x"`} {
		if err := os.WriteFile(filepath.Join(out, ".graphify_build.json"), []byte(bad), 0o644); err != nil {
			t.Fatal(err)
		}
		if code, _, errs := apex(t, r, "graph", "update", "."); code != 2 || !strings.Contains(errs, "JSON object") {
			t.Errorf("config %s: exit %d (%s), want 2 naming a JSON object", bad, code, errs)
		}
	}
}
