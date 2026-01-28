// Package algo provides parallel indexing infrastructure for high-performance FTS.
package algo

import (
	"runtime"
	"sync"
	"sync/atomic"
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
