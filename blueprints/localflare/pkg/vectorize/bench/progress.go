package bench

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// Progress represents progress for a running operation.
type Progress struct {
	mu        sync.Mutex
	operation string
	current   int
	total     int
	startTime time.Time
	lastPrint time.Time
	width     int
	enabled   bool
}

// NewProgress creates a new progress reporter.
func NewProgress(operation string, total int, enabled bool) *Progress {
	return &Progress{
		operation: operation,
		total:     total,
		startTime: time.Now(),
		lastPrint: time.Now(),
		width:     40,
		enabled:   enabled,
	}
}

// Update updates the progress.
func (p *Progress) Update(current int) {
	if !p.enabled {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.current = current

	// Rate limit printing to avoid too much output
	if time.Since(p.lastPrint) < 100*time.Millisecond && current < p.total {
		return
	}
	p.lastPrint = time.Now()

	p.print()
}

// Increment adds one to the current progress.
func (p *Progress) Increment() {
	if !p.enabled {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.current++

	// Rate limit printing
	if time.Since(p.lastPrint) < 100*time.Millisecond && p.current < p.total {
		return
	}
	p.lastPrint = time.Now()

	p.print()
}

func (p *Progress) print() {
	elapsed := time.Since(p.startTime)
	percent := float64(p.current) / float64(p.total)
	if percent > 1 {
		percent = 1
	}

	// Calculate ETA
	var eta time.Duration
	if p.current > 0 && percent < 1 {
		eta = time.Duration(float64(elapsed) / percent * (1 - percent))
	}

	// Calculate throughput
	var throughput float64
	if elapsed.Seconds() > 0 {
		throughput = float64(p.current) / elapsed.Seconds()
	}

	// Build progress bar
	filled := int(float64(p.width) * percent)
	bar := strings.Repeat("█", filled) + strings.Repeat("░", p.width-filled)

	// Format output
	fmt.Printf("\r  %s [%s] %d/%d (%.1f%%) %.1f/s ETA: %s",
		p.operation, bar, p.current, p.total,
		percent*100, throughput, formatDuration(eta))
}

// Done marks the progress as complete.
func (p *Progress) Done() {
	if !p.enabled {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.current = p.total
	p.print()
	fmt.Println() // Newline after completion
}

// DoneWithStats prints final statistics.
func (p *Progress) DoneWithStats(stats *SearchStats) {
	if !p.enabled {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	elapsed := time.Since(p.startTime)
	qps := float64(p.total) / elapsed.Seconds()

	fmt.Printf("\r  %s: completed %d queries in %v (%.1f QPS)\n",
		p.operation, p.total, elapsed.Round(time.Millisecond), qps)

	if stats != nil {
		fmt.Printf("    Stats: %d vectors scanned, %d clusters probed, %d graph hops, %d distance calcs\n",
			stats.VectorsScanned, stats.ClustersProbed, stats.GraphHops, stats.DistanceCalcs)
	}
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		return "--:--"
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
}

// SearchStats tracks statistics for search operations.
type SearchStats struct {
	VectorsScanned int64
	ClustersProbed int64
	GraphHops      int64
	DistanceCalcs  int64
}

// Add combines statistics.
func (s *SearchStats) Add(other *SearchStats) {
	if other == nil {
		return
	}
	s.VectorsScanned += other.VectorsScanned
	s.ClustersProbed += other.ClustersProbed
	s.GraphHops += other.GraphHops
	s.DistanceCalcs += other.DistanceCalcs
}

// ProgressWriter wraps an io.Writer to report progress for large operations.
type ProgressWriter struct {
	progress *Progress
}

// NewProgressWriter creates a progress-reporting writer.
func NewProgressWriter(operation string, total int) *ProgressWriter {
	return &ProgressWriter{
		progress: NewProgress(operation, total, true),
	}
}

// Increment reports one unit of progress.
func (pw *ProgressWriter) Increment() {
	pw.progress.Increment()
}

// Done completes the progress.
func (pw *ProgressWriter) Done() {
	pw.progress.Done()
}
