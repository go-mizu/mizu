package main

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// GenerateMarkdownReport creates a comprehensive markdown report from benchmark results.
func GenerateMarkdownReport(results *BenchResults, config *BenchConfig) string {
	var sb strings.Builder

	// Header
	sb.WriteString("# Spreadsheet Storage Benchmark Report\n\n")
	sb.WriteString(fmt.Sprintf("**Generated**: %s\n\n", results.StartTime.Format(time.RFC3339)))

	// System info
	writeSystemInfo(&sb, results.SystemInfo)

	// Configuration
	writeConfig(&sb, config)

	// Summary
	writeSummary(&sb, results)

	// Results by category
	writeResultsByCategory(&sb, results)

	// Driver comparison
	writeDriverComparison(&sb, results)

	// Recommendations
	writeRecommendations(&sb, results)

	return sb.String()
}

func writeSystemInfo(sb *strings.Builder, info SystemInfo) {
	sb.WriteString("## System Information\n\n")
	sb.WriteString("| Property | Value |\n")
	sb.WriteString("|----------|-------|\n")
	sb.WriteString(fmt.Sprintf("| OS | %s |\n", info.OS))
	sb.WriteString(fmt.Sprintf("| Architecture | %s |\n", info.Arch))
	sb.WriteString(fmt.Sprintf("| CPUs | %d |\n", info.CPUs))
	sb.WriteString(fmt.Sprintf("| Go Version | %s |\n", info.GoVersion))
	sb.WriteString(fmt.Sprintf("| GOMAXPROCS | %d |\n", info.GoMaxProcs))
	sb.WriteString("\n")
}

func writeConfig(sb *strings.Builder, config *BenchConfig) {
	sb.WriteString("## Configuration\n\n")
	sb.WriteString(fmt.Sprintf("- **Drivers**: %s\n", strings.Join(config.Drivers, ", ")))
	sb.WriteString(fmt.Sprintf("- **Categories**: %s\n", strings.Join(config.Categories, ", ")))
	sb.WriteString(fmt.Sprintf("- **Iterations**: %d\n", config.Iterations))
	sb.WriteString(fmt.Sprintf("- **Warmup**: %d\n", config.Warmup))
	sb.WriteString(fmt.Sprintf("- **Quick Mode**: %v\n", config.Quick))
	sb.WriteString("\n")
}

func writeSummary(sb *strings.Builder, results *BenchResults) {
	sb.WriteString("## Summary\n\n")
	sb.WriteString(fmt.Sprintf("- **Total Duration**: %v\n", results.TotalDuration.Round(time.Millisecond)))
	sb.WriteString(fmt.Sprintf("- **Benchmarks Run**: %d\n", len(results.Results)))

	// Count by category
	byCategory := make(map[string]int)
	for _, r := range results.Results {
		byCategory[r.Category]++
	}

	sb.WriteString("- **By Category**:\n")
	for cat, count := range byCategory {
		sb.WriteString(fmt.Sprintf("  - %s: %d\n", cat, count))
	}
	sb.WriteString("\n")

	// Error summary
	var errors []BenchResult
	for _, r := range results.Results {
		if r.Error != "" {
			errors = append(errors, r)
		}
	}
	if len(errors) > 0 {
		sb.WriteString(fmt.Sprintf("- **Errors**: %d benchmarks failed\n", len(errors)))
		for _, e := range errors {
			sb.WriteString(fmt.Sprintf("  - %s/%s (%s): %s\n", e.Category, e.Name, e.Driver, e.Error))
		}
	}
	sb.WriteString("\n")
}

func writeResultsByCategory(sb *strings.Builder, results *BenchResults) {
	// Group results by category
	byCategory := make(map[string][]BenchResult)
	for _, r := range results.Results {
		byCategory[r.Category] = append(byCategory[r.Category], r)
	}

	// Sort categories
	categories := make([]string, 0, len(byCategory))
	for cat := range byCategory {
		categories = append(categories, cat)
	}
	sort.Strings(categories)

	for _, category := range categories {
		catResults := byCategory[category]

		sb.WriteString(fmt.Sprintf("## %s Benchmarks\n\n", strings.Title(category)))

		// Group by benchmark name within category
		byName := make(map[string][]BenchResult)
		for _, r := range catResults {
			byName[r.Name] = append(byName[r.Name], r)
		}

		// Sort benchmark names
		names := make([]string, 0, len(byName))
		for name := range byName {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			benchResults := byName[name]

			// Sort by driver for consistent output
			sort.Slice(benchResults, func(i, j int) bool {
				return benchResults[i].Driver < benchResults[j].Driver
			})

			sb.WriteString(fmt.Sprintf("### %s\n\n", name))

			// Determine if this has load test percentiles
			hasPercentiles := false
			for _, r := range benchResults {
				if r.P50 > 0 {
					hasPercentiles = true
					break
				}
			}

			if hasPercentiles {
				writePercentileTable(sb, benchResults)
			} else {
				writeStandardTable(sb, benchResults)
			}

			sb.WriteString("\n")
		}
	}
}

