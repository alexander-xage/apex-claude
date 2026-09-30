package version

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestVersionFormat(t *testing.T) {
	if !regexp.MustCompile(`^\d+\.\d+\.\d+$`).MatchString(Version) {
		t.Errorf("Version %q does not match MAJOR.MINOR.PATCH", Version)
	}
}

// The plugin manifests carry their own copy of the version. They drifted once
// (marketplace.json sat at 0.1.0 while plugin.json said 0.2.0), so a version
// bump must move all three together or this fails.
func TestPluginManifestsMatchVersion(t *testing.T) {
	read := func(path string) []byte {
		t.Helper()
		b, err := os.ReadFile(filepath.Join("..", "..", ".claude-plugin", path))
		if err != nil {
			t.Fatal(err)
		}
		return b
	}

	var plugin struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(read("plugin.json"), &plugin); err != nil {
		t.Fatal(err)
	}
	if plugin.Version != Version {
		t.Errorf("plugin.json version = %q, want %q", plugin.Version, Version)
	}

	var market struct {
		Plugins []struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"plugins"`
	}
	if err := json.Unmarshal(read("marketplace.json"), &market); err != nil {
		t.Fatal(err)
	}
	if len(market.Plugins) == 0 {
		t.Fatal("marketplace.json lists no plugins")
	}
	for _, p := range market.Plugins {
		if p.Version != Version {
			t.Errorf("marketplace.json plugin %q version = %q, want %q", p.Name, p.Version, Version)
		}
	}
}
