// Package ant implements a storage driver backed by an Adaptive Radix Tree (ART),
// inspired by the SMART ART paper (OSDI 2023).
//
// The ART provides O(key_length) lookups by decomposing keys byte-by-byte.
// Four node types (Node4, Node16, Node48, Node256) adapt based on child occupancy.
// Values are stored in an append-only value log; the ART holds offsets into this log.
// A write-ahead log provides crash recovery.
//
// DSN format: ant:///path/to/root?sync=none
package ant

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
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
	storage.Register("ant", &driver{})
}

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

	noSync := strings.EqualFold(opts.Get("sync"), "none")

	if err := os.MkdirAll(root, 0o750); err != nil {
		return nil, fmt.Errorf("ant: create root %q: %w", root, err)
	}

	st := &store{
		root:      root,
		noSync:    noSync,
		tree:      &artTree{},
		bucketMap: make(map[string]time.Time),
	}

	// Open value log (append-only).
	vlogPath := filepath.Join(root, "values.dat")
	vf, err := os.OpenFile(vlogPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("ant: open value log: %w", err)
	}
	st.vlog = vf

	info, err := vf.Stat()
	if err != nil {
		vf.Close()
		return nil, fmt.Errorf("ant: stat value log: %w", err)
	}
	st.vlogSize = info.Size()

	// Open WAL.
	walPath := filepath.Join(root, "wal.log")
	wf, err := os.OpenFile(walPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		vf.Close()
		return nil, fmt.Errorf("ant: open wal: %w", err)
	}
	st.wal = wf

	// Replay WAL to rebuild ART.
	if err := st.replayWAL(); err != nil {
		wf.Close()
		vf.Close()
		return nil, fmt.Errorf("ant: replay wal: %w", err)
	}

	// Truncate WAL after successful replay to avoid unbounded growth.
	if err := st.truncateWAL(); err != nil {
		wf.Close()
		vf.Close()
		return nil, fmt.Errorf("ant: truncate wal after replay: %w", err)
	}

	return st, nil
}

func parseDSN(dsn string) (string, url.Values, error) {
	if strings.TrimSpace(dsn) == "" {
		return "", nil, errors.New("ant: empty dsn")
	}

	u, err := url.Parse(dsn)
	if err != nil {
		return "", nil, fmt.Errorf("ant: parse dsn: %w", err)
	}
	if u.Scheme != "ant" && u.Scheme != "" {
		return "", nil, fmt.Errorf("ant: unsupported scheme %q", u.Scheme)
	}

	root := u.Path
	if root == "" && u.Host != "" {
		root = "/" + u.Host + u.Path
	}
	if root == "" {
		return "", nil, errors.New("ant: missing root path")
	}

	return filepath.Clean(root), u.Query(), nil
}

// ---------------------------------------------------------------------------
// ART Node Types
// ---------------------------------------------------------------------------

type nodeKind byte

const (
	kindNode4   nodeKind = 4
	kindNode16  nodeKind = 16
	kindNode48  nodeKind = 48
	kindNode256 nodeKind = 0 // represents 256
)

// artNode is a single node in the Adaptive Radix Tree.
//
// The active portion of keys and children depends on nodeKind:
//   - Node4:   keys[0..numChildren-1], children[0..numChildren-1]
//   - Node16:  keys[0..numChildren-1] (sorted), children[0..numChildren-1]
//   - Node48:  childIndex[byte] -> slot (255=empty), children[0..numChildren-1]
//   - Node256: children256[byte] (direct array)
type artNode struct {
	kind        nodeKind
	numChildren uint16
	prefix      []byte // path compression prefix

	// Node4 / Node16: used portion is [0..numChildren).
	keys     [16]byte
	children [48]*artNode

	// Node48: key byte -> slot index (255 means empty).
	childIndex [256]byte

	// Node256: direct 256-slot pointer array.
	children256 [256]*artNode

	// Leaf data (non-nil for leaf nodes).
	leaf *leafData
}

type leafData struct {
	key         []byte // full composite key for verification
	valueOffset int64
	valueSize   int64
	contentType string
	created     int64 // Unix nano
	updated     int64 // Unix nano
	deleted     bool
}

// artTree is the top-level ART structure with a mutex for concurrent access.
type artTree struct {
	mu   sync.RWMutex
	root *artNode
	size int64 // number of live leaves
}

// ---------------------------------------------------------------------------
// ART Operations
// ---------------------------------------------------------------------------

// artSearch traverses the tree for a key and returns the leaf, or nil.
func artSearch(node *artNode, key []byte) *leafData {
	depth := 0
	cur := node
	for cur != nil {
		// Check prefix match.
		if len(cur.prefix) > 0 {
			pLen := len(cur.prefix)
			if depth+pLen > len(key) {
				return nil
			}
			for i := 0; i < pLen; i++ {
				if key[depth+i] != cur.prefix[i] {
					return nil
				}
			}
			depth += pLen
		}

		// If this node has a leaf, check for exact match.
		if cur.leaf != nil {
			if bytes.Equal(cur.leaf.key, key) && !cur.leaf.deleted {
				return cur.leaf
			}
			if depth >= len(key) {
				return nil
			}
		}

		if depth >= len(key) {
			return nil
		}

		// Find child for next byte.
		b := key[depth]
		depth++
		cur = findChild(cur, b)
	}
	return nil
}

// findChild returns the child of node for the given byte, or nil.
func findChild(node *artNode, b byte) *artNode {
	switch node.kind {
	case kindNode4:
		for i := uint16(0); i < node.numChildren; i++ {
			if node.keys[i] == b {
				return node.children[i]
			}
		}
	case kindNode16:
		// Sorted scan.
		lo, hi := 0, int(node.numChildren)
		for lo < hi {
			mid := lo + (hi-lo)/2
			if node.keys[mid] < b {
				lo = mid + 1
			} else if node.keys[mid] > b {
				hi = mid
			} else {
				return node.children[mid]
			}
		}
	case kindNode48:
		idx := node.childIndex[b]
		if idx != 255 {
			return node.children[idx]
		}
	case kindNode256:
		return node.children256[b]
	}
	return nil
}

// artInsert inserts or updates a leaf in the tree. Returns the root.
func artInsert(root *artNode, key []byte, leaf *leafData) *artNode {
	if root == nil {
		n := newNode4()
		n.leaf = leaf
		return n
	}
	insertRecursive(&root, root, key, leaf, 0)
	return root
}

