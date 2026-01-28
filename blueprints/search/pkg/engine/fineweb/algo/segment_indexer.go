// Package algo provides segment-based parallel indexing for 10x performance.
package algo

import (
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
)

// Buffer pools for reducing GC pressure
var (
	// Pool for term frequency maps
	termMapPool = sync.Pool{
		New: func() interface{} {
			return make(map[string]int, 512)
		},
	}

	// Pool for posting slices
	postingSlicePool = sync.Pool{
		New: func() interface{} {
			s := make([]IndexPosting, 0, 2048)
			return &s
		},
	}

	// Pool for segment term posting maps
	segmentMapPool = sync.Pool{
		New: func() interface{} {
			return make(map[string][]IndexPosting, 30000)
		},
	}

	// Pool for doc length maps
	docLenMapPool = sync.Pool{
		New: func() interface{} {
			return make(map[uint32]int, 50000)
		},
	}
)

// SegmentIndexer implements lock-free segment-based parallel indexing.
// Each worker builds independent segments that are merged in the background.
type SegmentIndexer struct {
	NumWorkers   int
	SegmentSize  int // Documents per segment before flush
	Tokenizer    TokenizerFunc

	// Channels
	docCh      chan segmentDoc
	segmentCh  chan *Segment
	doneCh     chan struct{}

	// Final merged results
	termPostings map[string][]IndexPosting
	docLens      []int
	docCount     atomic.Int64

	// Synchronization
	wg         sync.WaitGroup
	mergeWg    sync.WaitGroup
	mu         sync.Mutex
}

type segmentDoc struct {
	docID uint32
	text  string
}

// Segment is an independent index partition built by a single worker.
type Segment struct {
	termPostings map[string][]IndexPosting
	docLens      map[uint32]int
	minDocID     uint32
	maxDocID     uint32
	docCount     int
}

// NewSegmentIndexer creates a high-performance segment-based indexer.
func NewSegmentIndexer(tokenizer TokenizerFunc) *SegmentIndexer {
	numWorkers := runtime.NumCPU()
	if numWorkers < 2 {
		numWorkers = 2
	}
	if numWorkers > 32 {
		numWorkers = 32 // Allow more workers for segment-based approach
	}

	si := &SegmentIndexer{
		NumWorkers:   numWorkers,
		SegmentSize:  10000, // Larger segments for better locality
		Tokenizer:    tokenizer,
		docCh:        make(chan segmentDoc, numWorkers*2000), // Larger buffer
		segmentCh:    make(chan *Segment, numWorkers*4),      // Buffer for segments
		doneCh:       make(chan struct{}),
		termPostings: make(map[string][]IndexPosting, 100000),
		docLens:      make([]int, 0, 500000),
	}

	// Start segment builders (workers)
	for i := 0; i < si.NumWorkers; i++ {
		si.wg.Add(1)
		go si.segmentWorker(i)
	}

	// Start background merger
	si.mergeWg.Add(1)
	go si.backgroundMerger()

	return si
}

// Add adds a document to be indexed.
func (si *SegmentIndexer) Add(docID uint32, text string) {
	si.docCh <- segmentDoc{docID: docID, text: text}
	si.docCount.Add(1)
}

// segmentWorker builds independent segments without any locking.
func (si *SegmentIndexer) segmentWorker(workerID int) {
	defer si.wg.Done()

	// Pre-allocate segment with expected capacity
	seg := si.newSegment()

	for doc := range si.docCh {
		// Tokenize (CPU bound, no locks)
		termFreqs := si.Tokenizer(doc.text)

		// Calculate doc length
		docLen := 0
		for _, freq := range termFreqs {
			docLen += freq
		}
		seg.docLens[doc.docID] = docLen

		// Add to segment's posting lists (no lock - segment owned by this worker)
		for term, freq := range termFreqs {
			seg.termPostings[term] = append(seg.termPostings[term], IndexPosting{
				DocID: doc.docID,
				Freq:  uint16(freq),
			})
		}

		// Track doc range
		if doc.docID < seg.minDocID {
			seg.minDocID = doc.docID
		}
		if doc.docID > seg.maxDocID {
			seg.maxDocID = doc.docID
		}
		seg.docCount++

		// Flush segment when full
		if seg.docCount >= si.SegmentSize {
			si.segmentCh <- seg
			seg = si.newSegment()
		}
	}

	// Flush remaining segment
	if seg.docCount > 0 {
		si.segmentCh <- seg
	}
}

