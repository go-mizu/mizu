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

const maxInlineRecordBytes = 256 * 1024

var recordBufPool = sync.Pool{
	New: func() any {
		return make([]byte, 0, maxInlineRecordBytes)
	},
}

var copyBufPool = sync.Pool{
	New: func() any {
		return make([]byte, 256*1024)
	},
}

type entry struct {
	shard       int
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

	index *shardedIndex

	features storage.Features

	loadOnce sync.Once
	loadErr  error

	segmentShards int
	writers       []*segmentWriter

	manifestMu   sync.Mutex
	lastManifest time.Time

	prefixIndex *prefixIndex
	smallCache  *smallCache

	multipartMu      sync.Mutex
	multipartDir     string
	multipartUploads map[string]*multipartUpload

	segmentReaders *segmentReaderPools
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

	updated := time.Now()
	if size >= 0 && b.smallCache != nil && size <= b.smallCache.maxItem {
		data, err := readAllSized(src, size)
		if err != nil {
			return nil, err
		}
		sz := int64(len(data))
		checksum := crc32.Checksum(data, crcTable)
		shard, segID, off, err := b.appendRecord(recordOpPut, key, contentType, data, checksum, updated)
		if err != nil {
			return nil, err
		}
		entry := &entry{
			shard:       shard,
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
		b.smallCache.Put(key, data)
		b.maybeWriteManifest()
		return &storage.Object{
			Bucket:      b.name,
			Key:         key,
			Size:        sz,
			ContentType: contentType,
			Updated:     updated,
		}, nil
	}

	if size >= 0 && size <= maxInlineRecordBytes {
		data, err := readAllSized(src, size)
		if err != nil {
			return nil, err
		}
		sz := int64(len(data))
		checksum := crc32.Checksum(data, crcTable)
		shard, segID, off, err := b.appendRecord(recordOpPut, key, contentType, data, checksum, updated)
		if err != nil {
			return nil, err
		}
		entry := &entry{
			shard:       shard,
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
		if b.smallCache != nil && sz <= b.smallCache.maxItem {
			b.smallCache.Put(key, data)
		}
		b.maybeWriteManifest()
		return &storage.Object{
			Bucket:      b.name,
			Key:         key,
			Size:        sz,
			ContentType: contentType,
			Updated:     updated,
		}, nil
	}

	if size >= 0 {
		shard, segID, off, sz, checksum, err := b.appendRecordStream(key, contentType, src, size, updated)
		if err != nil {
			return nil, err
		}
		entry := &entry{
			shard:       shard,
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
		b.maybeWriteManifest()
		return &storage.Object{
			Bucket:      b.name,
			Key:         key,
			Size:        sz,
			ContentType: contentType,
			Updated:     updated,
		}, nil
	}

	data, err := readAllSized(src, size)
	if err != nil {
		return nil, err
	}
	sz := int64(len(data))
	checksum := crc32.Checksum(data, crcTable)
	shard, segID, off, err := b.appendRecord(recordOpPut, key, contentType, data, checksum, updated)
	if err != nil {
		return nil, err
	}
	entry := &entry{
		shard:       shard,
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
	if b.smallCache != nil && sz <= b.smallCache.maxItem {
		b.smallCache.Put(key, data)
	}
	b.maybeWriteManifest()

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

	if b.smallCache != nil {
		if data, ok := b.smallCache.Get(key); ok {
			if offset < 0 {
				offset = 0
			}
			readLen := int64(len(data)) - offset
			if readLen < 0 {
				return nil, nil, storage.ErrNotExist
			}
			if length > 0 && length < readLen {
				readLen = length
			}
			if length == 0 {
				readLen = int64(len(data)) - offset
			}
			if length < 0 {
				readLen = int64(len(data)) - offset
			}
			start := int(offset)
			end := start + int(readLen)
			if start < 0 {
				start = 0
			}
			if end > len(data) {
				end = len(data)
			}
			reader := bytes.NewReader(data[start:end])
			return io.NopCloser(reader), &storage.Object{
				Bucket:      b.name,
				Key:         key,
				Size:        e.size,
				ContentType: e.contentType,
				Updated:     e.updated,
			}, nil
		}
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

	file, release, err := b.getSegmentReader(e.shard, e.segmentID)
	if err != nil {
		return nil, nil, fmt.Errorf("usagi: open segment: %w", err)
	}
	reader := io.NewSectionReader(file, start, readLen)
	return &readCloser{SectionReader: reader, release: release}, &storage.Object{
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

	_, _, _, err := b.appendRecord(recordOpDelete, key, "", nil, 0, time.Now())
	if err != nil {
		return err
	}
	b.index.Delete(key)
	if b.prefixIndex != nil {
		b.prefixIndex.Remove(key)
	}
	b.maybeWriteManifest()
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
		if keys, ok := b.prefixIndex.Candidates(prefix); ok {
			keys = prefixSlice(keys, prefix)
			keys = applyOffsetLimit(keys, offset, limit)
			return b.keysToIter(keys)
		}
	}
	keys := b.index.KeysView(prefix)
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
	return &objectIter{bucket: b, keys: keys}, nil
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
	if b.segmentShards < 1 {
		b.segmentShards = 1
	}
	if len(b.writers) != b.segmentShards {
		b.writers = make([]*segmentWriter, b.segmentShards)
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

	if err := b.openLastSegments(); err != nil {
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
	firstSegmentPath := b.segmentPath(0, 1)
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
				shard:       v.Shard,
				segmentID:   v.SegmentID,
				offset:      v.Offset,
				size:        v.Size,
				contentType: v.ContentType,
				updated:     time.Unix(0, v.UpdatedUnix),
				checksum:    v.Checksum,
			})
		}
		last := make(map[int]manifestSegment)
		for _, seg := range m.LastSegments {
			if seg.Shard < 0 {
				continue
			}
			last[seg.Shard] = seg
		}
		if err := b.replaySegmentsAfter(last); err != nil {
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
	refs, err := b.listSegments()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("usagi: list segments: %w", err)
	}
	for _, ref := range refs {
		path := b.segmentPath(ref.shard, ref.id)
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("usagi: open segment: %w", err)
		}
		info, err := file.Stat()
		if err != nil {
			file.Close()
			return fmt.Errorf("usagi: stat segment: %w", err)
		}
		_, err = b.rebuildIndex(file, info.Size(), 0, ref.shard, ref.id)
		file.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *bucket) replaySegmentsAfter(last map[int]manifestSegment) error {
	refs, err := b.listSegments()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("usagi: list segments: %w", err)
	}
	for _, ref := range refs {
		path := b.segmentPath(ref.shard, ref.id)
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
		lastSeg, ok := last[ref.shard]
		if ok {
			if ref.id < lastSeg.ID {
				file.Close()
				continue
			}
			if ref.id == lastSeg.ID {
				start = lastSeg.Size
			}
		}
		_, err = b.rebuildIndex(file, info.Size(), start, ref.shard, ref.id)
		file.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *bucket) openLastSegments() error {
	refs, err := b.listSegments()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("usagi: list segments: %w", err)
	}
	byShard := make(map[int][]segmentRef)
	for _, ref := range refs {
		byShard[ref.shard] = append(byShard[ref.shard], ref)
	}
	for shard := 0; shard < b.segmentShards; shard++ {
		list := byShard[shard]
		w := b.writers[shard]
		if w == nil {
			w = &segmentWriter{shard: shard}
			b.writers[shard] = w
		}
		if len(list) == 0 {
			w.mu.Lock()
			err := b.openSegmentLocked(w, 1)
			w.mu.Unlock()
			if err != nil {
				return err
			}
			continue
		}
		last := list[len(list)-1]
		path := b.segmentPath(last.shard, last.id)
		file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
		if err != nil {
			return fmt.Errorf("usagi: open segment: %w", err)
		}
		info, err := file.Stat()
		if err != nil {
			file.Close()
			return fmt.Errorf("usagi: stat segment: %w", err)
		}
		w.file = file
		w.id = last.id
		w.size = info.Size()
	}
	return nil
}

func (b *bucket) rebuildIndex(file *os.File, size int64, start int64, shard int, segmentID int64) (int64, error) {
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
				shard:       shard,
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

func getRecordBuf(total int) []byte {
	if total <= 0 || total > maxInlineRecordBytes {
		return nil
	}
	buf := recordBufPool.Get().([]byte)
	if cap(buf) < total {
		buf = make([]byte, total)
	}
	return buf[:0]
}

func putRecordBuf(buf []byte) {
	if buf == nil {
		return
	}
	if cap(buf) > maxInlineRecordBytes {
		return
	}
	recordBufPool.Put(buf[:0])
}

func (b *bucket) writerForKey(key string) (*segmentWriter, int) {
	if b.segmentShards < 1 {
		b.segmentShards = 1
	}
	shard := 0
	if b.segmentShards > 1 {
		shard = int(fnv32a(key) % uint32(b.segmentShards))
	}
	if shard < 0 || shard >= len(b.writers) {
		return nil, shard
	}
	return b.writers[shard], shard
}

func (b *bucket) rotateSegmentLocked(writer *segmentWriter) error {
	if writer.file != nil {
		_ = writer.file.Sync()
		_ = writer.file.Close()
	}
	return b.openSegmentLocked(writer, writer.id+1)
}

func (b *bucket) openSegmentLocked(writer *segmentWriter, id int64) error {
	path := b.segmentPath(writer.shard, id)
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("usagi: create segment: %w", err)
	}
	writer.file = file
	writer.id = id
	writer.size = 0
	return nil
}

func (b *bucket) maybeWriteManifest() {
	if b.store.manifestEvery <= 0 {
		return
	}
	b.manifestMu.Lock()
	defer b.manifestMu.Unlock()
	if time.Since(b.lastManifest) < b.store.manifestEvery {
		return
	}
	_ = b.writeManifest()
	b.lastManifest = time.Now()
}

func (b *bucket) appendRecord(op uint8, key, contentType string, data []byte, checksum uint32, updated time.Time) (int, int64, int64, error) {
	writer, shard := b.writerForKey(key)
	if writer == nil {
		return 0, 0, 0, fmt.Errorf("usagi: segment not open")
	}
	writer.mu.Lock()
	defer writer.mu.Unlock()

	if writer.file == nil {
		return 0, 0, 0, fmt.Errorf("usagi: segment not open")
	}
	recordSize := int64(recordHeaderSize + len(key) + len(contentType) + len(data))
	if b.store.segmentSize > 0 && writer.size+recordSize > b.store.segmentSize {
		if err := b.rotateSegmentLocked(writer); err != nil {
			return 0, 0, 0, err
		}
	}
	off, err := writer.file.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("usagi: seek segment: %w", err)
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
	totalBytes := int(recordHeaderSize) + len(key) + len(contentType) + len(data)
	inlineBuf := getRecordBuf(totalBytes)
	if inlineBuf != nil {
		inlineBuf = encodeHeader(hdr, inlineBuf)
		inlineBuf = append(inlineBuf, key...)
		inlineBuf = append(inlineBuf, contentType...)
		inlineBuf = append(inlineBuf, data...)
		if _, err := writer.file.Write(inlineBuf); err != nil {
			putRecordBuf(inlineBuf)
			return 0, 0, 0, fmt.Errorf("usagi: write record: %w", err)
		}
		putRecordBuf(inlineBuf)
	} else {
		headerBuf := encodeHeader(hdr, nil)
		if _, err := writer.file.Write(headerBuf); err != nil {
			return 0, 0, 0, fmt.Errorf("usagi: write header: %w", err)
		}
		if _, err := writer.file.Write([]byte(key)); err != nil {
			return 0, 0, 0, fmt.Errorf("usagi: write key: %w", err)
		}
		if _, err := writer.file.Write([]byte(contentType)); err != nil {
			return 0, 0, 0, fmt.Errorf("usagi: write content type: %w", err)
		}
		if len(data) > 0 {
			if _, err := writer.file.Write(data); err != nil {
				return 0, 0, 0, fmt.Errorf("usagi: write data: %w", err)
			}
		}
	}
	if !b.store.nofsync {
		if err := writer.file.Sync(); err != nil {
			return 0, 0, 0, fmt.Errorf("usagi: sync log: %w", err)
		}
	}

	dataOffset := off + recordHeaderSize + int64(len(key)) + int64(len(contentType))
	writer.size = off + recordSize
	return shard, writer.id, dataOffset, nil
}

func (b *bucket) appendRecordStream(key, contentType string, src io.Reader, size int64, updated time.Time) (int, int64, int64, int64, uint32, error) {
	writer, shard := b.writerForKey(key)
	if writer == nil {
		return 0, 0, 0, 0, 0, fmt.Errorf("usagi: segment not open")
	}
	writer.mu.Lock()
	defer writer.mu.Unlock()

	if writer.file == nil {
		return 0, 0, 0, 0, 0, fmt.Errorf("usagi: segment not open")
	}
	recordSize := int64(recordHeaderSize+len(key)+len(contentType)) + size
	if b.store.segmentSize > 0 && writer.size+recordSize > b.store.segmentSize {
		if err := b.rotateSegmentLocked(writer); err != nil {
			return 0, 0, 0, 0, 0, err
		}
	}
	off, err := writer.file.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, 0, 0, 0, 0, fmt.Errorf("usagi: seek segment: %w", err)
	}
	hdr := recordHeader{
		Magic:          recordMagic,
		Version:        recordVersion,
		Op:             recordOpPut,
		KeyLen:         uint32(len(key)),
		ContentTypeLen: uint16(len(contentType)),
		DataLen:        uint64(size),
		UpdatedUnixNs:  updated.UnixNano(),
		Checksum:       0,
	}
	headerBuf := encodeHeader(hdr, nil)
	if _, err := writer.file.Write(headerBuf); err != nil {
		return 0, 0, 0, 0, 0, fmt.Errorf("usagi: write header: %w", err)
	}
	if _, err := writer.file.Write([]byte(key)); err != nil {
		return 0, 0, 0, 0, 0, fmt.Errorf("usagi: write key: %w", err)
	}
	if _, err := writer.file.Write([]byte(contentType)); err != nil {
		return 0, 0, 0, 0, 0, fmt.Errorf("usagi: write content type: %w", err)
	}
	dataOffset := off + recordHeaderSize + int64(len(key)) + int64(len(contentType))
	hasher := crc32.New(crcTable)
	buf := copyBufPool.Get().([]byte)
	written, err := io.CopyBuffer(io.MultiWriter(writer.file, hasher), src, buf)
	copyBufPool.Put(buf)
	if err != nil {
		return 0, 0, 0, 0, 0, fmt.Errorf("usagi: stream data: %w", err)
	}
	checksum := hasher.Sum32()
	hdr.Checksum = checksum
	hdr.DataLen = uint64(written)
	headerBuf = encodeHeader(hdr, headerBuf[:0])
	if _, err := writer.file.WriteAt(headerBuf, off); err != nil {
		return 0, 0, 0, 0, 0, fmt.Errorf("usagi: patch header: %w", err)
	}
	if !b.store.nofsync {
		if err := writer.file.Sync(); err != nil {
			return 0, 0, 0, 0, 0, fmt.Errorf("usagi: sync log: %w", err)
		}
	}
	writer.size = off + int64(recordHeaderSize+len(key)+len(contentType)) + written
	return shard, writer.id, dataOffset, written, checksum, nil
}

func (b *bucket) close() {
	_ = b.writeManifest()
	for _, w := range b.writers {
		if w == nil {
			continue
		}
		w.mu.Lock()
		if w.file != nil {
			_ = w.file.Close()
			w.file = nil
		}
		w.mu.Unlock()
	}

	if b.segmentReaders != nil {
		b.segmentReaders.closeAll()
	}
}

func (b *bucket) getSegmentReader(shard int, id int64) (*os.File, func(), error) {
	key := segmentKey(shard, id)
	if b.segmentReaders == nil {
		f, err := os.Open(b.segmentPath(shard, id))
		if err != nil {
			return nil, nil, err
		}
		return f, func() { _ = f.Close() }, nil
	}
	return b.segmentReaders.get(key, func() (*os.File, error) {
		return os.Open(b.segmentPath(shard, id))
	})
}

// readAllSized reads from src, honoring size when provided.
func readAllSized(src io.Reader, size int64) ([]byte, error) {
	if size == 0 {
		return nil, nil
	}
	if size > 0 {
		if size > int64(1<<31) {
			return nil, fmt.Errorf("usagi: object too large")
		}
		data := make([]byte, size)
		n, err := io.ReadFull(src, data)
		if err != nil {
			if err == io.ErrUnexpectedEOF || err == io.EOF {
				return data[:n], nil
			}
			return nil, fmt.Errorf("usagi: read data: %w", err)
		}
		return data[:n], nil
	}
	data, err := io.ReadAll(src)
	if err != nil {
		return nil, fmt.Errorf("usagi: read data: %w", err)
	}
	return data, nil
}

type readCloser struct {
	*io.SectionReader
	release func()
}

func (rc *readCloser) Close() error {
	if rc.release != nil {
		rc.release()
	}
	return nil
}

// objectIter implements storage.ObjectIter.
type objectIter struct {
	bucket *bucket
	keys   []string
	idx    int
}

func (it *objectIter) Next() (*storage.Object, error) {
	for it.idx < len(it.keys) {
		key := it.keys[it.idx]
		it.idx++
		if e, ok := it.bucket.index.Get(key); ok {
			return &storage.Object{
				Bucket:      it.bucket.name,
				Key:         key,
				Size:        e.size,
				ContentType: e.contentType,
				Updated:     e.updated,
			}, nil
		}
	}
	return nil, nil
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
