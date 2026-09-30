package main

// command is a registered subcommand: a handler and a one-line summary for usage.
type command struct {
	run     func(args []string) int
	summary string
}

// commands is the registry. cmd_*.go files populate it from init().
var commands = map[string]command{}

func register(name, summary string, run func([]string) int) {
	commands[name] = command{run: run, summary: summary}
}
