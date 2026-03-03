package rose

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// buildMem creates a simple in-memory posting map with the given terms and
// a fixed set of document IDs for each term.  docIDs must be sorted ascending.
func buildMem(terms []string, docIDsPerTerm [][]uint32) map[string][]memPosting {
	m := make(map[string][]memPosting, len(terms))
	for i, term := range terms {
		postings := make([]memPosting, len(docIDsPerTerm[i]))
		for j, id := range docIDsPerTerm[i] {
			postings[j] = memPosting{docID: id}
		}
		m[term] = postings
	}
	return m
}

// tmpSeg flushes a segment to a temp dir and returns the path.
func tmpSeg(t *testing.T, mem map[string][]memPosting, docCount, avgDocLen uint32) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.seg")
	if err := flushSegment(path, mem, docCount, avgDocLen); err != nil {
		t.Fatalf("flushSegment: %v", err)
	}
	return path
}

// ---------------------------------------------------------------------------
// Test 1: round-trip header fields
// ---------------------------------------------------------------------------

func TestFlushAndOpen_RoundTrip(t *testing.T) {
	// 3 terms, 10 docs spread across them.
	terms := []string{"alpha", "beta", "gamma"}
	docIDsPerTerm := [][]uint32{
		{0, 1, 2, 3},
		{2, 5, 7},
		{1, 4, 6, 8, 9},
	}
	mem := buildMem(terms, docIDsPerTerm)

	path := tmpSeg(t, mem, 10, 50)

	dict, _, docCount, avgDocLen, err := openSegment(path)
	if err != nil {
		t.Fatalf("openSegment: %v", err)
	}

	if docCount != 10 {
		t.Errorf("docCount: got %d, want 10", docCount)
	}
	if avgDocLen != 50 {
		t.Errorf("avgDocLen: got %d, want 50", avgDocLen)
	}
	if len(dict) != 3 {
		t.Fatalf("dictSize: got %d, want 3", len(dict))
	}

	// Verify all 3 terms are present (in any order that preserves lex sort).
	got := make(map[string]bool)
	for _, te := range dict {
		got[te.term] = true
	}
	for _, term := range terms {
		if !got[term] {
			t.Errorf("term %q missing from dictionary", term)
		}
	}

	// Verify df values.
	wantDF := map[string]uint32{"alpha": 4, "beta": 3, "gamma": 5}
	for _, te := range dict {
		if te.df != wantDF[te.term] {
			t.Errorf("term %q df: got %d, want %d", te.term, te.df, wantDF[te.term])
		}
	}
}

// ---------------------------------------------------------------------------
// Test 2: small posting list — exact docIDs + impacts in [1,255]
// ---------------------------------------------------------------------------

func TestReadPostings_SmallList(t *testing.T) {
	ids := []uint32{3, 7, 12, 55, 100}
	mem := buildMem([]string{"hello"}, [][]uint32{ids})

	path := tmpSeg(t, mem, 200, 80)

	dict, postingData, _, _, err := openSegment(path)
	if err != nil {
		t.Fatalf("openSegment: %v", err)
	}
	if len(dict) != 1 {
		t.Fatalf("expected 1 dict entry, got %d", len(dict))
	}

	docIDs, impacts, err := readPostings(postingData, dict[0])
	if err != nil {
		t.Fatalf("readPostings: %v", err)
	}

	if len(docIDs) != len(ids) {
		t.Fatalf("docIDs len: got %d, want %d", len(docIDs), len(ids))
	}
	for i, want := range ids {
		if docIDs[i] != want {
			t.Errorf("docIDs[%d]: got %d, want %d", i, docIDs[i], want)
		}
	}

	if len(impacts) != len(ids) {
		t.Fatalf("impacts len: got %d, want %d", len(impacts), len(ids))
	}
	for i, imp := range impacts {
		if imp < 1 || imp > 255 {
			t.Errorf("impacts[%d] = %d out of range [1,255]", i, imp)
		}
	}
}

// ---------------------------------------------------------------------------
// Test 3: exactly one full block (128 postings)
// ---------------------------------------------------------------------------

func TestReadPostings_FullBlock(t *testing.T) {
	ids := make([]uint32, 128)
	for i := range ids {
		ids[i] = uint32(i * 3) // 0, 3, 6, …, 381 — non-trivial deltas
	}
	mem := buildMem([]string{"word"}, [][]uint32{ids})

	path := tmpSeg(t, mem, 500, 100)

	dict, postingData, _, _, err := openSegment(path)
	if err != nil {
		t.Fatalf("openSegment: %v", err)
	}

	docIDs, impacts, err := readPostings(postingData, dict[0])
	if err != nil {
		t.Fatalf("readPostings: %v", err)
	}

	if len(docIDs) != 128 {
		t.Fatalf("docIDs len: got %d, want 128", len(docIDs))
	}
	for i, want := range ids {
		if docIDs[i] != want {
			t.Errorf("docIDs[%d]: got %d, want %d", i, docIDs[i], want)
		}
	}

	if len(impacts) != 128 {
		t.Fatalf("impacts len: got %d, want 128", len(impacts))
	}
	for i, imp := range impacts {
		if imp < 1 {
			t.Errorf("impacts[%d] = %d, must be >= 1", i, imp)
		}
	}
}

