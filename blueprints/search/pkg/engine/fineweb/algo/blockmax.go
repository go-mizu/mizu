package algo

import (
	"container/heap"
	"math"
	"sort"
)

// BlockSize is the number of documents per block for Block-Max WAND.
const BlockSize = 128

// Block represents a block of postings with pre-computed max score.
type Block struct {
	DocIDs    []uint32  // Document IDs in this block
	Freqs     []uint16  // Term frequencies
	MaxScore  float32   // Maximum BM25 score in this block
	MinDocID  uint32    // First doc ID in block
	MaxDocID  uint32    // Last doc ID in block
}

// BlockPostingList is a posting list divided into blocks with max scores.
type BlockPostingList struct {
	Term      string
	Blocks    []Block
	MaxScore  float32 // Global max score for this term
	DocFreq   int     // Total document frequency
	IDF       float32 // Inverse document frequency
}

// BlockMaxIndex is an inverted index optimized for Block-Max WAND.
type BlockMaxIndex struct {
	Terms     map[string]*BlockPostingList
	Documents map[string]Document // docID -> document
	DocLens   map[string]int      // docID -> document length
	NumDocs   int
	AvgDocLen float64
	K1        float32 // BM25 k1 parameter (default 1.2)
	B         float32 // BM25 b parameter (default 0.75)
}

// Document represents a stored document.
type Document struct {
	ID            string
	URL           string
	Text          string
	Dump          string
	Date          string
	Language      string
	LanguageScore float64
}

// SearchResult holds a document with its score.
type SearchResult struct {
	DocID string
	Score float32
	Doc   *Document
}

// NewBlockMaxIndex creates a new Block-Max index.
func NewBlockMaxIndex() *BlockMaxIndex {
	return &BlockMaxIndex{
		Terms:     make(map[string]*BlockPostingList),
		Documents: make(map[string]Document),
		DocLens:   make(map[string]int),
		K1:        1.2,
		B:         0.75,
	}
}

// AddDocument adds a document to the index.
func (idx *BlockMaxIndex) AddDocument(doc Document, tokens []string) {
	idx.Documents[doc.ID] = doc
	idx.DocLens[doc.ID] = len(tokens)
	idx.NumDocs++

	// Update average document length
	totalLen := idx.AvgDocLen * float64(idx.NumDocs-1)
	idx.AvgDocLen = (totalLen + float64(len(tokens))) / float64(idx.NumDocs)

	// Count term frequencies
	termFreqs := make(map[string]int)
	for _, token := range tokens {
		termFreqs[token]++
	}

	// Add to posting lists
	for term, freq := range termFreqs {
		pl, exists := idx.Terms[term]
		if !exists {
			pl = &BlockPostingList{Term: term}
			idx.Terms[term] = pl
		}

		// Add to current block or create new block
		if len(pl.Blocks) == 0 || len(pl.Blocks[len(pl.Blocks)-1].DocIDs) >= BlockSize {
			pl.Blocks = append(pl.Blocks, Block{
				DocIDs: make([]uint32, 0, BlockSize),
				Freqs:  make([]uint16, 0, BlockSize),
			})
		}

		block := &pl.Blocks[len(pl.Blocks)-1]
		docIDNum := uint32(idx.NumDocs - 1) // Use sequential numbering

		if len(block.DocIDs) == 0 {
			block.MinDocID = docIDNum
		}
		block.DocIDs = append(block.DocIDs, docIDNum)
		block.Freqs = append(block.Freqs, uint16(freq))
		block.MaxDocID = docIDNum

		pl.DocFreq++
	}
}

// Finalize computes IDF and max scores for all terms and blocks.
func (idx *BlockMaxIndex) Finalize() {
	for _, pl := range idx.Terms {
		// Compute IDF: log((N - df + 0.5) / (df + 0.5) + 1)
		df := float64(pl.DocFreq)
		n := float64(idx.NumDocs)
		pl.IDF = float32(math.Log((n-df+0.5)/(df+0.5) + 1))

		// Compute max score per block and global max
		pl.MaxScore = 0
		for i := range pl.Blocks {
			block := &pl.Blocks[i]
			block.MaxScore = 0

			for j, freq := range block.Freqs {
				docID := block.DocIDs[j]
				docLen := idx.getDocLenByNum(docID)

				// BM25 score for this posting
				tf := float32(freq)
				score := idx.computeBM25Score(tf, pl.IDF, docLen)

				if score > block.MaxScore {
					block.MaxScore = score
				}
			}

			if block.MaxScore > pl.MaxScore {
				pl.MaxScore = block.MaxScore
			}
		}
	}
}

