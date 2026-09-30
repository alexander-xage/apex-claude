package main

import (
	"fmt"
	"os"

	"apexclaude/internal/proj"
	"apexclaude/internal/validate"
)

func init() {
	register("validate", "lint artifacts/specs (exit 1 on issues)", runValidate)
}

func runValidate(args []string) int {
	root := proj.Root()
	target := "artifacts"
	if len(args) > 0 {
		target = args[0]
	}
	var issues []validate.Issue
	switch target {
	case "artifacts":
		issues = validate.Artifacts(root)
	case "spec":
		issues = validate.Spec(root, args[1:])
	case "all":
		issues = append(validate.Artifacts(root), validate.Spec(root, nil)...)
	default:
		fmt.Fprintln(os.Stderr, "usage: apex validate <artifacts|spec|all> [paths]")
		return 2
	}
	if len(issues) == 0 {
		fmt.Printf("%s: ok\n", target)
		return 0
	}
	for _, is := range issues {
		fmt.Printf("%s: %s\n", is.File, is.Msg)
	}
	return 1
}
