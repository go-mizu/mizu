package algo_test

import (
	"context"
	"fmt"
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

// PhaseResults holds timing for each phase
type PhaseResults struct {
	ParquetReadTime   time.Duration
	TokenizeTime      time.Duration
	ShardDistribTime  time.Duration
	PostingAccumTime  time.Duration
	TotalDocs         int
	TotalBytes        int64
}

func (p PhaseResults) String() string {
	total := p.ParquetReadTime + p.TokenizeTime + p.ShardDistribTime + p.PostingAccumTime
	return fmt.Sprintf(`
Phase Breakdown (%d docs, %d MB):
  1. Parquet Read:     %v (%.1f%%) - %.0f docs/sec
  2. Tokenization:     %v (%.1f%%) - %.0f docs/sec
  3. Shard Distribute: %v (%.1f%%) - %.0f docs/sec
  4. Posting Accum:    %v (%.1f%%) - %.0f docs/sec
  ─────────────────────────────────
  Total:               %v - %.0f docs/sec
`,
		p.TotalDocs, p.TotalBytes/(1024*1024),
		p.ParquetReadTime, 100*float64(p.ParquetReadTime)/float64(total), float64(p.TotalDocs)/p.ParquetReadTime.Seconds(),
		p.TokenizeTime, 100*float64(p.TokenizeTime)/float64(total), float64(p.TotalDocs)/p.TokenizeTime.Seconds(),
		p.ShardDistribTime, 100*float64(p.ShardDistribTime)/float64(total), float64(p.TotalDocs)/p.ShardDistribTime.Seconds(),
		p.PostingAccumTime, 100*float64(p.PostingAccumTime)/float64(total), float64(p.TotalDocs)/p.PostingAccumTime.Seconds(),
		total, float64(p.TotalDocs)/total.Seconds(),
	)
}

// TestPhaseBreakdown measures each phase separately
func TestPhaseBreakdown(t *testing.T) {
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

	var results PhaseResults

	// ═══════════════════════════════════════════════════════════════
	// PHASE 1: Parquet Reading (I/O + zstd decompression)
	// ═══════════════════════════════════════════════════════════════
	t.Log("Phase 1: Measuring parquet read time...")
	var allTexts []string

	reader := fineweb.NewParquetReader(parquetDir).WithBatchSize(10000)
	start := time.Now()

	for batch, err := range reader.ReadTextsOnlyParallel(ctx, 8) {
		if err != nil {
			t.Fatal(err)
		}
		for _, doc := range batch {
			allTexts = append(allTexts, doc.Text)
			results.TotalBytes += int64(len(doc.Text))
		}
		if len(allTexts) >= targetDocs {
			break
		}
	}
	results.ParquetReadTime = time.Since(start)
	results.TotalDocs = len(allTexts)
	t.Logf("  Parquet read: %v for %d docs (%.0f docs/sec)",
		results.ParquetReadTime, results.TotalDocs,
		float64(results.TotalDocs)/results.ParquetReadTime.Seconds())

	// ═══════════════════════════════════════════════════════════════
	// PHASE 2: Tokenization only (no shard distribution)
	// ═══════════════════════════════════════════════════════════════
	t.Log("Phase 2: Measuring tokenization time...")
	runtime.GC()

	numWorkers := runtime.NumCPU() * 5
	type tokenResult struct {
		hashes []uint64
		freqs  []uint16
		docLen int
	}
	tokenResults := make([]tokenResult, len(allTexts))

	start = time.Now()
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
			freqs := make(map[uint64]uint16, 256)

			for i := start; i < end; i++ {
				clear(freqs)
				docLen := tokenizeForBench(allTexts[i], freqs)

				// Collect results
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
	results.TokenizeTime = time.Since(start)
	t.Logf("  Tokenization: %v for %d docs (%.0f docs/sec)",
		results.TokenizeTime, results.TotalDocs,
		float64(results.TotalDocs)/results.TokenizeTime.Seconds())

	// ═══════════════════════════════════════════════════════════════
	// PHASE 3: Shard Distribution only
	// ═══════════════════════════════════════════════════════════════
	t.Log("Phase 3: Measuring shard distribution time...")
	runtime.GC()

	type posting struct {
		hash  uint64
		docID uint32
		freq  uint16
	}
	shardBuffers := make([][]posting, 256)
	for i := range shardBuffers {
		shardBuffers[i] = make([]posting, 0, len(allTexts)*10/256)
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
	results.ShardDistribTime = time.Since(start)
	t.Logf("  Shard distribution: %v for %d docs (%.0f docs/sec)",
		results.ShardDistribTime, results.TotalDocs,
		float64(results.TotalDocs)/results.ShardDistribTime.Seconds())

	// ═══════════════════════════════════════════════════════════════
	// PHASE 4: Posting Accumulation (building inverted index)
	// ═══════════════════════════════════════════════════════════════
	t.Log("Phase 4: Measuring posting accumulation time...")
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
	results.PostingAccumTime = time.Since(start)
	t.Logf("  Posting accumulation: %v for %d docs (%.0f docs/sec)",
		results.PostingAccumTime, results.TotalDocs,
		float64(results.TotalDocs)/results.PostingAccumTime.Seconds())

	// Print summary
	t.Log(results.String())
}

// tokenizeForBench is a standalone tokenizer for benchmarking
func tokenizeForBench(text string, freqs map[uint64]uint16) int {
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
		for i < n && !isAlphaNum(data[i]) {
			i++
		}
		if i >= n {
			break
		}

		start := i
		hash := uint64(fnvOffset)

		// Hash while scanning
		for i < n {
			c := toLower(data[i])
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

func isAlphaNum(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

func toLower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c | 0x20
	}
	if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
		return c
	}
	return 0 // delimiter
}

// ═══════════════════════════════════════════════════════════════════════════
// INDIVIDUAL PHASE BENCHMARKS
// ═══════════════════════════════════════════════════════════════════════════

// BenchmarkPhase1_ParquetRead measures pure parquet reading speed
func BenchmarkPhase1_ParquetRead(b *testing.B) {
	if testing.Short() {
		b.Skip()
	}

	home, _ := os.UserHomeDir()
	parquetDir := filepath.Join(home, "data", "fineweb-2", "vie_Latn", "train")

	if _, err := os.Stat(parquetDir); os.IsNotExist(err) {
		b.Skipf("Parquet directory not found: %s", parquetDir)
	}

	ctx := context.Background()
	targetDocs := 100000

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := fineweb.NewParquetReader(parquetDir).WithBatchSize(10000)
		count := 0
		for batch, err := range reader.ReadTextsOnlyParallel(ctx, 8) {
			if err != nil {
				b.Fatal(err)
			}
			count += len(batch)
			if count >= targetDocs {
				break
			}
		}
		b.ReportMetric(float64(count)/b.Elapsed().Seconds(), "docs/sec")
	}
}

// BenchmarkPhase2_Tokenization measures pure tokenization speed
func BenchmarkPhase2_Tokenization(b *testing.B) {
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
	b.Logf("Pre-loaded %d docs for tokenization", len(texts))

	numWorkers := runtime.NumCPU() * 5

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		batchSize := (len(texts) + numWorkers - 1) / numWorkers

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
					tokenizeForBench(texts[j], freqs)
				}
			}(startIdx, endIdx)
		}
		wg.Wait()
	}
	b.ReportMetric(float64(len(texts))/b.Elapsed().Seconds()*float64(b.N), "docs/sec")
}

