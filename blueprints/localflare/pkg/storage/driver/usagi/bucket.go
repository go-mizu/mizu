package usagi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/storage"
)

var crcTable = crc32.MakeTable(crc32.Castagnoli)

var _ storage.Bucket = (*bucket)(nil)
var _ storage.HasMultipart = (*bucket)(nil)

type entry struct {
	segmentID   int64
	offset      int64
	size        int64
	contentType string
	updated     time.Time
	checksum    uint32
}

type bucket struct {
	store *store
	name  string
	dir   string

	logPath string
	logMu   sync.Mutex
	log     *os.File

	index *shardedIndex

	features storage.Features

	loadOnce sync.Once
	loadErr  error

	segmentFile        *os.File
	currentSegmentID   int64
	currentSegmentSize int64

	lastManifest time.Time

	prefixIndex *prefixIndex

	multipartMu      sync.Mutex
	multipartDir     string
	multipartUploads map[string]*multipartUpload
}

func (b *bucket) Name() string {
	return b.name
}

func (b *bucket) Info(ctx context.Context) (*storage.BucketInfo, error) {
	_ = ctx
	if err := b.ensureLoaded(); err != nil {
		return nil, err
	}
	return &storage.BucketInfo{Name: b.name}, nil
}

func (b *bucket) Features() storage.Features {
	return b.features
}

func (b *bucket) Write(ctx context.Context, key string, src io.Reader, size int64, contentType string, opts storage.Options) (*storage.Object, error) {
	_ = ctx
	_ = opts
	if strings.TrimSpace(key) == "" {
		return nil, fmt.Errorf("usagi: empty key")
	}
	if err := b.ensureLoaded(); err != nil {
		return nil, err
	}

	data, err := readAllSized(src, size)
	if err != nil {
		return nil, err
	}
	sz := int64(len(data))
	checksum := crc32.Checksum(data, crcTable)
	updated := time.Now()

	segID, off, err := b.appendRecord(recordOpPut, key, contentType, data, checksum, updated)
	if err != nil {
		return nil, err
	}

	entry := &entry{
		segmentID:   segID,
		offset:      off,
		size:        sz,
		contentType: contentType,
		updated:     updated,
		checksum:    checksum,
	}
	b.index.Set(key, entry)
	if b.prefixIndex != nil {
		b.prefixIndex.Add(key)
	}

	return &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        sz,
		ContentType: contentType,
		Updated:     updated,
	}, nil
}

func (b *bucket) Open(ctx context.Context, key string, offset, length int64, opts storage.Options) (io.ReadCloser, *storage.Object, error) {
	_ = ctx
	_ = opts
	if strings.TrimSpace(key) == "" {
		return nil, nil, fmt.Errorf("usagi: empty key")
	}
	if err := b.ensureLoaded(); err != nil {
		return nil, nil, err
	}

	e, ok := b.index.Get(key)
	if !ok {
		return nil, nil, storage.ErrNotExist
	}

	if offset < 0 {
		offset = 0
	}
	dataLen := e.size
	start := e.offset + offset
	remain := dataLen - offset
	if remain < 0 {
		return nil, nil, storage.ErrNotExist
	}
	readLen := remain
	if length > 0 && length < readLen {
		readLen = length
	}

	file, err := os.Open(b.segmentPath(e.segmentID))
	if err != nil {
		return nil, nil, fmt.Errorf("usagi: open segment: %w", err)
	}
	reader := io.NewSectionReader(file, start, readLen)
	return &readCloser{SectionReader: reader, closer: file}, &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        e.size,
		ContentType: e.contentType,
		Updated:     e.updated,
	}, nil
}

func (b *bucket) Stat(ctx context.Context, key string, opts storage.Options) (*storage.Object, error) {
	_ = ctx
	_ = opts
	if strings.TrimSpace(key) == "" {
		return nil, fmt.Errorf("usagi: empty key")
	}
	if err := b.ensureLoaded(); err != nil {
		return nil, err
	}

	e, ok := b.index.Get(key)
	if !ok {
		return nil, storage.ErrNotExist
	}
	return &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        e.size,
		ContentType: e.contentType,
		Updated:     e.updated,
	}, nil
}

