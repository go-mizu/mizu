// Package falcon implements a two-tier hash-indexed storage driver inspired by
// the F2 paper (FASTER evolved, VLDB 2025).
//
// Architecture:
//   - Hot tier: sharded in-memory concurrent hash map for recently written/accessed entries
//   - Cold tier: on-disk hash-indexed file with 256-byte aligned slots
//   - Promotion: reading a cold entry copies it to the hot tier
//   - Demotion: when hot tier exceeds capacity, oldest entries flush to cold tier
//
// DSN format:
//
//	falcon:///path/to/data?sync=none&hot_size=1048576
package falcon

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash/fnv"
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
	storage.Register("falcon", &driver{})
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const (
	numShards     = 256
	slotSize      = 256
	coldHeaderLen = 64
	maxPartNumber = 10000

	// Slot flags.
	flagEmpty     = 0x00
	flagOccupied  = 0x01
	flagTombstone = 0x02
	flagOverflow  = 0x04

	// Slot field sizes.
	fieldHash    = 8
	fieldKeyLen  = 2
	fieldCTLen   = 2
	fieldValLen  = 8
	fieldCreated = 8
	fieldUpdated = 8
	fieldFlags   = 1

	// Fixed overhead inside a 256B slot (before variable-length key/ct/val).
	slotFixedOverhead = fieldHash + fieldKeyLen + fieldCTLen + fieldValLen + fieldCreated + fieldUpdated + fieldFlags // 37

	defaultHotSize  = 1_048_576
	defaultColdSlot = 1 << 20 // initial cold file capacity in slots (~256 MB)
	maxColdSlots    = 1 << 24 // 16M slots, ~256 MB at 16 bytes/slot

	maxBuckets = 10000

	dirPerms  = 0750
	filePerms = 0600
)

// ---------------------------------------------------------------------------
// Driver
// ---------------------------------------------------------------------------

type driver struct{}

func (d *driver) Open(_ context.Context, dsn string) (storage.Storage, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("falcon: parse dsn: %w", err)
	}
	if u.Scheme != "falcon" && u.Scheme != "" {
		return nil, fmt.Errorf("falcon: unexpected scheme %q", u.Scheme)
	}

	root := filepath.Clean(u.Path)
	if root == "" || root == "." {
		root = "/tmp/falcon-data"
	}

	syncMode := u.Query().Get("sync")
	if syncMode == "" {
		syncMode = "none"
	}

	hotSize := defaultHotSize
	if hs := u.Query().Get("hot_size"); hs != "" {
		if n, err := strconv.Atoi(hs); err == nil && n > 0 {
			hotSize = n
		}
	}

	if err := os.MkdirAll(root, dirPerms); err != nil {
		return nil, fmt.Errorf("falcon: mkdir root: %w", err)
	}

	cold, err := openColdFile(filepath.Join(root, "cold.dat"))
	if err != nil {
		return nil, err
	}

	over, err := openOverflowFile(filepath.Join(root, "overflow.dat"))
	if err != nil {
		cold.close()
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	st := &store{
		root:     root,
		syncMode: syncMode,
		hotSize:  hotSize,
		cold:     cold,
		overflow: over,
		buckets:  make(map[string]time.Time),
		mp:       newMultipartRegistry(),
		stopTick: make(chan struct{}),
		ctx:      ctx,
		cancel:   cancel,
	}

	for i := range st.hot {
		st.hot[i] = &hotShard{m: make(map[string]*hotEntry, 256)}
	}

	// Start per-store ticker for cachedNano.
	go func() {
		t := time.NewTicker(1 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-st.stopTick:
				return
			case <-t.C:
				cachedNano.Store(time.Now().UnixNano())
			}
		}
	}()

	return st, nil
}

// ---------------------------------------------------------------------------
// Cached time
// ---------------------------------------------------------------------------

var cachedNano atomic.Int64

func init() {
	cachedNano.Store(time.Now().UnixNano())
}

func fastNow() int64  { return cachedNano.Load() }
func fastTime() time.Time { return time.Unix(0, fastNow()) }

// ---------------------------------------------------------------------------
// FNV helpers
// ---------------------------------------------------------------------------

func fnv1a32(s string) uint32 {
	h := uint32(2166136261)
	for i := range len(s) {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return h
}

func fnv1a64(b []byte) uint64 {
	h := fnv.New64a()
	h.Write(b)
	return h.Sum64()
}

func compositeKey(bucket, key string) string {
	return bucket + "\x00" + key
}

// ---------------------------------------------------------------------------
// Hot tier
// ---------------------------------------------------------------------------

type hotEntry struct {
	value       []byte
	contentType string
	created     int64
	updated     int64
	size        int64
}

type hotShard struct {
	mu sync.RWMutex
	m  map[string]*hotEntry
}

// ---------------------------------------------------------------------------
// Cold file
// ---------------------------------------------------------------------------
//
// Layout: [header 64B] [slot0 256B] [slot1 256B] ...
//
// Each 256B slot:
//   hash(8) | keyLen(2) | key(...) | ctLen(2) | ct(...) | valLen(8) |
//   value(...) or overflowOffset(8)+overflowLen(8) | created(8) | updated(8) | flags(1)
//
// Flags: 0x01=occupied 0x02=tombstone 0x04=overflow

type coldFile struct {
	mu       sync.RWMutex
	f        *os.File
	numSlots int64
	count    int64
}

func openColdFile(path string) (*coldFile, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, filePerms)
	if err != nil {
		return nil, fmt.Errorf("falcon: open cold file: %w", err)
	}

	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("falcon: stat cold file: %w", err)
	}

	cf := &coldFile{f: f}

	if info.Size() < coldHeaderLen {
		// Initialize header + initial slots.
		cf.numSlots = int64(defaultColdSlot)
		totalSize := coldHeaderLen + cf.numSlots*slotSize
		if err := f.Truncate(totalSize); err != nil {
			f.Close()
			return nil, fmt.Errorf("falcon: truncate cold file: %w", err)
		}
		// Write slot count into header.
		var hdr [coldHeaderLen]byte
		binary.LittleEndian.PutUint64(hdr[0:8], uint64(cf.numSlots))
		if _, err := f.WriteAt(hdr[:], 0); err != nil {
			f.Close()
			return nil, fmt.Errorf("falcon: write cold header: %w", err)
		}
	} else {
		var hdr [coldHeaderLen]byte
		if _, err := f.ReadAt(hdr[:], 0); err != nil {
			f.Close()
			return nil, fmt.Errorf("falcon: read cold header: %w", err)
		}
		cf.numSlots = int64(binary.LittleEndian.Uint64(hdr[0:8]))
		if cf.numSlots == 0 {
			cf.numSlots = int64(defaultColdSlot)
		}

		// Count existing entries.
		cf.count = cf.countOccupied()
	}

	return cf, nil
}

