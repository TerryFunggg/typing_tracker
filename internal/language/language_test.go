package language

import "testing"

func TestResolve(t *testing.T) {
	if got := Resolve("go-mode", "/tmp/main.go"); got != "go" {
		t.Fatalf("emacs mode = %s", got)
	}
	if got := Resolve("python-ts-mode", "/tmp/a.py"); got != "python" {
		t.Fatalf("ts mode = %s", got)
	}
	if got := Resolve("", "/tmp/main.go"); got != "go" {
		t.Fatalf("extension = %s", got)
	}
	if got := Resolve("fundamental-mode", "/tmp/Makefile"); got != "make" {
		t.Fatalf("makefile = %s", got)
	}
	if got := Resolve("markdown", "/tmp/a.md"); got != "markdown" {
		t.Fatalf("vim filetype = %s", got)
	}
	if got := Resolve("", "/tmp/file.zz"); got != "zz" {
		t.Fatalf("unknown ext = %s", got)
	}
}
