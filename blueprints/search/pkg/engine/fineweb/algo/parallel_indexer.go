// Package algo provides parallel indexing infrastructure for high-performance FTS.
package algo

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// IndexPosting represents a single posting (docID, freq).
type IndexPosting struct {
	DocID uint32
	Freq  uint16
}

// TokenizerFunc tokenizes text and returns term frequencies.
type TokenizerFunc func(text string) map[string]int

// ParallelIndexer provides high-performance parallel document indexing.
type ParallelIndexer struct {
	NumWorkers int
	BatchSize  int
	Tokenizer  TokenizerFunc
}

// NewParallelIndexer creates a new parallel indexer.
func NewParallelIndexer(tokenizer TokenizerFunc) *ParallelIndexer {
	numWorkers := runtime.NumCPU()
	if numWorkers < 2 {
		numWorkers = 2
	}
	if numWorkers > 16 {
		numWorkers = 16 // Cap to avoid too much memory
	}

	return &ParallelIndexer{
		NumWorkers: numWorkers,
		BatchSize:  500,
		Tokenizer:  tokenizer,
	}
}

// textBatch represents a batch of texts for processing.
type textBatch struct {
	texts   []string
	startID uint32
}

// workerResult holds results from a worker.
type workerResult struct {
	terms   map[string][]IndexPosting
	docLens map[uint32]int
}

// IndexTexts indexes texts in parallel and returns merged term postings.
func (pi *ParallelIndexer) IndexTexts(texts []string) (map[string][]IndexPosting, []int) {
	if len(texts) == 0 {
		return make(map[string][]IndexPosting), nil
	}

	// Result channels
	resultCh := make(chan *workerResult, pi.NumWorkers*2)
	var wg sync.WaitGroup

	// Distribute work
	batchSize := (len(texts) + pi.NumWorkers - 1) / pi.NumWorkers
	if batchSize < 100 {
		batchSize = 100
	}

	for start := 0; start < len(texts); start += batchSize {
		end := start + batchSize
		if end > len(texts) {
			end = len(texts)
		}

		wg.Add(1)
		go func(batch []string, startIdx int) {
			defer wg.Done()

			localTerms := make(map[string][]IndexPosting, 10000)
			localDocLens := make(map[uint32]int, len(batch))

			for i, text := range batch {
				docID := uint32(startIdx + i)

				// Tokenize
				termFreqs := pi.Tokenizer(text)

				// Calculate doc length
				docLen := 0
				for _, freq := range termFreqs {
					docLen += freq
				}
				localDocLens[docID] = docLen

				// Add to local posting lists
				for term, freq := range termFreqs {
					localTerms[term] = append(localTerms[term], IndexPosting{
						DocID: docID,
						Freq:  uint16(freq),
					})
				}
			}

			resultCh <- &workerResult{
				terms:   localTerms,
				docLens: localDocLens,
			}
		}(texts[start:end], start)
	}

	// Close result channel when done
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Merge results
	mergedTerms := make(map[string][]IndexPosting, 50000)
	docLens := make([]int, len(texts))

	for result := range resultCh {
		for term, postings := range result.terms {
			mergedTerms[term] = append(mergedTerms[term], postings...)
		}
		for docID, length := range result.docLens {
			docLens[docID] = length
		}
	}

	return mergedTerms, docLens
}

// StreamingIndexer provides streaming parallel indexing for large datasets.
type StreamingIndexer struct {
	NumWorkers int
	BatchSize  int
	Tokenizer  TokenizerFunc

	// Channels
	textCh   chan indexItem
	resultCh chan *workerResult
	doneCh   chan struct{}

	// Results
	termPostings map[string][]IndexPosting
	docLens      []int
	docCount     int64
	mu           sync.Mutex
	wg           sync.WaitGroup
}

type indexItem struct {
	docID uint32
	text  string
}

