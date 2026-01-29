// Package algo provides PipelinedImporter for maximum throughput indexing.
// Overlaps parquet reading with tokenization and indexing via buffered channels.
package algo

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
)

// PipelinedImporter uses a multi-stage pipeline for maximum throughput.
// Stage 1: Parquet reading (I/O bound)
// Stage 2: Tokenization (CPU bound)
// Stage 3: Shard accumulation (memory bound)
type PipelinedImporter struct {
	config     PipelinedConfig
	outDir     string
	shards     [256]*pipelinedShard
	docLens    []uint16
	docCount   atomic.Uint64
	totalLen   atomic.Uint64
	docLensMu  sync.Mutex

	// Pipeline channels
	tokenizeCh chan pipelinedBatch
	indexCh    chan pipelinedTokenized

	// Sync
	tokenizeWg sync.WaitGroup
	indexWg    sync.WaitGroup
}

type pipelinedShard struct {
	mu    sync.Mutex
	terms map[uint64]*pipelinedPostings
}

type pipelinedPostings struct {
	docIDs []uint32
	freqs  []uint16
}

// PipelinedConfig configures the pipelined importer.
type PipelinedConfig struct {
	NumTokenizers int // Number of tokenization workers
	NumIndexers   int // Number of indexing workers
	BufferSize    int // Channel buffer size
}

// pipelinedBatch is a batch from parquet reader.
type pipelinedBatch struct {
	startDocID uint32
	texts      []string
}

// pipelinedTokenized is a tokenized document.
type pipelinedTokenized struct {
	docID   uint32
	docLen  uint16
	hashes  []uint64
	freqs   []uint16
}

// NewPipelinedImporter creates a pipelined importer.
func NewPipelinedImporter(outDir string, cfg PipelinedConfig) *PipelinedImporter {
	if cfg.NumTokenizers <= 0 {
		cfg.NumTokenizers = runtime.NumCPU() * 2
	}
	if cfg.NumIndexers <= 0 {
		cfg.NumIndexers = runtime.NumCPU()
	}
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 256
	}

	pi := &PipelinedImporter{
		config:     cfg,
		outDir:     outDir,
		docLens:    make([]uint16, 0, 4000000),
		tokenizeCh: make(chan pipelinedBatch, cfg.BufferSize),
		indexCh:    make(chan pipelinedTokenized, cfg.BufferSize*10),
	}

	for i := 0; i < 256; i++ {
		pi.shards[i] = &pipelinedShard{
			terms: make(map[uint64]*pipelinedPostings, 10000),
		}
	}

	// Start tokenization workers
	for i := 0; i < cfg.NumTokenizers; i++ {
		pi.tokenizeWg.Add(1)
		go pi.tokenizeWorker()
	}

	// Start indexing workers
	for i := 0; i < cfg.NumIndexers; i++ {
		pi.indexWg.Add(1)
		go pi.indexWorker()
	}

	return pi
}

// tokenizeWorker tokenizes batches from the input channel.
func (pi *PipelinedImporter) tokenizeWorker() {
	defer pi.tokenizeWg.Done()

	hashBuf := make([]uint64, 0, 512)
	freqBuf := make([]uint16, 0, 512)
	freqMap := make(map[uint64]uint16, 256)

	for batch := range pi.tokenizeCh {
		for i, text := range batch.texts {
			docID := batch.startDocID + uint32(i)

			// Clear buffers
			hashBuf = hashBuf[:0]
			freqBuf = freqBuf[:0]
			clear(freqMap)

			// Tokenize
			docLen := tokenizePipelined(text, freqMap)

			// Extract results
			for hash, freq := range freqMap {
				hashBuf = append(hashBuf, hash)
				freqBuf = append(freqBuf, freq)
			}

			if docLen > 65535 {
				docLen = 65535
			}

			// Copy for channel (avoid races)
			hashes := make([]uint64, len(hashBuf))
			freqs := make([]uint16, len(freqBuf))
			copy(hashes, hashBuf)
			copy(freqs, freqBuf)

			pi.indexCh <- pipelinedTokenized{
				docID:  docID,
				docLen: uint16(docLen),
				hashes: hashes,
				freqs:  freqs,
			}
		}
	}
}

