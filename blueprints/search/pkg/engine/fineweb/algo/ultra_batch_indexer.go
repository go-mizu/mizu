// Package algo provides UltraBatchIndexer - optimized for maximum batch throughput.
// Key innovations:
// 1. Process entire batches without per-document synchronization
// 2. Use a single large map per worker, merge at batch end
// 3. Minimize atomic operations
// 4. Pre-allocate all buffers at batch start
package algo

import (
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

// UltraBatchConfig configures the ultra batch indexer.
type UltraBatchConfig struct {
	NumWorkers int
}

// UltraBatchIndexer processes documents in large batches with minimal sync.
type UltraBatchIndexer struct {
	config    UltraBatchConfig
	outDir    string
	shards    [256]*ultraBatchShard
	docLens   []uint16
	docCount  atomic.Uint64
	totalLen  atomic.Uint64
	docLensMu sync.Mutex
}

type ultraBatchShard struct {
	mu    sync.Mutex
	terms map[uint64]*ultraBatchPostings
}

type ultraBatchPostings struct {
	docIDs []uint32
	freqs  []uint16
}

// NewUltraBatchIndexer creates an ultra batch indexer.
func NewUltraBatchIndexer(outDir string, cfg UltraBatchConfig) *UltraBatchIndexer {
	if cfg.NumWorkers <= 0 {
		// Optimal is 5x CPU based on profiling
		cfg.NumWorkers = runtime.NumCPU() * 5
	}
	if cfg.NumWorkers > 128 {
		cfg.NumWorkers = 128
	}

	ubi := &UltraBatchIndexer{
		config:  cfg,
		outDir:  outDir,
		docLens: make([]uint16, 0, 4000000),
	}

	for i := 0; i < 256; i++ {
		ubi.shards[i] = &ultraBatchShard{
			terms: make(map[uint64]*ultraBatchPostings, 10000),
		}
	}

	return ubi
}

// TokenizeUltraBatch is optimized for batch processing.
// Uses inline FNV-1a with minimal branching.
func TokenizeUltraBatch(text string, hashBuf *[]uint64, freqBuf *[]uint16) int {
	if len(text) == 0 {
		return 0
	}

	// Reset buffers
	*hashBuf = (*hashBuf)[:0]
	*freqBuf = (*freqBuf)[:0]

	data := unsafe.Slice(unsafe.StringData(text), len(text))
	n := len(data)
	tokenCount := 0

	// Use inline frequency tracking with sorted insert
	freqMap := make(map[uint64]int, 128) // hash -> index in buffers

	i := 0
	for i < n {
		// Skip delimiters
		for i < n && megaToLower[data[i]] == 0 {
			i++
		}
		if i >= n {
			break
		}

		start := i
		hash := uint64(fnvOffset)

		// Compute hash inline
		for i < n {
			c := megaToLower[data[i]]
			if c == 0 {
				break
			}
			hash ^= uint64(c)
			hash *= fnvPrime
			i++
		}

		tokenLen := i - start
		if tokenLen >= 2 && tokenLen <= 32 {
			if idx, exists := freqMap[hash]; exists {
				(*freqBuf)[idx]++
			} else {
				idx := len(*hashBuf)
				freqMap[hash] = idx
				*hashBuf = append(*hashBuf, hash)
				*freqBuf = append(*freqBuf, 1)
			}
			tokenCount++
		}
	}

	return tokenCount
}

// AddBatch indexes a batch with ultra-optimized processing.
func (ubi *UltraBatchIndexer) AddBatch(docIDs []uint32, texts []string) {
	if len(texts) == 0 {
		return
	}

	numDocs := len(texts)
	numWorkers := ubi.config.NumWorkers
	if numWorkers > numDocs {
		numWorkers = numDocs
	}

	// Per-worker accumulated postings
	type workerResult struct {
		postings [256][]struct {
			hash  uint64
			docID uint32
			freq  uint16
		}
		docLens  []uint16
		totalLen uint64
	}

	results := make([]workerResult, numWorkers)
	for w := 0; w < numWorkers; w++ {
		for s := 0; s < 256; s++ {
			results[w].postings[s] = make([]struct {
				hash  uint64
				docID uint32
				freq  uint16
			}, 0, (numDocs/numWorkers)*50/256+16)
		}
		results[w].docLens = make([]uint16, 0, numDocs/numWorkers+1)
	}

	var wg sync.WaitGroup
	batchSize := (numDocs + numWorkers - 1) / numWorkers

	// Phase 1: Parallel tokenization - each worker accumulates independently
	for w := 0; w < numWorkers; w++ {
		start := w * batchSize
		end := start + batchSize
		if end > numDocs {
			end = numDocs
		}
		if start >= end {
			break
		}

		wg.Add(1)
		go func(workerID, start, end int) {
			defer wg.Done()

			result := &results[workerID]
			hashBuf := make([]uint64, 0, 512)
			freqBuf := make([]uint16, 0, 512)

			for i := start; i < end; i++ {
				docID := docIDs[i]
				docLen := TokenizeUltraBatch(texts[i], &hashBuf, &freqBuf)

				if docLen > 65535 {
					docLen = 65535
				}
				result.docLens = append(result.docLens, uint16(docLen))
				result.totalLen += uint64(docLen)

				// Distribute postings to shards
				for j, hash := range hashBuf {
					shardID := hash & 0xFF
					result.postings[shardID] = append(result.postings[shardID], struct {
						hash  uint64
						docID uint32
						freq  uint16
					}{hash, docID, freqBuf[j]})
				}
			}
		}(w, start, end)
	}
	wg.Wait()

	// Collect doc lengths
	var allDocLens []uint16
	var totalLen uint64
	for w := 0; w < numWorkers; w++ {
		allDocLens = append(allDocLens, results[w].docLens...)
		totalLen += results[w].totalLen
	}

	ubi.docCount.Add(uint64(len(allDocLens)))
	ubi.totalLen.Add(totalLen)

	ubi.docLensMu.Lock()
	ubi.docLens = append(ubi.docLens, allDocLens...)
	ubi.docLensMu.Unlock()

	// Phase 2: Parallel shard updates
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
		go func(startShard, endShard int) {
			defer wg.Done()
			for shardID := startShard; shardID < endShard; shardID++ {
				shard := ubi.shards[shardID]

				// Count total postings
				total := 0
				for _, r := range results {
					total += len(r.postings[shardID])
				}
				if total == 0 {
					continue
				}

				shard.mu.Lock()
				for _, r := range results {
					for _, p := range r.postings[shardID] {
						pl, exists := shard.terms[p.hash]
						if !exists {
							pl = &ultraBatchPostings{
								docIDs: make([]uint32, 0, 64),
								freqs:  make([]uint16, 0, 64),
							}
							shard.terms[p.hash] = pl
						}
						pl.docIDs = append(pl.docIDs, p.docID)
						pl.freqs = append(pl.freqs, p.freq)
					}
				}
				shard.mu.Unlock()
			}
		}(startShard, endShard)
	}
	wg.Wait()
}

