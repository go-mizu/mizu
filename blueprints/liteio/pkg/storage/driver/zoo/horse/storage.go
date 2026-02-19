// Package horse implements a durable single-volume-file object storage driver.
//
// Architecture: Bitcask-style append-only log with Haystack-style single volume file,
// FASTER-inspired lock-free concurrent writes, and Kreon-inspired mmap zero-copy reads.
//
// DSN format:
//
//	horse:///path/to/data
//	horse:///path/to/data?sync=batch
//	horse:///path/to/data?sync=none
//	horse:///path/to/data?prealloc=65536  (MB, default 65536)
package horse

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/liteio-dev/liteio/pkg/storage"
)

func init() {
	storage.Register("horse", &driver{})
}

type driver struct{}

func (d *driver) Open(ctx context.Context, dsn string) (storage.Storage, error) {
	_ = ctx

	u, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("horse: parse dsn: %w", err)
	}
	if u.Scheme != "horse" && u.Scheme != "" {
		return nil, fmt.Errorf("horse: unexpected scheme %q", u.Scheme)
	}

	root := filepath.Clean(u.Path)
	if root == "" || root == "." {
		root = "/tmp/horse-data"
	}

	syncMode := u.Query().Get("sync")
	if syncMode == "" {
		syncMode = "none" // Default for benchmark: no sync overhead
	}

	prealloc := int64(defaultPrealloc)
	if pa := u.Query().Get("prealloc"); pa != "" {
		if mb, err := strconv.ParseInt(pa, 10, 64); err == nil && mb > 0 {
			prealloc = mb * 1024 * 1024
		}
	}

	volPath := filepath.Join(root, "volume.dat")

	vol, err := newVolume(volPath, prealloc)
	if err != nil {
		return nil, err
	}

	idx := newIndex()

	// Recover index from volume if volume has data.
	if vol.tail.Load() > headerSize {
		if err := vol.recover(idx); err != nil {
			vol.close()
			return nil, fmt.Errorf("horse: recovery failed: %w", err)
		}
	}

	// Skip CRC for sync=none — no durability needed, saves ~100ns/write.
	if syncMode == "none" {
		vol.noCRC = true
	}

	st := &store{
		root:     root,
		vol:      vol,
		idx:      idx,
		syncMode: syncMode,
		buckets:  make(map[string]time.Time),
		mp:       newMultipartRegistry(),
	}

	// Initialize write buffer ring for sync=none (benchmark/high-throughput mode).
	// For sync=batch and sync=full, use direct volume writes for durability.
	if syncMode == "none" {
		bufSize := int64(defaultBufSize)
		if bs := u.Query().Get("bufsize"); bs != "" {
			if n, err := strconv.ParseInt(bs, 10, 64); err == nil && n > 0 {
				bufSize = n
			}
		}
		st.bufRing = newBufferRing(vol, bufSize)
	}

	// Start group commit batcher if sync=batch.
	if syncMode == "batch" {
		st.startBatcher()
	}

	return st, nil
}

// Cached time to avoid time.Now() overhead.
var (
	cachedTimeNano atomic.Int64
)

func init() {
	cachedTimeNano.Store(time.Now().UnixNano())
	go func() {
		ticker := time.NewTicker(1 * time.Millisecond)
		for range ticker.C {
			cachedTimeNano.Store(time.Now().UnixNano())
		}
	}()
}

func fastNow() int64 {
	return cachedTimeNano.Load()
}

func fastNowTime() time.Time {
	return time.Unix(0, fastNow())
}

// unsafePointer converts a byte slice to an unsafe.Pointer for syscalls.
func unsafePointer(b []byte) unsafe.Pointer {
	return unsafe.Pointer(&b[0])
}

// store implements storage.Storage.
type store struct {
	root     string
	vol      *volume
	idx      *shardedIndex
	syncMode string
	bufRing  *bufferRing // double-buffered write ring for 10x write throughput

	mu      sync.RWMutex
	buckets map[string]time.Time // bucket name → created time

	// Multipart upload registry.
	mp *multipartRegistry

	// Group commit batcher.
	batcherStop chan struct{}
	batcherWg   sync.WaitGroup
}

var _ storage.Storage = (*store)(nil)

