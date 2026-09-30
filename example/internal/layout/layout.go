// Package layout resolves and inspects the Apex artifact root: where the
// commands/agents/skills/output-styles live and how hooks are wired for the
// current install (dev/plugin checkout vs. loose ~/.claude install).
package layout

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// ArtifactRoot prefers $CLAUDE_PLUGIN_ROOT, else infers the root as the parent
// of the binary's bin/ directory (bin/apex -> root). For a loose install that
// resolves to ~/.claude; for the repo it resolves to the repo root. A binary
// installed elsewhere on PATH (e.g. /usr/local/bin) infers a dir that carries
// no artifacts — verify the inference and fall back to ~/.claude, then the
// working directory.
func ArtifactRoot() string {
	if r := os.Getenv("CLAUDE_PLUGIN_ROOT"); r != "" {
		return r
	}
	if exe, err := os.Executable(); err == nil {
		if root := filepath.Dir(filepath.Dir(exe)); LooksLikeArtifactRoot(root) {
			return root
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		if root := filepath.Join(home, ".claude"); LooksLikeArtifactRoot(root) {
			return root
		}
	}
	wd, _ := os.Getwd()
	return wd
}

// LooksLikeArtifactRoot reports whether dir carries the Apex artifact surface:
// a plugin manifest (dev layout) or a commands/ dir (both layouts ship one).
func LooksLikeArtifactRoot(dir string) bool {
	for _, p := range []string{".claude-plugin", "commands"} {
		if fi, err := os.Stat(filepath.Join(dir, p)); err == nil && fi.IsDir() {
			return true
		}
	}
	return false
}

// IsLooseInstall reports whether root looks like a loose ~/.claude install
// rather than the repo / a plugin dir. The discriminator is the absence of a
// .claude-plugin manifest dir: install.sh never copies it.
func IsLooseInstall(root string) bool {
	if _, err := os.Stat(filepath.Join(root, ".claude-plugin")); err == nil {
		return false
	}
	return true
}

// isApexHookCmd reports whether a hook command invokes `apex hooks`. The binary
// is `apex` on Unix and `apex.exe` on Windows, so match both — keying on the
// bare `apex hooks` substring would false-negative on every Windows install.
func isApexHookCmd(cmd string) bool {
	return strings.Contains(cmd, "apex hooks") || strings.Contains(cmd, "apex.exe hooks")
}

// ApexHooksWired reports whether settings.json under root wires at least one
// apex hook (matching the `apex hooks` command the installer writes).
func ApexHooksWired(root string) bool {
	b, err := os.ReadFile(filepath.Join(root, "settings.json"))
	if err != nil {
		return false
	}
	var cfg struct {
		Hooks map[string][]struct {
			Hooks []struct {
				Command string `json:"command"`
			} `json:"hooks"`
		} `json:"hooks"`
	}
	// Windows PowerShell 5.1 writes UTF-8 with a BOM, which encoding/json
	// rejects outright; older install.ps1 runs left exactly that behind.
	b = bytes.TrimPrefix(b, []byte("\xEF\xBB\xBF"))
	if json.Unmarshal(b, &cfg) != nil {
		return false
	}
	for _, groups := range cfg.Hooks {
		for _, g := range groups {
			for _, h := range g.Hooks {
				if isApexHookCmd(h.Command) {
					return true
				}
			}
		}
	}
	return false
}
