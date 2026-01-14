package memdriver

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-mizu/blueprints/drive/lib/storage"
)

// DSN format:
//
//   mem://
//   mem://name
//   mem://name?bucket=default
//
// Notes:
//
//   - Each Open creates a new isolated in memory store (no global sharing).
//   - Host (name) is currently ignored, reserved for future sharing.
//   - "bucket" query param sets default bucket name for Bucket("").

func init() {
	storage.Register("mem", &driver{})
}

type driver struct{}

func (d *driver) Open(ctx context.Context, dsn string) (storage.Storage, error) {
	_ = ctx

	u, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("mem: parse dsn: %w", err)
	}
	if u.Scheme != "mem" && u.Scheme != "" {
		return nil, fmt.Errorf("mem: unexpected scheme %q", u.Scheme)
	}

	defaultBucket := strings.TrimSpace(u.Query().Get("bucket"))

	st := &store{
		defaultBucket: defaultBucket,
		buckets:       make(map[string]*bucket),
		features:      defaultFeatures(),
	}
	return st, nil
}

// store implements storage.Storage fully in memory.
type store struct {
	mu            sync.RWMutex
	defaultBucket string
	buckets       map[string]*bucket
	features      storage.Features
}

var _ storage.Storage = (*store)(nil)

func (s *store) Bucket(name string) storage.Bucket {
	s.mu.Lock()
	defer s.mu.Unlock()

	if name == "" {
		name = s.defaultBucket
	}
	if name == "" {
		name = "default"
	}
	b, ok := s.buckets[name]
	if !ok {
		now := time.Now()
		b = &bucket{
			st:        s,
			name:      name,
			obj:       make(map[string]*entry),
			created:   now,
			mpUploads: make(map[string]*multipartUpload),
		}
		s.buckets[name] = b
	}
	return b
}

func (s *store) Buckets(ctx context.Context, limit, offset int, opts storage.Options) (storage.BucketIter, error) {
	_ = ctx
	_ = opts

	s.mu.RLock()
	defer s.mu.RUnlock()

	names := make([]string, 0, len(s.buckets))
	for name := range s.buckets {
		names = append(names, name)
	}
	sort.Strings(names)

	infos := make([]*storage.BucketInfo, 0, len(names))
	for _, name := range names {
		b := s.buckets[name]
		infos = append(infos, &storage.BucketInfo{
			Name:      name,
			CreatedAt: b.created,
			Public:    false,
			Metadata:  map[string]string{},
		})
	}

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
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	_ = opts

	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("mem: bucket name is empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.buckets[name]; ok {
		return nil, storage.ErrExist
	}
	now := time.Now()
	b := &bucket{
		st:        s,
		name:      name,
		obj:       make(map[string]*entry),
		created:   now,
		mpUploads: make(map[string]*multipartUpload),
	}
	s.buckets[name] = b

	return &storage.BucketInfo{
		Name:      name,
		CreatedAt: now,
		Public:    false,
		Metadata:  map[string]string{},
	}, nil
}

func (s *store) DeleteBucket(ctx context.Context, name string, opts storage.Options) error {
	_ = ctx

	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("mem: bucket name is empty")
	}

	force, _ := opts["force"].(bool)

	s.mu.Lock()
	defer s.mu.Unlock()

	b, ok := s.buckets[name]
	if !ok {
		return storage.ErrNotExist
	}
	if !force && len(b.obj) > 0 {
		return storage.ErrPermission
	}

	delete(s.buckets, name)
	return nil
}

func (s *store) Features() storage.Features {
	return cloneFeatures(s.features)
}

func (s *store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.buckets = make(map[string]*bucket)
	return nil
}

// bucket implements storage.Bucket for in memory bucket.
type bucket struct {
	st   *store
	name string

	mu      sync.RWMutex
	obj     map[string]*entry
	created time.Time

	// multipart state
	mpMu      sync.RWMutex
	mpUploads map[string]*multipartUpload
}

var (
	_ storage.Bucket         = (*bucket)(nil)
	_ storage.HasDirectories = (*bucket)(nil)
)