func (idx *BlockMaxIndex) getDocLenByNum(docNum uint32) int {
	// For sequential numbering, we need to maintain a mapping
	// In production, use a more efficient structure
	i := 0
	for docID := range idx.Documents {
		if uint32(i) == docNum {
			return idx.DocLens[docID]
		}
		i++
	}
	return int(idx.AvgDocLen)
}

func (idx *BlockMaxIndex) computeBM25Score(tf, idf float32, docLen int) float32 {
	// BM25: IDF * (tf * (k1 + 1)) / (tf + k1 * (1 - b + b * dl/avgdl))
	k1 := idx.K1
	b := idx.B
	avgDL := float32(idx.AvgDocLen)
	dl := float32(docLen)

	tfNorm := (tf * (k1 + 1)) / (tf + k1*(1-b+b*dl/avgDL))
	return idf * tfNorm
}

// BlockIterator iterates over a block posting list.
type BlockIterator struct {
	pl         *BlockPostingList
	blockIdx   int
	posInBlock int
	exhausted  bool
}

// NewBlockIterator creates an iterator for a posting list.
func NewBlockIterator(pl *BlockPostingList) *BlockIterator {
	return &BlockIterator{pl: pl}
}

// DocID returns the current document ID.
func (it *BlockIterator) DocID() uint32 {
	if it.exhausted || it.blockIdx >= len(it.pl.Blocks) {
		return math.MaxUint32
	}
	block := &it.pl.Blocks[it.blockIdx]
	if it.posInBlock >= len(block.DocIDs) {
		return math.MaxUint32
	}
	return block.DocIDs[it.posInBlock]
}

// Freq returns the current term frequency.
func (it *BlockIterator) Freq() uint16 {
	if it.exhausted || it.blockIdx >= len(it.pl.Blocks) {
		return 0
	}
	block := &it.pl.Blocks[it.blockIdx]
	if it.posInBlock >= len(block.DocIDs) {
		return 0
	}
	return block.Freqs[it.posInBlock]
}

// BlockMaxScore returns the max score of the current block.
func (it *BlockIterator) BlockMaxScore() float32 {
	if it.exhausted || it.blockIdx >= len(it.pl.Blocks) {
		return 0
	}
	return it.pl.Blocks[it.blockIdx].MaxScore
}

// Next advances to the next posting.
func (it *BlockIterator) Next() {
	if it.exhausted {
		return
	}

	it.posInBlock++
	if it.posInBlock >= len(it.pl.Blocks[it.blockIdx].DocIDs) {
		it.blockIdx++
		it.posInBlock = 0
		if it.blockIdx >= len(it.pl.Blocks) {
			it.exhausted = true
		}
	}
}

// SkipToBlock advances to the first block containing docID >= target.
func (it *BlockIterator) SkipToBlock(target uint32) {
	if it.exhausted {
		return
	}

	// Binary search for block containing target
	for it.blockIdx < len(it.pl.Blocks) {
		block := &it.pl.Blocks[it.blockIdx]
		if block.MaxDocID >= target {
			// This block might contain target
			// Linear search within block (could use binary search for large blocks)
			for it.posInBlock < len(block.DocIDs) {
				if block.DocIDs[it.posInBlock] >= target {
					return
				}
				it.posInBlock++
			}
		}
		it.blockIdx++
		it.posInBlock = 0
	}

	it.exhausted = true
}

// Exhausted returns true if the iterator has no more postings.
func (it *BlockIterator) Exhausted() bool {
	return it.exhausted
}

