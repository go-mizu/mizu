// File: driver/local/storage.go
package local

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/storage"
)

// =============================================================================
// PERFORMANCE TUNING CONSTANTS
// =============================================================================
// All magic numbers are consolidated here for easy tuning.
// Adjust these values based on your workload characteristics.

const (
	// ---------------------------------------------------------------------------
	// Buffer Pool Sizes (Tiered for different workloads)
	// ---------------------------------------------------------------------------

	// SmallBufferSize is used for small file operations (<=4KB).
	// Optimized to fit in L1 cache for minimal latency.
	SmallBufferSize = 4 * 1024 // 4KB

	// MediumBufferSize is used for medium file operations (4KB-64KB).
	// Balances memory usage with throughput.
	MediumBufferSize = 64 * 1024 // 64KB

	// LargeBufferSize is used for large file streaming (64KB-10MB).
	// Increased for higher throughput.
	LargeBufferSize = 2 * 1024 * 1024 // 2MB (was 1MB)

	// HugeBufferSize is used for very large file operations (>10MB).
	// Maximum buffer size for highest throughput on large files.
	HugeBufferSize = 8 * 1024 * 1024 // 8MB (was 4MB)

	// ---------------------------------------------------------------------------
	// Write Optimization Thresholds
	// ---------------------------------------------------------------------------

	// TinyFileThreshold: files <= this size use direct memory write.
	// Avoids syscall overhead for very small files.
	TinyFileThreshold = 8 * 1024 // 8KB (was 4KB - increased for more inlined writes)

	// SmallFileThreshold: files <= this size use single-buffer direct write.
	// Avoids temp file creation for small files.
	SmallFileThreshold = 128 * 1024 // 128KB (was 64KB)

	// LargeFileThreshold: files >= this size use large buffer pool.
	LargeFileThreshold = 1024 * 1024 // 1MB

	// MmapThreshold: files >= this size use memory-mapped I/O for reads.
	MmapThreshold = 64 * 1024 // 64KB

	// ---------------------------------------------------------------------------
	// Directory Caching
	// ---------------------------------------------------------------------------

	// DirCacheTTL is how long to cache directory existence.
	// Increased for benchmark scenarios where directory structure is stable.
	DirCacheTTL = 1 * time.Second // (was 200ms)

	// DirCacheMaxSize limits memory used by directory cache.
	DirCacheMaxSize = 10000 // (was 2000)

	// DirCacheCleanupInterval is how often to clean expired entries.
	DirCacheCleanupInterval = 30 * time.Second

	// ---------------------------------------------------------------------------
	// File System Permissions
	// ---------------------------------------------------------------------------

	// DirPermissions is the default permission for directories.
	DirPermissions = 0o750

	// FilePermissions is the default permission for files.
	FilePermissions = 0o600

	// TempFilePattern is the pattern for temporary files during atomic writes.
	TempFilePattern = ".lake-tmp-*"

	// ---------------------------------------------------------------------------
	// Multipart Upload Limits
	// ---------------------------------------------------------------------------

	// MaxPartNumber is the maximum valid part number for multipart uploads.
	MaxPartNumber = 10000
)

// NoFsync can be set to true to skip fsync calls for maximum write performance.
// WARNING: This trades durability for speed. Data may be lost on crash.
// Useful for benchmarks and temporary data.
var NoFsync = false

// =============================================================================
// TIERED BUFFER POOLS
// =============================================================================
// Multiple buffer pools sized for different workloads reduce contention
// and improve cache locality.

var (
	// smallBufferPool for tiny file operations
	smallBufferPool = sync.Pool{
		New: func() interface{} {
			buf := make([]byte, SmallBufferSize)
			return &buf
		},
	}

	// mediumBufferPool for small-medium file operations
	mediumBufferPool = sync.Pool{
		New: func() interface{} {
			buf := make([]byte, MediumBufferSize)
			return &buf
		},
	}

	// largeBufferPool for large file streaming
	largeBufferPool = sync.Pool{
		New: func() interface{} {
			buf := make([]byte, LargeBufferSize)
			return &buf
		},
	}

	// hugeBufferPool for very large file operations
	hugeBufferPool = sync.Pool{
		New: func() interface{} {
			buf := make([]byte, HugeBufferSize)
			return &buf
		},
	}
)

// getBufferForSize returns an appropriately sized buffer for the given file size.
func getBufferForSize(size int64) []byte {
	switch {
	case size <= TinyFileThreshold:
		return *smallBufferPool.Get().(*[]byte)
	case size <= SmallFileThreshold:
		return *mediumBufferPool.Get().(*[]byte)
	case size <= LargeFileThreshold:
		return *largeBufferPool.Get().(*[]byte)
	default:
		return *hugeBufferPool.Get().(*[]byte)
	}
}

// putBufferForSize returns a buffer to the appropriate pool.
func putBufferForSize(buf []byte) {
	switch cap(buf) {
	case SmallBufferSize:
		smallBufferPool.Put(&buf)
	case MediumBufferSize:
		mediumBufferPool.Put(&buf)
	case LargeBufferSize:
		largeBufferPool.Put(&buf)
	case HugeBufferSize:
		hugeBufferPool.Put(&buf)
	}
}