// ---------------------------------------------------------------------------
// Test 4: multi-block posting list (200 postings → 2 blocks)
// ---------------------------------------------------------------------------

func TestReadPostings_MultiBlock(t *testing.T) {
	ids := make([]uint32, 200)
	for i := range ids {
		ids[i] = uint32(i * 7) // 0, 7, 14, …, 1393
	}
	mem := buildMem([]string{"term"}, [][]uint32{ids})

	path := tmpSeg(t, mem, 1000, 120)

	dict, postingData, _, _, err := openSegment(path)
	if err != nil {
		t.Fatalf("openSegment: %v", err)
	}

	if dict[0].numBlocks != 2 {
		t.Fatalf("numBlocks: got %d, want 2", dict[0].numBlocks)
	}

	docIDs, impacts, err := readPostings(postingData, dict[0])
	if err != nil {
		t.Fatalf("readPostings: %v", err)
	}

	if len(docIDs) != 200 {
		t.Fatalf("docIDs len: got %d, want 200", len(docIDs))
	}
	for i, want := range ids {
		if docIDs[i] != want {
			t.Errorf("docIDs[%d]: got %d, want %d", i, docIDs[i], want)
		}
	}

	if len(impacts) != 200 {
		t.Fatalf("impacts len: got %d, want 200", len(impacts))
	}
	for i, imp := range impacts {
		if imp < 1 {
			t.Errorf("impacts[%d] = %d, must be >= 1", i, imp)
		}
	}
}

// ---------------------------------------------------------------------------
// Test 5: magic + version bytes
// ---------------------------------------------------------------------------

func TestFlushSegment_MagicVersion(t *testing.T) {
	mem := buildMem([]string{"x"}, [][]uint32{{1}})
	path := tmpSeg(t, mem, 2, 10)

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(raw) < 5 {
		t.Fatalf("file too short: %d bytes", len(raw))
	}

	want := [5]byte{0x52, 0x4F, 0x53, 0x45, 0x01}
	for i, b := range want {
		if raw[i] != b {
			t.Errorf("byte[%d]: got 0x%02x, want 0x%02x", i, raw[i], b)
		}
	}
}

// ---------------------------------------------------------------------------
// Test 6: bad magic bytes → descriptive error
// ---------------------------------------------------------------------------

func TestOpenSegment_BadMagic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.seg")

	// Write a file whose first 4 bytes are wrong.
	bad := []byte{0x00, 0x00, 0x00, 0x00, 0x01}
	if err := os.WriteFile(path, bad, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, _, _, _, err := openSegment(path)
	if err == nil {
		t.Fatal("expected error on bad magic, got nil")
	}
	t.Logf("got expected error: %v", err)

	// The error must mention "magic".
	errMsg := err.Error()
	found := false
	for i := 0; i < len(errMsg)-4; i++ {
		if errMsg[i] == 'm' && errMsg[i+1] == 'a' && errMsg[i+2] == 'g' && errMsg[i+3] == 'i' {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("error message does not mention 'magi...': %q", errMsg)
	}
}

// ---------------------------------------------------------------------------
// Test 7: dict entries in lexicographic order regardless of input map order
// ---------------------------------------------------------------------------

func TestFlushSegment_TermsSorted(t *testing.T) {
	// Provide terms in reverse-alphabetical order via the map.
	// Go maps iterate in random order, so the output must be sorted.
	termNames := []string{"zebra", "mango", "apple", "cherry", "banana"}
	docIDsPerTerm := [][]uint32{
		{10, 20},
		{5, 15},
		{1, 2, 3},
		{7, 8},
		{4, 9},
	}
	mem := buildMem(termNames, docIDsPerTerm)
	path := tmpSeg(t, mem, 25, 40)

	dict, _, _, _, err := openSegment(path)
	if err != nil {
		t.Fatalf("openSegment: %v", err)
	}

	if len(dict) != len(termNames) {
		t.Fatalf("dict size: got %d, want %d", len(dict), len(termNames))
	}

	gotTerms := make([]string, len(dict))
	for i, te := range dict {
		gotTerms[i] = te.term
	}

	wantTerms := make([]string, len(termNames))
	copy(wantTerms, termNames)
	sort.Strings(wantTerms)

	for i := range wantTerms {
		if gotTerms[i] != wantTerms[i] {
			t.Errorf("dict[%d]: got %q, want %q", i, gotTerms[i], wantTerms[i])
		}
	}
}
