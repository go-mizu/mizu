package rabbit

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/storage"
)

// Store implements storage.Storage with high-performance in-process storage.
type Store struct {
	root    string
	buckets sync.Map // name -> *Bucket

	// Global caches
	hotCache  *HotCache
	warmCache *WarmCache

	// Statistics
	totalOps   atomic.Int64
	totalBytes atomic.Int64
}

var _ storage.Storage = (*Store)(nil)

// NewStore creates a new rabbit store.
func NewStore(root string) (*Store, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("rabbit: abs path: %w", err)
	}

	s := &Store{
		root:      absRoot,
		hotCache:  newHotCache(),
		warmCache: newWarmCache(L2CacheSize, 50000),
	}

	return s, nil
}

// Bucket returns a bucket handle. Creates bucket directory if needed.
func (s *Store) Bucket(name string) storage.Bucket {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "default"
	}
	name = safeBucketName(name)

	// Fast path: check if bucket already exists
	if b, ok := s.buckets.Load(name); ok {
		return b.(*Bucket)
	}

	// Slow path: create bucket
	root := filepath.Join(s.root, name)
	b := &Bucket{
		store:     s,
		name:      name,
		root:      root,
		hotCache:  s.hotCache,
		warmCache: s.warmCache,
	}

	// Initialize key shards
	for i := range b.keyShards {
		b.keyShards[i] = &keyShard{
			keys: make([]string, 0, 64),
		}
	}

	actual, _ := s.buckets.LoadOrStore(name, b)
	return actual.(*Bucket)
}

// Buckets lists all buckets.
func (s *Store) Buckets(ctx context.Context, limit, offset int, opts storage.Options) (storage.BucketIter, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(s.root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &bucketIter{}, nil
		}
		return nil, fmt.Errorf("rabbit: read root: %w", err)
	}

	var list []*storage.BucketInfo
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
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

// CreateBucket creates a new bucket.
func (s *Store) CreateBucket(ctx context.Context, name string, opts storage.Options) (*storage.BucketInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("rabbit: bucket name required")
	}
	name = safeBucketName(name)

	path := filepath.Join(s.root, name)
	if err := os.Mkdir(path, DirPermissions); err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, storage.ErrExist
		}
		return nil, fmt.Errorf("rabbit: create bucket %q: %w", name, err)
	}

	now := time.Now()
	return &storage.BucketInfo{
		Name:      name,
		CreatedAt: now,
	}, nil
}

// DeleteBucket deletes a bucket.
func (s *Store) DeleteBucket(ctx context.Context, name string, opts storage.Options) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("rabbit: bucket name required")
	}
	name = safeBucketName(name)

	path := filepath.Join(s.root, name)
	force := boolOpt(opts, "force")

	var err error
	if force {
		err = os.RemoveAll(path)
	} else {
		err = os.Remove(path)
	}

	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return storage.ErrNotExist
		}
		return fmt.Errorf("rabbit: delete bucket %q: %w", name, err)
	}

	s.buckets.Delete(name)
	return nil
}

// Features returns backend capabilities.
func (s *Store) Features() storage.Features {
	return storage.Features{
		"move":               true,
		"directories":        true,
		"object_move_server": true,
		"dir_move_server":    true,
		"multipart":          true,
	}
}

// Close closes the store.
func (s *Store) Close() error {
	return nil
}

// =============================================================================
// ITERATORS
// =============================================================================

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

// =============================================================================
// HELPERS
// =============================================================================

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

func boolOpt(opts storage.Options, key string) bool {
	if opts == nil {
		return false
	}
	v, ok := opts[key]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	return ok && b
}
