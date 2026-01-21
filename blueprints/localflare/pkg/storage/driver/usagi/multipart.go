package usagi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/storage"
)

type multipartUpload struct {
	id          string
	key         string
	contentType string
	created     time.Time
	parts       map[int]*multipartPart
}

type multipartPart struct {
	number int
	size   int64
	path   string
	etag   string
	mtime  time.Time
}

func (b *bucket) InitMultipart(ctx context.Context, key string, contentType string, opts storage.Options) (*storage.MultipartUpload, error) {
	_ = ctx
	_ = opts
	if err := validateKey(key); err != nil {
		return nil, err
	}
	if err := b.ensureLoaded(); err != nil {
		return nil, err
	}
	if err := b.ensureMultipartDir(); err != nil {
		return nil, fmt.Errorf("usagi: create multipart dir: %w", err)
	}

	id, err := newUploadID()
	if err != nil {
		return nil, err
	}

	b.multipartMu.Lock()
	b.multipartUploads[id] = &multipartUpload{
		id:          id,
		key:         key,
		contentType: contentType,
		created:     time.Now(),
		parts:       make(map[int]*multipartPart),
	}
	b.multipartMu.Unlock()

	if err := os.MkdirAll(b.uploadDir(id), defaultPermissions); err != nil {
		return nil, fmt.Errorf("usagi: create upload dir: %w", err)
	}

	return &storage.MultipartUpload{
		Bucket:   b.name,
		Key:      key,
		UploadID: id,
	}, nil
}

func (b *bucket) UploadPart(ctx context.Context, mu *storage.MultipartUpload, number int, src io.Reader, size int64, opts storage.Options) (*storage.PartInfo, error) {
	_ = ctx
	_ = opts
	if mu == nil {
		return nil, fmt.Errorf("usagi: multipart upload required")
	}
	if number <= 0 {
		return nil, fmt.Errorf("usagi: part number out of range")
	}
	if err := b.ensureLoaded(); err != nil {
		return nil, err
	}

	b.multipartMu.Lock()
	up, ok := b.multipartUploads[mu.UploadID]
	b.multipartMu.Unlock()
	if !ok {
		return nil, storage.ErrNotExist
	}

	partPath := b.multipartPath(mu.UploadID, number)
	if err := os.MkdirAll(filepathDir(partPath), defaultPermissions); err != nil {
		return nil, fmt.Errorf("usagi: create part dir: %w", err)
	}

	file, err := os.Create(partPath)
	if err != nil {
		return nil, fmt.Errorf("usagi: create part: %w", err)
	}
	defer file.Close()

	written, err := io.Copy(file, src)
	if err != nil {
		return nil, fmt.Errorf("usagi: write part: %w", err)
	}
	if size > 0 && written != size {
		// Best effort: accept actual size
	}

	etag := fmt.Sprintf("%x", crc32ChecksumFile(partPath))
	mtime := time.Now()

	b.multipartMu.Lock()
	up.parts[number] = &multipartPart{
		number: number,
		size:   written,
		path:   partPath,
		etag:   etag,
		mtime:  mtime,
	}
	b.multipartMu.Unlock()

	return &storage.PartInfo{
		Number:       number,
		Size:         written,
		ETag:         etag,
		LastModified: &mtime,
	}, nil
}

func (b *bucket) CopyPart(ctx context.Context, mu *storage.MultipartUpload, number int, opts storage.Options) (*storage.PartInfo, error) {
	if mu == nil {
		return nil, fmt.Errorf("usagi: multipart upload required")
	}
	srcBucket := b.name
	if opts != nil {
		if v, ok := opts["source_bucket"].(string); ok && v != "" {
			srcBucket = v
		}
	}
	srcKey, _ := opts["source_key"].(string)
	if srcKey == "" {
		return nil, fmt.Errorf("usagi: source_key required")
	}
	source := b
	if srcBucket != b.name {
		source = b.store.getBucket(srcBucket)
	}

	var offset, length int64
	if opts != nil {
		if v, ok := opts["source_offset"].(int64); ok {
			offset = v
		}
		if v, ok := opts["source_length"].(int64); ok {
			length = v
		}
	}

	rc, _, err := source.Open(ctx, srcKey, offset, length, nil)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return b.UploadPart(ctx, mu, number, rc, length, nil)
}

