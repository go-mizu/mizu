// Package fts_rust provides a high-performance full-text search driver
// using Rust via FFI/CGO. It supports multiple algorithm profiles:
//   - bmw_simd: Block-Max WAND with SIMD-accelerated posting intersection
//   - roaring_bm25: Roaring bitmaps with BM25 scoring
//   - ensemble: FST + Roaring + Block-Max WAND combined
//   - seismic: Learned sparse retrieval with geometry-cohesive blocks
//   - tantivy: Tantivy library integration for production performance
//   - turbo: Ultra-optimized parallel pipeline for maximum throughput
package fts_rust

/*
#cgo CFLAGS: -I${SRCDIR}/include
#cgo darwin,arm64 LDFLAGS: -L${SRCDIR}/lib -lfts_rust_core -ldl -lm -lpthread -framework Security
#cgo darwin,amd64 LDFLAGS: -L${SRCDIR}/lib -lfts_rust_core -ldl -lm -lpthread -framework Security
#cgo linux LDFLAGS: -L${SRCDIR}/lib -lfts_rust_core -ldl -lm -lpthread
#include "fts_rust.h"
#include <stdlib.h>
#include <stdint.h>
*/
import "C"

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"os"
	"runtime"
	"sync"
	"time"
	"unsafe"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb"
)

const (
	// DefaultProfile is the default search profile (ultra for max throughput)
	DefaultProfile = "ultra"
	// BatchSize for document indexing
	// Balanced at 50k - reduces memory pressure while maintaining parallelism
	BatchSize = 50000
)

// Available profiles
var Profiles = []string{"bmw_simd", "roaring_bm25", "ensemble", "seismic", "tantivy", "turbo", "ultra"}

func init() {
	fineweb.Register("fts_rust", func(cfg fineweb.DriverConfig) (fineweb.Driver, error) {
		return New(cfg)
	})
}

// Driver implements the fineweb.Driver interface using Rust FFI
type Driver struct {
	idx     *C.FtsIndex
	profile string
	dataDir string
	mu      sync.RWMutex
}

// New creates a new fts_rust driver
func New(cfg fineweb.DriverConfig) (*Driver, error) {
	profile := cfg.GetString("profile", DefaultProfile)

	// Validate profile
	valid := false
	for _, p := range Profiles {
		if p == profile {
			valid = true
			break
		}
	}
	if !valid {
		return nil, fmt.Errorf("invalid profile %q, must be one of: %v", profile, Profiles)
	}

	// Create index directory similar to fts_lowmem
	indexDir := cfg.DataDir + "/" + cfg.Language + ".fts_rust"
	if err := os.MkdirAll(indexDir, 0755); err != nil {
		return nil, fmt.Errorf("creating index directory: %w", err)
	}

	dataDirC := C.CString(indexDir)
	defer C.free(unsafe.Pointer(dataDirC))

	profileC := C.CString(profile)
	defer C.free(unsafe.Pointer(profileC))

	// Try to open existing index first
	idx := C.fts_index_open(dataDirC)
	if idx == nil {
		// Create new index
		idx = C.fts_index_create(dataDirC, profileC)
	}

	if idx == nil {
		errMsg := C.GoString(C.fts_last_error())
		return nil, fmt.Errorf("failed to create/open index: %s", errMsg)
	}

	d := &Driver{
		idx:     idx,
		profile: profile,
		dataDir: indexDir,
	}

	runtime.SetFinalizer(d, (*Driver).Close)
	return d, nil
}

// Name returns the driver name
func (d *Driver) Name() string {
	return "fts_rust"
}

// Close releases resources
func (d *Driver) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.idx != nil {
		C.fts_index_close(d.idx)
		d.idx = nil
	}
	return nil
}

// Search performs a full-text search
func (d *Driver) Search(ctx context.Context, query string, limit, offset int) (*fineweb.SearchResult, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.idx == nil {
		return nil, errors.New("index closed")
	}

	start := time.Now()

	queryC := C.CString(query)
	defer C.free(unsafe.Pointer(queryC))

	var result *C.FtsSearchResult
	status := C.fts_search(d.idx, queryC, C.uint32_t(limit), C.uint32_t(offset), &result)
	if status != 0 {
		errMsg := C.GoString(C.fts_last_error())
		return nil, fmt.Errorf("search failed: %s", errMsg)
	}
	defer C.fts_result_free(result)

	// Convert results
	docs := make([]fineweb.Document, 0, int(result.count))
	if result.hits != nil && result.count > 0 {
		hits := unsafe.Slice(result.hits, int(result.count))
		for _, hit := range hits {
			doc := fineweb.Document{
				ID:    C.GoString(hit.id),
				Score: float64(hit.score),
			}
			if hit.text != nil {
				doc.Text = C.GoString(hit.text)
			}
			docs = append(docs, doc)
		}
	}

	return &fineweb.SearchResult{
		Documents: docs,
		Duration:  time.Since(start),
		Method:    fmt.Sprintf("fts_rust/%s", d.profile),
		Total:     int64(result.total),
	}, nil
}

