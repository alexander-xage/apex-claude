package ledger

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

func newLedger(t *testing.T, agent string) *Ledger {
	t.Helper()
	root := t.TempDir()
	skill := filepath.Join(root, ".claude", "skills", agent)
	if err := os.MkdirAll(skill, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skill, "SKILL.md"), []byte("---\nname: "+agent+"\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	l, err := Open(root, agent)
	if err != nil {
		t.Fatal(err)
	}
	l.now = func() time.Time { return time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC) }
	if err := os.MkdirAll(l.dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return l
}

func mustAdd(t *testing.T, l *Ledger, title string) Item {
	t.Helper()
	it, _, err := l.Add("task", title, "user")
	if err != nil {
		t.Fatal(err)
	}
	return it
}

func TestRenderParseRoundTrip(t *testing.T) {
	in := Item{ID: 7, Title: "fix: the #1 thing -- now", Kind: "task", Status: "closed", Created: "2026-09-30",
		Origin: "LLMTraining:research/011", Closed: "2026-10-01", Reason: "done: see Outcome", Body: "## Goal\nx\n"}
	out, err := parse(in.render())
	if err != nil || out != in {
		t.Fatalf("round trip:\n got %+v, %v\nwant %+v", out, err, in)
	}
}

// Frontmatter is line-based: a newline in a value would let a title inject its own keys.
func TestAddRejectsUnsafeValues(t *testing.T) {
	l := newLedger(t, "dev")
	for _, title := range []string{"", "  ", "a\nid: 999", "a\rb", "bell\x07"} {
		if _, _, err := l.Add("task", title, "user"); !errors.Is(err, ErrUsage) {
			t.Errorf("Add(title %q) err = %v, want ErrUsage", title, err)
		}
	}
	if _, _, err := l.Add("task", "ok", "x\ny"); !errors.Is(err, ErrUsage) {
		t.Errorf("Add(origin with newline) err = %v, want ErrUsage", err)
	}
}

// debt items come only from the ponytail marker sync, never from a caller.
func TestAddRejectsDebtAndUnknownKinds(t *testing.T) {
	l := newLedger(t, "dev")
	for _, kind := range []string{"debt", "plan", ""} {
		if _, _, err := l.Add(kind, "x", "user"); !errors.Is(err, ErrUsage) {
			t.Errorf("Add(kind %q) err = %v, want ErrUsage", kind, err)
		}
	}
}

func TestAddAllocatesAfterHighestAndWritesSections(t *testing.T) {
	l := newLedger(t, "dev")
	if err := os.WriteFile(filepath.Join(l.dir, "handoff.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := mustAdd(t, l, "first")
	if err := os.WriteFile(filepath.Join(l.dir, "041.md"), (&Item{ID: 41, Title: "t", Kind: "task", Status: "open", Created: "2026-09-01", Origin: "user"}).render(), 0o644); err != nil {
		t.Fatal(err)
	}
	b, path, err := l.Add("followup", "second", "001")
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != 1 || b.ID != 42 || filepath.Base(path) != "042.md" {
		t.Fatalf("ids = %d, %d (%s); want 1, 42 (042.md)", a.ID, b.ID, filepath.Base(path))
	}
	data, _ := os.ReadFile(path)
	for _, s := range []string{"status: open", "created: 2026-09-30", "## Goal", "## Approach", "## Log", "## Outcome"} {
		if !strings.Contains(string(data), s) {
			t.Errorf("new item missing %q:\n%s", s, data)
		}
	}
}

// Parallel sessions and subagents file items at once; no two may get the same id.
func TestConcurrentAddGetsDistinctIDs(t *testing.T) {
	l := newLedger(t, "dev")
	const n = 25
	ids := make([]int, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			it, _, err := l.Add("followup", "x", "user")
			if err != nil {
				t.Error(err)
			}
			ids[i] = it.ID
		}()
	}
	wg.Wait()
	sort.Ints(ids)
	for i, id := range ids {
		if id != i+1 {
			t.Fatalf("ids = %v, want 1..%d", ids, n)
		}
	}
}

func TestTransitions(t *testing.T) {
	l := newLedger(t, "dev")
	a, b := mustAdd(t, l, "a"), mustAdd(t, l, "b")

	if err := l.Start(a.ID); err != nil {
		t.Fatal(err)
	}
	if err := l.Start(b.ID); !errors.Is(err, ErrTransition) {
		t.Fatalf("second in-progress: err = %v, want ErrTransition", err)
	}
	if err := l.Start(a.ID); !errors.Is(err, ErrTransition) {
		t.Fatalf("start in-progress: err = %v, want ErrTransition", err)
	}
	if err := l.Close(a.ID, ""); !errors.Is(err, ErrUsage) {
		t.Fatalf("close without reason: err = %v, want ErrUsage", err)
	}
	if err := l.Close(a.ID, "done"); err != nil {
		t.Fatal(err)
	}
	if err := l.Start(a.ID); !errors.Is(err, ErrTransition) {
		t.Fatalf("start closed: err = %v, want ErrTransition", err)
	}
	if err := l.Close(b.ID, "dropped: superseded by 001"); err != nil {
		t.Fatalf("open -> closed: %v", err)
	}
	if err := l.Reopen(a.ID); err != nil {
		t.Fatal(err)
	}
	got, err := l.Get(a.ID)
	if err != nil || got.Status != "open" || got.Closed != "" || got.Reason != "" {
		t.Fatalf("reopened = %+v, %v; want open with closed/reason cleared", got, err)
	}
	if err := l.Reopen(a.ID); !errors.Is(err, ErrTransition) {
		t.Fatalf("reopen open: err = %v, want ErrTransition", err)
	}
	if err := l.Start(99); !errors.Is(err, ErrNotFound) {
		t.Fatalf("start missing: err = %v, want ErrNotFound", err)
	}
}

// The model edits the body between status changes; the binary must never lose it.
func TestStatusChangePreservesBody(t *testing.T) {
	l := newLedger(t, "dev")
	_, path, err := l.Add("task", "a", "user")
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	edited := strings.Replace(string(data), "## Goal\n", "## Goal\nShip it: key: value\n---\nstill body\n", 1)
	if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := l.Start(1); err != nil {
		t.Fatal(err)
	}
	if err := l.Close(1, "done"); err != nil {
		t.Fatal(err)
	}
	it, err := l.Get(1)
	if err != nil || !strings.Contains(it.Body, "Ship it: key: value\n---\nstill body\n") {
		t.Fatalf("body lost after transitions: %q, %v", it.Body, err)
	}
}

func TestParseIDRejectsPaths(t *testing.T) {
	for _, s := range []string{"../x", "1/2", "", "-1", "1.md", "x", "0", "+1"} {
		if _, err := ParseID(s); !errors.Is(err, ErrUsage) {
			t.Errorf("ParseID(%q) err = %v, want ErrUsage", s, err)
		}
	}
	if id, err := ParseID("007"); err != nil || id != 7 {
		t.Errorf("ParseID(007) = %d, %v", id, err)
	}
}

func TestOpenValidatesAgent(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"../x", "Dev", "", "a/b", "-x"} {
		if _, err := Open(root, name); !errors.Is(err, ErrUsage) {
			t.Errorf("Open(%q) err = %v, want ErrUsage", name, err)
		}
	}
	// An Agent is its skill: no skill means a typo, not a new ledger.
	if _, err := Open(root, "ghost"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Open(ghost) err = %v, want ErrNotFound", err)
	}
}

func TestListDetectsCorruption(t *testing.T) {
	l := newLedger(t, "dev")
	mustAdd(t, l, "a")
	if err := os.WriteFile(filepath.Join(l.dir, "002.md"), (&Item{ID: 5, Title: "t", Kind: "task", Status: "open", Created: "2026-09-01", Origin: "user"}).render(), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := l.List(); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("id/filename mismatch: err = %v, want ErrCorrupt", err)
	}

	l = newLedger(t, "dev")
	for i := 1; i <= 2; i++ {
		it := Item{ID: i, Title: "t", Kind: "task", Status: "in-progress", Created: "2026-09-01", Origin: "user"}
		if err := os.WriteFile(filepath.Join(l.dir, itemName(i)), it.render(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := l.List(); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("two in-progress: err = %v, want ErrCorrupt", err)
	}
}

func TestListOrdersInProgressFirst(t *testing.T) {
	l := newLedger(t, "dev")
	mustAdd(t, l, "a")
	mustAdd(t, l, "b")
	mustAdd(t, l, "c")
	if err := l.Start(3); err != nil {
		t.Fatal(err)
	}
	items, err := l.List()
	if err != nil {
		t.Fatal(err)
	}
	var got []int
	for _, it := range items {
		got = append(got, it.ID)
	}
	if len(got) != 3 || got[0] != 3 || got[1] != 1 || got[2] != 2 {
		t.Fatalf("order = %v, want [3 1 2]", got)
	}
}

// The binary wrote the frontmatter, so anything it would not have written must surface as corruption, not be guessed at.
func TestParseRejectsMalformed(t *testing.T) {
	ok := "---\nid: 001\ntitle: t\nkind: task\nstatus: open\ncreated: 2026-09-30\norigin: user\n---\nbody\n"
	if _, err := parse([]byte(ok)); err != nil {
		t.Fatalf("valid item rejected: %v", err)
	}
	// An editor may drop the newline after the closing delimiter of an item with no body.
	if it, err := parse([]byte(strings.TrimSuffix(ok, "\nbody\n"))); err != nil || it.Body != "" {
		t.Fatalf("frontmatter ending in --- without newline: %+v, %v", it, err)
	}
	cases := map[string]string{
		"crlf":           strings.ReplaceAll(ok, "\n", "\r\n"),
		"no frontmatter": "id: 001\n",
		"unterminated":   "---\nid: 001\ntitle: t\n",
		"unknown key":    strings.Replace(ok, "origin: user\n", "origin: user\nowner: x\n", 1),
		"duplicate key":  strings.Replace(ok, "status: open\n", "status: closed\nstatus: open\n", 1),
		"bad status":     strings.Replace(ok, "status: open", "status: done", 1),
		"bad kind":       strings.Replace(ok, "kind: task", "kind: banana", 1),
		"missing origin": strings.Replace(ok, "origin: user\n", "", 1),
		"huge id":        strings.Replace(ok, "id: 001", "id: 99999999999999999999", 1),
		"closed only":    strings.Replace(ok, "origin: user\n", "origin: user\nclosed: 2026-09-30\n", 1),
		"reason only":    strings.Replace(ok, "origin: user\n", "origin: user\nreason: x\n", 1),
		"empty origin":   strings.Replace(ok, "origin: user", "origin: ", 1),
		"signed id":      strings.Replace(ok, "id: 001", "id: +1", 1),
		"control char":   strings.Replace(ok, "title: t", "title: t\r", 1),
		"bad line":       strings.Replace(ok, "title: t", "title:t", 1),
		"closed no date": strings.Replace(ok, "status: open", "status: closed", 1),
		"open w/ reason": strings.Replace(ok, "origin: user\n", "origin: user\nclosed: 2026-09-30\nreason: x\n", 1),
	}
	for name, in := range cases {
		if _, err := parse([]byte(in)); !errors.Is(err, ErrCorrupt) {
			t.Errorf("%s: err = %v, want ErrCorrupt", name, err)
		}
	}
}

// Two names for one id would list an item twice or hide it; only the canonical %03d.md form is an item.
func TestNonCanonicalNamesAreCorrupt(t *testing.T) {
	for _, name := range []string{"0005.md", "000.md", "99999999999999999999.md"} {
		l := newLedger(t, "dev")
		if err := os.WriteFile(filepath.Join(l.dir, name), nil, 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := l.List(); !errors.Is(err, ErrCorrupt) {
			t.Errorf("%s: List err = %v, want ErrCorrupt", name, err)
		}
	}
}

// On a case-insensitive disk 001.MD blocks 001.md without being listed; Add must fail instead of retrying forever.
func TestAddFailsOnUnlistedNameCollision(t *testing.T) {
	l := newLedger(t, "dev")
	if err := os.WriteFile(filepath.Join(l.dir, "001.MD"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(l.dir, "001.md")); err != nil {
		t.Skip("case-sensitive filesystem")
	}
	done := make(chan error, 1)
	go func() { _, _, err := l.Add("task", "x", "user"); done <- err }()
	select {
	case err := <-done:
		if !errors.Is(err, ErrCorrupt) {
			t.Fatalf("err = %v, want ErrCorrupt", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Add spun on an unlisted name collision")
	}
}

// Items are shared files; the temp-file write path must not leave them owner-only.
func TestWritesAreWorldReadable(t *testing.T) {
	l := newLedger(t, "dev")
	_, path, err := l.Add("task", "x", "user")
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Start(1); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Stat(path); err != nil || fi.Mode().Perm() != 0o644 {
		t.Fatalf("mode = %v, %v; want 0644", fi.Mode().Perm(), err)
	}
}