// Finish creates searchable index.
func (ubi *UltraBatchIndexer) Finish() (*SegmentedIndex, error) {
	numDocs := int(ubi.docCount.Load())
	if numDocs == 0 {
		return nil, nil
	}

	avgDocLen := float64(ubi.totalLen.Load()) / float64(numDocs)
	terms := make(map[string]*SegmentPostings)

	for shardID := 0; shardID < 256; shardID++ {
		shard := ubi.shards[shardID]
		for hash, pl := range shard.terms {
			hashKey := hashToKey(hash)
			terms[hashKey] = &SegmentPostings{
				DocIDs: pl.docIDs,
				Freqs:  pl.freqs,
			}
		}
	}

	docLensMap := make(map[uint32]uint16, numDocs)
	for i, dl := range ubi.docLens {
		docLensMap[uint32(i)] = dl
	}

	segment := &SearchSegment{
		id:      0,
		terms:   terms,
		docLens: docLensMap,
		numDocs: numDocs,
	}

	return &SegmentedIndex{
		segments:  []*SearchSegment{segment},
		numDocs:   numDocs,
		avgDocLen: avgDocLen,
		docLens:   ubi.docLens,
	}, nil
}

// StreamingBatchIndexer processes documents in a streaming fashion.
// Optimized for memory efficiency with high throughput.
type StreamingBatchIndexer struct {
	config    UltraBatchConfig
	outDir    string
	shards    [256]*streamBatchShard
	docLens   []uint16
	docCount  atomic.Uint64
	totalLen  atomic.Uint64
	docLensMu sync.Mutex
}

type streamBatchShard struct {
	mu    sync.Mutex
	terms map[uint64]*streamBatchPostings
}

type streamBatchPostings struct {
	docIDs []uint32
	freqs  []uint16
}

// NewStreamingBatchIndexer creates a streaming batch indexer.
func NewStreamingBatchIndexer(outDir string, cfg UltraBatchConfig) *StreamingBatchIndexer {
	if cfg.NumWorkers <= 0 {
		cfg.NumWorkers = runtime.NumCPU() * 4
	}

	sbi := &StreamingBatchIndexer{
		config:  cfg,
		outDir:  outDir,
		docLens: make([]uint16, 0, 4000000),
	}

	for i := 0; i < 256; i++ {
		sbi.shards[i] = &streamBatchShard{
			terms: make(map[uint64]*streamBatchPostings, 10000),
		}
	}

	return sbi
}