func (cf *coldFile) close() error {
	if cf.f != nil {
		return cf.f.Close()
	}
	return nil
}

func (cf *coldFile) slotOffset(idx int64) int64 {
	return coldHeaderLen + idx*slotSize
}

// readSlot reads a single 256B slot.
func (cf *coldFile) readSlot(idx int64) ([slotSize]byte, error) {
	var buf [slotSize]byte
	_, err := cf.f.ReadAt(buf[:], cf.slotOffset(idx))
	return buf, err
}

// writeSlot writes a single 256B slot.
func (cf *coldFile) writeSlot(idx int64, buf [slotSize]byte) error {
	_, err := cf.f.WriteAt(buf[:], cf.slotOffset(idx))
	return err
}

func (cf *coldFile) countOccupied() int64 {
	var count int64
	var buf [slotSize]byte
	for i := int64(0); i < cf.numSlots; i++ {
		if _, err := cf.f.ReadAt(buf[:], cf.slotOffset(i)); err != nil {
			break
		}
		flags := buf[slotSize-1]
		if flags&flagOccupied != 0 && flags&flagTombstone == 0 {
			count++
		}
	}
	return count
}

// probe finds the slot index for a composite key, or the first empty slot.
// Returns (slotIndex, found).
func (cf *coldFile) probe(ck string, hash uint64) (int64, bool, error) {
	start := int64(hash % uint64(cf.numSlots))
	for i := int64(0); i < cf.numSlots; i++ {
		idx := (start + i) % cf.numSlots
		buf, err := cf.readSlot(idx)
		if err != nil {
			return 0, false, err
		}
		flags := buf[slotSize-1]
		if flags == flagEmpty {
			return idx, false, nil
		}
		if flags&flagTombstone != 0 {
			continue
		}
		if flags&flagOccupied != 0 {
			slotHash := binary.LittleEndian.Uint64(buf[0:8])
			if slotHash == hash {
				kl := int(binary.LittleEndian.Uint16(buf[8:10]))
				if kl <= len(ck) && kl > 0 {
					slotKey := string(buf[10 : 10+kl])
					if slotKey == ck {
						return idx, true, nil
					}
				}
			}
		}
	}
	return 0, false, fmt.Errorf("falcon: cold file full")
}

// encodeSlot creates a 256B slot buffer for the given entry.
func (cf *coldFile) encodeSlot(ck string, hash uint64, e *hotEntry, over *overflowFile) ([slotSize]byte, error) {
	var buf [slotSize]byte

	binary.LittleEndian.PutUint64(buf[0:8], hash)

	kl := len(ck)
	binary.LittleEndian.PutUint16(buf[8:10], uint16(kl))
	pos := 10
	copy(buf[pos:], ck)
	pos += kl

	cl := len(e.contentType)
	binary.LittleEndian.PutUint16(buf[pos:pos+2], uint16(cl))
	pos += 2
	copy(buf[pos:], e.contentType)
	pos += cl

	// Calculate available space for inline value.
	// Need: valLen(8) + created(8) + updated(8) + flags(1) = 25 bytes at the end.
	tailFixed := fieldValLen + fieldCreated + fieldUpdated + fieldFlags // 25
	availableForValue := slotSize - pos - tailFixed

	binary.LittleEndian.PutUint64(buf[pos:pos+8], uint64(e.size))
	pos += 8

	if int(e.size) <= availableForValue {
		// Inline value.
		copy(buf[pos:], e.value)
		pos += int(e.size)
	} else {
		// Overflow: write value to overflow file and store offset+length.
		off, err := over.write(e.value)
		if err != nil {
			return buf, err
		}
		binary.LittleEndian.PutUint64(buf[pos:pos+8], uint64(off))
		pos += 8
		binary.LittleEndian.PutUint64(buf[pos:pos+8], uint64(e.size))
		pos += 8
		buf[slotSize-1] = flagOccupied | flagOverflow
		binary.LittleEndian.PutUint64(buf[slotSize-1-fieldUpdated-fieldCreated:slotSize-1-fieldUpdated], uint64(e.created))
		binary.LittleEndian.PutUint64(buf[slotSize-1-fieldUpdated:slotSize-1], uint64(e.updated))
		return buf, nil
	}

	// Write timestamps at known positions from the end.
	binary.LittleEndian.PutUint64(buf[slotSize-1-fieldUpdated-fieldCreated:slotSize-1-fieldUpdated], uint64(e.created))
	binary.LittleEndian.PutUint64(buf[slotSize-1-fieldUpdated:slotSize-1], uint64(e.updated))
	buf[slotSize-1] = flagOccupied

	return buf, nil
}

