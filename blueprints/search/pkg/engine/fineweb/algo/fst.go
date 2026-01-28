package algo

import (
	"encoding/binary"
	"sort"
)

// GobEncode implements gob.GobEncoder.
func (fst *FST) GobEncode() ([]byte, error) {
	return fst.Serialize(), nil
}

// GobDecode implements gob.GobDecoder.
func (fst *FST) GobDecode(data []byte) error {
	decoded := DeserializeFST(data)
	*fst = *decoded
	return nil
}

// FST implements a simplified Finite State Transducer for term dictionary.
// This provides space-efficient storage with O(term_length) lookup.
// Based on Lucene's FST implementation concepts.

// FSTBuilder builds an FST from sorted term-value pairs.
type FSTBuilder struct {
	terms  []string
	values []uint64
}

// NewFSTBuilder creates a new FST builder.
func NewFSTBuilder() *FSTBuilder {
	return &FSTBuilder{}
}

// Add adds a term-value pair. Terms must be added in sorted order.
func (b *FSTBuilder) Add(term string, value uint64) {
	b.terms = append(b.terms, term)
	b.values = append(b.values, value)
}

// Build creates the FST from added terms.
func (b *FSTBuilder) Build() *FST {
	if len(b.terms) == 0 {
		return &FST{root: &fstNode{}}
	}

	// Sort if not already sorted
	if !sort.StringsAreSorted(b.terms) {
		// Create index for sorting
		indices := make([]int, len(b.terms))
		for i := range indices {
			indices[i] = i
		}
		sort.Slice(indices, func(i, j int) bool {
			return b.terms[indices[i]] < b.terms[indices[j]]
		})

		newTerms := make([]string, len(b.terms))
		newValues := make([]uint64, len(b.values))
		for i, idx := range indices {
			newTerms[i] = b.terms[idx]
			newValues[i] = b.values[idx]
		}
		b.terms = newTerms
		b.values = newValues
	}

	// Build trie structure
	fst := &FST{root: &fstNode{}}

	for i, term := range b.terms {
		fst.insert(term, b.values[i])
	}

	// Compact the trie (merge single-child chains)
	fst.compact(fst.root)

	return fst
}

// FST is a Finite State Transducer for term lookup.
type FST struct {
	root *fstNode
}

type fstNode struct {
	children map[byte]*fstNode
	label    []byte  // For compacted nodes (path compression)
	value    uint64  // Output value (0 means no value)
	hasValue bool    // Whether this node has a value
}

func (fst *FST) insert(term string, value uint64) {
	node := fst.root

	for i := 0; i < len(term); i++ {
		c := term[i]

		if node.children == nil {
			node.children = make(map[byte]*fstNode)
		}

		child, exists := node.children[c]
		if !exists {
			child = &fstNode{}
			node.children[c] = child
		}
		node = child
	}

	node.value = value
	node.hasValue = true
}

func (fst *FST) compact(node *fstNode) {
	if node == nil {
		return
	}

	// First compact all children
	for _, child := range node.children {
		fst.compact(child)
	}

	// Merge single-child chains
	for len(node.children) == 1 && !node.hasValue {
		for c, child := range node.children {
			// Merge child into node
			node.label = append(node.label, c)
			node.label = append(node.label, child.label...)
			node.children = child.children
			node.value = child.value
			node.hasValue = child.hasValue
			break
		}
	}
}

// Get looks up a term and returns its value.
func (fst *FST) Get(term string) (uint64, bool) {
	node := fst.root
	pos := 0

	for pos < len(term) {
		if node == nil {
			return 0, false
		}

		// Check path-compressed label
		if len(node.label) > 0 {
			if pos+len(node.label) > len(term) {
				return 0, false
			}
			for i, c := range node.label {
				if term[pos+i] != c {
					return 0, false
				}
			}
			pos += len(node.label)
			continue
		}

		// Follow edge
		c := term[pos]
		if node.children == nil {
			return 0, false
		}
		child, exists := node.children[c]
		if !exists {
			return 0, false
		}
		node = child
		pos++
	}

	// Check if we need to match remaining label
	if len(node.label) > 0 {
		return 0, false
	}

	if node.hasValue {
		return node.value, true
	}
	return 0, false
}

// PrefixSearch returns all terms starting with prefix.
func (fst *FST) PrefixSearch(prefix string) []TermValue {
	var results []TermValue

	// Navigate to prefix node
	node := fst.root
	pos := 0

	for pos < len(prefix) {
		if node == nil {
			return nil
		}

		// Check path-compressed label
		if len(node.label) > 0 {
			matchLen := min(len(node.label), len(prefix)-pos)
			for i := 0; i < matchLen; i++ {
				if node.label[i] != prefix[pos+i] {
					return nil
				}
			}
			pos += len(node.label)
			continue
		}

		c := prefix[pos]
		if node.children == nil {
			return nil
		}
		child, exists := node.children[c]
		if !exists {
			return nil
		}
		node = child
		pos++
	}

	// Collect all terms under this node
	fst.collectTerms(node, prefix, &results)

	return results
}

func (fst *FST) collectTerms(node *fstNode, prefix string, results *[]TermValue) {
	if node == nil {
		return
	}

	currentPrefix := prefix
	if len(node.label) > 0 {
		currentPrefix += string(node.label)
	}

	if node.hasValue {
		*results = append(*results, TermValue{Term: currentPrefix, Value: node.value})
	}

	for c, child := range node.children {
		fst.collectTerms(child, currentPrefix+string(c), results)
	}
}

// TermValue holds a term-value pair.
type TermValue struct {
	Term  string
	Value uint64
}

// Size returns approximate memory usage in bytes.
func (fst *FST) Size() int {
	return fst.nodeSize(fst.root)
}

func (fst *FST) nodeSize(node *fstNode) int {
	if node == nil {
		return 0
	}

	size := 48 + len(node.label) // Base node size + label
	for _, child := range node.children {
		size += 1 + fst.nodeSize(child) // Edge byte + child
	}
	return size
}

// Serialize writes the FST to bytes.
func (fst *FST) Serialize() []byte {
	// Simple serialization: collect all terms and values
	var terms []TermValue
	fst.collectTerms(fst.root, "", &terms)

	// Calculate size
	size := 4 // Number of terms
	for _, tv := range terms {
		size += 4 + len(tv.Term) + 8 // len + term + value
	}

	buf := make([]byte, size)
	pos := 0

	binary.LittleEndian.PutUint32(buf[pos:], uint32(len(terms)))
	pos += 4

	for _, tv := range terms {
		binary.LittleEndian.PutUint32(buf[pos:], uint32(len(tv.Term)))
		pos += 4
		copy(buf[pos:], tv.Term)
		pos += len(tv.Term)
		binary.LittleEndian.PutUint64(buf[pos:], tv.Value)
		pos += 8
	}

	return buf
}

// DeserializeFST reads an FST from bytes.
func DeserializeFST(data []byte) *FST {
	if len(data) < 4 {
		return &FST{root: &fstNode{}}
	}

	pos := 0
	numTerms := int(binary.LittleEndian.Uint32(data[pos:]))
	pos += 4

	builder := NewFSTBuilder()

	for i := 0; i < numTerms; i++ {
		termLen := int(binary.LittleEndian.Uint32(data[pos:]))
		pos += 4
		term := string(data[pos : pos+termLen])
		pos += termLen
		value := binary.LittleEndian.Uint64(data[pos:])
		pos += 8

		builder.Add(term, value)
	}

	return builder.Build()
}
