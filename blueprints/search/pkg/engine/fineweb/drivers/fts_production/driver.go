// Package fts_production implements a production-ready FTS driver.
// Uses proven techniques: WAND with skip pointers, Roaring Bitmaps, FST, and Snappy compression.
// Target: 3x faster search, 4x smaller index, maximum stability.
package fts_production

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
	"github.com/golang/snappy"
	"github.com/kljensen/snowball"
)

// makeTokenizerFunc creates a tokenizer function for the parallel indexer.
func makeTokenizerFunc(tok *tokenizer.Vietnamese) algo.TokenizerFunc {
	return func(text string) map[string]int {
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
}

func init() {
	fineweb.Register("fts_production", func(cfg fineweb.DriverConfig) (fineweb.Driver, error) {
		return New(cfg)
	})
}

const (
	SkipInterval = 64 // Skip pointer every 64 postings
)

// Driver implements production-ready FTS.
type Driver struct {
	mu        sync.RWMutex
	index     *ProductionIndex
	indexDir  string
	tokenizer *tokenizer.Vietnamese
	language  string
}

// ProductionIndex uses battle-tested data structures.
type ProductionIndex struct {
	// Term dictionary
	TermDict *algo.FST

	// Posting lists with skip pointers
	Terms map[string]*ProductionPostingList

	// Document storage with Snappy compression
	Documents []compressedDoc

	// Metadata
	NumDocs   int
	AvgDocLen float64
	DocLens   []int
}

// ProductionPostingList uses Roaring + skip pointers.
type ProductionPostingList struct {
	// Roaring bitmap for efficient set operations
	DocBitmap *algo.RoaringBitmap

	// Arrays for scoring (parallel arrays)
	DocIDs []uint32
	Freqs  []uint16

	// Skip pointers for fast seeking
	SkipDocs   []uint32 // Doc ID at skip point
	SkipOffset []int    // Offset in DocIDs at skip point

	// Precomputed values
	MaxScore float32
	DocFreq  int
	IDF      float32
}

type compressedDoc struct {
	ID        string
	URL       string
	TextData  []byte // Snappy compressed
	Dump      string
	Date      string
	Language  string
	LangScore float64
}

// New creates a new production driver.
func New(cfg fineweb.DriverConfig) (*Driver, error) {
	dataDir := cfg.DataDir
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, "data", "blueprints", "search", "fineweb-2")
	}

	indexDir := filepath.Join(dataDir, cfg.Language+".fts_production")
	if err := os.MkdirAll(indexDir, 0755); err != nil {
		return nil, fmt.Errorf("creating index directory: %w", err)
	}

	d := &Driver{
		indexDir:  indexDir,
		tokenizer: tokenizer.NewVietnamese(),
		language:  cfg.Language,
	}

	if err := d.loadIndex(); err != nil {
		d.index = &ProductionIndex{
			Terms: make(map[string]*ProductionPostingList),
		}
	}

	return d, nil
}

// Name returns the driver name.
func (d *Driver) Name() string {
	return "fts_production"
}

// Info returns driver metadata.
func (d *Driver) Info() *fineweb.DriverInfo {
	return &fineweb.DriverInfo{
		Name:        "fts_production",
		Description: "Production: WAND + Roaring + FST + Snappy (proven techniques)",
		Features:    []string{"wand", "skip-pointers", "roaring-bitmaps", "fst", "snappy", "bm25"},
		External:    false,
	}
}

