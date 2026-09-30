// Package fm parses and renders simple key/value frontmatter (--- delimited).
// Deliberately minimal — no nested structures — to stay zero-dependency.
package fm

import "strings"

// Parse splits a frontmatter document into its key/value map and body. If there
// is no leading "---" delimiter, the whole input is returned as the body.
func Parse(s string) (meta map[string]string, body string) {
	meta = map[string]string{}
	lines := strings.Split(s, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return meta, s
	}
	i := 1
	for ; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			i++
			break
		}
		if idx := strings.Index(lines[i], ":"); idx >= 0 {
			k := strings.TrimSpace(lines[i][:idx])
			v := strings.TrimSpace(lines[i][idx+1:])
			meta[k] = v
		}
	}
	body = strings.TrimPrefix(strings.Join(lines[i:], "\n"), "\n")
	return meta, body
}

// Render builds a frontmatter document from ordered keys plus a body.
func Render(order []string, meta map[string]string, body string) string {
	var b strings.Builder
	b.WriteString("---\n")
	for _, k := range order {
		b.WriteString(k + ": " + meta[k] + "\n")
	}
	b.WriteString("---\n\n")
	b.WriteString(body)
	if !strings.HasSuffix(body, "\n") {
		b.WriteString("\n")
	}
	return b.String()
}