// BenchmarkPhase2_TokenizationVariants compares tokenization approaches
func BenchmarkPhase2_TokenizationVariants(b *testing.B) {
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
		if len(texts) >= 50000 {
			break
		}
	}

	b.Run("TokenizeMega", func(b *testing.B) {
		freqs := make(map[uint64]uint16, 256)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, text := range texts {
				algo.TokenizeMega(text, freqs)
			}
		}
		b.ReportMetric(float64(len(texts)*b.N)/b.Elapsed().Seconds(), "docs/sec")
	})

	b.Run("TokenizeMegaV2", func(b *testing.B) {
		freqs := make(map[uint64]uint16, 256)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, text := range texts {
				algo.TokenizeMegaV2(text, freqs)
			}
		}
		b.ReportMetric(float64(len(texts)*b.N)/b.Elapsed().Seconds(), "docs/sec")
	})

	b.Run("TokenizeMegaV3", func(b *testing.B) {
		freqs := make(map[uint64]uint16, 256)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, text := range texts {
				algo.TokenizeMegaV3(text, freqs)
			}
		}
		b.ReportMetric(float64(len(texts)*b.N)/b.Elapsed().Seconds(), "docs/sec")
	})

	b.Run("TokenizeHyper", func(b *testing.B) {
		freqs := make(map[uint64]uint16, 256)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, text := range texts {
				algo.TokenizeHyper(text, freqs)
			}
		}
		b.ReportMetric(float64(len(texts)*b.N)/b.Elapsed().Seconds(), "docs/sec")
	})
}

