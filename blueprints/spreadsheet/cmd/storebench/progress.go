package main

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// ProgressDisplay manages the benchmark progress output.
type ProgressDisplay struct {
	mu            sync.Mutex
	writer        io.Writer
	startTime     time.Time
	totalBenches  int
	completed     int
	currentPhase  string
	currentBench  string
	currentDriver string
	spinner       int
	lastResult    *BenchResult
	spinnerChars  []string
	ticker        *time.Ticker
	done          chan struct{}
	running       bool
}

// NewProgressDisplay creates a new progress display.
func NewProgressDisplay(w io.Writer) *ProgressDisplay {
	return &ProgressDisplay{
		writer:       w,
		spinnerChars: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		done:         make(chan struct{}),
	}
}

// Start begins the progress display.
func (p *ProgressDisplay) Start(totalBenches int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.startTime = time.Now()
	p.totalBenches = totalBenches
	p.completed = 0
	p.running = true

	// Start spinner ticker
	p.ticker = time.NewTicker(100 * time.Millisecond)
	go p.spinnerLoop()
}

// Stop ends the progress display.
func (p *ProgressDisplay) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.ticker != nil {
		p.ticker.Stop()
	}
	close(p.done)
	p.running = false

	// Clear the progress line
	fmt.Fprintf(p.writer, "\r%s\r", strings.Repeat(" ", 100))
}

func (p *ProgressDisplay) spinnerLoop() {
	for {
		select {
		case <-p.done:
			return
		case <-p.ticker.C:
			p.render()
		}
	}
}

// SetPhase sets the current benchmark phase (category).
func (p *ProgressDisplay) SetPhase(phase string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.currentPhase = phase
}

// StartBenchmark marks the start of a benchmark.
func (p *ProgressDisplay) StartBenchmark(name, driver string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.currentBench = name
	p.currentDriver = driver
}

// CompleteBenchmark marks a benchmark as complete.
func (p *ProgressDisplay) CompleteBenchmark(result *BenchResult) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.completed++
	p.lastResult = result
	p.printResult(result)
}

func (p *ProgressDisplay) render() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return
	}

	p.spinner = (p.spinner + 1) % len(p.spinnerChars)
	spinChar := p.spinnerChars[p.spinner]

	elapsed := time.Since(p.startTime)
	elapsedStr := formatElapsed(elapsed)

	// Calculate progress
	progress := float64(p.completed) / float64(p.totalBenches) * 100
	if p.totalBenches == 0 {
		progress = 0
	}

	// Build progress bar
	barWidth := 20
	filledWidth := int(progress / 100 * float64(barWidth))
	bar := strings.Repeat("█", filledWidth) + strings.Repeat("░", barWidth-filledWidth)

	// Estimate remaining time
	var etaStr string
	if p.completed > 0 {
		avgTime := elapsed / time.Duration(p.completed)
		remaining := avgTime * time.Duration(p.totalBenches-p.completed)
		etaStr = fmt.Sprintf(" ETA: %s", formatElapsed(remaining))
	}

	// Truncate current bench name if too long
	benchName := p.currentBench
	if len(benchName) > 25 {
		benchName = benchName[:22] + "..."
	}

	// Build status line
	status := fmt.Sprintf("\r%s [%s] %3.0f%% (%d/%d) | %s | %-25s %-10s | %s%s",
		spinChar,
		bar,
		progress,
		p.completed,
		p.totalBenches,
		elapsedStr,
		benchName,
		p.currentDriver,
		p.currentPhase,
		etaStr,
	)

	// Pad to clear previous content
	padded := fmt.Sprintf("%-120s", status)
	fmt.Fprint(p.writer, padded)
}

func (p *ProgressDisplay) printResult(result *BenchResult) {
	// Clear the progress line first
	fmt.Fprintf(p.writer, "\r%s\r", strings.Repeat(" ", 120))

	// Print result with color coding
	var statusIcon string
	var resultStr string

	if result.Error != "" {
		statusIcon = "✗"
		resultStr = fmt.Sprintf("ERROR: %s", result.Error)
	} else {
		statusIcon = "✓"
		if result.Throughput > 0 {
			resultStr = fmt.Sprintf("%s (%.0f cells/sec)", formatDuration(result.Duration), result.Throughput)
		} else {
			resultStr = formatDuration(result.Duration)
		}
	}

	fmt.Fprintf(p.writer, "  %s %-12s %-30s %s\n",
		statusIcon,
		result.Driver,
		result.Name,
		resultStr,
	)
}

// PrintHeader prints the benchmark header.
func (p *ProgressDisplay) PrintHeader(config *BenchConfig) {
	fmt.Fprintln(p.writer)
	fmt.Fprintln(p.writer, "┌────────────────────────────────────────────────────────────────────────┐")
	fmt.Fprintln(p.writer, "│              Spreadsheet Storage Benchmark Suite                       │")
	fmt.Fprintln(p.writer, "├────────────────────────────────────────────────────────────────────────┤")
	fmt.Fprintf(p.writer, "│  Drivers:    %-57s │\n", strings.Join(config.Drivers, ", "))
	fmt.Fprintf(p.writer, "│  Categories: %-57s │\n", strings.Join(config.Categories, ", "))
	fmt.Fprintf(p.writer, "│  Iterations: %-57d │\n", config.Iterations)
	fmt.Fprintf(p.writer, "│  Warmup:     %-57d │\n", config.Warmup)
	if config.Quick {
		fmt.Fprintln(p.writer, "│  Mode:       Quick                                                     │")
	} else if config.RunLoad {
		fmt.Fprintln(p.writer, "│  Mode:       Full (with load tests)                                    │")
	} else {
		fmt.Fprintln(p.writer, "│  Mode:       Standard                                                  │")
	}
	fmt.Fprintln(p.writer, "└────────────────────────────────────────────────────────────────────────┘")
	fmt.Fprintln(p.writer)
}

