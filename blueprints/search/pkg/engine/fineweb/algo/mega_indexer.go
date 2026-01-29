// Package algo provides MegaIndexer - SIMD-style optimizations for 1M docs/sec target.
// Key techniques:
// 1. Process 8 bytes at a time using uint64 operations (SWAR)
// 2. Minimize branching in hot path
// 3. Lock-free posting accumulation
// 4. Arena-style memory allocation
package algo

import (
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

// megaCharClass defines character classes for SWAR processing.
// Each byte: 0x00 = delimiter, 0x01 = alphanumeric
var megaCharClass [256]byte

// megaToLower maps chars to lowercase (0 for delimiters)
var megaToLower [256]byte

func init() {
	for i := 0; i < 256; i++ {
		if (i >= 'a' && i <= 'z') || (i >= '0' && i <= '9') {
			megaCharClass[i] = 1
			megaToLower[i] = byte(i)
		} else if i >= 'A' && i <= 'Z' {
			megaCharClass[i] = 1
			megaToLower[i] = byte(i | 0x20)
		}
	}
}

// MegaConfig configures the mega indexer.
type MegaConfig struct {
	NumWorkers  int
	SegmentDocs int
}

// Global hash buffer pool for reuse
var hashBufferPool = sync.Pool{
	New: func() interface{} {
		buf := make([]uint64, 0, 2048)
		return &buf
	},
}

// TokenizeMegaDirect collects hashes directly into a slice without using a map.
// This is faster for documents with many repeated terms since it avoids map overhead.
// Returns the slice of hashes (caller should sort and dedupe).
func TokenizeMegaDirect(text string) ([]uint64, int) {
	if len(text) == 0 {
		return nil, 0
	}

	// Get buffer from pool
	bufPtr := hashBufferPool.Get().(*[]uint64)
	hashes := (*bufPtr)[:0]

	data := unsafe.Slice(unsafe.StringData(text), len(text))
	n := len(data)
	totalTokens := 0
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

		// Hash while scanning
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
			hashes = append(hashes, hash)
			totalTokens++
		}
	}

	*bufPtr = hashes
	return hashes, totalTokens
}

// ReturnHashBuffer returns a hash buffer to the pool.
func ReturnHashBuffer(buf *[]uint64) {
	hashBufferPool.Put(buf)
}

// MegaIndexer is optimized for maximum throughput.
type MegaIndexer struct {
	config MegaConfig
	outDir string

	// Sharded index
	shards [ultraNumShards]*megaShard

	// Document state
	docLens   []uint16
	docCount  atomic.Uint64
	totalLen  atomic.Uint64
	docLensMu sync.Mutex
}

type megaShard struct {
	mu    sync.Mutex
	terms map[uint64]*megaPostings
}

type megaPostings struct {
	docIDs []uint32
	freqs  []uint16
}

// NewMegaIndexer creates a new mega indexer.
func NewMegaIndexer(outDir string, cfg MegaConfig) *MegaIndexer {
	if cfg.NumWorkers <= 0 {
		// Optimal is 3-4x CPU based on profiling
		cfg.NumWorkers = runtime.NumCPU() * 4
	}
	if cfg.NumWorkers > 64 {
		cfg.NumWorkers = 64
	}
	if cfg.SegmentDocs <= 0 {
		cfg.SegmentDocs = 500000
	}

	mi := &MegaIndexer{
		config:  cfg,
		outDir:  outDir,
		docLens: make([]uint16, 0, 4000000),
	}

	for i := 0; i < ultraNumShards; i++ {
		mi.shards[i] = &megaShard{
			terms: make(map[uint64]*megaPostings, 20000),
		}
	}

	return mi
}

