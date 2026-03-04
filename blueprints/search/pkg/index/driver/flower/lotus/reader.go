package lotus

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// segmentReader opens and reads a flushed segment directory via mmap.
type segmentReader struct {
	dir      string
	meta     segmentMeta
	termDict *termDictReader
	postings *postingsReader
	store    *storeReader
	norms    []byte // raw u8[docCount] fieldnorm bytes
}

func openSegmentReader(dir string) (*segmentReader, error) {
	// Load segment metadata
	metaData, err := os.ReadFile(filepath.Join(dir, "segment.meta"))
	if err != nil {
		return nil, fmt.Errorf("read segment.meta: %w", err)
	}
	var meta segmentMeta
	if err := json.Unmarshal(metaData, &meta); err != nil {
		return nil, fmt.Errorf("parse segment.meta: %w", err)
	}

	// Open term dictionary (FST)
	td, err := openTermDict(filepath.Join(dir, "segment.tdi"))
	if err != nil {
		return nil, fmt.Errorf("open termdict: %w", err)
	}

	// Open posting files
	pr, err := openPostingsReader(
		filepath.Join(dir, "segment.doc"),
		filepath.Join(dir, "segment.freq"),
		filepath.Join(dir, "segment.pos"),
		meta.LastBlockN,
	)
	if err != nil {
		td.close()
		return nil, fmt.Errorf("open postings: %w", err)
	}

	// Open stored fields
	sr, err := openStoreReader(filepath.Join(dir, "segment.store"))
	if err != nil {
		td.close()
		pr.close()
		return nil, fmt.Errorf("open store: %w", err)
	}

	// Load field norms
	norms, err := mmapFile(filepath.Join(dir, "segment.fnm"))
	if err != nil {
		td.close()
		pr.close()
		sr.close()
		return nil, fmt.Errorf("open fieldnorms: %w", err)
	}

	return &segmentReader{
		dir:      dir,
		meta:     meta,
		termDict: td,
		postings: pr,
		store:    sr,
		norms:    norms,
	}, nil
}

func (r *segmentReader) close() error {
	var firstErr error
	if err := r.termDict.close(); err != nil && firstErr == nil {
		firstErr = err
	}
	if err := r.postings.close(); err != nil && firstErr == nil {
		firstErr = err
	}
	if err := r.store.close(); err != nil && firstErr == nil {
		firstErr = err
	}
	if err := mmapRelease(r.norms); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}

// lookupTerm returns a posting iterator for the given term.
func (r *segmentReader) lookupTerm(term string) (*postingIterator, termInfo, bool) {
	ti, found := r.termDict.get(term)
	if !found {
		return nil, termInfo{}, false
	}
	return r.postings.iterator(ti), ti, true
}

// getDoc retrieves a stored document by its local docID.
func (r *segmentReader) getDoc(docID uint32) (string, []byte, error) {
	return r.store.get(docID)
}

// fieldNorm returns the fieldnorm byte for the given local docID.
func (r *segmentReader) fieldNorm(docID uint32) uint8 {
	if int(docID) < len(r.norms) {
		return r.norms[docID]
	}
	return 0
}