func (b *bucket) Delete(ctx context.Context, key string, opts storage.Options) error {
	_ = ctx
	_ = opts
	if strings.TrimSpace(key) == "" {
		return fmt.Errorf("usagi: empty key")
	}
	if err := b.ensureLoaded(); err != nil {
		return err
	}

	_, _, err := b.appendRecord(recordOpDelete, key, "", nil, 0, time.Now())
	if err != nil {
		return err
	}
	b.index.Delete(key)
	if b.prefixIndex != nil {
		b.prefixIndex.Remove(key)
	}
	return nil
}

func (b *bucket) Copy(ctx context.Context, dstKey string, srcBucket, srcKey string, opts storage.Options) (*storage.Object, error) {
	_ = opts
	if strings.TrimSpace(dstKey) == "" || strings.TrimSpace(srcKey) == "" {
		return nil, fmt.Errorf("usagi: empty key")
	}
	if err := b.ensureLoaded(); err != nil {
		return nil, err
	}

	src := b
	if srcBucket != "" && srcBucket != b.name {
		src = b.store.getBucket(srcBucket)
	}
	if err := src.ensureLoaded(); err != nil {
		return nil, err
	}

	rc, obj, err := src.Open(ctx, srcKey, 0, 0, nil)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	return b.Write(ctx, dstKey, rc, obj.Size, obj.ContentType, nil)
}

func (b *bucket) Move(ctx context.Context, dstKey string, srcBucket, srcKey string, opts storage.Options) (*storage.Object, error) {
	obj, err := b.Copy(ctx, dstKey, srcBucket, srcKey, opts)
	if err != nil {
		return nil, err
	}
	src := b
	if srcBucket != "" && srcBucket != b.name {
		src = b.store.getBucket(srcBucket)
	}
	_ = src.Delete(ctx, srcKey, nil)
	return obj, nil
}

func (b *bucket) List(ctx context.Context, prefix string, limit, offset int, opts storage.Options) (storage.ObjectIter, error) {
	_ = ctx
	_ = opts
	if err := b.ensureLoaded(); err != nil {
		return nil, err
	}

	if b.prefixIndex != nil && prefix != "" {
		if keys, ok := b.prefixIndex.Get(prefix); ok {
			keys = applyOffsetLimit(keys, offset, limit)
			return b.keysToIter(keys)
		}
	}
	keys := b.index.Keys(prefix)
	keys = applyOffsetLimit(keys, offset, limit)

	return b.keysToIter(keys)
}

func (b *bucket) SignedURL(ctx context.Context, key string, method string, expires time.Duration, opts storage.Options) (string, error) {
	_ = ctx
	_ = key
	_ = method
	_ = expires
	_ = opts
	return "", storage.ErrUnsupported
}

func (b *bucket) keysToIter(keys []string) (storage.ObjectIter, error) {
	items := make([]*storage.Object, 0, len(keys))
	for _, k := range keys {
		if e, ok := b.index.Get(k); ok {
			items = append(items, &storage.Object{
				Bucket:      b.name,
				Key:         k,
				Size:        e.size,
				ContentType: e.contentType,
				Updated:     e.updated,
			})
		}
	}
	return &objectIter{items: items}, nil
}

func applyOffsetLimit(keys []string, offset, limit int) []string {
	start := offset
	if start < 0 {
		start = 0
	}
	end := len(keys)
	if limit > 0 && start+limit < end {
		end = start + limit
	}
	if start > len(keys) {
		start = len(keys)
	}
	return keys[start:end]
}

func (b *bucket) ensureLoaded() error {
	b.loadOnce.Do(func() {
		b.loadErr = b.load()
	})
	return b.loadErr
}

