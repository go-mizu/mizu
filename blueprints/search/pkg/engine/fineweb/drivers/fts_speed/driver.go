// Package fts_speed implements a maximum search speed optimized FTS driver.
// Uses Block-Max WAND with pre-computed scores and SIMD-friendly data layout.
// Target: 10x faster search, similar index size to baseline.
package fts_speed

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"iter"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb"
	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/algo"
	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/tokenizer"
	"github.com/kljensen/snowball"
)

func init() {
	fineweb.Register("fts_speed", func(cfg fineweb.DriverConfig) (fineweb.Driver, error) {
		return New(cfg)
	})
}

const (
	BlockSize    = 128  // Documents per block
	SkipInterval = 128  // Skip pointer interval
)

// Driver implements maximum search speed optimization.
type Driver struct {
	mu        sync.RWMutex
	index     *BlockMaxIndex
	indexDir  string
	tokenizer *tokenizer.Vietnamese
	language  string
}

// BlockMaxIndex is the main index structure.
type BlockMaxIndex struct {
	Terms      map[string]*PostingList
	Documents  map[string]fineweb.Document
	DocNums    map[string]uint32 // docID -> sequential number
	NumToID    []string          // sequential number -> docID
	DocLens    []int             // Document lengths by doc number
	NumDocs    int
	AvgDocLen  float64
	TotalTerms int64
}

// PostingList with block-level max scores for WAND.
type PostingList struct {
	Blocks    []Block
	MaxScore  float32 // Global max score
	DocFreq   int
	IDF       float32
}

// Block is a group of postings with pre-computed max score.
type Block struct {
	DocNums   []uint32  // Document numbers (sequential)
	Freqs     []uint16  // Term frequencies
	MaxScore  float32   // Maximum BM25 score in this block
	MaxDocNum uint32    // Last doc number in block
}

// New creates a new maximum-speed driver.
func New(cfg fineweb.DriverConfig) (*Driver, error) {
	dataDir := cfg.DataDir
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, "data", "blueprints", "search", "fineweb-2")
	}

	indexDir := filepath.Join(dataDir, cfg.Language+".fts_speed")
	if err := os.MkdirAll(indexDir, 0755); err != nil {
		return nil, fmt.Errorf("creating index directory: %w", err)
	}

	d := &Driver{
		indexDir:  indexDir,
		tokenizer: tokenizer.NewVietnamese(),
		language:  cfg.Language,
	}

	// Try to load existing index
	if err := d.loadIndex(); err != nil {
		d.index = &BlockMaxIndex{
			Terms:     make(map[string]*PostingList),
			Documents: make(map[string]fineweb.Document),
			DocNums:   make(map[string]uint32),
		}
	}

	return d, nil
}

// Name returns the driver name.
func (d *Driver) Name() string {
	return "fts_speed"
}

// Info returns driver metadata.
func (d *Driver) Info() *fineweb.DriverInfo {
	return &fineweb.DriverInfo{
		Name:        "fts_speed",
		Description: "Maximum speed: Block-Max WAND with pre-computed scores",
		Features:    []string{"block-max-wand", "simd-friendly", "skip-pointers", "bm25"},
		External:    false,
	}
}

// Search performs Block-Max WAND search for maximum speed.
func (d *Driver) Search(ctx context.Context, query string, limit, offset int) (*fineweb.SearchResult, error) {
	start := time.Now()

	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.index.NumDocs == 0 {
		return &fineweb.SearchResult{
			Documents: []fineweb.Document{},
			Duration:  time.Since(start),
			Method:    "fts_speed",
		}, nil
	}

	// Tokenize and stem query
	tokens := d.tokenizer.Tokenize(query)
	queryTerms := make([]string, 0, len(tokens))
	for _, t := range tokens {
		stemmed, err := snowball.Stem(t, "english", false)
		if err != nil {
			stemmed = strings.ToLower(t)
		}
		queryTerms = append(queryTerms, stemmed)
	}

	// Block-Max WAND search
	results := d.blockMaxWAND(ctx, queryTerms, limit+offset)

	// Apply offset
	if offset > 0 {
		if offset >= len(results) {
			results = nil
		} else {
			results = results[offset:]
		}
	}

	// Limit results
	if len(results) > limit {
		results = results[:limit]
	}

	// Convert to fineweb.Document
	docs := make([]fineweb.Document, len(results))
	for i, r := range results {
		if doc, exists := d.index.Documents[r.DocID]; exists {
			doc.Score = float64(r.Score)
			docs[i] = doc
		}
	}

	return &fineweb.SearchResult{
		Documents: docs,
		Duration:  time.Since(start),
		Method:    "fts_speed",
		Total:     int64(len(results)),
	}, nil
}

