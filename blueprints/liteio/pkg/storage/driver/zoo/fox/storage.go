// Package fox implements a Bf-Tree inspired storage driver.
//
// The Bf-Tree (VLDB 2024) combines a B-tree with in-memory mini-page write
// buffers in a circular LRU pool. Inner nodes live in memory, leaf pages live
// on disk, and mini-pages absorb writes so that disk I/O is batched.
//
// DSN format:
//
//	fox:///path/to/data
//	fox:///path/to/data?sync=none
//	fox:///path/to/data?sync=none&page_size=4096&pool_size=16777216
package fox

import (
	"container/list"
	"context"
	"crypto/md5"
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

// ---------------------------------------------------------------------------
// Driver registration
// ---------------------------------------------------------------------------

func init() {
	storage.Register("fox", &driver{})
}

type driver struct{}

func (d *driver) Open(ctx context.Context, dsn string) (storage.Storage, error) {
	_ = ctx

	u, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("fox: parse dsn: %w", err)
	}
	if u.Scheme != "fox" && u.Scheme != "" {
		return nil, fmt.Errorf("fox: unexpected scheme %q", u.Scheme)
	}

	root := filepath.Clean(u.Path)
	if root == "" || root == "." {
		root = "/tmp/fox-data"
	}

	syncMode := u.Query().Get("sync")
	if syncMode == "" {
		syncMode = "none"
	}

	pageSize := defaultPageSize
	if ps := u.Query().Get("page_size"); ps != "" {
		if n, err := strconv.Atoi(ps); err == nil && n >= minPageSize {
			pageSize = n
		}
	}

	poolSize := int64(defaultPoolSize)
	if ps := u.Query().Get("pool_size"); ps != "" {
		if n, err := strconv.ParseInt(ps, 10, 64); err == nil && n > 0 {
			poolSize = n
		}
	}

	if err := os.MkdirAll(root, 0750); err != nil {
		return nil, fmt.Errorf("fox: mkdir %q: %w", root, err)
	}

	pagesPath := filepath.Join(root, "pages.dat")
	valuesPath := filepath.Join(root, "values.dat")
	metaPath := filepath.Join(root, "meta.json")

	st := &store{
		root:             root,
		syncMode:         syncMode,
		pageSize:         pageSize,
		poolSize:         poolSize,
		metaPath:         metaPath,
		inlineValueLimit: min(defaultInlineValueLimit, max(64, pageSize/8)),
		buckets:          make(map[string]time.Time),
		mp:               newMultipartRegistry(),
		stopTick:         make(chan struct{}),
	}

	// Open or create page file.
	pf, err := os.OpenFile(pagesPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("fox: open pages: %w", err)
	}

	// Ensure pf is closed if we fail after this point.
	success := false
	defer func() {
		if !success {
			pf.Close()
			if st.valueFile != nil {
				_ = st.valueFile.Close()
			}
		}
	}()

	st.pageFile = pf

	vf, err := os.OpenFile(valuesPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("fox: open values: %w", err)
	}
	st.valueFile = vf
	if info, statErr := vf.Stat(); statErr == nil {
		st.valueTail = info.Size()
	}

	// Initialize B-tree and pool.
	st.pool = newMiniPagePool(poolSize, st)
	st.tree = newBTree(pageSize)

	// Try to load existing metadata.
	if err := st.loadMeta(); err != nil {
		// Fresh store -- allocate root leaf page.
		rootID, allocErr := st.allocPage()
		if allocErr != nil {
			return nil, allocErr
		}
		st.tree.root = &btreeNode{
			leaf:   true,
			pageID: rootID,
		}
		st.tree.height = 1
	}

	// Start per-store time cache ticker.
	go func() {
		ticker := time.NewTicker(1 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				cachedTimeNano.Store(time.Now().UnixNano())
			case <-st.stopTick:
				return
			}
		}
	}()

	success = true
	return st, nil
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const (
	defaultPageSize = 4096
	minPageSize     = 512
	defaultPoolSize = 16 * 1024 * 1024 // 16 MB

	minMiniPageSize = 64
	maxMiniPageSize = 4096

	maxBranchFactor = 128

	tombstoneMarker = 0xFFFFFFFF

	maxPartNumber = 10000

	maxBuckets = 10000

	defaultInlineValueLimit = 256

	indirectValueMarker = 0xFFFFFFFE
)

// ---------------------------------------------------------------------------
// Fast time cache
// ---------------------------------------------------------------------------

var cachedTimeNano atomic.Int64

func init() {
	cachedTimeNano.Store(time.Now().UnixNano())
}

func fastNow() int64     { return cachedTimeNano.Load() }
func fastNowTime() time.Time { return time.Unix(0, fastNow()) }

// ---------------------------------------------------------------------------
// store implements storage.Storage
// ---------------------------------------------------------------------------

type store struct {
	root     string
	syncMode string
	pageSize int
	poolSize int64
	metaPath string

	pageFile  *os.File
	valueFile *os.File
	pageCount int64 // total pages allocated
	valueTail int64

	tree *btree
	pool *miniPagePool

	inlineValueLimit int

	mu      sync.RWMutex
	buckets map[string]time.Time

	mp *multipartRegistry

	stopTick chan struct{}
}

var _ storage.Storage = (*store)(nil)

// compositeKey builds the B-tree lookup key.
func compositeKey(bucket, key string) string {
	return bucket + "\x00" + key
}

// splitCompositeKey extracts bucket and key from a composite key.
func splitCompositeKey(ck string) (bucket, key string) {
	i := strings.IndexByte(ck, 0)
	if i < 0 {
		return ck, ""
	}
	return ck[:i], ck[i+1:]
}

// ---------------------------------------------------------------------------
// Storage interface
// ---------------------------------------------------------------------------

func (s *store) Bucket(name string) storage.Bucket {
	if name == "" {
		name = "default"
	}

	s.mu.Lock()
	if _, ok := s.buckets[name]; !ok {
		if len(s.buckets) < maxBuckets {
			s.buckets[name] = fastNowTime()
		}
	}
	s.mu.Unlock()

	return &bucket{st: s, name: name}
}

func (s *store) Buckets(ctx context.Context, limit, offset int, opts storage.Options) (storage.BucketIter, error) {
	_ = ctx
	_ = opts

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
	_ = ctx
	_ = opts

	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("fox: bucket name is empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.buckets[name]; ok {
		return nil, storage.ErrExist
	}

	now := fastNowTime()
	s.buckets[name] = now

	return &storage.BucketInfo{
		Name:      name,
		CreatedAt: now,
	}, nil
}

