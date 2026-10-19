// Package summary turns heartbeats into durations.
package summary

import (
	"path/filepath"
	"sort"
	"time"
)

// Beat is one stored activity point.
type Beat struct {
	At       time.Time
	Project  string
	Language string
	Editor   string
}

// Totals is coding time attributed to the heartbeat at the start of each gap.
type Totals struct {
	Total    time.Duration
	Project  map[string]time.Duration
	Language map[string]time.Duration
	Editor   map[string]time.Duration
}

// Sum adds gaps between consecutive heartbeats that are within gap.
// A longer space ends the session and adds nothing.
func Sum(beats []Beat, gap time.Duration) Totals {
	out := Totals{
		Project:  map[string]time.Duration{},
		Language: map[string]time.Duration{},
		Editor:   map[string]time.Duration{},
	}
	if len(beats) < 2 || gap <= 0 {
		return out
	}
	sorted := append([]Beat(nil), beats...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].At.Before(sorted[j].At)
	})
	for i := 1; i < len(sorted); i++ {
		delta := sorted[i].At.Sub(sorted[i-1].At)
		if delta <= 0 || delta > gap {
			continue
		}
		prev := sorted[i-1]
		out.Total += delta
		out.Project[prev.Project] += delta
		out.Language[prev.Language] += delta
		out.Editor[prev.Editor] += delta
	}
	return out
}

// Labels turns project roots into short names. Two roots with the same base
// name get a parent prefix so they stay distinct.
func Labels(roots []string) map[string]string {
	seen := map[string]struct{}{}
	var unique []string
	for _, root := range roots {
		if _, ok := seen[root]; ok {
			continue
		}
		seen[root] = struct{}{}
		unique = append(unique, root)
	}
	count := map[string]int{}
	for _, root := range unique {
		count[filepath.Base(root)]++
	}
	out := make(map[string]string, len(unique))
	for _, root := range unique {
		if root == "" || root == "unknown" {
			out[root] = "unknown"
			continue
		}
		base := filepath.Base(root)
		if count[base] > 1 {
			out[root] = filepath.Base(filepath.Dir(root)) + "/" + base
		} else {
			out[root] = base
		}
	}
	return out
}
