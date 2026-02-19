// Package bee implements a distributed, sharded, replicated cluster storage driver.
//
// Architecture:
// - Haystack-style append-only records on each node.
// - Rendezvous-hash shard placement across N nodes.
// - Replication factor R with quorum write/read semantics.
// - Background hinted-handoff/read-repair for eventual convergence.
//
// DSN examples:
//
//	bee:///tmp/bee-data?nodes=3&replicas=3&w=2&r=1&sync=none
//	bee:///tmp/bee-data?nodes=5&replicas=3&w=2&r=1&sync=batch&inline_kb=64
//	bee:///?peers=http://127.0.0.1:9401,http://127.0.0.1:9402,http://127.0.0.1:9403&replicas=3&w=2&r=1
package bee

import (
	"bytes"
	"container/list"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/liteio-dev/liteio/pkg/storage"
)

const (
	defaultNodes    = 3
	defaultReplicas = 3

	defaultRepairWorkers = 4
	defaultRepairMaxKB   = 8 * 1024
	repairQueueSize      = 4096

	defaultCacheObjectKB = 64
)

func init() {
	storage.Register("bee", &driver{})
}

type driver struct{}

func (d *driver) Open(ctx context.Context, dsn string) (storage.Storage, error) {
	_ = ctx

	u, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("bee: parse dsn: %w", err)
	}
	if u.Scheme != "bee" && u.Scheme != "" {
		return nil, fmt.Errorf("bee: unexpected scheme %q", u.Scheme)
	}

	root := filepath.Clean(u.Path)
	if root == "" || root == "." {
		root = "/tmp/bee-data"
	}

	q := u.Query()
	peers := parsePeers(q.Get("peers"))

	nodes := queryInt(q, "nodes", defaultNodes)
	if len(peers) > 0 {
		nodes = len(peers)
	}
	if nodes <= 0 {
		return nil, fmt.Errorf("bee: nodes must be > 0")
	}

	replicas := queryInt(q, "replicas", min(defaultReplicas, nodes))
	if replicas <= 0 {
		replicas = 1
	}
	if replicas > nodes {
		replicas = nodes
	}

	writeQuorum := queryInt(q, "w", replicas/2+1)
	if writeQuorum <= 0 {
		writeQuorum = 1
	}
	if writeQuorum > replicas {
		writeQuorum = replicas
	}

	readQuorum := queryInt(q, "r", 1)
	if readQuorum <= 0 {
		readQuorum = 1
	}
	if readQuorum > replicas {
		readQuorum = replicas
	}

	syncMode := strings.ToLower(strings.TrimSpace(q.Get("sync")))
	if syncMode == "" {
		syncMode = "none"
	}
	switch syncMode {
	case "none", "batch", "full":
	default:
		return nil, fmt.Errorf("bee: invalid sync mode %q", syncMode)
	}

	inlineKB := queryInt(q, "inline_kb", defaultInlineLimit/1024)
	if inlineKB <= 0 {
		inlineKB = defaultInlineLimit / 1024
	}
	inlineLimit := int64(inlineKB * 1024)

	mode := strings.ToLower(strings.TrimSpace(q.Get("mode")))
	turbo := mode == "turbo" || mode == "memory"
	repairDefault := true
	if turbo {
		repairDefault = false
	}
	repair := queryBool(q, "repair", repairDefault)
	workers := queryInt(q, "repair_workers", defaultRepairWorkers)
	if workers <= 0 {
		workers = defaultRepairWorkers
	}
	workers = min(workers, max(1, nodes*2))
	repairMaxKB := queryInt(q, "repair_max_kb", defaultRepairMaxKB)
	if repairMaxKB <= 0 {
		repairMaxKB = defaultRepairMaxKB
	}
	repairMaxBytes := int64(repairMaxKB) * 1024

	cacheMB := queryInt(q, "cache_mb", 0)
	cacheObjKB := queryInt(q, "cache_obj_kb", defaultCacheObjectKB)
	if cacheObjKB <= 0 {
		cacheObjKB = defaultCacheObjectKB
	}
	cacheTTLms := queryInt(q, "cache_ttl_ms", 1000)

	st := &store{
		root:        root,
		replicas:    replicas,
		writeQuorum: writeQuorum,
		readQuorum:  readQuorum,
		repair:      repair,
		repairMax:   repairMaxBytes,
		buckets:     make(map[string]time.Time, 16),
		repairCh:    make(chan repairTask, repairQueueSize),
		nodes:       make([]clusterNode, nodes),
		mp:          newMultipartRegistry(),
		turbo:       turbo,
	}
	if turbo {
		st.turboData = make(map[string]*turboEntry, 1<<16)
		st.turboBuckets = make(map[string]map[string]*turboEntry, 64)
		st.turboParents = make(map[string]map[string]map[string]*turboEntry, 64)
	}
	if cacheMB > 0 {
		st.cache = newReadCache(int64(cacheMB)*1024*1024, int64(cacheObjKB)*1024, time.Duration(cacheTTLms)*time.Millisecond)
	}

	if len(peers) > 0 {
		for i, endpoint := range peers {
			n, err := newRemoteNode(i, endpoint)
			if err != nil {
				st.closeNodes()
				return nil, err
			}
			st.nodes[i] = n
		}
	} else {
		for i := 0; i < nodes; i++ {
			nodePath := filepath.Join(root, fmt.Sprintf("node-%02d", i), "bee.log")
			n, err := openNodeEngine(i, nodePath, syncMode, inlineLimit)
			if err != nil {
				st.closeNodes()
				return nil, err
			}
			st.nodes[i] = n
		}
	}

	st.rebuildBuckets()
	st.startRepairWorkers(workers)

	return st, nil
}