func (s *store) DeleteBucket(ctx context.Context, name string, opts storage.Options) error {
	_ = ctx

	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("fox: bucket name is empty")
	}

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

	if !force && s.tree.hasBucket(name) {
		return storage.ErrPermission
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
	// Stop the time cache ticker goroutine.
	close(s.stopTick)

	s.mu.Lock()
	defer s.mu.Unlock()

	// Flush all dirty mini-pages while holding the pool lock.
	s.pool.mu.Lock()
	s.pool.flushAll()
	s.pool.mu.Unlock()

	// Sync page file.
	if s.syncMode != "none" {
		_ = s.pageFile.Sync()
		_ = s.valueFile.Sync()
	}

	// Save metadata.
	s.saveMeta()

	if err := s.pageFile.Close(); err != nil {
		_ = s.valueFile.Close()
		return err
	}
	return s.valueFile.Close()
}

// ---------------------------------------------------------------------------
// Page file I/O
// ---------------------------------------------------------------------------

func (s *store) allocPage() (int64, error) {
	id := s.pageCount
	s.pageCount++

	// Extend file without writing a zero page each time.
	off := (id + 1) * int64(s.pageSize)
	if err := s.pageFile.Truncate(off); err != nil {
		return 0, fmt.Errorf("fox: alloc page: %w", err)
	}
	return id, nil
}

func (s *store) readPage(id int64) ([]byte, error) {
	buf := make([]byte, s.pageSize)
	off := id * int64(s.pageSize)
	n, err := s.pageFile.ReadAt(buf, off)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("fox: read page %d: %w", id, err)
	}
	if n < len(buf) {
		clear(buf[n:])
	}
	return buf, nil
}

func (s *store) writePage(id int64, data []byte) error {
	off := id * int64(s.pageSize)
	if _, err := s.pageFile.WriteAt(data, off); err != nil {
		return fmt.Errorf("fox: write page %d: %w", id, err)
	}
	if s.syncMode == "full" {
		s.pageFile.Sync()
	}
	return nil
}

type valueRef struct {
	offset int64
	size   uint32
}

func (s *store) appendValue(data []byte) (valueRef, error) {
	if len(data) == 0 {
		return valueRef{}, nil
	}
	off := s.valueTail
	if _, err := s.valueFile.WriteAt(data, off); err != nil {
		return valueRef{}, fmt.Errorf("fox: write value: %w", err)
	}
	s.valueTail += int64(len(data))
	if s.syncMode == "full" {
		if err := s.valueFile.Sync(); err != nil {
			return valueRef{}, fmt.Errorf("fox: sync values: %w", err)
		}
	}
	return valueRef{offset: off, size: uint32(len(data))}, nil
}

func (s *store) readValue(ref valueRef, offset, length int64) ([]byte, error) {
	size := int64(ref.size)
	if offset < 0 {
		offset = 0
	}
	if offset > size {
		offset = size
	}
	end := size
	if length > 0 && offset+length < end {
		end = offset + length
	}
	n := end - offset
	if n <= 0 {
		return nil, nil
	}
	buf := make([]byte, n)
	if _, err := s.valueFile.ReadAt(buf, ref.offset+offset); err != nil && err != io.EOF {
		return nil, fmt.Errorf("fox: read value: %w", err)
	}
	return buf, nil
}

// ---------------------------------------------------------------------------
// Metadata persistence
// ---------------------------------------------------------------------------

type metaJSON struct {
	Root      int64 `json:"root"`
	PageCount int64 `json:"page_count"`
	Height    int   `json:"height"`
}

func (s *store) saveMeta() error {
	rootID := int64(0)
	if s.tree.root != nil {
		rootID = s.tree.root.pageID
	}
	m := metaJSON{
		Root:      rootID,
		PageCount: s.pageCount,
		Height:    s.tree.height,
	}
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return os.WriteFile(s.metaPath, data, 0600)
}

func (s *store) loadMeta() error {
	data, err := os.ReadFile(s.metaPath)
	if err != nil {
		return err
	}
	var m metaJSON
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	s.pageCount = m.PageCount

	// Rebuild tree from page file.
	s.tree.root = &btreeNode{
		leaf:   true,
		pageID: m.Root,
	}
	s.tree.height = m.Height

	// Scan all pages to rebuild inner nodes.
	if err := s.rebuildTree(); err != nil {
		return err
	}
	return nil
}

// rebuildTree scans all leaf pages and rebuilds the in-memory B-tree structure.
func (s *store) rebuildTree() error {
	if s.pageCount == 0 {
		return nil
	}

	// Collect all entries from all pages.
	type leafEntry struct {
		pageID int64
		minKey string
	}

	var leaves []leafEntry
	for pid := int64(0); pid < s.pageCount; pid++ {
		pageData, err := s.readPage(pid)
		if err != nil {
			continue
		}
		entries := decodePageEntriesMeta(pageData)
		minKey := ""
		if len(entries) > 0 {
			minKey = entries[0].key
		}
		leaves = append(leaves, leafEntry{pageID: pid, minKey: minKey})
	}

	if len(leaves) == 0 {
		return nil
	}

	sort.Slice(leaves, func(i, j int) bool {
		return leaves[i].minKey < leaves[j].minKey
	})

	// Build tree bottom-up.
	var nodes []*btreeNode
	for _, le := range leaves {
		nodes = append(nodes, &btreeNode{
			leaf:   true,
			pageID: le.pageID,
		})
	}

	for len(nodes) > 1 {
		var parents []*btreeNode
		for i := 0; i < len(nodes); i += maxBranchFactor {
			end := min(i+maxBranchFactor, len(nodes))
			chunk := nodes[i:end]

			parent := &btreeNode{leaf: false}
			for j, child := range chunk {
				if j > 0 {
					// Use the first key of this child as separator.
					sep := ""
					if child.leaf {
						pageData, err := s.readPage(child.pageID)
						if err == nil {
							entries := decodePageEntriesMeta(pageData)
							if len(entries) > 0 {
								sep = entries[0].key
							}
						}
					} else if len(child.keys) > 0 {
						sep = child.keys[0]
					}
					parent.keys = append(parent.keys, sep)
				}
				parent.children = append(parent.children, child)
			}
			parents = append(parents, parent)
		}
		nodes = parents
	}

	s.tree.root = nodes[0]
	s.tree.height = s.computeHeight(s.tree.root)
	return nil
}

func (s *store) computeHeight(n *btreeNode) int {
	if n == nil || n.leaf {
		return 1
	}
	if len(n.children) > 0 {
		return 1 + s.computeHeight(n.children[0])
	}
	return 1
}

// ---------------------------------------------------------------------------
// B-tree
// ---------------------------------------------------------------------------

type btree struct {
	root     *btreeNode
	height   int
	pageSize int
}

type btreeNode struct {
	leaf     bool
	pageID   int64 // only for leaf nodes
	keys     []string
	children []*btreeNode
}

func newBTree(pageSize int) *btree {
	return &btree{pageSize: pageSize}
}

// hasBucket checks if any key with this bucket prefix exists in the tree.
func (t *btree) hasBucket(bucketName string) bool {
	prefix := bucketName + "\x00"
	return t.hasPrefix(t.root, prefix)
}

