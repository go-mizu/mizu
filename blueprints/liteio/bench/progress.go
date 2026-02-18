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
	fmt.Printf("\r  %s [%s] %d/%d (%.1f%%) %.1f/s ETA: %s    ",
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
func (p *Progress) DoneWithStats(totalBytes int64) {
	if !p.enabled {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	elapsed := time.Since(p.startTime)
	opsPerSec := float64(p.total) / elapsed.Seconds()

	var mbPerSec float64
	if totalBytes > 0 {
		mbPerSec = float64(totalBytes) / (1024 * 1024) / elapsed.Seconds()
	}

	if mbPerSec > 0 {
		fmt.Printf("\r  %s: completed %d ops in %v (%.1f ops/s, %.2f MB/s)    \n",
			p.operation, p.total, elapsed.Round(time.Millisecond), opsPerSec, mbPerSec)
	} else {
		fmt.Printf("\r  %s: completed %d ops in %v (%.1f ops/s)    \n",
			p.operation, p.total, elapsed.Round(time.Millisecond), opsPerSec)
	}
}

// Reset resets the progress for reuse.
func (p *Progress) Reset(operation string, total int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.operation = operation
	p.current = 0
	p.total = total
	p.startTime = time.Now()
	p.lastPrint = time.Now()
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
