package rose

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// listCursor tests
// ---------------------------------------------------------------------------

func TestListCursor_DocID(t *testing.T) {
	docIDs := []uint32{1, 5, 10}
	impacts := []uint8{10, 20, 30}
	c := newListCursor(docIDs, impacts)

	// New cursor returns the first docID.
	if got := c.docID(); got != 1 {
		t.Errorf("docID() = %d, want 1", got)
	}

	// Exhaust the cursor by advancing past the end.
	c.advance(math.MaxUint32)

	// Exhausted cursor returns MaxUint32.
	if got := c.docID(); got != math.MaxUint32 {
		t.Errorf("exhausted docID() = %d, want MaxUint32", got)
	}
}

func TestListCursor_Advance(t *testing.T) {
	docIDs := []uint32{2, 5, 9, 15, 20}
	impacts := []uint8{1, 2, 3, 4, 5}
	c := newListCursor(docIDs, impacts)

	// Advance to an exact match.
	c.advance(9)
	if got := c.docID(); got != 9 {
		t.Errorf("advance(9): docID() = %d, want 9", got)
	}

	// Advance to a target between two existing docIDs — should land on the next one.
	c.advance(12) // 12 is between 9 and 15
	if got := c.docID(); got != 15 {
		t.Errorf("advance(12): docID() = %d, want 15", got)
	}

	// Advance past the end — cursor should be exhausted.
	c.advance(100)
	if got := c.docID(); got != math.MaxUint32 {
		t.Errorf("advance(100) past end: docID() = %d, want MaxUint32", got)
	}

	// Advancing an exhausted cursor is a no-op (should not panic).
	c.advance(50)
	if got := c.docID(); got != math.MaxUint32 {
		t.Errorf("advance on exhausted cursor: docID() = %d, want MaxUint32", got)
	}
}

func TestListCursor_MaxImpact(t *testing.T) {
	docIDs := []uint32{1, 2, 3, 4, 5}
	impacts := []uint8{10, 80, 30, 200, 50}
	c := newListCursor(docIDs, impacts)

	// From the beginning the max is 200.
	if got := c.maxImpact(); got != 200 {
		t.Errorf("maxImpact() from start = %d, want 200", got)
	}

	// Advance past the highest impact entry (doc 4, impact 200) and the next
	// position is doc 5 with impact 50.
	c.advance(5)
	if got := c.maxImpact(); got != 50 {
		t.Errorf("maxImpact() after advance(5) = %d, want 50", got)
	}

	// Exhaust the cursor.
	c.advance(math.MaxUint32)
	if got := c.maxImpact(); got != 0 {
		t.Errorf("maxImpact() when exhausted = %d, want 0", got)
	}
}

// ---------------------------------------------------------------------------
// wandTopK tests
// ---------------------------------------------------------------------------

// TestWAND_TopK_Basic creates 3 terms over 10 docs and verifies the top-3
// returned documents are those with the highest combined impacts.
func TestWAND_TopK_Basic(t *testing.T) {
	// Term A: appears in docs 1,3,5,7,9 with impacts
	// Term B: appears in docs 2,3,6,7,8 with impacts
	// Term C: appears in docs 4,5,6,7,10 with impacts
	//
	// Combined scores:
	//   doc1: A=10
	//   doc2: B=20
	//   doc3: A=50+B=60 = 110  ← top
	//   doc4: C=30
	//   doc5: A=40+C=70 = 110  ← top (tie with doc3)
	//   doc6: B=80+C=90 = 170  ← top
	//   doc7: A=100+B=110+C=120 = 330 ← highest
	//   doc8: B=15
	//   doc9: A=5
	//   doc10: C=25

	cA := newListCursor(
		[]uint32{1, 3, 5, 7, 9},
		[]uint8{10, 50, 40, 100, 5},
	)
	cB := newListCursor(
		[]uint32{2, 3, 6, 7, 8},
		[]uint8{20, 60, 80, 110, 15},
	)
	cC := newListCursor(
		[]uint32{4, 5, 6, 7, 10},
		[]uint8{30, 70, 90, 120, 25},
	)

	result := wandTopK([]*listCursor{cA, cB, cC}, 3)

	if len(result) != 3 {
		t.Fatalf("want 3 results, got %d", len(result))
	}

	// doc7 must be first (score=330), which is the highest.
	if result[0].docID != 7 {
		t.Errorf("result[0].docID = %d, want 7 (score=330)", result[0].docID)
	}

	// Verify all top 3 are from {3,5,6,7} (the highest combined).
	topDocIDs := make(map[uint32]bool)
	for _, sd := range result {
		topDocIDs[sd.docID] = true
	}
	expected := []uint32{3, 5, 6, 7}
	matched := 0
	for _, e := range expected {
		if topDocIDs[e] {
			matched++
		}
	}
	// The top 3 should all come from this expected set.
	if matched != 3 {
		t.Errorf("top-3 docIDs %v should be a subset of {3,5,6,7}", result)
	}
}

