package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Phase represents the current benchmark phase.
type Phase int

const (
	PhaseInit Phase = iota
	PhaseSetup
	PhaseBenchmark
	PhaseCleanup
	PhaseDone
)

// String returns the phase name.
func (p Phase) String() string {
	switch p {
	case PhaseInit:
		return "INIT"
	case PhaseSetup:
		return "SETUP"
	case PhaseBenchmark:
		return "BENCHMARK"
	case PhaseCleanup:
		return "CLEANUP"
	case PhaseDone:
		return "DONE"
	default:
		return "UNKNOWN"
	}
}

// Model is the Bubbletea model for the benchmark UI.
type Model struct {
	// Configuration
	Width  int
	Height int

	// State
	Phase         Phase
	CurrentDriver string
	ObjectSize    int
	Threads       int
	Progress      int
	ProgressTotal int
	ProgressMsg   string

	// Output buffer
	Output strings.Builder

	// Progress bar
	progressBar *ProgressBar

	// Current table
	currentTable *ResultsTable

	// Results
	Results []BenchmarkResultMsg

	// Timing
	StartTime time.Time

	// Error state
	Err error

	// Quit flag
	Quitting bool
}

// BenchmarkResultMsg is sent when a benchmark completes.
type BenchmarkResultMsg struct {
	Driver     string
	ObjectSize int
	Threads    int
	Throughput float64
	TTFBAvg    time.Duration
	TTFBMin    time.Duration
	TTFBP25    time.Duration
	TTFBP50    time.Duration
	TTFBP75    time.Duration
	TTFBP90    time.Duration
	TTFBP99    time.Duration
	TTFBMax    time.Duration
	TTLBAvg    time.Duration
	TTLBMin    time.Duration
	TTLBP25    time.Duration
	TTLBP50    time.Duration
	TTLBP75    time.Duration
	TTLBP90    time.Duration
	TTLBP99    time.Duration
	TTLBMax    time.Duration
}

// PhaseChangeMsg signals a phase change.
type PhaseChangeMsg struct {
	Phase  Phase
	Driver string
}

// ProgressMsg updates progress.
type ProgressMsg struct {
	Current int
	Total   int
	Message string
}

// LogMsg adds a log message.
type LogMsg struct {
	Message string
}

// SectionHeaderMsg starts a new results section.
type SectionHeaderMsg struct {
	ObjectSize int
}

// ErrorMsg signals an error.
type ErrorMsg struct {
	Err error
}

// QuitMsg signals to quit.
type QuitMsg struct{}

// NewModel creates a new UI model.
func NewModel() Model {
	return Model{
		Width:     120,
		Height:    40,
		Phase:     PhaseInit,
		StartTime: time.Now(),
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.Quitting = true
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height

	case PhaseChangeMsg:
		m.Phase = msg.Phase
		m.CurrentDriver = msg.Driver
		m.Output.WriteString("\n")
		m.Output.WriteString(RenderDivider(msg.Phase.String(), m.Width))
		m.Output.WriteString("\n\n")

	case ProgressMsg:
		m.Progress = msg.Current
		m.ProgressTotal = msg.Total
		m.ProgressMsg = msg.Message
		if m.progressBar == nil {
			m.progressBar = NewProgressBar(msg.Total, msg.Message)
		}
		m.progressBar.Update(msg.Current)
		m.progressBar.SetMessage(msg.Message)

	case LogMsg:
		m.Output.WriteString(msg.Message)
		m.Output.WriteString("\n")

	case SectionHeaderMsg:
		m.ObjectSize = msg.ObjectSize
		m.currentTable = NewResultsTable(msg.ObjectSize)
		m.Output.WriteString("\n")
		m.Output.WriteString(m.currentTable.RenderHeader())

	case BenchmarkResultMsg:
		m.Results = append(m.Results, msg)
		if m.currentTable != nil {
			row := TableRow{
				Driver:     msg.Driver,
				Threads:    msg.Threads,
				Throughput: msg.Throughput,
				TTFBAvg:    msg.TTFBAvg.Milliseconds(),
				TTFBMin:    msg.TTFBMin.Milliseconds(),
				TTFBP25:    msg.TTFBP25.Milliseconds(),
				TTFBP50:    msg.TTFBP50.Milliseconds(),
				TTFBP75:    msg.TTFBP75.Milliseconds(),
				TTFBP90:    msg.TTFBP90.Milliseconds(),
				TTFBP99:    msg.TTFBP99.Milliseconds(),
				TTFBMax:    msg.TTFBMax.Milliseconds(),
				TTLBAvg:    msg.TTLBAvg.Milliseconds(),
				TTLBMin:    msg.TTLBMin.Milliseconds(),
				TTLBP25:    msg.TTLBP25.Milliseconds(),
				TTLBP50:    msg.TTLBP50.Milliseconds(),
				TTLBP75:    msg.TTLBP75.Milliseconds(),
				TTLBP90:    msg.TTLBP90.Milliseconds(),
				TTLBP99:    msg.TTLBP99.Milliseconds(),
				TTLBMax:    msg.TTLBMax.Milliseconds(),
			}
			m.currentTable.AddRow(row)
			m.Output.WriteString(m.currentTable.RenderRow(row))
			m.Output.WriteString("\n")
		}

	case ErrorMsg:
		m.Err = msg.Err
		m.Output.WriteString(ErrorStyle.Render(fmt.Sprintf("[ERROR] %v", msg.Err)))
		m.Output.WriteString("\n")

	case QuitMsg:
		m.Quitting = true
		return m, tea.Quit
	}

	return m, nil
}

// View implements tea.Model.
func (m Model) View() string {
	if m.Quitting {
		return m.Output.String() + "\n"
	}

	var sb strings.Builder

	// Header
	sb.WriteString(TitleStyle.Render("S3 Benchmark"))
	sb.WriteString(" - Comparing S3-compatible storage backends\n\n")

	// Output buffer
	sb.WriteString(m.Output.String())

	// Current progress
	if m.progressBar != nil && m.Phase != PhaseDone {
		sb.WriteString(m.progressBar.RenderWithMessage())
		sb.WriteString("\n")
	}

	// Current table footer if we have a table
	if m.currentTable != nil && len(m.currentTable.Rows) > 0 && m.Phase == PhaseBenchmark {
		sb.WriteString(m.currentTable.RenderFooter())
		sb.WriteString("\n")
	}

	// Footer with elapsed time
	elapsed := time.Since(m.StartTime)
	sb.WriteString("\n")
	sb.WriteString(MutedStyle.Render(fmt.Sprintf("Elapsed: %s | Phase: %s", elapsed.Round(time.Second), m.Phase)))
	sb.WriteString("\n")
	sb.WriteString(MutedStyle.Render("Press q or Ctrl+C to quit"))

	return sb.String()
}
