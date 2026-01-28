// Package porter provides a manual inverted index driver with Porter stemming for fineweb full-text search.
package porter

import (
	"container/heap"
	"context"
	"encoding/gob"
	"fmt"
	"iter"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kljensen/snowball"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb"
	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/tokenizer"
)

// termFreqPool is used to recycle term frequency maps to reduce allocations
var termFreqPool = sync.Pool{
	New: func() any {
		return make(map[string]int, 256)
	},
}

// getTermFreqMap gets a map from the pool
func getTermFreqMap() map[string]int {
	return termFreqPool.Get().(map[string]int)
}

// putTermFreqMap returns a map to the pool after clearing it
func putTermFreqMap(m map[string]int) {
	clear(m)
	termFreqPool.Put(m)
}

func init() {
	fineweb.Register("porter", func(cfg fineweb.DriverConfig) (fineweb.Driver, error) {
		return New(cfg)
	})
}

// Driver implements the fineweb.Driver interface using a manual inverted index.
type Driver struct {
	index     *InvertedIndex
	indexDir  string
	dataDir   string
	language  string
	tokenizer *tokenizer.Vietnamese
	mu        sync.RWMutex
}

// New creates a new Porter driver.
func New(cfg fineweb.DriverConfig) (*Driver, error) {
	dataDir := cfg.DataDir
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, "data", "blueprints", "search", "fineweb-2")
	}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("creating data directory: %w", err)
	}

	indexName := "fineweb.porter"
	if cfg.Language != "" {
		indexName = cfg.Language + ".porter"
	}
	indexDir := filepath.Join(dataDir, indexName)

	if err := os.MkdirAll(indexDir, 0755); err != nil {
		return nil, fmt.Errorf("creating index directory: %w", err)
	}

	d := &Driver{
		indexDir:  indexDir,
		dataDir:   dataDir,
		language:  cfg.Language,
		tokenizer: tokenizer.NewVietnamese(),
	}

	// Try to load existing index
	index, err := LoadIndex(indexDir)
	if err != nil {
		// Create new index
		index = NewInvertedIndex()
	}
	d.index = index

	return d, nil
}

// Name returns the driver name.
func (d *Driver) Name() string {
	return "porter"
}

// Info returns driver metadata.
func (d *Driver) Info() *fineweb.DriverInfo {
	return &fineweb.DriverInfo{
		Name:        "porter",
		Description: "Manual inverted index with Porter stemming and BM25 scoring",
		Features:    []string{"bm25", "porter-stemmer", "pure-go", "educational"},
		External:    false,
	}
}

// Search performs full-text search.
func (d *Driver) Search(ctx context.Context, query string, limit, offset int) (*fineweb.SearchResult, error) {
	start := time.Now()

	d.mu.RLock()
	defer d.mu.RUnlock()

	// Tokenize and stem query
	tokens := d.tokenizer.Tokenize(query)
	stemmedTokens := make([]string, 0, len(tokens))
	for _, t := range tokens {
		stemmed, err := snowball.Stem(t, "english", false)
		if err != nil {
			stemmed = strings.ToLower(t)
		}
		stemmedTokens = append(stemmedTokens, stemmed)
	}

	// Search index
	results := d.index.Search(stemmedTokens, limit+offset)

	// Apply offset
	if offset >= len(results) {
		results = nil
	} else if offset > 0 {
		results = results[offset:]
	}

	// Apply limit
	if len(results) > limit {
		results = results[:limit]
	}

	// Convert to documents
	docs := make([]fineweb.Document, 0, len(results))
	for _, r := range results {
		doc, ok := d.index.GetDocument(r.DocID)
		if ok {
			doc.Score = r.Score
			docs = append(docs, doc)
		}
	}

	return &fineweb.SearchResult{
		Documents: docs,
		Duration:  time.Since(start),
		Method:    "porter",
		Total:     int64(d.index.DocCount()),
	}, nil
}

