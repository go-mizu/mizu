package ui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

const (
	ChartWidth      = 50
	ChartHeight     = 10
	MaxDataPoints   = 120 // 30s at 250ms intervals
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

// ThroughputChart displays real-time throughput for multiple drivers.
type ThroughputChart struct {
	series map[string]*RingBuffer
	width  int
	height int
}

// NewThroughputChart creates a new throughput chart.
func NewThroughputChart() *ThroughputChart {
	return &ThroughputChart{
		series: make(map[string]*RingBuffer),
		width:  ChartWidth,
		height: ChartHeight,
	}
}

// AddSample adds a throughput sample for a driver.
func (c *ThroughputChart) AddSample(driver string, throughput float64, timestamp time.Time) {
	if _, ok := c.series[driver]; !ok {
		c.series[driver] = NewRingBuffer(MaxDataPoints)
	}
	c.series[driver].Push(timestamp, throughput)
}

// SetSize sets the chart dimensions.
func (c *ThroughputChart) SetSize(width, height int) {
	c.width = width
	c.height = height
}

// Clear resets all series data.
func (c *ThroughputChart) Clear() {
	for _, buf := range c.series {
		buf.Clear()
	}
}

// Render returns the ASCII chart.
func (c *ThroughputChart) Render() string {
	if len(c.series) == 0 {
		return c.renderEmpty()
	}

	// Find global max for scaling
	var maxVal float64 = 100 // Minimum scale
	for _, buf := range c.series {
		if m := buf.Max(); m > maxVal {
			maxVal = m
		}
	}

	// Round max up to nice number
	maxVal = niceMax(maxVal)

	var sb strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorCyan)
	sb.WriteString(titleStyle.Render("Throughput (MB/s)"))
	sb.WriteString("\n")

	// Chart area with Y-axis labels
	chartWidth := c.width - 8 // Leave space for Y-axis labels
	if chartWidth < 20 {
		chartWidth = 20
	}

	// Render chart lines
	for row := c.height - 1; row >= 0; row-- {
		// Y-axis label
		yVal := maxVal * float64(row+1) / float64(c.height)
		sb.WriteString(MutedStyle.Render(fmt.Sprintf("%5.0f ", yVal)))
		sb.WriteString(MutedStyle.Render("│"))

		// Render data points for this row
		line := c.renderRow(row, chartWidth, maxVal)
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	// X-axis
	sb.WriteString(MutedStyle.Render("      └"))
	sb.WriteString(MutedStyle.Render(strings.Repeat("─", chartWidth)))
	sb.WriteString("\n")

	// X-axis labels
	sb.WriteString(MutedStyle.Render("       -30s"))
	spacing := chartWidth - 15
	if spacing > 0 {
		sb.WriteString(strings.Repeat(" ", spacing/2))
		sb.WriteString(MutedStyle.Render("-15s"))
		sb.WriteString(strings.Repeat(" ", spacing/2))
	}
	sb.WriteString(MutedStyle.Render("now"))
	sb.WriteString("\n")

	// Legend
	sb.WriteString(c.renderLegend())

	return sb.String()
}

// renderRow renders a single row of the chart.
func (c *ThroughputChart) renderRow(row, width int, maxVal float64) string {
	if maxVal == 0 {
		return strings.Repeat(" ", width)
	}

	// Build character array
	chars := make([]rune, width)
	colors := make([]lipgloss.Color, width)
	for i := range chars {
		chars[i] = ' '
	}

	// For each driver, plot their data
	for driver, buf := range c.series {
		values := buf.Values()
		if len(values) == 0 {
			continue
		}

		color := getDriverColor(driver)

		// Map values to chart width
		for i, val := range values {
			x := i * width / MaxDataPoints
			if x >= width {
				x = width - 1
			}

			// Calculate which row this value belongs to
			normalizedVal := val / maxVal
			targetRow := int(normalizedVal * float64(c.height))
			if targetRow > c.height {
				targetRow = c.height
			}

			if targetRow == row+1 || (row == c.height-1 && targetRow >= c.height) {
				// Use block character based on fraction within row
				fraction := (normalizedVal * float64(c.height)) - float64(row)
				char := getBlockChar(fraction)
				if chars[x] == ' ' || char > chars[x] {
					chars[x] = char
					colors[x] = color
				}
			} else if targetRow > row+1 {
				// Fill below with full block
				if chars[x] == ' ' {
					chars[x] = '█'
					colors[x] = color
				}
			}
		}
	}

	// Build styled string
	var result strings.Builder
	for i, ch := range chars {
		if ch != ' ' && colors[i] != "" {
			style := lipgloss.NewStyle().Foreground(colors[i])
			result.WriteString(style.Render(string(ch)))
		} else {
			result.WriteRune(ch)
		}
	}

	return result.String()
}

// renderLegend renders the chart legend.
func (c *ThroughputChart) renderLegend() string {
	if len(c.series) == 0 {
		return ""
	}

	var parts []string
	for driver, buf := range c.series {
		color := getDriverColor(driver)
		style := lipgloss.NewStyle().Foreground(color)
		current := buf.Last()
		avg := buf.Average()
		legend := fmt.Sprintf("%s %.0f/%.0f", style.Render("●"), current, avg)
		parts = append(parts, style.Render(driver)+": "+legend)
	}

	return MutedStyle.Render("       ") + strings.Join(parts, "  ")
}

// renderEmpty renders placeholder when no data.
func (c *ThroughputChart) renderEmpty() string {
	var sb strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorCyan)
	sb.WriteString(titleStyle.Render("Throughput (MB/s)"))
	sb.WriteString("\n")

	for i := 0; i < c.height; i++ {
		sb.WriteString(MutedStyle.Render("      │"))
		sb.WriteString(strings.Repeat(" ", c.width-7))
		sb.WriteString("\n")
	}

	sb.WriteString(MutedStyle.Render("      └"))
	sb.WriteString(MutedStyle.Render(strings.Repeat("─", c.width-7)))
	sb.WriteString("\n")

	sb.WriteString(MutedStyle.Render("       Waiting for data..."))

	return sb.String()
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

	// Use 8-step Unicode blocks
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

// SparklineChart creates a simple sparkline for inline display.
type SparklineChart struct {
	values []float64
	width  int
}

// NewSparklineChart creates a new sparkline chart.
func NewSparklineChart(width int) *SparklineChart {
	return &SparklineChart{
		values: make([]float64, 0, width),
		width:  width,
	}
}

// Add adds a value to the sparkline.
func (s *SparklineChart) Add(val float64) {
	s.values = append(s.values, val)
	if len(s.values) > s.width {
		s.values = s.values[1:]
	}
}

// Render returns the sparkline string.
func (s *SparklineChart) Render() string {
	if len(s.values) == 0 {
		return strings.Repeat("▁", s.width)
	}

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

	// Normalize and render
	var sb strings.Builder
	sparkChars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	for _, v := range s.values {
		var idx int
		if max > min {
			normalized := (v - min) / (max - min)
			idx = int(normalized * 7)
			if idx > 7 {
				idx = 7
			}
		}
		sb.WriteRune(sparkChars[idx])
	}

	// Pad if needed
	for i := len(s.values); i < s.width; i++ {
		sb.WriteRune('▁')
	}

	return sb.String()
}

// Clear resets the sparkline.
func (s *SparklineChart) Clear() {
	s.values = s.values[:0]
}
