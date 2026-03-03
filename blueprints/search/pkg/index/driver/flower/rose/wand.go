package rose

import (
	"math"
	"sort"
)

// ---------------------------------------------------------------------------
// listCursor
// ---------------------------------------------------------------------------

// listCursor iterates over one posting list for a single query term.
// docIDs and impacts are parallel slices; pos is the current index.
type listCursor struct {
	docIDs  []uint32 // all docIDs for this term (decoded from all blocks)
	impacts []uint8  // corresponding quantised BM25+ impacts
	pos     int      // current position in the list
}

// newListCursor creates a cursor positioned at index 0.
func newListCursor(docIDs []uint32, impacts []uint8) *listCursor {
	return &listCursor{
		docIDs:  docIDs,
		impacts: impacts,
		pos:     0,
	}
}

// docID returns the docID at the current position.
// Returns math.MaxUint32 when the cursor is exhausted.
func (c *listCursor) docID() uint32 {
	if c.pos >= len(c.docIDs) {
		return math.MaxUint32
	}
	return c.docIDs[c.pos]
}

// impact returns the impact at the current position.
// Returns 0 when the cursor is exhausted.
func (c *listCursor) impact() uint8 {
	if c.pos >= len(c.impacts) {
		return 0
	}
	return c.impacts[c.pos]
}

// advance moves the cursor to the first position where docIDs[pos] >= target.
// Uses binary search for efficiency. No-op if already exhausted or current >= target.
func (c *listCursor) advance(target uint32) {
	if c.pos >= len(c.docIDs) {
		return
	}
	if c.docIDs[c.pos] >= target {
		return
	}
	// Binary search over c.docIDs[c.pos:] for the first element >= target.
	remaining := c.docIDs[c.pos:]
	idx := sort.Search(len(remaining), func(i int) bool {
		return remaining[i] >= target
	})
	c.pos += idx
}

// maxImpact returns the maximum impact across all remaining postings (from pos onwards).
// Returns 0 if exhausted. Called infrequently (only during WAND pivot evaluation).
func (c *listCursor) maxImpact() uint8 {
	if c.pos >= len(c.impacts) {
		return 0
	}
	var max uint8
	for _, imp := range c.impacts[c.pos:] {
		if imp > max {
			max = imp
		}
	}
	return max
}

// ---------------------------------------------------------------------------
// scoreDoc
// ---------------------------------------------------------------------------

// scoreDoc pairs a docID with its accumulated impact score for the top-k heap.
type scoreDoc struct {
	docID uint32
	score float32
}

// ---------------------------------------------------------------------------
// Min-heap of scoreDoc (smallest score at top)
// ---------------------------------------------------------------------------
// We implement a manual min-heap to avoid importing container/heap.

func heapifyDown(h []scoreDoc, i int) {
	n := len(h)
	for {
		smallest := i
		left := 2*i + 1
		right := 2*i + 2
		if left < n && h[left].score < h[smallest].score {
			smallest = left
		}
		if right < n && h[right].score < h[smallest].score {
			smallest = right
		}
		if smallest == i {
			break
		}
		h[i], h[smallest] = h[smallest], h[i]
		i = smallest
	}
}

func heapifyUp(h []scoreDoc, i int) {
	for i > 0 {
		parent := (i - 1) / 2
		if h[parent].score <= h[i].score {
			break
		}
		h[parent], h[i] = h[i], h[parent]
		i = parent
	}
}

// heapify builds the min-heap in-place from an arbitrary slice (O(n)).
func heapify(h []scoreDoc) {
	n := len(h)
	for i := n/2 - 1; i >= 0; i-- {
		heapifyDown(h, i)
	}
}

// heapPush appends elem to h and sifts it up. Returns the extended slice.
func heapPush(h []scoreDoc, elem scoreDoc) []scoreDoc {
	h = append(h, elem)
	heapifyUp(h, len(h)-1)
	return h
}

// heapPop removes and returns the minimum element (root). Returns the shrunken slice.
func heapPop(h []scoreDoc) (scoreDoc, []scoreDoc) {
	min := h[0]
	last := len(h) - 1
	h[0] = h[last]
	h = h[:last]
	if len(h) > 0 {
		heapifyDown(h, 0)
	}
	return min, h
}

// ---------------------------------------------------------------------------
// wandTopK — Block-Max WAND top-k retrieval
// ---------------------------------------------------------------------------

// wandTopK returns the top-k documents by accumulated impact score across all
// cursors. The result is sorted descending by score.
//
// Algorithm: simplified WAND with per-cursor maxImpact upper bounds.
//
//  1. Sort cursors by their current docID (ascending).
//  2. Find the pivot: first cursor i where cumulative maxImpact exceeds threshold.
//     - If none, all remaining docs cannot beat threshold → done.
//  3. Let pivotDocID = cursors[pivot].docID().
//     - If MaxUint32 (all exhausted) → done.
//  4. If cursors[0].docID() == pivotDocID: fully score this candidate.
//     - Accumulate impact from every cursor whose current docID == pivotDocID.
//     - If score > threshold: add to heap (evict min if heap full to k).
//     - Update threshold when heap is full.
//     - Advance all cursors that sat at pivotDocID.
//  5. Else: advance cursors[0] to pivotDocID (WAND skip).
//  6. Repeat.
func wandTopK(cursors []*listCursor, k int) []scoreDoc {
	if k <= 0 || len(cursors) == 0 {
		return nil
	}

	var heap []scoreDoc // min-heap of top-k candidates
	threshold := float32(0.0)

	for {
		// Step 1: sort cursors by current docID ascending.
		sort.Slice(cursors, func(i, j int) bool {
			return cursors[i].docID() < cursors[j].docID()
		})

		// Step 2: find pivot — first cursor i where sum of maxImpact[0..i] > threshold.
		var cumScore float32
		pivot := -1
		for i, c := range cursors {
			if c.docID() == math.MaxUint32 {
				// This and all subsequent cursors are exhausted.
				break
			}
			cumScore += float32(c.maxImpact())
			if cumScore > threshold {
				pivot = i
				break
			}
		}

		if pivot == -1 {
			// No pivot found: no remaining doc can beat threshold.
			break
		}

		pivotDocID := cursors[pivot].docID()
		if pivotDocID == math.MaxUint32 {
			// All cursors exhausted.
			break
		}

		if cursors[0].docID() == pivotDocID {
			// Step 4: fully evaluate this candidate.
			var score float32
			for _, c := range cursors {
				if c.docID() == pivotDocID {
					score += float32(c.impact())
				}
			}

			if score > threshold {
				if len(heap) < k {
					heap = heapPush(heap, scoreDoc{docID: pivotDocID, score: score})
					if len(heap) == k {
						threshold = heap[0].score
					}
				} else {
					// Replace the minimum element if new score is better.
					if score > heap[0].score {
						heap[0] = scoreDoc{docID: pivotDocID, score: score}
						heapifyDown(heap, 0)
						threshold = heap[0].score
					}
				}
			}

			// Advance all cursors that were at pivotDocID past it.
			for _, c := range cursors {
				if c.docID() == pivotDocID {
					c.advance(pivotDocID + 1)
				}
			}
		} else {
			// Step 5: cursors[0].docID() < pivotDocID — WAND skip.
			cursors[0].advance(pivotDocID)
		}
	}

	if len(heap) == 0 {
		return nil
	}

	// Sort descending by score before returning.
	result := make([]scoreDoc, len(heap))
	copy(result, heap)
	sort.Slice(result, func(i, j int) bool {
		return result[i].score > result[j].score
	})
	return result
}
