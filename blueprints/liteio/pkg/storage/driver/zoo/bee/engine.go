package bee

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/liteio-dev/liteio/pkg/storage"
)

const (
	nodeMagic      = "BEELOG01"
	nodeVersion    = uint32(1)
	nodeHeaderSize = int64(16)

	recPut    byte = 1
	recDelete byte = 2

	// type(1) + crc(4) + bucketLen(2) + keyLen(2) + contentTypeLen(2) + valueLen(8) + timestamp(8)
	recFixedSize = 27

	defaultInlineLimit = 64 * 1024
)

type nodeEntry struct {
	valueOffset int64
	size        int64
	contentType string
	created     int64
	updated     int64
	inline      []byte
}

type nodeListItem struct {
	key   string
	entry *nodeEntry
}

type nodeEngine struct {
	id          int
	path        string
	fd          *os.File
	tail        atomic.Int64
	syncMode    string
	inlineLimit int64
	crcTable    *crc32.Table

	appendMu sync.Mutex
	idxMu    sync.RWMutex
	idx      map[string]*nodeEntry
	buckets  map[string]map[string]struct{}

	closed atomic.Bool

	batchStop chan struct{}
	batchWg   sync.WaitGroup
}

func openNodeEngine(id int, path, syncMode string, inlineLimit int64) (*nodeEngine, error) {
	if inlineLimit <= 0 {
		inlineLimit = defaultInlineLimit
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("bee: mkdir node dir %q: %w", dir, err)
	}

	fd, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("bee: open node log %q: %w", path, err)
	}

	n := &nodeEngine{
		id:          id,
		path:        path,
		fd:          fd,
		syncMode:    syncMode,
		inlineLimit: inlineLimit,
		crcTable:    crc32.MakeTable(crc32.IEEE),
		idx:         make(map[string]*nodeEntry, 1024),
		buckets:     make(map[string]map[string]struct{}, 16),
	}

	if err := n.initAndRecover(); err != nil {
		fd.Close()
		return nil, err
	}

	if syncMode == "batch" {
		n.batchStop = make(chan struct{})
		n.batchWg.Add(1)
		go n.batchSyncLoop()
	}

	return n, nil
}

func (n *nodeEngine) initAndRecover() error {
	info, err := n.fd.Stat()
	if err != nil {
		return fmt.Errorf("bee: stat node log %q: %w", n.path, err)
	}

	if info.Size() == 0 {
		h := make([]byte, nodeHeaderSize)
		copy(h[:8], nodeMagic)
		binary.LittleEndian.PutUint32(h[8:12], nodeVersion)
		if _, err := n.fd.WriteAt(h, 0); err != nil {
			return fmt.Errorf("bee: write node header %q: %w", n.path, err)
		}
		n.tail.Store(nodeHeaderSize)
		return nil
	}

	if info.Size() < nodeHeaderSize {
		return fmt.Errorf("bee: invalid node log %q: too small", n.path)
	}

	h := make([]byte, nodeHeaderSize)
	if _, err := n.fd.ReadAt(h, 0); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("bee: read node header %q: %w", n.path, err)
	}
	if string(h[:8]) != nodeMagic {
		return fmt.Errorf("bee: invalid node magic in %q", n.path)
	}
	if binary.LittleEndian.Uint32(h[8:12]) != nodeVersion {
		return fmt.Errorf("bee: unsupported node version in %q", n.path)
	}

	return n.recover(info.Size())
}

func (n *nodeEngine) recover(fileSize int64) error {
	pos := nodeHeaderSize
	validTail := pos

	fixed := make([]byte, recFixedSize)

	for pos+recFixedSize <= fileSize {
		if _, err := n.fd.ReadAt(fixed, pos); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				break
			}
			return fmt.Errorf("bee: recover read fixed at %d: %w", pos, err)
		}

		recType := fixed[0]
		if recType != recPut && recType != recDelete {
			break
		}

		storedCRC := binary.LittleEndian.Uint32(fixed[1:5])
		bl := int(binary.LittleEndian.Uint16(fixed[5:7]))
		kl := int(binary.LittleEndian.Uint16(fixed[7:9]))
		cl := int(binary.LittleEndian.Uint16(fixed[9:11]))
		vl := int64(binary.LittleEndian.Uint64(fixed[11:19]))
		ts := int64(binary.LittleEndian.Uint64(fixed[19:27]))

		if bl < 0 || kl < 0 || cl < 0 || vl < 0 {
			break
		}

		payloadLen := int64(bl + kl + cl)
		if vl > 0 {
			payloadLen += vl
		}
		total := int64(recFixedSize) + payloadLen
		if total <= int64(recFixedSize) || pos+total > fileSize {
			break
		}

		payload := make([]byte, payloadLen)
		if _, err := n.fd.ReadAt(payload, pos+int64(recFixedSize)); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				break
			}
			return fmt.Errorf("bee: recover read payload at %d: %w", pos, err)
		}

		h := crc32.New(n.crcTable)
		_, _ = h.Write(fixed[5:])
		_, _ = h.Write(payload)
		if h.Sum32() != storedCRC {
			break
		}

		off := 0
		bucket := string(payload[off : off+bl])
		off += bl
		key := string(payload[off : off+kl])
		off += kl
		contentType := string(payload[off : off+cl])
		off += cl

		if recType == recPut {
			valueOffset := pos + int64(recFixedSize+bl+kl+cl)
			var inline []byte
			if vl > 0 && vl <= n.inlineLimit {
				inline = make([]byte, vl)
				copy(inline, payload[off:off+int(vl)])
			}
			n.applyPut(bucket, key, contentType, valueOffset, vl, ts, inline)
		} else {
			n.applyDelete(bucket, key, ts)
		}

		validTail = pos + total
		pos = validTail
	}

	if validTail < fileSize {
		if err := n.fd.Truncate(validTail); err != nil {
			return fmt.Errorf("bee: truncate recovered log %q: %w", n.path, err)
		}
	}

	n.tail.Store(validTail)
	return nil
}

