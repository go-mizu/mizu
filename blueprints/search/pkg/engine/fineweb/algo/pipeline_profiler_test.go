package algo_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb"
)

// testCharLUT is pre-computed lookup table (same as megaToLower in algo package)
var testCharLUT = func() [256]byte {
	var lut [256]byte
	for i := 0; i < 256; i++ {
		if (i >= 'a' && i <= 'z') || (i >= '0' && i <= '9') {
			lut[i] = byte(i)
		} else if i >= 'A' && i <= 'Z' {
			lut[i] = byte(i | 0x20)
		}
	}
	return lut
}()

// ═══════════════════════════════════════════════════════════════════════════
// PHASE ISOLATION BENCHMARKS
// Each phase is measured in complete isolation to identify true bottlenecks
// ═══════════════════════════════════════════════════════════════════════════

// PhaseMetrics holds detailed metrics for each phase
type PhaseMetrics struct {
	Name         string
	Duration     time.Duration
	DocsPerSec   float64
	BytesPerSec  float64
	Bottleneck   string
	Improvement  string
}

func (m PhaseMetrics) String() string {
	return fmt.Sprintf("%-25s: %8.0f docs/sec | %s", m.Name, m.DocsPerSec, m.Bottleneck)
}

// TestIsolatedPhaseAnalysis runs each phase in complete isolation
func TestIsolatedPhaseAnalysis(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	home, _ := os.UserHomeDir()
	parquetDir := filepath.Join(home, "data", "fineweb-2", "vie_Latn", "train")

	if _, err := os.Stat(parquetDir); os.IsNotExist(err) {
		t.Skipf("Parquet directory not found: %s", parquetDir)
	}

	ctx := context.Background()
	targetDocs := 500000
	numWorkers := runtime.NumCPU() * 5

	t.Logf("CPU cores: %d, Workers: %d, Target: %d docs\n", runtime.NumCPU(), numWorkers, targetDocs)

	var metrics []PhaseMetrics

	// ═══════════════════════════════════════════════════════════════
	// PHASE 1: Pure Parquet Reading (I/O + zstd decompression)
	// ═══════════════════════════════════════════════════════════════
	t.Log("\n═══ PHASE 1: Parquet Reading ═══")
	var allTexts []string
	var totalBytes int64

	reader := fineweb.NewParquetReader(parquetDir).WithBatchSize(10000)
	start := time.Now()

	for batch, err := range reader.ReadTextsOnlyParallel(ctx, 8) {
		if err != nil {
			t.Fatal(err)
		}
		for _, doc := range batch {
			allTexts = append(allTexts, doc.Text)
			totalBytes += int64(len(doc.Text))
		}
		if len(allTexts) >= targetDocs {
			break
		}
	}
	phase1Duration := time.Since(start)
	phase1Rate := float64(len(allTexts)) / phase1Duration.Seconds()

	metrics = append(metrics, PhaseMetrics{
		Name:        "1. Parquet Read",
		Duration:    phase1Duration,
		DocsPerSec:  phase1Rate,
		BytesPerSec: float64(totalBytes) / phase1Duration.Seconds(),
		Bottleneck:  "zstd decompression (78% of read time)",
		Improvement: "CGO zstd or uncompressed data",
	})
	t.Logf("  Read %d docs (%d MB) in %v = %.0f docs/sec",
		len(allTexts), totalBytes/(1024*1024), phase1Duration, phase1Rate)

	// ═══════════════════════════════════════════════════════════════
	// PHASE 2A: Pure Tokenization (scanning + hashing only)
	// ═══════════════════════════════════════════════════════════════
	t.Log("\n═══ PHASE 2A: Pure Tokenization (no map, just scan+hash) ═══")
	runtime.GC()

	start = time.Now()
	var totalTokens atomic.Int64

	var wg sync.WaitGroup
	batchSize := (len(allTexts) + numWorkers - 1) / numWorkers

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
			localTokens := int64(0)
			for i := start; i < end; i++ {
				n := tokenizeScanOnly(allTexts[i])
				localTokens += int64(n)
			}
			totalTokens.Add(localTokens)
		}(startIdx, endIdx)
	}
	wg.Wait()
	phase2aDuration := time.Since(start)
	phase2aRate := float64(len(allTexts)) / phase2aDuration.Seconds()

	metrics = append(metrics, PhaseMetrics{
		Name:        "2A. Pure Scan+Hash",
		Duration:    phase2aDuration,
		DocsPerSec:  phase2aRate,
		Bottleneck:  "Character loop + FNV hash computation",
		Improvement: "SIMD/AVX2 vectorized scanning",
	})
	t.Logf("  Tokenized %d docs (%d tokens) in %v = %.0f docs/sec",
		len(allTexts), totalTokens.Load(), phase2aDuration, phase2aRate)

	// ═══════════════════════════════════════════════════════════════
	// PHASE 2B: Tokenization with Map (frequency counting)
	// ═══════════════════════════════════════════════════════════════
	t.Log("\n═══ PHASE 2B: Tokenization + Map (frequency counting) ═══")
	runtime.GC()

	type tokenResult struct {
		hashes []uint64
		freqs  []uint16
		docLen int
	}
	tokenResults := make([]tokenResult, len(allTexts))

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
				docLen := tokenizeWithMap(allTexts[i], freqs)

				// Extract results
				hashes := make([]uint64, 0, len(freqs))
				freqSlice := make([]uint16, 0, len(freqs))
				for h, f := range freqs {
					hashes = append(hashes, h)
					freqSlice = append(freqSlice, f)
				}
				tokenResults[i] = tokenResult{hashes, freqSlice, docLen}
			}
		}(startIdx, endIdx)
	}
	wg.Wait()
	phase2bDuration := time.Since(start)
	phase2bRate := float64(len(allTexts)) / phase2bDuration.Seconds()

	// Calculate map overhead
	mapOverhead := (phase2bDuration.Seconds() - phase2aDuration.Seconds()) / phase2bDuration.Seconds() * 100

	metrics = append(metrics, PhaseMetrics{
		Name:        "2B. Tokenize + Map",
		Duration:    phase2bDuration,
		DocsPerSec:  phase2bRate,
		Bottleneck:  fmt.Sprintf("Map operations add %.1f%% overhead", mapOverhead),
		Improvement: "Cannot avoid - map is fastest for freq counting",
	})
	t.Logf("  Tokenized with map in %v = %.0f docs/sec (map adds %.1f%% overhead)",
		phase2bDuration, phase2bRate, mapOverhead)

	// ═══════════════════════════════════════════════════════════════
	// PHASE 3: Shard Distribution
	// ═══════════════════════════════════════════════════════════════
	t.Log("\n═══ PHASE 3: Shard Distribution ═══")
	runtime.GC()

	type posting struct {
		hash  uint64
		docID uint32
		freq  uint16
	}
	shardBuffers := make([][]posting, 256)
	for i := range shardBuffers {
		shardBuffers[i] = make([]posting, 0, len(allTexts)*20/256)
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
		go func(start, end int) {
			defer wg.Done()
			localShards := make([][]posting, 256)
			for i := range localShards {
				localShards[i] = make([]posting, 0, 64)
			}

			for i := start; i < end; i++ {
				tr := tokenResults[i]
				for j, h := range tr.hashes {
					shardID := h & 0xFF
					localShards[shardID] = append(localShards[shardID],
						posting{h, uint32(i), tr.freqs[j]})
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
	phase3Duration := time.Since(start)
	phase3Rate := float64(len(allTexts)) / phase3Duration.Seconds()

	metrics = append(metrics, PhaseMetrics{
		Name:        "3. Shard Distribution",
		Duration:    phase3Duration,
		DocsPerSec:  phase3Rate,
		Bottleneck:  "Per-shard buffer allocation + mutex",
		Improvement: "Fuse with tokenization to eliminate",
	})
	t.Logf("  Distributed to shards in %v = %.0f docs/sec", phase3Duration, phase3Rate)

	// ═══════════════════════════════════════════════════════════════
	// PHASE 4: Posting Accumulation
	// ═══════════════════════════════════════════════════════════════
	t.Log("\n═══ PHASE 4: Posting Accumulation ═══")
	runtime.GC()

	type postingList struct {
		docIDs []uint32
		freqs  []uint16
	}
	shardMaps := make([]map[uint64]*postingList, 256)
	for i := range shardMaps {
		shardMaps[i] = make(map[uint64]*postingList, 10000)
	}

	start = time.Now()
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
	phase4Duration := time.Since(start)
	phase4Rate := float64(len(allTexts)) / phase4Duration.Seconds()

	metrics = append(metrics, PhaseMetrics{
		Name:        "4. Posting Accumulation",
		Duration:    phase4Duration,
		DocsPerSec:  phase4Rate,
		Bottleneck:  "Map lookups + slice appends",
		Improvement: "Already fast (2.8M docs/sec)",
	})
	t.Logf("  Built posting lists in %v = %.0f docs/sec", phase4Duration, phase4Rate)

	// ═══════════════════════════════════════════════════════════════
	// SUMMARY
	// ═══════════════════════════════════════════════════════════════
	t.Log("\n═══════════════════════════════════════════════════════════════")
	t.Log("PHASE BREAKDOWN SUMMARY")
	t.Log("═══════════════════════════════════════════════════════════════")

	totalTime := phase1Duration + phase2bDuration + phase3Duration + phase4Duration
	for _, m := range metrics {
		pct := float64(m.Duration) / float64(totalTime) * 100
		t.Logf("%-25s: %8.0f docs/sec (%5.1f%%) | %s",
			m.Name, m.DocsPerSec, pct, m.Bottleneck)
	}

	combinedRate := float64(len(allTexts)) / totalTime.Seconds()
	t.Logf("\n%-25s: %8.0f docs/sec", "Combined (sequential)", combinedRate)

	// Theoretical max if phases were perfectly parallelized
	slowestPhase := phase1Duration
	if phase2bDuration > slowestPhase {
		slowestPhase = phase2bDuration
	}
	theoreticalMax := float64(len(allTexts)) / slowestPhase.Seconds()
	t.Logf("%-25s: %8.0f docs/sec", "Theoretical max (parallel)", theoreticalMax)

	// Gap to 1M
	t.Logf("\n%-25s: %8.0f docs/sec", "Target", 1000000.0)
	t.Logf("%-25s: %8.1fx", "Gap from current", 1000000.0/combinedRate)
	t.Logf("%-25s: %8.1fx", "Gap from theoretical", 1000000.0/theoreticalMax)
}

// tokenizeScanOnly does pure scanning and hashing without storing results
func tokenizeScanOnly(text string) int {
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
		// Skip delimiters
		for i < n && testCharLUT[data[i]] == 0 {
			i++
		}
		if i >= n {
			break
		}

		start := i
		hash := uint64(fnvOffset)

		// Hash while scanning
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
			// Just count, don't store
			_ = hash
			tokenCount++
		}
	}

	return tokenCount
}

// tokenizeWithMap does tokenization with frequency counting via map
func tokenizeWithMap(text string, freqs map[uint64]uint16) int {
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
			freqs[hash]++
			tokenCount++
		}
	}

	return tokenCount
}

