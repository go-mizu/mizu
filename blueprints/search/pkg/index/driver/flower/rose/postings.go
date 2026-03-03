// Package rose implements the Rose inverted-index FTS engine.
// This file contains the low-level integer codec (VByte) and the 128-posting
// block pack/unpack primitives used to compress posting lists on disk.
package rose

import "fmt"

// blockSize is the number of (docID, impact) pairs per posting block.
// 128 matches Lucene's default and the optimum identified in the BMW paper.
const blockSize = 128

// ---------------------------------------------------------------------------
// VByte codec
// ---------------------------------------------------------------------------

// vbyteEncode appends the VByte encoding of v to buf and returns the result.
// VByte stores 7 data bits per byte; the MSB is set on all bytes except the
// last (continuation bit).  Encoding is LSB-first.
//
//	0–127        → 1 byte
//	128–16 383   → 2 bytes
//	16 384–2 097 151 → 3 bytes
//	2 097 152–268 435 455 → 4 bytes
//	268 435 456–max uint32 → 5 bytes
func vbyteEncode(buf []byte, v uint32) []byte {
	for v >= 0x80 {
		buf = append(buf, byte(v&0x7F)|0x80)
		v >>= 7
	}
	return append(buf, byte(v))
}

// vbyteDecode reads a VByte-encoded uint32 from buf starting at pos.
// It returns the decoded value, the position immediately after the last
// consumed byte, and an error if the buffer is truncated or the varint overflows.
func vbyteDecode(buf []byte, pos int) (uint32, int, error) {
	var v uint32
	var shift uint
	for {
		if pos >= len(buf) {
			return 0, pos, fmt.Errorf("vbyteDecode: buffer truncated at position %d", pos)
		}
		if shift >= 35 {
			return 0, pos, fmt.Errorf("vbyteDecode: varint overflow")
		}
		b := buf[pos]
		pos++
		v |= uint32(b&0x7F) << shift
		if b < 0x80 {
			break
		}
		shift += 7
	}
	return v, pos, nil
}

// ---------------------------------------------------------------------------
// Block pack / unpack
// ---------------------------------------------------------------------------

// packBlock encodes up to blockSize (docID, impact) pairs into a compact byte
// slice.  DocIDs are delta-encoded relative to blockBase (the last docID of the
// previous block, or 0 for the first block).  Impacts are stored as a raw
// uint8 array after all delta bytes.
//
// Layout:
//
//	[N bytes] VByte-encoded deltas (docID[i] - blockBase, parallel to docIDs)
//	[N bytes] Impact scores (uint8, parallel to docIDs)
//
// DocID deltas are absolute from blockBase (not chained): delta[i] = docID[i] - blockBase
//
// Returns (encodedBytes, BlockMaxImpact).
func packBlock(docIDs []uint32, impacts []uint8, blockBase uint32) ([]byte, uint8) {
	n := len(docIDs)
	if n == 0 {
		return nil, 0
	}

	// Compute BlockMaxImpact.
	var bmi uint8
	for _, imp := range impacts {
		if imp > bmi {
			bmi = imp
		}
	}

	// Encode deltas (each docID relative to blockBase) using VByte.
	// Then append raw impact bytes.
	//
	// Pre-allocate a reasonable buffer: worst case 5 bytes/delta + 1 byte/impact.
	buf := make([]byte, 0, n*5+n)
	for _, id := range docIDs {
		buf = vbyteEncode(buf, id-blockBase)
	}
	buf = append(buf, impacts...)

	return buf, bmi
}

// unpackBlock decodes n postings from data, using blockBase as the delta base.
// Returns (docIDs, impacts, nil) or (nil, nil, error) on malformed input.
// Returns (nil, nil, nil) gracefully when n == 0.
func unpackBlock(data []byte, blockBase uint32, n int) ([]uint32, []uint8, error) {
	if n == 0 {
		return nil, nil, nil
	}

	docIDs := make([]uint32, n)
	pos := 0
	for i := 0; i < n; i++ {
		delta, newPos, err := vbyteDecode(data, pos)
		if err != nil {
			return nil, nil, fmt.Errorf("unpackBlock: delta %d: %w", i, err)
		}
		docIDs[i] = blockBase + delta
		pos = newPos
	}

	// Impact bytes follow immediately after the VByte section.
	if pos+n > len(data) {
		return nil, nil, fmt.Errorf("unpackBlock: buffer too short for %d impact bytes (have %d)", n, len(data)-pos)
	}
	impacts := make([]uint8, n)
	copy(impacts, data[pos:pos+n])

	return docIDs, impacts, nil
}
