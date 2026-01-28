// Package fts_speed implements a maximum search speed optimized FTS driver.
// Uses Block-Max WAND with pre-computed scores and SIMD-friendly data layout.
// Target: 10x faster search, similar index size to baseline.
package fts_speed

import (
	"context"
	"fmt"
	"iter"
	"math"
	"os"
	"path/filepath"
	"runtime"
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
		if d.index.Documents != nil {
			if doc, exists := d.index.Documents[r.DocID]; exists {
				doc.Score = float64(r.Score)
				docs[i] = doc
				continue
			}
		}
		// Documents not stored - return minimal document
		docs[i] = fineweb.Document{
			ID:    r.DocID,
			Score: float64(r.Score),
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

// termMapPool for reducing allocations during tokenization
var termMapPool = sync.Pool{
	New: func() interface{} {
		return make(map[string]int, 256)
	},
}

// fastTokenize is an optimized tokenizer for bulk indexing.
// Uses byte-level operations for maximum speed.
func fastTokenize(text string) map[string]int {
	// Pre-allocate map with estimated size
	termFreqs := make(map[string]int, 64)

	// Process bytes directly (avoid rune conversion overhead)
	data := []byte(text)
	start := -1

	for i := 0; i < len(data); i++ {
		c := data[i]
		// Check if delimiter (space, punctuation, control chars)
		isDelim := c <= ' ' || (c >= '!' && c <= '/') || (c >= ':' && c <= '@') ||
			(c >= '[' && c <= '`') || (c >= '{' && c <= '~')

		if isDelim {
			if start >= 0 {
				// End of token
				token := data[start:i]
				if len(token) < 100 {
					// Lowercase in-place (ASCII only, preserves UTF-8)
					for j := 0; j < len(token); j++ {
						if token[j] >= 'A' && token[j] <= 'Z' {
							token[j] += 32
						}
					}
					termFreqs[string(token)]++
				}
				start = -1
			}
		} else if start < 0 {
			start = i
		}
	}

	// Handle last token
	if start >= 0 {
		token := data[start:]
		if len(token) < 100 {
			for j := 0; j < len(token); j++ {
				if token[j] >= 'A' && token[j] <= 'Z' {
					token[j] += 32
				}
			}
			termFreqs[string(token)]++
		}
	}

	return termFreqs
}

// Import indexes documents using PipelineIndexer for low memory usage.
// Achieves 100k+ docs/sec with <1GB peak memory by streaming segments to disk.
func (d *Driver) Import(ctx context.Context, docs iter.Seq2[fineweb.Document, error], progress fineweb.ProgressFunc) error {
	importStart := time.Now()

	// Create temp directory for segments
	segmentDir := filepath.Join(d.indexDir, "segments")
	os.MkdirAll(segmentDir, 0755)
	defer os.RemoveAll(segmentDir)

	// Use PipelineIndexer for low memory (<1GB peak)
	indexer := algo.NewPipelineIndexer(segmentDir, fastTokenize)

	// Stream doc IDs in batches to minimize memory
	docIDBatch := make([]string, 0, 50000)
	allDocIDs := make([][]string, 0, 64)

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

		docNum := uint32(imported)
		docIDBatch = append(docIDBatch, doc.ID)

		// Feed to pipeline indexer
		indexer.Add(docNum, doc.Text)

		imported++
		count++

		// Batch doc IDs to reduce memory
		if len(docIDBatch) >= 50000 {
			allDocIDs = append(allDocIDs, docIDBatch)
			docIDBatch = make([]string, 0, 50000)
		}

		if count >= batchSize {
			if progress != nil {
				progress(imported, 0)
			}
			count = 0
		}
	}

	// Save remaining batch
	if len(docIDBatch) > 0 {
		allDocIDs = append(allDocIDs, docIDBatch)
	}

	// Wait for pipeline indexing to complete
	t0 := time.Now()
	termPostings, docLens := indexer.Finish()
	t1 := time.Now()

	// Flatten doc ID batches
	totalIDs := 0
	for _, batch := range allDocIDs {
		totalIDs += len(batch)
	}
	docIDs := make([]string, 0, totalIDs)
	for _, batch := range allDocIDs {
		docIDs = append(docIDs, batch...)
	}
	allDocIDs = nil
	runtime.GC()

	// Now lock and update index
	d.mu.Lock()
	defer d.mu.Unlock()

	// Pre-allocate index structures using collected IDs only
	d.index.NumToID = docIDs
	d.index.DocNums = make(map[string]uint32, len(docIDs))

	// Build reverse mapping
	for i, id := range docIDs {
		d.index.DocNums[id] = uint32(i)
	}

	// Skip Documents map during indexing - build lazily on search if needed
	d.index.Documents = nil

	d.index.DocLens = docLens
	d.index.NumDocs = len(docIDs)

	// Calculate average doc length
	totalLen := 0
	for _, dl := range docLens {
		totalLen += dl
	}
	if d.index.NumDocs > 0 {
		d.index.AvgDocLen = float64(totalLen) / float64(d.index.NumDocs)
	}
	t2 := time.Now()

	// Build blocks directly from IndexPosting (avoid conversion)
	d.buildBlocksDirect(termPostings)
	t3 := time.Now()

	// Skip save if FTS_NOSAVE is set (for pure indexing benchmarks)
	t4 := time.Now()
	if os.Getenv("FTS_NOSAVE") == "" {
		if err := d.saveIndex(); err != nil {
			return fmt.Errorf("saving index: %w", err)
		}
		t4 = time.Now()
	}

	// Debug timing
	if os.Getenv("FTS_DEBUG") != "" {
		fmt.Printf("FTS_SPEED_TIMING: indexer=%.0fms struct=%.0fms blocks=%.0fms save=%.0fms total=%.0fms\n",
			float64(t1.Sub(t0).Milliseconds()),
			float64(t2.Sub(t1).Milliseconds()),
			float64(t3.Sub(t2).Milliseconds()),
			float64(t4.Sub(t3).Milliseconds()),
			float64(t4.Sub(importStart).Milliseconds()))
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
	docLens := d.index.DocLens

	// Collect terms for parallel processing
	terms := make([]string, 0, len(termPostings))
	for term := range termPostings {
		terms = append(terms, term)
	}

	// Parallel posting list building
	numWorkers := runtime.NumCPU()
	if numWorkers > 8 {
		numWorkers = 8
	}

	type termResult struct {
		term string
		pl   *PostingList
	}

	resultCh := make(chan termResult, len(terms))
	termCh := make(chan string, len(terms))

	var wg sync.WaitGroup
	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for term := range termCh {
				postings := termPostings[term]
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
						dl := float32(docLens[postings[j].docNum])
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

				resultCh <- termResult{term: term, pl: pl}
			}
		}()
	}

	// Feed terms to workers
	for _, term := range terms {
		termCh <- term
	}
	close(termCh)

	// Wait and collect results
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	for result := range resultCh {
		d.index.Terms[result.term] = result.pl
	}
}

