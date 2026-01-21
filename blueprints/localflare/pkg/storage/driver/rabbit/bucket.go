package rabbit

import (
	"context"
	"errors"
	"fmt"
	"io"
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

// Bucket implements storage.Bucket with high-performance operations.
type Bucket struct {
	store *Store
	name  string
	root  string

	// Caches (shared with store)
	hotCache  *HotCache
	warmCache *WarmCache

	// Object map (sync.Map for lock-free access)
	objects sync.Map // key -> *ObjectEntry

	// Sharded key index for LIST operations
	keyShards [NumShards]*keyShard
	keyCount  atomic.Int64

	// Multipart uploads
	mpMu      sync.RWMutex
	mpUploads map[string]*multipartUpload
}

var _ storage.Bucket = (*Bucket)(nil)

// ObjectEntry represents an object in the bucket.
type ObjectEntry struct {
	key         string
	data        []byte // nil if spilled to disk
	size        int64
	contentType string
	created     int64 // Unix nano
	updated     int64 // Unix nano
	diskPath    string
}

func (e *ObjectEntry) toObject(bucket string) *storage.Object {
	return &storage.Object{
		Bucket:      bucket,
		Key:         e.key,
		Size:        e.size,
		ContentType: e.contentType,
		Created:     time.Unix(0, e.created),
		Updated:     time.Unix(0, e.updated),
	}
}

// keyShard maintains sorted keys for a shard.
type keyShard struct {
	mu   sync.RWMutex
	keys []string
}

// multipartUpload tracks an in-progress multipart upload.
type multipartUpload struct {
	id       string
	key      string
	parts    map[int]*uploadPart
	created  time.Time
	tmpDir   string
	metadata map[string]string
}

type uploadPart struct {
	number int
	path   string
	size   int64
	etag   string
}

// Name returns the bucket name.
func (b *Bucket) Name() string { return b.name }

// Features returns bucket capabilities.
func (b *Bucket) Features() storage.Features {
	return b.store.Features()
}

// Info returns bucket information.
func (b *Bucket) Info(ctx context.Context) (*storage.BucketInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	info, err := os.Stat(b.root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, storage.ErrNotExist
		}
		return nil, fmt.Errorf("rabbit: stat bucket %q: %w", b.name, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("rabbit: bucket %q is not a directory", b.name)
	}

	return &storage.BucketInfo{
		Name:      b.name,
		CreatedAt: info.ModTime(),
	}, nil
}

// Write writes an object to the bucket.
func (b *Bucket) Write(ctx context.Context, key string, src io.Reader, size int64, contentType string, opts storage.Options) (*storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Fast path: skip validation for clean keys
	if key == "" {
		return nil, errors.New("rabbit: empty key")
	}
	if key[0] == ' ' || key[len(key)-1] == ' ' {
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, errors.New("rabbit: empty key")
		}
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

	// Ensure directory exists with caching
	if !globalDirCache.Check(dir) {
		if err := os.MkdirAll(dir, DirPermissions); err != nil {
			return nil, fmt.Errorf("rabbit: mkdir %q: %w", dir, err)
		}
		globalDirCache.Add(dir)
	}

	// Size-based routing
	var obj *storage.Object
	switch {
	case size == 0:
		obj, err = b.writeEmpty(full, relKey, contentType)
	case size > 0 && size <= TinyThreshold:
		obj, err = b.writeTiny(full, relKey, src, size, contentType)
	case size > 0 && size <= SmallThreshold:
		obj, err = b.writeSmall(full, relKey, src, size, contentType)
	default:
		obj, err = b.writeLarge(full, dir, relKey, src, contentType)
	}

	if err != nil {
		return nil, err
	}

	// Update key index
	b.addKey(relKey)

	return obj, nil
}

// writeEmpty creates an empty file.
func (b *Bucket) writeEmpty(full, relKey, contentType string) (*storage.Object, error) {
	if NoFsync {
		if err := os.WriteFile(full, nil, FilePermissions); err != nil {
			return nil, fmt.Errorf("rabbit: write %q: %w", relKey, err)
		}
	} else {
		f, err := os.OpenFile(full, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, FilePermissions)
		if err != nil {
			return nil, fmt.Errorf("rabbit: create %q: %w", relKey, err)
		}
		if err := f.Sync(); err != nil {
			f.Close()
			return nil, fmt.Errorf("rabbit: fsync %q: %w", relKey, err)
		}
		f.Close()
	}

	now := fastNow()
	return &storage.Object{
		Bucket:      b.name,
		Key:         relToKey(relKey),
		Size:        0,
		ContentType: contentType,
		Created:     now,
		Updated:     now,
	}, nil
}

// writeTiny writes very small files (<=8KB) with minimal syscalls.
func (b *Bucket) writeTiny(full, relKey string, src io.Reader, size int64, contentType string) (*storage.Object, error) {
	buf := tinyPool.Get()
	defer tinyPool.Put(buf)

	n, err := io.ReadFull(src, buf[:size])
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return nil, fmt.Errorf("rabbit: read %q: %w", relKey, err)
	}

	if NoFsync {
		if err := os.WriteFile(full, buf[:n], FilePermissions); err != nil {
			return nil, fmt.Errorf("rabbit: write %q: %w", relKey, err)
		}
	} else {
		f, err := os.OpenFile(full, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, FilePermissions)
		if err != nil {
			return nil, fmt.Errorf("rabbit: create %q: %w", relKey, err)
		}
		if _, err := f.Write(buf[:n]); err != nil {
			f.Close()
			return nil, fmt.Errorf("rabbit: write %q: %w", relKey, err)
		}
		if err := f.Sync(); err != nil {
			f.Close()
			return nil, fmt.Errorf("rabbit: fsync %q: %w", relKey, err)
		}
		f.Close()
	}

	now := fastNow()
	ck := cacheKey(b.name, relKey)
	b.hotCache.Put(ck, buf[:n], now)
	b.warmCache.Put(ck, buf[:n], now)

	return &storage.Object{
		Bucket:      b.name,
		Key:         relToKey(relKey),
		Size:        int64(n),
		ContentType: contentType,
		Created:     now,
		Updated:     now,
	}, nil
}

