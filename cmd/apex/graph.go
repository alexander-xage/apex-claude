package main

import (
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

// graphOut prunes stale graphs, then creates and returns this checkout's graph directory.
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
	return out, os.MkdirAll(out, 0o755)
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