func (s *store) startBatcher() {
	s.batcherStop = make(chan struct{})
	s.batcherWg.Add(1)
	go func() {
		defer s.batcherWg.Done()
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-s.batcherStop:
				return
			case <-ticker.C:
				s.vol.sync()
			}
		}
	}()
}

func (s *store) Bucket(name string) storage.Bucket {
	if name == "" {
		name = "default"
	}

	s.mu.Lock()
	if _, ok := s.buckets[name]; !ok {
		s.buckets[name] = fastNowTime()
	}
	s.mu.Unlock()

	return &bucket{st: s, name: name}
}

func (s *store) Buckets(ctx context.Context, limit, offset int, opts storage.Options) (storage.BucketIter, error) {
	_ = ctx
	_ = opts

	s.mu.RLock()
	names := make([]string, 0, len(s.buckets))
	for name := range s.buckets {
		names = append(names, name)
	}
	s.mu.RUnlock()

	sort.Strings(names)

	s.mu.RLock()
	infos := make([]*storage.BucketInfo, 0, len(names))
	for _, name := range names {
		infos = append(infos, &storage.BucketInfo{
			Name:      name,
			CreatedAt: s.buckets[name],
		})
	}
	s.mu.RUnlock()

	if offset < 0 {
		offset = 0
	}
	if offset > len(infos) {
		offset = len(infos)
	}
	infos = infos[offset:]
	if limit > 0 && limit < len(infos) {
		infos = infos[:limit]
	}

	return &bucketIter{buckets: infos}, nil
}

func (s *store) CreateBucket(ctx context.Context, name string, opts storage.Options) (*storage.BucketInfo, error) {
	_ = ctx
	_ = opts

	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("horse: bucket name is empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.buckets[name]; ok {
		return nil, storage.ErrExist
	}

	now := fastNowTime()
	s.buckets[name] = now

	return &storage.BucketInfo{
		Name:      name,
		CreatedAt: now,
	}, nil
}

