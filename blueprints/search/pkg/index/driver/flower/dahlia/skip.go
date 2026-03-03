package dahlia

import "encoding/binary"

// skipEntry represents a skip entry for block-level seeking in posting lists.
// Each entry is 21 bytes and contains offsets into the separate .doc/.freq/.pos files
// plus WAND data for block-max scoring.
type skipEntry struct {
	lastDoc      uint32 // highest docID in block
	docOff       uint32 // byte offset into .doc file
	freqOff      uint32 // byte offset into .freq file
	posOff       uint32 // byte offset into .pos file
	blockMaxTF   uint32 // max TF in block (for WAND upper bound)
	blockMaxNorm uint8  // shortest-doc norm in block (for WAND upper bound)
}

// encodeSkipEntry writes a skip entry to a 21-byte buffer.
func encodeSkipEntry(dst []byte, e skipEntry) {
	binary.LittleEndian.PutUint32(dst[0:4], e.lastDoc)
	binary.LittleEndian.PutUint32(dst[4:8], e.docOff)
	binary.LittleEndian.PutUint32(dst[8:12], e.freqOff)
	binary.LittleEndian.PutUint32(dst[12:16], e.posOff)
	binary.LittleEndian.PutUint32(dst[16:20], e.blockMaxTF)
	dst[20] = e.blockMaxNorm
}

// decodeSkipEntry reads a skip entry from a 21-byte buffer.
func decodeSkipEntry(src []byte) skipEntry {
	return skipEntry{
		lastDoc:      binary.LittleEndian.Uint32(src[0:4]),
		docOff:       binary.LittleEndian.Uint32(src[4:8]),
		freqOff:      binary.LittleEndian.Uint32(src[8:12]),
		posOff:       binary.LittleEndian.Uint32(src[12:16]),
		blockMaxTF:   binary.LittleEndian.Uint32(src[16:20]),
		blockMaxNorm: src[20],
	}
}

// skipIndex is a list of skip entries for a term's posting list.
type skipIndex []skipEntry

// findBlock returns the index of the skip entry whose block contains target,
// or -1 if target is beyond all blocks.
func (s skipIndex) findBlock(target uint32) int {
	// Binary search for the first block where lastDoc >= target
	lo, hi := 0, len(s)-1
	result := -1
	for lo <= hi {
		mid := (lo + hi) / 2
		if s[mid].lastDoc >= target {
			result = mid
			hi = mid - 1
		} else {
			lo = mid + 1
		}
	}
	return result
}
