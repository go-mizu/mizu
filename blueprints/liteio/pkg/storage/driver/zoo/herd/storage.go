// Package herd implements a high-performance striped object storage driver
// inspired by Facebook Haystack and SeaweedFS, fixing all their limitations.
//
// Architecture: 16-stripe independent partitions with per-stripe bloom filters,
// inline value caching (≤8KB), append-only volumes with mmap reads, and
// lock-free atomic write offset tracking.
//
// Key improvements over Haystack/SeaweedFS:
//   - No master SPOF (embedded mode or client-side consistent hashing)
//   - Per-stripe bloom filters for O(1) negative lookups
//   - Inline values skip volume I/O entirely for small objects
//   - 16 stripes × 256 shards = 4096 total shards (vs horse's 256)
//   - Native range reads with offset/length support
//   - Binary TCP wire protocol for cluster mode
//
// DSN format:
//
//	herd:///path/to/data
//	herd:///path/to/data?stripes=16&sync=none&inline_kb=8
//	herd:///path/to/data?stripes=16&sync=none&inline_kb=8&prealloc=1024&bufsize=8388608
package herd

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
	"unsafe"

	"github.com/liteio-dev/liteio/pkg/storage"
)

func init() {
	storage.Register("herd", &driver{})
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

// unsafePtr converts a byte slice to an unsafe.Pointer for syscalls.
func unsafePtr(b []byte) unsafe.Pointer {
	return unsafe.Pointer(&b[0])
}

// Driver is the exported type alias for the herd driver.
type Driver = driver

type driver struct{}

func (d *driver) Open(ctx context.Context, dsn string) (storage.Storage, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("herd: parse dsn: %w", err)
	}

	q := u.Query()

	// Embedded multi-node: herd:///path?nodes=3
	if q.Has("nodes") {
		return openMultiNode(ctx, u)
	}

	// TCP cluster with gossip: herd:///?seeds=...
	if q.Has("seeds") {
		return openGossipCluster(ctx, u)
	}

	// TCP cluster with static peers: herd:///?peers=...
	if q.Has("peers") {
		return openCluster(ctx, u)
	}

	return openEmbedded(ctx, u)
}

