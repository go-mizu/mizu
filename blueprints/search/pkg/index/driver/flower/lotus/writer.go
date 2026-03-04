package lotus

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync/atomic"
)

// segmentMeta is persisted as segment.meta JSON per segment.
type segmentMeta struct {
	DocCount  uint32  `json:"doc_count"`
	AvgDocLen float64 `json:"avg_doc_len"`
}

// segmentWriter accumulates documents and flushes a complete segment directory.
type segmentWriter struct {
	dir string // segment directory (e.g., seg_00000001/)

	// Accumulate per-term postings
	terms map[string]*termPostings

	// Stored fields
	store *storeWriter

	// Field norms (one byte per doc)
	norms []uint8

	docCount    uint32
	totalTokens uint64
}

type termPostings struct {
	docs      []uint32   // docIDs
	freqs     []uint32   // per-doc term frequency
	norms     []uint8    // per-doc fieldnorm byte
	positions [][]uint32 // per-doc position list
}

func newSegmentWriter(dir string) (*segmentWriter, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	sw, err := newStoreWriter(filepath.Join(dir, "segment.store"))
	if err != nil {
		return nil, err
	}
	return &segmentWriter{
		dir:   dir,
		terms: make(map[string]*termPostings),
		store: sw,
	}, nil
}

func (w *segmentWriter) addDoc(docID string, text []byte) error {
	localDocID := w.docCount
	w.docCount++

	// Analyze text into tokens with positions
	tokens := analyzeWithPositions(string(text))
	docLen := uint32(len(tokens))
	w.totalTokens += uint64(docLen)
	normByte := fieldNormEncode(docLen)
	w.norms = append(w.norms, normByte)

	// Build per-term frequency + positions for this doc
	termFreq := make(map[string]uint32)
	termPos := make(map[string][]uint32)
	for _, tok := range tokens {
		termFreq[tok.term]++
		termPos[tok.term] = append(termPos[tok.term], tok.pos)
	}

	// Add to inverted index
	for term, freq := range termFreq {
		tp := w.terms[term]
		if tp == nil {
			tp = &termPostings{}
			w.terms[term] = tp
		}
		tp.docs = append(tp.docs, localDocID)
		tp.freqs = append(tp.freqs, freq)
		tp.norms = append(tp.norms, normByte)
		tp.positions = append(tp.positions, termPos[term])
	}

	// Store document for retrieval
	return w.store.add(docID, text)
}

func (w *segmentWriter) flush() error {
	if w.docCount == 0 {
		return nil
	}

	// Sort terms for FST (must be lexicographic order)
	sortedTerms := make([]string, 0, len(w.terms))
	for t := range w.terms {
		sortedTerms = append(sortedTerms, t)
	}
	sort.Strings(sortedTerms)

	// Create postings builder and term dictionary writer
	pb := newSegmentPostingsBuilder()
	tdw, err := newTermDictWriter(filepath.Join(w.dir, "segment.tdi"))
	if err != nil {
		return fmt.Errorf("termdict writer: %w", err)
	}

	// Write each term's postings as a self-contained blob
	for _, term := range sortedTerms {
		tp := w.terms[term]
		postingsOff := pb.writeTermPostings(
			tp.docs, tp.freqs, tp.norms, tp.positions, true,
		)

		ti := termInfo{
			docFreq:      uint32(len(tp.docs)),
			postingsOff:  postingsOff,
			hasPositions: true,
		}
		if err := tdw.add(term, ti); err != nil {
			return fmt.Errorf("termdict add %q: %w", term, err)
		}
	}

	if err := tdw.close(); err != nil {
		return fmt.Errorf("termdict close: %w", err)
	}

	// Write posting files (.doc contains all blobs, .pos has position data)
	if err := pb.writeTo(
		filepath.Join(w.dir, "segment.doc"),
		filepath.Join(w.dir, "segment.pos"),
	); err != nil {
		return fmt.Errorf("postings write: %w", err)
	}

	// Write field norms
	fnmPath := filepath.Join(w.dir, "segment.fnm")
	if err := os.WriteFile(fnmPath, w.norms, 0644); err != nil {
		return fmt.Errorf("fieldnorms write: %w", err)
	}

	// Close stored fields
	if err := w.store.close(); err != nil {
		return fmt.Errorf("store close: %w", err)
	}

	// Write segment metadata
	avgDocLen := float64(0)
	if w.docCount > 0 {
		avgDocLen = float64(w.totalTokens) / float64(w.docCount)
	}
	meta := segmentMeta{
		DocCount:  w.docCount,
		AvgDocLen: avgDocLen,
	}
	metaJSON, _ := json.MarshalIndent(meta, "", "  ")
	if err := os.WriteFile(filepath.Join(w.dir, "segment.meta"), metaJSON, 0644); err != nil {
		return fmt.Errorf("meta write: %w", err)
	}

	return nil
}

// --- Index-level metadata (lotus.meta) ---

type indexMeta struct {
	Version    int      `json:"version"`
	DocCount   uint64   `json:"doc_count"`
	AvgDocLen  float64  `json:"avg_doc_len"`
	Segments   []string `json:"segments"`
	NextSegSeq uint64   `json:"next_seg_seq"`
}

func loadIndexMeta(dir string) (*indexMeta, error) {
	path := filepath.Join(dir, "lotus.meta")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &indexMeta{Version: 1}, nil
		}
		return nil, err
	}
	var m indexMeta
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func saveIndexMeta(dir string, m *indexMeta) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "lotus.meta"), data, 0644)
}

var globalSegSeq atomic.Uint64

func nextSegmentName(m *indexMeta) string {
	seq := m.NextSegSeq
	m.NextSegSeq++
	return fmt.Sprintf("seg_%08d", seq)
}
