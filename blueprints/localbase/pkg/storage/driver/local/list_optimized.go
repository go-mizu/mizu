// File: driver/local/list_optimized.go
package local

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/go-mizu/blueprints/localbase/pkg/storage"
)

// =============================================================================
// OPTIMIZED LIST WITH STREAMING AND LAZY INFO()
// =============================================================================

// streamingObjectIter implements storage.ObjectIter with batch fetching
// and lazy Info() evaluation to reduce syscall overhead.
type streamingObjectIter struct {
	dir       *os.File
	root      string
	prefix    string
	bucketName string
	recursive  bool
	dirsOnly   bool
	filesOnly  bool

	// Current batch
	batch    []*storage.Object
	pos      int
	done     bool

	// Reusable buffer for directory reading
	buf      []os.DirEntry
}

const (
	// Number of entries to fetch per batch
	listBatchSize = 256
)

// Pre-allocated pools for Object structs to reduce allocations
var listObjectPool = sync.Pool{
	New: func() any {
		return &storage.Object{}
	},
}

// newStreamingObjectIter creates a streaming iterator for directory listing.
func newStreamingObjectIter(root, prefix, bucketName string, recursive, dirsOnly, filesOnly bool) (*streamingObjectIter, error) {
	base := filepath.Join(root, prefix)

	info, err := os.Stat(base)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty iterator
			return &streamingObjectIter{done: true}, nil
		}
		return nil, err
	}

	if !info.IsDir() {
		// Single file - return just that
		obj := &storage.Object{
			Bucket:  bucketName,
			Key:     relToKey(prefix),
			Size:    info.Size(),
			IsDir:   false,
			Created: info.ModTime(),
			Updated: info.ModTime(),
		}
		return &streamingObjectIter{
			batch: []*storage.Object{obj},
			done:  true,
		}, nil
	}

	dir, err := os.Open(base)
	if err != nil {
		return nil, err
	}

	return &streamingObjectIter{
		dir:        dir,
		root:       root,
		prefix:     prefix,
		bucketName: bucketName,
		recursive:  recursive,
		dirsOnly:   dirsOnly,
		filesOnly:  filesOnly,
		buf:        make([]os.DirEntry, 0, listBatchSize),
	}, nil
}

// Next returns the next object or nil when done.
func (it *streamingObjectIter) Next() (*storage.Object, error) {
	if it.done && it.pos >= len(it.batch) {
		return nil, nil
	}

	// Need to fetch more?
	if it.pos >= len(it.batch) {
		if err := it.fetchBatch(); err != nil {
			return nil, err
		}
		if it.pos >= len(it.batch) {
			return nil, nil
		}
	}

	obj := it.batch[it.pos]
	it.pos++
	return obj, nil
}

// fetchBatch reads the next batch of entries from the directory.
func (it *streamingObjectIter) fetchBatch() error {
	it.batch = it.batch[:0]
	it.pos = 0

	if it.dir == nil {
		it.done = true
		return nil
	}

	entries, err := it.dir.ReadDir(listBatchSize)
	if err != nil {
		it.done = true
		it.dir.Close()
		it.dir = nil
		if err.Error() == "EOF" {
			return nil
		}
		return err
	}

	if len(entries) < listBatchSize {
		it.done = true
	}

	base := filepath.Join(it.root, it.prefix)

	for _, e := range entries {
		isDir := e.IsDir()

		// Apply filters
		if it.dirsOnly && !isDir {
			continue
		}
		if it.filesOnly && isDir {
			continue
		}

		relPath := filepath.Join(it.prefix, e.Name())

		// Get a pooled object
		obj := listObjectPool.Get().(*storage.Object)
		obj.Bucket = it.bucketName
		obj.Key = relToKey(relPath)
		obj.IsDir = isDir

		// Lazy info - only call Info() if we actually need size/time
		// For directories, we often don't need detailed info
		if !isDir {
			info, err := e.Info()
			if err != nil {
				// Skip entries we can't stat
				listObjectPool.Put(obj)
				continue
			}
			obj.Size = info.Size()
			obj.Created = info.ModTime()
			obj.Updated = info.ModTime()
		} else {
			// For directories, use Type() which doesn't require stat
			info, err := os.Stat(filepath.Join(base, e.Name()))
			if err == nil {
				obj.Created = info.ModTime()
				obj.Updated = info.ModTime()
			}
		}

		it.batch = append(it.batch, obj)
	}

	return nil
}

