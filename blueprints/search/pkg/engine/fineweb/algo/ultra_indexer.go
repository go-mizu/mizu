// Package algo provides UltraIndexer - Go port of Rust ultra.rs optimizations.
// Key techniques from Rust:
// 1. Pre-computed CHAR_LUT for instant lowercase check
// 2. Hash-as-key: uint64 FNV hash instead of string (massive memory savings)
// 3. Hash while tokenizing - no string allocation
// 4. Phase-based batch processing
// 5. 16 shards with fine-grained locking
package algo

import (
	"bufio"
	"encoding/binary"
	"os"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"unsafe"
)

// Pre-computed character lookup table
// 0 = delimiter, otherwise = lowercase ASCII value
var ultraCharLUT [256]byte

func init() {
	for i := 0; i < 256; i++ {
		if i >= 'a' && i <= 'z' {
			ultraCharLUT[i] = byte(i)
		} else if i >= 'A' && i <= 'Z' {
			ultraCharLUT[i] = byte(i) | 0x20 // lowercase
		} else if i >= '0' && i <= '9' {
			ultraCharLUT[i] = byte(i)
		} else {
			ultraCharLUT[i] = 0 // delimiter
		}
	}
}

// FNV-1a constants
const (
	fnvOffset = 0xcbf29ce484222325
	fnvPrime  = 0x100000001b3
)

// Shard configuration - 32 shards balances contention vs memory
const (
	ultraNumShards = 32
	ultraShardMask = ultraNumShards - 1
)

// UltraConfig configures the ultra indexer.
type UltraConfig struct {
	NumWorkers  int // Parallel workers (0 = auto)
	SegmentDocs int // Docs per segment for disk flush
}

// UltraIndexer is a high-performance indexer using hash-as-key approach.
type UltraIndexer struct {
	config  UltraConfig
	outDir  string

	// Sharded index - uses hash as key, not string
	shards [ultraNumShards]*ultraShard

	// Document state
	docLens   []uint16
	docCount  atomic.Uint64
	totalLen  atomic.Uint64
	docLensMu sync.Mutex
}

// ultraShard holds term postings for a hash range.
type ultraShard struct {
	mu       sync.Mutex
	terms    map[uint64]*ultraPostings // hash -> postings
}

// ultraPostings stores doc IDs and frequencies.
type ultraPostings struct {
	docIDs []uint32
	freqs  []uint16
}

// ultraTermFreq is the result of tokenization.
type ultraTermFreq struct {
	hash uint64
	freq uint16
}

// ultraDocResult holds tokenization result for a document.
type ultraDocResult struct {
	docID  uint32
	docLen uint16
	terms  []ultraTermFreq
}

// NewUltraIndexer creates a new ultra indexer.
func NewUltraIndexer(outDir string, cfg UltraConfig) *UltraIndexer {
	if cfg.NumWorkers <= 0 {
		// Optimal worker count is 1.5-2x CPU count based on profiling
		cfg.NumWorkers = runtime.NumCPU() * 2
	}
	if cfg.NumWorkers > 32 {
		cfg.NumWorkers = 32
	}
	if cfg.SegmentDocs <= 0 {
		cfg.SegmentDocs = 500000
	}

	os.MkdirAll(outDir, 0755)

	ui := &UltraIndexer{
		config:  cfg,
		outDir:  outDir,
		docLens: make([]uint16, 0, 4000000),
	}

	// Initialize shards with smaller initial capacity to reduce memory
	for i := 0; i < ultraNumShards; i++ {
		ui.shards[i] = &ultraShard{
			terms: make(map[uint64]*ultraPostings, 20000),
		}
	}

	return ui
}


