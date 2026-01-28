// Package fts_compact implements a minimum index size optimized FTS driver.
// Uses Elias-Fano encoding for doc IDs, StreamVByte for frequencies, and FST for terms.
// Target: 5-10x smaller index, slight search slowdown acceptable.
package fts_compact

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/gob"
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
	"github.com/klauspost/compress/zstd"
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
	fineweb.Register("fts_compact", func(cfg fineweb.DriverConfig) (fineweb.Driver, error) {
		return New(cfg)
	})
}

// Driver implements minimum index size optimization.
type Driver struct {
	mu        sync.RWMutex
	index     *CompactIndex
	indexDir  string
	tokenizer *tokenizer.Vietnamese
	language  string
}

// CompactIndex uses compressed data structures.
type CompactIndex struct {
	// Term dictionary using FST
	TermDict *algo.FST

	// Posting lists with Elias-Fano encoding
	PostingLists map[string]*CompactPostingList

	// Document storage (compressed)
	Documents []compressedDoc

	// Metadata
	NumDocs   int
	AvgDocLen float64
	DocLens   []uint16 // Use uint16 to save space (max 65535 tokens)
}

// CompactPostingList uses Elias-Fano for doc IDs and StreamVByte for frequencies.
type CompactPostingList struct {
	DocIDs    *algo.EliasFano // Elias-Fano encoded doc IDs
	FreqData  []byte          // StreamVByte encoded frequencies
	DocFreq   int
	IDF       float32
}

type compressedDoc struct {
	ID       string
	URL      string
	TextData []byte // Zstd compressed text
	Dump     string
	Date     string
	Language string
	LangScore float64
}

// New creates a new compact driver.
func New(cfg fineweb.DriverConfig) (*Driver, error) {
	dataDir := cfg.DataDir
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, "data", "blueprints", "search", "fineweb-2")
	}

	indexDir := filepath.Join(dataDir, cfg.Language+".fts_compact")
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
		d.index = &CompactIndex{
			PostingLists: make(map[string]*CompactPostingList),
		}
	}

	return d, nil
}

// Name returns the driver name.
func (d *Driver) Name() string {
	return "fts_compact"
}

// Info returns driver metadata.
func (d *Driver) Info() *fineweb.DriverInfo {
	return &fineweb.DriverInfo{
		Name:        "fts_compact",
		Description: "Minimum size: Elias-Fano + StreamVByte + FST + Zstd",
		Features:    []string{"elias-fano", "streamvbyte", "fst", "zstd", "bm25"},
		External:    false,
	}
}

