package ui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

const (
	ChartWidth      = 60
	ChartHeight     = 8
	MaxDataPoints   = 60 // 30s at 500ms intervals
	ChartUpdateRate = 250 * time.Millisecond
)

// DriverColors maps driver names to colors.
var DriverColors = map[string]lipgloss.Color{
	"liteio":     lipgloss.Color("#10B981"), // Green
	"minio":      lipgloss.Color("#F59E0B"), // Amber
	"rustfs":     lipgloss.Color("#06B6D4"), // Cyan
	"liteio_mem": lipgloss.Color("#8B5CF6"), // Purple
	"seaweedfs":  lipgloss.Color("#EC4899"), // Pink
	"localstack": lipgloss.Color("#6366F1"), // Indigo
}

// ThroughputChart displays real-time throughput - focused on CURRENT driver only.
type ThroughputChart struct {
	currentDriver string
	data          *RingBuffer
	width         int
	height        int
	// Historical best for each driver
	driverBest map[string]float64
}

// NewThroughputChart creates a new throughput chart.
func NewThroughputChart() *ThroughputChart {
	return &ThroughputChart{
		data:       NewRingBuffer(MaxDataPoints),
		width:      ChartWidth,
		height:     ChartHeight,
		driverBest: make(map[string]float64),
	}
}

// SetCurrentDriver sets the current driver being benchmarked and clears the chart.
func (c *ThroughputChart) SetCurrentDriver(driver string) {
	if c.currentDriver != driver {
		// Save best throughput for previous driver
		if c.currentDriver != "" && c.data.Max() > c.driverBest[c.currentDriver] {
			c.driverBest[c.currentDriver] = c.data.Max()
		}
		c.currentDriver = driver
		c.data.Clear()
	}
}

// AddSample adds a throughput sample for a driver.
func (c *ThroughputChart) AddSample(driver string, throughput float64, timestamp time.Time) {
	// Only add if it's for the current driver
	if driver == c.currentDriver || c.currentDriver == "" {
		c.currentDriver = driver
		c.data.Push(timestamp, throughput)
		// Track best
		if throughput > c.driverBest[driver] {
			c.driverBest[driver] = throughput
		}
	}
}

// SetSize sets the chart dimensions.
func (c *ThroughputChart) SetSize(width, height int) {
	c.width = width
	c.height = height
}

// Clear resets the chart data.
func (c *ThroughputChart) Clear() {
	c.data.Clear()
	c.currentDriver = ""
}

// Render returns the ASCII chart using a clean line-based sparkline approach.
func (c *ThroughputChart) Render() string {
	var sb strings.Builder

	// Title with current driver
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorCyan)
	if c.currentDriver != "" {
		color := getDriverColor(c.currentDriver)
		driverStyle := lipgloss.NewStyle().Bold(true).Foreground(color)
		sb.WriteString(titleStyle.Render("Throughput: "))
		sb.WriteString(driverStyle.Render(c.currentDriver))
	} else {
		sb.WriteString(titleStyle.Render("Throughput"))
	}
	sb.WriteString("\n\n")

	values := c.data.Values()
	if len(values) == 0 {
		sb.WriteString(MutedStyle.Render("  Waiting for data...\n"))
		return sb.String()
	}

	// Find max for scaling
	maxVal := c.data.Max()
	if maxVal < 10 {
		maxVal = 10
	}
	maxVal = niceMax(maxVal)

	// Chart dimensions
	chartWidth := c.width - 10
	if chartWidth < 20 {
		chartWidth = 20
	}

	// Render using braille-like characters for smoother lines
	for row := c.height - 1; row >= 0; row-- {
		// Y-axis label
		yVal := maxVal * float64(row+1) / float64(c.height)
		sb.WriteString(MutedStyle.Render(fmt.Sprintf("%6.0f │", yVal)))

		// Render this row
		for col := 0; col < chartWidth; col++ {
			// Map column to data index
			dataIdx := col * len(values) / chartWidth
			if dataIdx >= len(values) {
				dataIdx = len(values) - 1
			}

			val := values[dataIdx]
			normalizedVal := val / maxVal
			targetRow := int(normalizedVal * float64(c.height))

			color := getDriverColor(c.currentDriver)
			style := lipgloss.NewStyle().Foreground(color)

			if targetRow > row {
				// Below the line - fill with block
				sb.WriteString(style.Render("█"))
			} else if targetRow == row {
				// At the line - use partial block
				fraction := (normalizedVal * float64(c.height)) - float64(row)
				char := getBlockChar(fraction)
				sb.WriteString(style.Render(string(char)))
			} else {
				sb.WriteString(" ")
			}
		}
		sb.WriteString("\n")
	}

	// X-axis
	sb.WriteString(MutedStyle.Render("       └" + strings.Repeat("─", chartWidth)))
	sb.WriteString("\n")

	// Stats line
	current := c.data.Last()
	avg := c.data.Average()
	max := c.data.Max()
	statsStyle := lipgloss.NewStyle().Foreground(getDriverColor(c.currentDriver))
	sb.WriteString(fmt.Sprintf("        Current: %s  Avg: %s  Peak: %s",
		statsStyle.Render(fmt.Sprintf("%.0f MB/s", current)),
		MutedStyle.Render(fmt.Sprintf("%.0f", avg)),
		MutedStyle.Render(fmt.Sprintf("%.0f", max))))

	return sb.String()
}

