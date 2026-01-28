package algo

import (
	"encoding/binary"
	"math/bits"
	"sort"
)

// GobEncode implements gob.GobEncoder.
func (rb *RoaringBitmap) GobEncode() ([]byte, error) {
	return rb.Serialize(), nil
}

// GobDecode implements gob.GobDecoder.
func (rb *RoaringBitmap) GobDecode(data []byte) error {
	decoded := DeserializeRoaring(data)
	*rb = *decoded
	return nil
}

// RoaringBitmap implements a hybrid bitmap structure for efficient integer sets.
// Uses array containers for sparse sets and bitmap containers for dense sets.
// Reference: https://arxiv.org/abs/1709.07821

const (
	containerSize     = 1 << 16 // 65536 integers per container
	arrayThreshold    = 4096    // Switch to bitmap above this
	runThreshold      = 2048    // Use run encoding if beneficial
)

// RoaringBitmap is a compressed bitmap for storing sorted integers.
type RoaringBitmap struct {
	keys       []uint16      // High 16 bits of container
	containers []container   // Container for each key
}

// container interface for different container types
type container interface {
	add(x uint16) container
	contains(x uint16) bool
	cardinality() int
	iterator() containerIterator
	toArray() []uint16
	serialize() []byte
	containerType() byte
}

type containerIterator interface {
	hasNext() bool
	next() uint16
}

// arrayContainer stores values in a sorted array
type arrayContainer struct {
	values []uint16
}

// bitmapContainer stores values in a 64-bit bitmap array
type bitmapContainer struct {
	bitmap     [1024]uint64 // 64 * 1024 = 65536 bits
	card       int          // Cached cardinality
}

// NewRoaringBitmap creates an empty roaring bitmap.
func NewRoaringBitmap() *RoaringBitmap {
	return &RoaringBitmap{}
}

// Add adds an integer to the bitmap.
func (rb *RoaringBitmap) Add(x uint32) {
	high := uint16(x >> 16)
	low := uint16(x & 0xFFFF)

	idx := rb.findContainer(high)
	if idx < len(rb.keys) && rb.keys[idx] == high {
		rb.containers[idx] = rb.containers[idx].add(low)
	} else {
		// Insert new container
		rb.keys = append(rb.keys, 0)
		rb.containers = append(rb.containers, nil)

		// Shift elements
		copy(rb.keys[idx+1:], rb.keys[idx:])
		copy(rb.containers[idx+1:], rb.containers[idx:])

		rb.keys[idx] = high
		rb.containers[idx] = (&arrayContainer{}).add(low)
	}
}

// AddMany adds multiple integers efficiently.
func (rb *RoaringBitmap) AddMany(values []uint32) {
	// Sort values for efficient insertion
	sorted := make([]uint32, len(values))
	copy(sorted, values)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	for _, v := range sorted {
		rb.Add(v)
	}
}

// Contains checks if x is in the bitmap.
func (rb *RoaringBitmap) Contains(x uint32) bool {
	high := uint16(x >> 16)
	low := uint16(x & 0xFFFF)

	idx := rb.findContainer(high)
	if idx < len(rb.keys) && rb.keys[idx] == high {
		return rb.containers[idx].contains(low)
	}
	return false
}

// Cardinality returns the number of integers in the bitmap.
func (rb *RoaringBitmap) Cardinality() int {
	total := 0
	for _, c := range rb.containers {
		total += c.cardinality()
	}
	return total
}

// findContainer returns the index where high should be or is.
func (rb *RoaringBitmap) findContainer(high uint16) int {
	return sort.Search(len(rb.keys), func(i int) bool {
		return rb.keys[i] >= high
	})
}

// ToArray returns all integers as a slice.
func (rb *RoaringBitmap) ToArray() []uint32 {
	result := make([]uint32, 0, rb.Cardinality())

	for i, key := range rb.keys {
		high := uint32(key) << 16
		for _, low := range rb.containers[i].toArray() {
			result = append(result, high|uint32(low))
		}
	}

	return result
}

// And computes the intersection of two bitmaps.
func (rb *RoaringBitmap) And(other *RoaringBitmap) *RoaringBitmap {
	result := NewRoaringBitmap()

	i, j := 0, 0
	for i < len(rb.keys) && j < len(other.keys) {
		if rb.keys[i] < other.keys[j] {
			i++
		} else if rb.keys[i] > other.keys[j] {
			j++
		} else {
			// Same key - intersect containers
			c := intersectContainers(rb.containers[i], other.containers[j])
			if c.cardinality() > 0 {
				result.keys = append(result.keys, rb.keys[i])
				result.containers = append(result.containers, c)
			}
			i++
			j++
		}
	}

	return result
}

