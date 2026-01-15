package ui

import (
	"fmt"
	"strings"
)

// ResultsTable renders benchmark results as a formatted table.
type ResultsTable struct {
	ObjectSize int
	Rows       []TableRow
}

// TableRow represents a row in the results table.
type TableRow struct {
	Driver     string
	Threads    int
	Throughput float64 // MB/s

	// TTFB stats in milliseconds
	TTFBAvg int64
	TTFBMin int64
	TTFBP25 int64
	TTFBP50 int64
	TTFBP75 int64
	TTFBP90 int64
	TTFBP99 int64
	TTFBMax int64

	// TTLB stats in milliseconds
	TTLBAvg int64
	TTLBMin int64
	TTLBP25 int64
	TTLBP50 int64
	TTLBP75 int64
	TTLBP90 int64
	TTLBP99 int64
	TTLBMax int64
}

// NewResultsTable creates a new results table.
func NewResultsTable(objectSize int) *ResultsTable {
	return &ResultsTable{
		ObjectSize: objectSize,
		Rows:       make([]TableRow, 0),
	}
}

// AddRow adds a row to the table.
func (t *ResultsTable) AddRow(row TableRow) {
	t.Rows = append(t.Rows, row)
}

// RenderHeader renders the table header for a given object size.
func (t *ResultsTable) RenderHeader() string {
	sizeStr := FormatSizeCompact(t.ObjectSize)

	var sb strings.Builder

	// Title line
	title := fmt.Sprintf("Download performance with %s objects", sizeStr)
	sb.WriteString(title)
	sb.WriteString("\n")

	// Column headers - matching s3-benchmark format
	sb.WriteString("+")
	sb.WriteString(strings.Repeat("-", 10)) // Driver
	sb.WriteString("+")
	sb.WriteString(strings.Repeat("-", 13)) // Throughput
	sb.WriteString("+")
	sb.WriteString(strings.Repeat("-", 56)) // TTFB
	sb.WriteString("+")
	sb.WriteString(strings.Repeat("-", 56)) // TTLB
	sb.WriteString("+\n")

	// Sub-headers
	sb.WriteString(TableBorderStyle.Render("|"))
	sb.WriteString(fmt.Sprintf(" %-8s ", ""))
	sb.WriteString(TableBorderStyle.Render("|"))
	sb.WriteString(fmt.Sprintf(" %-11s ", ""))
	sb.WriteString(TableBorderStyle.Render("|"))
	sb.WriteString(TableHeaderStyle.Render(fmt.Sprintf(" %-54s ", "Time to First Byte (ms)")))
	sb.WriteString(TableBorderStyle.Render("|"))
	sb.WriteString(TableHeaderStyle.Render(fmt.Sprintf(" %-54s ", "Time to Last Byte (ms)")))
	sb.WriteString(TableBorderStyle.Render("|\n"))

	// Column labels
	sb.WriteString(TableBorderStyle.Render("|"))
	sb.WriteString(TableHeaderStyle.Render(fmt.Sprintf(" %-8s ", "Threads")))
	sb.WriteString(TableBorderStyle.Render("|"))
	sb.WriteString(TableHeaderStyle.Render(fmt.Sprintf(" %-11s ", "Throughput")))
	sb.WriteString(TableBorderStyle.Render("|"))
	sb.WriteString(fmt.Sprintf(" %5s %5s %5s %5s %5s %5s %5s %5s ",
		TableHeaderStyle.Render("avg"),
		TableHeaderStyle.Render("min"),
		TableHeaderStyle.Render("p25"),
		TableHeaderStyle.Render("p50"),
		TableHeaderStyle.Render("p75"),
		TableHeaderStyle.Render("p90"),
		TableHeaderStyle.Render("p99"),
		TableHeaderStyle.Render("max")))
	sb.WriteString(TableBorderStyle.Render("|"))
	sb.WriteString(fmt.Sprintf(" %5s %5s %5s %5s %5s %5s %5s %5s ",
		TableHeaderStyle.Render("avg"),
		TableHeaderStyle.Render("min"),
		TableHeaderStyle.Render("p25"),
		TableHeaderStyle.Render("p50"),
		TableHeaderStyle.Render("p75"),
		TableHeaderStyle.Render("p90"),
		TableHeaderStyle.Render("p99"),
		TableHeaderStyle.Render("max")))
	sb.WriteString(TableBorderStyle.Render("|\n"))

	// Separator
	sb.WriteString("+")
	sb.WriteString(strings.Repeat("-", 10))
	sb.WriteString("+")
	sb.WriteString(strings.Repeat("-", 13))
	sb.WriteString("+")
	sb.WriteString(strings.Repeat("-", 56))
	sb.WriteString("+")
	sb.WriteString(strings.Repeat("-", 56))
	sb.WriteString("+\n")

	return sb.String()
}

