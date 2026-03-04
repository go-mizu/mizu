package dahlia

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// mergeWorker runs background merge operations.
type mergeWorker struct {
	dir      string
	mu       *sync.RWMutex // shared with engine
	meta     **indexMeta
	segments *[]*segmentReader
	stopCh   chan struct{}
	doneCh   chan struct{}
}

func newMergeWorker(dir string, mu *sync.RWMutex, meta **indexMeta, segments *[]*segmentReader) *mergeWorker {
	return &mergeWorker{
		dir:      dir,
		mu:       mu,
		meta:     meta,
		segments: segments,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
}

func (mw *mergeWorker) start() {
	go mw.loop()
}

func (mw *mergeWorker) stop() {
	close(mw.stopCh)
	<-mw.doneCh
}

func (mw *mergeWorker) loop() {
	defer close(mw.doneCh)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-mw.stopCh:
			return
		case <-ticker.C:
			mw.mu.RLock()
			n := len(*mw.segments)
			mw.mu.RUnlock()
			if n > maxSegBeforeMerge {
				mw.tryMerge()
			}
		}
	}
}

func (mw *mergeWorker) tryMerge() {
	mw.mu.Lock()
	defer mw.mu.Unlock()

	segments := *mw.segments
	meta := *mw.meta

	if len(segments) <= 1 {
		return
	}

	// Keep background merge working set bounded.
	selected := pickMergeIndices(segments, maxMergeSegments, maxBgMergeDocs, false)
	if len(selected) < 2 {
		return
	}

	// Collect segment readers for merge
	mergeReaders := make([]*segmentReader, len(selected))
	mergeIndices := make(map[int]struct{}, len(selected))
	for i, idx := range selected {
		mergeReaders[i] = segments[idx]
		mergeIndices[idx] = struct{}{}
	}

	// Perform N-way merge
	segSeq := meta.NextSegSeq
	meta.NextSegSeq++
	segName := fmt.Sprintf(segDirFmt, segSeq)
	segDir := filepath.Join(mw.dir, segName)

	if err := mergeSegments(mergeReaders, segDir); err != nil {
		return // silently fail, will retry
	}

	// Open new segment
	newSeg, err := openSegmentReader(segDir)
	if err != nil {
		os.RemoveAll(segDir)
		return
	}

	// Build new segment list
	var newSegments []*segmentReader
	var newSegNames []string
	for i, seg := range segments {
		if _, ok := mergeIndices[i]; !ok {
			newSegments = append(newSegments, seg)
			newSegNames = append(newSegNames, meta.Segments[i])
		}
	}
	newSegments = append(newSegments, newSeg)
	newSegNames = append(newSegNames, segName)

	// Update metadata
	meta.Segments = newSegNames

	// Recompute totals
	var totalDocs uint64
	var totalTokens float64
	for _, seg := range newSegments {
		totalDocs += uint64(seg.meta.DocCount)
		totalTokens += seg.meta.AvgDocLen * float64(seg.meta.DocCount)
	}
	meta.DocCount = totalDocs
	if totalDocs > 0 {
		meta.AvgDocLen = totalTokens / float64(totalDocs)
	}

	saveIndexMeta(mw.dir, meta)

	// Close old segments and delete their directories
	for i, seg := range segments {
		if _, ok := mergeIndices[i]; ok {
			oldDir := seg.dir
			seg.Close()
			os.RemoveAll(oldDir)
		}
	}

	*mw.segments = newSegments
}

// mergeSegments performs an N-way merge of segments into a new segment directory.
func mergeSegments(readers []*segmentReader, outDir string) error {
	sw := newSegmentWriter()

	for _, reader := range readers {
		for docID := uint32(0); docID < reader.meta.DocCount; docID++ {
			id, text, err := reader.getDoc(docID)
			if err != nil {
				return fmt.Errorf("read doc %d: %w", docID, err)
			}
			sw.addDoc(id, text)
		}
	}

	if sw.docCount == 0 {
		return fmt.Errorf("no documents to merge")
	}

	_, err := sw.flush(outDir)
	return err
}