// decodeSlot extracts entry data from a 256B slot.
func (cf *coldFile) decodeSlot(buf [slotSize]byte, over *overflowFile) (ck string, e *hotEntry, err error) {
	flags := buf[slotSize-1]
	if flags&flagOccupied == 0 || flags&flagTombstone != 0 {
		return "", nil, fmt.Errorf("falcon: slot not occupied")
	}

	kl := int(binary.LittleEndian.Uint16(buf[8:10]))
	pos := 10
	ck = string(buf[pos : pos+kl])
	pos += kl

	cl := int(binary.LittleEndian.Uint16(buf[pos : pos+2]))
	pos += 2
	ct := string(buf[pos : pos+cl])
	pos += cl

	valLen := int64(binary.LittleEndian.Uint64(buf[pos : pos+8]))
	pos += 8

	created := int64(binary.LittleEndian.Uint64(buf[slotSize-1-fieldUpdated-fieldCreated : slotSize-1-fieldUpdated]))
	updated := int64(binary.LittleEndian.Uint64(buf[slotSize-1-fieldUpdated : slotSize-1]))

	var value []byte
	if flags&flagOverflow != 0 {
		off := int64(binary.LittleEndian.Uint64(buf[pos : pos+8]))
		pos += 8
		overLen := int64(binary.LittleEndian.Uint64(buf[pos : pos+8]))
		value, err = over.read(off, overLen)
		if err != nil {
			return "", nil, err
		}
	} else {
		value = make([]byte, valLen)
		copy(value, buf[pos:pos+int(valLen)])
	}

	e = &hotEntry{
		value:       value,
		contentType: ct,
		created:     created,
		updated:     updated,
		size:        valLen,
	}
	return ck, e, nil
}

// put writes an entry to the cold file.
func (cf *coldFile) put(ck string, hash uint64, e *hotEntry, over *overflowFile) error {
	// Check load factor and grow if needed.
	if float64(cf.count+1)/float64(cf.numSlots) > 0.7 {
		if err := cf.grow(over); err != nil {
			return err
		}
	}

	idx, found, err := cf.probe(ck, hash)
	if err != nil {
		return err
	}

	buf, err := cf.encodeSlot(ck, hash, e, over)
	if err != nil {
		return err
	}

	if err := cf.writeSlot(idx, buf); err != nil {
		return err
	}

	if !found {
		cf.count++
	}
	return nil
}

// get reads an entry from the cold file.
func (cf *coldFile) get(ck string, hash uint64, over *overflowFile) (*hotEntry, bool, error) {
	idx, found, err := cf.probe(ck, hash)
	if err != nil || !found {
		return nil, false, err
	}

	buf, err := cf.readSlot(idx)
	if err != nil {
		return nil, false, err
	}

	_, e, err := cf.decodeSlot(buf, over)
	if err != nil {
		return nil, false, err
	}

	return e, true, nil
}

// markTombstone marks a cold slot as deleted.
func (cf *coldFile) markTombstone(ck string, hash uint64) error {
	idx, found, err := cf.probe(ck, hash)
	if err != nil || !found {
		return err
	}

	buf, err := cf.readSlot(idx)
	if err != nil {
		return err
	}

	buf[slotSize-1] = flagTombstone
	if err := cf.writeSlot(idx, buf); err != nil {
		return err
	}
	cf.count--
	return nil
}

// grow doubles the cold file capacity and rehashes all entries.
func (cf *coldFile) grow(over *overflowFile) error {
	oldSlots := cf.numSlots
	newSlots := oldSlots * 2

	if newSlots > maxColdSlots {
		return fmt.Errorf("falcon: cold file would exceed max capacity (%d slots)", maxColdSlots)
	}

	// Read all existing entries.
	type entry struct {
		ck   string
		hash uint64
		buf  [slotSize]byte
	}
	var entries []entry

	for i := int64(0); i < oldSlots; i++ {
		buf, err := cf.readSlot(i)
		if err != nil {
			continue
		}
		flags := buf[slotSize-1]
		if flags&flagOccupied == 0 || flags&flagTombstone != 0 {
			continue
		}

		kl := int(binary.LittleEndian.Uint16(buf[8:10]))
		ck := string(buf[10 : 10+kl])
		hash := binary.LittleEndian.Uint64(buf[0:8])

		entries = append(entries, entry{ck: ck, hash: hash, buf: buf})
	}

	// Expand file.
	cf.numSlots = newSlots
	totalSize := coldHeaderLen + cf.numSlots*slotSize
	if err := cf.f.Truncate(totalSize); err != nil {
		return fmt.Errorf("falcon: grow cold file: %w", err)
	}

	// Update header.
	var hdr [coldHeaderLen]byte
	binary.LittleEndian.PutUint64(hdr[0:8], uint64(cf.numSlots))
	if _, err := cf.f.WriteAt(hdr[:], 0); err != nil {
		return fmt.Errorf("falcon: write cold header: %w", err)
	}

	// Zero out all slots.
	zero := make([]byte, slotSize)
	for i := int64(0); i < newSlots; i++ {
		if _, err := cf.f.WriteAt(zero, cf.slotOffset(i)); err != nil {
			return fmt.Errorf("falcon: zero slot: %w", err)
		}
	}

	// Reinsert entries.
	cf.count = 0
	for _, ent := range entries {
		start := int64(ent.hash % uint64(cf.numSlots))
		for j := int64(0); j < cf.numSlots; j++ {
			idx := (start + j) % cf.numSlots
			var check [slotSize]byte
			if _, err := cf.f.ReadAt(check[:], cf.slotOffset(idx)); err != nil {
				continue
			}
			if check[slotSize-1] == flagEmpty {
				if err := cf.writeSlot(idx, ent.buf); err != nil {
					return err
				}
				cf.count++
				break
			}
		}
	}

	return nil
}

