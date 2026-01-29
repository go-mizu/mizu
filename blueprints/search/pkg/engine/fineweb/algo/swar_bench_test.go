package algo_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"
	"unsafe"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb"
	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/algo"
)

const (
	fnvOffset = 14695981039346656037
	fnvPrime  = 1099511628211
)

func unsafeStringToBytes(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

// TestSWARTokenizerComparison compares SWAR tokenizer against existing approaches
func TestSWARTokenizerComparison(t *testing.T) {
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
	// TEST 1: SWAR Scan Only (pure scanning baseline)
	// ═══════════════════════════════════════════════════════════════
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
			for i := s; i < e; i++ {
				algo.SWARScanOnly(allTexts[i])
			}
		}(startIdx, endIdx)
	}
	wg.Wait()
	swarScanTime := time.Since(start)
	swarScanRate := float64(len(allTexts)) / swarScanTime.Seconds()

	// ═══════════════════════════════════════════════════════════════
	// TEST 2: Original scan (from bottleneck profiler)
	// ═══════════════════════════════════════════════════════════════
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
			for i := s; i < e; i++ {
				scanOnlyNoHash(allTexts[i])
			}
		}(startIdx, endIdx)
	}
	wg.Wait()
	origScanTime := time.Since(start)
	origScanRate := float64(len(allTexts)) / origScanTime.Seconds()

	// ═══════════════════════════════════════════════════════════════
	// TEST 3: SWAR Tokenize with Go map
	// ═══════════════════════════════════════════════════════════════
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
			freqs := make(map[uint64]uint16, 256)
			for i := s; i < e; i++ {
				clear(freqs)
				algo.SWARTokenizeSimple(allTexts[i], freqs)
			}
		}(startIdx, endIdx)
	}
	wg.Wait()
	swarMapTime := time.Since(start)
	swarMapRate := float64(len(allTexts)) / swarMapTime.Seconds()

	// ═══════════════════════════════════════════════════════════════
	// TEST 4: Original TokenizeMega with Go map
	// ═══════════════════════════════════════════════════════════════
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
			freqs := make(map[uint64]uint16, 256)
			for i := s; i < e; i++ {
				clear(freqs)
				algo.TokenizeMega(allTexts[i], freqs)
			}
		}(startIdx, endIdx)
	}
	wg.Wait()
	origMapTime := time.Since(start)
	origMapRate := float64(len(allTexts)) / origMapTime.Seconds()

	// ═══════════════════════════════════════════════════════════════
	// TEST 5: SWAR Tokenize with FixedHashTable
	// ═══════════════════════════════════════════════════════════════
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
			table := algo.NewFixedHashTable(4096)
			for i := s; i < e; i++ {
				SWARTokenizeFixed(allTexts[i], table)
			}
		}(startIdx, endIdx)
	}
	wg.Wait()
	swarFixedTime := time.Since(start)
	swarFixedRate := float64(len(allTexts)) / swarFixedTime.Seconds()

	// ═══════════════════════════════════════════════════════════════
	// TEST 6: Original FixedTokenize
	// ═══════════════════════════════════════════════════════════════
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
			table := algo.NewFixedHashTable(4096)
			for i := s; i < e; i++ {
				algo.FixedTokenize(allTexts[i], table)
			}
		}(startIdx, endIdx)
	}
	wg.Wait()
	origFixedTime := time.Since(start)
	origFixedRate := float64(len(allTexts)) / origFixedTime.Seconds()

	// Summary
	t.Log("\n═══════════════════════════════════════════════════════════════")
	t.Log("SWAR TOKENIZER COMPARISON")
	t.Log("═══════════════════════════════════════════════════════════════")
	t.Logf("%-35s: %8.0f docs/sec", "1. Original Scan (baseline)", origScanRate)
	t.Logf("%-35s: %8.0f docs/sec (%+.1f%%)", "2. SWAR Scan", swarScanRate, (swarScanRate/origScanRate-1)*100)
	t.Logf("%-35s: %8.0f docs/sec", "3. Original TokenizeMega + Map", origMapRate)
	t.Logf("%-35s: %8.0f docs/sec (%+.1f%%)", "4. SWAR Tokenize + Map", swarMapRate, (swarMapRate/origMapRate-1)*100)
	t.Logf("%-35s: %8.0f docs/sec", "5. Original FixedTokenize", origFixedRate)
	t.Logf("%-35s: %8.0f docs/sec (%+.1f%%)", "6. SWAR + FixedHashTable", swarFixedRate, (swarFixedRate/origFixedRate-1)*100)

	t.Log("\n═══════════════════════════════════════════════════════════════")
	t.Log("GAP TO 1M DOCS/SEC")
	t.Log("═══════════════════════════════════════════════════════════════")
	t.Logf("From SWAR Scan:           %.1fx needed", 1000000.0/swarScanRate)
	t.Logf("From SWAR + Map:          %.1fx needed", 1000000.0/swarMapRate)
	t.Logf("From SWAR + FixedTable:   %.1fx needed", 1000000.0/swarFixedRate)
	t.Logf("Best original (Fixed):    %.1fx needed", 1000000.0/origFixedRate)
}