// tokenizePipelined is the tokenization function for the pipeline.
func tokenizePipelined(text string, freqs map[uint64]uint16) int {
	if len(text) == 0 {
		return 0
	}

	data := []byte(text)
	n := len(data)
	tokenCount := 0

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
			freqs[hash]++
			tokenCount++
		}
	}

	return tokenCount
}

// indexPostingEntry is used for buffering postings in indexWorker.
type indexPostingEntry struct {
	docID uint32
	freq  uint16
}

// indexWorker accumulates tokenized documents into shards.
func (pi *PipelinedImporter) indexWorker() {
	defer pi.indexWg.Done()

	// Per-worker shard buffers to reduce lock contention
	shardBuffers := make([]map[uint64][]indexPostingEntry, 256)
	for i := 0; i < 256; i++ {
		shardBuffers[i] = make(map[uint64][]indexPostingEntry, 100)
	}

	flushCounter := 0
	const flushInterval = 1000 // Flush every N documents

	for tdoc := range pi.indexCh {
		// Record document length
		pi.docLensMu.Lock()
		for int(tdoc.docID) >= len(pi.docLens) {
			pi.docLens = append(pi.docLens, 0)
		}
		pi.docLens[tdoc.docID] = tdoc.docLen
		pi.docLensMu.Unlock()

		pi.docCount.Add(1)
		pi.totalLen.Add(uint64(tdoc.docLen))

		// Buffer postings by shard
		for i, hash := range tdoc.hashes {
			shardID := hash & 0xFF
			shardBuffers[shardID][hash] = append(shardBuffers[shardID][hash],
				indexPostingEntry{docID: tdoc.docID, freq: tdoc.freqs[i]})
		}

		flushCounter++
		if flushCounter >= flushInterval {
			// Flush buffers to shards
			pi.flushBuffers(shardBuffers)
			for i := 0; i < 256; i++ {
				shardBuffers[i] = make(map[uint64][]indexPostingEntry, 100)
			}
			flushCounter = 0
		}
	}

	// Final flush
	pi.flushBuffers(shardBuffers)
}

// flushBuffers merges buffered postings into shards.
func (pi *PipelinedImporter) flushBuffers(buffers []map[uint64][]indexPostingEntry) {
	for shardID := 0; shardID < 256; shardID++ {
		if len(buffers[shardID]) == 0 {
			continue
		}

		shard := pi.shards[shardID]
		shard.mu.Lock()
		for hash, postings := range buffers[shardID] {
			pl, exists := shard.terms[hash]
			if !exists {
				pl = &pipelinedPostings{
					docIDs: make([]uint32, 0, 64),
					freqs:  make([]uint16, 0, 64),
				}
				shard.terms[hash] = pl
			}
			for _, p := range postings {
				pl.docIDs = append(pl.docIDs, p.docID)
				pl.freqs = append(pl.freqs, p.freq)
			}
		}
		shard.mu.Unlock()
	}
}

// AddBatch adds a batch of documents to the pipeline.
func (pi *PipelinedImporter) AddBatch(startDocID uint32, texts []string) {
	if len(texts) == 0 {
		return
	}
	pi.tokenizeCh <- pipelinedBatch{
		startDocID: startDocID,
		texts:      texts,
	}
}

// Finish closes the pipeline and returns the searchable index.
func (pi *PipelinedImporter) Finish() (*SegmentedIndex, error) {
	// Close tokenize channel and wait for tokenizers to finish
	close(pi.tokenizeCh)
	pi.tokenizeWg.Wait()

	// Close index channel and wait for indexers to finish
	close(pi.indexCh)
	pi.indexWg.Wait()

	numDocs := int(pi.docCount.Load())
	if numDocs == 0 {
		return nil, nil
	}

	avgDocLen := float64(pi.totalLen.Load()) / float64(numDocs)
	terms := make(map[string]*SegmentPostings)

	for shardID := 0; shardID < 256; shardID++ {
		shard := pi.shards[shardID]
		for hash, pl := range shard.terms {
			hashKey := hashToKey(hash)
			terms[hashKey] = &SegmentPostings{
				DocIDs: pl.docIDs,
				Freqs:  pl.freqs,
			}
		}
	}

	docLensMap := make(map[uint32]uint16, numDocs)
	for i, dl := range pi.docLens {
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
		docLens:   pi.docLens,
	}, nil
}

