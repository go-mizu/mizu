// Package fts_lowmem implements a low-memory FTS driver using memory-mapped indexes.
// Target: <1GB peak memory for any dataset size with 50k+ docs/sec indexing.
// Memory usage is controlled by OS page cache, not Go heap.
package fts_lowmem

import (
	"context"
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb"
	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/algo"
)

func init() {
	fineweb.Register("fts_lowmem", func(cfg fineweb.DriverConfig) (fineweb.Driver, error) {
		return New(cfg)
	})
}

// fastTokenize is an optimized tokenizer.
func fastTokenize(text string) map[string]int {
	termFreqs := make(map[string]int, 64)
	data := []byte(text)
	start := -1

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
					termFreqs[string(token)]++
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
			termFreqs[string(token)]++
		}
	}

	return termFreqs
}

func tokenizeQuery(query string) []string {
	terms := strings.FieldsFunc(query, func(r rune) bool {
		return r <= ' ' || (r >= '!' && r <= '/') || (r >= ':' && r <= '@') ||
			(r >= '[' && r <= '`') || (r >= '{' && r <= '~')
	})
	result := make([]string, 0, len(terms))
	for _, t := range terms {
		if len(t) > 0 && len(t) < 100 {
			result = append(result, strings.ToLower(t))
		}
	}
	return result
}

// Driver implements low-memory FTS using mmap.
type Driver struct {
	indexDir  string
	language  string
	mmapIndex *algo.MmapIndex

	// Document metadata stored separately (minimal memory)
	docIDs []string
}

// New creates a new low-memory driver.
func New(cfg fineweb.DriverConfig) (*Driver, error) {
	dataDir := cfg.DataDir
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, "data", "blueprints", "search", "fineweb-2")
	}

	indexDir := filepath.Join(dataDir, cfg.Language+".fts_lowmem")
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
	// Load doc IDs from separate file
	docIDPath := filepath.Join(d.indexDir, "docids.bin")
	data, err := os.ReadFile(docIDPath)
	if err != nil {
		return
	}

	// Parse doc IDs (length-prefixed strings)
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
		// Length-prefixed string
		length := len(id)
		f.Write([]byte{byte(length & 0xFF), byte(length >> 8)})
		f.Write([]byte(id))
	}

	return nil
}

// Name returns the driver name.
func (d *Driver) Name() string {
	return "fts_lowmem"
}

// Info returns driver metadata.
func (d *Driver) Info() *fineweb.DriverInfo {
	return &fineweb.DriverInfo{
		Name:        "fts_lowmem",
		Description: "Low memory: mmap index, <1GB peak memory",
		Features:    []string{"mmap", "low-memory", "bm25", "pipeline-indexing"},
		External:    false,
	}
}

// Search performs BM25 search using mmap'd index.
func (d *Driver) Search(ctx context.Context, query string, limit, offset int) (*fineweb.SearchResult, error) {
	start := time.Now()

	if d.mmapIndex == nil || d.mmapIndex.NumDocs == 0 {
		return &fineweb.SearchResult{
			Documents: []fineweb.Document{},
			Duration:  time.Since(start),
			Method:    "fts_lowmem",
		}, nil
	}

	// Tokenize query
	queryTerms := tokenizeQuery(query)

	// Search using mmap index
	results := d.mmapIndex.Search(queryTerms, limit+offset)

	// Apply offset
	if offset >= len(results) {
		results = nil
	} else {
		results = results[offset:]
	}
	if len(results) > limit {
		results = results[:limit]
	}

	// Convert to documents
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
		Method:    "fts_lowmem",
		Total:     int64(len(results)),
	}, nil
}

// Import indexes documents using pipeline indexing with mmap output.
// Memory usage stays under 1GB regardless of dataset size.
func (d *Driver) Import(ctx context.Context, docs iter.Seq2[fineweb.Document, error], progress fineweb.ProgressFunc) error {
	// Create temp directory for segments
	segmentDir := filepath.Join(d.indexDir, "segments")
	os.MkdirAll(segmentDir, 0755)
	defer os.RemoveAll(segmentDir)

	// Use PipelineIndexer
	indexer := algo.NewPipelineIndexer(segmentDir, fastTokenize)

	// Collect doc IDs (minimal memory: just strings)
	d.docIDs = make([]string, 0, 100000)
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

		docNum := uint32(len(d.docIDs))
		d.docIDs = append(d.docIDs, doc.ID)

		// Feed to pipeline indexer
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

	// Finish indexing and write mmap index
	indexPath := filepath.Join(d.indexDir, "index.mmap")

	// Close existing mmap index if any
	if d.mmapIndex != nil {
		d.mmapIndex.Close()
		d.mmapIndex = nil
	}

	var err error
	d.mmapIndex, err = indexer.FinishToMmap(indexPath)
	if err != nil {
		return fmt.Errorf("creating mmap index: %w", err)
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
	if d.mmapIndex == nil {
		return 0, nil
	}
	return int64(d.mmapIndex.NumDocs), nil
}

// Close releases resources.
func (d *Driver) Close() error {
	if d.mmapIndex != nil {
		return d.mmapIndex.Close()
	}
	return nil
}

var (
	_ fineweb.Driver  = (*Driver)(nil)
	_ fineweb.Indexer = (*Driver)(nil)
	_ fineweb.Stats   = (*Driver)(nil)
)