// SWARTokenizeFixed uses SWAR tokenization with FixedHashTable
func SWARTokenizeFixed(text string, table *algo.FixedHashTable) int {
	if len(text) == 0 {
		return 0
	}

	data := unsafeStringToBytes(text)
	n := len(data)
	tokenCount := 0
	i := 0

	table.Reset()

	for i < n {
		// Skip delimiters with batch check
		for i+8 <= n {
			b0 := testCharLUT[data[i]]
			b1 := testCharLUT[data[i+1]]
			b2 := testCharLUT[data[i+2]]
			b3 := testCharLUT[data[i+3]]
			b4 := testCharLUT[data[i+4]]
			b5 := testCharLUT[data[i+5]]
			b6 := testCharLUT[data[i+6]]
			b7 := testCharLUT[data[i+7]]

			if b0|b1|b2|b3|b4|b5|b6|b7 != 0 {
				if b0 != 0 {
					break
				}
				if b1 != 0 {
					i++
					break
				}
				if b2 != 0 {
					i += 2
					break
				}
				if b3 != 0 {
					i += 3
					break
				}
				if b4 != 0 {
					i += 4
					break
				}
				if b5 != 0 {
					i += 5
					break
				}
				if b6 != 0 {
					i += 6
					break
				}
				i += 7
				break
			}
			i += 8
		}

		for i < n && testCharLUT[data[i]] == 0 {
			i++
		}
		if i >= n {
			break
		}

		start := i
		hash := uint64(fnvOffset)

		// Hash token with unrolled loop
		for i+4 <= n {
			c0 := testCharLUT[data[i]]
			c1 := testCharLUT[data[i+1]]
			c2 := testCharLUT[data[i+2]]
			c3 := testCharLUT[data[i+3]]

			if c0 == 0 {
				break
			}
			hash = (hash ^ uint64(c0)) * fnvPrime
			i++

			if c1 == 0 {
				break
			}
			hash = (hash ^ uint64(c1)) * fnvPrime
			i++

			if c2 == 0 {
				break
			}
			hash = (hash ^ uint64(c2)) * fnvPrime
			i++

			if c3 == 0 {
				break
			}
			hash = (hash ^ uint64(c3)) * fnvPrime
			i++
		}

		for i < n {
			c := testCharLUT[data[i]]
			if c == 0 {
				break
			}
			hash = (hash ^ uint64(c)) * fnvPrime
			i++
		}

		tokenLen := i - start
		if tokenLen >= 2 && tokenLen <= 32 {
			table.Insert(hash)
			tokenCount++
		}
	}

	return tokenCount
}

// TestBatchProcessingSpeedup tests if processing documents in batches provides speedup
func TestBatchProcessingSpeedup(t *testing.T) {
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

	batchSize := (len(allTexts) + numWorkers - 1) / numWorkers
	var wg sync.WaitGroup

	// Test 1: Individual document processing
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

	// Test 2: Batch processing with shared structures
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
			// Process in mini-batches of 100 docs
			table := algo.NewFixedHashTable(8192) // Larger table for batch
			for batchStart := s; batchStart < e; batchStart += 100 {
				batchEnd := batchStart + 100
				if batchEnd > e {
					batchEnd = e
				}
				for i := batchStart; i < batchEnd; i++ {
					algo.FixedTokenize(allTexts[i], table)
				}
			}
		}(startIdx, endIdx)
	}
	wg.Wait()
	batchTime := time.Since(start)
	batchRate := float64(len(allTexts)) / batchTime.Seconds()

	// Test 3: Concatenated text processing
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
			table := algo.NewFixedHashTable(4096)
			// Process 10 docs at a time by concatenating
			for batchStart := s; batchStart < e; batchStart += 10 {
				batchEnd := batchStart + 10
				if batchEnd > e {
					batchEnd = e
				}
				// Concatenate texts with delimiter
				var totalLen int
				for i := batchStart; i < batchEnd; i++ {
					totalLen += len(allTexts[i]) + 1
				}
				concat := make([]byte, 0, totalLen)
				for i := batchStart; i < batchEnd; i++ {
					concat = append(concat, allTexts[i]...)
					concat = append(concat, ' ')
				}
				// Tokenize concatenated text
				algo.FixedTokenize(string(concat), table)
			}
		}(startIdx, endIdx)
	}
	wg.Wait()
	concatTime := time.Since(start)
	concatRate := float64(len(allTexts)) / concatTime.Seconds()

	t.Log("\n═══════════════════════════════════════════════════════════════")
	t.Log("BATCH PROCESSING COMPARISON")
	t.Log("═══════════════════════════════════════════════════════════════")
	t.Logf("Individual processing: %.0f docs/sec (baseline)", individualRate)
	t.Logf("Batch processing:      %.0f docs/sec (%+.1f%%)", batchRate, (batchRate/individualRate-1)*100)
	t.Logf("Concatenated:          %.0f docs/sec (%+.1f%%)", concatRate, (concatRate/individualRate-1)*100)
}