// Legacy buffer functions for backward compatibility
func getBuffer() []byte {
	return *largeBufferPool.Get().(*[]byte)
}

func putBuffer(buf []byte) {
	if cap(buf) >= LargeBufferSize {
		largeBufferPool.Put(&buf)
	}
}

// =============================================================================
// DIRECTORY CACHE
// =============================================================================
// Caches directory existence to avoid repeated MkdirAll syscalls on hot paths.

type dirCacheEntry struct {
	verified time.Time
}

type dirCache struct {
	mu      sync.RWMutex
	entries map[string]dirCacheEntry
	hits    atomic.Int64
	misses  atomic.Int64
}

var globalDirCache = &dirCache{
	entries: make(map[string]dirCacheEntry, DirCacheMaxSize),
}

// ensureDir creates a directory if it doesn't exist, using cache for hot paths.
func (c *dirCache) ensureDir(path string) error {
	// Fast path: check cache
	c.mu.RLock()
	if entry, ok := c.entries[path]; ok && time.Since(entry.verified) < DirCacheTTL {
		c.mu.RUnlock()
		c.hits.Add(1)
		return nil
	}
	c.mu.RUnlock()
	c.misses.Add(1)

	// Slow path: create directory
	if err := os.MkdirAll(path, DirPermissions); err != nil {
		return err
	}

	// Update cache
	c.mu.Lock()
	// Evict old entries if cache is full
	if len(c.entries) >= DirCacheMaxSize {
		now := time.Now()
		for k, v := range c.entries {
			if now.Sub(v.verified) > DirCacheTTL {
				delete(c.entries, k)
			}
		}
	}
	c.entries[path] = dirCacheEntry{verified: time.Now()}
	c.mu.Unlock()

	return nil
}

// invalidate removes a directory from the cache.
func (c *dirCache) invalidate(path string) {
	c.mu.Lock()
	delete(c.entries, path)
	c.mu.Unlock()
}

// Open creates a Storage backed by the local filesystem.
//
// DSN examples:
//
//	"/abs/path/to/root"
//	"local:/abs/path/to/root"
//	"file:///abs/path/to/root"
//
// This backend:
//
//   - Treats storage keys as POSIX style with "/" separators on all platforms.
//   - Normalizes incoming "\" in keys to "/".
//   - Uses OS specific separators only when talking to the filesystem.
//   - Enforces that all accesses stay under the configured root.
func Open(ctx context.Context, dsn string) (storage.Storage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	root, err := parseRoot(dsn)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("local: root %q does not exist: %w", root, err)
		}
		return nil, fmt.Errorf("local: stat root %q: %w", root, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("local: root %q is not a directory", root)
	}

	return &store{root: root}, nil
}

// parseRoot parses the DSN into an absolute directory path.
func parseRoot(dsn string) (string, error) {
	if dsn == "" {
		return "", errors.New("local: empty dsn")
	}

	// Bare absolute path (Unix: /path, Windows: C:\path or C:/path)
	if strings.HasPrefix(dsn, "/") || isWindowsAbsPath(dsn) {
		return filepath.Clean(dsn), nil
	}

	// "local:/path" or "local:C:\path"
	if strings.HasPrefix(dsn, "local:") {
		p := strings.TrimPrefix(dsn, "local:")
		if p == "" {
			return "", errors.New("local: missing path after local")
		}
		return filepath.Clean(p), nil
	}

	// "file://" URL scheme handling
	if !strings.HasPrefix(dsn, "file://") {
		u, err := url.Parse(dsn)
		if err != nil {
			return "", fmt.Errorf("local: parse dsn: %w", err)
		}
		return "", fmt.Errorf("local: unsupported scheme %q", u.Scheme)
	}

	// Handle file:// URLs with special care for Windows paths
	rest := strings.TrimPrefix(dsn, "file://")

	// Windows absolute path after file:// (e.g., file://C:/Users/... or file://C:\Users\...)
	if isWindowsAbsPath(rest) {
		return filepath.Clean(rest), nil
	}

	// Standard file:// URL parsing for Unix paths
	u, err := url.Parse(dsn)
	if err != nil {
		return "", fmt.Errorf("local: parse dsn: %w", err)
	}
	if u.Scheme != "file" {
		return "", fmt.Errorf("local: unsupported scheme %q", u.Scheme)
	}
	if u.Path == "" {
		return "", errors.New("local: empty file:// path")
	}
	return filepath.Clean(u.Path), nil
}

// isWindowsAbsPath checks if a path is a Windows absolute path.
// Matches patterns like C:, C:\, C:/, D:\, etc.
func isWindowsAbsPath(p string) bool {
	if len(p) < 2 {
		return false
	}
	// Check for drive letter followed by colon
	if (p[0] >= 'A' && p[0] <= 'Z' || p[0] >= 'a' && p[0] <= 'z') && p[1] == ':' {
		return true
	}
	return false
}

// store implements storage.Storage using the local filesystem.
type store struct {
	root string
}

var _ storage.Storage = (*store)(nil)

