package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/mizu/bench/budget"
)

// table renders the budget as the markdown section 3 of the performance
// document carries.
//
// The document is generated from here rather than edited beside here, because a
// table maintained by hand beside a table the tests read is two tables and one
// of them is wrong.
func table() string {
	var b strings.Builder
	for i, group := range budget.Groups() {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "### %s\n\n", group)
		b.WriteString("| ID | Operation | Time | Allocations | Doc | Arrives |\n")
		b.WriteString("|---|---|---|---|---|---|\n")
		for _, r := range budget.Rows() {
			if r.Group != group {
				continue
			}
			fmt.Fprintf(&b, "| `%s` | %s | %s | %s | %s | %s |\n",
				r.ID, r.Op, duration(r.Time), allocs(r.Allocs), r.Doc, arrives(r.Since))
		}
	}
	return b.String()
}

// duration writes a budget the way the table reads it: a whole number of the
// largest unit that keeps it above one, with at most one decimal place. There
// is no point in three significant figures for a target somebody chose.
func duration(d time.Duration) string {
	if d == budget.NoBudget {
		return "n/a"
	}
	for _, u := range []struct {
		size time.Duration
		name string
	}{
		{time.Second, "s"},
		{time.Millisecond, "ms"},
		{time.Microsecond, "us"},
	} {
		if d >= u.size {
			return trim(float64(d)/float64(u.size)) + " " + u.name
		}
	}
	return trim(float64(d)) + " ns"
}

func trim(f float64) string {
	s := fmt.Sprintf("%.1f", f)
	return strings.TrimSuffix(s, ".0")
}

func allocs(n int) string {
	if n == budget.NoBudget {
		return "n/a"
	}
	return fmt.Sprint(n)
}

// arrives says which milestone brings the benchmark. A row that has one is a
// promise about work that has not started, which is worth printing next to the
// numbers so the table reads as the whole plan rather than as a report on the
// part that is finished.
func arrives(since string) string {
	if since == "" {
		return "measured"
	}
	return since
}
