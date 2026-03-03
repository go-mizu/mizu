package dahlia

import (
	"bytes"
	"fmt"

	"github.com/blevesearch/vellum"
)

// termInfo holds metadata about a term's posting list.
type termInfo struct {
	docFreq      uint32 // number of documents containing the term
	postingsOff  uint32 // byte offset of posting data in .doc file
	hasPositions bool   // whether position data is available
}

// packTermInfo encodes termInfo into a uint64 for FST value storage.
// Bits [0:30]  = docFreq (max ~1 billion)
// Bit  [30]    = hasPositions flag
// Bits [32:63] = postingsOff
func packTermInfo(ti termInfo) uint64 {
	v := uint64(ti.docFreq & 0x3FFFFFFF) // bits 0-29
	if ti.hasPositions {
		v |= 1 << 30 // bit 30
	}
	v |= uint64(ti.postingsOff) << 32 // bits 32-63
	return v
}

// unpackTermInfo decodes a uint64 FST value to termInfo.
func unpackTermInfo(v uint64) termInfo {
	return termInfo{
		docFreq:      uint32(v & 0x3FFFFFFF),
		hasPositions: (v>>30)&1 == 1,
		postingsOff:  uint32(v >> 32),
	}
}

// termDictWriter builds an FST-based term dictionary.
// Terms must be added in sorted lexicographic order.
type termDictWriter struct {
	buf     bytes.Buffer
	builder *vellum.Builder
}

func newTermDictWriter() (*termDictWriter, error) {
	w := &termDictWriter{}
	builder, err := vellum.New(&w.buf, nil)
	if err != nil {
		return nil, fmt.Errorf("vellum.New: %w", err)
	}
	w.builder = builder
	return w, nil
}

// add inserts a term with its metadata. Terms must be in sorted order.
func (w *termDictWriter) add(term string, ti termInfo) error {
	return w.builder.Insert([]byte(term), packTermInfo(ti))
}

// finish completes the FST and returns the serialized bytes.
func (w *termDictWriter) finish() ([]byte, error) {
	if err := w.builder.Close(); err != nil {
		return nil, err
	}
	return w.buf.Bytes(), nil
}

// termDictReader provides lookup into an FST term dictionary.
type termDictReader struct {
	fst *vellum.FST
}

func openTermDictReader(data []byte) (*termDictReader, error) {
	fst, err := vellum.Load(data)
	if err != nil {
		return nil, fmt.Errorf("vellum.Load: %w", err)
	}
	return &termDictReader{fst: fst}, nil
}

// lookup returns termInfo for the given term, or false if not found.
func (r *termDictReader) lookup(term string) (termInfo, bool) {
	v, exists, err := r.fst.Get([]byte(term))
	if err != nil || !exists {
		return termInfo{}, false
	}
	return unpackTermInfo(v), true
}

func (r *termDictReader) close() error {
	return r.fst.Close()
}