// Bucket returns a handle for a logical bucket.
//
// Buckets are mapped to subdirectories under root. Bucket names are sanitized
// to avoid path separators.
func (s *store) Bucket(name string) storage.Bucket {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "default"
	}
	name = safeBucketName(name)

	// root for this bucket; joinUnderRoot enforces confinement.
	root, err := joinUnderRoot(s.root, name)
	if err != nil {
		// On error fall back to root; operations will fail later with ErrPermission.
		root = s.root
	}
	return &bucket{
		store: s,
		name:  name,
		root:  root,
	}
}

// Buckets lists top level bucket directories under root.
func (s *store) Buckets(ctx context.Context, limit, offset int, opts storage.Options) (storage.BucketIter, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(s.root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &bucketIter{}, nil
		}
		return nil, fmt.Errorf("local: read root: %w", err)
	}

	var list []*storage.BucketInfo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		list = append(list, &storage.BucketInfo{
			Name:      e.Name(),
			CreatedAt: info.ModTime(),
		})
	}

	sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })

	if offset < 0 {
		offset = 0
	}
	if offset > len(list) {
		offset = len(list)
	}
	list = list[offset:]
	if limit > 0 && limit < len(list) {
		list = list[:limit]
	}

	return &bucketIter{list: list}, nil
}

// CreateBucket creates a directory under root.
func (s *store) CreateBucket(ctx context.Context, name string, opts storage.Options) (*storage.BucketInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("local: bucket name required")
	}
	name = safeBucketName(name)

	path, err := joinUnderRoot(s.root, name)
	if err != nil {
		return nil, err
	}

	err = os.Mkdir(path, DirPermissions)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, storage.ErrExist
		}
		return nil, fmt.Errorf("local: create bucket %q: %w", name, err)
	}

	now := time.Now()
	return &storage.BucketInfo{
		Name:      name,
		CreatedAt: now,
	}, nil
}

// DeleteBucket deletes the bucket directory.
//
// opts:
//
//	"force": bool  // if true, remove recursively
func (s *store) DeleteBucket(ctx context.Context, name string, opts storage.Options) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("local: bucket name required")
	}
	name = safeBucketName(name)

	path, err := joinUnderRoot(s.root, name)
	if err != nil {
		return err
	}

	force := boolOpt(opts, "force")

	if force {
		if err := os.RemoveAll(path); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return storage.ErrNotExist
			}
			return fmt.Errorf("local: remove bucket %q: %w", name, err)
		}
		return nil
	}

	err = os.Remove(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return storage.ErrNotExist
		}
		// On Unix, removing a non empty dir is ENOTEMPTY; treat as permission style error.
		return fmt.Errorf("local: remove bucket %q: %w", name, storage.ErrPermission)
	}
	return nil
}

// Features returns backend capabilities.
//
// We expose server side move for objects and directories because the local
// filesystem can rename within the same volume without streaming data.
func (s *store) Features() storage.Features {
	return storage.Features{
		"move":               true,
		"directories":        true,
		"object_move_server": true,
		"dir_move_server":    true,
		"multipart":          true,
	}
}

func (s *store) Close() error { return nil }

// bucket implements storage.Bucket on top of a directory.
type bucket struct {
	store *store
	name  string
	root  string // absolute path to bucket root on disk
}

var _ storage.Bucket = (*bucket)(nil)

func (b *bucket) Name() string { return b.name }

func (b *bucket) Info(ctx context.Context) (*storage.BucketInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	info, err := os.Stat(b.root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, storage.ErrNotExist
		}
		return nil, fmt.Errorf("local: stat bucket %q: %w", b.name, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("local: bucket %q root is not a directory", b.name)
	}
	return &storage.BucketInfo{
		Name:      b.name,
		CreatedAt: info.ModTime(),
	}, nil
}

func (b *bucket) Features() storage.Features {
	return b.store.Features()
}

// Write writes the object to the filesystem.
//
// Uses tiered optimization based on file size:
//   - Tiny files (<=4KB): Direct memory write, minimal syscalls
//   - Small files (4KB-64KB): Single-buffer direct write
//   - Large files (>64KB): Temp file + atomic rename for safety
//
// Keys always use "/" separators; filesystem paths use OS separators.
// This mirrors rclone style semantics where remote paths are slash based and
// the local backend handles platform differences.
func (b *bucket) Write(ctx context.Context, key string, src io.Reader, size int64, contentType string, opts storage.Options) (*storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// FAST PATH: In-memory mode bypasses all filesystem operations
	if inMemoryMode.Load() {
		return b.writeInMemory(key, src, size, contentType)
	}

	relKey, err := cleanKey(key)
	if err != nil {
		return nil, err
	}

	full, err := joinUnderRoot(b.root, relKey)
	if err != nil {
		return nil, err
	}
	dir := filepath.Dir(full)

	// Use optimized lock-free directory cache for fast path
	if err := optimizedDirCache.ensureDir(dir); err != nil {
		return nil, fmt.Errorf("local: mkdir %q: %w", dir, err)
	}

	// Select write strategy based on file size
	switch {
	case size > 0 && size <= TinyFileThreshold:
		// Tiny files: direct memory write with minimal overhead
		return b.writeTinyFile(full, relKey, src, size, contentType)
	case size > 0 && size <= SmallFileThreshold:
		// Small files: single-buffer direct write
		return b.writeSmallFile(full, relKey, src, size, contentType)
	case size >= ParallelWriteThreshold:
		// Very large files (>32MB): parallel chunked write for maximum throughput
		return b.writeVeryLargeFile(full, dir, relKey, key, src, size, contentType)
	default:
		// Large/unknown files: temp file + atomic rename
		return b.writeLargeFile(full, dir, relKey, key, src, contentType)
	}
}

