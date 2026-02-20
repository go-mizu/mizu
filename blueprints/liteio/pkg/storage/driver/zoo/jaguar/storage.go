// Package jaguar implements a storage driver inspired by the Jungle paper
// (HotStorage 2019, eBay). It uses a Write-Ahead Log and a single-level
// Copy-on-Write B+-tree for durable, low-write-amplification object storage.
//
// DSN format:
//
//	jaguar:///path/to/data
//	jaguar:///path/to/data?sync=none&memtable_size=4194304&wal=true
package jaguar

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/liteio-dev/liteio/pkg/storage"
)

func init() {
	storage.Register("jaguar", &driver{})
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const (
	defaultMemtableSize = 4 * 1024 * 1024 // 4 MB
	dirPerms            = 0750
	filePerms           = 0600

	// WAL entry types.
	walPut    byte = 0x01
	walDelete byte = 0x02

	// B+-tree node types.
	nodeLeaf  byte = 0x01
	nodeInner byte = 0x02

	// B+-tree file header.
	treeMagic   = "JAGUAR01"
	headerSize  = 24 // 8 magic + 8 root offset + 8 node count
	maxFanout   = 128
	maxLeafKeys = 128

	// Multipart limits.
	maxPartNumber = 10000
)

// ---------------------------------------------------------------------------
// Driver
// ---------------------------------------------------------------------------

type driver struct{}

func (d *driver) Open(ctx context.Context, dsn string) (storage.Storage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	root, opts, err := parseDSN(dsn)
	if err != nil {
		return nil, err
	}

	syncMode := opts.Get("sync")
	if syncMode == "" {
		syncMode = "none"
	}

	memSize := int64(defaultMemtableSize)
	if ms := opts.Get("memtable_size"); ms != "" {
		if n, err := strconv.ParseInt(ms, 10, 64); err == nil && n > 0 {
			memSize = n
		}
	}

	walEnabled := true
	if w := opts.Get("wal"); w == "false" {
		walEnabled = false
	}

	if err := os.MkdirAll(root, dirPerms); err != nil {
		return nil, fmt.Errorf("jaguar: mkdir root: %w", err)
	}

	st := &store{
		root:       root,
		syncMode:   syncMode,
		walEnabled: walEnabled,
		memLimit:   memSize,
		buckets:    make(map[string]time.Time),
		mpUploads:  make(map[string]*multipartUpload),
	}

	st.mem.entries = make(map[string]*memEntry)

	// Load metadata if it exists.
	if err := st.loadMeta(); err != nil {
		return nil, err
	}

	// Open or create the tree file.
	if err := st.openTree(); err != nil {
		return nil, err
	}

	// Open WAL and replay.
	if walEnabled {
		if err := st.openWAL(); err != nil {
			return nil, err
		}
	}

	return st, nil
}

func parseDSN(dsn string) (string, url.Values, error) {
	if dsn == "" {
		return "", nil, errors.New("jaguar: empty dsn")
	}

	var queryStr string
	if idx := strings.Index(dsn, "?"); idx >= 0 {
		queryStr = dsn[idx+1:]
		dsn = dsn[:idx]
	}
	opts, _ := url.ParseQuery(queryStr)

	if strings.HasPrefix(dsn, "jaguar:") {
		rest := strings.TrimPrefix(dsn, "jaguar:")
		if strings.HasPrefix(rest, "//") {
			rest = strings.TrimPrefix(rest, "//")
		}
		if rest == "" {
			return "", nil, errors.New("jaguar: missing path in dsn")
		}
		return filepath.Clean(rest), opts, nil
	}
	if strings.HasPrefix(dsn, "/") {
		return filepath.Clean(dsn), opts, nil
	}
	return "", nil, fmt.Errorf("jaguar: unsupported dsn %q", dsn)
}

// ---------------------------------------------------------------------------
// Metadata
// ---------------------------------------------------------------------------

type metaFile struct {
	RootOffset int64             `json:"root_offset"`
	NodeCount  int64             `json:"node_count"`
	WALPos     int64             `json:"wal_pos"`
	Buckets    map[string]string `json:"buckets"` // name -> RFC3339Nano
}

func (s *store) metaPath() string { return filepath.Join(s.root, "meta.json") }
func (s *store) walPath() string  { return filepath.Join(s.root, "wal.log") }
func (s *store) treePath() string { return filepath.Join(s.root, "level1.tree") }

func (s *store) loadMeta() error {
	data, err := os.ReadFile(s.metaPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("jaguar: read meta: %w", err)
	}

	var m metaFile
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("jaguar: parse meta: %w", err)
	}

	s.treeRoot = m.RootOffset
	s.nodeCount = m.NodeCount
	s.walPos = m.WALPos

	for name, ts := range m.Buckets {
		t, err := time.Parse(time.RFC3339Nano, ts)
		if err != nil {
			t = time.Now()
		}
		s.buckets[name] = t
	}
	return nil
}

func (s *store) saveMeta() error {
	m := metaFile{
		RootOffset: s.treeRoot,
		NodeCount:  s.nodeCount,
		WALPos:     s.walPos,
		Buckets:    make(map[string]string, len(s.buckets)),
	}
	for name, t := range s.buckets {
		m.Buckets[name] = t.Format(time.RFC3339Nano)
	}

	data, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("jaguar: marshal meta: %w", err)
	}

	tmp := s.metaPath() + ".tmp"
	if err := os.WriteFile(tmp, data, filePerms); err != nil {
		return fmt.Errorf("jaguar: write meta tmp: %w", err)
	}
	if err := os.Rename(tmp, s.metaPath()); err != nil {
		return fmt.Errorf("jaguar: rename meta: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// WAL
// ---------------------------------------------------------------------------

func (s *store) openWAL() error {
	f, err := os.OpenFile(s.walPath(), os.O_CREATE|os.O_RDWR, filePerms)
	if err != nil {
		return fmt.Errorf("jaguar: open wal: %w", err)
	}
	s.walFile = f

	// Replay WAL from s.walPos to end.
	if _, err := f.Seek(s.walPos, io.SeekStart); err != nil {
		return fmt.Errorf("jaguar: seek wal: %w", err)
	}

	for {
		entry, n, err := readWALEntry(f)
		if err != nil {
			break // EOF or corrupt tail
		}
		s.walPos += int64(n)

		switch entry.typ {
		case walPut:
			s.mem.put(entry.key, entry.value, entry.contentType, entry.ts, false)
		case walDelete:
			s.mem.put(entry.key, nil, "", entry.ts, true)
		}
	}

	// Seek to end for future appends.
	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		return fmt.Errorf("jaguar: seek wal end: %w", err)
	}
	return nil
}

type walEntry struct {
	typ         byte
	key         string
	contentType string
	value       []byte
	ts          int64
}

func readWALEntry(r io.Reader) (*walEntry, int, error) {
	var hdr [1]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return nil, 0, err
	}
	n := 1

	e := &walEntry{typ: hdr[0]}

	// keyLen
	var buf2 [2]byte
	if _, err := io.ReadFull(r, buf2[:]); err != nil {
		return nil, n, err
	}
	n += 2
	kl := int(binary.BigEndian.Uint16(buf2[:]))

	keyBuf := make([]byte, kl)
	if _, err := io.ReadFull(r, keyBuf); err != nil {
		return nil, n, err
	}
	n += kl
	e.key = string(keyBuf)

	// ctLen
	if _, err := io.ReadFull(r, buf2[:]); err != nil {
		return nil, n, err
	}
	n += 2
	cl := int(binary.BigEndian.Uint16(buf2[:]))

	ctBuf := make([]byte, cl)
	if _, err := io.ReadFull(r, ctBuf); err != nil {
		return nil, n, err
	}
	n += cl
	e.contentType = string(ctBuf)

	// valLen
	var buf8 [8]byte
	if _, err := io.ReadFull(r, buf8[:]); err != nil {
		return nil, n, err
	}
	n += 8
	vl := binary.BigEndian.Uint64(buf8[:])

	valBuf := make([]byte, vl)
	if _, err := io.ReadFull(r, valBuf); err != nil {
		return nil, n, err
	}
	n += int(vl)
	e.value = valBuf

	// ts
	if _, err := io.ReadFull(r, buf8[:]); err != nil {
		return nil, n, err
	}
	n += 8
	e.ts = int64(binary.BigEndian.Uint64(buf8[:]))

	return e, n, nil
}

