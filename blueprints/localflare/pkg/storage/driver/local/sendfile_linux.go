//go:build linux

// File: driver/local/sendfile_linux.go
// Zero-copy file transfer using splice(2)/sendfile(2) on Linux.
package local

import (
	"io"
	"net"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

// =============================================================================
// ZERO-COPY FILE TRANSFER (SENDFILE/SPLICE)
// =============================================================================
// Uses sendfile(2) to transfer data directly from file to socket,
// bypassing user-space entirely for maximum throughput.

// sendfileSupported returns true if sendfile is available.
func sendfileSupported() bool {
	return true
}

// SendfileThreshold is the minimum file size to use sendfile.
const SendfileThreshold = 64 * 1024 // 64KB

// LargeSendfileThreshold is the size above which we use aggressive optimization.
// Set to 4MB so 1MB files continue to use mmap which is faster for medium files.
const LargeSendfileThreshold = 4 * 1024 * 1024 // 4MB

// largeFileReader wraps a file for high-throughput reads using sendfile on Linux.
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

	// Apply readahead hints for sequential reading
	fd := int(f.Fd())
	unix.Fadvise(fd, offset, length, unix.FADV_SEQUENTIAL)
	unix.Fadvise(fd, offset, length, unix.FADV_WILLNEED)

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

// WriteTo implements io.WriterTo with sendfile optimization.
func (r *largeFileReader) WriteTo(w io.Writer) (int64, error) {
	if r.length <= 0 {
		return 0, nil
	}

	// Try to get the underlying TCP connection for sendfile
	conn := getUnderlyingConn(w)
	if conn != nil {
		if tcpConn, ok := conn.(*net.TCPConn); ok {
			return r.sendfileTo(tcpConn)
		}
	}

	// Fall back to buffered copy
	return r.copyTo(w)
}

// sendfileTo uses Linux sendfile for zero-copy transfer to TCP connection.
func (r *largeFileReader) sendfileTo(conn *net.TCPConn) (int64, error) {
	rawConn, err := conn.SyscallConn()
	if err != nil {
		return r.copyTo(conn)
	}

	var total int64
	var sendErr error

	for r.length > 0 {
		var written int64

		controlErr := rawConn.Write(func(fd uintptr) bool {
			// Linux sendfile: sendfile(out_fd, in_fd, offset, count)
			off := r.offset
			toSend := r.length
			if toSend > 1<<30 { // Cap at 1GB per call
				toSend = 1 << 30
			}

			n, err := syscall.Sendfile(int(fd), int(r.file.Fd()), &off, int(toSend))
			written = int64(n)
			if err != nil {
				sendErr = err
				return true
			}
			return true
		})

		if controlErr != nil {
			sendErr = controlErr
			break
		}

		if sendErr != nil {
			if sendErr == syscall.EAGAIN {
				sendErr = nil
				continue
			}
			// Sendfile not supported, fall back
			if sendErr == syscall.EINVAL || sendErr == syscall.ENOSYS {
				remaining, copyErr := r.copyTo(conn)
				return total + remaining, copyErr
			}
			break
		}

		if written == 0 {
			break
		}

		total += written
		r.offset += written
		r.length -= written
	}

	return total, sendErr
}

// copyTo performs regular buffered copy (fallback).
func (r *largeFileReader) copyTo(w io.Writer) (int64, error) {
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
// STREAMING READER
// =============================================================================
// Optimized reader for very large files with internal buffering.

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

	// Apply readahead hints for sequential streaming
	fd := int(f.Fd())
	unix.Fadvise(fd, offset, length, unix.FADV_SEQUENTIAL)
	unix.Fadvise(fd, offset, length, unix.FADV_WILLNEED)

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

// WriteTo implements io.WriterTo with sendfile optimization.
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

	// Try sendfile for zero-copy transfer
	conn := getUnderlyingConn(w)
	if conn != nil {
		if tcpConn, ok := conn.(*net.TCPConn); ok {
			written, err := r.sendfileTo(tcpConn)
			return total + written, err
		}
	}

	// Fall back to buffered copy
	written, err := r.copyTo(w)
	return total + written, err
}

// sendfileTo uses Linux sendfile for zero-copy transfer.
func (r *streamingReader) sendfileTo(conn *net.TCPConn) (int64, error) {
	rawConn, err := conn.SyscallConn()
	if err != nil {
		return r.copyTo(conn)
	}

	var total int64
	var sendErr error

	for r.remaining > 0 {
		var written int64

		controlErr := rawConn.Write(func(fd uintptr) bool {
			off := r.offset
			toSend := r.remaining
			if toSend > 1<<30 {
				toSend = 1 << 30
			}

			n, err := syscall.Sendfile(int(fd), int(r.file.Fd()), &off, int(toSend))
			written = int64(n)
			if err != nil {
				sendErr = err
				return true
			}
			return true
		})

		if controlErr != nil {
			sendErr = controlErr
			break
		}

		if sendErr != nil {
			if sendErr == syscall.EAGAIN {
				sendErr = nil
				continue
			}
			if sendErr == syscall.EINVAL || sendErr == syscall.ENOSYS {
				remaining, copyErr := r.copyTo(conn)
				return total + remaining, copyErr
			}
			break
		}

		if written == 0 {
			break
		}

		total += written
		r.offset += written
		r.remaining -= written
	}

	return total, sendErr
}

// copyTo performs regular buffered copy.
func (r *streamingReader) copyTo(w io.Writer) (int64, error) {
	buf := r.buf
	if buf == nil {
		buf = shardedHugePool.Get()
		defer shardedHugePool.Put(buf)
	}

	var total int64
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

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// getUnderlyingConn extracts the net.Conn from a writer.
func getUnderlyingConn(w io.Writer) net.Conn {
	type connGetter interface {
		Conn() net.Conn
	}

	if conn, ok := w.(net.Conn); ok {
		return conn
	}

	if cg, ok := w.(connGetter); ok {
		return cg.Conn()
	}

	return nil
}
