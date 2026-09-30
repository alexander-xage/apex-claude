// Package update implements the deterministic half of the auto-update
// pipeline: discovering the latest GitHub release tag, comparing versions,
// and caching the result. No LLM anywhere in this path.
package update

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// RepoSlug is the GitHub repo backing releases; must equal install.ps1's $Repo.
const RepoSlug = "FNGApex/apex-claude"

// defaultLatestURL is the release-latest redirect endpoint GitHub serves for
// RepoSlug without following redirects.
const defaultLatestURL = "https://github.com/" + RepoSlug + "/releases/latest"

// TTL is how long a cache entry stays fresh before Stale reports it needs
// re-checking.
const TTL = 24 * time.Hour

// tagRe matches the vMAJOR.MINOR.PATCH tag format. Anything else is ignored.
var tagRe = regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)$`)

// latestURL resolves the URL LatestTag GETs. APEX_UPDATE_BASE_URL overrides
// it wholesale — the test seam the spec names for httptest — defaulting to
// GitHub's redirect endpoint for RepoSlug.
func latestURL() string {
	if u := os.Getenv("APEX_UPDATE_BASE_URL"); u != "" {
		return u
	}
	return defaultLatestURL
}

// LatestTag GETs latestURL() with a client that does not follow redirects
// and a 5s timeout, then parses the tag from the last path segment of the
// response's Location header. Any of the following is treated as a check
// failure (returns an error, never panics): a network error, a response with
// no/blank Location header (including a non-redirect response), or a tag
// that doesn't match vMAJOR.MINOR.PATCH.
func LatestTag() (string, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Get(latestURL())
	if err != nil {
		return "", fmt.Errorf("update: fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	loc := strings.TrimRight(resp.Header.Get("Location"), "/")
	if loc == "" {
		return "", errors.New("update: no Location header in latest-release response")
	}
	tag := loc
	if i := strings.LastIndex(loc, "/"); i >= 0 {
		tag = loc[i+1:]
	}
	if !tagRe.MatchString(tag) {
		return "", fmt.Errorf("update: tag %q does not match vMAJOR.MINOR.PATCH", tag)
	}
	return tag, nil
}

// Compare orders vX.Y.Z tags as int triples, returning -1 if a<b, 0 if a==b,
// 1 if a>b — the strings.Compare convention. Input not matching
// vMAJOR.MINOR.PATCH is treated as v0.0.0 for comparison purposes: a
// malformed tag never outranks a well-formed one, and two malformed tags
// compare equal.
func Compare(a, b string) int {
	ta, tb := parseTriple(a), parseTriple(b)
	for i := 0; i < 3; i++ {
		if ta[i] != tb[i] {
			if ta[i] < tb[i] {
				return -1
			}
			return 1
		}
	}
	return 0
}

func parseTriple(v string) [3]int {
	m := tagRe.FindStringSubmatch(v)
	if m == nil {
		return [3]int{}
	}
	var t [3]int
	for i := 0; i < 3; i++ {
		t[i], _ = strconv.Atoi(m[i+1])
	}
	return t
}

// Cache is the on-disk schema at CachePath: {"checked_at":"<RFC3339>","latest":"vX.Y.Z"}.
type Cache struct {
	CheckedAt string `json:"checked_at"`
	Latest    string `json:"latest"`
}

// CachePath resolves the fixed cache file location via os.UserCacheDir. A
// UserCacheDir error (no cache dir available on this system) degrades to no
// cache — it returns "" rather than an error; ReadCache/WriteCache both treat
// a blank path as a no-op.
func CachePath() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "apex-claude", "update-check.json")
}

// ReadCache reads the cache at path. A blank path, missing file, or
// unreadable/malformed JSON all return a zero Cache and no error — callers
// treat that uniformly as "never checked".
func ReadCache(path string) Cache {
	if path == "" {
		return Cache{}
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return Cache{}
	}
	var c Cache
	if json.Unmarshal(b, &c) != nil {
		return Cache{}
	}
	return c
}

// WriteCache writes c as JSON to path, creating the parent dir on demand. A
// blank path is a no-op success — the UserCacheDir-degraded case.
func WriteCache(path string, c Cache) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// Stale reports whether c needs re-checking: never populated, unparseable, or
// older than TTL.
func Stale(c Cache) bool {
	if c.CheckedAt == "" {
		return true
	}
	t, err := time.Parse(time.RFC3339, c.CheckedAt)
	if err != nil {
		return true
	}
	return time.Since(t) > TTL
}

// Refresh fetches LatestTag and rewrites the cache at path, regardless of
// TTL — callers own TTL gating. A failed fetch still stamps CheckedAt (so
// callers can rate-limit retries) but preserves the prior Latest value. The
// returned Cache is always what got persisted; the returned error, if any, is
// the underlying LatestTag failure.
func Refresh(path string) (Cache, error) {
	prior := ReadCache(path)
	tag, err := LatestTag()
	c := Cache{CheckedAt: time.Now().UTC().Format(time.RFC3339)}
	if err != nil {
		c.Latest = prior.Latest
	} else {
		c.Latest = tag
	}
	if werr := WriteCache(path, c); werr != nil && err == nil {
		err = werr
	}
	return c, err
}
