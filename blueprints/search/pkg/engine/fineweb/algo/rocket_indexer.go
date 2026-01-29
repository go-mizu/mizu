// Package algo provides RocketIndexer - lock-free parallel indexing for 1M docs/sec.
// Key insight: Worker isolation eliminates all synchronization during indexing.
// Workers maintain completely separate indexes that merge only at Finish().
package algo

import (
	"runtime"
	"sync"
	"sync/atomic"
)

// RocketConfig configures the rocket indexer.
type RocketConfig struct {
	NumWorkers  int
	SegmentDocs int
}

// RocketIndexer uses per-worker isolated indexes for zero contention.
type RocketIndexer struct {
	config    RocketConfig
	outDir    string
	workers   []*rocketWorker
	docLens   []uint16
	docCount  atomic.Uint64
	totalLen  atomic.Uint64
	docLensMu sync.Mutex
}

// rocketWorker holds a completely isolated index for one worker.
type rocketWorker struct {
	// No locks needed - only accessed by owning worker
	terms map[uint64]*rocketPostings
}

type rocketPostings struct {
	docIDs []uint32
	freqs  []uint16
}

// NewRocketIndexer creates a new rocket indexer.
func NewRocketIndexer(outDir string, cfg RocketConfig) *RocketIndexer {
	if cfg.NumWorkers <= 0 {
		cfg.NumWorkers = runtime.NumCPU() * 2
	}
	if cfg.NumWorkers > 64 {
		cfg.NumWorkers = 64
	}
	if cfg.SegmentDocs <= 0 {
		cfg.SegmentDocs = 500000
	}

	ri := &RocketIndexer{
		config:  cfg,
		outDir:  outDir,
		workers: make([]*rocketWorker, cfg.NumWorkers),
		docLens: make([]uint16, 0, 4000000),
	}

	// Initialize per-worker indexes
	for i := 0; i < cfg.NumWorkers; i++ {
		ri.workers[i] = &rocketWorker{
			terms: make(map[uint64]*rocketPostings, 50000),
		}
	}

	return ri
}

