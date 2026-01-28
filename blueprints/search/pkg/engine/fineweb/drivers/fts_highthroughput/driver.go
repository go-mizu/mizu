// Package fts_highthroughput implements an ultra-high-throughput FTS driver.
// Target: 1M+ docs/sec indexing with <5GB peak memory.
// Uses sharded accumulators, arena allocation, and inline tokenization.
package fts_highthroughput

import (
	"context"
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb"
	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/algo"
)

func init() {
	fineweb.Register("fts_highthroughput", func(cfg fineweb.DriverConfig) (fineweb.Driver, error) {
		return New(cfg)
	})
}

// Driver implements ultra-high-throughput FTS.
type Driver struct {
	indexDir      string
	language      string
	mmapIndex     *algo.MmapIndex      // Legacy mmap index
	segmentedIdx  *algo.SegmentedIndex // New segment-based index (no merge)
	docIDs        []string
}

// New creates a new high-throughput driver.
func New(cfg fineweb.DriverConfig) (*Driver, error) {
	dataDir := cfg.DataDir
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, "data", "blueprints", "search", "fineweb-2")
	}

	indexDir := filepath.Join(dataDir, cfg.Language+".fts_highthroughput")
	if err := os.MkdirAll(indexDir, 0755); err != nil {
		return nil, fmt.Errorf("creating index directory: %w", err)
	}

	d := &Driver{
		indexDir: indexDir,
		language: cfg.Language,
	}

	// Try to load existing mmap index
	indexPath := filepath.Join(indexDir, "index.mmap")
	if idx, err := algo.OpenMmapIndex(indexPath); err == nil {
		d.mmapIndex = idx
		d.loadDocIDs()
	}

	return d, nil
}

func (d *Driver) loadDocIDs() {
	docIDPath := filepath.Join(d.indexDir, "docids.bin")
	data, err := os.ReadFile(docIDPath)
	if err != nil {
		return
	}

	d.docIDs = make([]string, 0, d.mmapIndex.NumDocs)
	offset := 0
	for offset < len(data) {
		if offset+2 > len(data) {
			break
		}
		length := int(data[offset]) | int(data[offset+1])<<8
		offset += 2
		if offset+length > len(data) {
			break
		}
		d.docIDs = append(d.docIDs, string(data[offset:offset+length]))
		offset += length
	}
}

func (d *Driver) saveDocIDs() error {
	docIDPath := filepath.Join(d.indexDir, "docids.bin")
	f, err := os.Create(docIDPath)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, id := range d.docIDs {
		length := len(id)
		f.Write([]byte{byte(length & 0xFF), byte(length >> 8)})
		f.Write([]byte(id))
	}

	return nil
}

// Name returns the driver name.
func (d *Driver) Name() string {
	return "fts_highthroughput"
}

// Info returns driver metadata.
func (d *Driver) Info() *fineweb.DriverInfo {
	return &fineweb.DriverInfo{
		Name:        "fts_highthroughput",
		Description: "Ultra-high-throughput: 1M+ docs/sec, <5GB memory",
		Features:    []string{"ultra-fast", "sharded-accumulator", "mmap", "bm25"},
		External:    false,
	}
}

// Search performs BM25 search across segments.
func (d *Driver) Search(ctx context.Context, query string, limit, offset int) (*fineweb.SearchResult, error) {
	start := time.Now()

	// Try segmented index first (new approach)
	if d.segmentedIdx != nil && d.segmentedIdx.NumDocs() > 0 {
		queryTerms := tokenizeQuery(query)
		results := d.segmentedIdx.Search(queryTerms, limit+offset)

		if offset >= len(results) {
			results = nil
		} else {
			results = results[offset:]
		}
		if len(results) > limit {
			results = results[:limit]
		}

		docs := make([]fineweb.Document, len(results))
		for i, r := range results {
			docID := ""
			if int(r.DocID) < len(d.docIDs) {
				docID = d.docIDs[r.DocID]
			}
			docs[i] = fineweb.Document{
				ID:    docID,
				Score: float64(r.Score),
			}
		}

		return &fineweb.SearchResult{
			Documents: docs,
			Duration:  time.Since(start),
			Method:    "fts_highthroughput",
			Total:     int64(len(results)),
		}, nil
	}

	// Fall back to mmap index (legacy)
	if d.mmapIndex == nil || d.mmapIndex.NumDocs == 0 {
		return &fineweb.SearchResult{
			Documents: []fineweb.Document{},
			Duration:  time.Since(start),
			Method:    "fts_highthroughput",
		}, nil
	}

	queryTerms := tokenizeQuery(query)
	results := d.mmapIndex.Search(queryTerms, limit+offset)

	if offset >= len(results) {
		results = nil
	} else {
		results = results[offset:]
	}
	if len(results) > limit {
		results = results[:limit]
	}

	docs := make([]fineweb.Document, len(results))
	for i, r := range results {
		docID := ""
		if int(r.DocID) < len(d.docIDs) {
			docID = d.docIDs[r.DocID]
		}
		docs[i] = fineweb.Document{
			ID:    docID,
			Score: float64(r.Score),
		}
	}

	return &fineweb.SearchResult{
		Documents: docs,
		Duration:  time.Since(start),
		Method:    "fts_highthroughput",
		Total:     int64(len(results)),
	}, nil
}