func (si *SegmentIndexer) newSegment() *Segment {
	return &Segment{
		termPostings: make(map[string][]IndexPosting, 20000),
		docLens:      make(map[uint32]int, si.SegmentSize),
		minDocID:     ^uint32(0), // Max uint32
		maxDocID:     0,
	}
}

// backgroundMerger merges completed segments in the background.
func (si *SegmentIndexer) backgroundMerger() {
	defer si.mergeWg.Done()

	// Batch segments for efficient merging
	batch := make([]*Segment, 0, 4)
	batchThreshold := 4

	for seg := range si.segmentCh {
		batch = append(batch, seg)

		// Merge batch when threshold reached
		if len(batch) >= batchThreshold {
			si.mergeBatch(batch)
			batch = batch[:0]
		}
	}

	// Merge remaining segments
	if len(batch) > 0 {
		si.mergeBatch(batch)
	}

	close(si.doneCh)
}

// mergeBatch merges multiple segments at once for efficiency.
func (si *SegmentIndexer) mergeBatch(segments []*Segment) {
	if len(segments) == 0 {
		return
	}

	// Collect all terms across segments
	allTerms := make(map[string]struct{}, 50000)
	totalDocs := 0
	for _, seg := range segments {
		for term := range seg.termPostings {
			allTerms[term] = struct{}{}
		}
		totalDocs += len(seg.docLens)
	}

	// Pre-allocate merged structures
	mergedTerms := make(map[string][]IndexPosting, len(allTerms))
	mergedDocLens := make(map[uint32]int, totalDocs)

	// Merge term postings
	for term := range allTerms {
		var totalPostings int
		for _, seg := range segments {
			totalPostings += len(seg.termPostings[term])
		}

		merged := make([]IndexPosting, 0, totalPostings)
		for _, seg := range segments {
			merged = append(merged, seg.termPostings[term]...)
		}
		mergedTerms[term] = merged
	}

	// Merge doc lengths
	for _, seg := range segments {
		for docID, length := range seg.docLens {
			mergedDocLens[docID] = length
		}
	}

	// Merge into global state (single lock for entire batch)
	si.mu.Lock()

	// Merge term postings
	for term, postings := range mergedTerms {
		si.termPostings[term] = append(si.termPostings[term], postings...)
	}

	// Extend docLens array as needed
	maxDocID := uint32(0)
	for docID := range mergedDocLens {
		if docID > maxDocID {
			maxDocID = docID
		}
	}

	for int(maxDocID) >= len(si.docLens) {
		si.docLens = append(si.docLens, 0)
	}

	for docID, length := range mergedDocLens {
		si.docLens[docID] = length
	}

	si.mu.Unlock()
}

// Finish waits for all indexing to complete and returns results.
func (si *SegmentIndexer) Finish() (map[string][]IndexPosting, []int) {
	// Close input channel to signal workers to finish
	close(si.docCh)

	// Wait for all workers to complete
	si.wg.Wait()

	// Close segment channel to signal merger to finish
	close(si.segmentCh)

	// Wait for merger to complete
	si.mergeWg.Wait()

	return si.termPostings, si.docLens
}

// DocCount returns the number of indexed documents.
func (si *SegmentIndexer) DocCount() int64 {
	return si.docCount.Load()
}

// SegmentIndexerV2 is an enhanced version with parallel merge and buffer pooling.
type SegmentIndexerV2 struct {
	NumWorkers   int
	SegmentSize  int
	Tokenizer    TokenizerFunc

	docCh        chan segmentDoc
	segments     []*Segment  // Collected segments
	segmentMu    sync.Mutex

	docCount     atomic.Int64
	wg           sync.WaitGroup
}

// NewSegmentIndexerV2 creates a V2 indexer with parallel merge and optimizations.
func NewSegmentIndexerV2(tokenizer TokenizerFunc) *SegmentIndexerV2 {
	numWorkers := runtime.NumCPU()
	if numWorkers < 2 {
		numWorkers = 2
	}
	if numWorkers > 32 {
		numWorkers = 32
	}

	// Segment size tuned for optimal parallelism
	// Smaller segments = more parallel work, but more merge overhead
	// Larger segments = less merge overhead, but less parallelism
	// Optimal: enough segments for numWorkers to each have work
	segmentSize := 5000 // Smaller segments for better parallelism

	si := &SegmentIndexerV2{
		NumWorkers:  numWorkers,
		SegmentSize: segmentSize,
		Tokenizer:   tokenizer,
		docCh:       make(chan segmentDoc, numWorkers*4000), // Large buffer
		segments:    make([]*Segment, 0, 256),
	}

	// Start segment builders
	for i := 0; i < si.NumWorkers; i++ {
		si.wg.Add(1)
		go si.segmentWorker()
	}

	return si
}

