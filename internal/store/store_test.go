package store

import (
	"path/filepath"
	"testing"
	"time"
)

func TestInsertAndRange(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "tracker.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	start := time.Date(2026, 10, 17, 10, 0, 0, 0, time.UTC)
	first := Heartbeat{
		At: start, FilePath: "/work/main.go", Project: "/work", Branch: "main",
		Language: "go", Editor: "vim", IsWrite: false,
	}
	second := first
	second.At = start.Add(30 * time.Second)
	second.IsWrite = true
	if err := db.Insert(first); err != nil {
		t.Fatal(err)
	}
	if err := db.Insert(second); err != nil {
		t.Fatal(err)
	}

	got, err := db.Between(start, start.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || !got[1].IsWrite || got[0].Language != "go" {
		t.Fatalf("between = %#v", got)
	}
	latest, ok, err := db.Latest()
	if err != nil || !ok {
		t.Fatal(err, ok)
	}
	if !latest.At.Equal(second.At) || latest.FilePath != "/work/main.go" {
		t.Fatalf("latest = %#v", latest)
	}
	none, err := db.Between(start.Add(time.Hour), start.Add(2*time.Hour))
	if err != nil || len(none) != 0 {
		t.Fatalf("empty range %#v %v", none, err)
	}
}
