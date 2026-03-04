package dahlia

import (
	"bytes"
	"encoding/binary"
)

// postingsWriter accumulates posting data for all terms in a segment,
// writing to separate .doc, .freq, and .pos buffers.
type postingsWriter struct {
	docBuf  bytes.Buffer
	freqBuf bytes.Buffer
	posBuf  bytes.Buffer
}

type termPostings struct {
	docs      []uint32
	freqs     []uint32
	norms     []uint8
	positions [][]uint32 // per-doc positions
}

// writeTerm writes one term's posting list to the three buffers.
// Returns the byte offset in docBuf where this term's blob starts.
//
// Blob format in .doc file:
//
//	[blobSize:4][freqOff:4][posOff:4]
//	[BP128 doc-delta blocks...]
//	[VInt doc-delta tail...]
//	[skip entries...]
//	[trailer: skipRelOff:4, numFullBlocks:4, tailCount:4]
func (pw *postingsWriter) writeTerm(tp *termPostings) uint32 {
	docStartOff := uint32(pw.docBuf.Len())
	freqStartOff := uint32(pw.freqBuf.Len())
	posStartOff := uint32(pw.posBuf.Len())

	// Reserve 12 bytes: blobSize(4) + freqOff(4) + posOff(4)
	pw.docBuf.Write([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})

	n := len(tp.docs)
	numFull := n / blockSize
	tailCount := n % blockSize

	var skips []skipEntry
	prevDoc := uint32(0)

	// Write full BP128 blocks
	for b := 0; b < numFull; b++ {
		start := b * blockSize
		end := start + blockSize
		blockDocs := tp.docs[start:end]
		blockFreqs := tp.freqs[start:end]
		blockNorms := tp.norms[start:end]

		se := skipEntry{
			lastDoc: blockDocs[blockSize-1],
			docOff:  uint32(pw.docBuf.Len()) - docStartOff - 12, // relative to after header
			freqOff: uint32(pw.freqBuf.Len()) - freqStartOff,
			posOff:  uint32(pw.posBuf.Len()) - posStartOff,
		}

		// Write doc ID deltas (BP128)
		packed := bp128DocBlock(blockDocs, prevDoc)
		pw.docBuf.Write(packed)

		// Write freqs (BP128)
		freqPacked := bp128FreqBlock(blockFreqs)
		pw.freqBuf.Write(freqPacked)

		// Write positions + compute block-max stats
		var maxTF uint32
		minNorm := blockNorms[0]
		for i := start; i < end; i++ {
			if tp.freqs[i] > maxTF {
				maxTF = tp.freqs[i]
			}
			if tp.norms[i] < minNorm {
				minNorm = tp.norms[i]
			}
			if i < len(tp.positions) && len(tp.positions[i]) > 0 {
				pw.writePositions(tp.positions[i])
			}
		}
		se.blockMaxTF = maxTF
		se.blockMaxNorm = minNorm
		skips = append(skips, se)

		prevDoc = blockDocs[blockSize-1]
	}

	// Write VInt tail (doc deltas + freqs)
	tailStart := numFull * blockSize
	if tailCount > 0 {
		prev := prevDoc
		for i := tailStart; i < n; i++ {
			var buf [5]byte
			nb := vintEncode(buf[:], tp.docs[i]-prev)
			pw.docBuf.Write(buf[:nb])
			prev = tp.docs[i]

			nb = vintEncode(buf[:], tp.freqs[i])
			pw.freqBuf.Write(buf[:nb])

			if i < len(tp.positions) && len(tp.positions[i]) > 0 {
				pw.writePositions(tp.positions[i])
			}
		}
	}

	// Write skip entries to docBuf
	skipRelOff := uint32(pw.docBuf.Len()) - docStartOff - 12
	for _, se := range skips {
		var buf [skipEntrySize]byte
		encodeSkipEntry(buf[:], se)
		pw.docBuf.Write(buf[:])
	}

	// Write trailer: [skipRelOff:4][numFullBlocks:4][tailCount:4]
	var trailer [12]byte
	binary.LittleEndian.PutUint32(trailer[0:4], skipRelOff)
	binary.LittleEndian.PutUint32(trailer[4:8], uint32(numFull))
	binary.LittleEndian.PutUint32(trailer[8:12], uint32(tailCount))
	pw.docBuf.Write(trailer[:])

	// Fill in the header: blobSize + freqOff + posOff
	blobSize := uint32(pw.docBuf.Len()) - docStartOff - 4 // excludes the blobSize field itself
	header := pw.docBuf.Bytes()[docStartOff:]
	binary.LittleEndian.PutUint32(header[0:4], blobSize)
	binary.LittleEndian.PutUint32(header[4:8], freqStartOff)
	binary.LittleEndian.PutUint32(header[8:12], posStartOff)

	return docStartOff
}

