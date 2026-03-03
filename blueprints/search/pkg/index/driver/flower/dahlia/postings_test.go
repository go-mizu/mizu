package dahlia

import "testing"

func TestPostingsWriteReadCycle(t *testing.T) {
	pw := &postingsWriter{}

	// Create a term with 200 docs (1 full block + 72 tail)
	tp := &termPostings{
		docs:      make([]uint32, 200),
		freqs:     make([]uint32, 200),
		norms:     make([]uint8, 200),
		positions: make([][]uint32, 200),
	}
	for i := 0; i < 200; i++ {
		tp.docs[i] = uint32(i * 3)         // 0, 3, 6, ...
		tp.freqs[i] = uint32(i%5 + 1)      // 1-5
		tp.norms[i] = encodeFieldNorm(uint32(50 + i%100))
		tp.positions[i] = []uint32{uint32(i), uint32(i + 10)}
	}

	docOff := pw.writeTerm(tp)

	// Create iterator
	it := newPostingIterator(pw.docBytes(), pw.freqBytes(), pw.posBytes(), docOff)

	// Verify all docs
	count := 0
	for it.next() {
		if it.doc() != tp.docs[count] {
			t.Fatalf("doc %d: got %d, want %d", count, it.doc(), tp.docs[count])
		}
		if it.freq() != tp.freqs[count] {
			t.Fatalf("freq %d: got %d, want %d", count, it.freq(), tp.freqs[count])
		}
		count++
	}
	if count != 200 {
		t.Fatalf("iterated %d docs, want 200", count)
	}
	if it.doc() != noMoreDocs {
		t.Fatalf("after exhaustion, doc should be noMoreDocs")
	}
}

func TestPostingsAdvance(t *testing.T) {
	pw := &postingsWriter{}

	// 300 docs spanning multiple blocks
	tp := &termPostings{
		docs:  make([]uint32, 300),
		freqs: make([]uint32, 300),
		norms: make([]uint8, 300),
	}
	for i := 0; i < 300; i++ {
		tp.docs[i] = uint32(i * 2) // 0, 2, 4, ..., 598
		tp.freqs[i] = 1
		tp.norms[i] = encodeFieldNorm(100)
	}

	docOff := pw.writeTerm(tp)
	it := newPostingIterator(pw.docBytes(), pw.freqBytes(), pw.posBytes(), docOff)

	// Advance to doc 100 (should be at index 50)
	if !it.advance(100) {
		t.Fatal("advance(100) failed")
	}
	if it.doc() != 100 {
		t.Fatalf("advance(100): got doc %d", it.doc())
	}

	// Advance to doc 401 (should land on 402 = index 201)
	if !it.advance(401) {
		t.Fatal("advance(401) failed")
	}
	if it.doc() != 402 {
		t.Fatalf("advance(401): got doc %d, want 402", it.doc())
	}

	// Advance past end
	if it.advance(600) {
		t.Fatal("advance(600) should fail")
	}
}

func TestPostingsPositions(t *testing.T) {
	pw := &postingsWriter{}

	tp := &termPostings{
		docs:      []uint32{10, 20, 30},
		freqs:     []uint32{2, 3, 1},
		norms:     []uint8{50, 60, 70},
		positions: [][]uint32{{5, 15}, {1, 8, 20}, {42}},
	}

	docOff := pw.writeTerm(tp)
	it := newPostingIterator(pw.docBytes(), pw.freqBytes(), pw.posBytes(), docOff)

	// Doc 10
	if !it.next() {
		t.Fatal("next() for doc 10 failed")
	}
	pos := it.positions()
	if len(pos) != 2 || pos[0] != 5 || pos[1] != 15 {
		t.Fatalf("doc 10 positions: got %v, want [5 15]", pos)
	}

	// Doc 20
	if !it.next() {
		t.Fatal("next() for doc 20 failed")
	}
	pos = it.positions()
	if len(pos) != 3 || pos[0] != 1 || pos[1] != 8 || pos[2] != 20 {
		t.Fatalf("doc 20 positions: got %v, want [1 8 20]", pos)
	}

	// Doc 30
	if !it.next() {
		t.Fatal("next() for doc 30 failed")
	}
	pos = it.positions()
	if len(pos) != 1 || pos[0] != 42 {
		t.Fatalf("doc 30 positions: got %v, want [42]", pos)
	}
}