func insertRecursive(ref **artNode, node *artNode, key []byte, leaf *leafData, depth int) {
	// Empty node: store leaf here.
	if node == nil {
		n := newNode4()
		n.leaf = leaf
		*ref = n
		return
	}

	// Check prefix.
	if len(node.prefix) > 0 {
		mismatch := prefixMismatch(node, key, depth)
		if mismatch < len(node.prefix) {
			// Split at mismatch.
			newInner := newNode4()
			newInner.prefix = make([]byte, mismatch)
			copy(newInner.prefix, node.prefix[:mismatch])

			// Old node becomes a child under the byte at mismatch.
			oldByte := node.prefix[mismatch]
			node.prefix = node.prefix[mismatch+1:]
			addChild(newInner, oldByte, node)

			// New leaf becomes a child under the key byte at depth+mismatch.
			if depth+mismatch < len(key) {
				newLeaf := newNode4()
				newLeaf.leaf = leaf
				newLeaf.prefix = make([]byte, len(key)-(depth+mismatch+1))
				copy(newLeaf.prefix, key[depth+mismatch+1:])
				addChild(newInner, key[depth+mismatch], newLeaf)
			} else {
				newInner.leaf = leaf
			}

			*ref = newInner
			return
		}
		depth += len(node.prefix)
	}

	// If node is a leaf-only node (no children, has leaf).
	if node.leaf != nil && node.numChildren == 0 {
		existingKey := node.leaf.key
		if bytes.Equal(existingKey, key) {
			// Update existing leaf.
			node.leaf = leaf
			return
		}
		// Need to split: find first differing byte.
		commonLen := commonPrefixLength(existingKey, key, depth)

		newInner := newNode4()
		newInner.prefix = make([]byte, commonLen)
		copy(newInner.prefix, key[depth:depth+commonLen])
		newDepth := depth + commonLen

		// Existing leaf as child.
		if newDepth < len(existingKey) {
			oldLeafNode := newNode4()
			oldLeafNode.leaf = node.leaf
			if newDepth+1 < len(existingKey) {
				oldLeafNode.prefix = make([]byte, len(existingKey)-(newDepth+1))
				copy(oldLeafNode.prefix, existingKey[newDepth+1:])
			}
			addChild(newInner, existingKey[newDepth], oldLeafNode)
		} else {
			newInner.leaf = node.leaf
		}

		// New leaf as child.
		if newDepth < len(key) {
			newLeafNode := newNode4()
			newLeafNode.leaf = leaf
			if newDepth+1 < len(key) {
				newLeafNode.prefix = make([]byte, len(key)-(newDepth+1))
				copy(newLeafNode.prefix, key[newDepth+1:])
			}
			addChild(newInner, key[newDepth], newLeafNode)
		} else {
			newInner.leaf = leaf
		}

		*ref = newInner
		return
	}

	// If at end of key, set leaf on this node.
	if depth >= len(key) {
		node.leaf = leaf
		return
	}

	// Find child.
	b := key[depth]
	child := findChild(node, b)
	if child != nil {
		childRef := findChildRef(node, b)
		insertRecursive(childRef, child, key, leaf, depth+1)
	} else {
		newLeafNode := newNode4()
		newLeafNode.leaf = leaf
		if depth+1 < len(key) {
			newLeafNode.prefix = make([]byte, len(key)-(depth+1))
			copy(newLeafNode.prefix, key[depth+1:])
		}
		addChild(node, b, newLeafNode)
	}
}

// findChildRef returns a pointer to the child slot for the given byte.
func findChildRef(node *artNode, b byte) **artNode {
	switch node.kind {
	case kindNode4:
		for i := uint16(0); i < node.numChildren; i++ {
			if node.keys[i] == b {
				return &node.children[i]
			}
		}
	case kindNode16:
		lo, hi := 0, int(node.numChildren)
		for lo < hi {
			mid := lo + (hi-lo)/2
			if node.keys[mid] < b {
				lo = mid + 1
			} else if node.keys[mid] > b {
				hi = mid
			} else {
				return &node.children[mid]
			}
		}
	case kindNode48:
		idx := node.childIndex[b]
		if idx != 255 {
			return &node.children[idx]
		}
	case kindNode256:
		return &node.children256[b]
	}
	return nil
}

func prefixMismatch(node *artNode, key []byte, depth int) int {
	maxLen := len(node.prefix)
	remaining := len(key) - depth
	if remaining < maxLen {
		maxLen = remaining
	}
	for i := 0; i < maxLen; i++ {
		if node.prefix[i] != key[depth+i] {
			return i
		}
	}
	return maxLen
}

func commonPrefixLength(a, b []byte, depth int) int {
	maxLen := len(a) - depth
	if bl := len(b) - depth; bl < maxLen {
		maxLen = bl
	}
	for i := 0; i < maxLen; i++ {
		if a[depth+i] != b[depth+i] {
			return i
		}
	}
	return maxLen
}

func newNode4() *artNode {
	n := &artNode{kind: kindNode4}
	return n
}

func newNode16() *artNode {
	n := &artNode{kind: kindNode16}
	return n
}

func newNode48() *artNode {
	n := &artNode{kind: kindNode48}
	for i := range n.childIndex {
		n.childIndex[i] = 255
	}
	return n
}

func newNode256() *artNode {
	return &artNode{kind: kindNode256}
}

// addChild adds a child to node, growing the node type if needed.
func addChild(node *artNode, b byte, child *artNode) {
	switch node.kind {
	case kindNode4:
		if node.numChildren < 4 {
			node.keys[node.numChildren] = b
			node.children[node.numChildren] = child
			node.numChildren++
		} else {
			growToNode16(node)
			addChild(node, b, child)
		}
	case kindNode16:
		if node.numChildren < 16 {
			// Insert sorted.
			idx := sort.Search(int(node.numChildren), func(i int) bool {
				return node.keys[i] >= b
			})
			copy(node.keys[idx+1:], node.keys[idx:node.numChildren])
			copy(node.children[idx+1:], node.children[idx:node.numChildren])
			node.keys[idx] = b
			node.children[idx] = child
			node.numChildren++
		} else {
			growToNode48(node)
			addChild(node, b, child)
		}
	case kindNode48:
		if node.numChildren < 48 {
			slot := node.numChildren
			node.childIndex[b] = byte(slot)
			node.children[slot] = child
			node.numChildren++
		} else {
			growToNode256(node)
			addChild(node, b, child)
		}
	case kindNode256:
		node.children256[b] = child
		node.numChildren++
	}
}

func growToNode16(node *artNode) {
	newN := newNode16()
	newN.prefix = node.prefix
	newN.leaf = node.leaf

	// Copy children from node4, sorted by key.
	type kv struct {
		k byte
		c *artNode
	}
	items := make([]kv, node.numChildren)
	for i := uint16(0); i < node.numChildren; i++ {
		items[i] = kv{node.keys[i], node.children[i]}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].k < items[j].k })
	for i, it := range items {
		newN.keys[i] = it.k
		newN.children[i] = it.c
	}
	newN.numChildren = node.numChildren

	// Swap in-place.
	*node = *newN
}

func growToNode48(node *artNode) {
	newN := newNode48()
	newN.prefix = node.prefix
	newN.leaf = node.leaf

	for i := uint16(0); i < node.numChildren; i++ {
		newN.childIndex[node.keys[i]] = byte(i)
		newN.children[i] = node.children[i]
	}
	newN.numChildren = node.numChildren

	*node = *newN
}

func growToNode256(node *artNode) {
	newN := newNode256()
	newN.prefix = node.prefix
	newN.leaf = node.leaf

	for i := 0; i < 256; i++ {
		idx := node.childIndex[byte(i)]
		if idx != 255 {
			newN.children256[i] = node.children[idx]
		}
	}
	newN.numChildren = node.numChildren

	*node = *newN
}

// artDelete removes a leaf from the tree. Returns true if a leaf was found and removed.
func artDelete(tree *artTree, key []byte) bool {
	if tree.root == nil {
		return false
	}
	found := artDeleteRecursive(&tree.root, tree.root, key, 0)
	if found {
		tree.size--
	}
	return found
}

