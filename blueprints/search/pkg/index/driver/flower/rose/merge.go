package rose

import (
	"fmt"
	"log"
	"os"
	"sort"
)

// runMergeLoop is the background merge goroutine started by Open.
// It waits for signals on mergeCh and performs tiered LSM merges.
func (s *roseEngine) runMergeLoop() {
	defer s.mergeWg.Done()
	for {
		select {
		case <-s.done:
			return
		case <-s.mergeCh:
			s.mu.Lock()
			if err := s.doMerge(); err != nil {
				log.Printf("rose: background merge failed: %v", err)
			}
			s.mu.Unlock()
		}
	}
}

// doMerge performs a tiered LSM merge if enough segments exist.
// Must be called with s.mu held for writing.
//
// Policy: merge the oldest mergeMinSegs (4) segments into one new segment
// whenever len(s.segments) >= mergeMinSegs.
func (s *roseEngine) doMerge() error {
	if len(s.segments) < mergeMinSegs {
		return nil
	}

	// Pick the oldest mergeMinSegs segments.
	toMerge := s.segments[:mergeMinSegs]

	// -----------------------------------------------------------------------
	// 1. Collect all terms from the segments to merge.
	// -----------------------------------------------------------------------
	termSet := make(map[string]struct{})
	for _, seg := range toMerge {
		for _, te := range seg.termDict {
			termSet[te.term] = struct{}{}
		}
	}

	terms := make([]string, 0, len(termSet))
	for t := range termSet {
		terms = append(terms, t)
	}
	sort.Strings(terms)

	// -----------------------------------------------------------------------
	// 2. For each term, concatenate docIDs and impacts from all segments.
	//    DocIDs are globally ordered (each segment covers a non-overlapping
	//    range of monotone-ascending docIDs), so simple concatenation in
	//    segment order produces a globally sorted list.
	// -----------------------------------------------------------------------
	merged := make(map[string][]memPosting, len(terms))
	for _, term := range terms {
		var postings []memPosting
		for _, seg := range toMerge {
			te, found := findTerm(seg.termDict, term)
			if !found {
				continue
			}
			docIDs, impacts, err := readPostings(seg.postData, te)
			if err != nil {
				return fmt.Errorf("rose doMerge: readPostings %q: %w", term, err)
			}
			for i, did := range docIDs {
				postings = append(postings, memPosting{docID: did, impact: impacts[i]})
			}
		}
		if len(postings) > 0 {
			merged[term] = postings
		}
	}

	// -----------------------------------------------------------------------
	// 3. Compute combined stats.
	// -----------------------------------------------------------------------
	var totalDocCount uint32
	var totalAvgLen uint64
	for _, seg := range toMerge {
		totalDocCount += seg.docCount
		totalAvgLen += uint64(seg.avgDocLen) * uint64(seg.docCount)
	}
	avgDocLen := uint32(0)
	if totalDocCount > 0 {
		avgDocLen = uint32(totalAvgLen / uint64(totalDocCount))
	}

	// -----------------------------------------------------------------------
	// 4. Write the new merged segment.
	//    Use len(s.segments) as the index to avoid clashing with existing names.
	// -----------------------------------------------------------------------
	newPath := s.nextSegPath()
	if err := flushSegment(newPath, merged, totalDocCount, avgDocLen); err != nil {
		return fmt.Errorf("rose doMerge: flushSegment: %w", err)
	}

	td, pd, dc, al, err := openSegment(newPath)
	if err != nil {
		return fmt.Errorf("rose doMerge: openSegment: %w", err)
	}
	newHandle := segmentHandle{
		path:      newPath,
		termDict:  td,
		postData:  pd,
		docCount:  dc,
		avgDocLen: al,
	}

	// -----------------------------------------------------------------------
	// 5. Remove the old segment files.
	// -----------------------------------------------------------------------
	for _, seg := range toMerge {
		if err := os.Remove(seg.path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("rose doMerge: remove %q: %w", seg.path, err)
		}
	}

	// -----------------------------------------------------------------------
	// 6. Replace the first mergeMinSegs entries with the new merged segment.
	// -----------------------------------------------------------------------
	s.segments = append([]segmentHandle{newHandle}, s.segments[mergeMinSegs:]...)

	return nil
}
