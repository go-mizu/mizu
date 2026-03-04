package lotus

import (
	"bytes"
	"encoding/binary"
	"math"
	"os"
)

// --- Skip entry: per 128-doc block metadata for seeking + Block-Max WAND ---

type skipEntry struct {
	lastDoc      uint32 // highest docID in this block
	blockOff     uint32 // byte offset of this block's data (relative to term blob data start)
	blockMaxTF   uint32 // max term freq in block (for WAND scoring)
	blockMaxNorm uint8  // fieldnorm byte of shortest doc in block
}

const skipEntrySize = 4 + 4 + 4 + 1 // 13 bytes

func encodeSkipEntry(se skipEntry) []byte {
	var buf [skipEntrySize]byte
	binary.LittleEndian.PutUint32(buf[0:], se.lastDoc)
	binary.LittleEndian.PutUint32(buf[4:], se.blockOff)
	binary.LittleEndian.PutUint32(buf[8:], se.blockMaxTF)
	buf[12] = se.blockMaxNorm
	return buf[:]
}

func decodeSkipEntry(buf []byte) skipEntry {
	return skipEntry{
		lastDoc:      binary.LittleEndian.Uint32(buf[0:]),
		blockOff:     binary.LittleEndian.Uint32(buf[4:]),
		blockMaxTF:   binary.LittleEndian.Uint32(buf[8:]),
		blockMaxNorm: buf[12],
	}
}

// --- Per-term blob layout in .doc file ---
//
// [blobSize u32]                         -- total bytes including this field
// For each full 128-doc block:
//   [BP128 docID deltas] [BP128 freqs]   -- back to back
// [VInt tail docID deltas] [VInt tail freqs]
// [skip entry 0] [skip entry 1] ...
// [numFullBlocks u32] [tailCount u32]
//
// postingsOff in termInfo points to the start of the blob (the blobSize field).
// Skip entry blockOff is relative to blobStart + 4 (past blobSize header).

// --- Segment-level postings builder ---

type segmentPostingsBuilder struct {
	docBuf bytes.Buffer // contains all terms' blobs concatenated
	posBuf bytes.Buffer // position data (VInt-encoded per doc)
}

func newSegmentPostingsBuilder() *segmentPostingsBuilder {
	return &segmentPostingsBuilder{}
}

// writeTermPostings writes one term's posting blob into the shared .doc buffer.
// Returns postingsOff = byte offset where this term's blob starts.
func (b *segmentPostingsBuilder) writeTermPostings(
	docs []uint32,
	freqs []uint32,
	norms []uint8,
	positions [][]uint32,
	hasPositions bool,
) uint32 {
	postingsOff := uint32(b.docBuf.Len())

	n := len(docs)
	if n == 0 {
		return postingsOff
	}

	// Build the blob in a temp buffer
	var blob bytes.Buffer

	var skipEntries []skipEntry
	relOff := uint32(0)

	fullBlocks := n / 128
	for blk := 0; blk < fullBlocks; blk++ {
		base := blk * 128
		var deltas [128]uint32
		var blockFreqs [128]uint32
		var prev uint32
		if blk > 0 {
			prev = docs[base-1]
		}
		var maxTF uint32
		maxNorm := norms[base]
		for i := 0; i < 128; i++ {
			deltas[i] = docs[base+i] - prev
			prev = docs[base+i]
			blockFreqs[i] = freqs[base+i]
			if freqs[base+i] > maxTF {
				maxTF = freqs[base+i]
			}
			if norms[base+i] < maxNorm {
				maxNorm = norms[base+i]
			}
		}

		skipEntries = append(skipEntries, skipEntry{
			lastDoc:      docs[base+127],
			blockOff:     relOff,
			blockMaxTF:   maxTF,
			blockMaxNorm: maxNorm,
		})

		// Write docID deltas then freqs back-to-back
		packed := bp128Pack(deltas[:])
		blob.Write(packed)
		relOff += uint32(len(packed))

		packedFreq := bp128Pack(blockFreqs[:])
		blob.Write(packedFreq)
		relOff += uint32(len(packedFreq))

		// Write positions
		if hasPositions {
			for i := 0; i < 128; i++ {
				idx := base + i
				if idx < len(positions) {
					writePositionsVInt(&b.posBuf, positions[idx])
				}
			}
		}
	}

	// VInt tail
	tailStart := fullBlocks * 128
	tailCount := n - tailStart
	if tailCount > 0 {
		var prev uint32
		if tailStart > 0 {
			prev = docs[tailStart-1]
		}
		for i := tailStart; i < n; i++ {
			vintPut(&blob, docs[i]-prev)
			prev = docs[i]
		}
		for i := tailStart; i < n; i++ {
			vintPut(&blob, freqs[i])
		}
		if hasPositions {
			for i := tailStart; i < n; i++ {
				if i < len(positions) {
					writePositionsVInt(&b.posBuf, positions[i])
				}
			}
		}
	}

	// Append skip entries
	for _, se := range skipEntries {
		blob.Write(encodeSkipEntry(se))
	}

	// Trailer: [numFullBlocks u32][tailCount u32]
	var trailer [8]byte
	binary.LittleEndian.PutUint32(trailer[0:], uint32(fullBlocks))
	binary.LittleEndian.PutUint32(trailer[4:], uint32(tailCount))
	blob.Write(trailer[:])

	// Write to main buffer: [blobSize u32][blob data]
	blobSize := uint32(4 + blob.Len())
	var hdr [4]byte
	binary.LittleEndian.PutUint32(hdr[:], blobSize)
	b.docBuf.Write(hdr[:])
	b.docBuf.Write(blob.Bytes())

	return postingsOff
}