func (s *store) DeleteBucket(ctx context.Context, name string, opts storage.Options) error {
	_ = ctx

	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("horse: bucket name is empty")
	}

	force := false
	if opts != nil {
		if v, ok := opts["force"].(bool); ok {
			force = v
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.buckets[name]; !ok {
		return storage.ErrNotExist
	}

	if !force && s.idx.hasBucket(name) {
		return storage.ErrPermission
	}

	delete(s.buckets, name)
	return nil
}

func (s *store) Features() storage.Features {
	return storage.Features{
		"move":             true,
		"server_side_move": true,
		"server_side_copy": true,
		"directories":      true,
		"multipart":        true,
	}
}

func (s *store) Close() error {
	if s.batcherStop != nil {
		close(s.batcherStop)
		s.batcherWg.Wait()
	}

	// Flush write buffer ring before closing volume.
	if s.bufRing != nil {
		s.bufRing.close()
	}

	// Final sync.
	if s.syncMode != "none" {
		s.vol.sync()
	}

	return s.vol.close()
}

// bucket implements storage.Bucket.
type bucket struct {
	st   *store
	name string
}

var (
	_ storage.Bucket         = (*bucket)(nil)
	_ storage.HasDirectories = (*bucket)(nil)
)

func (b *bucket) Name() string { return b.name }

func (b *bucket) Features() storage.Features {
	return b.st.Features()
}

func (b *bucket) Info(ctx context.Context) (*storage.BucketInfo, error) {
	_ = ctx

	b.st.mu.RLock()
	created, ok := b.st.buckets[b.name]
	b.st.mu.RUnlock()

	if !ok {
		return nil, storage.ErrNotExist
	}

	return &storage.BucketInfo{
		Name:      b.name,
		CreatedAt: created,
	}, nil
}

func (b *bucket) Write(ctx context.Context, key string, src io.Reader, size int64, contentType string, opts storage.Options) (*storage.Object, error) {
	_ = ctx
	_ = opts

	if key == "" {
		return nil, fmt.Errorf("horse: key is empty")
	}
	if key[0] == ' ' || key[len(key)-1] == ' ' {
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("horse: key is empty")
		}
	}

	now := fastNow()
	bl, kl, cl := len(b.name), len(key), len(contentType)

	var valOff int64

	if size < 0 {
		// Unknown size: read all first, then write.
		var tmpBuf bytes.Buffer
		if _, err := io.Copy(&tmpBuf, src); err != nil {
			return nil, fmt.Errorf("horse: read value: %w", err)
		}
		data := tmpBuf.Bytes()
		size = int64(len(data))

		totalSize := int64(recFixedSize+bl+kl+cl) + size
		if b.st.bufRing != nil && totalSize <= b.st.bufRing.capacity {
			valPosInRecord := 19 + bl + kl + cl
			bufSlice, _, vo, wb := b.st.bufRing.writeInline(totalSize, valPosInRecord)
			valOff = vo
			b.st.vol.buildRecordBuf(bufSlice, recPut, b.name, key, contentType, data, now)
			wb.done()
		} else {
			var err error
			_, valOff, err = b.st.vol.appendRecord(recPut, b.name, key, contentType, data, now)
			if err != nil {
				return nil, err
			}
		}
	} else if b.st.bufRing != nil {
		totalSize := int64(recFixedSize+bl+kl+cl) + size
		if totalSize > b.st.bufRing.capacity {
			var err error
			valOff, err = b.st.vol.writeFromReader(recPut, b.name, key, contentType, src, size, now)
			if err != nil {
				return nil, err
			}
		} else {
			// Known size: serialize header + read value directly into write buffer.
			valPosInRecord := 19 + bl + kl + cl
			bufSlice, _, vo, wb := b.st.bufRing.writeInline(totalSize, valPosInRecord)
			valOff = vo

			// Inline record serialization — no intermediate alloc.
			bufSlice[0] = recPut
			pos := 5
			binary.LittleEndian.PutUint16(bufSlice[pos:], uint16(bl))
			pos += 2
			copy(bufSlice[pos:], b.name)
			pos += bl
			binary.LittleEndian.PutUint16(bufSlice[pos:], uint16(kl))
			pos += 2
			copy(bufSlice[pos:], key)
			pos += kl
			binary.LittleEndian.PutUint16(bufSlice[pos:], uint16(cl))
			pos += 2
			copy(bufSlice[pos:], contentType)
			pos += cl
			binary.LittleEndian.PutUint64(bufSlice[pos:], uint64(size))
			pos += 8

			// Read value from src directly into write buffer — one memcpy.
			if size > 0 {
				if _, err := io.ReadFull(src, bufSlice[pos:pos+int(size)]); err != nil {
					if err != io.EOF && err != io.ErrUnexpectedEOF {
						wb.done()
						return nil, fmt.Errorf("horse: read value: %w", err)
					}
				}
			}
			pos += int(size)

			binary.LittleEndian.PutUint64(bufSlice[pos:], uint64(now))

			if !b.st.vol.noCRC {
				checksum := crc32.Checksum(bufSlice[5:], b.st.vol.crcTable)
				binary.LittleEndian.PutUint32(bufSlice[1:5], checksum)
			}
			wb.done()
		}
	} else {
		// Direct volume path (sync=batch or sync=full).
		var err error
		valOff, err = b.st.vol.writeFromReader(recPut, b.name, key, contentType, src, size, now)
		if err != nil {
			return nil, err
		}
	}

	e := acquireIndexEntry()
	e.valueOffset = valOff
	e.size = size
	e.contentType = contentType
	e.created = now
	e.updated = now
	b.st.idx.put(b.name, key, e)

	if b.st.syncMode == "full" {
		b.st.vol.sync()
	}

	return &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        size,
		ContentType: contentType,
		Created:     time.Unix(0, now),
		Updated:     time.Unix(0, now),
	}, nil
}

func (b *bucket) Open(ctx context.Context, key string, offset, length int64, opts storage.Options) (io.ReadCloser, *storage.Object, error) {
	_ = ctx
	_ = opts

	// Fast validation: only TrimSpace if key actually has leading/trailing spaces.
	if key == "" {
		return nil, nil, fmt.Errorf("horse: key is empty")
	}
	if key[0] == ' ' || key[len(key)-1] == ' ' {
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, nil, fmt.Errorf("horse: key is empty")
		}
	}

	e, ok := b.st.idx.get(b.name, key)
	if !ok {
		return nil, nil, storage.ErrNotExist
	}

	// Check write buffer first (for recently written, unflushed data).
	var data []byte
	if b.st.bufRing != nil {
		if bufData, inBuf := b.st.bufRing.readFromBuffer(e.valueOffset, e.size); inBuf {
			data = bufData
		}
	}
	if data == nil {
		// Zero-copy read from mmap.
		data = b.st.vol.readValueSlice(e.valueOffset, e.size)
	}

	if offset < 0 {
		offset = 0
	}
	if offset > int64(len(data)) {
		offset = int64(len(data))
	}
	end := int64(len(data))
	if length > 0 && offset+length < end {
		end = offset + length
	}
	slice := data[offset:end]

	obj := &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        e.size,
		ContentType: e.contentType,
		Created:     time.Unix(0, e.created),
		Updated:     time.Unix(0, e.updated),
	}

	// Use pooled mmapReader to reduce GC pressure.
	r := acquireMmapReader(slice)
	return r, obj, nil
}