func (s *store) appendWAL(typ byte, compositeKey, contentType string, value []byte, ts int64) error {
	if !s.walEnabled || s.walFile == nil {
		return nil
	}

	kl := len(compositeKey)
	cl := len(contentType)
	vl := len(value)
	total := 1 + 2 + kl + 2 + cl + 8 + vl + 8

	buf := make([]byte, total)
	pos := 0

	buf[pos] = typ
	pos++

	binary.BigEndian.PutUint16(buf[pos:], uint16(kl))
	pos += 2
	copy(buf[pos:], compositeKey)
	pos += kl

	binary.BigEndian.PutUint16(buf[pos:], uint16(cl))
	pos += 2
	copy(buf[pos:], contentType)
	pos += cl

	binary.BigEndian.PutUint64(buf[pos:], uint64(vl))
	pos += 8
	copy(buf[pos:], value)
	pos += vl

	binary.BigEndian.PutUint64(buf[pos:], uint64(ts))

	// Serialize WAL writes so concurrent goroutines don't interleave
	// partial entries, which would corrupt the log on replay.
	s.walMu.Lock()
	_, err := s.walFile.Write(buf)
	if err != nil {
		s.walMu.Unlock()
		return fmt.Errorf("jaguar: write wal: %w", err)
	}

	if s.syncMode == "full" {
		if err := s.walFile.Sync(); err != nil {
			s.walMu.Unlock()
			return fmt.Errorf("jaguar: sync wal: %w", err)
		}
	}
	s.walMu.Unlock()
	return nil
}

// ---------------------------------------------------------------------------
// MemTable
// ---------------------------------------------------------------------------

type memEntry struct {
	key         string // composite key
	value       []byte
	contentType string
	ts          int64 // UnixNano
	tombstone   bool
}

type memTable struct {
	mu      sync.RWMutex
	entries map[string]*memEntry
	size    int64 // approximate byte size
}

func (m *memTable) put(key string, value []byte, contentType string, ts int64, tombstone bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	old, exists := m.entries[key]
	if exists {
		m.size -= int64(len(old.key) + len(old.value) + len(old.contentType))
	}

	e := &memEntry{
		key:         key,
		value:       value,
		contentType: contentType,
		ts:          ts,
		tombstone:   tombstone,
	}
	m.entries[key] = e
	m.size += int64(len(key) + len(value) + len(contentType))
}

func (m *memTable) get(key string) (*memEntry, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	e, ok := m.entries[key]
	return e, ok
}

func (m *memTable) remove(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e, ok := m.entries[key]; ok {
		m.size -= int64(len(e.key) + len(e.value) + len(e.contentType))
		delete(m.entries, key)
	}
}

func (m *memTable) snapshot() []*memEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]*memEntry, 0, len(m.entries))
	for _, e := range m.entries {
		list = append(list, e)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].key < list[j].key
	})
	return list
}

func (m *memTable) clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = make(map[string]*memEntry)
	m.size = 0
}

func (m *memTable) listPrefix(prefix string) []*memEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*memEntry
	for _, e := range m.entries {
		if strings.HasPrefix(e.key, prefix) && !e.tombstone {
			result = append(result, e)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].key < result[j].key
	})
	return result
}

// ---------------------------------------------------------------------------
// CoW B+-tree
// ---------------------------------------------------------------------------

type treeNode struct {
	typ     byte
	entries []treeEntry
	// For inner nodes only: rightmost child offset.
	rightChild int64
}

type treeEntry struct {
	key         string
	contentType string
	value       []byte
	created     int64
	updated     int64
	childOffset int64 // inner nodes only
}

func (s *store) openTree() error {
	f, err := os.OpenFile(s.treePath(), os.O_CREATE|os.O_RDWR, filePerms)
	if err != nil {
		return fmt.Errorf("jaguar: open tree: %w", err)
	}
	s.treeFile = f

	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("jaguar: stat tree: %w", err)
	}

	if info.Size() == 0 {
		// Write header.
		hdr := make([]byte, headerSize)
		copy(hdr[:8], treeMagic)
		binary.BigEndian.PutUint64(hdr[8:], 0)  // root offset = 0 (no root)
		binary.BigEndian.PutUint64(hdr[16:], 0) // node count = 0
		if _, err := f.Write(hdr); err != nil {
			return fmt.Errorf("jaguar: write tree header: %w", err)
		}
		s.treeRoot = 0
		s.nodeCount = 0
	}

	return nil
}

func (s *store) readNode(offset int64) (*treeNode, error) {
	if offset <= 0 {
		return nil, fmt.Errorf("jaguar: invalid node offset %d", offset)
	}

	if _, err := s.treeFile.Seek(offset, io.SeekStart); err != nil {
		return nil, fmt.Errorf("jaguar: seek node: %w", err)
	}

	var typBuf [1]byte
	if _, err := io.ReadFull(s.treeFile, typBuf[:]); err != nil {
		return nil, fmt.Errorf("jaguar: read node type: %w", err)
	}

	var countBuf [2]byte
	if _, err := io.ReadFull(s.treeFile, countBuf[:]); err != nil {
		return nil, fmt.Errorf("jaguar: read node count: %w", err)
	}
	count := int(binary.BigEndian.Uint16(countBuf[:]))

	node := &treeNode{
		typ:     typBuf[0],
		entries: make([]treeEntry, count),
	}

	for i := 0; i < count; i++ {
		e, err := s.readTreeEntry(node.typ)
		if err != nil {
			return nil, err
		}
		node.entries[i] = e
	}

	// Inner nodes have a trailing rightmost child offset.
	if node.typ == nodeInner {
		var buf8 [8]byte
		if _, err := io.ReadFull(s.treeFile, buf8[:]); err != nil {
			return nil, fmt.Errorf("jaguar: read right child: %w", err)
		}
		node.rightChild = int64(binary.BigEndian.Uint64(buf8[:]))
	}

	return node, nil
}

