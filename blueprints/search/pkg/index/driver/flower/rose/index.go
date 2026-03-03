package rose

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

func init() {
	index.Register("rose", func() index.Engine { return &roseEngine{} })
}

const (
	memFlushBytes = 64 << 20 // 64 MB flush threshold
	mergeMinSegs  = 4        // trigger merge when >= 4 segments at same tier
)

// segmentHandle is the in-memory view of one flushed segment.
type segmentHandle struct {
	path      string
	termDict  []termEntry
	postData  []byte
	docCount  uint32
	avgDocLen uint32
}

// roseEngine is the Rose FTS engine implementing index.Engine.
type roseEngine struct {
	dir string
	mu  sync.RWMutex

	// In-memory buffer: term → sorted list of unique docIDs (monotone ascending).
	mem      map[string][]uint32
	memBytes int64  // estimated memory usage
	memDocs  uint32 // number of docs in the current buffer

	// Corpus-wide statistics.
	totalDocs uint32
	totalLen  uint64 // sum of all document lengths in tokens

	// Segments (immutable, on-disk).
	segments []segmentHandle

	// nextSegID is a strictly monotone counter used to generate unique segment
	// file names.  It is never reset, so merge operations cannot produce names
	// that clash with previously deleted segment files.
	nextSegID uint32

	// Document store.
	docs *docStore

	// Background merge goroutine.
	mergeCh   chan struct{}
	mergeWg   sync.WaitGroup
	closeOnce sync.Once
	done      chan struct{}
}

// ---------------------------------------------------------------------------
// index.Engine interface
// ---------------------------------------------------------------------------

var _ index.Engine = (*roseEngine)(nil)

func (s *roseEngine) Name() string { return "rose" }

// Open initialises the engine at dir, loading any existing segments and the
// document store.  If dir does not exist it is created.
func (s *roseEngine) Open(ctx context.Context, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("rose open: mkdir %q: %w", dir, err)
	}
	s.dir = dir

	// Open (or create) the document store.
	docsPath := filepath.Join(dir, "rose.docs")
	ds, err := openDocStore(docsPath)
	if err != nil {
		return fmt.Errorf("rose open: docstore: %w", err)
	}
	s.docs = ds

	// Discover existing segments (sorted by name = chronological order).
	segs, err := filepath.Glob(filepath.Join(dir, "*.seg"))
	if err != nil {
		return fmt.Errorf("rose open: glob segments: %w", err)
	}
	sort.Strings(segs)

	s.segments = make([]segmentHandle, 0, len(segs))
	for _, path := range segs {
		td, pd, dc, al, err := openSegment(path)
		if err != nil {
			return fmt.Errorf("rose open: load segment %q: %w", path, err)
		}
		s.segments = append(s.segments, segmentHandle{
			path:      path,
			termDict:  td,
			postData:  pd,
			docCount:  dc,
			avgDocLen: al,
		})
		s.totalDocs += dc
	}

	// Reconstruct totalLen from per-segment metadata.  This is an approximation
	// (integer rounding during flush means avg*count != exact original sum), but
	// it is far better than leaving totalLen=0 which would corrupt BM25+ scores
	// after reopen.  No new file-format fields are required.
	s.totalLen = 0
	for _, seg := range s.segments {
		s.totalLen += uint64(seg.avgDocLen) * uint64(seg.docCount)
	}

	// Initialise nextSegID to max(parsed IDs from existing .seg files) + 1 so
	// that new segments always get names that are strictly greater than any
	// previously existing name — even after a merge that deleted some of them.
	s.nextSegID = 0
	for _, path := range segs {
		var id uint32
		fmt.Sscanf(filepath.Base(path), "seg_%08d.seg", &id)
		if id+1 > s.nextSegID {
			s.nextSegID = id + 1
		}
	}

	// Reconcile totalDocs with the docstore entry count.  The docstore is the
	// authoritative source; segments may lag by at most one unflushed buffer.
	// At open time the mem buffer is empty, so docstore len == totalDocs.
	if dc := uint32(len(ds.entries)); dc > s.totalDocs {
		s.totalDocs = dc
	}

	// Initialise the in-memory buffer.
	s.mem = make(map[string][]uint32)

	// Background merge goroutine.
	s.done = make(chan struct{})
	s.mergeCh = make(chan struct{}, 1)
	s.mergeWg.Add(1)
	go s.runMergeLoop()

	return nil
}