// artDeleteRecursive removes a leaf node from the tree and cleans up empty parent nodes.
// Returns true if a leaf was found and removed.
func artDeleteRecursive(ref **artNode, node *artNode, key []byte, depth int) bool {
	if node == nil {
		return false
	}

	// Check prefix match.
	if len(node.prefix) > 0 {
		pLen := len(node.prefix)
		if depth+pLen > len(key) {
			return false
		}
		for i := 0; i < pLen; i++ {
			if key[depth+i] != node.prefix[i] {
				return false
			}
		}
		depth += pLen
	}

	// If this node has a leaf that matches, remove it.
	if node.leaf != nil && bytes.Equal(node.leaf.key, key) && !node.leaf.deleted {
		node.leaf = nil
		// If the node has no children, remove it entirely.
		if node.numChildren == 0 {
			*ref = nil
		} else if node.numChildren == 1 {
			// Merge with single remaining child (path compression).
			child := getOnlyChild(node)
			if child != nil {
				// Combine prefixes: node.prefix + child_byte + child.prefix
				// The child_byte is the key byte used to reach the child.
				childByte := getOnlyChildByte(node)
				newPrefix := make([]byte, 0, len(node.prefix)+1+len(child.prefix))
				newPrefix = append(newPrefix, node.prefix...)
				newPrefix = append(newPrefix, childByte)
				newPrefix = append(newPrefix, child.prefix...)
				child.prefix = newPrefix
				*ref = child
			}
		}
		return true
	}

	if depth >= len(key) {
		return false
	}

	// Find the child for the next byte.
	b := key[depth]
	childRef := findChildRef(node, b)
	if childRef == nil || *childRef == nil {
		return false
	}

	found := artDeleteRecursive(childRef, *childRef, key, depth+1)
	if !found {
		return false
	}

	// If the child was removed (set to nil), remove it from this node.
	if *childRef == nil {
		removeChild(node, b)
		// If this node now has no children and no leaf, remove it too.
		if node.numChildren == 0 && node.leaf == nil {
			*ref = nil
		} else if node.numChildren == 1 && node.leaf == nil {
			// Merge with single remaining child.
			child := getOnlyChild(node)
			if child != nil {
				childByte := getOnlyChildByte(node)
				newPrefix := make([]byte, 0, len(node.prefix)+1+len(child.prefix))
				newPrefix = append(newPrefix, node.prefix...)
				newPrefix = append(newPrefix, childByte)
				newPrefix = append(newPrefix, child.prefix...)
				child.prefix = newPrefix
				*ref = child
			}
		}
	}
	return true
}

// removeChild removes the child at byte b from node and decrements numChildren.
func removeChild(node *artNode, b byte) {
	switch node.kind {
	case kindNode4:
		for i := uint16(0); i < node.numChildren; i++ {
			if node.keys[i] == b {
				// Shift remaining entries left.
				last := node.numChildren - 1
				if i < last {
					node.keys[i] = node.keys[last]
					node.children[i] = node.children[last]
				}
				node.keys[last] = 0
				node.children[last] = nil
				node.numChildren--
				return
			}
		}
	case kindNode16:
		idx := -1
		for i := uint16(0); i < node.numChildren; i++ {
			if node.keys[i] == b {
				idx = int(i)
				break
			}
		}
		if idx >= 0 {
			copy(node.keys[idx:], node.keys[idx+1:node.numChildren])
			copy(node.children[idx:], node.children[idx+1:node.numChildren])
			node.keys[node.numChildren-1] = 0
			node.children[node.numChildren-1] = nil
			node.numChildren--
		}
	case kindNode48:
		slot := node.childIndex[b]
		if slot != 255 {
			node.childIndex[b] = 255
			node.children[slot] = nil
			node.numChildren--
		}
	case kindNode256:
		if node.children256[b] != nil {
			node.children256[b] = nil
			node.numChildren--
		}
	}
}

// getOnlyChild returns the single child of a node that has numChildren == 1.
func getOnlyChild(node *artNode) *artNode {
	switch node.kind {
	case kindNode4, kindNode16:
		if node.numChildren == 1 {
			return node.children[0]
		}
	case kindNode48:
		for i := 0; i < 256; i++ {
			idx := node.childIndex[byte(i)]
			if idx != 255 {
				return node.children[idx]
			}
		}
	case kindNode256:
		for i := 0; i < 256; i++ {
			if node.children256[i] != nil {
				return node.children256[i]
			}
		}
	}
	return nil
}

// getOnlyChildByte returns the key byte of the single child of a node with numChildren == 1.
func getOnlyChildByte(node *artNode) byte {
	switch node.kind {
	case kindNode4, kindNode16:
		if node.numChildren == 1 {
			return node.keys[0]
		}
	case kindNode48:
		for i := 0; i < 256; i++ {
			if node.childIndex[byte(i)] != 255 {
				return byte(i)
			}
		}
	case kindNode256:
		for i := 0; i < 256; i++ {
			if node.children256[i] != nil {
				return byte(i)
			}
		}
	}
	return 0
}

// artForEach iterates all non-deleted leaves in the tree, calling fn for each.
func artForEach(node *artNode, fn func(leaf *leafData)) {
	if node == nil {
		return
	}
	if node.leaf != nil && !node.leaf.deleted {
		fn(node.leaf)
	}
	switch node.kind {
	case kindNode4:
		for i := uint16(0); i < node.numChildren; i++ {
			artForEach(node.children[i], fn)
		}
	case kindNode16:
		for i := uint16(0); i < node.numChildren; i++ {
			artForEach(node.children[i], fn)
		}
	case kindNode48:
		for i := 0; i < 256; i++ {
			idx := node.childIndex[byte(i)]
			if idx != 255 {
				artForEach(node.children[idx], fn)
			}
		}
	case kindNode256:
		for i := 0; i < 256; i++ {
			if node.children256[i] != nil {
				artForEach(node.children256[i], fn)
			}
		}
	}
}

// artForEachPrefix iterates all non-deleted leaves whose key starts with prefix.
func artForEachPrefix(node *artNode, prefix []byte, fn func(leaf *leafData)) {
	if node == nil {
		return
	}
	artForEachPrefixHelper(node, prefix, 0, fn)
}

func artForEachPrefixHelper(node *artNode, prefix []byte, depth int, fn func(leaf *leafData)) {
	if node == nil {
		return
	}

	// Check prefix of this node.
	if len(node.prefix) > 0 {
		pLen := len(node.prefix)
		for i := 0; i < pLen && depth < len(prefix); i++ {
			if node.prefix[i] != prefix[depth] {
				return
			}
			depth++
		}
		// If we consumed node.prefix but haven't consumed search prefix,
		// continue to children. If we consumed search prefix, enumerate all.
		if depth < len(prefix) && pLen > len(prefix)-depth+pLen {
			// Prefix of node extends beyond search prefix;
			// check if node prefix starts with remaining search prefix.
		}
	}

	// If we have consumed the entire search prefix, enumerate everything.
	if depth >= len(prefix) {
		if node.leaf != nil && !node.leaf.deleted {
			if len(node.leaf.key) >= len(prefix) && bytes.HasPrefix(node.leaf.key, prefix) {
				fn(node.leaf)
			}
		}
		// Enumerate all children.
		switch node.kind {
		case kindNode4:
			for i := uint16(0); i < node.numChildren; i++ {
				artForEach(node.children[i], fn)
			}
		case kindNode16:
			for i := uint16(0); i < node.numChildren; i++ {
				artForEach(node.children[i], fn)
			}
		case kindNode48:
			for i := 0; i < 256; i++ {
				idx := node.childIndex[byte(i)]
				if idx != 255 {
					artForEach(node.children[idx], fn)
				}
			}
		case kindNode256:
			for i := 0; i < 256; i++ {
				if node.children256[i] != nil {
					artForEach(node.children256[i], fn)
				}
			}
		}
		return
	}

	// Check leaf at this node.
	if node.leaf != nil && !node.leaf.deleted {
		if bytes.HasPrefix(node.leaf.key, prefix) {
			fn(node.leaf)
		}
	}

	// Continue to child for next prefix byte.
	b := prefix[depth]
	child := findChild(node, b)
	if child != nil {
		artForEachPrefixHelper(child, prefix, depth+1, fn)
	}
}

