// Package ledger stores an Agent's work items: one numbered markdown file per item under
// <main>/.claude/ledger/<agent>/. The binary owns each item's frontmatter; the model owns its body.
package ledger

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

var (
	ErrUsage      = errors.New("usage")
	ErrNotFound   = errors.New("not found")
	ErrTransition = errors.New("illegal transition")
	ErrCorrupt    = errors.New("corrupt ledger")
)

const (
	StatusOpen       = "open"
	StatusInProgress = "in-progress"
	StatusClosed     = "closed"
)

var (
	kinds    = []string{"task", "followup"}
	statuses = []string{StatusOpen, StatusInProgress, StatusClosed}

	agentRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
	idRe    = regexp.MustCompile(`^[0-9]+$`)
	fileRe  = regexp.MustCompile(`^([0-9]+)\.md$`)
)

type Item struct {
	ID      int
	Title   string
	Kind    string
	Status  string
	Created string
	Closed  string
	Reason  string
	Body    string
}

const bodyTemplate = "## Goal\n\n## Approach\n\n## Log\n\n## Outcome\n"

func (it *Item) render() []byte {
	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "id: %03d\ntitle: %s\nkind: %s\nstatus: %s\ncreated: %s\n",
		it.ID, it.Title, it.Kind, it.Status, it.Created)
	if it.Closed != "" {
		fmt.Fprintf(&b, "closed: %s\nreason: %s\n", it.Closed, it.Reason)
	}
	b.WriteString("---\n")
	b.WriteString(it.Body)
	return []byte(b.String())
}

// parse is strict because the binary wrote the frontmatter: anything it would not have written is corruption.
func parse(data []byte) (Item, error) {
	s := string(data)
	if strings.HasPrefix(s, "---\r\n") {
		return Item{}, fmt.Errorf("%w: CRLF line endings", ErrCorrupt)
	}
	if !strings.HasPrefix(s, "---\n") {
		return Item{}, fmt.Errorf("%w: missing frontmatter", ErrCorrupt)
	}
	s = s[len("---\n"):]
	head, body, ok := strings.Cut(s, "\n---\n")
	if !ok {
		if head, ok = strings.CutSuffix(s, "\n---"); !ok {
			return Item{}, fmt.Errorf("%w: unterminated frontmatter", ErrCorrupt)
		}
	}
	it := Item{Body: body}
	seen := map[string]bool{}
	for _, line := range strings.Split(head, "\n") {
		k, v, ok := strings.Cut(line, ": ")
		if !ok {
			return Item{}, fmt.Errorf("%w: bad frontmatter line %q", ErrCorrupt, line)
		}
		if seen[k] {
			return Item{}, fmt.Errorf("%w: duplicate key %q", ErrCorrupt, k)
		}
		seen[k] = true
		if err := checkLine(k, v); err != nil {
			return Item{}, fmt.Errorf("%w: %v", ErrCorrupt, err)
		}
		switch k {
		case "id":
			n, err := strconv.Atoi(v)
			if !idRe.MatchString(v) || err != nil {
				return Item{}, fmt.Errorf("%w: bad id %q", ErrCorrupt, v)
			}
			it.ID = n
		case "title":
			it.Title = v
		case "kind":
			it.Kind = v
		case "status":
			it.Status = v
		case "created":
			it.Created = v
		case "closed":
			it.Closed = v
		case "reason":
			it.Reason = v
		default:
			return Item{}, fmt.Errorf("%w: unknown key %q", ErrCorrupt, k)
		}
	}
	for _, k := range []string{"id", "title", "kind", "status", "created"} {
		if !seen[k] {
			return Item{}, fmt.Errorf("%w: missing %s", ErrCorrupt, k)
		}
	}
	if !ValidKind(it.Kind) {
		return Item{}, fmt.Errorf("%w: bad kind %q", ErrCorrupt, it.Kind)
	}
	if !ValidStatus(it.Status) {
		return Item{}, fmt.Errorf("%w: bad status %q", ErrCorrupt, it.Status)
	}
	closed := it.Status == StatusClosed
	if closed != (it.Closed != "") || closed != (it.Reason != "") {
		return Item{}, fmt.Errorf("%w: closed and reason must be set exactly when status is closed", ErrCorrupt)
	}
	return it, nil
}

