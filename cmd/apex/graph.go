package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"syscall"

	"apex/internal/repo"
)

func graph(dir string, args []string, stdout, stderr io.Writer) int {
	bin, err := exec.LookPath("graphify")
	if err != nil {
		fmt.Fprintln(stderr, "apex graph: graphify is not on PATH; install it with `uv tool install graphifyy`")
		return 2
	}
	out, err := graphOut(dir)
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

// graphOut returns this checkout's graph directory, pruning stale graphs and excluding .claude/ on the way.
func graphOut(dir string) (string, error) {
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
	if err := os.MkdirAll(out, 0o755); err != nil {
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