// TokenizeToHashReuse tokenizes text into a reusable map.
// Returns the total token count for doc length calculation.
// Exported for profiling.
func TokenizeToHashReuse(text string, freqs map[uint64]uint16) int {
	// Clear the map for reuse (optimized clear pattern)
	clear(freqs)

	data := *(*[]byte)(unsafe.Pointer(&text))
	n := len(data)
	if n == 0 {
		return 0
	}

	totalTokens := 0
	i := 0
	for i < n {
		// Skip delimiters using LUT
		for i < n && ultraCharLUT[data[i]] == 0 {
			i++
		}
		if i >= n {
			break
		}

		// Hash while scanning token
		start := i
		hash := uint64(fnvOffset)
		for i < n {
			c := ultraCharLUT[data[i]]
			if c == 0 {
				break
			}
			hash ^= uint64(c)
			hash *= fnvPrime
			i++
		}

		// Keep tokens of reasonable length
		tokenLen := i - start
		if tokenLen >= 2 && tokenLen <= 32 {
			freqs[hash]++
			totalTokens++
		}
	}

	return totalTokens
}

// tokenizeFresh creates a new map for each document (avoids clear() overhead for large maps).
func tokenizeFresh(text string) (map[uint64]uint16, int) {
	data := *(*[]byte)(unsafe.Pointer(&text))
	n := len(data)
	if n == 0 {
		return nil, 0
	}

	// Estimate capacity based on text length (avg 6 chars per token)
	freqs := make(map[uint64]uint16, n/6)
	totalTokens := 0
	i := 0

	for i < n {
		// Skip delimiters
		for i < n && ultraCharLUT[data[i]] == 0 {
			i++
		}
		if i >= n {
			break
		}

		// Hash while scanning
		start := i
		hash := uint64(fnvOffset)
		for i < n {
			c := ultraCharLUT[data[i]]
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
			totalTokens++
		}
	}

	return freqs, totalTokens
}

