package fts_rust

import (
	"context"
	"fmt"
	"iter"
	"os"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb"
)

func TestBasicOperations(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fts_rust_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	for _, profile := range []string{"turbo", "ensemble"} {
		t.Run(profile, func(t *testing.T) {
			cfg := fineweb.DriverConfig{
				DataDir: tmpDir + "/" + profile,
				Options: map[string]any{"profile": profile},
			}

			driver, err := New(cfg)
			if err != nil {
				t.Fatalf("Failed to create driver: %v", err)
			}
			defer driver.Close()

			// Create test documents
			docs := createTestDocs(100)

			// Import
			indexer, ok := fineweb.AsIndexer(driver)
			if !ok {
				t.Fatal("Driver does not support indexing")
			}

			err = indexer.Import(context.Background(), docs, nil)
			if err != nil {
				t.Fatalf("Import failed: %v", err)
			}

			// Search
			result, err := driver.Search(context.Background(), "programming", 10, 0)
			if err != nil {
				t.Fatalf("Search failed: %v", err)
			}

			if len(result.Documents) == 0 {
				t.Log("Warning: No results found")
			}

			t.Logf("Profile %s: Found %d documents", profile, len(result.Documents))
		})
	}
}

func TestThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping throughput test in short mode")
	}

	tmpDir, err := os.MkdirTemp("", "fts_rust_throughput_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	for _, profile := range []string{"turbo", "ensemble", "tantivy"} {
		t.Run(profile, func(t *testing.T) {
			cfg := fineweb.DriverConfig{
				DataDir: tmpDir + "/" + profile,
				Options: map[string]any{"profile": profile},
			}

			driver, err := New(cfg)
			if err != nil {
				t.Fatalf("Failed to create driver: %v", err)
			}
			defer driver.Close()

			indexer, ok := fineweb.AsIndexer(driver)
			if !ok {
				t.Fatal("Driver does not support indexing")
			}

			// Create larger test dataset
			docCount := 100000
			docs := createTestDocs(docCount)

			// Measure indexing throughput
			start := time.Now()
			err = indexer.Import(context.Background(), docs, nil)
			if err != nil {
				t.Fatalf("Import failed: %v", err)
			}
			duration := time.Since(start)

			throughput := float64(docCount) / duration.Seconds()
			t.Logf("Profile %s: %d docs in %v (%.0f docs/sec)", profile, docCount, duration, throughput)

			// Verify search works
			result, err := driver.Search(context.Background(), "programming", 10, 0)
			if err != nil {
				t.Fatalf("Search failed: %v", err)
			}
			t.Logf("Search returned %d results", len(result.Documents))
		})
	}
}

func TestLargeScaleThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large scale test in short mode")
	}

	tmpDir, err := os.MkdirTemp("", "fts_rust_large_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test with 1M documents for 1M docs/sec target
	docCount := 1000000

	for _, profile := range []string{"ultra", "turbo", "ensemble"} {
		t.Run(profile, func(t *testing.T) {
			cfg := fineweb.DriverConfig{
				DataDir: tmpDir + "/" + profile,
				Options: map[string]any{"profile": profile},
			}

			driver, err := New(cfg)
			if err != nil {
				t.Fatalf("Failed to create driver: %v", err)
			}
			defer driver.Close()

			indexer, ok := fineweb.AsIndexer(driver)
			if !ok {
				t.Fatal("Driver does not support indexing")
			}

			docs := createTestDocs(docCount)

			// Measure indexing throughput
			start := time.Now()
			var lastReport time.Time
			progress := func(done, total int64) {
				if time.Since(lastReport) > 2*time.Second {
					elapsed := time.Since(start)
					rate := float64(done) / elapsed.Seconds()
					t.Logf("Progress: %d/%d docs (%.0f docs/sec)", done, docCount, rate)
					lastReport = time.Now()
				}
			}

			err = indexer.Import(context.Background(), docs, progress)
			if err != nil {
				t.Fatalf("Import failed: %v", err)
			}
			duration := time.Since(start)

			throughput := float64(docCount) / duration.Seconds()
			t.Logf("Profile %s: %d docs in %v (%.0f docs/sec)", profile, docCount, duration, throughput)

			// Memory stats
			stats := driver.MemoryStats()
			t.Logf("Memory: index=%.2f MB, heap=%.2f MB",
				float64(stats.IndexBytes)/1024/1024,
				float64(stats.HeapBytes())/1024/1024)
		})
	}
}

func BenchmarkIndexing(b *testing.B) {
	for _, profile := range []string{"turbo", "ensemble"} {
		b.Run(profile, func(b *testing.B) {
			tmpDir, err := os.MkdirTemp("", "fts_rust_bench_*")
			if err != nil {
				b.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			cfg := fineweb.DriverConfig{
				DataDir: tmpDir,
				Options: map[string]any{"profile": profile},
			}

			driver, err := New(cfg)
			if err != nil {
				b.Fatalf("Failed to create driver: %v", err)
			}
			defer driver.Close()

			indexer, ok := fineweb.AsIndexer(driver)
			if !ok {
				b.Fatal("Driver does not support indexing")
			}

			// Prepare documents
			docs := createTestDocs(b.N)

			b.ResetTimer()
			err = indexer.Import(context.Background(), docs, nil)
			if err != nil {
				b.Fatalf("Import failed: %v", err)
			}
			b.StopTimer()

			b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "docs/sec")
		})
	}
}

func BenchmarkSearch(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "fts_rust_search_bench_*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	for _, profile := range []string{"turbo", "ensemble"} {
		b.Run(profile, func(b *testing.B) {
			cfg := fineweb.DriverConfig{
				DataDir: tmpDir + "/" + profile,
				Options: map[string]any{"profile": profile},
			}

			driver, err := New(cfg)
			if err != nil {
				b.Fatalf("Failed to create driver: %v", err)
			}
			defer driver.Close()

			indexer, ok := fineweb.AsIndexer(driver)
			if !ok {
				b.Fatal("Driver does not support indexing")
			}

			// Index test data
			docs := createTestDocs(10000)
			err = indexer.Import(context.Background(), docs, nil)
			if err != nil {
				b.Fatalf("Import failed: %v", err)
			}

			queries := []string{
				"programming",
				"machine learning",
				"database system",
				"rust language",
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				query := queries[i%len(queries)]
				_, err := driver.Search(context.Background(), query, 10, 0)
				if err != nil {
					b.Fatalf("Search failed: %v", err)
				}
			}
		})
	}
}

func createTestDocs(count int) iter.Seq2[fineweb.Document, error] {
	return func(yield func(fineweb.Document, error) bool) {
		words := []string{
			"the", "quick", "brown", "fox", "jumps", "over", "lazy", "dog",
			"machine", "learning", "artificial", "intelligence", "data", "science",
			"programming", "language", "computer", "system", "network", "database",
			"rust", "go", "python", "java", "javascript", "typescript", "swift",
			"algorithm", "optimization", "performance", "throughput", "latency",
		}

		for i := 0; i < count; i++ {
			// Generate random text using words
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
