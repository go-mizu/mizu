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

// TestHashFunctionComparison compares different hash functions for tokenization
func TestHashFunctionComparison(t *testing.T) {
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
	// TEST 1: FNV-1a (current implementation)
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
			table := algo.NewFixedHashTable(4096)
			for i := s; i < e; i++ {
				algo.FixedTokenize(allTexts[i], table)
			}
		}(startIdx, endIdx)
	}
	wg.Wait()
	fnvTime := time.Since(start)
	fnvRate := float64(len(allTexts)) / fnvTime.Seconds()

	// ═══════════════════════════════════════════════════════════════
	// TEST 2: Simple polynomial hash (Rabin fingerprint style)
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
				tokenizePolynomial(allTexts[i], table)
			}
		}(startIdx, endIdx)
	}
	wg.Wait()
	polyTime := time.Since(start)
	polyRate := float64(len(allTexts)) / polyTime.Seconds()

	// ═══════════════════════════════════════════════════════════════
	// TEST 3: DJB2 hash (very simple, shift + add)
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
				tokenizeDJB2(allTexts[i], table)
			}
		}(startIdx, endIdx)
	}
	wg.Wait()
	djb2Time := time.Since(start)
	djb2Rate := float64(len(allTexts)) / djb2Time.Seconds()

	// ═══════════════════════════════════════════════════════════════
	// TEST 4: SDBM hash (shift + add, different constants)
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
				tokenizeSDBM(allTexts[i], table)
			}
		}(startIdx, endIdx)
	}
	wg.Wait()
	sdbmTime := time.Since(start)
	sdbmRate := float64(len(allTexts)) / sdbmTime.Seconds()

	// ═══════════════════════════════════════════════════════════════
	// TEST 5: xxHash-style (rotations and XOR)
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
				tokenizeXX(allTexts[i], table)
			}
		}(startIdx, endIdx)
	}
	wg.Wait()
	xxTime := time.Since(start)
	xxRate := float64(len(allTexts)) / xxTime.Seconds()

	// ═══════════════════════════════════════════════════════════════
	// TEST 6: Accumulate-only (just XOR bytes together - minimal hash)
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
				tokenizeMinimal(allTexts[i], table)
			}
		}(startIdx, endIdx)
	}
	wg.Wait()
	minimalTime := time.Since(start)
	minimalRate := float64(len(allTexts)) / minimalTime.Seconds()

	// Summary
	t.Log("\n═══════════════════════════════════════════════════════════════")
	t.Log("HASH FUNCTION COMPARISON")
	t.Log("═══════════════════════════════════════════════════════════════")
	t.Logf("%-30s: %8.0f docs/sec (baseline)", "1. FNV-1a (current)", fnvRate)
	t.Logf("%-30s: %8.0f docs/sec (%+.1f%%)", "2. Polynomial", polyRate, (polyRate/fnvRate-1)*100)
	t.Logf("%-30s: %8.0f docs/sec (%+.1f%%)", "3. DJB2", djb2Rate, (djb2Rate/fnvRate-1)*100)
	t.Logf("%-30s: %8.0f docs/sec (%+.1f%%)", "4. SDBM", sdbmRate, (sdbmRate/fnvRate-1)*100)
	t.Logf("%-30s: %8.0f docs/sec (%+.1f%%)", "5. xxHash-style", xxRate, (xxRate/fnvRate-1)*100)
	t.Logf("%-30s: %8.0f docs/sec (%+.1f%%)", "6. Minimal (XOR)", minimalRate, (minimalRate/fnvRate-1)*100)

	t.Log("\n═══════════════════════════════════════════════════════════════")
	t.Log("GAP TO 1M DOCS/SEC")
	t.Log("═══════════════════════════════════════════════════════════════")
	t.Logf("From FNV-1a:    %.1fx needed", 1000000.0/fnvRate)
	t.Logf("From DJB2:      %.1fx needed", 1000000.0/djb2Rate)
	t.Logf("From Minimal:   %.1fx needed", 1000000.0/minimalRate)
}