// tokenizeToSlice collects hashes in a slice instead of map for faster collection.
// The caller should then sort and dedupe the hashes.
func tokenizeToSlice(text string, hashes []uint64) ([]uint64, int) {
	hashes = hashes[:0] // Reuse slice

	data := *(*[]byte)(unsafe.Pointer(&text))
	n := len(data)
	if n == 0 {
		return hashes, 0
	}

	totalTokens := 0
	i := 0
	for i < n {
		// Skip delimiters - unrolled for common case
		for i < n && ultraCharLUT[data[i]] == 0 {
			i++
		}
		if i >= n {
			break
		}

		// Hash while scanning
		start := i
		hash := uint64(fnvOffset)

		// Unrolled inner loop - process 4 bytes at a time when possible
		for i+4 <= n {
			c0 := ultraCharLUT[data[i]]
			c1 := ultraCharLUT[data[i+1]]
			c2 := ultraCharLUT[data[i+2]]
			c3 := ultraCharLUT[data[i+3]]

			if c0 == 0 {
				break
			}
			hash ^= uint64(c0)
			hash *= fnvPrime
			i++

			if c1 == 0 {
				break
			}
			hash ^= uint64(c1)
			hash *= fnvPrime
			i++

			if c2 == 0 {
				break
			}
			hash ^= uint64(c2)
			hash *= fnvPrime
			i++

			if c3 == 0 {
				break
			}
			hash ^= uint64(c3)
			hash *= fnvPrime
			i++
		}

		// Handle remaining bytes
		for i < n {
			c := ultraCharLUT[data[i]]
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

	return hashes, totalTokens
}

// tokenizeToHash tokenizes text and returns hash-freq pairs (legacy).
func tokenizeToHash(text string) []ultraTermFreq {
	freqs := make(map[uint64]uint16, len(text)/6)
	TokenizeToHashReuse(text, freqs)

	result := make([]ultraTermFreq, 0, len(freqs))
	for hash, freq := range freqs {
		result = append(result, ultraTermFreq{hash: hash, freq: freq})
	}
	return result
}

// AddBatch indexes a batch of documents using phase-based processing.
// Optimized for reduced lock contention and GC pressure.
func (ui *UltraIndexer) AddBatch(docIDs []uint32, texts []string) {
	if len(texts) == 0 {
		return
	}

	numDocs := len(texts)
	numWorkers := ui.config.NumWorkers
	if numWorkers > numDocs {
		numWorkers = numDocs
	}

	// Pre-allocate per-worker shard postings to avoid Phase 3 contention
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

	// Phase 1: Parallel tokenization with per-worker shard accumulation
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
			// Fresh map per worker - sized for typical doc
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

				// Distribute postings to per-worker shard slices
				for hash, freq := range freqs {
					shardID := hash & ultraShardMask
					myShards[shardID] = append(myShards[shardID], posting{hash, docID, freq})
				}

				// Recreate map every 100 docs to prevent growth
				recreateCounter++
				if recreateCounter >= 100 {
					freqs = make(map[uint64]uint16, 256)
					recreateCounter = 0
				}
			}
		}(w, start, end)
	}
	wg.Wait()

	// Phase 2: Collect doc lengths
	var totalLen uint64
	for _, dl := range docLensLocal {
		totalLen += uint64(dl)
	}
	ui.docCount.Add(uint64(numDocs))
	ui.totalLen.Add(totalLen)

	ui.docLensMu.Lock()
	ui.docLens = append(ui.docLens, docLensLocal...)
	ui.docLensMu.Unlock()

	// Phase 3: Parallel shard updates - each worker handles a range of shards
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
				shard := ui.shards[shardID]

				// Collect postings from all workers for this shard
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
							pl = &ultraPostings{
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

// Add indexes a single document (for compatibility).
func (ui *UltraIndexer) Add(docID uint32, text string) {
	ui.AddBatch([]uint32{docID}, []string{text})
}

// AddBatchFast is an optimized version that reduces sync points.
// Uses slice-based collection instead of maps and overlaps phases.
func (ui *UltraIndexer) AddBatchFast(docIDs []uint32, texts []string) {
	if len(texts) == 0 {
		return
	}

	numDocs := len(texts)
	numWorkers := ui.config.NumWorkers
	if numWorkers > numDocs {
		numWorkers = numDocs
	}

	// Per-worker data - each worker processes independently then merges
	type workerResult struct {
		postings []struct {
			shardID int
			hash    uint64
			docID   uint32
			freq    uint16
		}
		docLens  []uint16
		startIdx int
		totalLen uint64
	}

	results := make([]workerResult, numWorkers)
	var wg sync.WaitGroup
	batchSize := (numDocs + numWorkers - 1) / numWorkers

	// Single phase: parallel tokenization with no synchronization
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

			// Pre-allocate for this worker's docs
			numLocal := end - start
			avgPostingsPerDoc := 200 // Estimate
			localPostings := make([]struct {
				shardID int
				hash    uint64
				docID   uint32
				freq    uint16
			}, 0, numLocal*avgPostingsPerDoc)
			localDocLens := make([]uint16, numLocal)

			// Reusable hash collection slice
			hashes := make([]uint64, 0, 1024)
			var totalLen uint64

			for i := start; i < end; i++ {
				docID := docIDs[i]

				// Tokenize using slice-based approach (faster than map)
				var docLen int
				hashes, docLen = tokenizeToSlice(texts[i], hashes)
				if docLen > 65535 {
					docLen = 65535
				}
				localDocLens[i-start] = uint16(docLen)
				totalLen += uint64(docLen)

				// Sort hashes and count frequencies in one pass
				if len(hashes) > 1 {
					// Simple sort for small slices
					sortUint64(hashes)
				}

				// Dedupe and count
				prevHash := uint64(0)
				freq := uint16(0)
				for _, h := range hashes {
					if h == prevHash && freq > 0 {
						freq++
					} else {
						if freq > 0 {
							shardID := int(prevHash & ultraShardMask)
							localPostings = append(localPostings, struct {
								shardID int
								hash    uint64
								docID   uint32
								freq    uint16
							}{shardID, prevHash, docID, freq})
						}
						prevHash = h
						freq = 1
					}
				}
				// Don't forget last term
				if freq > 0 {
					shardID := int(prevHash & ultraShardMask)
					localPostings = append(localPostings, struct {
						shardID int
						hash    uint64
						docID   uint32
						freq    uint16
					}{shardID, prevHash, docID, freq})
				}
			}

			results[workerID] = workerResult{
				postings: localPostings,
				docLens:  localDocLens,
				startIdx: start,
				totalLen: totalLen,
			}
		}(w, start, end)
	}
	wg.Wait()

	// Merge doc lengths (minimal sync)
	docLensLocal := make([]uint16, numDocs)
	var totalLen uint64
	for _, r := range results {
		if r.docLens != nil {
			copy(docLensLocal[r.startIdx:], r.docLens)
			totalLen += r.totalLen
		}
	}
	ui.docCount.Add(uint64(numDocs))
	ui.totalLen.Add(totalLen)

	ui.docLensMu.Lock()
	ui.docLens = append(ui.docLens, docLensLocal...)
	ui.docLensMu.Unlock()

	// Parallel shard updates
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
				shard := ui.shards[shardID]

				// Collect postings for this shard from all worker results
				shard.mu.Lock()
				for _, r := range results {
					for _, p := range r.postings {
						if p.shardID != shardID {
							continue
						}
						pl, exists := shard.terms[p.hash]
						if !exists {
							pl = &ultraPostings{
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

// sortUint64 sorts a slice of uint64 using insertion sort (fast for small slices).
func sortUint64(a []uint64) {
	n := len(a)
	for i := 1; i < n; i++ {
		key := a[i]
		j := i - 1
		for j >= 0 && a[j] > key {
			a[j+1] = a[j]
			j--
		}
		a[j+1] = key
	}
}

// hashToKey converts a uint64 hash to a string key.
func hashToKey(hash uint64) string {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, hash)
	return string(buf)
}

// Finish completes indexing and returns a searchable index.
func (ui *UltraIndexer) Finish() (*SegmentedIndex, error) {
	numDocs := int(ui.docCount.Load())
	if numDocs == 0 {
		return nil, nil
	}

	avgDocLen := float64(ui.totalLen.Load()) / float64(numDocs)

	// Build segment with hash-based term lookup
	terms := make(map[string]*SegmentPostings)

	// Collect all terms from all shards
	for shardID := 0; shardID < ultraNumShards; shardID++ {
		shard := ui.shards[shardID]
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

	// Build doc lengths map
	docLensMap := make(map[uint32]uint16, numDocs)
	for i, dl := range ui.docLens {
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
		docLens:   ui.docLens,
	}, nil
}

// FinishToFile writes index directly to disk.
func (ui *UltraIndexer) FinishToFile(outputPath string) error {
	numDocs := int(ui.docCount.Load())
	if numDocs == 0 {
		return nil
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriterSize(f, 8*1024*1024)

	// Header
	w.WriteString("ULTRA001")
	binary.Write(w, binary.LittleEndian, uint64(numDocs))
	binary.Write(w, binary.LittleEndian, ui.totalLen.Load())
	binary.Write(w, binary.LittleEndian, uint64(ultraNumShards))

	// Write each shard
	for shardID := 0; shardID < ultraNumShards; shardID++ {
		shard := ui.shards[shardID]
		binary.Write(w, binary.LittleEndian, uint64(len(shard.terms)))

		for hash, pl := range shard.terms {
			binary.Write(w, binary.LittleEndian, hash)
			binary.Write(w, binary.LittleEndian, uint32(len(pl.docIDs)))
			for i := range pl.docIDs {
				binary.Write(w, binary.LittleEndian, pl.docIDs[i])
				binary.Write(w, binary.LittleEndian, pl.freqs[i])
			}
		}
	}

	// Write doc lengths
	for _, dl := range ui.docLens {
		binary.Write(w, binary.LittleEndian, dl)
	}

	return w.Flush()
}

// UltraQueryTokenize tokenizes a query string using the hash algorithm.
func UltraQueryTokenize(query string) []string {
	data := *(*[]byte)(unsafe.Pointer(&query))
	n := len(data)
	if n == 0 {
		return nil
	}

	var result []string
	i := 0
	for i < n {
		for i < n && ultraCharLUT[data[i]] == 0 {
			i++
		}
		if i >= n {
			break
		}

		start := i
		hash := uint64(fnvOffset)
		for i < n {
			c := ultraCharLUT[data[i]]
			if c == 0 {
				break
			}
			hash ^= uint64(c)
			hash *= fnvPrime
			i++
		}

		tokenLen := i - start
		if tokenLen >= 2 && tokenLen <= 32 {
			result = append(result, hashToKey(hash))
		}
	}
	return result
}

// UltraSegmentedIndex is a search-optimized index using hash keys.
type UltraSegmentedIndex struct {
	shards    [ultraNumShards]map[uint64]*ultraPostings
	docLens   []uint16
	numDocs   int
	avgDocLen float64
}

// LoadUltraIndex loads an ultra index from file.
func LoadUltraIndex(path string) (*UltraSegmentedIndex, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := bufio.NewReaderSize(f, 8*1024*1024)

	magic := make([]byte, 8)
	r.Read(magic)
	if string(magic) != "ULTRA001" {
		return nil, os.ErrInvalid
	}

	var numDocs, totalLen, numShards uint64
	binary.Read(r, binary.LittleEndian, &numDocs)
	binary.Read(r, binary.LittleEndian, &totalLen)
	binary.Read(r, binary.LittleEndian, &numShards)

	idx := &UltraSegmentedIndex{
		docLens:   make([]uint16, numDocs),
		numDocs:   int(numDocs),
		avgDocLen: float64(totalLen) / float64(numDocs),
	}

	for shardID := 0; shardID < int(numShards) && shardID < ultraNumShards; shardID++ {
		var termCount uint64
		binary.Read(r, binary.LittleEndian, &termCount)

		idx.shards[shardID] = make(map[uint64]*ultraPostings, termCount)

		for i := uint64(0); i < termCount; i++ {
			var hash uint64
			var entryCount uint32
			binary.Read(r, binary.LittleEndian, &hash)
			binary.Read(r, binary.LittleEndian, &entryCount)

			pl := &ultraPostings{
				docIDs: make([]uint32, entryCount),
				freqs:  make([]uint16, entryCount),
			}

			for j := uint32(0); j < entryCount; j++ {
				binary.Read(r, binary.LittleEndian, &pl.docIDs[j])
				binary.Read(r, binary.LittleEndian, &pl.freqs[j])
			}

			idx.shards[shardID][hash] = pl
		}
	}

	for i := 0; i < int(numDocs); i++ {
		binary.Read(r, binary.LittleEndian, &idx.docLens[i])
	}

	return idx, nil
}

// Search performs BM25 search on the ultra index.
func (idx *UltraSegmentedIndex) Search(query string, limit int) []MmapSearchResult {
	if idx.numDocs == 0 {
		return nil
	}

	data := *(*[]byte)(unsafe.Pointer(&query))
	n := len(data)
	if n == 0 {
		return nil
	}

	var queryHashes []uint64
	i := 0
	for i < n {
		for i < n && ultraCharLUT[data[i]] == 0 {
			i++
		}
		if i >= n {
			break
		}

		start := i
		hash := uint64(fnvOffset)
		for i < n {
			c := ultraCharLUT[data[i]]
			if c == 0 {
				break
			}
			hash ^= uint64(c)
			hash *= fnvPrime
			i++
		}

		tokenLen := i - start
		if tokenLen >= 2 && tokenLen <= 32 {
			queryHashes = append(queryHashes, hash)
		}
	}

	if len(queryHashes) == 0 {
		return nil
	}

	const k1 = 1.2
	const b = 0.75
	n64 := float64(idx.numDocs)

	scores := make(map[uint32]float32)

	for _, hash := range queryHashes {
		shardID := hash & ultraShardMask
		pl, exists := idx.shards[shardID][hash]
		if !exists {
			continue
		}

		df := float64(len(pl.docIDs))
		idf := float32(((n64-df+0.5)/(df+0.5) + 1))

		for i, docID := range pl.docIDs {
			freq := float64(pl.freqs[i])
			docLen := float64(idx.docLens[docID])
			tfNorm := float32(freq * (k1 + 1) / (freq + k1*(1-b+b*docLen/idx.avgDocLen)))
			scores[docID] += idf * tfNorm
		}
	}

	results := make([]MmapSearchResult, 0, len(scores))
	for docID, score := range scores {
		results = append(results, MmapSearchResult{DocID: docID, Score: score})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > limit {
		results = results[:limit]
	}

	return results
}