func writeStandardTable(sb *strings.Builder, results []BenchResult) {
	// Check if results have throughput data
	hasThroughput := false
	for _, r := range results {
		if r.Throughput > 0 {
			hasThroughput = true
			break
		}
	}

	if hasThroughput {
		sb.WriteString("| Driver | Duration | Throughput | Cells/Op | Allocs/Op |\n")
		sb.WriteString("|--------|----------|------------|----------|----------|\n")
		for _, r := range results {
			if r.Error != "" {
				sb.WriteString(fmt.Sprintf("| %s | ERROR | - | - | - |\n", r.Driver))
				continue
			}
			sb.WriteString(fmt.Sprintf("| %s | %v | %.0f cells/sec | %d | %d |\n",
				r.Driver,
				formatDuration(r.Duration),
				r.Throughput,
				r.CellsPerOp,
				r.AllocsPerOp,
			))
		}
	} else {
		sb.WriteString("| Driver | Duration | ns/op | Cells/Op |\n")
		sb.WriteString("|--------|----------|-------|----------|\n")
		for _, r := range results {
			if r.Error != "" {
				sb.WriteString(fmt.Sprintf("| %s | ERROR | - | - |\n", r.Driver))
				continue
			}
			sb.WriteString(fmt.Sprintf("| %s | %v | %.0f | %d |\n",
				r.Driver,
				formatDuration(r.Duration),
				r.NsPerOp,
				r.CellsPerOp,
			))
		}
	}

	// Find fastest
	fastest := findFastest(results)
	if fastest != nil {
		sb.WriteString(fmt.Sprintf("\n**Fastest**: %s\n", fastest.Driver))
	}
}

func writePercentileTable(sb *strings.Builder, results []BenchResult) {
	sb.WriteString("| Driver | Throughput | p50 | p95 | p99 | max |\n")
	sb.WriteString("|--------|------------|-----|-----|-----|-----|\n")

	for _, r := range results {
		if r.Error != "" {
			sb.WriteString(fmt.Sprintf("| %s | ERROR | - | - | - | - |\n", r.Driver))
			continue
		}
		sb.WriteString(fmt.Sprintf("| %s | %.0f ops/sec | %v | %v | %v | %v |\n",
			r.Driver,
			r.Throughput,
			formatDuration(r.P50),
			formatDuration(r.P95),
			formatDuration(r.P99),
			formatDuration(r.Max),
		))
	}

	// Find best throughput
	var best *BenchResult
	for i := range results {
		if results[i].Error == "" {
			if best == nil || results[i].Throughput > best.Throughput {
				best = &results[i]
			}
		}
	}
	if best != nil {
		sb.WriteString(fmt.Sprintf("\n**Best Throughput**: %s (%.0f ops/sec)\n", best.Driver, best.Throughput))
	}
}

