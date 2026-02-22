// Package pony implements a memory-constrained striped object storage driver.
//
// Architecture: 4 stripes, each with an append-only volume and an mmap'd on-disk hash table
// index. This keeps total process heap near zero (all buffers are mmap-backed) while
// providing good parallel throughput via stripe-level isolation.
//
// DSN format:
//
//	pony:///path/to/data
//	pony:///path/to/data?sync=none
//	pony:///path/to/data?prealloc=256     (MB, default 256)
//	pony:///path/to/data?bufsize=4194304  (bytes, default 4MB)
//	pony:///path/to/data?slots=65536      (initial hash table slots, default 64K)
package pony

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

const (
	numStripes = 4
	stripeMask = numStripes - 1
	shardsPerStripe = 64
)

func init() {
	storage.Register("pony", &driver{})
}

type driver struct{}

func (d *driver) Open(ctx context.Context, dsn string) (storage.Storage, error) {
	_ = ctx

	u, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("pony: parse dsn: %w", err)
	}
	if u.Scheme != "pony" && u.Scheme != "" {
		return nil, fmt.Errorf("pony: unexpected scheme %q", u.Scheme)
	}

	root := filepath.Clean(u.Path)
	if root == "" || root == "." {
		root = "/tmp/pony-data"
	}

	syncMode := u.Query().Get("sync")
	if syncMode == "" {
		syncMode = "none"
	}

	prealloc := int64(defaultPrealloc)
	if pa := u.Query().Get("prealloc"); pa != "" {
		if mb, err := strconv.ParseInt(pa, 10, 64); err == nil && mb > 0 {
			prealloc = mb * 1024 * 1024
		}
	}

	initialSlots := uint64(0)
	if sl := u.Query().Get("slots"); sl != "" {
		if n, err := strconv.ParseUint(sl, 10, 64); err == nil && n > 0 {
			initialSlots = n
		}
	}

	bufSize := int64(defaultBufSize)
	if bs := u.Query().Get("bufsize"); bs != "" {
		if n, err := strconv.ParseInt(bs, 10, 64); err == nil && n > 0 {
			bufSize = n
		}
	}

	st := &store{
		root:     root,
		syncMode: syncMode,
		buckets:  make(map[string]time.Time),
		mp:       newMultipartRegistry(),
	}

	// Divide prealloc and slots across stripes.
	preallocPerStripe := prealloc / numStripes
	slotsPerShard := initialSlots / (numStripes * shardsPerStripe)

	for i := 0; i < numStripes; i++ {
		stripeDir := filepath.Join(root, fmt.Sprintf("stripe_%d", i))
		volPath := filepath.Join(stripeDir, "volume.dat")

		vol, err := newVolume(volPath, preallocPerStripe)
		if err != nil {
			// Close already-opened stripes.
			for j := 0; j < i; j++ {
				st.stripes[j].close()
			}
			return nil, err
		}

		if syncMode == "none" {
			vol.noCRC = true
		}

		idx, err := newShardedIndex(stripeDir, shardsPerStripe, slotsPerShard)
		if err != nil {
			vol.close()
			for j := 0; j < i; j++ {
				st.stripes[j].close()
			}
			return nil, err
		}

		// Recovery: if index is empty but volume has data, recover.
		if idx.totalEntryCount() == 0 && vol.tail.Load() > headerSize {
			idx.reset()
			if err := vol.recover(idx); err != nil {
				idx.close()
				vol.close()
				for j := 0; j < i; j++ {
					st.stripes[j].close()
				}
				return nil, fmt.Errorf("pony: stripe %d recovery failed: %w", i, err)
			}
		}

		s := &stripe{
			id:  i,
			vol: vol,
			idx: idx,
		}

		if syncMode == "none" {
			s.bufRing = newBufferRing(vol, bufSize)
		}

		st.stripes[i] = s
	}

	if syncMode == "batch" {
		st.startBatcher()
	}

	return st, nil
}

// stripe is an independent partition with its own volume, index, and buffer ring.
type stripe struct {
	id      int
	vol     *volume
	idx     *shardedIndex
	bufRing *bufferRing
}