type store struct {
	root        string
	nodes       []clusterNode
	replicas    int
	writeQuorum int
	readQuorum  int
	repair      bool
	repairMax   int64

	mu      sync.RWMutex
	buckets map[string]time.Time

	repairCh chan repairTask
	repairWg sync.WaitGroup

	closeOnce sync.Once

	cache *readCache
	mp    *multipartRegistry

	turbo   bool
	turboMu sync.RWMutex
	// key: bucket\\x00object-key
	turboData map[string]*turboEntry
	// bucket -> key -> entry
	turboBuckets map[string]map[string]*turboEntry
	// bucket -> parent-prefix -> key -> entry
	turboParents map[string]map[string]map[string]*turboEntry
}

var _ storage.Storage = (*store)(nil)

type repairTask struct {
	target      int
	source      int
	bucket      string
	key         string
	contentType string
	data        []byte
	timestamp   int64
	deleteOp    bool
	pull        bool
}

type readCache struct {
	maxBytes      int64
	maxObjectSize int64
	ttl           time.Duration

	mu    sync.Mutex
	bytes int64
	ll    *list.List // front = most recent
	items map[string]*cacheEntry
}

type cacheEntry struct {
	key      string
	data     []byte
	meta     nodeEntry
	size     int64
	expires  int64 // UnixNano, 0 means no TTL.
	listElem *list.Element
}

type turboEntry struct {
	bucket string
	key    string
	data   []byte
	meta   nodeEntry
}

func newReadCache(maxBytes, maxObjectSize int64, ttl time.Duration) *readCache {
	if maxBytes <= 0 || maxObjectSize <= 0 {
		return nil
	}
	return &readCache{
		maxBytes:      maxBytes,
		maxObjectSize: maxObjectSize,
		ttl:           ttl,
		ll:            list.New(),
		items:         make(map[string]*cacheEntry, 1024),
	}
}

func (c *readCache) get(key string) ([]byte, *nodeEntry, bool) {
	if c == nil {
		return nil, nil, false
	}
	now := time.Now().UnixNano()

	c.mu.Lock()
	defer c.mu.Unlock()

	e, ok := c.items[key]
	if !ok {
		return nil, nil, false
	}
	if e.expires > 0 && e.expires < now {
		c.removeLocked(e)
		return nil, nil, false
	}

	c.ll.MoveToFront(e.listElem)
	m := e.meta
	return e.data, &m, true
}

func (c *readCache) upsert(key string, data []byte, meta *nodeEntry) {
	if c == nil || meta == nil {
		return
	}
	if int64(len(data)) > c.maxObjectSize {
		return
	}

	exp := int64(0)
	if c.ttl > 0 {
		exp = time.Now().Add(c.ttl).UnixNano()
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if existing, ok := c.items[key]; ok {
		c.bytes -= existing.size
		existing.data = data
		existing.meta = *meta
		existing.size = int64(len(data))
		existing.expires = exp
		c.bytes += existing.size
		c.ll.MoveToFront(existing.listElem)
	} else {
		e := &cacheEntry{
			key:     key,
			data:    data,
			meta:    *meta,
			size:    int64(len(data)),
			expires: exp,
		}
		e.listElem = c.ll.PushFront(e)
		c.items[key] = e
		c.bytes += e.size
	}

	for c.bytes > c.maxBytes && c.ll.Len() > 0 {
		back := c.ll.Back()
		if back == nil {
			break
		}
		ev := back.Value.(*cacheEntry)
		c.removeLocked(ev)
	}
}

func (c *readCache) delete(key string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.items[key]; ok {
		c.removeLocked(e)
	}
}

func (c *readCache) removeLocked(e *cacheEntry) {
	delete(c.items, e.key)
	c.ll.Remove(e.listElem)
	c.bytes -= e.size
}

func (s *store) turboPut(bucket, key, contentType string, data []byte, ts int64) *nodeEntry {
	if !s.turbo {
		return nil
	}
	ck := compositeKey(bucket, key)
	s.turboMu.Lock()
	defer s.turboMu.Unlock()

	created := ts
	oldParent := ""
	if old, ok := s.turboData[ck]; ok {
		created = old.meta.created
		oldParent = parentPrefix(old.key)
	}
	e := &turboEntry{
		bucket: bucket,
		key:    key,
		data:   data,
		meta: nodeEntry{
			size:        int64(len(data)),
			contentType: contentType,
			created:     created,
			updated:     ts,
		},
	}
	s.turboData[ck] = e
	kb := s.turboBuckets[bucket]
	if kb == nil {
		kb = make(map[string]*turboEntry, 256)
		s.turboBuckets[bucket] = kb
	}
	kb[key] = e
	parent := parentPrefix(key)
	pb := s.turboParents[bucket]
	if pb == nil {
		pb = make(map[string]map[string]*turboEntry, 128)
		s.turboParents[bucket] = pb
	}
	kp := pb[parent]
	if kp == nil {
		kp = make(map[string]*turboEntry, 64)
		pb[parent] = kp
	}
	kp[key] = e
	if oldParent != "" && oldParent != parent {
		if oldSet := pb[oldParent]; oldSet != nil {
			delete(oldSet, key)
			if len(oldSet) == 0 {
				delete(pb, oldParent)
			}
		}
	}
	return cloneEntry(&e.meta)
}

func (s *store) turboGet(bucket, key string) ([]byte, *nodeEntry, bool) {
	if !s.turbo {
		return nil, nil, false
	}
	ck := compositeKey(bucket, key)
	s.turboMu.RLock()
	e, ok := s.turboData[ck]
	s.turboMu.RUnlock()
	if !ok {
		return nil, nil, false
	}
	return e.data, cloneEntry(&e.meta), true
}

func (s *store) turboDelete(bucket, key string, ts int64) bool {
	if !s.turbo {
		return false
	}
	ck := compositeKey(bucket, key)
	s.turboMu.Lock()
	_, ok := s.turboData[ck]
	if ok {
		delete(s.turboData, ck)
		if kb := s.turboBuckets[bucket]; kb != nil {
			delete(kb, key)
			if len(kb) == 0 {
				delete(s.turboBuckets, bucket)
			}
		}
		if pb := s.turboParents[bucket]; pb != nil {
			parent := parentPrefix(key)
			if kp := pb[parent]; kp != nil {
				delete(kp, key)
				if len(kp) == 0 {
					delete(pb, parent)
				}
			}
			if len(pb) == 0 {
				delete(s.turboParents, bucket)
			}
		}
	}
	s.turboMu.Unlock()
	_ = ts
	return ok
}

func (s *store) turboList(bucket, prefix string, recursive bool) []nodeListItem {
	if !s.turbo {
		return nil
	}
	s.turboMu.RLock()
	kb := s.turboBuckets[bucket]
	if len(kb) == 0 {
		s.turboMu.RUnlock()
		return nil
	}
	src := kb
	if prefix != "" {
		if pb := s.turboParents[bucket]; len(pb) > 0 {
			if kp := pb[prefix]; len(kp) > 0 {
				src = kp
			}
		}
	}
	out := make([]nodeListItem, 0, len(src))
	for _, e := range src {
		if prefix != "" && !strings.HasPrefix(e.key, prefix) {
			continue
		}
		if !recursive {
			rest := strings.TrimPrefix(e.key, prefix)
			rest = strings.TrimPrefix(rest, "/")
			if strings.Contains(rest, "/") {
				continue
			}
		}
		out = append(out, nodeListItem{key: e.key, entry: cloneEntry(&e.meta)})
	}
	s.turboMu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].key < out[j].key })
	return out
}

