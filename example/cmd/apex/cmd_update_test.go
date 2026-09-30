package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"apexclaude/internal/version"
)

// isolateCache redirects os.UserCacheDir to a fresh temp dir so tests never
// touch the real machine's update-check.json.
func isolateCache(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("LocalAppData", dir)   // windows: os.UserCacheDir()
	t.Setenv("XDG_CACHE_HOME", dir) // linux: os.UserCacheDir()
	t.Setenv("HOME", dir)           // darwin fallback: os.UserCacheDir()
}

// captureStdout runs fn with os.Stdout redirected to a pipe and returns what
// was printed, alongside fn's own return value.
func captureStdout(t *testing.T, fn func() int) (int, string) {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	code := fn()

	w.Close()
	os.Stdout = orig

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return code, buf.String()
}

func redirectServer(t *testing.T, tag string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "https://github.com/FNGApex/apex-claude/releases/tag/"+tag)
		w.WriteHeader(http.StatusFound)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestUpdateCheckUpToDate(t *testing.T) {
	isolateCache(t)
	srv := redirectServer(t, "v"+version.Version) // exactly the running version
	t.Setenv("APEX_UPDATE_BASE_URL", srv.URL)

	code, out := captureStdout(t, func() int { return runUpdateCheck(nil) })
	if code != 0 {
		t.Fatalf("want exit 0, got %d (out=%q)", code, out)
	}
	if !strings.Contains(out, "up to date") {
		t.Errorf("expected up-to-date message, got %q", out)
	}
}

func TestUpdateCheckAvailable(t *testing.T) {
	isolateCache(t)
	srv := redirectServer(t, "v9.9.9")
	t.Setenv("APEX_UPDATE_BASE_URL", srv.URL)

	code, out := captureStdout(t, func() int { return runUpdateCheck(nil) })
	if code != 1 {
		t.Fatalf("want exit 1, got %d (out=%q)", code, out)
	}
	if !strings.Contains(out, "update available") || !strings.Contains(out, "v9.9.9") {
		t.Errorf("expected update-available message with new tag, got %q", out)
	}
}

func TestUpdateCheckFailed(t *testing.T) {
	isolateCache(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	t.Setenv("APEX_UPDATE_BASE_URL", srv.URL)

	code, _ := captureStdout(t, func() int { return runUpdateCheck(nil) })
	if code != 2 {
		t.Fatalf("want exit 2 on check failure, got %d", code)
	}
}

func TestUpdateCheckQuietSuppressesStdout(t *testing.T) {
	isolateCache(t)
	srv := redirectServer(t, "v9.9.9")
	t.Setenv("APEX_UPDATE_BASE_URL", srv.URL)

	code, out := captureStdout(t, func() int { return runUpdateCheck([]string{"--quiet"}) })
	if code != 1 {
		t.Fatalf("want exit 1, got %d", code)
	}
	if out != "" {
		t.Errorf("--quiet must suppress stdout, got %q", out)
	}
}

// captureStderr mirrors captureStdout for os.Stderr.
func captureStderr(t *testing.T, fn func() int) (int, string) {
	t.Helper()
	orig := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w

	code := fn()

	w.Close()
	os.Stderr = orig

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return code, buf.String()
}

// --quiet silences the report, not the failure diagnostic: a hook-spawned
// check that dies must still leave a trace on stderr.
func TestUpdateCheckQuietKeepsStderrOnFailure(t *testing.T) {
	isolateCache(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	t.Setenv("APEX_UPDATE_BASE_URL", srv.URL)

	var errOut string
	code, out := captureStdout(t, func() int {
		var c int
		c, errOut = captureStderr(t, func() int { return runUpdateCheck([]string{"--quiet"}) })
		return c
	})
	if code != 2 {
		t.Fatalf("want exit 2, got %d", code)
	}
	if out != "" {
		t.Errorf("--quiet must suppress stdout, got %q", out)
	}
	if !strings.Contains(errOut, "apex update check:") {
		t.Errorf("--quiet must not swallow the stderr diagnostic, got %q", errOut)
	}
}

func TestRunUpdateDispatchesCheckSubcommand(t *testing.T) {
	isolateCache(t)
	srv := redirectServer(t, "v"+version.Version)
	t.Setenv("APEX_UPDATE_BASE_URL", srv.URL)

	code, out := captureStdout(t, func() int { return runUpdate([]string{"check"}) })
	if code != 0 {
		t.Fatalf("want exit 0, got %d (out=%q)", code, out)
	}
}

func TestRunUpdateUnknownSubcommand(t *testing.T) {
	code := runUpdate([]string{"bogus"})
	if code != 2 {
		t.Fatalf("want exit 2 on unknown subcommand, got %d", code)
	}
}