// writeTinyFile writes very small files (<=4KB) with minimal syscalls.
// Uses a sharded buffer pool for maximum speed under concurrency.
// OPTIMIZATION: Uses os.WriteFile for single atomic syscall when NoFsync is enabled.
func (b *bucket) writeTinyFile(full, relKey string, src io.Reader, size int64, contentType string) (*storage.Object, error) {
	// Get small buffer from sharded pool (reduces lock contention)
	buf := shardedSmallPool.Get()
	defer shardedSmallPool.Put(buf)

	// Read all data into buffer
	n, err := io.ReadFull(src, buf[:size])
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return nil, fmt.Errorf("local: read %q: %w", relKey, err)
	}

	// FAST PATH: Use os.WriteFile when fsync is not needed (reduces syscalls from 4 to 1)
	if NoFsync {
		// #nosec G306 -- file permissions are intentionally set to 0644
		if err := os.WriteFile(full, buf[:n], FilePermissions); err != nil {
			return nil, fmt.Errorf("local: write %q: %w", relKey, err)
		}
	} else {
		// Write directly to destination with fsync
		// #nosec G304 -- path is validated by cleanKey and joinUnderRoot
		f, err := os.OpenFile(full, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, FilePermissions)
		if err != nil {
			return nil, fmt.Errorf("local: create %q: %w", relKey, err)
		}

		if _, err := f.Write(buf[:n]); err != nil {
			f.Close()
			return nil, fmt.Errorf("local: write %q: %w", relKey, err)
		}

		if err := f.Sync(); err != nil {
			f.Close()
			return nil, fmt.Errorf("local: fsync %q: %w", relKey, err)
		}

		if err := f.Close(); err != nil {
			return nil, fmt.Errorf("local: close %q: %w", relKey, err)
		}
	}

	// OPTIMIZATION: Write-through cache for hot objects
	now := time.Now()
	ck := cacheKey(b.name, relKey)
	WriteThroughCache(ck, buf[:n], now)
	mixedWriteOps.Add(1)

	// Return object without extra stat call - we know the size
	return &storage.Object{
		Bucket:      b.name,
		Key:         relToKey(relKey),
		Size:        int64(n),
		ContentType: contentType,
		Created:     now,
		Updated:     now,
	}, nil
}

// writeSmallFile writes small files (4KB-128KB) directly to the destination.
// Uses optimized buffer selection based on exact file size.
// OPTIMIZATION: Uses os.WriteFile for single atomic syscall when NoFsync is enabled.
func (b *bucket) writeSmallFile(full, relKey string, src io.Reader, size int64, contentType string) (*storage.Object, error) {
	// OPTIMIZATION: Use exact-size buffer pool for 16KB (benchmark object size)
	var buf []byte
	if size == MixedBufferSize {
		buf = shardedMixedPool.Get()
		defer shardedMixedPool.Put(buf)
	} else if size <= TinyFileThreshold {
		buf = shardedSmallPool.Get()
		defer shardedSmallPool.Put(buf)
	} else {
		buf = shardedMediumPool.Get()
		defer shardedMediumPool.Put(buf)
	}

	// Read all data into buffer
	n, err := io.ReadFull(src, buf[:size])
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return nil, fmt.Errorf("local: read %q: %w", relKey, err)
	}

	// FAST PATH: Use os.WriteFile when fsync is not needed (reduces syscalls from 4 to 1)
	if NoFsync {
		// #nosec G306 -- file permissions are intentionally set to 0644
		if err := os.WriteFile(full, buf[:n], FilePermissions); err != nil {
			return nil, fmt.Errorf("local: write %q: %w", relKey, err)
		}
	} else {
		// Write directly to destination with fsync
		// #nosec G304 -- path is validated by cleanKey and joinUnderRoot
		f, err := os.OpenFile(full, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, FilePermissions)
		if err != nil {
			return nil, fmt.Errorf("local: create %q: %w", relKey, err)
		}

		if _, err := f.Write(buf[:n]); err != nil {
			f.Close()
			return nil, fmt.Errorf("local: write %q: %w", relKey, err)
		}

		if err := f.Sync(); err != nil {
			f.Close()
			return nil, fmt.Errorf("local: fsync %q: %w", relKey, err)
		}

		if err := f.Close(); err != nil {
			return nil, fmt.Errorf("local: close %q: %w", relKey, err)
		}
	}

	// OPTIMIZATION: Write-through cache for hot objects
	now := time.Now()
	if int64(n) <= CacheableThreshold {
		ck := cacheKey(b.name, relKey)
		WriteThroughCache(ck, buf[:n], now)
	}
	mixedWriteOps.Add(1)

	// Return object without extra stat call - we know the size
	return &storage.Object{
		Bucket:      b.name,
		Key:         relToKey(relKey),
		Size:        int64(n),
		ContentType: contentType,
		Created:     now,
		Updated:     now,
	}, nil
}