// TestProfileCriticalPath profiles the most critical code path
func TestProfileCriticalPath(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	home, _ := os.UserHomeDir()
	parquetDir := filepath.Join(home, "data", "fineweb-2", "vie_Latn", "train")

	if _, err := os.Stat(parquetDir); os.IsNotExist(err) {
		t.Skipf("Parquet directory not found: %s", parquetDir)
	}

	// Pre-load data
	ctx := context.Background()
	reader := fineweb.NewParquetReader(parquetDir).WithBatchSize(10000)
	var texts []string
	for batch, err := range reader.ReadTextsOnlyParallel(ctx, 8) {
		if err != nil {
			t.Fatal(err)
		}
		for _, doc := range batch {
			texts = append(texts, doc.Text)
		}
		if len(texts) >= 1000000 {
			break
		}
	}
	t.Logf("Pre-loaded %d docs", len(texts))

	// Create CPU profile
	f, err := os.Create("/tmp/critical_path.prof")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	numWorkers := runtime.NumCPU() * 5
	batchSize := (len(texts) + numWorkers - 1) / numWorkers

	t.Log("Starting profiled run...")
	pprof.StartCPUProfile(f)

	var wg sync.WaitGroup
	start := time.Now()

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
			freqs := make(map[uint64]uint16, 256)
			for i := start; i < end; i++ {
				clear(freqs)
				tokenizeWithMap(texts[i], freqs)
			}
		}(startIdx, endIdx)
	}
	wg.Wait()

	pprof.StopCPUProfile()
	elapsed := time.Since(start)

	t.Logf("Completed in %v = %.0f docs/sec", elapsed, float64(len(texts))/elapsed.Seconds())
	t.Log("Profile saved to /tmp/critical_path.prof")
	t.Log("Analyze with: go tool pprof -http=:8080 /tmp/critical_path.prof")
}

