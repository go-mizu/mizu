package herd

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/liteio-dev/liteio/pkg/storage"
)

// multipartUpload tracks an in-progress multipart upload.
type multipartUpload struct {
	mu          *storage.MultipartUpload
	contentType string
	createdAt   time.Time
	parts       map[int]*partData
}

type partData struct {
	number       int
	data         []byte
	etag         string
	lastModified time.Time
}

// multipartRegistry holds all active multipart uploads.
type multipartRegistry struct {
	mu      sync.RWMutex
	uploads map[string]*multipartUpload
}

func newMultipartRegistry() *multipartRegistry {
	return &multipartRegistry{
		uploads: make(map[string]*multipartUpload),
	}
}

func (b *bucket) InitMultipart(ctx context.Context, key string, contentType string, opts storage.Options) (*storage.MultipartUpload, error) {
	_ = ctx
	_ = opts

	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("herd: key is empty")
	}

	uploadID := newUploadID()

	mu := &storage.MultipartUpload{
		Bucket:   b.name,
		Key:      key,
		UploadID: uploadID,
	}

	b.st.mp.mu.Lock()
	b.st.mp.uploads[uploadID] = &multipartUpload{
		mu:          mu,
		contentType: contentType,
		createdAt:   fastNowTime(),
		parts:       make(map[int]*partData),
	}
	b.st.mp.mu.Unlock()

	return mu, nil
}

func (b *bucket) UploadPart(ctx context.Context, mu *storage.MultipartUpload, number int, src io.Reader, size int64, opts storage.Options) (*storage.PartInfo, error) {
	_ = ctx
	_ = opts

	if number <= 0 || number > 10000 {
		return nil, fmt.Errorf("herd: part number %d out of range (1-10000)", number)
	}

	var data []byte
	if size >= 0 {
		data = make([]byte, size)
		n, err := io.ReadFull(src, data)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return nil, err
		}
		data = data[:n]
	} else {
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, src); err != nil {
			return nil, err
		}
		data = buf.Bytes()
	}

	now := fastNowTime()
	sum := md5.Sum(data)
	etag := hex.EncodeToString(sum[:])

	pd := &partData{
		number:       number,
		data:         data,
		etag:         etag,
		lastModified: now,
	}

	b.st.mp.mu.Lock()
	upload, ok := b.st.mp.uploads[mu.UploadID]
	if !ok {
		b.st.mp.mu.Unlock()
		return nil, storage.ErrNotExist
	}
	upload.parts[number] = pd
	b.st.mp.mu.Unlock()

	return &storage.PartInfo{
		Number:       number,
		Size:         int64(len(data)),
		ETag:         etag,
		LastModified: &now,
	}, nil
}

func (b *bucket) CopyPart(ctx context.Context, mu *storage.MultipartUpload, number int, opts storage.Options) (*storage.PartInfo, error) {
	_ = ctx
	_ = mu
	_ = number
	_ = opts
	return nil, storage.ErrUnsupported
}

func (b *bucket) ListParts(ctx context.Context, mu *storage.MultipartUpload, limit, offset int, opts storage.Options) ([]*storage.PartInfo, error) {
	_ = ctx
	_ = opts

	b.st.mp.mu.RLock()
	defer b.st.mp.mu.RUnlock()

	upload, ok := b.st.mp.uploads[mu.UploadID]
	if !ok {
		return nil, storage.ErrNotExist
	}

	parts := make([]*storage.PartInfo, 0, len(upload.parts))
	for _, pd := range upload.parts {
		lastMod := pd.lastModified
		parts = append(parts, &storage.PartInfo{
			Number:       pd.number,
			Size:         int64(len(pd.data)),
			ETag:         pd.etag,
			LastModified: &lastMod,
		})
	}

	sort.Slice(parts, func(i, j int) bool {
		return parts[i].Number < parts[j].Number
	})

	if offset < 0 {
		offset = 0
	}
	if offset > len(parts) {
		offset = len(parts)
	}
	parts = parts[offset:]
	if limit > 0 && limit < len(parts) {
		parts = parts[:limit]
	}

	return parts, nil
}