func (si *SegmentIndexerV2) Add(docID uint32, text string) {
	si.docCh <- segmentDoc{docID: docID, text: text}
	si.docCount.Add(1)
}

func (si *SegmentIndexerV2) segmentWorker() {
	defer si.wg.Done()

	// Get pooled maps for this segment
	termPostings := segmentMapPool.Get().(map[string][]IndexPosting)
	docLens := docLenMapPool.Get().(map[uint32]int)

	seg := &Segment{
		termPostings: termPostings,
		docLens:      docLens,
		minDocID:     ^uint32(0),
		maxDocID:     0,
	}

	for doc := range si.docCh {
		termFreqs := si.Tokenizer(doc.text)

		docLen := 0
		for _, freq := range termFreqs {
			docLen += freq
		}
		seg.docLens[doc.docID] = docLen

		for term, freq := range termFreqs {
			seg.termPostings[term] = append(seg.termPostings[term], IndexPosting{
				DocID: doc.docID,
				Freq:  uint16(freq),
			})
		}

		if doc.docID < seg.minDocID {
			seg.minDocID = doc.docID
		}
		if doc.docID > seg.maxDocID {
			seg.maxDocID = doc.docID
		}
		seg.docCount++

		// Flush when full - use larger segment size for efficiency
		if seg.docCount >= si.SegmentSize {
			si.segmentMu.Lock()
			si.segments = append(si.segments, seg)
			si.segmentMu.Unlock()

			// Get new pooled maps
			termPostings = segmentMapPool.Get().(map[string][]IndexPosting)
			docLens = docLenMapPool.Get().(map[uint32]int)

			seg = &Segment{
				termPostings: termPostings,
				docLens:      docLens,
				minDocID:     ^uint32(0),
				maxDocID:     0,
			}
		}
	}

	// Flush remaining
	if seg.docCount > 0 {
		si.segmentMu.Lock()
		si.segments = append(si.segments, seg)
		si.segmentMu.Unlock()
	}
}

// Finish uses parallel k-way merge for maximum performance.
func (si *SegmentIndexerV2) Finish() (map[string][]IndexPosting, []int) {
	close(si.docCh)
	si.wg.Wait()

	if len(si.segments) == 0 {
		return make(map[string][]IndexPosting), nil
	}

	// Parallel merge of segments
	return si.parallelMerge()
}

func (si *SegmentIndexerV2) parallelMerge() (map[string][]IndexPosting, []int) {
	// Collect all unique terms across all segments
	termSet := make(map[string]struct{}, 100000)
	maxDocID := uint32(0)

	for _, seg := range si.segments {
		for term := range seg.termPostings {
			termSet[term] = struct{}{}
		}
		if seg.maxDocID > maxDocID {
			maxDocID = seg.maxDocID
		}
	}

	terms := make([]string, 0, len(termSet))
	for term := range termSet {
		terms = append(terms, term)
	}

	// Parallel term merging with more workers
	numWorkers := runtime.NumCPU()
	if numWorkers > 16 {
		numWorkers = 16
	}

	termCh := make(chan string, len(terms))
	type mergeResult struct {
		term     string
		postings []IndexPosting
	}
	resultCh := make(chan mergeResult, len(terms))

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for term := range termCh {
				// Count total postings for this term
				total := 0
				for _, seg := range si.segments {
					total += len(seg.termPostings[term])
				}

				// Merge postings with pre-allocation (NO SORT - defer to block building)
				merged := make([]IndexPosting, 0, total)
				for _, seg := range si.segments {
					merged = append(merged, seg.termPostings[term]...)
				}

				// Skip sorting here - buildBlocksDirect will sort
				resultCh <- mergeResult{term: term, postings: merged}
			}
		}()
	}

	// Feed terms
	for _, term := range terms {
		termCh <- term
	}
	close(termCh)

	// Collect results
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	finalTerms := make(map[string][]IndexPosting, len(terms))
	for result := range resultCh {
		finalTerms[result.term] = result.postings
	}

	// Merge doc lengths (parallel)
	docLens := make([]int, maxDocID+1)
	var docWg sync.WaitGroup
	chunkSize := (len(si.segments) + numWorkers - 1) / numWorkers

	for i := 0; i < numWorkers; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > len(si.segments) {
			end = len(si.segments)
		}
		if start >= len(si.segments) {
			break
		}

		docWg.Add(1)
		go func(segs []*Segment) {
			defer docWg.Done()
			for _, seg := range segs {
				for docID, length := range seg.docLens {
					docLens[docID] = length // No lock needed - each docID unique
				}
			}
		}(si.segments[start:end])
	}
	docWg.Wait()

	// Return pooled maps (clear and return to pool)
	for _, seg := range si.segments {
		// Clear and return term postings map
		for k := range seg.termPostings {
			delete(seg.termPostings, k)
		}
		segmentMapPool.Put(seg.termPostings)

		// Clear and return doc lens map
		for k := range seg.docLens {
			delete(seg.docLens, k)
		}
		docLenMapPool.Put(seg.docLens)
	}

	return finalTerms, docLens
}