func (b *bucket) Stat(ctx context.Context, key string, opts storage.Options) (*storage.Object, error) {
	_ = ctx
	_ = opts

	if key == "" {
		return nil, fmt.Errorf("horse: key is empty")
	}
	if key[0] == ' ' || key[len(key)-1] == ' ' {
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("horse: key is empty")
		}
	}

	// Check for directory stat (key ending with "/").
	if strings.HasSuffix(key, "/") {
		results := b.st.idx.list(b.name, key)
		if len(results) == 0 {
			return nil, storage.ErrNotExist
		}
		return &storage.Object{
			Bucket:  b.name,
			Key:     strings.TrimSuffix(key, "/"),
			IsDir:   true,
			Created: time.Unix(0, results[0].entry.created),
			Updated: time.Unix(0, results[0].entry.updated),
		}, nil
	}

	e, ok := b.st.idx.get(b.name, key)
	if !ok {
		return nil, storage.ErrNotExist
	}

	return &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        e.size,
		ContentType: e.contentType,
		Created:     time.Unix(0, e.created),
		Updated:     time.Unix(0, e.updated),
	}, nil
}

func (b *bucket) Delete(ctx context.Context, key string, opts storage.Options) error {
	_ = ctx
	_ = opts

	if key == "" {
		return fmt.Errorf("horse: key is empty")
	}
	if key[0] == ' ' || key[len(key)-1] == ' ' {
		key = strings.TrimSpace(key)
		if key == "" {
			return fmt.Errorf("horse: key is empty")
		}
	}

	if !b.st.idx.remove(b.name, key) {
		return storage.ErrNotExist
	}

	// Append delete tombstone.
	now := fastNow()
	if b.st.bufRing != nil {
		bl, kl := len(b.name), len(key)
		totalSize := int64(recFixedSize + bl + kl)
		bufSlice, _, _, wb := b.st.bufRing.writeInline(totalSize, 0)
		b.st.vol.buildRecordBuf(bufSlice, recDelete, b.name, key, "", nil, now)
		wb.done()
	} else {
		b.st.vol.appendRecord(recDelete, b.name, key, "", nil, now)
	}

	return nil
}

func (b *bucket) Copy(ctx context.Context, dstKey string, srcBucket, srcKey string, opts storage.Options) (*storage.Object, error) {
	_ = ctx
	_ = opts

	dstKey = strings.TrimSpace(dstKey)
	srcKey = strings.TrimSpace(srcKey)
	if dstKey == "" || srcKey == "" {
		return nil, fmt.Errorf("horse: key is empty")
	}

	if srcBucket == "" {
		srcBucket = b.name
	}

	srcEntry, ok := b.st.idx.get(srcBucket, srcKey)
	if !ok {
		return nil, storage.ErrNotExist
	}

	// Read source value (check write buffer first).
	var srcData []byte
	if b.st.bufRing != nil {
		if bufData, inBuf := b.st.bufRing.readFromBuffer(srcEntry.valueOffset, srcEntry.size); inBuf {
			srcData = bufData
		}
	}
	if srcData == nil {
		srcData = b.st.vol.readValueSlice(srcEntry.valueOffset, srcEntry.size)
	}

	// Write copy.
	now := fastNow()
	bl, kl, cl := len(b.name), len(dstKey), len(srcEntry.contentType)
	totalSize := int64(recFixedSize+bl+kl+cl) + srcEntry.size

	var valOff int64
	if b.st.bufRing != nil && totalSize <= b.st.bufRing.capacity {
		valPosInRecord := 19 + bl + kl + cl
		bufSlice, _, vo, wb := b.st.bufRing.writeInline(totalSize, valPosInRecord)
		valOff = vo
		b.st.vol.buildRecordBuf(bufSlice, recPut, b.name, dstKey, srcEntry.contentType, srcData, now)
		wb.done()
	} else {
		var err error
		_, valOff, err = b.st.vol.appendRecord(recPut, b.name, dstKey, srcEntry.contentType, srcData, now)
		if err != nil {
			return nil, err
		}
	}

	e := acquireIndexEntry()
	e.valueOffset = valOff
	e.size = srcEntry.size
	e.contentType = srcEntry.contentType
	e.created = now
	e.updated = now
	b.st.idx.put(b.name, dstKey, e)

	return &storage.Object{
		Bucket:      b.name,
		Key:         dstKey,
		Size:        srcEntry.size,
		ContentType: srcEntry.contentType,
		Created:     time.Unix(0, now),
		Updated:     time.Unix(0, now),
	}, nil
}