type clusterNode interface {
	write(bucket, key, contentType string, data []byte, ts int64) (*nodeEntry, error)
	read(bucket, key string) ([]byte, *nodeEntry, error)
	stat(bucket, key string) (*nodeEntry, error)
	delete(bucket, key string, ts int64) (bool, error)
	hasBucket(bucket string) bool
	bucketNames() []string
	list(bucket, prefix string, recursive bool) []nodeListItem
	deleteBucket(bucket string, ts int64)
	close() error
}

func (s *store) startRepairWorkers(n int) {
	if !s.repair {
		return
	}
	for i := 0; i < n; i++ {
		s.repairWg.Add(1)
		go func() {
			defer s.repairWg.Done()
			for task := range s.repairCh {
				if task.target < 0 || task.target >= len(s.nodes) {
					continue
				}
				n := s.nodes[task.target]
				for attempt := 0; attempt < 3; attempt++ {
					var err error
					if task.deleteOp {
						_, err = n.delete(task.bucket, task.key, task.timestamp)
						if errors.Is(err, storage.ErrNotExist) {
							err = nil
						}
					} else {
						if task.pull {
							if task.source < 0 || task.source >= len(s.nodes) || task.source == task.target {
								break
							}
							data, meta, readErr := s.nodes[task.source].read(task.bucket, task.key)
							if readErr != nil {
								if errors.Is(readErr, storage.ErrNotExist) {
									err = nil
									break
								}
								err = readErr
							} else {
								contentType := task.contentType
								ts := task.timestamp
								if meta != nil {
									if contentType == "" {
										contentType = meta.contentType
									}
									if ts <= 0 {
										ts = meta.updated
									}
								}
								_, err = n.write(task.bucket, task.key, contentType, data, ts)
							}
						} else {
							_, err = n.write(task.bucket, task.key, task.contentType, task.data, task.timestamp)
						}
					}
					if err == nil {
						break
					}
					time.Sleep(time.Duration(attempt+1) * 10 * time.Millisecond)
				}
			}
		}()
	}
}

func (s *store) enqueueRepair(task repairTask) {
	if !s.repair {
		return
	}
	select {
	case s.repairCh <- task:
	default:
		// Drop when saturated to preserve foreground latency.
	}
}

func (s *store) rebuildBuckets() {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, n := range s.nodes {
		for _, name := range n.bucketNames() {
			if _, ok := s.buckets[name]; !ok {
				s.buckets[name] = now
			}
		}
	}
}

func (s *store) Bucket(name string) storage.Bucket {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "default"
	}

	s.mu.Lock()
	if _, ok := s.buckets[name]; !ok {
		s.buckets[name] = time.Now()
	}
	s.mu.Unlock()

	return &bucket{st: s, name: name}
}

func (s *store) Buckets(ctx context.Context, limit, offset int, opts storage.Options) (storage.BucketIter, error) {
	_ = opts
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

	infos := make([]*storage.BucketInfo, 0, len(names))
	s.mu.RLock()
	for _, name := range names {
		infos = append(infos, &storage.BucketInfo{Name: name, CreatedAt: s.buckets[name]})
	}
	s.mu.RUnlock()

	infos = sliceBucketInfos(infos, limit, offset)
	return &bucketIter{items: infos}, nil
}

