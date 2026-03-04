package dahlia

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
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

	loadMu sync.Mutex
}

func openSegmentReader(dir string) (*segmentReader, error) {
	sr := &segmentReader{dir: dir}
	if err := sr.loadMeta(); err != nil {
		return nil, err
	}
	if err := sr.ensureLoaded(); err != nil {
		return nil, err
	}
	return sr, nil
}

// lookupTerm returns a posting iterator and term info for the given term.
func (sr *segmentReader) lookupTerm(term string) (*postingIterator, termInfo, bool) {
	if err := sr.ensureLoaded(); err != nil {
		return nil, termInfo{}, false
	}
	ti, found := sr.termDict.lookup(term)
	if !found {
		return nil, termInfo{}, false
	}
	it := newPostingIterator(sr.docData, sr.freqData, sr.posData, ti.postingsOff)
	return it, ti, true
}

// lookupTermInfo returns term metadata without constructing a postings iterator.
func (sr *segmentReader) lookupTermInfo(term string) (termInfo, bool) {
	if err := sr.ensureLoaded(); err != nil {
		return termInfo{}, false
	}
	return sr.termDict.lookup(term)
}

// getDoc retrieves a stored document by local docID.
func (sr *segmentReader) getDoc(docID uint32) (id string, text []byte, err error) {
	if err := sr.ensureLoaded(); err != nil {
		return "", nil, err
	}
	return sr.store.getDoc(docID)
}

// fieldNorm returns the field norm byte for a local docID.
func (sr *segmentReader) fieldNorm(docID uint32) uint8 {
	if err := sr.ensureLoaded(); err != nil {
		return 0
	}
	if int(docID) < len(sr.norms) {
		return sr.norms[docID]
	}
	return 0
}

func newLazySegmentReader(dir string, meta segmentMeta) *segmentReader {
	return &segmentReader{
		dir:  dir,
		meta: meta,
	}
}

func (sr *segmentReader) loadMeta() error {
	metaPath := filepath.Join(sr.dir, segMetaFile)
	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		return fmt.Errorf("read segment meta: %w", err)
	}
	if err := json.Unmarshal(metaData, &sr.meta); err != nil {
		return fmt.Errorf("parse segment meta: %w", err)
	}
	return nil
}

func (sr *segmentReader) ensureLoaded() error {
	if sr.termDict != nil {
		return nil
	}

	sr.loadMu.Lock()
	defer sr.loadMu.Unlock()
	if sr.termDict != nil {
		return nil
	}
	if sr.meta.DocCount == 0 {
		if err := sr.loadMeta(); err != nil {
			return err
		}
	}

	tdiData, err := mmapFile(filepath.Join(sr.dir, segTermDictFile))
	if err != nil {
		sr.Close()
		return fmt.Errorf("mmap term dict: %w", err)
	}
	sr.mmapped = append(sr.mmapped, tdiData)
	sr.termDict, err = openTermDictReader(tdiData)
	if err != nil {
		sr.Close()
		return fmt.Errorf("open term dict: %w", err)
	}

	sr.docData, err = mmapFile(filepath.Join(sr.dir, segDocFile))
	if err != nil {
		sr.Close()
		return fmt.Errorf("mmap doc data: %w", err)
	}
	sr.mmapped = append(sr.mmapped, sr.docData)

	sr.freqData, err = mmapFile(filepath.Join(sr.dir, segFreqFile))
	if err != nil {
		sr.Close()
		return fmt.Errorf("mmap freq data: %w", err)
	}
	sr.mmapped = append(sr.mmapped, sr.freqData)

	sr.posData, err = mmapFile(filepath.Join(sr.dir, segPosFile))
	if err != nil {
		sr.Close()
		return fmt.Errorf("mmap pos data: %w", err)
	}
	sr.mmapped = append(sr.mmapped, sr.posData)

	storeData, err := mmapFile(filepath.Join(sr.dir, segStoreFile))
	if err != nil {
		sr.Close()
		return fmt.Errorf("mmap store: %w", err)
	}
	sr.mmapped = append(sr.mmapped, storeData)
	sr.store, err = openStoreReader(storeData)
	if err != nil {
		sr.Close()
		return fmt.Errorf("open store: %w", err)
	}

	sr.norms, err = mmapFile(filepath.Join(sr.dir, segFieldNormFile))
	if err != nil {
		sr.Close()
		return fmt.Errorf("mmap field norms: %w", err)
	}
	sr.mmapped = append(sr.mmapped, sr.norms)

	return nil
}

// Close unmaps all files.
func (sr *segmentReader) Close() error {
	if sr.termDict != nil {
		sr.termDict.close()
		sr.termDict = nil
	}
	for _, data := range sr.mmapped {
		munmapFile(data)
	}
	sr.mmapped = nil
	sr.docData = nil
	sr.freqData = nil
	sr.posData = nil
	sr.store = nil
	sr.norms = nil
	return nil
}