// writeLargeFile writes large or unknown-size files using temp file + atomic rename.
// Optimized: uses sharded buffer pool and tracks bytes to avoid post-write stat.
// For very large files (>32MB), uses parallel write for higher throughput.
func (b *bucket) writeLargeFile(full, dir, relKey, key string, src io.Reader, contentType string) (*storage.Object, error) {
	// Safer temp file: randomly named in the target directory.
	// This avoids predictable names and keeps rename atomic on the same volume.
	tmp, err := os.CreateTemp(dir, TempFilePattern)
	if err != nil {
		return nil, fmt.Errorf("local: create temp file in %q: %w", dir, err)
	}
	tmpName := tmp.Name()
	defer func() {
		_ = tmp.Close()
		// Best effort cleanup; if rename succeeds this will fail harmlessly.
		_ = os.Remove(tmpName)
	}()

	// Use sharded huge buffer pool for optimized I/O under concurrency
	buf := shardedHugePool.Get()
	defer shardedHugePool.Put(buf)

	// Track bytes written to avoid post-write stat syscall
	var written int64
	for {
		nr, readErr := src.Read(buf)
		if nr > 0 {
			nw, writeErr := tmp.Write(buf[:nr])
			written += int64(nw)
			if writeErr != nil {
				return nil, fmt.Errorf("local: write %q: %w", key, writeErr)
			}
			if nr != nw {
				return nil, fmt.Errorf("local: short write %q", key)
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return nil, fmt.Errorf("local: read for %q: %w", key, readErr)
		}
	}

	// Optional fsync for durability (skip for benchmarks)
	if !NoFsync {
		if err := tmp.Sync(); err != nil {
			return nil, fmt.Errorf("local: fsync %q: %w", key, err)
		}
	}
	if err := tmp.Close(); err != nil {
		return nil, fmt.Errorf("local: close temp for %q: %w", key, err)
	}

	if err := os.Rename(tmpName, full); err != nil {
		return nil, fmt.Errorf("local: rename temp to %q: %w", full, err)
	}

	// Return object without stat call - we tracked the bytes written
	now := time.Now()
	return &storage.Object{
		Bucket:      b.name,
		Key:         relToKey(relKey),
		Size:        written,
		ContentType: contentType,
		Created:     now,
		Updated:     now,
	}, nil
}

// writeVeryLargeFile writes very large files (>32MB) using parallel chunked writes.
// This provides significantly higher throughput on modern SSDs.
func (b *bucket) writeVeryLargeFile(full, _, relKey, key string, src io.Reader, size int64, contentType string) (*storage.Object, error) {
	// Create parallel writer with pre-allocation
	pw, err := newParallelWriter(full, size)
	if err != nil {
		return nil, fmt.Errorf("local: create %q: %w", key, err)
	}
	defer pw.Close()

	// Write data in parallel chunks
	written, err := pw.WriteFrom(src)
	if err != nil {
		os.Remove(full) // Cleanup on error
		return nil, fmt.Errorf("local: write %q: %w", key, err)
	}

	// Sync to disk
	if err := pw.Sync(); err != nil {
		os.Remove(full)
		return nil, fmt.Errorf("local: fsync %q: %w", key, err)
	}

	// Return object without stat call - we tracked the bytes written
	now := time.Now()
	return &storage.Object{
		Bucket:      b.name,
		Key:         relToKey(relKey),
		Size:        written,
		ContentType: contentType,
		Created:     now,
		Updated:     now,
	}, nil
}

func (b *bucket) Open(ctx context.Context, key string, offset, length int64, opts storage.Options) (io.ReadCloser, *storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}

	// FAST PATH: In-memory mode bypasses all filesystem operations
	if inMemoryMode.Load() {
		return b.openInMemory(key, offset, length)
	}

	relKey, err := cleanKey(key)
	if err != nil {
		return nil, nil, err
	}

	// OPTIMIZATION 1: Check lock-free hot object cache first (fastest path)
	if offset == 0 && length <= 0 {
		ck := cacheKey(b.name, relKey)
		// Try hot cache first (zero-copy, lock-free)
		if data, modTime, ok := hotCache.GetHot(ck); ok {
			mixedCacheHits.Add(1)
			mixedReadOps.Add(1)
			obj := &storage.Object{
				Bucket:  b.name,
				Key:     relToKey(relKey),
				Size:    int64(len(data)),
				Created: modTime,
				Updated: modTime,
			}
			// Return zero-copy reader (no data copying)
			return &zeroCopyReader{data: data}, obj, nil
		}

		// Try regular cache (with LRU tracking)
		if data, modTime, ok := globalObjectCache.Get(ck); ok {
			mixedCacheHits.Add(1)
			mixedReadOps.Add(1)
			obj := &storage.Object{
				Bucket:  b.name,
				Key:     relToKey(relKey),
				Size:    int64(len(data)),
				Created: modTime,
				Updated: modTime,
			}
			return &cachedReader{data: data}, obj, nil
		}
		mixedCacheMiss.Add(1)
	}

	full, err := joinUnderRoot(b.root, relKey)
	if err != nil {
		return nil, nil, err
	}

	// First, stat the file to check existence and get size
	info, err := os.Stat(full)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil, storage.ErrNotExist
		}
		return nil, nil, fmt.Errorf("local: stat %q: %w", key, err)
	}
	if info.IsDir() {
		return nil, nil, storage.ErrPermission
	}

	fileSize := info.Size()
	modTime := info.ModTime()

	// Build object metadata once (avoid duplicate work)
	obj := &storage.Object{
		Bucket:  b.name,
		Key:     relToKey(relKey),
		Size:    fileSize,
		Created: modTime,
		Updated: modTime,
	}

	// OPTIMIZATION: For very large files (>=32MB), use streaming reader with large buffers
	if fileSize >= ParallelReadThreshold {
		// #nosec G304 -- path is validated by cleanKey and joinUnderRoot
		f, err := os.Open(full)
		if err == nil {
			// Use streaming reader for maximum throughput
			return newStreamingReader(f, fileSize, offset, length), obj, nil
		}
		// Fall through on error
	}

	// OPTIMIZATION: For large files (>=1MB), use optimized large file reader
	if fileSize >= LargeSendfileThreshold {
		// #nosec G304 -- path is validated by cleanKey and joinUnderRoot
		f, err := os.Open(full)
		if err == nil {
			return newLargeFileReader(f, fileSize, offset, length), obj, nil
		}
		// Fall through on error
	}

	// OPTIMIZATION: Use mmap for medium files (64KB-1MB) - eliminates duplicate stat
	if mmapSupported() && fileSize >= MmapThreshold && fileSize < LargeSendfileThreshold {
		// #nosec G304 -- path is validated by cleanKey and joinUnderRoot
		f, err := os.Open(full)
		if err == nil {
			rc, err := openWithMmapPrestatted(f, fileSize, offset, length)
			if err == nil {
				return rc, obj, nil
			}
			f.Close()
		}
		// Fall through to regular file I/O on mmap failure
	}

	// OPTIMIZATION: For small files (full read), read into cache and return cached reader
	if offset == 0 && length <= 0 && fileSize <= CacheableThreshold {
		// #nosec G304 -- path is validated by cleanKey and joinUnderRoot
		f, err := os.Open(full)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil, nil, storage.ErrNotExist
			}
			return nil, nil, fmt.Errorf("local: open %q: %w", key, err)
		}

		// Read entire file into buffer
		data := make([]byte, fileSize)
		n, err := io.ReadFull(f, data)
		f.Close()
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return nil, nil, fmt.Errorf("local: read %q: %w", key, err)
		}
		data = data[:n]

		// OPTIMIZATION: Promote to hot cache for faster subsequent reads
		ck := cacheKey(b.name, relKey)
		WriteThroughCache(ck, data, modTime)
		mixedReadOps.Add(1)

		return &cachedReader{data: data}, obj, nil
	}

	// Regular file I/O for larger files or partial reads
	// #nosec G304 -- path is validated by cleanKey and joinUnderRoot
	f, err := os.Open(full)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil, storage.ErrNotExist
		}
		return nil, nil, fmt.Errorf("local: open %q: %w", key, err)
	}

	if offset > 0 {
		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			_ = f.Close()
			return nil, nil, fmt.Errorf("local: seek %q: %w", key, err)
		}
	}

	var rc io.ReadCloser = f
	if length > 0 {
		rc = struct {
			io.Reader
			io.Closer
		}{
			Reader: io.LimitReader(f, length),
			Closer: f,
		}
	}

	return rc, obj, nil
}