func (s *store) CreateBucket(ctx context.Context, name string, opts storage.Options) (*storage.BucketInfo, error) {
	_ = opts
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("bee: bucket name is empty")
	}

	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.buckets[name]; ok {
		return nil, storage.ErrExist
	}
	s.buckets[name] = now

	return &storage.BucketInfo{Name: name, CreatedAt: now}, nil
}

func (s *store) DeleteBucket(ctx context.Context, name string, opts storage.Options) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("bee: bucket name is empty")
	}

	force := false
	if opts != nil {
		if v, ok := opts["force"].(bool); ok {
			force = v
		}
	}

	s.mu.Lock()
	_, exists := s.buckets[name]
	s.mu.Unlock()
	if !exists {
		return storage.ErrNotExist
	}

	if !force {
		for _, n := range s.nodes {
			if n.hasBucket(name) {
				return storage.ErrPermission
			}
		}
	}

	ts := time.Now().UnixNano()
	for _, n := range s.nodes {
		n.deleteBucket(name, ts)
	}

	s.mu.Lock()
	delete(s.buckets, name)
	s.mu.Unlock()
	return nil
}

func (s *store) Features() storage.Features {
	return storage.Features{
		"move":             true,
		"server_side_move": true,
		"server_side_copy": true,
		"directories":      true,
	}
}

func (s *store) Close() error {
	var err error
	s.closeOnce.Do(func() {
		if s.repair {
			close(s.repairCh)
			s.repairWg.Wait()
		}
		err = s.closeNodes()
	})
	return err
}

func (s *store) closeNodes() error {
	var firstErr error
	for _, n := range s.nodes {
		if n == nil {
			continue
		}
		if err := n.close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

type scoredNode struct {
	idx   int
	score uint64
}

func (s *store) replicaSet(bucket, key string) []int {
	id := compositeKey(bucket, key)
	scores := make([]scoredNode, 0, len(s.nodes))
	for i := range s.nodes {
		scores = append(scores, scoredNode{idx: i, score: nodeScore(id, uint64(i+1))})
	}
	sort.Slice(scores, func(i, j int) bool {
		if scores[i].score == scores[j].score {
			return scores[i].idx < scores[j].idx
		}
		return scores[i].score > scores[j].score
	})

	k := s.replicas
	if k > len(scores) {
		k = len(scores)
	}
	reps := make([]int, 0, k)
	for i := 0; i < k; i++ {
		reps = append(reps, scores[i].idx)
	}
	return reps
}

func nodeScore(id string, seed uint64) uint64 {
	h := seed ^ 1469598103934665603
	for i := 0; i < len(id); i++ {
		h ^= uint64(id[i])
		h *= 1099511628211
	}
	// Final avalanche.
	h ^= h >> 33
	h *= 0xff51afd7ed558ccd
	h ^= h >> 33
	h *= 0xc4ceb9fe1a85ec53
	h ^= h >> 33
	return h
}

// bucket implements storage.Bucket.
type bucket struct {
	st   *store
	name string
}

var (
	_ storage.Bucket         = (*bucket)(nil)
	_ storage.HasDirectories = (*bucket)(nil)
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
	return &storage.BucketInfo{Name: b.name, CreatedAt: created}, nil
}

func (b *bucket) Write(ctx context.Context, key string, src io.Reader, size int64, contentType string, opts storage.Options) (*storage.Object, error) {
	_ = opts
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("bee: key is empty")
	}

	data, err := readAllSized(src, size)
	if err != nil {
		return nil, err
	}
	now := time.Now().UnixNano()
	replicas := b.st.replicaSet(b.name, key)
	if b.st.turbo {
		best := b.st.turboPut(b.name, key, contentType, data, now)
		if best == nil {
			best = &nodeEntry{size: int64(len(data)), contentType: contentType, created: now, updated: now}
		}
		if b.st.repair && int64(len(data)) <= b.st.repairMax {
			for _, idx := range replicas {
				b.st.enqueueRepair(repairTask{
					target:      idx,
					bucket:      b.name,
					key:         key,
					contentType: contentType,
					data:        data,
					timestamp:   now,
				})
			}
		}
		return &storage.Object{
			Bucket:      b.name,
			Key:         key,
			Size:        int64(len(data)),
			ContentType: best.contentType,
			Created:     time.Unix(0, best.created),
			Updated:     time.Unix(0, best.updated),
		}, nil
	}

	type writeResult struct {
		node  int
		entry *nodeEntry
		err   error
	}

	quorum := b.st.writeQuorum
	if quorum <= 0 {
		quorum = 1
	}
	if quorum > len(replicas) {
		quorum = len(replicas)
	}
	syncTargets := replicas[:quorum]
	asyncTargets := replicas[quorum:]

	success := 0
	var best *nodeEntry
	bestNode := -1
	var firstErr error
	acked := make(map[int]struct{}, len(replicas))

	// Fast path: issue the minimum number of foreground writes needed for quorum.
	// Remaining replicas are repaired asynchronously on success.
	if len(syncTargets) == 1 {
		idx := syncTargets[0]
		e, err := b.st.nodes[idx].write(b.name, key, contentType, data, now)
		if err != nil {
			firstErr = err
		} else {
			success++
			best = e
			bestNode = idx
			acked[idx] = struct{}{}
		}
	} else {
		results := make(chan writeResult, len(syncTargets))
		for _, idx := range syncTargets {
			n := b.st.nodes[idx]
			go func(nodeID int, eng clusterNode) {
				e, err := eng.write(b.name, key, contentType, data, now)
				results <- writeResult{node: nodeID, entry: e, err: err}
			}(idx, n)
		}
		for i := 0; i < len(syncTargets); i++ {
			res := <-results
			if res.err != nil {
				if firstErr == nil {
					firstErr = res.err
				}
				continue
			}
			success++
			acked[res.node] = struct{}{}
			if best == nil || res.entry.updated > best.updated {
				best = res.entry
				bestNode = res.node
			}
		}
	}

	// Quorum fallback: only if minimal foreground set failed.
	if success < b.st.writeQuorum {
		for _, idx := range asyncTargets {
			e, err := b.st.nodes[idx].write(b.name, key, contentType, data, now)
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			success++
			acked[idx] = struct{}{}
			if best == nil || e.updated > best.updated {
				best = e
				bestNode = idx
			}
			if success >= b.st.writeQuorum {
				break
			}
		}
	}

	if success < b.st.writeQuorum {
		if firstErr == nil {
			firstErr = fmt.Errorf("bee: write quorum not met")
		}
		return nil, fmt.Errorf("bee: write quorum not met (%d/%d): %w", success, b.st.writeQuorum, firstErr)
	}

	for _, idx := range replicas {
		if _, ok := acked[idx]; ok {
			continue
		}

		task := repairTask{
			target:      idx,
			bucket:      b.name,
			key:         key,
			contentType: contentType,
			timestamp:   now,
		}
		if int64(len(data)) <= b.st.repairMax || bestNode < 0 || bestNode == idx {
			task.data = data
		} else {
			task.pull = true
			task.source = bestNode
		}
		b.st.enqueueRepair(task)
	}

	if best == nil {
		best = &nodeEntry{size: int64(len(data)), contentType: contentType, created: now, updated: now}
	}
	b.st.cache.upsert(compositeKey(b.name, key), data, best)

	return &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        int64(len(data)),
		ContentType: best.contentType,
		Created:     time.Unix(0, best.created),
		Updated:     time.Unix(0, best.updated),
	}, nil
}