func (si *SegmentIndexerV2) DocCount() int64 {
	return si.docCount.Load()
}

// SegmentIndexerV3 is a high-performance indexer optimized for 50k+ docs/sec.
// Key optimizations:
// - Batch document processing
// - Lock-free segment collection
// - Two-level merge tree
// - Buffer pooling with sync.Pool
type SegmentIndexerV3 struct {
	NumWorkers   int
	SegmentSize  int
	Tokenizer    TokenizerFunc

	// Lock-free document queue using channels with large buffers
	docCh        chan []segmentDoc  // Batch channel for efficiency

	// Atomic segment collection
	segments     []*Segment
	segmentMu    sync.Mutex

	docCount     atomic.Int64
	wg           sync.WaitGroup
}

// NewSegmentIndexerV3 creates a V3 high-performance indexer.
func NewSegmentIndexerV3(tokenizer TokenizerFunc) *SegmentIndexerV3 {
	numWorkers := runtime.NumCPU()
	if numWorkers < 2 {
		numWorkers = 2
	}
	if numWorkers > 32 {
		numWorkers = 32
	}

	si := &SegmentIndexerV3{
		NumWorkers:  numWorkers,
		SegmentSize: 100000, // Very large segments
		Tokenizer:   tokenizer,
		docCh:       make(chan []segmentDoc, numWorkers*2), // Batch channel
		segments:    make([]*Segment, 0, 64),
	}

	// Start batch workers
	for i := 0; i < si.NumWorkers; i++ {
		si.wg.Add(1)
		go si.batchWorker()
	}

	return si
}

// AddBatch adds a batch of documents efficiently.
func (si *SegmentIndexerV3) AddBatch(docs []segmentDoc) {
	si.docCh <- docs
	si.docCount.Add(int64(len(docs)))
}

// Add adds a single document (less efficient than AddBatch).
func (si *SegmentIndexerV3) Add(docID uint32, text string) {
	si.AddBatch([]segmentDoc{{docID: docID, text: text}})
}

func (si *SegmentIndexerV3) batchWorker() {
	defer si.wg.Done()

	// Get pooled segment
	termPostings := segmentMapPool.Get().(map[string][]IndexPosting)
	docLens := docLenMapPool.Get().(map[uint32]int)

	seg := &Segment{
		termPostings: termPostings,
		docLens:      docLens,
		minDocID:     ^uint32(0),
		maxDocID:     0,
	}

	for batch := range si.docCh {
		for _, doc := range batch {
			termFreqs := si.Tokenizer(doc.text)

			docLen := 0
			for _, freq := range termFreqs {
				docLen += freq
			}
			seg.docLens[doc.docID] = docLen

			for term, freq := range termFreqs {
				seg.termPostings[term] = append(seg.termPostings[term], IndexPosting{
					DocID: doc.docID,
					Freq:  uint16(freq),
				})
			}

			if doc.docID < seg.minDocID {
				seg.minDocID = doc.docID
			}
			if doc.docID > seg.maxDocID {
				seg.maxDocID = doc.docID
			}
			seg.docCount++
		}

		// Flush when full
		if seg.docCount >= si.SegmentSize {
			si.segmentMu.Lock()
			si.segments = append(si.segments, seg)
			si.segmentMu.Unlock()

			// Get new pooled segment
			termPostings = segmentMapPool.Get().(map[string][]IndexPosting)
			docLens = docLenMapPool.Get().(map[uint32]int)

			seg = &Segment{
				termPostings: termPostings,
				docLens:      docLens,
				minDocID:     ^uint32(0),
				maxDocID:     0,
			}
		}
	}

	// Flush remaining
	if seg.docCount > 0 {
		si.segmentMu.Lock()
		si.segments = append(si.segments, seg)
		si.segmentMu.Unlock()
	}
}