func (s *store) readTreeEntry(nodeType byte) (treeEntry, error) {
	var e treeEntry
	var buf2 [2]byte
	var buf8 [8]byte

	// keyLen + key
	if _, err := io.ReadFull(s.treeFile, buf2[:]); err != nil {
		return e, fmt.Errorf("jaguar: read entry keyLen: %w", err)
	}
	kl := int(binary.BigEndian.Uint16(buf2[:]))
	keyBuf := make([]byte, kl)
	if _, err := io.ReadFull(s.treeFile, keyBuf); err != nil {
		return e, fmt.Errorf("jaguar: read entry key: %w", err)
	}
	e.key = string(keyBuf)

	if nodeType == nodeLeaf {
		// ctLen + ct
		if _, err := io.ReadFull(s.treeFile, buf2[:]); err != nil {
			return e, fmt.Errorf("jaguar: read entry ctLen: %w", err)
		}
		cl := int(binary.BigEndian.Uint16(buf2[:]))
		ctBuf := make([]byte, cl)
		if _, err := io.ReadFull(s.treeFile, ctBuf); err != nil {
			return e, fmt.Errorf("jaguar: read entry ct: %w", err)
		}
		e.contentType = string(ctBuf)

		// valLen + value
		if _, err := io.ReadFull(s.treeFile, buf8[:]); err != nil {
			return e, fmt.Errorf("jaguar: read entry valLen: %w", err)
		}
		vl := binary.BigEndian.Uint64(buf8[:])
		if vl > 1<<30 { // sanity check: max 1GB value
			return e, fmt.Errorf("jaguar: read entry: corrupt valLen %d", vl)
		}
		valBuf := make([]byte, vl)
		if _, err := io.ReadFull(s.treeFile, valBuf); err != nil {
			return e, fmt.Errorf("jaguar: read entry value: %w", err)
		}
		e.value = valBuf

		// created + updated
		if _, err := io.ReadFull(s.treeFile, buf8[:]); err != nil {
			return e, fmt.Errorf("jaguar: read entry created: %w", err)
		}
		e.created = int64(binary.BigEndian.Uint64(buf8[:]))

		if _, err := io.ReadFull(s.treeFile, buf8[:]); err != nil {
			return e, fmt.Errorf("jaguar: read entry updated: %w", err)
		}
		e.updated = int64(binary.BigEndian.Uint64(buf8[:]))
	} else {
		// Inner: childOffset
		if _, err := io.ReadFull(s.treeFile, buf8[:]); err != nil {
			return e, fmt.Errorf("jaguar: read entry childOffset: %w", err)
		}
		e.childOffset = int64(binary.BigEndian.Uint64(buf8[:]))
	}

	return e, nil
}

func (s *store) appendNode(node *treeNode) (int64, error) {
	offset, err := s.treeFile.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, fmt.Errorf("jaguar: seek tree end: %w", err)
	}

	data := serializeNode(node)
	if _, err := s.treeFile.Write(data); err != nil {
		return 0, fmt.Errorf("jaguar: write node: %w", err)
	}

	s.nodeCount++
	return offset, nil
}

func serializeNode(node *treeNode) []byte {
	var buf bytes.Buffer
	buf.WriteByte(node.typ)

	var b2 [2]byte
	binary.BigEndian.PutUint16(b2[:], uint16(len(node.entries)))
	buf.Write(b2[:])

	var b8 [8]byte
	for _, e := range node.entries {
		binary.BigEndian.PutUint16(b2[:], uint16(len(e.key)))
		buf.Write(b2[:])
		buf.WriteString(e.key)

		if node.typ == nodeLeaf {
			binary.BigEndian.PutUint16(b2[:], uint16(len(e.contentType)))
			buf.Write(b2[:])
			buf.WriteString(e.contentType)

			binary.BigEndian.PutUint64(b8[:], uint64(len(e.value)))
			buf.Write(b8[:])
			buf.Write(e.value)

			binary.BigEndian.PutUint64(b8[:], uint64(e.created))
			buf.Write(b8[:])

			binary.BigEndian.PutUint64(b8[:], uint64(e.updated))
			buf.Write(b8[:])
		} else {
			binary.BigEndian.PutUint64(b8[:], uint64(e.childOffset))
			buf.Write(b8[:])
		}
	}

	if node.typ == nodeInner {
		binary.BigEndian.PutUint64(b8[:], uint64(node.rightChild))
		buf.Write(b8[:])
	}

	return buf.Bytes()
}

func (s *store) updateTreeHeader() error {
	hdr := make([]byte, headerSize)
	copy(hdr[:8], treeMagic)
	binary.BigEndian.PutUint64(hdr[8:], uint64(s.treeRoot))
	binary.BigEndian.PutUint64(hdr[16:], uint64(s.nodeCount))

	if _, err := s.treeFile.WriteAt(hdr, 0); err != nil {
		return fmt.Errorf("jaguar: update tree header: %w", err)
	}
	return nil
}

// treeGet searches the CoW B+-tree for a key.
func (s *store) treeGet(key string) (*treeEntry, bool) {
	s.treeMu.RLock()
	defer s.treeMu.RUnlock()

	if s.treeRoot == 0 {
		return nil, false
	}

	return s.treeSearch(s.treeRoot, key)
}

func (s *store) treeSearch(offset int64, key string) (*treeEntry, bool) {
	node, err := s.readNode(offset)
	if err != nil {
		return nil, false
	}

	if node.typ == nodeLeaf {
		idx := sort.Search(len(node.entries), func(i int) bool {
			return node.entries[i].key >= key
		})
		if idx < len(node.entries) && node.entries[idx].key == key {
			e := node.entries[idx]
			return &e, true
		}
		return nil, false
	}

	// Inner node: find child.
	childOff := node.rightChild
	for i := 0; i < len(node.entries); i++ {
		if key < node.entries[i].key {
			childOff = node.entries[i].childOffset
			break
		}
	}

	return s.treeSearch(childOff, key)
}

// treeListPrefix returns all leaf entries with the given prefix.
func (s *store) treeListPrefix(prefix string) []treeEntry {
	s.treeMu.RLock()
	defer s.treeMu.RUnlock()

	if s.treeRoot == 0 {
		return nil
	}

	var result []treeEntry
	s.treeCollect(s.treeRoot, prefix, &result)
	return result
}