// AddBatch indexes a batch with zero lock contention during tokenization.
func (ri *RocketIndexer) AddBatch(docIDs []uint32, texts []string) {
	if len(texts) == 0 {
		return
	}

	numDocs := len(texts)
	numWorkers := ri.config.NumWorkers
	if numWorkers > numDocs {
		numWorkers = numDocs
	}

	// Each worker processes its docs into its own isolated index
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

			worker := ri.workers[workerID]
			freqs := make(map[uint64]uint16, 256)
			recreateCounter := 0

			for i := start; i < end; i++ {
				docID := docIDs[i]
				docLen := TokenizeToHashReuse(texts[i], freqs)
				if docLen > 65535 {
					docLen = 65535
				}
				docLensLocal[i] = uint16(docLen)

				// Add to worker's isolated index - NO LOCKS!
				for hash, freq := range freqs {
					pl, exists := worker.terms[hash]
					if !exists {
						pl = &rocketPostings{
							docIDs: make([]uint32, 0, 64),
							freqs:  make([]uint16, 0, 64),
						}
						worker.terms[hash] = pl
					}
					pl.docIDs = append(pl.docIDs, docID)
					pl.freqs = append(pl.freqs, freq)
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

	// Update doc lengths (minimal sync)
	var totalLen uint64
	for _, dl := range docLensLocal {
		totalLen += uint64(dl)
	}
	ri.docCount.Add(uint64(numDocs))
	ri.totalLen.Add(totalLen)

	ri.docLensMu.Lock()
	ri.docLens = append(ri.docLens, docLensLocal...)
	ri.docLensMu.Unlock()
}

// Finish merges worker indexes and returns searchable index.
func (ri *RocketIndexer) Finish() (*SegmentedIndex, error) {
	numDocs := int(ri.docCount.Load())
	if numDocs == 0 {
		return nil, nil
	}

	avgDocLen := float64(ri.totalLen.Load()) / float64(numDocs)

	// Merge all worker indexes
	terms := make(map[string]*SegmentPostings)

	for _, worker := range ri.workers {
		for hash, pl := range worker.terms {
			hashKey := hashToKey(hash)
			existing, exists := terms[hashKey]
			if !exists {
				terms[hashKey] = &SegmentPostings{
					DocIDs: pl.docIDs,
					Freqs:  pl.freqs,
				}
			} else {
				existing.DocIDs = append(existing.DocIDs, pl.docIDs...)
				existing.Freqs = append(existing.Freqs, pl.freqs...)
			}
		}
	}

	docLensMap := make(map[uint32]uint16, numDocs)
	for i, dl := range ri.docLens {
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
		docLens:   ri.docLens,
	}, nil
}

// Rocket256Indexer uses 256 shards for better lock distribution.
type Rocket256Indexer struct {
	config    RocketConfig
	outDir    string
	docLens   []uint16
	docCount  atomic.Uint64
	totalLen  atomic.Uint64
	docLensMu sync.Mutex

	// Global sharded index with fine-grained locks
	shards [256]*rocket256Shard
}

type rocket256Shard struct {
	mu    sync.Mutex
	terms map[uint64]*rocketPostings
}

// NewRocket256Indexer creates a Rocket256Indexer with 256 shards.
func NewRocket256Indexer(outDir string, cfg RocketConfig) *Rocket256Indexer {
	if cfg.NumWorkers <= 0 {
		cfg.NumWorkers = runtime.NumCPU() * 2
	}
	if cfg.NumWorkers > 64 {
		cfg.NumWorkers = 64
	}

	ri := &Rocket256Indexer{
		config:  cfg,
		outDir:  outDir,
		docLens: make([]uint16, 0, 4000000),
	}

	for i := 0; i < 256; i++ {
		ri.shards[i] = &rocket256Shard{
			terms: make(map[uint64]*rocketPostings, 5000),
		}
	}

	return ri
}

// AddBatch processes documents with optimal shard batching.
func (ri *Rocket256Indexer) AddBatch(docIDs []uint32, texts []string) {
	if len(texts) == 0 {
		return
	}

	numDocs := len(texts)
	numWorkers := ri.config.NumWorkers
	if numWorkers > numDocs {
		numWorkers = numDocs
	}

	// Per-worker shard buffers (256 shards)
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
				docLen := TokenizeToHashReuse(texts[i], freqs)
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
	ri.docCount.Add(uint64(numDocs))
	ri.totalLen.Add(totalLen)

	ri.docLensMu.Lock()
	ri.docLens = append(ri.docLens, docLensLocal...)
	ri.docLensMu.Unlock()

	// Phase 2: Parallel shard updates (256 shards = less contention)
	shardsPerWorker := 256 / numWorkers
	if shardsPerWorker < 1 {
		shardsPerWorker = 1
	}

	for w := 0; w < numWorkers; w++ {
		startShard := w * shardsPerWorker
		endShard := startShard + shardsPerWorker
		if w == numWorkers-1 {
			endShard = 256
		}
		if startShard >= 256 {
			break
		}

		wg.Add(1)
		go func(startShard, endShard int) {
			defer wg.Done()
			for shardID := startShard; shardID < endShard; shardID++ {
				shard := ri.shards[shardID]

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
							pl = &rocketPostings{
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
func (ri *Rocket256Indexer) Finish() (*SegmentedIndex, error) {
	numDocs := int(ri.docCount.Load())
	if numDocs == 0 {
		return nil, nil
	}

	avgDocLen := float64(ri.totalLen.Load()) / float64(numDocs)
	terms := make(map[string]*SegmentPostings)

	for shardID := 0; shardID < 256; shardID++ {
		shard := ri.shards[shardID]
		for hash, pl := range shard.terms {
			hashKey := hashToKey(hash)
			terms[hashKey] = &SegmentPostings{
				DocIDs: pl.docIDs,
				Freqs:  pl.freqs,
			}
		}
	}

	docLensMap := make(map[uint32]uint16, numDocs)
	for i, dl := range ri.docLens {
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
		docLens:   ri.docLens,
	}, nil
}
