package herd

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/liteio-dev/liteio/pkg/storage"
)

// Wire protocol constants.
const (
	protoMagic = 0x4844 // "HD"

	opPut    byte = 1
	opGet    byte = 2
	opStat   byte = 3
	opDelete byte = 4
	opList   byte = 5
	opPing   byte = 6

	statusOK       byte = 0
	statusNotFound byte = 1
	statusError    byte = 2
)

// ---------------------------------------------------------------------------
// Embedded multi-node store (nodes=N): N independent stores in one process.
// Zero TCP overhead — pure function calls with rendezvous hashing.
// ---------------------------------------------------------------------------

// openMultiNode creates N independent herd stores in one process.
// Each node gets its own directory: {root}/node_{i}/ with full stripes.
// Routing uses the same rendezvous hashing as TCP cluster mode.
func openMultiNode(ctx context.Context, u *url.URL) (*multiNodeStore, error) {
	q := u.Query()
	root := filepath.Clean(u.Path)
	if root == "" || root == "." {
		root = "/tmp/herd-data"
	}

	numNodes := intParam(q, "nodes", 3)
	if numNodes < 1 {
		numNodes = 1
	}

	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("herd: mkdir %q: %w", root, err)
	}

	// Build per-node DSN (strip the nodes param, keep everything else).
	nodeQ := make(url.Values)
	for k, v := range q {
		if k != "nodes" {
			nodeQ[k] = v
		}
	}

	nodes := make([]*store, 0, numNodes)
	nodeNames := make([]string, 0, numNodes)
	for i := 0; i < numNodes; i++ {
		nodeDir := filepath.Join(root, fmt.Sprintf("node_%d", i))
		nodeURL := &url.URL{Path: nodeDir, RawQuery: nodeQ.Encode()}

		st, err := openEmbedded(ctx, nodeURL)
		if err != nil {
			// Close already-opened nodes.
			for _, n := range nodes {
				n.Close()
			}
			return nil, fmt.Errorf("herd: open node %d: %w", i, err)
		}
		nodes = append(nodes, st)
		nodeNames = append(nodeNames, fmt.Sprintf("node_%d", i))
	}

	return &multiNodeStore{
		root:      root,
		nodes:     nodes,
		nodeNames: nodeNames,
		buckets:   make(map[string]time.Time),
		mp:        newMultipartRegistry(),
	}, nil
}

// multiNodeStore wraps N independent herd stores with rendezvous-hash routing.
// All operations are direct function calls — zero serialization, zero TCP.
type multiNodeStore struct {
	root      string
	nodes     []*store
	nodeNames []string // "node_0", "node_1", ... for rendezvous hashing

	mu      sync.RWMutex
	buckets map[string]time.Time

	mp *multipartRegistry
}

var _ storage.Storage = (*multiNodeStore)(nil)

// nodeFor picks the primary node for a key using rendezvous hashing.
// Same algorithm as clusterStore.nodeFor for consistency.
func (ms *multiNodeStore) nodeFor(bucket, key string) *store {
	if len(ms.nodes) == 1 {
		return ms.nodes[0]
	}
	var bestIdx int
	var bestScore uint64
	ck := bucket + "\x00" + key
	for i, name := range ms.nodeNames {
		score := rendezvousScore(name, ck)
		if score > bestScore {
			bestScore = score
			bestIdx = i
		}
	}
	return ms.nodes[bestIdx]
}

func (ms *multiNodeStore) Bucket(name string) storage.Bucket {
	if name == "" {
		name = "default"
	}
	ms.mu.Lock()
	if _, ok := ms.buckets[name]; !ok {
		ms.buckets[name] = fastNowTime()
	}
	ms.mu.Unlock()
	return &multiNodeBucket{ms: ms, name: name}
}

func (ms *multiNodeStore) Buckets(_ context.Context, limit, offset int, _ storage.Options) (storage.BucketIter, error) {
	ms.mu.RLock()
	names := make([]string, 0, len(ms.buckets))
	for n := range ms.buckets {
		names = append(names, n)
	}
	ms.mu.RUnlock()
	sort.Strings(names)

	ms.mu.RLock()
	infos := make([]*storage.BucketInfo, 0, len(names))
	for _, n := range names {
		infos = append(infos, &storage.BucketInfo{Name: n, CreatedAt: ms.buckets[n]})
	}
	ms.mu.RUnlock()

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

func (ms *multiNodeStore) CreateBucket(_ context.Context, name string, _ storage.Options) (*storage.BucketInfo, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("herd: bucket name is empty")
	}
	ms.mu.Lock()
	defer ms.mu.Unlock()
	if _, ok := ms.buckets[name]; ok {
		return nil, storage.ErrExist
	}
	now := fastNowTime()
	ms.buckets[name] = now
	return &storage.BucketInfo{Name: name, CreatedAt: now}, nil
}

func (ms *multiNodeStore) DeleteBucket(_ context.Context, name string, _ storage.Options) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("herd: bucket name is empty")
	}
	ms.mu.Lock()
	defer ms.mu.Unlock()
	if _, ok := ms.buckets[name]; !ok {
		return storage.ErrNotExist
	}
	delete(ms.buckets, name)
	return nil
}

func (ms *multiNodeStore) Features() storage.Features {
	return storage.Features{
		"move":             true,
		"server_side_move": true,
		"server_side_copy": true,
		"directories":      true,
		"multipart":        true,
	}
}

func (ms *multiNodeStore) Close() error {
	for _, n := range ms.nodes {
		n.Close()
	}
	return nil
}

