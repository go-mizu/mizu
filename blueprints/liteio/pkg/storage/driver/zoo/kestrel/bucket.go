package kestrel

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/liteio-dev/liteio/pkg/storage"
)

// ---------------------------------------------------------------------------
// Reader pool
// ---------------------------------------------------------------------------

type dataReader struct {
	data []byte
	pos  int
}

var readerPool = sync.Pool{New: func() any { return &dataReader{} }}

func acquireReader(data []byte) *dataReader {
	r := readerPool.Get().(*dataReader)
	r.data = data
	r.pos = 0
	return r
}

func (r *dataReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func (r *dataReader) WriteTo(w io.Writer) (int64, error) {
	if r.pos >= len(r.data) {
		return 0, nil
	}
	n, err := w.Write(r.data[r.pos:])
	r.pos += n
	return int64(n), err
}

func (r *dataReader) Close() error {
	r.data = nil
	readerPool.Put(r)
	return nil
}

// ---------------------------------------------------------------------------
// Bucket
// ---------------------------------------------------------------------------

type bucket struct {
	st   *store
	name string
}

var (
	_ storage.Bucket         = (*bucket)(nil)
	_ storage.HasDirectories = (*bucket)(nil)
	_ storage.HasMultipart   = (*bucket)(nil)
)

func (b *bucket) Name() string              { return b.name }
func (b *bucket) Features() storage.Features { return b.st.Features() }

func (b *bucket) Info(_ context.Context) (*storage.BucketInfo, error) {
	b.st.mu.RLock()
	created, ok := b.st.storBkts[b.name]
	b.st.mu.RUnlock()
	if !ok {
		return nil, storage.ErrNotExist
	}
	return &storage.BucketInfo{Name: b.name, CreatedAt: created}, nil
}

func (b *bucket) Write(_ context.Context, key string, src io.Reader, size int64, contentType string, _ storage.Options) (*storage.Object, error) {
	if key == "" {
		return nil, fmt.Errorf("kestrel: key is empty")
	}

	now := fastNow()

	// Fast path: known size — allocate from chunk pool, read directly into it.
	if size >= 0 {
		var val []byte
		valLen := 0
		if size > 0 {
			val = allocValue(int(size))
			nr, err := io.ReadFull(src, val)
			if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
				return nil, fmt.Errorf("kestrel: read value: %w", err)
			}
			valLen = nr
			val = val[:nr]
		}
		b.st.hotPut(b.name, key, val, contentType, int64(valLen), now, now)

		return &storage.Object{
			Bucket: b.name, Key: key, Size: int64(valLen), ContentType: contentType,
			Created: time.Unix(0, now), Updated: time.Unix(0, now),
		}, nil
	}

	// Slow path: unknown size — buffer first, then copy to chunk pool.
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, src); err != nil {
		return nil, fmt.Errorf("kestrel: read value: %w", err)
	}
	data := buf.Bytes()
	val := allocValue(len(data))
	copy(val, data)
	b.st.hotPut(b.name, key, val, contentType, int64(len(data)), now, now)

	return &storage.Object{
		Bucket: b.name, Key: key, Size: int64(len(data)), ContentType: contentType,
		Created: time.Unix(0, now), Updated: time.Unix(0, now),
	}, nil
}

func (b *bucket) Open(_ context.Context, key string, offset, length int64, _ storage.Options) (io.ReadCloser, *storage.Object, error) {
	if key == "" {
		return nil, nil, fmt.Errorf("kestrel: key is empty")
	}

	value, ct, sz, created, updated, ok := b.st.hotGet(b.name, key)
	if !ok {
		return nil, nil, storage.ErrNotExist
	}

	obj := &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        sz,
		ContentType: ct,
		Created:     time.Unix(0, created),
		Updated:     time.Unix(0, updated),
	}

	data := value
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
	return acquireReader(data[offset:end]), obj, nil
}

func (b *bucket) Stat(_ context.Context, key string, _ storage.Options) (*storage.Object, error) {
	if key == "" {
		return nil, fmt.Errorf("kestrel: key is empty")
	}

	if strings.HasSuffix(key, "/") {
		b.st.syncIndex()
		keys := b.st.keyIdx.list(b.name, key)
		if len(keys) == 0 {
			return nil, storage.ErrNotExist
		}
		_, _, _, created, updated, ok := b.st.hotGet(b.name, keys[0])
		if ok {
			return &storage.Object{
				Bucket:  b.name,
				Key:     strings.TrimSuffix(key, "/"),
				IsDir:   true,
				Created: time.Unix(0, created),
				Updated: time.Unix(0, updated),
			}, nil
		}
		return &storage.Object{Bucket: b.name, Key: strings.TrimSuffix(key, "/"), IsDir: true}, nil
	}

	_, ct, sz, created, updated, ok := b.st.hotGet(b.name, key)
	if !ok {
		return nil, storage.ErrNotExist
	}
	return &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        sz,
		ContentType: ct,
		Created:     time.Unix(0, created),
		Updated:     time.Unix(0, updated),
	}, nil
}

