package gitinfo

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestLookup(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	dir := t.TempDir()
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")
	cmd := exec.Command("git", "init", "-b", "main")
	cmd.Dir = dir
	cmd.Env = os.Environ()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	file := filepath.Join(dir, "main.go")
	if err := os.WriteFile(file, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	root, branch := Lookup(file)
	if root != dir {
		t.Fatalf("root = %s, want %s", root, dir)
	}
	if branch != "main" {
		t.Fatalf("branch = %q", branch)
	}

	outside := filepath.Join(t.TempDir(), "lone.txt")
	if err := os.WriteFile(outside, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	root, branch = Lookup(outside)
	if root != "unknown" || branch != "" {
		t.Fatalf("outside git = %s %q", root, branch)
	}
}