// multiNodeBucket routes operations to the correct node via rendezvous hashing.
// All operations are direct function calls to the underlying store's bucket.
type multiNodeBucket struct {
	ms   *multiNodeStore
	name string
}

var _ storage.Bucket = (*multiNodeBucket)(nil)

func (b *multiNodeBucket) Name() string { return b.name }
func (b *multiNodeBucket) Features() storage.Features {
	return b.ms.Features()
}

func (b *multiNodeBucket) Info(_ context.Context) (*storage.BucketInfo, error) {
	b.ms.mu.RLock()
	created, ok := b.ms.buckets[b.name]
	b.ms.mu.RUnlock()
	if !ok {
		return nil, storage.ErrNotExist
	}
	return &storage.BucketInfo{Name: b.name, CreatedAt: created}, nil
}

func (b *multiNodeBucket) Write(ctx context.Context, key string, src io.Reader, size int64, contentType string, opts storage.Options) (*storage.Object, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("herd: key is empty")
	}
	node := b.ms.nodeFor(b.name, key)
	return node.Bucket(b.name).Write(ctx, key, src, size, contentType, opts)
}

func (b *multiNodeBucket) Open(ctx context.Context, key string, offset, length int64, opts storage.Options) (io.ReadCloser, *storage.Object, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, nil, fmt.Errorf("herd: key is empty")
	}
	node := b.ms.nodeFor(b.name, key)
	return node.Bucket(b.name).Open(ctx, key, offset, length, opts)
}

func (b *multiNodeBucket) Stat(ctx context.Context, key string, opts storage.Options) (*storage.Object, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("herd: key is empty")
	}
	node := b.ms.nodeFor(b.name, key)
	return node.Bucket(b.name).Stat(ctx, key, opts)
}

func (b *multiNodeBucket) Delete(ctx context.Context, key string, opts storage.Options) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("herd: key is empty")
	}
	node := b.ms.nodeFor(b.name, key)
	return node.Bucket(b.name).Delete(ctx, key, opts)
}

func (b *multiNodeBucket) Copy(ctx context.Context, dstKey string, srcBucket, srcKey string, opts storage.Options) (*storage.Object, error) {
	// Both src and dst route to their respective nodes.
	// For cross-node copy, read from src node, write to dst node.
	srcKey = strings.TrimSpace(srcKey)
	dstKey = strings.TrimSpace(dstKey)
	if srcKey == "" || dstKey == "" {
		return nil, fmt.Errorf("herd: key is empty")
	}
	if srcBucket == "" {
		srcBucket = b.name
	}

	srcNode := b.ms.nodeFor(srcBucket, srcKey)
	dstNode := b.ms.nodeFor(b.name, dstKey)

	// Same node: use the store's native copy (zero-copy inline).
	if srcNode == dstNode {
		return dstNode.Bucket(b.name).Copy(ctx, dstKey, srcBucket, srcKey, opts)
	}

	// Cross-node: read from src, write to dst.
	rc, obj, err := srcNode.Bucket(srcBucket).Open(ctx, srcKey, 0, 0, nil)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return dstNode.Bucket(b.name).Write(ctx, dstKey, rc, obj.Size, obj.ContentType, opts)
}

func (b *multiNodeBucket) Move(ctx context.Context, dstKey string, srcBucket, srcKey string, opts storage.Options) (*storage.Object, error) {
	obj, err := b.Copy(ctx, dstKey, srcBucket, srcKey, opts)
	if err != nil {
		return nil, err
	}
	if srcBucket == "" {
		srcBucket = b.name
	}
	srcNode := b.ms.nodeFor(srcBucket, srcKey)
	if err := srcNode.Bucket(srcBucket).Delete(ctx, srcKey, nil); err != nil {
		return nil, err
	}
	return obj, nil
}

func (b *multiNodeBucket) List(_ context.Context, prefix string, limit, offset int, opts storage.Options) (storage.ObjectIter, error) {
	prefix = strings.TrimSpace(prefix)
	recursive := true
	if opts != nil {
		if v, ok := opts["recursive"].(bool); ok {
			recursive = v
		}
	}

	// Fan out list to all nodes and merge.
	var all []*storage.Object
	for _, node := range b.ms.nodes {
		bkt := node.Bucket(b.name).(*bucket)
		results := bkt.listAll(prefix)
		for _, r := range results {
			if !recursive {
				rest := strings.TrimPrefix(r.key, prefix)
				rest = strings.TrimPrefix(rest, "/")
				if strings.Contains(rest, "/") {
					continue
				}
			}
			all = append(all, &storage.Object{
				Bucket: b.name, Key: r.key, Size: r.entry.size,
				ContentType: r.entry.contentType,
				Created:     time.Unix(0, r.entry.created),
				Updated:     time.Unix(0, r.entry.updated),
			})
		}
	}

	sort.Slice(all, func(i, j int) bool { return all[i].Key < all[j].Key })

	// Dedup by key.
	if len(all) > 1 {
		deduped := all[:1]
		for i := 1; i < len(all); i++ {
			if all[i].Key != all[i-1].Key {
				deduped = append(deduped, all[i])
			}
		}
		all = deduped
	}

	if offset < 0 {
		offset = 0
	}
	if offset > len(all) {
		offset = len(all)
	}
	all = all[offset:]
	if limit > 0 && limit < len(all) {
		all = all[:limit]
	}
	return &objectIter{objects: all}, nil
}

