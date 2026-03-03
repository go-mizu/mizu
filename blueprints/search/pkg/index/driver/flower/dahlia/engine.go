package dahlia

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

func init() {
	index.Register("dahlia", func() index.Engine { return &Engine{} })
}

// Engine implements index.Engine and index.Finalizer using a tantivy-style
// segment architecture with BP128 compression, FST term dictionaries,
// and Block-Max WAND scoring.
type Engine struct {
	dir      string
	meta     *indexMeta
	segments []*segmentReader
	merger   *mergeWorker
	mu       sync.RWMutex

	// In-memory buffer for memory-bounded indexing
	writer *segmentWriter

	// Background flush coordination
	flushWg sync.WaitGroup
}

func (e *Engine) Name() string { return "dahlia" }

func (e *Engine) Open(_ context.Context, dir string) error {
	e.dir = dir
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	meta, err := loadIndexMeta(dir)
	if err != nil {
		return fmt.Errorf("load meta: %w", err)
	}
	e.meta = meta

	// Open existing segments
	for _, segName := range meta.Segments {
		segDir := filepath.Join(dir, segName)
		sr, err := openSegmentReader(segDir)
		if err != nil {
			e.closeSegments()
			return fmt.Errorf("open segment %s: %w", segName, err)
		}
		e.segments = append(e.segments, sr)
	}

	// Start background merge worker
	e.merger = newMergeWorker(dir, &e.mu, &e.meta, &e.segments)
	e.merger.start()

	return nil
}

func (e *Engine) Close() error {
	if e.merger != nil {
		e.merger.stop()
	}

	// Wait for background flushes
	e.flushWg.Wait()

	// Flush any remaining buffered docs synchronously
	e.mu.Lock()
	if e.writer != nil && e.writer.docCount > 0 {
		e.flushWriterSync()
	}
	e.mu.Unlock()

	e.closeSegments()
	return nil
}

func (e *Engine) closeSegments() {
	for _, seg := range e.segments {
		seg.Close()
	}
	e.segments = nil
}

func (e *Engine) Stats(_ context.Context) (index.EngineStats, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var diskBytes int64
	for _, segName := range e.meta.Segments {
		segDir := filepath.Join(e.dir, segName)
		entries, _ := os.ReadDir(segDir)
		for _, entry := range entries {
			info, err := entry.Info()
			if err == nil {
				diskBytes += info.Size()
			}
		}
	}

	return index.EngineStats{
		DocCount:  int64(e.meta.DocCount),
		DiskBytes: diskBytes,
	}, nil
}

func (e *Engine) Index(_ context.Context, docs []index.Document) error {
	e.mu.Lock()

	if e.writer == nil {
		e.writer = newSegmentWriter()
	}

	for _, doc := range docs {
		e.writer.addDoc(doc.DocID, doc.Text)

		if e.writer.estimatedMemory() >= memoryFlushBytes {
			// Prepare flush job
			sw := e.writer
			segSeq := e.meta.NextSegSeq
			e.meta.NextSegSeq++
			segName := fmt.Sprintf(segDirFmt, segSeq)
			segDir := filepath.Join(e.dir, segName)
			e.writer = newSegmentWriter()

			// Launch background flush — release lock so flush goroutine
			// can acquire it later to update segments/meta
			e.flushWg.Add(1)
			go e.bgFlush(sw, segName, segDir)
		}
	}

	e.mu.Unlock()
	return nil
}

// bgFlush flushes a writer to disk and registers the new segment.
func (e *Engine) bgFlush(sw *segmentWriter, segName, segDir string) {
	defer e.flushWg.Done()

	segMeta, err := sw.flush(segDir)
	if err != nil {
		return
	}
	sr, err := openSegmentReader(segDir)
	if err != nil {
		os.RemoveAll(segDir)
		return
	}

	e.mu.Lock()
	e.segments = append(e.segments, sr)
	e.meta.Segments = append(e.meta.Segments, segName)
	e.meta.DocCount += uint64(segMeta.DocCount)
	var totalTokens float64
	for _, seg := range e.segments {
		totalTokens += seg.meta.AvgDocLen * float64(seg.meta.DocCount)
	}
	if e.meta.DocCount > 0 {
		e.meta.AvgDocLen = totalTokens / float64(e.meta.DocCount)
	}
	saveIndexMeta(e.dir, e.meta)
	e.mu.Unlock()
}

// flushWriterSync flushes the current writer synchronously. Caller must hold e.mu.
func (e *Engine) flushWriterSync() error {
	if e.writer == nil || e.writer.docCount == 0 {
		return nil
	}

	segSeq := e.meta.NextSegSeq
	e.meta.NextSegSeq++
	segName := fmt.Sprintf(segDirFmt, segSeq)
	segDir := filepath.Join(e.dir, segName)

	segMeta, err := e.writer.flush(segDir)
	if err != nil {
		return fmt.Errorf("flush segment: %w", err)
	}

	sr, err := openSegmentReader(segDir)
	if err != nil {
		return fmt.Errorf("open new segment: %w", err)
	}

	e.segments = append(e.segments, sr)
	e.meta.Segments = append(e.meta.Segments, segName)
	e.meta.DocCount += uint64(segMeta.DocCount)

	var totalTokens float64
	for _, seg := range e.segments {
		totalTokens += seg.meta.AvgDocLen * float64(seg.meta.DocCount)
	}
	if e.meta.DocCount > 0 {
		e.meta.AvgDocLen = totalTokens / float64(e.meta.DocCount)
	}

	e.writer = nil
	return saveIndexMeta(e.dir, e.meta)
}

func (e *Engine) Search(_ context.Context, q index.Query) (index.Results, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if len(e.segments) == 0 {
		return index.Results{}, nil
	}

	limit := q.Limit
	if limit <= 0 {
		limit = 10
	}

	parsed := parseQuery(q.Text)
	scored := multiSegmentSearch(e.segments, parsed, limit+q.Offset)

	// Apply offset
	if q.Offset > 0 {
		if q.Offset >= len(scored) {
			return index.Results{Total: len(scored)}, nil
		}
		scored = scored[q.Offset:]
	}
	if len(scored) > limit {
		scored = scored[:limit]
	}

	// Resolve stored docs for hits
	hits := make([]index.Hit, 0, len(scored))
	for _, sd := range scored {
		hit := e.resolveHit(sd)
		hits = append(hits, hit)
	}

	return index.Results{
		Hits:  hits,
		Total: len(scored),
	}, nil
}

func (e *Engine) resolveHit(sd scoredDoc) index.Hit {
	for _, seg := range e.segments {
		if sd.docID < seg.meta.DocCount {
			id, text, err := seg.getDoc(sd.docID)
			if err == nil {
				snippet := string(text)
				if len(snippet) > 200 {
					snippet = snippet[:200] + "..."
				}
				return index.Hit{
					DocID:   id,
					Score:   sd.score,
					Snippet: snippet,
				}
			}
		}
	}
	return index.Hit{
		DocID: fmt.Sprintf("doc_%d", sd.docID),
		Score: sd.score,
	}
}

// Finalize implements index.Finalizer. Forces a merge of all segments
// and flushes any buffered documents.
func (e *Engine) Finalize(_ context.Context) error {
	// Flush buffered docs synchronously
	e.mu.Lock()
	if e.writer != nil && e.writer.docCount > 0 {
		if err := e.flushWriterSync(); err != nil {
			e.mu.Unlock()
			return err
		}
	}
	e.mu.Unlock()

	// Wait for all background flushes before merge
	e.flushWg.Wait()

	return forceMerge(e.dir, &e.mu, &e.meta, &e.segments)
}