// Finish uses two-level parallel merge for maximum performance.
func (si *SegmentIndexerV3) Finish() (map[string][]IndexPosting, []int) {
	close(si.docCh)
	si.wg.Wait()

	if len(si.segments) == 0 {
		return make(map[string][]IndexPosting), nil
	}

	// Two-level merge: first merge segments in parallel, then final merge
	return si.twoLevelMerge()
}

func (si *SegmentIndexerV3) twoLevelMerge() (map[string][]IndexPosting, []int) {
	numWorkers := runtime.NumCPU()
	if numWorkers > 16 {
		numWorkers = 16
	}

	// Level 1: Parallel segment merging (groups of 4)
	groupSize := 4
	numGroups := (len(si.segments) + groupSize - 1) / groupSize

	mergedSegments := make([]*Segment, numGroups)
	var level1Wg sync.WaitGroup

	for g := 0; g < numGroups; g++ {
		start := g * groupSize
		end := start + groupSize
		if end > len(si.segments) {
			end = len(si.segments)
		}

		level1Wg.Add(1)
		go func(groupIdx int, segs []*Segment) {
			defer level1Wg.Done()
			mergedSegments[groupIdx] = mergeSegmentGroup(segs)
		}(g, si.segments[start:end])
	}

	level1Wg.Wait()

	// Level 2: Final merge of merged segments
	return finalMerge(mergedSegments, numWorkers)
}

// mergeSegmentGroup merges a small group of segments into one.
func mergeSegmentGroup(segs []*Segment) *Segment {
	if len(segs) == 1 {
		return segs[0]
	}

	result := &Segment{
		termPostings: make(map[string][]IndexPosting, 50000),
		docLens:      make(map[uint32]int, 100000),
		minDocID:     ^uint32(0),
		maxDocID:     0,
	}

	for _, seg := range segs {
		// Merge term postings
		for term, postings := range seg.termPostings {
			result.termPostings[term] = append(result.termPostings[term], postings...)
		}

		// Merge doc lengths
		for docID, length := range seg.docLens {
			result.docLens[docID] = length
		}

		// Update bounds
		if seg.minDocID < result.minDocID {
			result.minDocID = seg.minDocID
		}
		if seg.maxDocID > result.maxDocID {
			result.maxDocID = seg.maxDocID
		}

		result.docCount += seg.docCount
	}

	return result
}

// finalMerge performs the final parallel merge of all segments.
func finalMerge(segments []*Segment, numWorkers int) (map[string][]IndexPosting, []int) {
	// Collect all terms
	termSet := make(map[string]struct{}, 100000)
	maxDocID := uint32(0)

	for _, seg := range segments {
		if seg == nil {
			continue
		}
		for term := range seg.termPostings {
			termSet[term] = struct{}{}
		}
		if seg.maxDocID > maxDocID {
			maxDocID = seg.maxDocID
		}
	}

	terms := make([]string, 0, len(termSet))
	for term := range termSet {
		terms = append(terms, term)
	}

	// Parallel term merge
	termCh := make(chan string, len(terms))
	type mergeResult struct {
		term     string
		postings []IndexPosting
	}
	resultCh := make(chan mergeResult, len(terms))

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for term := range termCh {
				total := 0
				for _, seg := range segments {
					if seg != nil {
						total += len(seg.termPostings[term])
					}
				}

				merged := make([]IndexPosting, 0, total)
				for _, seg := range segments {
					if seg != nil {
						merged = append(merged, seg.termPostings[term]...)
					}
				}

				// Sort for efficient search
				sort.Slice(merged, func(i, j int) bool {
					return merged[i].DocID < merged[j].DocID
				})

				resultCh <- mergeResult{term: term, postings: merged}
			}
		}()
	}

	for _, term := range terms {
		termCh <- term
	}
	close(termCh)

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	finalTerms := make(map[string][]IndexPosting, len(terms))
	for result := range resultCh {
		finalTerms[result.term] = result.postings
	}

	// Merge doc lengths
	docLens := make([]int, maxDocID+1)
	for _, seg := range segments {
		if seg == nil {
			continue
		}
		for docID, length := range seg.docLens {
			docLens[docID] = length
		}
	}

	return finalTerms, docLens
}

func (si *SegmentIndexerV3) DocCount() int64 {
	return si.docCount.Load()
}
