package azureblob

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

	"github.com/go-mizu/blueprints/localbase/pkg/storage"
)

// DSN format:
//
//	azureblob://account/container
//	azureblob://account/container?default_public=true
//
// The account component is required. The path's first segment sets the default
// container name used when Bucket("") is called. Each Open call creates an
// isolated in-memory store; this driver focuses on exercising the storage
// interfaces without contacting Azure during tests.
func init() {
	storage.Register("azureblob", &driver{})
}

type driver struct {
	newStore func(account, container string, public bool) *store
}

func (d *driver) Open(ctx context.Context, dsn string) (storage.Storage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	account, container, public, err := parseDSN(dsn)
	if err != nil {
		return nil, err
	}

	ctor := d.newStore
	if ctor == nil {
		ctor = newStore
	}

	return ctor(account, container, public), nil
}

// parseDSN validates and extracts account, container and visibility settings.
func parseDSN(raw string) (account string, container string, public bool, err error) {
	if strings.TrimSpace(raw) == "" {
		return "", "", false, fmt.Errorf("azureblob: empty dsn")
	}

	u, err := url.Parse(raw)
	if err != nil {
		return "", "", false, fmt.Errorf("azureblob: parse dsn: %w", err)
	}
	if u.Scheme != "azureblob" {
		return "", "", false, fmt.Errorf("azureblob: unexpected scheme %q", u.Scheme)
	}
	if u.Host == "" {
		return "", "", false, fmt.Errorf("azureblob: missing account in host")
	}

	account = u.Host
	container = strings.Trim(strings.TrimPrefix(u.Path, "/"), "/")
	public = strings.EqualFold(u.Query().Get("default_public"), "true")

	return account, container, public, nil
}

// store implements storage.Storage using an in memory representation. The in
// memory backing keeps tests hermetic while mirroring Azure Blob semantics at a
// high level.
type store struct {
	mu            sync.RWMutex
	account       string
	defaultBucket string
	defaultPublic bool
	buckets       map[string]*bucket
	features      storage.Features
}

func newStore(account, container string, public bool) *store {
	return &store{
		account:       account,
		defaultBucket: container,
		defaultPublic: public,
		buckets:       make(map[string]*bucket),
		features: storage.Features{
			"move":              true,
			"server_side_copy":  true,
			"server_side_move":  true,
			"directories":       true,
			"public_url":        true,
			"signed_url":        true,
			"hash:md5":          true,
			"conditional_write": true,
		},
	}
}

func (s *store) Bucket(name string) storage.Bucket {
	s.mu.Lock()
	defer s.mu.Unlock()

	if name == "" {
		name = s.defaultBucket
	}
	if name == "" {
		name = "default"
	}
	if b, ok := s.buckets[name]; ok {
		return b
	}

	b := &bucket{st: s, name: name, obj: make(map[string]*entry), created: time.Now(), public: s.defaultPublic}
	s.buckets[name] = b
	return b
}

