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
	fmt.Fprintf(w, "**Go Version:** %s\n", r.System.GoVersion)
	fmt.Fprintf(w, "**Drivers Tested:** %d\n\n", len(r.Results))

	// Executive Summary
	fmt.Fprintf(w, "## Executive Summary\n\n")
	r.writeExecutiveSummary(w)

	// Summary table
	fmt.Fprintf(w, "## Performance Summary\n\n")
	fmt.Fprintf(w, "| Driver | Type | Index Time | Index Size | p50 | p95 | p99 | QPS (1) | QPS (10) | QPS (max) |\n")
	fmt.Fprintf(w, "|--------|------|------------|------------|-----|-----|-----|---------|----------|----------|\n")

	for _, result := range r.Results {
		if result.Error != "" {
			fmt.Fprintf(w, "| %s | - | ERROR | - | - | - | - | - | - | - |\n", result.Name)
			continue
		}

		driverType := "embedded"
		if isExternalDriver(result.Name) {
			driverType = "external"
		}

		indexTime := "-"
		if result.Indexing != nil {
			indexTime = result.Indexing.Duration.Round(time.Second).String()
		}

		indexSize := "-"
		if result.IndexSize > 0 {
			indexSize = FormatBytes(result.IndexSize)
		}

		p50, p95, p99 := "-", "-", "-"
		if result.Latency != nil {
			p50 = result.Latency.P50.Round(time.Microsecond).String()
			p95 = result.Latency.P95.Round(time.Microsecond).String()
			p99 = result.Latency.P99.Round(time.Microsecond).String()
		}

		qps1 := "-"
		if result.Throughput != nil {
			qps1 = fmt.Sprintf("%.0f", result.Throughput.QPS)
		}

		qps10 := "-"
		if t, ok := result.Concurrency[10]; ok {
			qps10 = fmt.Sprintf("%.0f", t.QPS)
		}

		qpsMax := "-"
		maxQPS := 0.0
		for _, t := range result.Concurrency {
			if t.QPS > maxQPS {
				maxQPS = t.QPS
			}
		}
		if maxQPS > 0 {
			qpsMax = fmt.Sprintf("%.0f", maxQPS)
		}

		fmt.Fprintf(w, "| %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |\n",
			result.Name, driverType, indexTime, indexSize, p50, p95, p99, qps1, qps10, qpsMax)
	}

	// Indexing Performance Comparison
	fmt.Fprintf(w, "\n## Indexing Performance\n\n")
	fmt.Fprintf(w, "| Driver | Duration | Docs/sec | Peak Memory | Total Docs |\n")
	fmt.Fprintf(w, "|--------|----------|----------|-------------|------------|\n")
	for _, result := range r.Results {
		if result.Error != "" || result.Indexing == nil {
			continue
		}
		fmt.Fprintf(w, "| %s | %s | %.0f | %s | %d |\n",
			result.Name,
			result.Indexing.Duration.Round(time.Second),
			result.Indexing.DocsPerSec,
			FormatBytes(result.Indexing.PeakMemory),
			result.Indexing.TotalDocs)
	}

	// Latency Distribution
	fmt.Fprintf(w, "\n## Latency Distribution\n\n")
	fmt.Fprintf(w, "| Driver | Min | Avg | p50 | p95 | p99 | Max |\n")
	fmt.Fprintf(w, "|--------|-----|-----|-----|-----|-----|-----|\n")
	for _, result := range r.Results {
		if result.Error != "" || result.Latency == nil {
			continue
		}
		fmt.Fprintf(w, "| %s | %v | %v | %v | %v | %v | %v |\n",
			result.Name,
			result.Latency.Min.Round(time.Microsecond),
			result.Latency.Avg.Round(time.Microsecond),
			result.Latency.P50.Round(time.Microsecond),
			result.Latency.P95.Round(time.Microsecond),
			result.Latency.P99.Round(time.Microsecond),
			result.Latency.Max.Round(time.Microsecond))
	}

	// Scalability Analysis
	fmt.Fprintf(w, "\n## Scalability Analysis (QPS by Concurrency)\n\n")
	r.writeConcurrencyTable(w)

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
	fmt.Fprintf(w, "| Driver | Tokenizer | Stemmer | Diacritics | Notes |\n")
	fmt.Fprintf(w, "|--------|-----------|---------|------------|-------|\n")
	fmt.Fprintf(w, "| duckdb | Basic | None | Preserved | Uses FTS extension |\n")
	fmt.Fprintf(w, "| sqlite | Unicode61 | None | Preserved | FTS5 virtual table |\n")
	fmt.Fprintf(w, "| bleve | ICU Vietnamese | None | Preserved | Best Vietnamese support |\n")
	fmt.Fprintf(w, "| bluge | Shared Vietnamese | None | Preserved | Custom tokenizer |\n")
	fmt.Fprintf(w, "| tantivy | Vietnamese | None | Preserved | Requires CGO |\n")
	fmt.Fprintf(w, "| meilisearch | Auto-detect | None | Preserved | Good Unicode handling |\n")
	fmt.Fprintf(w, "| zinc | Basic | None | Preserved | Bluge-based |\n")
	fmt.Fprintf(w, "| porter | Shared Vietnamese | Porter (English) | Preserved | Custom inverted index |\n")
	fmt.Fprintf(w, "| opensearch | ICU | None | Preserved | Plugin required |\n")
	fmt.Fprintf(w, "| elasticsearch | ICU | None | Preserved | Plugin required |\n")
	fmt.Fprintf(w, "| postgres | Simple | None | Preserved | tsvector + GIN |\n")
	fmt.Fprintf(w, "| typesense | Unicode | None | Preserved | Good typo tolerance |\n")
	fmt.Fprintf(w, "| manticore | Charset table | None | Preserved | SQL interface |\n")
	fmt.Fprintf(w, "| quickwit | Default | None | Preserved | Cloud-native |\n")
	fmt.Fprintf(w, "| lnx | Raw | None | Preserved | Tantivy REST |\n")
	fmt.Fprintf(w, "| sonic | Basic | None | Preserved | ID-only storage |\n")
	fmt.Fprintf(w, "\n")

	// Driver Categories
	fmt.Fprintf(w, "## Driver Categories\n\n")
	fmt.Fprintf(w, "### Embedded (No External Dependencies)\n")
	fmt.Fprintf(w, "- **duckdb**: Analytical database with FTS, great for batch processing\n")
	fmt.Fprintf(w, "- **sqlite**: Lightweight, ACID-compliant, perfect for single-user apps\n")
	fmt.Fprintf(w, "- **bleve**: Full-featured search library with excellent Vietnamese support\n")
	fmt.Fprintf(w, "- **bluge**: Modern Bleve successor, better performance\n")
	fmt.Fprintf(w, "- **porter**: Custom inverted index with Porter stemming\n\n")

	fmt.Fprintf(w, "### External Services (Docker Required)\n")
	fmt.Fprintf(w, "- **meilisearch**: Developer-friendly, instant search, typo tolerance\n")
	fmt.Fprintf(w, "- **zinc**: Lightweight Elasticsearch alternative\n")
	fmt.Fprintf(w, "- **opensearch**: AWS fork, enterprise-ready, scalable\n")
	fmt.Fprintf(w, "- **elasticsearch**: Industry standard, most features\n")
	fmt.Fprintf(w, "- **postgres**: Full-text search in your existing database\n")
	fmt.Fprintf(w, "- **typesense**: Fast, typo-tolerant, simple API\n")
	fmt.Fprintf(w, "- **manticore**: SQL interface, very fast indexing\n")
	fmt.Fprintf(w, "- **quickwit**: Cloud-native, designed for logs\n")
	fmt.Fprintf(w, "- **lnx**: Tantivy via REST, no CGO needed\n")
	fmt.Fprintf(w, "- **sonic**: Ultra-fast search index layer\n\n")

	// Recommendations
	fmt.Fprintf(w, "## Recommendations\n\n")
	fmt.Fprintf(w, "Based on benchmark results:\n\n")

	// Find best in each category
	bestLatency := findBestLatency(r.Results)
	bestThroughput := findBestThroughput(r.Results)
	smallestIndex := findSmallestIndex(r.Results)
	fastestIndexing := findFastestIndexing(r.Results)

	fmt.Fprintf(w, "### Performance Leaders\n")
	if bestLatency != "" {
		fmt.Fprintf(w, "- **Lowest Latency:** %s\n", bestLatency)
	}
	if bestThroughput != "" {
		fmt.Fprintf(w, "- **Best Throughput:** %s\n", bestThroughput)
	}
	if fastestIndexing != "" {
		fmt.Fprintf(w, "- **Fastest Indexing:** %s\n", fastestIndexing)
	}
	if smallestIndex != "" {
		fmt.Fprintf(w, "- **Smallest Index:** %s\n", smallestIndex)
	}

	fmt.Fprintf(w, "\n### Use Case Recommendations\n")
	fmt.Fprintf(w, "- **Simple embedded search:** sqlite (no dependencies, ACID)\n")
	fmt.Fprintf(w, "- **High-performance embedded:** bluge or porter\n")
	fmt.Fprintf(w, "- **Developer-friendly SaaS-like:** meilisearch or typesense\n")
	fmt.Fprintf(w, "- **Enterprise distributed:** elasticsearch or opensearch\n")
	fmt.Fprintf(w, "- **Existing PostgreSQL stack:** postgres (no new infra)\n")
	fmt.Fprintf(w, "- **Maximum indexing speed:** manticore\n")
	fmt.Fprintf(w, "- **Minimum memory footprint:** sonic\n")
	fmt.Fprintf(w, "- **Cloud-native logs/traces:** quickwit\n")
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