// ---------------------------------------------------------------------------
// Store (storage.Storage)
// ---------------------------------------------------------------------------

type store struct {
	root   string
	noSync bool

	mu   sync.RWMutex
	tree *artTree

	vlog     *os.File
	vlogSize int64
	vlogMu   sync.Mutex

	wal   *os.File
	walMu sync.Mutex

	bucketMu  sync.RWMutex
	bucketMap map[string]time.Time // name -> created
}

var _ storage.Storage = (*store)(nil)

const maxBuckets = 10000

func (s *store) Bucket(name string) storage.Bucket {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "default"
	}
	name = safeBucketName(name)
	return &bucket{store: s, name: name}
}

func (s *store) Buckets(ctx context.Context, limit, offset int, opts storage.Options) (storage.BucketIter, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.bucketMu.RLock()
	names := make([]string, 0, len(s.bucketMap))
	for n := range s.bucketMap {
		names = append(names, n)
	}
	s.bucketMu.RUnlock()

	sort.Strings(names)

	if offset < 0 {
		offset = 0
	}
	if offset > len(names) {
		offset = len(names)
	}
	names = names[offset:]
	if limit > 0 && limit < len(names) {
		names = names[:limit]
	}

	s.bucketMu.RLock()
	infos := make([]*storage.BucketInfo, len(names))
	for i, n := range names {
		infos[i] = &storage.BucketInfo{
			Name:      n,
			CreatedAt: s.bucketMap[n],
		}
	}
	s.bucketMu.RUnlock()

	return &bucketIter{list: infos}, nil
}

func (s *store) CreateBucket(ctx context.Context, name string, opts storage.Options) (*storage.BucketInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("ant: bucket name required")
	}
	name = safeBucketName(name)

	s.bucketMu.Lock()
	if _, exists := s.bucketMap[name]; exists {
		s.bucketMu.Unlock()
		return nil, storage.ErrExist
	}
	if len(s.bucketMap) >= maxBuckets {
		s.bucketMu.Unlock()
		return nil, fmt.Errorf("ant: too many buckets (max %d)", maxBuckets)
	}
	now := time.Now()
	s.bucketMap[name] = now
	s.bucketMu.Unlock()

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
		return errors.New("ant: bucket name required")
	}
	name = safeBucketName(name)

	s.bucketMu.Lock()
	if _, exists := s.bucketMap[name]; !exists {
		s.bucketMu.Unlock()
		return storage.ErrNotExist
	}

	force := boolOpt(opts, "force")
	if !force {
		// Check if bucket has any objects.
		prefix := compositePrefix(name)
		hasObjects := false
		s.tree.mu.RLock()
		artForEachPrefix(s.tree.root, prefix, func(leaf *leafData) {
			hasObjects = true
		})
		s.tree.mu.RUnlock()
		if hasObjects {
			s.bucketMu.Unlock()
			return storage.ErrPermission
		}
	}

	delete(s.bucketMap, name)
	s.bucketMu.Unlock()

	// If force, delete all objects in the bucket.
	if force {
		prefix := compositePrefix(name)
		s.tree.mu.Lock()
		artForEachPrefix(s.tree.root, prefix, func(leaf *leafData) {
			leaf.deleted = true
			s.tree.size--
		})
		s.tree.mu.Unlock()
	}

	return nil
}

func (s *store) Features() storage.Features {
	return storage.Features{
		"move":        true,
		"directories": true,
		"multipart":   true,
	}
}

