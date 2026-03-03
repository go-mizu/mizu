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

	// Select segments to merge (smallest first, up to maxMergeSegments)
	type segIdx struct {
		idx      int
		docCount uint32
	}
	indexed := make([]segIdx, len(segments))
	for i, seg := range segments {
		indexed[i] = segIdx{idx: i, docCount: seg.meta.DocCount}
	}
	sort.Slice(indexed, func(i, j int) bool {
		return indexed[i].docCount < indexed[j].docCount
	})

	mergeCount := len(indexed)
	if mergeCount > maxMergeSegments {
		mergeCount = maxMergeSegments
	}
	toMerge := indexed[:mergeCount]

	// Collect segment readers for merge
	mergeReaders := make([]*segmentReader, mergeCount)
	mergeIndices := make(map[int]bool)
	for i, si := range toMerge {
		mergeReaders[i] = segments[si.idx]
		mergeIndices[si.idx] = true
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
		if !mergeIndices[i] {
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
		if mergeIndices[i] {
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
	segSeq := m.NextSegSeq
	m.NextSegSeq++
	segName := fmt.Sprintf(segDirFmt, segSeq)
	segDir := filepath.Join(dir, segName)

	if err := mergeSegments(segs, segDir); err != nil {
		return err
	}

	newSeg, err := openSegmentReader(segDir)
	if err != nil {
		os.RemoveAll(segDir)
		return err
	}

	// Close and remove old segments
	for _, seg := range segs {
		oldDir := seg.dir
		seg.Close()
		os.RemoveAll(oldDir)
	}

	m.Segments = []string{segName}
	m.DocCount = uint64(newSeg.meta.DocCount)
	m.AvgDocLen = newSeg.meta.AvgDocLen

	*segments = []*segmentReader{newSeg}

	return saveIndexMeta(dir, m)
}
