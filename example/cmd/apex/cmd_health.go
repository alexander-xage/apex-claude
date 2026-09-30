package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"apexclaude/internal/health"
	"apexclaude/internal/proj"
)

func init() {
	register("health", "show/set the repo health/integrity score", runHealth)
}

func runHealth(args []string) int {
	root := proj.Root()
	sub := "show"
	if len(args) > 0 {
		sub = args[0]
	}
	switch sub {
	case "show":
		score, body, err := health.Show(root)
		if err != nil {
			fmt.Fprintln(os.Stderr, "health:", err)
			return 2
		}
		if score < 0 {
			fmt.Println("health: unset (no reviews recorded)")
			return 0
		}
		fmt.Print(body)
		return 0
	case "set":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: apex health set <0-100> [note]")
			return 2
		}
		score, err := strconv.Atoi(args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, "health set: score must be an integer")
			return 2
		}
		if err := health.Set(root, score, strings.Join(args[2:], " ")); err != nil {
			fmt.Fprintln(os.Stderr, "health set:", err)
			return 2
		}
		fmt.Printf("health set to %d/100\n", score)
		return 0
	default:
		fmt.Fprintln(os.Stderr, "usage: apex health <show|set>")
		return 2
	}
}