// GetDriverBest returns the best throughput for a driver.
func (c *ThroughputChart) GetDriverBest(driver string) float64 {
	return c.driverBest[driver]
}

// getDriverColor returns the color for a driver.
func getDriverColor(driver string) lipgloss.Color {
	if color, ok := DriverColors[driver]; ok {
		return color
	}
	return ColorGreen
}

// getBlockChar returns a Unicode block character for the given fraction (0-1).
func getBlockChar(fraction float64) rune {
	if fraction <= 0 {
		return ' '
	}
	if fraction >= 1 {
		return '█'
	}

	// Use 8-step Unicode blocks for vertical bars
	blocks := []rune{' ', '▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
	idx := int(fraction * 8)
	if idx >= len(blocks) {
		idx = len(blocks) - 1
	}
	return blocks[idx]
}

// niceMax rounds up to a "nice" maximum value for the Y-axis.
func niceMax(val float64) float64 {
	if val <= 0 {
		return 100
	}

	magnitude := math.Pow(10, math.Floor(math.Log10(val)))
	normalized := val / magnitude

	var nice float64
	switch {
	case normalized <= 1:
		nice = 1
	case normalized <= 2:
		nice = 2
	case normalized <= 5:
		nice = 5
	default:
		nice = 10
	}

	return nice * magnitude
}

// Sparkline creates a compact sparkline visualization.
type Sparkline struct {
	values []float64
	width  int
	label  string
}

// NewSparkline creates a new sparkline.
func NewSparkline(width int, label string) *Sparkline {
	return &Sparkline{
		values: make([]float64, 0, width),
		width:  width,
		label:  label,
	}
}

// Add adds a value to the sparkline.
func (s *Sparkline) Add(val float64) {
	s.values = append(s.values, val)
	if len(s.values) > s.width {
		s.values = s.values[1:]
	}
}

// Clear resets the sparkline.
func (s *Sparkline) Clear() {
	s.values = s.values[:0]
}

// Last returns the last value.
func (s *Sparkline) Last() float64 {
	if len(s.values) == 0 {
		return 0
	}
	return s.values[len(s.values)-1]
}

// Render returns the sparkline with label and current value.
func (s *Sparkline) Render(color lipgloss.Color) string {
	var sb strings.Builder

	style := lipgloss.NewStyle().Foreground(color)

	// Label
	sb.WriteString(MutedStyle.Render(fmt.Sprintf("%-12s ", s.label)))

	// Sparkline
	if len(s.values) == 0 {
		sb.WriteString(MutedStyle.Render(strings.Repeat("─", s.width)))
		sb.WriteString(MutedStyle.Render("  --"))
	} else {
		// Find min/max
		min, max := s.values[0], s.values[0]
		for _, v := range s.values {
			if v < min {
				min = v
			}
			if v > max {
				max = v
			}
		}

		sparkChars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

		for _, v := range s.values {
			var idx int
			if max > min {
				normalized := (v - min) / (max - min)
				idx = int(normalized * 7)
				if idx > 7 {
					idx = 7
				}
			} else {
				idx = 4 // Middle if all values are same
			}
			sb.WriteString(style.Render(string(sparkChars[idx])))
		}

		// Pad if needed
		for i := len(s.values); i < s.width; i++ {
			sb.WriteString(MutedStyle.Render("─"))
		}

		// Current value
		sb.WriteString(style.Render(fmt.Sprintf("  %.0f", s.Last())))
	}

	return sb.String()
}

// DriverSparklines manages sparklines for multiple drivers.
type DriverSparklines struct {
	sparklines map[string]*Sparkline
	width      int
}

// NewDriverSparklines creates a new driver sparklines manager.
func NewDriverSparklines(width int) *DriverSparklines {
	return &DriverSparklines{
		sparklines: make(map[string]*Sparkline),
		width:      width,
	}
}

// AddSample adds a throughput sample for a driver.
func (d *DriverSparklines) AddSample(driver string, throughput float64) {
	if _, ok := d.sparklines[driver]; !ok {
		d.sparklines[driver] = NewSparkline(d.width, driver)
	}
	d.sparklines[driver].Add(throughput)
}

// Render returns all sparklines.
func (d *DriverSparklines) Render() string {
	if len(d.sparklines) == 0 {
		return MutedStyle.Render("No data yet")
	}

	var sb strings.Builder
	for driver, sparkline := range d.sparklines {
		color := getDriverColor(driver)
		sb.WriteString(sparkline.Render(color))
		sb.WriteString("\n")
	}

	return sb.String()
}

// Clear resets all sparklines.
func (d *DriverSparklines) Clear() {
	for _, s := range d.sparklines {
		s.Clear()
	}
}
