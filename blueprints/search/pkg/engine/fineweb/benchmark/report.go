package benchmark

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

// WriteMarkdown generates a markdown report.
func (r *Report) WriteMarkdown(w io.Writer) error {
	// Header
	fmt.Fprintf(w, "# Fineweb Search Driver Benchmark Report\n\n")
	fmt.Fprintf(w, "**Date:** %s\n", r.StartTime.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "**Duration:** %v\n", r.EndTime.Sub(r.StartTime).Round(time.Second))
	fmt.Fprintf(w, "**System:** %s %s, %d CPUs, %s RAM\n", r.System.OS, r.System.Arch, r.System.CPUs, r.System.Memory)
	fmt.Fprintf(w, "**Go Version:** %s\n\n", r.System.GoVersion)

	// Summary table
	fmt.Fprintf(w, "## Summary\n\n")
	fmt.Fprintf(w, "| Driver | Index Time | Index Size | p50 Latency | p95 Latency | QPS (1) | QPS (10) |\n")
	fmt.Fprintf(w, "|--------|------------|------------|-------------|-------------|---------|----------|\n")

	for _, result := range r.Results {
		if result.Error != "" {
			fmt.Fprintf(w, "| %s | ERROR | - | - | - | - | - |\n", result.Name)
			continue
		}

		indexTime := "-"
		if result.Indexing != nil {
			indexTime = result.Indexing.Duration.Round(time.Second).String()
		}

		indexSize := "-"
		if result.IndexSize > 0 {
			indexSize = FormatBytes(result.IndexSize)
		}

		p50 := "-"
		p95 := "-"
		if result.Latency != nil {
			p50 = result.Latency.P50.Round(time.Microsecond).String()
			p95 = result.Latency.P95.Round(time.Microsecond).String()
		}

		qps1 := "-"
		if result.Throughput != nil {
			qps1 = fmt.Sprintf("%.0f", result.Throughput.QPS)
		}

		qps10 := "-"
		if t, ok := result.Concurrency[10]; ok {
			qps10 = fmt.Sprintf("%.0f", t.QPS)
		}

		fmt.Fprintf(w, "| %s | %s | %s | %s | %s | %s | %s |\n",
			result.Name, indexTime, indexSize, p50, p95, qps1, qps10)
	}

	// Detailed results for each driver
	fmt.Fprintf(w, "\n## Detailed Results\n")

	for _, result := range r.Results {
		fmt.Fprintf(w, "\n### %s\n\n", result.Name)

		if result.Error != "" {
			fmt.Fprintf(w, "**Error:** %s\n", result.Error)
			continue
		}

		// Indexing
		if result.Indexing != nil {
			fmt.Fprintf(w, "#### Indexing\n\n")
			fmt.Fprintf(w, "- Duration: %v\n", result.Indexing.Duration.Round(time.Second))
			fmt.Fprintf(w, "- Documents: %d\n", result.Indexing.TotalDocs)
			fmt.Fprintf(w, "- Throughput: %.0f docs/sec\n", result.Indexing.DocsPerSec)
			fmt.Fprintf(w, "- Peak Memory: %s\n\n", FormatBytes(result.Indexing.PeakMemory))
		}

		// Index Size
		if result.IndexSize > 0 {
			fmt.Fprintf(w, "#### Index Size\n\n")
			fmt.Fprintf(w, "- Size: %s\n\n", FormatBytes(result.IndexSize))
		}

		// Latency
		if result.Latency != nil {
			fmt.Fprintf(w, "#### Search Latency\n\n")
			fmt.Fprintf(w, "| Percentile | Latency |\n")
			fmt.Fprintf(w, "|------------|--------|\n")
			fmt.Fprintf(w, "| p50 | %v |\n", result.Latency.P50.Round(time.Microsecond))
			fmt.Fprintf(w, "| p95 | %v |\n", result.Latency.P95.Round(time.Microsecond))
			fmt.Fprintf(w, "| p99 | %v |\n", result.Latency.P99.Round(time.Microsecond))
			fmt.Fprintf(w, "| max | %v |\n", result.Latency.Max.Round(time.Microsecond))
			fmt.Fprintf(w, "| avg | %v |\n\n", result.Latency.Avg.Round(time.Microsecond))
		}

		// Throughput by concurrency
		if len(result.Concurrency) > 0 {
			fmt.Fprintf(w, "#### Throughput by Concurrency\n\n")
			fmt.Fprintf(w, "| Goroutines | QPS |\n")
			fmt.Fprintf(w, "|------------|-----|\n")

			// Sort concurrency levels
			var levels []int
			for level := range result.Concurrency {
				levels = append(levels, level)
			}
			sort.Ints(levels)

			for _, level := range levels {
				t := result.Concurrency[level]
				fmt.Fprintf(w, "| %d | %.0f |\n", level, t.QPS)
			}
			fmt.Fprintf(w, "\n")
		}

		// Cold start
		if result.ColdStart > 0 {
			fmt.Fprintf(w, "#### Cold Start\n\n")
			fmt.Fprintf(w, "Time to first search after restart: %v\n\n", result.ColdStart.Round(time.Millisecond))
		}

		// Memory
		if result.Memory != nil {
			fmt.Fprintf(w, "#### Memory Usage\n\n")
			fmt.Fprintf(w, "- Indexing Peak: %s\n", FormatBytes(result.Memory.IndexingPeak))
			if result.Memory.SearchIdle > 0 {
				fmt.Fprintf(w, "- Search Idle: %s\n", FormatBytes(result.Memory.SearchIdle))
			}
			if result.Memory.SearchPeak > 0 {
				fmt.Fprintf(w, "- Search Peak: %s\n", FormatBytes(result.Memory.SearchPeak))
			}
			fmt.Fprintf(w, "\n")
		}

		// Query stats
		if len(result.QueryStats) > 0 {
			fmt.Fprintf(w, "#### Query Performance\n\n")
			fmt.Fprintf(w, "| Query | Type | Results | Latency |\n")
			fmt.Fprintf(w, "|-------|------|---------|--------|\n")

			// Sort queries
			var queries []string
			for q := range result.QueryStats {
				queries = append(queries, q)
			}
			sort.Strings(queries)

			for _, q := range queries {
				qs := result.QueryStats[q]
				fmt.Fprintf(w, "| %s | %s | %d | %v |\n",
					q, qs.Query.Type, qs.Results, qs.Duration.Round(time.Microsecond))
			}
			fmt.Fprintf(w, "\n")
		}
	}

	// Vietnamese language notes
	fmt.Fprintf(w, "## Vietnamese Language Support\n\n")
	fmt.Fprintf(w, "| Driver | Tokenizer | Stemmer | Diacritics |\n")
	fmt.Fprintf(w, "|--------|-----------|---------|------------|\n")
	fmt.Fprintf(w, "| duckdb | Basic | None | Preserved |\n")
	fmt.Fprintf(w, "| sqlite | Unicode61 | None | Preserved |\n")
	fmt.Fprintf(w, "| bleve | ICU Vietnamese | None | Preserved |\n")
	fmt.Fprintf(w, "| bluge | Shared Vietnamese | None | Preserved |\n")
	fmt.Fprintf(w, "| tantivy | Vietnamese | None | Preserved |\n")
	fmt.Fprintf(w, "| meilisearch | Auto-detect | None | Preserved |\n")
	fmt.Fprintf(w, "| zinc | Basic | None | Preserved |\n")
	fmt.Fprintf(w, "| porter | Shared Vietnamese | Porter (English) | Preserved |\n")
	fmt.Fprintf(w, "\n")

	// Recommendations
	fmt.Fprintf(w, "## Recommendations\n\n")
	fmt.Fprintf(w, "Based on benchmark results:\n\n")

	// Find best in each category
	bestLatency := findBestLatency(r.Results)
	bestThroughput := findBestThroughput(r.Results)
	smallestIndex := findSmallestIndex(r.Results)

	if bestLatency != "" {
		fmt.Fprintf(w, "- **Lowest Latency:** %s\n", bestLatency)
	}
	if bestThroughput != "" {
		fmt.Fprintf(w, "- **Best Throughput:** %s\n", bestThroughput)
	}
	if smallestIndex != "" {
		fmt.Fprintf(w, "- **Smallest Index:** %s\n", smallestIndex)
	}
	fmt.Fprintf(w, "- **Easiest Setup:** sqlite (embedded, no dependencies)\n")
	fmt.Fprintf(w, "- **Most Features:** meilisearch (typo tolerance, facets, etc.)\n")
	fmt.Fprintf(w, "\n---\n")
	fmt.Fprintf(w, "*Report generated by fineweb benchmark suite*\n")

	return nil
}

