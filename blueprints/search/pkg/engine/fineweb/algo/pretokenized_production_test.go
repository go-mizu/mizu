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

// TestPreTokenizedProduction tests with full 2.3M doc dataset
func TestPreTokenizedProduction(t *testing.T) {
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

	t.Logf("CPU cores: %d, Workers: %d", runtime.NumCPU(), numWorkers)
	t.Logf("Target: Full dataset (~2.3M docs)")

	// Step 1: Load and tokenize in streaming fashion
	t.Log("\n=== Step 1: Stream load and tokenize ===")

	reader := fineweb.NewParquetReader(parquetDir).WithBatchSize(10000)

	// Pre-allocate for expected size
	preDocs := make([]algo.PreTokenizedDoc, 0, 2500000)
	var mu sync.Mutex
	var wg sync.WaitGroup
	docChan := make(chan []string, 32)
	resultChan := make(chan []algo.PreTokenizedDoc, 32)

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

	// Progress tracking
	var docCount int
	lastProgress := time.Now()
	for batch, err := range reader.ReadTextsOnlyParallel(ctx, 8) {
		if err != nil {
			t.Fatal(err)
		}

		texts := make([]string, len(batch))
		for i, doc := range batch {
			texts[i] = doc.Text
		}
		docChan <- texts
		docCount += len(batch)

		if time.Since(lastProgress) > 2*time.Second {
			elapsed := time.Since(start)
			rate := float64(docCount) / elapsed.Seconds()
			t.Logf("  Progress: %d docs (%.0f docs/sec)", docCount, rate)
			lastProgress = time.Now()
		}
	}
	close(docChan)
	wg.Wait()
	close(resultChan)
	collectWg.Wait()

	loadTokenizeTime := time.Since(start)
	loadTokenizeRate := float64(len(preDocs)) / loadTokenizeTime.Seconds()
	t.Logf("Load+Tokenize complete: %d docs in %v = %.0f docs/sec",
		len(preDocs), loadTokenizeTime, loadTokenizeRate)

	// Step 2: Index from pre-tokenized
	t.Log("\n=== Step 2: Index from pre-tokenized ===")

	runtime.GC()
	indexer := algo.NewPreTokenizedIndexer(numWorkers)
	start = time.Now()
	indexer.IndexBatch(preDocs)
	indexTime := time.Since(start)
	indexRate := float64(len(preDocs)) / indexTime.Seconds()
	t.Logf("Index complete: %d docs in %v = %.0f docs/sec",
		len(preDocs), indexTime, indexRate)

	// Summary
	t.Log("\n═══════════════════════════════════════════════════════════════")
	t.Log("PRODUCTION RESULTS")
	t.Log("═══════════════════════════════════════════════════════════════")
	t.Logf("Dataset size:         %d docs", len(preDocs))
	t.Logf("Load+Tokenize:        %.0f docs/sec", loadTokenizeRate)
	t.Logf("Index (pre-tokenized): %.0f docs/sec", indexRate)
	t.Logf("")

	if indexRate >= 1000000 {
		t.Log("🎯 1M DOCS/SEC TARGET ACHIEVED!")
		t.Logf("   Exceeded target by %.1f%%", (indexRate/1000000-1)*100)
	} else {
		t.Logf("Gap to 1M: %.1fx needed", 1000000.0/indexRate)
	}
}
