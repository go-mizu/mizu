// Package algo provides optimized algorithms for full-text search.
package algo

import (
	"encoding/binary"
)

// StreamVByte implements the StreamVByte integer compression algorithm.
// It separates control bytes from data for better CPU cache utilization.
// Reference: https://arxiv.org/abs/1709.08990

// StreamVByteEncode encodes a slice of uint32 values using StreamVByte.
// Returns compressed bytes with control stream followed by data stream.
func StreamVByteEncode(values []uint32) []byte {
	if len(values) == 0 {
		return nil
	}

	n := len(values)
	controlLen := (n + 3) / 4

	// Allocate buffer: control bytes + worst case data (4 bytes per int)
	buf := make([]byte, controlLen+n*4)

	dataPos := controlLen

	for i := 0; i < n; i += 4 {
		var control byte
		for j := 0; j < 4 && i+j < n; j++ {
			v := values[i+j]
			var size int

			switch {
			case v < 1<<8:
				size = 1
				buf[dataPos] = byte(v)
			case v < 1<<16:
				size = 2
				binary.LittleEndian.PutUint16(buf[dataPos:], uint16(v))
			case v < 1<<24:
				size = 3
				buf[dataPos] = byte(v)
				buf[dataPos+1] = byte(v >> 8)
				buf[dataPos+2] = byte(v >> 16)
			default:
				size = 4
				binary.LittleEndian.PutUint32(buf[dataPos:], v)
			}

			control |= byte(size-1) << (j * 2)
			dataPos += size
		}
		buf[i/4] = control
	}

	return buf[:dataPos]
}

// StreamVByteDecode decodes StreamVByte compressed data.
func StreamVByteDecode(data []byte, n int) []uint32 {
	if n == 0 || len(data) == 0 {
		return nil
	}

	result := make([]uint32, n)
	controlLen := (n + 3) / 4
	dataPos := controlLen

	for i := 0; i < n; i += 4 {
		control := data[i/4]
		for j := 0; j < 4 && i+j < n; j++ {
			size := int((control>>(j*2))&0x03) + 1

			switch size {
			case 1:
				result[i+j] = uint32(data[dataPos])
			case 2:
				result[i+j] = uint32(binary.LittleEndian.Uint16(data[dataPos:]))
			case 3:
				result[i+j] = uint32(data[dataPos]) |
					uint32(data[dataPos+1])<<8 |
					uint32(data[dataPos+2])<<16
			case 4:
				result[i+j] = binary.LittleEndian.Uint32(data[dataPos:])
			}
			dataPos += size
		}
	}

	return result
}

// DeltaEncode converts absolute doc IDs to delta-encoded gaps.
func DeltaEncode(docIDs []uint32) []uint32 {
	if len(docIDs) == 0 {
		return nil
	}

	result := make([]uint32, len(docIDs))
	result[0] = docIDs[0]
	for i := 1; i < len(docIDs); i++ {
		result[i] = docIDs[i] - docIDs[i-1]
	}
	return result
}

// DeltaDecode converts delta-encoded gaps back to absolute doc IDs.
func DeltaDecode(gaps []uint32) []uint32 {
	if len(gaps) == 0 {
		return nil
	}

	result := make([]uint32, len(gaps))
	result[0] = gaps[0]
	for i := 1; i < len(gaps); i++ {
		result[i] = result[i-1] + gaps[i]
	}
	return result
}

// CompressedPostingList stores a posting list with StreamVByte compression.
type CompressedPostingList struct {
	DocIDData []byte   // StreamVByte encoded delta doc IDs
	FreqData  []byte   // StreamVByte encoded frequencies
	Length    int      // Number of postings
	MaxDocID  uint32   // Last doc ID (for bounds checking)
}

// NewCompressedPostingList creates a compressed posting list from doc IDs and frequencies.
func NewCompressedPostingList(docIDs, freqs []uint32) *CompressedPostingList {
	if len(docIDs) == 0 {
		return &CompressedPostingList{}
	}

	return &CompressedPostingList{
		DocIDData: StreamVByteEncode(DeltaEncode(docIDs)),
		FreqData:  StreamVByteEncode(freqs),
		Length:    len(docIDs),
		MaxDocID:  docIDs[len(docIDs)-1],
	}
}

// Decode returns the original doc IDs and frequencies.
func (p *CompressedPostingList) Decode() (docIDs, freqs []uint32) {
	if p.Length == 0 {
		return nil, nil
	}

	gaps := StreamVByteDecode(p.DocIDData, p.Length)
	docIDs = DeltaDecode(gaps)
	freqs = StreamVByteDecode(p.FreqData, p.Length)
	return
}
