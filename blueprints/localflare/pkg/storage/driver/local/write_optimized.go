//go:build !windows

// File: driver/local/write_optimized.go
package local

import (
	"hash/fnv"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// =============================================================================
// SHARDED BUFFER POOLS
// =============================================================================
// Reduces lock contention by sharding pools across CPUs.

const numPoolShards = 32

type shardedBufferPool struct {
	pools [numPoolShards]sync.Pool
	size  int
}

func newShardedPool(size int) *shardedBufferPool {
	p := &shardedBufferPool{size: size}
	for i := range p.pools {
		sz := size
		p.pools[i].New = func() interface{} {
			buf := make([]byte, sz)
			return &buf
		}
	}
	return p
}

func (p *shardedBufferPool) Get() []byte {
	// Use fast CPU-local shard selection
	shard := fastrand() % numPoolShards
	return *p.pools[shard].Get().(*[]byte)
}

func (p *shardedBufferPool) Put(buf []byte) {
	if cap(buf) != p.size {
		return
	}
	shard := fastrand() % numPoolShards
	p.pools[shard].Put(&buf)
}

// Optimized buffer pools with sharding
var (
	shardedSmallPool  = newShardedPool(SmallBufferSize)
	shardedMediumPool = newShardedPool(MediumBufferSize)
	shardedLargePool  = newShardedPool(LargeBufferSize)
	shardedHugePool   = newShardedPool(HugeBufferSize)
)

// getShardedBuffer returns a buffer from sharded pool based on size.
func getShardedBuffer(size int64) []byte {
	switch {
	case size <= TinyFileThreshold:
		return shardedSmallPool.Get()
	case size <= SmallFileThreshold:
		return shardedMediumPool.Get()
	case size <= LargeFileThreshold:
		return shardedLargePool.Get()
	default:
		return shardedHugePool.Get()
	}
}

// putShardedBuffer returns a buffer to the appropriate sharded pool.
func putShardedBuffer(buf []byte) {
	switch cap(buf) {
	case SmallBufferSize:
		shardedSmallPool.Put(buf)
	case MediumBufferSize:
		shardedMediumPool.Put(buf)
	case LargeBufferSize:
		shardedLargePool.Put(buf)
	case HugeBufferSize:
		shardedHugePool.Put(buf)
	}
}

// =============================================================================
// LOCK-FREE DIRECTORY CACHE
// =============================================================================

const numDirCacheShards = 256

type lockFreeDirCache struct {
	shards   [numDirCacheShards]dirCacheShard
	hits     atomic.Int64
	misses   atomic.Int64
	maxItems int
}

type dirCacheShard struct {
	mu      sync.RWMutex
	entries map[string]time.Time
}

var optimizedDirCache = &lockFreeDirCache{
	maxItems: DirCacheMaxSize / numDirCacheShards,
}

func init() {
	for i := range optimizedDirCache.shards {
		optimizedDirCache.shards[i].entries = make(map[string]time.Time, 64)
	}
}

func (c *lockFreeDirCache) shardIndex(path string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(path))
	return h.Sum32() % numDirCacheShards
}

func (c *lockFreeDirCache) ensureDir(path string) error {
	shard := &c.shards[c.shardIndex(path)]

	// Fast path: check cache with read lock only
	shard.mu.RLock()
	if t, ok := shard.entries[path]; ok && time.Since(t) < DirCacheTTL {
		shard.mu.RUnlock()
		c.hits.Add(1)
		return nil
	}
	shard.mu.RUnlock()
	c.misses.Add(1)

	// Slow path: create directory
	if err := os.MkdirAll(path, DirPermissions); err != nil {
		return err
	}

	// Update cache
	shard.mu.Lock()
	// Evict if too full (simple random eviction)
	if len(shard.entries) >= c.maxItems {
		for k := range shard.entries {
			delete(shard.entries, k)
			if len(shard.entries) < c.maxItems/2 {
				break
			}
		}
	}
	shard.entries[path] = time.Now()
	shard.mu.Unlock()

	return nil
}

func (c *lockFreeDirCache) invalidate(path string) {
	shard := &c.shards[c.shardIndex(path)]
	shard.mu.Lock()
	delete(shard.entries, path)
	shard.mu.Unlock()
}