func (b *bucket) Open(ctx context.Context, key string, offset, length int64, opts storage.Options) (io.ReadCloser, *storage.Object, error) {
	_ = opts
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}

	key = strings.TrimSpace(key)
	if key == "" {
		return nil, nil, fmt.Errorf("bee: key is empty")
	}

	replicas := b.st.replicaSet(b.name, key)
	cacheKey := compositeKey(b.name, key)
	if data, meta, ok := b.st.turboGet(b.name, key); ok {
		slice := applyRange(data, offset, length)
		obj := &storage.Object{
			Bucket:      b.name,
			Key:         key,
			Size:        meta.size,
			ContentType: meta.contentType,
			Created:     time.Unix(0, meta.created),
			Updated:     time.Unix(0, meta.updated),
		}
		return io.NopCloser(bytes.NewReader(slice)), obj, nil
	}

	// Gateway-local read cache to bypass network on hot objects.
	if cachedData, cachedMeta, ok := b.st.cache.get(cacheKey); ok {
		slice := applyRange(cachedData, offset, length)
		obj := &storage.Object{
			Bucket:      b.name,
			Key:         key,
			Size:        cachedMeta.size,
			ContentType: cachedMeta.contentType,
			Created:     time.Unix(0, cachedMeta.created),
			Updated:     time.Unix(0, cachedMeta.updated),
		}
		return io.NopCloser(bytes.NewReader(slice)), obj, nil
	}

	// Fast path for eventual-consistent reads (r=1): try primary first, then fall back.
	// This avoids waiting on all replicas in the hot path.
	if b.st.readQuorum <= 1 {
		var firstErr error
		missing := make([]int, 0, len(replicas))
		for _, idx := range replicas {
			data, meta, err := b.st.nodes[idx].read(b.name, key)
			if err != nil {
				if errors.Is(err, storage.ErrNotExist) {
					missing = append(missing, idx)
					continue
				}
				if firstErr == nil {
					firstErr = err
				}
				continue
			}

			if len(missing) > 0 && int64(len(data)) <= b.st.repairMax {
				for _, nodeID := range missing {
					b.st.enqueueRepair(repairTask{
						target:      nodeID,
						bucket:      b.name,
						key:         key,
						contentType: meta.contentType,
						data:        data,
						timestamp:   meta.updated,
					})
				}
			}
			b.st.cache.upsert(cacheKey, data, meta)

			slice := applyRange(data, offset, length)
			obj := &storage.Object{
				Bucket:      b.name,
				Key:         key,
				Size:        meta.size,
				ContentType: meta.contentType,
				Created:     time.Unix(0, meta.created),
				Updated:     time.Unix(0, meta.updated),
			}
			return io.NopCloser(bytes.NewReader(slice)), obj, nil
		}

		if len(missing) == len(replicas) {
			return nil, nil, storage.ErrNotExist
		}
		if firstErr == nil {
			firstErr = storage.ErrNotExist
		}
		return nil, nil, firstErr
	}

	type readResult struct {
		node int
		data []byte
		meta *nodeEntry
		err  error
	}

	results := make(chan readResult, len(replicas))
	for _, idx := range replicas {
		n := b.st.nodes[idx]
		go func(nodeID int, eng clusterNode) {
			data, meta, err := eng.read(b.name, key)
			results <- readResult{node: nodeID, data: data, meta: meta, err: err}
		}(idx, n)
	}

	successes := make([]readResult, 0, len(replicas))
	missing := make([]int, 0, len(replicas))
	var firstErr error
	for i := 0; i < len(replicas); i++ {
		res := <-results
		if res.err != nil {
			if errors.Is(res.err, storage.ErrNotExist) {
				missing = append(missing, res.node)
			} else if firstErr == nil {
				firstErr = res.err
			}
			continue
		}
		successes = append(successes, res)
	}

	if len(successes) == 0 {
		if len(missing) == len(replicas) {
			return nil, nil, storage.ErrNotExist
		}
		if firstErr == nil {
			firstErr = storage.ErrNotExist
		}
		return nil, nil, firstErr
	}
	if len(successes) < b.st.readQuorum {
		return nil, nil, fmt.Errorf("bee: read quorum not met (%d/%d)", len(successes), b.st.readQuorum)
	}

	best := successes[0]
	for i := 1; i < len(successes); i++ {
		if successes[i].meta.updated > best.meta.updated {
			best = successes[i]
		}
	}

	if int64(len(best.data)) <= b.st.repairMax {
		for _, res := range successes {
			if res.node == best.node {
				continue
			}
			if res.meta.updated < best.meta.updated {
				b.st.enqueueRepair(repairTask{
					target:      res.node,
					bucket:      b.name,
					key:         key,
					contentType: best.meta.contentType,
					data:        best.data,
					timestamp:   best.meta.updated,
				})
			}
		}
		for _, nodeID := range missing {
			b.st.enqueueRepair(repairTask{
				target:      nodeID,
				bucket:      b.name,
				key:         key,
				contentType: best.meta.contentType,
				data:        best.data,
				timestamp:   best.meta.updated,
			})
		}
	}
	b.st.cache.upsert(cacheKey, best.data, best.meta)

	slice := applyRange(best.data, offset, length)
	obj := &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        best.meta.size,
		ContentType: best.meta.contentType,
		Created:     time.Unix(0, best.meta.created),
		Updated:     time.Unix(0, best.meta.updated),
	}
	return io.NopCloser(bytes.NewReader(slice)), obj, nil
}

