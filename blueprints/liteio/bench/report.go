package bench

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

// =============================================================================
// REPORT FORMAT CONSTANTS
// =============================================================================
// All display-related magic numbers consolidated here for easy tuning.

const (
	// ---------------------------------------------------------------------------
	// Performance Target Thresholds
	// ---------------------------------------------------------------------------

	// TargetRatio is the target performance multiplier for liteio vs competitors.
	// liteio should be this many times faster than other drivers.
	TargetRatio = 5.0

	// PassThreshold: ratios >= this value show as PASS (green).
	PassThreshold = 4.5

	// WarnThreshold: ratios >= this value but < PassThreshold show as WARN (yellow).
	WarnThreshold = 3.0

	// FailThreshold: ratios < WarnThreshold show as FAIL (red).
	// Anything below WarnThreshold is considered failing the 5x target.
	FailThreshold = 0.0

	// ---------------------------------------------------------------------------
	// Display Formatting
	// ---------------------------------------------------------------------------

	// DriverColumnWidth is the width for driver name columns.
	DriverColumnWidth = 12

	// MetricColumnWidth is the width for metric value columns.
	MetricColumnWidth = 12

	// ---------------------------------------------------------------------------
	// Comparison Thresholds
	// ---------------------------------------------------------------------------

	// SignificantDifferenceRatio is the minimum ratio to show in comparisons.
	// Ratios below this are considered "close competition".
	SignificantDifferenceRatio = 1.1

	// MajorDifferenceRatio indicates a significant performance gap.
	MajorDifferenceRatio = 2.0

	// ---------------------------------------------------------------------------
	// Report Section Limits
	// ---------------------------------------------------------------------------

	// MaxLeaderCategories is the maximum categories shown in leader table.
	MaxLeaderCategories = 12

	// MaxComparisonRows is the maximum rows in comparison tables.
	MaxComparisonRows = 50

	// ---------------------------------------------------------------------------
	// Reference Driver
	// ---------------------------------------------------------------------------

	// ReferenceDriver is the driver used as a theoretical baseline reference.
	// It is excluded from main comparisons but shown separately for context.
	ReferenceDriver = "devnull"
)

// BenchmarkEntry represents a parsed benchmark result.
type BenchmarkEntry struct {
	Name       string  `json:"name"`
	Driver     string  `json:"driver"`
	Category   string  `json:"category"`
	SubTest    string  `json:"subtest"`
	Iterations int64   `json:"iterations"`
	NsPerOp    float64 `json:"ns_per_op"`
	MBPerSec   float64 `json:"mb_per_sec,omitempty"`
	BytesPerOp int64   `json:"bytes_per_op,omitempty"`
	AllocsOp   int64   `json:"allocs_per_op,omitempty"`
}

// BenchmarkReport contains all benchmark data for report generation.
type BenchmarkReport struct {
	Timestamp   time.Time                   `json:"timestamp"`
	GoVersion   string                      `json:"go_version"`
	Platform    string                      `json:"platform"`
	Entries     []BenchmarkEntry            `json:"entries"`
	ByDriver    map[string][]BenchmarkEntry `json:"by_driver"`
	ByCategory  map[string][]BenchmarkEntry `json:"by_category"`
	Comparisons []DriverComparison          `json:"comparisons"`
}

// DriverComparison compares performance across drivers.
type DriverComparison struct {
	Benchmark string             `json:"benchmark"`
	Results   map[string]float64 `json:"results"` // driver -> ns/op
	Fastest   string             `json:"fastest"`
	Slowest   string             `json:"slowest"`
	Ratio     float64            `json:"ratio"` // slowest/fastest
}

// ParseBenchOutput parses the output from `go test -bench`.
func ParseBenchOutput(output string) *BenchmarkReport {
	report := &BenchmarkReport{
		Timestamp:  time.Now(),
		GoVersion:  runtime.Version(),
		Platform:   fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		ByDriver:   make(map[string][]BenchmarkEntry),
		ByCategory: make(map[string][]BenchmarkEntry),
	}

	// Regex to parse benchmark lines
	// BenchmarkWrite/memory/Small_1KB-8    100000    10234 ns/op    100.50 MB/s    1024 B/op    5 allocs/op
	benchRegex := regexp.MustCompile(`^(Benchmark\S+)-\d+\s+(\d+)\s+([\d.]+)\s+ns/op`)

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		match := benchRegex.FindStringSubmatch(line)
		if match == nil {
			continue
		}

		entry := BenchmarkEntry{
			Name: match[1],
		}

		// Parse iterations and ns/op
		entry.Iterations, _ = strconv.ParseInt(match[2], 10, 64)
		entry.NsPerOp, _ = strconv.ParseFloat(match[3], 64)

		// Parse optional fields
		if mbMatch := regexp.MustCompile(`([\d.]+)\s+MB/s`).FindStringSubmatch(line); mbMatch != nil {
			entry.MBPerSec, _ = strconv.ParseFloat(mbMatch[1], 64)
		}
		if bytesMatch := regexp.MustCompile(`(\d+)\s+B/op`).FindStringSubmatch(line); bytesMatch != nil {
			entry.BytesPerOp, _ = strconv.ParseInt(bytesMatch[1], 10, 64)
		}
		if allocsMatch := regexp.MustCompile(`(\d+)\s+allocs/op`).FindStringSubmatch(line); allocsMatch != nil {
			entry.AllocsOp, _ = strconv.ParseInt(allocsMatch[1], 10, 64)
		}

		// Parse name parts: BenchmarkCategory/Driver/SubTest
		nameParts := strings.Split(strings.TrimPrefix(entry.Name, "Benchmark"), "/")
		if len(nameParts) >= 2 {
			entry.Category = nameParts[0]
			entry.Driver = nameParts[1]
			if len(nameParts) > 2 {
				entry.SubTest = strings.Join(nameParts[2:], "/")
			}
		}

		report.Entries = append(report.Entries, entry)
		report.ByDriver[entry.Driver] = append(report.ByDriver[entry.Driver], entry)
		report.ByCategory[entry.Category] = append(report.ByCategory[entry.Category], entry)
	}

	// Generate comparisons
	report.Comparisons = generateComparisons(report.Entries)

	return report
}

// generateComparisons creates driver comparison data.
func generateComparisons(entries []BenchmarkEntry) []DriverComparison {
	// Group by benchmark (category + subtest)
	byBench := make(map[string]map[string]float64)

	for _, e := range entries {
		benchKey := e.Category
		if e.SubTest != "" {
			benchKey += "/" + e.SubTest
		}

		if byBench[benchKey] == nil {
			byBench[benchKey] = make(map[string]float64)
		}
		byBench[benchKey][e.Driver] = e.NsPerOp
	}

	var comparisons []DriverComparison
	for bench, results := range byBench {
		if len(results) < 2 {
			continue
		}

		comp := DriverComparison{
			Benchmark: bench,
			Results:   results,
		}

		// Find fastest and slowest
		var minNs, maxNs float64 = 1e18, 0
		for driver, ns := range results {
			if ns < minNs {
				minNs = ns
				comp.Fastest = driver
			}
			if ns > maxNs {
				maxNs = ns
				comp.Slowest = driver
			}
		}
		comp.Ratio = maxNs / minNs

		comparisons = append(comparisons, comp)
	}

	// Sort by benchmark name
	sort.Slice(comparisons, func(i, j int) bool {
		return comparisons[i].Benchmark < comparisons[j].Benchmark
	})

	return comparisons
}

// GenerateMarkdown creates a markdown report from benchmark data.
func GenerateMarkdown(report *BenchmarkReport) string {
	var sb strings.Builder

	// Header
	sb.WriteString("# Storage Benchmark Report\n\n")
	sb.WriteString(fmt.Sprintf("**Generated:** %s\n\n", report.Timestamp.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("**Go Version:** %s\n\n", report.GoVersion))
	sb.WriteString(fmt.Sprintf("**Platform:** %s\n\n", report.Platform))

	// Table of Contents
	sb.WriteString("## Table of Contents\n\n")
	sb.WriteString("1. [Executive Summary](#executive-summary)\n")
	sb.WriteString("2. [Driver Comparison](#driver-comparison)\n")
	sb.WriteString("3. [Detailed Results by Category](#detailed-results-by-category)\n")
	sb.WriteString("4. [Performance Analysis](#performance-analysis)\n")
	sb.WriteString("5. [Recommendations](#recommendations)\n\n")

	// Executive Summary
	sb.WriteString("## Executive Summary\n\n")
	sb.WriteString("### Drivers Tested\n\n")
	drivers := make([]string, 0, len(report.ByDriver))
	for d := range report.ByDriver {
		drivers = append(drivers, d)
	}
	sort.Strings(drivers)
	for _, d := range drivers {
		count := len(report.ByDriver[d])
		sb.WriteString(fmt.Sprintf("- **%s**: %d benchmarks\n", d, count))
	}
	sb.WriteString("\n")

	// Categories tested
	sb.WriteString("### Categories Tested\n\n")
	categories := make([]string, 0, len(report.ByCategory))
	for c := range report.ByCategory {
		categories = append(categories, c)
	}
	sort.Strings(categories)
	for _, c := range categories {
		count := len(report.ByCategory[c])
		sb.WriteString(fmt.Sprintf("- **%s**: %d benchmarks\n", c, count))
	}
	sb.WriteString("\n")

	// Driver Comparison
	sb.WriteString("## Driver Comparison\n\n")
	sb.WriteString("### Performance Leaders by Operation\n\n")
	sb.WriteString("| Benchmark | Fastest | Slowest | Ratio |\n")
	sb.WriteString("|-----------|---------|---------|-------|\n")

	for _, comp := range report.Comparisons {
		if comp.Ratio > 1.1 { // Only show significant differences
			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %.2fx |\n",
				comp.Benchmark, comp.Fastest, comp.Slowest, comp.Ratio))
		}
	}
	sb.WriteString("\n")

	// Detailed Results by Category
	sb.WriteString("## Detailed Results by Category\n\n")

	for _, cat := range categories {
		entries := report.ByCategory[cat]
		if len(entries) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("### %s\n\n", cat))
		sb.WriteString("| Driver | Sub-test | ops/sec | MB/s | Allocs/op | B/op |\n")
		sb.WriteString("|--------|----------|---------|------|-----------|------|\n")

		// Sort entries by driver then subtest
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].Driver != entries[j].Driver {
				return entries[i].Driver < entries[j].Driver
			}
			return entries[i].SubTest < entries[j].SubTest
		})

		for _, e := range entries {
			opsPerSec := 1e9 / e.NsPerOp
			sb.WriteString(fmt.Sprintf("| %s | %s | %.0f | %.2f | %d | %d |\n",
				e.Driver, e.SubTest, opsPerSec, e.MBPerSec, e.AllocsOp, e.BytesPerOp))
		}
		sb.WriteString("\n")
	}

	// Performance Analysis
	sb.WriteString("## Performance Analysis\n\n")

	// Find best performers
	sb.WriteString("### Overall Performance Rankings\n\n")

	// Calculate average performance per driver
	driverAvg := make(map[string]float64)
	driverCount := make(map[string]int)
	for _, e := range report.Entries {
		if e.NsPerOp > 0 {
			driverAvg[e.Driver] += e.NsPerOp
			driverCount[e.Driver]++
		}
	}

	type driverRank struct {
		name  string
		avgNs float64
		count int
	}
	var ranks []driverRank
	for d, total := range driverAvg {
		ranks = append(ranks, driverRank{
			name:  d,
			avgNs: total / float64(driverCount[d]),
			count: driverCount[d],
		})
	}
	sort.Slice(ranks, func(i, j int) bool {
		return ranks[i].avgNs < ranks[j].avgNs
	})

	sb.WriteString("| Rank | Driver | Avg ns/op | Benchmarks |\n")
	sb.WriteString("|------|--------|-----------|------------|\n")
	for i, r := range ranks {
		sb.WriteString(fmt.Sprintf("| %d | %s | %.0f | %d |\n",
			i+1, r.name, r.avgNs, r.count))
	}
	sb.WriteString("\n")

	// Recommendations
	sb.WriteString("## Recommendations\n\n")

	if len(ranks) > 0 {
		sb.WriteString(fmt.Sprintf("### Best Overall: %s\n\n", ranks[0].name))
		sb.WriteString("Based on average performance across all benchmarks.\n\n")
	}

	// Category-specific recommendations
	sb.WriteString("### By Use Case\n\n")

	// Find best for writes
	writeBest := findBestDriver(report.ByCategory["Write"])
	if writeBest != "" {
		sb.WriteString(fmt.Sprintf("- **Write-heavy workloads:** %s\n", writeBest))
	}

	readBest := findBestDriver(report.ByCategory["Read"])
	if readBest != "" {
		sb.WriteString(fmt.Sprintf("- **Read-heavy workloads:** %s\n", readBest))
	}

	listBest := findBestDriver(report.ByCategory["List"])
	if listBest != "" {
		sb.WriteString(fmt.Sprintf("- **List operations:** %s\n", listBest))
	}

	parallelBest := findBestDriver(report.ByCategory["ParallelWrite"])
	if parallelBest != "" {
		sb.WriteString(fmt.Sprintf("- **High concurrency:** %s\n", parallelBest))
	}

	sb.WriteString("\n---\n\n")
	sb.WriteString("*Report generated by storage benchmark suite*\n")

	return sb.String()
}

// findBestDriver finds the best performing driver for a category.
func findBestDriver(entries []BenchmarkEntry) string {
	if len(entries) == 0 {
		return ""
	}

	driverTotal := make(map[string]float64)
	driverCount := make(map[string]int)

	for _, e := range entries {
		if e.NsPerOp > 0 {
			driverTotal[e.Driver] += e.NsPerOp
			driverCount[e.Driver]++
		}
	}

	var bestDriver string
	var bestAvg float64 = 1e18

	for d, total := range driverTotal {
		avg := total / float64(driverCount[d])
		if avg < bestAvg {
			bestAvg = avg
			bestDriver = d
		}
	}

	return bestDriver
}

