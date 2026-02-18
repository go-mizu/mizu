//go:build !windows

// File: driver/local/mmap_unix.go
package local

import (
	"io"
	"os"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/edsrzf/mmap-go"
)

// =============================================================================
// PARALLEL READ CONSTANTS
// =============================================================================

const (
	// ParallelReadThreshold: files >= this size use parallel chunk reading
	ParallelReadThreshold = 32 * 1024 * 1024 // 32MB

	// ParallelChunkSize: size of each chunk for parallel reads
	ParallelChunkSize = 4 * 1024 * 1024 // 4MB

	// MaxReadWorkers: maximum number of parallel read workers
	MaxReadWorkers = 4
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
// OPTIMIZED: Now accepts fileSize to avoid duplicate stat() call.
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

	return openWithMmapWithSize(f, info.Size(), offset, length)
}

// openWithMmapWithSize opens a file with mmap using pre-known file size.
// This eliminates the duplicate stat() call when caller already knows the size.
func openWithMmapWithSize(f *os.File, fileSize, offset, length int64) (io.ReadCloser, int64, error) {
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

// openWithMmapPrestatted opens file with mmap when file is already open and size known.
// OPTIMIZED: No stat calls at all - caller provides everything.
func openWithMmapPrestatted(f *os.File, fileSize, offset, length int64) (io.ReadCloser, error) {
	// Calculate actual read length
	if length <= 0 {
		length = fileSize - offset
	}
	if offset+length > fileSize {
		length = fileSize - offset
	}

	// Map the file region
	m, err := mmap.MapRegion(f, int(fileSize), mmap.RDONLY, 0, 0)
	if err != nil {
		return nil, err
	}

	return &mmapReader{
		data:   m,
		file:   f,
		offset: offset,
		length: length,
	}, nil
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

// WriteTo implements io.WriterTo for optimized streaming.
// This allows direct copy from mmap to destination without intermediate buffers.
func (r *mmapReader) WriteTo(w io.Writer) (int64, error) {
	if r.length <= 0 {
		return 0, nil
	}

	// For very large data, use chunked writes to avoid holding too much memory
	const maxChunk = 8 * 1024 * 1024 // 8MB chunks
	var total int64

	for r.length > 0 {
		toWrite := r.length
		if toWrite > maxChunk {
			toWrite = maxChunk
		}

		n, err := w.Write(r.data[r.offset : r.offset+toWrite])
		total += int64(n)
		r.offset += int64(n)
		r.length -= int64(n)

		if err != nil {
			return total, err
		}
		if int64(n) != toWrite {
			return total, io.ErrShortWrite
		}
	}

	return total, nil
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

// =============================================================================
// PARALLEL FILE READER
// =============================================================================
// For very large files, read chunks in parallel using pread()

// parallelReader reads large files using parallel chunk fetching.
type parallelReader struct {
	file     *os.File
	fileSize int64
	offset   int64  // Current read position (for streaming)
	length   int64  // Remaining bytes to read
	buf      []byte // Pre-read buffer
	bufPos   int    // Position in buffer
	bufLen   int    // Valid bytes in buffer
	mu       sync.Mutex
}

// newParallelReader creates a reader that prefetches chunks in parallel.
func newParallelReader(f *os.File, fileSize, offset, length int64) *parallelReader {
	if length <= 0 {
		length = fileSize - offset
	}
	if offset+length > fileSize {
		length = fileSize - offset
	}

	return &parallelReader{
		file:     f,
		fileSize: fileSize,
		offset:   offset,
		length:   length,
	}
}

// Read implements io.Reader with parallel prefetching for large files.
func (r *parallelReader) Read(p []byte) (int, error) {
	if r.length <= 0 {
		return 0, io.EOF
	}

	// If we have buffered data, return it
	if r.bufPos < r.bufLen {
		n := copy(p, r.buf[r.bufPos:r.bufLen])
		r.bufPos += n
		r.length -= int64(n)
		if r.length <= 0 {
			return n, io.EOF
		}
		return n, nil
	}

	// Read directly for remaining data
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

// WriteTo implements io.WriterTo with parallel chunk reading.
func (r *parallelReader) WriteTo(w io.Writer) (int64, error) {
	if r.length <= 0 {
		return 0, nil
	}

	// For small files, use simple sequential read
	if r.length < ParallelReadThreshold {
		return r.writeToSequential(w)
	}

	// For large files, use parallel chunk reading
	return r.writeToParallel(w)
}

func (r *parallelReader) writeToSequential(w io.Writer) (int64, error) {
	buf := shardedLargePool.Get()
	defer shardedLargePool.Put(buf)

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

func (r *parallelReader) writeToParallel(w io.Writer) (int64, error) {
	numWorkers := runtime.NumCPU()
	if numWorkers > MaxReadWorkers {
		numWorkers = MaxReadWorkers
	}

	// Calculate chunks
	numChunks := (r.length + ParallelChunkSize - 1) / ParallelChunkSize
	if int64(numWorkers) > numChunks {
		numWorkers = int(numChunks)
	}

	// Channel for ordered chunk delivery
	type chunk struct {
		index int
		data  []byte
		err   error
	}

	chunks := make(chan chunk, numWorkers*2)
	var wg sync.WaitGroup
	var readErr atomic.Value

	// Start worker goroutines to read chunks in parallel
	chunkIndex := int64(0)
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				// Get next chunk index atomically
				idx := atomic.AddInt64(&chunkIndex, 1) - 1
				chunkOffset := r.offset + idx*ParallelChunkSize
				if chunkOffset >= r.offset+r.length {
					return
				}

				// Calculate chunk size
				chunkSize := int64(ParallelChunkSize)
				if chunkOffset+chunkSize > r.offset+r.length {
					chunkSize = r.offset + r.length - chunkOffset
				}

				// Read chunk
				buf := make([]byte, chunkSize)
				n, err := r.file.ReadAt(buf, chunkOffset)
				if err != nil && err != io.EOF {
					chunks <- chunk{index: int(idx), err: err}
					readErr.Store(err)
					return
				}

				chunks <- chunk{index: int(idx), data: buf[:n]}
			}
		}()
	}

	// Close chunks channel when all workers done
	go func() {
		wg.Wait()
		close(chunks)
	}()

	// Collect and write chunks in order
	pending := make(map[int][]byte)
	nextIndex := 0
	var total int64

	for c := range chunks {
		if c.err != nil {
			return total, c.err
		}

		pending[c.index] = c.data

		// Write consecutive chunks
		for {
			data, ok := pending[nextIndex]
			if !ok {
				break
			}
			delete(pending, nextIndex)
			nextIndex++

			n, err := w.Write(data)
			total += int64(n)
			if err != nil {
				return total, err
			}
		}
	}

	r.length = 0
	return total, nil
}

// Close implements io.Closer.
func (r *parallelReader) Close() error {
	return r.file.Close()
}