// allEntries returns all live (ck, hotEntry) pairs from cold storage.
func (cf *coldFile) allEntries(over *overflowFile) []coldEntry {
	var result []coldEntry
	for i := int64(0); i < cf.numSlots; i++ {
		buf, err := cf.readSlot(i)
		if err != nil {
			continue
		}
		flags := buf[slotSize-1]
		if flags&flagOccupied == 0 || flags&flagTombstone != 0 {
			continue
		}
		ck, e, err := cf.decodeSlot(buf, over)
		if err != nil {
			continue
		}
		result = append(result, coldEntry{ck: ck, entry: e})
	}
	return result
}

type coldEntry struct {
	ck    string
	entry *hotEntry
}

// ---------------------------------------------------------------------------
// Overflow file
// ---------------------------------------------------------------------------

type overflowFile struct {
	mu   sync.Mutex
	f    *os.File
	tail int64
}

func openOverflowFile(path string) (*overflowFile, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, filePerms)
	if err != nil {
		return nil, fmt.Errorf("falcon: open overflow file: %w", err)
	}

	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("falcon: stat overflow file: %w", err)
	}

	return &overflowFile{f: f, tail: info.Size()}, nil
}

func (of *overflowFile) write(data []byte) (int64, error) {
	of.mu.Lock()
	defer of.mu.Unlock()

	off := of.tail
	if _, err := of.f.WriteAt(data, off); err != nil {
		return 0, fmt.Errorf("falcon: write overflow: %w", err)
	}
	of.tail += int64(len(data))
	return off, nil
}

func (of *overflowFile) read(off, length int64) ([]byte, error) {
	buf := make([]byte, length)
	if _, err := of.f.ReadAt(buf, off); err != nil {
		return nil, fmt.Errorf("falcon: read overflow: %w", err)
	}
	return buf, nil
}

func (of *overflowFile) close() error {
	if of.f != nil {
		return of.f.Close()
	}
	return nil
}

// ---------------------------------------------------------------------------
// Store (storage.Storage)
// ---------------------------------------------------------------------------

type store struct {
	root     string
	syncMode string
	hotSize  int
	cold     *coldFile
	overflow *overflowFile
	hot      [numShards]*hotShard
	hotCount atomic.Int64

	mu      sync.RWMutex
	buckets map[string]time.Time

	mp *multipartRegistry

	// Epoch counter for safe concurrent demotion.
	epoch    atomic.Int64
	flushMu  sync.Mutex
	flushing atomic.Bool

	// Per-store stoppable ticker for cachedNano.
	stopTick chan struct{}

	// Context for stopping background goroutines (e.g. demote).
	ctx    context.Context
	cancel context.CancelFunc
}

var _ storage.Storage = (*store)(nil)

func (s *store) Bucket(name string) storage.Bucket {
	if name == "" {
		name = "default"
	}

	s.mu.Lock()
	if _, ok := s.buckets[name]; !ok {
		if len(s.buckets) < maxBuckets {
			s.buckets[name] = fastTime()
		}
	}
	s.mu.Unlock()

	return &bucket{st: s, name: name}
}