// Or computes the union of two bitmaps.
func (rb *RoaringBitmap) Or(other *RoaringBitmap) *RoaringBitmap {
	result := NewRoaringBitmap()

	i, j := 0, 0
	for i < len(rb.keys) || j < len(other.keys) {
		if i >= len(rb.keys) {
			result.keys = append(result.keys, other.keys[j:]...)
			result.containers = append(result.containers, other.containers[j:]...)
			break
		}
		if j >= len(other.keys) {
			result.keys = append(result.keys, rb.keys[i:]...)
			result.containers = append(result.containers, rb.containers[i:]...)
			break
		}

		if rb.keys[i] < other.keys[j] {
			result.keys = append(result.keys, rb.keys[i])
			result.containers = append(result.containers, rb.containers[i])
			i++
		} else if rb.keys[i] > other.keys[j] {
			result.keys = append(result.keys, other.keys[j])
			result.containers = append(result.containers, other.containers[j])
			j++
		} else {
			// Same key - union containers
			c := unionContainers(rb.containers[i], other.containers[j])
			result.keys = append(result.keys, rb.keys[i])
			result.containers = append(result.containers, c)
			i++
			j++
		}
	}

	return result
}

// Iterator returns an iterator over all integers.
func (rb *RoaringBitmap) Iterator() *RoaringIterator {
	return &RoaringIterator{rb: rb}
}

// RoaringIterator iterates over a roaring bitmap.
type RoaringIterator struct {
	rb            *RoaringBitmap
	containerIdx  int
	containerIter containerIterator
}

// HasNext returns true if there are more values.
func (it *RoaringIterator) HasNext() bool {
	for it.containerIdx < len(it.rb.containers) {
		if it.containerIter == nil {
			it.containerIter = it.rb.containers[it.containerIdx].iterator()
		}
		if it.containerIter.hasNext() {
			return true
		}
		it.containerIdx++
		it.containerIter = nil
	}
	return false
}

// Next returns the next value.
func (it *RoaringIterator) Next() uint32 {
	high := uint32(it.rb.keys[it.containerIdx]) << 16
	low := uint32(it.containerIter.next())
	return high | low
}

// Serialize writes the bitmap to bytes.
func (rb *RoaringBitmap) Serialize() []byte {
	// Format: numContainers + (key, type, data)*
	size := 4 // numContainers
	for i := range rb.containers {
		size += 2 + 1 + 4 + len(rb.containers[i].serialize()) // key + type + len + data
	}

	buf := make([]byte, size)
	pos := 0

	binary.LittleEndian.PutUint32(buf[pos:], uint32(len(rb.containers)))
	pos += 4

	for i, c := range rb.containers {
		binary.LittleEndian.PutUint16(buf[pos:], rb.keys[i])
		pos += 2

		buf[pos] = c.containerType()
		pos++

		data := c.serialize()
		binary.LittleEndian.PutUint32(buf[pos:], uint32(len(data)))
		pos += 4

		copy(buf[pos:], data)
		pos += len(data)
	}

	return buf
}

// DeserializeRoaring reads a roaring bitmap from bytes.
func DeserializeRoaring(data []byte) *RoaringBitmap {
	if len(data) < 4 {
		return NewRoaringBitmap()
	}

	rb := NewRoaringBitmap()
	pos := 0

	numContainers := int(binary.LittleEndian.Uint32(data[pos:]))
	pos += 4

	rb.keys = make([]uint16, numContainers)
	rb.containers = make([]container, numContainers)

	for i := 0; i < numContainers; i++ {
		rb.keys[i] = binary.LittleEndian.Uint16(data[pos:])
		pos += 2

		ctype := data[pos]
		pos++

		dataLen := int(binary.LittleEndian.Uint32(data[pos:]))
		pos += 4

		cdata := data[pos : pos+dataLen]
		pos += dataLen

		switch ctype {
		case 0: // array
			rb.containers[i] = deserializeArrayContainer(cdata)
		case 1: // bitmap
			rb.containers[i] = deserializeBitmapContainer(cdata)
		}
	}

	return rb
}

// arrayContainer implementation

func (ac *arrayContainer) add(x uint16) container {
	idx := sort.Search(len(ac.values), func(i int) bool {
		return ac.values[i] >= x
	})

	if idx < len(ac.values) && ac.values[idx] == x {
		return ac // Already exists
	}

	// Check if we should convert to bitmap
	if len(ac.values) >= arrayThreshold {
		bc := &bitmapContainer{}
		for _, v := range ac.values {
			bc.add(v)
		}
		return bc.add(x)
	}

	// Insert into sorted array
	ac.values = append(ac.values, 0)
	copy(ac.values[idx+1:], ac.values[idx:])
	ac.values[idx] = x

	return ac
}

func (ac *arrayContainer) contains(x uint16) bool {
	idx := sort.Search(len(ac.values), func(i int) bool {
		return ac.values[i] >= x
	})
	return idx < len(ac.values) && ac.values[idx] == x
}

func (ac *arrayContainer) cardinality() int {
	return len(ac.values)
}

func (ac *arrayContainer) toArray() []uint16 {
	return ac.values
}

