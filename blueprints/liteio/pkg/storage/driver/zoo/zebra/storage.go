// Package zebra implements a high-performance striped storage driver.
//
// Architecture: Haystack-inspired append-only volumes with Tectonic-style
// striped sharding. Multiple independent stripes eliminate cross-stripe
// contention. Inline value caching bypasses volume I/O for small objects.
//
// DSN formats:
//
//	Embedded: zebra:///path?stripes=8&sync=none&inline_kb=4
//	Cluster:  zebra:///?peers=host:port,...&replicas=1
package zebra

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/liteio-dev/liteio/pkg/storage"
)

func init() {
	storage.Register("zebra", &driver{})
}

// Cached time to avoid time.Now() overhead per operation.
var cachedTimeNano atomic.Int64

func init() {
	cachedTimeNano.Store(time.Now().UnixNano())
	go func() {
		ticker := time.NewTicker(1 * time.Millisecond)
		for range ticker.C {
			cachedTimeNano.Store(time.Now().UnixNano())
		}
	}()
}

func fastNow() int64     { return cachedTimeNano.Load() }
func fastNowTime() time.Time { return time.Unix(0, fastNow()) }

// Driver is the exported driver type for cmd/zebra.
type Driver = driver

type driver struct{}

func (d *driver) Open(ctx context.Context, dsn string) (storage.Storage, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("zebra: parse dsn: %w", err)
	}

	// Cluster mode: zebra:///?peers=...
	if u.Query().Has("peers") {
		return openCluster(ctx, u)
	}

	// Embedded mode: zebra:///path
	return openEmbedded(ctx, u)
}

func openEmbedded(_ context.Context, u *url.URL) (*store, error) {
	q := u.Query()
	root := filepath.Clean(u.Path)
	if root == "" || root == "." {
		root = "/tmp/zebra-data"
	}

	numStripes := intParam(q, "stripes", 8)
	syncMode := q.Get("sync")
	if syncMode == "" {
		syncMode = "none"
	}
	inlineKB := intParam(q, "inline_kb", 4)
	preallocMB := intParam(q, "prealloc", 1024)
	bufSize := intParam(q, "bufsize", defaultBufSize)

	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("zebra: mkdir %q: %w", root, err)
	}

	s := &store{
		root:       root,
		syncMode:   syncMode,
		inlineMax:  int64(inlineKB) * 1024,
		numStripes: numStripes,
		stripes:    make([]*stripe, numStripes),
		buckets:    make(map[string]time.Time),
		mp:         newMultipartRegistry(),
	}

	for i := 0; i < numStripes; i++ {
		st, err := newStripe(i, root, syncMode, int64(preallocMB)*1024*1024, int64(bufSize), s.inlineMax)
		if err != nil {
			s.Close()
			return nil, err
		}
		s.stripes[i] = st
	}

	return s, nil
}

// stripe is a fully independent storage partition: own volume, index, and buffer ring.
type stripe struct {
	id   int
	vol  *volume
	idx  *index
	ring *bufferRing

	batcherStop chan struct{}
	batcherWg   sync.WaitGroup
}

func newStripe(id int, root, syncMode string, prealloc, bufSize, inlineMax int64) (*stripe, error) {
	path := filepath.Join(root, fmt.Sprintf("stripe_%d.dat", id))
	vol, err := newVolume(path, prealloc)
	if err != nil {
		return nil, err
	}

	if syncMode == "none" {
		vol.noCRC = true
	}

	idx := newIndex()
	st := &stripe{id: id, vol: vol, idx: idx}

	if syncMode == "none" {
		st.ring = newBufferRing(vol, bufSize)
	}

	// Recover index from existing volume data.
	if vol.tail.Load() > headerSize {
		vol.recover(idx, inlineMax)
	}

	if syncMode == "batch" {
		st.startBatcher()
	}

	return st, nil
}

func (st *stripe) startBatcher() {
	st.batcherStop = make(chan struct{})
	st.batcherWg.Add(1)
	go func() {
		defer st.batcherWg.Done()
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-st.batcherStop:
				return
			case <-ticker.C:
				st.vol.sync()
			}
		}
	}()
}

func (st *stripe) close() error {
	if st.batcherStop != nil {
		close(st.batcherStop)
		st.batcherWg.Wait()
	}
	if st.ring != nil {
		st.ring.close()
	}
	return st.vol.close()
}