// NewStreamingIndexer creates a streaming parallel indexer.
func NewStreamingIndexer(tokenizer TokenizerFunc) *StreamingIndexer {
	numWorkers := runtime.NumCPU()
	if numWorkers < 2 {
		numWorkers = 2
	}
	if numWorkers > 16 {
		numWorkers = 16
	}

	si := &StreamingIndexer{
		NumWorkers:   numWorkers,
		BatchSize:    500,
		Tokenizer:    tokenizer,
		textCh:       make(chan indexItem, numWorkers*1000),
		resultCh:     make(chan *workerResult, numWorkers*2),
		doneCh:       make(chan struct{}),
		termPostings: make(map[string][]IndexPosting, 50000),
		docLens:      make([]int, 0, 100000),
	}

	// Start workers
	for range si.NumWorkers {
		si.wg.Add(1)
		go si.streamWorker()
	}

	// Start merger
	go si.merger()

	return si
}

// Add adds a document to the indexer.
func (si *StreamingIndexer) Add(docID uint32, text string) {
	si.textCh <- indexItem{docID: docID, text: text}
	atomic.AddInt64(&si.docCount, 1)
}

// streamWorker processes documents from the channel.
func (si *StreamingIndexer) streamWorker() {
	defer si.wg.Done()

	localTerms := make(map[string][]IndexPosting, 10000)
	localDocLens := make(map[uint32]int, si.BatchSize)
	count := 0

	for item := range si.textCh {
		// Tokenize
		termFreqs := si.Tokenizer(item.text)

		// Calculate doc length
		docLen := 0
		for _, freq := range termFreqs {
			docLen += freq
		}
		localDocLens[item.docID] = docLen

		// Add to local posting lists
		for term, freq := range termFreqs {
			localTerms[term] = append(localTerms[term], IndexPosting{
				DocID: item.docID,
				Freq:  uint16(freq),
			})
		}

		count++

		// Flush periodically
		if count >= si.BatchSize {
			si.resultCh <- &workerResult{
				terms:   localTerms,
				docLens: localDocLens,
			}
			localTerms = make(map[string][]IndexPosting, 10000)
			localDocLens = make(map[uint32]int, si.BatchSize)
			count = 0
		}
	}

	// Flush remaining
	if count > 0 {
		si.resultCh <- &workerResult{
			terms:   localTerms,
			docLens: localDocLens,
		}
	}
}

// merger merges worker results.
func (si *StreamingIndexer) merger() {
	for result := range si.resultCh {
		si.mu.Lock()
		for term, postings := range result.terms {
			si.termPostings[term] = append(si.termPostings[term], postings...)
		}
		// Extend docLens if needed
		for docID, length := range result.docLens {
			for int(docID) >= len(si.docLens) {
				si.docLens = append(si.docLens, 0)
			}
			si.docLens[docID] = length
		}
		si.mu.Unlock()
	}
	close(si.doneCh)
}

// Finish waits for indexing to complete and returns results.
func (si *StreamingIndexer) Finish() (map[string][]IndexPosting, []int) {
	close(si.textCh)
	si.wg.Wait()
	close(si.resultCh)
	<-si.doneCh

	return si.termPostings, si.docLens
}

// DocCount returns the number of indexed documents.
func (si *StreamingIndexer) DocCount() int64 {
	return atomic.LoadInt64(&si.docCount)
}

// TurboIndexer provides maximum throughput by deferring ALL merging to the end.
// Each worker accumulates its own postings with zero lock contention during indexing.
type TurboIndexer struct {
	NumWorkers int
	Tokenizer  TokenizerFunc

	// Input channel with large buffer
	docCh chan indexItem

	// Each worker has its own accumulator (no sharing during indexing)
	workerData []*turboWorkerData

	docCount atomic.Int64
	maxDocID atomic.Uint32
	wg       sync.WaitGroup
}

type turboWorkerData struct {
	terms   map[string][]IndexPosting
	docLens map[uint32]int
}