func (s *store) Buckets(_ context.Context, limit, offset int, _ storage.Options) (storage.BucketIter, error) {
	s.mu.RLock()
	names := make([]string, 0, len(s.buckets))
	for n := range s.buckets {
		names = append(names, n)
	}
	s.mu.RUnlock()

	sort.Strings(names)

	s.mu.RLock()
	infos := make([]*storage.BucketInfo, 0, len(names))
	for _, n := range names {
		infos = append(infos, &storage.BucketInfo{
			Name:      n,
			CreatedAt: s.buckets[n],
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

	return &bucketIter{list: infos}, nil
}

func (s *store) CreateBucket(_ context.Context, name string, _ storage.Options) (*storage.BucketInfo, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("falcon: bucket name is empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.buckets[name]; ok {
		return nil, storage.ErrExist
	}

	now := fastTime()
	s.buckets[name] = now

	return &storage.BucketInfo{
		Name:      name,
		CreatedAt: now,
	}, nil
}

func (s *store) DeleteBucket(_ context.Context, name string, opts storage.Options) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("falcon: bucket name is empty")
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

	if !force {
		// Check if any keys exist for this bucket in the hot tier.
		prefix := name + "\x00"
		for i := range numShards {
			sh := s.hot[i]
			sh.mu.RLock()
			for k := range sh.m {
				if strings.HasPrefix(k, prefix) {
					sh.mu.RUnlock()
					return storage.ErrPermission
				}
			}
			sh.mu.RUnlock()
		}
	}

	// If force, delete all keys for this bucket from hot tier.
	if force {
		prefix := name + "\x00"
		for i := range numShards {
			sh := s.hot[i]
			sh.mu.Lock()
			for k := range sh.m {
				if strings.HasPrefix(k, prefix) {
					delete(sh.m, k)
					s.hotCount.Add(-1)
				}
			}
			sh.mu.Unlock()
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
	// Stop background goroutines.
	s.cancel()
	close(s.stopTick)

	// Flush hot tier to cold.
	s.flushAll()

	if s.syncMode != "none" {
		s.cold.mu.Lock()
		s.cold.f.Sync()
		s.cold.mu.Unlock()
		s.overflow.mu.Lock()
		s.overflow.f.Sync()
		s.overflow.mu.Unlock()
	}

	err1 := s.cold.close()
	err2 := s.overflow.close()
	if err1 != nil {
		return err1
	}
	return err2
}

// hotPut writes an entry to the hot tier.
func (s *store) hotPut(ck string, e *hotEntry) {
	sh := s.hot[fnv1a32(ck)%numShards]
	sh.mu.Lock()
	_, existed := sh.m[ck]
	sh.m[ck] = e
	sh.mu.Unlock()

	if !existed {
		newCount := s.hotCount.Add(1)
		if int(newCount) > s.hotSize && s.flushing.CompareAndSwap(false, true) {
			go func() {
				select {
				case <-s.ctx.Done():
					s.flushing.Store(false)
					return
				default:
				}
				s.demote()
				s.flushing.Store(false)
			}()
		}
	}
}

// hotGet retrieves an entry from the hot tier.
func (s *store) hotGet(ck string) (*hotEntry, bool) {
	sh := s.hot[fnv1a32(ck)%numShards]
	sh.mu.RLock()
	e, ok := sh.m[ck]
	sh.mu.RUnlock()
	return e, ok
}

// hotDelete removes an entry from the hot tier.
func (s *store) hotDelete(ck string) bool {
	sh := s.hot[fnv1a32(ck)%numShards]
	sh.mu.Lock()
	_, ok := sh.m[ck]
	if ok {
		delete(sh.m, ck)
		s.hotCount.Add(-1)
	}
	sh.mu.Unlock()
	return ok
}

// coldGet retrieves an entry from the cold tier and promotes it to hot.
func (s *store) coldGet(ck string) (*hotEntry, bool) {
	hash := fnv1a64([]byte(ck))

	s.cold.mu.RLock()
	e, found, err := s.cold.get(ck, hash, s.overflow)
	s.cold.mu.RUnlock()

	if err != nil || !found {
		return nil, false
	}

	// Promote to hot tier.
	s.hotPut(ck, e)
	return e, true
}

// demote evicts the oldest ~25% of hot tier entries to cold tier.
func (s *store) demote() {
	s.flushMu.Lock()
	defer s.flushMu.Unlock()

	s.epoch.Add(1)

	// Collect all entries with timestamps.
	type kv struct {
		ck string
		e  *hotEntry
	}

	var all []kv
	for i := range numShards {
		sh := s.hot[i]
		sh.mu.RLock()
		for k, v := range sh.m {
			all = append(all, kv{ck: k, e: v})
		}
		sh.mu.RUnlock()
	}

	if len(all) <= s.hotSize/2 {
		return
	}

	// Sort by updated timestamp ascending (oldest first).
	sort.Slice(all, func(i, j int) bool {
		return all[i].e.updated < all[j].e.updated
	})

	// Evict oldest 25%.
	evictCount := len(all) / 4
	if evictCount < 1 {
		evictCount = 1
	}

	s.cold.mu.Lock()
	defer s.cold.mu.Unlock()

	for i := range evictCount {
		entry := all[i]
		hash := fnv1a64([]byte(entry.ck))

		if err := s.cold.put(entry.ck, hash, entry.e, s.overflow); err != nil {
			continue
		}

		// Remove from hot tier.
		sh := s.hot[fnv1a32(entry.ck)%numShards]
		sh.mu.Lock()
		// Verify the entry is still the same (not updated concurrently).
		if cur, ok := sh.m[entry.ck]; ok && cur.updated == entry.e.updated {
			delete(sh.m, entry.ck)
			s.hotCount.Add(-1)
		}
		sh.mu.Unlock()
	}

	if s.syncMode == "full" {
		s.cold.f.Sync()
		s.overflow.mu.Lock()
		s.overflow.f.Sync()
		s.overflow.mu.Unlock()
	}
}

// flushAll writes all hot tier entries to cold.
func (s *store) flushAll() {
	s.flushMu.Lock()
	defer s.flushMu.Unlock()

	s.cold.mu.Lock()
	defer s.cold.mu.Unlock()

	for i := range numShards {
		sh := s.hot[i]
		sh.mu.Lock()
		for ck, e := range sh.m {
			hash := fnv1a64([]byte(ck))
			s.cold.put(ck, hash, e, s.overflow)
		}
		sh.mu.Unlock()
	}
}

// listKeys returns all keys (from both tiers) matching bucket and prefix.
func (s *store) listKeys(bucketName, prefix string) []*storage.Object {
	fullPrefix := bucketName + "\x00" + prefix
	seen := make(map[string]struct{})
	var objs []*storage.Object

	// Hot tier.
	for i := range numShards {
		sh := s.hot[i]
		sh.mu.RLock()
		for ck, e := range sh.m {
			if !strings.HasPrefix(ck, bucketName+"\x00") {
				continue
			}
			key := ck[len(bucketName)+1:]
			if prefix != "" && !strings.HasPrefix(key, prefix) {
				continue
			}
			seen[ck] = struct{}{}
			objs = append(objs, &storage.Object{
				Bucket:      bucketName,
				Key:         key,
				Size:        e.size,
				ContentType: e.contentType,
				Created:     time.Unix(0, e.created),
				Updated:     time.Unix(0, e.updated),
			})
		}
		sh.mu.RUnlock()
	}

	// Cold tier.
	s.cold.mu.RLock()
	coldEntries := s.cold.allEntries(s.overflow)
	s.cold.mu.RUnlock()

	for _, ce := range coldEntries {
		if _, ok := seen[ce.ck]; ok {
			continue // Hot tier has newer version.
		}
		if !strings.HasPrefix(ce.ck, bucketName+"\x00") {
			continue
		}
		key := ce.ck[len(bucketName)+1:]
		if prefix != "" && !strings.HasPrefix(key, prefix) {
			continue
		}
		_ = fullPrefix
		objs = append(objs, &storage.Object{
			Bucket:      bucketName,
			Key:         key,
			Size:        ce.entry.size,
			ContentType: ce.entry.contentType,
			Created:     time.Unix(0, ce.entry.created),
			Updated:     time.Unix(0, ce.entry.updated),
		})
	}

	sort.Slice(objs, func(i, j int) bool { return objs[i].Key < objs[j].Key })
	return objs
}

// hasBucketKeys returns true if any key exists for the bucket.
func (s *store) hasBucketKeys(bucketName string) bool {
	prefix := bucketName + "\x00"
	for i := range numShards {
		sh := s.hot[i]
		sh.mu.RLock()
		for k := range sh.m {
			if strings.HasPrefix(k, prefix) {
				sh.mu.RUnlock()
				return true
			}
		}
		sh.mu.RUnlock()
	}
	return false
}

// ---------------------------------------------------------------------------
// Bucket (storage.Bucket + HasDirectories + HasMultipart)
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

func (b *bucket) Info(_ context.Context) (*storage.BucketInfo, error) {
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

func (b *bucket) Write(_ context.Context, key string, src io.Reader, size int64, contentType string, _ storage.Options) (*storage.Object, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("falcon: key is empty")
	}

	var data []byte
	if size >= 0 {
		data = make([]byte, size)
		if size > 0 {
			n, err := io.ReadFull(src, data)
			if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
				return nil, fmt.Errorf("falcon: read value: %w", err)
			}
			data = data[:n]
		}
	} else {
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, src); err != nil {
			return nil, fmt.Errorf("falcon: read value: %w", err)
		}
		data = buf.Bytes()
	}

	now := fastNow()
	ck := compositeKey(b.name, key)

	e := &hotEntry{
		value:       data,
		contentType: contentType,
		created:     now,
		updated:     now,
		size:        int64(len(data)),
	}

	// Check if updating an existing entry; preserve created time.
	if old, ok := b.st.hotGet(ck); ok {
		e.created = old.created
	}

	b.st.hotPut(ck, e)

	return &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        e.size,
		ContentType: contentType,
		Created:     time.Unix(0, e.created),
		Updated:     time.Unix(0, e.updated),
	}, nil
}

func (b *bucket) Open(_ context.Context, key string, offset, length int64, _ storage.Options) (io.ReadCloser, *storage.Object, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, nil, fmt.Errorf("falcon: key is empty")
	}

	ck := compositeKey(b.name, key)

	// Hot tier first.
	e, ok := b.st.hotGet(ck)
	if !ok {
		// Cold tier with promotion.
		e, ok = b.st.coldGet(ck)
		if !ok {
			return nil, nil, storage.ErrNotExist
		}
	}

	obj := &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        e.size,
		ContentType: e.contentType,
		Created:     time.Unix(0, e.created),
		Updated:     time.Unix(0, e.updated),
	}

	data := e.value
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

	// Copy the slice to avoid data races after demotion.
	slice := make([]byte, end-offset)
	copy(slice, data[offset:end])

	return &dataReader{data: slice}, obj, nil
}

