// Package algo provides a fast lock-free indexer optimized for throughput.
package algo

import (
	"runtime"
	"sync"
	"sync/atomic"
)

// FastIndexer provides maximum throughput by deferring all merging to the end.
// Workers accumulate results independently with no locking during indexing.
type FastIndexer struct {
	NumWorkers int
	Tokenizer  TokenizerFunc

	// Channel for document distribution
	docCh chan fastDoc

	// Each worker accumulates its own results (no sharing during indexing)
	workerResults []*workerData

	docCount atomic.Int64
	wg       sync.WaitGroup
}

type fastDoc struct {
	docID uint32
	text  string
}

type workerData struct {
	termPostings map[string][]IndexPosting
	docLens      map[uint32]int
}

// NewFastIndexer creates a new fast indexer.
func NewFastIndexer(tokenizer TokenizerFunc) *FastIndexer {
	numWorkers := runtime.NumCPU()
	if numWorkers < 2 {
		numWorkers = 2
	}
	if numWorkers > 32 {
		numWorkers = 32
	}

	fi := &FastIndexer{
		NumWorkers:    numWorkers,
		Tokenizer:     tokenizer,
		docCh:         make(chan fastDoc, numWorkers*5000), // Large buffer
		workerResults: make([]*workerData, numWorkers),
	}

	// Initialize worker data structures
	for i := 0; i < numWorkers; i++ {
		fi.workerResults[i] = &workerData{
			termPostings: make(map[string][]IndexPosting, 50000),
			docLens:      make(map[uint32]int, 10000),
		}
	}

	// Start workers
	for i := 0; i < numWorkers; i++ {
		fi.wg.Add(1)
		go fi.worker(i)
	}

	return fi
}

// Add adds a document to be indexed.
func (fi *FastIndexer) Add(docID uint32, text string) {
	fi.docCh <- fastDoc{docID: docID, text: text}
	fi.docCount.Add(1)
}

// worker processes documents and accumulates results locally.
func (fi *FastIndexer) worker(workerID int) {
	defer fi.wg.Done()

	data := fi.workerResults[workerID]

	for doc := range fi.docCh {
		termFreqs := fi.Tokenizer(doc.text)

		// Calculate doc length
		docLen := 0
		for _, freq := range termFreqs {
			docLen += freq
		}
		data.docLens[doc.docID] = docLen

		// Add to local posting lists (no locking!)
		for term, freq := range termFreqs {
			data.termPostings[term] = append(data.termPostings[term], IndexPosting{
				DocID: doc.docID,
				Freq:  uint16(freq),
			})
		}
	}
}

// Finish waits for indexing and merges all worker results.
func (fi *FastIndexer) Finish() (map[string][]IndexPosting, []int) {
	close(fi.docCh)
	fi.wg.Wait()

	// Parallel merge of worker results
	return fi.parallelMergeWorkers()
}

func (fi *FastIndexer) parallelMergeWorkers() (map[string][]IndexPosting, []int) {
	// Collect all unique terms
	termSet := make(map[string]struct{}, 100000)
	maxDocID := uint32(0)

	for _, data := range fi.workerResults {
		for term := range data.termPostings {
			termSet[term] = struct{}{}
		}
		for docID := range data.docLens {
			if docID > maxDocID {
				maxDocID = docID
			}
		}
	}

	terms := make([]string, 0, len(termSet))
	for term := range termSet {
		terms = append(terms, term)
	}

	// Parallel term merging
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
				// Count total
				total := 0
				for _, data := range fi.workerResults {
					total += len(data.termPostings[term])
				}

				// Merge (no sort - deferred to block building)
				merged := make([]IndexPosting, 0, total)
				for _, data := range fi.workerResults {
					merged = append(merged, data.termPostings[term]...)
				}

				resultCh <- mergeResult{term: term, postings: merged}
			}
		}()
	}

	// Feed terms
	for _, term := range terms {
		termCh <- term
	}
	close(termCh)

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Collect merged results
	finalTerms := make(map[string][]IndexPosting, len(terms))
	for result := range resultCh {
		finalTerms[result.term] = result.postings
	}

	// Merge doc lengths (parallel)
	docLens := make([]int, maxDocID+1)
	for _, data := range fi.workerResults {
		for docID, length := range data.docLens {
			docLens[docID] = length
		}
	}

	return finalTerms, docLens
}

// DocCount returns the number of indexed documents.
func (fi *FastIndexer) DocCount() int64 {
	return fi.docCount.Load()
}