// Search performs BM25 search with compressed data structures.
func (d *Driver) Search(ctx context.Context, query string, limit, offset int) (*fineweb.SearchResult, error) {
	start := time.Now()

	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.index.NumDocs == 0 {
		return &fineweb.SearchResult{
			Documents: []fineweb.Document{},
			Duration:  time.Since(start),
			Method:    "fts_compact",
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

	// Collect posting lists
	type termData struct {
		docIDs []uint32
		freqs  []uint32
		idf    float32
	}
	termDatas := make([]termData, 0, len(queryTerms))

	for _, term := range queryTerms {
		if pl, exists := d.index.PostingLists[term]; exists {
			docIDs := pl.DocIDs.Decode()
			freqs := algo.StreamVByteDecode(pl.FreqData, pl.DocFreq)
			termDatas = append(termDatas, termData{
				docIDs: docIDs,
				freqs:  freqs,
				idf:    pl.IDF,
			})
		}
	}

	if len(termDatas) == 0 {
		return &fineweb.SearchResult{
			Documents: []fineweb.Document{},
			Duration:  time.Since(start),
			Method:    "fts_compact",
		}, nil
	}

	// Score documents using document-at-a-time with heap
	k1 := float32(1.2)
	b := float32(0.75)
	avgDL := float32(d.index.AvgDocLen)

	// Merge posting lists and score
	scores := make(map[uint32]float32)

	for _, td := range termDatas {
		for i, docID := range td.docIDs {
			tf := float32(td.freqs[i])
			dl := float32(d.index.DocLens[docID])

			// BM25 score
			tfNorm := (tf * (k1 + 1)) / (tf + k1*(1-b+b*dl/avgDL))
			scores[docID] += td.idf * tfNorm
		}
	}

	// Top-k selection using heap
	type scored struct {
		docID uint32
		score float32
	}
	results := make([]scored, 0, len(scores))
	for docID, score := range scores {
		results = append(results, scored{docID, score})
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	// Apply offset and limit
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
	decoder, _ := zstd.NewReader(nil)
	defer decoder.Close()

	for i, r := range results {
		cdoc := d.index.Documents[r.docID]

		// Decompress text
		text, _ := decoder.DecodeAll(cdoc.TextData, nil)

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
		Method:    "fts_compact",
		Total:     int64(len(scores)),
	}, nil
}

// Import indexes documents with maximum compression using parallel processing.
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
			encoder, _ := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedDefault))
			defer encoder.Close()

			for i := range docCh {
				doc := allDocs[i]
				var textBuf bytes.Buffer
				encoder.Reset(&textBuf)
				encoder.Write([]byte(doc.Text))
				encoder.Close()

				compressedDocs[i] = compressedDoc{
					ID:        doc.ID,
					URL:       doc.URL,
					TextData:  textBuf.Bytes(),
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
	d.index.NumDocs = len(allDocs)

	// Convert doc lengths to uint16
	d.index.DocLens = make([]uint16, len(docLens))
	totalLen := 0
	for i, dl := range docLens {
		if dl > 65535 {
			dl = 65535
		}
		d.index.DocLens[i] = uint16(dl)
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

	// Build compressed posting lists
	d.buildCompressedPostings(postings)

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

func (d *Driver) buildCompressedPostings(termPostings map[string][]posting) {
	n := float64(d.index.NumDocs)

	// Sort terms for FST
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
		pl   *CompactPostingList
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

				// Sort postings by doc number
				sort.Slice(postings, func(i, j int) bool {
					return postings[i].docNum < postings[j].docNum
				})

				// Extract doc IDs and frequencies
				docIDs := make([]uint32, len(postings))
				freqs := make([]uint32, len(postings))
				for i, p := range postings {
					docIDs[i] = p.docNum
					freqs[i] = uint32(p.freq)
				}

				// Compute IDF
				df := float64(len(postings))
				idf := float32(math.Log((n-df+0.5)/(df+0.5) + 1))

				// Create compressed posting list
				resultCh <- termResult{
					term: term,
					pl: &CompactPostingList{
						DocIDs:   algo.NewEliasFano(docIDs),
						FreqData: algo.StreamVByteEncode(freqs),
						DocFreq:  len(postings),
						IDF:      idf,
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
		d.index.PostingLists[result.term] = result.pl
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

	d.index = &CompactIndex{
		PostingLists: make(map[string]*CompactPostingList),
	}

	reader := bytes.NewReader(data)

	// Read metadata
	var numDocs int32
	var avgDocLen float64
	binary.Read(reader, binary.LittleEndian, &numDocs)
	binary.Read(reader, binary.LittleEndian, &avgDocLen)
	d.index.NumDocs = int(numDocs)
	d.index.AvgDocLen = avgDocLen

	// Read doc lengths
	d.index.DocLens = make([]uint16, numDocs)
	binary.Read(reader, binary.LittleEndian, d.index.DocLens)

	// Read documents (using gob for simplicity)
	gob.NewDecoder(reader).Decode(&d.index.Documents)

	// Read posting lists
	gob.NewDecoder(reader).Decode(&d.index.PostingLists)

	return nil
}

func (d *Driver) saveIndex() error {
	indexPath := filepath.Join(d.indexDir, "index.bin")

	var buf bytes.Buffer

	// Write metadata
	binary.Write(&buf, binary.LittleEndian, int32(d.index.NumDocs))
	binary.Write(&buf, binary.LittleEndian, d.index.AvgDocLen)

	// Write doc lengths
	binary.Write(&buf, binary.LittleEndian, d.index.DocLens)

	// Write documents
	gob.NewEncoder(&buf).Encode(d.index.Documents)

	// Write posting lists
	gob.NewEncoder(&buf).Encode(d.index.PostingLists)

	return os.WriteFile(indexPath, buf.Bytes(), 0644)
}

var (
	_ fineweb.Driver  = (*Driver)(nil)
	_ fineweb.Indexer = (*Driver)(nil)
	_ fineweb.Stats   = (*Driver)(nil)
)