// entry holds object metadata and content in memory.
type entry struct {
	obj  storage.Object
	data []byte
}

func (b *bucket) Name() string {
	return b.name
}

func (b *bucket) Features() storage.Features {
	return cloneFeatures(b.st.features)
}

func (b *bucket) Info(ctx context.Context) (*storage.BucketInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	b.st.mu.RLock()
	defer b.st.mu.RUnlock()

	// Check if bucket still exists in the store
	if _, ok := b.st.buckets[b.name]; !ok {
		return nil, storage.ErrNotExist
	}

	return &storage.BucketInfo{
		Name:      b.name,
		CreatedAt: b.created,
		Public:    false,
		Metadata:  map[string]string{},
	}, nil
}

func (b *bucket) Write(ctx context.Context, key string, src io.Reader, size int64, contentType string, opts storage.Options) (*storage.Object, error) {
	_ = ctx

	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("mem: key is empty")
	}

	var buf bytes.Buffer
	if size >= 0 {
		if _, err := io.CopyN(&buf, src, size); err != nil && err != io.EOF {
			return nil, err
		}
	} else {
		if _, err := io.Copy(&buf, src); err != nil {
			return nil, err
		}
	}
	data := buf.Bytes()
	now := time.Now()

	meta := extractMetadata(opts)

	b.mu.Lock()
	defer b.mu.Unlock()

	e, ok := b.obj[key]
	if ok {
		// Preserve Created, update Updated.
		e.data = data
		e.obj.Size = int64(len(data))
		e.obj.ContentType = contentType
		e.obj.Updated = now
		e.obj.Metadata = cloneStringMap(meta)
	} else {
		e = &entry{
			obj: storage.Object{
				Bucket:      b.name,
				Key:         key,
				Size:        int64(len(data)),
				ContentType: contentType,
				Created:     now,
				Updated:     now,
				Hash:        nil,
				Metadata:    cloneStringMap(meta),
				IsDir:       false,
			},
			data: data,
		}
		b.obj[key] = e
	}

	objCopy := e.obj
	return &objCopy, nil
}

func (b *bucket) Open(ctx context.Context, key string, offset, length int64, opts storage.Options) (io.ReadCloser, *storage.Object, error) {
	_ = ctx
	_ = opts

	key = strings.TrimSpace(key)
	if key == "" {
		return nil, nil, fmt.Errorf("mem: key is empty")
	}

	b.mu.RLock()
	e, ok := b.obj[key]
	b.mu.RUnlock()
	if !ok {
		return nil, nil, storage.ErrNotExist
	}

	data := e.data
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

	rc := io.NopCloser(bytes.NewReader(slice))
	objCopy := e.obj
	return rc, &objCopy, nil
}

func (b *bucket) Stat(ctx context.Context, key string, opts storage.Options) (*storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	_ = opts

	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("mem: key is empty")
	}

	// Check if key ends with "/" indicating a directory request
	if strings.HasSuffix(key, "/") {
		// Look for any objects with this prefix
		prefix := key
		b.mu.RLock()
		defer b.mu.RUnlock()

		var created, updated time.Time
		found := false

		for k, e := range b.obj {
			if strings.HasPrefix(k, prefix) {
				if !found {
					created = e.obj.Created
					updated = e.obj.Updated
					found = true
				} else {
					if e.obj.Created.Before(created) {
						created = e.obj.Created
					}
					if e.obj.Updated.After(updated) {
						updated = e.obj.Updated
					}
				}
			}
		}

		if !found {
			return nil, storage.ErrNotExist
		}

		return &storage.Object{
			Bucket:   b.name,
			Key:      strings.TrimSuffix(key, "/"),
			Size:     0,
			IsDir:    true,
			Created:  created,
			Updated:  updated,
			Metadata: map[string]string{},
		}, nil
	}

	b.mu.RLock()
	e, ok := b.obj[key]
	b.mu.RUnlock()
	if !ok {
		return nil, storage.ErrNotExist
	}

	objCopy := e.obj
	return &objCopy, nil
}