func (b *bucket) Stat(ctx context.Context, key string, opts storage.Options) (*storage.Object, error) {
	_ = opts
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("bee: key is empty")
	}
	if _, meta, ok := b.st.turboGet(b.name, key); ok {
		return &storage.Object{
			Bucket:      b.name,
			Key:         key,
			Size:        meta.size,
			ContentType: meta.contentType,
			Created:     time.Unix(0, meta.created),
			Updated:     time.Unix(0, meta.updated),
		}, nil
	}
	cacheKey := compositeKey(b.name, key)
	if _, cachedMeta, ok := b.st.cache.get(cacheKey); ok {
		return &storage.Object{
			Bucket:      b.name,
			Key:         key,
			Size:        cachedMeta.size,
			ContentType: cachedMeta.contentType,
			Created:     time.Unix(0, cachedMeta.created),
			Updated:     time.Unix(0, cachedMeta.updated),
		}, nil
	}

	replicas := b.st.replicaSet(b.name, key)

	// Fast path for r=1: stat primary first, then fallback replicas.
	if b.st.readQuorum <= 1 {
		var firstErr error
		missing := 0
		for _, idx := range replicas {
			meta, err := b.st.nodes[idx].stat(b.name, key)
			if err != nil {
				if errors.Is(err, storage.ErrNotExist) {
					missing++
					continue
				}
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			return &storage.Object{
				Bucket:      b.name,
				Key:         key,
				Size:        meta.size,
				ContentType: meta.contentType,
				Created:     time.Unix(0, meta.created),
				Updated:     time.Unix(0, meta.updated),
			}, nil
		}

		if missing == len(replicas) {
			return nil, storage.ErrNotExist
		}
		if firstErr == nil {
			firstErr = fmt.Errorf("bee: stat failed for %s/%s", b.name, key)
		}
		return nil, firstErr
	}

	type statResult struct {
		node int
		meta *nodeEntry
		err  error
	}

	results := make(chan statResult, len(replicas))
	for _, idx := range replicas {
		n := b.st.nodes[idx]
		go func(nodeID int, eng clusterNode) {
			meta, err := eng.stat(b.name, key)
			results <- statResult{node: nodeID, meta: meta, err: err}
		}(idx, n)
	}

	success := 0
	var best *nodeEntry
	missing := 0
	for i := 0; i < len(replicas); i++ {
		res := <-results
		if res.err != nil {
			if errors.Is(res.err, storage.ErrNotExist) {
				missing++
			}
			continue
		}
		success++
		if best == nil || res.meta.updated > best.updated {
			best = res.meta
		}
	}

	if success == 0 {
		if missing == len(replicas) {
			return nil, storage.ErrNotExist
		}
		return nil, fmt.Errorf("bee: stat failed for %s/%s", b.name, key)
	}
	if success < b.st.readQuorum {
		return nil, fmt.Errorf("bee: read quorum not met for stat (%d/%d)", success, b.st.readQuorum)
	}

	return &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        best.size,
		ContentType: best.contentType,
		Created:     time.Unix(0, best.created),
		Updated:     time.Unix(0, best.updated),
	}, nil
}