func (b *multiNodeBucket) SignedURL(_ context.Context, _ string, _ string, _ time.Duration, _ storage.Options) (string, error) {
	return "", storage.ErrUnsupported
}

// openGossipCluster creates a cluster store with dynamic membership via HashiCorp memberlist.
// Nodes discover each other via seed addresses and the SWIM gossip protocol.
// The client routing table auto-updates when nodes join or leave.
func openGossipCluster(_ context.Context, u *url.URL) (*clusterStore, error) {
	q := u.Query()
	seeds := strings.Split(q.Get("seeds"), ",")
	if len(seeds) == 0 {
		return nil, fmt.Errorf("herd: no seeds specified")
	}

	replicas := intParam(q, "replicas", 1)
	gossipPort := intParam(q, "gossip_port", 7241)

	cs := &clusterStore{
		replicas: replicas,
		buckets:  make(map[string]time.Time),
		mp:       newMultipartRegistry(),
		nodeMap:  make(map[string]*remoteNode),
	}

	// Start gossip membership.
	membership, err := NewMembership(GossipConfig{
		BindPort: gossipPort,
		Seeds:    seeds,
		OnJoin: func(name string, meta NodeMeta) {
			if meta.DataAddr == "" || meta.Status != "ready" {
				return
			}
			cs.addNode(name, meta.DataAddr)
		},
		OnLeave: func(name string) {
			cs.removeNode(name)
		},
	})
	if err != nil {
		return nil, fmt.Errorf("herd: gossip: %w", err)
	}
	cs.membership = membership

	return cs, nil
}

// openCluster creates a cluster store that routes to remote herd nodes.
func openCluster(_ context.Context, u *url.URL) (*clusterStore, error) {
	q := u.Query()
	peers := strings.Split(q.Get("peers"), ",")
	if len(peers) == 0 {
		return nil, fmt.Errorf("herd: no peers specified")
	}

	replicas := intParam(q, "replicas", 1)
	if replicas > len(peers) {
		replicas = len(peers)
	}

	nodes := make([]*remoteNode, 0, len(peers))
	for _, addr := range peers {
		addr = strings.TrimSpace(addr)
		if addr == "" {
			continue
		}
		rn, err := newRemoteNode(addr)
		if err != nil {
			for _, n := range nodes {
				n.close()
			}
			return nil, fmt.Errorf("herd: connect to %s: %w", addr, err)
		}
		nodes = append(nodes, rn)
	}

	return &clusterStore{
		nodes:    nodes,
		replicas: replicas,
		buckets:  make(map[string]time.Time),
		mp:       newMultipartRegistry(),
	}, nil
}

// clusterStore routes operations to remote herd nodes via rendezvous hashing.
type clusterStore struct {
	nodes    []*remoteNode
	replicas int

	mu      sync.RWMutex
	buckets map[string]time.Time

	mp *multipartRegistry

	// Dynamic membership (gossip mode only).
	membership *Membership
	nodeMap    map[string]*remoteNode // node name → remote node
	nodeMu     sync.Mutex            // protects nodeMap and nodes during add/remove
}

var _ storage.Storage = (*clusterStore)(nil)

// nodeFor picks the primary node for a key using rendezvous hashing.
func (cs *clusterStore) nodeFor(bucket, key string) *remoteNode {
	if len(cs.nodes) == 1 {
		return cs.nodes[0]
	}
	var best *remoteNode
	var bestScore uint64
	ck := bucket + "\x00" + key
	for _, n := range cs.nodes {
		score := rendezvousScore(n.addr, ck)
		if score > bestScore {
			bestScore = score
			best = n
		}
	}
	return best
}

// rendezvousScore computes FNV-1a hash of node + key for rendezvous hashing.
func rendezvousScore(node, key string) uint64 {
	const offset64 = 14695981039346656037
	const prime64 = 1099511628211
	h := uint64(offset64)
	for i := 0; i < len(node); i++ {
		h ^= uint64(node[i])
		h *= prime64
	}
	h ^= 0xFF // separator
	h *= prime64
	for i := 0; i < len(key); i++ {
		h ^= uint64(key[i])
		h *= prime64
	}
	return h
}

func (cs *clusterStore) Bucket(name string) storage.Bucket {
	if name == "" {
		name = "default"
	}
	cs.mu.Lock()
	if _, ok := cs.buckets[name]; !ok {
		cs.buckets[name] = fastNowTime()
	}
	cs.mu.Unlock()
	return &clusterBucket{cs: cs, name: name}
}

func (cs *clusterStore) Buckets(_ context.Context, limit, offset int, _ storage.Options) (storage.BucketIter, error) {
	cs.mu.RLock()
	names := make([]string, 0, len(cs.buckets))
	for n := range cs.buckets {
		names = append(names, n)
	}
	cs.mu.RUnlock()
	sort.Strings(names)

	cs.mu.RLock()
	infos := make([]*storage.BucketInfo, 0, len(names))
	for _, n := range names {
		infos = append(infos, &storage.BucketInfo{Name: n, CreatedAt: cs.buckets[n]})
	}
	cs.mu.RUnlock()

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

func (cs *clusterStore) CreateBucket(_ context.Context, name string, _ storage.Options) (*storage.BucketInfo, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("herd: bucket name is empty")
	}
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if _, ok := cs.buckets[name]; ok {
		return nil, storage.ErrExist
	}
	now := fastNowTime()
	cs.buckets[name] = now
	return &storage.BucketInfo{Name: name, CreatedAt: now}, nil
}

