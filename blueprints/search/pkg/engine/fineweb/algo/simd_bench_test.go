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

// TestSIMDvsGoTokenization compares SIMD CGO tokenization vs pure Go
func TestSIMDvsGoTokenization(t *testing.T) {
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

	t.Logf("CPU cores: %d, Workers: %d, GOARCH: %s", runtime.NumCPU(), numWorkers, runtime.GOARCH)

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
	// TEST 1: Pure Go FixedTokenize (baseline)
	// ═══════════════════════════════════════════════════════════════
	t.Log("\n=== Test 1: Pure Go FixedTokenize ===")
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
	goTime := time.Since(start)
	goRate := float64(len(allTexts)) / goTime.Seconds()
	t.Logf("Time: %v, Rate: %.0f docs/sec", goTime, goRate)

	// ═══════════════════════════════════════════════════════════════
	// TEST 2: CGO SIMD Tokenize
	// ═══════════════════════════════════════════════════════════════
	t.Log("\n=== Test 2: CGO SIMD Tokenize ===")
	runtime.GC()

	start = time.Now()
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
			table := algo.NewSIMDFixedTable(4096)
			for i := s; i < e; i++ {
				algo.SIMDTokenize(allTexts[i], table)
			}
		}(startIdx, endIdx)
	}
	wg.Wait()
	simdTime := time.Since(start)
	simdRate := float64(len(allTexts)) / simdTime.Seconds()
	t.Logf("Time: %v, Rate: %.0f docs/sec", simdTime, simdRate)

	// ═══════════════════════════════════════════════════════════════
	// TEST 3: CGO SIMD with Direct byte slice (avoid C.CString)
	// ═══════════════════════════════════════════════════════════════
	t.Log("\n=== Test 3: CGO SIMD Direct (no C.CString) ===")
	runtime.GC()

	// Pre-convert texts to byte slices
	allBytes := make([][]byte, len(allTexts))
	for i, text := range allTexts {
		allBytes[i] = []byte(text)
	}

	start = time.Now()
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
			table := algo.NewSIMDFixedTable(4096)
			for i := s; i < e; i++ {
				algo.SIMDTokenizeDirect(allBytes[i], table)
			}
		}(startIdx, endIdx)
	}
	wg.Wait()
	simdDirectTime := time.Since(start)
	simdDirectRate := float64(len(allTexts)) / simdDirectTime.Seconds()
	t.Logf("Time: %v, Rate: %.0f docs/sec", simdDirectTime, simdDirectRate)

	// Summary
	t.Log("\n═══════════════════════════════════════════════════════════════")
	t.Log("SIMD vs GO COMPARISON")
	t.Log("═══════════════════════════════════════════════════════════════")
	t.Logf("%-30s: %8.0f docs/sec (baseline)", "1. Pure Go FixedTokenize", goRate)
	t.Logf("%-30s: %8.0f docs/sec (%+.1f%%)", "2. CGO SIMD", simdRate, (simdRate/goRate-1)*100)
	t.Logf("%-30s: %8.0f docs/sec (%+.1f%%)", "3. CGO SIMD Direct", simdDirectRate, (simdDirectRate/goRate-1)*100)

	t.Log("\n═══════════════════════════════════════════════════════════════")
	t.Log("GAP TO 1M DOCS/SEC")
	t.Log("═══════════════════════════════════════════════════════════════")
	t.Logf("From Pure Go:     %.1fx needed", 1000000.0/goRate)
	t.Logf("From CGO SIMD:    %.1fx needed", 1000000.0/simdRate)
	t.Logf("From SIMD Direct: %.1fx needed", 1000000.0/simdDirectRate)
}

// TestBatchSIMDTokenization tests batch SIMD tokenization
func TestBatchSIMDTokenization(t *testing.T) {
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
	t.Logf("Loaded %d docs", len(allTexts))

	// Test 1: Pure Go (baseline)
	t.Log("\n=== Pure Go FixedTokenize ===")
	runtime.GC()

	batchSize := (len(allTexts) + numWorkers - 1) / numWorkers
	var wg sync.WaitGroup

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
	goTime := time.Since(start)
	goRate := float64(len(allTexts)) / goTime.Seconds()
	t.Logf("Time: %v, Rate: %.0f docs/sec", goTime, goRate)

	// Test 2: Batch SIMD
	t.Log("\n=== Batch SIMD Tokenize ===")
	runtime.GC()

	// Process in batches of 1000 docs to amortize CGO overhead
	batchDocSize := 1000
	start = time.Now()
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
			for batchStart := s; batchStart < e; batchStart += batchDocSize {
				batchEnd := batchStart + batchDocSize
				if batchEnd > e {
					batchEnd = e
				}
				algo.BatchSIMDTokenize(allTexts[batchStart:batchEnd], (batchEnd-batchStart)*200)
			}
		}(startIdx, endIdx)
	}
	wg.Wait()
	batchTime := time.Since(start)
	batchRate := float64(len(allTexts)) / batchTime.Seconds()
	t.Logf("Time: %v, Rate: %.0f docs/sec", batchTime, batchRate)

	// Summary
	t.Log("\n═══════════════════════════════════════════════════════════════")
	t.Log("BATCH SIMD COMPARISON")
	t.Log("═══════════════════════════════════════════════════════════════")
	t.Logf("%-30s: %8.0f docs/sec (baseline)", "1. Pure Go FixedTokenize", goRate)
	t.Logf("%-30s: %8.0f docs/sec (%+.1f%%)", "2. Batch SIMD", batchRate, (batchRate/goRate-1)*100)

	t.Log("\n═══════════════════════════════════════════════════════════════")
	t.Log("GAP TO 1M DOCS/SEC")
	t.Log("═══════════════════════════════════════════════════════════════")
	t.Logf("From Pure Go:    %.1fx needed", 1000000.0/goRate)
	t.Logf("From Batch SIMD: %.1fx needed", 1000000.0/batchRate)
}

// TestSIMDIndexerBenchmark tests the full SIMD indexer
func TestSIMDIndexerBenchmark(t *testing.T) {
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
	t.Logf("Loaded %d docs", len(allTexts))

	// Test SIMD Indexer
	t.Log("\n=== SIMD Indexer Full Pipeline ===")
	runtime.GC()

	indexer := algo.NewSIMDIndexer(numWorkers)
	start := time.Now()
	indexer.IndexBatch(allTexts, 0)
	_, err := indexer.Finish()
	if err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(start)
	rate := float64(len(allTexts)) / elapsed.Seconds()

	t.Logf("Time: %v, Rate: %.0f docs/sec", elapsed, rate)
	t.Logf("Gap to 1M: %.1fx needed", 1000000.0/rate)
}