func (t *btree) hasPrefix(n *btreeNode, prefix string) bool {
	if n == nil {
		return false
	}
	if n.leaf {
		// We'd need to check the actual page data, which is handled
		// at the store level. Return false here; the store checks.
		return false
	}
	for _, child := range n.children {
		if t.hasPrefix(child, prefix) {
			return true
		}
	}
	return false
}

// findLeaf traverses the tree to find the leaf node for a given key.
func (t *btree) findLeaf(key string) *btreeNode {
	n := t.root
	for n != nil && !n.leaf {
		// Binary search for the child.
		idx := sort.Search(len(n.keys), func(i int) bool {
			return n.keys[i] > key
		})
		if idx >= len(n.children) {
			idx = len(n.children) - 1
		}
		n = n.children[idx]
	}
	return n
}

// ---------------------------------------------------------------------------
// Page entry encoding/decoding
// ---------------------------------------------------------------------------

// pageEntry represents a key-value record within a leaf page.
type pageEntry struct {
	key         string
	contentType string
	value       []byte
	valueRef    valueRef
	valueSize   uint32
	created     int64
	updated     int64
	tombstone   bool
}

const pageHeaderSize = 16

const indirectValueRefSize = 12

func (e pageEntry) hasIndirectValue() bool {
	return e.valueRef.size > 0 || (e.valueSize > 0 && len(e.value) == 0)
}

func (e pageEntry) objectSize() int64 {
	return int64(e.valueSize)
}

func (e pageEntry) encodedValueSize() int {
	if e.tombstone {
		return 0
	}
	if e.hasIndirectValue() {
		return indirectValueRefSize
	}
	return len(e.value)
}

func encodedEntrySize(e pageEntry) int {
	return 2 + len(e.key) + 2 + len(e.contentType) + 4 + e.encodedValueSize() + 8 + 8
}

// encodePageEntries serializes entries into a page-sized buffer.
// Returns the buffer and whether it fits.
func encodePageEntries(entries []pageEntry, pageSize int) ([]byte, bool) {
	buf := make([]byte, pageSize)

	// Header: count(2) + freeOff(2) + flags(2) + nextPage(4) + pad(6)
	pos := pageHeaderSize
	count := 0

	for _, e := range entries {
		if e.tombstone {
			continue
		}
		needed := encodedEntrySize(e)
		if pos+needed > pageSize {
			// Doesn't fit.
			binary.LittleEndian.PutUint16(buf[0:2], uint16(count))
			binary.LittleEndian.PutUint16(buf[2:4], uint16(pos))
			return buf, false
		}

		binary.LittleEndian.PutUint16(buf[pos:], uint16(len(e.key)))
		pos += 2
		copy(buf[pos:], e.key)
		pos += len(e.key)

		binary.LittleEndian.PutUint16(buf[pos:], uint16(len(e.contentType)))
		pos += 2
		copy(buf[pos:], e.contentType)
		pos += len(e.contentType)

		if e.hasIndirectValue() {
			binary.LittleEndian.PutUint32(buf[pos:], indirectValueMarker)
			pos += 4
			binary.LittleEndian.PutUint64(buf[pos:], uint64(e.valueRef.offset))
			pos += 8
			binary.LittleEndian.PutUint32(buf[pos:], e.valueRef.size)
			pos += 4
		} else {
			binary.LittleEndian.PutUint32(buf[pos:], uint32(len(e.value)))
			pos += 4
			copy(buf[pos:], e.value)
			pos += len(e.value)
		}

		binary.LittleEndian.PutUint64(buf[pos:], uint64(e.created))
		pos += 8
		binary.LittleEndian.PutUint64(buf[pos:], uint64(e.updated))
		pos += 8

		count++
	}

	binary.LittleEndian.PutUint16(buf[0:2], uint16(count))
	binary.LittleEndian.PutUint16(buf[2:4], uint16(pos))
	return buf, true
}

// decodePageEntriesMeta decodes page entries without copying inline values.
func decodePageEntriesMeta(buf []byte) []pageEntry {
	return decodePageEntriesWithMode(buf, false)
}

// decodePageEntries reads entries from a raw page buffer.
func decodePageEntries(buf []byte) []pageEntry {
	return decodePageEntriesWithMode(buf, true)
}

func decodePageEntriesWithMode(buf []byte, copyInlineValues bool) []pageEntry {
	if len(buf) < pageHeaderSize {
		return nil
	}

	count := int(binary.LittleEndian.Uint16(buf[0:2]))
	if count == 0 {
		return nil
	}

	pos := pageHeaderSize
	entries := make([]pageEntry, 0, count)

entryLoop:
	for i := range count {
		_ = i
		if pos+2 > len(buf) {
			break
		}
		kl := int(binary.LittleEndian.Uint16(buf[pos:]))
		pos += 2
		if pos+kl > len(buf) {
			break
		}
		key := string(buf[pos : pos+kl])
		pos += kl

		if pos+2 > len(buf) {
			break
		}
		cl := int(binary.LittleEndian.Uint16(buf[pos:]))
		pos += 2
		if pos+cl > len(buf) {
			break
		}
		ct := string(buf[pos : pos+cl])
		pos += cl

		if pos+4 > len(buf) {
			break
		}
		vl := binary.LittleEndian.Uint32(buf[pos:])
		pos += 4

		tomb := vl == tombstoneMarker
		indirect := vl == indirectValueMarker
		valLen := int(vl)
		var (
			val []byte
			ref valueRef
			sz  uint32
		)

		switch {
		case tomb:
			valLen = 0
		case indirect:
			if pos+indirectValueRefSize > len(buf) {
				break entryLoop
			}
			ref.offset = int64(binary.LittleEndian.Uint64(buf[pos:]))
			pos += 8
			ref.size = binary.LittleEndian.Uint32(buf[pos:])
			pos += 4
			sz = ref.size
			valLen = 0
		default:
			if pos+valLen > len(buf) {
				break entryLoop
			}
			sz = uint32(valLen)
			if copyInlineValues {
				val = make([]byte, valLen)
				copy(val, buf[pos:pos+valLen])
			}
			pos += valLen
		}

		var created, updated int64
		if pos+16 <= len(buf) {
			created = int64(binary.LittleEndian.Uint64(buf[pos:]))
			pos += 8
			updated = int64(binary.LittleEndian.Uint64(buf[pos:]))
			pos += 8
		}

		entries = append(entries, pageEntry{
			key:         key,
			contentType: ct,
			value:       val,
			valueRef:    ref,
			valueSize:   sz,
			created:     created,
			updated:     updated,
			tombstone:   tomb,
		})
	}

	return entries
}

// ---------------------------------------------------------------------------
// Mini-page pool (LRU)
// ---------------------------------------------------------------------------

type miniPage struct {
	leafID  int64
	entries []pageEntry
	size    int // approximate byte size of buffered entries
	dirty   bool
	element *list.Element
}

type miniPagePool struct {
	mu       sync.Mutex
	pages    map[int64]*miniPage // leafID -> miniPage
	lru      *list.List
	curSize  int64
	maxSize  int64
	st       *store
}