// freqMapPool is a sync.Pool for reusing frequency maps
var freqMapPool = sync.Pool{
	New: func() interface{} {
		return make(map[uint64]uint16, 256)
	},
}

// TestFixedHashTableVsMap compares fixed hash table vs Go map for tokenization
func TestFixedHashTableVsMap(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	home, _ := os.UserHomeDir()
	parquetDir := filepath.Join(home, "data", "fineweb-2", "vie_Latn", "train")

	if _, err := os.Stat(parquetDir); os.IsNotExist(err) {
		t.Skipf("Parquet directory not found: %s", parquetDir)
	}

	// Pre-load data
	ctx := context.Background()
	reader := fineweb.NewParquetReader(parquetDir).WithBatchSize(10000)
	var texts []string
	for batch, err := range reader.ReadTextsOnlyParallel(ctx, 8) {
		if err != nil {
			t.Fatal(err)
		}
		for _, doc := range batch {
			texts = append(texts, doc.Text)
		}
		if len(texts) >= 500000 {
			break
		}
	}
	t.Logf("Loaded %d docs", len(texts))

	numWorkers := runtime.NumCPU() * 5
	batchSize := (len(texts) + numWorkers - 1) / numWorkers

	// Test 1: Go map (baseline)
	t.Log("\n=== Go Map (baseline) ===")
	runtime.GC()

	start := time.Now()
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
			freqs := make(map[uint64]uint16, 256)
			for i := start; i < end; i++ {
				clear(freqs)
				tokenizeWithMap(texts[i], freqs)
			}
		}(startIdx, endIdx)
	}
	wg.Wait()
	mapDuration := time.Since(start)
	mapRate := float64(len(texts)) / mapDuration.Seconds()
	t.Logf("  Time: %v, Rate: %.0f docs/sec", mapDuration, mapRate)

	// Test 2: Fixed Hash Table
	t.Log("\n=== Fixed Hash Table ===")
	runtime.GC()

	start = time.Now()
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
			table := newFixedHashTable(4096) // Large enough for any document
			for i := start; i < end; i++ {
				tokenizeFixed(texts[i], table)
			}
		}(startIdx, endIdx)
	}
	wg.Wait()
	fixedDuration := time.Since(start)
	fixedRate := float64(len(texts)) / fixedDuration.Seconds()
	t.Logf("  Time: %v, Rate: %.0f docs/sec", fixedDuration, fixedRate)

	// Test 3: Compact Hash Table
	t.Log("\n=== Compact Hash Table ===")
	runtime.GC()

	start = time.Now()
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
			table := newCompactHashTable(4096) // Large enough for any document
			for i := start; i < end; i++ {
				tokenizeCompact(texts[i], table)
			}
		}(startIdx, endIdx)
	}
	wg.Wait()
	compactDuration := time.Since(start)
	compactRate := float64(len(texts)) / compactDuration.Seconds()
	t.Logf("  Time: %v, Rate: %.0f docs/sec", compactDuration, compactRate)

	// Test 4: Pooled Map
	t.Log("\n=== Pooled Map (sync.Pool) ===")
	runtime.GC()

	start = time.Now()
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
			for i := start; i < end; i++ {
				freqs := freqMapPool.Get().(map[uint64]uint16)
				clear(freqs)
				tokenizeWithMap(texts[i], freqs)
				freqMapPool.Put(freqs)
			}
		}(startIdx, endIdx)
	}
	wg.Wait()
	pooledDuration := time.Since(start)
	pooledRate := float64(len(texts)) / pooledDuration.Seconds()
	t.Logf("  Time: %v, Rate: %.0f docs/sec", pooledDuration, pooledRate)

	// Test 5: Pre-sized Map (larger initial capacity)
	t.Log("\n=== Pre-sized Map (1024 initial capacity) ===")
	runtime.GC()

	start = time.Now()
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
			freqs := make(map[uint64]uint16, 1024) // Larger initial capacity
			for i := start; i < end; i++ {
				clear(freqs)
				tokenizeWithMap(texts[i], freqs)
			}
		}(startIdx, endIdx)
	}
	wg.Wait()
	presizedDuration := time.Since(start)
	presizedRate := float64(len(texts)) / presizedDuration.Seconds()
	t.Logf("  Time: %v, Rate: %.0f docs/sec", presizedDuration, presizedRate)

	// Summary
	t.Log("\n=== SUMMARY ===")
	t.Logf("Go Map (256 cap):   %.0f docs/sec (baseline)", mapRate)
	t.Logf("Fixed Hash Table:   %.0f docs/sec (%+.1f%% vs map)", fixedRate, (fixedRate/mapRate-1)*100)
	t.Logf("Compact Hash Table: %.0f docs/sec (%+.1f%% vs map)", compactRate, (compactRate/mapRate-1)*100)
	t.Logf("Pooled Map:         %.0f docs/sec (%+.1f%% vs map)", pooledRate, (pooledRate/mapRate-1)*100)
	t.Logf("Pre-sized Map:      %.0f docs/sec (%+.1f%% vs map)", presizedRate, (presizedRate/mapRate-1)*100)
}