func (s *store) Buckets(ctx context.Context, limit, offset int, opts storage.Options) (storage.BucketIter, error) {
	_ = ctx
	_ = opts

	s.mu.RLock()
	defer s.mu.RUnlock()

	names := make([]string, 0, len(s.buckets))
	for n := range s.buckets {
		names = append(names, n)
	}
	sort.Strings(names)

	infos := make([]*storage.BucketInfo, 0, len(names))
	for _, name := range names {
		b := s.buckets[name]
		infos = append(infos, &storage.BucketInfo{
			Name:      name,
			CreatedAt: b.created,
			Public:    b.public,
			Metadata:  map[string]string{"account": s.account},
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
	_ = ctx
	_ = opts

	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("azureblob: bucket name is empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.buckets[name]; ok {
		return nil, storage.ErrExist
	}

	b := &bucket{st: s, name: name, obj: make(map[string]*entry), created: time.Now(), public: s.defaultPublic}
	s.buckets[name] = b

	return &storage.BucketInfo{
		Name:      name,
		CreatedAt: b.created,
		Public:    b.public,
		Metadata:  map[string]string{"account": s.account},
	}, nil
}

func (s *store) DeleteBucket(ctx context.Context, name string, opts storage.Options) error {
	_ = ctx

	s.mu.Lock()
	defer s.mu.Unlock()

	b, ok := s.buckets[name]
	if !ok {
		return storage.ErrNotExist
	}

	if len(b.obj) > 0 && !boolOpt(opts, "force") {
		return storage.ErrPermission
	}

	delete(s.buckets, name)
	return nil
}

func (s *store) Features() storage.Features { return s.features }
func (s *store) Close() error               { return nil }

type bucket struct {
	st      *store
	name    string
	created time.Time
	public  bool

	mu  sync.RWMutex
	obj map[string]*entry
}

type entry struct {
	data        []byte
	obj         storage.Object
	contentType string
}

func (b *bucket) Name() string { return b.name }

func (b *bucket) Info(ctx context.Context) (*storage.BucketInfo, error) {
	_ = ctx

	return &storage.BucketInfo{
		Name:      b.name,
		CreatedAt: b.created,
		Public:    b.public,
		Metadata:  map[string]string{"account": b.st.account},
	}, nil
}

func (b *bucket) Features() storage.Features { return b.st.features }

func (b *bucket) Write(ctx context.Context, key string, src io.Reader, size int64, contentType string, opts storage.Options) (*storage.Object, error) {
	_ = ctx
	if key = strings.TrimSpace(key); key == "" {
		return nil, fmt.Errorf("azureblob: empty key")
	}

	data, err := io.ReadAll(src)
	if err != nil {
		return nil, fmt.Errorf("azureblob: read source: %w", err)
	}
	if size >= 0 && int64(len(data)) != size {
		return nil, fmt.Errorf("azureblob: size mismatch; expected %d got %d", size, len(data))
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if _, ok := b.obj[key]; ok && !boolOpt(opts, "upsert") {
		return nil, storage.ErrExist
	}

	now := time.Now()
	md := map[string]string{}
	if m, ok := opts["metadata"].(map[string]string); ok {
		for k, v := range m {
			md[k] = v
		}
	}

	e := &entry{
		data:        data,
		contentType: contentType,
		obj: storage.Object{
			Key:         key,
			Size:        int64(len(data)),
			ContentType: contentType,
			ETag:        fmt.Sprintf("etag-%s", key),
			Created:     now,
			Updated:     now,
			Hash:        storage.Hashes{"etag": fmt.Sprintf("etag-%s", key)},
			Metadata:    md,
		},
	}
	b.obj[key] = e
	return &e.obj, nil
}

func (b *bucket) Open(ctx context.Context, key string, offset, length int64, opts storage.Options) (io.ReadCloser, *storage.Object, error) {
	_ = ctx
	b.mu.RLock()
	defer b.mu.RUnlock()

	e, ok := b.obj[key]
	if !ok {
		return nil, nil, storage.ErrNotExist
	}

	data := e.data
	if offset > int64(len(data)) {
		offset = int64(len(data))
	}
	end := int64(len(data))
	if length >= 0 && offset+length < end {
		end = offset + length
	}
	view := data[offset:end]

	rc := io.NopCloser(bytes.NewReader(view))
	obj := e.obj
	obj.Size = int64(len(view))
	return rc, &obj, nil
}

func (b *bucket) Stat(ctx context.Context, key string, opts storage.Options) (*storage.Object, error) {
	_ = ctx
	b.mu.RLock()
	defer b.mu.RUnlock()

	e, ok := b.obj[key]
	if !ok {
		return nil, storage.ErrNotExist
	}

	obj := e.obj
	return &obj, nil
}

func (b *bucket) Delete(ctx context.Context, key string, opts storage.Options) error {
	_ = ctx
	_ = opts
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
	if srcBucket != "" && srcBucket != b.name {
		return nil, storage.ErrUnsupported
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	src, ok := b.obj[srcKey]
	if !ok {
		return nil, storage.ErrNotExist
	}

	if _, exists := b.obj[dstKey]; exists {
		return nil, storage.ErrExist
	}

	clone := *src
	clone.obj.Key = dstKey
	b.obj[dstKey] = &clone
	return &clone.obj, nil
}

func (b *bucket) Move(ctx context.Context, dstKey string, srcBucket, srcKey string, opts storage.Options) (*storage.Object, error) {
	obj, err := b.Copy(ctx, dstKey, srcBucket, srcKey, opts)
	if err != nil {
		return nil, err
	}
	_ = b.Delete(ctx, srcKey, opts)
	return obj, nil
}

func (b *bucket) List(ctx context.Context, prefix string, limit, offset int, opts storage.Options) (storage.ObjectIter, error) {
	_ = ctx

	b.mu.RLock()
	defer b.mu.RUnlock()

	keys := make([]string, 0, len(b.obj))
	for key := range b.obj {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	objs := make([]*storage.Object, 0, len(keys))
	for _, key := range keys {
		e := b.obj[key]
		if boolOpt(opts, "dirs_only") && !strings.HasSuffix(key, "/") {
			continue
		}
		if boolOpt(opts, "files_only") && strings.HasSuffix(key, "/") {
			continue
		}
		obj := e.obj
		objs = append(objs, &obj)
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
	_ = ctx
	_ = opts
	if _, ok := b.obj[key]; !ok {
		return "", storage.ErrNotExist
	}
	return fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s?sig=dummy&method=%s&exp=%d", b.st.account, b.name, key, method, int64(expires/time.Second)), nil
}

func (b *bucket) URL(ctx context.Context, key string, method string, expires time.Duration, opts storage.Options) (string, error) {
	return b.SignedURL(ctx, key, method, expires, opts)
}

func boolOpt(opts storage.Options, key string) bool {
	if v, ok := opts[key].(bool); ok {
		return v
	}
	return false
}

type bucketIter struct {
	buckets []*storage.BucketInfo
	idx     int
}

func (it *bucketIter) Next() (*storage.BucketInfo, error) {
	if it.idx >= len(it.buckets) {
		return nil, io.EOF
	}
	b := it.buckets[it.idx]
	it.idx++
	return b, nil
}

func (it *bucketIter) Close() error { return nil }

type objectIter struct {
	objects []*storage.Object
	idx     int
}

func (it *objectIter) Next() (*storage.Object, error) {
	if it.idx >= len(it.objects) {
		return nil, io.EOF
	}
	obj := it.objects[it.idx]
	it.idx++
	return obj, nil
}

func (it *objectIter) Close() error { return nil }
