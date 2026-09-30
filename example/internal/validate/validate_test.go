package validate

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, p, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestArtifacts(t *testing.T) {
	root := t.TempDir()
	// good agent
	writeFile(t, filepath.Join(root, "agents", "ok.md"), "---\nname: ok\ndescription: fine\n---\nbody\n")
	// bad agent (no description)
	writeFile(t, filepath.Join(root, "agents", "bad.md"), "---\nname: bad\n---\nbody\n")
	writeFile(t, filepath.Join(root, ".claude-plugin", "plugin.json"), `{"name":"x"}`)
	writeFile(t, filepath.Join(root, "hooks", "hooks.json"), `{not json`)

	issues := Artifacts(root)
	// expect: bad.md missing description + hooks.json invalid JSON
	var gotBad, gotJSON bool
	for _, is := range issues {
		if is.File == filepath.Join("agents", "bad.md") {
			gotBad = true
		}
		if is.File == filepath.Join("hooks", "hooks.json") {
			gotJSON = true
		}
	}
	if !gotBad {
		t.Errorf("expected missing-description issue for agents/bad.md; issues=%v", issues)
	}
	if !gotJSON {
		t.Errorf("expected invalid-JSON issue for hooks/hooks.json; issues=%v", issues)
	}
}

func TestSpec(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "docs", "spec", "good.md"), "# Title\n\n## Change log\n")
	writeFile(t, filepath.Join(root, "docs", "spec", "bad.md"), "no title, no log\n")

	issues := Spec(root, nil)
	if len(issues) == 0 {
		t.Fatal("expected issues for bad.md")
	}
	for _, is := range issues {
		if is.File == filepath.Join("docs", "spec", "good.md") {
			t.Errorf("good.md should pass, got: %s", is.Msg)
		}
	}
}