func (pw *postingsWriter) writePositions(positions []uint32) {
	var buf [5]byte
	nb := vintEncode(buf[:], uint32(len(positions)))
	pw.posBuf.Write(buf[:nb])
	prev := uint32(0)
	for _, p := range positions {
		nb = vintEncode(buf[:], p-prev)
		pw.posBuf.Write(buf[:nb])
		prev = p
	}
}

func (pw *postingsWriter) docBytes() []byte  { return pw.docBuf.Bytes() }
func (pw *postingsWriter) freqBytes() []byte { return pw.freqBuf.Bytes() }
func (pw *postingsWriter) posBytes() []byte  { return pw.posBuf.Bytes() }

// postingIterator iterates over a term's posting list using mmap'd data.
type postingIterator struct {
	// Mmap'd file data
	docData  []byte
	freqData []byte
	posData  []byte

	// This term's data starts at these offsets in the respective files
	docBase  uint32 // after 12-byte header in .doc
	freqBase uint32
	posBase  uint32

	// From trailer
	skipRelOff    uint32
	numFullBlocks uint32
	tailCount     uint32

	// Decoded skip entries
	skips []skipEntry

	// Iteration state
	blockIdx     int
	inBlock      int
	blockDocs    [blockSize]uint32
	blockFreqs   [blockSize]uint32
	curDoc       uint32
	curFreq      uint32
	lastBlockDoc uint32

	// Read position (relative offsets from base)
	docReadOff  uint32
	freqReadOff uint32
	posReadOff  uint32

	blockLen  int
	inTail    bool
	tailDocs  []uint32
	tailFreqs []uint32
	exhausted bool

	// Position-state tracking for current doc.
	// posReadOff is forward-only; this ensures one doc's positions are consumed
	// exactly once (read or skipped) before moving on.
	curPosLoaded bool
	curPositions []uint32
}

// newPostingIterator creates an iterator for a term's posting list.
// docOff is the offset in docData where the term's blob starts (at the blobSize field).
func newPostingIterator(docData, freqData, posData []byte, docOff uint32) *postingIterator {
	// Read 12-byte header: [blobSize:4][freqOff:4][posOff:4]
	blobSize := binary.LittleEndian.Uint32(docData[docOff : docOff+4])
	freqOff := binary.LittleEndian.Uint32(docData[docOff+4 : docOff+8])
	posOff := binary.LittleEndian.Uint32(docData[docOff+8 : docOff+12])

	it := &postingIterator{
		docData:  docData,
		freqData: freqData,
		posData:  posData,
		docBase:  docOff + 12, // skip header
		freqBase: freqOff,
		posBase:  posOff,
		blockIdx: -1,
		curDoc:   noMoreDocs,
	}

	// Read trailer (last 12 bytes of the blob, after the 4-byte blobSize)
	blobEnd := docOff + 4 + blobSize
	trailerOff := blobEnd - 12
	it.skipRelOff = binary.LittleEndian.Uint32(docData[trailerOff : trailerOff+4])
	it.numFullBlocks = binary.LittleEndian.Uint32(docData[trailerOff+4 : trailerOff+8])
	it.tailCount = binary.LittleEndian.Uint32(docData[trailerOff+8 : trailerOff+12])

	// Load skip entries
	if it.numFullBlocks > 0 {
		skipStart := it.docBase + it.skipRelOff
		it.skips = make([]skipEntry, it.numFullBlocks)
		for i := uint32(0); i < it.numFullBlocks; i++ {
			off := skipStart + i*skipEntrySize
			it.skips[i] = decodeSkipEntry(docData[off : off+skipEntrySize])
		}
	}

	return it
}

