// Package algo provides OptimizedPipeline - maximum throughput pure Go implementation.
//
// Key optimizations:
// 1. Zero-copy tokenization (work directly on byte slices)
// 2. Pre-allocated output buffers
// 3. Minimized memory allocations per document
// 4. Batch processing to amortize per-doc overhead
package algo

import (
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

// OptimizedIndexer achieves maximum throughput through:
// - Zero allocation in hot path
// - Pre-allocated posting buffers
// - Batch document processing
// - Minimized lock contention
type OptimizedIndexer struct {
	shards   [256]*optimizedShard
	docLens  []uint16
	docCount atomic.Uint64
	totalLen atomic.Uint64
	mu       sync.Mutex

	// Pre-allocated worker resources
	workerTables []*FixedHashTable
	workerShards [][][]optimizedPosting
	numWorkers   int
}

type optimizedShard struct {
	mu    sync.Mutex
	terms map[uint64]*optimizedPostings
}

type optimizedPostings struct {
	docIDs []uint32
	freqs  []uint16
}

type optimizedPosting struct {
	hash  uint64
	docID uint32
	freq  uint16
}

// NewOptimizedIndexer creates a new optimized indexer
func NewOptimizedIndexer(numWorkers int) *OptimizedIndexer {
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU() * 5
	}

	idx := &OptimizedIndexer{
		docLens:    make([]uint16, 0, 4000000),
		numWorkers: numWorkers,
	}

	// Initialize shards
	for i := 0; i < 256; i++ {
		idx.shards[i] = &optimizedShard{
			terms: make(map[uint64]*optimizedPostings, 10000),
		}
	}

	// Pre-allocate worker resources
	idx.workerTables = make([]*FixedHashTable, numWorkers)
	idx.workerShards = make([][][]optimizedPosting, numWorkers)
	for w := 0; w < numWorkers; w++ {
		idx.workerTables[w] = NewFixedHashTable(4096)
		idx.workerShards[w] = make([][]optimizedPosting, 256)
		for s := 0; s < 256; s++ {
			idx.workerShards[w][s] = make([]optimizedPosting, 0, 1024)
		}
	}

	return idx
}