// Close flushes any remaining in-memory data and stops the merge goroutine.
func (s *roseEngine) Close() error {
	var closeErr error
	s.closeOnce.Do(func() {
		// Guard against Close() being called on an engine whose Open() failed
		// before s.done was initialised. Closing a nil channel panics.
		if s.done == nil {
			return
		}
		// Signal the merge goroutine to stop and wait for it.
		close(s.done)
		s.mergeWg.Wait()

		// Flush any remaining in-memory data.
		s.mu.Lock()
		flushErr := s.flushMem()
		s.mu.Unlock()
		if flushErr != nil {
			closeErr = flushErr
		}

		// Close the document store.
		if s.docs != nil {
			if err := s.docs.close(); err != nil && closeErr == nil {
				closeErr = err
			}
		}
	})
	return closeErr
}

// Stats returns corpus-level metadata.
func (s *roseEngine) Stats(ctx context.Context) (index.EngineStats, error) {
	s.mu.RLock()
	totalDocs := s.totalDocs
	s.mu.RUnlock()

	return index.EngineStats{
		DocCount:  int64(totalDocs),
		DiskBytes: index.DirSizeBytes(s.dir),
	}, nil
}

// Index ingests a batch of documents into the engine.
func (s *roseEngine) Index(ctx context.Context, docs []index.Document) error {
	if len(docs) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, doc := range docs {
		if err := s.indexOne(doc.DocID, string(doc.Text)); err != nil {
			return err
		}
	}
	return nil
}

// indexOne indexes a single document.  Must be called with s.mu held for writing.
func (s *roseEngine) indexOne(id string, body string) error {
	tokens := analyze(body)

	// Append to docstore.
	docIdx, err := s.docs.append(id, []byte(body))
	if err != nil {
		return fmt.Errorf("rose index %q: docstore: %w", id, err)
	}

	// Update corpus-wide statistics.
	s.totalDocs++
	s.totalLen += uint64(len(tokens))
	s.memDocs++

	// Add unique (term, docIdx) pairs to the in-memory buffer.
	// Because docIDs are monotonically increasing and we process one doc at a
	// time, deduplication is O(1): check that the last appended docID != docIdx.
	newTerms := int64(0)
	for _, term := range tokens {
		list := s.mem[term]
		if len(list) == 0 || list[len(list)-1] != docIdx {
			s.mem[term] = append(list, docIdx)
			newTerms++
		}
	}
	// 4 bytes per docID uint32.
	s.memBytes += newTerms * 4

	// Flush if the memory threshold is exceeded.
	if s.memBytes >= memFlushBytes {
		return s.flushMem()
	}
	return nil
}

