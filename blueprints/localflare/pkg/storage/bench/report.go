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

// SkippedBenchmark records a benchmark that was skipped.
type SkippedBenchmark struct {
	Driver    string `json:"driver"`
	Operation string `json:"operation"`
	Reason    string `json:"reason"`
}

// Report holds complete benchmark results from CLI runner.
type Report struct {
	Timestamp         time.Time               `json:"timestamp"`
	Config            *Config                 `json:"config"`
	Results           []*Metrics              `json:"results"`
	DockerStats       map[string]*DockerStats `json:"docker_stats,omitempty"`
	SkippedBenchmarks []SkippedBenchmark      `json:"skipped_benchmarks,omitempty"`
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
		case "csv":
			if err := r.SaveCSV(outputDir); err != nil {
				return fmt.Errorf("save csv: %w", err)
			}
		}
	}
	return nil
}

func (r *Report) generateMarkdown() string {
	var sb strings.Builder

	// Header
	sb.WriteString("# Storage Benchmark Report\n\n")
	sb.WriteString(fmt.Sprintf("**Generated:** %s\n\n", r.Timestamp.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("**Go Version:** %s\n\n", runtime.Version()))
	sb.WriteString(fmt.Sprintf("**Platform:** %s/%s\n\n", runtime.GOOS, runtime.GOARCH))

	// Executive Summary - generate first for quick insights
	r.generateExecutiveSummary(&sb)

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

		// Check if this is a read operation with TTFB data
		hasTTFB := strings.Contains(op, "Read") && results[0].TTFBAvg > 0

		if hasTTFB {
			sb.WriteString("| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |\n")
			sb.WriteString("|--------|------------|----------|----------|-----|-----|-----|--------|\n")
		} else {
			sb.WriteString("| Driver | Throughput | P50 | P95 | P99 | Errors |\n")
			sb.WriteString("|--------|------------|-----|-----|-----|--------|\n")
		}

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

			if hasTTFB {
				sb.WriteString(fmt.Sprintf("| %s | %s | %v | %v | %v | %v | %v | %d |\n",
					m.Driver,
					throughput,
					formatLatency(m.TTFBAvg),
					formatLatency(m.TTFBP95),
					formatLatency(m.P50Latency),
					formatLatency(m.P95Latency),
					formatLatency(m.P99Latency),
					m.Errors,
				))
			} else {
				sb.WriteString(fmt.Sprintf("| %s | %s | %v | %v | %v | %d |\n",
					m.Driver,
					throughput,
					formatLatency(m.P50Latency),
					formatLatency(m.P95Latency),
					formatLatency(m.P99Latency),
					m.Errors,
				))
			}
		}
		sb.WriteString("\n")

		// Add bar chart
		r.writeBarChart(&sb, results)
	}

	// Docker stats - enhanced with more details
	if len(r.DockerStats) > 0 {
		sb.WriteString("## Resource Usage\n\n")
		sb.WriteString("| Driver | Memory | RSS | Cache | CPU | Volume | Block I/O |\n")
		sb.WriteString("|--------|--------|-----|-------|-----|--------|----------|\n")

		for _, d := range driverList {
			if stats, ok := r.DockerStats[d]; ok {
				// Format memory
				mem := stats.MemoryUsage
				if mem == "" {
					mem = "-"
				}

				// Format RSS (actual application memory)
				rss := "-"
				if stats.MemoryRSSMB > 0 {
					rss = fmt.Sprintf("%.1f MB", stats.MemoryRSSMB)
				}

				// Format cache (page cache for disk drivers)
				cache := "-"
				if stats.MemoryCacheMB > 0 {
					cache = fmt.Sprintf("%.1f MB", stats.MemoryCacheMB)
				}

				// Format CPU
				cpu := fmt.Sprintf("%.1f%%", stats.CPUPercent)

				// Format volume size
				vol := "-"
				if stats.VolumeSize > 0 {
					vol = fmt.Sprintf("%.1f MB", stats.VolumeSize)
				} else if stats.VolumeName != "" {
					vol = "(no data)"
				}

				// Format block I/O
				blockIO := "-"
				if stats.BlockRead != "" || stats.BlockWrite != "" {
					blockIO = fmt.Sprintf("%s / %s", stats.BlockRead, stats.BlockWrite)
				}

				sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %s | %s |\n",
					d, mem, rss, cache, cpu, vol, blockIO))
			}
		}
		sb.WriteString("\n")

		// Memory analysis note
		sb.WriteString("### Memory Analysis Note\n\n")
		sb.WriteString("> **RSS (Resident Set Size)**: Actual application memory usage.\n")
		sb.WriteString("> \n")
		sb.WriteString("> **Cache**: Linux page cache from filesystem I/O. Disk-based drivers show higher ")
		sb.WriteString("total memory because the OS caches file pages in RAM. This memory is reclaimable ")
		sb.WriteString("and doesn't indicate a memory leak.\n")
		sb.WriteString("> \n")
		sb.WriteString("> Memory-based drivers (like `liteio_mem`) have minimal cache because data ")
		sb.WriteString("stays in application memory (RSS), not filesystem cache.\n\n")
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

