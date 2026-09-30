package main

import (
	"fmt"
	"os"

	"apexclaude/internal/handoff"
	"apexclaude/internal/proj"
)

func init() {
	register("handoff", "report state / route staleness / archive a handoff", runHandoff)
}

func runHandoff(args []string) int {
	root := proj.Root()
	sub := ""
	if len(args) > 0 {
		sub = args[0]
	}
	switch sub {
	case "scan":
		// Read-only: report the deterministic facts. Composing handoff.md from
		// this report is the model's job, so scan takes no mode and writes
		// nothing — it is safe to run while a doc is being consumed.
		if len(args) > 1 {
			fmt.Fprintln(os.Stderr, "usage: apex handoff scan")
			return 2
		}
		s, err := handoff.Scan(root)
		if err != nil {
			fmt.Fprintln(os.Stderr, "handoff scan:", err)
			return 1
		}
		fmt.Print(handoff.Report(s))
		return 0

	case "status":
		code := handoff.Status(root)
		switch code {
		case 0:
			fmt.Println("handoff present, fresh")
		case 2:
			fmt.Println("handoff present, STALE (HEAD moved)")
		default:
			fmt.Println("no active handoff")
		}
		return code

	case "archive":
		id, err := handoff.Archive(root)
		if err != nil {
			fmt.Fprintln(os.Stderr, "handoff archive:", err)
			return 1
		}
		fmt.Printf("archived to .claude/project/handoffs/%s.md\n", id)
		return 0

	default:
		fmt.Fprintln(os.Stderr, "usage: apex handoff <scan|status|archive>")
		return 2
	}
}