func (b *bucket) ListParts(ctx context.Context, mu *storage.MultipartUpload, limit, offset int, opts storage.Options) ([]*storage.PartInfo, error) {
	_ = ctx
	_ = opts
	if mu == nil {
		return nil, fmt.Errorf("usagi: multipart upload required")
	}
	b.multipartMu.Lock()
	up, ok := b.multipartUploads[mu.UploadID]
	b.multipartMu.Unlock()
	if !ok {
		return nil, storage.ErrNotExist
	}

	parts := make([]*multipartPart, 0, len(up.parts))
	for _, p := range up.parts {
		parts = append(parts, p)
	}
	sort.Slice(parts, func(i, j int) bool { return parts[i].number < parts[j].number })

	start := offset
	if start < 0 {
		start = 0
	}
	end := len(parts)
	if limit > 0 && start+limit < end {
		end = start + limit
	}
	if start > len(parts) {
		start = len(parts)
	}
	parts = parts[start:end]

	out := make([]*storage.PartInfo, 0, len(parts))
	for _, p := range parts {
		p := p
		out = append(out, &storage.PartInfo{
			Number:       p.number,
			Size:         p.size,
			ETag:         p.etag,
			LastModified: &p.mtime,
		})
	}
	return out, nil
}

func (b *bucket) CompleteMultipart(ctx context.Context, mu *storage.MultipartUpload, parts []*storage.PartInfo, opts storage.Options) (*storage.Object, error) {
	_ = ctx
	_ = opts
	if mu == nil {
		return nil, fmt.Errorf("usagi: multipart upload required")
	}
	b.multipartMu.Lock()
	up, ok := b.multipartUploads[mu.UploadID]
	b.multipartMu.Unlock()
	if !ok {
		return nil, storage.ErrNotExist
	}

	selected := make([]*multipartPart, 0, len(parts))
	for _, p := range parts {
		mp, ok := up.parts[p.Number]
		if !ok {
			return nil, fmt.Errorf("usagi: part %d missing", p.Number)
		}
		selected = append(selected, mp)
	}
	sort.Slice(selected, func(i, j int) bool { return selected[i].number < selected[j].number })

	totalSize := int64(0)
	paths := make([]string, 0, len(selected))
	for _, p := range selected {
		totalSize += p.size
		paths = append(paths, p.path)
	}
	reader, err := newMultipartReader(paths)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	obj, err := b.Write(ctx, up.key, reader, totalSize, up.contentType, nil)
	if err != nil {
		return nil, err
	}

	_ = os.RemoveAll(b.uploadDir(mu.UploadID))
	b.multipartMu.Lock()
	delete(b.multipartUploads, mu.UploadID)
	b.multipartMu.Unlock()
	return obj, nil
}

type multipartReader struct {
	paths []string
	idx   int
	file  *os.File
}

func newMultipartReader(paths []string) (*multipartReader, error) {
	return &multipartReader{paths: paths}, nil
}

func (r *multipartReader) Read(p []byte) (int, error) {
	for {
		if r.file == nil {
			if r.idx >= len(r.paths) {
				return 0, io.EOF
			}
			f, err := os.Open(r.paths[r.idx])
			if err != nil {
				return 0, err
			}
			r.file = f
		}
		n, err := r.file.Read(p)
		if err == io.EOF {
			_ = r.file.Close()
			r.file = nil
			r.idx++
			if n > 0 {
				return n, nil
			}
			continue
		}
		return n, err
	}
}

func (r *multipartReader) Close() error {
	if r.file != nil {
		_ = r.file.Close()
		r.file = nil
	}
	return nil
}

func (b *bucket) AbortMultipart(ctx context.Context, mu *storage.MultipartUpload, opts storage.Options) error {
	_ = ctx
	_ = opts
	if mu == nil {
		return fmt.Errorf("usagi: multipart upload required")
	}
	b.multipartMu.Lock()
	_, ok := b.multipartUploads[mu.UploadID]
	if ok {
		delete(b.multipartUploads, mu.UploadID)
	}
	b.multipartMu.Unlock()
	if ok {
		_ = os.RemoveAll(b.uploadDir(mu.UploadID))
	}
	return nil
}

func newUploadID() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", fmt.Errorf("usagi: upload id: %w", err)
	}
	return hex.EncodeToString(buf[:]), nil
}

func filepathDir(path string) string {
	idx := strings.LastIndex(path, string(os.PathSeparator))
	if idx <= 0 {
		return path
	}
	return path[:idx]
}

func crc32ChecksumFile(path string) uint32 {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	return crc32.Checksum(data, crcTable)
}