// BenchmarkPhase3_ShardDistribution measures shard distribution overhead
func BenchmarkPhase3_ShardDistribution(b *testing.B) {
	if testing.Short() {
		b.Skip()
	}

	// Create synthetic tokenization results
	numDocs := 100000
	avgTokens := 50

	type tokenResult struct {
		hashes []uint64
		freqs  []uint16
	}
	tokenResults := make([]tokenResult, numDocs)
	for i := range tokenResults {
		hashes := make([]uint64, avgTokens)
		freqs := make([]uint16, avgTokens)
		for j := range hashes {
			hashes[j] = uint64(i*avgTokens+j) * 0x9E3779B97F4A7C15
			freqs[j] = 1
		}
		tokenResults[i] = tokenResult{hashes, freqs}
	}

	numWorkers := runtime.NumCPU() * 5
	type posting struct {
		hash  uint64
		docID uint32
		freq  uint16
	}

	b.ResetTimer()
	for iter := 0; iter < b.N; iter++ {
		shardBuffers := make([][]posting, 256)
		for i := range shardBuffers {
			shardBuffers[i] = make([]posting, 0, numDocs*avgTokens/256)
		}
		var shardMu [256]sync.Mutex

		var wg sync.WaitGroup
		batchSize := (numDocs + numWorkers - 1) / numWorkers

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
	}
	b.ReportMetric(float64(numDocs*b.N)/b.Elapsed().Seconds(), "docs/sec")
}

// BenchmarkPhase4_PostingAccumulation measures posting list building
func BenchmarkPhase4_PostingAccumulation(b *testing.B) {
	if testing.Short() {
		b.Skip()
	}

	// Create synthetic shard buffers
	numDocs := 100000
	avgTokens := 50
	type posting struct {
		hash  uint64
		docID uint32
		freq  uint16
	}

	shardBuffers := make([][]posting, 256)
	for shardID := range shardBuffers {
		// Each shard gets ~numDocs*avgTokens/256 postings
		n := numDocs * avgTokens / 256
		shardBuffers[shardID] = make([]posting, n)
		for i := range shardBuffers[shardID] {
			shardBuffers[shardID][i] = posting{
				hash:  uint64(shardID)<<56 | uint64(i%10000),
				docID: uint32(i % numDocs),
				freq:  1,
			}
		}
	}

	numWorkers := runtime.NumCPU() * 5

	b.ResetTimer()
	for iter := 0; iter < b.N; iter++ {
		type postingList struct {
			docIDs []uint32
			freqs  []uint16
		}
		shardMaps := make([]map[uint64]*postingList, 256)
		for i := range shardMaps {
			shardMaps[i] = make(map[uint64]*postingList, 10000)
		}

		var wg sync.WaitGroup
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
	b.ReportMetric(float64(numDocs*b.N)/b.Elapsed().Seconds(), "docs/sec")
}

// BenchmarkEndToEnd_Comparison compares full pipeline variants
func BenchmarkEndToEnd_Comparison(b *testing.B) {
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
	var docIDs []uint32
	for batch, err := range reader.ReadTextsOnlyParallel(ctx, 8) {
		if err != nil {
			b.Fatal(err)
		}
		for _, doc := range batch {
			docIDs = append(docIDs, uint32(len(texts)))
			texts = append(texts, doc.Text)
		}
		if len(texts) >= 500000 {
			break
		}
	}
	b.Logf("Pre-loaded %d docs", len(texts))

	batchSize := 10000

	b.Run("MegaIndexer", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			runtime.GC()
			tmpDir := b.TempDir()
			indexer := algo.NewMegaIndexer(tmpDir, algo.MegaConfig{
				NumWorkers: runtime.NumCPU() * 4,
			})

			for j := 0; j < len(texts); j += batchSize {
				end := j + batchSize
				if end > len(texts) {
					end = len(texts)
				}
				indexer.AddBatch(docIDs[j:end], texts[j:end])
			}
		}
		b.ReportMetric(float64(len(texts)*b.N)/b.Elapsed().Seconds(), "docs/sec")
	})

	b.Run("UltraBatchIndexer", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			runtime.GC()
			tmpDir := b.TempDir()
			indexer := algo.NewUltraBatchIndexer(tmpDir, algo.UltraBatchConfig{
				NumWorkers: runtime.NumCPU() * 5,
			})

			for j := 0; j < len(texts); j += batchSize {
				end := j + batchSize
				if end > len(texts) {
					end = len(texts)
				}
				indexer.AddBatch(docIDs[j:end], texts[j:end])
			}
		}
		b.ReportMetric(float64(len(texts)*b.N)/b.Elapsed().Seconds(), "docs/sec")
	})
}

// TestProfileTokenization runs tokenization with CPU profiling
func TestProfileTokenization(t *testing.T) {
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

	numWorkers := runtime.NumCPU() * 5
	batchSize := (len(texts) + numWorkers - 1) / numWorkers

	var totalTokens atomic.Int64

	t.Log("Running tokenization (profile with: go test -cpuprofile=cpu.prof -run TestProfileTokenization)")

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
			localTokens := int64(0)

			for i := start; i < end; i++ {
				clear(freqs)
				n := algo.TokenizeMega(texts[i], freqs)
				localTokens += int64(n)
			}
			totalTokens.Add(localTokens)
		}(startIdx, endIdx)
	}
	wg.Wait()

	elapsed := time.Since(start)
	t.Logf("Tokenization: %v for %d docs (%.0f docs/sec, %d total tokens)",
		elapsed, len(texts), float64(len(texts))/elapsed.Seconds(), totalTokens.Load())
}