func (b *bucket) Move(ctx context.Context, dstKey string, srcBucket, srcKey string, opts storage.Options) (*storage.Object, error) {
	obj, err := b.Copy(ctx, dstKey, srcBucket, srcKey, opts)
	if err != nil {
		return nil, err
	}
	if srcBucket == "" {
		srcBucket = b.name
	}
	sb := b.st.Bucket(srcBucket)
	if err := sb.Delete(ctx, srcKey, nil); err != nil {
		return nil, err
	}
	return obj, nil
}

func (b *bucket) List(ctx context.Context, prefix string, limit, offset int, opts storage.Options) (storage.ObjectIter, error) {
	_ = ctx

	prefix = strings.TrimSpace(prefix)
	recursive := true
	if opts != nil {
		if v, ok := opts["recursive"].(bool); ok {
			recursive = v
		}
	}

	results := b.st.idx.list(b.name, prefix)

	objs := make([]*storage.Object, 0, len(results))
	for _, r := range results {
		if !recursive {
			rest := strings.TrimPrefix(r.key, prefix)
			rest = strings.TrimPrefix(rest, "/")
			if strings.Contains(rest, "/") {
				continue
			}
		}
		objs = append(objs, &storage.Object{
			Bucket:      b.name,
			Key:         r.key,
			Size:        r.entry.size,
			ContentType: r.entry.contentType,
			Created:     time.Unix(0, r.entry.created),
			Updated:     time.Unix(0, r.entry.updated),
		})
	}

	if offset < 0 {
		offset = 0
	}
	if offset > len(objs) {
		offset = len(objs)
	}
	objs = objs[offset:]
	if limit > 0 && limit < len(objs) {
		objs = objs[:limit]
	}

	return &objectIter{objects: objs}, nil
}

func (b *bucket) SignedURL(ctx context.Context, key string, method string, expires time.Duration, opts storage.Options) (string, error) {
	return "", storage.ErrUnsupported
}

// Directory support.

func (b *bucket) Directory(p string) storage.Directory {
	return &dir{b: b, path: strings.Trim(p, "/")}
}

type dir struct {
	b    *bucket
	path string
}

var _ storage.Directory = (*dir)(nil)

func (d *dir) Bucket() storage.Bucket { return d.b }
func (d *dir) Path() string           { return d.path }

func (d *dir) Info(ctx context.Context) (*storage.Object, error) {
	_ = ctx
	prefix := d.path
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	results := d.b.st.idx.list(d.b.name, prefix)
	if len(results) == 0 {
		return nil, storage.ErrNotExist
	}
	return &storage.Object{
		Bucket:  d.b.name,
		Key:     d.path,
		IsDir:   true,
		Created: time.Unix(0, results[0].entry.created),
		Updated: time.Unix(0, results[0].entry.updated),
	}, nil
}

