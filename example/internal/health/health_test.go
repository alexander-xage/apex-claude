package health

import "testing"

func TestSetShow(t *testing.T) {
	root := t.TempDir()
	if err := Set(root, 82, "init review"); err != nil {
		t.Fatalf("set: %v", err)
	}
	score, body, err := Show(root)
	if err != nil {
		t.Fatalf("show: %v", err)
	}
	if score != 82 {
		t.Errorf("score=%d, want 82", score)
	}
	if body == "" {
		t.Error("expected non-empty body")
	}
}

func TestShowUnset(t *testing.T) {
	score, _, err := Show(t.TempDir())
	if err != nil {
		t.Fatalf("show: %v", err)
	}
	if score != -1 {
		t.Errorf("unset score=%d, want -1", score)
	}
}

func TestSetRange(t *testing.T) {
	if err := Set(t.TempDir(), 101, ""); err == nil {
		t.Error("expected range error for score 101")
	}
}