// TokenizeStreaming uses minimal allocations.
func TokenizeStreaming(text string, emit func(hash uint64, freq uint16)) int {
	if len(text) == 0 {
		return 0
	}

	data := unsafe.Slice(unsafe.StringData(text), len(text))
	n := len(data)
	tokenCount := 0

	// Small inline frequency tracker
	seen := make(map[uint64]uint16, 64)

	i := 0
	for i < n {
		for i < n && megaToLower[data[i]] == 0 {
			i++
		}
		if i >= n {
			break
		}

		start := i
		hash := uint64(fnvOffset)

		for i < n {
			c := megaToLower[data[i]]
			if c == 0 {
				break
			}
			hash ^= uint64(c)
			hash *= fnvPrime
			i++
		}

		if i-start >= 2 && i-start <= 32 {
			seen[hash]++
			tokenCount++
		}
	}

	// Emit all unique terms
	for hash, freq := range seen {
		emit(hash, freq)
	}

	return tokenCount
}

// AddBatch indexes with streaming optimization.
func (sbi *StreamingBatchIndexer) AddBatch(docIDs []uint32, texts []string) {
	if len(texts) == 0 {
		return
	}

	numDocs := len(texts)
	numWorkers := sbi.config.NumWorkers
	if numWorkers > numDocs {
		numWorkers = numDocs
	}

	// Per-worker shard buffers
	type posting struct {
		hash  uint64
		docID uint32
		freq  uint16
	}
	workerShards := make([][][]posting, numWorkers)
	for w := 0; w < numWorkers; w++ {
		workerShards[w] = make([][]posting, 256)
		for s := 0; s < 256; s++ {
			workerShards[w][s] = make([]posting, 0, 32)
		}
	}

	docLensLocal := make([]uint16, numDocs)
	var wg sync.WaitGroup
	batchSize := (numDocs + numWorkers - 1) / numWorkers

	// Phase 1: Parallel tokenization with streaming emit
	for w := 0; w < numWorkers; w++ {
		start := w * batchSize
		end := start + batchSize
		if end > numDocs {
			end = numDocs
		}
		if start >= end {
			break
		}

		wg.Add(1)
		go func(workerID, start, end int) {
			defer wg.Done()
			myShards := workerShards[workerID]

			for i := start; i < end; i++ {
				docID := docIDs[i]

				emit := func(hash uint64, freq uint16) {
					shardID := hash & 0xFF
					myShards[shardID] = append(myShards[shardID], posting{hash, docID, freq})
				}

				docLen := TokenizeStreaming(texts[i], emit)
				if docLen > 65535 {
					docLen = 65535
				}
				docLensLocal[i] = uint16(docLen)
			}
		}(w, start, end)
	}
	wg.Wait()

	// Collect stats
	var totalLen uint64
	for _, dl := range docLensLocal {
		totalLen += uint64(dl)
	}
	sbi.docCount.Add(uint64(numDocs))
	sbi.totalLen.Add(totalLen)

	sbi.docLensMu.Lock()
	sbi.docLens = append(sbi.docLens, docLensLocal...)
	sbi.docLensMu.Unlock()

	// Phase 2: Merge to shards
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
		go func(startShard, endShard int) {
			defer wg.Done()
			for shardID := startShard; shardID < endShard; shardID++ {
				shard := sbi.shards[shardID]

				total := 0
				for _, ws := range workerShards {
					total += len(ws[shardID])
				}
				if total == 0 {
					continue
				}

				shard.mu.Lock()
				for _, ws := range workerShards {
					for _, p := range ws[shardID] {
						pl, exists := shard.terms[p.hash]
						if !exists {
							pl = &streamBatchPostings{
								docIDs: make([]uint32, 0, 32),
								freqs:  make([]uint16, 0, 32),
							}
							shard.terms[p.hash] = pl
						}
						pl.docIDs = append(pl.docIDs, p.docID)
						pl.freqs = append(pl.freqs, p.freq)
					}
				}
				shard.mu.Unlock()
			}
		}(startShard, endShard)
	}
	wg.Wait()
}

// Finish creates searchable index.
func (sbi *StreamingBatchIndexer) Finish() (*SegmentedIndex, error) {
	numDocs := int(sbi.docCount.Load())
	if numDocs == 0 {
		return nil, nil
	}

	avgDocLen := float64(sbi.totalLen.Load()) / float64(numDocs)
	terms := make(map[string]*SegmentPostings)

	for shardID := 0; shardID < 256; shardID++ {
		shard := sbi.shards[shardID]
		for hash, pl := range shard.terms {
			hashKey := hashToKey(hash)
			terms[hashKey] = &SegmentPostings{
				DocIDs: pl.docIDs,
				Freqs:  pl.freqs,
			}
		}
	}

	docLensMap := make(map[uint32]uint16, numDocs)
	for i, dl := range sbi.docLens {
		docLensMap[uint32(i)] = dl
	}

	segment := &SearchSegment{
		id:      0,
		terms:   terms,
		docLens: docLensMap,
		numDocs: numDocs,
	}

	return &SegmentedIndex{
		segments:  []*SearchSegment{segment},
		numDocs:   numDocs,
		avgDocLen: avgDocLen,
		docLens:   sbi.docLens,
	}, nil
}
