// Package gitinfo asks git for the repository of a file.
package gitinfo

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Lookup returns the repository root and branch for path.
// A file outside git comes back as project "unknown" and an empty branch.
func Lookup(path string) (root, branch string) {
	dir := path
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		dir = filepath.Dir(path)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	root = git(ctx, dir, "rev-parse", "--show-toplevel")
	if root == "" {
		return "unknown", ""
	}
	// symbolic-ref works before the first commit. rev-parse covers detached HEAD.
	branch = git(ctx, root, "symbolic-ref", "--short", "HEAD")
	if branch == "" {
		branch = git(ctx, root, "rev-parse", "--abbrev-ref", "HEAD")
	}
	return root, branch
}

func git(ctx context.Context, dir string, args ...string) string {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return ""
	}
	return strings.TrimSpace(stdout.String())
}
