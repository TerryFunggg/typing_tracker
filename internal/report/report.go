// Package report prints terminal summaries of local coding time.
package report

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"typing_tracker/internal/store"
	"typing_tracker/internal/summary"
)

// Today reports the local calendar day that contains now.
func Today(beats []store.Heartbeat, now time.Time, gap time.Duration) string {
	now = now.In(time.Local)
	start := dayStart(now)
	end := start.AddDate(0, 0, 1)
	totals := sumRange(beats, start, end, gap)
	var b strings.Builder
	fmt.Fprintf(&b, "Today  %s\n", formatDuration(totals.Total))
	b.WriteByte('\n')
	writeTable(&b, "Project", totals.Project, summary.Labels(keys(totals.Project)))
	b.WriteByte('\n')
	writeTable(&b, "Language", totals.Language, nil)
	b.WriteByte('\n')
	writeTable(&b, "Editor", totals.Editor, nil)
	return b.String()
}

// Week reports the last 7 local days, including today, plus a combined breakdown.
func Week(beats []store.Heartbeat, now time.Time, gap time.Duration) string {
	now = now.In(time.Local)
	start := dayStart(now).AddDate(0, 0, -6)
	end := dayStart(now).AddDate(0, 0, 1)
	totals := sumRange(beats, start, end, gap)
	var b strings.Builder
	fmt.Fprintf(&b, "Last 7 days  %s\n", formatDuration(totals.Total))
	b.WriteByte('\n')
	dayTotals := map[string]time.Duration{}
	var dayOrder []string
	for i := 0; i < 7; i++ {
		day := start.AddDate(0, 0, i)
		label := day.Format("Mon 02 Jan")
		dayOrder = append(dayOrder, label)
		dayTotals[label] = sumRange(beats, day, day.AddDate(0, 0, 1), gap).Total
	}
	writeOrdered(&b, "Day", dayOrder, dayTotals)
	b.WriteByte('\n')
	writeTable(&b, "Project", totals.Project, summary.Labels(keys(totals.Project)))
	b.WriteByte('\n')
	writeTable(&b, "Language", totals.Language, nil)
	b.WriteByte('\n')
	writeTable(&b, "Editor", totals.Editor, nil)
	return b.String()
}

// Status reports whether the server answered and what the newest heartbeat was.
func Status(up bool, addr string, last store.Heartbeat, hasLast bool) string {
	var b strings.Builder
	if up {
		fmt.Fprintf(&b, "Server   up  %s\n", addr)
	} else {
		fmt.Fprintf(&b, "Server   down  %s\n", addr)
	}
	if !hasLast {
		b.WriteString("Last     none\n")
		return b.String()
	}
	project := summary.Labels([]string{last.Project})[last.Project]
	fmt.Fprintf(&b, "Last     %s  %s  %s  %s\n",
		last.At.In(time.Local).Format("2006-01-02 15:04:05"),
		last.Editor,
		project,
		filepath.Base(last.FilePath),
	)
	return b.String()
}

func sumRange(beats []store.Heartbeat, start, end time.Time, gap time.Duration) summary.Totals {
	var slice []summary.Beat
	for _, h := range beats {
		at := h.At
		if !at.Before(start) && at.Before(end) {
			slice = append(slice, summary.Beat{
				At:       at,
				Project:  h.Project,
				Language: h.Language,
				Editor:   h.Editor,
			})
		}
	}
	return summary.Sum(slice, gap)
}

func writeTable(b *strings.Builder, title string, values map[string]time.Duration, labels map[string]string) {
	type row struct {
		name string
		d    time.Duration
	}
	rows := make([]row, 0, len(values))
	for key, d := range values {
		if d <= 0 {
			continue
		}
		name := key
		if labels != nil {
			if label, ok := labels[key]; ok {
				name = label
			}
		}
		rows = append(rows, row{name: name, d: d})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].d == rows[j].d {
			return rows[i].name < rows[j].name
		}
		return rows[i].d > rows[j].d
	})
	if len(rows) == 0 {
		fmt.Fprintf(b, "%s\n  (none)\n", title)
		return
	}
	names := make([]string, len(rows))
	durs := make([]time.Duration, len(rows))
	for i, r := range rows {
		names[i] = r.name
		durs[i] = r.d
	}
	// The header label is part of the width.
	names = append([]string{title}, names...)
	nameW, timeW := widths(names, durs)
	fmt.Fprintf(b, "%-*s  %*s\n", nameW, title, timeW, "Time")
	for _, r := range rows {
		fmt.Fprintf(b, "%-*s  %*s\n", nameW, r.name, timeW, formatDuration(r.d))
	}
}

func writeOrdered(b *strings.Builder, title string, order []string, values map[string]time.Duration) {
	durs := make([]time.Duration, len(order))
	for i, name := range order {
		durs[i] = values[name]
	}
	nameW, timeW := widths(append([]string{title}, order...), durs)
	fmt.Fprintf(b, "%-*s  %*s\n", nameW, title, timeW, "Time")
	for _, name := range order {
		fmt.Fprintf(b, "%-*s  %*s\n", nameW, name, timeW, formatDuration(values[name]))
	}
}

func dayStart(t time.Time) time.Time {
	t = t.In(time.Local)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
}

func widths(names []string, durs []time.Duration) (int, int) {
	nameW := 0
	timeW := len("Time")
	for _, name := range names {
		if n := len(name); n > nameW {
			nameW = n
		}
	}
	for _, d := range durs {
		if n := len(formatDuration(d)); n > timeW {
			timeW = n
		}
	}
	if nameW < 3 {
		nameW = 3
	}
	return nameW, timeW
}

func keys(m map[string]time.Duration) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	secs := int(d / time.Second)
	if secs < 60 {
		return fmt.Sprintf("%ds", secs)
	}
	h := secs / 3600
	m := (secs % 3600) / 60
	if h > 0 && m > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	if h > 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dm", m)
}
