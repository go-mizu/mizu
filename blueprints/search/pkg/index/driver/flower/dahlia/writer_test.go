package dahlia

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestSegmentWriterFlush(t *testing.T) {
	dir := t.TempDir()
	segDir := filepath.Join(dir, "seg_00000001")

	sw := newSegmentWriter()
	sw.addDoc("doc1", []byte("the quick brown fox jumps over the lazy dog"))
	sw.addDoc("doc2", []byte("a fast red fox leaps across the sleeping hound"))
	sw.addDoc("doc3", []byte("machine learning and artificial intelligence research"))

	meta, err := sw.flush(segDir)
	if err != nil {
		t.Fatal(err)
	}

	if meta.DocCount != 3 {
		t.Fatalf("DocCount=%d, want 3", meta.DocCount)
	}
	if meta.AvgDocLen <= 0 {
		t.Fatalf("AvgDocLen=%f, want > 0", meta.AvgDocLen)
	}

	// Verify all files exist
	for _, name := range []string{segMetaFile, segTermDictFile, segDocFile, segFreqFile, segPosFile, segStoreFile, segFieldNormFile} {
		path := filepath.Join(segDir, name)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("missing file %s: %v", name, err)
		}
	}
}

func TestSegmentWriterLarge(t *testing.T) {
	dir := t.TempDir()
	segDir := filepath.Join(dir, "seg_00000001")

	sw := newSegmentWriter()
	for i := 0; i < 500; i++ {
		id := fmt.Sprintf("doc_%06d", i)
		text := fmt.Sprintf("document number %d about topic %d with words foo bar baz qux %d research science", i, i%10, i*7)
		sw.addDoc(id, []byte(text))
	}

	meta, err := sw.flush(segDir)
	if err != nil {
		t.Fatal(err)
	}

	if meta.DocCount != 500 {
		t.Fatalf("DocCount=%d, want 500", meta.DocCount)
	}
}

func TestSegmentWriterEmpty(t *testing.T) {
	dir := t.TempDir()
	segDir := filepath.Join(dir, "seg_empty")

	sw := newSegmentWriter()
	_, err := sw.flush(segDir)
	if err == nil {
		t.Fatal("expected error for empty segment")
	}
}

func TestIndexMeta(t *testing.T) {
	dir := t.TempDir()

	// Load non-existent → fresh default
	meta, err := loadIndexMeta(dir)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Version != 1 || meta.NextSegSeq != 1 {
		t.Fatalf("unexpected default meta: %+v", meta)
	}

	// Save and reload
	meta.DocCount = 1000
	meta.AvgDocLen = 42.5
	meta.Segments = []string{"seg_00000001", "seg_00000002"}
	meta.NextSegSeq = 3
	if err := saveIndexMeta(dir, meta); err != nil {
		t.Fatal(err)
	}

	loaded, err := loadIndexMeta(dir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.DocCount != 1000 || loaded.NextSegSeq != 3 || len(loaded.Segments) != 2 {
		t.Fatalf("meta mismatch: %+v", loaded)
	}
}
