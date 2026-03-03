package rose

import (
	"path/filepath"
	"testing"
	"unicode/utf8"
)

func TestDocStore_AppendAndGet(t *testing.T) {
	ds, _ := openDocStore(filepath.Join(t.TempDir(), "rose.docs"))
	defer ds.close()
	idx0, _ := ds.append("ext-0", []byte("hello world machine learning"))
	idx1, _ := ds.append("ext-1", []byte("climate change energy"))
	if idx0 != 0 || idx1 != 1 {
		t.Errorf("indices %d %d", idx0, idx1)
	}
	e0, _ := ds.get(0)
	if e0.externalID != "ext-0" {
		t.Errorf("got %q", e0.externalID)
	}
	e1, _ := ds.get(1)
	if e1.externalID != "ext-1" {
		t.Errorf("got %q", e1.externalID)
	}
}

func TestDocStore_TextTruncation(t *testing.T) {
	ds, _ := openDocStore(filepath.Join(t.TempDir(), "rose.docs"))
	defer ds.close()
	long := make([]byte, 1000)
	for i := range long {
		long[i] = 'a'
	}
	ds.append("d", long)
	e, _ := ds.get(0)
	if len(e.text) > docStoreMaxText {
		t.Errorf("not truncated: %d > %d", len(e.text), docStoreMaxText)
	}
}

func TestDocStore_Reopen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rose.docs")
	ds, _ := openDocStore(path)
	ds.append("id-a", []byte("foo bar"))
	ds.append("id-b", []byte("qux quux"))
	ds.close()
	ds2, _ := openDocStore(path)
	defer ds2.close()
	e, _ := ds2.get(0)
	if e.externalID != "id-a" {
		t.Errorf("after reopen: got %q", e.externalID)
	}
	if len(ds2.entries) != 2 {
		t.Errorf("expected 2 entries after reopen, got %d", len(ds2.entries))
	}
}

func TestDocStore_GetOutOfRange(t *testing.T) {
	ds, _ := openDocStore(filepath.Join(t.TempDir(), "rose.docs"))
	defer ds.close()
	_, err := ds.get(0)
	if err == nil {
		t.Error("get on empty store should return error")
	}
}

func TestDocStore_EmptyText(t *testing.T) {
	ds, _ := openDocStore(filepath.Join(t.TempDir(), "rose.docs"))
	defer ds.close()
	idx, err := ds.append("empty", []byte{})
	if err != nil {
		t.Fatalf("append empty text: %v", err)
	}
	e, _ := ds.get(idx)
	if len(e.text) != 0 {
		t.Error("empty text should stay empty")
	}
}

func TestDocStore_TextTruncationUTF8Safe(t *testing.T) {
	ds, _ := openDocStore(filepath.Join(t.TempDir(), "rose.docs"))
	defer ds.close()
	// Build a 600-byte string where bytes 510-512 are part of a 3-byte rune (€ = 0xE2 0x82 0xAC)
	// The truncation at 512 would split the rune if not UTF-8-aware
	base := make([]byte, 510)
	for i := range base {
		base[i] = 'a'
	}
	euroSign := []byte{0xE2, 0x82, 0xAC} // €
	text := append(base, euroSign...)     // 513 bytes, € starts at 510
	text = append(text, make([]byte, 87)...) // pad to > 512

	ds.append("utf8doc", text)
	e, _ := ds.get(0)
	if !utf8.Valid(e.text) {
		t.Errorf("stored text is not valid UTF-8: last bytes %v", e.text[len(e.text)-3:])
	}
	if len(e.text) > docStoreMaxText {
		t.Errorf("text exceeds max: %d", len(e.text))
	}
}

func TestSnippetFor_Hit(t *testing.T) {
	text := []byte("The quick brown fox jumps over lazy dogs near machine learning systems")
	// "machin" and "learn" are pre-stemmed query terms
	snip := snippetFor(text, []string{"machin", "learn"})
	if snip == "" {
		t.Error("expected non-empty snippet")
	}
}

func TestSnippetFor_NoMatch(t *testing.T) {
	text := []byte("completely unrelated content here about nothing special")
	snip := snippetFor(text, []string{"machin"})
	// Should return first 20 words (or all if fewer)
	if snip == "" {
		t.Error("no-match snippet should still return something")
	}
}

func TestSnippetFor_EmptyText(t *testing.T) {
	snip := snippetFor([]byte{}, []string{"machin"})
	if snip != "" {
		t.Errorf("empty text snippet should be empty, got %q", snip)
	}
}