// PrintPhaseHeader prints a category header.
func (p *ProgressDisplay) PrintPhaseHeader(phase string) {
	fmt.Fprintf(p.writer, "\n━━━ %s ━━━\n\n", strings.ToUpper(phase))
}

// PrintDriverInit prints driver initialization status.
func (p *ProgressDisplay) PrintDriverInit(driver string, success bool, err error) {
	if success {
		fmt.Fprintf(p.writer, "  ✓ %s initialized\n", driver)
	} else {
		fmt.Fprintf(p.writer, "  ✗ %s skipped: %v\n", driver, err)
	}
}

// PrintSummary prints the final benchmark summary.
func (p *ProgressDisplay) PrintSummary(results *BenchResults) {
	fmt.Fprintln(p.writer)
	fmt.Fprintln(p.writer, "┌────────────────────────────────────────────────────────────────────────┐")
	fmt.Fprintln(p.writer, "│                          Benchmark Summary                             │")
	fmt.Fprintln(p.writer, "├────────────────────────────────────────────────────────────────────────┤")
	fmt.Fprintf(p.writer, "│  Total benchmarks: %-51d │\n", len(results.Results))
	fmt.Fprintf(p.writer, "│  Total duration:   %-51s │\n", results.TotalDuration.Round(time.Millisecond))

	// Count errors
	errors := 0
	for _, r := range results.Results {
		if r.Error != "" {
			errors++
		}
	}
	if errors > 0 {
		fmt.Fprintf(p.writer, "│  Errors:           %-51d │\n", errors)
	}

	fmt.Fprintln(p.writer, "├────────────────────────────────────────────────────────────────────────┤")

	// Calculate wins by driver
	categoryWins := make(map[string]map[string]int)
	for _, r := range results.Results {
		if categoryWins[r.Category] == nil {
			categoryWins[r.Category] = make(map[string]int)
		}
	}

	byName := make(map[string][]BenchResult)
	for _, r := range results.Results {
		key := r.Category + "/" + r.Name
		byName[key] = append(byName[key], r)
	}

	for _, rs := range byName {
		if len(rs) < 2 {
			continue
		}
		fastest := rs[0]
		for _, r := range rs[1:] {
			if r.Error == "" && r.NsPerOp < fastest.NsPerOp {
				fastest = r
			}
		}
		if fastest.Error == "" {
			categoryWins[fastest.Category][fastest.Driver]++
		}
	}

	// Total wins per driver
	totalWins := make(map[string]int)
	for _, catWins := range categoryWins {
		for driver, wins := range catWins {
			totalWins[driver] += wins
		}
	}

	fmt.Fprintln(p.writer, "│  Performance Wins by Driver:                                           │")
	for driver, wins := range totalWins {
		bar := strings.Repeat("█", wins)
		fmt.Fprintf(p.writer, "│    %-10s %3d %s\n", driver, wins, bar)
	}

	fmt.Fprintln(p.writer, "└────────────────────────────────────────────────────────────────────────┘")
}

func formatElapsed(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm%02ds", minutes, seconds)
}

// EstimateBenchmarkCount estimates the total number of benchmarks that will run.
func EstimateBenchmarkCount(config *BenchConfig, driverCount int) int {
	count := 0

	categories := config.Categories
	if contains(categories, "all") {
		categories = []string{"cells", "rows", "merge", "format", "query"}
	}

	for _, cat := range categories {
		switch cat {
		case "cells":
			if config.Quick {
				// BatchSet: 2 sizes, GetByPositions_Sparse: 2 sizes, GetByPositions_Dense: 2 sizes, GetRange: 2 sizes
				count += (2 + 2 + 2 + 2) * driverCount
			} else {
				// BatchSet: 5 sizes, GetByPositions_Sparse: 3 sizes, GetByPositions_Dense: 3 sizes, GetRange: 3 sizes
				count += (5 + 3 + 3 + 3) * driverCount
			}
		case "rows":
			if config.Quick {
				count += (2 + 2) * driverCount // ShiftRows + ShiftCols
			} else {
				count += (4 + 3) * driverCount
			}
		case "merge":
			if config.Quick {
				count += (2 + 2) * driverCount // Individual + Batch
			} else {
				count += (3 + 3) * driverCount
			}
		case "format":
			count += 3 * driverCount // WithFormat, NoFormat, PartialFormat
		case "query":
			if config.Quick {
				count += 2 * driverCount
			} else {
				count += 3 * driverCount
			}
		}
	}

	// Use cases
	usecases := config.Usecases
	if contains(config.Categories, "all") && contains(usecases, "all") {
		if config.Quick {
			// financial + import(2 sizes)
			count += (1 + 2) * driverCount
		} else {
			// financial + import(3 sizes) + report + sparse + bulk
			count += (1 + 3 + 1 + 1 + 1) * driverCount
		}
	}

	// Load tests
	if config.RunLoad {
		if config.Quick {
			count += (2 + 1) * driverCount // 2 sustained + 1 mixed
		} else {
			count += (3 + 1) * driverCount
		}
	}

	return count
}