func (s *store) treeCollect(offset int64, prefix string, result *[]treeEntry) {
	node, err := s.readNode(offset)
	if err != nil {
		return
	}

	if node.typ == nodeLeaf {
		for _, e := range node.entries {
			if strings.HasPrefix(e.key, prefix) {
				*result = append(*result, e)
			}
		}
		return
	}

	// Inner node: visit all children that might contain the prefix.
	for _, e := range node.entries {
		s.treeCollect(e.childOffset, prefix, result)
	}
	s.treeCollect(node.rightChild, prefix, result)
}

// flushMemtable flushes all memtable entries into the CoW B+-tree.
func (s *store) flushMemtable() error {
	entries := s.mem.snapshot()
	if len(entries) == 0 {
		return nil
	}

	s.treeMu.Lock()
	defer s.treeMu.Unlock()

	// Build leaf entries from memtable.
	var leafEntries []treeEntry
	for _, me := range entries {
		if me.tombstone {
			continue // skip tombstones for tree
		}
		leafEntries = append(leafEntries, treeEntry{
			key:         me.key,
			contentType: me.contentType,
			value:       me.value,
			created:     me.ts,
			updated:     me.ts,
		})
	}

	if len(leafEntries) == 0 && s.treeRoot == 0 {
		s.mem.clear()
		return nil
	}

	// Read existing tree entries if any.
	var existing []treeEntry
	if s.treeRoot != 0 {
		existing = s.collectAllLeafEntries(s.treeRoot)
	}

	// Merge: existing + new, new entries override existing by key.
	merged := mergeEntries(existing, leafEntries)

	// Handle tombstones: remove entries that were deleted.
	tombstones := make(map[string]bool)
	for _, me := range entries {
		if me.tombstone {
			tombstones[me.key] = true
		}
	}
	if len(tombstones) > 0 {
		filtered := merged[:0]
		for _, e := range merged {
			if !tombstones[e.key] {
				filtered = append(filtered, e)
			}
		}
		merged = filtered
	}

	// Build new tree from merged entries.
	if len(merged) == 0 {
		s.treeRoot = 0
	} else {
		rootOff, err := s.buildTree(merged)
		if err != nil {
			return err
		}
		s.treeRoot = rootOff
	}

	if err := s.updateTreeHeader(); err != nil {
		return err
	}

	if s.syncMode != "none" {
		if err := s.treeFile.Sync(); err != nil {
			return fmt.Errorf("jaguar: sync tree: %w", err)
		}
	}

	// Clear memtable and truncate WAL.
	// Hold walMu while truncating so concurrent appendWAL calls don't
	// race with the truncate/seek (locking order: flushMu -> treeMu -> walMu).
	s.mem.clear()
	if s.walFile != nil {
		s.walMu.Lock()
		if err := s.walFile.Truncate(0); err != nil {
			s.walMu.Unlock()
			return fmt.Errorf("jaguar: truncate wal: %w", err)
		}
		if _, err := s.walFile.Seek(0, io.SeekStart); err != nil {
			s.walMu.Unlock()
			return fmt.Errorf("jaguar: seek wal start: %w", err)
		}
		s.walPos = 0
		s.walMu.Unlock()
	}

	if err := s.saveMeta(); err != nil {
		return err
	}

	return nil
}

func (s *store) collectAllLeafEntries(offset int64) []treeEntry {
	node, err := s.readNode(offset)
	if err != nil {
		return nil
	}

	if node.typ == nodeLeaf {
		result := make([]treeEntry, len(node.entries))
		copy(result, node.entries)
		return result
	}

	var result []treeEntry
	for _, e := range node.entries {
		result = append(result, s.collectAllLeafEntries(e.childOffset)...)
	}
	result = append(result, s.collectAllLeafEntries(node.rightChild)...)
	return result
}

// mergeEntries merges two sorted entry slices. New entries override old.
func mergeEntries(old, new_ []treeEntry) []treeEntry {
	result := make([]treeEntry, 0, len(old)+len(new_))
	i, j := 0, 0
	for i < len(old) && j < len(new_) {
		if old[i].key < new_[j].key {
			result = append(result, old[i])
			i++
		} else if old[i].key > new_[j].key {
			result = append(result, new_[j])
			j++
		} else {
			// New overrides old, preserve created time from old.
			e := new_[j]
			e.created = old[i].created
			result = append(result, e)
			i++
			j++
		}
	}
	for ; i < len(old); i++ {
		result = append(result, old[i])
	}
	for ; j < len(new_); j++ {
		result = append(result, new_[j])
	}
	return result
}

// buildTree constructs a CoW B+-tree from sorted entries bottom-up.
func (s *store) buildTree(entries []treeEntry) (int64, error) {
	if len(entries) == 0 {
		return 0, nil
	}

	// Split entries into leaf nodes.
	var leafOffsets []int64
	var leafKeys []string

	for i := 0; i < len(entries); i += maxLeafKeys {
		end := i + maxLeafKeys
		if end > len(entries) {
			end = len(entries)
		}
		chunk := entries[i:end]

		leaf := &treeNode{
			typ:     nodeLeaf,
			entries: chunk,
		}
		off, err := s.appendNode(leaf)
		if err != nil {
			return 0, err
		}
		leafOffsets = append(leafOffsets, off)
		if len(chunk) > 0 {
			leafKeys = append(leafKeys, chunk[0].key)
		}
	}

	if len(leafOffsets) == 1 {
		return leafOffsets[0], nil
	}

	// Build inner nodes bottom-up.
	childOffsets := leafOffsets
	childKeys := leafKeys

	for len(childOffsets) > 1 {
		var newOffsets []int64
		var newKeys []string

		for i := 0; i < len(childOffsets); i += maxFanout {
			end := i + maxFanout
			if end > len(childOffsets) {
				end = len(childOffsets)
			}

			chunk := childOffsets[i:end]
			keys := childKeys[i:end]

			if len(chunk) == 1 {
				newOffsets = append(newOffsets, chunk[0])
				newKeys = append(newKeys, keys[0])
				continue
			}

			inner := &treeNode{
				typ:        nodeInner,
				rightChild: chunk[len(chunk)-1],
			}

			// Inner entries: separator keys + child offsets for all but the last child.
			for j := 1; j < len(chunk); j++ {
				inner.entries = append(inner.entries, treeEntry{
					key:         keys[j],
					childOffset: chunk[j-1],
				})
			}

			off, err := s.appendNode(inner)
			if err != nil {
				return 0, err
			}
			newOffsets = append(newOffsets, off)
			newKeys = append(newKeys, keys[0])
		}

		childOffsets = newOffsets
		childKeys = newKeys
	}

	return childOffsets[0], nil
}

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

