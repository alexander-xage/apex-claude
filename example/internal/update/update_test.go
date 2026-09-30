package update

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// --- LatestTag: redirect parsing ---------------------------------------

func TestLatestTagParsesRedirectLocation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "https://github.com/FNGApex/apex-claude/releases/tag/v1.2.3")
		w.WriteHeader(http.StatusFound)
	}))
	defer srv.Close()
	t.Setenv("APEX_UPDATE_BASE_URL", srv.URL)

	tag, err := LatestTag()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tag != "v1.2.3" {
		t.Errorf("got %q, want v1.2.3", tag)
	}
}

func TestLatestTagMissingLocationFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusFound) // no Location header
	}))
	defer srv.Close()
	t.Setenv("APEX_UPDATE_BASE_URL", srv.URL)

	if _, err := LatestTag(); err == nil {
		t.Error("expected error for missing Location header")
	}
}

func TestLatestTagMalformedLocationFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "https://github.com/x/y/releases/tag/")
		w.WriteHeader(http.StatusFound)
	}))
	defer srv.Close()
	t.Setenv("APEX_UPDATE_BASE_URL", srv.URL)

	if _, err := LatestTag(); err == nil {
		t.Error("expected error for malformed Location header")
	}
}

func TestLatestTagNonSemverTagIgnored(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "https://github.com/FNGApex/apex-claude/releases/tag/nightly-build")
		w.WriteHeader(http.StatusFound)
	}))
	defer srv.Close()
	t.Setenv("APEX_UPDATE_BASE_URL", srv.URL)

	if _, err := LatestTag(); err == nil {
		t.Error("expected error for non-semver tag")
	}
}

func TestLatestTagNonRedirectResponseFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	t.Setenv("APEX_UPDATE_BASE_URL", srv.URL)

	if _, err := LatestTag(); err == nil {
		t.Error("expected error for a non-redirect (no Location) response")
	}
}

// --- Compare -------------------------------------------------------------

func TestCompare(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"v1.2.3", "v1.2.3", 0},
		{"v1.2.3", "v1.2.4", -1},
		{"v1.2.4", "v1.2.3", 1},
		{"v1.3.0", "v1.2.9", 1},
		{"v2.0.0", "v1.9.9", 1},
		{"v0.2.0", "v0.3.0", -1},
		// Non-conforming input is treated as v0.0.0 (see Compare's doc comment).
		{"bogus", "v1.0.0", -1},
		{"v1.0.0", "bogus", 1},
		{"bogus", "also-bogus", 0},
		{"", "", 0},
	}
	for _, c := range cases {
		if got := Compare(c.a, c.b); got != c.want {
			t.Errorf("Compare(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

// --- Cache read/write ------------------------------------------------------

func TestReadCacheMissingFile(t *testing.T) {
	dir := t.TempDir()
	c := ReadCache(filepath.Join(dir, "nope.json"))
	if c != (Cache{}) {
		t.Errorf("missing cache file should read as zero value, got %+v", c)
	}
}

func TestReadCacheMalformedJSON(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(p, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := ReadCache(p)
	if c != (Cache{}) {
		t.Errorf("malformed cache should read as zero value, got %+v", c)
	}
}

func TestReadWriteCacheBlankPathNoop(t *testing.T) {
	if c := ReadCache(""); c != (Cache{}) {
		t.Errorf("blank path should read as zero cache, got %+v", c)
	}
	if err := WriteCache("", Cache{Latest: "v1.0.0"}); err != nil {
		t.Errorf("blank path write should be a no-op success, got %v", err)
	}
}

func TestWriteCacheCreatesDirOnDemand(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "nested", "update-check.json")
	if err := WriteCache(p, Cache{CheckedAt: "2026-01-01T00:00:00Z", Latest: "v1.0.0"}); err != nil {
		t.Fatal(err)
	}
	got := ReadCache(p)
	if got.Latest != "v1.0.0" {
		t.Errorf("round-tripped cache mismatch: %+v", got)
	}
}

func TestCacheJSONSchema(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "update-check.json")
	if err := WriteCache(p, Cache{CheckedAt: "2026-01-01T00:00:00Z", Latest: "v1.0.0"}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]string
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	if raw["checked_at"] != "2026-01-01T00:00:00Z" || raw["latest"] != "v1.0.0" {
		t.Errorf("unexpected on-disk schema: %s", b)
	}
}

func TestCachePathDegradesOnUserCacheDirError(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("UserCacheDir failure is simulated via %LocalAppData%, windows-only")
	}
	t.Setenv("LocalAppData", "")
	if p := CachePath(); p != "" {
		t.Errorf("expected empty cache path when UserCacheDir errors, got %q", p)
	}
}

// --- Stale / TTL -----------------------------------------------------------

func TestStaleTTLBoundary(t *testing.T) {
	now := time.Now().UTC()
	if Stale(Cache{}) != true {
		t.Error("empty cache (never checked) must be stale")
	}
	fresh := Cache{CheckedAt: now.Add(-23 * time.Hour).Format(time.RFC3339)}
	if Stale(fresh) {
		t.Error("23h-old cache should not be stale")
	}
	old := Cache{CheckedAt: now.Add(-25 * time.Hour).Format(time.RFC3339)}
	if !Stale(old) {
		t.Error("25h-old cache should be stale")
	}
	bad := Cache{CheckedAt: "not-a-time"}
	if !Stale(bad) {
		t.Error("unparseable checked_at should be treated as stale")
	}
}

// --- Refresh: ties LatestTag + cache together -------------------------------

func TestRefreshSuccessUpdatesCache(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "update-check.json")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "https://github.com/FNGApex/apex-claude/releases/tag/v9.9.9")
		w.WriteHeader(http.StatusFound)
	}))
	defer srv.Close()
	t.Setenv("APEX_UPDATE_BASE_URL", srv.URL)

	c, err := Refresh(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Latest != "v9.9.9" {
		t.Errorf("Latest = %q, want v9.9.9", c.Latest)
	}
	saved := ReadCache(path)
	if saved.Latest != "v9.9.9" {
		t.Errorf("persisted cache mismatch: %+v", saved)
	}
}

func TestRefreshFailedCheckPreservesLatestButStampsTime(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "update-check.json")
	prior := Cache{CheckedAt: "2020-01-01T00:00:00Z", Latest: "v1.0.0"}
	if err := WriteCache(path, prior); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	t.Setenv("APEX_UPDATE_BASE_URL", srv.URL)

	c, err := Refresh(path)
	if err == nil {
		t.Fatal("expected refresh error on a failed fetch")
	}
	if c.Latest != "v1.0.0" {
		t.Errorf("Latest should be preserved on failure, got %q", c.Latest)
	}
	if c.CheckedAt == prior.CheckedAt {
		t.Error("checked_at should be re-stamped even on a failed check")
	}

	saved := ReadCache(path)
	if saved.Latest != "v1.0.0" || saved.CheckedAt != c.CheckedAt {
		t.Errorf("persisted cache mismatch: %+v", saved)
	}
}