// next advances to the next document. Returns false when exhausted.
func (it *postingIterator) next() bool {
	if it.exhausted {
		return false
	}
	it.skipCurrentPositions()

	it.inBlock++

	if it.blockIdx < 0 || it.inBlock >= it.blockLen {
		return it.nextBlock()
	}

	if it.inTail {
		it.curDoc = it.tailDocs[it.inBlock]
		it.curFreq = it.tailFreqs[it.inBlock]
	} else {
		it.curDoc = it.blockDocs[it.inBlock]
		it.curFreq = it.blockFreqs[it.inBlock]
	}
	it.curPosLoaded = false
	it.curPositions = nil
	return true
}

func (it *postingIterator) nextBlock() bool {
	it.blockIdx++

	if uint32(it.blockIdx) < it.numFullBlocks {
		// Decode full BP128 block
		docOff := it.docBase + it.docReadOff
		n := bp128Unpack(it.docData[docOff:], it.blockDocs[:])
		it.docReadOff += uint32(n)

		// Convert deltas to absolute doc IDs
		prev := it.lastBlockDoc
		for i := 0; i < blockSize; i++ {
			it.blockDocs[i] += prev
			prev = it.blockDocs[i]
		}

		// Decode freqs
		freqOff := it.freqBase + it.freqReadOff
		fn := bp128Unpack(it.freqData[freqOff:], it.blockFreqs[:])
		it.freqReadOff += uint32(fn)

		it.lastBlockDoc = it.blockDocs[blockSize-1]
		it.blockLen = blockSize
		it.inBlock = 0
		it.inTail = false
		it.curDoc = it.blockDocs[0]
		it.curFreq = it.blockFreqs[0]
		it.curPosLoaded = false
		it.curPositions = nil
		return true
	}

	// Try tail
	if it.tailCount > 0 && !it.inTail {
		it.decodeTail()
		if len(it.tailDocs) > 0 {
			it.inTail = true
			it.blockLen = len(it.tailDocs)
			it.inBlock = 0
			it.curDoc = it.tailDocs[0]
			it.curFreq = it.tailFreqs[0]
			it.curPosLoaded = false
			it.curPositions = nil
			return true
		}
	}

	it.exhausted = true
	it.curDoc = noMoreDocs
	it.curPosLoaded = false
	it.curPositions = nil
	return false
}

func (it *postingIterator) decodeTail() {
	it.tailDocs = make([]uint32, 0, it.tailCount)
	it.tailFreqs = make([]uint32, 0, it.tailCount)

	docOff := it.docBase + it.docReadOff
	freqOff := it.freqBase + it.freqReadOff
	prev := it.lastBlockDoc

	for i := uint32(0); i < it.tailCount; i++ {
		delta, n := vintGet(it.docData[docOff:])
		docOff += uint32(n)
		prev += delta
		it.tailDocs = append(it.tailDocs, prev)

		freq, fn := vintGet(it.freqData[freqOff:])
		freqOff += uint32(fn)
		it.tailFreqs = append(it.tailFreqs, freq)
	}
}

