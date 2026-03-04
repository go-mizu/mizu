package dahlia

import (
	"container/heap"
	"sort"
)

type scoredDoc struct {
	seg   *segmentReader
	docID uint32
	score float64
}

// topKHeap is a min-heap of scored documents, keeping the top-K highest scores.
type topKHeap []scoredDoc

func (h topKHeap) Len() int            { return len(h) }
func (h topKHeap) Less(i, j int) bool  { return scoredDocLess(h[i], h[j]) }
func (h topKHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *topKHeap) Push(x interface{}) { *h = append(*h, x.(scoredDoc)) }
func (h *topKHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

func scoredDocLess(a, b scoredDoc) bool {
	if a.score != b.score {
		return a.score < b.score
	}
	// For equal scores, make ordering deterministic by segment then local docID.
	// This comparator is used by the min-heap, so "less" means a worse hit.
	aSeg := ""
	bSeg := ""
	if a.seg != nil {
		aSeg = a.seg.dir
	}
	if b.seg != nil {
		bSeg = b.seg.dir
	}
	if aSeg != bSeg {
		return aSeg > bSeg
	}
	return a.docID > b.docID
}

// wandEvaluator evaluates queries against a single segment using Block-Max WAND.
type wandEvaluator struct {
	seg       *segmentReader
	normTable [256]float32
	topK      int
	docCount  uint64
	termDF    map[string]uint64
}

func newWandEvaluator(seg *segmentReader, topK int, avgDocLen float64, docCount uint64, termDF map[string]uint64) *wandEvaluator {
	if avgDocLen <= 0 {
		avgDocLen = seg.meta.AvgDocLen
	}
	if docCount == 0 {
		docCount = uint64(seg.meta.DocCount)
	}
	return &wandEvaluator{
		seg:       seg,
		normTable: buildFieldNormBM25Table(avgDocLen),
		topK:      topK,
		docCount:  docCount,
		termDF:    termDF,
	}
}

func (w *wandEvaluator) idf(term string, localDocFreq uint32) float64 {
	n := w.docCount
	if n == 0 {
		n = uint64(w.seg.meta.DocCount)
	}
	df := uint64(localDocFreq)
	if gdf, ok := w.termDF[term]; ok && gdf > 0 {
		df = gdf
	}
	if df == 0 {
		df = 1
	}
	if df > n {
		df = n
	}
	return bm25IDF(df, n)
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
	idf := w.idf(q.term, ti.docFreq)

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
		idf := w.idf(term, ti.docFreq)
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
		if phraseTF := countPhraseMatches(phraseIts); phraseTF > 0 {
			// Approximate Lucene/Tantivy phrase scoring: use phrase frequency as TF
			// and sum term IDFs for the phrase weight.
			var phraseIDF float64
			for _, ti := range iters {
				phraseIDF += ti.idf
			}
			normComp := w.normTable[w.seg.fieldNorm(maxDoc)]
			score := bm25ScoreWithNormTable(float64(phraseTF), phraseIDF, normComp)
			w.addToHeap(h, maxDoc, score)
		}

		// Advance first iterator
		if !iters[0].it.next() {
			return heapToSorted(h)
		}
	}
}

func countPhraseMatches(iters []*postingIterator) int {
	// Get positions for each term
	allPositions := make([][]uint32, len(iters))
	for i, it := range iters {
		allPositions[i] = it.positions()
		if len(allPositions[i]) == 0 {
			return 0
		}
	}

	// Count position sequences p[0], p[0]+1, p[0]+2, ... .
	matches := 0
	for _, p0 := range allPositions[0] {
		ok := true
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
				ok = false
				break
			}
		}
		if ok {
			matches++
		}
	}
	return matches
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

