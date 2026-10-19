package summary

import (
	"testing"
	"time"
)

func at(sec int) time.Time {
	return time.Unix(1_700_000_000+int64(sec), 0).UTC()
}

func beat(sec int, project, lang, editor string) Beat {
	return Beat{At: at(sec), Project: project, Language: lang, Editor: editor}
}

func TestSumGaps(t *testing.T) {
	gap := 2 * time.Minute
	if got := Sum(nil, gap).Total; got != 0 {
		t.Fatalf("empty = %s", got)
	}
	one := Sum([]Beat{beat(0, "a", "go", "vim")}, gap)
	if one.Total != 0 {
		t.Fatalf("single heartbeat counted %s", one.Total)
	}

	got := Sum([]Beat{
		beat(0, "/repos/typing_tracker", "go", "vim"),
		beat(30, "/repos/typing_tracker", "go", "vim"),
		beat(60, "/repos/notes", "markdown", "emacs"),
		beat(90, "/repos/notes", "markdown", "emacs"),
		beat(90+181, "/repos/notes", "markdown", "emacs"),
	}, gap)
	if got.Total != 90*time.Second {
		t.Fatalf("total = %s, want 90s", got.Total)
	}
	// The gap that crosses projects belongs to the heartbeat at its start.
	if got.Project["/repos/typing_tracker"] != 60*time.Second {
		t.Fatalf("project time = %s", got.Project["/repos/typing_tracker"])
	}
	if got.Language["go"] != 60*time.Second || got.Editor["vim"] != 60*time.Second {
		t.Fatalf("language/editor attribution = %#v %#v", got.Language, got.Editor)
	}
	if got.Project["/repos/notes"] != 30*time.Second || got.Editor["emacs"] != 30*time.Second {
		t.Fatalf("notes gap = %s emacs %s", got.Project["/repos/notes"], got.Editor["emacs"])
	}
}

func TestSumExactGapCounts(t *testing.T) {
	got := Sum([]Beat{
		beat(0, "a", "go", "vim"),
		beat(120, "a", "go", "vim"),
		beat(241, "a", "go", "vim"),
	}, 120*time.Second)
	if got.Total != 120*time.Second {
		t.Fatalf("total = %s", got.Total)
	}
}

func TestLabelsDisambiguate(t *testing.T) {
	labels := Labels([]string{"/work/api", "/home/api", "unknown"})
	if labels["/work/api"] != "work/api" || labels["/home/api"] != "home/api" {
		t.Fatalf("labels %#v", labels)
	}
	if labels["unknown"] != "unknown" {
		t.Fatalf("unknown label %#v", labels)
	}
	one := Labels([]string{"/work/typing_tracker"})
	if one["/work/typing_tracker"] != "typing_tracker" {
		t.Fatalf("single label %#v", one)
	}
}