func openEmbedded(_ context.Context, u *url.URL) (*store, error) {
	q := u.Query()
	root := filepath.Clean(u.Path)
	if root == "" || root == "." {
		root = "/tmp/herd-data"
	}

	numStripes := intParam(q, "stripes", 16)
	syncMode := q.Get("sync")
	if syncMode == "" {
		syncMode = "none"
	}
	inlineKB := intParam(q, "inline_kb", 8)
	preallocMB := intParam(q, "prealloc", 1024)
	bufSize := intParam(q, "bufsize", defaultBufSize)

	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("herd: mkdir %q: %w", root, err)
	}

	s := &store{
		root:       root,
		syncMode:   syncMode,
		noSync:     syncMode == "none",
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

// stripe is a fully independent storage partition: own volume, index, bloom, and buffer ring.
type stripe struct {
	id    int
	vol   *volume
	idx   *shardedIndex
	bloom *bloomFilter
	ring  *bufferRing
}

func newStripe(id int, root, syncMode string, preallocBytes, bufSize, inlineMax int64) (*stripe, error) {
	volPath := filepath.Join(root, fmt.Sprintf("stripe_%02d.dat", id))

	vol, err := newVolume(volPath, preallocBytes)
	if err != nil {
		return nil, fmt.Errorf("herd: stripe %d: %w", id, err)
	}

	if syncMode == "none" {
		vol.noCRC = true
	}

	idx := newIndex()
	bloom := newBloomFilter(1 << 20) // 1M expected items per stripe

	// Recover index from volume if volume has data.
	if vol.tail.Load() > headerSize {
		if err := vol.recover(idx, bloom, inlineMax); err != nil {
			vol.close()
			return nil, fmt.Errorf("herd: stripe %d recovery: %w", id, err)
		}
	}

	s := &stripe{
		id:    id,
		vol:   vol,
		idx:   idx,
		bloom: bloom,
	}

	// Buffer ring batches many small writes into one large WriteAt,
	// dramatically improving throughput. Enable for both none and batch modes.
	// Only sync=full skips the ring (needs per-write msync).
	if syncMode != "full" {
		s.ring = newBufferRing(vol, bufSize)
	}

	return s, nil
}

func (s *stripe) close() error {
	if s.ring != nil {
		s.ring.close()
	}
	return s.vol.close()
}

// store implements storage.Storage with striped partitions.
type store struct {
	root       string
	syncMode   string
	noSync     bool // cached syncMode == "none" to avoid string comparison per op
	inlineMax  int64
	numStripes int
	stripes    []*stripe

	mu      sync.RWMutex
	buckets map[string]time.Time

	mp *multipartRegistry
}

var _ storage.Storage = (*store)(nil)

// stripeFor selects a stripe for a bucket+key using fast partial FNV-1a.
// Only hashes last 8 key bytes (where entropy is highest) + first bucket byte.
// 16 stripes don't need a full cryptographic-quality hash.
func (s *store) stripeFor(bucket, key string) *stripe {
	const prime32 = 16777619
	h := uint32(2166136261)
	if len(bucket) > 0 {
		h ^= uint32(bucket[0])
		h *= prime32
	}
	start := len(key) - 8
	if start < 0 {
		start = 0
	}
	for i := start; i < len(key); i++ {
		h ^= uint32(key[i])
		h *= prime32
	}
	return s.stripes[h%uint32(s.numStripes)]
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
	for n := range s.buckets {
		names = append(names, n)
	}
	s.mu.RUnlock()
	sort.Strings(names)

	s.mu.RLock()
	infos := make([]*storage.BucketInfo, 0, len(names))
	for _, n := range names {
		infos = append(infos, &storage.BucketInfo{Name: n, CreatedAt: s.buckets[n]})
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
		return nil, fmt.Errorf("herd: bucket name is empty")
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
		return fmt.Errorf("herd: bucket name is empty")
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
		// Check if any stripe has keys for this bucket.
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
	// Sync BEFORE closing — mmap must still be active for msync.
	if s.syncMode != "none" {
		for _, st := range s.stripes {
			if st != nil {
				st.vol.sync()
			}
		}
	}

	for _, st := range s.stripes {
		if st != nil {
			st.close()
		}
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
	_ storage.HasMultipart   = (*bucket)(nil)
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
		return nil, fmt.Errorf("herd: key is empty")
	}
	if key[0] == ' ' || key[len(key)-1] == ' ' {
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("herd: key is empty")
		}
	}

	now := fastNow()
	stripe := b.st.stripeFor(b.name, key)
	noSync := b.st.noSync

	// INLINE PATH: small known-size values skip volume I/O entirely in sync=none mode.
	// Read directly into the inline buffer — one allocation, zero volume writes.
	if size >= 0 && size <= b.st.inlineMax {
		e := acquireIndexEntry()
		e.size = size
		e.contentType = contentType
		e.created = now
		e.updated = now

		if size > 0 {
			// Single allocation: read directly into inline buffer.
			e.inline = make([]byte, size)
			if _, err := io.ReadFull(src, e.inline); err != nil {
				if err != io.EOF && err != io.ErrUnexpectedEOF {
					releaseIndexEntry(e)
					return nil, fmt.Errorf("herd: read value: %w", err)
				}
			}
		}

		stripe.idx.put(b.name, key, e)

		// Skip volume write in sync=none mode for pure speed.
		// In batch/full mode, write to volume for durability.
		if !noSync && size > 0 {
			if stripe.ring != nil {
				bl, kl, cl := len(b.name), len(key), len(contentType)
				totalSize := int64(recFixedSize+bl+kl+cl) + size
				bufSlice, _, _, wb := stripe.ring.writeInline(totalSize, 19+bl+kl+cl)
				stripe.vol.buildRecordBuf(bufSlice, recPut, b.name, key, contentType, e.inline, now)
				wb.done()
			} else {
				stripe.vol.appendRecord(recPut, b.name, key, contentType, e.inline, now)
			}
		}

		return &storage.Object{
			Bucket: b.name, Key: key, Size: size,
			ContentType: contentType,
			Created:     time.Unix(0, now), Updated: time.Unix(0, now),
		}, nil
	}

	bl, kl, cl := len(b.name), len(key), len(contentType)
	var valOff int64

	if size < 0 {
		// Unknown size: read all first.
		var tmpBuf bytes.Buffer
		if _, err := io.Copy(&tmpBuf, src); err != nil {
			return nil, fmt.Errorf("herd: read value: %w", err)
		}
		data := tmpBuf.Bytes()
		size = int64(len(data))

		// Check if we can inline after reading.
		if size <= b.st.inlineMax {
			e := acquireIndexEntry()
			e.size = size
			e.contentType = contentType
			e.created = now
			e.updated = now
			if size > 0 {
				e.inline = data // reuse the buffer directly
			}
			stripe.idx.put(b.name, key, e)

			if !noSync && size > 0 {
				if stripe.ring != nil {
					ts := int64(recFixedSize+len(b.name)+len(key)+len(contentType)) + size
					bufSlice, _, _, wb := stripe.ring.writeInline(ts, 19+len(b.name)+len(key)+len(contentType))
					stripe.vol.buildRecordBuf(bufSlice, recPut, b.name, key, contentType, data, now)
					wb.done()
				} else {
					stripe.vol.appendRecord(recPut, b.name, key, contentType, data, now)
				}
			}

			return &storage.Object{
				Bucket: b.name, Key: key, Size: size,
				ContentType: contentType,
				Created:     time.Unix(0, now), Updated: time.Unix(0, now),
			}, nil
		}

		totalSize := int64(recFixedSize+bl+kl+cl) + size
		if stripe.ring != nil && totalSize <= stripe.ring.capacity {
			valPosInRecord := 19 + bl + kl + cl
			bufSlice, _, vo, wb := stripe.ring.writeInline(totalSize, valPosInRecord)
			valOff = vo
			stripe.vol.buildRecordBuf(bufSlice, recPut, b.name, key, contentType, data, now)
			wb.done()
		} else {
			var err error
			_, valOff, err = stripe.vol.appendRecord(recPut, b.name, key, contentType, data, now)
			if err != nil {
				return nil, err
			}
		}
	} else if stripe.ring != nil {
		// Large value, buffer ring path.
		totalSize := int64(recFixedSize+bl+kl+cl) + size
		if totalSize > stripe.ring.capacity {
			var err error
			valOff, err = stripe.vol.writeFromReader(recPut, b.name, key, contentType, src, size, now)
			if err != nil {
				return nil, err
			}
		} else {
			valPosInRecord := 19 + bl + kl + cl
			bufSlice, _, vo, wb := stripe.ring.writeInline(totalSize, valPosInRecord)
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
						return nil, fmt.Errorf("herd: read value: %w", err)
					}
				}
			}
			pos += int(size)

			binary.LittleEndian.PutUint64(bufSlice[pos:], uint64(now))

			if !stripe.vol.noCRC {
				checksum := crc32.Checksum(bufSlice[5:], stripe.vol.crcTable)
				binary.LittleEndian.PutUint32(bufSlice[1:5], checksum)
			}
			wb.done()
		}
	} else {
		// Direct volume path (sync=batch or sync=full).
		var err error
		valOff, err = stripe.vol.writeFromReader(recPut, b.name, key, contentType, src, size, now)
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
	stripe.idx.put(b.name, key, e)

	if b.st.syncMode == "full" {
		stripe.vol.sync()
	}

	return &storage.Object{
		Bucket: b.name, Key: key, Size: size,
		ContentType: contentType,
		Created:     time.Unix(0, now), Updated: time.Unix(0, now),
	}, nil
}