// writeSmall writes small files (8KB-128KB).
func (b *Bucket) writeSmall(full, relKey string, src io.Reader, size int64, contentType string) (*storage.Object, error) {
	buf := getBuffer(size)
	defer putBuffer(buf)

	n, err := io.ReadFull(src, buf[:size])
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return nil, fmt.Errorf("rabbit: read %q: %w", relKey, err)
	}

	if NoFsync {
		if err := os.WriteFile(full, buf[:n], FilePermissions); err != nil {
			return nil, fmt.Errorf("rabbit: write %q: %w", relKey, err)
		}
	} else {
		f, err := os.OpenFile(full, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, FilePermissions)
		if err != nil {
			return nil, fmt.Errorf("rabbit: create %q: %w", relKey, err)
		}
		if _, err := f.Write(buf[:n]); err != nil {
			f.Close()
			return nil, fmt.Errorf("rabbit: write %q: %w", relKey, err)
		}
		if err := f.Sync(); err != nil {
			f.Close()
			return nil, fmt.Errorf("rabbit: fsync %q: %w", relKey, err)
		}
		f.Close()
	}

	now := fastNow()
	ck := cacheKey(b.name, relKey)
	b.warmCache.Put(ck, buf[:n], now)

	return &storage.Object{
		Bucket:      b.name,
		Key:         relToKey(relKey),
		Size:        int64(n),
		ContentType: contentType,
		Created:     now,
		Updated:     now,
	}, nil
}

