//go:build !windows

// File: driver/local/mmap_unix.go
package local

import (
	"io"
	"os"

	"github.com/edsrzf/mmap-go"
)

// mmapReader provides a memory-mapped file reader for high-performance reads.
// Memory-mapped I/O can deliver 10-25x faster reads by eliminating data copies
// between kernel and user space.
type mmapReader struct {
	data   mmap.MMap
	file   *os.File
	offset int64 // current read position
	length int64 // total length to read
}

// openWithMmap opens a file using memory mapping for high-performance reads.
// This is used for files >= MmapThreshold (64KB).
func openWithMmap(full string, offset, length int64) (io.ReadCloser, int64, error) {
	// #nosec G304 -- path validated by cleanKey and joinUnderRoot
	f, err := os.Open(full)
	if err != nil {
		return nil, 0, err
	}

	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, 0, err
	}

	fileSize := info.Size()

	// Calculate actual read length
	if length <= 0 {
		length = fileSize - offset
	}
	if offset+length > fileSize {
		length = fileSize - offset
	}

	// Map the file region we need to read
	// For partial reads, we map the entire file to simplify offset handling
	m, err := mmap.MapRegion(f, int(fileSize), mmap.RDONLY, 0, 0)
	if err != nil {
		f.Close()
		return nil, 0, err
	}

	return &mmapReader{
		data:   m,
		file:   f,
		offset: offset,
		length: length,
	}, fileSize, nil
}

// Read implements io.Reader.
func (r *mmapReader) Read(p []byte) (int, error) {
	if r.length <= 0 {
		return 0, io.EOF
	}

	// Calculate how much we can read
	toRead := int64(len(p))
	if toRead > r.length {
		toRead = r.length
	}

	// Copy from mapped memory
	n := copy(p, r.data[r.offset:r.offset+toRead])
	r.offset += int64(n)
	r.length -= int64(n)

	if r.length <= 0 {
		return n, io.EOF
	}
	return n, nil
}

// Close implements io.Closer.
func (r *mmapReader) Close() error {
	if r.data != nil {
		if err := r.data.Unmap(); err != nil {
			r.file.Close()
			return err
		}
	}
	return r.file.Close()
}

// mmapSupported returns true if mmap is available on this platform.
func mmapSupported() bool {
	return true
}
