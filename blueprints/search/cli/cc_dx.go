package cli

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	liptable "github.com/charmbracelet/lipgloss/table"
)

type ccTableOptions struct {
	RightAlignCols map[int]bool
}

func ccStatusChip(kind, text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		text = "-"
	}

	fg := mutedColor

	switch kind {
	case "ok":
		fg = secondaryColor
	case "warn":
		fg = warningColor
	case "error":
		fg = errorColor
	case "info":
		fg = primaryColor
	}

	return lipgloss.NewStyle().
		Foreground(fg).
		Bold(true).
		Padding(0, 1).
		Render("[" + text + "]")
}

func ccRenderKVCard(title string, rows [][2]string) string {
	if len(rows) == 0 {
		return ""
	}
	maxKey := 0
	for _, kv := range rows {
		if len(kv[0]) > maxKey {
			maxKey = len(kv[0])
		}
	}

	var b strings.Builder
	if strings.TrimSpace(title) != "" {
		b.WriteString(infoStyle.Bold(true).Render(title))
		b.WriteString("\n")
	}
	for i, kv := range rows {
		key := kv[0]
		val := kv[1]
		b.WriteString(labelStyle.Render(fmt.Sprintf("%-*s", maxKey, key)))
		b.WriteString("  ")
		b.WriteString(val)
		if i < len(rows)-1 {
			b.WriteString("\n")
		}
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(mutedColor).
		Padding(0, 1).
		Render(b.String())
}

func ccRenderHintBox(title string, lines []string) string {
	filtered := make([]string, 0, len(lines))
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln != "" {
			filtered = append(filtered, ln)
		}
	}
	if len(filtered) == 0 {
		return ""
	}

	var b strings.Builder
	if title != "" {
		b.WriteString(infoStyle.Bold(true).Render(title))
		b.WriteString("\n")
	}
	for i, ln := range filtered {
		b.WriteString("• ")
		b.WriteString(ln)
		if i < len(filtered)-1 {
			b.WriteString("\n")
		}
	}
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(mutedColor).
		Padding(0, 1).
		Render(b.String())
}

func ccRenderTable(headers []string, rows [][]string, opts ccTableOptions) string {
	t := liptable.New().
		Headers(headers...).
		Rows(rows...).
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(mutedColor)).
		BorderHeader(true).
		BorderColumn(false).
		BorderRow(false).
		Wrap(false)

	return t.StyleFunc(func(row, col int) lipgloss.Style {
		if row == liptable.HeaderRow {
			st := lipgloss.NewStyle().
				Bold(true).
				Foreground(primaryColor).
				Padding(0, 1)
			if opts.RightAlignCols != nil && opts.RightAlignCols[col] {
				st = st.Align(lipgloss.Right)
			}
			return st
		}

		st := lipgloss.NewStyle().Padding(0, 1)
		if row%2 == 1 {
			st = st.Foreground(lipgloss.AdaptiveColor{Light: "240", Dark: "245"})
		}
		if opts.RightAlignCols != nil && opts.RightAlignCols[col] {
			st = st.Align(lipgloss.Right)
		}
		return st
	}).String()
}

func ccRenderTwoCards(left, right string) string {
	left = strings.TrimRight(left, "\n")
	right = strings.TrimRight(right, "\n")
	switch {
	case left == "":
		return right
	case right == "":
		return left
	default:
		return lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right)
	}
}

func ccPct(part, total int64) string {
	if total <= 0 {
		return "-"
	}
	return fmt.Sprintf("%.1f%%", 100*float64(part)/float64(total))
}
