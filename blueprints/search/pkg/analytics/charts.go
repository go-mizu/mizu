package analytics

import (
	"fmt"
	"math"
	"strings"
)

// MermaidPie generates a Mermaid pie chart.
func MermaidPie(title string, data []PieSlice) string {
	var sb strings.Builder
	sb.WriteString("```mermaid\npie title " + title + "\n")
	for _, s := range data {
		sb.WriteString(fmt.Sprintf("    %q : %.1f\n", s.Label, s.Value))
	}
	sb.WriteString("```\n")
	return sb.String()
}

// MermaidXYBar generates a Mermaid XY bar chart.
func MermaidXYBar(title, xLabel, yLabel string, categories []string, values []float64) string {
	var sb strings.Builder
	sb.WriteString("```mermaid\nxychart-beta\n")
	sb.WriteString(fmt.Sprintf("    title %q\n", title))
	sb.WriteString("    x-axis " + formatMermaidCategories(categories) + "\n")
	sb.WriteString(fmt.Sprintf("    y-axis %q\n", yLabel))
	sb.WriteString("    bar " + formatMermaidValues(values) + "\n")
	sb.WriteString("```\n")
	return sb.String()
}

// MermaidXYLine generates a Mermaid XY line chart.
func MermaidXYLine(title, xLabel, yLabel string, categories []string, values []float64) string {
	var sb strings.Builder
	sb.WriteString("```mermaid\nxychart-beta\n")
	sb.WriteString(fmt.Sprintf("    title %q\n", title))
	sb.WriteString("    x-axis " + formatMermaidCategories(categories) + "\n")
	sb.WriteString(fmt.Sprintf("    y-axis %q\n", yLabel))
	sb.WriteString("    line " + formatMermaidValues(values) + "\n")
	sb.WriteString("```\n")
	return sb.String()
}

func formatMermaidCategories(cats []string) string {
	quoted := make([]string, len(cats))
	for i, c := range cats {
		quoted[i] = fmt.Sprintf("%q", c)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

func formatMermaidValues(vals []float64) string {
	parts := make([]string, len(vals))
	for i, v := range vals {
		parts[i] = fmt.Sprintf("%.0f", v)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// ASCIIBar renders a horizontal bar chart using Unicode block characters.
func ASCIIBar(title string, items []BarItem, maxWidth int) string {
	if len(items) == 0 {
		return ""
	}
	if maxWidth <= 0 {
		maxWidth = 40
	}

	// Find max value and max label width
	var maxVal float64
	maxLabelLen := 0
	for _, item := range items {
		if item.Value > maxVal {
			maxVal = item.Value
		}
		if len(item.Label) > maxLabelLen {
			maxLabelLen = len(item.Label)
		}
	}
	if maxLabelLen > 30 {
		maxLabelLen = 30
	}
	if maxVal == 0 {
		maxVal = 1
	}

	var sb strings.Builder
	sb.WriteString("**" + title + "**\n\n")
	sb.WriteString("```\n")

	for _, item := range items {
		label := item.Label
		if len(label) > 30 {
			label = label[:27] + "..."
		}
		// Pad label
		padded := label + strings.Repeat(" ", maxLabelLen-len(label))

		// Calculate bar width
		barLen := int(math.Round(item.Value / maxVal * float64(maxWidth)))
		bar := strings.Repeat("█", barLen)

		sb.WriteString(fmt.Sprintf("%s %s %s\n", padded, bar, formatCount(item.Value)))
	}
	sb.WriteString("```\n")
	return sb.String()
}

// ASCIIHeatmap renders a text-based heatmap grid.
func ASCIIHeatmap(title string, rowLabels, colLabels []string, values [][]float64) string {
	var sb strings.Builder
	sb.WriteString("**" + title + "**\n\n")
	sb.WriteString("```\n")

	// Header
	maxRowLabel := 0
	for _, l := range rowLabels {
		if len(l) > maxRowLabel {
			maxRowLabel = len(l)
		}
	}

	// Column headers
	sb.WriteString(strings.Repeat(" ", maxRowLabel+1))
	for _, cl := range colLabels {
		sb.WriteString(fmt.Sprintf(" %4s", cl))
	}
	sb.WriteString("\n")

	// Find max for scaling
	var maxV float64
	for _, row := range values {
		for _, v := range row {
			if v > maxV {
				maxV = v
			}
		}
	}
	if maxV == 0 {
		maxV = 1
	}

	blocks := []string{"·", "░", "▒", "▓", "█"}

	for i, rl := range rowLabels {
		padded := rl + strings.Repeat(" ", maxRowLabel-len(rl))
		sb.WriteString(padded + " ")
		for j := range colLabels {
			var v float64
			if i < len(values) && j < len(values[i]) {
				v = values[i][j]
			}
			level := int(v / maxV * float64(len(blocks)-1))
			if level >= len(blocks) {
				level = len(blocks) - 1
			}
			sb.WriteString(fmt.Sprintf("   %s ", blocks[level]))
		}
		sb.WriteString("\n")
	}
	sb.WriteString("```\n")
	return sb.String()
}

// MarkdownTable generates a markdown table.
func MarkdownTable(headers []string, rows [][]string) string {
	var sb strings.Builder

	// Header
	sb.WriteString("| " + strings.Join(headers, " | ") + " |\n")

	// Separator
	seps := make([]string, len(headers))
	for i := range seps {
		seps[i] = "---"
	}
	sb.WriteString("| " + strings.Join(seps, " | ") + " |\n")

	// Rows
	for _, row := range rows {
		sb.WriteString("| " + strings.Join(row, " | ") + " |\n")
	}
	return sb.String()
}

func formatCount(v float64) string {
	if v >= 1000000 {
		return fmt.Sprintf("%.1fM", v/1000000)
	}
	if v >= 1000 {
		return fmt.Sprintf("%.1fK", v/1000)
	}
	return fmt.Sprintf("%.0f", v)
}

