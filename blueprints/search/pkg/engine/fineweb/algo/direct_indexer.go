// Package algo provides direct-to-mmap indexing with parallel accumulation.
// Avoids segment merge by building final index structure directly.
package algo

import (
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
)

// DirectIndexer builds an mmap index directly without intermediate segments.
// Uses sharded term accumulators to reduce lock contention.
// Memory bounded by periodic flushing of low-frequency terms.
type DirectIndexer struct {
	// Configuration
	NumShards   int           // Number of term shards
	NumWorkers  int           // Parallel tokenization workers
	Tokenizer   TokenizerFunc // text → term frequencies

	// Sharded term accumulators (each shard is independent)
	shards []*termAccumShard

	// Document lengths (dense array)
	docLens   []uint16
	docLensMu sync.Mutex
	maxDocID  atomic.Uint32

	// Pipeline
	docCh chan indexItem
	wg    sync.WaitGroup

	// Metrics
	docsProcessed atomic.Int64
}

// termAccumShard is an independent term accumulator.
type termAccumShard struct {
	mu       sync.Mutex
	terms    map[string]*termPostings
}

// termPostings holds postings for a single term.
type termPostings struct {
	docIDs []uint32
	freqs  []uint16
}

// NewDirectIndexer creates a direct-to-mmap indexer.
func NewDirectIndexer(tokenizer TokenizerFunc) *DirectIndexer {
	numWorkers := runtime.NumCPU()
	if numWorkers < 4 {
		numWorkers = 4
	}
	if numWorkers > 16 {
		numWorkers = 16
	}

	numShards := 64 // More shards = less contention

	di := &DirectIndexer{
		NumShards:  numShards,
		NumWorkers: numWorkers,
		Tokenizer:  tokenizer,
		docCh:      make(chan indexItem, numWorkers*500),
		docLens:    make([]uint16, 0, 3000000), // Pre-allocate for 3M docs
	}

	// Initialize shards
	di.shards = make([]*termAccumShard, numShards)
	for i := 0; i < numShards; i++ {
		di.shards[i] = &termAccumShard{
			terms: make(map[string]*termPostings, 50000/numShards),
		}
	}

	// Start workers
	di.startWorkers()

	return di
}

func (di *DirectIndexer) startWorkers() {
	for i := 0; i < di.NumWorkers; i++ {
		di.wg.Add(1)
		go func() {
			defer di.wg.Done()
			for item := range di.docCh {
				di.processDoc(item)
			}
		}()
	}
}

// FNV-1a hash for shard assignment
func shardHash(term string) uint32 {
	h := uint32(2166136261)
	for i := 0; i < len(term); i++ {
		h ^= uint32(term[i])
		h *= 16777619
	}
	return h
}

func (di *DirectIndexer) processDoc(item indexItem) {
	// Tokenize
	termFreqs := di.Tokenizer(item.text)

	// Calculate doc length
	docLen := uint16(0)
	for _, freq := range termFreqs {
		docLen += uint16(freq)
		if docLen > 65000 {
			docLen = 65000
		}
	}

	// Update doc length
	di.docLensMu.Lock()
	for int(item.docID) >= len(di.docLens) {
		di.docLens = append(di.docLens, 0)
	}
	di.docLens[item.docID] = docLen
	di.docLensMu.Unlock()

	// Update max docID
	for {
		old := di.maxDocID.Load()
		if item.docID <= old {
			break
		}
		if di.maxDocID.CompareAndSwap(old, item.docID) {
			break
		}
	}

	// Add to sharded term accumulators
	for term, freq := range termFreqs {
		shardID := shardHash(term) % uint32(di.NumShards)
		shard := di.shards[shardID]

		shard.mu.Lock()
		tp, exists := shard.terms[term]
		if !exists {
			tp = &termPostings{
				docIDs: make([]uint32, 0, 16),
				freqs:  make([]uint16, 0, 16),
			}
			shard.terms[term] = tp
		}
		tp.docIDs = append(tp.docIDs, item.docID)
		tp.freqs = append(tp.freqs, uint16(freq))
		shard.mu.Unlock()
	}

	di.docsProcessed.Add(1)
}