func (cs *clusterStore) DeleteBucket(_ context.Context, name string, _ storage.Options) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("herd: bucket name is empty")
	}
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if _, ok := cs.buckets[name]; !ok {
		return storage.ErrNotExist
	}
	delete(cs.buckets, name)
	return nil
}

func (cs *clusterStore) Features() storage.Features {
	return storage.Features{
		"move":             true,
		"server_side_move": true,
		"server_side_copy": true,
		"directories":      true,
		"multipart":        true,
	}
}

// addNode dynamically adds a remote node (called by gossip OnJoin).
func (cs *clusterStore) addNode(name, addr string) {
	cs.nodeMu.Lock()
	defer cs.nodeMu.Unlock()

	// Already have this node.
	if _, ok := cs.nodeMap[name]; ok {
		return
	}

	rn, err := newRemoteNode(addr)
	if err != nil {
		return // Silently skip unreachable nodes; gossip will retry.
	}

	if cs.nodeMap == nil {
		cs.nodeMap = make(map[string]*remoteNode)
	}
	cs.nodeMap[name] = rn
	cs.nodes = append(cs.nodes, rn)
}

// removeNode dynamically removes a remote node (called by gossip OnLeave).
func (cs *clusterStore) removeNode(name string) {
	cs.nodeMu.Lock()
	defer cs.nodeMu.Unlock()

	rn, ok := cs.nodeMap[name]
	if !ok {
		return
	}
	delete(cs.nodeMap, name)

	// Remove from nodes slice.
	for i, n := range cs.nodes {
		if n == rn {
			cs.nodes = append(cs.nodes[:i], cs.nodes[i+1:]...)
			break
		}
	}
	rn.close()
}

func (cs *clusterStore) Close() error {
	if cs.membership != nil {
		cs.membership.Leave(5 * time.Second)
		cs.membership.Shutdown()
	}
	for _, n := range cs.nodes {
		n.close()
	}
	return nil
}

// clusterBucket routes operations to the appropriate node.
type clusterBucket struct {
	cs   *clusterStore
	name string
}

var _ storage.Bucket = (*clusterBucket)(nil)

func (b *clusterBucket) Name() string { return b.name }
func (b *clusterBucket) Features() storage.Features {
	return b.cs.Features()
}

func (b *clusterBucket) Info(_ context.Context) (*storage.BucketInfo, error) {
	b.cs.mu.RLock()
	created, ok := b.cs.buckets[b.name]
	b.cs.mu.RUnlock()
	if !ok {
		return nil, storage.ErrNotExist
	}
	return &storage.BucketInfo{Name: b.name, CreatedAt: created}, nil
}

func (b *clusterBucket) Write(_ context.Context, key string, src io.Reader, size int64, contentType string, _ storage.Options) (*storage.Object, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("herd: key is empty")
	}
	node := b.cs.nodeFor(b.name, key)
	return node.put(b.name, key, contentType, src, size)
}

func (b *clusterBucket) Open(_ context.Context, key string, offset, length int64, _ storage.Options) (io.ReadCloser, *storage.Object, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, nil, fmt.Errorf("herd: key is empty")
	}
	node := b.cs.nodeFor(b.name, key)
	return node.get(b.name, key, offset, length)
}

func (b *clusterBucket) Stat(_ context.Context, key string, _ storage.Options) (*storage.Object, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("herd: key is empty")
	}
	node := b.cs.nodeFor(b.name, key)
	return node.stat(b.name, key)
}

func (b *clusterBucket) Delete(_ context.Context, key string, _ storage.Options) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("herd: key is empty")
	}
	node := b.cs.nodeFor(b.name, key)
	return node.del(b.name, key)
}

func (b *clusterBucket) Copy(_ context.Context, _ string, _, _ string, _ storage.Options) (*storage.Object, error) {
	return nil, storage.ErrUnsupported
}

func (b *clusterBucket) Move(_ context.Context, _ string, _, _ string, _ storage.Options) (*storage.Object, error) {
	return nil, storage.ErrUnsupported
}

func (b *clusterBucket) List(_ context.Context, prefix string, limit, offset int, _ storage.Options) (storage.ObjectIter, error) {
	// Fan out list to all nodes and merge.
	var all []*storage.Object
	for _, node := range b.cs.nodes {
		objs, err := node.list(b.name, prefix, true)
		if err != nil {
			continue
		}
		all = append(all, objs...)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Key < all[j].Key })

	// Dedup by key (same key from multiple nodes).
	if len(all) > 1 {
		deduped := all[:1]
		for i := 1; i < len(all); i++ {
			if all[i].Key != all[i-1].Key {
				deduped = append(deduped, all[i])
			}
		}
		all = deduped
	}

	if offset < 0 {
		offset = 0
	}
	if offset > len(all) {
		offset = len(all)
	}
	all = all[offset:]
	if limit > 0 && limit < len(all) {
		all = all[:limit]
	}
	return &objectIter{objects: all}, nil
}

func (b *clusterBucket) SignedURL(_ context.Context, _ string, _ string, _ time.Duration, _ storage.Options) (string, error) {
	return "", storage.ErrUnsupported
}

// remoteNode is a TCP client to a single herd node.
type remoteNode struct {
	addr string
	pool chan net.Conn
}

func newRemoteNode(addr string) (*remoteNode, error) {
	rn := &remoteNode{
		addr: addr,
		pool: make(chan net.Conn, 64),
	}
	// Test connection.
	conn, err := rn.dial()
	if err != nil {
		return nil, err
	}
	rn.putConn(conn)
	return rn, nil
}