type store struct {
	root       string
	syncMode   string
	walEnabled bool
	memLimit   int64

	mu      sync.RWMutex
	buckets map[string]time.Time

	mem memTable

	// WAL state.
	walMu   sync.Mutex // serializes WAL appends under concurrent writes
	walFile *os.File
	walPos  int64

	// Tree state.
	treeMu    sync.RWMutex
	treeFile  *os.File
	treeRoot  int64
	nodeCount int64

	// Flush serialization: prevents thundering herd when multiple goroutines
	// hit the memtable size limit simultaneously.
	flushMu sync.Mutex

	// Multipart uploads.
	mpMu      sync.RWMutex
	mpUploads map[string]*multipartUpload

	closed atomic.Bool
}

var _ storage.Storage = (*store)(nil)

// Cached time.
var cachedTimeNano atomic.Int64

func init() {
	cachedTimeNano.Store(time.Now().UnixNano())
	go func() {
		ticker := time.NewTicker(5 * time.Millisecond)
		for range ticker.C {
			cachedTimeNano.Store(time.Now().UnixNano())
		}
	}()
}

func fastNow() time.Time {
	return time.Unix(0, cachedTimeNano.Load())
}

func compositeKey(bucket, key string) string {
	return bucket + "\x00" + key
}

func splitCompositeKey(ck string) (bucket, key string) {
	idx := strings.IndexByte(ck, '\x00')
	if idx < 0 {
		return "", ck
	}
	return ck[:idx], ck[idx+1:]
}

func (s *store) Bucket(name string) storage.Bucket {
	if name == "" {
		name = "default"
	}
	name = safeBucketName(name)

	s.mu.Lock()
	if _, ok := s.buckets[name]; !ok {
		s.buckets[name] = fastNow()
	}
	s.mu.Unlock()

	return &bucket{st: s, name: name}
}

func (s *store) Buckets(ctx context.Context, limit, offset int, opts storage.Options) (storage.BucketIter, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	names := make([]string, 0, len(s.buckets))
	for name := range s.buckets {
		names = append(names, name)
	}
	s.mu.RUnlock()

	sort.Strings(names)

	s.mu.RLock()
	infos := make([]*storage.BucketInfo, 0, len(names))
	for _, name := range names {
		infos = append(infos, &storage.BucketInfo{
			Name:      name,
			CreatedAt: s.buckets[name],
		})
	}
	s.mu.RUnlock()

	if offset < 0 {
		offset = 0
	}
	if offset > len(infos) {
		offset = len(infos)
	}
	infos = infos[offset:]
	if limit > 0 && limit < len(infos) {
		infos = infos[:limit]
	}

	return &bucketIter{buckets: infos}, nil
}

func (s *store) CreateBucket(ctx context.Context, name string, opts storage.Options) (*storage.BucketInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("jaguar: bucket name required")
	}
	name = safeBucketName(name)

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.buckets[name]; ok {
		return nil, storage.ErrExist
	}

	now := fastNow()
	s.buckets[name] = now

	return &storage.BucketInfo{
		Name:      name,
		CreatedAt: now,
	}, nil
}

func (s *store) DeleteBucket(ctx context.Context, name string, opts storage.Options) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("jaguar: bucket name required")
	}
	name = safeBucketName(name)

	force := false
	if opts != nil {
		if v, ok := opts["force"].(bool); ok {
			force = v
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.buckets[name]; !ok {
		return storage.ErrNotExist
	}

	// Check if bucket has objects (unless force).
	if !force {
		prefix := name + "\x00"
		memEntries := s.mem.listPrefix(prefix)
		if len(memEntries) > 0 {
			return storage.ErrPermission
		}
		treeEntries := s.treeListPrefix(prefix)
		if len(treeEntries) > 0 {
			return storage.ErrPermission
		}
	}

	delete(s.buckets, name)
	return nil
}

func (s *store) Features() storage.Features {
	return storage.Features{
		"move":             true,
		"server_side_move": true,
		"server_side_copy": true,
		"directories":      true,
		"multipart":        true,
	}
}

func (s *store) Close() error {
	if s.closed.Swap(true) {
		return nil
	}

	// Flush memtable to tree.
	if s.mem.size > 0 {
		if err := s.flushMemtable(); err != nil {
			return fmt.Errorf("jaguar: flush on close: %w", err)
		}
	}

	if err := s.saveMeta(); err != nil {
		return err
	}

	var errs []error
	if s.walFile != nil {
		if err := s.walFile.Close(); err != nil {
			errs = append(errs, fmt.Errorf("jaguar: close wal: %w", err))
		}
	}
	if s.treeFile != nil {
		if s.syncMode != "none" {
			s.treeFile.Sync()
		}
		if err := s.treeFile.Close(); err != nil {
			errs = append(errs, fmt.Errorf("jaguar: close tree: %w", err))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// maybeFlush checks if the memtable exceeds its size limit and flushes.
// Uses TryLock to prevent thundering herd: if another goroutine is already
// flushing, this call returns immediately instead of blocking all writers.
func (s *store) maybeFlush() {
	s.mem.mu.RLock()
	sz := s.mem.size
	s.mem.mu.RUnlock()

	if sz < s.memLimit {
		return
	}

	// TryLock: if another goroutine is already flushing, skip.
	if !s.flushMu.TryLock() {
		return
	}
	defer s.flushMu.Unlock()

	// Re-check after acquiring the lock: the concurrent flusher may have
	// already drained the memtable.
	s.mem.mu.RLock()
	sz = s.mem.size
	s.mem.mu.RUnlock()

	if sz >= s.memLimit {
		s.flushMemtable()
	}
}

// ---------------------------------------------------------------------------
// Bucket
// ---------------------------------------------------------------------------

type bucket struct {
	st   *store
	name string
}

var (
	_ storage.Bucket         = (*bucket)(nil)
	_ storage.HasDirectories = (*bucket)(nil)
	_ storage.HasMultipart   = (*bucket)(nil)
)

func (b *bucket) Name() string { return b.name }

func (b *bucket) Features() storage.Features { return b.st.Features() }

func (b *bucket) Info(ctx context.Context) (*storage.BucketInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	b.st.mu.RLock()
	created, ok := b.st.buckets[b.name]
	b.st.mu.RUnlock()

	if !ok {
		return nil, storage.ErrNotExist
	}

	return &storage.BucketInfo{
		Name:      b.name,
		CreatedAt: created,
	}, nil
}

func (b *bucket) Write(ctx context.Context, key string, src io.Reader, size int64, contentType string, opts storage.Options) (*storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	key = strings.TrimSpace(key)
	if key == "" {
		return nil, errors.New("jaguar: empty key")
	}

	// Read value.
	var data []byte
	if size >= 0 {
		data = make([]byte, size)
		n, err := io.ReadFull(src, data)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return nil, fmt.Errorf("jaguar: read value: %w", err)
		}
		data = data[:n]
	} else {
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, src); err != nil {
			return nil, fmt.Errorf("jaguar: read value: %w", err)
		}
		data = buf.Bytes()
	}

	now := fastNow()
	ts := now.UnixNano()
	ck := compositeKey(b.name, key)

	// Append to WAL.
	if err := b.st.appendWAL(walPut, ck, contentType, data, ts); err != nil {
		return nil, err
	}

	// Insert into memtable.
	b.st.mem.put(ck, data, contentType, ts, false)

	// Check if flush needed.
	b.st.maybeFlush()

	return &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        int64(len(data)),
		ContentType: contentType,
		Created:     now,
		Updated:     now,
	}, nil
}

func (b *bucket) Open(ctx context.Context, key string, offset, length int64, opts storage.Options) (io.ReadCloser, *storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}

	key = strings.TrimSpace(key)
	if key == "" {
		return nil, nil, errors.New("jaguar: empty key")
	}

	ck := compositeKey(b.name, key)

	// Check memtable first.
	if me, ok := b.st.mem.get(ck); ok {
		if me.tombstone {
			return nil, nil, storage.ErrNotExist
		}

		obj := &storage.Object{
			Bucket:      b.name,
			Key:         key,
			Size:        int64(len(me.value)),
			ContentType: me.contentType,
			Created:     time.Unix(0, me.ts),
			Updated:     time.Unix(0, me.ts),
		}

		data := me.value
		data = applyRange(data, offset, length)

		return io.NopCloser(bytes.NewReader(data)), obj, nil
	}

	// Check tree.
	te, ok := b.st.treeGet(ck)
	if !ok {
		return nil, nil, storage.ErrNotExist
	}

	obj := &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        int64(len(te.value)),
		ContentType: te.contentType,
		Created:     time.Unix(0, te.created),
		Updated:     time.Unix(0, te.updated),
	}

	data := te.value
	data = applyRange(data, offset, length)

	return io.NopCloser(bytes.NewReader(data)), obj, nil
}

