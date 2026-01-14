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
		name   string
		avgNs  float64
		count  int
	}
	var ranks []driverRank
	for d, total := range driverAvg {
		ranks = append(ranks, driverRank{
			name:   d,
			avgNs:  total / float64(driverCount[d]),
			count:  driverCount[d],
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

// Report holds complete benchmark results from CLI runner.
type Report struct {
	Timestamp   time.Time               `json:"timestamp"`
	Config      *Config                 `json:"config"`
	Results     []*Metrics              `json:"results"`
	DockerStats map[string]*DockerStats `json:"docker_stats,omitempty"`
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
	mdPath := filepath.Join(outputDir, "benchmark_report.md")
	return os.WriteFile(mdPath, []byte(markdown), 0644)
}

func (r *Report) generateMarkdown() string {
	var sb strings.Builder

	// Header
	sb.WriteString("# Storage Benchmark Report\n\n")
	sb.WriteString(fmt.Sprintf("**Generated:** %s\n\n", r.Timestamp.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("**Go Version:** %s\n\n", runtime.Version()))
	sb.WriteString(fmt.Sprintf("**Platform:** %s/%s\n\n", runtime.GOOS, runtime.GOARCH))

	// Configuration
	if r.Config != nil {
		sb.WriteString("## Configuration\n\n")
		sb.WriteString("| Parameter | Value |\n")
		sb.WriteString("|-----------|-------|\n")
		sb.WriteString(fmt.Sprintf("| Iterations | %d |\n", r.Config.Iterations))
		sb.WriteString(fmt.Sprintf("| Warmup | %d |\n", r.Config.WarmupIterations))
		sb.WriteString(fmt.Sprintf("| Concurrency | %d |\n", r.Config.Concurrency))
		sb.WriteString(fmt.Sprintf("| Timeout | %v |\n", r.Config.Timeout))
		sb.WriteString("\n")
	}

	// Group results by driver
	byDriver := make(map[string][]*Metrics)
	byOperation := make(map[string][]*Metrics)
	drivers := make(map[string]bool)

	for _, m := range r.Results {
		byDriver[m.Driver] = append(byDriver[m.Driver], m)
		byOperation[m.Operation] = append(byOperation[m.Operation], m)
		drivers[m.Driver] = true
	}

	// Driver list
	driverList := make([]string, 0, len(drivers))
	for d := range drivers {
		driverList = append(driverList, d)
	}
	sort.Strings(driverList)

	sb.WriteString("## Drivers Tested\n\n")
	for _, d := range driverList {
		sb.WriteString(fmt.Sprintf("- %s (%d benchmarks)\n", d, len(byDriver[d])))
	}
	sb.WriteString("\n")

	// Operation comparison tables
	sb.WriteString("## Performance Comparison\n\n")

	// Get unique operations
	operations := make([]string, 0, len(byOperation))
	for op := range byOperation {
		operations = append(operations, op)
	}
	sort.Strings(operations)

	for _, op := range operations {
		results := byOperation[op]
		if len(results) < 2 {
			continue
		}

		sb.WriteString(fmt.Sprintf("### %s\n\n", op))
		sb.WriteString("| Driver | Throughput | P50 | P95 | P99 | Errors |\n")
		sb.WriteString("|--------|------------|-----|-----|-----|--------|\n")

		// Sort by throughput (descending)
		sort.Slice(results, func(i, j int) bool {
			return results[i].Throughput > results[j].Throughput
		})

		for _, m := range results {
			var throughput string
			if m.ObjectSize > 0 {
				throughput = fmt.Sprintf("%.2f MB/s", m.Throughput)
			} else {
				throughput = fmt.Sprintf("%.0f ops/s", m.Throughput)
			}

			sb.WriteString(fmt.Sprintf("| %s | %s | %v | %v | %v | %d |\n",
				m.Driver,
				throughput,
				formatLatency(m.P50Latency),
				formatLatency(m.P95Latency),
				formatLatency(m.P99Latency),
				m.Errors,
			))
		}
		sb.WriteString("\n")

		// Add bar chart
		r.writeBarChart(&sb, results)
	}

	// Docker stats
	if len(r.DockerStats) > 0 {
		sb.WriteString("## Resource Usage\n\n")
		sb.WriteString("| Driver | Memory | CPU | Disk |\n")
		sb.WriteString("|--------|--------|-----|------|\n")

		for _, d := range driverList {
			if stats, ok := r.DockerStats[d]; ok {
				disk := "-"
				if stats.DiskUsage > 0 {
					disk = fmt.Sprintf("%.1f MB", stats.DiskUsage)
				}
				sb.WriteString(fmt.Sprintf("| %s | %s | %.1f%% | %s |\n",
					d, stats.MemoryUsage, stats.CPUPercent, disk))
			}
		}
		sb.WriteString("\n")
	}

	// Recommendations
	sb.WriteString("## Recommendations\n\n")

	// Find best performers
	writeBest := r.findBestForOperation("Write")
	readBest := r.findBestForOperation("Read")

	if writeBest != "" {
		sb.WriteString(fmt.Sprintf("- **Best for write-heavy workloads:** %s\n", writeBest))
	}
	if readBest != "" {
		sb.WriteString(fmt.Sprintf("- **Best for read-heavy workloads:** %s\n", readBest))
	}

	sb.WriteString("\n---\n\n")
	sb.WriteString("*Report generated by storage benchmark CLI*\n")

	return sb.String()
}

func (r *Report) writeBarChart(sb *strings.Builder, results []*Metrics) {
	if len(results) == 0 {
		return
	}

	sb.WriteString("```\n")
	maxVal := results[0].Throughput // Already sorted by throughput descending
	maxWidth := 40

	for _, m := range results {
		barLen := int(m.Throughput / maxVal * float64(maxWidth))
		if barLen < 1 {
			barLen = 1
		}
		bar := strings.Repeat("█", barLen)
		var val string
		if m.ObjectSize > 0 {
			val = fmt.Sprintf("%.2f MB/s", m.Throughput)
		} else {
			val = fmt.Sprintf("%.0f ops/s", m.Throughput)
		}
		sb.WriteString(fmt.Sprintf("  %-12s %s %s\n", m.Driver, bar, val))
	}
	sb.WriteString("```\n\n")
}

func (r *Report) findBestForOperation(prefix string) string {
	var best string
	var bestThroughput float64

	for _, m := range r.Results {
		if strings.HasPrefix(m.Operation, prefix) && m.Throughput > bestThroughput {
			bestThroughput = m.Throughput
			best = m.Driver
		}
	}

	return best
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
