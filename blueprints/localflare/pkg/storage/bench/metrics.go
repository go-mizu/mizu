package bench

import (
	"sort"
	"sync"
	"time"
)

// Metrics holds benchmark metrics for a single operation type.
type Metrics struct {
	Operation   string        `json:"operation"`
	Driver      string        `json:"driver"`
	ObjectSize  int           `json:"object_size,omitempty"`
	Iterations  int           `json:"iterations"`
	TotalTime   time.Duration `json:"total_time"`
	MinLatency  time.Duration `json:"min_latency"`
	MaxLatency  time.Duration `json:"max_latency"`
	AvgLatency  time.Duration `json:"avg_latency"`
	P50Latency  time.Duration `json:"p50_latency"`
	P95Latency  time.Duration `json:"p95_latency"`
	P99Latency  time.Duration `json:"p99_latency"`
	Throughput  float64       `json:"throughput"` // MB/s for data ops, ops/s for metadata ops
	TotalBytes  int64         `json:"total_bytes,omitempty"`
	Errors      int           `json:"errors"`
	LastError   string        `json:"last_error,omitempty"`
}

// Collector collects timing samples for a benchmark.
type Collector struct {
	mu      sync.Mutex
	samples []time.Duration
	errors  int
	lastErr string
}

// NewCollector creates a new metrics collector.
func NewCollector() *Collector {
	return &Collector{
		samples: make([]time.Duration, 0, 1000),
	}
}

// Record records a latency sample.
func (c *Collector) Record(d time.Duration) {
	c.mu.Lock()
	c.samples = append(c.samples, d)
	c.mu.Unlock()
}

// RecordError records an error.
func (c *Collector) RecordError(err error) {
	c.mu.Lock()
	c.errors++
	if err != nil {
		c.lastErr = err.Error()
	}
	c.mu.Unlock()
}

// RecordWithError records a latency sample and possible error.
func (c *Collector) RecordWithError(d time.Duration, err error) {
	c.mu.Lock()
	if err != nil {
		c.errors++
		c.lastErr = err.Error()
	} else {
		c.samples = append(c.samples, d)
	}
	c.mu.Unlock()
}

// Stats computes statistics from collected samples.
func (c *Collector) Stats() (min, max, avg, p50, p95, p99 time.Duration, total time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.samples) == 0 {
		return
	}

	// Copy and sort for percentile calculations
	sorted := make([]time.Duration, len(c.samples))
	copy(sorted, c.samples)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	min = sorted[0]
	max = sorted[len(sorted)-1]

	var sum time.Duration
	for _, s := range sorted {
		sum += s
	}
	avg = sum / time.Duration(len(sorted))
	total = sum

	p50 = percentile(sorted, 50)
	p95 = percentile(sorted, 95)
	p99 = percentile(sorted, 99)

	return
}

// Metrics returns a Metrics struct from collected data.
func (c *Collector) Metrics(operation, driver string, objectSize int) *Metrics {
	min, max, avg, p50, p95, p99, total := c.Stats()

	c.mu.Lock()
	errors := c.errors
	lastErr := c.lastErr
	iterations := len(c.samples)
	c.mu.Unlock()

	m := &Metrics{
		Operation:  operation,
		Driver:     driver,
		ObjectSize: objectSize,
		Iterations: iterations,
		TotalTime:  total,
		MinLatency: min,
		MaxLatency: max,
		AvgLatency: avg,
		P50Latency: p50,
		P95Latency: p95,
		P99Latency: p99,
		Errors:     errors,
		LastError:  lastErr,
	}

	// Calculate throughput
	if objectSize > 0 && total > 0 {
		// MB/s for data operations
		totalBytes := int64(iterations) * int64(objectSize)
		m.TotalBytes = totalBytes
		m.Throughput = float64(totalBytes) / (1024 * 1024) / total.Seconds()
	} else if total > 0 {
		// ops/s for metadata operations
		m.Throughput = float64(iterations) / total.Seconds()
	}

	return m
}

// Reset clears the collector for reuse.
func (c *Collector) Reset() {
	c.mu.Lock()
	c.samples = c.samples[:0]
	c.errors = 0
	c.lastErr = ""
	c.mu.Unlock()
}

// Count returns the number of samples collected.
func (c *Collector) Count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.samples)
}

func percentile(sorted []time.Duration, pct int) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	if pct >= 100 {
		return sorted[len(sorted)-1]
	}
	idx := (len(sorted) - 1) * pct / 100
	return sorted[idx]
}

// Timer is a simple timer for measuring operations.
type Timer struct {
	start time.Time
}

// NewTimer creates and starts a new timer.
func NewTimer() *Timer {
	return &Timer{start: time.Now()}
}

// Elapsed returns the elapsed time since the timer started.
func (t *Timer) Elapsed() time.Duration {
	return time.Since(t.start)
}

// Reset restarts the timer.
func (t *Timer) Reset() {
	t.start = time.Now()
}
