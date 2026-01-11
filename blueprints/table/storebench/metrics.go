package storebench

import (
	"math"
	"sort"
	"sync"
	"time"
)

// Metrics holds collected benchmark metrics.
type Metrics struct {
	mu         sync.Mutex
	latencies  []time.Duration
	errors     int
	startTime  time.Time
	endTime    time.Time
	recordsOps int64 // For batch ops: total records processed
}

// NewMetrics creates a new metrics collector.
func NewMetrics() *Metrics {
	return &Metrics{
		latencies: make([]time.Duration, 0, 1000),
	}
}

// Start marks the start of measurement.
func (m *Metrics) Start() {
	m.mu.Lock()
	m.startTime = time.Now()
	m.mu.Unlock()
}

// End marks the end of measurement.
func (m *Metrics) End() {
	m.mu.Lock()
	m.endTime = time.Now()
	m.mu.Unlock()
}

// Record records a single operation latency.
func (m *Metrics) Record(d time.Duration) {
	m.mu.Lock()
	m.latencies = append(m.latencies, d)
	m.mu.Unlock()
}

// RecordWithCount records a latency with a count (for batch operations).
func (m *Metrics) RecordWithCount(d time.Duration, count int64) {
	m.mu.Lock()
	m.latencies = append(m.latencies, d)
	m.recordsOps += count
	m.mu.Unlock()
}

// RecordError records an error.
func (m *Metrics) RecordError() {
	m.mu.Lock()
	m.errors++
	m.mu.Unlock()
}

// Stats returns computed statistics.
type Stats struct {
	Count        int           `json:"count"`
	Errors       int           `json:"errors"`
	ErrorRate    float64       `json:"error_rate"`
	Min          time.Duration `json:"min"`
	Max          time.Duration `json:"max"`
	Avg          time.Duration `json:"avg"`
	P50          time.Duration `json:"p50"`
	P90          time.Duration `json:"p90"`
	P95          time.Duration `json:"p95"`
	P99          time.Duration `json:"p99"`
	StdDev       time.Duration `json:"std_dev"`
	TotalTime    time.Duration `json:"total_time"`
	OpsPerSec    float64       `json:"ops_per_sec"`
	RecordsOps   int64         `json:"records_ops,omitempty"`
	RecordsPerSec float64      `json:"records_per_sec,omitempty"`
}

// Stats computes statistics from the collected metrics.
func (m *Metrics) Stats() Stats {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.latencies) == 0 {
		return Stats{}
	}

	// Sort for percentile calculations
	sorted := make([]time.Duration, len(m.latencies))
	copy(sorted, m.latencies)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	var sum time.Duration
	for _, d := range sorted {
		sum += d
	}
	avg := sum / time.Duration(len(sorted))

	// Standard deviation
	var sqDiffSum float64
	for _, d := range sorted {
		diff := float64(d - avg)
		sqDiffSum += diff * diff
	}
	stdDev := time.Duration(math.Sqrt(sqDiffSum / float64(len(sorted))))

	totalTime := m.endTime.Sub(m.startTime)
	if totalTime == 0 {
		totalTime = sum
	}

	stats := Stats{
		Count:     len(sorted),
		Errors:    m.errors,
		ErrorRate: float64(m.errors) / float64(len(sorted)+m.errors) * 100,
		Min:       sorted[0],
		Max:       sorted[len(sorted)-1],
		Avg:       avg,
		P50:       percentile(sorted, 50),
		P90:       percentile(sorted, 90),
		P95:       percentile(sorted, 95),
		P99:       percentile(sorted, 99),
		StdDev:    stdDev,
		TotalTime: totalTime,
		OpsPerSec: float64(len(sorted)) / totalTime.Seconds(),
	}

	if m.recordsOps > 0 {
		stats.RecordsOps = m.recordsOps
		stats.RecordsPerSec = float64(m.recordsOps) / totalTime.Seconds()
	}

	return stats
}

func percentile(sorted []time.Duration, p int) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(len(sorted)-1) * float64(p) / 100.0)
	return sorted[idx]
}

// Result holds the result of a single benchmark scenario.
type Result struct {
	Scenario  string `json:"scenario"`
	Backend   string `json:"backend"`
	Stats     Stats  `json:"stats"`
	Timestamp time.Time `json:"timestamp"`
}

// BenchmarkResults holds all results for a benchmark run.
type BenchmarkResults struct {
	Environment Environment `json:"environment"`
	Config      *Config     `json:"config"`
	Results     []Result    `json:"results"`
	StartTime   time.Time   `json:"start_time"`
	EndTime     time.Time   `json:"end_time"`
}

// NewBenchmarkResults creates a new results container.
func NewBenchmarkResults(cfg *Config) *BenchmarkResults {
	return &BenchmarkResults{
		Environment: GetEnvironment(cfg),
		Config:      cfg,
		Results:     make([]Result, 0),
		StartTime:   time.Now(),
	}
}

// Add adds a result.
func (br *BenchmarkResults) Add(r Result) {
	br.Results = append(br.Results, r)
}

// Finish marks the benchmark as complete.
func (br *BenchmarkResults) Finish() {
	br.EndTime = time.Now()
}

// GetResultsByScenario returns results grouped by scenario.
func (br *BenchmarkResults) GetResultsByScenario() map[string][]Result {
	grouped := make(map[string][]Result)
	for _, r := range br.Results {
		grouped[r.Scenario] = append(grouped[r.Scenario], r)
	}
	return grouped
}

// GetResultsByBackend returns results grouped by backend.
func (br *BenchmarkResults) GetResultsByBackend() map[string][]Result {
	grouped := make(map[string][]Result)
	for _, r := range br.Results {
		grouped[r.Backend] = append(grouped[r.Backend], r)
	}
	return grouped
}
