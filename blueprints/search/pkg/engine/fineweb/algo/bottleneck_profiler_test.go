package algo_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sync"
	"testing"
	"time"
	"unsafe"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb"
	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/algo"
)

// ═══════════════════════════════════════════════════════════════════════════
// BOTTLENECK PROFILER - Identifies exact CPU hotspots per phase
// ═══════════════════════════════════════════════════════════════════════════

// TestBottleneckProfile creates CPU profiles for each phase
func TestBottleneckProfile(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	home, _ := os.UserHomeDir()
	// Use test data for faster iteration
	parquetDir := filepath.Join(home, "data", "fineweb-2", "vie_Latn", "test")

	if _, err := os.Stat(parquetDir); os.IsNotExist(err) {
		t.Skipf("Parquet directory not found: %s", parquetDir)
	}

	ctx := context.Background()
	numWorkers := runtime.NumCPU() * 5

	t.Logf("CPU cores: %d, Workers: %d", runtime.NumCPU(), numWorkers)

	// Load all data first
	t.Log("\n=== Loading data ===")
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

	// Create profiles directory
	profileDir := "/tmp/fts_profiles"
	os.MkdirAll(profileDir, 0755)

	batchSize := (len(allTexts) + numWorkers - 1) / numWorkers

	// ═══════════════════════════════════════════════════════════════
	// PROFILE 1: Pure Tokenization with FixedHashTable
	// ═══════════════════════════════════════════════════════════════
	t.Log("\n=== Profiling: Tokenization (FixedHashTable) ===")
	runtime.GC()

	f1, _ := os.Create(filepath.Join(profileDir, "1_tokenize_fixed.prof"))
	pprof.StartCPUProfile(f1)

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
		go func(start, end int) {
			defer wg.Done()
			table := algo.NewFixedHashTable(4096)
			for i := start; i < end; i++ {
				algo.FixedTokenize(allTexts[i], table)
			}
		}(startIdx, endIdx)
	}
	wg.Wait()

	pprof.StopCPUProfile()
	f1.Close()
	elapsed := time.Since(start)
	t.Logf("  FixedTokenize: %v = %.0f docs/sec", elapsed, float64(len(allTexts))/elapsed.Seconds())

	// ═══════════════════════════════════════════════════════════════
	// PROFILE 2: Tokenization with Go Map (baseline comparison)
	// ═══════════════════════════════════════════════════════════════
	t.Log("\n=== Profiling: Tokenization (Go Map) ===")
	runtime.GC()

	f2, _ := os.Create(filepath.Join(profileDir, "2_tokenize_map.prof"))
	pprof.StartCPUProfile(f2)

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
		go func(start, end int) {
			defer wg.Done()
			freqs := make(map[uint64]uint16, 256)
			for i := start; i < end; i++ {
				clear(freqs)
				algo.TokenizeMega(allTexts[i], freqs)
			}
		}(startIdx, endIdx)
	}
	wg.Wait()

	pprof.StopCPUProfile()
	f2.Close()
	elapsed = time.Since(start)
	t.Logf("  TokenizeMega: %v = %.0f docs/sec", elapsed, float64(len(allTexts))/elapsed.Seconds())

	// ═══════════════════════════════════════════════════════════════
	// PROFILE 3: Pure Scanning (no hashing, no map)
	// ═══════════════════════════════════════════════════════════════
	t.Log("\n=== Profiling: Pure Scanning (no hash) ===")
	runtime.GC()

	f3, _ := os.Create(filepath.Join(profileDir, "3_scan_only.prof"))
	pprof.StartCPUProfile(f3)

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
		go func(start, end int) {
			defer wg.Done()
			for i := start; i < end; i++ {
				scanOnlyNoHash(allTexts[i])
			}
		}(startIdx, endIdx)
	}
	wg.Wait()

	pprof.StopCPUProfile()
	f3.Close()
	elapsed = time.Since(start)
	t.Logf("  ScanOnly: %v = %.0f docs/sec", elapsed, float64(len(allTexts))/elapsed.Seconds())

	// ═══════════════════════════════════════════════════════════════
	// PROFILE 4: Full Pipeline (tokenize + shard + accumulate)
	// ═══════════════════════════════════════════════════════════════
	t.Log("\n=== Profiling: Full Pipeline ===")
	runtime.GC()

	f4, _ := os.Create(filepath.Join(profileDir, "4_full_pipeline.prof"))
	pprof.StartCPUProfile(f4)

	start = time.Now()
	runFullPipeline(allTexts, numWorkers)

	pprof.StopCPUProfile()
	f4.Close()
	elapsed = time.Since(start)
	t.Logf("  Full Pipeline: %v = %.0f docs/sec", elapsed, float64(len(allTexts))/elapsed.Seconds())

	// Summary
	t.Log("\n═══════════════════════════════════════════════════════════════")
	t.Log("PROFILES SAVED TO: " + profileDir)
	t.Log("═══════════════════════════════════════════════════════════════")
	t.Log("Analyze with:")
	t.Log("  go tool pprof -http=:8080 " + filepath.Join(profileDir, "1_tokenize_fixed.prof"))
	t.Log("  go tool pprof -http=:8081 " + filepath.Join(profileDir, "2_tokenize_map.prof"))
	t.Log("  go tool pprof -http=:8082 " + filepath.Join(profileDir, "3_scan_only.prof"))
	t.Log("  go tool pprof -http=:8083 " + filepath.Join(profileDir, "4_full_pipeline.prof"))
}