// writeLarge writes large files using temp file + atomic rename.
func (b *Bucket) writeLarge(full, dir, relKey string, src io.Reader, contentType string) (*storage.Object, error) {
	tmp, err := os.CreateTemp(dir, TempFilePattern)
	if err != nil {
		return nil, fmt.Errorf("rabbit: create temp: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		tmp.Close()
		os.Remove(tmpName)
	}()

	buf := hugePool.Get()
	defer hugePool.Put(buf)

	var written int64
	for {
		nr, readErr := src.Read(buf)
		if nr > 0 {
			nw, writeErr := tmp.Write(buf[:nr])
			written += int64(nw)
			if writeErr != nil {
				return nil, fmt.Errorf("rabbit: write %q: %w", relKey, writeErr)
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return nil, fmt.Errorf("rabbit: read %q: %w", relKey, readErr)
		}
	}

	if !NoFsync {
		if err := tmp.Sync(); err != nil {
			return nil, fmt.Errorf("rabbit: fsync %q: %w", relKey, err)
		}
	}
	tmp.Close()

	if err := os.Rename(tmpName, full); err != nil {
		return nil, fmt.Errorf("rabbit: rename %q: %w", relKey, err)
	}

	now := fastNow()
	return &storage.Object{
		Bucket:      b.name,
		Key:         relToKey(relKey),
		Size:        written,
		ContentType: contentType,
		Created:     now,
		Updated:     now,
	}, nil
}

// Open reads an object.
func (b *Bucket) Open(ctx context.Context, key string, offset, length int64, opts storage.Options) (io.ReadCloser, *storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}

	relKey, err := cleanKey(key)
	if err != nil {
		return nil, nil, err
	}

	ck := cacheKey(b.name, relKey)

	// L1: Hot cache (lock-free)
	if offset == 0 && length <= 0 {
		if data, modTime, ok := b.hotCache.GetHot(ck); ok {
			obj := &storage.Object{
				Bucket:  b.name,
				Key:     relToKey(relKey),
				Size:    int64(len(data)),
				Created: modTime,
				Updated: modTime,
			}
			return &zeroCopyReader{data: data}, obj, nil
		}

		// L2: Warm cache
		if data, modTime, ok := b.warmCache.Get(ck); ok {
			b.hotCache.Put(ck, data, modTime)
			obj := &storage.Object{
				Bucket:  b.name,
				Key:     relToKey(relKey),
				Size:    int64(len(data)),
				Created: modTime,
				Updated: modTime,
			}
			return &cachedReader{data: data}, obj, nil
		}
	}

	// Disk read
	full, err := joinUnderRoot(b.root, relKey)
	if err != nil {
		return nil, nil, err
	}

	info, err := os.Stat(full)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil, storage.ErrNotExist
		}
		return nil, nil, fmt.Errorf("rabbit: stat %q: %w", key, err)
	}
	if info.IsDir() {
		return nil, nil, storage.ErrPermission
	}

	fileSize := info.Size()
	modTime := info.ModTime()

	obj := &storage.Object{
		Bucket:  b.name,
		Key:     relToKey(relKey),
		Size:    fileSize,
		Created: modTime,
		Updated: modTime,
	}

	// Cache small files
	if offset == 0 && length <= 0 && fileSize <= SmallThreshold {
		f, err := os.Open(full)
		if err != nil {
			return nil, nil, fmt.Errorf("rabbit: open %q: %w", key, err)
		}
		data := make([]byte, fileSize)
		n, err := io.ReadFull(f, data)
		f.Close()
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return nil, nil, fmt.Errorf("rabbit: read %q: %w", key, err)
		}
		data = data[:n]

		b.warmCache.Put(ck, data, modTime)
		if n <= TinyThreshold {
			b.hotCache.Put(ck, data, modTime)
		}

		return &cachedReader{data: data}, obj, nil
	}

	// Regular file I/O
	f, err := os.Open(full)
	if err != nil {
		return nil, nil, fmt.Errorf("rabbit: open %q: %w", key, err)
	}

	if offset > 0 {
		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			f.Close()
			return nil, nil, fmt.Errorf("rabbit: seek %q: %w", key, err)
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

// Stat returns object metadata.
func (b *Bucket) Stat(ctx context.Context, key string, opts storage.Options) (*storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	relKey, err := cleanKey(key)
	if err != nil {
		return nil, err
	}

	// Check warm cache for metadata
	ck := cacheKey(b.name, relKey)
	if data, modTime, ok := b.warmCache.Get(ck); ok {
		return &storage.Object{
			Bucket:  b.name,
			Key:     relToKey(relKey),
			Size:    int64(len(data)),
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
		return nil, fmt.Errorf("rabbit: stat %q: %w", key, err)
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

// Delete removes an object.
func (b *Bucket) Delete(ctx context.Context, key string, opts storage.Options) error {
	if err := ctx.Err(); err != nil {
		return err
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
		b.warmCache.InvalidatePrefix(cacheKey(b.name, relKey))
	} else {
		delErr = os.Remove(full)
		ck := cacheKey(b.name, relKey)
		b.hotCache.Invalidate(ck)
		b.warmCache.Invalidate(ck)
	}

	if delErr != nil {
		if errors.Is(delErr, os.ErrNotExist) {
			return storage.ErrNotExist
		}
		return fmt.Errorf("rabbit: delete %q: %w", key, delErr)
	}

	b.removeKey(relKey)
	return nil
}

// Copy copies an object.
func (b *Bucket) Copy(ctx context.Context, dstKey string, srcBucket, srcKey string, opts storage.Options) (*storage.Object, error) {
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
	srcRoot := filepath.Join(b.store.root, srcBucketName)

	srcFull, err := joinUnderRoot(srcRoot, srcRel)
	if err != nil {
		return nil, err
	}
	dstFull, err := joinUnderRoot(b.root, dstRel)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(dstFull), DirPermissions); err != nil {
		return nil, fmt.Errorf("rabbit: mkdir: %w", err)
	}

	if err := copyFile(srcFull, dstFull); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, storage.ErrNotExist
		}
		return nil, fmt.Errorf("rabbit: copy: %w", err)
	}

	b.addKey(dstRel)

	info, err := os.Stat(dstFull)
	if err != nil {
		return nil, fmt.Errorf("rabbit: stat dst: %w", err)
	}

	return &storage.Object{
		Bucket:  b.name,
		Key:     relToKey(dstRel),
		Size:    info.Size(),
		Created: info.ModTime(),
		Updated: info.ModTime(),
	}, nil
}

// Move moves an object.
func (b *Bucket) Move(ctx context.Context, dstKey string, srcBucket, srcKey string, opts storage.Options) (*storage.Object, error) {
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
	srcRoot := filepath.Join(b.store.root, srcBucketName)

	srcFull, err := joinUnderRoot(srcRoot, srcRel)
	if err != nil {
		return nil, err
	}
	dstFull, err := joinUnderRoot(b.root, dstRel)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(dstFull), DirPermissions); err != nil {
		return nil, fmt.Errorf("rabbit: mkdir: %w", err)
	}

	// Try atomic rename
	if err := os.Rename(srcFull, dstFull); err != nil {
		// Fallback to copy + delete
		if err := copyFile(srcFull, dstFull); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil, storage.ErrNotExist
			}
			return nil, fmt.Errorf("rabbit: copy for move: %w", err)
		}
		os.Remove(srcFull)
	}

	// Update caches
	srcCK := cacheKey(srcBucketName, srcRel)
	b.hotCache.Invalidate(srcCK)
	b.warmCache.Invalidate(srcCK)
	b.addKey(dstRel)

	info, err := os.Stat(dstFull)
	if err != nil {
		return nil, fmt.Errorf("rabbit: stat dst: %w", err)
	}

	return &storage.Object{
		Bucket:  b.name,
		Key:     relToKey(dstRel),
		Size:    info.Size(),
		Created: info.ModTime(),
		Updated: info.ModTime(),
	}, nil
}