func (rn *remoteNode) dial() (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", rn.addr, 5*time.Second)
	if err != nil {
		return nil, err
	}
	if tc, ok := conn.(*net.TCPConn); ok {
		tc.SetNoDelay(true)
		tc.SetKeepAlive(true)
	}
	return conn, nil
}

func (rn *remoteNode) getConn() (net.Conn, error) {
	select {
	case c := <-rn.pool:
		return c, nil
	default:
		return rn.dial()
	}
}

func (rn *remoteNode) putConn(c net.Conn) {
	select {
	case rn.pool <- c:
	default:
		c.Close()
	}
}

func (rn *remoteNode) close() {
	close(rn.pool)
	for c := range rn.pool {
		c.Close()
	}
}

// Binary protocol helpers.

func writeRequest(w *bufio.Writer, op byte, body []byte) error {
	var hdr [7]byte
	binary.BigEndian.PutUint16(hdr[0:2], protoMagic)
	hdr[2] = op
	binary.BigEndian.PutUint32(hdr[3:7], uint32(len(body)))
	if _, err := w.Write(hdr[:]); err != nil {
		return err
	}
	if len(body) > 0 {
		if _, err := w.Write(body); err != nil {
			return err
		}
	}
	return w.Flush()
}

func readResponse(r *bufio.Reader) (byte, []byte, error) {
	var hdr [5]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return 0, nil, err
	}
	status := hdr[0]
	bodyLen := binary.BigEndian.Uint32(hdr[1:5])
	var body []byte
	if bodyLen > 0 {
		body = make([]byte, bodyLen)
		if _, err := io.ReadFull(r, body); err != nil {
			return 0, nil, err
		}
	}
	return status, body, nil
}

func encodePutBody(bucket, key, contentType string, data []byte, ts int64) []byte {
	bl, kl, cl := len(bucket), len(key), len(contentType)
	size := 2 + bl + 2 + kl + 2 + cl + 8 + 8 + len(data)
	buf := make([]byte, size)
	pos := 0
	binary.BigEndian.PutUint16(buf[pos:], uint16(bl))
	pos += 2
	copy(buf[pos:], bucket)
	pos += bl
	binary.BigEndian.PutUint16(buf[pos:], uint16(kl))
	pos += 2
	copy(buf[pos:], key)
	pos += kl
	binary.BigEndian.PutUint16(buf[pos:], uint16(cl))
	pos += 2
	copy(buf[pos:], contentType)
	pos += cl
	binary.BigEndian.PutUint64(buf[pos:], uint64(ts))
	pos += 8
	binary.BigEndian.PutUint64(buf[pos:], uint64(len(data)))
	pos += 8
	copy(buf[pos:], data)
	return buf
}

func encodeKeyBody(bucket, key string) []byte {
	bl, kl := len(bucket), len(key)
	buf := make([]byte, 2+bl+2+kl)
	pos := 0
	binary.BigEndian.PutUint16(buf[pos:], uint16(bl))
	pos += 2
	copy(buf[pos:], bucket)
	pos += bl
	binary.BigEndian.PutUint16(buf[pos:], uint16(kl))
	pos += 2
	copy(buf[pos:], key)
	return buf
}

func encodeGetBody(bucket, key string, offset, length int64) []byte {
	bl, kl := len(bucket), len(key)
	buf := make([]byte, 2+bl+2+kl+8+8)
	pos := 0
	binary.BigEndian.PutUint16(buf[pos:], uint16(bl))
	pos += 2
	copy(buf[pos:], bucket)
	pos += bl
	binary.BigEndian.PutUint16(buf[pos:], uint16(kl))
	pos += 2
	copy(buf[pos:], key)
	pos += kl
	binary.BigEndian.PutUint64(buf[pos:], uint64(offset))
	pos += 8
	binary.BigEndian.PutUint64(buf[pos:], uint64(length))
	return buf
}

func encodeListBody(bucket, prefix string, recursive bool) []byte {
	bl, pl := len(bucket), len(prefix)
	buf := make([]byte, 2+bl+2+pl+1)
	pos := 0
	binary.BigEndian.PutUint16(buf[pos:], uint16(bl))
	pos += 2
	copy(buf[pos:], bucket)
	pos += bl
	binary.BigEndian.PutUint16(buf[pos:], uint16(pl))
	pos += 2
	copy(buf[pos:], prefix)
	pos += pl
	if recursive {
		buf[pos] = 1
	}
	return buf
}

// Remote node operations.

func (rn *remoteNode) put(bucket, key, contentType string, src io.Reader, size int64) (*storage.Object, error) {
	var data []byte
	if size >= 0 {
		data = make([]byte, size)
		if size > 0 {
			if _, err := io.ReadFull(src, data); err != nil {
				if err != io.EOF && err != io.ErrUnexpectedEOF {
					return nil, err
				}
			}
		}
	} else {
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, src); err != nil {
			return nil, err
		}
		data = buf.Bytes()
		size = int64(len(data))
	}

	now := fastNow()
	body := encodePutBody(bucket, key, contentType, data, now)

	conn, err := rn.getConn()
	if err != nil {
		return nil, err
	}

	w := bufio.NewWriterSize(conn, 65536)
	r := bufio.NewReaderSize(conn, 65536)

	if err := writeRequest(w, opPut, body); err != nil {
		conn.Close()
		return nil, err
	}

	status, _, err := readResponse(r)
	if err != nil {
		conn.Close()
		return nil, err
	}
	rn.putConn(conn)

	if status != statusOK {
		return nil, fmt.Errorf("herd: remote put failed (status %d)", status)
	}

	return &storage.Object{
		Bucket:      bucket,
		Key:         key,
		Size:        size,
		ContentType: contentType,
		Created:     time.Unix(0, now),
		Updated:     time.Unix(0, now),
	}, nil
}