// Search executes a full-text query and returns the top results.
func (s *roseEngine) Search(ctx context.Context, q index.Query) (index.Results, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 10
	}

	// Analyse the query.
	queryTerms := analyzeQuery(q.Text)
	if len(queryTerms) == 0 {
		return index.Results{}, nil
	}

	// Deduplicate query terms.
	seen := make(map[string]struct{}, len(queryTerms))
	unique := queryTerms[:0]
	for _, t := range queryTerms {
		if _, ok := seen[t]; !ok {
			seen[t] = struct{}{}
			unique = append(unique, t)
		}
	}
	queryTerms = unique

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Build one listCursor per query term by merging across all segments and
	// the in-memory buffer.
	cursors := make([]*listCursor, 0, len(queryTerms))
	for _, term := range queryTerms {
		var allDocIDs []uint32
		var allImpacts []uint8

		// Collect from each flushed segment.
		for _, seg := range s.segments {
			te, found := findTerm(seg.termDict, term)
			if !found {
				continue
			}
			docIDs, impacts, err := readPostings(seg.postData, te)
			if err != nil {
				return index.Results{}, fmt.Errorf("rose search: readPostings: %w", err)
			}
			allDocIDs = append(allDocIDs, docIDs...)
			allImpacts = append(allImpacts, impacts...)
		}

		// Collect from the in-memory buffer. Unflushed docs have not had
		// BM25+ quantisation applied yet, so we assign a mid-range impact
		// of 128. This produces slightly inconsistent scores vs. flushed
		// segments for the same term, but only until the next flush.
		if memList, ok := s.mem[term]; ok && len(memList) > 0 {
			for _, did := range memList {
				allDocIDs = append(allDocIDs, did)
				allImpacts = append(allImpacts, 128)
			}
		}

		if len(allDocIDs) == 0 {
			continue
		}

		cursors = append(cursors, newListCursor(allDocIDs, allImpacts))
	}

	if len(cursors) == 0 {
		return index.Results{}, nil
	}

	// Compute top-k via Block-Max WAND.
	topDocs := wandTopK(cursors, limit)

	// Build result hits.
	hits := make([]index.Hit, 0, len(topDocs))
	for _, sd := range topDocs {
		entry, err := s.docs.get(sd.docID)
		if err != nil {
			// The docID is out of range; skip gracefully.
			continue
		}
		hits = append(hits, index.Hit{
			DocID:   entry.externalID,
			Score:   float64(sd.score),
			Snippet: snippetFor(entry.text, queryTerms),
		})
	}

	// Total is set to len(hits) because WAND terminates after collecting
	// the top-k candidates — the full corpus match count is not computed.
	// This is explicitly allowed by the index.Searcher contract: "If not
	// available, drivers may set Total equal to len(Hits)."
	return index.Results{
		Hits:  hits,
		Total: len(hits),
	}, nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// flushMem writes the current in-memory buffer to a new segment file.
// Must be called with s.mu held for writing.
func (s *roseEngine) flushMem() error {
	if len(s.mem) == 0 {
		return nil
	}

	// Convert mem map[string][]uint32 to map[string][]memPosting.
	postings := make(map[string][]memPosting, len(s.mem))
	for term, docIDs := range s.mem {
		mp := make([]memPosting, len(docIDs))
		for i, id := range docIDs {
			mp[i] = memPosting{docID: id}
		}
		postings[term] = mp
	}

	avgDocLen := uint32(0)
	if s.totalDocs > 0 {
		avgDocLen = uint32(s.totalLen / uint64(s.totalDocs))
	}

	path := s.nextSegPath()
	if err := flushSegment(path, postings, s.memDocs, avgDocLen); err != nil {
		return fmt.Errorf("rose flushMem: %w", err)
	}

	// Load the newly written segment into memory.
	td, pd, dc, al, err := openSegment(path)
	if err != nil {
		return fmt.Errorf("rose flushMem: openSegment: %w", err)
	}
	s.segments = append(s.segments, segmentHandle{
		path:      path,
		termDict:  td,
		postData:  pd,
		docCount:  dc,
		avgDocLen: al,
	})

	// Reset the in-memory buffer.
	s.mem = make(map[string][]uint32)
	s.memBytes = 0
	s.memDocs = 0

	// Notify the merge goroutine (non-blocking).
	select {
	case s.mergeCh <- struct{}{}:
	default:
	}
	return nil
}

// nextSegPath returns the path for the next segment file.
// The returned name is strictly monotone — it never reuses a previously
// assigned name even after merges shrink s.segments.
// Must be called with s.mu held.
func (s *roseEngine) nextSegPath() string {
	id := s.nextSegID
	s.nextSegID++
	return filepath.Join(s.dir, fmt.Sprintf("seg_%08d.seg", id))
}

// findTerm binary-searches termDict for term and returns (entry, true) if found.
func findTerm(termDict []termEntry, term string) (termEntry, bool) {
	n := len(termDict)
	idx := sort.Search(n, func(i int) bool {
		return termDict[i].term >= term
	})
	if idx < n && termDict[idx].term == term {
		return termDict[idx], true
	}
	return termEntry{}, false
}
