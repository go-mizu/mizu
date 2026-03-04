package lotus

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// mergePolicy decides when segments should be merged.
const (
	mergeMinSegments = 3  // merge when >= 3 segments at same tier
	mergeCheckPeriod = 5 * time.Second
)

// mergeWorker runs in the background, merging segments when the tiered policy triggers.
type mergeWorker struct {
	dir    string
	mu     sync.Mutex
	stopCh chan struct{}
	doneCh chan struct{}
}

func newMergeWorker(dir string) *mergeWorker {
	return &mergeWorker{
		dir:    dir,
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
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
	ticker := time.NewTicker(mergeCheckPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-mw.stopCh:
			return
		case <-ticker.C:
			mw.tryMerge()
		}
	}
}

func (mw *mergeWorker) tryMerge() {
	mw.mu.Lock()
	defer mw.mu.Unlock()

	meta, err := loadIndexMeta(mw.dir)
	if err != nil || len(meta.Segments) < mergeMinSegments {
		return
	}

	// Simple tiered policy: group segments by doc count tier (order of magnitude)
	type segInfo struct {
		name     string
		docCount uint32
		tier     int
	}

	var segs []segInfo
	for _, name := range meta.Segments {
		sm, err := loadSegmentMeta(filepath.Join(mw.dir, name))
		if err != nil {
			continue
		}
		tier := docCountTier(sm.DocCount)
		segs = append(segs, segInfo{name: name, docCount: sm.DocCount, tier: tier})
	}

	// Group by tier
	tierGroups := make(map[int][]segInfo)
	for _, s := range segs {
		tierGroups[s.tier] = append(tierGroups[s.tier], s)
	}

	// Merge groups with >= mergeMinSegments
	for _, group := range tierGroups {
		if len(group) < mergeMinSegments {
			continue
		}
		// Merge this group
		segNames := make([]string, len(group))
		for i, s := range group {
			segNames[i] = s.name
		}
		newName := nextSegmentName(meta)
		if err := mergeSegments(mw.dir, segNames, newName); err != nil {
			continue
		}

		// Update meta: remove old segments, add new one
		remaining := make([]string, 0, len(meta.Segments))
		mergedSet := make(map[string]bool)
		for _, s := range segNames {
			mergedSet[s] = true
		}
		for _, s := range meta.Segments {
			if !mergedSet[s] {
				remaining = append(remaining, s)
			}
		}
		remaining = append(remaining, newName)
		meta.Segments = remaining
		saveIndexMeta(mw.dir, meta)

		// Clean up old segment directories
		for _, s := range segNames {
			os.RemoveAll(filepath.Join(mw.dir, s))
		}

		break // one merge per tick
	}
}

func docCountTier(docCount uint32) int {
	if docCount == 0 {
		return 0
	}
	tier := 0
	n := docCount
	for n >= 10 {
		n /= 10
		tier++
	}
	return tier
}

func loadSegmentMeta(dir string) (*segmentMeta, error) {
	data, err := os.ReadFile(filepath.Join(dir, "segment.meta"))
	if err != nil {
		return nil, err
	}
	var m segmentMeta
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// mergeSegments merges multiple segments into a new one.
func mergeSegments(baseDir string, segNames []string, newName string) error {
	// Open all source segments
	readers := make([]*segmentReader, len(segNames))
	for i, name := range segNames {
		r, err := openSegmentReader(filepath.Join(baseDir, name))
		if err != nil {
			for j := 0; j < i; j++ {
				readers[j].close()
			}
			return fmt.Errorf("open segment %s: %w", name, err)
		}
		readers[i] = r
	}
	defer func() {
		for _, r := range readers {
			r.close()
		}
	}()

	// Create new segment writer
	newDir := filepath.Join(baseDir, newName)
	writer, err := newSegmentWriter(newDir)
	if err != nil {
		return fmt.Errorf("create merge segment: %w", err)
	}

	// Iterate all docs from all source segments in order
	for _, r := range readers {
		for docID := uint32(0); docID < r.meta.DocCount; docID++ {
			id, text, err := r.getDoc(docID)
			if err != nil {
				continue // skip docs we can't read
			}
			if err := writer.addDoc(id, text); err != nil {
				return fmt.Errorf("merge add doc: %w", err)
			}
		}
	}

	return writer.flush()
}

// mergeAllSegments forces a merge of all segments (used for compaction).
func mergeAllSegments(dir string) error {
	meta, err := loadIndexMeta(dir)
	if err != nil || len(meta.Segments) <= 1 {
		return err
	}

	newName := nextSegmentName(meta)
	if err := mergeSegments(dir, meta.Segments, newName); err != nil {
		return err
	}

	// Clean up old segments
	oldSegs := meta.Segments
	meta.Segments = []string{newName}
	if err := saveIndexMeta(dir, meta); err != nil {
		return err
	}
	for _, s := range oldSegs {
		os.RemoveAll(filepath.Join(dir, s))
	}
	return nil
}

// Used by merge to keep merged results sorted by docID
type mergeDocEntry struct {
	segIdx int
	docID  uint32
}

type mergeDocHeap []mergeDocEntry

func (h mergeDocHeap) Len() int            { return len(h) }
func (h mergeDocHeap) Less(i, j int) bool  { return h[i].docID < h[j].docID }
func (h mergeDocHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *mergeDocHeap) Push(x interface{}) { *h = append(*h, x.(mergeDocEntry)) }
func (h *mergeDocHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

// collectAllTerms returns sorted unique terms across all segments.
func collectAllTerms(readers []*segmentReader) []string {
	termSet := make(map[string]struct{})
	for _, r := range readers {
		if r.termDict.fst == nil {
			continue
		}
		it, _ := r.termDict.fst.Iterator(nil, nil)
		for {
			key, _ := it.Current()
			if key == nil {
				break
			}
			termSet[string(key)] = struct{}{}
			if err := it.Next(); err != nil {
				break
			}
		}
	}
	terms := make([]string, 0, len(termSet))
	for t := range termSet {
		terms = append(terms, t)
	}
	sort.Strings(terms)
	return terms
}
