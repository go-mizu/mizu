package ui

import (
	"fmt"
	"strings"
	"time"
)

// ProgressBar represents a text-based progress bar.
type ProgressBar struct {
	Total     int
	Current   int
	Width     int
	StartTime time.Time
	Message   string
}

// NewProgressBar creates a new progress bar.
func NewProgressBar(total int, message string) *ProgressBar {
	return &ProgressBar{
		Total:     total,
		Width:     40,
		StartTime: time.Now(),
		Message:   message,
	}
}

// Update sets the current progress.
func (p *ProgressBar) Update(current int) {
	p.Current = current
}

// SetMessage sets the progress message.
func (p *ProgressBar) SetMessage(msg string) {
	p.Message = msg
}

// Render returns the progress bar string.
func (p *ProgressBar) Render() string {
	if p.Total == 0 {
		return ""
	}

	percent := float64(p.Current) / float64(p.Total)
	if percent > 1 {
		percent = 1
	}

	filled := int(float64(p.Width) * percent)
	if filled > p.Width {
		filled = p.Width
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", p.Width-filled)

	elapsed := time.Since(p.StartTime)
	var eta time.Duration
	if p.Current > 0 && percent < 1 {
		eta = time.Duration(float64(elapsed) / percent * (1 - percent))
	}

	// Format: "100% |████████████████████████████████████████| [1s:0s]"
	return fmt.Sprintf(" %3.0f%% |%s| [%s:%s]",
		percent*100,
		ProgressCompleteStyle.Render(bar[:filled])+ProgressIncompleteStyle.Render(bar[filled:]),
		formatShortDuration(elapsed),
		formatShortDuration(eta))
}

// RenderWithMessage renders progress with the message above.
func (p *ProgressBar) RenderWithMessage() string {
	return p.Message + "\n" + p.Render()
}

func formatShortDuration(d time.Duration) string {
	if d < 0 {
		return "0s"
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%ds", int(d.Seconds()))
}
