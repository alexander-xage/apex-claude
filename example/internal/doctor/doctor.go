// Package doctor runs deterministic integrity checks on the Apex artifact layout
// and project state. Exit 1 if any hard check fails — a command can branch on it.
//
// Apex ships in two layouts and doctor checks the right contract for each:
//   - dev/plugin layout: the repo (or a marketplace plugin dir). Carries a
//     .claude-plugin/plugin.json manifest and hooks/hooks.json.
//   - loose install: ~/.claude, populated by scripts/install.sh. There is NO
//     plugin manifest and NO hooks/hooks.json — hooks are wired into
//     settings.json — so those checks are skipped and replaced by a
//     settings.json hook-wiring check.
package doctor

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"apexclaude/internal/layout"
	"apexclaude/internal/proj"
	"apexclaude/internal/signals"
)

// Run executes the checks, writing a report to w. Returns 0 if all pass, else 1.
func Run(w io.Writer) int {
	root := layout.ArtifactRoot()
	loose := layout.IsLooseInstall(root)
	layoutLabel := "dev/plugin layout"
	if loose {
		layoutLabel = "loose install"
	}
	fmt.Fprintf(w, "artifact root: %s (%s)\n\n", root, layoutLabel)

	ok := true
	check := func(label string, pass bool) {
		mark := "✓"
		if !pass {
			mark = "✗"
			ok = false
		}
		fmt.Fprintf(w, "  %s %s\n", mark, label)
	}

	if loose {
		// Loose model owns a different contract: no manifest, hooks live in
		// settings.json. Validate the wiring the installer is responsible for.
		check("apex hooks wired in settings.json", layout.ApexHooksWired(root))
	} else {
		check("plugin.json is valid JSON", validJSON(filepath.Join(root, ".claude-plugin", "plugin.json")))
		check("hooks/hooks.json is valid JSON", validJSON(filepath.Join(root, "hooks", "hooks.json")))
		// bin/ is gitignored, so a fresh clone or a marketplace plugin install
		// has hooks.json pointing at a binary that was never built. Left
		// unchecked that surfaces as a hook failing every session; make it a
		// loud, deterministic failure here instead.
		if missing, ok := hookTargetsPresent(root); !ok {
			check("hooks.json targets exist (missing: "+strings.Join(missing, ", ")+") — run `make build`", false)
		} else {
			check("hooks.json targets exist", true)
		}
	}
	check("output-styles/ has a style", countGlob(root, "output-styles", "*.md") >= 1)
	check("agents/ has an agent", countGlob(root, "agents", "*.md") >= 1)
	check("commands/ has a command", countGlob(root, "commands", "*.md") >= 1)
	check("skills/ has a SKILL.md", countSkills(root) >= 1)

	// PATH reachability is info, not a hard failure: hooks invoke the binary by
	// absolute path, so Apex works regardless. It only matters for running `apex`
	// by hand. Warn when the binary's own directory is absent from $PATH.
	if dir, on := binOnPath(); !on {
		fmt.Fprintf(w, "  • PATH: %s is not on $PATH — add it to run `apex` directly\n", dir)
	} else {
		check("binary dir on $PATH", true)
	}

	// Project-state is reported as info, not a hard failure of the artifacts.
	if code, reason := signals.Stale(proj.Root()); code == 0 {
		check("project signals fresh", true)
	} else {
		fmt.Fprintf(w, "  • signals: %s\n", reason)
	}

	fmt.Fprintln(w)
	if ok {
		fmt.Fprintln(w, "doctor: all checks passed")
		return 0
	}
	fmt.Fprintln(w, "doctor: failures above")
	return 1
}

// binOnPath reports the directory holding the running binary and whether that
// directory appears in $PATH. Returns ("", true) if the executable can't be
// resolved — we don't warn on what we can't measure.
func binOnPath() (string, bool) {
	exe, err := os.Executable()
	if err != nil {
		return "", true
	}
	dir := filepath.Dir(exe)
	return dir, dirOnPath(dir, os.Getenv("PATH"))
}

// dirOnPath reports whether dir appears in the PATH-style list. Entries are
// compared after cleaning (trailing separators) and symlink resolution, and
// case-insensitively on Windows — a raw string compare false-negatives on
// `C:\x\bin\` vs `C:\x\bin` and on symlinked bin dirs.
func dirOnPath(dir, pathEnv string) bool {
	want := canonPath(dir)
	for _, p := range filepath.SplitList(pathEnv) {
		if p == "" {
			continue
		}
		if pathsEqual(canonPath(p), want) {
			return true
		}
	}
	return false
}

// canonPath cleans p and resolves symlinks when the path exists; a path that
// can't be resolved compares by its cleaned form.
func canonPath(p string) string {
	p = filepath.Clean(p)
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	return p
}

func pathsEqual(a, b string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// hookTargetsPresent resolves every command in hooks/hooks.json to a path and
// reports which of those binaries are absent. ${CLAUDE_PLUGIN_ROOT} expands to
// root; the command's first whitespace-separated token is the binary. Windows
// installs name it apex.exe, so either spelling satisfies the check.
func hookTargetsPresent(root string) (missing []string, ok bool) {
	b, err := os.ReadFile(filepath.Join(root, "hooks", "hooks.json"))
	if err != nil {
		return nil, true // validity is the neighbouring check's job
	}
	var cfg struct {
		Hooks map[string][]struct {
			Hooks []struct {
				Command string `json:"command"`
			} `json:"hooks"`
		} `json:"hooks"`
	}
	if json.Unmarshal(b, &cfg) != nil {
		return nil, true
	}
	seen := map[string]bool{}
	for _, groups := range cfg.Hooks {
		for _, g := range groups {
			for _, h := range g.Hooks {
				fields := strings.Fields(h.Command)
				if len(fields) == 0 {
					continue
				}
				bin := strings.ReplaceAll(fields[0], "${CLAUDE_PLUGIN_ROOT}", root)
				bin = filepath.FromSlash(bin)
				if seen[bin] {
					continue
				}
				seen[bin] = true
				if !fileExists(bin) && !fileExists(bin+".exe") {
					missing = append(missing, filepath.Base(bin))
				}
			}
		}
	}
	sort.Strings(missing)
	return missing, len(missing) == 0
}

func fileExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir()
}

func validJSON(p string) bool {
	b, err := os.ReadFile(p)
	if err != nil {
		return false
	}
	return json.Valid(b)
}

func countGlob(root, dir, pattern string) int {
	m, _ := filepath.Glob(filepath.Join(root, dir, pattern))
	return len(m)
}

func countSkills(root string) int {
	m, _ := filepath.Glob(filepath.Join(root, "skills", "*", "SKILL.md"))
	return len(m)
}
