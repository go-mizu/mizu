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

// TestCGOBatchTokenization compares CGO batch tokenization with pure Go
func TestCGOBatchTokenization(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	home, _ := os.UserHomeDir()
	parquetDir := filepath.Join(home, "data", "fineweb-2", "vie_Latn", "test")

	if _, err := os.Stat(parquetDir); os.IsNotExist(err) {
		t.Skipf("Parquet directory not found: %s", parquetDir)
	}

	ctx := context.Background()
	numWorkers := runtime.NumCPU()

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

	// ═══════════════════════════════════════════════════════════════
	// TEST 1: Pure Go FixedTokenize
	// ═══════════════════════════════════════════════════════════════
	t.Log("\n=== Pure Go FixedTokenize ===")
	runtime.GC()

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
	pureGoTime := time.Since(start)
	pureGoRate := float64(len(allTexts)) / pureGoTime.Seconds()
	t.Logf("Time: %v, Rate: %.0f docs/sec", pureGoTime, pureGoRate)

	// ═══════════════════════════════════════════════════════════════
	// TEST 2: CGO Batch Tokenize
	// ═══════════════════════════════════════════════════════════════
	t.Log("\n=== CGO Batch Tokenize ===")
	runtime.GC()

	tokenizer := algo.NewCGOBatchTokenizer(1000)
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
			tokenizer.TokenizeBatch(allTexts[s:e])
		}(startIdx, endIdx)
	}
	wg.Wait()
	cgoBatchTime := time.Since(start)
	cgoBatchRate := float64(len(allTexts)) / cgoBatchTime.Seconds()
	t.Logf("Time: %v, Rate: %.0f docs/sec", cgoBatchTime, cgoBatchRate)

	// ═══════════════════════════════════════════════════════════════
	// TEST 3: CGO Batch with larger batches
	// ═══════════════════════════════════════════════════════════════
	t.Log("\n=== CGO Batch (larger batches, 5000 docs each) ===")
	runtime.GC()

	largeBatchTokenizer := algo.NewCGOBatchTokenizer(5000)
	start = time.Now()
	// Process in larger batches
	for i := 0; i < len(allTexts); i += 5000 {
		end := i + 5000
		if end > len(allTexts) {
			end = len(allTexts)
		}
		largeBatchTokenizer.TokenizeBatch(allTexts[i:end])
	}
	largeBatchTime := time.Since(start)
	largeBatchRate := float64(len(allTexts)) / largeBatchTime.Seconds()
	t.Logf("Time: %v, Rate: %.0f docs/sec", largeBatchTime, largeBatchRate)

	// Summary
	t.Log("\n═══════════════════════════════════════════════════════════════")
	t.Log("CGO BATCH TOKENIZATION COMPARISON")
	t.Log("═══════════════════════════════════════════════════════════════")
	t.Logf("%-35s: %8.0f docs/sec (baseline)", "1. Pure Go FixedTokenize", pureGoRate)
	t.Logf("%-35s: %8.0f docs/sec (%+.1f%%)", "2. CGO Batch (parallel workers)", cgoBatchRate, (cgoBatchRate/pureGoRate-1)*100)
	t.Logf("%-35s: %8.0f docs/sec (%+.1f%%)", "3. CGO Batch (5000 doc batches)", largeBatchRate, (largeBatchRate/pureGoRate-1)*100)

	t.Log("\n═══════════════════════════════════════════════════════════════")
	t.Log("GAP TO 1M DOCS/SEC")
	t.Log("═══════════════════════════════════════════════════════════════")
	t.Logf("From Pure Go:          %.1fx needed", 1000000.0/pureGoRate)
	t.Logf("From CGO Batch:        %.1fx needed", 1000000.0/cgoBatchRate)
	t.Logf("From CGO Large Batch:  %.1fx needed", 1000000.0/largeBatchRate)
}

// TestCGOBatchIndexer tests the full CGO batch indexer
func TestCGOBatchIndexer(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	home, _ := os.UserHomeDir()
	parquetDir := filepath.Join(home, "data", "fineweb-2", "vie_Latn", "test")

	if _, err := os.Stat(parquetDir); os.IsNotExist(err) {
		t.Skipf("Parquet directory not found: %s", parquetDir)
	}

	ctx := context.Background()
	numWorkers := runtime.NumCPU()

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

	// Test OptimizedIndexer (pure Go)
	t.Log("\n=== OptimizedIndexer (Pure Go) ===")
	runtime.GC()

	pureGoIndexer := algo.NewOptimizedIndexer(numWorkers)
	start := time.Now()
	pureGoIndexer.IndexBatch(allTexts, 0)
	pureGoTime := time.Since(start)
	pureGoRate := float64(len(allTexts)) / pureGoTime.Seconds()
	t.Logf("Time: %v, Rate: %.0f docs/sec", pureGoTime, pureGoRate)

	// Test CGOBatchIndexer
	t.Log("\n=== CGOBatchIndexer ===")
	runtime.GC()

	cgoIndexer := algo.NewCGOBatchIndexer(numWorkers, 1000)
	start = time.Now()
	cgoIndexer.IndexBatch(allTexts, 0)
	cgoTime := time.Since(start)
	cgoRate := float64(len(allTexts)) / cgoTime.Seconds()
	t.Logf("Time: %v, Rate: %.0f docs/sec", cgoTime, cgoRate)

	// Summary
	speedup := cgoRate / pureGoRate
	t.Logf("\nSpeedup: %.2fx", speedup)
	t.Logf("Gap to 1M from Pure Go: %.1fx", 1000000.0/pureGoRate)
	t.Logf("Gap to 1M from CGO:     %.1fx", 1000000.0/cgoRate)
}
