// Package health persists the repo health/integrity signal. Reviewers and
// doc-checks emit confidence scores; the orchestrator aggregates them and records
// the result here, where `apex doctor` and the session-start hook can read it.
package health

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const fileName = "health.md"

var scoreRe = regexp.MustCompile(`apex-health-score:\s*(\d+)`)

func filePath(root string) string {
	return filepath.Join(root, ".claude", "project", fileName)
}

// Show returns the current score (-1 if unset) and the file body ("" if unset).
func Show(root string) (score int, body string, err error) {
	data, err := os.ReadFile(filePath(root))
	if os.IsNotExist(err) {
		return -1, "", nil
	}
	if err != nil {
		return -1, "", err
	}
	score = -1
	if m := scoreRe.FindSubmatch(data); m != nil {
		score, _ = strconv.Atoi(string(m[1]))
	}
	return score, string(data), nil
}

// Set writes the score (0-100) with an optional note.
func Set(root string, score int, note string) error {
	if score < 0 || score > 100 {
		return fmt.Errorf("score must be 0-100, got %d", score)
	}
	var b strings.Builder
	b.WriteString("# Repo health / integrity\n\n")
	b.WriteString(fmt.Sprintf("<!-- apex-health-score: %d -->\n\n", score))
	b.WriteString(fmt.Sprintf("Score: **%d/100**\n", score))
	if note != "" {
		b.WriteString("\nLatest: " + note + "\n")
	}
	p := filePath(root)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(b.String()), 0o644)
}