// Import implements the fineweb.Indexer interface
func (d *Driver) Import(ctx context.Context, docs iter.Seq2[fineweb.Document, error], progress fineweb.ProgressFunc) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.idx == nil {
		return errors.New("index closed")
	}

	// Collect documents into batches
	batch := make([]docForBinary, 0, BatchSize)
	var totalIndexed int64

	for doc, err := range docs {
		if err != nil {
			return fmt.Errorf("error reading document: %w", err)
		}

		// Check context
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		batch = append(batch, docForBinary{
			ID:   doc.ID,
			Text: doc.Text,
		})

		if len(batch) >= BatchSize {
			n, err := d.indexBatch(batch)
			if err != nil {
				return err
			}
			totalIndexed += int64(n)
			if progress != nil {
				progress(totalIndexed, -1) // Total unknown
			}
			batch = batch[:0]
		}
	}

	// Index remaining
	if len(batch) > 0 {
		n, err := d.indexBatch(batch)
		if err != nil {
			return err
		}
		totalIndexed += int64(n)
		if progress != nil {
			progress(totalIndexed, -1)
		}
	}

	// Commit
	status := C.fts_index_commit(d.idx)
	if status != 0 {
		errMsg := C.GoString(C.fts_last_error())
		return fmt.Errorf("commit failed: %s", errMsg)
	}

	return nil
}

// docForBinary is the document format for binary serialization
type docForBinary struct {
	ID   string
	Text string
}

// bufferPool provides reusable buffers to reduce allocations
var bufferPool = sync.Pool{
	New: func() any {
		// Start with 10MB buffer, will grow if needed
		return make([]byte, 10*1024*1024)
	},
}

// indexBatch indexes a batch of documents using binary format for max throughput
// Binary format per doc: id_len(u32) + id + text_len(u32) + text
func (d *Driver) indexBatch(docs []docForBinary) (int, error) {
	if len(docs) == 0 {
		return 0, nil
	}

	// Pre-calculate total size for efficient allocation
	totalSize := 0
	for i := range docs {
		totalSize += 8 + len(docs[i].ID) + len(docs[i].Text)
	}

	// Get buffer from pool or allocate new one
	bufInterface := bufferPool.Get()
	buf := bufInterface.([]byte)
	if cap(buf) < totalSize {
		buf = make([]byte, totalSize)
	} else {
		buf = buf[:totalSize]
	}
	defer bufferPool.Put(buf)

	// Build binary buffer - optimized with direct indexing
	pos := 0
	for i := range docs {
		idLen := len(docs[i].ID)
		textLen := len(docs[i].Text)

		// ID length + ID (use direct byte writes for small values)
		buf[pos] = byte(idLen)
		buf[pos+1] = byte(idLen >> 8)
		buf[pos+2] = byte(idLen >> 16)
		buf[pos+3] = byte(idLen >> 24)
		pos += 4
		copy(buf[pos:], docs[i].ID)
		pos += idLen

		// Text length + Text
		buf[pos] = byte(textLen)
		buf[pos+1] = byte(textLen >> 8)
		buf[pos+2] = byte(textLen >> 16)
		buf[pos+3] = byte(textLen >> 24)
		pos += 4
		copy(buf[pos:], docs[i].Text)
		pos += textLen
	}

	result := C.fts_index_batch_binary(
		d.idx,
		(*C.uchar)(unsafe.Pointer(&buf[0])),
		C.size_t(pos),
		C.uint64_t(len(docs)),
		nil, // No progress callback for batches
	)

	if result < 0 {
		errMsg := C.GoString(C.fts_last_error())
		return 0, fmt.Errorf("index batch failed: %s", errMsg)
	}

	return int(result), nil
}

// Count implements the fineweb.Stats interface
func (d *Driver) Count(ctx context.Context) (int64, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.idx == nil {
		return 0, errors.New("index closed")
	}

	return int64(C.fts_doc_count(d.idx)), nil
}

// Info implements the driver info interface
func (d *Driver) Info() *fineweb.DriverInfo {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.idx == nil {
		return &fineweb.DriverInfo{Name: "fts_rust"}
	}

	return &fineweb.DriverInfo{
		Name:        "fts_rust",
		Description: fmt.Sprintf("Rust FTS driver (profile: %s)", d.profile),
		Features:    []string{"fts", "bm25", d.profile},
		External:    false,
	}
}

// MemoryStats returns detailed memory statistics
func (d *Driver) MemoryStats() MemoryStats {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.idx == nil {
		return MemoryStats{}
	}

	stats := C.fts_memory_stats(d.idx)
	return MemoryStats{
		IndexBytes:    uint64(stats.index_bytes),
		TermDictBytes: uint64(stats.term_dict_bytes),
		PostingsBytes: uint64(stats.postings_bytes),
		DocsIndexed:   uint64(stats.docs_indexed),
		MmapBytes:     uint64(stats.mmap_bytes),
	}
}

// MemoryStats contains detailed memory usage information
type MemoryStats struct {
	IndexBytes    uint64
	TermDictBytes uint64
	PostingsBytes uint64
	DocsIndexed   uint64
	MmapBytes     uint64
}

// HeapBytes returns heap-allocated memory (total - mmap)
func (m MemoryStats) HeapBytes() uint64 {
	if m.IndexBytes > m.MmapBytes {
		return m.IndexBytes - m.MmapBytes
	}
	return 0
}

// Profile returns the current search profile
func (d *Driver) Profile() string {
	return d.profile
}

// ListProfiles returns all available profiles
func ListProfiles() []string {
	return Profiles
}