func (b *bucket) load() error {
	if b.name == "" {
		return fmt.Errorf("usagi: bucket name required")
	}
	if _, err := os.Stat(b.dir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return storage.ErrNotExist
		}
		return fmt.Errorf("usagi: stat bucket: %w", err)
	}
	if err := os.MkdirAll(b.dir, defaultPermissions); err != nil {
		return fmt.Errorf("usagi: create bucket dir: %w", err)
	}

	if err := os.MkdirAll(b.segmentDir(), defaultPermissions); err != nil {
		return fmt.Errorf("usagi: create segment dir: %w", err)
	}

	if err := b.migrateLegacyLog(); err != nil {
		return err
	}

	if err := b.loadFromManifest(); err != nil {
		return err
	}

	if err := b.openLastSegment(); err != nil {
		return err
	}

	if b.prefixIndex != nil {
		b.prefixIndex.BuildFromIndex(b.index.Snapshot())
	}

	return nil
}

func (b *bucket) migrateLegacyLog() error {
	if _, err := os.Stat(b.logPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("usagi: stat legacy log: %w", err)
	}
	segmentDir := b.segmentDir()
	if err := os.MkdirAll(segmentDir, defaultPermissions); err != nil {
		return fmt.Errorf("usagi: create segment dir: %w", err)
	}
	firstSegmentPath := b.segmentPath(1)
	if _, err := os.Stat(firstSegmentPath); err == nil {
		return nil
	}
	if err := os.Rename(b.logPath, firstSegmentPath); err != nil {
		return fmt.Errorf("usagi: migrate legacy log: %w", err)
	}
	return nil
}

func (b *bucket) loadFromManifest() error {
	m, err := b.loadManifest()
	if err == nil {
		for k, v := range m.Index {
			b.index.Set(k, &entry{
				segmentID:   v.SegmentID,
				offset:      v.Offset,
				size:        v.Size,
				contentType: v.ContentType,
				updated:     time.Unix(0, v.UpdatedUnix),
				checksum:    v.Checksum,
			})
		}
		b.currentSegmentID = m.LastSegmentID
		b.currentSegmentSize = m.LastSegmentSize
		if err := b.replaySegmentsAfter(m.LastSegmentID, m.LastSegmentSize); err != nil {
			return err
		}
		b.lastManifest = time.Now()
		return nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return b.fullReplaySegments()
	}
	return fmt.Errorf("usagi: load manifest: %w", err)
}

func (b *bucket) fullReplaySegments() error {
	ids, err := b.listSegments()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("usagi: list segments: %w", err)
	}
	for _, id := range ids {
		path := b.segmentPath(id)
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("usagi: open segment: %w", err)
		}
		info, err := file.Stat()
		if err != nil {
			file.Close()
			return fmt.Errorf("usagi: stat segment: %w", err)
		}
		_, err = b.rebuildIndex(file, info.Size(), 0, id)
		file.Close()
		if err != nil {
			return err
		}
		b.currentSegmentID = id
		b.currentSegmentSize = info.Size()
	}
	return nil
}

func (b *bucket) replaySegmentsAfter(lastID int64, lastOffset int64) error {
	ids, err := b.listSegments()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("usagi: list segments: %w", err)
	}
	for _, id := range ids {
		path := b.segmentPath(id)
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("usagi: open segment: %w", err)
		}
		info, err := file.Stat()
		if err != nil {
			file.Close()
			return fmt.Errorf("usagi: stat segment: %w", err)
		}
		start := int64(0)
		if id == lastID {
			start = lastOffset
		}
		if id < lastID {
			file.Close()
			continue
		}
		lastPos, err := b.rebuildIndex(file, info.Size(), start, id)
		file.Close()
		if err != nil {
			return err
		}
		if id > lastID {
			b.currentSegmentID = id
			b.currentSegmentSize = info.Size()
		} else {
			b.currentSegmentSize = lastPos
		}
	}
	return nil
}

