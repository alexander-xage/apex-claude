// Package reminder stores time-based reminders as frontmatter files under
// .claude/project/reminders/. The session-start hook surfaces due ones.
package reminder

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"apexclaude/internal/fm"
)

func dir(root string) string { return filepath.Join(root, ".claude", "project", "reminders") }

// Reminder is one entry.
type Reminder struct {
	ID, Created, Due, Transport, Status, Text string
}

func nextID(d string) string {
	max := 0
	entries, _ := os.ReadDir(d)
	for _, e := range entries {
		var n int
		if _, err := fmt.Sscanf(strings.TrimSuffix(e.Name(), ".md"), "%d", &n); err == nil && n > max {
			max = n
		}
	}
	return fmt.Sprintf("%03d", max+1)
}

// Add writes a reminder. due is an RFC3339 timestamp or "" for no due date.
func Add(root, text, due, transport string, now time.Time) (string, error) {
	d := dir(root)
	if err := os.MkdirAll(d, 0o755); err != nil {
		return "", err
	}
	if transport == "" {
		transport = "none"
	}
	id := nextID(d)
	meta := map[string]string{
		"id":        id,
		"created":   now.UTC().Format(time.RFC3339),
		"due":       due,
		"transport": transport,
		"status":    "open",
	}
	doc := fm.Render([]string{"id", "created", "due", "transport", "status"}, meta, text)
	return id, os.WriteFile(filepath.Join(d, id+".md"), []byte(doc), 0o644)
}

func load(d, name string) (Reminder, error) {
	data, err := os.ReadFile(filepath.Join(d, name))
	if err != nil {
		return Reminder{}, err
	}
	m, body := fm.Parse(string(data))
	return Reminder{
		ID: m["id"], Created: m["created"], Due: m["due"],
		Transport: m["transport"], Status: m["status"],
		Text: strings.TrimSpace(body),
	}, nil
}

// List returns all reminders sorted by id.
func List(root string) ([]Reminder, error) {
	d := dir(root)
	entries, err := os.ReadDir(d)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Reminder
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		if r, err := load(d, e.Name()); err == nil {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// Get returns a single reminder by id.
func Get(root, id string) (Reminder, error) {
	return load(dir(root), id+".md")
}

// Rm deletes a reminder by id.
func Rm(root, id string) error {
	return os.Remove(filepath.Join(dir(root), id+".md"))
}

// Due returns open reminders whose due time is at or before now.
func Due(root string, now time.Time) ([]Reminder, error) {
	all, err := List(root)
	if err != nil {
		return nil, err
	}
	var out []Reminder
	for _, r := range all {
		if r.Status != "open" || r.Due == "" {
			continue
		}
		if t, err := time.Parse(time.RFC3339, r.Due); err == nil && !t.After(now) {
			out = append(out, r)
		}
	}
	return out, nil
}
