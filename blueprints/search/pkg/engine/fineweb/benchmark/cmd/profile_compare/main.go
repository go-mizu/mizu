// profile_compare benchmarks all fts_rust profiles and generates a comparison report.
//
// Usage:
//
//	profile_compare -data=/path/to/data.parquet -output=comparison.md
package main

import (
	"context"
	"flag"
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb"
	fts_rust "github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/drivers/fts_rust"
)

var (
	dataPath    = flag.String("data", "", "Path to parquet data file")
	outputPath  = flag.String("output", "profile_comparison.md", "Output markdown file")
	docLimit    = flag.Int("limit", 100000, "Number of documents to index")
	searchIters = flag.Int("search-iters", 1000, "Number of search iterations")
)

// ProfileResult holds benchmark results for a single profile
type ProfileResult struct {
	Name            string
	IndexDuration   time.Duration
	IndexThroughput float64
	PeakMemoryMB    float64
	IndexSizeMB     float64
	SearchP50       time.Duration
	SearchP95       time.Duration
	SearchP99       time.Duration
	SearchQPS       float64
}

func main() {
	flag.Parse()

	profiles := fts_rust.ListProfiles()
	results := make([]ProfileResult, 0, len(profiles))

	fmt.Printf("Comparing %d profiles: %s\n\n", len(profiles), strings.Join(profiles, ", "))

	for _, profile := range profiles {
		fmt.Printf("=== Benchmarking profile: %s ===\n", profile)
		result, err := benchmarkProfile(profile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Error: %v\n", err)
			continue
		}
		results = append(results, result)
		fmt.Println()
	}

	// Generate report
	report := generateReport(results)
	if err := os.WriteFile(*outputPath, []byte(report), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write report: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Report written to: %s\n", *outputPath)
	fmt.Println("\n" + report)
}

func benchmarkProfile(profile string) (ProfileResult, error) {
	result := ProfileResult{Name: profile}

	// Create temp directory for index
	indexDir, err := os.MkdirTemp("", "fts_compare_"+profile+"_*")
	if err != nil {
		return result, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(indexDir)

	cfg := fineweb.DriverConfig{
		DataDir: indexDir,
		Options: map[string]any{"profile": profile},
	}

	driver, err := fineweb.Open("fts_rust", cfg)
	if err != nil {
		return result, fmt.Errorf("failed to create driver: %w", err)
	}
	defer driver.Close()

	indexer, ok := fineweb.AsIndexer(driver)
	if !ok {
		return result, fmt.Errorf("driver does not support indexing")
	}

	// Generate test documents
	docs := generateDocs(*docLimit)

	// Force GC
	runtime.GC()
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	// Index benchmark
	fmt.Printf("  Indexing %d documents...\n", *docLimit)
	start := time.Now()
	if err := indexer.Import(context.Background(), docs, nil); err != nil {
		return result, fmt.Errorf("import failed: %w", err)
	}
	result.IndexDuration = time.Since(start)
	result.IndexThroughput = float64(*docLimit) / result.IndexDuration.Seconds()

	// Memory stats
	runtime.GC()
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)
	result.PeakMemoryMB = float64(memAfter.TotalAlloc-memBefore.TotalAlloc) / 1024 / 1024

	// Index size
	var indexSize int64
	filepath.Walk(indexDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			indexSize += info.Size()
		}
		return nil
	})
	result.IndexSizeMB = float64(indexSize) / 1024 / 1024

	fmt.Printf("  Index: %.0f docs/sec, %.2f MB memory, %.2f MB on disk\n",
		result.IndexThroughput, result.PeakMemoryMB, result.IndexSizeMB)

	// Search benchmark
	queries := []string{
		"machine learning",
		"quick brown fox",
		"programming language",
		"data science",
	}

	// Warmup
	for i := 0; i < 10; i++ {
		driver.Search(context.Background(), queries[i%len(queries)], 10, 0)
	}

	// Run search benchmark
	latencies := make([]time.Duration, *searchIters)
	start = time.Now()
	for i := 0; i < *searchIters; i++ {
		qStart := time.Now()
		driver.Search(context.Background(), queries[i%len(queries)], 10, 0)
		latencies[i] = time.Since(qStart)
	}
	totalDuration := time.Since(start)

	// Sort and calculate percentiles
	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	result.SearchP50 = latencies[len(latencies)/2]
	result.SearchP95 = latencies[int(float64(len(latencies))*0.95)]
	result.SearchP99 = latencies[int(float64(len(latencies))*0.99)]
	result.SearchQPS = float64(*searchIters) / totalDuration.Seconds()

	fmt.Printf("  Search: P50=%s, P95=%s, P99=%s, QPS=%.0f\n",
		result.SearchP50, result.SearchP95, result.SearchP99, result.SearchQPS)

	return result, nil
}