// =============================================================================
// PER-CPU TEMP DIRECTORIES
// =============================================================================
// Reduces inode contention during parallel writes.

var (
	tempDirSetup sync.Once
	tempDirs     []string
)

func setupTempDirs(baseDir string) {
	tempDirSetup.Do(func() {
		numCPU := runtime.NumCPU()
		tempDirs = make([]string, numCPU)
		for i := 0; i < numCPU; i++ {
			dir := filepath.Join(baseDir, ".tmp", "cpu"+string(rune('0'+i%10)))
			os.MkdirAll(dir, DirPermissions)
			tempDirs[i] = dir
		}
	})
}

// getTempDirForCPU returns a temp directory for the current CPU.
func getTempDirForCPU(baseDir string) string {
	setupTempDirs(baseDir)
	if len(tempDirs) == 0 {
		return baseDir
	}
	// Use fastrand for CPU-local selection without actual pinning
	idx := fastrand() % uint32(len(tempDirs))
	return tempDirs[idx]
}

// =============================================================================
// FAST RANDOM NUMBER GENERATOR
// =============================================================================
// Used for shard selection without lock contention.
// Uses atomic counter + xorshift for fast pseudo-random distribution.

var fastrandState atomic.Uint64

func init() {
	// Seed with current time
	fastrandState.Store(uint64(time.Now().UnixNano()))
}

func fastrand() uint32 {
	// Simple xorshift* for fast pseudo-random numbers
	for {
		old := fastrandState.Load()
		// xorshift*
		x := old
		x ^= x >> 12
		x ^= x << 25
		x ^= x >> 27
		if fastrandState.CompareAndSwap(old, x) {
			return uint32(x * 0x2545F4914F6CDD1D >> 32)
		}
	}
}

// =============================================================================
// WRITE TRACKING (Eliminate post-write stat)
// =============================================================================

type writeResult struct {
	written int64
	modTime time.Time
}

// trackingWriter wraps a writer and tracks bytes written.
type trackingWriter struct {
	w       *os.File
	written int64
}

func (t *trackingWriter) Write(p []byte) (int, error) {
	n, err := t.w.Write(p)
	t.written += int64(n)
	return n, err
}

func (t *trackingWriter) Close() error {
	return t.w.Close()
}

func (t *trackingWriter) Sync() error {
	return t.w.Sync()
}

func (t *trackingWriter) Name() string {
	return t.w.Name()
}

// =============================================================================
// OBJECT POOL FOR RESPONSE OBJECTS
// =============================================================================
// Reduces allocations in hot paths.

var objectPool = sync.Pool{
	New: func() interface{} {
		return &objectPoolEntry{}
	},
}

type objectPoolEntry struct {
	bucket      string
	key         string
	size        int64
	contentType string
	created     time.Time
	updated     time.Time
}

func (e *objectPoolEntry) reset() {
	e.bucket = ""
	e.key = ""
	e.size = 0
	e.contentType = ""
	e.created = time.Time{}
	e.updated = time.Time{}
}

// =============================================================================
// PARALLEL WRITE CONSTANTS
// =============================================================================

const (
	// ParallelWriteThreshold: files >= this size use parallel chunk writing
	ParallelWriteThreshold = 32 * 1024 * 1024 // 32MB

	// ParallelWriteChunkSize: size of each chunk for parallel writes
	ParallelWriteChunkSize = 4 * 1024 * 1024 // 4MB

	// MaxWriteWorkers: maximum number of parallel write workers
	MaxWriteWorkers = 4
)

// =============================================================================
// PARALLEL FILE WRITER
// =============================================================================
// For very large files, write chunks in parallel using pwrite()

// parallelWriter writes large files using parallel chunk writing.
// Chunks are read from source and written to different file offsets concurrently.
type parallelWriter struct {
	file    *os.File
	size    int64
	written int64
}