func (s *store) Close() error {
	var errs []error

	s.walMu.Lock()
	if s.wal != nil {
		if err := s.wal.Close(); err != nil {
			errs = append(errs, err)
		}
		s.wal = nil
	}
	s.walMu.Unlock()

	s.vlogMu.Lock()
	if s.vlog != nil {
		if err := s.vlog.Close(); err != nil {
			errs = append(errs, err)
		}
		s.vlog = nil
	}
	s.vlogMu.Unlock()

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Value log operations
// ---------------------------------------------------------------------------

// vlogEntry format:
//
//	ctLen(2B) | contentType | valLen(8B) | value | created(8B) | updated(8B)
func (s *store) appendValue(data []byte, contentType string, created, updated int64) (offset int64, totalSize int64, err error) {
	s.vlogMu.Lock()
	defer s.vlogMu.Unlock()

	ctBytes := []byte(contentType)
	ctLen := uint16(len(ctBytes))
	valLen := int64(len(data))
	totalSize = 2 + int64(ctLen) + 8 + valLen + 8 + 8

	buf := make([]byte, totalSize)
	binary.LittleEndian.PutUint16(buf[0:2], ctLen)
	copy(buf[2:2+ctLen], ctBytes)
	binary.LittleEndian.PutUint64(buf[2+int64(ctLen):2+int64(ctLen)+8], uint64(valLen))
	copy(buf[2+int64(ctLen)+8:2+int64(ctLen)+8+valLen], data)
	binary.LittleEndian.PutUint64(buf[2+int64(ctLen)+8+valLen:], uint64(created))
	binary.LittleEndian.PutUint64(buf[2+int64(ctLen)+8+valLen+8:], uint64(updated))

	offset = s.vlogSize
	if _, err = s.vlog.WriteAt(buf, offset); err != nil {
		return 0, 0, fmt.Errorf("ant: write vlog: %w", err)
	}

	if !s.noSync {
		if err = s.vlog.Sync(); err != nil {
			return 0, 0, fmt.Errorf("ant: sync vlog: %w", err)
		}
	}

	s.vlogSize += totalSize
	return offset, totalSize, nil
}

// readValue reads the value data from the value log at the given offset.
func (s *store) readValue(offset, totalSize int64) ([]byte, string, int64, int64, error) {
	buf := make([]byte, totalSize)
	if _, err := s.vlog.ReadAt(buf, offset); err != nil {
		return nil, "", 0, 0, fmt.Errorf("ant: read vlog: %w", err)
	}

	ctLen := binary.LittleEndian.Uint16(buf[0:2])
	ct := string(buf[2 : 2+ctLen])
	valLen := int64(binary.LittleEndian.Uint64(buf[2+int64(ctLen) : 2+int64(ctLen)+8]))
	val := buf[2+int64(ctLen)+8 : 2+int64(ctLen)+8+valLen]
	created := int64(binary.LittleEndian.Uint64(buf[2+int64(ctLen)+8+valLen:]))
	updated := int64(binary.LittleEndian.Uint64(buf[2+int64(ctLen)+8+valLen+8:]))

	return val, ct, created, updated, nil
}

// readValueOnly reads just the value data (no metadata) from the value log.
func (s *store) readValueOnly(offset, totalSize int64) ([]byte, error) {
	buf := make([]byte, totalSize)
	if _, err := s.vlog.ReadAt(buf, offset); err != nil {
		return nil, fmt.Errorf("ant: read vlog: %w", err)
	}

	ctLen := binary.LittleEndian.Uint16(buf[0:2])
	valLen := int64(binary.LittleEndian.Uint64(buf[2+int64(ctLen) : 2+int64(ctLen)+8]))
	val := make([]byte, valLen)
	copy(val, buf[2+int64(ctLen)+8:2+int64(ctLen)+8+valLen])
	return val, nil
}

// ---------------------------------------------------------------------------
// WAL operations
// ---------------------------------------------------------------------------

// WAL entry format:
//
//	op(1B) | keyLen(2B) | key | valOffset(8B) | valSize(8B) | ts(8B)
//
// op: 'P' = put, 'D' = delete
const (
	walOpPut    byte = 'P'
	walOpDelete byte = 'D'
)

func (s *store) appendWAL(op byte, key []byte, valOffset, valSize int64, ts int64) error {
	s.walMu.Lock()
	defer s.walMu.Unlock()

	keyLen := uint16(len(key))
	entrySize := 1 + 2 + int(keyLen) + 8 + 8 + 8
	buf := make([]byte, entrySize)

	buf[0] = op
	binary.LittleEndian.PutUint16(buf[1:3], keyLen)
	copy(buf[3:3+keyLen], key)
	binary.LittleEndian.PutUint64(buf[3+keyLen:3+keyLen+8], uint64(valOffset))
	binary.LittleEndian.PutUint64(buf[3+keyLen+8:3+keyLen+16], uint64(valSize))
	binary.LittleEndian.PutUint64(buf[3+keyLen+16:3+keyLen+24], uint64(ts))

	if _, err := s.wal.Write(buf); err != nil {
		return fmt.Errorf("ant: write wal: %w", err)
	}

	if !s.noSync {
		if err := s.wal.Sync(); err != nil {
			return fmt.Errorf("ant: sync wal: %w", err)
		}
	}

	return nil
}

func (s *store) replayWAL() error {
	info, err := s.wal.Stat()
	if err != nil {
		return fmt.Errorf("ant: stat wal: %w", err)
	}
	if info.Size() == 0 {
		return nil
	}

	data, err := io.ReadAll(io.NewSectionReader(s.wal, 0, info.Size()))
	if err != nil {
		return fmt.Errorf("ant: read wal: %w", err)
	}

	pos := 0
	for pos < len(data) {
		if pos+1+2 > len(data) {
			break // truncated entry
		}

		op := data[pos]
		keyLen := int(binary.LittleEndian.Uint16(data[pos+1 : pos+3]))
		pos += 3

		if pos+keyLen+24 > len(data) {
			break // truncated entry
		}

		key := make([]byte, keyLen)
		copy(key, data[pos:pos+keyLen])
		pos += keyLen

		valOffset := int64(binary.LittleEndian.Uint64(data[pos : pos+8]))
		valSize := int64(binary.LittleEndian.Uint64(data[pos+8 : pos+16]))
		ts := int64(binary.LittleEndian.Uint64(data[pos+16 : pos+24]))
		pos += 24

		switch op {
		case walOpPut:
			// Read content type from value log to reconstruct leaf.
			var ct string
			if valSize > 0 {
				ctBuf := make([]byte, 2)
				if _, err := s.vlog.ReadAt(ctBuf, valOffset); err == nil {
					ctLen := binary.LittleEndian.Uint16(ctBuf)
					if ctLen > 0 {
						ctData := make([]byte, ctLen)
						if _, err := s.vlog.ReadAt(ctData, valOffset+2); err == nil {
							ct = string(ctData)
						}
					}
				}
			}

			leaf := &leafData{
				key:         key,
				valueOffset: valOffset,
				valueSize:   valSize,
				contentType: ct,
				created:     ts,
				updated:     ts,
			}
			s.tree.root = artInsert(s.tree.root, key, leaf)
			s.tree.size++

			// Register bucket.
			bucketName, _ := splitCompositeKey(key)
			if bucketName != "" {
				s.bucketMu.Lock()
				if _, exists := s.bucketMap[bucketName]; !exists {
					s.bucketMap[bucketName] = time.Unix(0, ts)
				}
				s.bucketMu.Unlock()
			}

		case walOpDelete:
			lf := artSearch(s.tree.root, key)
			if lf != nil {
				lf.deleted = true
				s.tree.size--
			}
		}
	}

	return nil
}

// truncateWAL resets the WAL file to zero length.
func (s *store) truncateWAL() error {
	s.walMu.Lock()
	defer s.walMu.Unlock()

	if s.wal == nil {
		return nil
	}
	if err := s.wal.Truncate(0); err != nil {
		return fmt.Errorf("ant: truncate wal: %w", err)
	}
	if _, err := s.wal.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("ant: seek wal: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Bucket (storage.Bucket + HasDirectories + HasMultipart)
// ---------------------------------------------------------------------------

type bucket struct {
	store *store
	name  string

	mpMu      sync.RWMutex
	mpUploads map[string]*multipartUpload
}

var (
	_ storage.Bucket         = (*bucket)(nil)
	_ storage.HasDirectories = (*bucket)(nil)
	_ storage.HasMultipart   = (*bucket)(nil)
)

func (b *bucket) Name() string { return b.name }

func (b *bucket) Features() storage.Features {
	return b.store.Features()
}

func (b *bucket) Info(ctx context.Context) (*storage.BucketInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	b.store.bucketMu.RLock()
	created, exists := b.store.bucketMap[b.name]
	b.store.bucketMu.RUnlock()

	if !exists {
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

	relKey, err := cleanKey(key)
	if err != nil {
		return nil, err
	}

	// Ensure bucket exists.
	b.store.bucketMu.Lock()
	if _, exists := b.store.bucketMap[b.name]; !exists {
		if len(b.store.bucketMap) < maxBuckets {
			b.store.bucketMap[b.name] = time.Now()
		}
	}
	b.store.bucketMu.Unlock()

	// Read all data.
	var data []byte
	if size > 0 {
		data = make([]byte, size)
		n, err := io.ReadFull(src, data)
		if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
			return nil, fmt.Errorf("ant: read: %w", err)
		}
		data = data[:n]
	} else {
		data, err = io.ReadAll(src)
		if err != nil {
			return nil, fmt.Errorf("ant: read: %w", err)
		}
	}

	now := time.Now().UnixNano()

	// Check if key already exists (for preserving created time).
	compositeKey := compositeKey(b.name, relKey)
	created := now
	b.store.tree.mu.RLock()
	existing := artSearch(b.store.tree.root, compositeKey)
	if existing != nil {
		created = existing.created
	}
	b.store.tree.mu.RUnlock()

	// Append value to value log.
	offset, totalSize, err := b.store.appendValue(data, contentType, created, now)
	if err != nil {
		return nil, err
	}

	// Append to WAL.
	if err := b.store.appendWAL(walOpPut, compositeKey, offset, totalSize, created); err != nil {
		return nil, err
	}

	// Insert into ART.
	leaf := &leafData{
		key:         compositeKey,
		valueOffset: offset,
		valueSize:   totalSize,
		contentType: contentType,
		created:     created,
		updated:     now,
	}

	b.store.tree.mu.Lock()
	// If updating, remove old leaf count first.
	oldLeaf := artSearch(b.store.tree.root, compositeKey)
	if oldLeaf != nil {
		b.store.tree.size-- // will be re-added by insert
	}
	b.store.tree.root = artInsert(b.store.tree.root, compositeKey, leaf)
	b.store.tree.size++
	b.store.tree.mu.Unlock()

	return &storage.Object{
		Bucket:      b.name,
		Key:         relToKey(relKey),
		Size:        int64(len(data)),
		ContentType: contentType,
		Created:     time.Unix(0, created),
		Updated:     time.Unix(0, now),
	}, nil
}

func (b *bucket) Open(ctx context.Context, key string, offset, length int64, opts storage.Options) (io.ReadCloser, *storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}

	relKey, err := cleanKey(key)
	if err != nil {
		return nil, nil, err
	}

	compositeK := compositeKey(b.name, relKey)

	b.store.tree.mu.RLock()
	leaf := artSearch(b.store.tree.root, compositeK)
	b.store.tree.mu.RUnlock()

	if leaf == nil {
		return nil, nil, storage.ErrNotExist
	}

	data, ct, created, updated, err := b.store.readValue(leaf.valueOffset, leaf.valueSize)
	if err != nil {
		return nil, nil, err
	}

	objSize := int64(len(data))
	obj := &storage.Object{
		Bucket:      b.name,
		Key:         relToKey(relKey),
		Size:        objSize,
		ContentType: ct,
		Created:     time.Unix(0, created),
		Updated:     time.Unix(0, updated),
	}

	// Apply range.
	if offset > 0 {
		if offset >= int64(len(data)) {
			data = nil
		} else {
			data = data[offset:]
		}
	}
	if length > 0 && int64(len(data)) > length {
		data = data[:length]
	}

	return io.NopCloser(bytes.NewReader(data)), obj, nil
}

func (b *bucket) Stat(ctx context.Context, key string, opts storage.Options) (*storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	relKey, err := cleanKey(key)
	if err != nil {
		return nil, err
	}

	compositeK := compositeKey(b.name, relKey)

	b.store.tree.mu.RLock()
	leaf := artSearch(b.store.tree.root, compositeK)
	b.store.tree.mu.RUnlock()

	if leaf == nil {
		// Check if it's a directory prefix.
		dirPrefix := compositeKey(b.name, relKey+"/")
		hasChildren := false
		b.store.tree.mu.RLock()
		artForEachPrefix(b.store.tree.root, dirPrefix, func(lf *leafData) {
			hasChildren = true
		})
		b.store.tree.mu.RUnlock()

		if hasChildren {
			return &storage.Object{
				Bucket: b.name,
				Key:    relToKey(relKey),
				IsDir:  true,
			}, nil
		}
		return nil, storage.ErrNotExist
	}

	// Read metadata from value log to get sizes.
	data, ct, created, updated, err := b.store.readValue(leaf.valueOffset, leaf.valueSize)
	if err != nil {
		return nil, err
	}

	return &storage.Object{
		Bucket:      b.name,
		Key:         relToKey(relKey),
		Size:        int64(len(data)),
		ContentType: ct,
		Created:     time.Unix(0, created),
		Updated:     time.Unix(0, updated),
	}, nil
}

func (b *bucket) Delete(ctx context.Context, key string, opts storage.Options) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	relKey, err := cleanKey(key)
	if err != nil {
		return err
	}

	recursive := boolOpt(opts, "recursive")

	if recursive {
		prefix := compositeKey(b.name, relKey)
		now := time.Now().UnixNano()

		var toDelete [][]byte
		b.store.tree.mu.RLock()
		artForEachPrefix(b.store.tree.root, prefix, func(leaf *leafData) {
			toDelete = append(toDelete, leaf.key)
		})
		b.store.tree.mu.RUnlock()

		if len(toDelete) == 0 {
			return storage.ErrNotExist
		}

		b.store.tree.mu.Lock()
		for _, k := range toDelete {
			artDelete(b.store.tree, k)
		}
		b.store.tree.mu.Unlock()

		for _, k := range toDelete {
			_ = b.store.appendWAL(walOpDelete, k, 0, 0, now)
		}
		return nil
	}

	compositeK := compositeKey(b.name, relKey)

	b.store.tree.mu.Lock()
	found := artDelete(b.store.tree, compositeK)
	b.store.tree.mu.Unlock()

	if !found {
		return storage.ErrNotExist
	}

	now := time.Now().UnixNano()
	return b.store.appendWAL(walOpDelete, compositeK, 0, 0, now)
}

func (b *bucket) Copy(ctx context.Context, dstKey string, srcBucket, srcKey string, opts storage.Options) (*storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	srcRelKey, err := cleanKey(srcKey)
	if err != nil {
		return nil, err
	}
	dstRelKey, err := cleanKey(dstKey)
	if err != nil {
		return nil, err
	}

	srcBucketName := safeBucketName(strings.TrimSpace(srcBucket))
	srcCK := compositeKey(srcBucketName, srcRelKey)

	b.store.tree.mu.RLock()
	srcLeaf := artSearch(b.store.tree.root, srcCK)
	b.store.tree.mu.RUnlock()

	if srcLeaf == nil {
		return nil, storage.ErrNotExist
	}

	// Read source value.
	data, ct, _, _, err := b.store.readValue(srcLeaf.valueOffset, srcLeaf.valueSize)
	if err != nil {
		return nil, err
	}

	// Write as new object.
	return b.Write(ctx, dstRelKey, bytes.NewReader(data), int64(len(data)), ct, opts)
}

func (b *bucket) Move(ctx context.Context, dstKey string, srcBucket, srcKey string, opts storage.Options) (*storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	obj, err := b.Copy(ctx, dstKey, srcBucket, srcKey, opts)
	if err != nil {
		return nil, err
	}

	// Delete source.
	srcRelKey, _ := cleanKey(srcKey)
	srcBucketName := safeBucketName(strings.TrimSpace(srcBucket))
	srcCK := compositeKey(srcBucketName, srcRelKey)

	b.store.tree.mu.Lock()
	artDelete(b.store.tree, srcCK)
	b.store.tree.mu.Unlock()

	now := time.Now().UnixNano()
	_ = b.store.appendWAL(walOpDelete, srcCK, 0, 0, now)

	return obj, nil
}

func (b *bucket) List(ctx context.Context, prefix string, limit, offset int, opts storage.Options) (storage.ObjectIter, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	recursive := true
	if v, ok := opts["recursive"].(bool); ok {
		recursive = v
	}

	relPrefix, err := cleanPrefix(prefix)
	if err != nil {
		return nil, err
	}

	searchPrefix := compositePrefix(b.name)
	if relPrefix != "" {
		searchPrefix = compositeKey(b.name, relPrefix)
	}

	var objects []*storage.Object
	b.store.tree.mu.RLock()
	artForEachPrefix(b.store.tree.root, searchPrefix, func(leaf *leafData) {
		_, objKey := splitCompositeKey(leaf.key)
		if objKey == "" {
			return
		}

		// Apply prefix filter.
		if relPrefix != "" {
			if !strings.HasPrefix(objKey, relPrefix) {
				return
			}
		}

		if !recursive {
			// Only include direct children (no deeper slashes after prefix).
			rest := objKey
			if relPrefix != "" {
				rest = strings.TrimPrefix(objKey, relPrefix)
				if len(rest) > 0 && rest[0] == '/' {
					rest = rest[1:]
				}
			}
			if strings.Contains(rest, "/") {
				// This is a directory entry. Check if we should add a dir marker.
				dirName := rest[:strings.Index(rest, "/")]
				dirKey := relPrefix
				if dirKey != "" {
					dirKey += "/"
				}
				dirKey += dirName

				// Check if we already have this dir in our list.
				found := false
				for _, o := range objects {
					if o.Key == dirKey && o.IsDir {
						found = true
						break
					}
				}
				if !found {
					objects = append(objects, &storage.Object{
						Bucket: b.name,
						Key:    dirKey,
						IsDir:  true,
					})
				}
				return
			}
		}

		// Compute value size from leaf metadata.
		valSize := computeValueSize(leaf.valueSize, leaf.contentType)

		objects = append(objects, &storage.Object{
			Bucket:      b.name,
			Key:         objKey,
			Size:        valSize,
			ContentType: leaf.contentType,
			Created:     time.Unix(0, leaf.created),
			Updated:     time.Unix(0, leaf.updated),
		})
	})
	b.store.tree.mu.RUnlock()

	sort.Slice(objects, func(i, j int) bool { return objects[i].Key < objects[j].Key })

	if offset < 0 {
		offset = 0
	}
	if offset > len(objects) {
		offset = len(objects)
	}
	objects = objects[offset:]
	if limit > 0 && limit < len(objects) {
		objects = objects[:limit]
	}

	return &objectIter{list: objects}, nil
}

func (b *bucket) SignedURL(ctx context.Context, key string, method string, expires time.Duration, opts storage.Options) (string, error) {
	return "", storage.ErrUnsupported
}

// computeValueSize extracts the actual value size from the total vlog entry size
// and the content type.
func computeValueSize(totalSize int64, contentType string) int64 {
	// totalSize = 2 + ctLen + 8 + valLen + 8 + 8
	ctLen := int64(len(contentType))
	overhead := int64(2 + ctLen + 8 + 8 + 8)
	valSize := totalSize - overhead
	if valSize < 0 {
		return 0
	}
	return valSize
}

// ---------------------------------------------------------------------------
// Directory support (storage.HasDirectories)
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

	searchPrefix := compositeKey(d.b.name, prefix)
	hasChildren := false

	d.b.store.tree.mu.RLock()
	artForEachPrefix(d.b.store.tree.root, searchPrefix, func(leaf *leafData) {
		hasChildren = true
	})
	d.b.store.tree.mu.RUnlock()

	if !hasChildren {
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

	searchPrefix := compositeKey(d.b.name, prefix)

	var objs []*storage.Object
	d.b.store.tree.mu.RLock()
	artForEachPrefix(d.b.store.tree.root, searchPrefix, func(leaf *leafData) {
		_, objKey := splitCompositeKey(leaf.key)
		rest := strings.TrimPrefix(objKey, prefix)
		if strings.Contains(rest, "/") {
			return // skip nested
		}
		valSize := computeValueSize(leaf.valueSize, leaf.contentType)
		objs = append(objs, &storage.Object{
			Bucket:      d.b.name,
			Key:         objKey,
			Size:        valSize,
			ContentType: leaf.contentType,
			Created:     time.Unix(0, leaf.created),
			Updated:     time.Unix(0, leaf.updated),
		})
	})
	d.b.store.tree.mu.RUnlock()

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

	return &objectIter{list: objs}, nil
}

func (d *dir) Delete(ctx context.Context, opts storage.Options) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	recursive := boolOpt(opts, "recursive")

	prefix := d.path
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	searchPrefix := compositeKey(d.b.name, prefix)

	var toDelete [][]byte
	d.b.store.tree.mu.RLock()
	artForEachPrefix(d.b.store.tree.root, searchPrefix, func(leaf *leafData) {
		if !recursive {
			_, objKey := splitCompositeKey(leaf.key)
			rest := strings.TrimPrefix(objKey, prefix)
			if strings.Contains(rest, "/") {
				return // skip nested if not recursive
			}
		}
		toDelete = append(toDelete, leaf.key)
	})
	d.b.store.tree.mu.RUnlock()

	if len(toDelete) == 0 {
		return storage.ErrNotExist
	}

	now := time.Now().UnixNano()

	d.b.store.tree.mu.Lock()
	for _, k := range toDelete {
		artDelete(d.b.store.tree, k)
	}
	d.b.store.tree.mu.Unlock()

	for _, k := range toDelete {
		_ = d.b.store.appendWAL(walOpDelete, k, 0, 0, now)
	}

	return nil
}

