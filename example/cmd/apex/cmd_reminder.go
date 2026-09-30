package main

import (
	"fmt"
	"os"
	"time"

	"apexclaude/internal/proj"
	"apexclaude/internal/reminder"
)

func init() {
	register("reminder", "add/list/show/rm/due time-based reminders", runReminder)
}

func runReminder(args []string) int {
	root := proj.Root()
	sub := ""
	if len(args) > 0 {
		sub = args[0]
	}
	switch sub {
	case "add":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, `usage: apex reminder add "<text>" [due-RFC3339] [transport]`)
			return 2
		}
		due, transport := "", ""
		if len(args) > 2 {
			due = args[2]
		}
		if len(args) > 3 {
			transport = args[3]
		}
		id, err := reminder.Add(root, args[1], due, transport, time.Now())
		if err != nil {
			fmt.Fprintln(os.Stderr, "reminder add:", err)
			return 2
		}
		fmt.Println("added reminder", id)
		return 0
	case "list":
		all, err := reminder.List(root)
		if err != nil {
			fmt.Fprintln(os.Stderr, "reminder list:", err)
			return 2
		}
		if len(all) == 0 {
			fmt.Println("no reminders")
			return 0
		}
		for _, r := range all {
			fmt.Printf("%s  [%s]  due=%s  %s\n", r.ID, r.Status, orDash(r.Due), r.Text)
		}
		return 0
	case "show":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: apex reminder show <id>")
			return 2
		}
		r, err := reminder.Get(root, args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, "reminder show:", err)
			return 1
		}
		fmt.Printf("%s  [%s]  due=%s\n%s\n", r.ID, r.Status, orDash(r.Due), r.Text)
		return 0
	case "rm":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: apex reminder rm <id>")
			return 2
		}
		if err := reminder.Rm(root, args[1]); err != nil {
			fmt.Fprintln(os.Stderr, "reminder rm:", err)
			return 1
		}
		fmt.Println("removed", args[1])
		return 0
	case "due":
		due, err := reminder.Due(root, time.Now())
		if err != nil {
			fmt.Fprintln(os.Stderr, "reminder due:", err)
			return 2
		}
		for _, r := range due {
			fmt.Printf("%s  %s\n", r.ID, r.Text)
		}
		return 0
	default:
		fmt.Fprintln(os.Stderr, "usage: apex reminder <add|list|show|rm|due>")
		return 2
	}
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