// NewTurboIndexer creates a new turbo indexer optimized for maximum throughput.
func NewTurboIndexer(tokenizer TokenizerFunc) *TurboIndexer {
	// Use 8 workers - optimal for balancing tokenization vs merge overhead
	numWorkers := 8
	if runtime.NumCPU() < 8 {
		numWorkers = runtime.NumCPU()
	}
	if numWorkers < 4 {
		numWorkers = 4
	}

	// Channel buffer sized for ~10 seconds of work at 50k docs/sec
	// Smaller buffer reduces memory while still allowing I/O overlap
	bufferSize := numWorkers * 5000
	if bufferSize > 50000 {
		bufferSize = 50000
	}

	ti := &TurboIndexer{
		NumWorkers: numWorkers,
		Tokenizer:  tokenizer,
		docCh:      make(chan indexItem, bufferSize),
		workerData: make([]*turboWorkerData, numWorkers),
	}

	// Initialize worker data structures with moderate pre-allocation
	// Maps will grow as needed - avoid over-allocation for smaller datasets
	for i := 0; i < numWorkers; i++ {
		ti.workerData[i] = &turboWorkerData{
			terms:   make(map[string][]IndexPosting, 20000),
			docLens: make(map[uint32]int, 10000),
		}
	}

	// Start workers
	for i := 0; i < numWorkers; i++ {
		ti.wg.Add(1)
		go ti.worker(i)
	}

	return ti
}

// Add adds a document to be indexed.
func (ti *TurboIndexer) Add(docID uint32, text string) {
	ti.docCh <- indexItem{docID: docID, text: text}
	ti.docCount.Add(1)
	// Track max docID for pre-allocation
	for {
		current := ti.maxDocID.Load()
		if docID <= current {
			break
		}
		if ti.maxDocID.CompareAndSwap(current, docID) {
			break
		}
	}
}

// worker processes documents and accumulates results locally (ZERO locking).
func (ti *TurboIndexer) worker(workerID int) {
	defer ti.wg.Done()

	data := ti.workerData[workerID]

	for item := range ti.docCh {
		termFreqs := ti.Tokenizer(item.text)

		// Calculate doc length
		docLen := 0
		for _, freq := range termFreqs {
			docLen += freq
		}
		data.docLens[item.docID] = docLen

		// Add to local posting lists (no locking!)
		for term, freq := range termFreqs {
			data.terms[term] = append(data.terms[term], IndexPosting{
				DocID: item.docID,
				Freq:  uint16(freq),
			})
		}
	}
}

// Finish waits for indexing and performs parallel k-way merge.
func (ti *TurboIndexer) Finish() (map[string][]IndexPosting, []int) {
	close(ti.docCh)
	ti.wg.Wait()

	return ti.parallelMerge()
}

func (ti *TurboIndexer) parallelMerge() (map[string][]IndexPosting, []int) {
	// Use worker 0 as base, merge others into it (avoids creating new map)
	finalTerms := ti.workerData[0].terms

	// Merge remaining workers into worker 0's map
	for i := 1; i < len(ti.workerData); i++ {
		for term, postings := range ti.workerData[i].terms {
			existing := finalTerms[term]
			if existing == nil {
				finalTerms[term] = postings // Direct assignment (no copy)
			} else {
				finalTerms[term] = append(existing, postings...)
			}
		}
		// Clear worker data immediately to help GC release memory
		ti.workerData[i].terms = nil
		ti.workerData[i].docLens = nil
	}

	// Merge doc lengths (pre-allocate array)
	maxDocID := ti.maxDocID.Load()
	docLens := make([]int, maxDocID+1)
	for _, data := range ti.workerData {
		if data != nil && data.docLens != nil {
			for docID, length := range data.docLens {
				docLens[docID] = length
			}
			// Clear after merge
			data.docLens = nil
		}
	}

	// Clear worker 0's docLens (terms kept as finalTerms)
	ti.workerData[0].docLens = nil

	return finalTerms, docLens
}

// DocCount returns the number of indexed documents.
func (ti *TurboIndexer) DocCount() int64 {
	return ti.docCount.Load()
}

// BatchIndexer collects all documents first, then processes in parallel batches.
// This approach minimizes merge overhead by using a two-phase algorithm:
// Phase 1: Collect all documents (streaming)
// Phase 2: Parallel batch processing with direct merge to final structure
type BatchIndexer struct {
	Tokenizer TokenizerFunc
	docs      []indexItem
	mu        sync.Mutex
}