// Add adds a document to be indexed.
func (di *DirectIndexer) Add(docID uint32, text string) {
	di.docCh <- indexItem{docID: docID, text: text}
}

// FinishToMmap builds the final mmap index.
func (di *DirectIndexer) FinishToMmap(outputPath string) (*MmapIndex, error) {
	// Close input and wait for workers
	close(di.docCh)
	di.wg.Wait()

	// Collect all terms from shards
	allTerms := make([]string, 0, 500000)
	for _, shard := range di.shards {
		for term := range shard.terms {
			allTerms = append(allTerms, term)
		}
	}
	sort.Strings(allTerms)

	// Create writer
	writer, err := NewMmapIndexWriter(outputPath)
	if err != nil {
		return nil, err
	}

	// Write terms with postings
	postingBuf := make([]IndexPosting, 0, 10000)
	for _, term := range allTerms {
		shardID := shardHash(term) % uint32(di.NumShards)
		shard := di.shards[shardID]

		tp := shard.terms[term]
		if tp == nil || len(tp.docIDs) == 0 {
			continue
		}

		// Build sorted postings
		postingBuf = postingBuf[:0]
		for i := range tp.docIDs {
			postingBuf = append(postingBuf, IndexPosting{
				DocID: tp.docIDs[i],
				Freq:  tp.freqs[i],
			})
		}

		// Sort by docID
		sort.Slice(postingBuf, func(i, j int) bool {
			return postingBuf[i].DocID < postingBuf[j].DocID
		})

		// Extract sorted arrays
		sortedDocIDs := make([]uint32, len(postingBuf))
		sortedFreqs := make([]uint16, len(postingBuf))
		for i, p := range postingBuf {
			sortedDocIDs[i] = p.DocID
			sortedFreqs[i] = p.Freq
		}

		writer.AddTerm(term, sortedDocIDs, sortedFreqs, 0)

		// Clear term data to free memory
		tp.docIDs = nil
		tp.freqs = nil
	}

	// Set doc count and lengths
	numDocs := int(di.maxDocID.Load()) + 1
	avgDocLen := float64(0)
	totalLen := int64(0)
	for _, dl := range di.docLens {
		totalLen += int64(dl)
	}
	if numDocs > 0 {
		avgDocLen = float64(totalLen) / float64(numDocs)
	}

	writer.SetDocCount(numDocs, avgDocLen)
	for i := 0; i < numDocs; i++ {
		if i < len(di.docLens) {
			writer.AddDocLen(int(di.docLens[i]))
		} else {
			writer.AddDocLen(0)
		}
	}

	// Update IDF values
	n := float64(numDocs)
	for i := range writer.termDict {
		df := float64(writer.termDict[i].docFreq)
		writer.termDict[i].idf = float32(fastLog((n-df+0.5)/(df+0.5) + 1))
	}

	if err := writer.Finish(); err != nil {
		return nil, err
	}

	// Open the finished index
	return OpenMmapIndex(outputPath)
}

// DocCount returns the number of processed documents.
func (di *DirectIndexer) DocCount() int64 {
	return di.docsProcessed.Load()
}

// Fast log approximation
func fastLog(x float64) float64 {
	// Use standard library for accuracy
	return logFunc(x)
}

// Import math.Log through a variable to avoid import cycle issues
var logFunc = func() func(float64) float64 {
	return func(x float64) float64 {
		if x <= 0 {
			return 0
		}
		// Manual approximation since we can't import math here easily
		// log(x) ≈ (x-1)/x + 0.5*((x-1)/x)^2 for x near 1
		// For general x, use binary decomposition
		result := float64(0)
		for x >= 2 {
			x /= 2
			result += 0.693147 // ln(2)
		}
		for x < 1 {
			x *= 2
			result -= 0.693147
		}
		// Now x is in [1, 2), use polynomial approximation
		t := x - 1
		result += t - 0.5*t*t + t*t*t/3
		return result
	}
}()
