// Command apex is the Apex Claude deterministic backbone: the layer the markdown
// artifacts can't be. It owns hooks (session start), scanning (signals), integrity
// (doctor), and the repo health signal — work that must run reliably regardless
// of what the model decides.
//
// Subcommands self-register via init() in cmd_*.go files (see registry.go), so
// adding one never touches a central switch.
package main

import (
	"fmt"
	"os"
	"sort"

	"apexclaude/internal/version"
)

func main() {
	if len(os.Args) < 2 {
		usage(os.Stderr)
		os.Exit(2)
	}
	switch os.Args[1] {
	case "version", "--version", "-v":
		fmt.Printf("apex %s\n", version.Version)
		return
	case "help", "--help", "-h":
		usage(os.Stdout)
		return
	}
	cmd, ok := commands[os.Args[1]]
	if !ok {
		fmt.Fprintf(os.Stderr, "apex: unknown command %q\n\n", os.Args[1])
		usage(os.Stderr)
		os.Exit(2)
	}
	os.Exit(cmd.run(os.Args[2:]))
}

func usage(w *os.File) {
	fmt.Fprint(w, "apex — Apex Claude deterministic backbone\n\nusage:\n  apex <command> [args]\n\ncommands:\n")
	names := make([]string, 0, len(commands))
	for n := range commands {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		fmt.Fprintf(w, "  %-12s %s\n", n, commands[n].summary)
	}
	fmt.Fprintln(w, "  version      print version")
}