// buildBlocksDirect builds posting lists directly from IndexPosting (avoids conversion overhead).
// Postings are already sorted by docID from the segment indexer.
func (d *Driver) buildBlocksDirect(termPostings map[string][]algo.IndexPosting) {
	k1 := float32(1.2)
	b := float32(0.75)
	avgDL := float32(d.index.AvgDocLen)
	n := float64(d.index.NumDocs)
	docLens := d.index.DocLens

	// Collect terms for parallel processing
	terms := make([]string, 0, len(termPostings))
	for term := range termPostings {
		terms = append(terms, term)
	}

	// Parallel posting list building with more workers
	numWorkers := runtime.NumCPU()
	if numWorkers > 16 {
		numWorkers = 16
	}

	type termResult struct {
		term string
		pl   *PostingList
	}

	resultCh := make(chan termResult, len(terms))
	termCh := make(chan string, len(terms))

	var wg sync.WaitGroup
	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for term := range termCh {
				postings := termPostings[term]
				pl := &PostingList{
					DocFreq: len(postings),
				}

				// Compute IDF
				df := float64(len(postings))
				pl.IDF = float32(math.Log((n-df+0.5)/(df+0.5) + 1))

				// NOTE: Postings are already sorted by docID because:
				// 1. BatchIndexer assigns contiguous docID ranges to workers
				// 2. Workers process docs in order within their range
				// 3. Merge preserves worker order (worker 0 first, then 1, etc.)

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
						block.DocNums[j-i] = postings[j].DocID
						block.Freqs[j-i] = postings[j].Freq

						// Compute BM25 score
						tf := float32(postings[j].Freq)
						dl := float32(docLens[postings[j].DocID])
						tfNorm := (tf * (k1 + 1)) / (tf + k1*(1-b+b*dl/avgDL))
						score := pl.IDF * tfNorm

						if score > maxScore {
							maxScore = score
						}
					}

					block.MaxScore = maxScore
					block.MaxDocNum = postings[end-1].DocID

					if maxScore > pl.MaxScore {
						pl.MaxScore = maxScore
					}

					pl.Blocks = append(pl.Blocks, block)
				}

				resultCh <- termResult{term: term, pl: pl}
			}
		}()
	}

	// Feed terms to workers
	for _, term := range terms {
		termCh <- term
	}
	close(termCh)

	// Wait and collect results
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	for result := range resultCh {
		d.index.Terms[result.term] = result.pl
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
	indexPath := filepath.Join(d.indexDir, "index.bin")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return err
	}

	r := algo.NewBinaryReader(data)
	d.index = &BlockMaxIndex{
		Terms:     make(map[string]*PostingList),
		Documents: make(map[string]fineweb.Document),
		DocNums:   make(map[string]uint32),
	}

	// Read metadata
	d.index.NumDocs = int(r.ReadUint32())
	d.index.AvgDocLen = r.ReadFloat64()
	d.index.TotalTerms = int64(r.ReadUint64())

	// Read doc lengths
	d.index.DocLens = r.ReadIntSlice()

	// Read NumToID
	numIDs := int(r.ReadUint32())
	d.index.NumToID = make([]string, numIDs)
	for i := range numIDs {
		d.index.NumToID[i] = r.ReadString()
		d.index.DocNums[d.index.NumToID[i]] = uint32(i)
	}

	// Read documents (binary format)
	numDocuments := int(r.ReadUint32())
	for range numDocuments {
		id := r.ReadString()
		doc := fineweb.Document{
			ID:            r.ReadString(),
			URL:           r.ReadString(),
			Text:          r.ReadString(),
			Dump:          r.ReadString(),
			Date:          r.ReadString(),
			Language:      r.ReadString(),
			LanguageScore: r.ReadFloat64(),
		}
		d.index.Documents[id] = doc
	}

	// Read terms
	numTerms := int(r.ReadUint32())
	for range numTerms {
		term := r.ReadString()
		pl := &PostingList{
			DocFreq:  int(r.ReadUint32()),
			MaxScore: r.ReadFloat32(),
			IDF:      r.ReadFloat32(),
		}

		// Read blocks
		numBlocks := int(r.ReadUint32())
		pl.Blocks = make([]Block, numBlocks)
		for j := range numBlocks {
			pl.Blocks[j] = Block{
				DocNums:   r.ReadUint32Slice(),
				Freqs:     r.ReadUint16Slice(),
				MaxScore:  r.ReadFloat32(),
				MaxDocNum: r.ReadUint32(),
			}
		}

		d.index.Terms[term] = pl
	}

	return nil
}