func (b *bucket) Open(_ context.Context, key string, offset, length int64, _ storage.Options) (io.ReadCloser, *storage.Object, error) {
	if key == "" {
		return nil, nil, fmt.Errorf("herd: key is empty")
	}
	if key[0] == ' ' || key[len(key)-1] == ' ' {
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, nil, fmt.Errorf("herd: key is empty")
		}
	}

	stripe := b.st.stripeFor(b.name, key)

	e, ok := stripe.idx.get(b.name, key)
	if !ok {
		return nil, nil, storage.ErrNotExist
	}

	// Read value data: inline → buffer ring → mmap.
	var data []byte
	if e.inline != nil {
		data = e.inline
	} else {
		if stripe.ring != nil {
			if bufData, inBuf := stripe.ring.readFromBuffer(e.valueOffset, e.size); inBuf {
				data = bufData
			}
		}
		if data == nil {
			data = stripe.vol.readValueSlice(e.valueOffset, e.size)
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
		Bucket: b.name, Key: key, Size: e.size,
		ContentType: e.contentType,
		Created:     time.Unix(0, e.created), Updated: time.Unix(0, e.updated),
	}

	r := acquireMmapReader(slice)
	return r, obj, nil
}

func (b *bucket) Stat(_ context.Context, key string, _ storage.Options) (*storage.Object, error) {
	if key == "" {
		return nil, fmt.Errorf("herd: key is empty")
	}
	if key[0] == ' ' || key[len(key)-1] == ' ' {
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("herd: key is empty")
		}
	}

	// Check for directory stat.
	if strings.HasSuffix(key, "/") {
		// Fan-out to all stripes for directory listing.
		for _, st := range b.st.stripes {
			results := st.idx.list(b.name, key)
			if len(results) > 0 {
				return &storage.Object{
					Bucket: b.name, Key: strings.TrimSuffix(key, "/"),
					IsDir: true,
					Created: time.Unix(0, results[0].entry.created),
					Updated: time.Unix(0, results[0].entry.updated),
				}, nil
			}
		}
		return nil, storage.ErrNotExist
	}

	stripe := b.st.stripeFor(b.name, key)

	e, ok := stripe.idx.get(b.name, key)
	if !ok {
		return nil, storage.ErrNotExist
	}

	return &storage.Object{
		Bucket: b.name, Key: key, Size: e.size,
		ContentType: e.contentType,
		Created:     time.Unix(0, e.created), Updated: time.Unix(0, e.updated),
	}, nil
}

