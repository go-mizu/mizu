// Package ui provides reusable Bubbletea models for CLI progress display.
package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ItemStatus represents the status of a tracked progress item.
type ItemStatus int

const (
	StatusPending ItemStatus = iota
	StatusActive
	StatusDone
	StatusError
)

// ProgressItem represents one tracked operation (e.g., one file download).
type ProgressItem struct {
	Label  string
	Done   int64
	Total  int64 // 0 = indeterminate
	Status ItemStatus
	Err    error
}

// Pct returns the completion fraction [0..1].
func (p ProgressItem) Pct() float64 {
	if p.Total <= 0 {
		return 0
	}
	if p.Done >= p.Total {
		return 1
	}
	return float64(p.Done) / float64(p.Total)
}

// ProgressModel is a Bubbletea model for live multi-item progress display.
type ProgressModel struct {
	Title    string
	Items    []ProgressItem
	Overall  ProgressItem // aggregate progress
	Speed    int64        // bytes or items per second
	Unit     string       // "B/s", "rec/s", etc.
	Elapsed  time.Duration
	ETA      time.Duration
	Finished bool
	FinalMsg string

	bars    []progress.Model
	overall progress.Model
	start   time.Time
	width   int
}

// Msg types for communicating with the model from goroutines.

// ItemUpdateMsg updates a specific item's progress.
type ItemUpdateMsg struct {
	Index  int
	Done   int64
	Total  int64
	Status ItemStatus
	Err    error
}

// OverallUpdateMsg updates the aggregate stats.
type OverallUpdateMsg struct {
	Done    int64
	Total   int64
	Speed   int64
	Elapsed time.Duration
	ETA     time.Duration
}

// DoneMsg signals the program to display a final message and quit.
type DoneMsg struct {
	Msg string
	Err error
}

// TickMsg is a periodic refresh trigger.
type TickMsg struct{}

var tickCmd = tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg { return TickMsg{} })

// NewProgressModel creates a ProgressModel with the given title and item labels.
func NewProgressModel(title string, labels []string) ProgressModel {
	items := make([]ProgressItem, len(labels))
	for i, l := range labels {
		items[i] = ProgressItem{Label: l, Status: StatusPending}
	}
	bars := make([]progress.Model, len(labels))
	for i := range bars {
		bars[i] = progress.New(
			progress.WithDefaultGradient(),
			progress.WithoutPercentage(),
			progress.WithWidth(40),
		)
	}
	overall := progress.New(
		progress.WithDefaultGradient(),
		progress.WithoutPercentage(),
		progress.WithWidth(40),
	)
	return ProgressModel{
		Title:   title,
		Items:   items,
		bars:    bars,
		overall: overall,
		start:   time.Now(),
		width:   80,
	}
}

// Init starts the tick loop.
func (m ProgressModel) Init() tea.Cmd {
	return tickCmd
}

// Update handles messages.
func (m ProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width

	case ItemUpdateMsg:
		if msg.Index >= 0 && msg.Index < len(m.Items) {
			m.Items[msg.Index].Done = msg.Done
			m.Items[msg.Index].Total = msg.Total
			m.Items[msg.Index].Status = msg.Status
			m.Items[msg.Index].Err = msg.Err
		}
		return m, tickCmd

	case OverallUpdateMsg:
		m.Overall.Done = msg.Done
		m.Overall.Total = msg.Total
		m.Speed = msg.Speed
		m.Elapsed = msg.Elapsed
		m.ETA = msg.ETA
		return m, tickCmd

	case DoneMsg:
		m.Finished = true
		m.FinalMsg = msg.Msg
		if msg.Err != nil {
			m.FinalMsg = "Error: " + msg.Err.Error()
		}
		return m, tea.Quit

	case TickMsg:
		return m, tickCmd

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}
	return m, nil
}

// View renders the progress display.
func (m ProgressModel) View() string {
	if m.Finished {
		st := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#34A853"))
		return st.Render("  ✓ "+m.FinalMsg) + "\n"
	}

	var sb strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#4285F4"))
	sb.WriteString("\n  " + titleStyle.Render(m.Title) + "\n")
	sep := strings.Repeat("─", min(m.width-4, 60))
	sb.WriteString("  " + lipgloss.NewStyle().Foreground(lipgloss.Color("#9AA0A6")).Render(sep) + "\n")

	// Per-item bars
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9AA0A6"))
	for i, item := range m.Items {
		label := item.Label
		if len(label) > 50 {
			label = "..." + label[len(label)-47:]
		}
		sb.WriteString("  " + labelStyle.Render(label) + "\n")
		sb.WriteString("  ")

		switch item.Status {
		case StatusPending:
			sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#9AA0A6")).Render("  waiting…"))
		case StatusDone:
			sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#34A853")).Bold(true).Render("  ✓ done"))
		case StatusError:
			sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#EA4335")).Render("  ✗ error"))
		default:
			if i < len(m.bars) {
				sb.WriteString(m.bars[i].ViewAs(item.Pct()))
				sb.WriteString(fmt.Sprintf(" %s / %s",
					fmtBytes(item.Done), fmtBytes(item.Total)))
			}
		}
		sb.WriteString("\n")
	}

	// Overall separator
	sb.WriteString("  " + lipgloss.NewStyle().Foreground(lipgloss.Color("#9AA0A6")).Render(sep) + "\n")

	// Overall progress
	sb.WriteString("  Overall  ")
	sb.WriteString(m.overall.ViewAs(m.Overall.Pct()))
	sb.WriteString(fmt.Sprintf("  %s / %s\n", fmtBytes(m.Overall.Done), fmtBytes(m.Overall.Total)))

	// Speed + ETA
	statsStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9AA0A6"))
	sb.WriteString("  " + statsStyle.Render(fmt.Sprintf("Speed  %s   ETA  %s   Elapsed  %s",
		fmtBytesPerSec(m.Speed),
		fmtDuration(m.ETA),
		fmtDuration(m.Elapsed),
	)) + "\n")

	return sb.String()
}

func fmtBytes(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%d B", n)
	}
}

func fmtBytesPerSec(n int64) string {
	if n <= 0 {
		return "—"
	}
	return fmtBytes(n) + "/s"
}

func fmtDuration(d time.Duration) string {
	if d <= 0 {
		return "—"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
}