func (b *bucket) Delete(ctx context.Context, key string, opts storage.Options) error {
	_ = opts
	if err := ctx.Err(); err != nil {
		return err
	}

	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("bee: key is empty")
	}

	replicas := b.st.replicaSet(b.name, key)
	now := time.Now().UnixNano()
	if b.st.turbo {
		existed := b.st.turboDelete(b.name, key, now)
		if b.st.repair {
			for _, idx := range replicas {
				b.st.enqueueRepair(repairTask{
					target:    idx,
					bucket:    b.name,
					key:       key,
					timestamp: now,
					deleteOp:  true,
				})
			}
		}
		if !existed {
			return storage.ErrNotExist
		}
		return nil
	}
	type delResult struct {
		node    int
		existed bool
		err     error
	}

	quorum := b.st.writeQuorum
	if quorum <= 0 {
		quorum = 1
	}
	if quorum > len(replicas) {
		quorum = len(replicas)
	}
	syncTargets := replicas[:quorum]
	asyncTargets := replicas[quorum:]

	success := 0
	existedAny := false
	var firstErr error
	acked := make(map[int]struct{}, len(replicas))

	ackDelete := func(nodeID int, existed bool, err error) {
		if err != nil {
			if errors.Is(err, storage.ErrNotExist) {
				success++
				acked[nodeID] = struct{}{}
				return
			}
			if firstErr == nil {
				firstErr = err
			}
			return
		}
		success++
		acked[nodeID] = struct{}{}
		existedAny = existedAny || existed
	}

	if len(syncTargets) == 1 {
		idx := syncTargets[0]
		existed, err := b.st.nodes[idx].delete(b.name, key, now)
		ackDelete(idx, existed, err)
	} else {
		results := make(chan delResult, len(syncTargets))
		for _, idx := range syncTargets {
			n := b.st.nodes[idx]
			go func(nodeID int, eng clusterNode) {
				existed, err := eng.delete(b.name, key, now)
				results <- delResult{node: nodeID, existed: existed, err: err}
			}(idx, n)
		}
		for i := 0; i < len(syncTargets); i++ {
			res := <-results
			ackDelete(res.node, res.existed, res.err)
		}
	}

	if success < b.st.writeQuorum {
		for _, idx := range asyncTargets {
			existed, err := b.st.nodes[idx].delete(b.name, key, now)
			ackDelete(idx, existed, err)
			if success >= b.st.writeQuorum {
				break
			}
		}
	}
	if success < b.st.writeQuorum {
		if firstErr != nil {
			return fmt.Errorf("bee: delete quorum not met (%d/%d): %w", success, b.st.writeQuorum, firstErr)
		}
		return fmt.Errorf("bee: delete quorum not met (%d/%d)", success, b.st.writeQuorum)
	}

	for _, idx := range replicas {
		if _, ok := acked[idx]; ok {
			continue
		}
		b.st.enqueueRepair(repairTask{
			target:    idx,
			bucket:    b.name,
			key:       key,
			timestamp: now,
			deleteOp:  true,
		})
	}
	b.st.cache.delete(compositeKey(b.name, key))

	// If we received explicit non-existence from all replicas, surface ErrNotExist.
	// Otherwise keep delete idempotent and avoid false negatives.
	if !existedAny && len(acked) == len(replicas) {
		return storage.ErrNotExist
	}
	return nil
}

func (b *bucket) Copy(ctx context.Context, dstKey string, srcBucket, srcKey string, opts storage.Options) (*storage.Object, error) {
	_ = opts
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	dstKey = strings.TrimSpace(dstKey)
	srcKey = strings.TrimSpace(srcKey)
	if dstKey == "" || srcKey == "" {
		return nil, fmt.Errorf("bee: key is empty")
	}
	if srcBucket == "" {
		srcBucket = b.name
	}

	sb := b.st.Bucket(srcBucket)
	rc, srcObj, err := sb.Open(ctx, srcKey, 0, 0, nil)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, err
	}

	return b.Write(ctx, dstKey, bytes.NewReader(data), int64(len(data)), srcObj.ContentType, nil)
}