// Local implementations for testing (to avoid import cycle with algo package)
type fixedHashTable struct {
	keys      []uint64
	counts    []uint16
	usedSlots []int // Track which slots are used
	mask      uint64
	used      int
}

func newFixedHashTable(capacity int) *fixedHashTable {
	size := 1
	for size < capacity*2 {
		size *= 2
	}
	return &fixedHashTable{
		keys:      make([]uint64, size),
		counts:    make([]uint16, size),
		usedSlots: make([]int, 0, capacity),
		mask:      uint64(size - 1),
	}
}

func (h *fixedHashTable) reset() {
	// Only clear used slots for efficiency
	for _, idx := range h.usedSlots {
		h.keys[idx] = 0
		h.counts[idx] = 0
	}
	h.usedSlots = h.usedSlots[:0]
	h.used = 0
}

func (h *fixedHashTable) insert(hash uint64) bool {
	if hash == 0 {
		hash = 1
	}
	idx := hash & h.mask
	size := int(h.mask) + 1
	// Limit probing to prevent infinite loop
	for i := 0; i < size; i++ {
		if h.keys[idx] == 0 {
			h.keys[idx] = hash
			h.counts[idx] = 1
			h.usedSlots = append(h.usedSlots, int(idx)) // Track used slot
			h.used++
			return true
		}
		if h.keys[idx] == hash {
			h.counts[idx]++
			return false
		}
		idx = (idx + 1) & h.mask
	}
	// Table is full - just ignore
	return false
}

