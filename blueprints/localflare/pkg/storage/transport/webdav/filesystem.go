// File: lib/storage/transport/webdav/filesystem.go

package webdav

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/go-mizu/blueprints/drive/lib/storage"
	"golang.org/x/net/webdav"
)

// StorageFileSystem implements webdav.FileSystem backed by storage.Storage.
type StorageFileSystem struct {
	store              storage.Storage
	bucket             string // optional single-bucket mode
	readOnly           bool
	hideDotFiles       bool
	defaultContentType string
	maxUploadSize      int64
	writeBufferSize    int64
	tempDir            string
	logger             *slog.Logger
}

// Mkdir creates a directory (bucket or virtual directory).
func (fs *StorageFileSystem) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	if fs.readOnly {
		return os.ErrPermission
	}

	bucket, key, err := fs.parsePath(name)
	if err != nil {
		return err
	}

	if bucket == "" {
		// Cannot create root
		return os.ErrPermission
	}

	if key == "" {
		// Creating a bucket
		_, err := fs.store.CreateBucket(ctx, bucket, nil)
		if err != nil {
			return mapError(err)
		}
		return nil
	}

	// Creating a virtual directory within a bucket
	// In object storage, directories are implicit - we can either:
	// 1. Do nothing (most common)
	// 2. Create a marker object

	// Check if bucket exists first
	bkt := fs.store.Bucket(bucket)
	_, err = bkt.Info(ctx)
	if err != nil {
		return mapError(err)
	}

	// Virtual directories don't need explicit creation
	return nil
}

// OpenFile opens or creates a file.
func (fs *StorageFileSystem) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	bucket, key, err := fs.parsePath(name)
	if err != nil {
		return nil, err
	}

	isWrite := flag&(os.O_WRONLY|os.O_RDWR|os.O_CREATE|os.O_TRUNC) != 0

	if isWrite && fs.readOnly {
		return nil, os.ErrPermission
	}

	// Root directory
	if bucket == "" {
		return &StorageFile{
			fs:    fs,
			isDir: true,
			info: &StorageFileInfo{
				name:    "/",
				mode:    os.ModeDir | 0755,
				modTime: time.Now(),
				isDir:   true,
			},
		}, nil
	}

	// Bucket directory
	if key == "" {
		bkt := fs.store.Bucket(bucket)
		bktInfo, err := bkt.Info(ctx)
		if err != nil {
			return nil, mapError(err)
		}

		return &StorageFile{
			fs:     fs,
			bucket: bucket,
			isDir:  true,
			info: &StorageFileInfo{
				name:    bucket,
				mode:    os.ModeDir | 0755,
				modTime: bktInfo.CreatedAt,
				isDir:   true,
			},
		}, nil
	}

	bkt := fs.store.Bucket(bucket)

	// For write operations, create a write handle
	if isWrite {
		return &StorageFile{
			fs:          fs,
			bucket:      bucket,
			key:         key,
			isDir:       false,
			isWrite:     true,
			writeBuffer: new(bytes.Buffer),
			info: &StorageFileInfo{
				name:    path.Base(key),
				mode:    0644,
				modTime: time.Now(),
				isDir:   false,
			},
		}, nil
	}

	// For read operations, check if it's a file or directory
	obj, err := bkt.Stat(ctx, key, nil)
	if err == nil {
		// It's a file
		if obj.IsDir {
			return &StorageFile{
				fs:     fs,
				bucket: bucket,
				key:    key,
				isDir:  true,
				info:   objectToFileInfo(obj),
			}, nil
		}

		return &StorageFile{
			fs:     fs,
			bucket: bucket,
			key:    key,
			isDir:  false,
			info:   objectToFileInfo(obj),
		}, nil
	}

	// Check if it's a virtual directory (prefix with children)
	iter, err := bkt.List(ctx, key+"/", 1, 0, nil)
	if err == nil {
		defer func() { _ = iter.Close() }()
		child, _ := iter.Next()
		if child != nil {
			// It's a virtual directory
			return &StorageFile{
				fs:     fs,
				bucket: bucket,
				key:    key,
				isDir:  true,
				info: &StorageFileInfo{
					name:    path.Base(key),
					mode:    os.ModeDir | 0755,
					modTime: time.Now(),
					isDir:   true,
				},
			}, nil
		}
	}

	return nil, os.ErrNotExist
}