func (b *bucket) Delete(_ context.Context, key string, _ storage.Options) error {
	if key == "" {
		return fmt.Errorf("kestrel: key is empty")
	}
	if !b.st.hotDelete(b.name, key) {
		return storage.ErrNotExist
	}
	return nil
}

func (b *bucket) Copy(_ context.Context, dstKey string, srcBucket, srcKey string, _ storage.Options) (*storage.Object, error) {
	if dstKey == "" || srcKey == "" {
		return nil, fmt.Errorf("kestrel: key is empty")
	}
	if srcBucket == "" {
		srcBucket = b.name
	}
	value, ct, sz, _, _, ok := b.st.hotGet(srcBucket, srcKey)
	if !ok {
		return nil, storage.ErrNotExist
	}

	now := fastNow()
	var val []byte
	if len(value) > 0 {
		val = allocValue(len(value))
		copy(val, value)
	}

	b.st.hotPut(b.name, dstKey, val, ct, sz, now, now)

	return &storage.Object{
		Bucket: b.name, Key: dstKey, Size: sz, ContentType: ct,
		Created: time.Unix(0, now), Updated: time.Unix(0, now),
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

func (b *bucket) List(_ context.Context, prefix string, limit, offset int, opts storage.Options) (storage.ObjectIter, error) {
	prefix = strings.TrimSpace(prefix)
	recursive := true
	if opts != nil {
		if v, ok := opts["recursive"].(bool); ok {
			recursive = v
		}
	}

	results := b.st.listKeys(b.name, prefix)
	if !recursive {
		var filtered []*storage.Object
		for _, obj := range results {
			rest := strings.TrimPrefix(obj.Key, prefix)
			rest = strings.TrimPrefix(rest, "/")
			if strings.Contains(rest, "/") {
				continue
			}
			filtered = append(filtered, obj)
		}
		results = filtered
	}

	if offset < 0 {
		offset = 0
	}
	if offset > len(results) {
		offset = len(results)
	}
	results = results[offset:]
	if limit > 0 && limit < len(results) {
		results = results[:limit]
	}
	return &objectIter{list: results}, nil
}

func (b *bucket) SignedURL(_ context.Context, _ string, _ string, _ time.Duration, _ storage.Options) (string, error) {
	return "", storage.ErrUnsupported
}

// ---------------------------------------------------------------------------
// Directory support
// ---------------------------------------------------------------------------

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
	keys := d.b.st.keyIdx.list(d.b.name, prefix)
	if len(keys) == 0 {
		return nil, storage.ErrNotExist
	}
	var created, updated time.Time
	_, _, _, c, u, ok := d.b.st.hotGet(d.b.name, keys[0])
	if ok {
		created = time.Unix(0, c)
		updated = time.Unix(0, u)
	}
	return &storage.Object{Bucket: d.b.name, Key: d.path, IsDir: true, Created: created, Updated: updated}, nil
}

func (d *dir) List(_ context.Context, limit, offset int, _ storage.Options) (storage.ObjectIter, error) {
	prefix := d.path
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	results := d.b.st.listKeys(d.b.name, prefix)
	var objs []*storage.Object
	for _, r := range results {
		rest := strings.TrimPrefix(r.Key, prefix)
		if strings.Contains(rest, "/") {
			continue
		}
		objs = append(objs, r)
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
	return &objectIter{list: objs}, nil
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
	keys := d.b.st.keyIdx.list(d.b.name, prefix)
	if len(keys) == 0 {
		return storage.ErrNotExist
	}
	for _, key := range keys {
		if !recursive {
			rest := strings.TrimPrefix(key, prefix)
			if strings.Contains(rest, "/") {
				continue
			}
		}
		d.b.st.hotDelete(d.b.name, key)
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
	keys := d.b.st.keyIdx.list(d.b.name, srcPrefix)
	if len(keys) == 0 {
		return nil, storage.ErrNotExist
	}
	for _, key := range keys {
		rel := strings.TrimPrefix(key, srcPrefix)
		newKey := dstPrefix + rel
		value, ct, sz, created, _, ok := d.b.st.hotGet(d.b.name, key)
		if !ok {
			continue
		}
		now := fastNow()
		valCopy := allocValue(len(value))
		copy(valCopy, value)
		d.b.st.hotPut(d.b.name, newKey, valCopy, ct, sz, created, now)
		d.b.st.hotDelete(d.b.name, key)
	}
	return &dir{b: d.b, path: strings.Trim(dstPath, "/")}, nil
}

// ---------------------------------------------------------------------------
// Iterators
// ---------------------------------------------------------------------------

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

func (it *bucketIter) Close() error { it.list = nil; return nil }

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

func (it *objectIter) Close() error { it.list = nil; return nil }
