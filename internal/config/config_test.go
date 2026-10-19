package config

import (
	"path/filepath"
	"testing"
)

func TestParseAndIgnore(t *testing.T) {
	cfg, err := Parse(`
# comment
port = 58000
bind = "127.0.0.1"
gap_seconds = 90
ignore = ["/tmp/notes", "/tmp/other"]
`)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 58000 || cfg.GapSeconds != 90 || cfg.Bind != "127.0.0.1" {
		t.Fatalf("parsed %#v", cfg)
	}
	if !Ignored("/tmp/notes/a.md", cfg.Ignore) {
		t.Fatal("expected notes to be ignored")
	}
	if Ignored("/tmp/notes-public/a.md", cfg.Ignore) {
		t.Fatal("prefix must match a path segment")
	}
	if Ignored("/tmp/code/main.go", cfg.Ignore) {
		t.Fatal("unrelated path was ignored")
	}
}

func TestParseRejectsRelativeIgnore(t *testing.T) {
	_, err := Parse("ignore = [\"notes\"]\n")
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestLoadCreatesTokenOnce(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "cfg"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, "data"))

	cfg, err := Load(true)
	if err != nil {
		t.Fatal(err)
	}
	token, err := Token(cfg.Dir)
	if err != nil || len(token) < 32 {
		t.Fatalf("token %q err %v", token, err)
	}
	again, err := Load(true)
	if err != nil {
		t.Fatal(err)
	}
	token2, err := Token(again.Dir)
	if err != nil {
		t.Fatal(err)
	}
	if token != token2 {
		t.Fatal("second serve rewrote the token")
	}
}