// RemoveAll removes a file or directory recursively.
func (fs *StorageFileSystem) RemoveAll(ctx context.Context, name string) error {
	if fs.readOnly {
		return os.ErrPermission
	}

	bucket, key, err := fs.parsePath(name)
	if err != nil {
		return err
	}

	if bucket == "" {
		// Cannot remove root
		return os.ErrPermission
	}

	if key == "" {
		// Removing a bucket
		err := fs.store.DeleteBucket(ctx, bucket, storage.Options{"force": true})
		return mapError(err)
	}

	bkt := fs.store.Bucket(bucket)

	// Try to delete as a single object first
	err = bkt.Delete(ctx, key, nil)
	if err == nil {
		return nil
	}

	// If not found, try to delete as a directory prefix
	if !errors.Is(err, storage.ErrNotExist) {
		return mapError(err)
	}

	// Delete all objects with this prefix
	prefix := strings.TrimSuffix(key, "/") + "/"
	iter, err := bkt.List(ctx, prefix, 0, 0, storage.Options{"recursive": true})
	if err != nil {
		return mapError(err)
	}
	defer func() { _ = iter.Close() }()

	var deleted int
	for {
		obj, err := iter.Next()
		if err != nil {
			return mapError(err)
		}
		if obj == nil {
			break
		}

		if err := bkt.Delete(ctx, obj.Key, nil); err != nil {
			return mapError(err)
		}
		deleted++
	}

	if deleted == 0 {
		return os.ErrNotExist
	}

	return nil
}

// Rename renames/moves a file or directory.
func (fs *StorageFileSystem) Rename(ctx context.Context, oldName, newName string) error {
	if fs.readOnly {
		return os.ErrPermission
	}

	oldBucket, oldKey, err := fs.parsePath(oldName)
	if err != nil {
		return err
	}

	newBucket, newKey, err := fs.parsePath(newName)
	if err != nil {
		return err
	}

	// Cannot rename root or buckets
	if oldBucket == "" || newBucket == "" {
		return os.ErrPermission
	}

	// Cannot rename buckets themselves
	if oldKey == "" || newKey == "" {
		return os.ErrPermission
	}

	// Cross-bucket moves
	if oldBucket != newBucket {
		// Try server-side copy then delete
		srcBkt := fs.store.Bucket(oldBucket)
		dstBkt := fs.store.Bucket(newBucket)

		_, err := dstBkt.Copy(ctx, newKey, oldBucket, oldKey, nil)
		if err != nil {
			return mapError(err)
		}

		return mapError(srcBkt.Delete(ctx, oldKey, nil))
	}

	// Same bucket move
	bkt := fs.store.Bucket(oldBucket)
	_, err = bkt.Move(ctx, newKey, oldBucket, oldKey, nil)
	return mapError(err)
}

// Stat returns file info.
func (fs *StorageFileSystem) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	bucket, key, err := fs.parsePath(name)
	if err != nil {
		return nil, err
	}

	// Root directory
	if bucket == "" {
		return &StorageFileInfo{
			name:    "/",
			mode:    os.ModeDir | 0755,
			modTime: time.Now(),
			isDir:   true,
		}, nil
	}

	bkt := fs.store.Bucket(bucket)

	// Bucket directory
	if key == "" {
		bktInfo, err := bkt.Info(ctx)
		if err != nil {
			return nil, mapError(err)
		}
		return &StorageFileInfo{
			name:    bucket,
			mode:    os.ModeDir | 0755,
			modTime: bktInfo.CreatedAt,
			isDir:   true,
		}, nil
	}

	// Try as a file
	obj, err := bkt.Stat(ctx, key, nil)
	if err == nil {
		return objectToFileInfo(obj), nil
	}

	// Check if it's a virtual directory
	iter, err := bkt.List(ctx, key+"/", 1, 0, nil)
	if err == nil {
		defer func() { _ = iter.Close() }()
		child, _ := iter.Next()
		if child != nil {
			return &StorageFileInfo{
				name:    path.Base(key),
				mode:    os.ModeDir | 0755,
				modTime: time.Now(),
				isDir:   true,
			}, nil
		}
	}

	return nil, os.ErrNotExist
}

