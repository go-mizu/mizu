package dahlia

import (
	"container/heap"
	"sort"
)

type scoredDoc struct {
	docID uint32
	score float64
}

// topKHeap is a min-heap of scored documents, keeping the top-K highest scores.
type topKHeap []scoredDoc

func (h topKHeap) Len() int            { return len(h) }
func (h topKHeap) Less(i, j int) bool   { return h[i].score < h[j].score }
func (h topKHeap) Swap(i, j int)        { h[i], h[j] = h[j], h[i] }
func (h *topKHeap) Push(x interface{})   { *h = append(*h, x.(scoredDoc)) }
func (h *topKHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

// wandEvaluator evaluates queries against a single segment using Block-Max WAND.
type wandEvaluator struct {
	seg       *segmentReader
	normTable [256]float32
	topK      int
}

func newWandEvaluator(seg *segmentReader, topK int) *wandEvaluator {
	return &wandEvaluator{
		seg:       seg,
		normTable: buildFieldNormBM25Table(seg.meta.AvgDocLen),
		topK:      topK,
	}
}

// searchQuery dispatches to the appropriate search method based on query type.
func (w *wandEvaluator) searchQuery(q query) []scoredDoc {
	switch q := q.(type) {
	case termQuery:
		return w.searchTerm(q)
	case phraseQuery:
		return w.searchPhrase(q)
	case booleanQuery:
		return w.searchBoolean(q)
	default:
		return nil
	}
}

func (w *wandEvaluator) searchTerm(q termQuery) []scoredDoc {
	it, ti, found := w.seg.lookupTerm(q.term)
	if !found {
		return nil
	}
	idf := bm25IDF(uint64(ti.docFreq), uint64(w.seg.meta.DocCount))

	h := &topKHeap{}
	for it.next() {
		tf := float64(it.freq())
		normComp := w.normTable[w.seg.fieldNorm(it.doc())]
		score := bm25ScoreWithNormTable(tf, idf, normComp)
		w.addToHeap(h, it.doc(), score)
	}
	return heapToSorted(h)
}

func (w *wandEvaluator) searchPhrase(q phraseQuery) []scoredDoc {
	if len(q.terms) == 0 {
		return nil
	}

	// Get iterators for all terms
	type termIter struct {
		it  *postingIterator
		ti  termInfo
		idf float64
	}
	var iters []termIter
	for _, term := range q.terms {
		it, ti, found := w.seg.lookupTerm(term)
		if !found {
			return nil // ALL terms must exist for phrase
		}
		idf := bm25IDF(uint64(ti.docFreq), uint64(w.seg.meta.DocCount))
		iters = append(iters, termIter{it: it, ti: ti, idf: idf})
	}

	// Intersect + position check
	h := &topKHeap{}

	// Advance all to first doc
	for i := range iters {
		if !iters[i].it.next() {
			return heapToSorted(h)
		}
	}

	for {
		// Find the max docID among all iterators
		maxDoc := uint32(0)
		for _, ti := range iters {
			if ti.it.doc() > maxDoc {
				maxDoc = ti.it.doc()
			}
		}

		// Advance all iterators to maxDoc
		allMatch := true
		for i := range iters {
			if iters[i].it.doc() < maxDoc {
				if !iters[i].it.advance(maxDoc) {
					return heapToSorted(h) // exhausted
				}
			}
			if iters[i].it.doc() != maxDoc {
				allMatch = false
			}
		}

		if !allMatch {
			continue
		}

		// All iterators at same doc — check position adjacency
		phraseIts := make([]*postingIterator, len(iters))
		for i := range iters {
			phraseIts[i] = iters[i].it
		}
		if checkPhrasePositions(phraseIts) {
			// Score using sum of term scores
			var totalScore float64
			for _, ti := range iters {
				tf := float64(ti.it.freq())
				normComp := w.normTable[w.seg.fieldNorm(maxDoc)]
				totalScore += bm25ScoreWithNormTable(tf, ti.idf, normComp)
			}
			w.addToHeap(h, maxDoc, totalScore)
		}

		// Advance first iterator
		if !iters[0].it.next() {
			return heapToSorted(h)
		}
	}
}

func checkPhrasePositions(iters []*postingIterator) bool {
	// Get positions for each term
	allPositions := make([][]uint32, len(iters))
	for i, it := range iters {
		allPositions[i] = it.positions()
		if len(allPositions[i]) == 0 {
			return false
		}
	}

	// Check if any position sequence p[0], p[0]+1, p[0]+2, ... exists
	// For each position of the first term, check subsequent terms
	for _, p0 := range allPositions[0] {
		match := true
		for k := 1; k < len(iters); k++ {
			target := p0 + uint32(k)
			found := false
			for _, p := range allPositions[k] {
				if p == target {
					found = true
					break
				}
			}
			if !found {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func (w *wandEvaluator) searchBoolean(q booleanQuery) []scoredDoc {
	if len(q.must) > 0 {
		return w.searchBooleanMust(q)
	}
	if len(q.should) > 0 {
		return w.searchBooleanShould(q)
	}
	return nil
}

// searchBooleanShould implements OR semantics with Block-Max WAND pruning.
func (w *wandEvaluator) searchBooleanShould(q booleanQuery) []scoredDoc {
	type cursor struct {
		it  *postingIterator
		idf float64
	}
	var cursors []cursor

	for _, sq := range q.should {
		tq, ok := sq.(termQuery)
		if !ok {
			// For non-term sub-queries, evaluate separately and merge
			results := w.searchQuery(sq)
			// Just use term queries for WAND
			_ = results
			continue
		}
		it, ti, found := w.seg.lookupTerm(tq.term)
		if !found {
			continue
		}
		idf := bm25IDF(uint64(ti.docFreq), uint64(w.seg.meta.DocCount))
		if !it.next() {
			continue
		}
		cursors = append(cursors, cursor{it: it, idf: idf})
	}

	if len(cursors) == 0 {
		return nil
	}

	h := &topKHeap{}
	threshold := float64(0)

	for {
		// Sort cursors by current docID
		sort.Slice(cursors, func(i, j int) bool {
			return cursors[i].it.doc() < cursors[j].it.doc()
		})

		// Remove exhausted cursors
		active := cursors[:0]
		for _, c := range cursors {
			if c.it.doc() != noMoreDocs {
				active = append(active, c)
			}
		}
		cursors = active
		if len(cursors) == 0 {
			break
		}

		// Find pivot: first position where sum of upper bounds >= threshold
		pivotIdx := -1
		if threshold > 0 {
			var cumUB float64
			for i, c := range cursors {
				cumUB += c.it.blockMaxImpact(c.idf, w.normTable)
				if cumUB >= threshold {
					pivotIdx = i
					break
				}
			}
			if pivotIdx < 0 {
				// No combination can beat threshold — try advancing
				// the cursor with the smallest upper bound
				if !cursors[0].it.next() {
					cursors[0].it.curDoc = noMoreDocs
				}
				continue
			}
		} else {
			pivotIdx = 0
		}

		pivotDoc := cursors[pivotIdx].it.doc()

		// Score all cursors at pivotDoc
		var score float64
		for i := range cursors {
			if cursors[i].it.doc() == pivotDoc {
				tf := float64(cursors[i].it.freq())
				normComp := w.normTable[w.seg.fieldNorm(pivotDoc)]
				score += bm25ScoreWithNormTable(tf, cursors[i].idf, normComp)
			}
		}
		if score > 0 {
			w.addToHeap(h, pivotDoc, score)
			if h.Len() >= w.topK {
				threshold = (*h)[0].score
			}
		}

		// Advance all cursors that were at pivotDoc
		for i := range cursors {
			if cursors[i].it.doc() == pivotDoc {
				if !cursors[i].it.next() {
					cursors[i].it.curDoc = noMoreDocs
				}
			}
		}
	}

	return heapToSorted(h)
}

// searchBooleanMust implements AND semantics with mustNot exclusion.
func (w *wandEvaluator) searchBooleanMust(q booleanQuery) []scoredDoc {
	type cursor struct {
		it  *postingIterator
		idf float64
	}

	var mustCursors []cursor
	for _, mq := range q.must {
		switch mq := mq.(type) {
		case termQuery:
			it, ti, found := w.seg.lookupTerm(mq.term)
			if !found {
				return nil
			}
			idf := bm25IDF(uint64(ti.docFreq), uint64(w.seg.meta.DocCount))
			mustCursors = append(mustCursors, cursor{it: it, idf: idf})
		case phraseQuery:
			// Evaluate phrase separately
			return w.searchPhrase(mq)
		}
	}

	// Also include should clauses for scoring
	var shouldCursors []cursor
	for _, sq := range q.should {
		tq, ok := sq.(termQuery)
		if !ok {
			continue
		}
		it, ti, found := w.seg.lookupTerm(tq.term)
		if !found {
			continue
		}
		idf := bm25IDF(uint64(ti.docFreq), uint64(w.seg.meta.DocCount))
		shouldCursors = append(shouldCursors, cursor{it: it, idf: idf})
	}

	// Build mustNot set
	mustNotSet := make(map[uint32]bool)
	for _, nq := range q.mustNot {
		tq, ok := nq.(termQuery)
		if !ok {
			continue
		}
		it, _, found := w.seg.lookupTerm(tq.term)
		if !found {
			continue
		}
		for it.next() {
			mustNotSet[it.doc()] = true
		}
	}

	if len(mustCursors) == 0 {
		return nil
	}

	// Advance all must cursors to first doc
	for i := range mustCursors {
		if !mustCursors[i].it.next() {
			return nil
		}
	}
	for i := range shouldCursors {
		shouldCursors[i].it.next()
	}

	h := &topKHeap{}

	for {
		// Find max docID among must cursors
		maxDoc := uint32(0)
		for _, c := range mustCursors {
			if c.it.doc() > maxDoc {
				maxDoc = c.it.doc()
			}
		}

		// Advance all must cursors to maxDoc
		allMatch := true
		for i := range mustCursors {
			if mustCursors[i].it.doc() < maxDoc {
				if !mustCursors[i].it.advance(maxDoc) {
					return heapToSorted(h) // exhausted
				}
			}
			if mustCursors[i].it.doc() != maxDoc {
				allMatch = false
			}
		}

		if !allMatch {
			continue
		}

		// Check mustNot
		if !mustNotSet[maxDoc] {
			// Score
			var score float64
			for _, c := range mustCursors {
				tf := float64(c.it.freq())
				normComp := w.normTable[w.seg.fieldNorm(maxDoc)]
				score += bm25ScoreWithNormTable(tf, c.idf, normComp)
			}
			// Add should contributions
			for i := range shouldCursors {
				if shouldCursors[i].it.doc() != noMoreDocs {
					shouldCursors[i].it.advance(maxDoc)
					if shouldCursors[i].it.doc() == maxDoc {
						tf := float64(shouldCursors[i].it.freq())
						normComp := w.normTable[w.seg.fieldNorm(maxDoc)]
						score += bm25ScoreWithNormTable(tf, shouldCursors[i].idf, normComp)
					}
				}
			}
			w.addToHeap(h, maxDoc, score)
		}

		// Advance first must cursor
		if !mustCursors[0].it.next() {
			return heapToSorted(h)
		}
	}
}

func (w *wandEvaluator) addToHeap(h *topKHeap, docID uint32, score float64) {
	if h.Len() < w.topK {
		heap.Push(h, scoredDoc{docID: docID, score: score})
	} else if score > (*h)[0].score {
		(*h)[0] = scoredDoc{docID: docID, score: score}
		heap.Fix(h, 0)
	}
}

func heapToSorted(h *topKHeap) []scoredDoc {
	result := make([]scoredDoc, h.Len())
	for i := len(result) - 1; i >= 0; i-- {
		result[i] = heap.Pop(h).(scoredDoc)
	}
	return result
}

// multiSegmentSearch searches all segments and merges results.
func multiSegmentSearch(segments []*segmentReader, q query, topK int) []scoredDoc {
	var allResults []scoredDoc
	for _, seg := range segments {
		eval := newWandEvaluator(seg, topK)
		results := eval.searchQuery(q)
		allResults = append(allResults, results...)
	}

	// Sort by score descending
	sort.Slice(allResults, func(i, j int) bool {
		return allResults[i].score > allResults[j].score
	})

	if len(allResults) > topK {
		allResults = allResults[:topK]
	}
	return allResults
}
