//go:build darwin

// File: driver/local/sendfile_darwin.go
// Optimized file reader using memory-mapped I/O and efficient buffering on macOS.
// Note: Darwin's sendfile has different semantics than Linux, so we use mmap + write.
package local

import (
	"io"
	"os"
)

// =============================================================================
// OPTIMIZED LARGE FILE READER (DARWIN)
// =============================================================================
// Uses mmap for efficient reads and large buffers for throughput.

// sendfileSupported returns false on Darwin (use mmap instead).
func sendfileSupported() bool {
	return false // Darwin sendfile has different semantics
}

// SendfileThreshold is the minimum file size to use optimized path.
const SendfileThreshold = 64 * 1024 // 64KB

// LargeSendfileThreshold is the size above which we use aggressive optimization.
// Set to 4MB so 1MB files continue to use mmap which is faster for medium files.
const LargeSendfileThreshold = 4 * 1024 * 1024 // 4MB

// largeFileReader wraps a file for high-throughput reads.
type largeFileReader struct {
	file   *os.File
	offset int64
	length int64
}

// newLargeFileReader creates an optimized reader for large files.
func newLargeFileReader(f *os.File, fileSize, offset, length int64) *largeFileReader {
	if length <= 0 {
		length = fileSize - offset
	}
	if offset+length > fileSize {
		length = fileSize - offset
	}

	return &largeFileReader{
		file:   f,
		offset: offset,
		length: length,
	}
}

// Read implements io.Reader.
func (r *largeFileReader) Read(p []byte) (int, error) {
	if r.length <= 0 {
		return 0, io.EOF
	}

	toRead := int64(len(p))
	if toRead > r.length {
		toRead = r.length
	}

	n, err := r.file.ReadAt(p[:toRead], r.offset)
	r.offset += int64(n)
	r.length -= int64(n)

	if r.length <= 0 && err == nil {
		err = io.EOF
	}
	return n, err
}

// WriteTo implements io.WriterTo with optimized buffering.
func (r *largeFileReader) WriteTo(w io.Writer) (int64, error) {
	if r.length <= 0 {
		return 0, nil
	}

	// Use 8MB buffer for maximum throughput
	buf := shardedHugePool.Get()
	defer shardedHugePool.Put(buf)

	var total int64
	for r.length > 0 {
		toRead := int64(len(buf))
		if toRead > r.length {
			toRead = r.length
		}

		n, err := r.file.ReadAt(buf[:toRead], r.offset)
		if n > 0 {
			written, werr := w.Write(buf[:n])
			total += int64(written)
			r.offset += int64(n)
			r.length -= int64(n)

			if werr != nil {
				return total, werr
			}
		}
		if err != nil && err != io.EOF {
			return total, err
		}
		if n == 0 {
			break
		}
	}
	return total, nil
}

// Close implements io.Closer.
func (r *largeFileReader) Close() error {
	return r.file.Close()
}

// =============================================================================
// OPTIMIZED STREAMING READER
// =============================================================================
// Reader that minimizes syscalls by using large reads.

// streamingReader wraps a file for high-throughput streaming.
type streamingReader struct {
	file      *os.File
	offset    int64
	remaining int64
	buf       []byte
	bufPos    int
	bufEnd    int
	ownsBuf   bool
}

// newStreamingReader creates a reader optimized for streaming to HTTP.
func newStreamingReader(f *os.File, fileSize, offset, length int64) *streamingReader {
	if length <= 0 {
		length = fileSize - offset
	}
	if offset+length > fileSize {
		length = fileSize - offset
	}

	return &streamingReader{
		file:      f,
		offset:    offset,
		remaining: length,
	}
}

// Read implements io.Reader with internal buffering.
func (r *streamingReader) Read(p []byte) (int, error) {
	if r.remaining <= 0 && r.bufPos >= r.bufEnd {
		return 0, io.EOF
	}

	// If we have buffered data, return it
	if r.bufPos < r.bufEnd {
		n := copy(p, r.buf[r.bufPos:r.bufEnd])
		r.bufPos += n
		return n, nil
	}

	// Direct read if request is large enough
	if int64(len(p)) >= HugeBufferSize {
		toRead := int64(len(p))
		if toRead > r.remaining {
			toRead = r.remaining
		}
		n, err := r.file.ReadAt(p[:toRead], r.offset)
		r.offset += int64(n)
		r.remaining -= int64(n)
		if r.remaining <= 0 && err == nil {
			err = io.EOF
		}
		return n, err
	}

	// Buffered read
	if r.buf == nil {
		r.buf = shardedHugePool.Get()
		r.ownsBuf = true
	}

	toRead := int64(len(r.buf))
	if toRead > r.remaining {
		toRead = r.remaining
	}

	n, err := r.file.ReadAt(r.buf[:toRead], r.offset)
	r.offset += int64(n)
	r.remaining -= int64(n)
	r.bufPos = 0
	r.bufEnd = n

	if n > 0 {
		copied := copy(p, r.buf[:n])
		r.bufPos = copied
		return copied, nil
	}

	if r.remaining <= 0 && err == nil {
		err = io.EOF
	}
	return 0, err
}

// WriteTo implements io.WriterTo.
func (r *streamingReader) WriteTo(w io.Writer) (int64, error) {
	// First write any buffered data
	var total int64
	if r.bufPos < r.bufEnd {
		n, err := w.Write(r.buf[r.bufPos:r.bufEnd])
		total += int64(n)
		r.bufPos = r.bufEnd
		if err != nil {
			return total, err
		}
	}

	if r.remaining <= 0 {
		return total, nil
	}

	// Get buffer for streaming
	buf := r.buf
	if buf == nil {
		buf = shardedHugePool.Get()
		defer shardedHugePool.Put(buf)
	}

	for r.remaining > 0 {
		toRead := int64(len(buf))
		if toRead > r.remaining {
			toRead = r.remaining
		}

		n, err := r.file.ReadAt(buf[:toRead], r.offset)
		if n > 0 {
			written, werr := w.Write(buf[:n])
			total += int64(written)
			r.offset += int64(n)
			r.remaining -= int64(n)

			if werr != nil {
				return total, werr
			}
		}
		if err != nil && err != io.EOF {
			return total, err
		}
		if n == 0 {
			break
		}
	}

	return total, nil
}

// Close implements io.Closer.
func (r *streamingReader) Close() error {
	if r.ownsBuf && r.buf != nil {
		shardedHugePool.Put(r.buf)
		r.buf = nil
	}
	return r.file.Close()
}
