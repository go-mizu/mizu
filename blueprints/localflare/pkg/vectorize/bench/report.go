package bench

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Report holds all benchmark results.
type Report struct {
	Timestamp   time.Time `json:"timestamp"`
	Config      *Config   `json:"config"`
	Results     []Result  `json:"results"`
	DriverStats map[string]*DriverStats `json:"driver_stats"`
}

// DriverStats holds aggregated stats for a driver.
type DriverStats struct {
	Driver           string  `json:"driver"`
	Available        bool    `json:"available"`
	ConnectionTime   int64   `json:"connection_time_ms"`
	InsertThroughput float64 `json:"insert_throughput"`
	SearchP50        int64   `json:"search_p50_us"`
	SearchP99        int64   `json:"search_p99_us"`
	SearchQPS        float64 `json:"search_qps"`
	TotalErrors      int     `json:"total_errors"`
	// Docker container stats (for server-based drivers)
	MemoryUsageMB    float64 `json:"memory_usage_mb"`
	MemoryLimitMB    float64 `json:"memory_limit_mb"`
	MemoryPercent    float64 `json:"memory_percent"`
	CPUPercent       float64 `json:"cpu_percent"`
	DiskUsageMB      float64 `json:"disk_usage_mb"`
	IsEmbedded       bool    `json:"is_embedded"`
	// Embedded driver memory stats (from runtime.MemStats)
	HeapAllocMB      float64 `json:"heap_alloc_mb,omitempty"`
	HeapInUseMB      float64 `json:"heap_inuse_mb,omitempty"`
	HeapObjectsK     float64 `json:"heap_objects_k,omitempty"`
	MemPerVectorB    float64 `json:"mem_per_vector_bytes,omitempty"`
}

// NewReport creates a new report.
func NewReport(cfg *Config) *Report {
	return &Report{
		Timestamp:   time.Now(),
		Config:      cfg,
		Results:     make([]Result, 0),
		DriverStats: make(map[string]*DriverStats),
	}
}

// AddResult adds a benchmark result.
func (r *Report) AddResult(result Result) {
	r.Results = append(r.Results, result)
}

// ComputeStats computes aggregate statistics per driver.
func (r *Report) ComputeStats() {
	for _, result := range r.Results {
		stats, ok := r.DriverStats[result.Driver]
		if !ok {
			stats = &DriverStats{
				Driver:    result.Driver,
				Available: true,
			}
			r.DriverStats[result.Driver] = stats
		}

		stats.TotalErrors += result.Errors

		switch result.Operation {
		case "insert":
			if result.Throughput > stats.InsertThroughput {
				stats.InsertThroughput = result.Throughput
			}
		case "search":
			stats.SearchP50 = result.P50Latency.Microseconds()
			stats.SearchP99 = result.P99Latency.Microseconds()
			if result.TotalTime > 0 {
				stats.SearchQPS = float64(result.Iterations) / result.TotalTime.Seconds()
			}
		case "connect":
			stats.ConnectionTime = result.AvgLatency.Milliseconds()
		}
	}
}

