package dahlia

import (
	"path/filepath"
	"testing"
)

func TestSegmentReaderWriteOpenLookup(t *testing.T) {
	dir := t.TempDir()
	segDir := filepath.Join(dir, "seg_00000001")

	// Write segment
	sw := newSegmentWriter()
	sw.addDoc("doc_fox", []byte("the quick brown fox jumps over the lazy dog"))
	sw.addDoc("doc_cat", []byte("the small gray cat sleeps on the warm mat"))
	sw.addDoc("doc_ml", []byte("machine learning algorithms process large datasets"))

	_, err := sw.flush(segDir)
	if err != nil {
		t.Fatal(err)
	}

	// Open and query
	sr, err := openSegmentReader(segDir)
	if err != nil {
		t.Fatal(err)
	}
	defer sr.Close()

	if sr.meta.DocCount != 3 {
		t.Fatalf("DocCount=%d, want 3", sr.meta.DocCount)
	}

	// Lookup "fox" (stemmed)
	foxTerm := analyze("fox")[0]
	it, ti, found := sr.lookupTerm(foxTerm)
	if !found {
		t.Fatalf("term %q not found", foxTerm)
	}
	if ti.docFreq != 1 {
		t.Fatalf("fox docFreq=%d, want 1", ti.docFreq)
	}
	if !it.next() {
		t.Fatal("no docs for fox")
	}
	if it.doc() != 0 { // doc_fox is localID 0
		t.Fatalf("fox doc=%d, want 0", it.doc())
	}

	// Retrieve stored doc
	id, text, err := sr.getDoc(0)
	if err != nil {
		t.Fatal(err)
	}
	if id != "doc_fox" {
		t.Fatalf("doc 0 id=%q, want doc_fox", id)
	}
	if len(text) == 0 {
		t.Fatal("doc 0 has empty text")
	}

	// Field norm
	norm := sr.fieldNorm(0)
	if norm == 0 {
		t.Fatal("field norm should not be 0 for a non-empty doc")
	}
}

func TestSegmentReaderMissingTerm(t *testing.T) {
	dir := t.TempDir()
	segDir := filepath.Join(dir, "seg_00000001")

	sw := newSegmentWriter()
	sw.addDoc("doc1", []byte("hello world"))
	sw.flush(segDir)

	sr, err := openSegmentReader(segDir)
	if err != nil {
		t.Fatal(err)
	}
	defer sr.Close()

	_, _, found := sr.lookupTerm("nonexistent_xyzzy")
	if found {
		t.Fatal("should not find nonexistent term")
	}
}
