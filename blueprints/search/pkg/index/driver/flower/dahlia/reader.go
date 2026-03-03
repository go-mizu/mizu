package dahlia

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// segmentReader provides read access to a flushed segment via mmap.
type segmentReader struct {
	dir      string
	meta     segmentMeta
	termDict *termDictReader
	docData  []byte // mmap'd .doc file
	freqData []byte // mmap'd .freq file
	posData  []byte // mmap'd .pos file
	store    *storeReader
	norms    []byte // mmap'd .fnm file

	// mmap'd slices to munmap on close
	mmapped [][]byte
}

func openSegmentReader(dir string) (*segmentReader, error) {
	sr := &segmentReader{dir: dir}

	// Load segment metadata
	metaPath := filepath.Join(dir, segMetaFile)
	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, fmt.Errorf("read segment meta: %w", err)
	}
	if err := json.Unmarshal(metaData, &sr.meta); err != nil {
		return nil, fmt.Errorf("parse segment meta: %w", err)
	}

	// mmap term dictionary
	tdiData, err := mmapFile(filepath.Join(dir, segTermDictFile))
	if err != nil {
		sr.Close()
		return nil, fmt.Errorf("mmap term dict: %w", err)
	}
	sr.mmapped = append(sr.mmapped, tdiData)
	sr.termDict, err = openTermDictReader(tdiData)
	if err != nil {
		sr.Close()
		return nil, fmt.Errorf("open term dict: %w", err)
	}

	// mmap posting files
	sr.docData, err = mmapFile(filepath.Join(dir, segDocFile))
	if err != nil {
		sr.Close()
		return nil, fmt.Errorf("mmap doc data: %w", err)
	}
	sr.mmapped = append(sr.mmapped, sr.docData)

	sr.freqData, err = mmapFile(filepath.Join(dir, segFreqFile))
	if err != nil {
		sr.Close()
		return nil, fmt.Errorf("mmap freq data: %w", err)
	}
	sr.mmapped = append(sr.mmapped, sr.freqData)

	sr.posData, err = mmapFile(filepath.Join(dir, segPosFile))
	if err != nil {
		sr.Close()
		return nil, fmt.Errorf("mmap pos data: %w", err)
	}
	sr.mmapped = append(sr.mmapped, sr.posData)

	// mmap store
	storeData, err := mmapFile(filepath.Join(dir, segStoreFile))
	if err != nil {
		sr.Close()
		return nil, fmt.Errorf("mmap store: %w", err)
	}
	sr.mmapped = append(sr.mmapped, storeData)
	sr.store, err = openStoreReader(storeData)
	if err != nil {
		sr.Close()
		return nil, fmt.Errorf("open store: %w", err)
	}

	// mmap field norms
	sr.norms, err = mmapFile(filepath.Join(dir, segFieldNormFile))
	if err != nil {
		sr.Close()
		return nil, fmt.Errorf("mmap field norms: %w", err)
	}
	sr.mmapped = append(sr.mmapped, sr.norms)

	return sr, nil
}

// lookupTerm returns a posting iterator and term info for the given term.
func (sr *segmentReader) lookupTerm(term string) (*postingIterator, termInfo, bool) {
	ti, found := sr.termDict.lookup(term)
	if !found {
		return nil, termInfo{}, false
	}
	it := newPostingIterator(sr.docData, sr.freqData, sr.posData, ti.postingsOff)
	return it, ti, true
}

// getDoc retrieves a stored document by local docID.
func (sr *segmentReader) getDoc(docID uint32) (id string, text []byte, err error) {
	return sr.store.getDoc(docID)
}

// fieldNorm returns the field norm byte for a local docID.
func (sr *segmentReader) fieldNorm(docID uint32) uint8 {
	if int(docID) < len(sr.norms) {
		return sr.norms[docID]
	}
	return 0
}

// Close unmaps all files.
func (sr *segmentReader) Close() error {
	if sr.termDict != nil {
		sr.termDict.close()
	}
	for _, data := range sr.mmapped {
		munmapFile(data)
	}
	sr.mmapped = nil
	return nil
}