// RenderRow renders a single result row.
func (t *ResultsTable) RenderRow(row TableRow) string {
	var sb strings.Builder

	sb.WriteString(TableBorderStyle.Render("|"))
	sb.WriteString(fmt.Sprintf(" %8d ", row.Threads))
	sb.WriteString(TableBorderStyle.Render("|"))
	sb.WriteString(ThroughputStyle.Render(fmt.Sprintf(" %8.1f MB/s", row.Throughput)))
	sb.WriteString(TableBorderStyle.Render("|"))

	// TTFB columns
	sb.WriteString(fmt.Sprintf(" %5d %5d %5d %5d %5d %5d %5d %5d ",
		row.TTFBAvg, row.TTFBMin, row.TTFBP25, row.TTFBP50,
		row.TTFBP75, row.TTFBP90, row.TTFBP99, row.TTFBMax))
	sb.WriteString(TableBorderStyle.Render("|"))

	// TTLB columns
	sb.WriteString(fmt.Sprintf(" %5d %5d %5d %5d %5d %5d %5d %5d ",
		row.TTLBAvg, row.TTLBMin, row.TTLBP25, row.TTLBP50,
		row.TTLBP75, row.TTLBP90, row.TTLBP99, row.TTLBMax))
	sb.WriteString(TableBorderStyle.Render("|"))

	return sb.String()
}

// RenderFooter renders the table footer.
func (t *ResultsTable) RenderFooter() string {
	var sb strings.Builder
	sb.WriteString("+")
	sb.WriteString(strings.Repeat("-", 10))
	sb.WriteString("+")
	sb.WriteString(strings.Repeat("-", 13))
	sb.WriteString("+")
	sb.WriteString(strings.Repeat("-", 56))
	sb.WriteString("+")
	sb.WriteString(strings.Repeat("-", 56))
	sb.WriteString("+")
	return sb.String()
}

// Render renders the complete table.
func (t *ResultsTable) Render() string {
	if len(t.Rows) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(t.RenderHeader())
	for _, row := range t.Rows {
		sb.WriteString(t.RenderRow(row))
		sb.WriteString("\n")
	}
	sb.WriteString(t.RenderFooter())
	return sb.String()
}

// FormatSizeCompact returns a compact size string.
func FormatSizeCompact(bytes int) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%d GB", bytes/GB)
	case bytes >= MB:
		return fmt.Sprintf("%d MB", bytes/MB)
	case bytes >= KB:
		return fmt.Sprintf("%d KB", bytes/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// SimpleTable renders a simpler comparison table for the summary.
type SimpleTable struct {
	Headers []string
	Rows    [][]string
}

// NewSimpleTable creates a new simple table.
func NewSimpleTable(headers ...string) *SimpleTable {
	return &SimpleTable{
		Headers: headers,
		Rows:    make([][]string, 0),
	}
}

// AddRow adds a row to the table.
func (t *SimpleTable) AddRow(values ...string) {
	t.Rows = append(t.Rows, values)
}

// Render renders the table.
func (t *SimpleTable) Render() string {
	if len(t.Headers) == 0 {
		return ""
	}

	// Calculate column widths
	widths := make([]int, len(t.Headers))
	for i, h := range t.Headers {
		widths[i] = len(h)
	}
	for _, row := range t.Rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	var sb strings.Builder

	// Header separator
	sb.WriteString("+")
	for _, w := range widths {
		sb.WriteString(strings.Repeat("-", w+2))
		sb.WriteString("+")
	}
	sb.WriteString("\n")

	// Headers
	sb.WriteString(TableBorderStyle.Render("|"))
	for i, h := range t.Headers {
		sb.WriteString(fmt.Sprintf(" %-*s ", widths[i], TableHeaderStyle.Render(h)))
		sb.WriteString(TableBorderStyle.Render("|"))
	}
	sb.WriteString("\n")

	// Separator
	sb.WriteString("+")
	for _, w := range widths {
		sb.WriteString(strings.Repeat("-", w+2))
		sb.WriteString("+")
	}
	sb.WriteString("\n")

	// Rows
	for _, row := range t.Rows {
		sb.WriteString(TableBorderStyle.Render("|"))
		for i := 0; i < len(widths); i++ {
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			sb.WriteString(fmt.Sprintf(" %-*s ", widths[i], cell))
			sb.WriteString(TableBorderStyle.Render("|"))
		}
		sb.WriteString("\n")
	}

	// Footer separator
	sb.WriteString("+")
	for _, w := range widths {
		sb.WriteString(strings.Repeat("-", w+2))
		sb.WriteString("+")
	}

	return sb.String()
}
