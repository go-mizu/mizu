//go:build !windows

// File: driver/local/read_optimized_unix.go
package local

import (
	"os"
	"syscall"
)

// =============================================================================
// POSIX FADVISE HINTS (STUBS FOR DARWIN)
// =============================================================================
// On macOS, fadvise is not available. These are no-op stubs.
// On Linux, these would be implemented using unix.Fadvise.

// FadviseSequential hints to the kernel that the file will be read sequentially.
// No-op on macOS, effective on Linux.
func fadviseSequential(f *os.File, offset, length int64) error {
	// macOS doesn't support fadvise - this is a no-op
	_ = f
	_ = offset
	_ = length
	return nil
}

// FadviseRandom hints to the kernel that the file will be read randomly.
func fadviseRandom(f *os.File, offset, length int64) error {
	_ = f
	_ = offset
	_ = length
	return nil
}

// FadviseWillNeed hints that the specified data will be accessed soon.
func fadviseWillNeed(f *os.File, offset, length int64) error {
	_ = f
	_ = offset
	_ = length
	return nil
}

// FadviseDontNeed hints that the specified data won't be needed.
func fadviseDontNeed(f *os.File, offset, length int64) error {
	_ = f
	_ = offset
	_ = length
	return nil
}

// =============================================================================
// MADVISE FOR MMAP REGIONS (STUBS FOR DARWIN)
// =============================================================================
// On macOS, madvise has different semantics. These are safe no-op stubs.

// MadviseSequential hints that the mmap region will be accessed sequentially.
func madviseSequential(data []byte) error {
	_ = data
	return nil
}

// MadviseWillNeed prefetches the specified mmap region into memory.
func madviseWillNeed(data []byte) error {
	_ = data
	return nil
}

// MadviseDontNeed hints that the mmap region is no longer needed.
func madviseDontNeed(data []byte) error {
	_ = data
	return nil
}

// =============================================================================
// OPTIMIZED FILE READER WITH HINTS
// =============================================================================

// optimizedFileReader wraps a file with read optimization hints.
type optimizedFileReader struct {
	file     *os.File
	fileSize int64
	offset   int64
	length   int64
}

// newOptimizedFileReader creates a reader with appropriate hints for the access pattern.
func newOptimizedFileReader(path string, offset, length int64) (*optimizedFileReader, error) {
	// #nosec G304 -- path validated by cleanKey and joinUnderRoot
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}

	fileSize := info.Size()

	// Calculate actual read range
	if length <= 0 {
		length = fileSize - offset
	}
	if offset+length > fileSize {
		length = fileSize - offset
	}

	// Apply appropriate fadvise hints based on access pattern
	if offset == 0 && length == fileSize {
		// Full file read: hint sequential access
		fadviseSequential(f, 0, fileSize)
	} else if length < fileSize/4 {
		// Small range read: hint random access (avoid wasted readahead)
		fadviseRandom(f, 0, fileSize)
	}

	// Prefetch the range we're about to read
	fadviseWillNeed(f, offset, length)

	return &optimizedFileReader{
		file:     f,
		fileSize: fileSize,
		offset:   offset,
		length:   length,
	}, nil
}

func (r *optimizedFileReader) Read(p []byte) (int, error) {
	return r.file.Read(p)
}

func (r *optimizedFileReader) Close() error {
	return r.file.Close()
}

// =============================================================================
// DIRECT I/O SUPPORT (for large files on Linux)
// =============================================================================

// AlignedBufferSize returns the size aligned to the system page size.
func alignedBufferSize(size int64) int64 {
	pageSize := int64(syscall.Getpagesize())
	if size%pageSize == 0 {
		return size
	}
	return ((size / pageSize) + 1) * pageSize
}