// IndexBatch indexes a batch of documents with optimized memory usage
func (idx *OptimizedIndexer) IndexBatch(texts []string, startDocID int) {
	if len(texts) == 0 {
		return
	}

	numDocs := len(texts)
	numWorkers := idx.numWorkers
	if numWorkers > numDocs {
		numWorkers = numDocs
	}

	// Clear worker buffers
	for w := 0; w < numWorkers; w++ {
		for s := 0; s < 256; s++ {
			idx.workerShards[w][s] = idx.workerShards[w][s][:0]
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
			table := idx.workerTables[workerID]
			myShards := idx.workerShards[workerID]

			for i := start; i < end; i++ {
				docID := uint32(startDocID + i)

				// Tokenize with zero-allocation in hot path
				docLen := optimizedTokenize(texts[i], table)
				if docLen > 65535 {
					docLen = 65535
				}
				docLensLocal[i] = uint16(docLen)

				// Distribute to local shards
				slots := table.UsedSlots()
				keys := table.Keys()
				counts := table.Counts()

				for _, slotIdx := range slots {
					hash := keys[slotIdx]
					freq := counts[slotIdx]
					shardID := hash & 0xFF
					myShards[shardID] = append(myShards[shardID],
						optimizedPosting{hash, docID, freq})
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

	idx.docCount.Add(uint64(numDocs))
	idx.totalLen.Add(totalLen)

	idx.mu.Lock()
	idx.docLens = append(idx.docLens, docLensLocal...)
	idx.mu.Unlock()

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
				shard := idx.shards[shardID]

				var totalPostings int
				for _, ws := range idx.workerShards[:numWorkers] {
					totalPostings += len(ws[shardID])
				}
				if totalPostings == 0 {
					continue
				}

				shard.mu.Lock()
				for _, ws := range idx.workerShards[:numWorkers] {
					for _, p := range ws[shardID] {
						pl, exists := shard.terms[p.hash]
						if !exists {
							pl = &optimizedPostings{
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

// optimizedTokenize is the fastest pure Go tokenization
func optimizedTokenize(text string, table *FixedHashTable) int {
	if len(text) == 0 {
		return 0
	}

	const fnvOffset = 14695981039346656037
	const fnvPrime = 1099511628211

	data := unsafe.Slice(unsafe.StringData(text), len(text))
	n := len(data)
	tokenCount := 0
	i := 0

	table.Reset()

	// Unrolled main loop
	for i < n {
		// Skip delimiters - unrolled for common case
		for i+4 <= n {
			if megaToLower[data[i]] != 0 {
				break
			}
			if megaToLower[data[i+1]] != 0 {
				i++
				break
			}
			if megaToLower[data[i+2]] != 0 {
				i += 2
				break
			}
			if megaToLower[data[i+3]] != 0 {
				i += 3
				break
			}
			i += 4
		}
		for i < n && megaToLower[data[i]] == 0 {
			i++
		}
		if i >= n {
			break
		}

		start := i
		hash := uint64(fnvOffset)

		// Hash token - unrolled
		for i+4 <= n {
			c0 := megaToLower[data[i]]
			if c0 == 0 {
				goto done
			}
			hash = (hash ^ uint64(c0)) * fnvPrime

			c1 := megaToLower[data[i+1]]
			if c1 == 0 {
				i++
				goto done
			}
			hash = (hash ^ uint64(c1)) * fnvPrime

			c2 := megaToLower[data[i+2]]
			if c2 == 0 {
				i += 2
				goto done
			}
			hash = (hash ^ uint64(c2)) * fnvPrime

			c3 := megaToLower[data[i+3]]
			if c3 == 0 {
				i += 3
				goto done
			}
			hash = (hash ^ uint64(c3)) * fnvPrime
			i += 4
		}

		for i < n {
			c := megaToLower[data[i]]
			if c == 0 {
				break
			}
			hash = (hash ^ uint64(c)) * fnvPrime
			i++
		}

	done:
		tokenLen := i - start
		if tokenLen >= 2 && tokenLen <= 32 {
			table.Insert(hash)
			tokenCount++
		}
	}

	return tokenCount
}

// Finish returns the built index
func (idx *OptimizedIndexer) Finish() (*SegmentedIndex, error) {
	numDocs := int(idx.docCount.Load())
	if numDocs == 0 {
		return nil, nil
	}

	avgDocLen := float64(idx.totalLen.Load()) / float64(numDocs)
	terms := make(map[string]*SegmentPostings)

	for shardID := 0; shardID < 256; shardID++ {
		shard := idx.shards[shardID]
		for hash, pl := range shard.terms {
			hashKey := hashToKey(hash)
			terms[hashKey] = &SegmentPostings{
				DocIDs: pl.docIDs,
				Freqs:  pl.freqs,
			}
		}
	}

	docLensMap := make(map[uint32]uint16, numDocs)
	for i, dl := range idx.docLens {
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
		docLens:   idx.docLens,
	}, nil
}

// StreamingOptimizedIndexer processes documents in streaming fashion
// for maximum memory efficiency during I/O-bound operations
type StreamingOptimizedIndexer struct {
	indexer *OptimizedIndexer
	batch   []string
	docID   int
}

// NewStreamingOptimizedIndexer creates a streaming indexer
func NewStreamingOptimizedIndexer(numWorkers int) *StreamingOptimizedIndexer {
	return &StreamingOptimizedIndexer{
		indexer: NewOptimizedIndexer(numWorkers),
		batch:   make([]string, 0, 10000),
		docID:   0,
	}
}

// Add adds a document to the current batch
func (s *StreamingOptimizedIndexer) Add(text string) {
	s.batch = append(s.batch, text)
	if len(s.batch) >= 10000 {
		s.Flush()
	}
}

// Flush processes the current batch
func (s *StreamingOptimizedIndexer) Flush() {
	if len(s.batch) == 0 {
		return
	}
	s.indexer.IndexBatch(s.batch, s.docID)
	s.docID += len(s.batch)
	s.batch = s.batch[:0]
}

// Finish returns the built index
func (s *StreamingOptimizedIndexer) Finish() (*SegmentedIndex, error) {
	s.Flush()
	return s.indexer.Finish()
}