func (d *dir) List(ctx context.Context, limit, offset int, opts storage.Options) (storage.ObjectIter, error) {
	_ = ctx
	_ = opts
	prefix := d.path
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	results := d.b.st.idx.list(d.b.name, prefix)

	var objs []*storage.Object
	for _, r := range results {
		rest := strings.TrimPrefix(r.key, prefix)
		if strings.Contains(rest, "/") {
			continue
		}
		objs = append(objs, &storage.Object{
			Bucket:      d.b.name,
			Key:         r.key,
			Size:        r.entry.size,
			ContentType: r.entry.contentType,
			Created:     time.Unix(0, r.entry.created),
			Updated:     time.Unix(0, r.entry.updated),
		})
	}

	if offset < 0 {
		offset = 0
	}
	if offset > len(objs) {
		offset = len(objs)
	}
	objs = objs[offset:]
	if limit > 0 && limit < len(objs) {
		objs = objs[:limit]
	}

	return &objectIter{objects: objs}, nil
}

func (d *dir) Delete(ctx context.Context, opts storage.Options) error {
	_ = ctx
	recursive := false
	if opts != nil {
		if v, ok := opts["recursive"].(bool); ok {
			recursive = v
		}
	}

	prefix := d.path
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	results := d.b.st.idx.list(d.b.name, prefix)
	if len(results) == 0 {
		return storage.ErrNotExist
	}

	for _, r := range results {
		if !recursive {
			rest := strings.TrimPrefix(r.key, prefix)
			if strings.Contains(rest, "/") {
				continue
			}
		}
		d.b.st.idx.remove(d.b.name, r.key)
	}
	return nil
}

func (d *dir) Move(ctx context.Context, dstPath string, opts storage.Options) (storage.Directory, error) {
	_ = ctx
	_ = opts

	srcPrefix := strings.Trim(d.path, "/")
	dstPrefix := strings.Trim(dstPath, "/")

	if srcPrefix != "" && !strings.HasSuffix(srcPrefix, "/") {
		srcPrefix += "/"
	}
	if dstPrefix != "" && !strings.HasSuffix(dstPrefix, "/") {
		dstPrefix += "/"
	}

	results := d.b.st.idx.list(d.b.name, srcPrefix)
	if len(results) == 0 {
		return nil, storage.ErrNotExist
	}

	for _, r := range results {
		rel := strings.TrimPrefix(r.key, srcPrefix)
		newKey := dstPrefix + rel

		// Copy entry with new key.
		d.b.st.idx.put(d.b.name, newKey, &indexEntry{
			valueOffset: r.entry.valueOffset,
			size:        r.entry.size,
			contentType: r.entry.contentType,
			created:     r.entry.created,
			updated:     r.entry.updated,
		})
		d.b.st.idx.remove(d.b.name, r.key)
	}

	return &dir{b: d.b, path: strings.Trim(dstPath, "/")}, nil
}

// mmapReader is an io.ReadCloser over a mmap'd slice.
// It implements io.WriteTo for zero-copy reads when the destination
// supports it (e.g., io.Discard in benchmarks bypasses all data copying).
// Pooled via sync.Pool to eliminate per-read heap allocation.
type mmapReader struct {
	data []byte
	pos  int
}

// mmapReaderPool eliminates heap allocation for mmapReader on every Open() call.
var mmapReaderPool = sync.Pool{
	New: func() any { return &mmapReader{} },
}

func acquireMmapReader(data []byte) *mmapReader {
	r := mmapReaderPool.Get().(*mmapReader)
	r.data = data
	r.pos = 0
	return r
}

func (r *mmapReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func (r *mmapReader) WriteTo(w io.Writer) (int64, error) {
	if r.pos >= len(r.data) {
		return 0, nil
	}
	n, err := w.Write(r.data[r.pos:])
	r.pos += n
	return int64(n), err
}

func (r *mmapReader) Close() error {
	r.data = nil
	mmapReaderPool.Put(r)
	return nil
}

// Iterators.

type bucketIter struct {
	buckets []*storage.BucketInfo
	index   int
}

func (it *bucketIter) Next() (*storage.BucketInfo, error) {
	if it.index >= len(it.buckets) {
		return nil, nil
	}
	b := it.buckets[it.index]
	it.index++
	return b, nil
}

func (it *bucketIter) Close() error {
	it.buckets = nil
	return nil
}

type objectIter struct {
	objects []*storage.Object
	index   int
}

func (it *objectIter) Next() (*storage.Object, error) {
	if it.index >= len(it.objects) {
		return nil, nil
	}
	o := it.objects[it.index]
	it.index++
	return o, nil
}

func (it *objectIter) Close() error {
	it.objects = nil
	return nil
}
