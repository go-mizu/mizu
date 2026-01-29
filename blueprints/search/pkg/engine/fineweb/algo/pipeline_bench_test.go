package algo_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb"
	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/algo"
)

// TestPipelineBreakdown measures time spent in each pipeline phase
func TestPipelineBreakdown(t *testing.T) {
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

	numDocs := len(allTexts)
	batchSize := (numDocs + numWorkers - 1) / numWorkers
	var wg sync.WaitGroup

	// ═══════════════════════════════════════════════════════════════
	// PHASE A: Pure Tokenization Only
	// ═══════════════════════════════════════════════════════════════
	t.Log("\n=== Phase A: Pure Tokenization ===")
	runtime.GC()

	start := time.Now()
	for w := 0; w < numWorkers; w++ {
		startIdx := w * batchSize
		endIdx := startIdx + batchSize
		if endIdx > numDocs {
			endIdx = numDocs
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
	phaseA := time.Since(start)
	rateA := float64(numDocs) / phaseA.Seconds()
	t.Logf("Time: %v, Rate: %.0f docs/sec", phaseA, rateA)

	// ═══════════════════════════════════════════════════════════════
	// PHASE B: Tokenization + Shard Distribution (local)
	// ═══════════════════════════════════════════════════════════════
	t.Log("\n=== Phase B: Tokenization + Local Shard Distribution ===")
	runtime.GC()

	type posting struct {
		hash  uint64
		docID uint32
		freq  uint16
	}

	start = time.Now()
	for w := 0; w < numWorkers; w++ {
		startIdx := w * batchSize
		endIdx := startIdx + batchSize
		if endIdx > numDocs {
			endIdx = numDocs
		}
		if startIdx >= endIdx {
			break
		}
		wg.Add(1)
		go func(s, e int) {
			defer wg.Done()
			table := algo.NewFixedHashTable(4096)
			localShards := make([][]posting, 256)
			for i := range localShards {
				localShards[i] = make([]posting, 0, 64)
			}

			for docID := s; docID < e; docID++ {
				algo.FixedTokenize(allTexts[docID], table)

				// Distribute to local shards
				for _, idx := range table.UsedSlots() {
					hash := table.Keys()[idx]
					shardID := hash & 0xFF
					localShards[shardID] = append(localShards[shardID],
						posting{hash, uint32(docID), table.Counts()[idx]})
				}
			}
		}(startIdx, endIdx)
	}
	wg.Wait()
	phaseB := time.Since(start)
	rateB := float64(numDocs) / phaseB.Seconds()
	t.Logf("Time: %v, Rate: %.0f docs/sec (%.1f%% overhead from A)", phaseB, rateB, (phaseB.Seconds()/phaseA.Seconds()-1)*100)

	// ═══════════════════════════════════════════════════════════════
	// PHASE C: Full Pipeline with Global Shard Merge
	// ═══════════════════════════════════════════════════════════════
	t.Log("\n=== Phase C: Full Pipeline with Global Merge ===")
	runtime.GC()

	shardBuffers := make([][]posting, 256)
	for i := range shardBuffers {
		shardBuffers[i] = make([]posting, 0, numDocs*10/256)
	}
	var shardMu [256]sync.Mutex

	start = time.Now()

	// Phase C1: Tokenize and collect
	for w := 0; w < numWorkers; w++ {
		startIdx := w * batchSize
		endIdx := startIdx + batchSize
		if endIdx > numDocs {
			endIdx = numDocs
		}
		if startIdx >= endIdx {
			break
		}
		wg.Add(1)
		go func(s, e int) {
			defer wg.Done()
			table := algo.NewFixedHashTable(4096)
			localShards := make([][]posting, 256)
			for i := range localShards {
				localShards[i] = make([]posting, 0, 64)
			}

			for docID := s; docID < e; docID++ {
				algo.FixedTokenize(allTexts[docID], table)

				for _, idx := range table.UsedSlots() {
					hash := table.Keys()[idx]
					shardID := hash & 0xFF
					localShards[shardID] = append(localShards[shardID],
						posting{hash, uint32(docID), table.Counts()[idx]})
				}
			}

			// Merge to global
			for shardID := range localShards {
				if len(localShards[shardID]) > 0 {
					shardMu[shardID].Lock()
					shardBuffers[shardID] = append(shardBuffers[shardID], localShards[shardID]...)
					shardMu[shardID].Unlock()
				}
			}
		}(startIdx, endIdx)
	}
	wg.Wait()
	phaseC1 := time.Since(start)

	// Phase C2: Build posting lists
	type postingList struct {
		docIDs []uint32
		freqs  []uint16
	}
	shardMaps := make([]map[uint64]*postingList, 256)
	for i := range shardMaps {
		shardMaps[i] = make(map[uint64]*postingList, 10000)
	}

	shardsPerWorker := (256 + numWorkers - 1) / numWorkers
	buildStart := time.Now()

	for w := 0; w < numWorkers; w++ {
		startShard := w * shardsPerWorker
		endShard := startShard + shardsPerWorker
		if endShard > 256 {
			endShard = 256
		}
		if startShard >= endShard {
			break
		}
		wg.Add(1)
		go func(ss, es int) {
			defer wg.Done()
			for shardID := ss; shardID < es; shardID++ {
				m := shardMaps[shardID]
				for _, p := range shardBuffers[shardID] {
					pl, exists := m[p.hash]
					if !exists {
						pl = &postingList{
							docIDs: make([]uint32, 0, 32),
							freqs:  make([]uint16, 0, 32),
						}
						m[p.hash] = pl
					}
					pl.docIDs = append(pl.docIDs, p.docID)
					pl.freqs = append(pl.freqs, p.freq)
				}
			}
		}(startShard, endShard)
	}
	wg.Wait()
	phaseC2 := time.Since(buildStart)
	phaseC := time.Since(start)
	rateC := float64(numDocs) / phaseC.Seconds()

	t.Logf("Time: %v (collect: %v, build: %v)", phaseC, phaseC1, phaseC2)
	t.Logf("Rate: %.0f docs/sec (%.1f%% overhead from A)", rateC, (phaseC.Seconds()/phaseA.Seconds()-1)*100)

	// ═══════════════════════════════════════════════════════════════
	// Summary
	// ═══════════════════════════════════════════════════════════════
	t.Log("\n═══════════════════════════════════════════════════════════════")
	t.Log("PIPELINE BREAKDOWN SUMMARY")
	t.Log("═══════════════════════════════════════════════════════════════")
	t.Logf("%-40s: %.0f docs/sec", "A. Pure Tokenization", rateA)
	t.Logf("%-40s: %.0f docs/sec (%+.1f%%)", "B. + Local Shard Distribution", rateB, (rateB/rateA-1)*100)
	t.Logf("%-40s: %.0f docs/sec (%+.1f%%)", "C. + Global Merge + Build", rateC, (rateC/rateA-1)*100)

	t.Log("\n═══════════════════════════════════════════════════════════════")
	t.Log("TIME BREAKDOWN FOR FULL PIPELINE")
	t.Log("═══════════════════════════════════════════════════════════════")
	t.Logf("Tokenization baseline:    %v (%.1f%%)", phaseA, phaseA.Seconds()/phaseC.Seconds()*100)
	t.Logf("Shard distribution:       %v (%.1f%%)", phaseC1-phaseA, (phaseC1.Seconds()-phaseA.Seconds())/phaseC.Seconds()*100)
	t.Logf("Posting list build:       %v (%.1f%%)", phaseC2, phaseC2.Seconds()/phaseC.Seconds()*100)

	t.Logf("\nGap to 1M from full pipeline: %.1fx", 1000000.0/rateC)
}

// TestIOvsCPU measures I/O overhead vs pure CPU processing
func TestIOvsCPU(t *testing.T) {
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

	// ═══════════════════════════════════════════════════════════════
	// TEST 1: Load + Process (I/O bound)
	// ═══════════════════════════════════════════════════════════════
	t.Log("\n=== Test 1: Load + Process (I/O bound) ===")
	runtime.GC()

	reader := fineweb.NewParquetReader(parquetDir).WithBatchSize(10000)
	var docCount atomic.Int64
	var wg sync.WaitGroup

	start := time.Now()
	for batch, err := range reader.ReadTextsOnlyParallel(ctx, 8) {
		if err != nil {
			t.Fatal(err)
		}

		texts := make([]string, 0, len(batch))
		for _, doc := range batch {
			texts = append(texts, doc.Text)
		}
		docCount.Add(int64(len(texts)))

		wg.Add(1)
		go func(texts []string) {
			defer wg.Done()
			table := algo.NewFixedHashTable(4096)
			for _, text := range texts {
				algo.FixedTokenize(text, table)
			}
		}(texts)
	}
	wg.Wait()
	ioTime := time.Since(start)
	ioRate := float64(docCount.Load()) / ioTime.Seconds()
	t.Logf("Docs: %d, Time: %v, Rate: %.0f docs/sec", docCount.Load(), ioTime, ioRate)

	// ═══════════════════════════════════════════════════════════════
	// TEST 2: Pre-loaded processing (CPU bound)
	// ═══════════════════════════════════════════════════════════════
	t.Log("\n=== Test 2: Pre-loaded Processing (CPU bound) ===")

	// Re-load data
	var allTexts []string
	reader = fineweb.NewParquetReader(parquetDir).WithBatchSize(10000)
	for batch, err := range reader.ReadTextsOnlyParallel(ctx, 8) {
		if err != nil {
			t.Fatal(err)
		}
		for _, doc := range batch {
			allTexts = append(allTexts, doc.Text)
		}
	}

	runtime.GC()
	batchSize := (len(allTexts) + numWorkers - 1) / numWorkers

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
	cpuTime := time.Since(start)
	cpuRate := float64(len(allTexts)) / cpuTime.Seconds()
	t.Logf("Docs: %d, Time: %v, Rate: %.0f docs/sec", len(allTexts), cpuTime, cpuRate)

	// Summary
	t.Log("\n═══════════════════════════════════════════════════════════════")
	t.Log("I/O vs CPU SUMMARY")
	t.Log("═══════════════════════════════════════════════════════════════")
	t.Logf("I/O bound (load+process): %.0f docs/sec", ioRate)
	t.Logf("CPU bound (pre-loaded):   %.0f docs/sec", cpuRate)
	t.Logf("I/O overhead:             %.1fx slowdown", cpuRate/ioRate)
	t.Logf("\nGap to 1M from I/O bound: %.1fx", 1000000.0/ioRate)
	t.Logf("Gap to 1M from CPU bound: %.1fx", 1000000.0/cpuRate)
}

// TestConcatenatedDocuments tests if concatenating documents provides speedup
func TestConcatenatedDocuments(t *testing.T) {
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

	// Test 1: Individual documents
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
				scanOnlyNoHash(allTexts[i])
			}
		}(startIdx, endIdx)
	}
	wg.Wait()
	individualTime := time.Since(start)
	individualRate := float64(len(allTexts)) / individualTime.Seconds()

	// Test 2: Concatenated into mega-documents
	// Concatenate 100 docs at a time
	concatSize := 100
	var megaDocs []string
	for i := 0; i < len(allTexts); i += concatSize {
		end := i + concatSize
		if end > len(allTexts) {
			end = len(allTexts)
		}
		var totalLen int
		for j := i; j < end; j++ {
			totalLen += len(allTexts[j]) + 1
		}
		concat := make([]byte, 0, totalLen)
		for j := i; j < end; j++ {
			concat = append(concat, allTexts[j]...)
			concat = append(concat, ' ')
		}
		megaDocs = append(megaDocs, string(concat))
	}
	t.Logf("Created %d mega-docs from %d originals", len(megaDocs), len(allTexts))

	runtime.GC()
	megaBatchSize := (len(megaDocs) + numWorkers - 1) / numWorkers
	start = time.Now()
	for w := 0; w < numWorkers; w++ {
		startIdx := w * megaBatchSize
		endIdx := startIdx + megaBatchSize
		if endIdx > len(megaDocs) {
			endIdx = len(megaDocs)
		}
		if startIdx >= endIdx {
			break
		}
		wg.Add(1)
		go func(s, e int) {
			defer wg.Done()
			for i := s; i < e; i++ {
				scanOnlyNoHash(megaDocs[i])
			}
		}(startIdx, endIdx)
	}
	wg.Wait()
	concatTime := time.Since(start)
	concatRate := float64(len(allTexts)) / concatTime.Seconds() // Rate in original docs

	t.Log("\n═══════════════════════════════════════════════════════════════")
	t.Log("CONCATENATED DOCUMENT TEST")
	t.Log("═══════════════════════════════════════════════════════════════")
	t.Logf("Individual docs:    %.0f docs/sec", individualRate)
	t.Logf("Concatenated:       %.0f docs/sec (%+.1f%%)", concatRate, (concatRate/individualRate-1)*100)
}

// scanOnlyNoHashLocal is a local copy for testing
func scanOnlyNoHashLocal(text string) int {
	if len(text) == 0 {
		return 0
	}

	data := unsafe.Slice(unsafe.StringData(text), len(text))
	n := len(data)
	tokenCount := 0
	i := 0

	for i < n {
		for i < n && testCharLUT[data[i]] == 0 {
			i++
		}
		if i >= n {
			break
		}

		start := i
		for i < n && testCharLUT[data[i]] != 0 {
			i++
		}

		tokenLen := i - start
		if tokenLen >= 2 && tokenLen <= 32 {
			tokenCount++
		}
	}

	return tokenCount
}