// Import ingests documents from an iterator.
// Periodically releases lock to allow concurrent searches during long imports.
func (d *Driver) Import(ctx context.Context, docs iter.Seq2[fineweb.Document, error], progress fineweb.ProgressFunc) error {
	var imported int64
	batchSize := 10000
	count := 0

	// Batch buffer for processing outside lock
	type docData struct {
		doc    fineweb.Document
		tokens []string
	}
	batch := make([]docData, 0, batchSize)

	for doc, err := range docs {
		if err != nil {
			return fmt.Errorf("reading document: %w", err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Tokenize and stem text (outside lock - CPU intensive)
		tokens := d.tokenizer.Tokenize(doc.Text)
		stemmedTokens := make([]string, 0, len(tokens))
		for _, t := range tokens {
			stemmed, err := snowball.Stem(t, "english", false)
			if err != nil {
				stemmed = strings.ToLower(t)
			}
			stemmedTokens = append(stemmedTokens, stemmed)
		}

		batch = append(batch, docData{doc: doc, tokens: stemmedTokens})
		count++

		if count >= batchSize {
			// Index batch with lock held briefly
			d.mu.Lock()
			for _, item := range batch {
				d.index.IndexDocument(item.doc, item.tokens)
			}
			d.mu.Unlock()

			imported += int64(len(batch))
			batch = batch[:0]
			count = 0

			if progress != nil {
				progress(imported, 0)
			}
		}
	}

	// Index remaining documents
	if len(batch) > 0 {
		d.mu.Lock()
		for _, item := range batch {
			d.index.IndexDocument(item.doc, item.tokens)
		}
		d.mu.Unlock()
		imported += int64(len(batch))
	}

	// Save index to disk
	d.mu.Lock()
	err := d.index.Save(d.indexDir)
	d.mu.Unlock()
	if err != nil {
		return fmt.Errorf("saving index: %w", err)
	}

	if progress != nil {
		progress(imported, imported)
	}

	return nil
}

// Count returns the number of indexed documents.
func (d *Driver) Count(ctx context.Context) (int64, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return int64(d.index.DocCount()), nil
}

// Close saves and closes the index.
func (d *Driver) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.index.Save(d.indexDir)
}

// InvertedIndex is a simple in-memory inverted index with BM25 scoring.
type InvertedIndex struct {
	// Term -> DocID -> term frequency
	Index map[string]map[string]int

	// Document storage
	Documents map[string]fineweb.Document

	// Document lengths (number of tokens)
	DocLengths map[string]int

	// Total number of documents
	NumDocs int

	// Average document length
	AvgDocLen float64

	// Total tokens
	TotalTokens int64
}

// SearchResult holds a search result with score.
type SearchResult struct {
	DocID string
	Score float64
}

// NewInvertedIndex creates a new empty inverted index.
func NewInvertedIndex() *InvertedIndex {
	return &InvertedIndex{
		Index:      make(map[string]map[string]int),
		Documents:  make(map[string]fineweb.Document),
		DocLengths: make(map[string]int),
	}
}

// IndexDocument adds a document to the index.
func (idx *InvertedIndex) IndexDocument(doc fineweb.Document, tokens []string) {
	// Skip if already indexed
	if _, exists := idx.Documents[doc.ID]; exists {
		return
	}

	// Store document
	idx.Documents[doc.ID] = doc
	idx.DocLengths[doc.ID] = len(tokens)
	idx.NumDocs++
	idx.TotalTokens += int64(len(tokens))
	idx.AvgDocLen = float64(idx.TotalTokens) / float64(idx.NumDocs)

	// Count term frequencies using pooled map
	termFreqs := getTermFreqMap()
	for _, token := range tokens {
		termFreqs[token]++
	}

	// Update inverted index
	for term, freq := range termFreqs {
		if idx.Index[term] == nil {
			idx.Index[term] = make(map[string]int)
		}
		idx.Index[term][doc.ID] = freq
	}

	// Return map to pool
	putTermFreqMap(termFreqs)
}

// resultHeap implements a min-heap for top-K selection
type resultHeap []SearchResult

func (h resultHeap) Len() int           { return len(h) }
func (h resultHeap) Less(i, j int) bool { return h[i].Score < h[j].Score } // Min-heap by score
func (h resultHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *resultHeap) Push(x any)        { *h = append(*h, x.(SearchResult)) }
func (h *resultHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// Search performs BM25 search and returns ranked results.
// Uses heap-based top-K selection for better performance with large result sets.
func (idx *InvertedIndex) Search(queryTerms []string, limit int) []SearchResult {
	if idx.NumDocs == 0 {
		return nil
	}

	// BM25 parameters
	k1 := 1.2
	b := 0.75

	// Calculate scores
	scores := make(map[string]float64)

	for _, term := range queryTerms {
		postings, exists := idx.Index[term]
		if !exists {
			continue
		}

		// IDF calculation
		df := float64(len(postings))
		idf := math.Log((float64(idx.NumDocs)-df+0.5)/(df+0.5) + 1)

		// Score each document containing this term
		for docID, tf := range postings {
			docLen := float64(idx.DocLengths[docID])
			tfNorm := (float64(tf) * (k1 + 1)) / (float64(tf) + k1*(1-b+b*docLen/idx.AvgDocLen))
			scores[docID] += idf * tfNorm
		}
	}

	// Use heap for efficient top-K selection
	// This is O(n log k) instead of O(n log n) for full sort
	h := make(resultHeap, 0, limit+1)
	heap.Init(&h)

	for docID, score := range scores {
		if h.Len() < limit {
			heap.Push(&h, SearchResult{DocID: docID, Score: score})
		} else if score > h[0].Score {
			// Replace minimum if current score is higher
			h[0] = SearchResult{DocID: docID, Score: score}
			heap.Fix(&h, 0)
		}
	}

	// Extract results in descending order
	results := make([]SearchResult, h.Len())
	for i := len(results) - 1; i >= 0; i-- {
		results[i] = heap.Pop(&h).(SearchResult)
	}

	return results
}

// GetDocument retrieves a document by ID.
func (idx *InvertedIndex) GetDocument(id string) (fineweb.Document, bool) {
	doc, ok := idx.Documents[id]
	return doc, ok
}

// DocCount returns the number of documents.
func (idx *InvertedIndex) DocCount() int {
	return idx.NumDocs
}

// Save persists the index to disk.
func (idx *InvertedIndex) Save(dir string) error {
	f, err := os.Create(filepath.Join(dir, "index.gob"))
	if err != nil {
		return err
	}
	defer f.Close()

	return gob.NewEncoder(f).Encode(idx)
}

// LoadIndex loads an index from disk.
func LoadIndex(dir string) (*InvertedIndex, error) {
	f, err := os.Open(filepath.Join(dir, "index.gob"))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var idx InvertedIndex
	if err := gob.NewDecoder(f).Decode(&idx); err != nil {
		return nil, err
	}

	return &idx, nil
}

// Ensure Driver implements all required interfaces
var (
	_ fineweb.Driver  = (*Driver)(nil)
	_ fineweb.Indexer = (*Driver)(nil)
	_ fineweb.Stats   = (*Driver)(nil)
)