func (s *stripe) close() {
	if s.bufRing != nil {
		s.bufRing.close()
	}
	s.idx.close()
	s.vol.close()
}

// Cached time — both nano and time.Time pointer to avoid time.Unix(0, n) allocs.
var cachedTimeNano atomic.Int64
var cachedTimePtr atomic.Pointer[time.Time]

func init() {
	now := time.Now()
	cachedTimeNano.Store(now.UnixNano())
	cachedTimePtr.Store(&now)
	go func() {
		ticker := time.NewTicker(500 * time.Microsecond)
		for range ticker.C {
			now := time.Now()
			cachedTimeNano.Store(now.UnixNano())
			cachedTimePtr.Store(&now)
		}
	}()
}

func fastNow() int64 {
	return cachedTimeNano.Load()
}

func fastNowTime() time.Time {
	return *cachedTimePtr.Load()
}

// Content type interning — most apps use <10 distinct types.
// Used by index.go getWithHash for zero-copy content type reads.
var contentTypeIntern sync.Map

func unsafePointer(b []byte) unsafe.Pointer {
	return unsafe.Pointer(&b[0])
}

// store implements storage.Storage with 4 independent stripes.
type store struct {
	root     string
	stripes  [numStripes]*stripe
	syncMode string

	mu      sync.RWMutex
	buckets map[string]time.Time

	mp *multipartRegistry

	batcherStop chan struct{}
	batcherWg   sync.WaitGroup
}

var _ storage.Storage = (*store)(nil)

// stripeFor routes a bucket+key to a stripe via FNV-1a hash.
func (s *store) stripeFor(bucket, key string) *stripe {
	h := hashComposite(bucket, key)
	return s.stripes[(h>>8)&stripeMask]
}

// stripeAndHash computes the hash once and returns both stripe and hash.
// Use this on hot paths to avoid double hashing (stripe routing + index lookup).
func (s *store) stripeAndHash(bucket, key string) (*stripe, uint64) {
	h := hashComposite(bucket, key)
	return s.stripes[(h>>8)&stripeMask], h
}

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
				for i := 0; i < numStripes; i++ {
					s.stripes[i].vol.sync()
				}
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
		return nil, fmt.Errorf("pony: bucket name is empty")
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
		return fmt.Errorf("pony: bucket name is empty")
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
		for i := 0; i < numStripes; i++ {
			if s.stripes[i].idx.hasBucket(name) {
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
	if s.batcherStop != nil {
		close(s.batcherStop)
		s.batcherWg.Wait()
	}

	for i := 0; i < numStripes; i++ {
		s.stripes[i].close()
	}

	return nil
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
		return nil, fmt.Errorf("pony: key is empty")
	}
	if key[0] == ' ' || key[len(key)-1] == ' ' {
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("pony: key is empty")
		}
	}

	now := fastNow()
	nowTime := fastNowTime()
	st, h := b.st.stripeAndHash(b.name, key)
	bl, kl, cl := len(b.name), len(key), len(contentType)

	var valOff int64

	if size < 0 {
		var tmpBuf bytes.Buffer
		if _, err := io.Copy(&tmpBuf, src); err != nil {
			return nil, fmt.Errorf("pony: read value: %w", err)
		}
		data := tmpBuf.Bytes()
		size = int64(len(data))

		totalSize := int64(recFixedSize+bl+kl+cl) + size
		if st.bufRing != nil && totalSize <= st.bufRing.capacity {
			valPosInRecord := 19 + bl + kl + cl
			bufSlice, _, vo, wb, isDirect := st.bufRing.writeInline(totalSize, valPosInRecord)
			valOff = vo
			st.vol.buildRecordBuf(bufSlice, recPut, b.name, key, contentType, data, now)
			if isDirect {
				st.bufRing.directFlush(bufSlice, vo-int64(valPosInRecord))
			} else {
				wb.done()
			}
		} else {
			var err error
			_, valOff, err = st.vol.appendRecord(recPut, b.name, key, contentType, data, now)
			if err != nil {
				return nil, err
			}
		}
	} else if st.bufRing != nil {
		totalSize := int64(recFixedSize+bl+kl+cl) + size
		if totalSize > st.bufRing.capacity {
			var err error
			valOff, err = st.vol.writeFromReader(recPut, b.name, key, contentType, src, size, now)
			if err != nil {
				return nil, err
			}
		} else {
			valPosInRecord := 19 + bl + kl + cl
			bufSlice, recOff, vo, wb, isDirect := st.bufRing.writeInline(totalSize, valPosInRecord)
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
				// Fast path: avoid io.ReadFull interface dispatch for bytes.Reader.
				if br, ok := src.(*bytes.Reader); ok {
					br.Read(bufSlice[pos : pos+int(size)])
				} else if _, err := io.ReadFull(src, bufSlice[pos:pos+int(size)]); err != nil {
					if err != io.EOF && err != io.ErrUnexpectedEOF {
						if !isDirect {
							wb.done()
						}
						return nil, fmt.Errorf("pony: read value: %w", err)
					}
				}
			}
			pos += int(size)

			binary.LittleEndian.PutUint64(bufSlice[pos:], uint64(now))

			if !st.vol.noCRC {
				checksum := crc32.Checksum(bufSlice[5:], st.vol.crcTable)
				binary.LittleEndian.PutUint32(bufSlice[1:5], checksum)
			}
			if isDirect {
				st.bufRing.directFlush(bufSlice, recOff)
			} else {
				wb.done()
			}
		}
	} else {
		var err error
		valOff, err = st.vol.writeFromReader(recPut, b.name, key, contentType, src, size, now)
		if err != nil {
			return nil, err
		}
	}

	st.idx.putWithHash(b.name, key, contentType, valOff, size, now, now, h)

	if b.st.syncMode == "full" {
		st.vol.sync()
	}

	return &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        size,
		ContentType: contentType,
		Created:     nowTime,
		Updated:     nowTime,
	}, nil
}

