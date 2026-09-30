package main

import (
	"fmt"
	"os"

	"apexclaude/internal/proj"
	"apexclaude/internal/signals"
)

func init() {
	register("signals", "scan/show/stale deterministic project signals", runSignals)
}

func runSignals(args []string) int {
	root := proj.Root()
	sub := ""
	if len(args) > 0 {
		sub = args[0]
	}
	switch sub {
	case "scan":
		p, err := signals.Scan(root)
		if err != nil {
			fmt.Fprintln(os.Stderr, "signals scan:", err)
			return 2
		}
		fmt.Println("wrote", p)
		return 0
	case "show":
		s, err := signals.Show(root)
		if err != nil {
			fmt.Fprintln(os.Stderr, "signals show:", err)
			return 1
		}
		fmt.Print(s)
		return 0
	case "stale":
		code, reason := signals.Stale(root)
		if code == 0 {
			fmt.Println(reason)
		} else {
			fmt.Fprintln(os.Stderr, reason)
		}
		return code
	default:
		fmt.Fprintln(os.Stderr, "usage: apex signals <scan|show|stale>")
		return 2
	}
}