// ImportParallel imports documents from a text iterator using pipelined processing.
func (pi *PipelinedImporter) ImportParallel(ctx context.Context, texts []string, batchSize int) error {
	if batchSize <= 0 {
		batchSize = 1000
	}

	docID := uint32(0)
	for i := 0; i < len(texts); i += batchSize {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		end := i + batchSize
		if end > len(texts) {
			end = len(texts)
		}

		pi.AddBatch(docID, texts[i:end])
		docID += uint32(end - i)
	}

	return nil
}

// FastBatchImporter combines reading and indexing with minimal overhead.
type FastBatchImporter struct {
	shards    [256]*fastShard
	docLens   []uint16
	docCount  atomic.Uint64
	totalLen  atomic.Uint64
	docLensMu sync.Mutex
}

type fastShard struct {
	mu    sync.Mutex
	terms map[uint64]*fastPostings
}

type fastPostings struct {
	docIDs []uint32
	freqs  []uint16
}

// NewFastBatchImporter creates a fast batch importer.
func NewFastBatchImporter() *FastBatchImporter {
	fbi := &FastBatchImporter{
		docLens: make([]uint16, 0, 4000000),
	}

	for i := 0; i < 256; i++ {
		fbi.shards[i] = &fastShard{
			terms: make(map[uint64]*fastPostings, 10000),
		}
	}

	return fbi
}

// AddBatch adds a batch using parallel workers.
func (fbi *FastBatchImporter) AddBatch(docIDs []uint32, texts []string) {
	if len(texts) == 0 {
		return
	}

	numDocs := len(texts)
	numWorkers := runtime.NumCPU() * 4
	if numWorkers > numDocs {
		numWorkers = numDocs
	}

	// Worker-local shard postings
	type posting struct {
		hash  uint64
		docID uint32
		freq  uint16
	}
	workerShards := make([][][]posting, numWorkers)
	for w := 0; w < numWorkers; w++ {
		workerShards[w] = make([][]posting, 256)
		for s := 0; s < 256; s++ {
			workerShards[w][s] = make([]posting, 0, 64)
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
			myShards := workerShards[workerID]

			for i := start; i < end; i++ {
				docID := docIDs[i]
				clear(freqs)

				// Fast tokenization
				docLen := tokenizePipelined(texts[i], freqs)
				if docLen > 65535 {
					docLen = 65535
				}
				docLensLocal[i] = uint16(docLen)

				// Distribute to shards
				for hash, freq := range freqs {
					shardID := hash & 0xFF
					myShards[shardID] = append(myShards[shardID],
						posting{hash: hash, docID: docID, freq: freq})
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
	fbi.docCount.Add(uint64(numDocs))
	fbi.totalLen.Add(totalLen)

	fbi.docLensMu.Lock()
	fbi.docLens = append(fbi.docLens, docLensLocal...)
	fbi.docLensMu.Unlock()

	// Phase 2: Parallel shard merging
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
				shard := fbi.shards[shardID]

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
							pl = &fastPostings{
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
func (fbi *FastBatchImporter) Finish() (*SegmentedIndex, error) {
	numDocs := int(fbi.docCount.Load())
	if numDocs == 0 {
		return nil, nil
	}

	avgDocLen := float64(fbi.totalLen.Load()) / float64(numDocs)
	terms := make(map[string]*SegmentPostings)

	for shardID := 0; shardID < 256; shardID++ {
		shard := fbi.shards[shardID]
		for hash, pl := range shard.terms {
			hashKey := hashToKey(hash)
			terms[hashKey] = &SegmentPostings{
				DocIDs: pl.docIDs,
				Freqs:  pl.freqs,
			}
		}
	}

	docLensMap := make(map[uint32]uint16, numDocs)
	for i, dl := range fbi.docLens {
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
		docLens:   fbi.docLens,
	}, nil
}
