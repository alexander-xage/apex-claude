package reminder

import (
	"testing"
	"time"
)

func TestAddListDue(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)

	past := now.Add(-time.Hour).Format(time.RFC3339)
	future := now.Add(time.Hour).Format(time.RFC3339)

	if _, err := Add(root, "overdue thing", past, "none", now); err != nil {
		t.Fatal(err)
	}
	if _, err := Add(root, "later thing", future, "cron", now); err != nil {
		t.Fatal(err)
	}

	all, err := List(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("List len=%d, want 2", len(all))
	}

	due, err := Due(root, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(due) != 1 || due[0].Text != "overdue thing" {
		t.Fatalf("Due=%v, want 1 (overdue thing)", due)
	}
}

func TestRm(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	id, err := Add(root, "x", "", "none", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := Rm(root, id); err != nil {
		t.Fatal(err)
	}
	all, _ := List(root)
	if len(all) != 0 {
		t.Fatalf("after rm len=%d, want 0", len(all))
	}
}