func (b *bucket) Stat(_ context.Context, key string, _ storage.Options) (*storage.Object, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("falcon: key is empty")
	}

	// Check for directory stat.
	if strings.HasSuffix(key, "/") {
		results := b.st.listKeys(b.name, key)
		if len(results) == 0 {
			return nil, storage.ErrNotExist
		}
		return &storage.Object{
			Bucket:  b.name,
			Key:     strings.TrimSuffix(key, "/"),
			IsDir:   true,
			Created: results[0].Created,
			Updated: results[0].Updated,
		}, nil
	}

	ck := compositeKey(b.name, key)

	e, ok := b.st.hotGet(ck)
	if !ok {
		// Check cold tier (no promotion for stat).
		hash := fnv1a64([]byte(ck))
		b.st.cold.mu.RLock()
		e, ok, _ = b.st.cold.get(ck, hash, b.st.overflow)
		b.st.cold.mu.RUnlock()
		if !ok {
			return nil, storage.ErrNotExist
		}
	}

	return &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        e.size,
		ContentType: e.contentType,
		Created:     time.Unix(0, e.created),
		Updated:     time.Unix(0, e.updated),
	}, nil
}

func (b *bucket) Delete(_ context.Context, key string, _ storage.Options) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("falcon: key is empty")
	}

	ck := compositeKey(b.name, key)

	hotDeleted := b.st.hotDelete(ck)

	// Mark tombstone in cold tier.
	hash := fnv1a64([]byte(ck))
	b.st.cold.mu.Lock()
	coldErr := b.st.cold.markTombstone(ck, hash)
	b.st.cold.mu.Unlock()

	if !hotDeleted && coldErr != nil {
		return storage.ErrNotExist
	}

	return nil
}

