package algo_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb"
	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/algo"
)

// TestOptimizedPipeline compares the optimized pipeline with baseline
func TestOptimizedPipeline(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	home, _ := os.UserHomeDir()
	parquetDir := filepath.Join(home, "data", "fineweb-2", "vie_Latn", "test")

	if _, err := os.Stat(parquetDir); os.IsNotExist(err) {
		t.Skipf("Parquet directory not found: %s", parquetDir)
	}

	ctx := context.Background()
	numWorkers := runtime.NumCPU() * 5

	t.Logf("CPU cores: %d, Workers: %d", runtime.NumCPU(), numWorkers)

	// Load data
	var allTexts []string
	reader := fineweb.NewParquetReader(parquetDir).WithBatchSize(10000)

	for batch, err := range reader.ReadTextsOnlyParallel(ctx, 8) {
		if err != nil {
			t.Fatal(err)
		}
		for _, doc := range batch {
			allTexts = append(allTexts, doc.Text)
		}
	}
	t.Logf("Loaded %d docs\n", len(allTexts))

	batchSize := (len(allTexts) + numWorkers - 1) / numWorkers
	var wg sync.WaitGroup

	// ═══════════════════════════════════════════════════════════════
	// TEST 1: Pure Go FixedTokenize (individual docs)
	// ═══════════════════════════════════════════════════════════════
	t.Log("\n=== Test 1: Pure Go FixedTokenize (individual) ===")
	runtime.GC()

	start := time.Now()
	for w := 0; w < numWorkers; w++ {
		startIdx := w * batchSize
		endIdx := startIdx + batchSize
		if endIdx > len(allTexts) {
			endIdx = len(allTexts)
		}
		if startIdx >= endIdx {
			break
		}
		wg.Add(1)
		go func(s, e int) {
			defer wg.Done()
			table := algo.NewFixedHashTable(4096)
			for i := s; i < e; i++ {
				algo.FixedTokenize(allTexts[i], table)
			}
		}(startIdx, endIdx)
	}
	wg.Wait()
	individualTime := time.Since(start)
	individualRate := float64(len(allTexts)) / individualTime.Seconds()
	t.Logf("Time: %v, Rate: %.0f docs/sec", individualTime, individualRate)

	// ═══════════════════════════════════════════════════════════════
	// TEST 2: OptimizedIndexer (pre-allocated, batch processing)
	// ═══════════════════════════════════════════════════════════════
	t.Log("\n=== Test 2: OptimizedIndexer (pre-allocated) ===")
	runtime.GC()

	indexer := algo.NewOptimizedIndexer(numWorkers)
	start = time.Now()
	indexer.IndexBatch(allTexts, 0)
	optimizedTime := time.Since(start)
	optimizedRate := float64(len(allTexts)) / optimizedTime.Seconds()
	t.Logf("Time: %v, Rate: %.0f docs/sec", optimizedTime, optimizedRate)

	// ═══════════════════════════════════════════════════════════════
	// TEST 3: StreamingOptimizedIndexer
	// ═══════════════════════════════════════════════════════════════
	t.Log("\n=== Test 3: StreamingOptimizedIndexer ===")
	runtime.GC()

	streamIndexer := algo.NewStreamingOptimizedIndexer(numWorkers)
	start = time.Now()
	for _, text := range allTexts {
		streamIndexer.Add(text)
	}
	streamIndexer.Flush()
	streamingTime := time.Since(start)
	streamingRate := float64(len(allTexts)) / streamingTime.Seconds()
	t.Logf("Time: %v, Rate: %.0f docs/sec", streamingTime, streamingRate)

	// ═══════════════════════════════════════════════════════════════
	// TEST 4: Full Pipeline (Tokenize + Shard + Accumulate)
	// ═══════════════════════════════════════════════════════════════
	t.Log("\n=== Test 4: Full Pipeline (with index build) ===")
	runtime.GC()

	fullIndexer := algo.NewOptimizedIndexer(numWorkers)
	start = time.Now()
	fullIndexer.IndexBatch(allTexts, 0)
	_, err := fullIndexer.Finish()
	if err != nil {
		t.Fatal(err)
	}
	fullTime := time.Since(start)
	fullRate := float64(len(allTexts)) / fullTime.Seconds()
	t.Logf("Time: %v, Rate: %.0f docs/sec", fullTime, fullRate)

	// Summary
	t.Log("\n═══════════════════════════════════════════════════════════════")
	t.Log("OPTIMIZED PIPELINE COMPARISON")
	t.Log("═══════════════════════════════════════════════════════════════")
	t.Logf("%-40s: %8.0f docs/sec (baseline)", "1. Individual FixedTokenize", individualRate)
	t.Logf("%-40s: %8.0f docs/sec (%+.1f%%)", "2. OptimizedIndexer", optimizedRate, (optimizedRate/individualRate-1)*100)
	t.Logf("%-40s: %8.0f docs/sec (%+.1f%%)", "3. StreamingOptimizedIndexer", streamingRate, (streamingRate/individualRate-1)*100)
	t.Logf("%-40s: %8.0f docs/sec (%+.1f%%)", "4. Full Pipeline (with build)", fullRate, (fullRate/individualRate-1)*100)

	t.Log("\n═══════════════════════════════════════════════════════════════")
	t.Log("GAP TO 1M DOCS/SEC")
	t.Log("═══════════════════════════════════════════════════════════════")
	t.Logf("From Individual:    %.1fx needed", 1000000.0/individualRate)
	t.Logf("From Optimized:     %.1fx needed", 1000000.0/optimizedRate)
	t.Logf("From Full Pipeline: %.1fx needed", 1000000.0/fullRate)
}

// TestIOBoundWithOptimized tests I/O bound performance with optimized indexer
func TestIOBoundWithOptimized(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	home, _ := os.UserHomeDir()
	parquetDir := filepath.Join(home, "data", "fineweb-2", "vie_Latn", "test")

	if _, err := os.Stat(parquetDir); os.IsNotExist(err) {
		t.Skipf("Parquet directory not found: %s", parquetDir)
	}

	ctx := context.Background()
	numWorkers := runtime.NumCPU() * 5

	t.Logf("CPU cores: %d, Workers: %d", runtime.NumCPU(), numWorkers)

	// Test I/O + indexing together
	t.Log("\n=== I/O Bound with OptimizedIndexer ===")
	runtime.GC()

	reader := fineweb.NewParquetReader(parquetDir).WithBatchSize(10000)
	indexer := algo.NewStreamingOptimizedIndexer(numWorkers)

	start := time.Now()
	var docCount int

	for batch, err := range reader.ReadTextsOnlyParallel(ctx, 8) {
		if err != nil {
			t.Fatal(err)
		}
		for _, doc := range batch {
			indexer.Add(doc.Text)
			docCount++
		}
	}
	indexer.Flush()
	elapsed := time.Since(start)
	rate := float64(docCount) / elapsed.Seconds()

	t.Logf("Docs: %d, Time: %v, Rate: %.0f docs/sec", docCount, elapsed, rate)
	t.Logf("Gap to 1M: %.1fx needed", 1000000.0/rate)
}
