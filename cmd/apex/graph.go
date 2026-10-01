package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"syscall"

	"apex/internal/repo"
)

var graphName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

func graph(dir string, args []string, stdout, stderr io.Writer) int {
	// --name is apex's own flag, so it is read only as the first argument; everything after goes to graphify unchanged.
	name := ""
	if len(args) > 0 && args[0] == "--name" {
		if len(args) < 2 || !graphName.MatchString(args[1]) {
			fmt.Fprintln(stderr, "apex graph: --name takes a graph name of lowercase letters, digits and hyphens")
			return 64
		}
		name, args = args[1], args[2:]
	}
	bin, err := exec.LookPath("graphify")
	if err != nil {
		fmt.Fprintln(stderr, "apex graph: graphify is not on PATH; install it with `uv tool install graphifyy`")
		return 2
	}
	out, err := graphOut(dir, name)
	if err != nil {
		fmt.Fprintln(stderr, "apex graph:", err)
		return 2
	}
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GRAPHIFY_OUT="+out)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, stdout, stderr
	err = cmd.Run()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		if ws, ok := exit.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			return 128 + int(ws.Signal())
		}
		return exit.ExitCode()
	}
	if err != nil {
		fmt.Fprintln(stderr, "apex graph:", err)
		return 2
	}
	return 0
}

// graphOut returns this checkout's graph directory, pruning stale graphs, self-ignoring the graph directory and
// excluding .claude/ on the way. A named graph nests under graphs/ in the checkout's directory, so it is pruned with
// its checkout and cannot collide with the files graphify keeps beside the default graph.
func graphOut(dir, name string) (string, error) {
	main, err := repo.Main(dir)
	if err != nil {
		return "", err
	}
	checkout, err := repo.Checkout(dir, main)
	if err != nil {
		return "", err
	}
	graphs := filepath.Join(main, ".claude", "graphify")
	if err := pruneGraphs(dir, graphs); err != nil {
		return "", err
	}
	out := filepath.Join(graphs, checkout)
	if name != "" {
		out = filepath.Join(out, "graphs", name)
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return "", err
	}
	if err := selfIgnore(graphs); err != nil {
		return "", err
	}
	return out, excludeProtocolState(out)
}

// pruneGraphs deletes the graph of every worktree that no longer exists. It lists the graphs before the worktrees: a
// graph created after the worktree listing then cannot be mistaken for a stale one.
func pruneGraphs(dir, graphs string) error {
	entries, err := os.ReadDir(graphs)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	live, err := repo.LiveWorktrees(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() && e.Name() != "main" && !slices.Contains(live, e.Name()) {
			if err := os.RemoveAll(filepath.Join(graphs, e.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

// selfIgnore makes the graph directory hide itself from git, as tool caches do: graphs are rebuilt, never committed.
func selfIgnore(graphs string) error {
	f, err := os.OpenFile(filepath.Join(graphs, ".gitignore"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if errors.Is(err, os.ErrExist) {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = f.WriteString("*\n")
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}

// excludeProtocolState adds .claude to graphify's persisted excludes (docs/design/agents.md, Graphify discipline).
// ponytail: .graphify_build.json is graphify 0.9.x's internal format; if a release drops it, fall back to .graphifyignore.
func excludeProtocolState(out string) error {
	path := filepath.Join(out, ".graphify_build.json")
	cfg := map[string]any{}
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err == nil {
		if err := json.Unmarshal(data, &cfg); err != nil || cfg == nil {
			return fmt.Errorf("%s is not a JSON object, so graphify would ignore it", path)
		}
	}
	excludes, _ := cfg["excludes"].([]any)
	for _, e := range excludes {
		if e == ".claude" {
			return nil
		}
	}
	cfg["excludes"] = append(excludes, ".claude")
	data, err = json.Marshal(cfg)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(out, ".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	_, err = tmp.Write(data)
	if err == nil {
		err = tmp.Chmod(0o644)
	}
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