// WriteReport writes the benchmark report to a file.
func WriteReport(benchOutput, outputPath string) error {
	report := ParseBenchOutput(benchOutput)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("create report directory: %w", err)
	}

	// Generate markdown
	markdown := GenerateMarkdown(report)

	// Write markdown report
	if err := os.WriteFile(outputPath, []byte(markdown), 0644); err != nil {
		return fmt.Errorf("write markdown report: %w", err)
	}

	// Also write JSON for further analysis
	jsonPath := strings.TrimSuffix(outputPath, ".md") + ".json"
	jsonData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	if err := os.WriteFile(jsonPath, jsonData, 0644); err != nil {
		return fmt.Errorf("write json report: %w", err)
	}

	return nil
}

// SkippedBenchmark records a benchmark that was skipped.
type SkippedBenchmark struct {
	Driver    string `json:"driver"`
	Operation string `json:"operation"`
	Reason    string `json:"reason"`
}

// Report holds complete benchmark results from CLI runner.
type Report struct {
	Timestamp          time.Time                    `json:"timestamp"`
	Config             *Config                      `json:"config"`
	Results            []*Metrics                   `json:"results"`
	DockerStats        map[string]*DockerStats      `json:"docker_stats,omitempty"`
	SkippedBenchmarks  []SkippedBenchmark           `json:"skipped_benchmarks,omitempty"`
	ResourceSnapshots  map[string]*ResourceSummary  `json:"resource_snapshots,omitempty"`
	ServerMetrics      map[string]*ServerMetrics    `json:"server_metrics,omitempty"`
	ProfileAnalyses    map[string]*ProfileAnalysis  `json:"profile_analyses,omitempty"`
}

// SaveJSON saves the report as JSON.
func (r *Report) SaveJSON(outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	jsonPath := filepath.Join(outputDir, "raw_results.json")
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}

	return os.WriteFile(jsonPath, data, 0644)
}

// SaveMarkdown saves the report as Markdown.
func (r *Report) SaveMarkdown(outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	markdown := r.generateMarkdown()
	mdPath := filepath.Join(outputDir, "report.md")
	return os.WriteFile(mdPath, []byte(markdown), 0644)
}

// SaveCSV saves the report as CSV for spreadsheet analysis.
func (r *Report) SaveCSV(outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	csvPath := filepath.Join(outputDir, "benchmark_results.csv")
	f, err := os.Create(csvPath)
	if err != nil {
		return fmt.Errorf("create csv file: %w", err)
	}
	defer f.Close()

	// Write CSV header
	header := "driver,operation,object_size,iterations,throughput_mbps,ops_per_sec,avg_latency_ms,p50_ms,p95_ms,p99_ms,ttfb_avg_ms,ttfb_p50_ms,ttfb_p95_ms,ttfb_p99_ms,errors\n"
	f.WriteString(header)

	// Write data rows
	for _, m := range r.Results {
		// Convert latencies to milliseconds
		avgMs := float64(m.AvgLatency.Nanoseconds()) / 1e6
		p50Ms := float64(m.P50Latency.Nanoseconds()) / 1e6
		p95Ms := float64(m.P95Latency.Nanoseconds()) / 1e6
		p99Ms := float64(m.P99Latency.Nanoseconds()) / 1e6
		ttfbAvgMs := float64(m.TTFBAvg.Nanoseconds()) / 1e6
		ttfbP50Ms := float64(m.TTFBP50.Nanoseconds()) / 1e6
		ttfbP95Ms := float64(m.TTFBP95.Nanoseconds()) / 1e6
		ttfbP99Ms := float64(m.TTFBP99.Nanoseconds()) / 1e6

		row := fmt.Sprintf("%s,%s,%d,%d,%.4f,%.2f,%.4f,%.4f,%.4f,%.4f,%.4f,%.4f,%.4f,%.4f,%d\n",
			m.Driver,
			m.Operation,
			m.ObjectSize,
			m.Iterations,
			m.Throughput,
			m.OpsPerSec,
			avgMs,
			p50Ms,
			p95Ms,
			p99Ms,
			ttfbAvgMs,
			ttfbP50Ms,
			ttfbP95Ms,
			ttfbP99Ms,
			m.Errors,
		)
		f.WriteString(row)
	}

	return nil
}

// SaveSummary saves a summary report showing best driver for each category.
func (r *Report) SaveSummary(outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	summary := r.generateSummary()
	summaryPath := filepath.Join(outputDir, "summary.md")
	return os.WriteFile(summaryPath, []byte(summary), 0644)
}