func TestPostingsSmall(t *testing.T) {
	pw := &postingsWriter{}

	// Only 3 docs (tail only, no full blocks)
	tp := &termPostings{
		docs:  []uint32{5, 10, 15},
		freqs: []uint32{1, 2, 3},
		norms: []uint8{50, 50, 50},
	}

	docOff := pw.writeTerm(tp)
	it := newPostingIterator(pw.docBytes(), pw.freqBytes(), pw.posBytes(), docOff)

	for i, wantDoc := range tp.docs {
		if !it.next() {
			t.Fatalf("next() %d failed", i)
		}
		if it.doc() != wantDoc {
			t.Fatalf("doc %d: got %d, want %d", i, it.doc(), wantDoc)
		}
		if it.freq() != tp.freqs[i] {
			t.Fatalf("freq %d: got %d, want %d", i, it.freq(), tp.freqs[i])
		}
	}
	if it.next() {
		t.Fatal("should be exhausted")
	}
}

func TestPostingsExactBlock(t *testing.T) {
	pw := &postingsWriter{}

	// Exactly 128 docs (1 full block, no tail)
	tp := &termPostings{
		docs:  make([]uint32, blockSize),
		freqs: make([]uint32, blockSize),
		norms: make([]uint8, blockSize),
	}
	for i := 0; i < blockSize; i++ {
		tp.docs[i] = uint32(i)
		tp.freqs[i] = 1
		tp.norms[i] = 50
	}

	docOff := pw.writeTerm(tp)
	it := newPostingIterator(pw.docBytes(), pw.freqBytes(), pw.posBytes(), docOff)

	count := 0
	for it.next() {
		if it.doc() != uint32(count) {
			t.Fatalf("doc %d: got %d", count, it.doc())
		}
		count++
	}
	if count != blockSize {
		t.Fatalf("iterated %d docs, want %d", count, blockSize)
	}
}

func TestPostingsMultipleTerms(t *testing.T) {
	pw := &postingsWriter{}

	// Write two terms
	tp1 := &termPostings{
		docs:  []uint32{1, 2, 3},
		freqs: []uint32{1, 1, 1},
		norms: []uint8{50, 50, 50},
	}
	tp2 := &termPostings{
		docs:  []uint32{2, 4, 6, 8},
		freqs: []uint32{2, 2, 2, 2},
		norms: []uint8{60, 60, 60, 60},
	}

	off1 := pw.writeTerm(tp1)
	off2 := pw.writeTerm(tp2)

	docData := pw.docBytes()
	freqData := pw.freqBytes()
	posData := pw.posBytes()

	// Verify term 1
	it1 := newPostingIterator(docData, freqData, posData, off1)
	for i, wantDoc := range tp1.docs {
		if !it1.next() {
			t.Fatalf("term1 doc %d: next failed", i)
		}
		if it1.doc() != wantDoc {
			t.Fatalf("term1 doc %d: got %d, want %d", i, it1.doc(), wantDoc)
		}
	}

	// Verify term 2
	it2 := newPostingIterator(docData, freqData, posData, off2)
	for i, wantDoc := range tp2.docs {
		if !it2.next() {
			t.Fatalf("term2 doc %d: next failed", i)
		}
		if it2.doc() != wantDoc {
			t.Fatalf("term2 doc %d: got %d, want %d", i, it2.doc(), wantDoc)
		}
		if it2.freq() != 2 {
			t.Fatalf("term2 doc %d: freq=%d, want 2", i, it2.freq())
		}
	}
}