func (b *bucket) Stat(ctx context.Context, key string, opts storage.Options) (*storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// FAST PATH: In-memory mode bypasses all filesystem operations
	if inMemoryMode.Load() {
		return b.statInMemory(key)
	}

	relKey, err := cleanKey(key)
	if err != nil {
		return nil, err
	}

	// OPTIMIZATION: Check hot object cache first to avoid disk I/O
	ck := cacheKey(b.name, relKey)
	if data, modTime, ok := globalObjectCache.Get(ck); ok {
		return &storage.Object{
			Bucket:  b.name,
			Key:     relToKey(relKey),
			Size:    int64(len(data)),
			IsDir:   false,
			Created: modTime,
			Updated: modTime,
		}, nil
	}

	full, err := joinUnderRoot(b.root, relKey)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(full)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, storage.ErrNotExist
		}
		return nil, fmt.Errorf("local: stat %q: %w", key, err)
	}
	return &storage.Object{
		Bucket:  b.name,
		Key:     relToKey(relKey),
		Size:    info.Size(),
		IsDir:   info.IsDir(),
		Created: info.ModTime(),
		Updated: info.ModTime(),
	}, nil
}

func (b *bucket) Delete(ctx context.Context, key string, opts storage.Options) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	// FAST PATH: In-memory mode bypasses all filesystem operations
	if inMemoryMode.Load() {
		return b.deleteInMemory(key)
	}

	relKey, err := cleanKey(key)
	if err != nil {
		return err
	}
	full, err := joinUnderRoot(b.root, relKey)
	if err != nil {
		return err
	}
	recursive := boolOpt(opts, "recursive")

	var delErr error
	if recursive {
		delErr = os.RemoveAll(full)
		// Invalidate all cached objects under this prefix
		globalObjectCache.InvalidatePrefix(cacheKey(b.name, relKey))
	} else {
		delErr = os.Remove(full)
		// Invalidate this specific object from cache
		globalObjectCache.Invalidate(cacheKey(b.name, relKey))
	}
	if delErr != nil {
		if errors.Is(delErr, os.ErrNotExist) {
			return storage.ErrNotExist
		}
		return fmt.Errorf("local: delete %q: %w", key, delErr)
	}
	return nil
}