// generateLeaderASCIITable creates a visually appealing ASCII table showing leaders per category.
func (r *Report) generateLeaderASCIITable(sb *strings.Builder) {
	// Collect leader data
	type leaderInfo struct {
		category string
		leader   string
		perf     string
		notes    string
	}

	var leaders []leaderInfo

	// Group results by operation category
	categoryBest := make(map[string]struct {
		driver     string
		throughput float64
		second     string
		secondVal  float64
	})

	for _, m := range r.Results {
		cat := m.Operation
		curr := categoryBest[cat]
		if m.Throughput > curr.throughput {
			// Current best becomes second
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

	// Define key categories to highlight
	keyCategories := []struct {
		operation string
		display   string
	}{
		{"Read/1KB", "Small File Read (1KB)"},
		{"Write/1KB", "Small File Write (1KB)"},
		{"Read/100MB", "Large File Read (100MB)"},
		{"Write/100MB", "Large File Write (100MB)"},
		{"Delete", "Delete Operations"},
		{"Stat", "Stat Operations"},
		{"List/100", "List Operations (100 obj)"},
		{"Copy/1KB", "Copy Operations"},
		{"RangeRead/Start_256KB", "Range Reads"},
		{"MixedWorkload/Balanced_50_50", "Mixed Workload"},
		{"ParallelRead/1KB/C200", "High Concurrency Read"},
		{"ParallelWrite/1KB/C200", "High Concurrency Write"},
	}

	for _, kc := range keyCategories {
		if best, ok := categoryBest[kc.operation]; ok && best.driver != "" {
			var perfStr string
			// Check if it's ops/s or MB/s based on operation
			for _, m := range r.Results {
				if m.Operation == kc.operation && m.Driver == best.driver {
					if m.ObjectSize > 0 {
						perfStr = fmt.Sprintf("%.1f MB/s", best.throughput)
					} else {
						perfStr = fmt.Sprintf("%.0f ops/s", best.throughput)
					}
					break
				}
			}

			// Calculate lead factor
			var notes string
			if best.second != "" && best.secondVal > 0 {
				factor := best.throughput / best.secondVal
				if factor >= 1.5 {
					notes = fmt.Sprintf("%.1fx faster than %s", factor, best.second)
				} else if factor >= 1.1 {
					notes = fmt.Sprintf("%.0f%% faster than %s", (factor-1)*100, best.second)
				} else {
					notes = "Close competition"
				}
			}

			leaders = append(leaders, leaderInfo{
				category: kc.display,
				leader:   best.driver + " " + perfStr,
				notes:    notes,
			})
		}
	}

	if len(leaders) == 0 {
		return
	}

	// Calculate column widths
	catWidth := 27
	leaderWidth := 23
	notesWidth := 31

	// Build ASCII table
	sb.WriteString("### Performance Leaders\n\n")
	sb.WriteString("```\n")

	// Top border
	sb.WriteString(fmt.Sprintf("┌%s┬%s┬%s┐\n",
		strings.Repeat("─", catWidth),
		strings.Repeat("─", leaderWidth),
		strings.Repeat("─", notesWidth)))

	// Header
	sb.WriteString(fmt.Sprintf("│%s│%s│%s│\n",
		centerString("Category", catWidth),
		centerString("Leader", leaderWidth),
		centerString("Notes", notesWidth)))

	// Header separator
	sb.WriteString(fmt.Sprintf("├%s┼%s┼%s┤\n",
		strings.Repeat("─", catWidth),
		strings.Repeat("─", leaderWidth),
		strings.Repeat("─", notesWidth)))

	// Data rows
	for i, l := range leaders {
		sb.WriteString(fmt.Sprintf("│%s│%s│%s│\n",
			padRight(l.category, catWidth),
			padRight(l.leader, leaderWidth),
			padRight(l.notes, notesWidth)))

		// Add separator between rows (except last)
		if i < len(leaders)-1 {
			sb.WriteString(fmt.Sprintf("├%s┼%s┼%s┤\n",
				strings.Repeat("─", catWidth),
				strings.Repeat("─", leaderWidth),
				strings.Repeat("─", notesWidth)))
		}
	}

	// Bottom border
	sb.WriteString(fmt.Sprintf("└%s┴%s┴%s┘\n",
		strings.Repeat("─", catWidth),
		strings.Repeat("─", leaderWidth),
		strings.Repeat("─", notesWidth)))

	sb.WriteString("```\n\n")
}

// centerString centers a string within a given width.
func centerString(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	left := (width - len(s)) / 2
	right := width - len(s) - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}

// padRight pads a string to the right with spaces.
func padRight(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return " " + s + strings.Repeat(" ", width-len(s)-1)
}

// generateExecutiveSummary creates a quick overview section at the top of the report.
func (r *Report) generateExecutiveSummary(sb *strings.Builder) {
	sb.WriteString("## Executive Summary\n\n")

	// Add ASCII leader table first
	r.generateLeaderASCIITable(sb)

	// Find the largest file size tested
	largestSize := 0
	largestSizeLabel := "1MB"
	for _, m := range r.Results {
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
		// Large file performance (dynamic - largest size tested)
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
		// Errors, skipped, and resource
		errors      int
		skipped     int
		skippedInfo []string
		memoryMB    float64
	}

	summaries := make(map[string]*driverSummary)
	largeWriteOp := "Write/" + largestSizeLabel
	largeReadOp := "Read/" + largestSizeLabel

	for _, m := range r.Results {
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

	// Track skipped benchmarks from SkippedBenchmarks field
	for _, skip := range r.SkippedBenchmarks {
		if summaries[skip.Driver] != nil {
			summaries[skip.Driver].skipped++
			summaries[skip.Driver].skippedInfo = append(summaries[skip.Driver].skippedInfo, skip.Reason)
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

	// File Count Performance Summary (if available)
	r.generateFileCountSummary(sb, drivers)

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

// generateFileCountSummary creates a summary of file count benchmark results.
func (r *Report) generateFileCountSummary(sb *strings.Builder, drivers []string) {
	// Collect file count results by operation type
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

	// Extract file count from operation name (e.g., "FileCount/Write/1000" -> 1000)
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
		if strings.HasPrefix(m.Operation, "FileCount/Write/") {
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
		if strings.HasPrefix(m.Operation, "FileCount/List/") {
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
		if strings.HasPrefix(m.Operation, "FileCount/Delete/") {
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

	sb.WriteString("### File Count Performance\n\n")
	sb.WriteString("Performance with varying numbers of files (1KB each).\n\n")

	if len(writeResults) > 0 {
		sb.WriteString("**Write N Files (total time)**\n\n")
		sb.WriteString("| Driver |")

		// Get all file counts
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
	Driver    string
	Operation string
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
			status = "⚠️ Regression"
		} else if c.Improvement {
			status = "✅ Improved"
		} else {
			status = "➖ Stable"
		}

		sb.WriteString(fmt.Sprintf("| %s | %s | %.2f | %.2f | %+.1f%% | %s |\n",
			c.Driver, c.Operation,
			c.BaselineThroughput, c.CurrentThroughput,
			c.ThroughputDelta, status))
	}
	sb.WriteString("\n")

	return sb.String()
}
