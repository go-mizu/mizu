//go:build cgo
// +build cgo

package fts_zig

/*
#cgo LDFLAGS: -L${SRCDIR}/.. -lfts_zig -lc
#cgo CFLAGS: -I${SRCDIR}/..

#include "fts_zig.h"
#include <stdlib.h>
*/
import "C"
import (
	"sync"
	"unsafe"
)

// cgoDriver implements Driver using CGO.
type cgoDriver struct {
	mu       sync.RWMutex
	profile  Profile
	builder  C.fts_handle_t
	index    C.fts_handle_t
	built    bool
	docCount uint32
}

func newCGODriver(cfg Config) (Driver, error) {
	d := &cgoDriver{
		profile: cfg.Profile,
	}

	// Create builder based on profile
	switch cfg.Profile {
	case ProfileSpeed:
		d.builder = C.fts_speed_builder_create()
	case ProfileBalanced:
		d.builder = C.fts_balanced_builder_create()
	case ProfileCompact:
		d.builder = C.fts_compact_builder_create()
	}

	if d.builder == nil {
		return nil, ErrNotInitialized
	}

	return d, nil
}

func (d *cgoDriver) AddDocument(text string) (uint32, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.built {
		return 0, ErrAlreadyBuilt
	}

	cText := C.CString(text)
	defer C.free(unsafe.Pointer(cText))

	var ret C.int
	switch d.profile {
	case ProfileSpeed:
		ret = C.fts_speed_builder_add(d.builder, cText, C.size_t(len(text)))
	case ProfileBalanced:
		ret = C.fts_balanced_builder_add(d.builder, cText, C.size_t(len(text)))
	case ProfileCompact:
		ret = C.fts_compact_builder_add(d.builder, cText, C.size_t(len(text)))
	}

	if ret != 0 {
		return 0, ErrInvalidHandle
	}

	docID := d.docCount
	d.docCount++
	return docID, nil
}

func (d *cgoDriver) AddDocuments(texts []string) error {
	for _, text := range texts {
		if _, err := d.AddDocument(text); err != nil {
			return err
		}
	}
	return nil
}

func (d *cgoDriver) Build() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.built {
		return ErrAlreadyBuilt
	}

	switch d.profile {
	case ProfileSpeed:
		d.index = C.fts_speed_builder_build(d.builder)
		C.fts_speed_builder_destroy(d.builder)
	case ProfileBalanced:
		d.index = C.fts_balanced_builder_build(d.builder)
		C.fts_balanced_builder_destroy(d.builder)
	case ProfileCompact:
		d.index = C.fts_compact_builder_build(d.builder)
		C.fts_compact_builder_destroy(d.builder)
	}

	d.builder = nil

	if d.index == nil {
		return ErrInvalidHandle
	}

	d.built = true
	return nil
}

func (d *cgoDriver) Search(query string, limit int) ([]SearchResult, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if !d.built {
		return nil, ErrNotBuilt
	}

	cQuery := C.CString(query)
	defer C.free(unsafe.Pointer(cQuery))

	results := make([]C.fts_search_result_t, limit)
	var count C.int

	switch d.profile {
	case ProfileSpeed:
		count = C.fts_speed_search(d.index, cQuery, C.size_t(len(query)),
			&results[0], C.size_t(limit))
	case ProfileBalanced:
		count = C.fts_balanced_search(d.index, cQuery, C.size_t(len(query)),
			&results[0], C.size_t(limit))
	case ProfileCompact:
		count = C.fts_compact_search(d.index, cQuery, C.size_t(len(query)),
			&results[0], C.size_t(limit))
	}

	out := make([]SearchResult, int(count))
	for i := 0; i < int(count); i++ {
		out[i] = SearchResult{
			DocID: uint32(results[i].doc_id),
			Score: float32(results[i].score),
		}
	}

	return out, nil
}

func (d *cgoDriver) Stats() (Stats, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if !d.built {
		return Stats{DocCount: d.docCount}, nil
	}

	var stats C.fts_stats_t

	switch d.profile {
	case ProfileSpeed:
		C.fts_speed_stats(d.index, &stats)
	default:
		// Other profiles don't have stats function yet
		return Stats{DocCount: d.docCount}, nil
	}

	return Stats{
		DocCount:    uint32(stats.doc_count),
		TermCount:   uint32(stats.term_count),
		MemoryBytes: uint64(stats.memory_bytes),
	}, nil
}

func (d *cgoDriver) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.builder != nil {
		switch d.profile {
		case ProfileSpeed:
			C.fts_speed_builder_destroy(d.builder)
		case ProfileBalanced:
			C.fts_balanced_builder_destroy(d.builder)
		case ProfileCompact:
			C.fts_compact_builder_destroy(d.builder)
		}
		d.builder = nil
	}

	if d.index != nil {
		switch d.profile {
		case ProfileSpeed:
			C.fts_speed_destroy(d.index)
		case ProfileBalanced:
			C.fts_balanced_destroy(d.index)
		case ProfileCompact:
			C.fts_compact_destroy(d.index)
		}
		d.index = nil
	}

	return nil
}

// Version returns the fts_zig library version.
func Version() string {
	return C.GoString(C.fts_version())
}

// Hash computes the fts_zig hash of a string (for debugging).
func Hash(text string) uint64 {
	cText := C.CString(text)
	defer C.free(unsafe.Pointer(cText))
	return uint64(C.fts_hash(cText, C.size_t(len(text))))
}