func (b *bucket) Copy(ctx context.Context, dstKey string, srcBucket, srcKey string, opts storage.Options) (*storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	srcRel, err := cleanKey(srcKey)
	if err != nil {
		return nil, err
	}
	dstRel, err := cleanKey(dstKey)
	if err != nil {
		return nil, err
	}

	srcBucketName := safeBucketName(strings.TrimSpace(srcBucket))
	srcRoot, err := joinUnderRoot(b.store.root, srcBucketName)
	if err != nil {
		return nil, err
	}

	srcFull, err := joinUnderRoot(srcRoot, srcRel)
	if err != nil {
		return nil, err
	}
	dstFull, err := joinUnderRoot(b.root, dstRel)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(dstFull), DirPermissions); err != nil {
		return nil, fmt.Errorf("local: mkdir dst dir: %w", err)
	}

	if err := copyFile(srcFull, dstFull); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, storage.ErrNotExist
		}
		return nil, fmt.Errorf("local: copy %q -> %q: %w", srcKey, dstKey, err)
	}

	info, err := os.Stat(dstFull)
	if err != nil {
		return nil, fmt.Errorf("local: stat dst %q: %w", dstKey, err)
	}
	return &storage.Object{
		Bucket:  b.name,
		Key:     relToKey(dstRel),
		Size:    info.Size(),
		Created: info.ModTime(),
		Updated: info.ModTime(),
	}, nil
}

func (b *bucket) Move(ctx context.Context, dstKey string, srcBucket, srcKey string, opts storage.Options) (*storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	srcRel, err := cleanKey(srcKey)
	if err != nil {
		return nil, err
	}
	dstRel, err := cleanKey(dstKey)
	if err != nil {
		return nil, err
	}

	srcBucketName := safeBucketName(strings.TrimSpace(srcBucket))
	srcRoot, err := joinUnderRoot(b.store.root, srcBucketName)
	if err != nil {
		return nil, err
	}
	srcFull, err := joinUnderRoot(srcRoot, srcRel)
	if err != nil {
		return nil, err
	}
	dstFull, err := joinUnderRoot(b.root, dstRel)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(dstFull), DirPermissions); err != nil {
		return nil, fmt.Errorf("local: mkdir dst dir: %w", err)
	}

	// Try atomic server side rename first.
	if err := os.Rename(srcFull, dstFull); err != nil {
		// Fallback to copy plus delete.
		if err := copyFile(srcFull, dstFull); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil, storage.ErrNotExist
			}
			return nil, fmt.Errorf("local: copy for move %q -> %q: %w", srcKey, dstKey, err)
		}
		_ = os.Remove(srcFull)
	}

	info, err := os.Stat(dstFull)
	if err != nil {
		return nil, fmt.Errorf("local: stat dst %q: %w", dstKey, err)
	}
	return &storage.Object{
		Bucket:  b.name,
		Key:     relToKey(dstRel),
		Size:    info.Size(),
		Created: info.ModTime(),
		Updated: info.ModTime(),
	}, nil
}

func (b *bucket) List(ctx context.Context, prefix string, limit, offset int, opts storage.Options) (storage.ObjectIter, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Default to recursive listing to match S3-like behavior where all keys with prefix are returned
	recursive := true
	if opts != nil {
		if v, ok := opts["recursive"]; ok {
			if b, ok := v.(bool); ok {
				recursive = b
			}
		}
	}
	dirsOnly := boolOpt(opts, "dirs_only")
	filesOnly := boolOpt(opts, "files_only")
	if dirsOnly && filesOnly {
		dirsOnly, filesOnly = false, false
	}

	relPrefix, err := cleanPrefix(prefix)
	if err != nil {
		return nil, err
	}
	base, err := joinUnderRoot(b.root, relPrefix)
	if err != nil {
		return nil, err
	}

	var objects []*storage.Object

	if recursive {
		err = filepath.WalkDir(base, func(p string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return nil
			}
			if p == base {
				return nil
			}
			relPath, err := filepath.Rel(b.root, p)
			if err != nil {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return nil
			}
			obj := &storage.Object{
				Bucket:  b.name,
				Key:     relToKey(filepath.ToSlash(relPath)),
				Size:    info.Size(),
				IsDir:   info.IsDir(),
				Created: info.ModTime(),
				Updated: info.ModTime(),
			}
			if dirsOnly && !obj.IsDir {
				return nil
			}
			if filesOnly && obj.IsDir {
				return nil
			}
			objects = append(objects, obj)
			return nil
		})
		if err != nil && !errors.Is(err, context.Canceled) {
			return nil, fmt.Errorf("local: walk %q: %w", prefix, err)
		}
	} else {
		entries, err := os.ReadDir(base)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return &objectIter{}, nil
			}
			return nil, fmt.Errorf("local: list %q: %w", prefix, err)
		}
		for _, e := range entries {
			info, err := e.Info()
			if err != nil {
				continue
			}
			relPath := filepath.Join(relPrefix, e.Name())
			obj := &storage.Object{
				Bucket:  b.name,
				Key:     relToKey(filepath.ToSlash(relPath)),
				Size:    info.Size(),
				IsDir:   info.IsDir(),
				Created: info.ModTime(),
				Updated: info.ModTime(),
			}
			if dirsOnly && !obj.IsDir {
				continue
			}
			if filesOnly && obj.IsDir {
				continue
			}
			objects = append(objects, obj)
		}
	}

	sort.Slice(objects, func(i, j int) bool { return objects[i].Key < objects[j].Key })

	if offset < 0 {
		offset = 0
	}
	if offset > len(objects) {
		offset = len(objects)
	}
	objects = objects[offset:]
	if limit > 0 && limit < len(objects) {
		objects = objects[:limit]
	}

	return &objectIter{list: objects}, nil
}