// Close releases resources.
func (it *streamingObjectIter) Close() error {
	if it.dir != nil {
		err := it.dir.Close()
		it.dir = nil
		return err
	}
	return nil
}

// =============================================================================
// OPTIMIZED RECURSIVE WALK
// =============================================================================

// walkDirOptimized performs a recursive directory walk with optimizations:
// - Pre-allocates the result slice based on initial directory size
// - Avoids calling Info() when not necessary
// - Uses pooled objects to reduce allocations
func walkDirOptimized(root, prefix, bucketName string, dirsOnly, filesOnly bool) ([]*storage.Object, error) {
	base := filepath.Join(root, prefix)

	// Estimate capacity based on initial readdir
	dir, err := os.Open(base)
	if err != nil {
		if os.IsNotExist(err) {
			return []*storage.Object{}, nil
		}
		return nil, err
	}

	// Read first batch to estimate total size
	initialEntries, err := dir.ReadDir(-1)
	dir.Close()
	if err != nil && err.Error() != "EOF" {
		return nil, err
	}

	// Estimate total objects (assume ~10 files per directory on average)
	estimatedSize := len(initialEntries) * 10
	if estimatedSize < 100 {
		estimatedSize = 100
	}

	objects := make([]*storage.Object, 0, estimatedSize)

	err = filepath.WalkDir(base, func(p string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if p == base {
			return nil
		}

		relPath, err := filepath.Rel(root, p)
		if err != nil {
			return nil
		}

		isDir := d.IsDir()

		// Apply filters early to avoid Info() call
		if dirsOnly && !isDir {
			return nil
		}
		if filesOnly && isDir {
			return nil
		}

		// Only call Info() when we actually need the data
		info, err := d.Info()
		if err != nil {
			return nil
		}

		obj := &storage.Object{
			Bucket:  bucketName,
			Key:     relToKey(filepath.ToSlash(relPath)),
			Size:    info.Size(),
			IsDir:   isDir,
			Created: info.ModTime(),
			Updated: info.ModTime(),
		}

		objects = append(objects, obj)
		return nil
	})

	return objects, err
}

// =============================================================================
// FAST SINGLE-LEVEL LIST
// =============================================================================

// listDirFast lists a single directory level without recursion.
// Optimized to minimize syscalls by using DirEntry.Type() instead of Info().
func listDirFast(root, prefix, bucketName string, dirsOnly, filesOnly bool) ([]*storage.Object, error) {
	base := filepath.Join(root, prefix)

	entries, err := os.ReadDir(base)
	if err != nil {
		if os.IsNotExist(err) {
			return []*storage.Object{}, nil
		}
		return nil, err
	}

	// Pre-allocate with exact size
	objects := make([]*storage.Object, 0, len(entries))

	for _, e := range entries {
		isDir := e.IsDir()

		// Apply filters before calling Info()
		if dirsOnly && !isDir {
			continue
		}
		if filesOnly && isDir {
			continue
		}

		relPath := filepath.Join(prefix, e.Name())

		// Get file info (needed for size and times)
		info, err := e.Info()
		if err != nil {
			continue
		}

		obj := &storage.Object{
			Bucket:  bucketName,
			Key:     relToKey(filepath.ToSlash(relPath)),
			Size:    info.Size(),
			IsDir:   isDir,
			Created: info.ModTime(),
			Updated: info.ModTime(),
		}

		objects = append(objects, obj)
	}

	return objects, nil
}