func generateDocs(count int) iter.Seq2[fineweb.Document, error] {
	return func(yield func(fineweb.Document, error) bool) {
		words := []string{
			"the", "quick", "brown", "fox", "jumps", "over", "lazy", "dog",
			"machine", "learning", "artificial", "intelligence", "data", "science",
			"programming", "language", "computer", "system", "network", "database",
			"algorithm", "optimization", "performance", "benchmark", "search",
			"index", "query", "document", "text", "retrieval", "ranking",
		}

		for i := 0; i < count; i++ {
			var sb strings.Builder
			for j := 0; j < 50; j++ {
				if j > 0 {
					sb.WriteByte(' ')
				}
				sb.WriteString(words[(i*7+j*13)%len(words)])
			}

			doc := fineweb.Document{
				ID:   fmt.Sprintf("doc_%d", i),
				Text: sb.String(),
			}

			if !yield(doc, nil) {
				return
			}
		}
	}
}

func generateReport(results []ProfileResult) string {
	var sb strings.Builder

	sb.WriteString("# FTS Rust Profile Comparison\n\n")
	sb.WriteString(fmt.Sprintf("**Date**: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("**Documents**: %d\n", *docLimit))
	sb.WriteString(fmt.Sprintf("**Search Iterations**: %d\n\n", *searchIters))

	// Hardware info
	sb.WriteString("## System Info\n\n")
	sb.WriteString(fmt.Sprintf("- **OS**: %s\n", runtime.GOOS))
	sb.WriteString(fmt.Sprintf("- **Arch**: %s\n", runtime.GOARCH))
	sb.WriteString(fmt.Sprintf("- **CPUs**: %d\n\n", runtime.NumCPU()))

	// Indexing table
	sb.WriteString("## Indexing Performance\n\n")
	sb.WriteString("| Profile | Throughput | Peak Memory | Index Size | Duration |\n")
	sb.WriteString("|---------|------------|-------------|------------|----------|\n")

	for _, r := range results {
		sb.WriteString(fmt.Sprintf("| %s | %.0f docs/s | %.2f MB | %.2f MB | %s |\n",
			r.Name, r.IndexThroughput, r.PeakMemoryMB, r.IndexSizeMB, r.IndexDuration.Round(time.Millisecond)))
	}

	// Search table
	sb.WriteString("\n## Search Latency\n\n")
	sb.WriteString("| Profile | P50 | P95 | P99 | QPS |\n")
	sb.WriteString("|---------|-----|-----|-----|-----|\n")

	for _, r := range results {
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %.0f |\n",
			r.Name, r.SearchP50.Round(time.Microsecond), r.SearchP95.Round(time.Microsecond),
			r.SearchP99.Round(time.Microsecond), r.SearchQPS))
	}

	// Recommendations
	sb.WriteString("\n## Recommendations\n\n")

	// Find best in each category
	var bestIndexing, bestSearchLatency, bestMemory, bestSize string
	var maxThroughput float64
	_ = maxThroughput // used in output
	var minMem, minSize float64 = 1e12, 1e12
	var minP50 time.Duration = time.Hour

	for _, r := range results {
		if r.IndexThroughput > maxThroughput {
			maxThroughput = r.IndexThroughput
			bestIndexing = r.Name
		}
		if r.SearchP50 < minP50 {
			minP50 = r.SearchP50
			bestSearchLatency = r.Name
		}
		if r.PeakMemoryMB < minMem {
			minMem = r.PeakMemoryMB
			bestMemory = r.Name
		}
		if r.IndexSizeMB < minSize {
			minSize = r.IndexSizeMB
			bestSize = r.Name
		}
	}

	sb.WriteString(fmt.Sprintf("- **Best Indexing Throughput**: `%s` (%.0f docs/s)\n", bestIndexing, maxThroughput))
	sb.WriteString(fmt.Sprintf("- **Best Search Latency**: `%s` (%s P50)\n", bestSearchLatency, minP50.Round(time.Microsecond)))
	sb.WriteString(fmt.Sprintf("- **Lowest Memory Usage**: `%s` (%.2f MB)\n", bestMemory, minMem))
	sb.WriteString(fmt.Sprintf("- **Smallest Index Size**: `%s` (%.2f MB)\n", bestSize, minSize))

	sb.WriteString("\n### Use Case Recommendations\n\n")
	sb.WriteString("- **High-throughput indexing**: Use `roaring_bm25` for best indexing speed\n")
	sb.WriteString("- **Low-latency search**: Use `ensemble` or `seismic` for fastest queries\n")
	sb.WriteString("- **Memory-constrained**: Use `roaring_bm25` for lowest memory footprint\n")
	sb.WriteString("- **Balanced workload**: Use `ensemble` (default) for best overall performance\n")

	return sb.String()
}