// Search performs WAND search with skip pointers.
func (d *Driver) Search(ctx context.Context, query string, limit, offset int) (*fineweb.SearchResult, error) {
	start := time.Now()

	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.index.NumDocs == 0 {
		return &fineweb.SearchResult{
			Documents: []fineweb.Document{},
			Duration:  time.Since(start),
			Method:    "fts_production",
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

	// Get posting lists
	pls := make([]*ProductionPostingList, 0, len(queryTerms))
	for _, term := range queryTerms {
		if pl, exists := d.index.Terms[term]; exists && pl.DocFreq > 0 {
			pls = append(pls, pl)
		}
	}

	if len(pls) == 0 {
		return &fineweb.SearchResult{
			Documents: []fineweb.Document{},
			Duration:  time.Since(start),
			Method:    "fts_production",
		}, nil
	}

	// WAND search with skip pointers
	results := d.wandSearch(ctx, pls, limit+offset)

	// Apply offset
	if offset >= len(results) {
		results = nil
	} else {
		results = results[offset:]
	}
	if len(results) > limit {
		results = results[:limit]
	}

	// Decompress and return documents
	docs := make([]fineweb.Document, len(results))
	for i, r := range results {
		cdoc := d.index.Documents[r.docID]

		// Decompress text
		text, _ := snappy.Decode(nil, cdoc.TextData)

		docs[i] = fineweb.Document{
			ID:            cdoc.ID,
			URL:           cdoc.URL,
			Text:          string(text),
			Dump:          cdoc.Dump,
			Date:          cdoc.Date,
			Language:      cdoc.Language,
			LanguageScore: cdoc.LangScore,
			Score:         float64(r.score),
		}
	}

	return &fineweb.SearchResult{
		Documents: docs,
		Duration:  time.Since(start),
		Method:    "fts_production",
		Total:     int64(len(results)),
	}, nil
}

type searchResult struct {
	docID uint32
	score float32
}

// plIter is an iterator over a posting list.
type plIter struct {
	pl    *ProductionPostingList
	pos   int
	docID uint32
	freq  uint16
}

// wandSearch implements WAND with skip pointers.
func (d *Driver) wandSearch(ctx context.Context, pls []*ProductionPostingList, k int) []searchResult {
	// Create iterators
	iters := make([]*plIter, len(pls))
	for i, pl := range pls {
		iters[i] = &plIter{pl: pl}
		if len(pl.DocIDs) > 0 {
			iters[i].docID = pl.DocIDs[0]
			iters[i].freq = pl.Freqs[0]
		} else {
			iters[i].docID = math.MaxUint32
		}
	}

	// Result heap (min-heap)
	results := make([]searchResult, 0, k)
	threshold := float32(0)

	// BM25 parameters
	k1 := float32(1.2)
	b := float32(0.75)
	avgDL := float32(d.index.AvgDocLen)

	// Main WAND loop
mainLoop:
	for {
		select {
		case <-ctx.Done():
			break mainLoop
		default:
		}

		// Sort iterators by current doc ID
		sort.Slice(iters, func(i, j int) bool {
			return iters[i].docID < iters[j].docID
		})

		// Remove exhausted iterators
		activeIters := iters[:0]
		for _, it := range iters {
			if it.docID != math.MaxUint32 {
				activeIters = append(activeIters, it)
			}
		}
		iters = activeIters

		if len(iters) == 0 {
			break
		}

		// Find pivot
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

		pivotDoc := iters[pivotIdx].docID

		// Check if all iterators up to pivot are at the same document
		if iters[0].docID == pivotDoc {
			// Score the document
			score := float32(0)
			for i := 0; i <= pivotIdx; i++ {
				if iters[i].docID == pivotDoc {
					tf := float32(iters[i].freq)
					dl := float32(d.index.DocLens[pivotDoc])
					tfNorm := (tf * (k1 + 1)) / (tf + k1*(1-b+b*dl/avgDL))
					score += iters[i].pl.IDF * tfNorm
				}
			}

			// Update results
			if len(results) < k {
				results = append(results, searchResult{docID: pivotDoc, score: score})
				sort.Slice(results, func(i, j int) bool {
					return results[i].score < results[j].score
				})
				if len(results) == k {
					threshold = results[0].score
				}
			} else if score > threshold {
				results[0] = searchResult{docID: pivotDoc, score: score}
				sort.Slice(results, func(i, j int) bool {
					return results[i].score < results[j].score
				})
				threshold = results[0].score
			}

			// Advance all iterators at pivot
			for i := 0; i <= pivotIdx; i++ {
				if iters[i].docID == pivotDoc {
					d.advanceIterator(iters[i])
				}
			}
		} else {
			// Skip first iterator to pivot using skip pointers
			d.skipTo(iters[0], pivotDoc)
		}
	}

	// Sort results by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	return results
}

func (d *Driver) advanceIterator(it *plIter) {
	it.pos++
	if it.pos < len(it.pl.DocIDs) {
		it.docID = it.pl.DocIDs[it.pos]
		it.freq = it.pl.Freqs[it.pos]
	} else {
		it.docID = math.MaxUint32
	}
}

func (d *Driver) skipTo(it *plIter, target uint32) {
	// Use skip pointers for fast seeking
	for i := len(it.pl.SkipDocs) - 1; i >= 0; i-- {
		if it.pl.SkipDocs[i] <= target && it.pl.SkipOffset[i] > it.pos {
			it.pos = it.pl.SkipOffset[i]
			break
		}
	}

	// Linear scan from skip point
	for it.pos < len(it.pl.DocIDs) && it.pl.DocIDs[it.pos] < target {
		it.pos++
	}

	if it.pos < len(it.pl.DocIDs) {
		it.docID = it.pl.DocIDs[it.pos]
		it.freq = it.pl.Freqs[it.pos]
	} else {
		it.docID = math.MaxUint32
	}
}

// Import indexes documents using parallel processing.
func (d *Driver) Import(ctx context.Context, docs iter.Seq2[fineweb.Document, error], progress fineweb.ProgressFunc) error {
	// Create streaming indexer
	indexer := algo.NewStreamingIndexer(makeTokenizerFunc(d.tokenizer))

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

		allDocs = append(allDocs, doc)

		// Feed to parallel indexer
		indexer.Add(uint32(len(allDocs)-1), doc.Text)

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

	// Parallel document compression
	compressedDocs := make([]compressedDoc, len(allDocs))
	numWorkers := runtime.NumCPU()
	if numWorkers > 8 {
		numWorkers = 8
	}

	var compressWg sync.WaitGroup
	docCh := make(chan int, len(allDocs))

	for range numWorkers {
		compressWg.Add(1)
		go func() {
			defer compressWg.Done()
			for i := range docCh {
				doc := allDocs[i]
				textData := snappy.Encode(nil, []byte(doc.Text))
				compressedDocs[i] = compressedDoc{
					ID:        doc.ID,
					URL:       doc.URL,
					TextData:  textData,
					Dump:      doc.Dump,
					Date:      doc.Date,
					Language:  doc.Language,
					LangScore: doc.LanguageScore,
				}
			}
		}()
	}

	for i := range allDocs {
		docCh <- i
	}
	close(docCh)
	compressWg.Wait()

	// Now lock and update index
	d.mu.Lock()
	defer d.mu.Unlock()

	// Store compressed documents
	d.index.Documents = compressedDocs
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

	// Convert posting format
	postings := make(map[string][]posting, len(termPostings))
	for term, plist := range termPostings {
		converted := make([]posting, len(plist))
		for i, p := range plist {
			converted[i] = posting{docNum: p.DocID, freq: p.Freq}
		}
		postings[term] = converted
	}

	// Build production posting lists
	d.buildProductionPostings(postings)

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

func (d *Driver) buildProductionPostings(termPostings map[string][]posting) {
	n := float64(d.index.NumDocs)
	k1 := float32(1.2)
	b := float32(0.75)
	avgDL := float32(d.index.AvgDocLen)
	docLens := d.index.DocLens

	// Collect terms for parallel processing
	terms := make([]string, 0, len(termPostings))
	for term := range termPostings {
		terms = append(terms, term)
	}
	sort.Strings(terms)

	// Parallel posting list building
	numWorkers := runtime.NumCPU()
	if numWorkers > 8 {
		numWorkers = 8
	}

	type termResult struct {
		term string
		pl   *ProductionPostingList
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

				// Sort by doc number
				sort.Slice(postings, func(i, j int) bool {
					return postings[i].docNum < postings[j].docNum
				})

				// Build arrays
				docIDs := make([]uint32, len(postings))
				freqs := make([]uint16, len(postings))
				bitmap := algo.NewRoaringBitmap()

				for i, p := range postings {
					docIDs[i] = p.docNum
					freqs[i] = p.freq
					bitmap.Add(p.docNum)
				}

				// Build skip pointers
				var skipDocs []uint32
				var skipOffsets []int
				for i := 0; i < len(docIDs); i += SkipInterval {
					skipDocs = append(skipDocs, docIDs[i])
					skipOffsets = append(skipOffsets, i)
				}

				// Compute IDF
				df := float64(len(postings))
				idf := float32(math.Log((n-df+0.5)/(df+0.5) + 1))

				// Compute max score
				maxScore := float32(0)
				for i, p := range postings {
					tf := float32(p.freq)
					dl := float32(docLens[docIDs[i]])
					tfNorm := (tf * (k1 + 1)) / (tf + k1*(1-b+b*dl/avgDL))
					score := idf * tfNorm
					if score > maxScore {
						maxScore = score
					}
				}

				resultCh <- termResult{
					term: term,
					pl: &ProductionPostingList{
						DocBitmap:  bitmap,
						DocIDs:     docIDs,
						Freqs:      freqs,
						SkipDocs:   skipDocs,
						SkipOffset: skipOffsets,
						MaxScore:   maxScore,
						DocFreq:    len(postings),
						IDF:        idf,
					},
				}
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

	// Build FST (must be sequential due to sorted insertion)
	fstBuilder := algo.NewFSTBuilder()
	for idx, term := range terms {
		fstBuilder.Add(term, uint64(idx))
	}
	d.index.TermDict = fstBuilder.Build()
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
	d.index = &ProductionIndex{
		Terms: make(map[string]*ProductionPostingList),
	}

	// Read metadata
	d.index.NumDocs = int(r.ReadUint32())
	d.index.AvgDocLen = r.ReadFloat64()

	// Read doc lengths
	d.index.DocLens = r.ReadIntSlice()

	// Read documents (binary format)
	numDocuments := int(r.ReadUint32())
	d.index.Documents = make([]compressedDoc, numDocuments)
	for i := range numDocuments {
		d.index.Documents[i] = compressedDoc{
			ID:        r.ReadString(),
			URL:       r.ReadString(),
			TextData:  r.ReadBytes(),
			Dump:      r.ReadString(),
			Date:      r.ReadString(),
			Language:  r.ReadString(),
			LangScore: r.ReadFloat64(),
		}
	}

	// Read terms
	numTerms := int(r.ReadUint32())
	for range numTerms {
		term := r.ReadString()
		pl := &ProductionPostingList{
			DocFreq:  int(r.ReadUint32()),
			MaxScore: r.ReadFloat32(),
			IDF:      r.ReadFloat32(),
		}

		// Read arrays
		pl.DocIDs = r.ReadUint32Slice()
		pl.Freqs = r.ReadUint16Slice()
		pl.SkipDocs = r.ReadUint32Slice()
		pl.SkipOffset = r.ReadIntSlice()

		// Rebuild bitmap from DocIDs
		pl.DocBitmap = algo.NewRoaringBitmap()
		for _, docID := range pl.DocIDs {
			pl.DocBitmap.Add(docID)
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

	// Write doc lengths
	w.WriteIntSlice(d.index.DocLens)

	// Write documents (binary format)
	w.WriteUint32(uint32(len(d.index.Documents)))
	for _, doc := range d.index.Documents {
		w.WriteString(doc.ID)
		w.WriteString(doc.URL)
		w.WriteBytes(doc.TextData)
		w.WriteString(doc.Dump)
		w.WriteString(doc.Date)
		w.WriteString(doc.Language)
		w.WriteFloat64(doc.LangScore)
	}

	// Write terms
	w.WriteUint32(uint32(len(d.index.Terms)))
	for term, pl := range d.index.Terms {
		w.WriteString(term)
		w.WriteUint32(uint32(pl.DocFreq))
		w.WriteFloat32(pl.MaxScore)
		w.WriteFloat32(pl.IDF)

		// Write arrays
		w.WriteUint32Slice(pl.DocIDs)
		w.WriteUint16Slice(pl.Freqs)
		w.WriteUint32Slice(pl.SkipDocs)
		w.WriteIntSlice(pl.SkipOffset)
	}

	return os.WriteFile(indexPath, w.Bytes(), 0644)
}

var (
	_ fineweb.Driver  = (*Driver)(nil)
	_ fineweb.Indexer = (*Driver)(nil)
	_ fineweb.Stats   = (*Driver)(nil)
)
