package lotus

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"

	"github.com/klauspost/compress/zstd"
)

const storeBlockSize = 16 * 1024

type storeWriter struct {
	f          *os.File
	enc        *zstd.Encoder
	buf        bytes.Buffer
	skipIndex  []storeSkipEntry
	docsInBuf  int
	totalDocs  uint32
	byteOffset uint32
}

type storeSkipEntry struct {
	lastDoc     uint32
	blockOffset uint32
}

func newStoreWriter(path string) (*storeWriter, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	enc, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedFastest))
	if err != nil {
		f.Close()
		return nil, err
	}
	return &storeWriter{f: f, enc: enc}, nil
}

func (w *storeWriter) add(id string, text []byte) error {
	var hdr [4]byte
	binary.LittleEndian.PutUint32(hdr[:], uint32(len(id)))
	w.buf.Write(hdr[:])
	w.buf.WriteString(id)
	binary.LittleEndian.PutUint32(hdr[:], uint32(len(text)))
	w.buf.Write(hdr[:])
	w.buf.Write(text)
	w.docsInBuf++
	w.totalDocs++
	if w.buf.Len() >= storeBlockSize {
		return w.flushBlock()
	}
	return nil
}

func (w *storeWriter) flushBlock() error {
	if w.docsInBuf == 0 {
		return nil
	}
	compressed := w.enc.EncodeAll(w.buf.Bytes(), nil)
	n, err := w.f.Write(compressed)
	if err != nil {
		return err
	}
	w.skipIndex = append(w.skipIndex, storeSkipEntry{
		lastDoc:     w.totalDocs - 1,
		blockOffset: w.byteOffset,
	})
	w.byteOffset += uint32(n)
	w.buf.Reset()
	w.docsInBuf = 0
	return nil
}

func (w *storeWriter) close() error {
	if w.docsInBuf > 0 {
		if err := w.flushBlock(); err != nil {
			return err
		}
	}
	w.enc.Close()
	// Write skip index
	skipOff := w.byteOffset
	var hdr [4]byte
	for _, se := range w.skipIndex {
		binary.LittleEndian.PutUint32(hdr[:], se.lastDoc)
		if _, err := w.f.Write(hdr[:]); err != nil {
			return err
		}
		binary.LittleEndian.PutUint32(hdr[:], se.blockOffset)
		if _, err := w.f.Write(hdr[:]); err != nil {
			return err
		}
	}
	// Footer: skip index offset (uint64 LE)
	var footer [8]byte
	binary.LittleEndian.PutUint64(footer[:], uint64(skipOff))
	if _, err := w.f.Write(footer[:]); err != nil {
		return err
	}
	return w.f.Close()
}

// --- Reader ---

type storeReader struct {
	data      []byte
	skipIndex []storeSkipEntry
	dec       *zstd.Decoder
}

func openStoreReader(path string) (*storeReader, error) {
	data, err := mmapFile(path)
	if err != nil {
		return nil, err
	}
	if data == nil || len(data) < 8 {
		return &storeReader{}, nil
	}
	// Read footer
	skipOff := binary.LittleEndian.Uint64(data[len(data)-8:])
	// Parse skip index
	skipData := data[skipOff : len(data)-8]
	numEntries := len(skipData) / 8
	skip := make([]storeSkipEntry, numEntries)
	for i := 0; i < numEntries; i++ {
		off := i * 8
		skip[i] = storeSkipEntry{
			lastDoc:     binary.LittleEndian.Uint32(skipData[off:]),
			blockOffset: binary.LittleEndian.Uint32(skipData[off+4:]),
		}
	}
	dec, err := zstd.NewReader(nil)
	if err != nil {
		mmapRelease(data)
		return nil, err
	}
	return &storeReader{data: data, skipIndex: skip, dec: dec}, nil
}

func (r *storeReader) get(docID uint32) (string, []byte, error) {
	if r.data == nil {
		return "", nil, fmt.Errorf("store: not loaded")
	}
	// Find block containing docID via binary search on skipIndex
	blockIdx := -1
	lo, hi := 0, len(r.skipIndex)-1
	for lo <= hi {
		mid := (lo + hi) / 2
		if r.skipIndex[mid].lastDoc < docID {
			lo = mid + 1
		} else {
			blockIdx = mid
			hi = mid - 1
		}
	}
	if blockIdx < 0 {
		return "", nil, fmt.Errorf("store: docID %d out of range", docID)
	}

	// Determine block byte range
	blockStart := r.skipIndex[blockIdx].blockOffset
	var blockEnd uint32
	if blockIdx+1 < len(r.skipIndex) {
		blockEnd = r.skipIndex[blockIdx+1].blockOffset
	} else {
		blockEnd = uint32(binary.LittleEndian.Uint64(r.data[len(r.data)-8:])) // skipOff
	}

	// Decompress block
	compressed := r.data[blockStart:blockEnd]
	decompressed, err := r.dec.DecodeAll(compressed, nil)
	if err != nil {
		return "", nil, fmt.Errorf("store: decompress: %w", err)
	}

	// Determine first docID in this block
	var firstDoc uint32
	if blockIdx > 0 {
		firstDoc = r.skipIndex[blockIdx-1].lastDoc + 1
	}

	// Scan to target doc
	off := 0
	for curDoc := firstDoc; curDoc <= r.skipIndex[blockIdx].lastDoc && off < len(decompressed); curDoc++ {
		if off+4 > len(decompressed) {
			break
		}
		idLen := binary.LittleEndian.Uint32(decompressed[off:])
		off += 4
		if off+int(idLen) > len(decompressed) {
			break
		}
		id := string(decompressed[off : off+int(idLen)])
		off += int(idLen)
		if off+4 > len(decompressed) {
			break
		}
		textLen := binary.LittleEndian.Uint32(decompressed[off:])
		off += 4
		if off+int(textLen) > len(decompressed) {
			break
		}
		text := decompressed[off : off+int(textLen)]
		off += int(textLen)
		if curDoc == docID {
			return id, text, nil
		}
	}
	return "", nil, fmt.Errorf("store: docID %d not found in block", docID)
}

func (r *storeReader) close() error {
	if r.dec != nil {
		r.dec.Close()
	}
	return mmapRelease(r.data)
}