// advance moves the iterator to the first doc >= target.
func (it *postingIterator) advance(target uint32) bool {
	if it.exhausted {
		return false
	}

	if it.curDoc != noMoreDocs && it.curDoc >= target {
		return true
	}

	// Use skip entries for block-level seeking
	if len(it.skips) > 0 {
		si := skipIndex(it.skips)
		blockIdx := si.findBlock(target)
		if blockIdx >= 0 && blockIdx > it.blockIdx {
			it.seekToBlock(blockIdx)
		}
	}

	for {
		if it.curDoc != noMoreDocs && it.curDoc >= target {
			return true
		}
		if !it.next() {
			return false
		}
	}
}

func (it *postingIterator) seekToBlock(idx int) {
	if idx >= len(it.skips) {
		return
	}
	se := it.skips[idx]

	it.docReadOff = se.docOff
	it.freqReadOff = se.freqOff
	it.posReadOff = se.posOff

	if idx > 0 {
		it.lastBlockDoc = it.skips[idx-1].lastDoc
	} else {
		it.lastBlockDoc = 0
	}

	it.blockIdx = idx - 1
	it.inBlock = blockSize
	it.blockLen = 0
	it.inTail = false
	it.curDoc = noMoreDocs
	it.curFreq = 0
	it.curPosLoaded = false
	it.curPositions = nil
}

// blockMaxImpact returns the upper-bound BM25 TF+norm score for the current block.
func (it *postingIterator) blockMaxImpact(idf float64, normTable [256]float32) float64 {
	if it.blockIdx >= 0 && it.blockIdx < len(it.skips) && !it.inTail {
		se := it.skips[it.blockIdx]
		return idf * fieldNormUpperBound(se.blockMaxTF, se.blockMaxNorm, normTable)
	}
	// Conservative estimate for tail
	tf := float64(it.curFreq)
	if tf < 1 {
		tf = 1
	}
	return idf * ((tf*(bm25K1+1.0))/(tf+float64(normTable[0])) + bm25Delta)
}

// positions returns the positions for the current document.
func (it *postingIterator) positions() []uint32 {
	if it.curDoc == noMoreDocs {
		return nil
	}
	if it.curPosLoaded {
		return it.curPositions
	}
	if it.posData == nil || len(it.posData) == 0 {
		it.curPosLoaded = true
		it.curPositions = nil
		return nil
	}
	off := it.posBase + it.posReadOff
	if int(off) >= len(it.posData) {
		it.curPosLoaded = true
		it.curPositions = nil
		return nil
	}
	count, n := vintGet(it.posData[off:])
	off += uint32(n)

	positions := make([]uint32, count)
	prev := uint32(0)
	for i := uint32(0); i < count; i++ {
		delta, dn := vintGet(it.posData[off:])
		off += uint32(dn)
		prev += delta
		positions[i] = prev
	}
	// Advance posReadOff past what we just read
	it.posReadOff = off - it.posBase
	it.curPosLoaded = true
	it.curPositions = positions
	return positions
}

// skipPositions advances the position read offset past the current doc's positions
// without allocating. Call this for docs you score but don't need positions for.
func (it *postingIterator) skipPositions() {
	if it.curDoc == noMoreDocs {
		return
	}
	if it.curPosLoaded {
		return
	}
	if it.posData == nil || len(it.posData) == 0 {
		it.curPosLoaded = true
		it.curPositions = nil
		return
	}
	off := it.posBase + it.posReadOff
	if int(off) >= len(it.posData) {
		it.curPosLoaded = true
		it.curPositions = nil
		return
	}
	count, n := vintGet(it.posData[off:])
	off += uint32(n)
	for i := uint32(0); i < count; i++ {
		_, dn := vintGet(it.posData[off:])
		off += uint32(dn)
	}
	it.posReadOff = off - it.posBase
	it.curPosLoaded = true
	it.curPositions = nil
}

func (it *postingIterator) doc() uint32  { return it.curDoc }
func (it *postingIterator) freq() uint32 { return it.curFreq }

func (it *postingIterator) skipCurrentPositions() {
	if it.curDoc == noMoreDocs {
		return
	}
	it.skipPositions()
}