func (d *dir) Move(ctx context.Context, dstPath string, opts storage.Options) (storage.Directory, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	srcPrefix := d.path
	if srcPrefix != "" && !strings.HasSuffix(srcPrefix, "/") {
		srcPrefix += "/"
	}
	dstPrefix := strings.Trim(dstPath, "/")
	if dstPrefix != "" && !strings.HasSuffix(dstPrefix, "/") {
		dstPrefix += "/"
	}

	searchPrefix := compositeKey(d.b.name, srcPrefix)

	type moveEntry struct {
		oldKey []byte
		newKey string
		leaf   *leafData
	}

	var entries []moveEntry

	d.b.store.tree.mu.RLock()
	artForEachPrefix(d.b.store.tree.root, searchPrefix, func(leaf *leafData) {
		_, objKey := splitCompositeKey(leaf.key)
		rel := strings.TrimPrefix(objKey, srcPrefix)
		newObjKey := dstPrefix + rel
		entries = append(entries, moveEntry{
			oldKey: leaf.key,
			newKey: newObjKey,
			leaf:   leaf,
		})
	})
	d.b.store.tree.mu.RUnlock()

	if len(entries) == 0 {
		return nil, storage.ErrNotExist
	}

	now := time.Now().UnixNano()

	d.b.store.tree.mu.Lock()
	for _, e := range entries {
		newCK := compositeKey(d.b.name, e.newKey)
		newLeaf := &leafData{
			key:         newCK,
			valueOffset: e.leaf.valueOffset,
			valueSize:   e.leaf.valueSize,
			contentType: e.leaf.contentType,
			created:     e.leaf.created,
			updated:     now,
		}
		d.b.store.tree.root = artInsert(d.b.store.tree.root, newCK, newLeaf)
		d.b.store.tree.size++

		artDelete(d.b.store.tree, e.oldKey)
	}
	d.b.store.tree.mu.Unlock()

	for _, e := range entries {
		newCK := compositeKey(d.b.name, e.newKey)
		_ = d.b.store.appendWAL(walOpPut, newCK, e.leaf.valueOffset, e.leaf.valueSize, e.leaf.created)
		_ = d.b.store.appendWAL(walOpDelete, e.oldKey, 0, 0, now)
	}

	return &dir{b: d.b, path: strings.Trim(dstPath, "/")}, nil
}

