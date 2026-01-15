package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Report holds all benchmark data for export.
type Report struct {
	Timestamp time.Time
	Config    *Config
	Results   []*BenchmarkResult
}

// NewReport creates a new report from benchmark results.
func NewReport(cfg *Config, results []*BenchmarkResult) *Report {
	return &Report{
		Timestamp: time.Now(),
		Config:    cfg,
		Results:   results,
	}
}

// SaveAll saves all report formats to the output directory.
func (r *Report) SaveAll(outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	if err := r.SaveCSV(filepath.Join(outputDir, "benchmark_results.csv")); err != nil {
		return fmt.Errorf("save CSV: %w", err)
	}

	if err := r.SaveJSON(filepath.Join(outputDir, "benchmark_results.json")); err != nil {
		return fmt.Errorf("save JSON: %w", err)
	}

	if err := r.SaveMarkdown(filepath.Join(outputDir, "benchmark_report.md")); err != nil {
		return fmt.Errorf("save markdown: %w", err)
	}

	return nil
}

// SaveCSV saves results to a CSV file.
func (r *Report) SaveCSV(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// Header
	headers := []string{
		"driver", "object_size_bytes", "threads", "throughput_mbps",
		"ttfb_avg_ms", "ttfb_min_ms", "ttfb_p25_ms", "ttfb_p50_ms",
		"ttfb_p75_ms", "ttfb_p90_ms", "ttfb_p99_ms", "ttfb_max_ms",
		"ttlb_avg_ms", "ttlb_min_ms", "ttlb_p25_ms", "ttlb_p50_ms",
		"ttlb_p75_ms", "ttlb_p90_ms", "ttlb_p99_ms", "ttlb_max_ms",
		"samples", "errors", "duration_ms",
	}
	if err := w.Write(headers); err != nil {
		return err
	}

	// Data rows
	for _, res := range r.Results {
		row := []string{
			res.Driver,
			fmt.Sprintf("%d", res.ObjectSize),
			fmt.Sprintf("%d", res.Threads),
			fmt.Sprintf("%.2f", res.Throughput),
			fmt.Sprintf("%d", res.TTFB.Avg.Milliseconds()),
			fmt.Sprintf("%d", res.TTFB.Min.Milliseconds()),
			fmt.Sprintf("%d", res.TTFB.P25.Milliseconds()),
			fmt.Sprintf("%d", res.TTFB.P50.Milliseconds()),
			fmt.Sprintf("%d", res.TTFB.P75.Milliseconds()),
			fmt.Sprintf("%d", res.TTFB.P90.Milliseconds()),
			fmt.Sprintf("%d", res.TTFB.P99.Milliseconds()),
			fmt.Sprintf("%d", res.TTFB.Max.Milliseconds()),
			fmt.Sprintf("%d", res.TTLB.Avg.Milliseconds()),
			fmt.Sprintf("%d", res.TTLB.Min.Milliseconds()),
			fmt.Sprintf("%d", res.TTLB.P25.Milliseconds()),
			fmt.Sprintf("%d", res.TTLB.P50.Milliseconds()),
			fmt.Sprintf("%d", res.TTLB.P75.Milliseconds()),
			fmt.Sprintf("%d", res.TTLB.P90.Milliseconds()),
			fmt.Sprintf("%d", res.TTLB.P99.Milliseconds()),
			fmt.Sprintf("%d", res.TTLB.Max.Milliseconds()),
			fmt.Sprintf("%d", res.Samples),
			fmt.Sprintf("%d", res.Errors),
			fmt.Sprintf("%d", res.Duration.Milliseconds()),
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// JSONReport is the JSON export format.
type JSONReport struct {
	Timestamp string          `json:"timestamp"`
	Config    JSONConfig      `json:"config"`
	Results   []JSONResult    `json:"results"`
	Summary   JSONSummary     `json:"summary"`
}

// JSONConfig is the JSON config export.
type JSONConfig struct {
	ThreadsMin  int `json:"threads_min"`
	ThreadsMax  int `json:"threads_max"`
	PayloadsMin int `json:"payloads_min"`
	PayloadsMax int `json:"payloads_max"`
	Samples     int `json:"samples"`
}

// JSONResult is a single result in JSON format.
type JSONResult struct {
	Driver      string      `json:"driver"`
	ObjectSize  int         `json:"object_size_bytes"`
	Threads     int         `json:"threads"`
	Throughput  float64     `json:"throughput_mbps"`
	TTFB        JSONLatency `json:"ttfb"`
	TTLB        JSONLatency `json:"ttlb"`
	Samples     int         `json:"samples"`
	Errors      int         `json:"errors"`
	DurationMs  int64       `json:"duration_ms"`
}

// JSONLatency is latency stats in JSON format.
type JSONLatency struct {
	AvgMs int64 `json:"avg_ms"`
	MinMs int64 `json:"min_ms"`
	P25Ms int64 `json:"p25_ms"`
	P50Ms int64 `json:"p50_ms"`
	P75Ms int64 `json:"p75_ms"`
	P90Ms int64 `json:"p90_ms"`
	P99Ms int64 `json:"p99_ms"`
	MaxMs int64 `json:"max_ms"`
}

// JSONSummary is the summary section.
type JSONSummary struct {
	BestThroughput JSONBestResult `json:"best_throughput"`
	BestTTFB       JSONBestResult `json:"best_ttfb"`
	BestTTLB       JSONBestResult `json:"best_ttlb"`
}

// JSONBestResult identifies the best performer.
type JSONBestResult struct {
	Driver     string  `json:"driver"`
	ObjectSize int     `json:"object_size_bytes"`
	Threads    int     `json:"threads"`
	Value      float64 `json:"value"`
	Unit       string  `json:"unit"`
}

// SaveJSON saves results to a JSON file.
func (r *Report) SaveJSON(path string) error {
	jr := JSONReport{
		Timestamp: r.Timestamp.Format(time.RFC3339),
		Config: JSONConfig{
			ThreadsMin:  r.Config.ThreadsMin,
			ThreadsMax:  r.Config.ThreadsMax,
			PayloadsMin: r.Config.PayloadsMin,
			PayloadsMax: r.Config.PayloadsMax,
			Samples:     r.Config.Samples,
		},
		Results: make([]JSONResult, 0, len(r.Results)),
	}

	var bestThroughput *BenchmarkResult
	var bestTTFB *BenchmarkResult
	var bestTTLB *BenchmarkResult

	for _, res := range r.Results {
		jr.Results = append(jr.Results, JSONResult{
			Driver:     res.Driver,
			ObjectSize: res.ObjectSize,
			Threads:    res.Threads,
			Throughput: res.Throughput,
			TTFB: JSONLatency{
				AvgMs: res.TTFB.Avg.Milliseconds(),
				MinMs: res.TTFB.Min.Milliseconds(),
				P25Ms: res.TTFB.P25.Milliseconds(),
				P50Ms: res.TTFB.P50.Milliseconds(),
				P75Ms: res.TTFB.P75.Milliseconds(),
				P90Ms: res.TTFB.P90.Milliseconds(),
				P99Ms: res.TTFB.P99.Milliseconds(),
				MaxMs: res.TTFB.Max.Milliseconds(),
			},
			TTLB: JSONLatency{
				AvgMs: res.TTLB.Avg.Milliseconds(),
				MinMs: res.TTLB.Min.Milliseconds(),
				P25Ms: res.TTLB.P25.Milliseconds(),
				P50Ms: res.TTLB.P50.Milliseconds(),
				P75Ms: res.TTLB.P75.Milliseconds(),
				P90Ms: res.TTLB.P90.Milliseconds(),
				P99Ms: res.TTLB.P99.Milliseconds(),
				MaxMs: res.TTLB.Max.Milliseconds(),
			},
			Samples:    res.Samples,
			Errors:     res.Errors,
			DurationMs: res.Duration.Milliseconds(),
		})

		// Track best results
		if bestThroughput == nil || res.Throughput > bestThroughput.Throughput {
			bestThroughput = res
		}
		if bestTTFB == nil || (res.TTFB.Avg > 0 && res.TTFB.Avg < bestTTFB.TTFB.Avg) {
			bestTTFB = res
		}
		if bestTTLB == nil || (res.TTLB.Avg > 0 && res.TTLB.Avg < bestTTLB.TTLB.Avg) {
			bestTTLB = res
		}
	}

	if bestThroughput != nil {
		jr.Summary.BestThroughput = JSONBestResult{
			Driver:     bestThroughput.Driver,
			ObjectSize: bestThroughput.ObjectSize,
			Threads:    bestThroughput.Threads,
			Value:      bestThroughput.Throughput,
			Unit:       "MB/s",
		}
	}
	if bestTTFB != nil {
		jr.Summary.BestTTFB = JSONBestResult{
			Driver:     bestTTFB.Driver,
			ObjectSize: bestTTFB.ObjectSize,
			Threads:    bestTTFB.Threads,
			Value:      float64(bestTTFB.TTFB.Avg.Milliseconds()),
			Unit:       "ms",
		}
	}
	if bestTTLB != nil {
		jr.Summary.BestTTLB = JSONBestResult{
			Driver:     bestTTLB.Driver,
			ObjectSize: bestTTLB.ObjectSize,
			Threads:    bestTTLB.Threads,
			Value:      float64(bestTTLB.TTLB.Avg.Milliseconds()),
			Unit:       "ms",
		}
	}

	data, err := json.MarshalIndent(jr, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// SaveMarkdown saves results to a Markdown file.
func (r *Report) SaveMarkdown(path string) error {
	var sb strings.Builder

	sb.WriteString("# S3 Benchmark Results\n\n")
	sb.WriteString(fmt.Sprintf("**Date:** %s\n\n", r.Timestamp.Format("2006-01-02 15:04:05")))

	// Configuration
	sb.WriteString("## Configuration\n\n")
	sb.WriteString(fmt.Sprintf("- Threads: %d - %d\n", r.Config.ThreadsMin, r.Config.ThreadsMax))
	sb.WriteString(fmt.Sprintf("- Payload sizes: %s - %s\n",
		FormatSize(1<<(r.Config.PayloadsMin+10)),
		FormatSize(1<<(r.Config.PayloadsMax+10))))
	sb.WriteString(fmt.Sprintf("- Samples per test: %d\n\n", r.Config.Samples))

	// Drivers
	drivers := make(map[string]bool)
	for _, res := range r.Results {
		drivers[res.Driver] = true
	}
	driverList := make([]string, 0, len(drivers))
	for d := range drivers {
		driverList = append(driverList, d)
	}
	sort.Strings(driverList)
	sb.WriteString(fmt.Sprintf("- Drivers: %s\n\n", strings.Join(driverList, ", ")))

	// Summary
	sb.WriteString("## Summary\n\n")

	var bestThroughput, bestTTFB, bestTTLB *BenchmarkResult
	for _, res := range r.Results {
		if bestThroughput == nil || res.Throughput > bestThroughput.Throughput {
			bestThroughput = res
		}
		if bestTTFB == nil || (res.TTFB.Avg > 0 && res.TTFB.Avg < bestTTFB.TTFB.Avg) {
			bestTTFB = res
		}
		if bestTTLB == nil || (res.TTLB.Avg > 0 && res.TTLB.Avg < bestTTLB.TTLB.Avg) {
			bestTTLB = res
		}
	}

	if bestThroughput != nil {
		sb.WriteString(fmt.Sprintf("- **Best Throughput:** %s (%.1f MB/s with %s, %d threads)\n",
			bestThroughput.Driver, bestThroughput.Throughput, FormatSize(bestThroughput.ObjectSize), bestThroughput.Threads))
	}
	if bestTTFB != nil {
		sb.WriteString(fmt.Sprintf("- **Best TTFB:** %s (%d ms avg)\n",
			bestTTFB.Driver, bestTTFB.TTFB.Avg.Milliseconds()))
	}
	if bestTTLB != nil {
		sb.WriteString(fmt.Sprintf("- **Best TTLB:** %s (%d ms avg)\n",
			bestTTLB.Driver, bestTTLB.TTLB.Avg.Milliseconds()))
	}
	sb.WriteString("\n")

	// Results by object size
	sizes := make(map[int]bool)
	for _, res := range r.Results {
		sizes[res.ObjectSize] = true
	}
	sizeList := make([]int, 0, len(sizes))
	for s := range sizes {
		sizeList = append(sizeList, s)
	}
	sort.Ints(sizeList)

	for _, size := range sizeList {
		sb.WriteString(fmt.Sprintf("## %s Objects\n\n", FormatSize(size)))

		// Table header
		sb.WriteString("| Driver | Threads | Throughput | TTFB p50 | TTFB p99 | TTLB p50 | TTLB p99 |\n")
		sb.WriteString("|--------|---------|------------|----------|----------|----------|----------|\n")

		// Filter and sort results for this size
		var sizeResults []*BenchmarkResult
		for _, res := range r.Results {
			if res.ObjectSize == size {
				sizeResults = append(sizeResults, res)
			}
		}
		sort.Slice(sizeResults, func(i, j int) bool {
			if sizeResults[i].Driver != sizeResults[j].Driver {
				return sizeResults[i].Driver < sizeResults[j].Driver
			}
			return sizeResults[i].Threads < sizeResults[j].Threads
		})

		for _, res := range sizeResults {
			sb.WriteString(fmt.Sprintf("| %s | %d | %.1f MB/s | %d ms | %d ms | %d ms | %d ms |\n",
				res.Driver, res.Threads, res.Throughput,
				res.TTFB.P50.Milliseconds(), res.TTFB.P99.Milliseconds(),
				res.TTLB.P50.Milliseconds(), res.TTLB.P99.Milliseconds()))
		}
		sb.WriteString("\n")
	}

	return os.WriteFile(path, []byte(sb.String()), 0644)
}
