package main

import (
	"fmt"
	"os"

	"apexclaude/internal/docs"
	"apexclaude/internal/proj"
)

func init() {
	register("docs", "scan/stale documentation surfaces", runDocs)
}

func runDocs(args []string) int {
	root := proj.Root()
	sub := ""
	if len(args) > 0 {
		sub = args[0]
	}
	switch sub {
	case "scan":
		p, err := docs.Scan(root)
		if err != nil {
			fmt.Fprintln(os.Stderr, "docs scan:", err)
			return 2
		}
		fmt.Println("wrote", p)
		return 0
	case "stale":
		code, reason := docs.Stale(root)
		if code == 0 {
			fmt.Println(reason)
		} else {
			fmt.Fprintln(os.Stderr, reason)
		}
		return code
	default:
		fmt.Fprintln(os.Stderr, "usage: apex docs <scan|stale>")
		return 2
	}
}
