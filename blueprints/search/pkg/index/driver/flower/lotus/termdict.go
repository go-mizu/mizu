package lotus

import (
	"os"

	"github.com/blevesearch/vellum"
)

type termInfo struct {
	docFreq     uint32
	postingsOff uint32
	// hasPositions indicates whether this term has position data in the .pos file.
	hasPositions bool
}

func packTermInfo(ti termInfo) uint64 {
	v := uint64(ti.docFreq & 0x7FFFFFFF)
	if ti.hasPositions {
		v |= 1 << 31
	}
	v |= uint64(ti.postingsOff) << 32
	return v
}

func unpackTermInfo(v uint64) termInfo {
	return termInfo{
		docFreq:     uint32(v & 0x7FFFFFFF),
		postingsOff: uint32(v >> 32),
		hasPositions: (v>>31)&1 == 1,
	}
}

type termDictWriter struct {
	f       *os.File
	builder *vellum.Builder
}

func newTermDictWriter(path string) (*termDictWriter, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	b, err := vellum.New(f, nil)
	if err != nil {
		f.Close()
		return nil, err
	}
	return &termDictWriter{f: f, builder: b}, nil
}

// add inserts a term. Terms MUST be added in sorted lexicographic order.
func (w *termDictWriter) add(term string, info termInfo) error {
	return w.builder.Insert([]byte(term), packTermInfo(info))
}

func (w *termDictWriter) close() error {
	if err := w.builder.Close(); err != nil {
		w.f.Close()
		return err
	}
	return w.f.Close()
}

type termDictReader struct {
	fst  *vellum.FST
	data []byte
}

func openTermDict(path string) (*termDictReader, error) {
	data, err := mmapFile(path)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return &termDictReader{}, nil
	}
	fst, err := vellum.Load(data)
	if err != nil {
		mmapRelease(data)
		return nil, err
	}
	return &termDictReader{fst: fst, data: data}, nil
}

func (r *termDictReader) get(term string) (termInfo, bool) {
	if r.fst == nil {
		return termInfo{}, false
	}
	v, exists, _ := r.fst.Get([]byte(term))
	if !exists {
		return termInfo{}, false
	}
	return unpackTermInfo(v), true
}

func (r *termDictReader) close() error {
	return mmapRelease(r.data)
}
