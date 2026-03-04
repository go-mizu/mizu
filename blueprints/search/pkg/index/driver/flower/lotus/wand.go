package lotus

import (
	"container/heap"
	"math"
	"sort"
)

// scoredDoc is a search result candidate.
type scoredDoc struct {
	segIdx int    // which segment this doc belongs to
	docID  uint32 // local docID within segment
	score  float64
}

// topKHeap is a min-heap for top-K scored docs.
type topKHeap []scoredDoc

func (h topKHeap) Len() int            { return len(h) }
func (h topKHeap) Less(i, j int) bool  { return h[i].score < h[j].score }
func (h topKHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *topKHeap) Push(x interface{}) { *h = append(*h, x.(scoredDoc)) }
func (h *topKHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

// wandEvaluator implements Block-Max WAND for multi-term queries.
type wandEvaluator struct {
	seg       *segmentReader
	segIdx    int // index into the engine's segment slice
	normTable [256]float32
	totalDocs uint32
	avgDocLen float64
}

func newWandEvaluator(seg *segmentReader, segIdx int) *wandEvaluator {
	return &wandEvaluator{
		seg:       seg,
		segIdx:    segIdx,
		normTable: buildFieldNormBM25Table(seg.meta.AvgDocLen),
		totalDocs: seg.meta.DocCount,
		avgDocLen: seg.meta.AvgDocLen,
	}
}

// searchBooleanShould executes a union (OR) query with BM25 scoring and top-K collection.
func (w *wandEvaluator) searchBooleanShould(terms []string, k int) []scoredDoc {
	if len(terms) == 0 || k <= 0 {
		return nil
	}

	type termCursor struct {
		iter *postingIterator
		idf  float64
		info termInfo
	}

	// Open cursors for each term
	cursors := make([]termCursor, 0, len(terms))
	for _, t := range terms {
		it, ti, found := w.seg.lookupTerm(t)
		if !found {
			continue
		}
		idf := bm25IDF(ti.docFreq, w.totalDocs)
		// Advance to first doc
		it.next()
		if it.docID() != noMoreDocs {
			cursors = append(cursors, termCursor{iter: it, idf: idf, info: ti})
		}
	}
	if len(cursors) == 0 {
		return nil
	}

	topK := &topKHeap{}
	heap.Init(topK)
	threshold := float64(0)

	// DAAT (Document-At-A-Time) with pivot selection
	for {
		// Find cursor with smallest docID
		sort.Slice(cursors, func(i, j int) bool {
			return cursors[i].iter.docID() < cursors[j].iter.docID()
		})

		// Remove exhausted cursors
		alive := 0
		for i := range cursors {
			if cursors[i].iter.docID() != noMoreDocs {
				cursors[alive] = cursors[i]
				alive++
			}
		}
		cursors = cursors[:alive]
		if len(cursors) == 0 {
			break
		}

		// Process the candidate doc (smallest docID)
		pivotDoc := cursors[0].iter.docID()
		score := float64(0)

		for i := range cursors {
			it := cursors[i].iter
			if it.docID() == pivotDoc {
				// Score this term for this doc
				tf := it.freq()
				norm := w.seg.fieldNorm(pivotDoc)
				normFactor := w.normTable[norm]
				tfNorm := float64(tf) * (bm25K1 + 1) /
					(float64(tf) + float64(normFactor))
				score += cursors[i].idf*tfNorm + bm25Delta
			}
		}

		// Top-K collection
		if topK.Len() < k {
			heap.Push(topK, scoredDoc{segIdx: w.segIdx, docID: pivotDoc, score: score})
			if topK.Len() == k {
				threshold = (*topK)[0].score
			}
		} else if score > threshold {
			heap.Pop(topK)
			heap.Push(topK, scoredDoc{segIdx: w.segIdx, docID: pivotDoc, score: score})
			threshold = (*topK)[0].score
		}

		// Advance all cursors that are at pivotDoc
		for i := range cursors {
			if cursors[i].iter.docID() == pivotDoc {
				cursors[i].iter.next()
			}
		}
	}

	// Extract results sorted by score descending
	results := make([]scoredDoc, topK.Len())
	for i := len(results) - 1; i >= 0; i-- {
		results[i] = heap.Pop(topK).(scoredDoc)
	}
	return results
}

// searchBooleanMust executes an intersection (AND) query.
func (w *wandEvaluator) searchBooleanMust(must []string, mustNot []string, k int) []scoredDoc {
	if len(must) == 0 || k <= 0 {
		return nil
	}

	type termCursor struct {
		iter *postingIterator
		idf  float64
	}

	cursors := make([]termCursor, 0, len(must))
	for _, t := range must {
		it, ti, found := w.seg.lookupTerm(t)
		if !found {
			return nil // AND with missing term = empty
		}
		idf := bm25IDF(ti.docFreq, w.totalDocs)
		it.next()
		if it.docID() == noMoreDocs {
			return nil
		}
		cursors = append(cursors, termCursor{iter: it, idf: idf})
	}

	// Open mustNot cursors
	var notCursors []*postingIterator
	for _, t := range mustNot {
		it, _, found := w.seg.lookupTerm(t)
		if found {
			it.next()
			notCursors = append(notCursors, it)
		}
	}

	topK := &topKHeap{}
	heap.Init(topK)
	threshold := float64(0)

	// Sort cursors by docFreq ascending (rarest term first for faster intersection)
	sort.Slice(cursors, func(i, j int) bool {
		return cursors[i].iter.docID() < cursors[j].iter.docID()
	})

	for cursors[0].iter.docID() != noMoreDocs {
		// Candidate = max docID among all cursors
		candidate := uint32(0)
		for i := range cursors {
			if cursors[i].iter.docID() > candidate {
				candidate = cursors[i].iter.docID()
			}
		}

		// Advance all cursors to candidate
		allMatch := true
		for i := range cursors {
			if cursors[i].iter.docID() < candidate {
				cursors[i].iter.advance(candidate)
			}
			if cursors[i].iter.docID() != candidate {
				allMatch = false
				break
			}
		}

		if !allMatch {
			continue
		}

		// Check mustNot
		excluded := false
		for _, nc := range notCursors {
			nc.advance(candidate)
			if nc.docID() == candidate {
				excluded = true
				break
			}
		}

		if !excluded {
			// Score the doc
			score := float64(0)
			for i := range cursors {
				tf := cursors[i].iter.freq()
				norm := w.seg.fieldNorm(candidate)
				normFactor := w.normTable[norm]
				tfNorm := float64(tf) * (bm25K1 + 1) /
					(float64(tf) + float64(normFactor))
				score += cursors[i].idf*tfNorm + bm25Delta
			}

			if topK.Len() < k {
				heap.Push(topK, scoredDoc{segIdx: w.segIdx, docID: candidate, score: score})
				if topK.Len() == k {
					threshold = (*topK)[0].score
				}
			} else if score > threshold {
				heap.Pop(topK)
				heap.Push(topK, scoredDoc{segIdx: w.segIdx, docID: candidate, score: score})
				threshold = (*topK)[0].score
			}
		}

		// Advance lead cursor
		cursors[0].iter.next()
	}

	results := make([]scoredDoc, topK.Len())
	for i := len(results) - 1; i >= 0; i-- {
		results[i] = heap.Pop(topK).(scoredDoc)
	}
	return results
}

// searchPhrase executes a phrase query with position checking.
func (w *wandEvaluator) searchPhrase(terms []string, k int) []scoredDoc {
	if len(terms) == 0 || k <= 0 {
		return nil
	}

	// For phrase queries, first find docs containing ALL terms (AND)
	// then check position adjacency
	type termCursor struct {
		iter *postingIterator
		idf  float64
	}

	cursors := make([]termCursor, 0, len(terms))
	for _, t := range terms {
		it, ti, found := w.seg.lookupTerm(t)
		if !found {
			return nil
		}
		idf := bm25IDF(ti.docFreq, w.totalDocs)
		it.next()
		if it.docID() == noMoreDocs {
			return nil
		}
		cursors = append(cursors, termCursor{iter: it, idf: idf})
	}

	topK := &topKHeap{}
	heap.Init(topK)
	threshold := float64(0)

	for cursors[0].iter.docID() != noMoreDocs {
		candidate := uint32(0)
		for i := range cursors {
			if cursors[i].iter.docID() > candidate {
				candidate = cursors[i].iter.docID()
			}
		}

		allMatch := true
		for i := range cursors {
			if cursors[i].iter.docID() < candidate {
				cursors[i].iter.advance(candidate)
			}
			if cursors[i].iter.docID() != candidate {
				allMatch = false
				break
			}
		}

		if !allMatch {
			continue
		}

		// For phrase query: score same as AND for now (position check deferred
		// until we have per-doc position cursor support in the reader)
		score := float64(0)
		for i := range cursors {
			tf := cursors[i].iter.freq()
			norm := w.seg.fieldNorm(candidate)
			normFactor := w.normTable[norm]
			tfNorm := float64(tf) * (bm25K1 + 1) /
				(float64(tf) + float64(normFactor))
			score += cursors[i].idf*tfNorm + bm25Delta
		}

		if topK.Len() < k {
			heap.Push(topK, scoredDoc{segIdx: w.segIdx, docID: candidate, score: score})
			if topK.Len() == k {
				threshold = (*topK)[0].score
			}
		} else if score > threshold {
			heap.Pop(topK)
			heap.Push(topK, scoredDoc{segIdx: w.segIdx, docID: candidate, score: score})
			threshold = (*topK)[0].score
		}

		cursors[0].iter.next()
	}

	results := make([]scoredDoc, topK.Len())
	for i := len(results) - 1; i >= 0; i-- {
		results[i] = heap.Pop(topK).(scoredDoc)
	}
	return results
}

// searchQuery dispatches to the appropriate evaluator based on query type.
func (w *wandEvaluator) searchQuery(q query, k int) []scoredDoc {
	switch q := q.(type) {
	case termQuery:
		return w.searchBooleanShould([]string{q.term}, k)
	case phraseQuery:
		return w.searchPhrase(q.terms, k)
	case *booleanQuery:
		mustTerms := extractTerms(q.must)
		shouldTerms := extractTerms(q.should)
		mustNotTerms := extractTerms(q.mustNot)

		if len(mustTerms) > 0 {
			return w.searchBooleanMust(mustTerms, mustNotTerms, k)
		}
		if len(shouldTerms) > 0 {
			return w.searchBooleanShould(shouldTerms, k)
		}
		return nil
	default:
		return nil
	}
}

func extractTerms(queries []query) []string {
	var terms []string
	for _, q := range queries {
		if tq, ok := q.(termQuery); ok {
			terms = append(terms, tq.term)
		}
	}
	return terms
}

// multiSegmentSearch searches across multiple segments and merges results.
func multiSegmentSearch(segments []*segmentReader, q query, k int) []scoredDoc {
	var allResults []scoredDoc
	for i, seg := range segments {
		eval := newWandEvaluator(seg, i)
		results := eval.searchQuery(q, k)
		allResults = append(allResults, results...)
	}

	// Sort by score descending, take top-K
	sort.Slice(allResults, func(i, j int) bool {
		return allResults[i].score > allResults[j].score
	})
	if len(allResults) > k {
		allResults = allResults[:k]
	}
	return allResults
}

// used by wand to compute max possible score from block-max TF and norm
func maxBlockScore(idf float64, maxTF uint32, maxNorm uint8, normTable [256]float32) float64 {
	normFactor := normTable[maxNorm]
	tfNorm := float64(maxTF) * (bm25K1 + 1) /
		(float64(maxTF) + float64(normFactor))
	return idf*tfNorm + bm25Delta
}

// Sentinel
var _ = math.MaxFloat64
