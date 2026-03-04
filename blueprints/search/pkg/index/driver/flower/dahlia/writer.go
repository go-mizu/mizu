package dahlia

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// segmentMeta holds per-segment metadata persisted as segment.meta JSON.
type segmentMeta struct {
	DocCount  uint32  `json:"doc_count"`
	AvgDocLen float64 `json:"avg_doc_len"`
}

// indexMeta holds index-level metadata persisted as dahlia.meta JSON.
type indexMeta struct {
	Version    int      `json:"version"`
	DocCount   uint64   `json:"doc_count"`
	AvgDocLen  float64  `json:"avg_doc_len"`
	Segments   []string `json:"segments"`
	NextSegSeq uint64   `json:"next_seg_seq"`
}

// segmentWriter accumulates documents and flushes a complete segment directory.
type segmentWriter struct {
	terms       map[string]*termPostings
	store       storeWriter
	norms       []uint8
	docCount    uint32
	totalTokens uint64
	memEstimate int // rough memory usage estimate
}

func newSegmentWriter() *segmentWriter {
	return &segmentWriter{
		terms: make(map[string]*termPostings),
	}
}

// addDoc analyzes and indexes a single document.
func (sw *segmentWriter) addDoc(docID string, text []byte) {
	localID := sw.docCount
	sw.docCount++

	tokens := analyzeWithPositions(string(text))
	docLen := uint32(len(tokens))
	normByte := encodeFieldNorm(docLen)
	sw.norms = append(sw.norms, normByte)
	sw.totalTokens += uint64(docLen)

	// Build per-term frequency and position maps
	type termData struct {
		freq      uint32
		positions []uint32
	}
	termMap := make(map[string]*termData, len(tokens)/2+1)
	for _, tok := range tokens {
		td, ok := termMap[tok.term]
		if !ok {
			td = &termData{}
			termMap[tok.term] = td
		}
		td.freq++
		td.positions = append(td.positions, uint32(tok.pos))
	}

	// Add to per-term posting lists
	for term, td := range termMap {
		tp, ok := sw.terms[term]
		if !ok {
			tp = &termPostings{}
			sw.terms[term] = tp
			sw.memEstimate += len(term) + 64
		}
		tp.docs = append(tp.docs, localID)
		tp.freqs = append(tp.freqs, td.freq)
		tp.norms = append(tp.norms, normByte)
		tp.positions = append(tp.positions, td.positions)
		sw.memEstimate += 16 + len(td.positions)*4
	}

	sw.store.addDoc(localID, docID, text)
	sw.memEstimate += len(docID) + len(text)
}

// estimatedMemory returns the approximate memory usage of buffered data.
func (sw *segmentWriter) estimatedMemory() int {
	return sw.memEstimate
}

// flush writes the complete segment directory. Returns the segment metadata.
func (sw *segmentWriter) flush(dir string) (*segmentMeta, error) {
	if sw.docCount == 0 {
		return nil, fmt.Errorf("no documents to flush")
	}

	// Create segment directory
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	// Sort terms lexicographically
	sortedTerms := make([]string, 0, len(sw.terms))
	for term := range sw.terms {
		sortedTerms = append(sortedTerms, term)
	}
	sort.Strings(sortedTerms)

	// Build postings + term dictionary
	pw := &postingsWriter{}
	tdw, err := newTermDictWriter()
	if err != nil {
		return nil, fmt.Errorf("term dict writer: %w", err)
	}

	for _, term := range sortedTerms {
		tp := sw.terms[term]
		docOff := pw.writeTerm(tp)
		ti := termInfo{
			docFreq:      uint32(len(tp.docs)),
			postingsOff:  docOff,
			hasPositions: len(tp.positions) > 0 && len(tp.positions[0]) > 0,
		}
		if err := tdw.add(term, ti); err != nil {
			return nil, fmt.Errorf("add term %q: %w", term, err)
		}
	}

	// Serialize term dictionary
	tdiData, err := tdw.finish()
	if err != nil {
		return nil, fmt.Errorf("term dict finish: %w", err)
	}

	// Store data
	storeData := sw.store.finish()

	// Segment metadata
	avgDocLen := float64(0)
	if sw.docCount > 0 {
		avgDocLen = float64(sw.totalTokens) / float64(sw.docCount)
	}
	meta := &segmentMeta{
		DocCount:  sw.docCount,
		AvgDocLen: avgDocLen,
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return nil, err
	}

	// Write all files atomically (write to temp, rename)
	files := map[string][]byte{
		segTermDictFile:  tdiData,
		segDocFile:       pw.docBytes(),
		segFreqFile:      pw.freqBytes(),
		segPosFile:       pw.posBytes(),
		segStoreFile:     storeData,
		segFieldNormFile: sw.norms,
		segMetaFile:      metaJSON,
	}

	for name, data := range files {
		path := filepath.Join(dir, name)
		if err := writeFileAtomic(path, data); err != nil {
			return nil, fmt.Errorf("write %s: %w", name, err)
		}
	}

	return meta, nil
}

// writeFileAtomic writes data to path via temp file + rename.
func writeFileAtomic(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// saveIndexMeta persists the index metadata.
func saveIndexMeta(dir string, meta *indexMeta) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(filepath.Join(dir, metaFile), data)
}

// loadIndexMeta loads the index metadata, or returns a fresh default.
func loadIndexMeta(dir string) (*indexMeta, error) {
	path := filepath.Join(dir, metaFile)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &indexMeta{Version: 1, NextSegSeq: 1}, nil
	}
	if err != nil {
		return nil, err
	}
	var meta indexMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}
