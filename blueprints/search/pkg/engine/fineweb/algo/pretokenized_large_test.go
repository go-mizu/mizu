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

// TestPreTokenizedLargeScale tests pre-tokenized indexing with full dataset
func TestPreTokenizedLargeScale(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	home, _ := os.UserHomeDir()
	parquetDir := filepath.Join(home, "data", "fineweb-2", "vie_Latn", "train")

	if _, err := os.Stat(parquetDir); os.IsNotExist(err) {
		t.Skipf("Parquet directory not found: %s", parquetDir)
	}

	ctx := context.Background()
	numWorkers := runtime.NumCPU() * 5
	targetDocs := 500000

	t.Logf("CPU cores: %d, Workers: %d, Target: %d docs", runtime.NumCPU(), numWorkers, targetDocs)

	// Step 1: Load and tokenize from parquet (parallel)
	t.Log("\n=== Step 1: Load from parquet and tokenize in parallel ===")

	reader := fineweb.NewParquetReader(parquetDir).WithBatchSize(10000)
	preDocs := make([]algo.PreTokenizedDoc, 0, targetDocs)
	var mu sync.Mutex
	var wg sync.WaitGroup
	docChan := make(chan []string, 16)
	resultChan := make(chan []algo.PreTokenizedDoc, 16)

	runtime.GC()
	start := time.Now()

	// Tokenization workers
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			table := algo.NewFixedHashTable(4096)

			for batch := range docChan {
				results := make([]algo.PreTokenizedDoc, len(batch))
				for i, text := range batch {
					docLen := algo.FixedTokenize(text, table)

					slots := table.UsedSlots()
					tokens := make([]uint64, len(slots))
					freqs := make([]uint16, len(slots))

					keys := table.Keys()
					counts := table.Counts()

					for j, slotIdx := range slots {
						tokens[j] = keys[slotIdx]
						freqs[j] = counts[slotIdx]
					}

					results[i] = algo.PreTokenizedDoc{
						Tokens: tokens,
						Freqs:  freqs,
						DocLen: uint16(docLen),
					}
				}
				resultChan <- results
			}
		}()
	}

	// Collector
	var collectWg sync.WaitGroup
	collectWg.Add(1)
	go func() {
		defer collectWg.Done()
		docID := uint32(0)
		for batch := range resultChan {
			for i := range batch {
				batch[i].DocID = docID
				docID++
			}
			mu.Lock()
			preDocs = append(preDocs, batch...)
			mu.Unlock()
		}
	}()

	// Read parquet and send to workers
	var docCount int
	for batch, err := range reader.ReadTextsOnlyParallel(ctx, 8) {
		if err != nil {
			t.Fatal(err)
		}

		texts := make([]string, 0, len(batch))
		for _, doc := range batch {
			texts = append(texts, doc.Text)
			docCount++
			if docCount >= targetDocs {
				break
			}
		}
		docChan <- texts

		if docCount >= targetDocs {
			break
		}
	}
	close(docChan)
	wg.Wait()
	close(resultChan)
	collectWg.Wait()

	loadTokenizeTime := time.Since(start)
	loadTokenizeRate := float64(len(preDocs)) / loadTokenizeTime.Seconds()
	t.Logf("Load+Tokenize: %d docs in %v = %.0f docs/sec",
		len(preDocs), loadTokenizeTime, loadTokenizeRate)

	// Step 2: Index from pre-tokenized (multiple runs for accuracy)
	t.Log("\n=== Step 2: Index from pre-tokenized (no tokenization) ===")

	var bestIndexRate float64
	for run := 0; run < 3; run++ {
		runtime.GC()
		indexer := algo.NewPreTokenizedIndexer(numWorkers)
		start = time.Now()
		indexer.IndexBatch(preDocs)
		indexTime := time.Since(start)
		indexRate := float64(len(preDocs)) / indexTime.Seconds()
		if indexRate > bestIndexRate {
			bestIndexRate = indexRate
		}
		t.Logf("  Run %d: %v = %.0f docs/sec", run+1, indexTime, indexRate)
	}

	// Step 3: Full pipeline comparison
	t.Log("\n=== Step 3: Full pipeline (tokenize + index) ===")

	allTexts := make([]string, len(preDocs))
	for i := 0; i < len(allTexts); i++ {
		// Reconstruct text would be complex, so we'll reuse loaded data
		// For fair comparison, we measure full pipeline on same docs
	}

	// Since we can't easily reconstruct texts, let's reload a subset
	reader2 := fineweb.NewParquetReader(parquetDir).WithBatchSize(10000)
	allTexts = allTexts[:0]
	for batch, err := range reader2.ReadTextsOnlyParallel(ctx, 8) {
		if err != nil {
			t.Fatal(err)
		}
		for _, doc := range batch {
			allTexts = append(allTexts, doc.Text)
			if len(allTexts) >= len(preDocs) {
				break
			}
		}
		if len(allTexts) >= len(preDocs) {
			break
		}
	}

	runtime.GC()
	fullIndexer := algo.NewOptimizedIndexer(numWorkers)
	start = time.Now()
	fullIndexer.IndexBatch(allTexts, 0)
	fullTime := time.Since(start)
	fullRate := float64(len(allTexts)) / fullTime.Seconds()
	t.Logf("Full pipeline: %d docs in %v = %.0f docs/sec", len(allTexts), fullTime, fullRate)

	// Summary
	t.Log("\n═══════════════════════════════════════════════════════════════")
	t.Log("LARGE SCALE RESULTS")
	t.Log("═══════════════════════════════════════════════════════════════")
	t.Logf("%-35s: %8.0f docs/sec", "Load+Tokenize (from parquet)", loadTokenizeRate)
	t.Logf("%-35s: %8.0f docs/sec", "Index only (pre-tokenized, best)", bestIndexRate)
	t.Logf("%-35s: %8.0f docs/sec", "Full pipeline (tokenize+index)", fullRate)
	t.Logf("")
	t.Logf("Speedup from pre-tokenized: %.1fx", bestIndexRate/fullRate)
	t.Logf("")
	t.Logf("%-35s: %.1fx", "Gap to 1M from pre-tokenized", 1000000.0/bestIndexRate)
	t.Logf("%-35s: %.1fx", "Gap to 1M from full pipeline", 1000000.0/fullRate)

	if bestIndexRate >= 1000000 {
		t.Log("\n🎯 1M DOCS/SEC TARGET ACHIEVED with pre-tokenized data!")
	}
}