// generateSummary creates a concise summary showing the best driver for each category.
func (r *Report) generateSummary() string {
	var sb strings.Builder

	sb.WriteString("# Storage Benchmark Summary\n\n")
	sb.WriteString(fmt.Sprintf("**Generated:** %s\n\n", r.Timestamp.Format(time.RFC3339)))

	// Group results by operation (excluding reference driver)
	byOperation := make(map[string][]*Metrics)
	for _, m := range r.Results {
		if !isReferenceDriver(m.Driver) {
			byOperation[m.Operation] = append(byOperation[m.Operation], m)
		}
	}

	// Get sorted list of operations
	operations := make([]string, 0, len(byOperation))
	for op := range byOperation {
		operations = append(operations, op)
	}
	sort.Strings(operations)

	// Categorize operations
	type categoryResult struct {
		operation string
		winner    string
		winnerVal float64
		runnerUp  string
		runnerVal float64
		unit      string
		margin    string
	}

	var results []categoryResult

	for _, op := range operations {
		metrics := byOperation[op]
		if len(metrics) < 2 {
			continue
		}

		// Sort by throughput (descending)
		sort.Slice(metrics, func(i, j int) bool {
			return metrics[i].Throughput > metrics[j].Throughput
		})

		winner := metrics[0]
		runnerUp := metrics[1]

		// Determine unit
		unit := "ops/s"
		if winner.ObjectSize > 0 {
			unit = "MB/s"
		}

		// Calculate margin
		var margin string
		if runnerUp.Throughput > 0 {
			factor := winner.Throughput / runnerUp.Throughput
			if factor >= 2.0 {
				margin = fmt.Sprintf("%.1fx faster", factor)
			} else if factor >= 1.1 {
				margin = fmt.Sprintf("+%.0f%%", (factor-1)*100)
			} else {
				margin = "~equal"
			}
		}

		results = append(results, categoryResult{
			operation: op,
			winner:    winner.Driver,
			winnerVal: winner.Throughput,
			runnerUp:  runnerUp.Driver,
			runnerVal: runnerUp.Throughput,
			unit:      unit,
			margin:    margin,
		})
	}

	// Overall wins tally
	wins := make(map[string]int)
	for _, res := range results {
		wins[res.winner]++
	}

	// Sort drivers by wins
	type driverWins struct {
		name string
		wins int
	}
	var rankings []driverWins
	for d, w := range wins {
		rankings = append(rankings, driverWins{d, w})
	}
	sort.Slice(rankings, func(i, j int) bool {
		return rankings[i].wins > rankings[j].wins
	})

	// Overall Winner Section
	sb.WriteString("## Overall Winner\n\n")
	if len(rankings) > 0 {
		winner := rankings[0]
		pct := float64(winner.wins) / float64(len(results)) * 100
		sb.WriteString(fmt.Sprintf("**%s** won %d/%d categories (%.0f%%)\n\n", winner.name, winner.wins, len(results), pct))

		if len(rankings) > 1 {
			sb.WriteString("### Win Counts\n\n")
			sb.WriteString("| Driver | Wins | Percentage |\n")
			sb.WriteString("|--------|------|------------|\n")
			for _, r := range rankings {
				pct := float64(r.wins) / float64(len(results)) * 100
				sb.WriteString(fmt.Sprintf("| %s | %d | %.0f%% |\n", r.name, r.wins, pct))
			}
			sb.WriteString("\n")
		}
	}

	// Best Driver by Category Table
	sb.WriteString("## Best Driver by Category\n\n")
	sb.WriteString("| Category | Winner | Performance | Runner-up | Runner-up Perf | Margin |\n")
	sb.WriteString("|----------|--------|-------------|-----------|----------------|--------|\n")

	for _, res := range results {
		winnerPerf := formatPerformance(res.winnerVal, res.unit)
		runnerPerf := formatPerformance(res.runnerVal, res.unit)
		sb.WriteString(fmt.Sprintf("| %s | **%s** | %s | %s | %s | %s |\n",
			res.operation, res.winner, winnerPerf, res.runnerUp, runnerPerf, res.margin))
	}
	sb.WriteString("\n")

	// Category Summaries by Operation Type
	sb.WriteString("## Category Summaries\n\n")

	// Group by operation type prefix
	opGroups := map[string][]categoryResult{
		"Write":         {},
		"Read":          {},
		"ParallelWrite": {},
		"ParallelRead":  {},
		"Delete":        {},
		"Stat":          {},
		"List":          {},
		"Copy":          {},
		"Scale":         {},
	}

	for _, res := range results {
		for prefix := range opGroups {
			if strings.HasPrefix(res.operation, prefix) {
				opGroups[prefix] = append(opGroups[prefix], res)
				break
			}
		}
	}

	// Print summaries for each group
	groupOrder := []string{"Write", "Read", "ParallelWrite", "ParallelRead", "Delete", "Stat", "List", "Copy", "Scale"}
	for _, group := range groupOrder {
		groupResults := opGroups[group]
		if len(groupResults) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("### %s Operations\n\n", group))

		// Find most common winner in this group
		groupWins := make(map[string]int)
		for _, res := range groupResults {
			groupWins[res.winner]++
		}

		var bestDriver string
		var bestWins int
		for d, w := range groupWins {
			if w > bestWins {
				bestWins = w
				bestDriver = d
			}
		}

		sb.WriteString(fmt.Sprintf("**Best for %s:** %s (won %d/%d)\n\n", group, bestDriver, bestWins, len(groupResults)))

		sb.WriteString("| Operation | Winner | Performance | vs Runner-up |\n")
		sb.WriteString("|-----------|--------|-------------|-------------|\n")
		for _, res := range groupResults {
			perf := formatPerformance(res.winnerVal, res.unit)
			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", res.operation, res.winner, perf, res.margin))
		}
		sb.WriteString("\n")
	}

	// Runtime resource usage in summary
	if len(r.ResourceSnapshots) > 0 {
		sb.WriteString("## Runtime Resource Usage\n\n")
		sb.WriteString("| Driver | Peak RSS | Go Heap | Go Sys | Disk Usage | GC Cycles |\n")
		sb.WriteString("|--------|----------|---------|--------|------------|----------|\n")

		// Collect and sort driver names for consistent output.
		var rsDrivers []string
		for d := range r.ResourceSnapshots {
			rsDrivers = append(rsDrivers, d)
		}
		sort.Strings(rsDrivers)
		for _, d := range rsDrivers {
			rs := r.ResourceSnapshots[d]
			sb.WriteString(fmt.Sprintf("| %s | %.1f MB | %.1f MB | %.1f MB | %.1f MB | %d |\n",
				d, rs.PeakRSSMB, rs.PeakHeapMB, rs.PeakSysMB, rs.FinalDiskMB, rs.NumGC))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("---\n\n")
	sb.WriteString("*Generated by storage benchmark CLI*\n")

	return sb.String()
}

// formatPerformance formats a performance value with appropriate unit.
func formatPerformance(val float64, unit string) string {
	if unit == "MB/s" {
		if val >= 1000 {
			return fmt.Sprintf("%.1f GB/s", val/1000)
		}
		return fmt.Sprintf("%.1f MB/s", val)
	}
	if val >= 1000000 {
		return fmt.Sprintf("%.1fM %s", val/1000000, unit)
	}
	if val >= 1000 {
		return fmt.Sprintf("%.1fK %s", val/1000, unit)
	}
	return fmt.Sprintf("%.0f %s", val, unit)
}

// SaveAll saves report in all configured formats.
func (r *Report) SaveAll(outputDir string, formats []string) error {
	for _, format := range formats {
		switch format {
		case "json":
			if err := r.SaveJSON(outputDir); err != nil {
				return fmt.Errorf("save json: %w", err)
			}
		case "markdown":
			if err := r.SaveMarkdown(outputDir); err != nil {
				return fmt.Errorf("save markdown: %w", err)
			}
			// Also generate summary.md alongside the full report
			if err := r.SaveSummary(outputDir); err != nil {
				return fmt.Errorf("save summary: %w", err)
			}
		case "csv":
			if err := r.SaveCSV(outputDir); err != nil {
				return fmt.Errorf("save csv: %w", err)
			}
		}
	}
	return nil
}

// isReferenceDriver returns true if the driver is the reference/baseline driver.
func isReferenceDriver(driver string) bool {
	return driver == ReferenceDriver
}

// filterNonReferenceResults filters out the reference driver from results.
func filterNonReferenceResults(results []*Metrics) []*Metrics {
	var filtered []*Metrics
	for _, m := range results {
		if !isReferenceDriver(m.Driver) {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

func (r *Report) generateMarkdown() string {
	var sb strings.Builder

	// Collect all drivers (excluding reference) and index results
	driverList, byDriver, byOperation := r.indexResults()

	// =========================================================================
	// 1. Header + System Info
	// =========================================================================
	sb.WriteString("# Storage Benchmark Report\n\n")

	sb.WriteString("## System Info\n\n")
	sb.WriteString("| Property | Value |\n")
	sb.WriteString("|----------|-------|\n")
	sb.WriteString(fmt.Sprintf("| Timestamp | %s |\n", r.Timestamp.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("| Go Version | %s |\n", runtime.Version()))
	sb.WriteString(fmt.Sprintf("| Platform | %s/%s |\n", runtime.GOOS, runtime.GOARCH))
	sb.WriteString(fmt.Sprintf("| CPUs | %d |\n", runtime.NumCPU()))
	if r.Config != nil {
		sb.WriteString(fmt.Sprintf("| BenchTime | %v |\n", r.Config.BenchTime))
		sb.WriteString(fmt.Sprintf("| Concurrency | %d |\n", r.Config.Concurrency))
		sb.WriteString(fmt.Sprintf("| Levels | %v |\n", r.Config.ConcurrencyLevels))
		sizeLabels := make([]string, len(r.Config.ObjectSizes))
		for i, s := range r.Config.ObjectSizes {
			sizeLabels[i] = SizeLabel(s)
		}
		sb.WriteString(fmt.Sprintf("| Object Sizes | %s |\n", strings.Join(sizeLabels, ", ")))
		sb.WriteString(fmt.Sprintf("| Warmup | %d iterations |\n", r.Config.WarmupIterations))
		sb.WriteString(fmt.Sprintf("| Timeout | %v |\n", r.Config.Timeout))
	}
	sb.WriteString(fmt.Sprintf("| Drivers | %d |\n", len(driverList)))
	sb.WriteString("\n")

	// Table of Contents
	sb.WriteString("## Table of Contents\n\n")
	sb.WriteString("1. [Executive Summary Dashboard](#executive-summary-dashboard)\n")
	sb.WriteString("2. [Performance Matrix](#performance-matrix)\n")
	sb.WriteString("3. [Write Performance Deep Dive](#write-performance-deep-dive)\n")
	sb.WriteString("4. [Read Performance Deep Dive](#read-performance-deep-dive)\n")
	sb.WriteString("5. [Parallel Scalability](#parallel-scalability)\n")
	sb.WriteString("6. [Latency Analysis](#latency-analysis)\n")
	sb.WriteString("7. [Resource Efficiency](#resource-efficiency)\n")
	sb.WriteString("8. [Error & Timeout Summary](#error--timeout-summary)\n")
	sb.WriteString("9. [Recommendations](#recommendations)\n\n")

	// =========================================================================
	// 2. Executive Summary Dashboard
	// =========================================================================
	r.mdExecutiveDashboard(&sb, driverList, byDriver, byOperation)

	// =========================================================================
	// 3. Performance Matrix
	// =========================================================================
	r.mdPerformanceMatrix(&sb, driverList, byOperation)

	// =========================================================================
	// 4. Write Performance Deep Dive
	// =========================================================================
	r.mdWriteDeepDive(&sb, driverList, byOperation)

	// =========================================================================
	// 5. Read Performance Deep Dive
	// =========================================================================
	r.mdReadDeepDive(&sb, driverList, byOperation)

	// =========================================================================
	// 6. Parallel Scalability
	// =========================================================================
	r.mdParallelScalability(&sb, driverList)

	// =========================================================================
	// 7. Latency Analysis
	// =========================================================================
	r.mdLatencyAnalysis(&sb, driverList, byOperation)

	// =========================================================================
	// 8. Resource Efficiency
	// =========================================================================
	r.mdResourceEfficiency(&sb, driverList, byDriver)

	// =========================================================================
	// 9. Profiling (if available)
	// =========================================================================
	if len(r.ProfileAnalyses) > 0 {
		sb.WriteString("## Profiling\n\n")
		// Sort driver names for deterministic output
		paDrivers := make([]string, 0, len(r.ProfileAnalyses))
		for d := range r.ProfileAnalyses {
			paDrivers = append(paDrivers, d)
		}
		sort.Strings(paDrivers)
		for _, d := range paDrivers {
			pa := r.ProfileAnalyses[d]
			sb.WriteString(pa.FormatMarkdown())
		}
	}

	// =========================================================================
	// 10. Error & Timeout Summary
	// =========================================================================
	r.mdErrorSummary(&sb, driverList, byDriver)

	// =========================================================================
	// 11. Recommendations
	// =========================================================================
	r.mdRecommendations(&sb, driverList, byDriver, byOperation)

	sb.WriteString("\n---\n\n")
	sb.WriteString("*Generated by storage benchmark CLI*\n")

	return sb.String()
}

// =============================================================================
// INDEX HELPERS
// =============================================================================

// indexResults builds sorted driver list and lookup maps, excluding reference driver.
func (r *Report) indexResults() ([]string, map[string][]*Metrics, map[string][]*Metrics) {
	byDriver := make(map[string][]*Metrics)
	byOperation := make(map[string][]*Metrics)
	drivers := make(map[string]bool)

	for _, m := range r.Results {
		if isReferenceDriver(m.Driver) {
			continue
		}
		byDriver[m.Driver] = append(byDriver[m.Driver], m)
		byOperation[m.Operation] = append(byOperation[m.Operation], m)
		drivers[m.Driver] = true
	}

	driverList := make([]string, 0, len(drivers))
	for d := range drivers {
		driverList = append(driverList, d)
	}
	sort.Strings(driverList)

	return driverList, byDriver, byOperation
}

// findBestForOperationExcludeRef finds best driver excluding reference driver.
func (r *Report) findBestForOperationExcludeRef(prefix string) string {
	var best string
	var bestThroughput float64

	for _, m := range r.Results {
		if isReferenceDriver(m.Driver) {
			continue
		}
		if strings.HasPrefix(m.Operation, prefix) && m.Throughput > bestThroughput {
			bestThroughput = m.Throughput
			best = m.Driver
		}
	}

	return best
}

// =============================================================================
// COMPACT NUMBER FORMATTING
// =============================================================================

// fmtOps formats ops/sec compactly: 1234567 -> "1.2M/s", 45678 -> "45.7K/s".
func fmtOps(v float64) string {
	if v >= 1e6 {
		return fmt.Sprintf("%.1fM/s", v/1e6)
	}
	if v >= 1e3 {
		return fmt.Sprintf("%.1fK/s", v/1e3)
	}
	return fmt.Sprintf("%.0f/s", v)
}

// fmtMBs formats MB/s compactly.
func fmtMBs(v float64) string {
	if v >= 1000 {
		return fmt.Sprintf("%.1f GB/s", v/1000)
	}
	if v >= 1 {
		return fmt.Sprintf("%.1f MB/s", v)
	}
	return fmt.Sprintf("%.2f MB/s", v)
}

// fmtCompact formats throughput for dense tables: ops/sec or MB/s depending on ObjectSize.
func fmtCompact(m *Metrics) string {
	if m == nil {
		return "-"
	}
	if m.ObjectSize > 0 {
		return fmtMBs(m.Throughput)
	}
	return fmtOps(m.Throughput)
}

// fmtMB formats megabytes.
func fmtMB(v float64) string {
	if v == 0 {
		return "-"
	}
	if v >= 1024 {
		return fmt.Sprintf("%.1f GB", v/1024)
	}
	return fmt.Sprintf("%.1f MB", v)
}

// starRating returns a text star rating (1-5) based on value relative to best.
func starRating(value, best float64) string {
	if best == 0 || value == 0 {
		return "     "
	}
	ratio := value / best
	switch {
	case ratio >= 0.9:
		return "*****"
	case ratio >= 0.7:
		return "**** "
	case ratio >= 0.5:
		return "***  "
	case ratio >= 0.3:
		return "**   "
	default:
		return "*    "
	}
}

// asciiBar returns an ASCII bar chart string proportional to max, with maxLen chars.
func asciiBar(value, maxVal float64, maxLen int) string {
	if maxVal == 0 || value == 0 {
		return ""
	}
	barLen := int(value / maxVal * float64(maxLen))
	if barLen < 1 && value > 0 {
		barLen = 1
	}
	if barLen > maxLen {
		barLen = maxLen
	}
	filled := strings.Repeat("#", barLen)
	empty := strings.Repeat(".", maxLen-barLen)
	return filled + empty
}

// pctOf formats value as percentage of best. Returns "100%" if equal.
func pctOf(value, best float64) string {
	if best == 0 {
		return "-"
	}
	pct := (value / best) * 100
	if pct >= 99.5 {
		return "100%"
	}
	return fmt.Sprintf("%.0f%%", pct)
}

// =============================================================================
// LOOKUP HELPERS
// =============================================================================

// lookupMetric finds a *Metrics for a specific driver+operation.
func lookupMetric(byOperation map[string][]*Metrics, operation, driver string) *Metrics {
	for _, m := range byOperation[operation] {
		if m.Driver == driver {
			return m
		}
	}
	return nil
}

// bestThroughputForOp returns the highest throughput in an operation.
func bestThroughputForOp(byOperation map[string][]*Metrics, operation string) float64 {
	var best float64
	for _, m := range byOperation[operation] {
		if !isReferenceDriver(m.Driver) && m.Throughput > best {
			best = m.Throughput
		}
	}
	return best
}

// bestDriverForOp returns the driver name with highest throughput for an operation.
func bestDriverForOp(byOperation map[string][]*Metrics, operation string) string {
	var best string
	var bestVal float64
	for _, m := range byOperation[operation] {
		if !isReferenceDriver(m.Driver) && m.Throughput > bestVal {
			bestVal = m.Throughput
			best = m.Driver
		}
	}
	return best
}

// collectSizesForPrefix finds all distinct object sizes for operations matching prefix (e.g. "Write/").
func collectSizesForPrefix(byOperation map[string][]*Metrics, prefix string) []int {
	sizeSet := make(map[int]bool)
	for op, metrics := range byOperation {
		if strings.HasPrefix(op, prefix) && !strings.Contains(op, "Parallel") && !strings.Contains(op, "Scale") {
			for _, m := range metrics {
				if m.ObjectSize > 0 {
					sizeSet[m.ObjectSize] = true
				}
			}
		}
	}
	sizes := make([]int, 0, len(sizeSet))
	for s := range sizeSet {
		sizes = append(sizes, s)
	}
	sort.Ints(sizes)
	return sizes
}

// totalOpsForDriver returns total successful iterations across all operations for a driver.
func totalOpsForDriver(byDriver map[string][]*Metrics, driver string) int {
	var total int
	for _, m := range byDriver[driver] {
		total += m.Iterations
	}
	return total
}

// totalErrorsForDriver returns total errors across all operations for a driver.
func totalErrorsForDriver(byDriver map[string][]*Metrics, driver string) int {
	var total int
	for _, m := range byDriver[driver] {
		total += m.Errors
	}
	return total
}

// driverWinsCount counts how many operations a driver has the best throughput.
func driverWinsCount(byOperation map[string][]*Metrics, driver string) int {
	wins := 0
	for _, metrics := range byOperation {
		filtered := filterNonReferenceResults(metrics)
		if len(filtered) < 2 {
			continue
		}
		var bestDriver string
		var bestVal float64
		for _, m := range filtered {
			if m.Throughput > bestVal {
				bestVal = m.Throughput
				bestDriver = m.Driver
			}
		}
		if bestDriver == driver {
			wins++
		}
	}
	return wins
}

// countCompetitiveOps counts operations where a driver has >2 competitors.
func countCompetitiveOps(byOperation map[string][]*Metrics) int {
	count := 0
	for _, metrics := range byOperation {
		filtered := filterNonReferenceResults(metrics)
		if len(filtered) >= 2 {
			count++
		}
	}
	return count
}

// =============================================================================
// SECTION 2: EXECUTIVE SUMMARY DASHBOARD
// =============================================================================

func (r *Report) mdExecutiveDashboard(sb *strings.Builder, driverList []string, byDriver, byOperation map[string][]*Metrics) {
	sb.WriteString("## Executive Summary Dashboard\n\n")

	totalOps := countCompetitiveOps(byOperation)

	// Compute per-driver scores
	type driverScore struct {
		name       string
		writeScore float64 // avg throughput pct for write ops
		readScore  float64 // avg throughput pct for read ops
		wins       int
		totalOps   int
		errors     int
		peakRSS    float64
		diskMB     float64
		status     string
	}

	// Find best throughput per write and read operation for normalization
	writeOps := make(map[string]float64)
	readOps := make(map[string]float64)
	for op, metrics := range byOperation {
		filtered := filterNonReferenceResults(metrics)
		if len(filtered) < 2 {
			continue
		}
		var best float64
		for _, m := range filtered {
			if m.Throughput > best {
				best = m.Throughput
			}
		}
		if strings.HasPrefix(op, "Write") || strings.HasPrefix(op, "ParallelWrite") {
			writeOps[op] = best
		}
		if strings.HasPrefix(op, "Read") || strings.HasPrefix(op, "ParallelRead") {
			readOps[op] = best
		}
	}

	scores := make([]driverScore, 0, len(driverList))
	for _, d := range driverList {
		ds := driverScore{
			name:     d,
			wins:     driverWinsCount(byOperation, d),
			totalOps: totalOpsForDriver(byDriver, d),
			errors:   totalErrorsForDriver(byDriver, d),
		}

		// Compute write score (average percentage of best across write ops)
		var wSum float64
		var wCount int
		for op, best := range writeOps {
			m := lookupMetric(byOperation, op, d)
			if m != nil && best > 0 {
				wSum += m.Throughput / best
				wCount++
			}
		}
		if wCount > 0 {
			ds.writeScore = wSum / float64(wCount)
		}

		// Compute read score
		var rSum float64
		var rCount int
		for op, best := range readOps {
			m := lookupMetric(byOperation, op, d)
			if m != nil && best > 0 {
				rSum += m.Throughput / best
				rCount++
			}
		}
		if rCount > 0 {
			ds.readScore = rSum / float64(rCount)
		}

		// Resource info
		if rs, ok := r.ResourceSnapshots[d]; ok {
			ds.peakRSS = rs.PeakRSSMB
			ds.diskMB = rs.FinalDiskMB
		}
		if dkStats, ok := r.DockerStats[d]; ok {
			if ds.peakRSS == 0 {
				ds.peakRSS = dkStats.MemoryUsageMB
			}
		}

		// Status
		skipped := 0
		for _, sk := range r.SkippedBenchmarks {
			if sk.Driver == d {
				skipped++
			}
		}
		if ds.errors == 0 && skipped == 0 {
			ds.status = "OK"
		} else {
			parts := make([]string, 0, 2)
			if ds.errors > 0 {
				parts = append(parts, fmt.Sprintf("%d err", ds.errors))
			}
			if skipped > 0 {
				parts = append(parts, fmt.Sprintf("%d skip", skipped))
			}
			ds.status = strings.Join(parts, ", ")
		}

		scores = append(scores, ds)
	}

	// Sort by overall score (combined write+read)
	sort.Slice(scores, func(i, j int) bool {
		oi := scores[i].writeScore + scores[i].readScore
		oj := scores[j].writeScore + scores[j].readScore
		return oi > oj
	})

	sb.WriteString("| # | Driver | Write | Read | Wins | Memory | Disk | Status |\n")
	sb.WriteString("|---|--------|-------|------|------|--------|------|--------|\n")

	bestWrite := 0.0
	bestRead := 0.0
	for _, ds := range scores {
		if ds.writeScore > bestWrite {
			bestWrite = ds.writeScore
		}
		if ds.readScore > bestRead {
			bestRead = ds.readScore
		}
	}

	for i, ds := range scores {
		sb.WriteString(fmt.Sprintf("| %d | **%s** | %s | %s | %d/%d | %s | %s | %s |\n",
			i+1,
			ds.name,
			starRating(ds.writeScore, bestWrite),
			starRating(ds.readScore, bestRead),
			ds.wins, totalOps,
			fmtMB(ds.peakRSS),
			fmtMB(ds.diskMB),
			ds.status,
		))
	}
	sb.WriteString("\n")

	// Quick insight
	if len(scores) > 0 {
		winner := scores[0]
		sb.WriteString(fmt.Sprintf("> **Overall Leader: %s** -- won %d/%d benchmarks with combined write+read score of %.0f%%.\n\n",
			winner.name, winner.wins, totalOps, (winner.writeScore+winner.readScore)/2*100))
	}
}

// =============================================================================
// SECTION 3: PERFORMANCE MATRIX
// =============================================================================

func (r *Report) mdPerformanceMatrix(sb *strings.Builder, driverList []string, byOperation map[string][]*Metrics) {
	sb.WriteString("## Performance Matrix\n\n")
	sb.WriteString("All drivers x key operations. Values show throughput (ops/s or MB/s). **Bold** = best in column.\n\n")

	// Select key operations to display in the matrix
	type colDef struct {
		operation string
		label     string
	}

	// Discover available operations dynamically
	writeSizes := collectSizesForPrefix(byOperation, "Write/")
	readSizes := collectSizesForPrefix(byOperation, "Read/")

	var cols []colDef

	// Write columns
	for _, sz := range writeSizes {
		label := "W/" + SizeLabel(sz)
		op := "Write/" + SizeLabel(sz)
		if _, ok := byOperation[op]; ok {
			cols = append(cols, colDef{op, label})
		}
	}

	// Read columns
	for _, sz := range readSizes {
		label := "R/" + SizeLabel(sz)
		op := "Read/" + SizeLabel(sz)
		if _, ok := byOperation[op]; ok {
			cols = append(cols, colDef{op, label})
		}
	}

	// Metadata operations
	for _, op := range []string{"Stat", "Delete", "Copy/1KB"} {
		if _, ok := byOperation[op]; ok {
			cols = append(cols, colDef{op, op})
		}
	}

	// List operations (find the largest)
	var listOps []string
	for op := range byOperation {
		if strings.HasPrefix(op, "List/") {
			listOps = append(listOps, op)
		}
	}
	sort.Strings(listOps)
	if len(listOps) > 0 {
		last := listOps[len(listOps)-1]
		cols = append(cols, colDef{last, last})
	}

	if len(cols) == 0 {
		sb.WriteString("*No comparable operations found.*\n\n")
		return
	}

	// Find best per column
	bestPerCol := make(map[string]string) // operation -> driver name
	for _, col := range cols {
		bestPerCol[col.operation] = bestDriverForOp(byOperation, col.operation)
	}

	// Header
	sb.WriteString("| Driver |")
	for _, col := range cols {
		sb.WriteString(fmt.Sprintf(" %s |", col.label))
	}
	sb.WriteString("\n|--------|")
	for range cols {
		sb.WriteString("--------|")
	}
	sb.WriteString("\n")

	// Rows
	for _, d := range driverList {
		sb.WriteString(fmt.Sprintf("| %s |", d))
		for _, col := range cols {
			m := lookupMetric(byOperation, col.operation, d)
			val := fmtCompact(m)
			if bestPerCol[col.operation] == d && m != nil {
				val = "**" + val + "**"
			}
			sb.WriteString(fmt.Sprintf(" %s |", val))
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
}

// =============================================================================
// SECTION 4: WRITE PERFORMANCE DEEP DIVE
// =============================================================================

func (r *Report) mdWriteDeepDive(sb *strings.Builder, driverList []string, byOperation map[string][]*Metrics) {
	sb.WriteString("## Write Performance Deep Dive\n\n")

	writeSizes := collectSizesForPrefix(byOperation, "Write/")
	if len(writeSizes) == 0 {
		sb.WriteString("*No write benchmarks found.*\n\n")
		return
	}

	// Table: all drivers x write sizes
	sb.WriteString("### Write Throughput & Latency\n\n")
	sb.WriteString("| Driver |")
	for _, sz := range writeSizes {
		label := SizeLabel(sz)
		sb.WriteString(fmt.Sprintf(" %s ops/s | %s MB/s | %s P50 | %s P99 |", label, label, label, label))
	}
	sb.WriteString("\n|--------|")
	for range writeSizes {
		sb.WriteString("--------|--------|--------|--------|")
	}
	sb.WriteString("\n")

	for _, d := range driverList {
		sb.WriteString(fmt.Sprintf("| %s |", d))
		for _, sz := range writeSizes {
			op := "Write/" + SizeLabel(sz)
			m := lookupMetric(byOperation, op, d)
			if m == nil {
				sb.WriteString(" - | - | - | - |")
			} else {
				sb.WriteString(fmt.Sprintf(" %s | %s | %s | %s |",
					fmtOps(m.OpsPerSec),
					fmtMBs(m.Throughput),
					formatLatency(m.P50Latency),
					formatLatency(m.P99Latency),
				))
			}
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	// Bar chart for each write size
	for _, sz := range writeSizes {
		op := "Write/" + SizeLabel(sz)
		metrics := filterNonReferenceResults(byOperation[op])
		if len(metrics) < 2 {
			continue
		}
		sort.Slice(metrics, func(i, j int) bool {
			return metrics[i].Throughput > metrics[j].Throughput
		})

		best := metrics[0].Throughput
		maxNameLen := 0
		for _, m := range metrics {
			if len(m.Driver) > maxNameLen {
				maxNameLen = len(m.Driver)
			}
		}

		sb.WriteString(fmt.Sprintf("**Write/%s Throughput:**\n```\n", SizeLabel(sz)))
		for _, m := range metrics {
			bar := asciiBar(m.Throughput, best, 40)
			sb.WriteString(fmt.Sprintf("  %-*s |%s %s (%s)\n",
				maxNameLen, m.Driver, bar, fmtCompact(m), pctOf(m.Throughput, best)))
		}
		sb.WriteString("```\n\n")
	}
}

// =============================================================================
// SECTION 5: READ PERFORMANCE DEEP DIVE
// =============================================================================

func (r *Report) mdReadDeepDive(sb *strings.Builder, driverList []string, byOperation map[string][]*Metrics) {
	sb.WriteString("## Read Performance Deep Dive\n\n")

	readSizes := collectSizesForPrefix(byOperation, "Read/")
	if len(readSizes) == 0 {
		sb.WriteString("*No read benchmarks found.*\n\n")
		return
	}

	// Table: all drivers x read sizes
	sb.WriteString("### Read Throughput & Latency\n\n")
	sb.WriteString("| Driver |")
	for _, sz := range readSizes {
		label := SizeLabel(sz)
		sb.WriteString(fmt.Sprintf(" %s ops/s | %s MB/s | %s P50 | %s P99 |", label, label, label, label))
	}
	sb.WriteString("\n|--------|")
	for range readSizes {
		sb.WriteString("--------|--------|--------|--------|")
	}
	sb.WriteString("\n")

	for _, d := range driverList {
		sb.WriteString(fmt.Sprintf("| %s |", d))
		for _, sz := range readSizes {
			op := "Read/" + SizeLabel(sz)
			m := lookupMetric(byOperation, op, d)
			if m == nil {
				sb.WriteString(" - | - | - | - |")
			} else {
				sb.WriteString(fmt.Sprintf(" %s | %s | %s | %s |",
					fmtOps(m.OpsPerSec),
					fmtMBs(m.Throughput),
					formatLatency(m.P50Latency),
					formatLatency(m.P99Latency),
				))
			}
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	// TTFB section if available
	hasTTFB := false
	for _, sz := range readSizes {
		op := "Read/" + SizeLabel(sz)
		for _, m := range byOperation[op] {
			if m.TTFBAvg > 0 {
				hasTTFB = true
				break
			}
		}
		if hasTTFB {
			break
		}
	}

	if hasTTFB {
		sb.WriteString("### Time To First Byte (TTFB)\n\n")
		sb.WriteString("| Driver |")
		for _, sz := range readSizes {
			label := SizeLabel(sz)
			sb.WriteString(fmt.Sprintf(" %s Avg | %s P95 |", label, label))
		}
		sb.WriteString("\n|--------|")
		for range readSizes {
			sb.WriteString("--------|--------|")
		}
		sb.WriteString("\n")

		for _, d := range driverList {
			sb.WriteString(fmt.Sprintf("| %s |", d))
			for _, sz := range readSizes {
				op := "Read/" + SizeLabel(sz)
				m := lookupMetric(byOperation, op, d)
				if m == nil || m.TTFBAvg == 0 {
					sb.WriteString(" - | - |")
				} else {
					sb.WriteString(fmt.Sprintf(" %s | %s |",
						formatLatency(m.TTFBAvg),
						formatLatency(m.TTFBP95),
					))
				}
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	// Bar chart for each read size
	for _, sz := range readSizes {
		op := "Read/" + SizeLabel(sz)
		metrics := filterNonReferenceResults(byOperation[op])
		if len(metrics) < 2 {
			continue
		}
		sort.Slice(metrics, func(i, j int) bool {
			return metrics[i].Throughput > metrics[j].Throughput
		})

		best := metrics[0].Throughput
		maxNameLen := 0
		for _, m := range metrics {
			if len(m.Driver) > maxNameLen {
				maxNameLen = len(m.Driver)
			}
		}

		sb.WriteString(fmt.Sprintf("**Read/%s Throughput:**\n```\n", SizeLabel(sz)))
		for _, m := range metrics {
			bar := asciiBar(m.Throughput, best, 40)
			sb.WriteString(fmt.Sprintf("  %-*s |%s %s (%s)\n",
				maxNameLen, m.Driver, bar, fmtCompact(m), pctOf(m.Throughput, best)))
		}
		sb.WriteString("```\n\n")
	}
}

// =============================================================================
// SECTION 6: PARALLEL SCALABILITY
// =============================================================================

func (r *Report) mdParallelScalability(sb *strings.Builder, driverList []string) {
	sb.WriteString("## Parallel Scalability\n\n")

	// Collect parallel results by driver and concurrency
	type concEntry struct {
		concurrency int
		throughput  float64
		p99         time.Duration
		errors      int
	}

	writeByDriver := make(map[string][]concEntry)
	readByDriver := make(map[string][]concEntry)
	allConcLevels := make(map[int]bool)

	extractConc := func(op string) int {
		if idx := strings.Index(op, "/C"); idx > 0 {
			var c int
			fmt.Sscanf(op[idx+2:], "%d", &c)
			return c
		}
		return 0
	}

	for _, m := range r.Results {
		if isReferenceDriver(m.Driver) {
			continue
		}
		if strings.HasPrefix(m.Operation, "ParallelWrite/") {
			conc := extractConc(m.Operation)
			if conc > 0 {
				writeByDriver[m.Driver] = append(writeByDriver[m.Driver], concEntry{conc, m.Throughput, m.P99Latency, m.Errors})
				allConcLevels[conc] = true
			}
		}
		if strings.HasPrefix(m.Operation, "ParallelRead/") {
			conc := extractConc(m.Operation)
			if conc > 0 {
				readByDriver[m.Driver] = append(readByDriver[m.Driver], concEntry{conc, m.Throughput, m.P99Latency, m.Errors})
				allConcLevels[conc] = true
			}
		}
	}

	if len(allConcLevels) == 0 {
		sb.WriteString("*No parallel benchmarks found.*\n\n")
		return
	}

	levels := make([]int, 0, len(allConcLevels))
	for l := range allConcLevels {
		levels = append(levels, l)
	}
	sort.Ints(levels)

	// Find C1 throughput for scaling efficiency computation
	minConc := levels[0]
	maxConc := levels[len(levels)-1]

	// Write scalability table
	if len(writeByDriver) > 0 {
		sb.WriteString("### Parallel Write Scalability\n\n")
		sb.WriteString("| Driver |")
		for _, l := range levels {
			sb.WriteString(fmt.Sprintf(" C%d |", l))
		}
		sb.WriteString(" Scaling |\n|--------|")
		for range levels {
			sb.WriteString("--------|")
		}
		sb.WriteString("---------|\n")

		for _, d := range driverList {
			entries := writeByDriver[d]
			if len(entries) == 0 {
				continue
			}
			entryMap := make(map[int]concEntry)
			for _, e := range entries {
				entryMap[e.concurrency] = e
			}

			sb.WriteString(fmt.Sprintf("| %s |", d))
			for _, l := range levels {
				if e, ok := entryMap[l]; ok {
					errMark := ""
					if e.errors > 0 {
						errMark = "*"
					}
					sb.WriteString(fmt.Sprintf(" %s%s |", fmtMBs(e.throughput), errMark))
				} else {
					sb.WriteString(" - |")
				}
			}

			// Scaling efficiency: (throughput_maxC / throughput_minC) / (maxC / minC)
			baseEntry, hasBase := entryMap[minConc]
			topEntry, hasTop := entryMap[maxConc]
			if hasBase && hasTop && baseEntry.throughput > 0 && minConc > 0 {
				scalingFactor := (topEntry.throughput / baseEntry.throughput) / (float64(maxConc) / float64(minConc))
				sb.WriteString(fmt.Sprintf(" %.0f%% |", scalingFactor*100))
			} else {
				sb.WriteString(" - |")
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("> Scaling = (throughput at C%d / throughput at C%d) / (%d/%d). 100%% = perfect linear scaling.\n\n", maxConc, minConc, maxConc, minConc))
	}

	// Read scalability table
	if len(readByDriver) > 0 {
		sb.WriteString("### Parallel Read Scalability\n\n")
		sb.WriteString("| Driver |")
		for _, l := range levels {
			sb.WriteString(fmt.Sprintf(" C%d |", l))
		}
		sb.WriteString(" Scaling |\n|--------|")
		for range levels {
			sb.WriteString("--------|")
		}
		sb.WriteString("---------|\n")

		for _, d := range driverList {
			entries := readByDriver[d]
			if len(entries) == 0 {
				continue
			}
			entryMap := make(map[int]concEntry)
			for _, e := range entries {
				entryMap[e.concurrency] = e
			}

			sb.WriteString(fmt.Sprintf("| %s |", d))
			for _, l := range levels {
				if e, ok := entryMap[l]; ok {
					errMark := ""
					if e.errors > 0 {
						errMark = "*"
					}
					sb.WriteString(fmt.Sprintf(" %s%s |", fmtMBs(e.throughput), errMark))
				} else {
					sb.WriteString(" - |")
				}
			}

			baseEntry, hasBase := entryMap[minConc]
			topEntry, hasTop := entryMap[maxConc]
			if hasBase && hasTop && baseEntry.throughput > 0 && minConc > 0 {
				scalingFactor := (topEntry.throughput / baseEntry.throughput) / (float64(maxConc) / float64(minConc))
				sb.WriteString(fmt.Sprintf(" %.0f%% |", scalingFactor*100))
			} else {
				sb.WriteString(" - |")
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}
}

// =============================================================================
// SECTION 7: LATENCY ANALYSIS
// =============================================================================

func (r *Report) mdLatencyAnalysis(sb *strings.Builder, driverList []string, byOperation map[string][]*Metrics) {
	sb.WriteString("## Latency Analysis\n\n")

	// Select key operations for latency analysis
	var keyOps []string
	for op := range byOperation {
		filtered := filterNonReferenceResults(byOperation[op])
		if len(filtered) >= 2 {
			keyOps = append(keyOps, op)
		}
	}
	sort.Strings(keyOps)

	if len(keyOps) == 0 {
		sb.WriteString("*No latency data available.*\n\n")
		return
	}

	sb.WriteString("### Latency Distribution by Operation\n\n")
	sb.WriteString("| Driver | Operation | Min | P50 | P95 | P99 | Max | Tail Ratio |\n")
	sb.WriteString("|--------|-----------|-----|-----|-----|-----|-----|------------|\n")

	// Collect tail latency warnings
	type tailIssue struct {
		driver    string
		operation string
		ratio     float64
	}
	var tailIssues []tailIssue

	for _, op := range keyOps {
		metrics := filterNonReferenceResults(byOperation[op])
		sort.Slice(metrics, func(i, j int) bool {
			return metrics[i].Throughput > metrics[j].Throughput
		})

		for _, m := range metrics {
			// Tail ratio = P99 / P50 (high means tail latency problem)
			var tailRatio float64
			var tailStr string
			if m.P50Latency > 0 {
				tailRatio = float64(m.P99Latency) / float64(m.P50Latency)
				tailStr = fmt.Sprintf("%.1fx", tailRatio)
				if tailRatio > 10 {
					tailStr += " (!)"
					tailIssues = append(tailIssues, tailIssue{m.Driver, op, tailRatio})
				}
			} else {
				tailStr = "-"
			}

			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %s | %s | %s |\n",
				m.Driver, op,
				formatLatency(m.MinLatency),
				formatLatency(m.P50Latency),
				formatLatency(m.P95Latency),
				formatLatency(m.P99Latency),
				formatLatency(m.MaxLatency),
				tailStr,
			))
		}
	}
	sb.WriteString("\n")

	// Tail latency warnings
	if len(tailIssues) > 0 {
		sb.WriteString("### Tail Latency Warnings\n\n")
		sb.WriteString("> Drivers with P99/P50 ratio > 10x indicate significant tail latency.\n\n")
		for _, t := range tailIssues {
			sb.WriteString(fmt.Sprintf("- **%s** on %s: P99 is %.0fx the P50 latency\n", t.driver, t.operation, t.ratio))
		}
		sb.WriteString("\n")
	}
}

// =============================================================================
// SECTION 8: RESOURCE EFFICIENCY
// =============================================================================

func (r *Report) mdResourceEfficiency(sb *strings.Builder, driverList []string, byDriver map[string][]*Metrics) {
	sb.WriteString("## Resource Efficiency\n\n")

	hasResources := len(r.ResourceSnapshots) > 0
	hasDocker := len(r.DockerStats) > 0

	if !hasResources && !hasDocker {
		sb.WriteString("*No resource data collected. Enable --resource-tracking or --docker-stats.*\n\n")
		return
	}

	// Combined resource table
	if hasResources {
		sb.WriteString("### Runtime Resources\n\n")
		sb.WriteString("| Driver | Peak RSS | Go Heap | Disk Used | GC Cycles | Total Ops | Efficiency |\n")
		sb.WriteString("|--------|----------|---------|-----------|-----------|-----------|------------|\n")

		for _, d := range driverList {
			rs, ok := r.ResourceSnapshots[d]
			if !ok {
				continue
			}

			totalOps := totalOpsForDriver(byDriver, d)

			// Efficiency = total ops / peak memory (ops per MB)
			var efficiency string
			if rs.PeakRSSMB > 0 {
				eff := float64(totalOps) / rs.PeakRSSMB
				if eff >= 1e6 {
					efficiency = fmt.Sprintf("%.1fM ops/MB", eff/1e6)
				} else if eff >= 1e3 {
					efficiency = fmt.Sprintf("%.1fK ops/MB", eff/1e3)
				} else {
					efficiency = fmt.Sprintf("%.0f ops/MB", eff)
				}
			} else {
				efficiency = "-"
			}

			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %d | %s | %s |\n",
				d,
				fmtMB(rs.PeakRSSMB),
				fmtMB(rs.PeakHeapMB),
				fmtMB(rs.FinalDiskMB),
				rs.NumGC,
				fmtOps(float64(totalOps)),
				efficiency,
			))
		}
		sb.WriteString("\n")
		sb.WriteString("> Peak RSS = process-level resident memory. Go Heap = Go runtime heap. Efficiency = total iterations / peak RSS.\n\n")
	}

	// Docker resource table
	if hasDocker {
		sb.WriteString("### Docker Container Resources\n\n")
		sb.WriteString("| Driver | Memory | RSS | Cache | CPU | Volume | Block I/O |\n")
		sb.WriteString("|--------|--------|-----|-------|-----|--------|----------|\n")

		for _, d := range driverList {
			stats, ok := r.DockerStats[d]
			if !ok {
				continue
			}

			mem := stats.MemoryUsage
			if mem == "" {
				mem = "-"
			}

			rss := "-"
			if stats.MemoryRSSMB > 0 {
				rss = fmtMB(stats.MemoryRSSMB)
			}

			cache := "-"
			if stats.MemoryCacheMB > 0 {
				cache = fmtMB(stats.MemoryCacheMB)
			}

			cpu := fmt.Sprintf("%.1f%%", stats.CPUPercent)

			vol := "-"
			if stats.VolumeSize > 0 {
				vol = fmtMB(stats.VolumeSize)
			} else if stats.VolumeName != "" {
				vol = "(no data)"
			}

			blockIO := "-"
			if stats.BlockRead != "" || stats.BlockWrite != "" {
				blockIO = fmt.Sprintf("%s / %s", stats.BlockRead, stats.BlockWrite)
			}

			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %s | %s |\n",
				d, mem, rss, cache, cpu, vol, blockIO))
		}
		sb.WriteString("\n")
		sb.WriteString("> RSS = actual process memory. Cache = OS page cache (reclaimable). Block I/O = read/write.\n\n")
	}

	// Memory bar chart
	if hasResources {
		type memEntry struct {
			name    string
			peakRSS float64
		}
		var entries []memEntry
		for _, d := range driverList {
			if rs, ok := r.ResourceSnapshots[d]; ok && rs.PeakRSSMB > 0 {
				entries = append(entries, memEntry{d, rs.PeakRSSMB})
			}
		}

		if len(entries) > 1 {
			sort.Slice(entries, func(i, j int) bool {
				return entries[i].peakRSS < entries[j].peakRSS
			})
			maxRSS := entries[len(entries)-1].peakRSS
			maxNameLen := 0
			for _, e := range entries {
				if len(e.name) > maxNameLen {
					maxNameLen = len(e.name)
				}
			}

			sb.WriteString("**Memory Usage (Peak RSS, ascending):**\n```\n")
			for _, e := range entries {
				bar := asciiBar(e.peakRSS, maxRSS, 40)
				sb.WriteString(fmt.Sprintf("  %-*s |%s %s\n", maxNameLen, e.name, bar, fmtMB(e.peakRSS)))
			}
			sb.WriteString("```\n\n")
		}
	}
}

// =============================================================================
// SECTION 9: ERROR & TIMEOUT SUMMARY
// =============================================================================

func (r *Report) mdErrorSummary(sb *strings.Builder, driverList []string, byDriver map[string][]*Metrics) {
	sb.WriteString("## Error & Timeout Summary\n\n")

	// Collect errors per driver
	type driverErrors struct {
		name       string
		totalErrs  int
		operations []string // operations with errors
		lastError  string
	}

	var errorDrivers []driverErrors
	for _, d := range driverList {
		de := driverErrors{name: d}
		for _, m := range byDriver[d] {
			if m.Errors > 0 {
				de.totalErrs += m.Errors
				de.operations = append(de.operations, fmt.Sprintf("%s (%d)", m.Operation, m.Errors))
				if m.LastError != "" {
					de.lastError = m.LastError
				}
			}
		}
		if de.totalErrs > 0 {
			errorDrivers = append(errorDrivers, de)
		}
	}

	// Skipped benchmarks
	skippedByDriver := make(map[string][]SkippedBenchmark)
	for _, sk := range r.SkippedBenchmarks {
		skippedByDriver[sk.Driver] = append(skippedByDriver[sk.Driver], sk)
	}

	if len(errorDrivers) == 0 && len(skippedByDriver) == 0 {
		sb.WriteString("No errors or timeouts recorded. All benchmarks completed successfully.\n\n")
		return
	}

	// Error table
	if len(errorDrivers) > 0 {
		sb.WriteString("### Errors\n\n")
		sb.WriteString("| Driver | Total Errors | Affected Operations | Last Error |\n")
		sb.WriteString("|--------|-------------|--------------------|-----------|\n")

		sort.Slice(errorDrivers, func(i, j int) bool {
			return errorDrivers[i].totalErrs > errorDrivers[j].totalErrs
		})

		for _, de := range errorDrivers {
			lastErr := de.lastError
			if len(lastErr) > 60 {
				lastErr = lastErr[:60] + "..."
			}
			ops := strings.Join(de.operations, ", ")
			if len(ops) > 80 {
				ops = ops[:80] + "..."
			}
			sb.WriteString(fmt.Sprintf("| %s | %d | %s | %s |\n",
				de.name, de.totalErrs, ops, lastErr))
		}
		sb.WriteString("\n")
	}

	// Skipped benchmarks
	if len(skippedByDriver) > 0 {
		sb.WriteString("### Skipped Benchmarks\n\n")

		for _, d := range driverList {
			skips, ok := skippedByDriver[d]
			if !ok {
				continue
			}
			sb.WriteString(fmt.Sprintf("**%s** (%d skipped):\n", d, len(skips)))
			for _, sk := range skips {
				sb.WriteString(fmt.Sprintf("- %s: %s\n", sk.Operation, sk.Reason))
			}
			sb.WriteString("\n")
		}
	}
}

// =============================================================================
// SECTION 10: RECOMMENDATIONS
// =============================================================================

func (r *Report) mdRecommendations(sb *strings.Builder, driverList []string, byDriver, byOperation map[string][]*Metrics) {
	sb.WriteString("## Recommendations\n\n")

	// Best for writes (average throughput across all write ops)
	type avgScore struct {
		driver string
		avg    float64
	}

	computeAvg := func(prefix string) []avgScore {
		driverSum := make(map[string]float64)
		driverCnt := make(map[string]int)
		for op, metrics := range byOperation {
			if !strings.HasPrefix(op, prefix) {
				continue
			}
			for _, m := range metrics {
				if !isReferenceDriver(m.Driver) {
					driverSum[m.Driver] += m.Throughput
					driverCnt[m.Driver]++
				}
			}
		}
		var result []avgScore
		for d, sum := range driverSum {
			result = append(result, avgScore{d, sum / float64(driverCnt[d])})
		}
		sort.Slice(result, func(i, j int) bool {
			return result[i].avg > result[j].avg
		})
		return result
	}

	writeRanking := computeAvg("Write")
	readRanking := computeAvg("Read")

	// Best for memory
	var memBest string
	var memBestVal float64 = 1e18
	for _, d := range driverList {
		if rs, ok := r.ResourceSnapshots[d]; ok && rs.PeakRSSMB > 0 && rs.PeakRSSMB < memBestVal {
			memBestVal = rs.PeakRSSMB
			memBest = d
		}
	}

	// Overall winner by win count
	type winEntry struct {
		driver string
		wins   int
	}
	var winRanking []winEntry
	for _, d := range driverList {
		w := driverWinsCount(byOperation, d)
		if w > 0 {
			winRanking = append(winRanking, winEntry{d, w})
		}
	}
	sort.Slice(winRanking, func(i, j int) bool {
		return winRanking[i].wins > winRanking[j].wins
	})

	sb.WriteString("### Best for Write-Heavy Workloads\n\n")
	if len(writeRanking) > 0 {
		sb.WriteString(fmt.Sprintf("> **%s** -- highest average write throughput across all object sizes.\n\n", writeRanking[0].driver))
		sb.WriteString("| Rank | Driver | Avg Write Throughput |\n")
		sb.WriteString("|------|--------|---------------------|\n")
		limit := len(writeRanking)
		if limit > 5 {
			limit = 5
		}
		for i := 0; i < limit; i++ {
			sb.WriteString(fmt.Sprintf("| %d | %s | %s |\n", i+1, writeRanking[i].driver, fmtMBs(writeRanking[i].avg)))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("### Best for Read-Heavy Workloads\n\n")
	if len(readRanking) > 0 {
		sb.WriteString(fmt.Sprintf("> **%s** -- highest average read throughput across all object sizes.\n\n", readRanking[0].driver))
		sb.WriteString("| Rank | Driver | Avg Read Throughput |\n")
		sb.WriteString("|------|--------|--------------------|\n")
		limit := len(readRanking)
		if limit > 5 {
			limit = 5
		}
		for i := 0; i < limit; i++ {
			sb.WriteString(fmt.Sprintf("| %d | %s | %s |\n", i+1, readRanking[i].driver, fmtMBs(readRanking[i].avg)))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("### Most Memory Efficient\n\n")
	if memBest != "" {
		sb.WriteString(fmt.Sprintf("> **%s** -- lowest peak RSS at %s.\n\n", memBest, fmtMB(memBestVal)))
	} else {
		sb.WriteString("> *No memory data available.*\n\n")
	}

	sb.WriteString("### Best Overall\n\n")
	if len(winRanking) > 0 {
		totalOps := countCompetitiveOps(byOperation)
		sb.WriteString(fmt.Sprintf("> **%s** -- won %d/%d competitive benchmarks.\n\n", winRanking[0].driver, winRanking[0].wins, totalOps))

		sb.WriteString("| Rank | Driver | Wins |\n")
		sb.WriteString("|------|--------|------|\n")
		limit := len(winRanking)
		if limit > 5 {
			limit = 5
		}
		for i := 0; i < limit; i++ {
			sb.WriteString(fmt.Sprintf("| %d | %s | %d |\n", i+1, winRanking[i].driver, winRanking[i].wins))
		}
		sb.WriteString("\n")
	}
}

// writeBarCharts generates throughput and latency bar charts.
func writeBarCharts(sb *strings.Builder, results []*Metrics) {
	if len(results) == 0 {
		return
	}

	// Throughput bar chart
	sb.WriteString("**Throughput**\n```\n")
	maxThroughput := results[0].Throughput
	for _, m := range results {
		if m.Throughput > maxThroughput {
			maxThroughput = m.Throughput
		}
	}

	for _, m := range results {
		barLen := int(m.Throughput / maxThroughput * 30)
		if barLen < 1 && m.Throughput > 0 {
			barLen = 1
		}
		bar := strings.Repeat("#", barLen)
		var val string
		if m.ObjectSize > 0 {
			val = fmt.Sprintf("%.2f MB/s", m.Throughput)
		} else {
			val = fmt.Sprintf("%.0f ops/s", m.Throughput)
		}
		sb.WriteString(fmt.Sprintf("%-12s %s %s\n", m.Driver, bar, val))
	}
	sb.WriteString("```\n\n")

	// Latency bar chart (P50)
	sb.WriteString("**Latency (P50)**\n```\n")
	var maxLatency time.Duration
	for _, m := range results {
		if m.P50Latency > maxLatency {
			maxLatency = m.P50Latency
		}
	}

	if maxLatency > 0 {
		for _, m := range results {
			barLen := int(float64(m.P50Latency) / float64(maxLatency) * 30)
			if barLen < 1 && m.P50Latency > 0 {
				barLen = 1
			}
			bar := strings.Repeat("#", barLen)
			sb.WriteString(fmt.Sprintf("%-12s %s %s\n", m.Driver, bar, formatLatency(m.P50Latency)))
		}
	}
	sb.WriteString("```\n\n")
}

func formatLatency(d time.Duration) string {
	if d < time.Microsecond {
		return fmt.Sprintf("%dns", d.Nanoseconds())
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%.1fus", float64(d.Nanoseconds())/1000)
	}
	if d < time.Second {
		return fmt.Sprintf("%.1fms", float64(d.Nanoseconds())/1000000)
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

// =============================================================================
// FORMATTING HELPERS
// =============================================================================

// formatThroughput formats throughput with appropriate unit.
func formatThroughput(throughput float64, hasSize bool) string {
	if hasSize {
		if throughput >= 1000 {
			return fmt.Sprintf("%.1f GB/s", throughput/1000)
		}
		return fmt.Sprintf("%.1f MB/s", throughput)
	}
	if throughput >= 1000000 {
		return fmt.Sprintf("%.1fM ops/s", throughput/1000000)
	}
	if throughput >= 1000 {
		return fmt.Sprintf("%.1fK ops/s", throughput/1000)
	}
	return fmt.Sprintf("%.0f ops/s", throughput)
}

// generateLeaderTable creates a markdown table showing leaders per category.
// Reference driver (devnull) is excluded from comparisons.
func (r *Report) generateLeaderTable(sb *strings.Builder) {
	// Group results by operation category (excluding reference driver)
	categoryBest := make(map[string]struct {
		driver     string
		throughput float64
		second     string
		secondVal  float64
	})

	for _, m := range r.Results {
		if isReferenceDriver(m.Driver) {
			continue
		}
		cat := m.Operation
		curr := categoryBest[cat]
		if m.Throughput > curr.throughput {
			if curr.driver != "" {
				curr.second = curr.driver
				curr.secondVal = curr.throughput
			}
			curr.driver = m.Driver
			curr.throughput = m.Throughput
			categoryBest[cat] = curr
		} else if m.Throughput > curr.secondVal {
			curr.second = m.Driver
			curr.secondVal = m.Throughput
			categoryBest[cat] = curr
		}
	}

	// Find the largest file size, highest concurrency, and highest list count from actual results
	largestReadSize := 0
	largestReadLabel := "10MB"
	largestWriteSize := 0
	largestWriteLabel := "10MB"
	highestConcurrency := 10
	highestListCount := 100

	for _, m := range r.Results {
		if isReferenceDriver(m.Driver) {
			continue
		}
		if strings.HasPrefix(m.Operation, "Read/") && !strings.Contains(m.Operation, "Parallel") {
			if m.ObjectSize > largestReadSize {
				largestReadSize = m.ObjectSize
				largestReadLabel = SizeLabel(m.ObjectSize)
			}
		}
		if strings.HasPrefix(m.Operation, "Write/") && !strings.Contains(m.Operation, "Parallel") {
			if m.ObjectSize > largestWriteSize {
				largestWriteSize = m.ObjectSize
				largestWriteLabel = SizeLabel(m.ObjectSize)
			}
		}
		if strings.HasPrefix(m.Operation, "ParallelRead/") || strings.HasPrefix(m.Operation, "ParallelWrite/") {
			if idx := strings.Index(m.Operation, "/C"); idx > 0 {
				var c int
				fmt.Sscanf(m.Operation[idx+2:], "%d", &c)
				if c > highestConcurrency {
					highestConcurrency = c
				}
			}
		}
		if strings.HasPrefix(m.Operation, "List/") {
			var c int
			fmt.Sscanf(strings.TrimPrefix(m.Operation, "List/"), "%d", &c)
			if c > highestListCount {
				highestListCount = c
			}
		}
	}

	// Define key categories to highlight with dynamic values
	keyCategories := []struct {
		operation string
		display   string
	}{
		{"Read/1KB", "Small Read (1KB)"},
		{"Write/1KB", "Small Write (1KB)"},
		{fmt.Sprintf("Read/%s", largestReadLabel), fmt.Sprintf("Large Read (%s)", largestReadLabel)},
		{fmt.Sprintf("Write/%s", largestWriteLabel), fmt.Sprintf("Large Write (%s)", largestWriteLabel)},
		{"Delete", "Delete"},
		{"Stat", "Stat"},
		{fmt.Sprintf("List/%d", highestListCount), fmt.Sprintf("List (%d objects)", highestListCount)},
		{fmt.Sprintf("ParallelRead/1MB/C%d", highestConcurrency), fmt.Sprintf("High Concurrency (C%d)", highestConcurrency)},
		{"Copy/1KB", "Copy"},
	}

	type leaderInfo struct {
		category string
		leader   string
		perf     string
		margin   string
	}

	var leaders []leaderInfo
	for _, kc := range keyCategories {
		if best, ok := categoryBest[kc.operation]; ok && best.driver != "" {
			var perfStr string
			for _, m := range r.Results {
				if m.Operation == kc.operation && m.Driver == best.driver {
					if m.ObjectSize > 0 {
						perfStr = formatThroughput(best.throughput, true)
					} else {
						perfStr = formatThroughput(best.throughput, false)
					}
					break
				}
			}

			var margin string
			if best.second != "" && best.secondVal > 0 {
				factor := best.throughput / best.secondVal
				if factor >= 2.0 {
					margin = fmt.Sprintf("%.1fx vs %s", factor, best.second)
				} else if factor >= 1.1 {
					margin = fmt.Sprintf("+%.0f%% vs %s", (factor-1)*100, best.second)
				} else {
					margin = "close"
				}
			}

			leaders = append(leaders, leaderInfo{
				category: kc.display,
				leader:   best.driver,
				perf:     perfStr,
				margin:   margin,
			})
		}
	}

	if len(leaders) == 0 {
		return
	}

	sb.WriteString("### Performance Leaders\n\n")
	sb.WriteString("| Operation | Leader | Performance | Margin |\n")
	sb.WriteString("|-----------|--------|-------------|--------|\n")

	for _, l := range leaders {
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n",
			l.category, l.leader, l.perf, l.margin))
	}
	sb.WriteString("\n")
}

// generateQuickResults creates a summary showing overall winners and rankings.
// Reference driver (devnull) is excluded from rankings.
func (r *Report) generateQuickResults(sb *strings.Builder) {
	// Count wins per driver across all benchmarks (excluding reference driver)
	wins := make(map[string]int)
	totalBenchmarks := 0

	// Group by operation to find winners
	byOperation := make(map[string][]*Metrics)
	for _, m := range r.Results {
		if !isReferenceDriver(m.Driver) {
			byOperation[m.Operation] = append(byOperation[m.Operation], m)
		}
	}

	// Find winner for each operation
	for _, results := range byOperation {
		if len(results) < 2 {
			continue
		}
		totalBenchmarks++

		var bestDriver string
		var bestThroughput float64
		for _, m := range results {
			if m.Throughput > bestThroughput {
				bestThroughput = m.Throughput
				bestDriver = m.Driver
			}
		}
		if bestDriver != "" {
			wins[bestDriver]++
		}
	}

	if totalBenchmarks == 0 {
		return
	}

	// Sort drivers by wins
	type driverWins struct {
		name string
		wins int
	}
	var rankings []driverWins
	for d, w := range wins {
		rankings = append(rankings, driverWins{d, w})
	}
	sort.Slice(rankings, func(i, j int) bool {
		return rankings[i].wins > rankings[j].wins
	})

	sb.WriteString("### Summary\n\n")

	// Show overall winner
	if len(rankings) > 0 {
		winner := rankings[0]
		winPct := float64(winner.wins) / float64(totalBenchmarks) * 100
		sb.WriteString(fmt.Sprintf("**Overall Winner:** %s (won %d/%d benchmarks, %.0f%%)\n\n",
			winner.name, winner.wins, totalBenchmarks, winPct))
	}

	// Show ranking table
	sb.WriteString("| Rank | Driver | Wins | Win Rate |\n")
	sb.WriteString("|------|--------|------|----------|\n")

	for i, r := range rankings {
		winPct := float64(r.wins) / float64(totalBenchmarks) * 100
		sb.WriteString(fmt.Sprintf("| %d | %s | %d | %.0f%% |\n",
			i+1, r.name, r.wins, winPct))
	}
	sb.WriteString("\n")
}

// generateExecutiveSummary creates a quick overview section at the top of the report.
func (r *Report) generateExecutiveSummary(sb *strings.Builder) {
	sb.WriteString("## Executive Summary\n\n")

	// Summary with rankings
	r.generateQuickResults(sb)

	// Performance leaders table
	r.generateLeaderTable(sb)

	// Driver-specific summaries
	r.continueExecutiveSummary(sb)
}

// continueExecutiveSummary generates detailed performance breakdown tables.
// Reference driver (devnull) is excluded from all comparisons.
func (r *Report) continueExecutiveSummary(sb *strings.Builder) {
	// Find the largest file size tested (excluding reference driver)
	largestSize := 0
	largestSizeLabel := "1MB"
	for _, m := range r.Results {
		if isReferenceDriver(m.Driver) {
			continue
		}
		if strings.HasPrefix(m.Operation, "Write/") || strings.HasPrefix(m.Operation, "Read/") {
			if m.ObjectSize > largestSize {
				largestSize = m.ObjectSize
				largestSizeLabel = SizeLabel(m.ObjectSize)
			}
		}
	}

	// Collect driver statistics with detailed breakdown
	type driverSummary struct {
		name string
		// Large file performance
		writeLargeThroughput float64
		readLargeThroughput  float64
		writeLargeLatencyP50 time.Duration
		readLargeLatencyP50  time.Duration
		// Small file performance (1KB)
		write1KBOpsPerSec  float64
		read1KBOpsPerSec   float64
		write1KBLatencyP50 time.Duration
		read1KBLatencyP50  time.Duration
		// Parallel performance (C10)
		parallelWriteC10 float64
		parallelReadC10  float64
		// Operations
		listOpsPerSec   float64
		deleteOpsPerSec float64
		statOpsPerSec   float64
		// Errors and resource usage
		errors   int
		memoryMB float64
	}

	summaries := make(map[string]*driverSummary)
	largeWriteOp := "Write/" + largestSizeLabel
	largeReadOp := "Read/" + largestSizeLabel

	for _, m := range r.Results {
		// Skip reference driver
		if isReferenceDriver(m.Driver) {
			continue
		}
		if summaries[m.Driver] == nil {
			summaries[m.Driver] = &driverSummary{name: m.Driver}
		}
		s := summaries[m.Driver]
		s.errors += m.Errors

		// Categorize by operation type
		switch {
		case m.Operation == largeWriteOp:
			s.writeLargeThroughput = m.Throughput
			s.writeLargeLatencyP50 = m.P50Latency
		case m.Operation == largeReadOp:
			s.readLargeThroughput = m.Throughput
			s.readLargeLatencyP50 = m.P50Latency
		case m.Operation == "Write/1KB":
			s.write1KBOpsPerSec = m.OpsPerSec
			s.write1KBLatencyP50 = m.P50Latency
		case m.Operation == "Read/1KB":
			s.read1KBOpsPerSec = m.OpsPerSec
			s.read1KBLatencyP50 = m.P50Latency
		case strings.HasPrefix(m.Operation, "ParallelWrite/") && strings.HasSuffix(m.Operation, "/C10"):
			s.parallelWriteC10 = m.Throughput
		case strings.HasPrefix(m.Operation, "ParallelRead/") && strings.HasSuffix(m.Operation, "/C10"):
			s.parallelReadC10 = m.Throughput
		case m.Operation == "List/100":
			s.listOpsPerSec = m.OpsPerSec
		case m.Operation == "Delete":
			s.deleteOpsPerSec = m.OpsPerSec
		case m.Operation == "Stat":
			s.statOpsPerSec = m.OpsPerSec
		}
	}

	// Add memory info
	for name, stats := range r.DockerStats {
		if s, ok := summaries[name]; ok {
			s.memoryMB = stats.MemoryUsageMB
		}
	}

	// Sort drivers for consistent output
	var drivers []string
	for d := range summaries {
		drivers = append(drivers, d)
	}
	sort.Strings(drivers)

	// Use Case Recommendations
	sb.WriteString("### Best Driver by Use Case\n\n")
	sb.WriteString("| Use Case | Recommended | Performance | Notes |\n")
	sb.WriteString("|----------|-------------|-------------|-------|\n")

	// Find best for each use case
	var bestLargeWrite, bestLargeRead, bestSmallOps, bestConcurrent, bestLowMem string
	var bestLargeWriteVal, bestLargeReadVal, bestSmallOpsVal, bestConcurrentVal float64
	var bestLowMemVal float64 = 1e12

	for d, s := range summaries {
		if s.writeLargeThroughput > bestLargeWriteVal {
			bestLargeWriteVal = s.writeLargeThroughput
			bestLargeWrite = d
		}
		if s.readLargeThroughput > bestLargeReadVal {
			bestLargeReadVal = s.readLargeThroughput
			bestLargeRead = d
		}
		smallOps := (s.write1KBOpsPerSec + s.read1KBOpsPerSec) / 2
		if smallOps > bestSmallOpsVal {
			bestSmallOpsVal = smallOps
			bestSmallOps = d
		}
		concurrent := s.parallelReadC10 + s.parallelWriteC10
		if concurrent > bestConcurrentVal {
			bestConcurrentVal = concurrent
			bestConcurrent = d
		}
		if s.memoryMB > 0 && s.memoryMB < bestLowMemVal {
			bestLowMemVal = s.memoryMB
			bestLowMem = d
		}
	}

	if bestLargeWrite != "" {
		sb.WriteString(fmt.Sprintf("| Large File Uploads (%s+) | **%s** | %.0f MB/s | Best for media, backups |\n",
			largestSizeLabel, bestLargeWrite, bestLargeWriteVal))
	}
	if bestLargeRead != "" {
		sb.WriteString(fmt.Sprintf("| Large File Downloads (%s) | **%s** | %.0f MB/s | Best for streaming, CDN |\n",
			largestSizeLabel, bestLargeRead, bestLargeReadVal))
	}
	if bestSmallOps != "" {
		sb.WriteString(fmt.Sprintf("| Small File Operations | **%s** | %.0f ops/s | Best for metadata, configs |\n",
			bestSmallOps, bestSmallOpsVal))
	}
	if bestConcurrent != "" {
		sb.WriteString(fmt.Sprintf("| High Concurrency (C10) | **%s** | - | Best for multi-user apps |\n",
			bestConcurrent))
	}
	if bestLowMem != "" {
		sb.WriteString(fmt.Sprintf("| Memory Constrained | **%s** | %.0f MB RAM | Best for edge/embedded |\n",
			bestLowMem, bestLowMemVal))
	}
	sb.WriteString("\n")

	// Large File Performance (dynamic - largest size tested)
	sb.WriteString(fmt.Sprintf("### Large File Performance (%s)\n\n", largestSizeLabel))
	sb.WriteString("| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |\n")
	sb.WriteString("|--------|-------------|-------------|---------------|---------------|\n")

	for _, d := range drivers {
		s := summaries[d]
		sb.WriteString(fmt.Sprintf("| %s | %.1f | %.1f | %s | %s |\n",
			s.name, s.writeLargeThroughput, s.readLargeThroughput,
			formatLatency(s.writeLargeLatencyP50), formatLatency(s.readLargeLatencyP50)))
	}
	sb.WriteString("\n")

	// Small File Performance (1KB)
	sb.WriteString("### Small File Performance (1KB)\n\n")
	sb.WriteString("| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |\n")
	sb.WriteString("|--------|--------------|--------------|---------------|---------------|\n")

	for _, d := range drivers {
		s := summaries[d]
		sb.WriteString(fmt.Sprintf("| %s | %.0f | %.0f | %s | %s |\n",
			s.name, s.write1KBOpsPerSec, s.read1KBOpsPerSec,
			formatLatency(s.write1KBLatencyP50), formatLatency(s.read1KBLatencyP50)))
	}
	sb.WriteString("\n")

	// Metadata Operations
	sb.WriteString("### Metadata Operations (ops/s)\n\n")
	sb.WriteString("| Driver | Stat | List (100 objects) | Delete |\n")
	sb.WriteString("|--------|------|-------------------|--------|\n")

	for _, d := range drivers {
		s := summaries[d]
		sb.WriteString(fmt.Sprintf("| %s | %.0f | %.0f | %.0f |\n",
			s.name, s.statOpsPerSec, s.listOpsPerSec, s.deleteOpsPerSec))
	}
	sb.WriteString("\n")

	// Concurrency Performance Summary (if available)
	r.generateConcurrencySummary(sb, drivers)

	// Scale Performance Summary (if available)
	r.generateScaleSummary(sb, drivers)

	// Warnings
	hasWarnings := false
	for _, d := range drivers {
		s := summaries[d]
		if s.errors > 0 {
			if !hasWarnings {
				sb.WriteString("### Warnings\n\n")
				hasWarnings = true
			}
			sb.WriteString(fmt.Sprintf("- **%s**: %d errors during benchmarks\n", s.name, s.errors))
		}
	}
	if hasWarnings {
		sb.WriteString("\n")
	}

	// Skipped Benchmarks (show drivers with reduced coverage)
	if len(r.SkippedBenchmarks) > 0 {
		sb.WriteString("### Skipped Benchmarks\n\n")
		sb.WriteString("Some benchmarks were skipped due to driver limitations:\n\n")

		// Group by driver
		skippedByDriver := make(map[string][]string)
		for _, skip := range r.SkippedBenchmarks {
			skippedByDriver[skip.Driver] = append(skippedByDriver[skip.Driver], skip.Operation+" ("+skip.Reason+")")
		}

		for _, d := range drivers {
			if skips, ok := skippedByDriver[d]; ok {
				sb.WriteString(fmt.Sprintf("- **%s**: %d skipped\n", d, len(skips)))
				for _, s := range skips {
					sb.WriteString(fmt.Sprintf("  - %s\n", s))
				}
			}
		}
		sb.WriteString("\n")
	}

	// Resource Usage Summary
	if len(r.DockerStats) > 0 {
		sb.WriteString("### Resource Usage Summary\n\n")
		sb.WriteString("| Driver | Memory | CPU |\n")
		sb.WriteString("|--------|--------|-----|\n")

		for _, d := range drivers {
			if stats, ok := r.DockerStats[d]; ok {
				sb.WriteString(fmt.Sprintf("| %s | %.1f MB | %.1f%% |\n",
					d, stats.MemoryUsageMB, stats.CPUPercent))
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString("---\n\n")
}

// generateConcurrencySummary creates a summary of parallel benchmark results.
func (r *Report) generateConcurrencySummary(sb *strings.Builder, drivers []string) {
	// Collect parallel results by concurrency level
	type concResult struct {
		driver      string
		concurrency int
		throughput  float64
		p50         time.Duration
		p99         time.Duration
		errors      int
	}

	writeResults := make(map[string][]concResult)
	readResults := make(map[string][]concResult)

	// Extract concurrency level from operation name
	extractConc := func(op string) int {
		if idx := strings.Index(op, "/C"); idx > 0 {
			var c int
			fmt.Sscanf(op[idx+2:], "%d", &c)
			return c
		}
		return 0
	}

	for _, m := range r.Results {
		if strings.HasPrefix(m.Operation, "ParallelWrite/") {
			conc := extractConc(m.Operation)
			if conc > 0 {
				writeResults[m.Driver] = append(writeResults[m.Driver], concResult{
					driver:      m.Driver,
					concurrency: conc,
					throughput:  m.Throughput,
					p50:         m.P50Latency,
					p99:         m.P99Latency,
					errors:      m.Errors,
				})
			}
		}
		if strings.HasPrefix(m.Operation, "ParallelRead/") {
			conc := extractConc(m.Operation)
			if conc > 0 {
				readResults[m.Driver] = append(readResults[m.Driver], concResult{
					driver:      m.Driver,
					concurrency: conc,
					throughput:  m.Throughput,
					p50:         m.P50Latency,
					p99:         m.P99Latency,
					errors:      m.Errors,
				})
			}
		}
	}

	// Only show if we have results
	if len(writeResults) == 0 && len(readResults) == 0 {
		return
	}

	sb.WriteString("### Concurrency Performance\n\n")

	if len(writeResults) > 0 {
		sb.WriteString("**Parallel Write (MB/s by concurrency)**\n\n")
		sb.WriteString("| Driver |")

		// Get all concurrency levels
		concLevels := make(map[int]bool)
		for _, results := range writeResults {
			for _, r := range results {
				concLevels[r.concurrency] = true
			}
		}
		var levels []int
		for l := range concLevels {
			levels = append(levels, l)
		}
		sort.Ints(levels)

		for _, l := range levels {
			sb.WriteString(fmt.Sprintf(" C%d |", l))
		}
		sb.WriteString("\n|--------|")
		for range levels {
			sb.WriteString("------|")
		}
		sb.WriteString("\n")

		for _, d := range drivers {
			results := writeResults[d]
			resultByConc := make(map[int]concResult)
			for _, r := range results {
				resultByConc[r.concurrency] = r
			}

			sb.WriteString(fmt.Sprintf("| %s |", d))
			for _, l := range levels {
				if r, ok := resultByConc[l]; ok {
					if r.errors > 0 {
						sb.WriteString(fmt.Sprintf(" %.2f* |", r.throughput))
					} else {
						sb.WriteString(fmt.Sprintf(" %.2f |", r.throughput))
					}
				} else {
					sb.WriteString(" - |")
				}
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n*\\* indicates errors occurred*\n\n")
	}

	if len(readResults) > 0 {
		sb.WriteString("**Parallel Read (MB/s by concurrency)**\n\n")
		sb.WriteString("| Driver |")

		concLevels := make(map[int]bool)
		for _, results := range readResults {
			for _, r := range results {
				concLevels[r.concurrency] = true
			}
		}
		var levels []int
		for l := range concLevels {
			levels = append(levels, l)
		}
		sort.Ints(levels)

		for _, l := range levels {
			sb.WriteString(fmt.Sprintf(" C%d |", l))
		}
		sb.WriteString("\n|--------|")
		for range levels {
			sb.WriteString("------|")
		}
		sb.WriteString("\n")

		for _, d := range drivers {
			results := readResults[d]
			resultByConc := make(map[int]concResult)
			for _, r := range results {
				resultByConc[r.concurrency] = r
			}

			sb.WriteString(fmt.Sprintf("| %s |", d))
			for _, l := range levels {
				if r, ok := resultByConc[l]; ok {
					if r.errors > 0 {
						sb.WriteString(fmt.Sprintf(" %.2f* |", r.throughput))
					} else {
						sb.WriteString(fmt.Sprintf(" %.2f |", r.throughput))
					}
				} else {
					sb.WriteString(" - |")
				}
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n*\\* indicates errors occurred*\n\n")
	}
}

// generateScaleSummary creates a summary of scale benchmark results.
func (r *Report) generateScaleSummary(sb *strings.Builder, drivers []string) {
	// Collect scale results by operation type
	type fileCountResult struct {
		driver    string
		count     int
		duration  time.Duration
		opsPerSec float64
		errors    int
	}

	writeResults := make(map[string][]fileCountResult)
	listResults := make(map[string][]fileCountResult)
	deleteResults := make(map[string][]fileCountResult)

	// Extract object count from operation name (e.g., "Scale/Write/1000" -> 1000)
	extractCount := func(op string) int {
		parts := strings.Split(op, "/")
		if len(parts) >= 3 {
			var c int
			fmt.Sscanf(parts[2], "%d", &c)
			return c
		}
		return 0
	}

	for _, m := range r.Results {
		if strings.HasPrefix(m.Operation, "Scale/Write/") {
			count := extractCount(m.Operation)
			if count > 0 {
				writeResults[m.Driver] = append(writeResults[m.Driver], fileCountResult{
					driver:    m.Driver,
					count:     count,
					duration:  m.TotalTime,
					opsPerSec: m.OpsPerSec,
					errors:    m.Errors,
				})
			}
		}
		if strings.HasPrefix(m.Operation, "Scale/List/") {
			count := extractCount(m.Operation)
			if count > 0 {
				listResults[m.Driver] = append(listResults[m.Driver], fileCountResult{
					driver:    m.Driver,
					count:     count,
					duration:  m.TotalTime,
					opsPerSec: m.OpsPerSec,
					errors:    m.Errors,
				})
			}
		}
		if strings.HasPrefix(m.Operation, "Scale/Delete/") {
			count := extractCount(m.Operation)
			if count > 0 {
				deleteResults[m.Driver] = append(deleteResults[m.Driver], fileCountResult{
					driver:    m.Driver,
					count:     count,
					duration:  m.TotalTime,
					opsPerSec: m.OpsPerSec,
					errors:    m.Errors,
				})
			}
		}
	}

	// Only show if we have results
	if len(writeResults) == 0 && len(listResults) == 0 && len(deleteResults) == 0 {
		return
	}

	sb.WriteString("### Scale Performance\n\n")
	if r.Config != nil && r.Config.ScaleObjectSize > 0 {
		sb.WriteString(fmt.Sprintf("Performance with varying numbers of objects (%s each).\n\n", SizeLabel(r.Config.ScaleObjectSize)))
	} else {
		sb.WriteString("Performance with varying numbers of objects.\n\n")
	}

	if len(writeResults) > 0 {
		sb.WriteString("**Write N Files (total time)**\n\n")
		sb.WriteString("| Driver |")

		// Get all scale counts
		countSet := make(map[int]bool)
		for _, results := range writeResults {
			for _, r := range results {
				countSet[r.count] = true
			}
		}
		var counts []int
		for c := range countSet {
			counts = append(counts, c)
		}
		sort.Ints(counts)

		for _, c := range counts {
			sb.WriteString(fmt.Sprintf(" %d |", c))
		}
		sb.WriteString("\n|--------|")
		for range counts {
			sb.WriteString("------|")
		}
		sb.WriteString("\n")

		for _, d := range drivers {
			results := writeResults[d]
			resultByCount := make(map[int]fileCountResult)
			for _, r := range results {
				resultByCount[r.count] = r
			}

			sb.WriteString(fmt.Sprintf("| %s |", d))
			for _, c := range counts {
				if r, ok := resultByCount[c]; ok {
					if r.errors > 0 {
						sb.WriteString(fmt.Sprintf(" %s* |", formatLatency(r.duration)))
					} else {
						sb.WriteString(fmt.Sprintf(" %s |", formatLatency(r.duration)))
					}
				} else {
					sb.WriteString(" - |")
				}
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n*\\* indicates errors occurred*\n\n")
	}

	if len(listResults) > 0 {
		sb.WriteString("**List N Files (total time)**\n\n")
		sb.WriteString("| Driver |")

		countSet := make(map[int]bool)
		for _, results := range listResults {
			for _, r := range results {
				countSet[r.count] = true
			}
		}
		var counts []int
		for c := range countSet {
			counts = append(counts, c)
		}
		sort.Ints(counts)

		for _, c := range counts {
			sb.WriteString(fmt.Sprintf(" %d |", c))
		}
		sb.WriteString("\n|--------|")
		for range counts {
			sb.WriteString("------|")
		}
		sb.WriteString("\n")

		for _, d := range drivers {
			results := listResults[d]
			resultByCount := make(map[int]fileCountResult)
			for _, r := range results {
				resultByCount[r.count] = r
			}

			sb.WriteString(fmt.Sprintf("| %s |", d))
			for _, c := range counts {
				if r, ok := resultByCount[c]; ok {
					if r.errors > 0 {
						sb.WriteString(fmt.Sprintf(" %s* |", formatLatency(r.duration)))
					} else {
						sb.WriteString(fmt.Sprintf(" %s |", formatLatency(r.duration)))
					}
				} else {
					sb.WriteString(" - |")
				}
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n*\\* indicates errors occurred*\n\n")
	}
}

// CompareResult holds comparison data between baseline and current benchmark.
type CompareResult struct {
	Driver     string
	Operation  string
	ObjectSize int

	BaselineThroughput float64
	BaselineP50        time.Duration
	BaselineP99        time.Duration

	CurrentThroughput float64
	CurrentP50        time.Duration
	CurrentP99        time.Duration

	ThroughputDelta float64 // percentage change
	P50Delta        float64 // percentage change
	P99Delta        float64 // percentage change

	Regression  bool
	Improvement bool
}

// LoadBaseline loads a baseline report from a JSON file.
func LoadBaseline(path string) (*Report, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read baseline: %w", err)
	}

	var report Report
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("parse baseline: %w", err)
	}

	return &report, nil
}

// CompareReports compares current results against a baseline.
func CompareReports(baseline, current *Report) []CompareResult {
	// Index baseline results by driver+operation
	baselineMap := make(map[string]*Metrics)
	for _, m := range baseline.Results {
		key := m.Driver + "|" + m.Operation
		baselineMap[key] = m
	}

	var results []CompareResult

	for _, curr := range current.Results {
		key := curr.Driver + "|" + curr.Operation
		base, ok := baselineMap[key]
		if !ok {
			continue // No baseline comparison available
		}

		result := CompareResult{
			Driver:     curr.Driver,
			Operation:  curr.Operation,
			ObjectSize: curr.ObjectSize,

			BaselineThroughput: base.Throughput,
			BaselineP50:        base.P50Latency,
			BaselineP99:        base.P99Latency,

			CurrentThroughput: curr.Throughput,
			CurrentP50:        curr.P50Latency,
			CurrentP99:        curr.P99Latency,
		}

		// Calculate deltas as percentages
		if base.Throughput > 0 {
			result.ThroughputDelta = ((curr.Throughput - base.Throughput) / base.Throughput) * 100
		}
		if base.P50Latency > 0 {
			result.P50Delta = ((float64(curr.P50Latency) - float64(base.P50Latency)) / float64(base.P50Latency)) * 100
		}
		if base.P99Latency > 0 {
			result.P99Delta = ((float64(curr.P99Latency) - float64(base.P99Latency)) / float64(base.P99Latency)) * 100
		}

		// Determine if regression or improvement (>10% change threshold)
		if result.ThroughputDelta < -10 || result.P99Delta > 10 {
			result.Regression = true
		}
		if result.ThroughputDelta > 10 || result.P99Delta < -10 {
			result.Improvement = true
		}

		results = append(results, result)
	}

	return results
}

// GenerateDetailedBenchmarkTable creates a markdown table showing all drivers
// with winner highlighted and percentages compared to winner.
func GenerateDetailedBenchmarkTable(results []*Metrics) string {
	var sb strings.Builder

	// Group results by operation
	byOperation := make(map[string][]*Metrics)
	for _, m := range results {
		if !isReferenceDriver(m.Driver) {
			byOperation[m.Operation] = append(byOperation[m.Operation], m)
		}
	}

	// Sort operations
	operations := make([]string, 0, len(byOperation))
	for op := range byOperation {
		operations = append(operations, op)
	}
	sort.Strings(operations)

	sb.WriteString("## Detailed Benchmark Results\n\n")
	sb.WriteString("**Legend:** 🥇 = Winner, % = relative to winner (higher is better for throughput, lower is better for latency)\n\n")

	for _, op := range operations {
		metrics := byOperation[op]
		if len(metrics) < 2 {
			continue
		}

		// Sort by throughput (descending) to find winner
		sort.Slice(metrics, func(i, j int) bool {
			return metrics[i].Throughput > metrics[j].Throughput
		})

		winner := metrics[0]

		sb.WriteString(fmt.Sprintf("### %s\n\n", op))

		// Determine if this is a throughput or ops/s benchmark
		hasThroughput := winner.ObjectSize > 0

		if hasThroughput {
			sb.WriteString("| Driver | Throughput | vs Winner | Latency P50 | Latency P99 | Errors |\n")
			sb.WriteString("|--------|------------|-----------|-------------|-------------|--------|\n")
		} else {
			sb.WriteString("| Driver | ops/sec | vs Winner | Latency P50 | Latency P99 | Errors |\n")
			sb.WriteString("|--------|---------|-----------|-------------|-------------|--------|\n")
		}

		for i, m := range metrics {
			driverName := m.Driver
			isWinner := i == 0

			if isWinner {
				driverName = "🥇 **" + driverName + "**"
			}

			// Calculate percentage vs winner
			var vsWinner string
			if isWinner {
				vsWinner = "100%"
			} else if winner.Throughput > 0 {
				pct := (m.Throughput / winner.Throughput) * 100
				vsWinner = fmt.Sprintf("%.1f%%", pct)
			}

			var throughputStr string
			if hasThroughput {
				if m.Throughput >= 1000 {
					throughputStr = fmt.Sprintf("%.2f GB/s", m.Throughput/1000)
				} else {
					throughputStr = fmt.Sprintf("%.2f MB/s", m.Throughput)
				}
			} else {
				if m.Throughput >= 1000000 {
					throughputStr = fmt.Sprintf("%.2fM", m.Throughput/1000000)
				} else if m.Throughput >= 1000 {
					throughputStr = fmt.Sprintf("%.2fK", m.Throughput/1000)
				} else {
					throughputStr = fmt.Sprintf("%.0f", m.Throughput)
				}
			}

			if isWinner {
				throughputStr = "**" + throughputStr + "**"
			}

			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %d |\n",
				driverName,
				throughputStr,
				vsWinner,
				formatLatency(m.P50Latency),
				formatLatency(m.P99Latency),
				m.Errors,
			))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// GenerateComparisonReport creates a markdown comparison report.
func GenerateComparisonReport(comparisons []CompareResult) string {
	var sb strings.Builder

	sb.WriteString("## Performance Comparison vs Baseline\n\n")

	// Collect regressions and improvements
	var regressions, improvements []CompareResult
	for _, c := range comparisons {
		if c.Regression {
			regressions = append(regressions, c)
		}
		if c.Improvement {
			improvements = append(improvements, c)
		}
	}

	// Summary
	sb.WriteString("### Summary\n\n")
	sb.WriteString(fmt.Sprintf("- **Total comparisons:** %d\n", len(comparisons)))
	sb.WriteString(fmt.Sprintf("- **Regressions detected:** %d\n", len(regressions)))
	sb.WriteString(fmt.Sprintf("- **Improvements detected:** %d\n", len(improvements)))
	sb.WriteString("\n")

	// Regressions
	if len(regressions) > 0 {
		sb.WriteString("### Regressions (>10% slower)\n\n")
		sb.WriteString("| Driver | Operation | Baseline | Current | Throughput Δ | P99 Δ |\n")
		sb.WriteString("|--------|-----------|----------|---------|--------------|-------|\n")

		for _, r := range regressions {
			sb.WriteString(fmt.Sprintf("| %s | %s | %.2f MB/s | %.2f MB/s | %.1f%% | %.1f%% |\n",
				r.Driver, r.Operation,
				r.BaselineThroughput, r.CurrentThroughput,
				r.ThroughputDelta, r.P99Delta))
		}
		sb.WriteString("\n")
	}

	// Improvements
	if len(improvements) > 0 {
		sb.WriteString("### Improvements (>10% faster)\n\n")
		sb.WriteString("| Driver | Operation | Baseline | Current | Throughput Δ | P99 Δ |\n")
		sb.WriteString("|--------|-----------|----------|---------|--------------|-------|\n")

		for _, r := range improvements {
			sb.WriteString(fmt.Sprintf("| %s | %s | %.2f MB/s | %.2f MB/s | +%.1f%% | %.1f%% |\n",
				r.Driver, r.Operation,
				r.BaselineThroughput, r.CurrentThroughput,
				r.ThroughputDelta, r.P99Delta))
		}
		sb.WriteString("\n")
	}

	// Full comparison table
	sb.WriteString("### Full Comparison\n\n")
	sb.WriteString("| Driver | Operation | Baseline | Current | Throughput Δ | Status |\n")
	sb.WriteString("|--------|-----------|----------|---------|--------------|--------|\n")

	for _, c := range comparisons {
		var status string
		if c.Regression {
			status = "REGRESSION"
		} else if c.Improvement {
			status = "IMPROVED"
		} else {
			status = "STABLE"
		}

		sb.WriteString(fmt.Sprintf("| %s | %s | %.2f | %.2f | %+.1f%% | %s |\n",
			c.Driver, c.Operation,
			c.BaselineThroughput, c.CurrentThroughput,
			c.ThroughputDelta, status))
	}
	sb.WriteString("\n")

	return sb.String()
}
