package cli

import (
	"fmt"
	"io"
	"strings"
)

// table formats tabular data for terminal output.
type table struct {
	headers []string
	rows    [][]string
	widths  []int
}

func newTable(headers ...string) *table {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	return &table{
		headers: headers,
		widths:  widths,
	}
}

func (t *table) addRow(cols ...string) {
	// Ensure we have enough columns
	for len(cols) < len(t.headers) {
		cols = append(cols, "")
	}

	// Update column widths
	for i, c := range cols {
		if i < len(t.widths) && len(c) > t.widths[i] {
			t.widths[i] = len(c)
		}
	}

	t.rows = append(t.rows, cols)
}

func (t *table) write(w io.Writer) {
	// Print headers
	for i, h := range t.headers {
		if i > 0 {
			_, _ = fmt.Fprint(w, "  ")
		}
		_, _ = fmt.Fprint(w, padRight(h, t.widths[i]))
	}
	_, _ = fmt.Fprintln(w)

	// Print rows
	for _, row := range t.rows {
		for i, c := range row {
			if i >= len(t.widths) {
				break
			}
			if i > 0 {
				_, _ = fmt.Fprint(w, "  ")
			}
			_, _ = fmt.Fprint(w, padRight(c, t.widths[i]))
		}
		_, _ = fmt.Fprintln(w)
	}
}

// pluralize returns singular or plural form based on count.
func pluralize(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}

// padRight pads s to width with spaces.
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