func (b *bucket) openLastSegment() error {
	ids, err := b.listSegments()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return b.createSegment(1)
		}
		return fmt.Errorf("usagi: list segments: %w", err)
	}
	if len(ids) == 0 {
		return b.createSegment(1)
	}
	lastID := ids[len(ids)-1]
	path := b.segmentPath(lastID)
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return fmt.Errorf("usagi: open segment: %w", err)
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return fmt.Errorf("usagi: stat segment: %w", err)
	}
	b.segmentFile = file
	b.currentSegmentID = lastID
	b.currentSegmentSize = info.Size()
	return nil
}

func (b *bucket) createSegment(id int64) error {
	path := b.segmentPath(id)
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("usagi: create segment: %w", err)
	}
	b.segmentFile = file
	b.currentSegmentID = id
	b.currentSegmentSize = 0
	return nil
}

func (b *bucket) rotateSegment() error {
	if b.segmentFile != nil {
		_ = b.segmentFile.Sync()
		_ = b.segmentFile.Close()
	}
	return b.createSegment(b.currentSegmentID + 1)
}

func (b *bucket) maybeWriteManifestLocked() {
	if b.store.manifestEvery <= 0 {
		return
	}
	if time.Since(b.lastManifest) < b.store.manifestEvery {
		return
	}
	_ = b.writeManifest()
	b.lastManifest = time.Now()
}

func (b *bucket) rebuildIndex(file *os.File, size int64, start int64, segmentID int64) (int64, error) {
	offset := start
	headerBuf := make([]byte, recordHeaderSize)
	for offset+recordHeaderSize <= size {
		if _, err := file.ReadAt(headerBuf, offset); err != nil {
			return offset, fmt.Errorf("usagi: read header: %w", err)
		}
		hdr, err := decodeHeader(headerBuf)
		if err != nil {
			return offset, err
		}
		if hdr.Magic != recordMagic || hdr.Version != recordVersion {
			return offset, errCorruptRecord
		}
		keyLen := int(hdr.KeyLen)
		ctLen := int(hdr.ContentTypeLen)
		payloadLen := keyLen + ctLen
		entryStart := offset + recordHeaderSize
		entryEnd := entryStart + int64(payloadLen)
		dataEnd := entryEnd + int64(hdr.DataLen)
		if dataEnd > size {
			return offset, errCorruptRecord
		}

		payload := make([]byte, payloadLen)
		if payloadLen > 0 {
			if _, err := file.ReadAt(payload, entryStart); err != nil {
				return offset, fmt.Errorf("usagi: read payload: %w", err)
			}
		}
		key := string(payload[:keyLen])
		contentType := ""
		if ctLen > 0 {
			contentType = string(payload[keyLen:])
		}

		switch hdr.Op {
		case recordOpPut:
			b.index.Set(key, &entry{
				segmentID:   segmentID,
				offset:      entryEnd,
				size:        int64(hdr.DataLen),
				contentType: contentType,
				updated:     time.Unix(0, hdr.UpdatedUnixNs),
				checksum:    hdr.Checksum,
			})
		case recordOpDelete:
			b.index.Delete(key)
		default:
			return offset, errCorruptRecord
		}

		offset = dataEnd
	}
	return offset, nil
}