func tokenizeFixed(text string, table *fixedHashTable) int {
	if len(text) == 0 {
		return 0
	}

	const fnvOffset = 14695981039346656037
	const fnvPrime = 1099511628211

	data := unsafe.Slice(unsafe.StringData(text), len(text))
	n := len(data)
	tokenCount := 0
	i := 0

	table.reset()

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
			table.insert(hash)
			tokenCount++
		}
	}

	return tokenCount
}

type compactHashTable struct {
	keys   []uint64
	counts []uint16
	size   int
}

func newCompactHashTable(capacity int) *compactHashTable {
	size := 1
	for size < capacity*2 {
		size *= 2
	}
	return &compactHashTable{
		keys:   make([]uint64, size),
		counts: make([]uint16, size),
		size:   size,
	}
}

func (h *compactHashTable) reset() {
	clear(h.keys)
	clear(h.counts)
}

func (h *compactHashTable) insert(hash uint64) bool {
	if hash == 0 {
		hash = 1
	}
	mask := uint64(h.size - 1)
	idx := hash & mask
	for i := 0; i < h.size; i++ {
		if h.keys[idx] == 0 {
			h.keys[idx] = hash
			h.counts[idx] = 1
			return true
		}
		if h.keys[idx] == hash {
			h.counts[idx]++
			return false
		}
		idx = (idx + 1) & mask
	}
	return false
}

func tokenizeCompact(text string, table *compactHashTable) int {
	if len(text) == 0 {
		return 0
	}

	const fnvOffset = 14695981039346656037
	const fnvPrime = 1099511628211

	data := unsafe.Slice(unsafe.StringData(text), len(text))
	n := len(data)
	tokenCount := 0
	i := 0

	table.reset()

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
			table.insert(hash)
			tokenCount++
		}
	}

	return tokenCount
}