func tokenizeQuery(query string) []string {
	var terms []string
	var start int = -1
	data := []byte(query)

	for i := 0; i < len(data); i++ {
		c := data[i]
		isDelim := c <= ' ' || (c >= '!' && c <= '/') || (c >= ':' && c <= '@') ||
			(c >= '[' && c <= '`') || (c >= '{' && c <= '~')

		if isDelim {
			if start >= 0 {
				token := data[start:i]
				if len(token) < 100 {
					for j := 0; j < len(token); j++ {
						if token[j] >= 'A' && token[j] <= 'Z' {
							token[j] += 32
						}
					}
					terms = append(terms, string(token))
				}
				start = -1
			}
		} else if start < 0 {
			start = i
		}
	}

	if start >= 0 {
		token := data[start:]
		if len(token) < 100 {
			for j := 0; j < len(token); j++ {
				if token[j] >= 'A' && token[j] <= 'Z' {
					token[j] += 32
				}
			}
			terms = append(terms, string(token))
		}
	}

	return terms
}

// fastTokenize uses the ultra-fast tokenizer with unsafe.String optimization.
func fastTokenize(text string) map[string]int {
	return algo.UltraFastTokenize(text)
}

// Import indexes documents using NoMergeIndexer (segment-based, no merge phase).
// This achieves high throughput by skipping the expensive k-way merge.
func (d *Driver) Import(ctx context.Context, docs iter.Seq2[fineweb.Document, error], progress fineweb.ProgressFunc) error {
	// Create segments directory (persistent - not deleted)
	segmentDir := filepath.Join(d.indexDir, "segments")
	os.MkdirAll(segmentDir, 0755)

	// Close existing indexes
	if d.mmapIndex != nil {
		d.mmapIndex.Close()
		d.mmapIndex = nil
	}
	if d.segmentedIdx != nil {
		d.segmentedIdx.Close()
		d.segmentedIdx = nil
	}

	// Use NoMergeIndexer - segments are kept separate, searched in parallel
	indexer := algo.NewNoMergeIndexer(segmentDir, fastTokenize, 100000) // 100k docs per segment

	// Collect doc IDs
	d.docIDs = make([]string, 0, 3000000)
	var imported int64

	for doc, err := range docs {
		if err != nil {
			return fmt.Errorf("reading document: %w", err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		docNum := uint32(len(d.docIDs))
		d.docIDs = append(d.docIDs, doc.ID)

		// Feed to indexer
		indexer.Add(docNum, doc.Text)

		imported++

		// Report progress every 10k docs
		if imported%10000 == 0 && progress != nil {
			progress(imported, 0)
		}
	}

	// Finalize - returns SegmentedIndex (no merge!)
	var err error
	d.segmentedIdx, err = indexer.Finish()
	if err != nil {
		return fmt.Errorf("creating segmented index: %w", err)
	}

	// Save doc IDs
	if err := d.saveDocIDs(); err != nil {
		return fmt.Errorf("saving doc IDs: %w", err)
	}

	if progress != nil {
		progress(imported, imported)
	}

	return nil
}

// Count returns document count.
func (d *Driver) Count(ctx context.Context) (int64, error) {
	if d.segmentedIdx != nil {
		return int64(d.segmentedIdx.NumDocs()), nil
	}
	if d.mmapIndex != nil {
		return int64(d.mmapIndex.NumDocs), nil
	}
	return 0, nil
}

// Close releases resources.
func (d *Driver) Close() error {
	if d.segmentedIdx != nil {
		d.segmentedIdx.Close()
	}
	if d.mmapIndex != nil {
		d.mmapIndex.Close()
	}
	return nil
}

var (
	_ fineweb.Driver  = (*Driver)(nil)
	_ fineweb.Indexer = (*Driver)(nil)
	_ fineweb.Stats   = (*Driver)(nil)
)