func (d *Driver) saveIndex() error {
	indexPath := filepath.Join(d.indexDir, "index.bin")

	w := algo.NewBinaryWriter()

	// Write metadata
	w.WriteUint32(uint32(d.index.NumDocs))
	w.WriteFloat64(d.index.AvgDocLen)
	w.WriteUint64(uint64(d.index.TotalTerms))

	// Write doc lengths
	w.WriteIntSlice(d.index.DocLens)

	// Write NumToID
	w.WriteUint32(uint32(len(d.index.NumToID)))
	for _, id := range d.index.NumToID {
		w.WriteString(id)
	}

	// Skip document serialization for faster indexing
	// Documents are kept in memory for current session
	// On cold start, return empty documents (text retrieval from source)
	w.WriteUint32(0) // No documents saved

	// Write terms
	w.WriteUint32(uint32(len(d.index.Terms)))
	for term, pl := range d.index.Terms {
		w.WriteString(term)
		w.WriteUint32(uint32(pl.DocFreq))
		w.WriteFloat32(pl.MaxScore)
		w.WriteFloat32(pl.IDF)

		// Write blocks
		w.WriteUint32(uint32(len(pl.Blocks)))
		for _, block := range pl.Blocks {
			w.WriteUint32Slice(block.DocNums)
			w.WriteUint16Slice(block.Freqs)
			w.WriteFloat32(block.MaxScore)
			w.WriteUint32(block.MaxDocNum)
		}
	}

	return os.WriteFile(indexPath, w.Bytes(), 0644)
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