func (b *bucket) Move(ctx context.Context, dstKey string, srcBucket, srcKey string, opts storage.Options) (*storage.Object, error) {
	obj, err := b.Copy(ctx, dstKey, srcBucket, srcKey, opts)
	if err != nil {
		return nil, err
	}
	if srcBucket == "" {
		srcBucket = b.name
	}
	if err := b.st.Bucket(srcBucket).Delete(ctx, srcKey, nil); err != nil {
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
	if b.st.turbo {
		items := b.st.turboList(b.name, prefix, recursive)
		objs := make([]*storage.Object, 0, len(items))
		for _, item := range items {
			objs = append(objs, &storage.Object{
				Bucket:      b.name,
				Key:         item.key,
				Size:        item.entry.size,
				ContentType: item.entry.contentType,
				Created:     time.Unix(0, item.entry.created),
				Updated:     time.Unix(0, item.entry.updated),
			})
		}
		objs = sliceObjects(objs, limit, offset)
		return &objectIter{items: objs}, nil
	}

	type listResult struct {
		items []nodeListItem
	}

	results := make(chan listResult, len(b.st.nodes))
	for _, n := range b.st.nodes {
		go func(eng clusterNode) {
			results <- listResult{items: eng.list(b.name, prefix, recursive)}
		}(n)
	}

	dedup := make(map[string]*storage.Object, 256)
	for i := 0; i < len(b.st.nodes); i++ {
		res := <-results
		for _, item := range res.items {
			obj := &storage.Object{
				Bucket:      b.name,
				Key:         item.key,
				Size:        item.entry.size,
				ContentType: item.entry.contentType,
				Created:     time.Unix(0, item.entry.created),
				Updated:     time.Unix(0, item.entry.updated),
			}
			prev, ok := dedup[item.key]
			if !ok || obj.Updated.After(prev.Updated) {
				dedup[item.key] = obj
			}
		}
	}

	keys := make([]string, 0, len(dedup))
	for k := range dedup {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	objs := make([]*storage.Object, 0, len(keys))
	for _, k := range keys {
		objs = append(objs, dedup[k])
	}

	objs = sliceObjects(objs, limit, offset)
	return &objectIter{items: objs}, nil
}

func (b *bucket) SignedURL(ctx context.Context, key string, method string, expires time.Duration, opts storage.Options) (string, error) {
	_ = ctx
	_ = key
	_ = method
	_ = expires
	_ = opts
	return "", storage.ErrUnsupported
}

func (b *bucket) Directory(path string) storage.Directory {
	return &dir{b: b, path: strings.Trim(path, "/")}
}

// Directory support.
type dir struct {
	b    *bucket
	path string
}

var _ storage.Directory = (*dir)(nil)

func (d *dir) Bucket() storage.Bucket { return d.b }

func (d *dir) Path() string { return d.path }

func (d *dir) Info(ctx context.Context) (*storage.Object, error) {
	prefix := d.path
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	it, err := d.b.List(ctx, prefix, 1, 0, storage.Options{"recursive": true})
	if err != nil {
		return nil, err
	}
	defer it.Close()
	obj, err := it.Next()
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, storage.ErrNotExist
	}
	return &storage.Object{
		Bucket:  d.b.name,
		Key:     d.path,
		IsDir:   true,
		Created: obj.Created,
		Updated: obj.Updated,
	}, nil
}

func (d *dir) List(ctx context.Context, limit, offset int, opts storage.Options) (storage.ObjectIter, error) {
	prefix := d.path
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	if opts == nil {
		opts = storage.Options{}
	}
	opts["recursive"] = false
	return d.b.List(ctx, prefix, limit, offset, opts)
}

func (d *dir) Delete(ctx context.Context, opts storage.Options) error {
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

	it, err := d.b.List(ctx, prefix, 0, 0, storage.Options{"recursive": recursive})
	if err != nil {
		return err
	}
	defer it.Close()

	any := false
	for {
		obj, err := it.Next()
		if err != nil {
			return err
		}
		if obj == nil {
			break
		}
		any = true
		if err := d.b.Delete(ctx, obj.Key, nil); err != nil && !errors.Is(err, storage.ErrNotExist) {
			return err
		}
	}
	if !any {
		return storage.ErrNotExist
	}
	return nil
}

func (d *dir) Move(ctx context.Context, dstPath string, opts storage.Options) (storage.Directory, error) {
	_ = opts
	srcPrefix := strings.Trim(d.path, "/")
	dstPrefix := strings.Trim(dstPath, "/")
	if srcPrefix != "" {
		srcPrefix += "/"
	}
	if dstPrefix != "" {
		dstPrefix += "/"
	}

	it, err := d.b.List(ctx, srcPrefix, 0, 0, storage.Options{"recursive": true})
	if err != nil {
		return nil, err
	}
	defer it.Close()

	moved := false
	for {
		obj, err := it.Next()
		if err != nil {
			return nil, err
		}
		if obj == nil {
			break
		}
		moved = true

		rel := strings.TrimPrefix(obj.Key, srcPrefix)
		dstKey := dstPrefix + rel
		if _, err := d.b.Move(ctx, dstKey, d.b.name, obj.Key, nil); err != nil {
			return nil, err
		}
	}
	if !moved {
		return nil, storage.ErrNotExist
	}
	return &dir{b: d.b, path: strings.Trim(dstPath, "/")}, nil
}

// Iterators.
type bucketIter struct {
	items []*storage.BucketInfo
	idx   int
}

func (it *bucketIter) Next() (*storage.BucketInfo, error) {
	if it.idx >= len(it.items) {
		return nil, nil
	}
	v := it.items[it.idx]
	it.idx++
	return v, nil
}

func (it *bucketIter) Close() error {
	it.items = nil
	return nil
}

type objectIter struct {
	items []*storage.Object
	idx   int
}

func (it *objectIter) Next() (*storage.Object, error) {
	if it.idx >= len(it.items) {
		return nil, nil
	}
	v := it.items[it.idx]
	it.idx++
	return v, nil
}

func (it *objectIter) Close() error {
	it.items = nil
	return nil
}

func readAllSized(src io.Reader, size int64) ([]byte, error) {
	if size >= 0 {
		buf := make([]byte, size)
		n, err := io.ReadFull(src, buf)
		if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
			return nil, fmt.Errorf("bee: read value: %w", err)
		}
		return buf[:n], nil
	}

	data, err := io.ReadAll(src)
	if err != nil {
		return nil, fmt.Errorf("bee: read value: %w", err)
	}
	return data, nil
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

func queryInt(q url.Values, key string, def int) int {
	v := strings.TrimSpace(q.Get(key))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func queryBool(q url.Values, key string, def bool) bool {
	v := strings.TrimSpace(strings.ToLower(q.Get(key)))
	if v == "" {
		return def
	}
	if v == "1" || v == "true" || v == "yes" || v == "on" {
		return true
	}
	if v == "0" || v == "false" || v == "no" || v == "off" {
		return false
	}
	return def
}

func parsePeers(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

func parentPrefix(key string) string {
	i := strings.LastIndexByte(key, '/')
	if i <= 0 {
		return ""
	}
	return key[:i]
}

func sliceBucketInfos(infos []*storage.BucketInfo, limit, offset int) []*storage.BucketInfo {
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
	return infos
}

func sliceObjects(objs []*storage.Object, limit, offset int) []*storage.Object {
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
	return objs
}