func (b *bucket) appendRecord(op uint8, key, contentType string, data []byte, checksum uint32, updated time.Time) (int64, int64, error) {
	b.logMu.Lock()
	defer b.logMu.Unlock()

	if b.segmentFile == nil {
		return 0, 0, fmt.Errorf("usagi: segment not open")
	}
	recordSize := int64(recordHeaderSize + len(key) + len(contentType) + len(data))
	if b.store.segmentSize > 0 && b.currentSegmentSize+recordSize > b.store.segmentSize {
		if err := b.rotateSegment(); err != nil {
			return 0, 0, err
		}
	}
	off, err := b.segmentFile.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, 0, fmt.Errorf("usagi: seek segment: %w", err)
	}

	hdr := recordHeader{
		Magic:          recordMagic,
		Version:        recordVersion,
		Op:             op,
		KeyLen:         uint32(len(key)),
		ContentTypeLen: uint16(len(contentType)),
		DataLen:        uint64(len(data)),
		UpdatedUnixNs:  updated.UnixNano(),
		Checksum:       checksum,
	}
	headerBuf := encodeHeader(hdr, nil)
	if _, err := b.segmentFile.Write(headerBuf); err != nil {
		return 0, 0, fmt.Errorf("usagi: write header: %w", err)
	}
	if _, err := b.segmentFile.Write([]byte(key)); err != nil {
		return 0, 0, fmt.Errorf("usagi: write key: %w", err)
	}
	if _, err := b.segmentFile.Write([]byte(contentType)); err != nil {
		return 0, 0, fmt.Errorf("usagi: write content type: %w", err)
	}
	if len(data) > 0 {
		if _, err := b.segmentFile.Write(data); err != nil {
			return 0, 0, fmt.Errorf("usagi: write data: %w", err)
		}
	}
	if !b.store.nofsync {
		if err := b.segmentFile.Sync(); err != nil {
			return 0, 0, fmt.Errorf("usagi: sync log: %w", err)
		}
	}

	dataOffset := off + recordHeaderSize + int64(len(key)) + int64(len(contentType))
	b.currentSegmentSize = off + recordSize
	b.maybeWriteManifestLocked()
	return b.currentSegmentID, dataOffset, nil
}

func (b *bucket) close() {
	_ = b.writeManifest()
	b.logMu.Lock()
	if b.segmentFile != nil {
		b.segmentFile.Close()
		b.segmentFile = nil
	}
	b.logMu.Unlock()
}

// readAllSized reads from src, honoring size when provided.
func readAllSized(src io.Reader, size int64) ([]byte, error) {
	if size == 0 {
		return nil, nil
	}
	var buf bytes.Buffer
	if size > 0 {
		if size > int64(1<<31) {
			return nil, fmt.Errorf("usagi: object too large")
		}
		buf.Grow(int(size))
	}
	if _, err := io.Copy(&buf, src); err != nil {
		return nil, fmt.Errorf("usagi: read data: %w", err)
	}
	return buf.Bytes(), nil
}

type readCloser struct {
	*io.SectionReader
	closer io.Closer
}

func (rc *readCloser) Close() error {
	return rc.closer.Close()
}

// objectIter implements storage.ObjectIter.
type objectIter struct {
	items []*storage.Object
	idx   int
}

func (it *objectIter) Next() (*storage.Object, error) {
	if it.idx >= len(it.items) {
		return nil, nil
	}
	item := it.items[it.idx]
	it.idx++
	return item, nil
}

func (it *objectIter) Close() error {
	return nil
}

// Ensure the multipart directory exists.
func (b *bucket) ensureMultipartDir() error {
	return os.MkdirAll(b.multipartDir, defaultPermissions)
}

func (b *bucket) multipartPath(uploadID string, partNumber int) string {
	return filepath.Join(b.multipartDir, uploadID, fmt.Sprintf("part-%06d", partNumber))
}

func (b *bucket) uploadDir(uploadID string) string {
	return filepath.Join(b.multipartDir, uploadID)
}

// Helper to assemble parts into a single buffer.
func assembleParts(parts []*multipartPart) ([]byte, error) {
	var total int64
	for _, p := range parts {
		total += p.size
	}
	if total > int64(1<<31) {
		return nil, fmt.Errorf("usagi: multipart object too large")
	}
	buf := make([]byte, 0, total)
	for _, p := range parts {
		data, err := os.ReadFile(p.path)
		if err != nil {
			return nil, fmt.Errorf("usagi: read part: %w", err)
		}
		buf = append(buf, data...)
	}
	return buf, nil
}

func validateKey(key string) error {
	if strings.TrimSpace(key) == "" {
		return fmt.Errorf("usagi: empty key")
	}
	return nil
}