func (rn *remoteNode) get(bucket, key string, offset, length int64) (io.ReadCloser, *storage.Object, error) {
	body := encodeGetBody(bucket, key, offset, length)

	conn, err := rn.getConn()
	if err != nil {
		return nil, nil, err
	}

	w := bufio.NewWriterSize(conn, 4096)
	r := bufio.NewReaderSize(conn, 65536)

	if err := writeRequest(w, opGet, body); err != nil {
		conn.Close()
		return nil, nil, err
	}

	status, respBody, err := readResponse(r)
	if err != nil {
		conn.Close()
		return nil, nil, err
	}
	rn.putConn(conn)

	if status == statusNotFound {
		return nil, nil, storage.ErrNotExist
	}
	if status != statusOK {
		return nil, nil, fmt.Errorf("herd: remote get failed (status %d)", status)
	}

	// Parse response: [8B size][2B ct_len][ct][8B created][8B updated][value]
	if len(respBody) < 26 {
		return nil, nil, fmt.Errorf("herd: truncated get response")
	}
	objSize := int64(binary.BigEndian.Uint64(respBody[0:8]))
	ctLen := int(binary.BigEndian.Uint16(respBody[8:10]))
	ct := string(respBody[10 : 10+ctLen])
	created := int64(binary.BigEndian.Uint64(respBody[10+ctLen : 18+ctLen]))
	updated := int64(binary.BigEndian.Uint64(respBody[18+ctLen : 26+ctLen]))
	value := respBody[26+ctLen:]

	obj := &storage.Object{
		Bucket:      bucket,
		Key:         key,
		Size:        objSize,
		ContentType: ct,
		Created:     time.Unix(0, created),
		Updated:     time.Unix(0, updated),
	}

	return acquireMmapReader(value), obj, nil
}

func (rn *remoteNode) stat(bucket, key string) (*storage.Object, error) {
	body := encodeKeyBody(bucket, key)

	conn, err := rn.getConn()
	if err != nil {
		return nil, err
	}

	w := bufio.NewWriterSize(conn, 4096)
	r := bufio.NewReaderSize(conn, 4096)

	if err := writeRequest(w, opStat, body); err != nil {
		conn.Close()
		return nil, err
	}

	status, respBody, err := readResponse(r)
	if err != nil {
		conn.Close()
		return nil, err
	}
	rn.putConn(conn)

	if status == statusNotFound {
		return nil, storage.ErrNotExist
	}
	if status != statusOK {
		return nil, fmt.Errorf("herd: remote stat failed (status %d)", status)
	}

	// Parse: [8B size][2B ct_len][ct][8B created][8B updated]
	if len(respBody) < 26 {
		return nil, fmt.Errorf("herd: truncated stat response")
	}
	objSize := int64(binary.BigEndian.Uint64(respBody[0:8]))
	ctLen := int(binary.BigEndian.Uint16(respBody[8:10]))
	ct := string(respBody[10 : 10+ctLen])
	created := int64(binary.BigEndian.Uint64(respBody[10+ctLen : 18+ctLen]))
	updated := int64(binary.BigEndian.Uint64(respBody[18+ctLen : 26+ctLen]))

	return &storage.Object{
		Bucket:      bucket,
		Key:         key,
		Size:        objSize,
		ContentType: ct,
		Created:     time.Unix(0, created),
		Updated:     time.Unix(0, updated),
	}, nil
}

func (rn *remoteNode) del(bucket, key string) error {
	body := encodeKeyBody(bucket, key)

	conn, err := rn.getConn()
	if err != nil {
		return err
	}

	w := bufio.NewWriterSize(conn, 4096)
	r := bufio.NewReaderSize(conn, 4096)

	if err := writeRequest(w, opDelete, body); err != nil {
		conn.Close()
		return err
	}

	status, _, err := readResponse(r)
	if err != nil {
		conn.Close()
		return err
	}
	rn.putConn(conn)

	if status == statusNotFound {
		return storage.ErrNotExist
	}
	if status != statusOK {
		return fmt.Errorf("herd: remote delete failed (status %d)", status)
	}
	return nil
}

