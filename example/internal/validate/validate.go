// Package validate runs deterministic lints over plugin artifacts and spec docs.
// It checks structure (required frontmatter, valid JSON, spec change-log), not
// semantics — judgment stays with the model.
package validate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"apexclaude/internal/fm"
)

// Issue is one lint finding.
type Issue struct {
	File, Msg string
}

func glob(root, dir, pattern string) []string {
	m, _ := filepath.Glob(filepath.Join(root, dir, pattern))
	return m
}

func skillFiles(root string) []string {
	m, _ := filepath.Glob(filepath.Join(root, "skills", "*", "SKILL.md"))
	return m
}

func frontmatter(p string) map[string]string {
	data, err := os.ReadFile(p)
	if err != nil {
		return map[string]string{}
	}
	m, _ := fm.Parse(string(data))
	return m
}

func rel(root, p string) string {
	if r, err := filepath.Rel(root, p); err == nil {
		return r
	}
	return p
}

// Artifacts lints plugin artifact frontmatter and JSON validity.
func Artifacts(root string) []Issue {
	var issues []Issue
	require := func(files []string, keys ...string) {
		for _, p := range files {
			m := frontmatter(p)
			for _, k := range keys {
				if strings.TrimSpace(m[k]) == "" {
					issues = append(issues, Issue{rel(root, p), "missing frontmatter: " + k})
				}
			}
		}
	}
	require(glob(root, "agents", "*.md"), "name", "description")
	require(skillFiles(root), "name", "description")
	require(glob(root, "commands", "*.md"), "description")

	for _, p := range []string{
		filepath.Join(root, ".claude-plugin", "plugin.json"),
		filepath.Join(root, "hooks", "hooks.json"),
	} {
		b, err := os.ReadFile(p)
		switch {
		case err != nil:
			issues = append(issues, Issue{rel(root, p), "unreadable: " + err.Error()})
		case !json.Valid(b):
			issues = append(issues, Issue{rel(root, p), "invalid JSON"})
		}
	}
	return issues
}

// Spec lints spec docs for a title and a change-log section (spec-currency rule).
func Spec(root string, paths []string) []Issue {
	if len(paths) == 0 {
		paths = glob(root, "docs/spec", "*.md")
	}
	var issues []Issue
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			issues = append(issues, Issue{rel(root, p), "unreadable"})
			continue
		}
		s := string(data)
		if !strings.Contains(s, "# ") {
			issues = append(issues, Issue{rel(root, p), "no title heading"})
		}
		if !strings.Contains(s, "## Change log") {
			issues = append(issues, Issue{rel(root, p), "missing ## Change log section"})
		}
	}
	return issues
}
