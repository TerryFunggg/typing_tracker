package report

import (
	"strings"
	"testing"
	"time"

	"typing_tracker/internal/store"
)

func TestTodayAndWeek(t *testing.T) {
	loc := time.Local
	day := time.Date(2026, 10, 17, 15, 0, 0, 0, loc)
	beats := []store.Heartbeat{
		{At: day, FilePath: "/repos/typing_tracker/main.go", Project: "/repos/typing_tracker", Language: "go", Editor: "vim"},
		{At: day.Add(30 * time.Second), FilePath: "/repos/typing_tracker/main.go", Project: "/repos/typing_tracker", Language: "go", Editor: "vim"},
		{At: day.Add(60 * time.Second), FilePath: "/repos/typing_tracker/main.go", Project: "/repos/typing_tracker", Language: "go", Editor: "vim"},
		{At: day.Add(-24 * time.Hour), FilePath: "/repos/notes/a.md", Project: "/repos/notes", Language: "markdown", Editor: "emacs"},
		{At: day.Add(-24*time.Hour + time.Minute), FilePath: "/repos/notes/a.md", Project: "/repos/notes", Language: "markdown", Editor: "emacs"},
	}
	today := Today(beats, day, 2*time.Minute)
	if !strings.Contains(today, "Today  1m") {
		t.Fatalf("today header:\n%s", today)
	}
	if !strings.Contains(today, "typing_tracker") || strings.Contains(today, "notes") {
		t.Fatalf("today projects:\n%s", today)
	}
	if !strings.Contains(today, "go") || !strings.Contains(today, "vim") {
		t.Fatalf("today breakdown:\n%s", today)
	}

	week := Week(beats, day, 2*time.Minute)
	if !strings.Contains(week, "Last 7 days  2m") {
		t.Fatalf("week header:\n%s", week)
	}
	if !strings.Contains(week, "notes") || !strings.Contains(week, "emacs") {
		t.Fatalf("week breakdown:\n%s", week)
	}
	if strings.Count(week, "\n") < 10 {
		t.Fatalf("week should list each day:\n%s", week)
	}

	empty := Today(nil, day, 2*time.Minute)
	if !strings.Contains(empty, "Today  0s") || !strings.Contains(empty, "(none)") {
		t.Fatalf("empty:\n%s", empty)
	}
}

func TestStatusHidesNothingSensitiveBeyondMetadata(t *testing.T) {
	last := store.Heartbeat{
		At:       time.Date(2026, 10, 17, 7, 4, 1, 0, time.UTC),
		FilePath: "/repos/typing_tracker/main.go",
		Project:  "/repos/typing_tracker",
		Editor:   "vim",
	}
	text := Status(true, "127.0.0.1:57147", last, true)
	if !strings.Contains(text, "Server   up  127.0.0.1:57147") {
		t.Fatalf("status:\n%s", text)
	}
	if !strings.Contains(text, "vim") || !strings.Contains(text, "main.go") || !strings.Contains(text, "typing_tracker") {
		t.Fatalf("status line:\n%s", text)
	}
	if strings.Contains(text, "token") || strings.Contains(text, "/repos/typing_tracker/main.go") {
		t.Fatalf("status leaked extra path or token:\n%s", text)
	}
	down := Status(false, "127.0.0.1:57147", store.Heartbeat{}, false)
	if !strings.Contains(down, "down") || !strings.Contains(down, "none") {
		t.Fatalf("down:\n%s", down)
	}
}