// termIter is an iterator over a posting list.
type termIter struct {
	pl       *PostingList
	blockIdx int
	posInBlk int
	idf      float32
}

func (it *termIter) currentDocNum() uint32 {
	if it.blockIdx >= len(it.pl.Blocks) {
		return math.MaxUint32
	}
	block := &it.pl.Blocks[it.blockIdx]
	if it.posInBlk >= len(block.DocNums) {
		return math.MaxUint32
	}
	return block.DocNums[it.posInBlk]
}

func (it *termIter) currentFreq() uint16 {
	if it.blockIdx >= len(it.pl.Blocks) {
		return 0
	}
	block := &it.pl.Blocks[it.blockIdx]
	if it.posInBlk >= len(block.DocNums) {
		return 0
	}
	return block.Freqs[it.posInBlk]
}

func (it *termIter) currentBlockMaxScore() float32 {
	if it.blockIdx >= len(it.pl.Blocks) {
		return 0
	}
	return it.pl.Blocks[it.blockIdx].MaxScore
}

func (it *termIter) isExhausted() bool {
	return it.blockIdx >= len(it.pl.Blocks)
}

func (it *termIter) advance() {
	it.posInBlk++
	if it.blockIdx < len(it.pl.Blocks) && it.posInBlk >= len(it.pl.Blocks[it.blockIdx].DocNums) {
		it.blockIdx++
		it.posInBlk = 0
	}
}

func (it *termIter) skipToDoc(target uint32) {
	for it.blockIdx < len(it.pl.Blocks) {
		block := &it.pl.Blocks[it.blockIdx]
		if block.MaxDocNum >= target {
			for it.posInBlk < len(block.DocNums) {
				if block.DocNums[it.posInBlk] >= target {
					return
				}
				it.posInBlk++
			}
		}
		it.blockIdx++
		it.posInBlk = 0
	}
}

// blockMaxWAND implements the Block-Max WAND algorithm.
func (d *Driver) blockMaxWAND(ctx context.Context, queryTerms []string, k int) []searchResult {
	// Get posting lists
	iters := make([]*termIter, 0, len(queryTerms))
	for _, term := range queryTerms {
		if pl, exists := d.index.Terms[term]; exists && pl.DocFreq > 0 {
			iters = append(iters, &termIter{pl: pl, idf: pl.IDF})
		}
	}

	if len(iters) == 0 {
		return nil
	}

	// Result heap
	results := &minHeap{}
	threshold := float32(0)

	// BM25 parameters
	k1 := float32(1.2)
	b := float32(0.75)
	avgDL := float32(d.index.AvgDocLen)

	// Main WAND loop
	for {
		select {
		case <-ctx.Done():
			break
		default:
		}

		// Sort by current doc number
		sort.Slice(iters, func(i, j int) bool {
			return iters[i].currentDocNum() < iters[j].currentDocNum()
		})

		// Remove exhausted iterators
		active := iters[:0]
		for _, it := range iters {
			if !it.isExhausted() {
				active = append(active, it)
			}
		}
		iters = active

		if len(iters) == 0 {
			break
		}

		// Find pivot where cumulative max scores >= threshold
		pivotIdx := -1
		cumSum := float32(0)
		for i, it := range iters {
			cumSum += it.pl.MaxScore
			if cumSum >= threshold {
				pivotIdx = i
				break
			}
		}

		if pivotIdx < 0 {
			break
		}

		pivotDoc := iters[pivotIdx].currentDocNum()

		// Check if all point to same doc
		if iters[0].currentDocNum() == pivotDoc {
			// Score document
			score := float32(0)
			for i := 0; i <= pivotIdx; i++ {
				if iters[i].currentDocNum() == pivotDoc {
					tf := float32(iters[i].currentFreq())
					dl := float32(d.index.DocLens[pivotDoc])
					// BM25 score
					tfNorm := (tf * (k1 + 1)) / (tf + k1*(1-b+b*dl/avgDL))
					score += iters[i].idf * tfNorm
				}
			}

			if score > threshold || results.Len() < k {
				docID := d.index.NumToID[pivotDoc]
				results.push(searchResult{DocID: docID, Score: score})
				if results.Len() > k {
					results.pop()
				}
				if results.Len() >= k {
					threshold = results.peek().Score
				}
			}

			// Advance iterators at pivot
			for i := 0; i <= pivotIdx; i++ {
				if iters[i].currentDocNum() == pivotDoc {
					iters[i].advance()
				}
			}
		} else {
			// Block skip
			it := iters[0]
			blockMax := it.currentBlockMaxScore()

			if blockMax < threshold-cumSum+it.pl.MaxScore {
				// Skip entire block
				it.blockIdx++
				it.posInBlk = 0
			} else {
				// Advance within block to pivot
				it.skipToDoc(pivotDoc)
			}
		}
	}

	// Extract results in descending score order
	finalResults := make([]searchResult, results.Len())
	for i := results.Len() - 1; i >= 0; i-- {
		finalResults[i] = results.pop()
	}

	return finalResults
}