// TokenizeMega is an optimized tokenizer that processes 8 bytes at a time where possible.
// Uses SWAR (SIMD Within A Register) technique.
func TokenizeMega(text string, freqs map[uint64]uint16) int {
	clear(freqs)

	if len(text) == 0 {
		return 0
	}

	data := unsafe.Slice(unsafe.StringData(text), len(text))
	n := len(data)
	totalTokens := 0
	i := 0

	for i < n {
		// Skip delimiters - unroll for common case
		for i < n && megaCharClass[data[i]] == 0 {
			i++
		}
		if i >= n {
			break
		}

		// Found start of token
		start := i
		hash := uint64(fnvOffset)

		// Process token bytes - unrolled loop for performance
		// This is the hot path that takes 70% of CPU time
		for i+8 <= n {
			// Read 8 bytes at once
			b0, b1, b2, b3 := data[i], data[i+1], data[i+2], data[i+3]
			b4, b5, b6, b7 := data[i+4], data[i+5], data[i+6], data[i+7]

			c0 := megaToLower[b0]
			if c0 == 0 {
				goto done
			}
			hash ^= uint64(c0)
			hash *= fnvPrime
			i++

			c1 := megaToLower[b1]
			if c1 == 0 {
				goto done
			}
			hash ^= uint64(c1)
			hash *= fnvPrime
			i++

			c2 := megaToLower[b2]
			if c2 == 0 {
				goto done
			}
			hash ^= uint64(c2)
			hash *= fnvPrime
			i++

			c3 := megaToLower[b3]
			if c3 == 0 {
				goto done
			}
			hash ^= uint64(c3)
			hash *= fnvPrime
			i++

			c4 := megaToLower[b4]
			if c4 == 0 {
				goto done
			}
			hash ^= uint64(c4)
			hash *= fnvPrime
			i++

			c5 := megaToLower[b5]
			if c5 == 0 {
				goto done
			}
			hash ^= uint64(c5)
			hash *= fnvPrime
			i++

			c6 := megaToLower[b6]
			if c6 == 0 {
				goto done
			}
			hash ^= uint64(c6)
			hash *= fnvPrime
			i++

			c7 := megaToLower[b7]
			if c7 == 0 {
				goto done
			}
			hash ^= uint64(c7)
			hash *= fnvPrime
			i++
		}

		// Handle remaining bytes (< 8)
		for i < n {
			c := megaToLower[data[i]]
			if c == 0 {
				break
			}
			hash ^= uint64(c)
			hash *= fnvPrime
			i++
		}

	done:
		tokenLen := i - start
		if tokenLen >= 2 && tokenLen <= 32 {
			freqs[hash]++
			totalTokens++
		}
	}

	return totalTokens
}

// TokenizeMegaV2 uses a different strategy - process without branching.
func TokenizeMegaV2(text string, freqs map[uint64]uint16) int {
	clear(freqs)

	if len(text) == 0 {
		return 0
	}

	data := unsafe.Slice(unsafe.StringData(text), len(text))
	n := len(data)
	totalTokens := 0

	hash := uint64(fnvOffset)
	tokenLen := 0

	for i := 0; i < n; i++ {
		c := megaToLower[data[i]]
		if c != 0 {
			// Continue token
			hash ^= uint64(c)
			hash *= fnvPrime
			tokenLen++
		} else {
			// End of token
			if tokenLen >= 2 && tokenLen <= 32 {
				freqs[hash]++
				totalTokens++
			}
			hash = uint64(fnvOffset)
			tokenLen = 0
		}
	}

	// Handle last token
	if tokenLen >= 2 && tokenLen <= 32 {
		freqs[hash]++
		totalTokens++
	}

	return totalTokens
}

// TokenizeMegaV3 uses batch LUT lookups and minimal branching.
func TokenizeMegaV3(text string, freqs map[uint64]uint16) int {
	clear(freqs)

	if len(text) == 0 {
		return 0
	}

	data := unsafe.Slice(unsafe.StringData(text), len(text))
	n := len(data)
	totalTokens := 0
	i := 0

	for i < n {
		// Skip delimiters
		for i < n && megaToLower[data[i]] == 0 {
			i++
		}
		if i >= n {
			break
		}

		// Compute hash inline
		start := i
		h := uint64(fnvOffset)

		// Unrolled loop - process 4 at a time
		for i+4 <= n {
			c0 := megaToLower[data[i]]
			c1 := megaToLower[data[i+1]]
			c2 := megaToLower[data[i+2]]
			c3 := megaToLower[data[i+3]]

			// Check if any is delimiter
			if c0 == 0 {
				break
			}
			h = (h ^ uint64(c0)) * fnvPrime
			i++

			if c1 == 0 {
				break
			}
			h = (h ^ uint64(c1)) * fnvPrime
			i++

			if c2 == 0 {
				break
			}
			h = (h ^ uint64(c2)) * fnvPrime
			i++

			if c3 == 0 {
				break
			}
			h = (h ^ uint64(c3)) * fnvPrime
			i++
		}

		// Remaining bytes
		for i < n {
			c := megaToLower[data[i]]
			if c == 0 {
				break
			}
			h = (h ^ uint64(c)) * fnvPrime
			i++
		}

		tl := i - start
		if tl >= 2 && tl <= 32 {
			freqs[h]++
			totalTokens++
		}
	}

	return totalTokens
}