// WriteJSON generates a JSON report.
func (r *Report) WriteJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

// CombineReports merges multiple single-driver reports.
func CombineReports(reports ...*Report) *Report {
	if len(reports) == 0 {
		return &Report{}
	}

	combined := &Report{
		StartTime: reports[0].StartTime,
		System:    reports[0].System,
	}

	for _, r := range reports {
		combined.Results = append(combined.Results, r.Results...)
		if r.EndTime.After(combined.EndTime) {
			combined.EndTime = r.EndTime
		}
	}

	return combined
}

// LoadReport loads a report from JSON.
func LoadReport(r io.Reader) (*Report, error) {
	var report Report
	if err := json.NewDecoder(r).Decode(&report); err != nil {
		return nil, err
	}
	return &report, nil
}

func findBestLatency(results []*DriverResult) string {
	var best string
	var bestP50 time.Duration = time.Hour

	for _, r := range results {
		if r.Error != "" || r.Latency == nil {
			continue
		}
		if r.Latency.P50 < bestP50 {
			bestP50 = r.Latency.P50
			best = r.Name
		}
	}

	if best != "" {
		return fmt.Sprintf("%s (p50=%v)", best, bestP50.Round(time.Microsecond))
	}
	return ""
}

func findBestThroughput(results []*DriverResult) string {
	var best string
	var bestQPS float64

	for _, r := range results {
		if r.Error != "" || r.Throughput == nil {
			continue
		}
		if r.Throughput.QPS > bestQPS {
			bestQPS = r.Throughput.QPS
			best = r.Name
		}
	}

	if best != "" {
		return fmt.Sprintf("%s (%.0f QPS)", best, bestQPS)
	}
	return ""
}

func findSmallestIndex(results []*DriverResult) string {
	var best string
	var smallest int64 = 1 << 62

	for _, r := range results {
		if r.Error != "" || r.IndexSize == 0 {
			continue
		}
		if r.IndexSize < smallest {
			smallest = r.IndexSize
			best = r.Name
		}
	}

	if best != "" {
		return fmt.Sprintf("%s (%s)", best, FormatBytes(smallest))
	}
	return ""
}

// String returns a short summary of the report.
func (r *Report) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Benchmark Report (%d drivers)\n", len(r.Results)))

	for _, result := range r.Results {
		if result.Error != "" {
			sb.WriteString(fmt.Sprintf("  %s: ERROR - %s\n", result.Name, result.Error))
			continue
		}

		p50 := "-"
		if result.Latency != nil {
			p50 = result.Latency.P50.Round(time.Microsecond).String()
		}

		qps := "-"
		if result.Throughput != nil {
			qps = fmt.Sprintf("%.0f", result.Throughput.QPS)
		}

		sb.WriteString(fmt.Sprintf("  %s: p50=%s QPS=%s\n", result.Name, p50, qps))
	}

	return sb.String()
}