func writeDriverComparison(sb *strings.Builder, results *BenchResults) {
	sb.WriteString("## Driver Comparison\n\n")

	// Collect all drivers
	driversMap := make(map[string]bool)
	for _, r := range results.Results {
		driversMap[r.Driver] = true
	}
	drivers := make([]string, 0, len(driversMap))
	for d := range driversMap {
		drivers = append(drivers, d)
	}
	sort.Strings(drivers)

	// Wins table
	sb.WriteString("### Performance Wins by Category\n\n")
	sb.WriteString("| Category |")
	for _, d := range drivers {
		sb.WriteString(fmt.Sprintf(" %s |", d))
	}
	sb.WriteString("\n|----------|")
	for range drivers {
		sb.WriteString("------|")
	}
	sb.WriteString("\n")

	// Group by category and name, then find winner
	byCategory := make(map[string]map[string][]BenchResult)
	for _, r := range results.Results {
		if byCategory[r.Category] == nil {
			byCategory[r.Category] = make(map[string][]BenchResult)
		}
		byCategory[r.Category][r.Name] = append(byCategory[r.Category][r.Name], r)
	}

	categoryWins := make(map[string]map[string]int)
	for cat, benchmarks := range byCategory {
		categoryWins[cat] = make(map[string]int)
		for _, benchResults := range benchmarks {
			fastest := findFastest(benchResults)
			if fastest != nil {
				categoryWins[cat][fastest.Driver]++
			}
		}
	}

	// Sort categories
	categories := make([]string, 0, len(categoryWins))
	for cat := range categoryWins {
		categories = append(categories, cat)
	}
	sort.Strings(categories)

	for _, cat := range categories {
		sb.WriteString(fmt.Sprintf("| %s |", cat))
		for _, d := range drivers {
			wins := categoryWins[cat][d]
			if wins > 0 {
				sb.WriteString(fmt.Sprintf(" %d |", wins))
			} else {
				sb.WriteString(" - |")
			}
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	// Overall winner
	totalWins := make(map[string]int)
	for _, catWins := range categoryWins {
		for driver, wins := range catWins {
			totalWins[driver] += wins
		}
	}

	sb.WriteString("### Overall Winners\n\n")
	type driverWins struct {
		driver string
		wins   int
	}
	winsList := make([]driverWins, 0, len(totalWins))
	for d, w := range totalWins {
		winsList = append(winsList, driverWins{d, w})
	}
	sort.Slice(winsList, func(i, j int) bool {
		return winsList[i].wins > winsList[j].wins
	})

	for i, dw := range winsList {
		medal := ""
		switch i {
		case 0:
			medal = "1st"
		case 1:
			medal = "2nd"
		case 2:
			medal = "3rd"
		default:
			medal = fmt.Sprintf("%dth", i+1)
		}
		sb.WriteString(fmt.Sprintf("- **%s**: %s (%d wins)\n", medal, dw.driver, dw.wins))
	}
	sb.WriteString("\n")

	// Relative performance table
	writeRelativePerformance(sb, results, drivers)
}

func writeRelativePerformance(sb *strings.Builder, results *BenchResults, drivers []string) {
	sb.WriteString("### Relative Performance (vs Fastest)\n\n")

	// Group by benchmark
	byBenchmark := make(map[string][]BenchResult)
	for _, r := range results.Results {
		key := r.Category + "/" + r.Name
		byBenchmark[key] = append(byBenchmark[key], r)
	}

	// Calculate average relative performance
	driverRelative := make(map[string][]float64)
	for _, benchResults := range byBenchmark {
		fastest := findFastest(benchResults)
		if fastest == nil || fastest.NsPerOp == 0 {
			continue
		}

		for _, r := range benchResults {
			if r.Error == "" && r.NsPerOp > 0 {
				relative := r.NsPerOp / fastest.NsPerOp
				driverRelative[r.Driver] = append(driverRelative[r.Driver], relative)
			}
		}
	}

	sb.WriteString("| Driver | Avg Relative Time | Interpretation |\n")
	sb.WriteString("|--------|-------------------|----------------|\n")

	for _, d := range drivers {
		relatives := driverRelative[d]
		if len(relatives) == 0 {
			sb.WriteString(fmt.Sprintf("| %s | N/A | No data |\n", d))
			continue
		}

		// Calculate average
		var sum float64
		for _, r := range relatives {
			sum += r
		}
		avg := sum / float64(len(relatives))

		interpretation := ""
		switch {
		case avg <= 1.1:
			interpretation = "Fastest or near-fastest"
		case avg <= 1.5:
			interpretation = "Competitive"
		case avg <= 2.0:
			interpretation = "Slower but acceptable"
		default:
			interpretation = "Significantly slower"
		}

		sb.WriteString(fmt.Sprintf("| %s | %.2fx | %s |\n", d, avg, interpretation))
	}
	sb.WriteString("\n")
}

func writeRecommendations(sb *strings.Builder, results *BenchResults) {
	sb.WriteString("## Recommendations\n\n")

	// Analyze results to make recommendations
	recommendations := analyzeRecommendations(results)

	for useCase, rec := range recommendations {
		sb.WriteString(fmt.Sprintf("### %s\n\n", useCase))
		sb.WriteString(fmt.Sprintf("**Recommended**: %s\n\n", rec.Driver))
		sb.WriteString("**Reasons**:\n")
		for _, reason := range rec.Reasons {
			sb.WriteString(fmt.Sprintf("- %s\n", reason))
		}
		sb.WriteString("\n")
	}
}

type recommendation struct {
	Driver  string
	Reasons []string
}

func analyzeRecommendations(results *BenchResults) map[string]recommendation {
	recs := make(map[string]recommendation)

	// Analyze by looking at specific benchmark categories
	categoryWins := make(map[string]map[string]int)
	categoryThroughput := make(map[string]map[string]float64)

	for _, r := range results.Results {
		if r.Error != "" {
			continue
		}

		if categoryWins[r.Category] == nil {
			categoryWins[r.Category] = make(map[string]int)
			categoryThroughput[r.Category] = make(map[string]float64)
		}

		// Track throughput for comparison
		if r.Throughput > categoryThroughput[r.Category][r.Driver] {
			categoryThroughput[r.Category][r.Driver] = r.Throughput
		}
	}

	// Group by benchmark to find wins
	byBenchmark := make(map[string][]BenchResult)
	for _, r := range results.Results {
		key := r.Category + "/" + r.Name
		byBenchmark[key] = append(byBenchmark[key], r)
	}

	for _, benchResults := range byBenchmark {
		if len(benchResults) == 0 {
			continue
		}
		fastest := findFastest(benchResults)
		if fastest != nil {
			categoryWins[fastest.Category][fastest.Driver]++
		}
	}

	// Financial Modeling (needs fast single-cell ops and batch writes)
	financialDriver := findBestDriver(categoryWins, "cells", "format")
	recs["Financial Modeling"] = recommendation{
		Driver: financialDriver,
		Reasons: []string{
			"Best performance for cell operations",
			"Efficient handling of formatted cells",
			"Good batch write performance for large models",
		},
	}

	// Data Import (needs high throughput batch writes)
	importDriver := findBestDriver(categoryWins, "usecase", "cells")
	recs["Data Import Pipeline"] = recommendation{
		Driver: importDriver,
		Reasons: []string{
			"Highest batch import throughput",
			"Efficient handling of large datasets",
			"Good memory efficiency during bulk operations",
		},
	}

	// Real-time Collaboration (needs fast reads and low latency)
	if _, ok := categoryWins["load"]; ok {
		collabDriver := findBestDriver(categoryWins, "load", "cells")
		recs["Real-time Collaboration"] = recommendation{
			Driver: collabDriver,
			Reasons: []string{
				"Lowest latency for read/write operations",
				"Good performance under concurrent load",
				"Consistent p95/p99 latencies",
			},
		}
	}

	// Report Generation (needs fast range queries)
	reportDriver := findBestDriver(categoryWins, "query", "cells")
	recs["Report Generation"] = recommendation{
		Driver: reportDriver,
		Reasons: []string{
			"Fast range query performance",
			"Efficient large data retrieval",
			"Good aggregation query support",
		},
	}

	// Desktop/Embedded (SQLite is usually best)
	recs["Desktop/Embedded Use"] = recommendation{
		Driver: "sqlite",
		Reasons: []string{
			"Zero server configuration required",
			"Single-file database deployment",
			"Good single-user performance",
			"WAL mode for concurrent reads",
		},
	}

	return recs
}

func findBestDriver(categoryWins map[string]map[string]int, categories ...string) string {
	totalWins := make(map[string]int)
	for _, cat := range categories {
		for driver, wins := range categoryWins[cat] {
			totalWins[driver] += wins
		}
	}

	best := ""
	bestWins := 0
	for driver, wins := range totalWins {
		if wins > bestWins {
			best = driver
			bestWins = wins
		}
	}

	if best == "" {
		return "duckdb" // default
	}
	return best
}

func findFastest(results []BenchResult) *BenchResult {
	var fastest *BenchResult
	for i := range results {
		if results[i].Error != "" {
			continue
		}
		if fastest == nil || results[i].NsPerOp < fastest.NsPerOp {
			fastest = &results[i]
		}
	}
	return fastest
}

func formatDuration(d time.Duration) string {
	if d == 0 {
		return "-"
	}
	if d < time.Microsecond {
		return fmt.Sprintf("%dns", d.Nanoseconds())
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%.1fus", float64(d.Nanoseconds())/1000)
	}
	if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Nanoseconds())/1000000)
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}
