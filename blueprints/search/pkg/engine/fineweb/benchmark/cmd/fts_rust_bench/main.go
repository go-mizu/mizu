// fts_rust_bench benchmarks the fts_rust driver with various profiles.
//
// Usage:
//
//	fts_rust_bench -mode=index -data=/path/to/data.parquet -profile=ensemble
//	fts_rust_bench -mode=search -queries=/path/to/queries.txt -profile=ensemble
//	fts_rust_bench -mode=memory -profile=ensemble
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/drivers/fts_rust"
)

var (
	mode       = flag.String("mode", "index", "Benchmark mode: index, search, memory")
	dataPath   = flag.String("data", "", "Path to parquet data file")
	queriesPath = flag.String("queries", "", "Path to queries file (one per line)")
	profile    = flag.String("profile", "ensemble", "Search profile to use")
	outputPath = flag.String("output", "", "Output file for results (JSON)")
	indexDir   = flag.String("index", "", "Index directory (default: temp)")
	limit      = flag.Int("limit", 0, "Limit number of documents to index (0 = all)")
	searchLimit = flag.Int("search-limit", 10, "Number of results per search")
	iterations = flag.Int("iterations", 1000, "Number of search iterations")
)

func main() {
	flag.Parse()

	switch *mode {
	case "index":
		runIndexBenchmark()
	case "search":
		runSearchBenchmark()
	case "memory":
		runMemoryBenchmark()
	default:
		fmt.Fprintf(os.Stderr, "Unknown mode: %s\n", *mode)
		os.Exit(1)
	}
}

// IndexResult contains indexing benchmark results
type IndexResult struct {
	Profile         string        `json:"profile"`
	DocumentsTotal  int64         `json:"documents_total"`
	Duration        time.Duration `json:"duration_ns"`
	DurationStr     string        `json:"duration"`
	Throughput      float64       `json:"throughput_docs_per_sec"`
	PeakMemoryBytes uint64        `json:"peak_memory_bytes"`
	IndexSizeBytes  int64         `json:"index_size_bytes"`
}

func runIndexBenchmark() {
	if *dataPath == "" {
		fmt.Fprintln(os.Stderr, "Error: -data is required for index mode")
		os.Exit(1)
	}

	// Set up index directory
	idxDir := *indexDir
	if idxDir == "" {
		var err error
		idxDir, err = os.MkdirTemp("", "fts_rust_bench_*")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create temp dir: %v\n", err)
			os.Exit(1)
		}
		defer os.RemoveAll(idxDir)
	}

	fmt.Printf("Benchmarking indexing with profile: %s\n", *profile)
	fmt.Printf("Data: %s\n", *dataPath)
	fmt.Printf("Index dir: %s\n", idxDir)

	// Create driver
	cfg := fineweb.DriverConfig{
		DataDir: idxDir,
		Options: map[string]any{"profile": *profile},
	}

	driver, err := fineweb.Open("fts_rust", cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create driver: %v\n", err)
		os.Exit(1)
	}
	defer driver.Close()

	indexer, ok := fineweb.AsIndexer(driver)
	if !ok {
		fmt.Fprintln(os.Stderr, "Driver does not support indexing")
		os.Exit(1)
	}

	// Generate test documents
	docs := generateTestDocs(*limit)

	// Force GC before benchmark
	runtime.GC()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	startAlloc := m.TotalAlloc

	// Run indexing
	start := time.Now()
	var indexed int64

	err = indexer.Import(context.Background(), docs, func(done, total int64) {
		indexed = done
		if done%100000 == 0 {
			fmt.Printf("  Indexed %d documents...\n", done)
		}
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Import failed: %v\n", err)
		os.Exit(1)
	}

	duration := time.Since(start)

	runtime.ReadMemStats(&m)
	peakMem := m.TotalAlloc - startAlloc

	// Calculate index size
	var indexSize int64
	filepath.Walk(idxDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			indexSize += info.Size()
		}
		return nil
	})

	result := IndexResult{
		Profile:         *profile,
		DocumentsTotal:  indexed,
		Duration:        duration,
		DurationStr:     duration.String(),
		Throughput:      float64(indexed) / duration.Seconds(),
		PeakMemoryBytes: peakMem,
		IndexSizeBytes:  indexSize,
	}

	fmt.Printf("\n=== Index Benchmark Results ===\n")
	fmt.Printf("Profile:       %s\n", result.Profile)
	fmt.Printf("Documents:     %d\n", result.DocumentsTotal)
	fmt.Printf("Duration:      %s\n", result.DurationStr)
	fmt.Printf("Throughput:    %.0f docs/sec\n", result.Throughput)
	fmt.Printf("Peak Memory:   %.2f MB\n", float64(result.PeakMemoryBytes)/1024/1024)
	fmt.Printf("Index Size:    %.2f MB\n", float64(result.IndexSizeBytes)/1024/1024)

	if *outputPath != "" {
		writeJSON(*outputPath, result)
	}
}