// TestFusedTokenizeAndShard tests tokenization fused with shard distribution
func TestFusedTokenizeAndShard(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	home, _ := os.UserHomeDir()
	parquetDir := filepath.Join(home, "data", "fineweb-2", "vie_Latn", "train")

	if _, err := os.Stat(parquetDir); os.IsNotExist(err) {
		t.Skipf("Parquet directory not found: %s", parquetDir)
	}

	ctx := context.Background()
	targetDocs := 500000
	numWorkers := runtime.NumCPU() * 5

	t.Logf("CPU cores: %d, Workers: %d, Target: %d docs\n", runtime.NumCPU(), numWorkers, targetDocs)

	// Load data
	t.Log("\n=== Loading data ===")
	var allTexts []string
	reader := fineweb.NewParquetReader(parquetDir).WithBatchSize(10000)
	start := time.Now()

	for batch, err := range reader.ReadTextsOnlyParallel(ctx, 8) {
		if err != nil {
			t.Fatal(err)
		}
		for _, doc := range batch {
			allTexts = append(allTexts, doc.Text)
		}
		if len(allTexts) >= targetDocs {
			break
		}
	}
	loadDuration := time.Since(start)
	t.Logf("Loaded %d docs in %v = %.0f docs/sec", len(allTexts), loadDuration, float64(len(allTexts))/loadDuration.Seconds())

	// Fused Tokenization + Shard Distribution
	t.Log("\n=== Fused Tokenization + Shard Distribution ===")
	runtime.GC()

	type posting struct {
		hash  uint64
		docID uint32
		freq  uint16
	}
	shardBuffers := make([][]posting, 256)
	for i := range shardBuffers {
		shardBuffers[i] = make([]posting, 0, len(allTexts)*20/256)
	}
	var shardMu [256]sync.Mutex
	batchSize := (len(allTexts) + numWorkers - 1) / numWorkers

	var wg sync.WaitGroup
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
			table := newFixedHashTable(4096)
			localShards := make([][]posting, 256)
			for i := range localShards {
				localShards[i] = make([]posting, 0, 64)
			}

			for docID := start; docID < end; docID++ {
				tokenizeFixed(allTexts[docID], table)

				// Direct shard distribution from table using usedSlots
				for _, idx := range table.usedSlots {
					hash := table.keys[idx]
					shardID := hash & 0xFF
					localShards[shardID] = append(localShards[shardID],
						posting{hash, uint32(docID), table.counts[idx]})
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
	fusedDuration := time.Since(start)
	fusedRate := float64(len(allTexts)) / fusedDuration.Seconds()
	t.Logf("Fused tokenize+shard in %v = %.0f docs/sec", fusedDuration, fusedRate)

	// Posting Accumulation
	t.Log("\n=== Posting Accumulation ===")
	runtime.GC()

	type postingList struct {
		docIDs []uint32
		freqs  []uint16
	}
	shardMaps := make([]map[uint64]*postingList, 256)
	for i := range shardMaps {
		shardMaps[i] = make(map[uint64]*postingList, 10000)
	}

	start = time.Now()
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
	postingDuration := time.Since(start)
	postingRate := float64(len(allTexts)) / postingDuration.Seconds()
	t.Logf("Built posting lists in %v = %.0f docs/sec", postingDuration, postingRate)

	// Summary
	t.Log("\n═══════════════════════════════════════════════════════════════")
	t.Log("FUSED PIPELINE (TOKENIZE + SHARD)")
	t.Log("═══════════════════════════════════════════════════════════════")

	totalTime := loadDuration + fusedDuration + postingDuration
	t.Logf("%-30s: %8.0f docs/sec (%5.1f%%)", "1. Data Load", float64(len(allTexts))/loadDuration.Seconds(),
		float64(loadDuration)/float64(totalTime)*100)
	t.Logf("%-30s: %8.0f docs/sec (%5.1f%%)", "2. Tokenize+Shard (Fused)", fusedRate,
		float64(fusedDuration)/float64(totalTime)*100)
	t.Logf("%-30s: %8.0f docs/sec (%5.1f%%)", "3. Posting Accumulation", postingRate,
		float64(postingDuration)/float64(totalTime)*100)

	combinedRate := float64(len(allTexts)) / totalTime.Seconds()
	t.Logf("\n%-30s: %8.0f docs/sec", "Combined (sequential)", combinedRate)

	// Theoretical max
	slowest := fusedDuration
	if loadDuration > slowest {
		slowest = loadDuration
	}
	theoreticalMax := float64(len(allTexts)) / slowest.Seconds()
	t.Logf("%-30s: %8.0f docs/sec", "Theoretical max (parallel)", theoreticalMax)

	// Gap to 1M
	t.Logf("\n%-30s: %8.0f docs/sec", "Target", 1000000.0)
	t.Logf("%-30s: %8.1fx", "Gap from current", 1000000.0/combinedRate)
	t.Logf("%-30s: %8.1fx", "Gap from theoretical", 1000000.0/theoreticalMax)
}

// TestFullPipelineWithFixedTokenize tests the full pipeline using FixedTokenize
func TestFullPipelineWithFixedTokenize(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	home, _ := os.UserHomeDir()
	parquetDir := filepath.Join(home, "data", "fineweb-2", "vie_Latn", "train")

	if _, err := os.Stat(parquetDir); os.IsNotExist(err) {
		t.Skipf("Parquet directory not found: %s", parquetDir)
	}

	ctx := context.Background()
	targetDocs := 500000
	numWorkers := runtime.NumCPU() * 5

	t.Logf("CPU cores: %d, Workers: %d, Target: %d docs\n", runtime.NumCPU(), numWorkers, targetDocs)

	// Load data
	t.Log("\n=== Loading data ===")
	var allTexts []string
	reader := fineweb.NewParquetReader(parquetDir).WithBatchSize(10000)
	start := time.Now()

	for batch, err := range reader.ReadTextsOnlyParallel(ctx, 8) {
		if err != nil {
			t.Fatal(err)
		}
		for _, doc := range batch {
			allTexts = append(allTexts, doc.Text)
		}
		if len(allTexts) >= targetDocs {
			break
		}
	}
	loadDuration := time.Since(start)
	t.Logf("Loaded %d docs in %v = %.0f docs/sec", len(allTexts), loadDuration, float64(len(allTexts))/loadDuration.Seconds())

	// Tokenization with FixedHashTable
	t.Log("\n=== Tokenization (FixedHashTable) ===")
	runtime.GC()

	type tokenResult struct {
		hashes []uint64
		freqs  []uint16
	}
	tokenResults := make([]tokenResult, len(allTexts))
	batchSize := (len(allTexts) + numWorkers - 1) / numWorkers

	var wg sync.WaitGroup
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
			table := newFixedHashTable(4096) // Large enough for any document

			for i := start; i < end; i++ {
				tokenizeFixed(allTexts[i], table)

				// Extract results using usedSlots for efficiency
				hashes := make([]uint64, 0, len(table.usedSlots))
				freqs := make([]uint16, 0, len(table.usedSlots))
				for _, idx := range table.usedSlots {
					hashes = append(hashes, table.keys[idx])
					freqs = append(freqs, table.counts[idx])
				}
				tokenResults[i] = tokenResult{hashes, freqs}
			}
		}(startIdx, endIdx)
	}
	wg.Wait()
	tokenizeDuration := time.Since(start)
	tokenizeRate := float64(len(allTexts)) / tokenizeDuration.Seconds()
	t.Logf("Tokenized in %v = %.0f docs/sec", tokenizeDuration, tokenizeRate)

	// Shard Distribution
	t.Log("\n=== Shard Distribution ===")
	runtime.GC()

	type posting struct {
		hash  uint64
		docID uint32
		freq  uint16
	}
	shardBuffers := make([][]posting, 256)
	for i := range shardBuffers {
		shardBuffers[i] = make([]posting, 0, len(allTexts)*20/256)
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
		go func(start, end int) {
			defer wg.Done()
			localShards := make([][]posting, 256)
			for i := range localShards {
				localShards[i] = make([]posting, 0, 64)
			}

			for i := start; i < end; i++ {
				tr := tokenResults[i]
				for j, h := range tr.hashes {
					shardID := h & 0xFF
					localShards[shardID] = append(localShards[shardID],
						posting{h, uint32(i), tr.freqs[j]})
				}
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
	shardDuration := time.Since(start)
	shardRate := float64(len(allTexts)) / shardDuration.Seconds()
	t.Logf("Distributed in %v = %.0f docs/sec", shardDuration, shardRate)

	// Posting Accumulation
	t.Log("\n=== Posting Accumulation ===")
	runtime.GC()

	type postingList struct {
		docIDs []uint32
		freqs  []uint16
	}
	shardMaps := make([]map[uint64]*postingList, 256)
	for i := range shardMaps {
		shardMaps[i] = make(map[uint64]*postingList, 10000)
	}

	start = time.Now()
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
	postingDuration := time.Since(start)
	postingRate := float64(len(allTexts)) / postingDuration.Seconds()
	t.Logf("Built posting lists in %v = %.0f docs/sec", postingDuration, postingRate)

	// Summary
	t.Log("\n═══════════════════════════════════════════════════════════════")
	t.Log("PIPELINE WITH FIXED HASH TABLE")
	t.Log("═══════════════════════════════════════════════════════════════")

	totalTime := loadDuration + tokenizeDuration + shardDuration + postingDuration
	t.Logf("%-25s: %8.0f docs/sec (%5.1f%%)", "1. Data Load", float64(len(allTexts))/loadDuration.Seconds(),
		float64(loadDuration)/float64(totalTime)*100)
	t.Logf("%-25s: %8.0f docs/sec (%5.1f%%)", "2. Tokenization (Fixed)", tokenizeRate,
		float64(tokenizeDuration)/float64(totalTime)*100)
	t.Logf("%-25s: %8.0f docs/sec (%5.1f%%)", "3. Shard Distribution", shardRate,
		float64(shardDuration)/float64(totalTime)*100)
	t.Logf("%-25s: %8.0f docs/sec (%5.1f%%)", "4. Posting Accumulation", postingRate,
		float64(postingDuration)/float64(totalTime)*100)

	combinedRate := float64(len(allTexts)) / totalTime.Seconds()
	t.Logf("\n%-25s: %8.0f docs/sec", "Combined (sequential)", combinedRate)

	// Theoretical max
	slowest := tokenizeDuration
	if loadDuration > slowest {
		slowest = loadDuration
	}
	theoreticalMax := float64(len(allTexts)) / slowest.Seconds()
	t.Logf("%-25s: %8.0f docs/sec", "Theoretical max (parallel)", theoreticalMax)

	// Gap to 1M
	t.Logf("\n%-25s: %8.0f docs/sec", "Target", 1000000.0)
	t.Logf("%-25s: %8.1fx", "Gap from current", 1000000.0/combinedRate)
	t.Logf("%-25s: %8.1fx", "Gap from theoretical", 1000000.0/theoreticalMax)
}

// BenchmarkTokenizationVariants compares different tokenization approaches
func BenchmarkTokenizationVariants(b *testing.B) {
	if testing.Short() {
		b.Skip()
	}

	home, _ := os.UserHomeDir()
	parquetDir := filepath.Join(home, "data", "fineweb-2", "vie_Latn", "train")

	if _, err := os.Stat(parquetDir); os.IsNotExist(err) {
		b.Skipf("Parquet directory not found: %s", parquetDir)
	}

	// Pre-load data
	ctx := context.Background()
	reader := fineweb.NewParquetReader(parquetDir).WithBatchSize(10000)
	var texts []string
	for batch, err := range reader.ReadTextsOnlyParallel(ctx, 8) {
		if err != nil {
			b.Fatal(err)
		}
		for _, doc := range batch {
			texts = append(texts, doc.Text)
		}
		if len(texts) >= 100000 {
			break
		}
	}

	numWorkers := runtime.NumCPU() * 5
	batchSize := (len(texts) + numWorkers - 1) / numWorkers

	b.Run("ScanOnly", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
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
					for j := start; j < end; j++ {
						tokenizeScanOnly(texts[j])
					}
				}(startIdx, endIdx)
			}
			wg.Wait()
		}
		b.ReportMetric(float64(len(texts)*b.N)/b.Elapsed().Seconds(), "docs/sec")
	})

	b.Run("WithMap", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
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
					freqs := make(map[uint64]uint16, 256)
					for j := start; j < end; j++ {
						clear(freqs)
						tokenizeWithMap(texts[j], freqs)
					}
				}(startIdx, endIdx)
			}
			wg.Wait()
		}
		b.ReportMetric(float64(len(texts)*b.N)/b.Elapsed().Seconds(), "docs/sec")
	})
}
