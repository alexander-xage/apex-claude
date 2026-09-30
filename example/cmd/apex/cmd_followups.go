package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"apexclaude/internal/followups"
	"apexclaude/internal/proj"
)

func init() {
	register("followups", "list/add/close/render/path follow-up records", runFollowups)
}

func runFollowups(args []string) int {
	root := proj.Root()
	sub := ""
	if len(args) > 0 {
		sub = args[0]
	}
	switch sub {
	case "add":
		title, kind, sev, origin, err := parseFollowupAdd(args[1:])
		if err != nil {
			fmt.Fprintln(os.Stderr, "followups add:", err)
			fmt.Fprintln(os.Stderr, followupAddUsage)
			return 2
		}
		id, err := followups.Add(root, title, kind, sev, origin, "", time.Now())
		if err != nil {
			fmt.Fprintln(os.Stderr, "followups add:", err)
			return 2
		}
		fmt.Println("added follow-up", id)
		return 0
	case "list":
		list, err := followups.List(root)
		if err != nil {
			fmt.Fprintln(os.Stderr, "followups list:", err)
			return 2
		}
		if len(list) == 0 {
			fmt.Println("no open follow-ups")
			return 0
		}
		for _, e := range list {
			fmt.Printf("%s  %-7s  %-8s  %s\n", e.ID, e.Kind, e.Severity, e.Title)
		}
		return 0
	case "close":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: apex followups close <id> [reason]")
			return 2
		}
		if err := followups.Close(root, args[1], strings.Join(args[2:], " ")); err != nil {
			fmt.Fprintln(os.Stderr, "followups close:", err)
			return 1
		}
		fmt.Println("closed", args[1])
		return 0
	case "render":
		if err := followups.Render(root); err != nil {
			fmt.Fprintln(os.Stderr, "followups render:", err)
			return 2
		}
		fmt.Println("rendered INDEX.md")
		return 0
	case "path":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: apex followups path <id>")
			return 2
		}
		fmt.Println(followups.Path(root, args[1]))
		return 0
	default:
		fmt.Fprintln(os.Stderr, "usage: apex followups <list|add|close|render|path>")
		return 2
	}
}

const followupAddUsage = `usage: apex followups add "<title>" [--kind finding|plan] [--severity <sev>] [--origin <origin>]
       apex followups add "<title>" [finding|plan] [severity] [origin...]`

// parseFollowupAdd accepts two forms. Flags (--kind/--severity/--origin, as
// "--flag value" or "--flag=value", anywhere around one positional title), or
// the original positional form: title [kind] [severity] [origin...]. The two do
// not mix — with any flag present, the title must be the only positional.
//
// Anything else that starts with "-" is a usage error. The positional-only
// parser used to store flag-style args verbatim, which is how a real ledger
// entry ended up titled "--kind" with severity "--severity".
func parseFollowupAdd(args []string) (title, kind, sev, origin string, err error) {
	var pos []string
	flagged := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "-") {
			pos = append(pos, a)
			continue
		}
		name, val, hasEq := strings.Cut(a, "=")
		switch name {
		case "--kind", "--severity", "--origin":
		default:
			return "", "", "", "", fmt.Errorf("unknown flag %q", a)
		}
		if !hasEq {
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				return "", "", "", "", fmt.Errorf("%s needs a value", name)
			}
			i++
			val = args[i]
		}
		flagged = true
		switch name {
		case "--kind":
			kind = val
		case "--severity":
			sev = val
		case "--origin":
			origin = val
		}
	}

	switch {
	case len(pos) == 0:
		return "", "", "", "", fmt.Errorf("missing title")
	case flagged && len(pos) > 1:
		return "", "", "", "", fmt.Errorf("with flags, the title must be the only positional argument (got %d)", len(pos))
	case !flagged:
		if len(pos) > 1 {
			kind = pos[1]
		}
		if len(pos) > 2 {
			sev = pos[2]
		}
		if len(pos) > 3 {
			origin = strings.Join(pos[3:], " ")
		}
	}
	title = pos[0]
	if kind != "" && kind != "finding" && kind != "plan" {
		return "", "", "", "", fmt.Errorf("kind must be finding or plan, got %q", kind)
	}
	if origin == "" {
		origin = "cli"
	}
	return title, kind, sev, origin, nil
}
