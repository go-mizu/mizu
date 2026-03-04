package lotus

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

func init() {
	index.Register("lotus", func() index.Engine { return &Engine{} })
}

// Engine implements index.Engine for the lotus full-text search engine.
type Engine struct {
	dir      string
	meta     *indexMeta
	segments []*segmentReader
	merger   *mergeWorker
	mu       sync.RWMutex
}

func (e *Engine) Name() string { return "lotus" }

func (e *Engine) Open(_ context.Context, dir string) error {
	e.dir = dir
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("lotus: mkdir: %w", err)
	}

	meta, err := loadIndexMeta(dir)
	if err != nil {
		return fmt.Errorf("lotus: load meta: %w", err)
	}
	e.meta = meta

	// Open existing segments
	for _, name := range meta.Segments {
		r, err := openSegmentReader(filepath.Join(dir, name))
		if err != nil {
			e.closeSegments()
			return fmt.Errorf("lotus: open segment %s: %w", name, err)
		}
		e.segments = append(e.segments, r)
	}

	// Start background merge worker
	e.merger = newMergeWorker(dir)
	e.merger.start()

	return nil
}

func (e *Engine) Close() error {
	if e.merger != nil {
		e.merger.stop()
	}
	return e.closeSegments()
}

func (e *Engine) closeSegments() error {
	var firstErr error
	for _, s := range e.segments {
		if err := s.close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	e.segments = nil
	return firstErr
}

func (e *Engine) Stats(_ context.Context) (index.EngineStats, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var stats index.EngineStats
	stats.DocCount = int64(e.meta.DocCount)

	// Calculate disk usage
	entries, err := os.ReadDir(e.dir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				segDir := filepath.Join(e.dir, entry.Name())
				segEntries, _ := os.ReadDir(segDir)
				for _, se := range segEntries {
					info, _ := se.Info()
					if info != nil {
						stats.DiskBytes += info.Size()
					}
				}
			} else {
				info, _ := entry.Info()
				if info != nil {
					stats.DiskBytes += info.Size()
				}
			}
		}
	}
	return stats, nil
}

func (e *Engine) Index(_ context.Context, docs []index.Document) error {
	if len(docs) == 0 {
		return nil
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// Create a new segment for this batch
	segName := nextSegmentName(e.meta)
	segDir := filepath.Join(e.dir, segName)
	writer, err := newSegmentWriter(segDir)
	if err != nil {
		return fmt.Errorf("lotus: create segment: %w", err)
	}

	for _, doc := range docs {
		if err := writer.addDoc(doc.DocID, doc.Text); err != nil {
			return fmt.Errorf("lotus: add doc: %w", err)
		}
	}

	if err := writer.flush(); err != nil {
		return fmt.Errorf("lotus: flush segment: %w", err)
	}

	// Open the new segment for reading
	reader, err := openSegmentReader(segDir)
	if err != nil {
		return fmt.Errorf("lotus: open new segment: %w", err)
	}
	e.segments = append(e.segments, reader)

	// Update index metadata
	e.meta.Segments = append(e.meta.Segments, segName)
	e.meta.DocCount += uint64(len(docs))

	// Recompute avgDocLen
	totalTokens := float64(0)
	for _, seg := range e.segments {
		totalTokens += seg.meta.AvgDocLen * float64(seg.meta.DocCount)
	}
	if e.meta.DocCount > 0 {
		e.meta.AvgDocLen = totalTokens / float64(e.meta.DocCount)
	}

	return saveIndexMeta(e.dir, e.meta)
}

func (e *Engine) Search(_ context.Context, q index.Query) (index.Results, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if len(e.segments) == 0 {
		return index.Results{}, nil
	}

	k := q.Limit
	if k <= 0 {
		k = 10
	}

	parsed := parseQuery(q.Text)
	scored := multiSegmentSearch(e.segments, parsed, k+q.Offset)

	// Apply offset
	start := q.Offset
	if start > len(scored) {
		start = len(scored)
	}
	end := start + k
	if end > len(scored) {
		end = len(scored)
	}
	page := scored[start:end]

	// Resolve stored documents for hits
	hits := make([]index.Hit, 0, len(page))
	for _, sd := range page {
		docID, text := e.resolveDoc(sd.segIdx, sd.docID)
		snippet := ""
		if len(text) > 200 {
			snippet = string(text[:200]) + "..."
		} else {
			snippet = string(text)
		}
		hits = append(hits, index.Hit{
			DocID:   docID,
			Score:   sd.score,
			Snippet: snippet,
		})
	}

	return index.Results{
		Hits:  hits,
		Total: len(scored),
	}, nil
}

// resolveDoc finds the stored document by segment index and local docID.
func (e *Engine) resolveDoc(segIdx int, localDocID uint32) (string, []byte) {
	if segIdx >= 0 && segIdx < len(e.segments) {
		id, text, err := e.segments[segIdx].getDoc(localDocID)
		if err == nil {
			return id, text
		}
	}
	return fmt.Sprintf("doc_%d", localDocID), nil
}

// Finalize implements index.Finalizer — forces a compaction merge.
func (e *Engine) Finalize(_ context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if len(e.segments) <= 1 {
		return nil
	}

	// Close current segments before merge
	e.closeSegments()

	if err := mergeAllSegments(e.dir); err != nil {
		return fmt.Errorf("lotus: finalize merge: %w", err)
	}

	// Reload
	meta, err := loadIndexMeta(e.dir)
	if err != nil {
		return err
	}
	e.meta = meta
	for _, name := range meta.Segments {
		r, err := openSegmentReader(filepath.Join(e.dir, name))
		if err != nil {
			return err
		}
		e.segments = append(e.segments, r)
	}
	return nil
}