func (n *nodeEngine) batchSyncLoop() {
	defer n.batchWg.Done()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-n.batchStop:
			_ = n.fd.Sync()
			return
		case <-ticker.C:
			_ = n.fd.Sync()
		}
	}
}

func (n *nodeEngine) appendRecord(recType byte, bucket, key, contentType string, value []byte, ts int64) (int64, int64, error) {
	if n.closed.Load() {
		return 0, 0, fmt.Errorf("bee: node %d is closed", n.id)
	}

	bl := len(bucket)
	kl := len(key)
	cl := len(contentType)
	vl := len(value)

	total := int64(recFixedSize + bl + kl + cl + vl)
	buf := make([]byte, total)

	buf[0] = recType
	binary.LittleEndian.PutUint16(buf[5:7], uint16(bl))
	binary.LittleEndian.PutUint16(buf[7:9], uint16(kl))
	binary.LittleEndian.PutUint16(buf[9:11], uint16(cl))
	binary.LittleEndian.PutUint64(buf[11:19], uint64(vl))
	binary.LittleEndian.PutUint64(buf[19:27], uint64(ts))

	pos := recFixedSize
	copy(buf[pos:pos+bl], bucket)
	pos += bl
	copy(buf[pos:pos+kl], key)
	pos += kl
	copy(buf[pos:pos+cl], contentType)
	pos += cl
	if vl > 0 {
		copy(buf[pos:pos+vl], value)
	}

	crc := crc32.Checksum(buf[5:], n.crcTable)
	binary.LittleEndian.PutUint32(buf[1:5], crc)

	n.appendMu.Lock()
	offset := n.tail.Load()
	_, err := n.fd.WriteAt(buf, offset)
	if err == nil {
		n.tail.Store(offset + total)
		if n.syncMode == "full" {
			err = n.fd.Sync()
		}
	}
	n.appendMu.Unlock()

	if err != nil {
		return 0, 0, fmt.Errorf("bee: write node record at %d: %w", offset, err)
	}

	valueOffset := offset + int64(recFixedSize+bl+kl+cl)
	return offset, valueOffset, nil
}

func (n *nodeEngine) applyPut(bucket, key, contentType string, valueOffset, size, ts int64, inline []byte) *nodeEntry {
	ck := compositeKey(bucket, key)

	old, exists := n.idx[ck]
	if exists && old.updated > ts {
		return old
	}

	created := ts
	if exists {
		created = old.created
	}

	e := &nodeEntry{
		valueOffset: valueOffset,
		size:        size,
		contentType: contentType,
		created:     created,
		updated:     ts,
		inline:      inline,
	}
	n.idx[ck] = e

	kb, ok := n.buckets[bucket]
	if !ok {
		kb = make(map[string]struct{}, 128)
		n.buckets[bucket] = kb
	}
	kb[key] = struct{}{}

	return e
}

func (n *nodeEngine) applyDelete(bucket, key string, ts int64) bool {
	ck := compositeKey(bucket, key)
	old, exists := n.idx[ck]
	if !exists {
		return false
	}
	if old.updated > ts {
		return false
	}

	delete(n.idx, ck)
	if kb, ok := n.buckets[bucket]; ok {
		delete(kb, key)
		if len(kb) == 0 {
			delete(n.buckets, bucket)
		}
	}
	return true
}

func (n *nodeEngine) write(bucket, key, contentType string, data []byte, ts int64) (*nodeEntry, error) {
	if bucket == "" || key == "" {
		return nil, fmt.Errorf("bee: empty bucket or key")
	}

	_, valueOffset, err := n.appendRecord(recPut, bucket, key, contentType, data, ts)
	if err != nil {
		return nil, err
	}

	var inline []byte
	if int64(len(data)) <= n.inlineLimit {
		inline = make([]byte, len(data))
		copy(inline, data)
	}

	n.idxMu.Lock()
	e := n.applyPut(bucket, key, contentType, valueOffset, int64(len(data)), ts, inline)
	out := cloneEntry(e)
	n.idxMu.Unlock()

	return out, nil
}

