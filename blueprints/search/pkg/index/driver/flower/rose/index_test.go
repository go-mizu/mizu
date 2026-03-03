package rose

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// newEngine opens a roseEngine in a temp directory and registers cleanup.
func newEngine(t *testing.T) *roseEngine {
	t.Helper()
	dir := t.TempDir()
	e := &roseEngine{}
	if err := e.Open(context.Background(), dir); err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = e.Close() })
	return e
}

// indexDocs is a convenience wrapper for indexing a batch of docs.
func indexDocs(t *testing.T, e *roseEngine, docs []index.Document) {
	t.Helper()
	if err := e.Index(context.Background(), docs); err != nil {
		t.Fatalf("Index: %v", err)
	}
}

// search is a convenience wrapper.
func search(t *testing.T, e *roseEngine, q string, limit int) index.Results {
	t.Helper()
	res, err := e.Search(context.Background(), index.Query{Text: q, Limit: limit})
	if err != nil {
		t.Fatalf("Search(%q): %v", q, err)
	}
	return res
}

// hitIDs extracts DocID strings from a Results.Hits slice.
func hitIDs(r index.Results) []string {
	ids := make([]string, len(r.Hits))
	for i, h := range r.Hits {
		ids[i] = h.DocID
	}
	return ids
}

// containsID reports whether DocID id appears in hits.
func containsID(r index.Results, id string) bool {
	for _, h := range r.Hits {
		if h.DocID == id {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Test 1: TestRoseEngine_IndexAndSearch
// ---------------------------------------------------------------------------

func TestRoseEngine_IndexAndSearch(t *testing.T) {
	e := newEngine(t)

	docs := []index.Document{
		{DocID: "doc1", Text: []byte("the quick brown fox jumps over the lazy dog")},
		{DocID: "doc2", Text: []byte("a quick brown dog outpaces a lazy fox")},
		{DocID: "doc3", Text: []byte("machine learning and artificial intelligence research")},
		{DocID: "doc4", Text: []byte("deep learning neural networks for computer vision")},
		{DocID: "doc5", Text: []byte("the fox ran quickly through the forest")},
	}
	indexDocs(t, e, docs)

	// "fox" appears in doc1, doc2, doc5.
	res := search(t, e, "fox", 10)
	if len(res.Hits) == 0 {
		t.Fatal("expected at least 1 hit for 'fox', got 0")
	}
	for _, h := range res.Hits {
		found := false
		for _, want := range []string{"doc1", "doc2", "doc5"} {
			if h.DocID == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("unexpected hit DocID %q for query 'fox'", h.DocID)
		}
	}

	// "learning" appears in doc3 and doc4.
	res2 := search(t, e, "learning", 10)
	if !containsID(res2, "doc3") || !containsID(res2, "doc4") {
		t.Errorf("expected doc3 and doc4 in 'learning' results, got %v", hitIDs(res2))
	}

	// Scores must be > 0.
	for _, h := range res.Hits {
		if h.Score <= 0 {
			t.Errorf("hit %q has non-positive score %f", h.DocID, h.Score)
		}
	}
}

// ---------------------------------------------------------------------------
// Test 2: TestRoseEngine_TopK
// ---------------------------------------------------------------------------

func TestRoseEngine_TopK(t *testing.T) {
	e := newEngine(t)

	// Index 20 docs; "alpha" appears with varying frequency via repetition.
	var docs []index.Document
	for i := 0; i < 20; i++ {
		var b strings.Builder
		// doc0 gets 1 mention of alpha, doc1 gets 2, …, doc4 gets 5.
		// docs 5-19 mention only "beta".
		if i < 5 {
			for j := 0; j <= i; j++ {
				b.WriteString("alpha ")
			}
		} else {
			b.WriteString("beta gamma delta")
		}
		docs = append(docs, index.Document{
			DocID: strings.Repeat("x", i+1), // unique IDs of different lengths
			Text:  []byte(b.String()),
		})
	}
	indexDocs(t, e, docs)

	// Request top 3 results for "alpha".
	res := search(t, e, "alpha", 3)
	if len(res.Hits) > 3 {
		t.Errorf("expected at most 3 hits, got %d", len(res.Hits))
	}
	if len(res.Hits) == 0 {
		t.Fatal("expected at least 1 hit for 'alpha', got 0")
	}

	// All returned hits must actually contain "alpha".
	alphaDocs := map[string]bool{}
	for i := 0; i < 5; i++ {
		alphaDocs[strings.Repeat("x", i+1)] = true
	}
	for _, h := range res.Hits {
		if !alphaDocs[h.DocID] {
			t.Errorf("unexpected DocID %q in 'alpha' top-3", h.DocID)
		}
	}

	// Results must be sorted descending by score.
	for i := 1; i < len(res.Hits); i++ {
		if res.Hits[i].Score > res.Hits[i-1].Score {
			t.Errorf("results not sorted descending: hits[%d].Score=%f > hits[%d].Score=%f",
				i, res.Hits[i].Score, i-1, res.Hits[i-1].Score)
		}
	}
}

// ---------------------------------------------------------------------------
// Test 3: TestRoseEngine_Reopen
// ---------------------------------------------------------------------------

func TestRoseEngine_Reopen(t *testing.T) {
	dir := t.TempDir()

	// First session: index some documents and flush them to disk.
	e1 := &roseEngine{}
	if err := e1.Open(context.Background(), dir); err != nil {
		t.Fatalf("Open (session 1): %v", err)
	}
	docs := []index.Document{
		{DocID: "a", Text: []byte("persistent storage works correctly")},
		{DocID: "b", Text: []byte("persistent data survives restarts")},
		{DocID: "c", Text: []byte("transient memory is volatile")},
	}
	if err := e1.Index(context.Background(), docs); err != nil {
		t.Fatalf("Index: %v", err)
	}
	// Force flush so data lands on disk.
	e1.mu.Lock()
	if err := e1.flushMem(); err != nil {
		e1.mu.Unlock()
		t.Fatalf("flushMem: %v", err)
	}
	e1.mu.Unlock()
	if err := e1.Close(); err != nil {
		t.Fatalf("Close (session 1): %v", err)
	}

	// Second session: reopen and search.
	e2 := &roseEngine{}
	if err := e2.Open(context.Background(), dir); err != nil {
		t.Fatalf("Open (session 2): %v", err)
	}
	defer e2.Close()

	res := search(t, e2, "persistent", 10)
	if len(res.Hits) == 0 {
		t.Fatal("expected hits after reopen, got 0")
	}
	if !containsID(res, "a") || !containsID(res, "b") {
		t.Errorf("expected doc 'a' and 'b' after reopen, got %v", hitIDs(res))
	}
}

// ---------------------------------------------------------------------------
// Test 4: TestRoseEngine_Stats
// ---------------------------------------------------------------------------

func TestRoseEngine_Stats(t *testing.T) {
	e := newEngine(t)

	// Index 7 documents.
	var docs []index.Document
	for i := 0; i < 7; i++ {
		docs = append(docs, index.Document{
			DocID: strings.Repeat("d", i+1),
			Text:  []byte("statistics test document number"),
		})
	}
	indexDocs(t, e, docs)

	stats, err := e.Stats(context.Background())
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.DocCount != 7 {
		t.Errorf("DocCount: got %d, want 7", stats.DocCount)
	}
}

// ---------------------------------------------------------------------------
// Test 5: TestRoseEngine_SegmentFlush
// ---------------------------------------------------------------------------

func TestRoseEngine_SegmentFlush(t *testing.T) {
	dir := t.TempDir()
	e := &roseEngine{}
	if err := e.Open(context.Background(), dir); err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer e.Close()

	// Index a few documents.
	docs := []index.Document{
		{DocID: "f1", Text: []byte("flush segment test one")},
		{DocID: "f2", Text: []byte("flush segment test two")},
	}
	indexDocs(t, e, docs)

	// Explicitly trigger a flush.
	e.mu.Lock()
	if err := e.flushMem(); err != nil {
		e.mu.Unlock()
		t.Fatalf("flushMem: %v", err)
	}
	e.mu.Unlock()

	// Verify a .seg file exists in the directory.
	segs, err := filepath.Glob(filepath.Join(dir, "*.seg"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(segs) == 0 {
		t.Fatal("expected at least one .seg file after flushMem, found none")
	}

	// After flush, the in-memory buffer should be empty.
	e.mu.RLock()
	memLen := len(e.mem)
	e.mu.RUnlock()
	if memLen != 0 {
		t.Errorf("mem not empty after flushMem: %d entries remain", memLen)
	}

	// Search should still work.
	res := search(t, e, "flush", 10)
	if len(res.Hits) == 0 {
		t.Error("expected hits after flush, got 0")
	}
}

// ---------------------------------------------------------------------------
// Test 6: TestRoseEngine_MultiSegmentSearch
// ---------------------------------------------------------------------------

func TestRoseEngine_MultiSegmentSearch(t *testing.T) {
	dir := t.TempDir()
	e := &roseEngine{}
	if err := e.Open(context.Background(), dir); err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer e.Close()

	// Batch 1.
	indexDocs(t, e, []index.Document{
		{DocID: "s1", Text: []byte("orange fruit tropical vitamin")},
		{DocID: "s2", Text: []byte("mango tropical fruit delicious")},
	})
	e.mu.Lock()
	if err := e.flushMem(); err != nil {
		e.mu.Unlock()
		t.Fatalf("flushMem (1): %v", err)
	}
	e.mu.Unlock()

	// Batch 2.
	indexDocs(t, e, []index.Document{
		{DocID: "s3", Text: []byte("tropical islands palm trees vacation")},
		{DocID: "s4", Text: []byte("winter snow cold mountains skiing")},
	})
	e.mu.Lock()
	if err := e.flushMem(); err != nil {
		e.mu.Unlock()
		t.Fatalf("flushMem (2): %v", err)
	}
	e.mu.Unlock()

	// Verify we have at least 2 segments.
	e.mu.RLock()
	nSegs := len(e.segments)
	e.mu.RUnlock()
	if nSegs < 2 {
		t.Fatalf("expected >= 2 segments, got %d", nSegs)
	}

	// "tropical" spans both segments (s1, s2, s3).
	res := search(t, e, "tropical", 10)
	if len(res.Hits) < 3 {
		t.Errorf("expected >= 3 hits for 'tropical' across 2 segments, got %d: %v",
			len(res.Hits), hitIDs(res))
	}
	for _, want := range []string{"s1", "s2", "s3"} {
		if !containsID(res, want) {
			t.Errorf("DocID %q missing from multi-segment search results", want)
		}
	}

	// "skiing" is only in the second segment.
	res2 := search(t, e, "skiing", 10)
	if !containsID(res2, "s4") {
		t.Errorf("DocID 's4' missing from skiing results: %v", hitIDs(res2))
	}
}
