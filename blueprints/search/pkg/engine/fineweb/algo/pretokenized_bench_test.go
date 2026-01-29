package algo_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb"
	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/algo"
)

// TestPreTokenizedSpeed tests indexing speed with pre-tokenized data
func TestPreTokenizedSpeed(t *testing.T) {
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

	// Step 1: Load and tokenize data (simulate creating pre-tokenized format)
	t.Log("\n=== Step 1: Load and tokenize from parquet ===")

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

	// Tokenize to create pre-tokenized docs
	preDocs := make([]algo.PreTokenizedDoc, len(allTexts))
	runtime.GC()
	start := time.Now()

	for i, text := range allTexts {
		table := algo.NewFixedHashTable(4096)
		docLen := algo.FixedTokenize(text, table)

		slots := table.UsedSlots()
		tokens := make([]uint64, 0, len(slots))
		freqs := make([]uint16, 0, len(slots))

		keys := table.Keys()
		counts := table.Counts()

		for _, slotIdx := range slots {
			tokens = append(tokens, keys[slotIdx])
			freqs = append(freqs, counts[slotIdx])
		}

		preDocs[i] = algo.PreTokenizedDoc{
			DocID:  uint32(i),
			Tokens: tokens,
			Freqs:  freqs,
			DocLen: uint16(docLen),
		}
	}
	tokenizeTime := time.Since(start)
	tokenizeRate := float64(len(allTexts)) / tokenizeTime.Seconds()
	t.Logf("Tokenization: %v = %.0f docs/sec", tokenizeTime, tokenizeRate)

	// Step 2: Index from pre-tokenized data (NO tokenization)
	t.Log("\n=== Step 2: Index from pre-tokenized (no tokenization) ===")
	runtime.GC()

	indexer := algo.NewPreTokenizedIndexer(numWorkers)
	start = time.Now()
	indexer.IndexBatch(preDocs)
	indexTime := time.Since(start)
	indexRate := float64(len(preDocs)) / indexTime.Seconds()
	t.Logf("Index only: %v = %.0f docs/sec", indexTime, indexRate)

	// Step 3: Compare with full pipeline (tokenize + index)
	t.Log("\n=== Step 3: Full pipeline (tokenize + index) ===")
	runtime.GC()

	fullIndexer := algo.NewOptimizedIndexer(numWorkers)
	start = time.Now()
	fullIndexer.IndexBatch(allTexts, 0)
	fullTime := time.Since(start)
	fullRate := float64(len(allTexts)) / fullTime.Seconds()
	t.Logf("Full pipeline: %v = %.0f docs/sec", fullTime, fullRate)

	// Summary
	t.Log("\n═══════════════════════════════════════════════════════════════")
	t.Log("SUMMARY: PRE-TOKENIZED vs FULL PIPELINE")
	t.Log("═══════════════════════════════════════════════════════════════")
	t.Logf("%-30s: %8.0f docs/sec", "Tokenization only", tokenizeRate)
	t.Logf("%-30s: %8.0f docs/sec", "Index only (pre-tokenized)", indexRate)
	t.Logf("%-30s: %8.0f docs/sec", "Full pipeline (tokenize+index)", fullRate)
	t.Logf("")
	speedup := indexRate / fullRate
	t.Logf("Speedup from pre-tokenized: %.1fx", speedup)
	t.Logf("Gap to 1M from pre-tokenized: %.1fx", 1000000.0/indexRate)
	t.Logf("Gap to 1M from full pipeline: %.1fx", 1000000.0/fullRate)

	// Estimate with I/O
	t.Log("\n═══════════════════════════════════════════════════════════════")
	t.Log("ESTIMATED PRODUCTION RATES")
	t.Log("═══════════════════════════════════════════════════════════════")
	// Assume I/O is 251k docs/sec (from phase analysis)
	ioRate := 251000.0
	t.Logf("I/O rate (parquet+zstd): %.0f docs/sec", ioRate)

	// With tokenization: min(I/O, tokenize+index)
	withTokenize := fullRate
	if ioRate < withTokenize {
		withTokenize = ioRate
	}
	t.Logf("Production with tokenization: %.0f docs/sec (bottleneck: %s)",
		withTokenize, func() string {
			if ioRate < fullRate {
				return "I/O"
			}
			return "CPU"
		}())

	// With pre-tokenized: min(I/O, index_only)
	// Pre-tokenized file I/O is much faster (simple binary format)
	preTokenizedIO := 1000000.0 // Assume 1M docs/sec for simple binary read
	withPreTokenized := indexRate
	if preTokenizedIO < withPreTokenized {
		withPreTokenized = preTokenizedIO
	}
	t.Logf("Production with pre-tokenized: %.0f docs/sec", withPreTokenized)
}

// TestPreTokenizedFileIO tests file I/O for pre-tokenized format
func TestPreTokenizedFileIO(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	// Create test data
	testDocs := make([]algo.PreTokenizedDoc, 10000)
	for i := 0; i < 10000; i++ {
		numTokens := 50 + i%100 // 50-149 tokens per doc
		tokens := make([]uint64, numTokens)
		freqs := make([]uint16, numTokens)
		for j := 0; j < numTokens; j++ {
			tokens[j] = uint64(i*1000 + j)
			freqs[j] = uint16(1 + j%10)
		}
		testDocs[i] = algo.PreTokenizedDoc{
			DocID:  uint32(i),
			Tokens: tokens,
			Freqs:  freqs,
			DocLen: uint16(numTokens),
		}
	}

	tmpFile := "/tmp/pretokenized_test.bin"
	defer os.Remove(tmpFile)

	// Write
	start := time.Now()
	if err := algo.WritePreTokenized(tmpFile, testDocs); err != nil {
		t.Fatal(err)
	}
	writeTime := time.Since(start)

	stat, _ := os.Stat(tmpFile)
	t.Logf("Wrote %d docs in %v (%.2f MB, %.0f docs/sec)",
		len(testDocs), writeTime, float64(stat.Size())/(1024*1024),
		float64(len(testDocs))/writeTime.Seconds())

	// Read
	start = time.Now()
	data, err := algo.ReadPreTokenized(tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	readTime := time.Since(start)
	t.Logf("Read %d docs in %v (%.0f docs/sec)",
		data.NumDocs, readTime, float64(data.NumDocs)/readTime.Seconds())

	// Verify
	if data.NumDocs != uint32(len(testDocs)) {
		t.Errorf("NumDocs mismatch: got %d, want %d", data.NumDocs, len(testDocs))
	}
}