// ---------------------------------------------------------------------------
// Multipart support (storage.HasMultipart)
// ---------------------------------------------------------------------------

var mpIDCounter atomic.Int64

func init() {
	mpIDCounter.Store(time.Now().UnixNano())
}

type multipartUpload struct {
	id          string
	key         string
	contentType string
	parts       map[int]*mpPart
	created     time.Time
	metadata    map[string]string
}

type mpPart struct {
	number int
	data   []byte
	size   int64
	etag   string
}

func (b *bucket) InitMultipart(ctx context.Context, key string, contentType string, opts storage.Options) (*storage.MultipartUpload, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	relKey, err := cleanKey(key)
	if err != nil {
		return nil, err
	}

	id := strconv.FormatInt(mpIDCounter.Add(1), 36)

	var metadata map[string]string
	if opts != nil {
		if m, ok := opts["metadata"].(map[string]string); ok {
			metadata = m
		}
	}

	upload := &multipartUpload{
		id:          id,
		key:         relKey,
		contentType: contentType,
		parts:       make(map[int]*mpPart),
		created:     time.Now(),
		metadata:    metadata,
	}

	b.mpMu.Lock()
	if b.mpUploads == nil {
		b.mpUploads = make(map[string]*multipartUpload)
	}
	b.mpUploads[id] = upload
	b.mpMu.Unlock()

	return &storage.MultipartUpload{
		Bucket:   b.name,
		Key:      relToKey(relKey),
		UploadID: id,
		Metadata: metadata,
	}, nil
}