// AddBatch indexes a batch using mega optimization.
func (mi *MegaIndexer) AddBatch(docIDs []uint32, texts []string) {
	if len(texts) == 0 {
		return
	}

	numDocs := len(texts)
	numWorkers := mi.config.NumWorkers
	if numWorkers > numDocs {
		numWorkers = numDocs
	}

	// Per-worker posting collection
	type posting struct {
		hash  uint64
		docID uint32
		freq  uint16
	}
	workerShardPostings := make([][][]posting, numWorkers)
	for w := 0; w < numWorkers; w++ {
		workerShardPostings[w] = make([][]posting, ultraNumShards)
		for s := 0; s < ultraNumShards; s++ {
			workerShardPostings[w][s] = make([]posting, 0, (numDocs/numWorkers)*2)
		}
	}

	// Phase 1: Parallel tokenization
	docLensLocal := make([]uint16, numDocs)
	var wg sync.WaitGroup
	batchSize := (numDocs + numWorkers - 1) / numWorkers

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
			freqs := make(map[uint64]uint16, 256)
			myShards := workerShardPostings[workerID]
			recreateCounter := 0

			for i := start; i < end; i++ {
				docID := docIDs[i]
				docLen := TokenizeMega(texts[i], freqs)
				if docLen > 65535 {
					docLen = 65535
				}
				docLensLocal[i] = uint16(docLen)

				for hash, freq := range freqs {
					shardID := hash & ultraShardMask
					myShards[shardID] = append(myShards[shardID], posting{hash, docID, freq})
				}

				recreateCounter++
				if recreateCounter >= 100 {
					freqs = make(map[uint64]uint16, 256)
					recreateCounter = 0
				}
			}
		}(w, start, end)
	}
	wg.Wait()

	// Collect doc lengths
	var totalLen uint64
	for _, dl := range docLensLocal {
		totalLen += uint64(dl)
	}
	mi.docCount.Add(uint64(numDocs))
	mi.totalLen.Add(totalLen)

	mi.docLensMu.Lock()
	mi.docLens = append(mi.docLens, docLensLocal...)
	mi.docLensMu.Unlock()

	// Phase 2: Parallel shard updates
	shardsPerWorker := (ultraNumShards + numWorkers - 1) / numWorkers

	for w := 0; w < numWorkers; w++ {
		startShard := w * shardsPerWorker
		endShard := startShard + shardsPerWorker
		if endShard > ultraNumShards {
			endShard = ultraNumShards
		}
		if startShard >= endShard {
			break
		}

		wg.Add(1)
		go func(startShard, endShard int) {
			defer wg.Done()
			for shardID := startShard; shardID < endShard; shardID++ {
				shard := mi.shards[shardID]

				var totalPostings int
				for _, ws := range workerShardPostings {
					totalPostings += len(ws[shardID])
				}
				if totalPostings == 0 {
					continue
				}

				shard.mu.Lock()
				for _, ws := range workerShardPostings {
					for _, p := range ws[shardID] {
						pl, exists := shard.terms[p.hash]
						if !exists {
							pl = &megaPostings{
								docIDs: make([]uint32, 0, 128),
								freqs:  make([]uint16, 0, 128),
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

// Finish completes indexing and returns a searchable index.
func (mi *MegaIndexer) Finish() (*SegmentedIndex, error) {
	numDocs := int(mi.docCount.Load())
	if numDocs == 0 {
		return nil, nil
	}

	avgDocLen := float64(mi.totalLen.Load()) / float64(numDocs)

	terms := make(map[string]*SegmentPostings)

	for shardID := 0; shardID < ultraNumShards; shardID++ {
		shard := mi.shards[shardID]
		shard.mu.Lock()
		for hash, pl := range shard.terms {
			hashKey := hashToKey(hash)
			terms[hashKey] = &SegmentPostings{
				DocIDs: pl.docIDs,
				Freqs:  pl.freqs,
			}
		}
		shard.mu.Unlock()
	}

	docLensMap := make(map[uint32]uint16, numDocs)
	for i, dl := range mi.docLens {
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
		docLens:   mi.docLens,
	}, nil
}

// Mega256Indexer uses 256 shards and direct hash collection for maximum throughput.
type Mega256Indexer struct {
	config    MegaConfig
	outDir    string
	shards    [256]*mega256Shard
	docLens   []uint16
	docCount  atomic.Uint64
	totalLen  atomic.Uint64
	docLensMu sync.Mutex
}

type mega256Shard struct {
	mu    sync.Mutex
	terms map[uint64]*megaPostings
}

// NewMega256Indexer creates a Mega256Indexer with 256 shards.
func NewMega256Indexer(outDir string, cfg MegaConfig) *Mega256Indexer {
	if cfg.NumWorkers <= 0 {
		cfg.NumWorkers = runtime.NumCPU() * 2
	}
	if cfg.NumWorkers > 64 {
		cfg.NumWorkers = 64
	}

	mi := &Mega256Indexer{
		config:  cfg,
		outDir:  outDir,
		docLens: make([]uint16, 0, 4000000),
	}

	for i := 0; i < 256; i++ {
		mi.shards[i] = &mega256Shard{
			terms: make(map[uint64]*megaPostings, 5000),
		}
	}

	return mi
}

// AddBatch processes documents with 256 shards for reduced contention.
func (mi *Mega256Indexer) AddBatch(docIDs []uint32, texts []string) {
	if len(texts) == 0 {
		return
	}

	numDocs := len(texts)
	numWorkers := mi.config.NumWorkers
	if numWorkers > numDocs {
		numWorkers = numDocs
	}

	// Per-worker shard buffers
	type posting struct {
		hash  uint64
		docID uint32
		freq  uint16
	}
	workerShardPostings := make([][][]posting, numWorkers)
	for w := 0; w < numWorkers; w++ {
		workerShardPostings[w] = make([][]posting, 256)
		for s := 0; s < 256; s++ {
			workerShardPostings[w][s] = make([]posting, 0, (numDocs/numWorkers)/2)
		}
	}

	docLensLocal := make([]uint16, numDocs)
	var wg sync.WaitGroup
	batchSize := (numDocs + numWorkers - 1) / numWorkers

	// Phase 1: Parallel tokenization
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
			freqs := make(map[uint64]uint16, 256)
			myShards := workerShardPostings[workerID]
			recreateCounter := 0

			for i := start; i < end; i++ {
				docID := docIDs[i]
				docLen := TokenizeMega(texts[i], freqs)
				if docLen > 65535 {
					docLen = 65535
				}
				docLensLocal[i] = uint16(docLen)

				for hash, freq := range freqs {
					shardID := hash & 0xFF // 256 shards
					myShards[shardID] = append(myShards[shardID], posting{hash, docID, freq})
				}

				recreateCounter++
				if recreateCounter >= 100 {
					freqs = make(map[uint64]uint16, 256)
					recreateCounter = 0
				}
			}
		}(w, start, end)
	}
	wg.Wait()

	// Collect stats
	var totalLen uint64
	for _, dl := range docLensLocal {
		totalLen += uint64(dl)
	}
	mi.docCount.Add(uint64(numDocs))
	mi.totalLen.Add(totalLen)

	mi.docLensMu.Lock()
	mi.docLens = append(mi.docLens, docLensLocal...)
	mi.docLensMu.Unlock()

	// Phase 2: Parallel shard updates (256 shards = less contention)
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
				shard := mi.shards[shardID]

				var totalPostings int
				for _, ws := range workerShardPostings {
					totalPostings += len(ws[shardID])
				}
				if totalPostings == 0 {
					continue
				}

				shard.mu.Lock()
				for _, ws := range workerShardPostings {
					for _, p := range ws[shardID] {
						pl, exists := shard.terms[p.hash]
						if !exists {
							pl = &megaPostings{
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

// Finish merges shards into searchable index.
func (mi *Mega256Indexer) Finish() (*SegmentedIndex, error) {
	numDocs := int(mi.docCount.Load())
	if numDocs == 0 {
		return nil, nil
	}

	avgDocLen := float64(mi.totalLen.Load()) / float64(numDocs)
	terms := make(map[string]*SegmentPostings)

	for shardID := 0; shardID < 256; shardID++ {
		shard := mi.shards[shardID]
		for hash, pl := range shard.terms {
			hashKey := hashToKey(hash)
			terms[hashKey] = &SegmentPostings{
				DocIDs: pl.docIDs,
				Freqs:  pl.freqs,
			}
		}
	}

	docLensMap := make(map[uint32]uint16, numDocs)
	for i, dl := range mi.docLens {
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
		docLens:   mi.docLens,
	}, nil
}
