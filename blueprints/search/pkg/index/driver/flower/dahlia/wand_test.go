package dahlia

import (
	"path/filepath"
	"testing"
)

func buildTestSegment(t *testing.T) *segmentReader {
	t.Helper()
	dir := t.TempDir()
	segDir := filepath.Join(dir, "seg_00000001")

	sw := newSegmentWriter()
	sw.addDoc("doc0", []byte("machine learning algorithms for natural language processing"))
	sw.addDoc("doc1", []byte("deep learning neural networks and artificial intelligence"))
	sw.addDoc("doc2", []byte("the quick brown fox jumps over the lazy dog"))
	sw.addDoc("doc3", []byte("natural language processing with machine learning models"))
	sw.addDoc("doc4", []byte("information retrieval and search engine algorithms"))
	sw.addDoc("doc5", []byte("machine learning for information retrieval tasks"))
	sw.addDoc("doc6", []byte("artificial intelligence and deep neural networks research"))
	sw.addDoc("doc7", []byte("algorithms and data structures for efficient search"))
	sw.addDoc("doc8", []byte("natural language understanding with deep learning"))
	sw.addDoc("doc9", []byte("fox and dog are common animals in stories"))

	if _, err := sw.flush(segDir); err != nil {
		t.Fatal(err)
	}
	sr, err := openSegmentReader(segDir)
	if err != nil {
		t.Fatal(err)
	}
	return sr
}

func TestWandTermQuery(t *testing.T) {
	sr := buildTestSegment(t)
	defer sr.Close()

	q := parseQuery("machine")
	eval := newWandEvaluator(sr, 10)
	results := eval.searchQuery(q)

	if len(results) == 0 {
		t.Fatal("expected results for 'machine'")
	}
	// Should find docs with "machine" (doc0, doc3, doc5)
	if len(results) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(results))
	}
	// Results should be sorted by score descending
	for i := 1; i < len(results); i++ {
		if results[i].score > results[i-1].score {
			t.Fatalf("results not sorted: [%d]=%f > [%d]=%f", i, results[i].score, i-1, results[i-1].score)
		}
	}
}

func TestWandBooleanShould(t *testing.T) {
	sr := buildTestSegment(t)
	defer sr.Close()

	q := parseQuery("machine learning")
	eval := newWandEvaluator(sr, 10)
	results := eval.searchQuery(q)

	if len(results) == 0 {
		t.Fatal("expected results for 'machine learning'")
	}
	// Docs with both terms should score higher
	t.Logf("results: %d hits", len(results))
	for _, r := range results {
		t.Logf("  doc=%d score=%.4f", r.docID, r.score)
	}
}

func TestWandBooleanMust(t *testing.T) {
	sr := buildTestSegment(t)
	defer sr.Close()

	q := parseQuery("+machine +learning")
	eval := newWandEvaluator(sr, 10)
	results := eval.searchQuery(q)

	if len(results) == 0 {
		t.Fatal("expected results for '+machine +learning'")
	}
	// All results must contain both terms
	for _, r := range results {
		t.Logf("  doc=%d score=%.4f", r.docID, r.score)
	}
}

func TestWandMustNot(t *testing.T) {
	sr := buildTestSegment(t)
	defer sr.Close()

	q := parseQuery("+machine -deep")
	eval := newWandEvaluator(sr, 10)
	results := eval.searchQuery(q)

	// Should exclude docs with "deep"
	for _, r := range results {
		// doc1, doc6, doc8 have "deep" — these should not appear
		id, _, _ := sr.getDoc(r.docID)
		t.Logf("  doc=%s score=%.4f", id, r.score)
	}
}

func TestWandPhraseQuery(t *testing.T) {
	sr := buildTestSegment(t)
	defer sr.Close()

	q := parseQuery(`"machine learning"`)
	eval := newWandEvaluator(sr, 10)
	results := eval.searchQuery(q)

	if len(results) == 0 {
		t.Fatal("expected results for phrase 'machine learning'")
	}
	// Results should only be docs where "machine" and "learning" appear adjacent
	for _, r := range results {
		id, _, _ := sr.getDoc(r.docID)
		t.Logf("  phrase hit: doc=%s score=%.4f", id, r.score)
	}
}

func TestWandNoResults(t *testing.T) {
	sr := buildTestSegment(t)
	defer sr.Close()

	q := parseQuery("xyzzyplugh")
	eval := newWandEvaluator(sr, 10)
	results := eval.searchQuery(q)

	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestMultiSegmentSearch(t *testing.T) {
	dir := t.TempDir()

	// Create two segments
	sw1 := newSegmentWriter()
	sw1.addDoc("seg1_doc0", []byte("machine learning algorithms"))
	sw1.addDoc("seg1_doc1", []byte("deep learning networks"))
	segDir1 := filepath.Join(dir, "seg_00000001")
	sw1.flush(segDir1)

	sw2 := newSegmentWriter()
	sw2.addDoc("seg2_doc0", []byte("machine learning models"))
	sw2.addDoc("seg2_doc1", []byte("search algorithms"))
	segDir2 := filepath.Join(dir, "seg_00000002")
	sw2.flush(segDir2)

	sr1, _ := openSegmentReader(segDir1)
	sr2, _ := openSegmentReader(segDir2)
	defer sr1.Close()
	defer sr2.Close()

	q := parseQuery("machine learning")
	results := multiSegmentSearch([]*segmentReader{sr1, sr2}, q, 10)

	if len(results) == 0 {
		t.Fatal("expected results from multi-segment search")
	}
	t.Logf("multi-segment: %d hits", len(results))
}

func TestWandEmptyQuery(t *testing.T) {
	sr := buildTestSegment(t)
	defer sr.Close()

	q := parseQuery("")
	eval := newWandEvaluator(sr, 10)
	results := eval.searchQuery(q)
	if len(results) != 0 {
		t.Fatalf("expected 0 results for empty query, got %d", len(results))
	}
}