func findFastestIndexing(results []*DriverResult) string {
	var best string
	var fastest float64

	for _, r := range results {
		if r.Error != "" || r.Indexing == nil {
			continue
		}
		if r.Indexing.DocsPerSec > fastest {
			fastest = r.Indexing.DocsPerSec
			best = r.Name
		}
	}

	if best != "" {
		return fmt.Sprintf("%s (%.0f docs/sec)", best, fastest)
	}
	return ""
}

func findBestScalability(results []*DriverResult) string {
	var best string
	var bestRatio float64

	for _, r := range results {
		if r.Error != "" || r.Throughput == nil || len(r.Concurrency) == 0 {
			continue
		}
		// Find max QPS at high concurrency
		var maxConcurrentQPS float64
		for _, t := range r.Concurrency {
			if t.QPS > maxConcurrentQPS {
				maxConcurrentQPS = t.QPS
			}
		}
		ratio := maxConcurrentQPS / r.Throughput.QPS
		if ratio > bestRatio {
			bestRatio = ratio
			best = r.Name
		}
	}

	if best != "" {
		return fmt.Sprintf("%s (%.1fx scaling)", best, bestRatio)
	}
	return ""
}

func isExternalDriver(name string) bool {
	external := map[string]bool{
		"meilisearch":   true,
		"zinc":          true,
		"opensearch":    true,
		"elasticsearch": true,
		"postgres":      true,
		"typesense":     true,
		"manticore":     true,
		"quickwit":      true,
		"lnx":           true,
		"sonic":         true,
	}
	return external[name]
}

