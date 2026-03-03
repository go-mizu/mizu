package dahlia

import (
	"fmt"
	"sort"
	"testing"
)

func TestTermInfoPack(t *testing.T) {
	tests := []termInfo{
		{docFreq: 0, postingsOff: 0, hasPositions: false},
		{docFreq: 1, postingsOff: 0, hasPositions: true},
		{docFreq: 100000, postingsOff: 999999, hasPositions: true},
		{docFreq: 0x3FFFFFFF, postingsOff: 0xFFFFFFFF, hasPositions: true},
	}
	for _, ti := range tests {
		packed := packTermInfo(ti)
		unpacked := unpackTermInfo(packed)
		if unpacked != ti {
			t.Fatalf("pack/unpack: got %+v, want %+v", unpacked, ti)
		}
	}
}

func TestTermDict1000Terms(t *testing.T) {
	// Generate 1000 sorted terms
	terms := make([]string, 1000)
	for i := range terms {
		terms[i] = fmt.Sprintf("term_%05d", i)
	}
	sort.Strings(terms)

	// Build dictionary
	w, err := newTermDictWriter()
	if err != nil {
		t.Fatal(err)
	}
	for i, term := range terms {
		ti := termInfo{
			docFreq:      uint32(i + 1),
			postingsOff:  uint32(i * 100),
			hasPositions: i%2 == 0,
		}
		if err := w.add(term, ti); err != nil {
			t.Fatalf("add %q: %v", term, err)
		}
	}
	data, err := w.finish()
	if err != nil {
		t.Fatal(err)
	}

	// Read and verify
	r, err := openTermDictReader(data)
	if err != nil {
		t.Fatal(err)
	}
	defer r.close()

	for i, term := range terms {
		ti, found := r.lookup(term)
		if !found {
			t.Fatalf("term %q not found", term)
		}
		if ti.docFreq != uint32(i+1) {
			t.Fatalf("term %q: docFreq=%d, want %d", term, ti.docFreq, i+1)
		}
		if ti.postingsOff != uint32(i*100) {
			t.Fatalf("term %q: postingsOff=%d, want %d", term, ti.postingsOff, i*100)
		}
		if ti.hasPositions != (i%2 == 0) {
			t.Fatalf("term %q: hasPositions=%v, want %v", term, ti.hasPositions, i%2 == 0)
		}
	}

	// Verify missing term
	_, found := r.lookup("nonexistent_term")
	if found {
		t.Fatal("should not find nonexistent term")
	}
}