func applyRange(data []byte, offset, length int64) []byte {
	if offset < 0 {
		offset = 0
	}
	if offset > int64(len(data)) {
		offset = int64(len(data))
	}
	end := int64(len(data))
	if length > 0 && offset+length < end {
		end = offset + length
	}
	return data[offset:end]
}

func (b *bucket) Stat(ctx context.Context, key string, opts storage.Options) (*storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	key = strings.TrimSpace(key)
	if key == "" {
		return nil, errors.New("jaguar: empty key")
	}

	// Check for directory stat.
	if strings.HasSuffix(key, "/") {
		prefix := compositeKey(b.name, key)
		memEntries := b.st.mem.listPrefix(prefix)
		treeEntries := b.st.treeListPrefix(prefix)
		if len(memEntries) == 0 && len(treeEntries) == 0 {
			return nil, storage.ErrNotExist
		}
		return &storage.Object{
			Bucket: b.name,
			Key:    strings.TrimSuffix(key, "/"),
			IsDir:  true,
		}, nil
	}

	ck := compositeKey(b.name, key)

	// Check memtable.
	if me, ok := b.st.mem.get(ck); ok {
		if me.tombstone {
			return nil, storage.ErrNotExist
		}
		return &storage.Object{
			Bucket:      b.name,
			Key:         key,
			Size:        int64(len(me.value)),
			ContentType: me.contentType,
			Created:     time.Unix(0, me.ts),
			Updated:     time.Unix(0, me.ts),
		}, nil
	}

	// Check tree.
	te, ok := b.st.treeGet(ck)
	if !ok {
		return nil, storage.ErrNotExist
	}

	return &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        int64(len(te.value)),
		ContentType: te.contentType,
		Created:     time.Unix(0, te.created),
		Updated:     time.Unix(0, te.updated),
	}, nil
}

func (b *bucket) Delete(ctx context.Context, key string, opts storage.Options) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	key = strings.TrimSpace(key)
	if key == "" {
		return errors.New("jaguar: empty key")
	}

	recursive := false
	if opts != nil {
		if v, ok := opts["recursive"].(bool); ok {
			recursive = v
		}
	}

	if recursive {
		// Delete all objects with this prefix.
		prefix := compositeKey(b.name, key)
		if !strings.HasSuffix(prefix, "/") {
			prefix += "/"
		}

		memEntries := b.st.mem.listPrefix(prefix)
		treeEntries := b.st.treeListPrefix(prefix)

		now := fastNow().UnixNano()
		for _, me := range memEntries {
			b.st.appendWAL(walDelete, me.key, "", nil, now)
			b.st.mem.put(me.key, nil, "", now, true)
		}
		for _, te := range treeEntries {
			b.st.appendWAL(walDelete, te.key, "", nil, now)
			b.st.mem.put(te.key, nil, "", now, true)
		}
		b.st.maybeFlush()
		return nil
	}

	ck := compositeKey(b.name, key)

	// Verify the object exists.
	exists := false
	if me, ok := b.st.mem.get(ck); ok && !me.tombstone {
		exists = true
	}
	if !exists {
		if _, ok := b.st.treeGet(ck); ok {
			exists = true
		}
	}
	if !exists {
		return storage.ErrNotExist
	}

	now := fastNow().UnixNano()

	if err := b.st.appendWAL(walDelete, ck, "", nil, now); err != nil {
		return err
	}
	b.st.mem.put(ck, nil, "", now, true)

	return nil
}

func (b *bucket) Copy(ctx context.Context, dstKey string, srcBucket, srcKey string, opts storage.Options) (*storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	dstKey = strings.TrimSpace(dstKey)
	srcKey = strings.TrimSpace(srcKey)
	if dstKey == "" || srcKey == "" {
		return nil, errors.New("jaguar: empty key")
	}
	if srcBucket == "" {
		srcBucket = b.name
	}
	srcBucket = safeBucketName(srcBucket)

	srcCK := compositeKey(srcBucket, srcKey)

	// Find source data.
	var data []byte
	var contentType string
	var srcTS int64

	if me, ok := b.st.mem.get(srcCK); ok && !me.tombstone {
		data = make([]byte, len(me.value))
		copy(data, me.value)
		contentType = me.contentType
		srcTS = me.ts
	} else if te, ok := b.st.treeGet(srcCK); ok {
		data = make([]byte, len(te.value))
		copy(data, te.value)
		contentType = te.contentType
		srcTS = te.created
	} else {
		return nil, storage.ErrNotExist
	}

	now := fastNow()
	ts := now.UnixNano()
	dstCK := compositeKey(b.name, dstKey)

	if err := b.st.appendWAL(walPut, dstCK, contentType, data, ts); err != nil {
		return nil, err
	}
	b.st.mem.put(dstCK, data, contentType, ts, false)
	b.st.maybeFlush()

	created := time.Unix(0, srcTS)
	return &storage.Object{
		Bucket:      b.name,
		Key:         dstKey,
		Size:        int64(len(data)),
		ContentType: contentType,
		Created:     created,
		Updated:     now,
	}, nil
}