func (b *bucket) Copy(_ context.Context, dstKey string, srcBucket, srcKey string, _ storage.Options) (*storage.Object, error) {
	dstKey = strings.TrimSpace(dstKey)
	srcKey = strings.TrimSpace(srcKey)
	if dstKey == "" || srcKey == "" {
		return nil, fmt.Errorf("falcon: key is empty")
	}
	if srcBucket == "" {
		srcBucket = b.name
	}

	srcCK := compositeKey(srcBucket, srcKey)

	e, ok := b.st.hotGet(srcCK)
	if !ok {
		e, ok = b.st.coldGet(srcCK)
		if !ok {
			return nil, storage.ErrNotExist
		}
	}

	now := fastNow()
	dstCK := compositeKey(b.name, dstKey)

	// Copy value bytes.
	valCopy := make([]byte, len(e.value))
	copy(valCopy, e.value)

	dst := &hotEntry{
		value:       valCopy,
		contentType: e.contentType,
		created:     now,
		updated:     now,
		size:        e.size,
	}

	b.st.hotPut(dstCK, dst)

	return &storage.Object{
		Bucket:      b.name,
		Key:         dstKey,
		Size:        dst.size,
		ContentType: dst.contentType,
		Created:     time.Unix(0, now),
		Updated:     time.Unix(0, now),
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

func (b *bucket) List(_ context.Context, prefix string, limit, offset int, opts storage.Options) (storage.ObjectIter, error) {
	prefix = strings.TrimSpace(prefix)
	recursive := true
	if opts != nil {
		if v, ok := opts["recursive"].(bool); ok {
			recursive = v
		}
	}

	results := b.st.listKeys(b.name, prefix)

	if !recursive {
		var filtered []*storage.Object
		for _, obj := range results {
			rest := strings.TrimPrefix(obj.Key, prefix)
			rest = strings.TrimPrefix(rest, "/")
			if strings.Contains(rest, "/") {
				continue
			}
			filtered = append(filtered, obj)
		}
		results = filtered
	}

	if offset < 0 {
		offset = 0
	}
	if offset > len(results) {
		offset = len(results)
	}
	results = results[offset:]
	if limit > 0 && limit < len(results) {
		results = results[:limit]
	}

	return &objectIter{list: results}, nil
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

func (d *dir) Info(_ context.Context) (*storage.Object, error) {
	prefix := d.path
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	results := d.b.st.listKeys(d.b.name, prefix)
	if len(results) == 0 {
		return nil, storage.ErrNotExist
	}
	return &storage.Object{
		Bucket:  d.b.name,
		Key:     d.path,
		IsDir:   true,
		Created: results[0].Created,
		Updated: results[0].Updated,
	}, nil
}

func (d *dir) List(_ context.Context, limit, offset int, _ storage.Options) (storage.ObjectIter, error) {
	prefix := d.path
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	results := d.b.st.listKeys(d.b.name, prefix)

	// Filter to direct children only.
	var objs []*storage.Object
	for _, r := range results {
		rest := strings.TrimPrefix(r.Key, prefix)
		if strings.Contains(rest, "/") {
			continue
		}
		objs = append(objs, r)
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

	return &objectIter{list: objs}, nil
}

func (d *dir) Delete(_ context.Context, opts storage.Options) error {
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

	results := d.b.st.listKeys(d.b.name, prefix)
	if len(results) == 0 {
		return storage.ErrNotExist
	}

	for _, r := range results {
		if !recursive {
			rest := strings.TrimPrefix(r.Key, prefix)
			if strings.Contains(rest, "/") {
				continue
			}
		}
		ck := compositeKey(d.b.name, r.Key)
		d.b.st.hotDelete(ck)

		hash := fnv1a64([]byte(ck))
		d.b.st.cold.mu.Lock()
		d.b.st.cold.markTombstone(ck, hash)
		d.b.st.cold.mu.Unlock()
	}
	return nil
}

func (d *dir) Move(_ context.Context, dstPath string, _ storage.Options) (storage.Directory, error) {
	srcPrefix := strings.Trim(d.path, "/")
	dstPrefix := strings.Trim(dstPath, "/")

	if srcPrefix != "" && !strings.HasSuffix(srcPrefix, "/") {
		srcPrefix += "/"
	}
	if dstPrefix != "" && !strings.HasSuffix(dstPrefix, "/") {
		dstPrefix += "/"
	}

	results := d.b.st.listKeys(d.b.name, srcPrefix)
	if len(results) == 0 {
		return nil, storage.ErrNotExist
	}

	for _, r := range results {
		rel := strings.TrimPrefix(r.Key, srcPrefix)
		newKey := dstPrefix + rel

		srcCK := compositeKey(d.b.name, r.Key)
		e, ok := d.b.st.hotGet(srcCK)
		if !ok {
			e, ok = d.b.st.coldGet(srcCK)
		}
		if !ok {
			continue
		}

		now := fastNow()
		dstCK := compositeKey(d.b.name, newKey)
		valCopy := make([]byte, len(e.value))
		copy(valCopy, e.value)
		d.b.st.hotPut(dstCK, &hotEntry{
			value:       valCopy,
			contentType: e.contentType,
			created:     e.created,
			updated:     now,
			size:        e.size,
		})

		// Delete old.
		d.b.st.hotDelete(srcCK)
		hash := fnv1a64([]byte(srcCK))
		d.b.st.cold.mu.Lock()
		d.b.st.cold.markTombstone(srcCK, hash)
		d.b.st.cold.mu.Unlock()
	}

	return &dir{b: d.b, path: strings.Trim(dstPath, "/")}, nil
}

// ---------------------------------------------------------------------------
// Multipart support
// ---------------------------------------------------------------------------

type multipartRegistry struct {
	mu      sync.RWMutex
	uploads map[string]*multipartUpload
	counter atomic.Int64
}

func newMultipartRegistry() *multipartRegistry {
	r := &multipartRegistry{
		uploads: make(map[string]*multipartUpload),
	}
	r.counter.Store(time.Now().UnixNano())
	return r
}

type multipartUpload struct {
	id          string
	bucket      string
	key         string
	contentType string
	parts       map[int]*partData
	metadata    map[string]string
	created     time.Time
}

type partData struct {
	number int
	data   []byte
	size   int64
	etag   string
}

func (b *bucket) InitMultipart(_ context.Context, key string, contentType string, opts storage.Options) (*storage.MultipartUpload, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("falcon: key is empty")
	}

	id := strconv.FormatInt(b.st.mp.counter.Add(1), 36)

	var metadata map[string]string
	if opts != nil {
		if m, ok := opts["metadata"].(map[string]string); ok {
			metadata = m
		}
	}

	upload := &multipartUpload{
		id:          id,
		bucket:      b.name,
		key:         key,
		contentType: contentType,
		parts:       make(map[int]*partData),
		metadata:    metadata,
		created:     fastTime(),
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

func (b *bucket) UploadPart(_ context.Context, mu *storage.MultipartUpload, number int, src io.Reader, size int64, _ storage.Options) (*storage.PartInfo, error) {
	if number < 1 || number > maxPartNumber {
		return nil, fmt.Errorf("falcon: part number %d out of range [1, %d]", number, maxPartNumber)
	}

	b.st.mp.mu.RLock()
	upload, ok := b.st.mp.uploads[mu.UploadID]
	b.st.mp.mu.RUnlock()
	if !ok {
		return nil, storage.ErrNotExist
	}

	var data []byte
	if size >= 0 {
		data = make([]byte, size)
		if size > 0 {
			n, err := io.ReadFull(src, data)
			if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
				return nil, fmt.Errorf("falcon: read part: %w", err)
			}
			data = data[:n]
		}
	} else {
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, src); err != nil {
			return nil, fmt.Errorf("falcon: read part: %w", err)
		}
		data = buf.Bytes()
	}

	hash := md5.Sum(data)
	etag := hex.EncodeToString(hash[:])

	b.st.mp.mu.Lock()
	upload.parts[number] = &partData{
		number: number,
		data:   data,
		size:   int64(len(data)),
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
	if number < 1 || number > maxPartNumber {
		return nil, fmt.Errorf("falcon: part number %d out of range", number)
	}

	b.st.mp.mu.RLock()
	_, ok := b.st.mp.uploads[mu.UploadID]
	b.st.mp.mu.RUnlock()
	if !ok {
		return nil, storage.ErrNotExist
	}

	srcBucket := mu.Bucket
	if sb, ok := opts["source_bucket"].(string); ok && sb != "" {
		srcBucket = sb
	}
	srcKey, _ := opts["source_key"].(string)
	if srcKey == "" {
		return nil, fmt.Errorf("falcon: source_key required for CopyPart")
	}
	srcOffset, _ := opts["source_offset"].(int64)
	srcLength, _ := opts["source_length"].(int64)

	// Open source object.
	srcCK := compositeKey(srcBucket, srcKey)
	e, found := b.st.hotGet(srcCK)
	if !found {
		e, found = b.st.coldGet(srcCK)
		if !found {
			return nil, storage.ErrNotExist
		}
	}

	data := e.value
	if srcOffset > 0 {
		if srcOffset >= int64(len(data)) {
			data = nil
		} else {
			data = data[srcOffset:]
		}
	}
	if srcLength > 0 && int64(len(data)) > srcLength {
		data = data[:srcLength]
	}

	return b.UploadPart(ctx, mu, number, bytes.NewReader(data), int64(len(data)), nil)
}

func (b *bucket) ListParts(_ context.Context, mu *storage.MultipartUpload, limit, offset int, _ storage.Options) ([]*storage.PartInfo, error) {
	b.st.mp.mu.RLock()
	upload, ok := b.st.mp.uploads[mu.UploadID]
	if !ok {
		b.st.mp.mu.RUnlock()
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
	b.st.mp.mu.RUnlock()

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

func (b *bucket) CompleteMultipart(_ context.Context, mu *storage.MultipartUpload, parts []*storage.PartInfo, _ storage.Options) (*storage.Object, error) {
	b.st.mp.mu.Lock()
	upload, ok := b.st.mp.uploads[mu.UploadID]
	if !ok {
		b.st.mp.mu.Unlock()
		return nil, storage.ErrNotExist
	}
	delete(b.st.mp.uploads, mu.UploadID)
	b.st.mp.mu.Unlock()

	// Sort parts by number.
	sort.Slice(parts, func(i, j int) bool {
		return parts[i].Number < parts[j].Number
	})

	// Verify all parts exist.
	for _, p := range parts {
		if _, ok := upload.parts[p.Number]; !ok {
			return nil, fmt.Errorf("falcon: part %d not found", p.Number)
		}
	}

	// Assemble final value.
	var totalSize int64
	for _, p := range parts {
		totalSize += upload.parts[p.Number].size
	}

	assembled := make([]byte, 0, totalSize)
	for _, p := range parts {
		assembled = append(assembled, upload.parts[p.Number].data...)
	}

	now := fastNow()
	ck := compositeKey(b.name, upload.key)

	e := &hotEntry{
		value:       assembled,
		contentType: upload.contentType,
		created:     now,
		updated:     now,
		size:        int64(len(assembled)),
	}

	b.st.hotPut(ck, e)

	return &storage.Object{
		Bucket:      b.name,
		Key:         upload.key,
		Size:        e.size,
		ContentType: upload.contentType,
		Created:     time.Unix(0, now),
		Updated:     time.Unix(0, now),
	}, nil
}

func (b *bucket) AbortMultipart(_ context.Context, mu *storage.MultipartUpload, _ storage.Options) error {
	b.st.mp.mu.Lock()
	defer b.st.mp.mu.Unlock()

	if _, ok := b.st.mp.uploads[mu.UploadID]; !ok {
		return storage.ErrNotExist
	}
	delete(b.st.mp.uploads, mu.UploadID)
	return nil
}

// ---------------------------------------------------------------------------
// Iterators
// ---------------------------------------------------------------------------

type bucketIter struct {
	list []*storage.BucketInfo
	pos  int
}

func (it *bucketIter) Next() (*storage.BucketInfo, error) {
	if it.pos >= len(it.list) {
		return nil, nil
	}
	b := it.list[it.pos]
	it.pos++
	return b, nil
}

func (it *bucketIter) Close() error {
	it.list = nil
	return nil
}

type objectIter struct {
	list []*storage.Object
	pos  int
}

func (it *objectIter) Next() (*storage.Object, error) {
	if it.pos >= len(it.list) {
		return nil, nil
	}
	o := it.list[it.pos]
	it.pos++
	return o, nil
}

func (it *objectIter) Close() error {
	it.list = nil
	return nil
}

// ---------------------------------------------------------------------------
// Data reader
// ---------------------------------------------------------------------------

type dataReader struct {
	data []byte
	pos  int
}

func (r *dataReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func (r *dataReader) WriteTo(w io.Writer) (int64, error) {
	if r.pos >= len(r.data) {
		return 0, nil
	}
	n, err := w.Write(r.data[r.pos:])
	r.pos = len(r.data)
	return int64(n), err
}

func (r *dataReader) Close() error { return nil }