func writePositionsVInt(buf *bytes.Buffer, positions []uint32) {
	vintPut(buf, uint32(len(positions)))
	var prev uint32
	for _, p := range positions {
		vintPut(buf, p-prev)
		prev = p
	}
}

func (b *segmentPostingsBuilder) writeTo(docPath, posPath string) error {
	if err := os.WriteFile(docPath, b.docBuf.Bytes(), 0644); err != nil {
		return err
	}
	if b.posBuf.Len() > 0 {
		if err := os.WriteFile(posPath, b.posBuf.Bytes(), 0644); err != nil {
			return err
		}
	}
	return nil
}

// --- Postings reader ---

type postingsReader struct {
	docData []byte
	posData []byte
}

func openPostingsReader(docPath, posPath string) (*postingsReader, error) {
	docData, err := mmapFile(docPath)
	if err != nil {
		return nil, err
	}
	posData, _ := mmapFile(posPath)

	return &postingsReader{
		docData: docData,
		posData: posData,
	}, nil
}

func (pr *postingsReader) close() error {
	var firstErr error
	if err := mmapRelease(pr.docData); err != nil && firstErr == nil {
		firstErr = err
	}
	if err := mmapRelease(pr.posData); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}

func (pr *postingsReader) iterator(postingsOff uint32) *postingIterator {
	return newPostingIterator(pr, postingsOff)
}

// --- Posting iterator ---

const noMoreDocs = math.MaxUint32

type postingIterator struct {
	pr      *postingsReader
	blobOff uint32 // byte offset of blobSize in .doc

	// Parsed from blob
	dataStart     uint32 // = blobOff + 4 (past blobSize)
	numFullBlocks uint32
	tailCount     uint32
	skips         []skipEntry

	curDocID uint32
	curFreq  uint32

	blockDocs  [128]uint32
	blockFreqs [128]uint32
	blockPos   int
	blockLen   int

	blocksParsed uint32
	inLastBlock  bool

	lastBlockDocs  []uint32
	lastBlockFreqs []uint32
}

func newPostingIterator(pr *postingsReader, blobOff uint32) *postingIterator {
	it := &postingIterator{
		pr:       pr,
		blobOff:  blobOff,
		curDocID: noMoreDocs,
	}
	it.parseBlob()
	return it
}

func (it *postingIterator) parseBlob() {
	docData := it.pr.docData
	if int(it.blobOff)+4 > len(docData) {
		return
	}

	blobSize := binary.LittleEndian.Uint32(docData[it.blobOff:])
	blobEnd := it.blobOff + blobSize
	if int(blobEnd) > len(docData) || blobSize < 12 {
		return
	}

	it.dataStart = it.blobOff + 4

	// Trailer at end of blob
	it.numFullBlocks = binary.LittleEndian.Uint32(docData[blobEnd-8:])
	it.tailCount = binary.LittleEndian.Uint32(docData[blobEnd-4:])

	// Skip entries before trailer
	skipSize := int(it.numFullBlocks) * skipEntrySize
	skipStart := int(blobEnd) - 8 - skipSize
	if skipStart < int(it.dataStart) {
		return
	}
	it.skips = make([]skipEntry, it.numFullBlocks)
	for i := range it.skips {
		off := skipStart + i*skipEntrySize
		it.skips[i] = decodeSkipEntry(docData[off:])
	}
}

func (it *postingIterator) docID() uint32 { return it.curDocID }
func (it *postingIterator) freq() uint32  { return it.curFreq }

func (it *postingIterator) next() uint32 {
	if it.blockPos+1 < it.blockLen {
		it.blockPos++
		if it.inLastBlock {
			it.curDocID = it.lastBlockDocs[it.blockPos]
			it.curFreq = it.lastBlockFreqs[it.blockPos]
		} else {
			it.curDocID = it.blockDocs[it.blockPos]
			it.curFreq = it.blockFreqs[it.blockPos]
		}
		return it.curDocID
	}
	return it.nextBlock()
}