func (b *bucket) Open(ctx context.Context, key string, offset, length int64, opts storage.Options) (io.ReadCloser, *storage.Object, error) {
	_ = ctx
	_ = opts

	if key == "" {
		return nil, nil, fmt.Errorf("pony: key is empty")
	}
	if key[0] == ' ' || key[len(key)-1] == ' ' {
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, nil, fmt.Errorf("pony: key is empty")
		}
	}

	st, h := b.st.stripeAndHash(b.name, key)
	r, ok := st.idx.getWithHash(b.name, key, h)
	if !ok {
		return nil, nil, storage.ErrNotExist
	}

	var data []byte
	if st.bufRing != nil {
		if bufData, inBuf := st.bufRing.readFromBuffer(r.valOff, r.valSize); inBuf {
			data = bufData
		}
	}
	if data == nil {
		data = st.vol.readValueSlice(r.valOff, r.valSize)
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
		Size:        r.valSize,
		ContentType: r.contentType,
		Created:     time.Unix(0, r.created),
		Updated:     time.Unix(0, r.updated),
	}

	mr := acquireMmapReader(slice)
	return mr, obj, nil
}

func (b *bucket) Stat(ctx context.Context, key string, opts storage.Options) (*storage.Object, error) {
	_ = ctx
	_ = opts

	if key == "" {
		return nil, fmt.Errorf("pony: key is empty")
	}
	if key[0] == ' ' || key[len(key)-1] == ' ' {
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("pony: key is empty")
		}
	}

	if key[len(key)-1] == '/' {
		// Directory stat — scan all stripes for first match.
		for i := 0; i < numStripes; i++ {
			r, ok := b.st.stripes[i].idx.firstMatch(b.name, key)
			if ok {
				return &storage.Object{
					Bucket:  b.name,
					Key:     strings.TrimSuffix(key, "/"),
					IsDir:   true,
					Created: time.Unix(0, r.created),
					Updated: time.Unix(0, r.updated),
				}, nil
			}
		}
		return nil, storage.ErrNotExist
	}

	st, h := b.st.stripeAndHash(b.name, key)
	r, ok := st.idx.getWithHash(b.name, key, h)
	if !ok {
		return nil, storage.ErrNotExist
	}

	return &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        r.valSize,
		ContentType: r.contentType,
		Created:     time.Unix(0, r.created),
		Updated:     time.Unix(0, r.updated),
	}, nil
}

