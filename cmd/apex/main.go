// Command apex is the deterministic backbone of the Agent protocol: it resolves the main checkout and keeps each
// Agent's ledger.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"apex/internal/ledger"
	"apex/internal/repo"
)

const usage = `usage: apex <verb> [args] --agent <name>

  add <task|followup> <title>          file a new open item, print its path
  list [--status s,s|all] [--kind k]   default status: in-progress,open
  start <id>                           open -> in-progress
  close <id> <reason>                  -> closed
  reopen <id>                          in-progress|closed -> open
  graph <graphify args>                run graphify on this checkout's graph (no --agent)

exit: 0 ok, 1 not found, 2 environment, 3 illegal transition, 4 corrupt ledger, 64 usage;
      graph exits with graphify's code, or 128+N when graphify dies by signal N
`

func main() {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "apex:", err)
		os.Exit(2)
	}
	os.Exit(run(dir, os.Args[1:], os.Stdout, os.Stderr))
}

func run(dir string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 64
	}
	verb, args := args[0], args[1:]
	if verb == "-h" || verb == "--help" || verb == "help" {
		fmt.Fprint(stdout, usage)
		return 0
	}
	if verb == "graph" {
		return graph(dir, args, stdout, stderr)
	}
	err := dispatch(dir, verb, args, stdout)
	if errors.Is(err, flag.ErrHelp) {
		fmt.Fprint(stdout, usage)
		return 0
	}
	if err != nil {
		fmt.Fprintf(stderr, "apex %s: %v\n", verb, err)
	}
	return exitCode(err)
}

func exitCode(err error) int {
	switch {
	case err == nil:
		return 0
	case errors.Is(err, ledger.ErrUsage):
		return 64
	case errors.Is(err, ledger.ErrNotFound):
		return 1
	case errors.Is(err, ledger.ErrTransition):
		return 3
	case errors.Is(err, ledger.ErrCorrupt):
		return 4
	default:
		return 2
	}
}

func usageErr(format string, a ...any) error {
	return fmt.Errorf("%w: %s", ledger.ErrUsage, fmt.Sprintf(format, a...))
}

// dispatch validates every argument before reading the disk, so a usage error exits 64 whatever state the ledger is in.
func dispatch(dir, verb string, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet(verb, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	agent := fs.String("agent", "", "")
	var status, kind *string
	var want int
	switch verb {
	case "add":
		want = 2
	case "list":
		status = fs.String("status", "in-progress,open", "")
		kind = fs.String("kind", "", "")
	case "start", "reopen":
		want = 1
	case "close":
		want = 2
	default:
		return usageErr("unknown verb %q", verb)
	}
	pos, err := parseInterleaved(fs, args)
	if errors.Is(err, flag.ErrHelp) {
		return err
	}
	if err != nil {
		return usageErr("%v", err)
	}
	if len(pos) != want {
		return usageErr("%s takes %d argument(s), got %d", verb, want, len(pos))
	}
	if *agent == "" {
		return usageErr("--agent is required")
	}
	if err := ledger.CheckAgent(*agent); err != nil {
		return err
	}

	var id int
	var show map[string]bool
	switch verb {
	case "add":
		err = ledger.CheckNew(pos[0], pos[1])
	case "list":
		show, err = parseStatuses(*status)
		if err == nil && *kind != "" && !ledger.ValidKind(*kind) {
			err = usageErr("unknown kind %q", *kind)
		}
	default:
		id, err = ledger.ParseID(pos[0])
		if err == nil && verb == "close" {
			err = ledger.CheckReason(pos[1])
		}
	}
	if err != nil {
		return err
	}

	root, err := repo.Main(dir)
	if err != nil {
		return err
	}
	l, err := ledger.Open(root, *agent)
	if err != nil {
		return err
	}
	switch verb {
	case "add":
		_, path, err := l.Add(pos[0], pos[1])
		if err == nil {
			fmt.Fprintln(stdout, path)
		}
		return err
	case "list":
		return list(l, show, *kind, stdout)
	case "start":
		err = l.Start(id)
	case "reopen":
		err = l.Reopen(id)
	case "close":
		err = l.Close(id, pos[1])
	}
	if err == nil {
		fmt.Fprintln(stdout, l.Path(id))
	}
	return err
}

// parseInterleaved lets flags appear before, between, or after positionals. Everything after "--" is positional.
func parseInterleaved(fs *flag.FlagSet, args []string) ([]string, error) {
	var tail []string
	if i := slices.Index(args, "--"); i >= 0 {
		args, tail = args[:i], args[i+1:]
	}
	var pos []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		args = fs.Args()
		if len(args) == 0 {
			return append(pos, tail...), nil
		}
		pos, args = append(pos, args[0]), args[1:]
	}
}

func parseStatuses(s string) (map[string]bool, error) {
	show := map[string]bool{}
	for _, st := range strings.Split(s, ",") {
		switch st = strings.TrimSpace(st); {
		case st == "all":
			show[ledger.StatusOpen], show[ledger.StatusInProgress], show[ledger.StatusClosed] = true, true, true
		case ledger.ValidStatus(st):
			show[st] = true
		default:
			return nil, usageErr("unknown status %q", st)
		}
	}
	return show, nil
}

func list(l *ledger.Ledger, show map[string]bool, kind string, stdout io.Writer) error {
	items, err := l.List()
	if err != nil {
		return err
	}
	counts := map[string]int{}
	for _, it := range items {
		if kind != "" && it.Kind != kind {
			continue
		}
		counts[it.Status]++
		if show[it.Status] {
			fmt.Fprintf(stdout, "%03d  %-11s  %-8s  %s\n", it.ID, it.Status, it.Kind, it.Title)
		}
	}
	fmt.Fprintf(stdout, "%d open, %d in-progress, %d closed\n",
		counts[ledger.StatusOpen], counts[ledger.StatusInProgress], counts[ledger.StatusClosed])
	return nil
}
