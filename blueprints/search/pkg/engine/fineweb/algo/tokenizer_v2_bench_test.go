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

// TestTokenizerVersions compares different tokenizer implementations
func TestTokenizerVersions(t *testing.T) {
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

	type tokenizer struct {
		name string
		fn   func(string, *algo.FixedHashTable) int
	}

	tokenizers := []tokenizer{
		{"V1 (Original)", algo.FixedTokenize},
		{"V2 (8-byte reads)", algo.FixedTokenizeV2},
		{"V3 (Bitmask)", algo.FixedTokenizeV3},
		{"V4 (Two-pass)", algo.FixedTokenizeV4},
	}

	var baseline float64

	for i, tok := range tokenizers {
		t.Logf("\n=== %s ===", tok.name)
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
			go func(s, e int, fn func(string, *algo.FixedHashTable) int) {
				defer wg.Done()
				table := algo.NewFixedHashTable(4096)
				for j := s; j < e; j++ {
					fn(allTexts[j], table)
				}
			}(startIdx, endIdx, tok.fn)
		}
		wg.Wait()
		elapsed := time.Since(start)
		rate := float64(len(allTexts)) / elapsed.Seconds()

		if i == 0 {
			baseline = rate
			t.Logf("Rate: %.0f docs/sec (baseline)", rate)
		} else {
			pctChange := (rate/baseline - 1) * 100
			t.Logf("Rate: %.0f docs/sec (%+.1f%%)", rate, pctChange)
		}
	}

	t.Log("\n═══════════════════════════════════════════════════════════════")
	t.Log("SUMMARY")
	t.Log("═══════════════════════════════════════════════════════════════")
	t.Logf("Gap to 1M from baseline: %.1fx needed", 1000000.0/baseline)
}

// TestTokenizerCorrectness verifies all tokenizers produce same results
func TestTokenizerCorrectness(t *testing.T) {
	testTexts := []string{
		"Hello World",
		"The quick brown fox jumps over the lazy dog",
		"Testing 123 special!@#$ characters",
		"Đây là văn bản tiếng Việt",
		"UPPERCASE lowercase MiXeD",
		"   spaces   and   tabs\t\tand\nnewlines",
		"",
		"a",
		"ab",
		"abc def ghi jkl mno pqr stu vwx yz",
	}

	for _, text := range testTexts {
		t1 := algo.NewFixedHashTable(4096)
		t2 := algo.NewFixedHashTable(4096)
		t3 := algo.NewFixedHashTable(4096)
		t4 := algo.NewFixedHashTable(4096)

		c1 := algo.FixedTokenize(text, t1)
		c2 := algo.FixedTokenizeV2(text, t2)
		c3 := algo.FixedTokenizeV3(text, t3)
		c4 := algo.FixedTokenizeV4(text, t4)

		if c1 != c2 || c1 != c3 || c1 != c4 {
			t.Errorf("Token count mismatch for %q: V1=%d V2=%d V3=%d V4=%d",
				text, c1, c2, c3, c4)
		}
	}
}
