package dahlia

import (
	"fmt"
	"path/filepath"
	"testing"
)

func TestMergeSegments(t *testing.T) {
	dir := t.TempDir()

	// Create 4 small segments
	var readers []*segmentReader
	for s := 0; s < 4; s++ {
		segDir := filepath.Join(dir, fmt.Sprintf("seg_%08d", s+1))
		sw := newSegmentWriter()
		for i := 0; i < 25; i++ {
			id := fmt.Sprintf("seg%d_doc%d", s, i)
			text := fmt.Sprintf("document %d from segment %d about topic %d", i, s, i%5)
			sw.addDoc(id, []byte(text))
		}
		sw.flush(segDir)
		sr, err := openSegmentReader(segDir)
		if err != nil {
			t.Fatal(err)
		}
		readers = append(readers, sr)
		defer sr.Close()
	}

	// Merge into one
	mergedDir := filepath.Join(dir, "seg_merged")
	if err := mergeSegments(readers, mergedDir); err != nil {
		t.Fatal(err)
	}

	merged, err := openSegmentReader(mergedDir)
	if err != nil {
		t.Fatal(err)
	}
	defer merged.Close()

	if merged.meta.DocCount != 100 {
		t.Fatalf("merged DocCount=%d, want 100", merged.meta.DocCount)
	}

	// Verify we can search the merged segment
	q := parseQuery("document")
	eval := newWandEvaluator(merged, 10, 0, 0, nil)
	results := eval.searchQuery(q)
	if len(results) == 0 {
		t.Fatal("expected results from merged segment")
	}

	// Verify all docs are retrievable
	for docID := uint32(0); docID < merged.meta.DocCount; docID++ {
		id, _, err := merged.getDoc(docID)
		if err != nil {
			t.Fatalf("getDoc(%d): %v", docID, err)
		}
		if id == "" {
			t.Fatalf("empty doc ID at %d", docID)
		}
	}
}

func TestMergeEmpty(t *testing.T) {
	dir := t.TempDir()
	mergedDir := filepath.Join(dir, "seg_merged")
	err := mergeSegments(nil, mergedDir)
	if err == nil {
		t.Fatal("expected error merging empty")
	}
}

func TestPickMergeIndicesRespectsBudget(t *testing.T) {
	segs := []*segmentReader{
		{meta: segmentMeta{DocCount: 1000}},
		{meta: segmentMeta{DocCount: 1200}},
		{meta: segmentMeta{DocCount: 2000}},
		{meta: segmentMeta{DocCount: 5000}},
	}

	idx := pickMergeIndices(segs, 10, 2500, false)
	if len(idx) != 2 {
		t.Fatalf("picked=%v, want 2 segments under budget", idx)
	}
	if idx[0] != 0 || idx[1] != 1 {
		t.Fatalf("picked=%v, want [0 1]", idx)
	}
}

func TestPickMergeIndicesForceProgress(t *testing.T) {
	segs := []*segmentReader{
		{meta: segmentMeta{DocCount: 10_000}},
		{meta: segmentMeta{DocCount: 11_000}},
		{meta: segmentMeta{DocCount: 12_000}},
	}

	idx := pickMergeIndices(segs, 10, 1, true)
	if len(idx) != 2 {
		t.Fatalf("picked=%v, want fallback 2 segments", idx)
	}
}
