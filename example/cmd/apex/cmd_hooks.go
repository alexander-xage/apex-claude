package main

import (
	"fmt"
	"os"

	"apexclaude/internal/hooks"
)

func init() {
	register("hooks", "SessionStart context", runHooks)
}

func runHooks(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: apex hooks <session-start>")
		return 2
	}
	switch args[0] {
	case "session-start":
		return hooks.SessionStart(args[1:], os.Stdout)
	default:
		fmt.Fprintf(os.Stderr, "apex hooks: unknown subcommand %q\n", args[0])
		return 2
	}
}