// newParallelWriter creates a writer optimized for large files.
// Pre-allocates the file to avoid fragmentation.
func newParallelWriter(path string, size int64) (*parallelWriter, error) {
	// Create file with truncate
	// #nosec G304 -- path validated by caller
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, FilePermissions)
	if err != nil {
		return nil, err
	}

	// Pre-allocate file to final size (reduces fragmentation)
	// This also ensures we have disk space before starting the write
	if size > 0 {
		if err := preallocateIfSupported(f, size); err != nil {
			// Pre-allocation failed, but we can still try to write
			// Seek to end and write byte to reserve space
			if _, err := f.Seek(size-1, 0); err == nil {
				f.Write([]byte{0})
				f.Seek(0, 0)
			}
		}
	}

	return &parallelWriter{
		file: f,
		size: size,
	}, nil
}

// WriteFrom reads from source and writes in parallel chunks.
func (w *parallelWriter) WriteFrom(src io.Reader) (int64, error) {
	// For small writes, use sequential
	if w.size < ParallelWriteThreshold {
		return w.writeSequential(src)
	}

	// For large writes, use parallel chunked approach
	return w.writeParallel(src)
}

func (w *parallelWriter) writeSequential(src io.Reader) (int64, error) {
	buf := shardedHugePool.Get()
	defer shardedHugePool.Put(buf)

	var total int64
	for {
		n, err := src.Read(buf)
		if n > 0 {
			written, werr := w.file.Write(buf[:n])
			total += int64(written)
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
	}
	w.written = total
	return total, nil
}

func (w *parallelWriter) writeParallel(src io.Reader) (int64, error) {
	numWorkers := 4
	if numWorkers > MaxWriteWorkers {
		numWorkers = MaxWriteWorkers
	}

	// Channel for chunks to write
	type chunk struct {
		offset int64
		data   []byte
		err    error
	}

	chunks := make(chan chunk, numWorkers*2)
	var wg sync.WaitGroup
	var writeErr error
	var writeErrOnce sync.Once
	var writeFailed atomic.Bool

	setWriteErr := func(err error) {
		if err == nil {
			return
		}
		writeErrOnce.Do(func() {
			writeErr = err
			writeFailed.Store(true)
		})
	}

	// Start writer goroutines
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for c := range chunks {
				if writeFailed.Load() {
					continue
				}
				if c.err != nil {
					setWriteErr(c.err)
					continue
				}
				_, err := w.file.WriteAt(c.data, c.offset)
				if err != nil {
					setWriteErr(err)
				}
			}
		}()
	}

	// Read chunks and send to workers
	var offset int64
	var totalRead int64
	var readErr error
	for totalRead < w.size {
		if writeFailed.Load() {
			break
		}
		chunkSize := int64(ParallelWriteChunkSize)
		if totalRead+chunkSize > w.size {
			chunkSize = w.size - totalRead
		}

		// Read chunk from source
		buf := make([]byte, chunkSize)
		n, err := io.ReadFull(src, buf)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			readErr = err
			break
		}

		if n > 0 {
			chunks <- chunk{offset: offset, data: buf[:n]}
			offset += int64(n)
			totalRead += int64(n)
		}

		if err == io.EOF || n == 0 {
			break
		}
	}

	close(chunks)
	wg.Wait()

	if readErr != nil {
		return totalRead, readErr
	}
	if writeErr != nil {
		return totalRead, writeErr
	}

	w.written = totalRead
	return totalRead, nil
}

// Sync flushes data to disk.
func (w *parallelWriter) Sync() error {
	if NoFsync {
		return nil
	}
	return w.file.Sync()
}

// Close closes the file.
func (w *parallelWriter) Close() error {
	return w.file.Close()
}

// Written returns total bytes written.
func (w *parallelWriter) Written() int64 {
	return w.written
}

// preallocateIfSupported tries to pre-allocate disk space.
// This is a no-op on systems that don't support fallocate.
func preallocateIfSupported(f *os.File, size int64) error {
	// Try to use fallocate on Linux
	// On other systems, this will fail and we fall back to normal writes
	return tryFallocate(f, size)
}

// tryFallocate attempts to use fallocate syscall.
// Implemented in platform-specific files.
func tryFallocate(f *os.File, size int64) error {
	// Default implementation - no-op on unsupported platforms
	// Linux version in write_optimized_linux.go
	_ = f
	_ = size
	return nil
}