// searchBooleanShould implements OR semantics with exact additive BM25 scoring.
func (w *wandEvaluator) searchBooleanShould(q booleanQuery) []scoredDoc {
	type cursor struct {
		it  *postingIterator
		idf float64
	}
	var cursors []cursor
	extraScores := make(map[uint32]float64)
	mustNotSet := make(map[uint32]struct{})

	for _, sq := range q.should {
		tq, ok := sq.(termQuery)
		if !ok {
			// Evaluate non-term sub-queries and merge by docID.
			results := w.searchQuery(sq)
			for _, sd := range results {
				extraScores[sd.docID] += sd.score
			}
			continue
		}
		it, ti, found := w.seg.lookupTerm(tq.term)
		if !found {
			continue
		}
		idf := w.idf(tq.term, ti.docFreq)
		if !it.next() {
			continue
		}
		cursors = append(cursors, cursor{it: it, idf: idf})
	}
	for _, nq := range q.mustNot {
		switch nq := nq.(type) {
		case termQuery:
			it, _, found := w.seg.lookupTerm(nq.term)
			if !found {
				continue
			}
			for it.next() {
				mustNotSet[it.doc()] = struct{}{}
			}
		case phraseQuery:
			phraseHits := w.searchPhrase(nq)
			for _, sd := range phraseHits {
				mustNotSet[sd.docID] = struct{}{}
			}
		}
	}

	if len(cursors) == 0 && len(extraScores) == 0 {
		return nil
	}

	h := &topKHeap{}
	seen := make(map[uint32]struct{}, w.topK*2)

	for len(cursors) > 0 {
		// Find the next candidate doc as the minimum current posting doc.
		minDoc := noMoreDocs
		for _, c := range cursors {
			if d := c.it.doc(); d < minDoc {
				minDoc = d
			}
		}
		if minDoc == noMoreDocs {
			break
		}

		if _, exists := seen[minDoc]; !exists {
			var score float64
			for i := range cursors {
				if cursors[i].it.doc() != minDoc {
					continue
				}
				tf := float64(cursors[i].it.freq())
				normComp := w.normTable[w.seg.fieldNorm(minDoc)]
				score += bm25ScoreWithNormTable(tf, cursors[i].idf, normComp)
			}
			if extra, ok := extraScores[minDoc]; ok {
				score += extra
			}
			if score > 0 {
				if _, excluded := mustNotSet[minDoc]; excluded {
					score = 0
				}
			}
			if score > 0 {
				seen[minDoc] = struct{}{}
				w.addToHeap(h, minDoc, score)
			}
		}

		// Advance all posting cursors positioned at the scored doc.
		for i := range cursors {
			if cursors[i].it.doc() == minDoc {
				if !cursors[i].it.next() {
					cursors[i].it.curDoc = noMoreDocs
				}
			}
		}

		// Remove exhausted cursors.
		active := cursors[:0]
		for _, c := range cursors {
			if c.it.doc() != noMoreDocs {
				active = append(active, c)
			}
		}
		cursors = active
	}

	// Include docs from non-term should clauses when term cursors had no hit.
	for docID, extra := range extraScores {
		if extra <= 0 {
			continue
		}
		if _, excluded := mustNotSet[docID]; excluded {
			continue
		}
		if _, exists := seen[docID]; exists {
			continue
		}
		w.addToHeap(h, docID, extra)
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
	var phraseMustScores []map[uint32]float64
	for _, mq := range q.must {
		switch mq := mq.(type) {
		case termQuery:
			it, ti, found := w.seg.lookupTerm(mq.term)
			if !found {
				return nil
			}
			idf := w.idf(mq.term, ti.docFreq)
			mustCursors = append(mustCursors, cursor{it: it, idf: idf})
		case phraseQuery:
			phraseHits := w.searchPhrase(mq)
			if len(phraseHits) == 0 {
				return nil
			}
			scoreByDoc := make(map[uint32]float64, len(phraseHits))
			for _, sd := range phraseHits {
				scoreByDoc[sd.docID] = sd.score
			}
			phraseMustScores = append(phraseMustScores, scoreByDoc)
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
		idf := w.idf(tq.term, ti.docFreq)
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
		// Phrase-only MUST: intersect phrase results.
		if len(phraseMustScores) == 0 {
			return nil
		}
		h := &topKHeap{}
		for docID, score := range phraseMustScores[0] {
			if mustNotSet[docID] {
				continue
			}
			ok := true
			for i := 1; i < len(phraseMustScores); i++ {
				v, exists := phraseMustScores[i][docID]
				if !exists {
					ok = false
					break
				}
				score += v
			}
			if !ok {
				continue
			}
			w.addToHeap(h, docID, score)
		}
		return heapToSorted(h)
	}

	containsAllMustPhrases := func(docID uint32) bool {
		for _, scoreByDoc := range phraseMustScores {
			if _, ok := scoreByDoc[docID]; !ok {
				return false
			}
		}
		return true
	}

	phraseScore := func(docID uint32) float64 {
		var total float64
		for _, scoreByDoc := range phraseMustScores {
			total += scoreByDoc[docID]
		}
		return total
	}

	if len(mustCursors) == 0 && len(phraseMustScores) == 0 {
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
	seen := make(map[uint32]struct{}, w.topK*2)

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

		// Check phrase MUST and mustNot.
		if containsAllMustPhrases(maxDoc) && !mustNotSet[maxDoc] {
			if _, exists := seen[maxDoc]; exists {
				if !mustCursors[0].it.next() {
					return heapToSorted(h)
				}
				continue
			}
			// Score
			var score float64
			for _, c := range mustCursors {
				tf := float64(c.it.freq())
				normComp := w.normTable[w.seg.fieldNorm(maxDoc)]
				score += bm25ScoreWithNormTable(tf, c.idf, normComp)
			}
			score += phraseScore(maxDoc)
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
			seen[maxDoc] = struct{}{}
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
		heap.Push(h, scoredDoc{seg: w.seg, docID: docID, score: score})
	} else if score > (*h)[0].score {
		(*h)[0] = scoredDoc{seg: w.seg, docID: docID, score: score}
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
	if topK <= 0 {
		topK = 10
	}

	globalDocCount, globalAvgDocLen := globalSearchStats(segments)
	termDF := globalTermDocFreq(segments, q, globalDocCount)

	var allResults []scoredDoc
	for _, seg := range segments {
		eval := newWandEvaluator(seg, topK, globalAvgDocLen, globalDocCount, termDF)
		results := eval.searchQuery(q)
		allResults = append(allResults, results...)
	}

	// Sort by score descending.
	sort.Slice(allResults, func(i, j int) bool {
		return scoredDocLess(allResults[j], allResults[i])
	})

	if len(allResults) > topK {
		allResults = allResults[:topK]
	}
	return allResults
}

func globalSearchStats(segments []*segmentReader) (docCount uint64, avgDocLen float64) {
	var totalLen float64
	for _, seg := range segments {
		n := uint64(seg.meta.DocCount)
		docCount += n
		totalLen += seg.meta.AvgDocLen * float64(seg.meta.DocCount)
	}
	if docCount > 0 {
		avgDocLen = totalLen / float64(docCount)
	}
	return docCount, avgDocLen
}

func globalTermDocFreq(segments []*segmentReader, q query, globalDocCount uint64) map[string]uint64 {
	terms := make(map[string]struct{})
	collectQueryTerms(q, terms)
	if len(terms) == 0 {
		return nil
	}

	out := make(map[string]uint64, len(terms))
	for term := range terms {
		var df uint64
		for _, seg := range segments {
			ti, ok := seg.lookupTermInfo(term)
			if !ok {
				continue
			}
			df += uint64(ti.docFreq)
			if globalDocCount > 0 && df >= globalDocCount {
				df = globalDocCount
				break
			}
		}
		if df > 0 {
			out[term] = df
		}
	}
	return out
}

func collectQueryTerms(q query, out map[string]struct{}) {
	switch qq := q.(type) {
	case termQuery:
		if qq.term != "" {
			out[qq.term] = struct{}{}
		}
	case phraseQuery:
		for _, t := range qq.terms {
			if t != "" {
				out[t] = struct{}{}
			}
		}
	case booleanQuery:
		for _, sq := range qq.must {
			collectQueryTerms(sq, out)
		}
		for _, sq := range qq.should {
			collectQueryTerms(sq, out)
		}
		for _, sq := range qq.mustNot {
			collectQueryTerms(sq, out)
		}
	}
}