func (rn *remoteNode) list(bucket, prefix string, recursive bool) ([]*storage.Object, error) {
	body := encodeListBody(bucket, prefix, recursive)

	conn, err := rn.getConn()
	if err != nil {
		return nil, err
	}

	w := bufio.NewWriterSize(conn, 4096)
	r := bufio.NewReaderSize(conn, 65536)

	if err := writeRequest(w, opList, body); err != nil {
		conn.Close()
		return nil, err
	}

	status, respBody, err := readResponse(r)
	if err != nil {
		conn.Close()
		return nil, err
	}
	rn.putConn(conn)

	if status != statusOK {
		return nil, fmt.Errorf("herd: remote list failed (status %d)", status)
	}

	// Parse: [4B count][for each: 2B key_len + key + 8B size + 2B ct_len + ct + 8B created + 8B updated]
	if len(respBody) < 4 {
		return nil, nil
	}
	count := int(binary.BigEndian.Uint32(respBody[0:4]))
	pos := 4
	objs := make([]*storage.Object, 0, count)
	for i := 0; i < count && pos < len(respBody); i++ {
		if pos+2 > len(respBody) {
			break
		}
		kl := int(binary.BigEndian.Uint16(respBody[pos:]))
		pos += 2
		if pos+kl > len(respBody) {
			break
		}
		key := string(respBody[pos : pos+kl])
		pos += kl
		if pos+8 > len(respBody) {
			break
		}
		size := int64(binary.BigEndian.Uint64(respBody[pos:]))
		pos += 8
		if pos+2 > len(respBody) {
			break
		}
		cl := int(binary.BigEndian.Uint16(respBody[pos:]))
		pos += 2
		if pos+cl > len(respBody) {
			break
		}
		ct := string(respBody[pos : pos+cl])
		pos += cl
		if pos+16 > len(respBody) {
			break
		}
		created := int64(binary.BigEndian.Uint64(respBody[pos:]))
		pos += 8
		updated := int64(binary.BigEndian.Uint64(respBody[pos:]))
		pos += 8

		objs = append(objs, &storage.Object{
			Bucket: bucket, Key: key, Size: size, ContentType: ct,
			Created: time.Unix(0, created), Updated: time.Unix(0, updated),
		})
	}

	return objs, nil
}

// StoreEngine is the exported interface for accessing the underlying store.
type StoreEngine interface {
	storage.Storage
	storeEngine()
}

func (s *store) storeEngine() {}

// NodeServer serves the herd binary protocol for a standalone node.
type NodeServer struct {
	engine   *store
	listener net.Listener
}

// NewNodeServer creates a new TCP server backed by the given store.
func NewNodeServer(engine *store) *NodeServer {
	return &NodeServer{engine: engine}
}

// NewNodeServerFromEngine creates a server from the exported StoreEngine interface.
func NewNodeServerFromEngine(engine StoreEngine) *NodeServer {
	return &NodeServer{engine: engine.(*store)}
}

// ListenAndServe starts the TCP server.
func (ns *NodeServer) ListenAndServe(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	ns.listener = ln

	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		go ns.handleConn(conn)
	}
}

// Close stops the server.
func (ns *NodeServer) Close() error {
	if ns.listener != nil {
		return ns.listener.Close()
	}
	return nil
}

func (ns *NodeServer) handleConn(conn net.Conn) {
	defer conn.Close()

	if tc, ok := conn.(*net.TCPConn); ok {
		tc.SetNoDelay(true)
	}

	r := bufio.NewReaderSize(conn, 65536)
	w := bufio.NewWriterSize(conn, 65536)

	for {
		// Read request header.
		var hdr [7]byte
		if _, err := io.ReadFull(r, hdr[:]); err != nil {
			return
		}
		magic := binary.BigEndian.Uint16(hdr[0:2])
		if magic != protoMagic {
			return
		}
		op := hdr[2]
		bodyLen := binary.BigEndian.Uint32(hdr[3:7])

		var body []byte
		if bodyLen > 0 {
			body = make([]byte, bodyLen)
			if _, err := io.ReadFull(r, body); err != nil {
				return
			}
		}

		ns.handleRequest(w, op, body)
	}
}

func (ns *NodeServer) handleRequest(w *bufio.Writer, op byte, body []byte) {
	switch op {
	case opPing:
		writeResponseMsg(w, statusOK, nil)
	case opPut:
		ns.handlePut(w, body)
	case opGet:
		ns.handleGet(w, body)
	case opStat:
		ns.handleStat(w, body)
	case opDelete:
		ns.handleDelete(w, body)
	case opList:
		ns.handleList(w, body)
	default:
		writeResponseMsg(w, statusError, []byte("unknown op"))
	}
}

func (ns *NodeServer) handlePut(w *bufio.Writer, body []byte) {
	pos := 0
	bl := int(binary.BigEndian.Uint16(body[pos:]))
	pos += 2
	bucket := string(body[pos : pos+bl])
	pos += bl
	kl := int(binary.BigEndian.Uint16(body[pos:]))
	pos += 2
	key := string(body[pos : pos+kl])
	pos += kl
	cl := int(binary.BigEndian.Uint16(body[pos:]))
	pos += 2
	contentType := string(body[pos : pos+cl])
	pos += cl
	_ = int64(binary.BigEndian.Uint64(body[pos:])) // timestamp
	pos += 8
	dataLen := int64(binary.BigEndian.Uint64(body[pos:]))
	pos += 8
	data := body[pos : pos+int(dataLen)]

	bkt := ns.engine.Bucket(bucket)
	_, err := bkt.Write(context.Background(), key, bytes.NewReader(data), dataLen, contentType, nil)
	if err != nil {
		writeResponseMsg(w, statusError, []byte(err.Error()))
		return
	}
	writeResponseMsg(w, statusOK, nil)
}