// SaveJSON writes raw results to JSON.
func (r *Report) SaveJSON(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path := filepath.Join(dir, "raw_results.json")
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// SaveMarkdown generates and saves the markdown report.
func (r *Report) SaveMarkdown(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path := filepath.Join(dir, "benchmark_report.md")
	content := r.GenerateMarkdown()
	return os.WriteFile(path, []byte(content), 0644)
}

// GenerateMarkdown creates the markdown report content.
func (r *Report) GenerateMarkdown() string {
	var sb strings.Builder

	sb.WriteString("# Vectorize Driver Benchmark Report\n\n")
	sb.WriteString(fmt.Sprintf("**Generated:** %s\n\n", r.Timestamp.Format(time.RFC3339)))

	// Configuration
	sb.WriteString("## Configuration\n\n")
	sb.WriteString("| Parameter | Value |\n")
	sb.WriteString("|-----------|-------|\n")
	sb.WriteString(fmt.Sprintf("| Dimensions | %d |\n", r.Config.Dimensions))
	sb.WriteString(fmt.Sprintf("| Dataset Size | %d |\n", r.Config.DatasetSize))
	sb.WriteString(fmt.Sprintf("| Batch Size | %d |\n", r.Config.BatchSize))
	sb.WriteString(fmt.Sprintf("| Search Iterations | %d |\n", r.Config.SearchIterations))
	sb.WriteString(fmt.Sprintf("| TopK | %d |\n", r.Config.TopK))
	sb.WriteString("\n")

	// Summary table
	sb.WriteString("## Summary\n\n")
	sb.WriteString("| Driver | Status | Connect (ms) | Insert (vec/s) | Search p50 (μs) | Search p99 (μs) | QPS | Errors |\n")
	sb.WriteString("|--------|--------|--------------|----------------|-----------------|-----------------|-----|--------|\n")

	// Sort drivers by name
	var drivers []string
	for d := range r.DriverStats {
		drivers = append(drivers, d)
	}
	sort.Strings(drivers)

	for _, d := range drivers {
		stats := r.DriverStats[d]
		status := "✅"
		if !stats.Available || stats.TotalErrors > 0 {
			status = "❌"
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %d | %.0f | %d | %d | %.1f | %d |\n",
			stats.Driver,
			status,
			stats.ConnectionTime,
			stats.InsertThroughput,
			stats.SearchP50,
			stats.SearchP99,
			stats.SearchQPS,
			stats.TotalErrors,
		))
	}
	sb.WriteString("\n")

	// Resource usage table - Server drivers
	sb.WriteString("## Resource Usage (Docker Containers)\n\n")
	sb.WriteString("| Driver | Type | Memory (MB) | Memory Limit (MB) | Memory % | CPU % | Disk (MB) |\n")
	sb.WriteString("|--------|------|-------------|-------------------|----------|-------|----------|\n")

	for _, d := range drivers {
		stats := r.DriverStats[d]
		driverType := "Server"
		if stats.IsEmbedded {
			driverType = "Embedded"
		}

		memStr := fmt.Sprintf("%.1f", stats.MemoryUsageMB)
		limitStr := fmt.Sprintf("%.1f", stats.MemoryLimitMB)
		memPctStr := fmt.Sprintf("%.1f%%", stats.MemoryPercent)
		cpuStr := fmt.Sprintf("%.2f%%", stats.CPUPercent)
		diskStr := fmt.Sprintf("%.1f", stats.DiskUsageMB)

		if stats.IsEmbedded {
			memStr = "-"
			limitStr = "-"
			memPctStr = "-"
			cpuStr = "-"
			diskStr = "-"
		} else if stats.MemoryUsageMB == 0 {
			memStr = "-"
			limitStr = "-"
			memPctStr = "-"
			cpuStr = "-"
			diskStr = "-"
		}

		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %s | %s |\n",
			stats.Driver,
			driverType,
			memStr,
			limitStr,
			memPctStr,
			cpuStr,
			diskStr,
		))
	}
	sb.WriteString("\n")

	// Memory usage table - Embedded drivers
	sb.WriteString("## Memory Usage (Embedded Drivers)\n\n")
	sb.WriteString("| Driver | Heap Alloc (MB) | Heap InUse (MB) | Heap Objects (K) | Bytes/Vector |\n")
	sb.WriteString("|--------|-----------------|-----------------|------------------|-------------|\n")

	for _, d := range drivers {
		stats := r.DriverStats[d]
		if !stats.IsEmbedded || stats.HeapAllocMB == 0 {
			continue
		}
		sb.WriteString(fmt.Sprintf("| %s | %.2f | %.2f | %.1f | %.0f |\n",
			stats.Driver,
			stats.HeapAllocMB,
			stats.HeapInUseMB,
			stats.HeapObjectsK,
			stats.MemPerVectorB,
		))
	}
	sb.WriteString("\n")

	// Detailed results per driver
	sb.WriteString("## Detailed Results\n\n")

	// Group results by driver
	resultsByDriver := make(map[string][]Result)
	for _, result := range r.Results {
		resultsByDriver[result.Driver] = append(resultsByDriver[result.Driver], result)
	}

	for _, d := range drivers {
		results := resultsByDriver[d]
		if len(results) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("### %s\n\n", d))
		sb.WriteString("| Operation | Iterations | Avg (ms) | p50 (ms) | p95 (ms) | p99 (ms) | Throughput | Errors |\n")
		sb.WriteString("|-----------|------------|----------|----------|----------|----------|------------|--------|\n")

		for _, result := range results {
			throughputStr := "-"
			if result.Throughput > 0 {
				throughputStr = fmt.Sprintf("%.1f/s", result.Throughput)
			}
			sb.WriteString(fmt.Sprintf("| %s | %d | %.2f | %.2f | %.2f | %.2f | %s | %d |\n",
				result.Operation,
				result.Iterations,
				float64(result.AvgLatency.Microseconds())/1000,
				float64(result.P50Latency.Microseconds())/1000,
				float64(result.P95Latency.Microseconds())/1000,
				float64(result.P99Latency.Microseconds())/1000,
				throughputStr,
				result.Errors,
			))
		}
		sb.WriteString("\n")
	}

	// Performance comparison charts (ASCII)
	sb.WriteString("## Performance Comparison\n\n")
	sb.WriteString("### Insert Throughput (vectors/second)\n\n")
	sb.WriteString("```\n")
	r.writeBarChart(&sb, drivers, func(d string) float64 {
		if stats, ok := r.DriverStats[d]; ok {
			return stats.InsertThroughput
		}
		return 0
	})
	sb.WriteString("```\n\n")

	sb.WriteString("### Search Latency p50 (microseconds, lower is better)\n\n")
	sb.WriteString("```\n")
	r.writeBarChart(&sb, drivers, func(d string) float64 {
		if stats, ok := r.DriverStats[d]; ok {
			return float64(stats.SearchP50)
		}
		return 0
	})
	sb.WriteString("```\n\n")

	sb.WriteString("### Search QPS (queries per second)\n\n")
	sb.WriteString("```\n")
	r.writeBarChart(&sb, drivers, func(d string) float64 {
		if stats, ok := r.DriverStats[d]; ok {
			return stats.SearchQPS
		}
		return 0
	})
	sb.WriteString("```\n\n")

	// Errors section
	var hasErrors bool
	for _, stats := range r.DriverStats {
		if stats.TotalErrors > 0 {
			hasErrors = true
			break
		}
	}

	if hasErrors {
		sb.WriteString("## Errors\n\n")
		for _, d := range drivers {
			for _, result := range resultsByDriver[d] {
				if result.Errors > 0 && result.ErrorMsg != "" {
					sb.WriteString(fmt.Sprintf("- **%s/%s**: %s\n", d, result.Operation, result.ErrorMsg))
				}
			}
		}
		sb.WriteString("\n")
	}

	// Footer
	sb.WriteString("---\n")
	sb.WriteString("*Generated by pkg/vectorize/bench*\n")

	return sb.String()
}

func (r *Report) writeBarChart(sb *strings.Builder, drivers []string, getValue func(string) float64) {
	maxVal := 0.0
	for _, d := range drivers {
		val := getValue(d)
		if val > maxVal {
			maxVal = val
		}
	}

	maxWidth := 40
	for _, d := range drivers {
		val := getValue(d)
		barLen := 0
		if maxVal > 0 {
			barLen = int(val / maxVal * float64(maxWidth))
		}
		bar := strings.Repeat("█", barLen)
		sb.WriteString(fmt.Sprintf("%-15s |%s %.1f\n", d, bar, val))
	}
}
