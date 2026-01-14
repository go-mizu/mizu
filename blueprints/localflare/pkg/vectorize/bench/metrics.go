package bench

import (
	"sort"
	"time"
)

// Result holds benchmark results for a single operation.
type Result struct {
	Driver     string        `json:"driver"`
	Operation  string        `json:"operation"`
	Config     string        `json:"config"`
	Iterations int           `json:"iterations"`
	TotalTime  time.Duration `json:"total_time_ns"`
	MinLatency time.Duration `json:"min_latency_ns"`
	MaxLatency time.Duration `json:"max_latency_ns"`
	AvgLatency time.Duration `json:"avg_latency_ns"`
	P50Latency time.Duration `json:"p50_latency_ns"`
	P95Latency time.Duration `json:"p95_latency_ns"`
	P99Latency time.Duration `json:"p99_latency_ns"`
	Throughput float64       `json:"throughput"` // ops/sec or vectors/sec
	Errors     int           `json:"errors"`
	ErrorMsg   string        `json:"error_msg,omitempty"`
}

// Collector collects timing samples and computes statistics.
type Collector struct {
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

// Record adds a sample.
func (c *Collector) Record(d time.Duration) {
	c.samples = append(c.samples, d)
}

// RecordError increments error count.
func (c *Collector) RecordError(err error) {
	c.errors++
	if err != nil {
		c.lastErr = err.Error()
	}
}

// Stats computes statistics from collected samples.
func (c *Collector) Stats() (min, max, avg, p50, p95, p99 time.Duration, total time.Duration) {
	if len(c.samples) == 0 {
		return
	}

	// Sort for percentiles
	sorted := make([]time.Duration, len(c.samples))
	copy(sorted, c.samples)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	min = sorted[0]
	max = sorted[len(sorted)-1]

	var sum time.Duration
	for _, s := range sorted {
		sum += s
	}
	total = sum
	avg = sum / time.Duration(len(sorted))

	p50 = percentile(sorted, 50)
	p95 = percentile(sorted, 95)
	p99 = percentile(sorted, 99)

	return
}

// Result creates a Result from the collected samples.
func (c *Collector) Result(driver, operation, config string, itemsProcessed int) Result {
	min, max, avg, p50, p95, p99, total := c.Stats()

	var throughput float64
	if total > 0 && itemsProcessed > 0 {
		throughput = float64(itemsProcessed) / total.Seconds()
	}

	return Result{
		Driver:     driver,
		Operation:  operation,
		Config:     config,
		Iterations: len(c.samples),
		TotalTime:  total,
		MinLatency: min,
		MaxLatency: max,
		AvgLatency: avg,
		P50Latency: p50,
		P95Latency: p95,
		P99Latency: p99,
		Throughput: throughput,
		Errors:     c.errors,
		ErrorMsg:   c.lastErr,
	}
}

// Errors returns the error count.
func (c *Collector) Errors() int {
	return c.errors
}

// Count returns the sample count.
func (c *Collector) Count() int {
	return len(c.samples)
}

func percentile(sorted []time.Duration, p int) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	idx := (len(sorted) - 1) * p / 100
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

// Timer is a helper for timing operations.
type Timer struct {
	start time.Time
}

// Start begins timing.
func (t *Timer) Start() {
	t.start = time.Now()
}

// Stop returns the elapsed time.
func (t *Timer) Stop() time.Duration {
	return time.Since(t.start)
}

// NewTimer creates and starts a timer.
func NewTimer() *Timer {
	return &Timer{start: time.Now()}
}
