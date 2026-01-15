package ui

import "time"

// ViewMode represents the current view mode.
type ViewMode int

const (
	ViewDashboard ViewMode = iota
	ViewDetails
	ViewLogs
	ViewHelp
)

// String returns the view mode name.
func (v ViewMode) String() string {
	switch v {
	case ViewDashboard:
		return "Dashboard"
	case ViewDetails:
		return "Details"
	case ViewLogs:
		return "Logs"
	case ViewHelp:
		return "Help"
	default:
		return "Unknown"
	}
}

// ThroughputSampleMsg is sent for real-time chart updates.
type ThroughputSampleMsg struct {
	Driver     string
	Throughput float64 // MB/s
	Timestamp  time.Time
}

// DriverProgressMsg updates per-driver progress.
type DriverProgressMsg struct {
	Driver     string
	Completed  int
	Total      int
	Throughput float64 // Current throughput MB/s
}

// ConfigChangeMsg signals a configuration change.
type ConfigChangeMsg struct {
	ObjectSize int
	Threads    int
}

// ViewChangeMsg changes the current view.
type ViewChangeMsg struct {
	View ViewMode
}

// PauseMsg toggles pause state.
type PauseMsg struct {
	Paused bool
}

// TickMsg is sent for periodic updates.
type TickMsg struct {
	Time time.Time
}

// ChartDataPoint represents a single data point for the chart.
type ChartDataPoint struct {
	Timestamp  time.Time
	Throughput float64
}