func (b *bucket) SignedURL(ctx context.Context, key string, method string, expires time.Duration, opts storage.Options) (string, error) {
	// Local backend does not expose HTTP URLs.
	return "", storage.ErrUnsupported
}

// bucketIter implements storage.BucketIter.
type bucketIter struct {
	list []*storage.BucketInfo
	pos  int
}

func (it *bucketIter) Next() (*storage.BucketInfo, error) {
	if it.pos >= len(it.list) {
		return nil, nil
	}
	b := it.list[it.pos]
	it.pos++
	return b, nil
}

func (it *bucketIter) Close() error { return nil }

// objectIter implements storage.ObjectIter.
type objectIter struct {
	list []*storage.Object
	pos  int
}

func (it *objectIter) Next() (*storage.Object, error) {
	if it.pos >= len(it.list) {
		return nil, nil
	}
	o := it.list[it.pos]
	it.pos++
	return o, nil
}

func (it *objectIter) Close() error { return nil }

// Helpers

func boolOpt(opts storage.Options, key string) bool {
	if opts == nil {
		return false
	}
	v, ok := opts[key]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	if !ok {
		return false
	}
	return b
}

// safeBucketName strips separators and special cases.
func safeBucketName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, string(os.PathSeparator), "_")
	if name == "" {
		return "default"
	}
	if name == "." || name == ".." {
		return "_" + name
	}
	return name
}

// cleanKey normalizes an object key into a relative slash separated path.
// It uses path.Clean so it is platform independent and forbids ".." segments.
func cleanKey(key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", errors.New("local: empty key")
	}
	// Normalize backslashes to slash first so users can pass Windows style keys.
	key = strings.ReplaceAll(key, "\\", "/")
	key = strings.TrimPrefix(key, "/")
	if key == "" {
		return "", errors.New("local: empty key")
	}

	key = path.Clean(key)
	if key == "." {
		return "", errors.New("local: empty key")
	}

	for _, part := range strings.Split(key, "/") {
		if part == ".." {
			return "", storage.ErrPermission
		}
	}
	return key, nil
}

// cleanPrefix is like cleanKey but allows empty result.
func cleanPrefix(prefix string) (string, error) {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return "", nil
	}
	prefix = strings.ReplaceAll(prefix, "\\", "/")
	prefix = strings.TrimPrefix(prefix, "/")
	if prefix == "" {
		return "", nil
	}
	prefix = path.Clean(prefix)
	if prefix == "." {
		return "", nil
	}
	for _, part := range strings.Split(prefix, "/") {
		if part == ".." {
			return "", storage.ErrPermission
		}
	}
	return prefix, nil
}

// joinUnderRoot joins root and rel (slash separated) to an absolute path,
// cleans it and verifies it does not escape the root directory.
func joinUnderRoot(root, rel string) (string, error) {
	rootClean := filepath.Clean(root)

	if rel == "" {
		return rootClean, nil
	}

	// Convert the logical slash separated path into OS form.
	relPath := filepath.FromSlash(rel)
	// Trim leading separators to keep it relative.
	relPath = strings.TrimLeft(relPath, string(os.PathSeparator))

	joined := filepath.Join(rootClean, relPath)
	joined = filepath.Clean(joined)

	relative, err := filepath.Rel(rootClean, joined)
	if err != nil {
		return "", fmt.Errorf("local: rel path error: %w", err)
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return "", storage.ErrPermission
	}
	return joined, nil
}

// relToKey converts a filesystem relative path back to a slash separated key.
func relToKey(rel string) string {
	rel = filepath.ToSlash(rel)
	rel = strings.TrimPrefix(rel, "/")
	return rel
}

func copyFile(src, dst string) (err error) {
	// #nosec G304 -- internal function with validated paths
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		_ = in.Close()
	}()

	// #nosec G304 -- internal function with validated paths
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, FilePermissions)
	if err != nil {
		return err
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
		if err != nil {
			_ = os.Remove(dst)
		}
	}()

	// Use sharded large buffer pool for optimized I/O under concurrency
	buf := shardedLargePool.Get()
	defer shardedLargePool.Put(buf)

	if _, err = io.CopyBuffer(out, in, buf); err != nil {
		return err
	}
	// Optional fsync for durability (skip for benchmarks)
	if NoFsync {
		return nil
	}
	return out.Sync()
}