// store implements storage.Storage with striped architecture.
type store struct {
	root       string
	syncMode   string
	inlineMax  int64
	numStripes int
	stripes    []*stripe

	mu      sync.RWMutex
	buckets map[string]time.Time

	mp *multipartRegistry
}

var _ storage.Storage = (*store)(nil)

// stripeFor routes a key to its stripe using FNV-1a hash.
func (s *store) stripeFor(bucket, key string) *stripe {
	h := fnvHash(bucket, key)
	return s.stripes[h%uint64(s.numStripes)]
}

// stripeForH returns both the stripe and hash (for single-hash read/write paths).
func (s *store) stripeForH(bucket, key string) (*stripe, uint64) {
	h := fnvHash(bucket, key)
	return s.stripes[h%uint64(s.numStripes)], h
}

// fnvHash computes FNV-1a over bucket + 0x00 + key without allocation.
func fnvHash(bucket, key string) uint64 {
	const offset64 = 14695981039346656037
	const prime64 = 1099511628211
	h := uint64(offset64)
	for i := 0; i < len(bucket); i++ {
		h ^= uint64(bucket[i])
		h *= prime64
	}
	h ^= 0
	h *= prime64
	for i := 0; i < len(key); i++ {
		h ^= uint64(key[i])
		h *= prime64
	}
	return h
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

func (s *store) Buckets(_ context.Context, limit, offset int, _ storage.Options) (storage.BucketIter, error) {
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
		infos = append(infos, &storage.BucketInfo{Name: name, CreatedAt: s.buckets[name]})
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

func (s *store) CreateBucket(_ context.Context, name string, _ storage.Options) (*storage.BucketInfo, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("zebra: bucket name is empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.buckets[name]; ok {
		return nil, storage.ErrExist
	}
	now := fastNowTime()
	s.buckets[name] = now
	return &storage.BucketInfo{Name: name, CreatedAt: now}, nil
}

func (s *store) DeleteBucket(_ context.Context, name string, opts storage.Options) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("zebra: bucket name is empty")
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

	if !force {
		for _, st := range s.stripes {
			if st.idx.hasBucket(name) {
				return storage.ErrPermission
			}
		}
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
	var firstErr error
	for _, st := range s.stripes {
		if st != nil {
			if err := st.close(); err != nil && firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
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

func (b *bucket) Info(_ context.Context) (*storage.BucketInfo, error) {
	b.st.mu.RLock()
	created, ok := b.st.buckets[b.name]
	b.st.mu.RUnlock()
	if !ok {
		return nil, storage.ErrNotExist
	}
	return &storage.BucketInfo{Name: b.name, CreatedAt: created}, nil
}

func (b *bucket) Write(_ context.Context, key string, src io.Reader, size int64, contentType string, _ storage.Options) (*storage.Object, error) {
	if key == "" {
		return nil, fmt.Errorf("zebra: key is empty")
	}
	if key[0] == ' ' || key[len(key)-1] == ' ' {
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("zebra: key is empty")
		}
	}

	now := fastNow()
	st, h := b.st.stripeForH(b.name, key)

	// === Inline path: small values in sync=none bypass volume entirely ===
	if b.st.inlineMax > 0 && b.st.syncMode == "none" && size >= 0 && size <= b.st.inlineMax {
		data := make([]byte, size)
		if size > 0 {
			if _, err := io.ReadFull(src, data); err != nil {
				if err != io.EOF && err != io.ErrUnexpectedEOF {
					return nil, fmt.Errorf("zebra: read value: %w", err)
				}
			}
		}
		e := &indexEntry{
			size:        size,
			contentType: contentType,
			created:     now,
			updated:     now,
			inline:      data,
		}
		st.idx.putH(h, b.name, key, e)

		return &storage.Object{
			Bucket:      b.name,
			Key:         key,
			Size:        size,
			ContentType: contentType,
			Created:     time.Unix(0, now),
			Updated:     time.Unix(0, now),
		}, nil
	}

	// === Volume path (same as horse) ===
	bl, kl, cl := len(b.name), len(key), len(contentType)
	var valOff int64

	if size < 0 {
		// Unknown size: buffer then write.
		var tmpBuf bytes.Buffer
		if _, err := io.Copy(&tmpBuf, src); err != nil {
			return nil, fmt.Errorf("zebra: read value: %w", err)
		}
		data := tmpBuf.Bytes()
		size = int64(len(data))

		// Check if it fits inline after reading.
		if b.st.inlineMax > 0 && b.st.syncMode == "none" && size <= b.st.inlineMax {
			e := &indexEntry{
				size:        size,
				contentType: contentType,
				created:     now,
				updated:     now,
				inline:      data,
			}
			st.idx.put(b.name, key, e)
			return &storage.Object{
				Bucket: b.name, Key: key, Size: size, ContentType: contentType,
				Created: time.Unix(0, now), Updated: time.Unix(0, now),
			}, nil
		}

		totalSize := int64(recFixedSize+bl+kl+cl) + size
		if st.ring != nil && totalSize <= st.ring.capacity {
			valPosInRecord := 19 + bl + kl + cl
			bufSlice, _, vo, wb := st.ring.writeInline(totalSize, valPosInRecord)
			valOff = vo
			st.vol.buildRecordBuf(bufSlice, recPut, b.name, key, contentType, data, now)
			wb.done()
		} else {
			var err error
			_, valOff, err = st.vol.appendRecord(recPut, b.name, key, contentType, data, now)
			if err != nil {
				return nil, err
			}
		}
	} else if st.ring != nil {
		totalSize := int64(recFixedSize+bl+kl+cl) + size
		if totalSize > st.ring.capacity {
			var err error
			valOff, err = st.vol.writeFromReader(recPut, b.name, key, contentType, src, size, now)
			if err != nil {
				return nil, err
			}
		} else {
			valPosInRecord := 19 + bl + kl + cl
			bufSlice, _, vo, wb := st.ring.writeInline(totalSize, valPosInRecord)
			valOff = vo

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

			if size > 0 {
				if _, err := io.ReadFull(src, bufSlice[pos:pos+int(size)]); err != nil {
					if err != io.EOF && err != io.ErrUnexpectedEOF {
						wb.done()
						return nil, fmt.Errorf("zebra: read value: %w", err)
					}
				}
			}
			pos += int(size)
			binary.LittleEndian.PutUint64(bufSlice[pos:], uint64(now))

			if !st.vol.noCRC {
				checksum := crc32.Checksum(bufSlice[5:], st.vol.crcTable)
				binary.LittleEndian.PutUint32(bufSlice[1:5], checksum)
			}
			wb.done()
		}
	} else {
		var err error
		valOff, err = st.vol.writeFromReader(recPut, b.name, key, contentType, src, size, now)
		if err != nil {
			return nil, err
		}
	}

	e := acquireEntry()
	e.valueOffset = valOff
	e.size = size
	e.contentType = contentType
	e.created = now
	e.updated = now
	st.idx.putH(h, b.name, key, e)

	if b.st.syncMode == "full" {
		st.vol.sync()
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

func (b *bucket) Open(_ context.Context, key string, offset, length int64, _ storage.Options) (io.ReadCloser, *storage.Object, error) {
	if key == "" {
		return nil, nil, fmt.Errorf("zebra: key is empty")
	}
	if key[0] == ' ' || key[len(key)-1] == ' ' {
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, nil, fmt.Errorf("zebra: key is empty")
		}
	}

	st, h := b.st.stripeForH(b.name, key)
	e, ok := st.idx.getH(h, b.name, key)
	if !ok {
		return nil, nil, storage.ErrNotExist
	}

	var data []byte
	if e.inline != nil {
		data = e.inline
	} else {
		if st.ring != nil {
			if bufData, inBuf := st.ring.readFromBuffer(e.valueOffset, e.size); inBuf {
				data = bufData
			}
		}
		if data == nil {
			data = st.vol.readValueSlice(e.valueOffset, e.size)
		}
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

	r := acquireReader(slice)
	return r, obj, nil
}

func (b *bucket) Stat(_ context.Context, key string, _ storage.Options) (*storage.Object, error) {
	if key == "" {
		return nil, fmt.Errorf("zebra: key is empty")
	}
	if key[0] == ' ' || key[len(key)-1] == ' ' {
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("zebra: key is empty")
		}
	}

	if strings.HasSuffix(key, "/") {
		results := b.listAll(key)
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

	st, h := b.st.stripeForH(b.name, key)
	e, ok := st.idx.getH(h, b.name, key)
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

func (b *bucket) Delete(_ context.Context, key string, _ storage.Options) error {
	if key == "" {
		return fmt.Errorf("zebra: key is empty")
	}
	if key[0] == ' ' || key[len(key)-1] == ' ' {
		key = strings.TrimSpace(key)
		if key == "" {
			return fmt.Errorf("zebra: key is empty")
		}
	}

	st := b.st.stripeFor(b.name, key)

	// Check if inline — if so, just remove from index (no volume tombstone in sync=none).
	e, ok := st.idx.get(b.name, key)
	if !ok {
		return storage.ErrNotExist
	}
	isInline := e.inline != nil

	if !st.idx.remove(b.name, key) {
		return storage.ErrNotExist
	}

	// Skip volume tombstone for inline values in sync=none (no recovery needed).
	if isInline && b.st.syncMode == "none" {
		return nil
	}

	// Append delete tombstone for volume-backed entries.
	now := fastNow()
	if st.ring != nil {
		bl, kl := len(b.name), len(key)
		totalSize := int64(recFixedSize + bl + kl)
		bufSlice, _, _, wb := st.ring.writeInline(totalSize, 0)
		st.vol.buildRecordBuf(bufSlice, recDelete, b.name, key, "", nil, now)
		wb.done()
	} else {
		st.vol.appendRecord(recDelete, b.name, key, "", nil, now)
	}

	return nil
}

func (b *bucket) Copy(_ context.Context, dstKey string, srcBucket, srcKey string, _ storage.Options) (*storage.Object, error) {
	dstKey = strings.TrimSpace(dstKey)
	srcKey = strings.TrimSpace(srcKey)
	if dstKey == "" || srcKey == "" {
		return nil, fmt.Errorf("zebra: key is empty")
	}
	if srcBucket == "" {
		srcBucket = b.name
	}

	srcStripe := b.st.stripeFor(srcBucket, srcKey)
	srcEntry, ok := srcStripe.idx.get(srcBucket, srcKey)
	if !ok {
		return nil, storage.ErrNotExist
	}

	// Read source value.
	var srcData []byte
	if srcEntry.inline != nil {
		srcData = srcEntry.inline
	} else {
		if srcStripe.ring != nil {
			if bufData, inBuf := srcStripe.ring.readFromBuffer(srcEntry.valueOffset, srcEntry.size); inBuf {
				srcData = bufData
			}
		}
		if srcData == nil {
			srcData = srcStripe.vol.readValueSlice(srcEntry.valueOffset, srcEntry.size)
		}
	}

	// Write copy to destination stripe.
	now := fastNow()
	dstStripe := b.st.stripeFor(b.name, dstKey)

	// Inline path for copy.
	if b.st.inlineMax > 0 && b.st.syncMode == "none" && srcEntry.size <= b.st.inlineMax {
		dataCopy := make([]byte, len(srcData))
		copy(dataCopy, srcData)
		e := &indexEntry{
			size:        srcEntry.size,
			contentType: srcEntry.contentType,
			created:     now,
			updated:     now,
			inline:      dataCopy,
		}
		dstStripe.idx.put(b.name, dstKey, e)
		return &storage.Object{
			Bucket: b.name, Key: dstKey, Size: srcEntry.size,
			ContentType: srcEntry.contentType,
			Created:     time.Unix(0, now), Updated: time.Unix(0, now),
		}, nil
	}

	bl, kl, cl := len(b.name), len(dstKey), len(srcEntry.contentType)
	totalSize := int64(recFixedSize+bl+kl+cl) + srcEntry.size

	var valOff int64
	if dstStripe.ring != nil && totalSize <= dstStripe.ring.capacity {
		valPosInRecord := 19 + bl + kl + cl
		bufSlice, _, vo, wb := dstStripe.ring.writeInline(totalSize, valPosInRecord)
		valOff = vo
		dstStripe.vol.buildRecordBuf(bufSlice, recPut, b.name, dstKey, srcEntry.contentType, srcData, now)
		wb.done()
	} else {
		var err error
		_, valOff, err = dstStripe.vol.appendRecord(recPut, b.name, dstKey, srcEntry.contentType, srcData, now)
		if err != nil {
			return nil, err
		}
	}

	e := acquireEntry()
	e.valueOffset = valOff
	e.size = srcEntry.size
	e.contentType = srcEntry.contentType
	e.created = now
	e.updated = now
	dstStripe.idx.put(b.name, dstKey, e)

	return &storage.Object{
		Bucket: b.name, Key: dstKey, Size: srcEntry.size,
		ContentType: srcEntry.contentType,
		Created:     time.Unix(0, now), Updated: time.Unix(0, now),
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

// listAll gathers results from ALL stripes for the given bucket+prefix.
func (b *bucket) listAll(prefix string) []listResult {
	var all []listResult
	for _, st := range b.st.stripes {
		results := st.idx.list(b.name, prefix)
		all = append(all, results...)
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].key < all[j].key
	})
	return all
}

func (b *bucket) List(_ context.Context, prefix string, limit, offset int, opts storage.Options) (storage.ObjectIter, error) {
	prefix = strings.TrimSpace(prefix)
	recursive := true
	if opts != nil {
		if v, ok := opts["recursive"].(bool); ok {
			recursive = v
		}
	}

	results := b.listAll(prefix)

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

func (b *bucket) SignedURL(_ context.Context, _ string, _ string, _ time.Duration, _ storage.Options) (string, error) {
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

func (d *dir) Info(_ context.Context) (*storage.Object, error) {
	prefix := d.path
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	results := d.b.listAll(prefix)
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

func (d *dir) List(_ context.Context, limit, offset int, _ storage.Options) (storage.ObjectIter, error) {
	prefix := d.path
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	results := d.b.listAll(prefix)

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

func (d *dir) Delete(_ context.Context, opts storage.Options) error {
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

	results := d.b.listAll(prefix)
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
		st := d.b.st.stripeFor(d.b.name, r.key)
		st.idx.remove(d.b.name, r.key)
	}
	return nil
}

func (d *dir) Move(_ context.Context, dstPath string, _ storage.Options) (storage.Directory, error) {
	srcPrefix := strings.Trim(d.path, "/")
	dstPrefix := strings.Trim(dstPath, "/")

	if srcPrefix != "" && !strings.HasSuffix(srcPrefix, "/") {
		srcPrefix += "/"
	}
	if dstPrefix != "" && !strings.HasSuffix(dstPrefix, "/") {
		dstPrefix += "/"
	}

	results := d.b.listAll(srcPrefix)
	if len(results) == 0 {
		return nil, storage.ErrNotExist
	}

	for _, r := range results {
		rel := strings.TrimPrefix(r.key, srcPrefix)
		newKey := dstPrefix + rel

		srcStripe := d.b.st.stripeFor(d.b.name, r.key)
		dstStripe := d.b.st.stripeFor(d.b.name, newKey)

		dstStripe.idx.put(d.b.name, newKey, &indexEntry{
			valueOffset: r.entry.valueOffset,
			size:        r.entry.size,
			contentType: r.entry.contentType,
			created:     r.entry.created,
			updated:     r.entry.updated,
			inline:      r.entry.inline,
		})
		srcStripe.idx.remove(d.b.name, r.key)
	}

	return &dir{b: d.b, path: strings.Trim(dstPath, "/")}, nil
}

// Reader with pooling and WriteTo support.
type zebraReader struct {
	data []byte
	pos  int
}

var readerPool = sync.Pool{
	New: func() any { return &zebraReader{} },
}

func acquireReader(data []byte) *zebraReader {
	r := readerPool.Get().(*zebraReader)
	r.data = data
	r.pos = 0
	return r
}

func (r *zebraReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func (r *zebraReader) WriteTo(w io.Writer) (int64, error) {
	if r.pos >= len(r.data) {
		return 0, nil
	}
	n, err := w.Write(r.data[r.pos:])
	r.pos += n
	return int64(n), err
}

func (r *zebraReader) Close() error {
	r.data = nil
	readerPool.Put(r)
	return nil
}

// Iterators.

type bucketIter struct {
	buckets []*storage.BucketInfo
	idx     int
}

func (it *bucketIter) Next() (*storage.BucketInfo, error) {
	if it.idx >= len(it.buckets) {
		return nil, nil
	}
	b := it.buckets[it.idx]
	it.idx++
	return b, nil
}

func (it *bucketIter) Close() error {
	it.buckets = nil
	return nil
}

type objectIter struct {
	objects []*storage.Object
	idx     int
}

func (it *objectIter) Next() (*storage.Object, error) {
	if it.idx >= len(it.objects) {
		return nil, nil
	}
	o := it.objects[it.idx]
	it.idx++
	return o, nil
}

func (it *objectIter) Close() error {
	it.objects = nil
	return nil
}

// Helpers.

func intParam(q url.Values, key string, defaultVal int) int {
	s := q.Get(key)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		return defaultVal
	}
	return v
}