func (n *nodeEngine) delete(bucket, key string, ts int64) (bool, error) {
	if bucket == "" || key == "" {
		return false, fmt.Errorf("bee: empty bucket or key")
	}

	_, _, err := n.appendRecord(recDelete, bucket, key, "", nil, ts)
	if err != nil {
		return false, err
	}

	n.idxMu.Lock()
	existed := n.applyDelete(bucket, key, ts)
	n.idxMu.Unlock()

	if !existed {
		return false, storage.ErrNotExist
	}
	return true, nil
}

func (n *nodeEngine) read(bucket, key string) ([]byte, *nodeEntry, error) {
	if n.closed.Load() {
		return nil, nil, fmt.Errorf("bee: node %d is closed", n.id)
	}

	n.idxMu.RLock()
	e, ok := n.idx[compositeKey(bucket, key)]
	if !ok {
		n.idxMu.RUnlock()
		return nil, nil, storage.ErrNotExist
	}
	meta := cloneEntry(e)
	n.idxMu.RUnlock()

	if meta.inline != nil {
		return meta.inline, meta, nil
	}
	if meta.size < 0 {
		return nil, nil, fmt.Errorf("bee: invalid size for %s/%s", bucket, key)
	}

	data := make([]byte, meta.size)
	if meta.size == 0 {
		return data, meta, nil
	}

	if _, err := n.fd.ReadAt(data, meta.valueOffset); err != nil && !errors.Is(err, io.EOF) {
		return nil, nil, fmt.Errorf("bee: read value at %d: %w", meta.valueOffset, err)
	}
	return data, meta, nil
}

func (n *nodeEngine) stat(bucket, key string) (*nodeEntry, error) {
	if n.closed.Load() {
		return nil, fmt.Errorf("bee: node %d is closed", n.id)
	}

	n.idxMu.RLock()
	e, ok := n.idx[compositeKey(bucket, key)]
	n.idxMu.RUnlock()
	if !ok {
		return nil, storage.ErrNotExist
	}
	return cloneEntry(e), nil
}

func (n *nodeEngine) hasBucket(bucket string) bool {
	n.idxMu.RLock()
	kb := n.buckets[bucket]
	ok := len(kb) > 0
	n.idxMu.RUnlock()
	return ok
}

func (n *nodeEngine) bucketNames() []string {
	n.idxMu.RLock()
	names := make([]string, 0, len(n.buckets))
	for name, keys := range n.buckets {
		if len(keys) > 0 {
			names = append(names, name)
		}
	}
	n.idxMu.RUnlock()
	sort.Strings(names)
	return names
}

func (n *nodeEngine) list(bucket, prefix string, recursive bool) []nodeListItem {
	n.idxMu.RLock()
	keys := n.buckets[bucket]
	if len(keys) == 0 {
		n.idxMu.RUnlock()
		return nil
	}

	matched := make([]string, 0, len(keys))
	for key := range keys {
		if prefix != "" && !strings.HasPrefix(key, prefix) {
			continue
		}
		if !recursive {
			rest := strings.TrimPrefix(key, prefix)
			rest = strings.TrimPrefix(rest, "/")
			if strings.Contains(rest, "/") {
				continue
			}
		}
		matched = append(matched, key)
	}
	sort.Strings(matched)

	out := make([]nodeListItem, 0, len(matched))
	for _, key := range matched {
		e, ok := n.idx[compositeKey(bucket, key)]
		if !ok {
			continue
		}
		out = append(out, nodeListItem{key: key, entry: cloneEntry(e)})
	}
	n.idxMu.RUnlock()
	return out
}

func (n *nodeEngine) deleteBucket(bucket string, ts int64) {
	n.idxMu.RLock()
	keysMap := n.buckets[bucket]
	keys := make([]string, 0, len(keysMap))
	for key := range keysMap {
		keys = append(keys, key)
	}
	n.idxMu.RUnlock()

	for _, key := range keys {
		_, _ = n.delete(bucket, key, ts)
	}
}

func (n *nodeEngine) close() error {
	if !n.closed.CompareAndSwap(false, true) {
		return nil
	}

	if n.batchStop != nil {
		close(n.batchStop)
		n.batchWg.Wait()
	}

	if n.syncMode != "none" {
		_ = n.fd.Sync()
	}

	return n.fd.Close()
}

func compositeKey(bucket, key string) string {
	return bucket + "\x00" + key
}

func cloneEntry(e *nodeEntry) *nodeEntry {
	if e == nil {
		return nil
	}
	c := *e
	return &c
}
