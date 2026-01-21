package rabbit

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/storage"
)

var uploadIDCounter atomic.Int64

func init() {
	uploadIDCounter.Store(time.Now().UnixNano())
}

// Ensure Bucket implements HasMultipart
var _ storage.HasMultipart = (*Bucket)(nil)

// InitMultipart starts a new multipart upload.
func (b *Bucket) InitMultipart(ctx context.Context, key string, contentType string, opts storage.Options) (*storage.MultipartUpload, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	relKey, err := cleanKey(key)
	if err != nil {
		return nil, err
	}

	// Generate unique upload ID
	id := strconv.FormatInt(uploadIDCounter.Add(1), 36)

	// Create temp directory for parts
	tmpDir := filepath.Join(b.root, ".multipart", id)
	if err := os.MkdirAll(tmpDir, DirPermissions); err != nil {
		return nil, fmt.Errorf("rabbit: create multipart dir: %w", err)
	}

	var metadata map[string]string
	if opts != nil {
		if m, ok := opts["metadata"].(map[string]string); ok {
			metadata = m
		}
	}

	upload := &multipartUpload{
		id:       id,
		key:      relKey,
		parts:    make(map[int]*uploadPart),
		created:  time.Now(),
		tmpDir:   tmpDir,
		metadata: metadata,
	}

	b.mpMu.Lock()
	if b.mpUploads == nil {
		b.mpUploads = make(map[string]*multipartUpload)
	}
	b.mpUploads[id] = upload
	b.mpMu.Unlock()

	return &storage.MultipartUpload{
		Bucket:   b.name,
		Key:      relToKey(relKey),
		UploadID: id,
		Metadata: metadata,
	}, nil
}

// UploadPart uploads a part of a multipart upload.
func (b *Bucket) UploadPart(ctx context.Context, mu *storage.MultipartUpload, number int, src io.Reader, size int64, opts storage.Options) (*storage.PartInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if number < 1 || number > MaxPartNumber {
		return nil, fmt.Errorf("rabbit: part number %d out of range [1, %d]", number, MaxPartNumber)
	}

	b.mpMu.RLock()
	upload, ok := b.mpUploads[mu.UploadID]
	b.mpMu.RUnlock()
	if !ok {
		return nil, storage.ErrNotExist
	}

	// Create part file
	partPath := filepath.Join(upload.tmpDir, fmt.Sprintf("part-%05d", number))
	f, err := os.OpenFile(partPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, FilePermissions)
	if err != nil {
		return nil, fmt.Errorf("rabbit: create part file: %w", err)
	}

	// Write part and compute MD5
	hash := md5.New()
	writer := io.MultiWriter(f, hash)

	buf := getBuffer(size)
	defer putBuffer(buf)

	var written int64
	for {
		nr, readErr := src.Read(buf)
		if nr > 0 {
			nw, writeErr := writer.Write(buf[:nr])
			written += int64(nw)
			if writeErr != nil {
				f.Close()
				os.Remove(partPath)
				return nil, fmt.Errorf("rabbit: write part: %w", writeErr)
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			f.Close()
			os.Remove(partPath)
			return nil, fmt.Errorf("rabbit: read part: %w", readErr)
		}
	}

	if !NoFsync {
		if err := f.Sync(); err != nil {
			f.Close()
			os.Remove(partPath)
			return nil, fmt.Errorf("rabbit: sync part: %w", err)
		}
	}
	f.Close()

	etag := hex.EncodeToString(hash.Sum(nil))

	// Register part
	b.mpMu.Lock()
	upload.parts[number] = &uploadPart{
		number: number,
		path:   partPath,
		size:   written,
		etag:   etag,
	}
	b.mpMu.Unlock()

	return &storage.PartInfo{
		Number: number,
		Size:   written,
		ETag:   etag,
	}, nil
}

// CopyPart copies part of an existing object as a multipart part.
func (b *Bucket) CopyPart(ctx context.Context, mu *storage.MultipartUpload, number int, opts storage.Options) (*storage.PartInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if number < 1 || number > MaxPartNumber {
		return nil, fmt.Errorf("rabbit: part number %d out of range", number)
	}

	b.mpMu.RLock()
	_, ok := b.mpUploads[mu.UploadID]
	b.mpMu.RUnlock()
	if !ok {
		return nil, storage.ErrNotExist
	}

	// Get source options
	srcBucket := mu.Bucket
	if sb, ok := opts["source_bucket"].(string); ok && sb != "" {
		srcBucket = sb
	}
	srcKey, _ := opts["source_key"].(string)
	if srcKey == "" {
		return nil, errors.New("rabbit: source_key required for CopyPart")
	}
	offset, _ := opts["source_offset"].(int64)
	length, _ := opts["source_length"].(int64)

	// Open source
	srcBucketName := safeBucketName(srcBucket)
	srcRoot := filepath.Join(b.store.root, srcBucketName)
	srcRel, err := cleanKey(srcKey)
	if err != nil {
		return nil, err
	}
	srcFull, err := joinUnderRoot(srcRoot, srcRel)
	if err != nil {
		return nil, err
	}

	src, err := os.Open(srcFull)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, storage.ErrNotExist
		}
		return nil, fmt.Errorf("rabbit: open source: %w", err)
	}
	defer src.Close()

	if offset > 0 {
		if _, err := src.Seek(offset, io.SeekStart); err != nil {
			return nil, fmt.Errorf("rabbit: seek source: %w", err)
		}
	}

	var reader io.Reader = src
	if length > 0 {
		reader = io.LimitReader(src, length)
	}

	return b.UploadPart(ctx, mu, number, reader, length, opts)
}