// parsePath converts WebDAV path to bucket and key.
func (fs *StorageFileSystem) parsePath(name string) (bucket, key string, err error) {
	// Clean and normalize path
	name = path.Clean("/" + name)
	name = strings.TrimPrefix(name, "/")

	// Single-bucket mode
	if fs.bucket != "" {
		return fs.bucket, name, nil
	}

	// Multi-bucket mode
	if name == "" {
		return "", "", nil // Root listing
	}

	parts := strings.SplitN(name, "/", 2)
	bucket = parts[0]
	if len(parts) > 1 {
		key = parts[1]
	}

	return bucket, key, nil
}

// objectToFileInfo converts storage.Object to os.FileInfo.
func objectToFileInfo(obj *storage.Object) *StorageFileInfo {
	mode := os.FileMode(0644)
	if obj.IsDir {
		mode = os.ModeDir | 0755
	}

	modTime := obj.Updated
	if modTime.IsZero() {
		modTime = obj.Created
	}
	if modTime.IsZero() {
		modTime = time.Now()
	}

	return &StorageFileInfo{
		name:        path.Base(obj.Key),
		size:        obj.Size,
		mode:        mode,
		modTime:     modTime,
		isDir:       obj.IsDir,
		contentType: obj.ContentType,
		etag:        obj.ETag,
	}
}

// StorageFile implements webdav.File.
type StorageFile struct {
	fs     *StorageFileSystem
	bucket string
	key    string
	isDir  bool
	info   *StorageFileInfo

	// For reading
	mu     sync.Mutex
	reader io.ReadCloser
	offset int64

	// For writing
	isWrite     bool
	writeBuffer *bytes.Buffer
	tempFile    *os.File
	written     int64
}

