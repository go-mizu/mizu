// Package algo provides StreamIndexer - eliminates per-doc frequency counting.
//
// Key insight from profiling:
// - Pure scan+hash: 500k+ docs/sec (2.0x from 1M)
// - With FixedHashTable: 400k docs/sec (adds 20% overhead)
// - With Go Map: 335k docs/sec (adds 35% overhead)
//
// New approach:
// 1. Emit raw (hash, docID) pairs during tokenization (no counting)
// 2. Sort pairs by (hash, docID) using radix sort
// 3. Count frequencies during the sorted scan (linear)
//
// This moves frequency counting to a cache-friendly linear scan
// instead of random hash table accesses.
package algo

import (
	"runtime"
	"sync"
	"unsafe"
)

// HashDocPair represents a single token occurrence.
type HashDocPair struct {
	Hash  uint64
	DocID uint32
}

// StreamIndexer emits raw token occurrences without frequency counting.
// Frequency counting is deferred to a cache-friendly sorted merge.
type StreamIndexer struct {
	shardBuffers [][]HashDocPair
	shardMu      []sync.Mutex
	numShards    int
	numWorkers   int
}

// NewStreamIndexer creates a new stream-based indexer.
func NewStreamIndexer(numShards int) *StreamIndexer {
	if numShards == 0 {
		numShards = 256
	}
	numWorkers := runtime.NumCPU() * 5

	buffers := make([][]HashDocPair, numShards)
	for i := range buffers {
		buffers[i] = make([]HashDocPair, 0, 100000)
	}

	return &StreamIndexer{
		shardBuffers: buffers,
		shardMu:      make([]sync.Mutex, numShards),
		numShards:    numShards,
		numWorkers:   numWorkers,
	}
}

// IndexBatch processes a batch of documents, emitting raw (hash, docID) pairs.
func (s *StreamIndexer) IndexBatch(texts []string, startDocID int) {
	batchSize := (len(texts) + s.numWorkers - 1) / s.numWorkers

	var wg sync.WaitGroup
	for w := 0; w < s.numWorkers; w++ {
		startIdx := w * batchSize
		endIdx := startIdx + batchSize
		if endIdx > len(texts) {
			endIdx = len(texts)
		}
		if startIdx >= endIdx {
			break
		}

		wg.Add(1)
		go func(start, end, docOffset int) {
			defer wg.Done()

			// Local buffers per shard
			localShards := make([][]HashDocPair, s.numShards)
			for i := range localShards {
				localShards[i] = make([]HashDocPair, 0, 256)
			}

			for i := start; i < end; i++ {
				docID := uint32(docOffset + i)
				streamTokenize(texts[i], func(hash uint64) {
					shardID := int(hash & uint64(s.numShards-1))
					localShards[shardID] = append(localShards[shardID], HashDocPair{hash, docID})
				})
			}

			// Merge to global shards
			for shardID := range localShards {
				if len(localShards[shardID]) > 0 {
					s.shardMu[shardID].Lock()
					s.shardBuffers[shardID] = append(s.shardBuffers[shardID], localShards[shardID]...)
					s.shardMu[shardID].Unlock()
				}
			}
		}(startIdx, endIdx, startDocID)
	}
	wg.Wait()
}

// streamTokenize emits each token hash without frequency counting.
func streamTokenize(text string, emit func(hash uint64)) {
	if len(text) == 0 {
		return
	}

	const fnvOffset = 14695981039346656037
	const fnvPrime = 1099511628211

	data := unsafe.Slice(unsafe.StringData(text), len(text))
	n := len(data)
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

		tokenLen := i - start
		if tokenLen >= 2 && tokenLen <= 32 {
			emit(hash)
		}
	}
}

