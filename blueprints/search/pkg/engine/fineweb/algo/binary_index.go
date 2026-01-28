// Package algo provides binary serialization for fast index I/O.
package algo

import (
	"encoding/binary"
	"io"
	"math"
)

// BinaryWriter provides fast binary writing.
type BinaryWriter struct {
	buf []byte
}

// NewBinaryWriter creates a new binary writer.
func NewBinaryWriter() *BinaryWriter {
	return &BinaryWriter{buf: make([]byte, 0, 1024*1024)}
}

// WriteUint32 writes a uint32.
func (w *BinaryWriter) WriteUint32(v uint32) {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	w.buf = append(w.buf, b...)
}

// WriteUint16 writes a uint16.
func (w *BinaryWriter) WriteUint16(v uint16) {
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, v)
	w.buf = append(w.buf, b...)
}

// WriteUint64 writes a uint64.
func (w *BinaryWriter) WriteUint64(v uint64) {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, v)
	w.buf = append(w.buf, b...)
}

// WriteFloat32 writes a float32.
func (w *BinaryWriter) WriteFloat32(v float32) {
	w.WriteUint32(uint32FromFloat32(v))
}

// WriteFloat64 writes a float64.
func (w *BinaryWriter) WriteFloat64(v float64) {
	w.WriteUint64(uint64FromFloat64(v))
}

// WriteBytes writes a byte slice with length prefix.
func (w *BinaryWriter) WriteBytes(data []byte) {
	w.WriteUint32(uint32(len(data)))
	w.buf = append(w.buf, data...)
}

// WriteString writes a string with length prefix.
func (w *BinaryWriter) WriteString(s string) {
	w.WriteBytes([]byte(s))
}

// WriteUint32Slice writes a slice of uint32.
func (w *BinaryWriter) WriteUint32Slice(vals []uint32) {
	w.WriteUint32(uint32(len(vals)))
	for _, v := range vals {
		w.WriteUint32(v)
	}
}

// WriteUint16Slice writes a slice of uint16.
func (w *BinaryWriter) WriteUint16Slice(vals []uint16) {
	w.WriteUint32(uint32(len(vals)))
	for _, v := range vals {
		w.WriteUint16(v)
	}
}

// WriteIntSlice writes a slice of int.
func (w *BinaryWriter) WriteIntSlice(vals []int) {
	w.WriteUint32(uint32(len(vals)))
	for _, v := range vals {
		w.WriteUint32(uint32(v))
	}
}

// Bytes returns the written bytes.
func (w *BinaryWriter) Bytes() []byte {
	return w.buf
}

// WriteTo writes to an io.Writer.
func (w *BinaryWriter) WriteTo(writer io.Writer) (int64, error) {
	n, err := writer.Write(w.buf)
	return int64(n), err
}

// BinaryReader provides fast binary reading.
type BinaryReader struct {
	data []byte
	pos  int
}

// NewBinaryReader creates a new binary reader.
func NewBinaryReader(data []byte) *BinaryReader {
	return &BinaryReader{data: data}
}

// ReadUint32 reads a uint32.
func (r *BinaryReader) ReadUint32() uint32 {
	if r.pos+4 > len(r.data) {
		return 0
	}
	v := binary.LittleEndian.Uint32(r.data[r.pos:])
	r.pos += 4
	return v
}

// ReadUint16 reads a uint16.
func (r *BinaryReader) ReadUint16() uint16 {
	if r.pos+2 > len(r.data) {
		return 0
	}
	v := binary.LittleEndian.Uint16(r.data[r.pos:])
	r.pos += 2
	return v
}

// ReadUint64 reads a uint64.
func (r *BinaryReader) ReadUint64() uint64 {
	if r.pos+8 > len(r.data) {
		return 0
	}
	v := binary.LittleEndian.Uint64(r.data[r.pos:])
	r.pos += 8
	return v
}

// ReadFloat32 reads a float32.
func (r *BinaryReader) ReadFloat32() float32 {
	return float32FromUint32(r.ReadUint32())
}

// ReadFloat64 reads a float64.
func (r *BinaryReader) ReadFloat64() float64 {
	return float64FromUint64(r.ReadUint64())
}

// ReadBytes reads a byte slice.
func (r *BinaryReader) ReadBytes() []byte {
	length := int(r.ReadUint32())
	if r.pos+length > len(r.data) {
		return nil
	}
	data := make([]byte, length)
	copy(data, r.data[r.pos:r.pos+length])
	r.pos += length
	return data
}

// ReadString reads a string.
func (r *BinaryReader) ReadString() string {
	return string(r.ReadBytes())
}

// ReadUint32Slice reads a slice of uint32.
func (r *BinaryReader) ReadUint32Slice() []uint32 {
	length := int(r.ReadUint32())
	vals := make([]uint32, length)
	for i := range vals {
		vals[i] = r.ReadUint32()
	}
	return vals
}

// ReadUint16Slice reads a slice of uint16.
func (r *BinaryReader) ReadUint16Slice() []uint16 {
	length := int(r.ReadUint32())
	vals := make([]uint16, length)
	for i := range vals {
		vals[i] = r.ReadUint16()
	}
	return vals
}

// ReadIntSlice reads a slice of int.
func (r *BinaryReader) ReadIntSlice() []int {
	length := int(r.ReadUint32())
	vals := make([]int, length)
	for i := range vals {
		vals[i] = int(r.ReadUint32())
	}
	return vals
}

// Helper functions for float conversion
func uint32FromFloat32(f float32) uint32 {
	return math.Float32bits(f)
}

func float32FromUint32(u uint32) float32 {
	return math.Float32frombits(u)
}

func uint64FromFloat64(f float64) uint64 {
	return math.Float64bits(f)
}

func float64FromUint64(u uint64) float64 {
	return math.Float64frombits(u)
}