// Read implements io.Reader.
func (f *StorageFile) Read(p []byte) (n int, err error) {
	if f.isDir {
		return 0, os.ErrInvalid
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if f.reader == nil {
		// Open reader on first read
		ctx := context.Background()
		bkt := f.fs.store.Bucket(f.bucket)
		reader, _, err := bkt.Open(ctx, f.key, f.offset, -1, nil)
		if err != nil {
			return 0, mapError(err)
		}
		f.reader = reader
	}

	n, err = f.reader.Read(p)
	f.offset += int64(n)
	return n, err
}

// Seek implements io.Seeker.
func (f *StorageFile) Seek(offset int64, whence int) (int64, error) {
	if f.isDir {
		return 0, os.ErrInvalid
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	var newOffset int64
	switch whence {
	case io.SeekStart:
		newOffset = offset
	case io.SeekCurrent:
		newOffset = f.offset + offset
	case io.SeekEnd:
		if f.info != nil {
			newOffset = f.info.size + offset
		} else {
			return 0, errors.New("cannot seek from end: unknown file size")
		}
	default:
		return 0, os.ErrInvalid
	}

	if newOffset < 0 {
		return 0, os.ErrInvalid
	}

	// Close existing reader if seeking to different position
	if f.reader != nil && newOffset != f.offset {
		_ = f.reader.Close()
		f.reader = nil
	}

	f.offset = newOffset
	return f.offset, nil
}

// Readdir returns directory entries.
func (f *StorageFile) Readdir(count int) ([]os.FileInfo, error) {
	if !f.isDir {
		return nil, os.ErrInvalid
	}

	ctx := context.Background()
	var entries []os.FileInfo

	if f.bucket == "" {
		// List buckets
		iter, err := f.fs.store.Buckets(ctx, 0, 0, nil)
		if err != nil {
			return nil, mapError(err)
		}
		defer func() { _ = iter.Close() }()

		for {
			bktInfo, err := iter.Next()
			if err != nil {
				return nil, mapError(err)
			}
			if bktInfo == nil {
				break
			}

			if f.fs.hideDotFiles && strings.HasPrefix(bktInfo.Name, ".") {
				continue
			}

			entries = append(entries, &StorageFileInfo{
				name:    bktInfo.Name,
				mode:    os.ModeDir | 0755,
				modTime: bktInfo.CreatedAt,
				isDir:   true,
			})

			if count > 0 && len(entries) >= count {
				break
			}
		}
	} else {
		// List objects in bucket
		bkt := f.fs.store.Bucket(f.bucket)
		prefix := f.key
		if prefix != "" && !strings.HasSuffix(prefix, "/") {
			prefix += "/"
		}

		iter, err := bkt.List(ctx, prefix, 0, 0, nil)
		if err != nil {
			return nil, mapError(err)
		}
		defer func() { _ = iter.Close() }()

		seen := make(map[string]bool)

		for {
			obj, err := iter.Next()
			if err != nil {
				return nil, mapError(err)
			}
			if obj == nil {
				break
			}

			// Extract immediate child name
			remaining := strings.TrimPrefix(obj.Key, prefix)
			if remaining == "" {
				continue
			}

			parts := strings.SplitN(remaining, "/", 2)
			name := parts[0]

			if seen[name] {
				continue
			}
			seen[name] = true

			if f.fs.hideDotFiles && strings.HasPrefix(name, ".") {
				continue
			}

			isDir := len(parts) > 1 || obj.IsDir

			var mode os.FileMode
			var size int64
			var modTime time.Time
			var contentType string
			var etag string

			if isDir {
				mode = os.ModeDir | 0755
			} else {
				mode = 0644
				size = obj.Size
				contentType = obj.ContentType
				etag = obj.ETag
			}

			modTime = obj.Updated
			if modTime.IsZero() {
				modTime = obj.Created
			}
			if modTime.IsZero() {
				modTime = time.Now()
			}

			entries = append(entries, &StorageFileInfo{
				name:        name,
				size:        size,
				mode:        mode,
				modTime:     modTime,
				isDir:       isDir,
				contentType: contentType,
				etag:        etag,
			})

			if count > 0 && len(entries) >= count {
				break
			}
		}
	}

	if len(entries) == 0 && count != 0 {
		return nil, io.EOF
	}

	return entries, nil
}

// Stat returns file info.
func (f *StorageFile) Stat() (os.FileInfo, error) {
	if f.info != nil {
		return f.info, nil
	}
	return nil, os.ErrInvalid
}

// Write implements io.Writer for file uploads.
func (f *StorageFile) Write(p []byte) (n int, err error) {
	if !f.isWrite {
		return 0, os.ErrPermission
	}

	if f.fs.maxUploadSize > 0 && f.written+int64(len(p)) > f.fs.maxUploadSize {
		return 0, fmt.Errorf("upload size exceeds limit of %d bytes", f.fs.maxUploadSize)
	}

	// Buffer small writes
	if f.tempFile == nil && f.written+int64(len(p)) <= f.fs.writeBufferSize {
		n, err = f.writeBuffer.Write(p)
		f.written += int64(n)
		return n, err
	}

	// Spill to temp file for large uploads
	if f.tempFile == nil {
		tempDir := f.fs.tempDir
		if tempDir == "" {
			tempDir = os.TempDir()
		}

		f.tempFile, err = os.CreateTemp(tempDir, "webdav-upload-*")
		if err != nil {
			return 0, err
		}

		// Write buffered data to temp file
		if f.writeBuffer.Len() > 0 {
			_, err = f.tempFile.Write(f.writeBuffer.Bytes())
			if err != nil {
				return 0, err
			}
			f.writeBuffer.Reset()
		}
	}

	n, err = f.tempFile.Write(p)
	f.written += int64(n)
	return n, err
}

// Close closes the file.
func (f *StorageFile) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Close reader if open
	if f.reader != nil {
		err := f.reader.Close()
		f.reader = nil
		if err != nil {
			return err
		}
	}

	// For write operations, commit data to storage
	if f.isWrite && (f.writeBuffer.Len() > 0 || f.tempFile != nil) {
		ctx := context.Background()
		bkt := f.fs.store.Bucket(f.bucket)

		var reader io.Reader
		var size int64

		if f.tempFile != nil {
			// Seek to beginning of temp file
			_, err := f.tempFile.Seek(0, io.SeekStart)
			if err != nil {
				_ = f.tempFile.Close()
				_ = os.Remove(f.tempFile.Name())
				return err
			}

			reader = f.tempFile
			size = f.written
		} else {
			reader = f.writeBuffer
			size = int64(f.writeBuffer.Len())
		}

		contentType := detectContentType(f.key)
		if contentType == "" {
			contentType = f.fs.defaultContentType
		}

		_, err := bkt.Write(ctx, f.key, reader, size, contentType, nil)

		// Clean up temp file
		if f.tempFile != nil {
			_ = f.tempFile.Close()
			_ = os.Remove(f.tempFile.Name())
			f.tempFile = nil
		}

		if err != nil {
			return mapError(err)
		}
	}

	// Clean up temp file if write was aborted
	if f.tempFile != nil {
		_ = f.tempFile.Close()
		_ = os.Remove(f.tempFile.Name())
		f.tempFile = nil
	}

	return nil
}

// StorageFileInfo implements os.FileInfo.
type StorageFileInfo struct {
	name        string
	size        int64
	mode        os.FileMode
	modTime     time.Time
	isDir       bool
	contentType string
	etag        string
}

func (fi *StorageFileInfo) Name() string       { return fi.name }
func (fi *StorageFileInfo) Size() int64        { return fi.size }
func (fi *StorageFileInfo) Mode() os.FileMode  { return fi.mode }
func (fi *StorageFileInfo) ModTime() time.Time { return fi.modTime }
func (fi *StorageFileInfo) IsDir() bool        { return fi.isDir }
func (fi *StorageFileInfo) Sys() interface{}   { return nil }

// ContentType implements webdav.ContentTyper.
func (fi *StorageFileInfo) ContentType(ctx context.Context) (string, error) {
	if fi.contentType != "" {
		return fi.contentType, nil
	}
	// Return ErrNotImplemented to use default behavior
	return "", webdav.ErrNotImplemented
}

// ETag implements webdav.ETager.
func (fi *StorageFileInfo) ETag(ctx context.Context) (string, error) {
	if fi.etag != "" {
		return fi.etag, nil
	}
	// Return ErrNotImplemented to use default (ModTime + Size based)
	return "", webdav.ErrNotImplemented
}

// mapError maps storage errors to OS errors.
func mapError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, storage.ErrNotExist):
		return os.ErrNotExist
	case errors.Is(err, storage.ErrExist):
		return os.ErrExist
	case errors.Is(err, storage.ErrPermission):
		return os.ErrPermission
	default:
		return err
	}
}

// detectContentType guesses content type from file extension.
func detectContentType(filename string) string {
	ext := strings.ToLower(path.Ext(filename))
	switch ext {
	case ".html", ".htm":
		return "text/html"
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".txt":
		return "text/plain"
	case ".md":
		return "text/markdown"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".webp":
		return "image/webp"
	case ".ico":
		return "image/x-icon"
	case ".pdf":
		return "application/pdf"
	case ".zip":
		return "application/zip"
	case ".tar":
		return "application/x-tar"
	case ".gz", ".gzip":
		return "application/gzip"
	case ".mp3":
		return "audio/mpeg"
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".ttf":
		return "font/ttf"
	case ".otf":
		return "font/otf"
	case ".csv":
		return "text/csv"
	case ".yaml", ".yml":
		return "application/x-yaml"
	case ".toml":
		return "application/toml"
	default:
		return ""
	}
}