// BuildIndex builds the final index from accumulated pairs.
// Uses radix sort + linear scan for frequency counting.
func (s *StreamIndexer) BuildIndex() map[uint64]*PostingListBuilder {
	result := make(map[uint64]*PostingListBuilder, 100000)

	var wg sync.WaitGroup
	var resultMu sync.Mutex

	shardsPerWorker := (s.numShards + s.numWorkers - 1) / s.numWorkers

	for w := 0; w < s.numWorkers; w++ {
		startShard := w * shardsPerWorker
		endShard := startShard + shardsPerWorker
		if endShard > s.numShards {
			endShard = s.numShards
		}
		if startShard >= endShard {
			break
		}

		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			localResult := make(map[uint64]*PostingListBuilder, 10000)

			for shardID := start; shardID < end; shardID++ {
				pairs := s.shardBuffers[shardID]
				if len(pairs) == 0 {
					continue
				}

				// Sort by (hash, docID)
				radixSortPairs(pairs)

				// Count frequencies with linear scan
				var currentHash uint64
				var currentDocID uint32
				var count uint16

				for i := range pairs {
					if pairs[i].Hash == currentHash && pairs[i].DocID == currentDocID {
						count++
					} else {
						// Emit previous
						if currentHash != 0 {
							pl, exists := localResult[currentHash]
							if !exists {
								pl = &PostingListBuilder{
									DocIDs: make([]uint32, 0, 32),
									Freqs:  make([]uint16, 0, 32),
								}
								localResult[currentHash] = pl
							}
							pl.DocIDs = append(pl.DocIDs, currentDocID)
							pl.Freqs = append(pl.Freqs, count)
						}
						// Start new
						currentHash = pairs[i].Hash
						currentDocID = pairs[i].DocID
						count = 1
					}
				}
				// Don't forget the last one
				if currentHash != 0 {
					pl, exists := localResult[currentHash]
					if !exists {
						pl = &PostingListBuilder{
							DocIDs: make([]uint32, 0, 32),
							Freqs:  make([]uint16, 0, 32),
						}
						localResult[currentHash] = pl
					}
					pl.DocIDs = append(pl.DocIDs, currentDocID)
					pl.Freqs = append(pl.Freqs, count)
				}
			}

			// Merge to global result
			resultMu.Lock()
			for hash, pl := range localResult {
				if existing, ok := result[hash]; ok {
					existing.DocIDs = append(existing.DocIDs, pl.DocIDs...)
					existing.Freqs = append(existing.Freqs, pl.Freqs...)
				} else {
					result[hash] = pl
				}
			}
			resultMu.Unlock()
		}(startShard, endShard)
	}
	wg.Wait()

	return result
}

// PostingListBuilder accumulates postings for a term.
type PostingListBuilder struct {
	DocIDs []uint32
	Freqs  []uint16
}

// radixSortPairs sorts HashDocPairs by (Hash, DocID).
func radixSortPairs(data []HashDocPair) {
	if len(data) < 64 {
		// Insertion sort for small slices
		for i := 1; i < len(data); i++ {
			key := data[i]
			j := i - 1
			for j >= 0 && (data[j].Hash > key.Hash || (data[j].Hash == key.Hash && data[j].DocID > key.DocID)) {
				data[j+1] = data[j]
				j--
			}
			data[j+1] = key
		}
		return
	}

	// Radix sort on Hash (8 passes of 8 bits each)
	aux := make([]HashDocPair, len(data))
	var count [256]int

	// Sort by hash first (8 passes)
	for shift := uint(0); shift < 64; shift += 8 {
		for i := range count {
			count[i] = 0
		}
		for _, v := range data {
			digit := (v.Hash >> shift) & 0xFF
			count[digit]++
		}
		for i := 1; i < 256; i++ {
			count[i] += count[i-1]
		}
		for i := len(data) - 1; i >= 0; i-- {
			digit := (data[i].Hash >> shift) & 0xFF
			count[digit]--
			aux[count[digit]] = data[i]
		}
		copy(data, aux)
	}

	// Then sort by DocID within same hash (4 passes of 8 bits each)
	for shift := uint(0); shift < 32; shift += 8 {
		for i := range count {
			count[i] = 0
		}
		for _, v := range data {
			digit := (v.DocID >> shift) & 0xFF
			count[digit]++
		}
		for i := 1; i < 256; i++ {
			count[i] += count[i-1]
		}
		for i := len(data) - 1; i >= 0; i-- {
			digit := (data[i].DocID >> shift) & 0xFF
			count[digit]--
			aux[count[digit]] = data[i]
		}
		copy(data, aux)
	}
}

// NoCountTokenize tokenizes without any frequency tracking.
// Returns only the total token count.
func NoCountTokenize(text string, emit func(hash uint64)) int {
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

		tokenLen := i - start
		if tokenLen >= 2 && tokenLen <= 32 {
			emit(hash)
			tokenCount++
		}
	}

	return tokenCount
}