// List lists objects with prefix.
func (b *Bucket) List(ctx context.Context, prefix string, limit, offset int, opts storage.Options) (storage.ObjectIter, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	recursive := true
	if v, ok := opts["recursive"].(bool); ok {
		recursive = v
	}

	relPrefix, err := cleanPrefix(prefix)
	if err != nil {
		return nil, err
	}

	var objects []*storage.Object
	if recursive {
		objects, err = walkDir(b.root, relPrefix, b.name)
	} else {
		objects, err = listDir(b.root, relPrefix, b.name)
	}
	if err != nil {
		return nil, fmt.Errorf("rabbit: list %q: %w", prefix, err)
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

// SignedURL is not supported for rabbit backend.
func (b *Bucket) SignedURL(ctx context.Context, key string, method string, expires time.Duration, opts storage.Options) (string, error) {
	return "", storage.ErrUnsupported
}

// =============================================================================
// KEY INDEX
// =============================================================================

func (b *Bucket) addKey(key string) {
	shard := b.keyShards[fnv1a(key)%NumShards]
	shard.mu.Lock()
	idx := sort.SearchStrings(shard.keys, key)
	if idx >= len(shard.keys) || shard.keys[idx] != key {
		shard.keys = append(shard.keys, "")
		copy(shard.keys[idx+1:], shard.keys[idx:])
		shard.keys[idx] = key
		b.keyCount.Add(1)
	}
	shard.mu.Unlock()
}

func (b *Bucket) removeKey(key string) {
	shard := b.keyShards[fnv1a(key)%NumShards]
	shard.mu.Lock()
	idx := sort.SearchStrings(shard.keys, key)
	if idx < len(shard.keys) && shard.keys[idx] == key {
		shard.keys = append(shard.keys[:idx], shard.keys[idx+1:]...)
		b.keyCount.Add(-1)
	}
	shard.mu.Unlock()
}

// =============================================================================
// ITERATOR
// =============================================================================

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

// =============================================================================
// HELPERS
// =============================================================================

func cleanKey(key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", errors.New("rabbit: empty key")
	}
	key = strings.ReplaceAll(key, "\\", "/")
	key = strings.TrimPrefix(key, "/")
	if key == "" {
		return "", errors.New("rabbit: empty key")
	}
	key = path.Clean(key)
	if key == "." {
		return "", errors.New("rabbit: empty key")
	}
	for _, part := range strings.Split(key, "/") {
		if part == ".." {
			return "", storage.ErrPermission
		}
	}
	return key, nil
}

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

func joinUnderRoot(root, rel string) (string, error) {
	rootClean := filepath.Clean(root)
	if rel == "" {
		return rootClean, nil
	}
	relPath := filepath.FromSlash(rel)
	relPath = strings.TrimLeft(relPath, string(os.PathSeparator))
	joined := filepath.Join(rootClean, relPath)
	joined = filepath.Clean(joined)

	relative, err := filepath.Rel(rootClean, joined)
	if err != nil {
		return "", fmt.Errorf("rabbit: rel path: %w", err)
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return "", storage.ErrPermission
	}
	return joined, nil
}

func relToKey(rel string) string {
	rel = filepath.ToSlash(rel)
	rel = strings.TrimPrefix(rel, "/")
	return rel
}

func cacheKey(bucket, key string) string {
	return bucket + "\x00" + key
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

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
			os.Remove(dst)
		}
	}()

	buf := largePool.Get()
	defer largePool.Put(buf)

	if _, err = io.CopyBuffer(out, in, buf); err != nil {
		return err
	}
	if NoFsync {
		return nil
	}
	return out.Sync()
}

func walkDir(root, prefix, bucketName string) ([]*storage.Object, error) {
	baseDir := root
	if prefix != "" {
		baseDir = filepath.Join(root, filepath.FromSlash(prefix))
	}

	var objects []*storage.Object
	err := filepath.WalkDir(baseDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return err
		}
		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		key := relToKey(rel)

		info, err := d.Info()
		if err != nil {
			return nil
		}

		objects = append(objects, &storage.Object{
			Bucket:  bucketName,
			Key:     key,
			Size:    info.Size(),
			Created: info.ModTime(),
			Updated: info.ModTime(),
		})
		return nil
	})
	return objects, err
}

func listDir(root, prefix, bucketName string) ([]*storage.Object, error) {
	baseDir := root
	if prefix != "" {
		baseDir = filepath.Join(root, filepath.FromSlash(prefix))
	}

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var objects []*storage.Object
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}

		key := prefix
		if key != "" {
			key += "/"
		}
		key += e.Name()

		objects = append(objects, &storage.Object{
			Bucket:  bucketName,
			Key:     key,
			Size:    info.Size(),
			IsDir:   e.IsDir(),
			Created: info.ModTime(),
			Updated: info.ModTime(),
		})
	}
	return objects, nil
}