// SearchResult contains search benchmark results
type SearchResult struct {
	Profile      string        `json:"profile"`
	Queries      int           `json:"queries"`
	TotalHits    int64         `json:"total_hits"`
	Duration     time.Duration `json:"duration_ns"`
	DurationStr  string        `json:"duration"`
	AvgLatencyNs int64         `json:"avg_latency_ns"`
	P50LatencyNs int64         `json:"p50_latency_ns"`
	P95LatencyNs int64         `json:"p95_latency_ns"`
	P99LatencyNs int64         `json:"p99_latency_ns"`
	QPS          float64       `json:"qps"`
}

func runSearchBenchmark() {
	idxDir := *indexDir
	if idxDir == "" {
		fmt.Fprintln(os.Stderr, "Error: -index is required for search mode")
		os.Exit(1)
	}

	// Load queries
	queries := loadQueries()
	if len(queries) == 0 {
		// Generate random queries
		queries = generateTestQueries(100)
	}

	fmt.Printf("Benchmarking search with profile: %s\n", *profile)
	fmt.Printf("Index dir: %s\n", idxDir)
	fmt.Printf("Queries: %d\n", len(queries))
	fmt.Printf("Iterations: %d\n", *iterations)

	// Open driver
	cfg := fineweb.DriverConfig{
		DataDir: idxDir,
		Options: map[string]any{"profile": *profile},
	}

	driver, err := fineweb.Open("fts_rust", cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open driver: %v\n", err)
		os.Exit(1)
	}
	defer driver.Close()

	// Warmup
	fmt.Println("Warming up...")
	for i := 0; i < 10; i++ {
		driver.Search(context.Background(), queries[i%len(queries)], *searchLimit, 0)
	}

	// Run benchmark
	latencies := make([]time.Duration, *iterations)
	var totalHits int64

	start := time.Now()
	for i := 0; i < *iterations; i++ {
		query := queries[i%len(queries)]
		qStart := time.Now()
		result, err := driver.Search(context.Background(), query, *searchLimit, 0)
		latencies[i] = time.Since(qStart)
		if err == nil {
			totalHits += int64(len(result.Documents))
		}
	}
	duration := time.Since(start)

	// Sort latencies for percentiles
	sortDurations(latencies)

	result := SearchResult{
		Profile:      *profile,
		Queries:      *iterations,
		TotalHits:    totalHits,
		Duration:     duration,
		DurationStr:  duration.String(),
		AvgLatencyNs: int64(duration.Nanoseconds() / int64(*iterations)),
		P50LatencyNs: int64(latencies[len(latencies)/2].Nanoseconds()),
		P95LatencyNs: int64(latencies[int(float64(len(latencies))*0.95)].Nanoseconds()),
		P99LatencyNs: int64(latencies[int(float64(len(latencies))*0.99)].Nanoseconds()),
		QPS:          float64(*iterations) / duration.Seconds(),
	}

	fmt.Printf("\n=== Search Benchmark Results ===\n")
	fmt.Printf("Profile:       %s\n", result.Profile)
	fmt.Printf("Queries:       %d\n", result.Queries)
	fmt.Printf("Total Hits:    %d\n", result.TotalHits)
	fmt.Printf("Duration:      %s\n", result.DurationStr)
	fmt.Printf("Avg Latency:   %s\n", time.Duration(result.AvgLatencyNs))
	fmt.Printf("P50 Latency:   %s\n", time.Duration(result.P50LatencyNs))
	fmt.Printf("P95 Latency:   %s\n", time.Duration(result.P95LatencyNs))
	fmt.Printf("P99 Latency:   %s\n", time.Duration(result.P99LatencyNs))
	fmt.Printf("QPS:           %.0f\n", result.QPS)

	if *outputPath != "" {
		writeJSON(*outputPath, result)
	}
}

// MemoryResult contains memory benchmark results
type MemoryResult struct {
	Profile         string `json:"profile"`
	DocsIndexed     uint64 `json:"docs_indexed"`
	IndexBytes      uint64 `json:"index_bytes"`
	TermDictBytes   uint64 `json:"term_dict_bytes"`
	PostingsBytes   uint64 `json:"postings_bytes"`
	MmapBytes       uint64 `json:"mmap_bytes"`
	HeapBytes       uint64 `json:"heap_bytes"`
	GoHeapBytes     uint64 `json:"go_heap_bytes"`
	GoHeapAllocated uint64 `json:"go_heap_allocated"`
}