// ListParts lists already uploaded parts for a multipart upload.
func (b *Bucket) ListParts(ctx context.Context, mu *storage.MultipartUpload, limit, offset int, opts storage.Options) ([]*storage.PartInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	b.mpMu.RLock()
	upload, ok := b.mpUploads[mu.UploadID]
	if !ok {
		b.mpMu.RUnlock()
		return nil, storage.ErrNotExist
	}

	var parts []*storage.PartInfo
	for _, p := range upload.parts {
		parts = append(parts, &storage.PartInfo{
			Number: p.number,
			Size:   p.size,
			ETag:   p.etag,
		})
	}
	b.mpMu.RUnlock()

	sort.Slice(parts, func(i, j int) bool {
		return parts[i].Number < parts[j].Number
	})

	if offset > 0 && offset < len(parts) {
		parts = parts[offset:]
	}
	if limit > 0 && limit < len(parts) {
		parts = parts[:limit]
	}

	return parts, nil
}

// CompleteMultipart assembles parts into final object.
func (b *Bucket) CompleteMultipart(ctx context.Context, mu *storage.MultipartUpload, parts []*storage.PartInfo, opts storage.Options) (*storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	b.mpMu.Lock()
	upload, ok := b.mpUploads[mu.UploadID]
	if !ok {
		b.mpMu.Unlock()
		return nil, storage.ErrNotExist
	}
	delete(b.mpUploads, mu.UploadID)
	b.mpMu.Unlock()

	defer os.RemoveAll(upload.tmpDir)

	// Sort parts by number
	sort.Slice(parts, func(i, j int) bool {
		return parts[i].Number < parts[j].Number
	})

	// Verify all parts exist
	for _, p := range parts {
		if _, ok := upload.parts[p.Number]; !ok {
			return nil, fmt.Errorf("rabbit: part %d not found", p.Number)
		}
	}

	// Create final object
	full, err := joinUnderRoot(b.root, upload.key)
	if err != nil {
		return nil, err
	}
	dir := filepath.Dir(full)
	if !globalDirCache.Check(dir) {
		if err := os.MkdirAll(dir, DirPermissions); err != nil {
			return nil, fmt.Errorf("rabbit: mkdir: %w", err)
		}
		globalDirCache.Add(dir)
	}

	// Assemble parts
	dst, err := os.OpenFile(full, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, FilePermissions)
	if err != nil {
		return nil, fmt.Errorf("rabbit: create final: %w", err)
	}

	buf := largePool.Get()
	defer largePool.Put(buf)

	var totalSize int64
	for _, p := range parts {
		part := upload.parts[p.Number]
		src, err := os.Open(part.path)
		if err != nil {
			dst.Close()
			os.Remove(full)
			return nil, fmt.Errorf("rabbit: open part: %w", err)
		}

		n, err := io.CopyBuffer(dst, src, buf)
		src.Close()
		if err != nil {
			dst.Close()
			os.Remove(full)
			return nil, fmt.Errorf("rabbit: copy part: %w", err)
		}
		totalSize += n
	}

	if !NoFsync {
		if err := dst.Sync(); err != nil {
			dst.Close()
			os.Remove(full)
			return nil, fmt.Errorf("rabbit: sync final: %w", err)
		}
	}
	dst.Close()

	b.addKey(upload.key)

	now := fastNow()
	return &storage.Object{
		Bucket:  b.name,
		Key:     relToKey(upload.key),
		Size:    totalSize,
		Created: now,
		Updated: now,
	}, nil
}

// AbortMultipart cancels a multipart upload.
func (b *Bucket) AbortMultipart(ctx context.Context, mu *storage.MultipartUpload, opts storage.Options) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	b.mpMu.Lock()
	upload, ok := b.mpUploads[mu.UploadID]
	if !ok {
		b.mpMu.Unlock()
		return storage.ErrNotExist
	}
	delete(b.mpUploads, mu.UploadID)
	b.mpMu.Unlock()

	return os.RemoveAll(upload.tmpDir)
}