// tokenizePolynomial uses a simple polynomial hash (base 31)
func tokenizePolynomial(text string, table *algo.FixedHashTable) int {
	if len(text) == 0 {
		return 0
	}

	data := unsafe.Slice(unsafe.StringData(text), len(text))
	n := len(data)
	tokenCount := 0
	i := 0

	table.Reset()

	for i < n {
		for i < n && testCharLUT[data[i]] == 0 {
			i++
		}
		if i >= n {
			break
		}

		start := i
		hash := uint64(0)

		for i < n {
			c := testCharLUT[data[i]]
			if c == 0 {
				break
			}
			hash = hash*31 + uint64(c)
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

// tokenizeDJB2 uses the DJB2 hash function (hash * 33 + c)
// This uses shifts instead of multiplication: hash * 33 = hash << 5 + hash
func tokenizeDJB2(text string, table *algo.FixedHashTable) int {
	if len(text) == 0 {
		return 0
	}

	data := unsafe.Slice(unsafe.StringData(text), len(text))
	n := len(data)
	tokenCount := 0
	i := 0

	table.Reset()

	for i < n {
		for i < n && testCharLUT[data[i]] == 0 {
			i++
		}
		if i >= n {
			break
		}

		start := i
		hash := uint64(5381) // DJB2 magic number

		for i < n {
			c := testCharLUT[data[i]]
			if c == 0 {
				break
			}
			// hash * 33 + c = (hash << 5) + hash + c
			hash = ((hash << 5) + hash) + uint64(c)
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

// tokenizeSDBM uses the SDBM hash function
// hash * 65599 = hash * 65536 + hash * 64 - hash = (hash << 16) + (hash << 6) - hash
func tokenizeSDBM(text string, table *algo.FixedHashTable) int {
	if len(text) == 0 {
		return 0
	}

	data := unsafe.Slice(unsafe.StringData(text), len(text))
	n := len(data)
	tokenCount := 0
	i := 0

	table.Reset()

	for i < n {
		for i < n && testCharLUT[data[i]] == 0 {
			i++
		}
		if i >= n {
			break
		}

		start := i
		hash := uint64(0)

		for i < n {
			c := testCharLUT[data[i]]
			if c == 0 {
				break
			}
			// hash * 65599 + c
			hash = uint64(c) + (hash << 6) + (hash << 16) - hash
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

// tokenizeXX uses xxHash-style rotations
func tokenizeXX(text string, table *algo.FixedHashTable) int {
	if len(text) == 0 {
		return 0
	}

	data := unsafe.Slice(unsafe.StringData(text), len(text))
	n := len(data)
	tokenCount := 0
	i := 0

	const prime1 = 11400714785074694791
	const prime2 = 14029467366897019727

	table.Reset()

	for i < n {
		for i < n && testCharLUT[data[i]] == 0 {
			i++
		}
		if i >= n {
			break
		}

		start := i
		hash := uint64(0)

		for i < n {
			c := testCharLUT[data[i]]
			if c == 0 {
				break
			}
			hash ^= uint64(c) * prime1
			hash = (hash<<27 | hash>>(64-27)) * prime2
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

// tokenizeMinimal uses the simplest possible hash: position-weighted XOR
// This is fast but may have more collisions
func tokenizeMinimal(text string, table *algo.FixedHashTable) int {
	if len(text) == 0 {
		return 0
	}

	data := unsafe.Slice(unsafe.StringData(text), len(text))
	n := len(data)
	tokenCount := 0
	i := 0

	table.Reset()

	for i < n {
		for i < n && testCharLUT[data[i]] == 0 {
			i++
		}
		if i >= n {
			break
		}

		start := i
		hash := uint64(0)
		pos := 0

		for i < n {
			c := testCharLUT[data[i]]
			if c == 0 {
				break
			}
			// Position-weighted accumulation: shift by position mod 56
			hash ^= uint64(c) << (pos & 0x3F)
			pos += 7
			i++
		}

		tokenLen := i - start
		if tokenLen >= 2 && tokenLen <= 32 {
			// Add length to differentiate same-prefix tokens
			hash ^= uint64(tokenLen) << 56
			table.Insert(hash)
			tokenCount++
		}
	}

	return tokenCount
}

// TestWorkerScalingHash tests different worker counts for optimal throughput (hash version)
func TestWorkerScalingHash(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	home, _ := os.UserHomeDir()
	parquetDir := filepath.Join(home, "data", "fineweb-2", "vie_Latn", "test")

	if _, err := os.Stat(parquetDir); os.IsNotExist(err) {
		t.Skipf("Parquet directory not found: %s", parquetDir)
	}

	ctx := context.Background()
	cpuCount := runtime.NumCPU()

	t.Logf("CPU cores: %d", cpuCount)

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

	workerCounts := []int{
		cpuCount,
		cpuCount * 2,
		cpuCount * 3,
		cpuCount * 4,
		cpuCount * 5,
		cpuCount * 6,
		cpuCount * 8,
		cpuCount * 10,
	}

	t.Log("\n═══════════════════════════════════════════════════════════════")
	t.Log("WORKER SCALING TEST")
	t.Log("═══════════════════════════════════════════════════════════════")

	var bestRate float64
	var bestWorkers int

	for _, numWorkers := range workerCounts {
		var wg sync.WaitGroup
		batchSize := (len(allTexts) + numWorkers - 1) / numWorkers

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
		elapsed := time.Since(start)
		rate := float64(len(allTexts)) / elapsed.Seconds()

		marker := ""
		if rate > bestRate {
			bestRate = rate
			bestWorkers = numWorkers
			marker = " ***"
		}

		t.Logf("%3d workers (%2dx CPU): %8.0f docs/sec%s", numWorkers, numWorkers/cpuCount, rate, marker)
	}

	t.Logf("\nBest: %d workers (%.0f docs/sec)", bestWorkers, bestRate)
	t.Logf("Gap to 1M: %.1fx", 1000000.0/bestRate)
}