func (b *bucket) Delete(_ context.Context, key string, _ storage.Options) error {
	if key == "" {
		return fmt.Errorf("herd: key is empty")
	}
	if key[0] == ' ' || key[len(key)-1] == ' ' {
		key = strings.TrimSpace(key)
		if key == "" {
			return fmt.Errorf("herd: key is empty")
		}
	}

	stripe := b.st.stripeFor(b.name, key)

	if !stripe.idx.remove(b.name, key) {
		return storage.ErrNotExist
	}

	// In sync=none mode, skip tombstone write for speed.
	// In batch/full mode, append delete tombstone for durability.
	if !b.st.noSync {
		now := fastNow()
		if stripe.ring != nil {
			bl, kl := len(b.name), len(key)
			totalSize := int64(recFixedSize + bl + kl)
			bufSlice, _, _, wb := stripe.ring.writeInline(totalSize, 0)
			stripe.vol.buildRecordBuf(bufSlice, recDelete, b.name, key, "", nil, now)
			wb.done()
		} else {
			stripe.vol.appendRecord(recDelete, b.name, key, "", nil, now)
		}
	}

	return nil
}

func (b *bucket) Copy(_ context.Context, dstKey string, srcBucket, srcKey string, _ storage.Options) (*storage.Object, error) {
	dstKey = strings.TrimSpace(dstKey)
	srcKey = strings.TrimSpace(srcKey)
	if dstKey == "" || srcKey == "" {
		return nil, fmt.Errorf("herd: key is empty")
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
	dstStripe := b.st.stripeFor(b.name, dstKey)
	now := fastNow()

	e := acquireIndexEntry()
	e.size = srcEntry.size
	e.contentType = srcEntry.contentType
	e.created = now
	e.updated = now

	if srcEntry.size <= b.st.inlineMax && srcEntry.size > 0 {
		e.inline = make([]byte, srcEntry.size)
		copy(e.inline, srcData)
	}

	// Write to volume for durability.
	bl, kl, cl := len(b.name), len(dstKey), len(srcEntry.contentType)
	totalSize := int64(recFixedSize+bl+kl+cl) + srcEntry.size

	if dstStripe.ring != nil && totalSize <= dstStripe.ring.capacity {
		valPosInRecord := 19 + bl + kl + cl
		bufSlice, _, vo, wb := dstStripe.ring.writeInline(totalSize, valPosInRecord)
		if e.inline == nil {
			e.valueOffset = vo
		}
		dstStripe.vol.buildRecordBuf(bufSlice, recPut, b.name, dstKey, srcEntry.contentType, srcData, now)
		wb.done()
	} else {
		_, valOff, err := dstStripe.vol.appendRecord(recPut, b.name, dstKey, srcEntry.contentType, srcData, now)
		if err != nil {
			return nil, err
		}
		if e.inline == nil {
			e.valueOffset = valOff
		}
	}

	dstStripe.idx.put(b.name, dstKey, e)

	return &storage.Object{
		Bucket: b.name, Key: dstKey, Size: srcEntry.size,
		ContentType: srcEntry.contentType,
		Created:     time.Unix(0, now), Updated: time.Unix(0, now),
	}, nil
}

func (b *bucket) Move(_ context.Context, dstKey string, srcBucket, srcKey string, opts storage.Options) (*storage.Object, error) {
	obj, err := b.Copy(context.Background(), dstKey, srcBucket, srcKey, opts)
	if err != nil {
		return nil, err
	}
	if srcBucket == "" {
		srcBucket = b.name
	}
	sb := b.st.Bucket(srcBucket)
	if err := sb.Delete(context.Background(), srcKey, nil); err != nil {
		return nil, err
	}
	return obj, nil
}

// listAll returns all objects from all stripes for a given prefix.
// Used by NodeServer for cluster list operations.
func (b *bucket) listAll(prefix string) []listResult {
	var all []listResult
	for _, st := range b.st.stripes {
		all = append(all, st.idx.list(b.name, prefix)...)
	}
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

	// Fan-out list to all stripes and merge.
	var all []*storage.Object
	for _, st := range b.st.stripes {
		results := st.idx.list(b.name, prefix)
		for _, r := range results {
			if !recursive {
				rest := strings.TrimPrefix(r.key, prefix)
				rest = strings.TrimPrefix(rest, "/")
				if strings.Contains(rest, "/") {
					continue
				}
			}
			all = append(all, &storage.Object{
				Bucket: b.name, Key: r.key, Size: r.entry.size,
				ContentType: r.entry.contentType,
				Created:     time.Unix(0, r.entry.created),
				Updated:     time.Unix(0, r.entry.updated),
			})
		}
	}

	// Sort merged results.
	sort.Slice(all, func(i, j int) bool { return all[i].Key < all[j].Key })

	if offset < 0 {
		offset = 0
	}
	if offset > len(all) {
		offset = len(all)
	}
	all = all[offset:]
	if limit > 0 && limit < len(all) {
		all = all[:limit]
	}

	return &objectIter{objects: all}, nil
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
	for _, st := range d.b.st.stripes {
		results := st.idx.list(d.b.name, prefix)
		if len(results) > 0 {
			return &storage.Object{
				Bucket: d.b.name, Key: d.path, IsDir: true,
				Created: time.Unix(0, results[0].entry.created),
				Updated: time.Unix(0, results[0].entry.updated),
			}, nil
		}
	}
	return nil, storage.ErrNotExist
}

func (d *dir) List(_ context.Context, limit, offset int, _ storage.Options) (storage.ObjectIter, error) {
	prefix := d.path
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	var objs []*storage.Object
	for _, st := range d.b.st.stripes {
		results := st.idx.list(d.b.name, prefix)
		for _, r := range results {
			rest := strings.TrimPrefix(r.key, prefix)
			if strings.Contains(rest, "/") {
				continue
			}
			objs = append(objs, &storage.Object{
				Bucket: d.b.name, Key: r.key, Size: r.entry.size,
				ContentType: r.entry.contentType,
				Created:     time.Unix(0, r.entry.created),
				Updated:     time.Unix(0, r.entry.updated),
			})
		}
	}

	sort.Slice(objs, func(i, j int) bool { return objs[i].Key < objs[j].Key })

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

	found := false
	for _, st := range d.b.st.stripes {
		results := st.idx.list(d.b.name, prefix)
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
			st.idx.remove(d.b.name, r.key)
		}
	}
	if !found {
		return storage.ErrNotExist
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

	found := false
	for _, st := range d.b.st.stripes {
		results := st.idx.list(d.b.name, srcPrefix)
		if len(results) > 0 {
			found = true
		}
		for _, r := range results {
			rel := strings.TrimPrefix(r.key, srcPrefix)
			newKey := dstPrefix + rel

			// Get destination stripe.
			dstStripe := d.b.st.stripeFor(d.b.name, newKey)
			dstStripe.idx.put(d.b.name, newKey, &indexEntry{
				valueOffset: r.entry.valueOffset,
				size:        r.entry.size,
				contentType: r.entry.contentType,
				created:     r.entry.created,
				updated:     r.entry.updated,
				inline:      r.entry.inline,
			})
			st.idx.remove(d.b.name, r.key)
		}
	}
	if !found {
		return nil, storage.ErrNotExist
	}

	return &dir{b: d.b, path: strings.Trim(dstPath, "/")}, nil
}

// mmapReader is an io.ReadCloser over a mmap'd or inline slice.
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

// Helper to parse integer query parameters.
func intParam(q url.Values, key string, def int) int {
	v := q.Get(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return def
	}
	return n
}