func (it *postingIterator) nextBlock() uint32 {
	if it.inLastBlock {
		it.curDocID = noMoreDocs
		return noMoreDocs
	}
	if it.blocksParsed < it.numFullBlocks {
		it.decodeFullBlock()
		it.blocksParsed++
		it.blockPos = 0
		it.blockLen = 128
		it.curDocID = it.blockDocs[0]
		it.curFreq = it.blockFreqs[0]
		return it.curDocID
	}
	if it.tailCount > 0 {
		it.decodeLastBlock()
		if it.blockLen > 0 {
			it.inLastBlock = true
			it.blockPos = 0
			it.curDocID = it.lastBlockDocs[0]
			it.curFreq = it.lastBlockFreqs[0]
			return it.curDocID
		}
	}
	it.curDocID = noMoreDocs
	return noMoreDocs
}

func (it *postingIterator) decodeFullBlock() {
	if int(it.blocksParsed) >= len(it.skips) {
		return
	}
	se := it.skips[it.blocksParsed]
	absOff := it.dataStart + se.blockOff

	// Read docID deltas
	var deltas [128]uint32
	bp128Unpack(it.pr.docData[absOff:], deltas[:])
	docBlockSize := 1 + uint32(it.pr.docData[absOff])*16

	// Convert deltas to absolute docIDs
	var prev uint32
	if it.blocksParsed > 0 {
		prev = it.skips[it.blocksParsed-1].lastDoc
	}
	for i := 0; i < 128; i++ {
		prev += deltas[i]
		it.blockDocs[i] = prev
	}

	// Read freqs (immediately after docID block)
	freqOff := absOff + docBlockSize
	bp128Unpack(it.pr.docData[freqOff:], it.blockFreqs[:])
}

func (it *postingIterator) decodeLastBlock() {
	// Find where VInt tail starts: after all full blocks
	// Each full block = [BP128 docIDs][BP128 freqs]
	var tailOff uint32
	if it.numFullBlocks > 0 {
		lastSkip := it.skips[it.numFullBlocks-1]
		absOff := it.dataStart + lastSkip.blockOff
		// Doc block size
		docBits := uint32(it.pr.docData[absOff])
		docBlockSize := 1 + docBits*16
		// Freq block size
		freqOff := absOff + docBlockSize
		freqBits := uint32(it.pr.docData[freqOff])
		freqBlockSize := 1 + freqBits*16
		tailOff = lastSkip.blockOff + docBlockSize + freqBlockSize
	}

	vintData := it.pr.docData[it.dataStart+tailOff:]
	var prev uint32
	if it.numFullBlocks > 0 {
		prev = it.skips[it.numFullBlocks-1].lastDoc
	}

	// Decode docID deltas
	it.lastBlockDocs = it.lastBlockDocs[:0]
	off := 0
	for i := uint32(0); i < it.tailCount && off < len(vintData); i++ {
		delta, n := vintGet(vintData[off:])
		if n == 0 {
			break
		}
		off += n
		prev += delta
		it.lastBlockDocs = append(it.lastBlockDocs, prev)
	}

	// Decode freqs (immediately after docID deltas)
	it.lastBlockFreqs = it.lastBlockFreqs[:0]
	for i := 0; i < len(it.lastBlockDocs) && off < len(vintData); i++ {
		freq, n := vintGet(vintData[off:])
		if n == 0 {
			break
		}
		off += n
		it.lastBlockFreqs = append(it.lastBlockFreqs, freq)
	}
	it.blockLen = len(it.lastBlockDocs)
}

func (it *postingIterator) advance(target uint32) uint32 {
	if it.curDocID != noMoreDocs && it.curDocID >= target {
		return it.curDocID
	}

	lo, hi := int(it.blocksParsed), len(it.skips)-1
	blockIdx := -1
	for lo <= hi {
		mid := (lo + hi) / 2
		if it.skips[mid].lastDoc < target {
			lo = mid + 1
		} else {
			blockIdx = mid
			hi = mid - 1
		}
	}

	if blockIdx >= 0 && uint32(blockIdx) >= it.blocksParsed {
		it.blocksParsed = uint32(blockIdx)
		it.inLastBlock = false
		it.decodeFullBlock()
		it.blocksParsed++
		it.blockLen = 128

		for i := 0; i < 128; i++ {
			if it.blockDocs[i] >= target {
				it.blockPos = i
				it.curDocID = it.blockDocs[i]
				it.curFreq = it.blockFreqs[i]
				return it.curDocID
			}
		}
	}

	for {
		doc := it.next()
		if doc >= target || doc == noMoreDocs {
			return doc
		}
	}
}

func (it *postingIterator) blockMaxImpact() (maxTF uint32, maxNorm uint8, ok bool) {
	idx := int(it.blocksParsed) - 1
	if idx < 0 || idx >= len(it.skips) {
		return 0, 0, false
	}
	se := it.skips[idx]
	return se.blockMaxTF, se.blockMaxNorm, true
}