// Search performs Block-Max WAND search.
func (idx *BlockMaxIndex) Search(queryTerms []string, k int) []SearchResult {
	// Get posting lists for query terms
	iters := make([]*BlockIterator, 0, len(queryTerms))
	upperBounds := make([]float32, 0, len(queryTerms))

	for _, term := range queryTerms {
		if pl, exists := idx.Terms[term]; exists {
			iters = append(iters, NewBlockIterator(pl))
			upperBounds = append(upperBounds, pl.MaxScore)
		}
	}

	if len(iters) == 0 {
		return nil
	}

	// Result heap (min-heap to maintain top-k)
	results := &resultHeap{}
	heap.Init(results)
	threshold := float32(0)

	// Main Block-Max WAND loop
	for {
		// Sort iterators by current doc ID
		sort.Slice(iters, func(i, j int) bool {
			return iters[i].DocID() < iters[j].DocID()
		})

		// Remove exhausted iterators
		activeIters := iters[:0]
		activeUB := upperBounds[:0]
		for i, it := range iters {
			if !it.Exhausted() {
				activeIters = append(activeIters, it)
				activeUB = append(activeUB, upperBounds[i])
			}
		}
		iters = activeIters
		upperBounds = activeUB

		if len(iters) == 0 {
			break
		}

		// Find pivot: smallest i where sum of upper bounds >= threshold
		pivotIdx := -1
		cumSum := float32(0)
		for i, ub := range upperBounds {
			cumSum += ub
			if cumSum >= threshold {
				pivotIdx = i
				break
			}
		}

		if pivotIdx < 0 {
			// No document can exceed threshold
			break
		}

		pivotDoc := iters[pivotIdx].DocID()

		// Check if all iterators up to pivot point to same document
		allSame := true
		for i := 0; i < pivotIdx; i++ {
			if iters[i].DocID() != pivotDoc {
				allSame = false
				break
			}
		}

		if allSame && iters[0].DocID() == pivotDoc {
			// Score the document
			score := float32(0)
			for i := 0; i <= pivotIdx; i++ {
				if iters[i].DocID() == pivotDoc {
					tf := float32(iters[i].Freq())
					idf := iters[i].pl.IDF
					docLen := idx.getDocLenByNum(pivotDoc)
					score += idx.computeBM25Score(tf, idf, docLen)
				}
			}

			if score > threshold {
				docIDStr := idx.getDocIDByNum(pivotDoc)
				heap.Push(results, SearchResult{DocID: docIDStr, Score: score})
				if results.Len() > k {
					heap.Pop(results)
				}
				if results.Len() >= k {
					threshold = (*results)[0].Score
				}
			}

			// Advance all iterators at pivot doc
			for i := 0; i <= pivotIdx; i++ {
				if iters[i].DocID() == pivotDoc {
					iters[i].Next()
				}
			}
		} else {
			// Block-level skip: advance first iterator past pivot doc using block skipping
			// Check if we can skip entire blocks
			firstIter := iters[0]
			blockMax := firstIter.BlockMaxScore()

			// If current block's max score can't help, skip to next block
			if blockMax < threshold-cumSum+upperBounds[0] {
				// Skip entire block
				nextBlock := firstIter.blockIdx + 1
				if nextBlock < len(firstIter.pl.Blocks) {
					firstIter.blockIdx = nextBlock
					firstIter.posInBlock = 0
				} else {
					firstIter.exhausted = true
				}
			} else {
				// Need to check within block
				firstIter.SkipToBlock(pivotDoc)
			}
		}
	}

	// Convert to sorted results (highest score first)
	finalResults := make([]SearchResult, results.Len())
	for i := results.Len() - 1; i >= 0; i-- {
		finalResults[i] = heap.Pop(results).(SearchResult)
	}

	// Attach documents
	for i := range finalResults {
		if doc, exists := idx.Documents[finalResults[i].DocID]; exists {
			finalResults[i].Doc = &doc
		}
	}

	return finalResults
}

func (idx *BlockMaxIndex) getDocIDByNum(docNum uint32) string {
	i := uint32(0)
	for docID := range idx.Documents {
		if i == docNum {
			return docID
		}
		i++
	}
	return ""
}

// resultHeap is a min-heap of search results for top-k selection.
type resultHeap []SearchResult

func (h resultHeap) Len() int           { return len(h) }
func (h resultHeap) Less(i, j int) bool { return h[i].Score < h[j].Score }
func (h resultHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *resultHeap) Push(x any) {
	*h = append(*h, x.(SearchResult))
}

func (h *resultHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}