func (ns *NodeServer) handleGet(w *bufio.Writer, body []byte) {
	pos := 0
	bl := int(binary.BigEndian.Uint16(body[pos:]))
	pos += 2
	bucket := string(body[pos : pos+bl])
	pos += bl
	kl := int(binary.BigEndian.Uint16(body[pos:]))
	pos += 2
	key := string(body[pos : pos+kl])
	pos += kl
	offset := int64(binary.BigEndian.Uint64(body[pos:]))
	pos += 8
	length := int64(binary.BigEndian.Uint64(body[pos:]))

	bkt := ns.engine.Bucket(bucket)
	rc, obj, err := bkt.Open(context.Background(), key, offset, length, nil)
	if err != nil {
		if err == storage.ErrNotExist {
			writeResponseMsg(w, statusNotFound, nil)
		} else {
			writeResponseMsg(w, statusError, []byte(err.Error()))
		}
		return
	}
	defer rc.Close()

	data, _ := io.ReadAll(rc)
	ct := obj.ContentType
	ctLen := len(ct)

	resp := make([]byte, 26+ctLen+len(data))
	binary.BigEndian.PutUint64(resp[0:8], uint64(obj.Size))
	binary.BigEndian.PutUint16(resp[8:10], uint16(ctLen))
	copy(resp[10:], ct)
	binary.BigEndian.PutUint64(resp[10+ctLen:], uint64(obj.Created.UnixNano()))
	binary.BigEndian.PutUint64(resp[18+ctLen:], uint64(obj.Updated.UnixNano()))
	copy(resp[26+ctLen:], data)

	writeResponseMsg(w, statusOK, resp)
}

func (ns *NodeServer) handleStat(w *bufio.Writer, body []byte) {
	pos := 0
	bl := int(binary.BigEndian.Uint16(body[pos:]))
	pos += 2
	bucket := string(body[pos : pos+bl])
	pos += bl
	kl := int(binary.BigEndian.Uint16(body[pos:]))
	pos += 2
	key := string(body[pos : pos+kl])

	bkt := ns.engine.Bucket(bucket)
	obj, err := bkt.Stat(context.Background(), key, nil)
	if err != nil {
		if err == storage.ErrNotExist {
			writeResponseMsg(w, statusNotFound, nil)
		} else {
			writeResponseMsg(w, statusError, []byte(err.Error()))
		}
		return
	}

	ct := obj.ContentType
	ctLen := len(ct)
	resp := make([]byte, 26+ctLen)
	binary.BigEndian.PutUint64(resp[0:8], uint64(obj.Size))
	binary.BigEndian.PutUint16(resp[8:10], uint16(ctLen))
	copy(resp[10:], ct)
	binary.BigEndian.PutUint64(resp[10+ctLen:], uint64(obj.Created.UnixNano()))
	binary.BigEndian.PutUint64(resp[18+ctLen:], uint64(obj.Updated.UnixNano()))

	writeResponseMsg(w, statusOK, resp)
}

func (ns *NodeServer) handleDelete(w *bufio.Writer, body []byte) {
	pos := 0
	bl := int(binary.BigEndian.Uint16(body[pos:]))
	pos += 2
	bucket := string(body[pos : pos+bl])
	pos += bl
	kl := int(binary.BigEndian.Uint16(body[pos:]))
	pos += 2
	key := string(body[pos : pos+kl])

	bkt := ns.engine.Bucket(bucket)
	err := bkt.Delete(context.Background(), key, nil)
	if err != nil {
		if err == storage.ErrNotExist {
			writeResponseMsg(w, statusNotFound, nil)
		} else {
			writeResponseMsg(w, statusError, []byte(err.Error()))
		}
		return
	}
	writeResponseMsg(w, statusOK, nil)
}

func (ns *NodeServer) handleList(w *bufio.Writer, body []byte) {
	pos := 0
	bl := int(binary.BigEndian.Uint16(body[pos:]))
	pos += 2
	bucketName := string(body[pos : pos+bl])
	pos += bl
	pl := int(binary.BigEndian.Uint16(body[pos:]))
	pos += 2
	prefix := string(body[pos : pos+pl])
	pos += pl
	recursive := body[pos] == 1

	bkt := ns.engine.Bucket(bucketName).(*bucket)
	results := bkt.listAll(prefix)

	if !recursive {
		var filtered []listResult
		for _, r := range results {
			rest := strings.TrimPrefix(r.key, prefix)
			rest = strings.TrimPrefix(rest, "/")
			if !strings.Contains(rest, "/") {
				filtered = append(filtered, r)
			}
		}
		results = filtered
	}

	// Encode: [4B count][per item: 2B key_len + key + 8B size + 2B ct_len + ct + 8B created + 8B updated]
	totalSize := 4
	for _, r := range results {
		totalSize += 2 + len(r.key) + 8 + 2 + len(r.entry.contentType) + 16
	}
	resp := make([]byte, totalSize)
	binary.BigEndian.PutUint32(resp[0:4], uint32(len(results)))
	off := 4
	for _, r := range results {
		binary.BigEndian.PutUint16(resp[off:], uint16(len(r.key)))
		off += 2
		copy(resp[off:], r.key)
		off += len(r.key)
		binary.BigEndian.PutUint64(resp[off:], uint64(r.entry.size))
		off += 8
		binary.BigEndian.PutUint16(resp[off:], uint16(len(r.entry.contentType)))
		off += 2
		copy(resp[off:], r.entry.contentType)
		off += len(r.entry.contentType)
		binary.BigEndian.PutUint64(resp[off:], uint64(r.entry.created))
		off += 8
		binary.BigEndian.PutUint64(resp[off:], uint64(r.entry.updated))
		off += 8
	}

	writeResponseMsg(w, statusOK, resp)
}

func writeResponseMsg(w *bufio.Writer, status byte, body []byte) {
	var hdr [5]byte
	hdr[0] = status
	binary.BigEndian.PutUint32(hdr[1:5], uint32(len(body)))
	w.Write(hdr[:])
	if len(body) > 0 {
		w.Write(body)
	}
	w.Flush()
}