func (b *bucket) Delete(ctx context.Context, key string, opts storage.Options) error {
	_ = ctx
	_ = opts

	if key == "" {
		return fmt.Errorf("pony: key is empty")
	}
	if key[0] == ' ' || key[len(key)-1] == ' ' {
		key = strings.TrimSpace(key)
		if key == "" {
			return fmt.Errorf("pony: key is empty")
		}
	}

	st, h := b.st.stripeAndHash(b.name, key)
	if !st.idx.removeWithHash(b.name, key, h) {
		return storage.ErrNotExist
	}

	now := fastNow()
	if st.bufRing != nil {
		bl, kl := len(b.name), len(key)
		totalSize := int64(recFixedSize + bl + kl)
		bufSlice, recOff, _, wb, isDirect := st.bufRing.writeInline(totalSize, 0)
		st.vol.buildRecordBuf(bufSlice, recDelete, b.name, key, "", nil, now)
		if isDirect {
			st.bufRing.directFlush(bufSlice, recOff)
		} else {
			wb.done()
		}
	} else {
		st.vol.appendRecord(recDelete, b.name, key, "", nil, now)
	}

	return nil
}

func (b *bucket) Copy(ctx context.Context, dstKey string, srcBucket, srcKey string, opts storage.Options) (*storage.Object, error) {
	_ = ctx
	_ = opts

	dstKey = strings.TrimSpace(dstKey)
	srcKey = strings.TrimSpace(srcKey)
	if dstKey == "" || srcKey == "" {
		return nil, fmt.Errorf("pony: key is empty")
	}

	if srcBucket == "" {
		srcBucket = b.name
	}

	srcStripe, srcH := b.st.stripeAndHash(srcBucket, srcKey)
	srcResult, ok := srcStripe.idx.getWithHash(srcBucket, srcKey, srcH)
	if !ok {
		return nil, storage.ErrNotExist
	}

	var srcData []byte
	if srcStripe.bufRing != nil {
		if bufData, inBuf := srcStripe.bufRing.readFromBuffer(srcResult.valOff, srcResult.valSize); inBuf {
			srcData = bufData
		}
	}
	if srcData == nil {
		srcData = srcStripe.vol.readValueSlice(srcResult.valOff, srcResult.valSize)
	}

	now := fastNow()
	nowTime := fastNowTime()
	dstStripe, dstH := b.st.stripeAndHash(b.name, dstKey)
	bl, kl, cl := len(b.name), len(dstKey), len(srcResult.contentType)
	totalSize := int64(recFixedSize+bl+kl+cl) + srcResult.valSize

	var valOff int64
	if dstStripe.bufRing != nil && totalSize <= dstStripe.bufRing.capacity {
		valPosInRecord := 19 + bl + kl + cl
		bufSlice, recOff, vo, wb, isDirect := dstStripe.bufRing.writeInline(totalSize, valPosInRecord)
		valOff = vo
		dstStripe.vol.buildRecordBuf(bufSlice, recPut, b.name, dstKey, srcResult.contentType, srcData, now)
		if isDirect {
			dstStripe.bufRing.directFlush(bufSlice, recOff)
		} else {
			wb.done()
		}
	} else {
		var err error
		_, valOff, err = dstStripe.vol.appendRecord(recPut, b.name, dstKey, srcResult.contentType, srcData, now)
		if err != nil {
			return nil, err
		}
	}

	dstStripe.idx.putWithHash(b.name, dstKey, srcResult.contentType, valOff, srcResult.valSize, now, now, dstH)

	return &storage.Object{
		Bucket:      b.name,
		Key:         dstKey,
		Size:        srcResult.valSize,
		ContentType: srcResult.contentType,
		Created:     nowTime,
		Updated:     nowTime,
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

	// Collect results from all stripes.
	var allResults []listResult
	for i := 0; i < numStripes; i++ {
		results := b.st.stripes[i].idx.list(b.name, prefix)
		allResults = append(allResults, results...)
	}
	// Sort merged results.
	sort.Slice(allResults, func(i, j int) bool {
		return allResults[i].key < allResults[j].key
	})

	// Batch allocate objects: 2 allocs instead of N+1.
	nowTime := fastNowTime()
	_ = nowTime
	filtered := allResults
	if !recursive {
		filtered = filtered[:0]
		for _, r := range allResults {
			rest := strings.TrimPrefix(r.key, prefix)
			rest = strings.TrimPrefix(rest, "/")
			if !strings.Contains(rest, "/") {
				filtered = append(filtered, r)
			}
		}
	}

	objSlice := make([]storage.Object, len(filtered))
	ptrs := make([]*storage.Object, len(filtered))
	for i, r := range filtered {
		objSlice[i] = storage.Object{
			Bucket:      b.name,
			Key:         r.key,
			Size:        r.valSize,
			ContentType: r.contentType,
			Created:     time.Unix(0, r.created),
			Updated:     time.Unix(0, r.updated),
		}
		ptrs[i] = &objSlice[i]
	}

	if offset < 0 {
		offset = 0
	}
	if offset > len(ptrs) {
		offset = len(ptrs)
	}
	ptrs = ptrs[offset:]
	if limit > 0 && limit < len(ptrs) {
		ptrs = ptrs[:limit]
	}

	return &objectIter{objects: ptrs}, nil
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
	for i := 0; i < numStripes; i++ {
		r, ok := d.b.st.stripes[i].idx.firstMatch(d.b.name, prefix)
		if ok {
			return &storage.Object{
				Bucket:  d.b.name,
				Key:     d.path,
				IsDir:   true,
				Created: time.Unix(0, r.created),
				Updated: time.Unix(0, r.updated),
			}, nil
		}
	}
	return nil, storage.ErrNotExist
}

func (d *dir) List(ctx context.Context, limit, offset int, opts storage.Options) (storage.ObjectIter, error) {
	_ = ctx
	_ = opts
	prefix := d.path
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	var allResults []listResult
	for i := 0; i < numStripes; i++ {
		results := d.b.st.stripes[i].idx.list(d.b.name, prefix)
		allResults = append(allResults, results...)
	}
	sort.Slice(allResults, func(i, j int) bool {
		return allResults[i].key < allResults[j].key
	})

	var objs []*storage.Object
	for _, r := range allResults {
		rest := strings.TrimPrefix(r.key, prefix)
		if strings.Contains(rest, "/") {
			continue
		}
		objs = append(objs, &storage.Object{
			Bucket:      d.b.name,
			Key:         r.key,
			Size:        r.valSize,
			ContentType: r.contentType,
			Created:     time.Unix(0, r.created),
			Updated:     time.Unix(0, r.updated),
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

	found := false
	for i := 0; i < numStripes; i++ {
		results := d.b.st.stripes[i].idx.list(d.b.name, prefix)
		if len(results) > 0 {
			found = true
		}
		for _, r := range results {
			if !recursive {
				rest := strings.TrimPrefix(r.key, prefix)
				if strings.Contains(rest, "/") {
					continue
				}
			}
			d.b.st.stripes[i].idx.remove(d.b.name, r.key)
		}
	}

	if !found {
		return storage.ErrNotExist
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

	found := false
	for i := 0; i < numStripes; i++ {
		results := d.b.st.stripes[i].idx.list(d.b.name, srcPrefix)
		if len(results) > 0 {
			found = true
		}
		for _, r := range results {
			rel := strings.TrimPrefix(r.key, srcPrefix)
			newKey := dstPrefix + rel

			// Route new key to its destination stripe.
			dstStripe := d.b.st.stripeFor(d.b.name, newKey)
			dstStripe.idx.put(d.b.name, newKey, r.contentType, r.valOff, r.valSize, r.created, r.updated)
			d.b.st.stripes[i].idx.remove(d.b.name, r.key)
		}
	}

	if !found {
		return nil, storage.ErrNotExist
	}

	return &dir{b: d.b, path: strings.Trim(dstPath, "/")}, nil
}

// mmapReader is an io.ReadCloser over a mmap'd slice.
type mmapReader struct {
	data []byte
	pos  int
}

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