// forceMerge merges all segments into one (used by Finalize).
func forceMerge(dir string, mu *sync.RWMutex, meta **indexMeta, segments *[]*segmentReader) error {
	mu.Lock()
	defer mu.Unlock()

	segs := *segments
	if len(segs) <= 1 {
		return nil
	}

	m := *meta
	segNames := append([]string(nil), m.Segments...)
	for len(segs) > 1 {
		selected := pickMergeIndices(segs, maxMergeSegments, maxFMMergeDocs, true)
		if len(selected) < 2 {
			return fmt.Errorf("force merge stalled: no eligible segments")
		}

		mergeReaders := make([]*segmentReader, len(selected))
		mergeSet := make(map[int]struct{}, len(selected))
		for i, idx := range selected {
			mergeReaders[i] = segs[idx]
			mergeSet[idx] = struct{}{}
		}

		segSeq := m.NextSegSeq
		m.NextSegSeq++
		segName := fmt.Sprintf(segDirFmt, segSeq)
		segDir := filepath.Join(dir, segName)

		if err := mergeSegments(mergeReaders, segDir); err != nil {
			return err
		}
		newSeg, err := openSegmentReader(segDir)
		if err != nil {
			os.RemoveAll(segDir)
			return err
		}

		nextSegs := make([]*segmentReader, 0, len(segs)-len(selected)+1)
		nextNames := make([]string, 0, len(segNames)-len(selected)+1)
		for i, seg := range segs {
			if _, ok := mergeSet[i]; ok {
				oldDir := seg.dir
				seg.Close()
				os.RemoveAll(oldDir)
				continue
			}
			nextSegs = append(nextSegs, seg)
			nextNames = append(nextNames, segNames[i])
		}
		nextSegs = append(nextSegs, newSeg)
		nextNames = append(nextNames, segName)
		segs = nextSegs
		segNames = nextNames
	}

	m.Segments = segNames
	if len(segs) == 1 {
		m.DocCount = uint64(segs[0].meta.DocCount)
		m.AvgDocLen = segs[0].meta.AvgDocLen
	} else {
		var totalDocs uint64
		var totalTokens float64
		for _, seg := range segs {
			totalDocs += uint64(seg.meta.DocCount)
			totalTokens += seg.meta.AvgDocLen * float64(seg.meta.DocCount)
		}
		m.DocCount = totalDocs
		if totalDocs > 0 {
			m.AvgDocLen = totalTokens / float64(totalDocs)
		}
	}

	*segments = segs
	return saveIndexMeta(dir, m)
}

type segIdx struct {
	idx      int
	docCount uint32
}

// pickMergeIndices selects small segments first while keeping total docs under maxDocs.
// If forceProgress is true and budget blocks selection, it still picks 2 smallest segments.
func pickMergeIndices(segments []*segmentReader, maxSegments int, maxDocs int, forceProgress bool) []int {
	if len(segments) < 2 || maxSegments < 2 {
		return nil
	}

	indexed := make([]segIdx, len(segments))
	for i, seg := range segments {
		indexed[i] = segIdx{idx: i, docCount: seg.meta.DocCount}
	}
	sort.Slice(indexed, func(i, j int) bool {
		if indexed[i].docCount != indexed[j].docCount {
			return indexed[i].docCount < indexed[j].docCount
		}
		return indexed[i].idx < indexed[j].idx
	})

	limit := len(indexed)
	if limit > maxSegments {
		limit = maxSegments
	}

	chosen := make([]int, 0, limit)
	var total int
	for i := 0; i < limit; i++ {
		cand := indexed[i]
		nextTotal := total + int(cand.docCount)
		if maxDocs > 0 && len(chosen) > 0 && nextTotal > maxDocs {
			break
		}
		chosen = append(chosen, cand.idx)
		total = nextTotal
	}

	if len(chosen) >= 2 {
		return chosen
	}
	if !forceProgress || len(indexed) < 2 {
		return nil
	}
	return []int{indexed[0].idx, indexed[1].idx}
}