func (b *bucket) Move(ctx context.Context, dstKey string, srcBucket, srcKey string, opts storage.Options) (*storage.Object, error) {
	obj, err := b.Copy(ctx, dstKey, srcBucket, srcKey, opts)
	if err != nil {
		return nil, err
	}

	if srcBucket == "" {
		srcBucket = b.name
	}
	sb := b.st.Bucket(srcBucket)
	if err := sb.Delete(ctx, srcKey, nil); err != nil && !errors.Is(err, storage.ErrNotExist) {
		return nil, err
	}
	return obj, nil
}

func (b *bucket) List(ctx context.Context, prefix string, limit, offset int, opts storage.Options) (storage.ObjectIter, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	prefix = strings.TrimSpace(prefix)

	recursive := true
	if opts != nil {
		if v, ok := opts["recursive"].(bool); ok {
			recursive = v
		}
	}

	ckPrefix := compositeKey(b.name, prefix)

	// Gather from memtable.
	memEntries := b.st.mem.listPrefix(ckPrefix)

	// Gather from tree.
	treeEntries := b.st.treeListPrefix(ckPrefix)

	// Merge and deduplicate (memtable wins).
	seen := make(map[string]bool, len(memEntries))
	var objs []*storage.Object

	for _, me := range memEntries {
		_, objKey := splitCompositeKey(me.key)
		seen[me.key] = true
		objs = append(objs, &storage.Object{
			Bucket:      b.name,
			Key:         objKey,
			Size:        int64(len(me.value)),
			ContentType: me.contentType,
			Created:     time.Unix(0, me.ts),
			Updated:     time.Unix(0, me.ts),
		})
	}

	for _, te := range treeEntries {
		if seen[te.key] {
			continue
		}
		_, objKey := splitCompositeKey(te.key)
		objs = append(objs, &storage.Object{
			Bucket:      b.name,
			Key:         objKey,
			Size:        int64(len(te.value)),
			ContentType: te.contentType,
			Created:     time.Unix(0, te.created),
			Updated:     time.Unix(0, te.updated),
		})
	}

	if !recursive {
		filtered := objs[:0]
		for _, o := range objs {
			rest := strings.TrimPrefix(o.Key, prefix)
			rest = strings.TrimPrefix(rest, "/")
			if !strings.Contains(rest, "/") {
				filtered = append(filtered, o)
			}
		}
		objs = filtered
	}

	sort.Slice(objs, func(i, j int) bool { return objs[i].Key < objs[j].Key })

	if offset < 0 {
		offset = 0
	}
	if offset > len(objs) {
		offset = len(objs)
	}
	objs = objs[offset:]
	if limit > 0 && limit < len(objs) {
		objs = objs[:limit]
	}

	return &objectIter{objects: objs}, nil
}

func (b *bucket) SignedURL(_ context.Context, _ string, _ string, _ time.Duration, _ storage.Options) (string, error) {
	return "", storage.ErrUnsupported
}

// ---------------------------------------------------------------------------
// Directory support
// ---------------------------------------------------------------------------

func (b *bucket) Directory(p string) storage.Directory {
	return &dir{b: b, path: strings.Trim(p, "/")}
}

type dir struct {
	b    *bucket
	path string
}

var _ storage.Directory = (*dir)(nil)

func (d *dir) Bucket() storage.Bucket { return d.b }
func (d *dir) Path() string           { return d.path }

func (d *dir) Info(ctx context.Context) (*storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	prefix := d.path
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	ckPrefix := compositeKey(d.b.name, prefix)
	memEntries := d.b.st.mem.listPrefix(ckPrefix)
	treeEntries := d.b.st.treeListPrefix(ckPrefix)

	if len(memEntries) == 0 && len(treeEntries) == 0 {
		return nil, storage.ErrNotExist
	}

	return &storage.Object{
		Bucket: d.b.name,
		Key:    d.path,
		IsDir:  true,
	}, nil
}

func (d *dir) List(ctx context.Context, limit, offset int, opts storage.Options) (storage.ObjectIter, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	prefix := d.path
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	ckPrefix := compositeKey(d.b.name, prefix)
	memEntries := d.b.st.mem.listPrefix(ckPrefix)
	treeEntries := d.b.st.treeListPrefix(ckPrefix)

	seen := make(map[string]bool, len(memEntries))
	var objs []*storage.Object

	for _, me := range memEntries {
		_, objKey := splitCompositeKey(me.key)
		seen[me.key] = true
		rest := strings.TrimPrefix(objKey, prefix)
		if strings.Contains(rest, "/") {
			continue
		}
		objs = append(objs, &storage.Object{
			Bucket:      d.b.name,
			Key:         objKey,
			Size:        int64(len(me.value)),
			ContentType: me.contentType,
			Created:     time.Unix(0, me.ts),
			Updated:     time.Unix(0, me.ts),
		})
	}

	for _, te := range treeEntries {
		if seen[te.key] {
			continue
		}
		_, objKey := splitCompositeKey(te.key)
		rest := strings.TrimPrefix(objKey, prefix)
		if strings.Contains(rest, "/") {
			continue
		}
		objs = append(objs, &storage.Object{
			Bucket:      d.b.name,
			Key:         objKey,
			Size:        int64(len(te.value)),
			ContentType: te.contentType,
			Created:     time.Unix(0, te.created),
			Updated:     time.Unix(0, te.updated),
		})
	}

	sort.Slice(objs, func(i, j int) bool { return objs[i].Key < objs[j].Key })

	if offset < 0 {
		offset = 0
	}
	if offset > len(objs) {
		offset = len(objs)
	}
	objs = objs[offset:]
	if limit > 0 && limit < len(objs) {
		objs = objs[:limit]
	}

	return &objectIter{objects: objs}, nil
}

func (d *dir) Delete(ctx context.Context, opts storage.Options) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	recursive := false
	if opts != nil {
		if v, ok := opts["recursive"].(bool); ok {
			recursive = v
		}
	}

	prefix := d.path
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	ckPrefix := compositeKey(d.b.name, prefix)
	memEntries := d.b.st.mem.listPrefix(ckPrefix)
	treeEntries := d.b.st.treeListPrefix(ckPrefix)

	if len(memEntries) == 0 && len(treeEntries) == 0 {
		return storage.ErrNotExist
	}

	now := fastNow().UnixNano()

	deleteKey := func(ck string) {
		_, objKey := splitCompositeKey(ck)
		if !recursive {
			rest := strings.TrimPrefix(objKey, prefix)
			if strings.Contains(rest, "/") {
				return
			}
		}
		d.b.st.appendWAL(walDelete, ck, "", nil, now)
		d.b.st.mem.put(ck, nil, "", now, true)
	}

	for _, me := range memEntries {
		deleteKey(me.key)
	}
	for _, te := range treeEntries {
		deleteKey(te.key)
	}

	return nil
}