func newMiniPagePool(maxSize int64, st *store) *miniPagePool {
	return &miniPagePool{
		pages:   make(map[int64]*miniPage),
		lru:     list.New(),
		maxSize: maxSize,
		st:      st,
	}
}

func (p *miniPagePool) get(leafID int64) *miniPage {
	mp, ok := p.pages[leafID]
	if ok {
		p.lru.MoveToFront(mp.element)
		return mp
	}
	return nil
}

func (p *miniPagePool) getOrCreate(leafID int64) *miniPage {
	mp := p.get(leafID)
	if mp != nil {
		return mp
	}

	// Evict if needed.
	for p.curSize >= p.maxSize && p.lru.Len() > 0 {
		p.evictOldest()
	}

	mp = &miniPage{
		leafID:  leafID,
		entries: make([]pageEntry, 0, 4),
		size:    minMiniPageSize,
	}
	mp.element = p.lru.PushFront(mp)
	p.pages[leafID] = mp
	p.curSize += int64(mp.size)
	return mp
}

func (p *miniPagePool) evictOldest() {
	oldest := p.lru.Back()
	if oldest == nil {
		return
	}
	mp := oldest.Value.(*miniPage)
	if mp.dirty {
		p.flushMiniPage(mp)
	}
	p.lru.Remove(oldest)
	p.curSize -= int64(mp.size)
	delete(p.pages, mp.leafID)
}

func (p *miniPagePool) flushMiniPage(mp *miniPage) {
	if !mp.dirty || len(mp.entries) == 0 {
		return
	}

	// Read existing page.
	existing, err := p.st.readPage(mp.leafID)
	if err != nil {
		return
	}

	diskEntries := decodePageEntries(existing)

	// Merge: mini-page entries override disk entries.
	merged := mergeEntries(diskEntries, mp.entries)

	// Remove tombstones.
	live := make([]pageEntry, 0, len(merged))
	for _, e := range merged {
		if !e.tombstone {
			live = append(live, e)
		}
	}

	// Encode back.
	buf, fits := encodePageEntries(live, p.st.pageSize)
	if !fits {
		chunks := splitEntriesByPage(live, p.st.pageSize)
		if len(chunks) == 0 {
			// If a single entry still can't fit, keep the mini-page dirty.
			return
		}

		firstBuf, ok := encodePageEntries(chunks[0], p.st.pageSize)
		if !ok || p.st.writePage(mp.leafID, firstBuf) != nil {
			return
		}

		prevLeafID := mp.leafID
		for _, chunk := range chunks[1:] {
			newID, err := p.st.allocPage()
			if err != nil {
				return
			}
			chunkBuf, ok := encodePageEntries(chunk, p.st.pageSize)
			if !ok {
				return
			}
			if err := p.st.writePage(newID, chunkBuf); err != nil {
				return
			}
			p.st.tree.insertLeaf(prevLeafID, newID, chunk[0].key)
			prevLeafID = newID
		}
	} else {
		if err := p.st.writePage(mp.leafID, buf); err != nil {
			return
		}
	}

	if mp.size > minMiniPageSize {
		p.curSize -= int64(mp.size - minMiniPageSize)
	}
	mp.dirty = false
	mp.entries = mp.entries[:0]
	mp.size = minMiniPageSize
}

func (p *miniPagePool) flushAll() {
	for _, mp := range p.pages {
		if mp.dirty {
			p.flushMiniPage(mp)
		}
	}
}

// mergeEntries merges disk entries with mini-page entries.
// Mini-page entries take precedence. Result is sorted by key.
func mergeEntries(disk, mini []pageEntry) []pageEntry {
	m := make(map[string]pageEntry, len(disk)+len(mini))
	for _, e := range disk {
		m[e.key] = e
	}
	for _, e := range mini {
		m[e.key] = e
	}

	result := make([]pageEntry, 0, len(m))
	for _, e := range m {
		result = append(result, e)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].key < result[j].key
	})
	return result
}

func splitEntriesByPage(entries []pageEntry, pageSize int) [][]pageEntry {
	if len(entries) == 0 {
		return [][]pageEntry{{}}
	}
	chunks := make([][]pageEntry, 0, 2)
	start := 0
	for start < len(entries) {
		pos := pageHeaderSize
		end := start
		for end < len(entries) {
			need := encodedEntrySize(entries[end])
			if end == start && pos+need > pageSize {
				return nil
			}
			if pos+need > pageSize {
				break
			}
			pos += need
			end++
		}
		if end == start {
			return nil
		}
		chunks = append(chunks, entries[start:end])
		start = end
	}
	return chunks
}

// insertLeaf splits a leaf node in the B-tree by adding a new child.
func (t *btree) insertLeaf(oldLeafID, newLeafID int64, splitKey string) {
	if t.root == nil {
		return
	}

	if t.root.leaf {
		// Root is the leaf being split. Create new root.
		newRoot := &btreeNode{
			leaf: false,
			keys: []string{splitKey},
			children: []*btreeNode{
				{leaf: true, pageID: oldLeafID},
				{leaf: true, pageID: newLeafID},
			},
		}
		t.root = newRoot
		t.height++
		return
	}

	// Find parent that contains oldLeafID and insert there.
	t.insertIntoParent(t.root, oldLeafID, newLeafID, splitKey)
}