// Import indexes documents using parallel processing.
func (d *Driver) Import(ctx context.Context, docs iter.Seq2[fineweb.Document, error], progress fineweb.ProgressFunc) error {
	// Create tokenizer function for parallel indexer
	tok := d.tokenizer
	tokenizerFunc := func(text string) map[string]int {
		tokens := tok.Tokenize(text)
		termFreqs := make(map[string]int, len(tokens)/2)
		for _, t := range tokens {
			stemmed, err := snowball.Stem(t, "english", false)
			if err != nil {
				stemmed = strings.ToLower(t)
			}
			termFreqs[stemmed]++
		}
		return termFreqs
	}

	// Create streaming indexer
	indexer := algo.NewStreamingIndexer(tokenizerFunc)

	// Collect documents and feed to indexer
	var allDocs []fineweb.Document
	var imported int64
	batchSize := 10000
	count := 0

	for doc, err := range docs {
		if err != nil {
			return fmt.Errorf("reading document: %w", err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		docNum := uint32(len(allDocs))
		allDocs = append(allDocs, doc)

		// Feed to parallel indexer
		indexer.Add(docNum, doc.Text)

		imported++
		count++

		if count >= batchSize {
			if progress != nil {
				progress(imported, 0)
			}
			count = 0
		}
	}

	// Wait for parallel indexing to complete
	termPostings, docLens := indexer.Finish()

	// Now lock and update index
	d.mu.Lock()
	defer d.mu.Unlock()

	// Build index structures
	for i, doc := range allDocs {
		docNum := uint32(i)
		d.index.DocNums[doc.ID] = docNum
		d.index.NumToID = append(d.index.NumToID, doc.ID)
		d.index.Documents[doc.ID] = doc
	}

	d.index.DocLens = docLens
	d.index.NumDocs = len(allDocs)

	// Calculate average doc length
	totalLen := 0
	for _, dl := range docLens {
		totalLen += dl
	}
	if d.index.NumDocs > 0 {
		d.index.AvgDocLen = float64(totalLen) / float64(d.index.NumDocs)
	}

	// Convert posting format and build blocks
	postings := make(map[string][]posting, len(termPostings))
	for term, plist := range termPostings {
		converted := make([]posting, len(plist))
		for i, p := range plist {
			converted[i] = posting{docNum: p.DocID, freq: p.Freq}
		}
		postings[term] = converted
	}

	d.buildBlocks(postings)

	// Save index
	if err := d.saveIndex(); err != nil {
		return fmt.Errorf("saving index: %w", err)
	}

	if progress != nil {
		progress(imported, imported)
	}

	return nil
}

type posting struct {
	docNum uint32
	freq   uint16
}

func (d *Driver) buildBlocks(termPostings map[string][]posting) {
	k1 := float32(1.2)
	b := float32(0.75)
	avgDL := float32(d.index.AvgDocLen)
	n := float64(d.index.NumDocs)

	for term, postings := range termPostings {
		pl := &PostingList{
			DocFreq: len(postings),
		}

		// Compute IDF
		df := float64(len(postings))
		pl.IDF = float32(math.Log((n-df+0.5)/(df+0.5) + 1))

		// Sort postings by doc number
		sort.Slice(postings, func(i, j int) bool {
			return postings[i].docNum < postings[j].docNum
		})

		// Build blocks
		for i := 0; i < len(postings); i += BlockSize {
			end := i + BlockSize
			if end > len(postings) {
				end = len(postings)
			}

			block := Block{
				DocNums: make([]uint32, end-i),
				Freqs:   make([]uint16, end-i),
			}

			maxScore := float32(0)
			for j := i; j < end; j++ {
				block.DocNums[j-i] = postings[j].docNum
				block.Freqs[j-i] = postings[j].freq

				// Compute BM25 score
				tf := float32(postings[j].freq)
				dl := float32(d.index.DocLens[postings[j].docNum])
				tfNorm := (tf * (k1 + 1)) / (tf + k1*(1-b+b*dl/avgDL))
				score := pl.IDF * tfNorm

				if score > maxScore {
					maxScore = score
				}
			}

			block.MaxScore = maxScore
			block.MaxDocNum = postings[end-1].docNum

			if maxScore > pl.MaxScore {
				pl.MaxScore = maxScore
			}

			pl.Blocks = append(pl.Blocks, block)
		}

		d.index.Terms[term] = pl
	}
}

// Count returns document count.
func (d *Driver) Count(ctx context.Context) (int64, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return int64(d.index.NumDocs), nil
}

// Close releases resources.
func (d *Driver) Close() error {
	return nil
}

func (d *Driver) loadIndex() error {
	indexPath := filepath.Join(d.indexDir, "index.gob")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return err
	}

	d.index = &BlockMaxIndex{}
	return gob.NewDecoder(bytes.NewReader(data)).Decode(d.index)
}

func (d *Driver) saveIndex() error {
	indexPath := filepath.Join(d.indexDir, "index.gob")

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(d.index); err != nil {
		return err
	}

	return os.WriteFile(indexPath, buf.Bytes(), 0644)
}

// searchResult for internal use.
type searchResult struct {
	DocID string
	Score float32
}

// minHeap for top-k results.
type minHeap []searchResult

func (h *minHeap) Len() int            { return len(*h) }
func (h *minHeap) peek() searchResult  { return (*h)[0] }
func (h *minHeap) push(r searchResult) {
	*h = append(*h, r)
	h.up(len(*h) - 1)
}
func (h *minHeap) pop() searchResult {
	old := *h
	n := len(old)
	x := old[0]
	old[0] = old[n-1]
	*h = old[:n-1]
	if len(*h) > 0 {
		h.down(0)
	}
	return x
}
func (h *minHeap) up(i int) {
	for i > 0 {
		p := (i - 1) / 2
		if (*h)[p].Score <= (*h)[i].Score {
			break
		}
		(*h)[p], (*h)[i] = (*h)[i], (*h)[p]
		i = p
	}
}
func (h *minHeap) down(i int) {
	for {
		l := 2*i + 1
		if l >= len(*h) {
			break
		}
		min := l
		if r := l + 1; r < len(*h) && (*h)[r].Score < (*h)[l].Score {
			min = r
		}
		if (*h)[i].Score <= (*h)[min].Score {
			break
		}
		(*h)[i], (*h)[min] = (*h)[min], (*h)[i]
		i = min
	}
}

var (
	_ fineweb.Driver  = (*Driver)(nil)
	_ fineweb.Indexer = (*Driver)(nil)
	_ fineweb.Stats   = (*Driver)(nil)
)
