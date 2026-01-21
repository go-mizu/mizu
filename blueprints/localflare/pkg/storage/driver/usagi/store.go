package usagi

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/storage"
)

const (
	logFileName        = "data.usagi"
	multipartDirName   = ".usagi-multipart"
	defaultPermissions = 0o755
)

type store struct {
	root          string
	defaultBucket string
	nofsync       bool
	segmentSize   int64
	manifestEvery time.Duration

	mu       sync.RWMutex
	buckets  map[string]*bucket
	features storage.Features
}

func (s *store) Bucket(name string) storage.Bucket {
	if strings.TrimSpace(name) == "" {
		name = s.defaultBucket
	}
	b := s.getBucket(name)
	return b
}

func (s *store) Buckets(ctx context.Context, limit, offset int, opts storage.Options) (storage.BucketIter, error) {
	_ = ctx
	_ = opts
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return nil, fmt.Errorf("usagi: list buckets: %w", err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		n := e.Name()
		if strings.HasPrefix(n, ".") {
			continue
		}
		names = append(names, n)
	}
	sort.Strings(names)

	start := offset
	if start < 0 {
		start = 0
	}
	end := len(names)
	if limit > 0 && start+limit < end {
		end = start + limit
	}
	if start > len(names) {
		start = len(names)
	}
	selected := names[start:end]

	infos := make([]*storage.BucketInfo, 0, len(selected))
	for _, name := range selected {
		infos = append(infos, &storage.BucketInfo{Name: name})
	}
	return &bucketIter{items: infos}, nil
}

func (s *store) CreateBucket(ctx context.Context, name string, opts storage.Options) (*storage.BucketInfo, error) {
	_ = ctx
	_ = opts
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("usagi: bucket name required")
	}
	bucketDir := filepath.Join(s.root, name)
	if _, err := os.Stat(bucketDir); err == nil {
		return nil, storage.ErrExist
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("usagi: stat bucket: %w", err)
	}
	if err := os.MkdirAll(bucketDir, defaultPermissions); err != nil {
		return nil, fmt.Errorf("usagi: create bucket: %w", err)
	}

	segmentDir := filepath.Join(bucketDir, segmentDirName)
	if err := os.MkdirAll(segmentDir, defaultPermissions); err != nil {
		return nil, fmt.Errorf("usagi: create segment dir: %w", err)
	}
	segmentPath := filepath.Join(segmentDir, segmentFileName(1))
	file, err := os.OpenFile(segmentPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("usagi: create segment: %w", err)
	}
	file.Close()

	b := s.getBucket(name)
	b.ensureLoaded()

	return &storage.BucketInfo{
		Name:      name,
		CreatedAt: time.Now(),
	}, nil
}

func (s *store) DeleteBucket(ctx context.Context, name string, opts storage.Options) error {
	_ = ctx
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("usagi: bucket name required")
	}
	bucketDir := filepath.Join(s.root, name)
	if _, err := os.Stat(bucketDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return storage.ErrNotExist
		}
		return fmt.Errorf("usagi: stat bucket: %w", err)
	}
	force := false
	if opts != nil {
		if v, ok := opts["force"].(bool); ok {
			force = v
		}
	}

	b := s.getBucket(name)
	if err := b.ensureLoaded(); err != nil {
		return err
	}
	if !force {
		nonEmpty := b.index.Len() > 0
		if nonEmpty {
			return storage.ErrPermission
		}
	}

	s.mu.Lock()
	delete(s.buckets, name)
	s.mu.Unlock()

	b.close()
	return os.RemoveAll(bucketDir)
}

func (s *store) Features() storage.Features {
	return s.features
}

func (s *store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, b := range s.buckets {
		b.close()
	}
	return nil
}

func (s *store) getBucket(name string) *bucket {
	s.mu.RLock()
	b, ok := s.buckets[name]
	s.mu.RUnlock()
	if ok {
		return b
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if b, ok := s.buckets[name]; ok {
		return b
	}
	b = &bucket{
		store:            s,
		name:             name,
		dir:              filepath.Join(s.root, name),
		logPath:          filepath.Join(s.root, name, logFileName),
		index:            newShardedIndex(),
		prefixIndex:      newPrefixIndex(2),
		features:         storage.Features{"move": true, "multipart": true},
		multipartDir:     filepath.Join(s.root, name, multipartDirName),
		multipartUploads: make(map[string]*multipartUpload),
	}
	s.buckets[name] = b
	return b
}

// bucketIter implements storage.BucketIter.
type bucketIter struct {
	items []*storage.BucketInfo
	idx   int
}

func (it *bucketIter) Next() (*storage.BucketInfo, error) {
	if it.idx >= len(it.items) {
		return nil, nil
	}
	item := it.items[it.idx]
	it.idx++
	return item, nil
}

func (it *bucketIter) Close() error {
	return nil
}
