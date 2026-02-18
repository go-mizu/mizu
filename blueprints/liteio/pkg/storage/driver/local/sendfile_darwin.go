//go:build darwin

// File: driver/local/sendfile_darwin.go
// Optimized file reader using memory-mapped I/O and efficient buffering on macOS.
// Note: Darwin's sendfile has different semantics than Linux, so we use mmap + write.
package local

import (
	"io"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
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

// NoCacheThreshold is the size above which we disable file caching (F_NOCACHE).
// Set to 256MB to only affect very large streaming files and avoid cache thrashing.
// For files below this, kernel caching provides better performance for repeated reads.
const NoCacheThreshold = 256 * 1024 * 1024 // 256MB

// setNoCache disables file caching for the given file on Darwin.
// This is beneficial for large files that won't fit in cache anyway.
func setNoCache(f *os.File) error {
	_, err := unix.FcntlInt(f.Fd(), unix.F_NOCACHE, 1)
	return err
}

// setReadahead hints to the kernel about readahead behavior.
// On Darwin, F_RDAHEAD with 1 enables aggressive readahead.
func setReadahead(f *os.File, enable bool) error {
	val := 0
	if enable {
		val = 1
	}
	_, err := unix.FcntlInt(f.Fd(), syscall.F_RDAHEAD, val)
	return err
}

// largeFileReader wraps a file for high-throughput reads.
type largeFileReader struct {
	file       *os.File
	offset     int64
	length     int64
	startOff   int64 // Original start offset for detecting sequential reads
	fileSize   int64 // Total file size
	sequential bool  // True if this is a full-file sequential read
}

// newLargeFileReader creates an optimized reader for large files.
func newLargeFileReader(f *os.File, fileSize, offset, length int64) *largeFileReader {
	if length <= 0 {
		length = fileSize - offset
	}
	if offset+length > fileSize {
		length = fileSize - offset
	}

	// Detect full-file sequential read pattern
	isSequential := offset == 0 && length == fileSize

	// Enable aggressive readahead for sequential access
	if isSequential {
		_ = setReadahead(f, true)
		// For very large files, seek to start for sequential reads
		if fileSize >= NoCacheThreshold {
			_ = setNoCache(f)
		}
	}

	return &largeFileReader{
		file:       f,
		offset:     offset,
		length:     length,
		startOff:   offset,
		fileSize:   fileSize,
		sequential: isSequential,
	}
}

// Read implements io.Reader.
func (r *largeFileReader) Read(p []byte) (int, error) {
	if r.length <= 0 {
		return 0, io.EOF
	}

	toRead := min(int64(len(p)), r.length)

	var n int
	var err error

	// Use sequential read for full-file access (benefits from kernel readahead)
	if r.sequential {
		n, err = r.file.Read(p[:toRead])
	} else {
		n, err = r.file.ReadAt(p[:toRead], r.offset)
	}

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

	// For sequential full-file reads, use io.CopyBuffer which benefits from
	// kernel readahead (Read) instead of random access (ReadAt/pread).
	if r.sequential {
		return r.copySequential(w, buf)
	}

	return r.copyWithPread(w, buf)
}

// copySequential uses sequential Read() which benefits from kernel readahead.
func (r *largeFileReader) copySequential(w io.Writer, buf []byte) (int64, error) {
	var total int64
	for r.length > 0 {
		toRead := min(int64(len(buf)), r.length)

		n, err := r.file.Read(buf[:toRead])
		if n > 0 {
			written, werr := w.Write(buf[:n])
			total += int64(written)
			r.offset += int64(n)
			r.length -= int64(n)

			if werr != nil {
				return total, werr
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return total, err
		}
		if n == 0 {
			break
		}
	}
	return total, nil
}

// copyWithPread uses ReadAt (pread syscall) for random access.
func (r *largeFileReader) copyWithPread(w io.Writer, buf []byte) (int64, error) {
	var total int64
	for r.length > 0 {
		toRead := min(int64(len(buf)), r.length)

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
	file       *os.File
	offset     int64
	remaining  int64
	buf        []byte
	bufPos     int
	bufEnd     int
	ownsBuf    bool
	sequential bool // True if this is a full-file sequential read
}

// newStreamingReader creates a reader optimized for streaming to HTTP.
func newStreamingReader(f *os.File, fileSize, offset, length int64) *streamingReader {
	if length <= 0 {
		length = fileSize - offset
	}
	if offset+length > fileSize {
		length = fileSize - offset
	}

	// Detect full-file sequential read pattern
	isSequential := offset == 0 && length == fileSize

	// Enable aggressive readahead and disable caching for sequential large files
	if isSequential {
		_ = setReadahead(f, true)
		if fileSize >= NoCacheThreshold {
			_ = setNoCache(f)
		}
	}

	return &streamingReader{
		file:       f,
		offset:     offset,
		remaining:  length,
		sequential: isSequential,
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
		toRead := min(int64(len(p)), r.remaining)
		var n int
		var err error
		if r.sequential {
			n, err = r.file.Read(p[:toRead])
		} else {
			n, err = r.file.ReadAt(p[:toRead], r.offset)
		}
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

	toRead := min(int64(len(r.buf)), r.remaining)

	var n int
	var err error
	if r.sequential {
		n, err = r.file.Read(r.buf[:toRead])
	} else {
		n, err = r.file.ReadAt(r.buf[:toRead], r.offset)
	}
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

	// Use sequential reads for full-file access
	if r.sequential {
		return r.writeToSequential(w, buf, total)
	}

	return r.writeToWithPread(w, buf, total)
}

// writeToSequential uses sequential Read() which benefits from kernel readahead.
func (r *streamingReader) writeToSequential(w io.Writer, buf []byte, total int64) (int64, error) {
	for r.remaining > 0 {
		toRead := min(int64(len(buf)), r.remaining)

		n, err := r.file.Read(buf[:toRead])
		if n > 0 {
			written, werr := w.Write(buf[:n])
			total += int64(written)
			r.offset += int64(n)
			r.remaining -= int64(n)

			if werr != nil {
				return total, werr
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return total, err
		}
		if n == 0 {
			break
		}
	}
	return total, nil
}

// writeToWithPread uses ReadAt (pread syscall) for random access.
func (r *streamingReader) writeToWithPread(w io.Writer, buf []byte, total int64) (int64, error) {
	for r.remaining > 0 {
		toRead := min(int64(len(buf)), r.remaining)

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