func (b *bucket) UploadPart(ctx context.Context, mu *storage.MultipartUpload, number int, src io.Reader, size int64, opts storage.Options) (*storage.PartInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if number < 1 || number > 10000 {
		return nil, fmt.Errorf("ant: part number %d out of range [1, 10000]", number)
	}

	b.mpMu.RLock()
	upload, ok := b.mpUploads[mu.UploadID]
	b.mpMu.RUnlock()
	if !ok {
		return nil, storage.ErrNotExist
	}

	data, err := io.ReadAll(src)
	if err != nil {
		return nil, fmt.Errorf("ant: read part: %w", err)
	}

	hash := md5.Sum(data)
	etag := hex.EncodeToString(hash[:])

	b.mpMu.Lock()
	upload.parts[number] = &mpPart{
		number: number,
		data:   data,
		size:   int64(len(data)),
		etag:   etag,
	}
	b.mpMu.Unlock()

	return &storage.PartInfo{
		Number: number,
		Size:   int64(len(data)),
		ETag:   etag,
	}, nil
}

func (b *bucket) CopyPart(ctx context.Context, mu *storage.MultipartUpload, number int, opts storage.Options) (*storage.PartInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if number < 1 || number > 10000 {
		return nil, fmt.Errorf("ant: part number %d out of range", number)
	}

	b.mpMu.RLock()
	_, ok := b.mpUploads[mu.UploadID]
	b.mpMu.RUnlock()
	if !ok {
		return nil, storage.ErrNotExist
	}

	srcBucket := mu.Bucket
	if sb, ok := opts["source_bucket"].(string); ok && sb != "" {
		srcBucket = sb
	}
	srcKey, _ := opts["source_key"].(string)
	if srcKey == "" {
		return nil, errors.New("ant: source_key required for CopyPart")
	}
	srcOffset, _ := opts["source_offset"].(int64)
	srcLength, _ := opts["source_length"].(int64)

	srcRelKey, err := cleanKey(srcKey)
	if err != nil {
		return nil, err
	}
	srcCK := compositeKey(safeBucketName(srcBucket), srcRelKey)

	b.store.tree.mu.RLock()
	srcLeaf := artSearch(b.store.tree.root, srcCK)
	b.store.tree.mu.RUnlock()

	if srcLeaf == nil {
		return nil, storage.ErrNotExist
	}

	data, err := b.store.readValueOnly(srcLeaf.valueOffset, srcLeaf.valueSize)
	if err != nil {
		return nil, err
	}

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

	return b.UploadPart(ctx, mu, number, bytes.NewReader(data), int64(len(data)), opts)
}

func (b *bucket) ListParts(ctx context.Context, mu *storage.MultipartUpload, limit, offset int, opts storage.Options) ([]*storage.PartInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	b.mpMu.RLock()
	upload, ok := b.mpUploads[mu.UploadID]
	if !ok {
		b.mpMu.RUnlock()
		return nil, storage.ErrNotExist
	}

	parts := make([]*storage.PartInfo, 0, len(upload.parts))
	for _, p := range upload.parts {
		parts = append(parts, &storage.PartInfo{
			Number: p.number,
			Size:   p.size,
			ETag:   p.etag,
		})
	}
	b.mpMu.RUnlock()

	sort.Slice(parts, func(i, j int) bool { return parts[i].Number < parts[j].Number })

	if offset > 0 && offset < len(parts) {
		parts = parts[offset:]
	}
	if limit > 0 && limit < len(parts) {
		parts = parts[:limit]
	}

	return parts, nil
}

func (b *bucket) CompleteMultipart(ctx context.Context, mu *storage.MultipartUpload, parts []*storage.PartInfo, opts storage.Options) (*storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	b.mpMu.Lock()
	upload, ok := b.mpUploads[mu.UploadID]
	if !ok {
		b.mpMu.Unlock()
		return nil, storage.ErrNotExist
	}
	delete(b.mpUploads, mu.UploadID)
	b.mpMu.Unlock()

	// Sort and verify parts.
	sort.Slice(parts, func(i, j int) bool { return parts[i].Number < parts[j].Number })

	for _, p := range parts {
		if _, ok := upload.parts[p.Number]; !ok {
			return nil, fmt.Errorf("ant: part %d not found", p.Number)
		}
	}

	// Assemble final data.
	var totalSize int64
	for _, p := range parts {
		totalSize += upload.parts[p.Number].size
	}

	assembled := make([]byte, 0, totalSize)
	for _, p := range parts {
		assembled = append(assembled, upload.parts[p.Number].data...)
	}

	// Write as a single object.
	return b.Write(ctx, upload.key, bytes.NewReader(assembled), int64(len(assembled)), upload.contentType, opts)
}

func (b *bucket) AbortMultipart(ctx context.Context, mu *storage.MultipartUpload, opts storage.Options) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	b.mpMu.Lock()
	_, ok := b.mpUploads[mu.UploadID]
	if !ok {
		b.mpMu.Unlock()
		return storage.ErrNotExist
	}
	delete(b.mpUploads, mu.UploadID)
	b.mpMu.Unlock()

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

func (it *bucketIter) Close() error { return nil }

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

func (it *objectIter) Close() error { return nil }

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

func compositeKey(bucketName, key string) []byte {
	return []byte(bucketName + "\x00" + key)
}

func compositePrefix(bucketName string) []byte {
	return []byte(bucketName + "\x00")
}

func splitCompositeKey(ck []byte) (bucket, key string) {
	idx := bytes.IndexByte(ck, 0)
	if idx < 0 {
		return string(ck), ""
	}
	return string(ck[:idx]), string(ck[idx+1:])
}

func cleanKey(key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", errors.New("ant: empty key")
	}
	key = strings.ReplaceAll(key, "\\", "/")
	key = strings.TrimPrefix(key, "/")
	if key == "" {
		return "", errors.New("ant: empty key")
	}
	key = path.Clean(key)
	if key == "." {
		return "", errors.New("ant: empty key")
	}
	for _, part := range strings.Split(key, "/") {
		if part == ".." {
			return "", storage.ErrPermission
		}
	}
	return key, nil
}

func cleanPrefix(prefix string) (string, error) {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return "", nil
	}
	prefix = strings.ReplaceAll(prefix, "\\", "/")
	prefix = strings.TrimPrefix(prefix, "/")
	if prefix == "" {
		return "", nil
	}
	prefix = path.Clean(prefix)
	if prefix == "." {
		return "", nil
	}
	for _, part := range strings.Split(prefix, "/") {
		if part == ".." {
			return "", storage.ErrPermission
		}
	}
	return prefix, nil
}

func relToKey(rel string) string {
	return strings.TrimPrefix(strings.ReplaceAll(rel, "\\", "/"), "/")
}

func boolOpt(opts storage.Options, key string) bool {
	if opts == nil {
		return false
	}
	v, ok := opts[key]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	return ok && b
}