// checkLine keeps a frontmatter value to one printable line, so no value can inject a key or a terminal control.
func checkLine(field, v string) error {
	if v == "" {
		return fmt.Errorf("%s is empty", field)
	}
	if strings.IndexFunc(v, unicode.IsControl) >= 0 {
		return fmt.Errorf("%s must be one line without control characters", field)
	}
	return nil
}

// ValidKind reports whether k is an item kind.
func ValidKind(k string) bool { return slices.Contains(kinds, k) }

// ValidStatus reports whether s is an item status.
func ValidStatus(s string) bool { return slices.Contains(statuses, s) }

// CheckAgent validates an Agent name before it becomes a path segment.
func CheckAgent(agent string) error {
	if !agentRe.MatchString(agent) {
		return fmt.Errorf("%w: agent name must match %s, got %q", ErrUsage, agentRe, agent)
	}
	return nil
}

// CheckNew validates the fields of an item to add.
func CheckNew(kind, title string) error {
	if !ValidKind(kind) {
		return fmt.Errorf("%w: kind must be %s, got %q", ErrUsage, strings.Join(kinds, " or "), kind)
	}
	if err := checkLine("title", strings.TrimSpace(title)); err != nil {
		return fmt.Errorf("%w: %v", ErrUsage, err)
	}
	return nil
}

// CheckReason validates the one-line reason a close requires.
func CheckReason(reason string) error {
	if err := checkLine("reason", strings.TrimSpace(reason)); err != nil {
		return fmt.Errorf("%w: %v", ErrUsage, err)
	}
	return nil
}

// ParseID accepts digits only, so an id can never become a path.
func ParseID(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if !idRe.MatchString(s) || err != nil || n == 0 {
		return 0, fmt.Errorf("%w: id must be a positive number, got %q", ErrUsage, s)
	}
	return n, nil
}

type Ledger struct {
	dir string
	now func() time.Time
}

// Open returns the ledger of agent in the main checkout root. The agent must exist as a skill.
func Open(root, agent string) (*Ledger, error) {
	if err := CheckAgent(agent); err != nil {
		return nil, err
	}
	skill := filepath.Join(root, ".claude", "skills", agent, "SKILL.md")
	if _, err := os.Stat(skill); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: no agent %q (missing %s)", ErrNotFound, agent, skill)
		}
		return nil, err
	}
	return &Ledger{dir: filepath.Join(root, ".claude", "ledger", agent), now: time.Now}, nil
}

func itemName(id int) string { return fmt.Sprintf("%03d.md", id) }

func (l *Ledger) Path(id int) string { return filepath.Join(l.dir, itemName(id)) }

// Add files a new open item and returns it with its path. The file appears complete or not at all: it is written to a
// temp file and hard-linked into place, and the link fails if a concurrent Add took the id first.
func (l *Ledger) Add(kind, title string) (Item, string, error) {
	if err := CheckNew(kind, title); err != nil {
		return Item{}, "", err
	}
	title = strings.TrimSpace(title)
	if err := os.MkdirAll(l.dir, 0o755); err != nil {
		return Item{}, "", err
	}
	for {
		ids, err := l.ids()
		if err != nil {
			return Item{}, "", err
		}
		next := 1
		if len(ids) > 0 {
			next = ids[len(ids)-1] + 1
		}
		it := Item{ID: next, Title: title, Kind: kind, Status: StatusOpen, Created: l.today(), Body: bodyTemplate}
		err = l.writeAtomic(it, os.Link)
		if !errors.Is(err, os.ErrExist) {
			if err != nil {
				return Item{}, "", err
			}
			return it, l.Path(next), nil
		}
		// A concurrent Add won the id, so retry. If the id is still not listed, something unlisted (such as 002.MD on
		// a case-insensitive disk) holds the name and retrying would spin forever.
		if ids, err = l.ids(); err != nil {
			return Item{}, "", err
		}
		if !slices.Contains(ids, next) {
			return Item{}, "", fmt.Errorf("%w: %s is taken by a file that is not a ledger item", ErrCorrupt, l.Path(next))
		}
	}
}