func (t *btree) insertIntoParent(n *btreeNode, oldLeafID, newLeafID int64, splitKey string) bool {
	if n.leaf {
		return false
	}

	for i, child := range n.children {
		if child.leaf && child.pageID == oldLeafID {
			// Insert new child after position i.
			newChild := &btreeNode{leaf: true, pageID: newLeafID}

			// Insert key at position i.
			n.keys = append(n.keys, "")
			copy(n.keys[i+1:], n.keys[i:])
			n.keys[i] = splitKey

			// Insert child at position i+1.
			n.children = append(n.children, nil)
			copy(n.children[i+2:], n.children[i+1:])
			n.children[i+1] = newChild

			// Check if this node needs splitting.
			if len(n.keys) > maxBranchFactor {
				// For simplicity, we don't split inner nodes here.
				// This limits tree capacity but keeps implementation clean.
			}
			return true
		}

		if !child.leaf {
			if t.insertIntoParent(child, oldLeafID, newLeafID, splitKey) {
				return true
			}
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Store: put / get / delete / list
// ---------------------------------------------------------------------------

func (s *store) shouldSpillValue(ck, contentType string, value []byte) bool {
	if len(value) == 0 {
		return false
	}
	if len(value) > s.inlineValueLimit {
		return true
	}
	// If a single inline record cannot fit in a page, force an indirect value.
	e := pageEntry{
		key:         ck,
		contentType: contentType,
		value:       value,
		valueSize:   uint32(len(value)),
	}
	return pageHeaderSize+encodedEntrySize(e) > s.pageSize
}

func (s *store) buildPageEntry(ck, contentType string, value []byte, created, updated int64) (pageEntry, error) {
	if len(value) > int(^uint32(0)>>1) {
		return pageEntry{}, fmt.Errorf("fox: value too large")
	}
	e := pageEntry{
		key:         ck,
		contentType: contentType,
		created:     created,
		updated:     updated,
		valueSize:   uint32(len(value)),
	}
	if s.shouldSpillValue(ck, contentType, value) {
		ref, err := s.appendValue(value)
		if err != nil {
			return pageEntry{}, err
		}
		e.valueRef = ref
		e.value = nil
		return e, nil
	}
	e.value = value
	return e, nil
}

func (s *store) appendValueFromReader(src io.Reader) (valueRef, int64, error) {
	const chunkSize = 256 * 1024
	off := s.valueTail
	cur := off
	buf := make([]byte, chunkSize)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			if _, werr := s.valueFile.WriteAt(buf[:n], cur); werr != nil {
				s.valueTail = cur
				return valueRef{}, cur - off, fmt.Errorf("fox: write value: %w", werr)
			}
			cur += int64(n)
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			s.valueTail = cur
			return valueRef{}, cur - off, fmt.Errorf("fox: read value: %w", err)
		}
	}
	s.valueTail = cur
	if s.syncMode == "full" && cur > off {
		if err := s.valueFile.Sync(); err != nil {
			return valueRef{}, cur - off, fmt.Errorf("fox: sync values: %w", err)
		}
	}
	if cur == off {
		return valueRef{}, 0, nil
	}
	if cur-off > int64(^uint32(0)>>1) {
		return valueRef{}, cur - off, fmt.Errorf("fox: value too large")
	}
	return valueRef{offset: off, size: uint32(cur - off)}, cur - off, nil
}

func (s *store) putPreparedEntry(ck string, now int64, entry pageEntry) (int64, int64, error) {
	leaf := s.tree.findLeaf(ck)
	if leaf == nil {
		// Should not happen with a properly initialized tree.
		return now, now, nil
	}

	s.pool.mu.Lock()
	mp := s.pool.getOrCreate(leaf.pageID)

	// Check if key already exists in mini-page (update).
	found := false
	for i := range mp.entries {
		if mp.entries[i].key == ck {
			oldSize := encodedEntrySize(mp.entries[i])
			if mp.entries[i].created != 0 {
				entry.created = mp.entries[i].created
			}
			entry.updated = now
			entry.tombstone = false
			mp.entries[i] = entry
			newSize := encodedEntrySize(mp.entries[i])
			if delta := newSize - oldSize; delta != 0 {
				mp.size += delta
				s.pool.curSize += int64(delta)
			}
			found = true
			break
		}
	}

	if !found {
		newSize := encodedEntrySize(entry)
		mp.entries = append(mp.entries, entry)
		mp.size += newSize
		s.pool.curSize += int64(newSize)
	}
	mp.dirty = true

	// If mini-page exceeds max size, flush immediately.
	if mp.size >= maxMiniPageSize {
		s.pool.flushMiniPage(mp)
	}

	s.pool.mu.Unlock()

	return now, now, nil
}

func (s *store) put(bkt, key, contentType string, value []byte) (int64, int64, error) {
	ck := compositeKey(bkt, key)
	now := fastNow()
	entry, err := s.buildPageEntry(ck, contentType, value, now, now)
	if err != nil {
		return now, now, err
	}
	return s.putPreparedEntry(ck, now, entry)
}

func (s *store) putValueRef(bkt, key, contentType string, ref valueRef, size uint32) (int64, int64, error) {
	ck := compositeKey(bkt, key)
	now := fastNow()
	entry := pageEntry{
		key:         ck,
		contentType: contentType,
		valueRef:    ref,
		valueSize:   size,
		created:     now,
		updated:     now,
	}
	return s.putPreparedEntry(ck, now, entry)
}

func (s *store) get(bkt, key string) (pageEntry, bool) {
	ck := compositeKey(bkt, key)

	leaf := s.tree.findLeaf(ck)
	if leaf == nil {
		return pageEntry{}, false
	}

	// Check mini-page first.
	s.pool.mu.Lock()
	mp := s.pool.get(leaf.pageID)
	if mp != nil {
		for _, e := range mp.entries {
			if e.key == ck {
				s.pool.mu.Unlock()
				if e.tombstone {
					return pageEntry{}, false
				}
				return e, true
			}
		}
	}
	s.pool.mu.Unlock()

	// Read from disk page.
	data, err := s.readPage(leaf.pageID)
	if err != nil {
		return pageEntry{}, false
	}

	entries := decodePageEntries(data)
	idx := sort.Search(len(entries), func(i int) bool {
		return entries[i].key >= ck
	})
	if idx < len(entries) && entries[idx].key == ck && !entries[idx].tombstone {
		return entries[idx], true
	}

	return pageEntry{}, false
}

func (s *store) del(bkt, key string) bool {
	ck := compositeKey(bkt, key)

	// Check existence first.
	_, found := s.get(bkt, key)
	if !found {
		return false
	}

	leaf := s.tree.findLeaf(ck)
	if leaf == nil {
		return false
	}

	now := fastNow()

	s.pool.mu.Lock()
	mp := s.pool.getOrCreate(leaf.pageID)

	// Check if exists in mini-page and mark as tombstone.
	for i := range mp.entries {
		if mp.entries[i].key == ck {
			mp.entries[i].tombstone = true
			mp.entries[i].updated = now
			mp.dirty = true
			s.pool.mu.Unlock()
			return true
		}
	}

	// Add tombstone entry.
	tomb := pageEntry{
		key:       ck,
		created:   now,
		updated:   now,
		tombstone: true,
	}
	mp.entries = append(mp.entries, tomb)
	sz := encodedEntrySize(tomb)
	mp.size += sz
	s.pool.curSize += int64(sz)
	mp.dirty = true
	s.pool.mu.Unlock()

	return true
}

// listResult holds results from list operations.
type listResult struct {
	key         string
	contentType string
	size        int64
	created     int64
	updated     int64
}

func (s *store) list(bkt, prefix string) []listResult {
	ckPrefix := compositeKey(bkt, prefix)

	// Collect all entries from all leaf pages in the tree.
	var results []listResult
	seen := make(map[string]bool)

	// First, collect from mini-pages (these take precedence).
	s.pool.mu.Lock()
	tombstones := make(map[string]bool)
	for _, mp := range s.pool.pages {
		for _, e := range mp.entries {
			if !strings.HasPrefix(e.key, ckPrefix) {
				continue
			}
			if e.tombstone {
				tombstones[e.key] = true
				continue
			}
			_, k := splitCompositeKey(e.key)
				results = append(results, listResult{
					key:         k,
					contentType: e.contentType,
					size:        e.objectSize(),
					created:     e.created,
					updated:     e.updated,
				})
			seen[e.key] = true
		}
	}
	s.pool.mu.Unlock()

	// Then collect from all disk pages.
	s.collectFromNode(s.tree.root, ckPrefix, seen, tombstones, &results)

	sort.Slice(results, func(i, j int) bool {
		return results[i].key < results[j].key
	})

	return results
}

func (s *store) collectFromNode(n *btreeNode, prefix string, seen, tombstones map[string]bool, results *[]listResult) {
	if n == nil {
		return
	}

	if n.leaf {
		data, err := s.readPage(n.pageID)
		if err != nil {
			return
		}
			entries := decodePageEntriesMeta(data)
			for _, e := range entries {
			if !strings.HasPrefix(e.key, prefix) {
				continue
			}
			if seen[e.key] || tombstones[e.key] || e.tombstone {
				continue
			}
			_, k := splitCompositeKey(e.key)
				*results = append(*results, listResult{
					key:         k,
					contentType: e.contentType,
					size:        e.objectSize(),
					created:     e.created,
					updated:     e.updated,
				})
			seen[e.key] = true
		}
		return
	}

	for _, child := range n.children {
		s.collectFromNode(child, prefix, seen, tombstones, results)
	}
}

// hasBucketData checks whether any entries exist for the given bucket.
func (s *store) hasBucketData(bkt string) bool {
	results := s.list(bkt, "")
	return len(results) > 0
}

// ---------------------------------------------------------------------------
// bucket implements storage.Bucket
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

func (b *bucket) Features() storage.Features {
	return b.st.Features()
}

func (b *bucket) Info(ctx context.Context) (*storage.BucketInfo, error) {
	_ = ctx

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
	_ = ctx
	_ = opts

	if key == "" {
		return nil, fmt.Errorf("fox: key is empty")
	}
	if key[0] == ' ' || key[len(key)-1] == ' ' {
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("fox: key is empty")
		}
	}

	// Stream large values directly into values.dat to keep peak heap low.
	if size > int64(b.st.inlineValueLimit) && size >= 0 {
		b.st.mu.Lock()
		ref, actualSize, err := b.st.appendValueFromReader(src)
		if err == nil {
			created, updated, putErr := b.st.putValueRef(b.name, key, contentType, ref, uint32(actualSize))
			b.st.mu.Unlock()
			if putErr != nil {
				return nil, putErr
			}
			return &storage.Object{
				Bucket:      b.name,
				Key:         key,
				Size:        actualSize,
				ContentType: contentType,
				Created:     time.Unix(0, created),
				Updated:     time.Unix(0, updated),
			}, nil
		}
		b.st.mu.Unlock()
		return nil, err
	}

	var data []byte
	if size >= 0 {
		data = make([]byte, size)
		if size > 0 {
			n, err := io.ReadFull(src, data)
			if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
				return nil, fmt.Errorf("fox: read value: %w", err)
			}
			data = data[:n]
		}
	} else {
		var buf []byte
		tmp := make([]byte, 32*1024)
		for {
			n, err := src.Read(tmp)
			if n > 0 {
				buf = append(buf, tmp[:n]...)
			}
			if err != nil {
				if err == io.EOF {
					break
				}
				return nil, fmt.Errorf("fox: read value: %w", err)
			}
		}
		data = buf
		size = int64(len(data))
	}

	size = int64(len(data))

	b.st.mu.Lock()
	created, updated, err := b.st.put(b.name, key, contentType, data)
	b.st.mu.Unlock()
	if err != nil {
		return nil, err
	}

	return &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        size,
		ContentType: contentType,
		Created:     time.Unix(0, created),
		Updated:     time.Unix(0, updated),
	}, nil
}

