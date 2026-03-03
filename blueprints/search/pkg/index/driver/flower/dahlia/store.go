package dahlia

import (
	"encoding/binary"
	"fmt"
	"sort"

	"github.com/klauspost/compress/zstd"
)

var (
	zstdEncoder, _ = zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedDefault))
	zstdDecoder, _ = zstd.NewReader(nil)
)

// storeWriter writes document fields to a compressed block store.
// Documents are grouped into 16KB blocks, each zstd-compressed.
// A skip index at the end of the file maps lastDocID → blockOffset.
type storeWriter struct {
	buf       []byte // current block accumulator
	blocks    [][]byte
	skipIndex []storeSkipEntry
	docCount  uint32
	lastDoc   uint32
}

type storeSkipEntry struct {
	lastDocID   uint32
	blockOffset uint64
}

// addDoc adds a document to the store.
func (sw *storeWriter) addDoc(localID uint32, docID string, text []byte) {
	// Format: [idLen:4][id:N][textLen:4][text:M]
	rec := make([]byte, 4+len(docID)+4+len(text))
	binary.LittleEndian.PutUint32(rec[0:4], uint32(len(docID)))
	copy(rec[4:4+len(docID)], docID)
	binary.LittleEndian.PutUint32(rec[4+len(docID):8+len(docID)], uint32(len(text)))
	copy(rec[8+len(docID):], text)

	sw.buf = append(sw.buf, rec...)
	sw.lastDoc = localID
	sw.docCount++

	// Flush block when it exceeds 16KB
	if len(sw.buf) >= storeBlockSize {
		sw.flushBlock()
	}
}

func (sw *storeWriter) flushBlock() {
	if len(sw.buf) == 0 {
		return
	}
	compressed := zstdEncoder.EncodeAll(sw.buf, nil)
	sw.skipIndex = append(sw.skipIndex, storeSkipEntry{
		lastDocID: sw.lastDoc,
	})
	sw.blocks = append(sw.blocks, compressed)
	sw.buf = sw.buf[:0]
}

// finish returns the complete store file bytes.
func (sw *storeWriter) finish() []byte {
	// Flush remaining docs
	sw.flushBlock()

	// Compute block offsets and total data size
	offset := uint64(0)
	for i, block := range sw.blocks {
		sw.skipIndex[i].blockOffset = offset
		offset += uint64(len(block))
	}

	// Write: [blocks...][skip index...][skip index offset: 8 bytes]
	total := int(offset) + len(sw.skipIndex)*12 + 8
	out := make([]byte, 0, total)
	for _, block := range sw.blocks {
		out = append(out, block...)
	}

	skipOff := len(out)
	for _, se := range sw.skipIndex {
		var buf [12]byte
		binary.LittleEndian.PutUint32(buf[0:4], se.lastDocID)
		binary.LittleEndian.PutUint64(buf[4:12], se.blockOffset)
		out = append(out, buf[:]...)
	}

	var footer [8]byte
	binary.LittleEndian.PutUint64(footer[:], uint64(skipOff))
	out = append(out, footer[:]...)

	return out
}

// storeReader reads from a compressed block store.
type storeReader struct {
	data      []byte
	skipIndex []storeSkipEntry
}

func openStoreReader(data []byte) (*storeReader, error) {
	if len(data) < 8 {
		return &storeReader{data: data}, nil
	}
	// Read footer
	skipOff := binary.LittleEndian.Uint64(data[len(data)-8:])
	if skipOff > uint64(len(data)-8) {
		return nil, fmt.Errorf("invalid skip index offset %d", skipOff)
	}

	// Parse skip index
	skipData := data[skipOff : len(data)-8]
	numEntries := len(skipData) / 12
	skips := make([]storeSkipEntry, numEntries)
	for i := 0; i < numEntries; i++ {
		off := i * 12
		skips[i] = storeSkipEntry{
			lastDocID:   binary.LittleEndian.Uint32(skipData[off : off+4]),
			blockOffset: binary.LittleEndian.Uint64(skipData[off+4 : off+12]),
		}
	}

	return &storeReader{data: data, skipIndex: skips}, nil
}

// getDoc retrieves a document by local docID.
func (sr *storeReader) getDoc(docID uint32) (id string, text []byte, err error) {
	if len(sr.skipIndex) == 0 {
		return "", nil, fmt.Errorf("empty store")
	}

	// Binary search for the block containing this docID
	blockIdx := sort.Search(len(sr.skipIndex), func(i int) bool {
		return sr.skipIndex[i].lastDocID >= docID
	})
	if blockIdx >= len(sr.skipIndex) {
		return "", nil, fmt.Errorf("docID %d not found", docID)
	}

	// Determine block boundaries
	blockStart := sr.skipIndex[blockIdx].blockOffset
	var blockEnd uint64
	if blockIdx+1 < len(sr.skipIndex) {
		blockEnd = sr.skipIndex[blockIdx+1].blockOffset
	} else {
		// Last block ends at skip index offset
		blockEnd = binary.LittleEndian.Uint64(sr.data[len(sr.data)-8:])
	}

	// Decompress block
	compressed := sr.data[blockStart:blockEnd]
	decompressed, err := zstdDecoder.DecodeAll(compressed, nil)
	if err != nil {
		return "", nil, fmt.Errorf("zstd decompress: %w", err)
	}

	// Scan within block for target docID
	// We need to track docIDs. The block contains sequential docs starting
	// from the previous block's lastDoc + 1 (or 0 for first block).
	startDocID := uint32(0)
	if blockIdx > 0 {
		startDocID = sr.skipIndex[blockIdx-1].lastDocID + 1
	}

	off := 0
	curDocID := startDocID
	for off < len(decompressed) {
		if off+4 > len(decompressed) {
			break
		}
		idLen := binary.LittleEndian.Uint32(decompressed[off : off+4])
		off += 4
		if off+int(idLen)+4 > len(decompressed) {
			break
		}
		docIDStr := string(decompressed[off : off+int(idLen)])
		off += int(idLen)
		textLen := binary.LittleEndian.Uint32(decompressed[off : off+4])
		off += 4
		if off+int(textLen) > len(decompressed) {
			break
		}
		docText := decompressed[off : off+int(textLen)]
		off += int(textLen)

		if curDocID == docID {
			return docIDStr, docText, nil
		}
		curDocID++
	}

	return "", nil, fmt.Errorf("docID %d not found in block", docID)
}