// scanOnlyNoHash does pure character scanning without hashing
func scanOnlyNoHash(text string) int {
	if len(text) == 0 {
		return 0
	}

	data := unsafe.Slice(unsafe.StringData(text), len(text))
	n := len(data)
	tokenCount := 0
	i := 0

	for i < n {
		// Skip delimiters
		for i < n && testCharLUT[data[i]] == 0 {
			i++
		}
		if i >= n {
			break
		}

		start := i
		// Scan token
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

// runFullPipeline runs the complete indexing pipeline
func runFullPipeline(texts []string, numWorkers int) {
	type posting struct {
		hash  uint64
		docID uint32
		freq  uint16
	}

	// Phase 1: Tokenize and distribute to shards
	shardBuffers := make([][]posting, 256)
	for i := range shardBuffers {
		shardBuffers[i] = make([]posting, 0, len(texts)*20/256)
	}
	var shardMu [256]sync.Mutex

	batchSize := (len(texts) + numWorkers - 1) / numWorkers
	var wg sync.WaitGroup

	for w := 0; w < numWorkers; w++ {
		startIdx := w * batchSize
		endIdx := startIdx + batchSize
		if endIdx > len(texts) {
			endIdx = len(texts)
		}
		if startIdx >= endIdx {
			break
		}

		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			table := algo.NewFixedHashTable(4096)
			localShards := make([][]posting, 256)
			for i := range localShards {
				localShards[i] = make([]posting, 0, 64)
			}

			for docID := start; docID < end; docID++ {
				algo.FixedTokenize(texts[docID], table)

				// Distribute to shards
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

	// Phase 2: Build posting lists
	type postingList struct {
		docIDs []uint32
		freqs  []uint16
	}
	shardMaps := make([]map[uint64]*postingList, 256)
	for i := range shardMaps {
		shardMaps[i] = make(map[uint64]*postingList, 10000)
	}

	shardsPerWorker := (256 + numWorkers - 1) / numWorkers

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
		go func(start, end int) {
			defer wg.Done()
			for shardID := start; shardID < end; shardID++ {
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
}

// TestPhaseBreakdownDetailed provides detailed timing for each phase
func TestPhaseBreakdownDetailed(t *testing.T) {
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
	loadStart := time.Now()

	for batch, err := range reader.ReadTextsOnlyParallel(ctx, 8) {
		if err != nil {
			t.Fatal(err)
		}
		for _, doc := range batch {
			allTexts = append(allTexts, doc.Text)
		}
	}
	loadDuration := time.Since(loadStart)
	t.Logf("Loaded %d docs in %v = %.0f docs/sec\n", len(allTexts), loadDuration, float64(len(allTexts))/loadDuration.Seconds())

	batchSize := (len(allTexts) + numWorkers - 1) / numWorkers
	var wg sync.WaitGroup

	// ═══════════════════════════════════════════════════════════════
	// PHASE 1: Pure Character Scanning (baseline)
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
				scanOnlyNoHash(allTexts[i])
			}
		}(startIdx, endIdx)
	}
	wg.Wait()
	phase1 := time.Since(start)
	phase1Rate := float64(len(allTexts)) / phase1.Seconds()

	// ═══════════════════════════════════════════════════════════════
	// PHASE 2: Scan + FNV Hash (no frequency counting)
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
				scanWithHash(allTexts[i])
			}
		}(startIdx, endIdx)
	}
	wg.Wait()
	phase2 := time.Since(start)
	phase2Rate := float64(len(allTexts)) / phase2.Seconds()

	// ═══════════════════════════════════════════════════════════════
	// PHASE 3: Scan + Hash + FixedHashTable
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
	phase3 := time.Since(start)
	phase3Rate := float64(len(allTexts)) / phase3.Seconds()

	// ═══════════════════════════════════════════════════════════════
	// PHASE 4: Scan + Hash + Go Map
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
	phase4 := time.Since(start)
	phase4Rate := float64(len(allTexts)) / phase4.Seconds()

	// Summary
	t.Log("\n═══════════════════════════════════════════════════════════════")
	t.Log("PHASE BREAKDOWN (Cumulative Overhead)")
	t.Log("═══════════════════════════════════════════════════════════════")
	t.Logf("%-35s: %8.0f docs/sec (baseline)", "1. Pure Scan (no hash)", phase1Rate)
	t.Logf("%-35s: %8.0f docs/sec (%+.0f%% from baseline)", "2. Scan + FNV Hash", phase2Rate, (phase2Rate/phase1Rate-1)*100)
	t.Logf("%-35s: %8.0f docs/sec (%+.0f%% from baseline)", "3. Scan + Hash + FixedHashTable", phase3Rate, (phase3Rate/phase1Rate-1)*100)
	t.Logf("%-35s: %8.0f docs/sec (%+.0f%% from baseline)", "4. Scan + Hash + Go Map", phase4Rate, (phase4Rate/phase1Rate-1)*100)

	t.Log("\n═══════════════════════════════════════════════════════════════")
	t.Log("OVERHEAD ANALYSIS")
	t.Log("═══════════════════════════════════════════════════════════════")
	hashOverhead := (phase1.Seconds() - phase2.Seconds()) / phase1.Seconds() * 100
	fixedTableOverhead := (phase2.Seconds() - phase3.Seconds()) / phase2.Seconds() * 100
	goMapOverhead := (phase3.Seconds() - phase4.Seconds()) / phase3.Seconds() * 100

	t.Logf("FNV Hash overhead:        %+.1f%% (from pure scan)", hashOverhead)
	t.Logf("FixedHashTable overhead:  %+.1f%% (from scan+hash)", fixedTableOverhead)
	t.Logf("Go Map vs FixedHashTable: %+.1f%%", goMapOverhead)

	// Gap analysis
	t.Log("\n═══════════════════════════════════════════════════════════════")
	t.Log("GAP TO 1M DOCS/SEC")
	t.Log("═══════════════════════════════════════════════════════════════")
	t.Logf("From pure scan:           %.1fx needed", 1000000.0/phase1Rate)
	t.Logf("From scan+hash:           %.1fx needed", 1000000.0/phase2Rate)
	t.Logf("From FixedHashTable:      %.1fx needed", 1000000.0/phase3Rate)
	t.Logf("From Go Map:              %.1fx needed", 1000000.0/phase4Rate)
}