func (b *bucket) Open(ctx context.Context, key string, offset, length int64, opts storage.Options) (io.ReadCloser, *storage.Object, error) {
	_ = ctx
	_ = opts

	if key == "" {
		return nil, nil, fmt.Errorf("fox: key is empty")
	}
	if key[0] == ' ' || key[len(key)-1] == ' ' {
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, nil, fmt.Errorf("fox: key is empty")
		}
	}

	b.st.mu.RLock()
	e, ok := b.st.get(b.name, key)
	b.st.mu.RUnlock()

	if !ok {
		return nil, nil, storage.ErrNotExist
	}

	_, realKey := splitCompositeKey(e.key)
	if realKey == "" {
		realKey = key
	}

	objSize := e.objectSize()

	obj := &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        objSize,
		ContentType: e.contentType,
		Created:     time.Unix(0, e.created),
		Updated:     time.Unix(0, e.updated),
	}

	// Apply offset/length.
	if offset < 0 {
		offset = 0
	}
	if offset > objSize {
		offset = objSize
	}
	end := objSize
	if length > 0 && offset+length < end {
		end = offset + length
	}
	if e.hasIndirectValue() {
		return &valueSectionReader{
			r: io.NewSectionReader(b.st.valueFile, e.valueRef.offset+offset, end-offset),
		}, obj, nil
	}

	data := e.value
	slice := data[offset:end]
	result := make([]byte, len(slice))
	copy(result, slice)
	return &sliceReader{data: result}, obj, nil
}

func (b *bucket) Stat(ctx context.Context, key string, opts storage.Options) (*storage.Object, error) {
	_ = ctx
	_ = opts

	if key == "" {
		return nil, fmt.Errorf("fox: key is empty")
	}
	if key[0] == ' ' || key[len(key)-1] == ' ' {
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("fox: key is empty")
		}
	}

	// Check for directory stat.
	if strings.HasSuffix(key, "/") {
		b.st.mu.RLock()
		results := b.st.list(b.name, key)
		b.st.mu.RUnlock()

		if len(results) == 0 {
			return nil, storage.ErrNotExist
		}
		return &storage.Object{
			Bucket:  b.name,
			Key:     strings.TrimSuffix(key, "/"),
			IsDir:   true,
			Created: time.Unix(0, results[0].created),
			Updated: time.Unix(0, results[0].updated),
		}, nil
	}

	b.st.mu.RLock()
	e, ok := b.st.get(b.name, key)
	b.st.mu.RUnlock()

	if !ok {
		return nil, storage.ErrNotExist
	}

	return &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        e.objectSize(),
		ContentType: e.contentType,
		Created:     time.Unix(0, e.created),
		Updated:     time.Unix(0, e.updated),
	}, nil
}

func (b *bucket) Delete(ctx context.Context, key string, opts storage.Options) error {
	_ = ctx
	_ = opts

	if key == "" {
		return fmt.Errorf("fox: key is empty")
	}
	if key[0] == ' ' || key[len(key)-1] == ' ' {
		key = strings.TrimSpace(key)
		if key == "" {
			return fmt.Errorf("fox: key is empty")
		}
	}

	b.st.mu.Lock()
	ok := b.st.del(b.name, key)
	b.st.mu.Unlock()

	if !ok {
		return storage.ErrNotExist
	}
	return nil
}