func (r *Report) writeExecutiveSummary(w io.Writer) {
	bestLatency := findBestLatency(r.Results)
	bestThroughput := findBestThroughput(r.Results)
	smallestIndex := findSmallestIndex(r.Results)
	fastestIndexing := findFastestIndexing(r.Results)
	bestScalability := findBestScalability(r.Results)

	fmt.Fprintf(w, "| Category | Winner |\n")
	fmt.Fprintf(w, "|----------|--------|\n")
	if bestLatency != "" {
		fmt.Fprintf(w, "| Lowest Latency (p50) | %s |\n", bestLatency)
	}
	if bestThroughput != "" {
		fmt.Fprintf(w, "| Highest Single-Thread QPS | %s |\n", bestThroughput)
	}
	if fastestIndexing != "" {
		fmt.Fprintf(w, "| Fastest Indexing | %s |\n", fastestIndexing)
	}
	if smallestIndex != "" {
		fmt.Fprintf(w, "| Smallest Index Size | %s |\n", smallestIndex)
	}
	if bestScalability != "" {
		fmt.Fprintf(w, "| Best Scalability | %s |\n", bestScalability)
	}
	fmt.Fprintf(w, "\n")
}

func (r *Report) writeConcurrencyTable(w io.Writer) {
	// Collect all concurrency levels
	levels := make(map[int]bool)
	for _, result := range r.Results {
		for level := range result.Concurrency {
			levels[level] = true
		}
	}

	// Sort levels
	var sortedLevels []int
	for level := range levels {
		sortedLevels = append(sortedLevels, level)
	}
	sort.Ints(sortedLevels)

	if len(sortedLevels) == 0 {
		return
	}

	// Header
	fmt.Fprintf(w, "| Driver |")
	for _, level := range sortedLevels {
		fmt.Fprintf(w, " %d |", level)
	}
	fmt.Fprintf(w, " Scaling |\n")

	// Separator
	fmt.Fprintf(w, "|--------|")
	for range sortedLevels {
		fmt.Fprintf(w, "------|")
	}
	fmt.Fprintf(w, "---------|\n")

	// Data
	for _, result := range r.Results {
		if result.Error != "" || len(result.Concurrency) == 0 {
			continue
		}

		fmt.Fprintf(w, "| %s |", result.Name)

		var maxQPS, minQPS float64 = 0, 1e12
		for _, level := range sortedLevels {
			if t, ok := result.Concurrency[level]; ok {
				fmt.Fprintf(w, " %.0f |", t.QPS)
				if t.QPS > maxQPS {
					maxQPS = t.QPS
				}
				if t.QPS < minQPS {
					minQPS = t.QPS
				}
			} else {
				fmt.Fprintf(w, " - |")
			}
		}

		// Calculate scaling factor
		if minQPS > 0 && minQPS < 1e12 {
			fmt.Fprintf(w, " %.1fx |\n", maxQPS/minQPS)
		} else {
			fmt.Fprintf(w, " - |\n")
		}
	}
	fmt.Fprintf(w, "\n")
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
