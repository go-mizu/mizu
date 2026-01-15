package main

import (
	"fmt"
	"io"
	"sort"
	"sync"
	"time"
)

// Sample represents a single benchmark measurement.
type Sample struct {
	TTFB      time.Duration
	TTLB      time.Duration
	Size      int64
	Timestamp time.Time
	Error     error
}

// LatencyStats holds latency statistics.
type LatencyStats struct {
	Avg time.Duration
	Min time.Duration
	Max time.Duration
	P25 time.Duration
	P50 time.Duration
	P75 time.Duration
	P90 time.Duration
	P99 time.Duration
}

// BenchmarkResult holds the result of a benchmark run.
type BenchmarkResult struct {
	Driver     string
	ObjectSize int
	Threads    int
	Throughput float64 // MB/s
	TTFB       LatencyStats
	TTLB       LatencyStats
	TotalBytes int64
	Duration   time.Duration
	Samples    int
	Errors     int
}

// Collector collects benchmark samples.
type Collector struct {
	samples []Sample
	mu      sync.Mutex
}

// NewCollector creates a new sample collector.
func NewCollector() *Collector {
	return &Collector{
		samples: make([]Sample, 0, 1000),
	}
}

// AddSample adds a sample to the collector.
func (c *Collector) AddSample(ttfb, ttlb time.Duration, size int64, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.samples = append(c.samples, Sample{
		TTFB:      ttfb,
		TTLB:      ttlb,
		Size:      size,
		Timestamp: time.Now(),
		Error:     err,
	})
}

// Calculate computes statistics from the collected samples.
func (c *Collector) Calculate() (ttfb, ttlb LatencyStats, throughput float64, errors int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.samples) == 0 {
		return
	}

	var ttfbValues, ttlbValues []time.Duration
	var totalBytes int64
	var totalDuration time.Duration

	for _, s := range c.samples {
		if s.Error != nil {
			errors++
			continue
		}
		ttfbValues = append(ttfbValues, s.TTFB)
		ttlbValues = append(ttlbValues, s.TTLB)
		totalBytes += s.Size
		totalDuration += s.TTLB
	}

	if len(ttfbValues) > 0 {
		ttfb = calculateStats(ttfbValues)
		ttlb = calculateStats(ttlbValues)
	}

	// Calculate throughput in MB/s
	if totalDuration > 0 {
		throughput = float64(totalBytes) / totalDuration.Seconds() / 1024 / 1024
	}

	return
}

// Count returns the number of samples.
func (c *Collector) Count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.samples)
}

// Reset clears all samples.
func (c *Collector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.samples = c.samples[:0]
}

func calculateStats(values []time.Duration) LatencyStats {
	if len(values) == 0 {
		return LatencyStats{}
	}

	// Sort for percentile calculation
	sorted := make([]time.Duration, len(values))
	copy(sorted, values)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	// Calculate average
	var sum time.Duration
	for _, v := range sorted {
		sum += v
	}

	return LatencyStats{
		Avg: sum / time.Duration(len(sorted)),
		Min: sorted[0],
		Max: sorted[len(sorted)-1],
		P25: percentile(sorted, 0.25),
		P50: percentile(sorted, 0.50),
		P75: percentile(sorted, 0.75),
		P90: percentile(sorted, 0.90),
		P99: percentile(sorted, 0.99),
	}
}

func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(len(sorted)-1) * p)
	return sorted[idx]
}

// TTFBReader wraps a reader to measure Time to First Byte.
type TTFBReader struct {
	reader    io.Reader
	startTime time.Time
	firstRead bool
	ttfb      time.Duration
}

// NewTTFBReader creates a new TTFB measuring reader.
func NewTTFBReader(r io.Reader, startTime time.Time) *TTFBReader {
	return &TTFBReader{
		reader:    r,
		startTime: startTime,
	}
}

// Read implements io.Reader and captures TTFB on first read.
func (r *TTFBReader) Read(p []byte) (n int, err error) {
	n, err = r.reader.Read(p)
	if !r.firstRead && n > 0 {
		r.ttfb = time.Since(r.startTime)
		r.firstRead = true
	}
	return
}

// TTFB returns the time to first byte.
func (r *TTFBReader) TTFB() time.Duration {
	return r.ttfb
}

// FormatSize returns a human-readable size string.
func FormatSize(bytes int) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%d GB", bytes/GB)
	case bytes >= MB:
		return fmt.Sprintf("%d MB", bytes/MB)
	case bytes >= KB:
		return fmt.Sprintf("%d KB", bytes/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// FormatDuration returns a formatted duration string.
func FormatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}
