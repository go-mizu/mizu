package benchmark

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"time"
)

// SystemInfo contains information about the benchmark system.
type SystemInfo struct {
	OS            string `json:"os"`
	Arch          string `json:"arch"`
	CPUs          int    `json:"cpus"`
	Memory        string `json:"memory"`
	GoVersion     string `json:"go_version"`
	DocumentCount int64  `json:"document_count"`
}

// CollectSystemInfo gathers system information.
func CollectSystemInfo() SystemInfo {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return SystemInfo{
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		CPUs:      runtime.NumCPU(),
		Memory:    formatBytes(memStats.Sys),
		GoVersion: runtime.Version(),
	}
}

// IndexingMetrics contains metrics from the indexing phase.
type IndexingMetrics struct {
	Duration   time.Duration `json:"duration"`
	DocsPerSec float64       `json:"docs_per_sec"`
	PeakMemory int64         `json:"peak_memory"`
	TotalDocs  int64         `json:"total_docs"`
}

// LatencyMetrics contains search latency percentiles.
type LatencyMetrics struct {
	P50 time.Duration `json:"p50"`
	P95 time.Duration `json:"p95"`
	P99 time.Duration `json:"p99"`
	Max time.Duration `json:"max"`
	Min time.Duration `json:"min"`
	Avg time.Duration `json:"avg"`
}

// ThroughputMetrics contains throughput measurements.
type ThroughputMetrics struct {
	QPS        float64       `json:"qps"`
	Duration   time.Duration `json:"duration"`
	TotalOps   int64         `json:"total_ops"`
	Goroutines int           `json:"goroutines"`
}

// MemoryMetrics contains memory usage measurements.
type MemoryMetrics struct {
	IndexingPeak int64 `json:"indexing_peak"`
	SearchIdle   int64 `json:"search_idle"`
	SearchPeak   int64 `json:"search_peak"`
}

// QualityMetrics contains result quality measurements.
type QualityMetrics struct {
	BaselineDriver string  `json:"baseline_driver"`
	Overlap        int     `json:"overlap"`
	OverlapPct     float64 `json:"overlap_pct"`
}

// QueryMetrics contains per-query metrics.
type QueryMetrics struct {
	Query    Query         `json:"query"`
	Duration time.Duration `json:"duration"`
	Results  int           `json:"results"`
}

// LatencyCollector collects latency samples and computes percentiles.
type LatencyCollector struct {
	samples []time.Duration
}

// NewLatencyCollector creates a new latency collector.
func NewLatencyCollector() *LatencyCollector {
	return &LatencyCollector{
		samples: make([]time.Duration, 0, 1000),
	}
}

// Add adds a latency sample.
func (c *LatencyCollector) Add(d time.Duration) {
	c.samples = append(c.samples, d)
}

// Metrics computes latency metrics from collected samples.
func (c *LatencyCollector) Metrics() *LatencyMetrics {
	if len(c.samples) == 0 {
		return &LatencyMetrics{}
	}

	// Sort samples
	sorted := make([]time.Duration, len(c.samples))
	copy(sorted, c.samples)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	// Calculate metrics
	var sum time.Duration
	for _, s := range sorted {
		sum += s
	}

	return &LatencyMetrics{
		P50: percentile(sorted, 50),
		P95: percentile(sorted, 95),
		P99: percentile(sorted, 99),
		Min: sorted[0],
		Max: sorted[len(sorted)-1],
		Avg: sum / time.Duration(len(sorted)),
	}
}

// percentile returns the p-th percentile from sorted samples.
func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(len(sorted)-1) * p / 100)
	return sorted[idx]
}

// MemoryTracker tracks memory usage.
type MemoryTracker struct {
	startAlloc uint64
	peakAlloc  uint64
}

// NewMemoryTracker creates a new memory tracker.
func NewMemoryTracker() *MemoryTracker {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return &MemoryTracker{
		startAlloc: m.Alloc,
		peakAlloc:  m.Alloc,
	}
}

// Sample takes a memory sample and updates peak if necessary.
func (t *MemoryTracker) Sample() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	if m.Alloc > t.peakAlloc {
		t.peakAlloc = m.Alloc
	}
}

// Peak returns the peak memory allocation.
func (t *MemoryTracker) Peak() int64 {
	return int64(t.peakAlloc)
}

// Delta returns the change from start to peak.
func (t *MemoryTracker) Delta() int64 {
	return int64(t.peakAlloc - t.startAlloc)
}

// MeasureIndexSize returns the size of an index directory in bytes.
func MeasureIndexSize(dir string) (int64, error) {
	var size int64
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

// formatBytes formats bytes as human-readable string.
func formatBytes(bytes uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return formatFloat(float64(bytes)/GB) + " GB"
	case bytes >= MB:
		return formatFloat(float64(bytes)/MB) + " MB"
	case bytes >= KB:
		return formatFloat(float64(bytes)/KB) + " KB"
	default:
		return formatInt(int64(bytes)) + " B"
	}
}

// FormatBytes is exported version of formatBytes.
func FormatBytes(bytes int64) string {
	return formatBytes(uint64(bytes))
}

func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', 2, 64)
}

func formatInt(i int64) string {
	return strconv.FormatInt(i, 10)
}