// TestStreamIndexerVsFixedHashTable compares the stream approach vs hash table
func TestStreamIndexerVsFixedHashTable(t *testing.T) {
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

	// Test 1: FixedHashTable approach (current best)
	t.Log("\n=== FixedHashTable Approach ===")
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
	fixedDuration := time.Since(start)
	fixedRate := float64(len(allTexts)) / fixedDuration.Seconds()
	t.Logf("  Time: %v, Rate: %.0f docs/sec", fixedDuration, fixedRate)

	// Test 2: Stream approach (no per-doc frequency counting)
	t.Log("\n=== Stream Approach (no per-doc counting) ===")
	runtime.GC()

	type hashDocPair struct {
		hash  uint64
		docID uint32
	}
	shardBuffers := make([][]hashDocPair, 256)
	for i := range shardBuffers {
		shardBuffers[i] = make([]hashDocPair, 0, len(allTexts)*20)
	}
	var shardMu [256]sync.Mutex

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
			localShards := make([][]hashDocPair, 256)
			for i := range localShards {
				localShards[i] = make([]hashDocPair, 0, 256)
			}

			for docID := s; docID < e; docID++ {
				algo.NoCountTokenize(allTexts[docID], func(hash uint64) {
					shardID := hash & 0xFF
					localShards[shardID] = append(localShards[shardID], hashDocPair{hash, uint32(docID)})
				})
			}

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
	streamDuration := time.Since(start)
	streamRate := float64(len(allTexts)) / streamDuration.Seconds()
	t.Logf("  Time: %v, Rate: %.0f docs/sec", streamDuration, streamRate)

	// Summary
	t.Log("\n=== SUMMARY ===")
	t.Logf("FixedHashTable: %.0f docs/sec (baseline)", fixedRate)
	t.Logf("Stream (no count): %.0f docs/sec (%+.1f%%)", streamRate, (streamRate/fixedRate-1)*100)
	t.Logf("Gap to 1M from FixedHashTable: %.1fx", 1000000.0/fixedRate)
	t.Logf("Gap to 1M from Stream: %.1fx", 1000000.0/streamRate)
}

// scanWithHash does scanning with FNV hash but no frequency counting
func scanWithHash(text string) int {
	if len(text) == 0 {
		return 0
	}

	const fnvOffset = 14695981039346656037
	const fnvPrime = 1099511628211

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
		hash := uint64(fnvOffset)

		for i < n {
			c := testCharLUT[data[i]]
			if c == 0 {
				break
			}
			hash ^= uint64(c)
			hash *= fnvPrime
			i++
		}

		tokenLen := i - start
		if tokenLen >= 2 && tokenLen <= 32 {
			_ = hash // Use hash to prevent optimization
			tokenCount++
		}
	}

	return tokenCount
}