func (d *dir) Move(ctx context.Context, dstPath string, opts storage.Options) (storage.Directory, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	srcPrefix := strings.Trim(d.path, "/")
	dstPrefix := strings.Trim(dstPath, "/")

	if srcPrefix != "" && !strings.HasSuffix(srcPrefix, "/") {
		srcPrefix += "/"
	}
	if dstPrefix != "" && !strings.HasSuffix(dstPrefix, "/") {
		dstPrefix += "/"
	}

	srcCKPrefix := compositeKey(d.b.name, srcPrefix)

	memEntries := d.b.st.mem.listPrefix(srcCKPrefix)
	treeEntries := d.b.st.treeListPrefix(srcCKPrefix)

	if len(memEntries) == 0 && len(treeEntries) == 0 {
		return nil, storage.ErrNotExist
	}

	now := fastNow().UnixNano()

	moveEntry := func(ck string, value []byte, contentType string) {
		_, objKey := splitCompositeKey(ck)
		rel := strings.TrimPrefix(objKey, srcPrefix)
		newKey := dstPrefix + rel
		newCK := compositeKey(d.b.name, newKey)

		// Write new entry.
		d.b.st.appendWAL(walPut, newCK, contentType, value, now)
		d.b.st.mem.put(newCK, value, contentType, now, false)

		// Delete old entry.
		d.b.st.appendWAL(walDelete, ck, "", nil, now)
		d.b.st.mem.put(ck, nil, "", now, true)
	}

	seen := make(map[string]bool)
	for _, me := range memEntries {
		seen[me.key] = true
		moveEntry(me.key, me.value, me.contentType)
	}
	for _, te := range treeEntries {
		if seen[te.key] {
			continue
		}
		moveEntry(te.key, te.value, te.contentType)
	}

	d.b.st.maybeFlush()

	return &dir{b: d.b, path: strings.Trim(dstPath, "/")}, nil
}

// ---------------------------------------------------------------------------
// Multipart upload support
// ---------------------------------------------------------------------------

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

func (b *bucket) InitMultipart(ctx context.Context, key string, contentType string, opts storage.Options) (*storage.MultipartUpload, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	key = strings.TrimSpace(key)
	if key == "" {
		return nil, errors.New("jaguar: empty key")
	}

	uploadID := newUploadID()
	mu := &storage.MultipartUpload{
		Bucket:   b.name,
		Key:      key,
		UploadID: uploadID,
	}

	b.st.mpMu.Lock()
	b.st.mpUploads[uploadID] = &multipartUpload{
		mu:          mu,
		contentType: contentType,
		createdAt:   fastNow(),
		parts:       make(map[int]*partData),
	}
	b.st.mpMu.Unlock()

	return mu, nil
}

func (b *bucket) UploadPart(ctx context.Context, mu *storage.MultipartUpload, number int, src io.Reader, size int64, opts storage.Options) (*storage.PartInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if number < 1 || number > maxPartNumber {
		return nil, fmt.Errorf("jaguar: part number %d out of range (1-%d)", number, maxPartNumber)
	}

	var data []byte
	if size >= 0 {
		data = make([]byte, size)
		n, err := io.ReadFull(src, data)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return nil, fmt.Errorf("jaguar: read part: %w", err)
		}
		data = data[:n]
	} else {
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, src); err != nil {
			return nil, fmt.Errorf("jaguar: read part: %w", err)
		}
		data = buf.Bytes()
	}

	now := fastNow()
	sum := md5.Sum(data)
	etag := hex.EncodeToString(sum[:])

	pd := &partData{
		number:       number,
		data:         data,
		etag:         etag,
		lastModified: now,
	}

	b.st.mpMu.Lock()
	upload, ok := b.st.mpUploads[mu.UploadID]
	if !ok {
		b.st.mpMu.Unlock()
		return nil, storage.ErrNotExist
	}
	upload.parts[number] = pd
	b.st.mpMu.Unlock()

	return &storage.PartInfo{
		Number:       number,
		Size:         int64(len(data)),
		ETag:         etag,
		LastModified: &now,
	}, nil
}

func (b *bucket) CopyPart(_ context.Context, _ *storage.MultipartUpload, _ int, _ storage.Options) (*storage.PartInfo, error) {
	return nil, storage.ErrUnsupported
}

func (b *bucket) ListParts(ctx context.Context, mu *storage.MultipartUpload, limit, offset int, opts storage.Options) ([]*storage.PartInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	b.st.mpMu.RLock()
	defer b.st.mpMu.RUnlock()

	upload, ok := b.st.mpUploads[mu.UploadID]
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
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if len(parts) == 0 {
		return nil, errors.New("jaguar: no parts to complete")
	}

	b.st.mpMu.Lock()
	upload, ok := b.st.mpUploads[mu.UploadID]
	if !ok {
		b.st.mpMu.Unlock()
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
			b.st.mpMu.Unlock()
			return nil, fmt.Errorf("jaguar: part %d not found", part.Number)
		}
		totalSize += len(pd.data)
	}

	data := make([]byte, 0, totalSize)
	for _, part := range sortedParts {
		pd := upload.parts[part.Number]
		data = append(data, pd.data...)
	}

	contentType := upload.contentType
	key := upload.mu.Key
	delete(b.st.mpUploads, mu.UploadID)
	b.st.mpMu.Unlock()

	// Write the assembled object.
	now := fastNow()
	ts := now.UnixNano()
	ck := compositeKey(b.name, key)

	if err := b.st.appendWAL(walPut, ck, contentType, data, ts); err != nil {
		return nil, err
	}
	b.st.mem.put(ck, data, contentType, ts, false)
	b.st.maybeFlush()

	return &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        int64(totalSize),
		ContentType: contentType,
		Created:     now,
		Updated:     now,
	}, nil
}

func (b *bucket) AbortMultipart(ctx context.Context, mu *storage.MultipartUpload, opts storage.Options) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	b.st.mpMu.Lock()
	defer b.st.mpMu.Unlock()

	if _, ok := b.st.mpUploads[mu.UploadID]; !ok {
		return storage.ErrNotExist
	}

	delete(b.st.mpUploads, mu.UploadID)
	return nil
}

// ---------------------------------------------------------------------------
// Iterators
// ---------------------------------------------------------------------------

type bucketIter struct {
	buckets []*storage.BucketInfo
	index   int
}

func (it *bucketIter) Next() (*storage.BucketInfo, error) {
	if it.index >= len(it.buckets) {
		return nil, nil
	}
	b := it.buckets[it.index]
	it.index++
	return b, nil
}

func (it *bucketIter) Close() error {
	it.buckets = nil
	return nil
}

type objectIter struct {
	objects []*storage.Object
	index   int
}

func (it *objectIter) Next() (*storage.Object, error) {
	if it.index >= len(it.objects) {
		return nil, nil
	}
	o := it.objects[it.index]
	it.index++
	return o, nil
}

func (it *objectIter) Close() error {
	it.objects = nil
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

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

func newUploadID() string {
	now := time.Now().UTC().UnixNano()
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%x-0", now)
	}
	r := binary.LittleEndian.Uint64(b[:])
	return fmt.Sprintf("%x-%x", now, r)
}