func (b *bucket) Delete(ctx context.Context, key string, opts storage.Options) error {
	_ = ctx
	_ = opts

	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("mem: key is empty")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if _, ok := b.obj[key]; !ok {
		return storage.ErrNotExist
	}
	delete(b.obj, key)
	return nil
}

func (b *bucket) Copy(ctx context.Context, dstKey string, srcBucket, srcKey string, opts storage.Options) (*storage.Object, error) {
	_ = ctx
	_ = opts

	dstKey = strings.TrimSpace(dstKey)
	srcKey = strings.TrimSpace(srcKey)
	if dstKey == "" {
		return nil, fmt.Errorf("mem: dstKey is empty")
	}
	if srcKey == "" {
		return nil, fmt.Errorf("mem: srcKey is empty")
	}

	if srcBucket == "" {
		srcBucket = b.name
	}

	var srcB *bucket
	if srcBucket == b.name {
		srcB = b
	} else {
		// cross bucket copy
		sb := b.st.Bucket(srcBucket)
		var ok bool
		srcB, ok = sb.(*bucket)
		if !ok {
			return nil, fmt.Errorf("mem: unexpected bucket type for %q", srcBucket)
		}
	}

	srcB.mu.RLock()
	e, ok := srcB.obj[srcKey]
	srcB.mu.RUnlock()
	if !ok {
		return nil, storage.ErrNotExist
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	dataCopy := make([]byte, len(e.data))
	copy(dataCopy, e.data)

	entry := &entry{
		obj: storage.Object{
			Bucket:      b.name,
			Key:         dstKey,
			Size:        int64(len(dataCopy)),
			ContentType: e.obj.ContentType,
			Created:     now,
			Updated:     now,
			Hash:        nil,
			Metadata:    cloneStringMap(e.obj.Metadata),
			IsDir:       false,
		},
		data: dataCopy,
	}
	b.obj[dstKey] = entry

	objCopy := entry.obj
	return &objCopy, nil
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
	if v, ok := opts["recursive"].(bool); ok {
		recursive = v
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	keys := make([]string, 0, len(b.obj))
	for k := range b.obj {
		if prefix != "" && !strings.HasPrefix(k, prefix) {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	objs := make([]*storage.Object, 0, len(keys))
	for _, k := range keys {
		if !recursive {
			rest := strings.TrimPrefix(k, prefix)
			rest = strings.TrimPrefix(rest, "/")
			if i := strings.Index(rest, "/"); i >= 0 {
				// subdir, skip in non recursive
				continue
			}
		}
		e := b.obj[k]
		objCopy := e.obj
		objs = append(objs, &objCopy)
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

// SignedURL returns ErrUnsupported for mem backend.
func (b *bucket) SignedURL(ctx context.Context, key string, method string, expires time.Duration, opts storage.Options) (string, error) {
	_ = ctx
	_ = key
	_ = method
	_ = expires
	_ = opts

	return "", storage.ErrUnsupported
}

// Directory: prefix based directories over keys.

func (b *bucket) Directory(p string) storage.Directory {
	clean := strings.Trim(p, "/")
	return &dir{
		b:    b,
		path: clean,
	}
}

type dir struct {
	b    *bucket
	path string
}

var _ storage.Directory = (*dir)(nil)

func (d *dir) Bucket() storage.Bucket {
	return d.b
}

func (d *dir) Path() string {
	return d.path
}

func (d *dir) Info(ctx context.Context) (*storage.Object, error) {
	_ = ctx

	prefix := d.path
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	d.b.mu.RLock()
	defer d.b.mu.RUnlock()

	found := false
	var created, updated time.Time

	for k, e := range d.b.obj {
		if prefix != "" {
			if !strings.HasPrefix(k, prefix) {
				continue
			}
		} else {
			// root directory exists if bucket has any object
			if k == "" {
				continue
			}
		}
		if !found {
			created = e.obj.Created
			updated = e.obj.Updated
			found = true
		} else {
			if e.obj.Created.Before(created) {
				created = e.obj.Created
			}
			if e.obj.Updated.After(updated) {
				updated = e.obj.Updated
			}
		}
	}

	if !found {
		return nil, storage.ErrNotExist
	}

	return &storage.Object{
		Bucket:   d.b.name,
		Key:      d.path,
		Size:     0,
		IsDir:    true,
		Created:  created,
		Updated:  updated,
		Metadata: map[string]string{},
	}, nil
}

func (d *dir) List(ctx context.Context, limit, offset int, opts storage.Options) (storage.ObjectIter, error) {
	_ = ctx
	_ = opts

	prefix := d.path
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	d.b.mu.RLock()
	defer d.b.mu.RUnlock()

	var keys []string
	for k := range d.b.obj {
		if prefix != "" && !strings.HasPrefix(k, prefix) {
			continue
		}
		rest := strings.TrimPrefix(k, prefix)
		if i := strings.Index(rest, "/"); i >= 0 {
			// has deeper directory, skip
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	objs := make([]*storage.Object, 0, len(keys))
	for _, k := range keys {
		e := d.b.obj[k]
		objCopy := e.obj
		objs = append(objs, &objCopy)
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

	recursive, _ := opts["recursive"].(bool)

	prefix := d.path
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	d.b.mu.Lock()
	defer d.b.mu.Unlock()

	if !recursive {
		// Non recursive delete: only delete objects directly under this directory.
		deleted := 0
		for k := range d.b.obj {
			if prefix != "" && !strings.HasPrefix(k, prefix) {
				continue
			}
			rest := strings.TrimPrefix(k, prefix)
			if strings.Contains(rest, "/") {
				continue
			}
			delete(d.b.obj, k)
			deleted++
		}
		if deleted == 0 {
			return storage.ErrNotExist
		}
		return nil
	}

	// Recursive delete: remove all keys with prefix.
	found := false
	for k := range d.b.obj {
		if prefix == "" {
			// root directory recursive delete: clear bucket
			delete(d.b.obj, k)
			found = true
			continue
		}
		if strings.HasPrefix(k, prefix) {
			delete(d.b.obj, k)
			found = true
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

	d.b.mu.Lock()
	defer d.b.mu.Unlock()

	newObjects := make(map[string]*entry)

	for k, e := range d.b.obj {
		if srcPrefix == "" {
			// moving root, treat all keys as under prefix
			rel := k
			newKey := dstPrefix + rel
			newE := &entry{
				obj:  e.obj,
				data: make([]byte, len(e.data)),
			}
			copy(newE.data, e.data)
			newE.obj.Key = newKey
			newObjects[newKey] = newE
			continue
		}
		if !strings.HasPrefix(k, srcPrefix) {
			continue
		}
		rel := strings.TrimPrefix(k, srcPrefix)
		newKey := dstPrefix + rel
		newE := &entry{
			obj:  e.obj,
			data: make([]byte, len(e.data)),
		}
		copy(newE.data, e.data)
		newE.obj.Key = newKey
		newObjects[newKey] = newE
	}

	if len(newObjects) == 0 {
		return nil, storage.ErrNotExist
	}

	// Remove old keys and insert new ones.
	for k := range d.b.obj {
		if srcPrefix == "" {
			delete(d.b.obj, k)
			continue
		}
		if strings.HasPrefix(k, srcPrefix) {
			delete(d.b.obj, k)
		}
	}
	for k, e := range newObjects {
		d.b.obj[k] = e
	}

	return &dir{
		b:    d.b,
		path: strings.Trim(dstPath, "/"),
	}, nil
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

// helpers.

func defaultFeatures() storage.Features {
	return storage.Features{
		"move":             true,
		"server_side_move": true,
		"server_side_copy": true,
		"directories":      true,
		"multipart":        true,
		"hash:md5":         false,
		"watch":            false,
		"public_url":       false,
		"signed_url":       false,
	}
}

func cloneFeatures(in storage.Features) storage.Features {
	if in == nil {
		return nil
	}
	out := make(storage.Features, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func extractMetadata(opts storage.Options) map[string]string {
	if opts == nil {
		return map[string]string{}
	}
	if m, ok := opts["metadata"].(map[string]string); ok && m != nil {
		return cloneStringMap(m)
	}
	return map[string]string{}
}