// TestWAND_TopK_KLargerThanResults verifies that when k > number of docs,
// all docs are returned.
func TestWAND_TopK_KLargerThanResults(t *testing.T) {
	docIDs := []uint32{10, 20, 30, 40, 50}
	impacts := []uint8{5, 10, 15, 20, 25}
	c := newListCursor(docIDs, impacts)

	result := wandTopK([]*listCursor{c}, 100)

	if len(result) != 5 {
		t.Fatalf("want 5 results (all docs), got %d", len(result))
	}
}

// TestWAND_TopK_EmptyCursors verifies that zero cursors returns nil.
func TestWAND_TopK_EmptyCursors(t *testing.T) {
	result := wandTopK([]*listCursor{}, 5)
	if result != nil {
		t.Errorf("empty cursors: want nil, got %v", result)
	}
}

// TestWAND_TopK_SingleCursor verifies that with a single term, the top-k
// are the k docIDs with the highest impacts.
func TestWAND_TopK_SingleCursor(t *testing.T) {
	// 8 docs; impacts 10,90,30,70,50,80,20,60 — top-3 by impact: doc2=90, doc6=80, doc4=70.
	docIDs := []uint32{1, 2, 3, 4, 5, 6, 7, 8}
	impacts := []uint8{10, 90, 30, 70, 50, 80, 20, 60}
	c := newListCursor(docIDs, impacts)

	result := wandTopK([]*listCursor{c}, 3)

	if len(result) != 3 {
		t.Fatalf("want 3 results, got %d", len(result))
	}

	topDocIDs := make(map[uint32]bool)
	for _, sd := range result {
		topDocIDs[sd.docID] = true
	}

	for _, expected := range []uint32{2, 4, 6} {
		if !topDocIDs[expected] {
			t.Errorf("expected docID %d in top-3, got %v", expected, result)
		}
	}
}

// TestWAND_TopK_TieBreaking verifies that when two docs have identical scores,
// both are valid top-k entries. We just verify the count and that the expected
// scores are present.
func TestWAND_TopK_TieBreaking(t *testing.T) {
	// Two cursors, each covering disjoint docs. Docs 2 and 4 each have impact 100
	// from their respective cursors; all others have impact 50.
	cA := newListCursor(
		[]uint32{1, 2, 3},
		[]uint8{50, 100, 50},
	)
	cB := newListCursor(
		[]uint32{4, 5, 6},
		[]uint8{100, 50, 50},
	)

	result := wandTopK([]*listCursor{cA, cB}, 2)

	if len(result) != 2 {
		t.Fatalf("want 2 results, got %d", len(result))
	}

	// Both top docs should have score == 100.
	for _, sd := range result {
		if sd.score != 100 {
			t.Errorf("expected score 100 for tied doc, got %v", sd)
		}
	}
}

// TestWAND_TopK_ResultSorted verifies that the returned slice is sorted
// descending by score.
func TestWAND_TopK_ResultSorted(t *testing.T) {
	// Create a single cursor with varied impacts so the result order is non-trivial.
	docIDs := []uint32{1, 2, 3, 4, 5, 6, 7}
	impacts := []uint8{30, 70, 10, 90, 50, 80, 20}
	c := newListCursor(docIDs, impacts)

	result := wandTopK([]*listCursor{c}, 5)

	if len(result) != 5 {
		t.Fatalf("want 5 results, got %d", len(result))
	}

	for i := 1; i < len(result); i++ {
		if result[i].score > result[i-1].score {
			t.Errorf("result not sorted descending at index %d: score[%d]=%v > score[%d]=%v",
				i, i, result[i].score, i-1, result[i-1].score)
		}
	}
}

// TestWAND_TopK_KZero verifies k<=0 returns nil.
func TestWAND_TopK_KZero(t *testing.T) {
	c := newListCursor([]uint32{1, 2, 3}, []uint8{10, 20, 30})
	if r := wandTopK([]*listCursor{c}, 0); r != nil {
		t.Errorf("k=0: want nil, got %v", r)
	}
	if r := wandTopK([]*listCursor{c}, -1); r != nil {
		t.Errorf("k=-1: want nil, got %v", r)
	}
}