func (b *bucket) CompleteMultipart(ctx context.Context, mu *storage.MultipartUpload, parts []*storage.PartInfo, opts storage.Options) (*storage.Object, error) {
	_ = ctx
	_ = opts

	if len(parts) == 0 {
		return nil, fmt.Errorf("herd: no parts to complete")
	}

	b.st.mp.mu.Lock()
	upload, ok := b.st.mp.uploads[mu.UploadID]
	if !ok {
		b.st.mp.mu.Unlock()
		return nil, storage.ErrNotExist
	}

	sortedParts := make([]*storage.PartInfo, len(parts))
	copy(sortedParts, parts)
	sort.Slice(sortedParts, func(i, j int) bool {
		return sortedParts[i].Number < sortedParts[j].Number
	})

	totalSize := 0
	for _, part := range sortedParts {
		pd, exists := upload.parts[part.Number]
		if !exists {
			b.st.mp.mu.Unlock()
			return nil, fmt.Errorf("herd: part %d not found", part.Number)
		}
		totalSize += len(pd.data)
	}

	data := make([]byte, 0, totalSize)
	for _, part := range sortedParts {
		pd := upload.parts[part.Number]
		data = append(data, pd.data...)
	}

	delete(b.st.mp.uploads, mu.UploadID)
	b.st.mp.mu.Unlock()

	// Route to stripe and write.
	now := fastNow()
	size := int64(totalSize)
	stripe := b.st.stripeFor(b.name, upload.mu.Key)

	bl, kl, cl := len(b.name), len(upload.mu.Key), len(upload.contentType)
	recTotalSize := int64(recFixedSize+bl+kl+cl) + size

	var valOff int64
	if stripe.ring != nil && recTotalSize <= stripe.ring.capacity {
		valPosInRecord := 19 + bl + kl + cl
		bufSlice, _, vo, wb := stripe.ring.writeInline(recTotalSize, valPosInRecord)
		valOff = vo
		stripe.vol.buildRecordBuf(bufSlice, recPut, b.name, upload.mu.Key, upload.contentType, data, now)
		wb.done()
	} else {
		var err error
		_, valOff, err = stripe.vol.appendRecord(recPut, b.name, upload.mu.Key, upload.contentType, data, now)
		if err != nil {
			return nil, err
		}
	}

	e := acquireIndexEntry()
	e.size = size
	e.contentType = upload.contentType
	e.created = now
	e.updated = now

	// Inline if small enough.
	if size <= b.st.inlineMax && size > 0 {
		e.inline = make([]byte, size)
		copy(e.inline, data)
		e.valueOffset = 0
	} else {
		e.valueOffset = valOff
	}

	stripe.idx.put(b.name, upload.mu.Key, e)
	stripe.bloom.add(b.name, upload.mu.Key)

	return &storage.Object{
		Bucket:      b.name,
		Key:         upload.mu.Key,
		Size:        size,
		ContentType: upload.contentType,
		Created:     time.Unix(0, now),
		Updated:     time.Unix(0, now),
	}, nil
}

func (b *bucket) AbortMultipart(ctx context.Context, mu *storage.MultipartUpload, opts storage.Options) error {
	_ = ctx
	_ = opts

	b.st.mp.mu.Lock()
	defer b.st.mp.mu.Unlock()

	if _, ok := b.st.mp.uploads[mu.UploadID]; !ok {
		return storage.ErrNotExist
	}

	delete(b.st.mp.uploads, mu.UploadID)
	return nil
}

func newUploadID() string {
	now := time.Now().UTC().UnixNano()
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%x-0", now)
	}
	r := binary.LittleEndian.Uint64(b[:])
	return fmt.Sprintf("%x-%x", now, r)
}