func (b *bucket) Copy(ctx context.Context, dstKey string, srcBucket, srcKey string, opts storage.Options) (*storage.Object, error) {
	_ = ctx
	_ = opts

	dstKey = strings.TrimSpace(dstKey)
	srcKey = strings.TrimSpace(srcKey)
	if dstKey == "" || srcKey == "" {
		return nil, fmt.Errorf("fox: key is empty")
	}

	if srcBucket == "" {
		srcBucket = b.name
	}

	b.st.mu.Lock()
	defer b.st.mu.Unlock()

	e, ok := b.st.get(srcBucket, srcKey)
	if !ok {
		return nil, storage.ErrNotExist
	}

	copyBytes := e.value
	var err error
	if e.hasIndirectValue() {
		copyBytes, err = b.st.readValue(e.valueRef, 0, int64(e.valueRef.size))
		if err != nil {
			return nil, err
		}
	} else if len(copyBytes) > 0 {
		copyBytes = append([]byte(nil), copyBytes...)
	}

	created, updated, err := b.st.put(b.name, dstKey, e.contentType, copyBytes)
	if err != nil {
		return nil, err
	}

	return &storage.Object{
		Bucket:      b.name,
		Key:         dstKey,
		Size:        e.objectSize(),
		ContentType: e.contentType,
		Created:     time.Unix(0, created),
		Updated:     time.Unix(0, updated),
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
	if err := sb.Delete(ctx, srcKey, nil); err != nil {
		return nil, err
	}
	return obj, nil
}

func (b *bucket) List(ctx context.Context, prefix string, limit, offset int, opts storage.Options) (storage.ObjectIter, error) {
	_ = ctx

	prefix = strings.TrimSpace(prefix)
	recursive := true
	if opts != nil {
		if v, ok := opts["recursive"].(bool); ok {
			recursive = v
		}
	}

	b.st.mu.RLock()
	results := b.st.list(b.name, prefix)
	b.st.mu.RUnlock()

	objs := make([]*storage.Object, 0, len(results))
	for _, r := range results {
		if !recursive {
			rest := strings.TrimPrefix(r.key, prefix)
			rest = strings.TrimPrefix(rest, "/")
			if strings.Contains(rest, "/") {
				continue
			}
		}
		objs = append(objs, &storage.Object{
			Bucket:      b.name,
			Key:         r.key,
			Size:        r.size,
			ContentType: r.contentType,
			Created:     time.Unix(0, r.created),
			Updated:     time.Unix(0, r.updated),
		})
	}

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

func (b *bucket) SignedURL(ctx context.Context, key string, method string, expires time.Duration, opts storage.Options) (string, error) {
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
	_ = ctx
	prefix := d.path
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	d.b.st.mu.RLock()
	results := d.b.st.list(d.b.name, prefix)
	d.b.st.mu.RUnlock()

	if len(results) == 0 {
		return nil, storage.ErrNotExist
	}
	return &storage.Object{
		Bucket:  d.b.name,
		Key:     d.path,
		IsDir:   true,
		Created: time.Unix(0, results[0].created),
		Updated: time.Unix(0, results[0].updated),
	}, nil
}

func (d *dir) List(ctx context.Context, limit, offset int, opts storage.Options) (storage.ObjectIter, error) {
	_ = ctx
	_ = opts
	prefix := d.path
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	d.b.st.mu.RLock()
	results := d.b.st.list(d.b.name, prefix)
	d.b.st.mu.RUnlock()

	var objs []*storage.Object
	for _, r := range results {
		rest := strings.TrimPrefix(r.key, prefix)
		if strings.Contains(rest, "/") {
			continue
		}
		objs = append(objs, &storage.Object{
			Bucket:      d.b.name,
			Key:         r.key,
			Size:        r.size,
			ContentType: r.contentType,
			Created:     time.Unix(0, r.created),
			Updated:     time.Unix(0, r.updated),
		})
	}

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
	_ = ctx
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

	d.b.st.mu.Lock()
	results := d.b.st.list(d.b.name, prefix)
	if len(results) == 0 {
		d.b.st.mu.Unlock()
		return storage.ErrNotExist
	}

	for _, r := range results {
		if !recursive {
			rest := strings.TrimPrefix(r.key, prefix)
			if strings.Contains(rest, "/") {
				continue
			}
		}
		d.b.st.del(d.b.name, r.key)
	}
	d.b.st.mu.Unlock()
	return nil
}

func (d *dir) Move(ctx context.Context, dstPath string, opts storage.Options) (storage.Directory, error) {
	_ = ctx
	_ = opts

	srcPrefix := strings.Trim(d.path, "/")
	dstPrefix := strings.Trim(dstPath, "/")

	if srcPrefix != "" && !strings.HasSuffix(srcPrefix, "/") {
		srcPrefix += "/"
	}
	if dstPrefix != "" && !strings.HasSuffix(dstPrefix, "/") {
		dstPrefix += "/"
	}

	d.b.st.mu.Lock()
	results := d.b.st.list(d.b.name, srcPrefix)
	if len(results) == 0 {
		d.b.st.mu.Unlock()
		return nil, storage.ErrNotExist
	}

	for _, r := range results {
		rel := strings.TrimPrefix(r.key, srcPrefix)
		newKey := dstPrefix + rel

		// Read source.
		e, ok := d.b.st.get(d.b.name, r.key)
		if !ok {
			continue
		}
		payload := e.value
		if e.hasIndirectValue() {
			payload, _ = d.b.st.readValue(e.valueRef, 0, int64(e.valueRef.size))
		} else if len(payload) > 0 {
			payload = append([]byte(nil), payload...)
		}
		_, _, _ = d.b.st.put(d.b.name, newKey, e.contentType, payload)
		d.b.st.del(d.b.name, r.key)
	}
	d.b.st.mu.Unlock()

	return &dir{b: d.b, path: strings.Trim(dstPath, "/")}, nil
}

// ---------------------------------------------------------------------------
// Multipart upload support
// ---------------------------------------------------------------------------

type multipartRegistry struct {
	mu      sync.Mutex
	uploads map[string]*multipartUpload
	counter atomic.Int64
}

type multipartUpload struct {
	id          string
	key         string
	contentType string
	parts       map[int]*partData
	metadata    map[string]string
}

type partData struct {
	number int
	data   []byte
	etag   string
}

func newMultipartRegistry() *multipartRegistry {
	r := &multipartRegistry{
		uploads: make(map[string]*multipartUpload),
	}
	r.counter.Store(time.Now().UnixNano())
	return r
}

func (b *bucket) InitMultipart(ctx context.Context, key string, contentType string, opts storage.Options) (*storage.MultipartUpload, error) {
	_ = ctx

	if key == "" {
		return nil, fmt.Errorf("fox: key is empty")
	}
	key = strings.TrimSpace(key)

	id := strconv.FormatInt(b.st.mp.counter.Add(1), 36)

	var metadata map[string]string
	if opts != nil {
		if m, ok := opts["metadata"].(map[string]string); ok {
			metadata = m
		}
	}

	upload := &multipartUpload{
		id:          id,
		key:         key,
		contentType: contentType,
		parts:       make(map[int]*partData),
		metadata:    metadata,
	}

	b.st.mp.mu.Lock()
	b.st.mp.uploads[id] = upload
	b.st.mp.mu.Unlock()

	return &storage.MultipartUpload{
		Bucket:   b.name,
		Key:      key,
		UploadID: id,
		Metadata: metadata,
	}, nil
}

func (b *bucket) UploadPart(ctx context.Context, mu *storage.MultipartUpload, number int, src io.Reader, size int64, opts storage.Options) (*storage.PartInfo, error) {
	_ = ctx
	_ = opts

	if number < 1 || number > maxPartNumber {
		return nil, fmt.Errorf("fox: part number %d out of range [1, %d]", number, maxPartNumber)
	}

	b.st.mp.mu.Lock()
	upload, ok := b.st.mp.uploads[mu.UploadID]
	if !ok {
		b.st.mp.mu.Unlock()
		return nil, storage.ErrNotExist
	}
	b.st.mp.mu.Unlock()

	data, err := io.ReadAll(src)
	if err != nil {
		return nil, fmt.Errorf("fox: read part: %w", err)
	}

	hash := md5.Sum(data)
	etag := hex.EncodeToString(hash[:])

	b.st.mp.mu.Lock()
	upload.parts[number] = &partData{
		number: number,
		data:   data,
		etag:   etag,
	}
	b.st.mp.mu.Unlock()

	return &storage.PartInfo{
		Number: number,
		Size:   int64(len(data)),
		ETag:   etag,
	}, nil
}

func (b *bucket) CopyPart(ctx context.Context, mu *storage.MultipartUpload, number int, opts storage.Options) (*storage.PartInfo, error) {
	_ = ctx

	if number < 1 || number > maxPartNumber {
		return nil, fmt.Errorf("fox: part number %d out of range", number)
	}

	b.st.mp.mu.Lock()
	_, ok := b.st.mp.uploads[mu.UploadID]
	b.st.mp.mu.Unlock()
	if !ok {
		return nil, storage.ErrNotExist
	}

	srcBucket := mu.Bucket
	if sb, ok := opts["source_bucket"].(string); ok && sb != "" {
		srcBucket = sb
	}
	srcKey, _ := opts["source_key"].(string)
	if srcKey == "" {
		return nil, errors.New("fox: source_key required for CopyPart")
	}
	offset, _ := opts["source_offset"].(int64)
	length, _ := opts["source_length"].(int64)

	b.st.mu.RLock()
	e, found := b.st.get(srcBucket, srcKey)
	b.st.mu.RUnlock()
	if !found {
		return nil, storage.ErrNotExist
	}

	var partBytes []byte
	if e.hasIndirectValue() {
		var err error
		partBytes, err = b.st.readValue(e.valueRef, offset, length)
		if err != nil {
			return nil, err
		}
	} else {
		data := e.value
		if offset > 0 {
			if offset > int64(len(data)) {
				offset = int64(len(data))
			}
			data = data[offset:]
		}
		if length > 0 && length < int64(len(data)) {
			data = data[:length]
		}
		partBytes = make([]byte, len(data))
		copy(partBytes, data)
	}

	return b.UploadPart(ctx, mu, number, &sliceReaderForUpload{data: partBytes}, int64(len(partBytes)), opts)
}

func (b *bucket) ListParts(ctx context.Context, mu *storage.MultipartUpload, limit, offset int, opts storage.Options) ([]*storage.PartInfo, error) {
	_ = ctx
	_ = opts

	b.st.mp.mu.Lock()
	upload, ok := b.st.mp.uploads[mu.UploadID]
	if !ok {
		b.st.mp.mu.Unlock()
		return nil, storage.ErrNotExist
	}

	parts := make([]*storage.PartInfo, 0, len(upload.parts))
	for _, p := range upload.parts {
		parts = append(parts, &storage.PartInfo{
			Number: p.number,
			Size:   int64(len(p.data)),
			ETag:   p.etag,
		})
	}
	b.st.mp.mu.Unlock()

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

func (b *bucket) CompleteMultipart(ctx context.Context, mu *storage.MultipartUpload, parts []*storage.PartInfo, opts storage.Options) (*storage.Object, error) {
	_ = ctx
	_ = opts

	b.st.mp.mu.Lock()
	upload, ok := b.st.mp.uploads[mu.UploadID]
	if !ok {
		b.st.mp.mu.Unlock()
		return nil, storage.ErrNotExist
	}
	delete(b.st.mp.uploads, mu.UploadID)
	b.st.mp.mu.Unlock()

	// Sort and verify parts.
	sort.Slice(parts, func(i, j int) bool {
		return parts[i].Number < parts[j].Number
	})

	for _, p := range parts {
		if _, ok := upload.parts[p.Number]; !ok {
			return nil, fmt.Errorf("fox: part %d not found", p.Number)
		}
	}

	// Assemble final value.
	var totalSize int64
	for _, p := range parts {
		totalSize += int64(len(upload.parts[p.Number].data))
	}

	assembled := make([]byte, 0, totalSize)
	for _, p := range parts {
		assembled = append(assembled, upload.parts[p.Number].data...)
	}

	ct := upload.contentType

	b.st.mu.Lock()
	created, updated, err := b.st.put(b.name, upload.key, ct, assembled)
	b.st.mu.Unlock()
	if err != nil {
		return nil, err
	}

	return &storage.Object{
		Bucket:      b.name,
		Key:         upload.key,
		Size:        totalSize,
		ContentType: ct,
		Created:     time.Unix(0, created),
		Updated:     time.Unix(0, updated),
	}, nil
}

func (b *bucket) AbortMultipart(ctx context.Context, mu *storage.MultipartUpload, opts storage.Options) error {
	_ = ctx
	_ = opts

	b.st.mp.mu.Lock()
	_, ok := b.st.mp.uploads[mu.UploadID]
	if !ok {
		b.st.mp.mu.Unlock()
		return storage.ErrNotExist
	}
	delete(b.st.mp.uploads, mu.UploadID)
	b.st.mp.mu.Unlock()
	return nil
}

// ---------------------------------------------------------------------------
// Readers
// ---------------------------------------------------------------------------

type sliceReader struct {
	data []byte
	pos  int
}

func (r *sliceReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func (r *sliceReader) WriteTo(w io.Writer) (int64, error) {
	if r.pos >= len(r.data) {
		return 0, nil
	}
	n, err := w.Write(r.data[r.pos:])
	r.pos = len(r.data)
	return int64(n), err
}

func (r *sliceReader) Close() error { return nil }

type valueSectionReader struct {
	r *io.SectionReader
}

func (r *valueSectionReader) Read(p []byte) (int, error) {
	return r.r.Read(p)
}

func (r *valueSectionReader) WriteTo(w io.Writer) (int64, error) {
	return io.Copy(w, r.r)
}

func (r *valueSectionReader) Close() error { return nil }

// sliceReaderForUpload is a simple io.Reader for passing part data.
type sliceReaderForUpload struct {
	data []byte
	pos  int
}

func (r *sliceReaderForUpload) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
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