func (ac *arrayContainer) serialize() []byte {
	buf := make([]byte, len(ac.values)*2)
	for i, v := range ac.values {
		binary.LittleEndian.PutUint16(buf[i*2:], v)
	}
	return buf
}

func (ac *arrayContainer) containerType() byte { return 0 }

func (ac *arrayContainer) iterator() containerIterator {
	return &arrayIterator{values: ac.values}
}

type arrayIterator struct {
	values []uint16
	pos    int
}

func (it *arrayIterator) hasNext() bool { return it.pos < len(it.values) }
func (it *arrayIterator) next() uint16 {
	v := it.values[it.pos]
	it.pos++
	return v
}

func deserializeArrayContainer(data []byte) *arrayContainer {
	values := make([]uint16, len(data)/2)
	for i := range values {
		values[i] = binary.LittleEndian.Uint16(data[i*2:])
	}
	return &arrayContainer{values: values}
}

// bitmapContainer implementation

func (bc *bitmapContainer) add(x uint16) container {
	idx := x / 64
	bit := x % 64

	if (bc.bitmap[idx] & (1 << bit)) == 0 {
		bc.bitmap[idx] |= 1 << bit
		bc.card++
	}

	return bc
}

func (bc *bitmapContainer) contains(x uint16) bool {
	idx := x / 64
	bit := x % 64
	return (bc.bitmap[idx] & (1 << bit)) != 0
}

func (bc *bitmapContainer) cardinality() int {
	return bc.card
}

func (bc *bitmapContainer) toArray() []uint16 {
	result := make([]uint16, 0, bc.card)
	for i, word := range bc.bitmap {
		if word == 0 {
			continue
		}
		for j := 0; j < 64; j++ {
			if (word & (1 << j)) != 0 {
				result = append(result, uint16(i*64+j))
			}
		}
	}
	return result
}

func (bc *bitmapContainer) serialize() []byte {
	buf := make([]byte, 4+1024*8) // card + bitmap
	binary.LittleEndian.PutUint32(buf, uint32(bc.card))
	for i, word := range bc.bitmap {
		binary.LittleEndian.PutUint64(buf[4+i*8:], word)
	}
	return buf
}

func (bc *bitmapContainer) containerType() byte { return 1 }

func (bc *bitmapContainer) iterator() containerIterator {
	return &bitmapIterator{bitmap: &bc.bitmap}
}

type bitmapIterator struct {
	bitmap   *[1024]uint64
	wordIdx  int
	bitIdx   int
	current  uint64
}

func (it *bitmapIterator) hasNext() bool {
	for it.wordIdx < 1024 {
		if it.current == 0 {
			it.current = it.bitmap[it.wordIdx]
			it.bitIdx = 0
		}
		if it.current != 0 {
			return true
		}
		it.wordIdx++
	}
	return false
}

func (it *bitmapIterator) next() uint16 {
	// Find next set bit
	tz := bits.TrailingZeros64(it.current)
	result := uint16(it.wordIdx*64 + tz)

	// Clear this bit
	it.current &= it.current - 1

	if it.current == 0 {
		it.wordIdx++
	}

	return result
}

func deserializeBitmapContainer(data []byte) *bitmapContainer {
	bc := &bitmapContainer{}
	bc.card = int(binary.LittleEndian.Uint32(data))
	for i := 0; i < 1024; i++ {
		bc.bitmap[i] = binary.LittleEndian.Uint64(data[4+i*8:])
	}
	return bc
}

// Container operations

func intersectContainers(a, b container) container {
	arrA := a.toArray()
	arrB := b.toArray()

	result := &arrayContainer{}

	i, j := 0, 0
	for i < len(arrA) && j < len(arrB) {
		if arrA[i] < arrB[j] {
			i++
		} else if arrA[i] > arrB[j] {
			j++
		} else {
			result.values = append(result.values, arrA[i])
			i++
			j++
		}
	}

	return result
}

func unionContainers(a, b container) container {
	arrA := a.toArray()
	arrB := b.toArray()

	// Estimate result size
	if len(arrA)+len(arrB) > arrayThreshold {
		// Use bitmap
		bc := &bitmapContainer{}
		for _, v := range arrA {
			bc.add(v)
		}
		for _, v := range arrB {
			bc.add(v)
		}
		return bc
	}

	// Merge arrays
	result := &arrayContainer{values: make([]uint16, 0, len(arrA)+len(arrB))}

	i, j := 0, 0
	for i < len(arrA) && j < len(arrB) {
		if arrA[i] < arrB[j] {
			result.values = append(result.values, arrA[i])
			i++
		} else if arrA[i] > arrB[j] {
			result.values = append(result.values, arrB[j])
			j++
		} else {
			result.values = append(result.values, arrA[i])
			i++
			j++
		}
	}

	result.values = append(result.values, arrA[i:]...)
	result.values = append(result.values, arrB[j:]...)

	return result
}