func (l *Ledger) today() string { return l.now().Format("2006-01-02") }

func (l *Ledger) ids() ([]int, error) {
	entries, err := os.ReadDir(l.dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var ids []int
	for _, e := range entries {
		m := fileRe.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		n, err := strconv.Atoi(m[1])
		if err != nil || n == 0 || e.Name() != itemName(n) {
			return nil, fmt.Errorf("%w: %s is not a canonical item name", ErrCorrupt, e.Name())
		}
		ids = append(ids, n)
	}
	sort.Ints(ids)
	return ids, nil
}

func (l *Ledger) Get(id int) (Item, error) {
	data, err := os.ReadFile(l.Path(id))
	if os.IsNotExist(err) {
		return Item{}, fmt.Errorf("%w: item %03d", ErrNotFound, id)
	}
	if err != nil {
		return Item{}, err
	}
	it, err := parse(data)
	if err != nil {
		return Item{}, fmt.Errorf("%s: %w", itemName(id), err)
	}
	if it.ID != id {
		return Item{}, fmt.Errorf("%w: %s has id %03d", ErrCorrupt, itemName(id), it.ID)
	}
	return it, nil
}

// List returns every item, in-progress first, then by id. It fails on any corrupt item or on more than one item in
// progress, so a broken ledger is noticed on the next read instead of silently skipped.
func (l *Ledger) List() ([]Item, error) {
	ids, err := l.ids()
	if err != nil {
		return nil, err
	}
	items := make([]Item, 0, len(ids))
	active := 0
	for _, id := range ids {
		it, err := l.Get(id)
		if err != nil {
			return nil, err
		}
		if it.Status == StatusInProgress {
			active++
		}
		items = append(items, it)
	}
	if active > 1 {
		return nil, fmt.Errorf("%w: %d items in progress", ErrCorrupt, active)
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Status == StatusInProgress && items[j].Status != StatusInProgress
	})
	return items, nil
}

// Start moves an open item to in-progress; the Agent may have only one.
// ponytail: two sessions of one Agent can race past this check; add a lock file if that happens.
func (l *Ledger) Start(id int) error {
	items, err := l.List()
	if err != nil {
		return err
	}
	var it *Item
	for i := range items {
		if items[i].ID == id {
			it = &items[i]
		} else if items[i].Status == StatusInProgress {
			return fmt.Errorf("%w: %03d is already in progress", ErrTransition, items[i].ID)
		}
	}
	if it == nil {
		return fmt.Errorf("%w: item %03d", ErrNotFound, id)
	}
	if it.Status != StatusOpen {
		return fmt.Errorf("%w: %03d is %s, start needs open", ErrTransition, id, it.Status)
	}
	it.Status = StatusInProgress
	return l.writeAtomic(*it, os.Rename)
}

func (l *Ledger) Close(id int, reason string) error {
	if err := CheckReason(reason); err != nil {
		return err
	}
	reason = strings.TrimSpace(reason)
	it, err := l.Get(id)
	if err != nil {
		return err
	}
	if it.Status == StatusClosed {
		return fmt.Errorf("%w: %03d is already closed", ErrTransition, id)
	}
	it.Status, it.Closed, it.Reason = StatusClosed, l.today(), reason
	return l.writeAtomic(it, os.Rename)
}

func (l *Ledger) Reopen(id int) error {
	it, err := l.Get(id)
	if err != nil {
		return err
	}
	if it.Status == StatusOpen {
		return fmt.Errorf("%w: %03d is already open", ErrTransition, id)
	}
	it.Status, it.Closed, it.Reason = StatusOpen, "", ""
	return l.writeAtomic(it, os.Rename)
}

// writeAtomic writes the item to a synced temp file, then places it with place (os.Link to create, os.Rename to
// replace), so readers and crashes never see a partial item.
func (l *Ledger) writeAtomic(it Item, place func(oldpath, newpath string) error) error {
	tmp, err := os.CreateTemp(l.dir, ".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	_, err = tmp.Write(it.render())
	if err == nil {
		err = tmp.Chmod(0o644)
	}
	if err == nil {
		err = tmp.Sync()
	}
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	return place(tmp.Name(), l.Path(it.ID))
}