func runMemoryBenchmark() {
	idxDir := *indexDir
	if idxDir == "" {
		fmt.Fprintln(os.Stderr, "Error: -index is required for memory mode")
		os.Exit(1)
	}

	fmt.Printf("Memory benchmark for profile: %s\n", *profile)
	fmt.Printf("Index dir: %s\n", idxDir)

	cfg := fineweb.DriverConfig{
		DataDir: idxDir,
		Options: map[string]any{"profile": *profile},
	}

	driver, err := fineweb.Open("fts_rust", cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open driver: %v\n", err)
		os.Exit(1)
	}
	defer driver.Close()

	// Get Go memory stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Get driver memory stats (if available)
	var indexBytes, termDictBytes, postingsBytes, mmapBytes, docsIndexed uint64
	if ms, ok := driver.(interface {
		MemoryStats() interface {
			IndexBytes() uint64
			TermDictBytes() uint64
			PostingsBytes() uint64
			MmapBytes() uint64
			DocsIndexed() uint64
		}
	}); ok {
		stats := ms.MemoryStats()
		indexBytes = stats.IndexBytes()
		termDictBytes = stats.TermDictBytes()
		postingsBytes = stats.PostingsBytes()
		mmapBytes = stats.MmapBytes()
		docsIndexed = stats.DocsIndexed()
	}

	result := MemoryResult{
		Profile:         *profile,
		DocsIndexed:     docsIndexed,
		IndexBytes:      indexBytes,
		TermDictBytes:   termDictBytes,
		PostingsBytes:   postingsBytes,
		MmapBytes:       mmapBytes,
		HeapBytes:       indexBytes - mmapBytes,
		GoHeapBytes:     m.HeapInuse,
		GoHeapAllocated: m.HeapAlloc,
	}

	fmt.Printf("\n=== Memory Benchmark Results ===\n")
	fmt.Printf("Profile:         %s\n", result.Profile)
	fmt.Printf("Docs Indexed:    %d\n", result.DocsIndexed)
	fmt.Printf("Index Size:      %.2f MB\n", float64(result.IndexBytes)/1024/1024)
	fmt.Printf("Term Dict:       %.2f MB\n", float64(result.TermDictBytes)/1024/1024)
	fmt.Printf("Postings:        %.2f MB\n", float64(result.PostingsBytes)/1024/1024)
	fmt.Printf("Mmap:            %.2f MB\n", float64(result.MmapBytes)/1024/1024)
	fmt.Printf("Heap (Rust):     %.2f MB\n", float64(result.HeapBytes)/1024/1024)
	fmt.Printf("Heap (Go):       %.2f MB\n", float64(result.GoHeapBytes)/1024/1024)

	if *outputPath != "" {
		writeJSON(*outputPath, result)
	}
}

// generateTestDocs generates test documents
func generateTestDocs(count int) iter.Seq2[fineweb.Document, error] {
	if count <= 0 {
		count = 100000 // Default 100k docs
	}

	return func(yield func(fineweb.Document, error) bool) {
		words := []string{
			"the", "quick", "brown", "fox", "jumps", "over", "lazy", "dog",
			"machine", "learning", "artificial", "intelligence", "data", "science",
			"programming", "language", "computer", "system", "network", "database",
		}

		for i := 0; i < count; i++ {
			// Generate random text
			text := ""
			for j := 0; j < 50; j++ {
				text += words[(i+j)%len(words)] + " "
			}

			doc := fineweb.Document{
				ID:   fmt.Sprintf("doc_%d", i),
				Text: text,
			}

			if !yield(doc, nil) {
				return
			}
		}
	}
}

// generateTestQueries generates test queries
func generateTestQueries(count int) []string {
	queries := []string{
		"machine learning",
		"quick brown fox",
		"programming language",
		"data science",
		"computer network",
		"artificial intelligence",
		"database system",
		"lazy dog",
	}

	result := make([]string, count)
	for i := 0; i < count; i++ {
		result[i] = queries[i%len(queries)]
	}
	return result
}

// loadQueries loads queries from file
func loadQueries() []string {
	if *queriesPath == "" {
		return nil
	}

	file, err := os.Open(*queriesPath)
	if err != nil {
		return nil
	}
	defer file.Close()

	var queries []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if q := scanner.Text(); q != "" {
			queries = append(queries, q)
		}
	}
	return queries
}

// sortDurations sorts a slice of durations
func sortDurations(d []time.Duration) {
	for i := 0; i < len(d); i++ {
		for j := i + 1; j < len(d); j++ {
			if d[i] > d[j] {
				d[i], d[j] = d[j], d[i]
			}
		}
	}
}

// writeJSON writes result to JSON file
func writeJSON(path string, v any) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal JSON: %v\n", err)
		return
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write file: %v\n", err)
	}
	fmt.Printf("\nResults written to: %s\n", path)
}