// NewBatchIndexer creates a new batch indexer.
func NewBatchIndexer(tokenizer TokenizerFunc) *BatchIndexer {
	return &BatchIndexer{
		Tokenizer: tokenizer,
		docs:      make([]indexItem, 0, 50000),
	}
}

// Add adds a document to be indexed.
func (bi *BatchIndexer) Add(docID uint32, text string) {
	bi.mu.Lock()
	bi.docs = append(bi.docs, indexItem{docID: docID, text: text})
	bi.mu.Unlock()
}

// Finish processes all documents in parallel and returns merged results.
func (bi *BatchIndexer) Finish() (map[string][]IndexPosting, []int) {
	if len(bi.docs) == 0 {
		return make(map[string][]IndexPosting), nil
	}

	// Use up to 16 CPUs for parallelism
	numWorkers := runtime.NumCPU()
	if numWorkers > 16 {
		numWorkers = 16
	}
	if numWorkers < 2 {
		numWorkers = 2
	}

	// Divide documents evenly across workers
	docsPerWorker := (len(bi.docs) + numWorkers - 1) / numWorkers

	type workerOutput struct {
		terms   map[string][]IndexPosting
		docLens map[uint32]int
	}

	results := make([]workerOutput, numWorkers)
	var wg sync.WaitGroup

	t0 := time.Now()

	// Parallel tokenization and local posting list building
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			start := workerID * docsPerWorker
			end := start + docsPerWorker
			if end > len(bi.docs) {
				end = len(bi.docs)
			}
			if start >= len(bi.docs) {
				return
			}

			batch := bi.docs[start:end]
			localTerms := make(map[string][]IndexPosting, 30000)
			localDocLens := make(map[uint32]int, len(batch))

			for _, item := range batch {
				termFreqs := bi.Tokenizer(item.text)

				// Calculate doc length
				docLen := 0
				for _, freq := range termFreqs {
					docLen += freq
				}
				localDocLens[item.docID] = docLen

				// Add to local posting lists
				for term, freq := range termFreqs {
					localTerms[term] = append(localTerms[term], IndexPosting{
						DocID: item.docID,
						Freq:  uint16(freq),
					})
				}
			}

			results[workerID] = workerOutput{
				terms:   localTerms,
				docLens: localDocLens,
			}
		}(i)
	}

	wg.Wait()
	t1 := time.Now()

	// Phase 1: Count total postings per term (parallel)
	termCounts := make(map[string]int, 100000)
	for i := 0; i < numWorkers; i++ {
		if results[i].terms == nil {
			continue
		}
		for term, postings := range results[i].terms {
			termCounts[term] += len(postings)
		}
	}
	t2 := time.Now()

	// Phase 2: Pre-allocate final slices with exact capacity
	finalTerms := make(map[string][]IndexPosting, len(termCounts))
	for term, count := range termCounts {
		finalTerms[term] = make([]IndexPosting, 0, count)
	}
	t3 := time.Now()

	// Phase 3: Copy postings (no reallocation needed)
	for i := 0; i < numWorkers; i++ {
		if results[i].terms == nil {
			continue
		}
		for term, postings := range results[i].terms {
			finalTerms[term] = append(finalTerms[term], postings...)
		}
		results[i].terms = nil // Help GC
	}
	t4 := time.Now()

	// Merge doc lengths
	docLens := make([]int, len(bi.docs))
	for i := 0; i < numWorkers; i++ {
		if results[i].docLens == nil {
			continue
		}
		for docID, length := range results[i].docLens {
			docLens[docID] = length
		}
	}
	t5 := time.Now()

	// Debug timing removed for production
	_ = t0
	_ = t1
	_ = t2
	_ = t3
	_ = t4
	_ = t5

	return finalTerms, docLens
}

// DocCount returns the number of indexed documents.
func (bi *BatchIndexer) DocCount() int64 {
	bi.mu.Lock()
	defer bi.mu.Unlock()
	return int64(len(bi.docs))
}
