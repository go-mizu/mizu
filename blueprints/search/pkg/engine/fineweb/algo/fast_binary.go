// Package algo provides fast binary encoding without reflection.
package algo

import (
	"io"
)

// FastBinaryWriter provides zero-allocation binary writing.
type FastBinaryWriter struct {
	w   io.Writer
	buf [8]byte // Reusable buffer for small writes
	err error
}

// NewFastBinaryWriter creates a fast binary writer.
func NewFastBinaryWriter(w io.Writer) *FastBinaryWriter {
	return &FastBinaryWriter{w: w}
}

// WriteUint16 writes a uint16 in little-endian.
func (f *FastBinaryWriter) WriteUint16(v uint16) {
	if f.err != nil {
		return
	}
	f.buf[0] = byte(v)
	f.buf[1] = byte(v >> 8)
	_, f.err = f.w.Write(f.buf[:2])
}

// WriteUint32 writes a uint32 in little-endian.
func (f *FastBinaryWriter) WriteUint32(v uint32) {
	if f.err != nil {
		return
	}
	f.buf[0] = byte(v)
	f.buf[1] = byte(v >> 8)
	f.buf[2] = byte(v >> 16)
	f.buf[3] = byte(v >> 24)
	_, f.err = f.w.Write(f.buf[:4])
}

// WriteInt64 writes an int64 in little-endian.
func (f *FastBinaryWriter) WriteInt64(v int64) {
	if f.err != nil {
		return
	}
	f.buf[0] = byte(v)
	f.buf[1] = byte(v >> 8)
	f.buf[2] = byte(v >> 16)
	f.buf[3] = byte(v >> 24)
	f.buf[4] = byte(v >> 32)
	f.buf[5] = byte(v >> 40)
	f.buf[6] = byte(v >> 48)
	f.buf[7] = byte(v >> 56)
	_, f.err = f.w.Write(f.buf[:8])
}

// WriteBytes writes raw bytes.
func (f *FastBinaryWriter) WriteBytes(b []byte) {
	if f.err != nil {
		return
	}
	_, f.err = f.w.Write(b)
}

// Err returns any error that occurred during writing.
func (f *FastBinaryWriter) Err() error {
	return f.err
}

// FastBinaryReader provides fast binary reading from a byte slice.
type FastBinaryReader struct {
	data []byte
	pos  int
}

// NewFastBinaryReader creates a reader from a byte slice.
func NewFastBinaryReader(data []byte) *FastBinaryReader {
	return &FastBinaryReader{data: data}
}

// ReadUint16 reads a uint16 in little-endian.
func (f *FastBinaryReader) ReadUint16() uint16 {
	if f.pos+2 > len(f.data) {
		return 0
	}
	v := uint16(f.data[f.pos]) | uint16(f.data[f.pos+1])<<8
	f.pos += 2
	return v
}

// ReadUint32 reads a uint32 in little-endian.
func (f *FastBinaryReader) ReadUint32() uint32 {
	if f.pos+4 > len(f.data) {
		return 0
	}
	v := uint32(f.data[f.pos]) |
		uint32(f.data[f.pos+1])<<8 |
		uint32(f.data[f.pos+2])<<16 |
		uint32(f.data[f.pos+3])<<24
	f.pos += 4
	return v
}

// ReadInt64 reads an int64 in little-endian.
func (f *FastBinaryReader) ReadInt64() int64 {
	if f.pos+8 > len(f.data) {
		return 0
	}
	v := int64(f.data[f.pos]) |
		int64(f.data[f.pos+1])<<8 |
		int64(f.data[f.pos+2])<<16 |
		int64(f.data[f.pos+3])<<24 |
		int64(f.data[f.pos+4])<<32 |
		int64(f.data[f.pos+5])<<40 |
		int64(f.data[f.pos+6])<<48 |
		int64(f.data[f.pos+7])<<56
	f.pos += 8
	return v
}

// ReadBytes reads n bytes.
func (f *FastBinaryReader) ReadBytes(n int) []byte {
	if f.pos+n > len(f.data) {
		return nil
	}
	b := f.data[f.pos : f.pos+n]
	f.pos += n
	return b
}

// Skip skips n bytes.
func (f *FastBinaryReader) Skip(n int) {
	f.pos += n
}

// Remaining returns the number of bytes remaining.
func (f *FastBinaryReader) Remaining() int {
	return len(f.data) - f.pos
}

// Pos returns the current position.
func (f *FastBinaryReader) Pos() int {
	return f.pos
}

// BatchPostingWriter writes postings in batches for better I/O efficiency.
type BatchPostingWriter struct {
	w      io.Writer
	buf    []byte
	bufPos int
}

// NewBatchPostingWriter creates a batched posting writer.
func NewBatchPostingWriter(w io.Writer, bufSize int) *BatchPostingWriter {
	return &BatchPostingWriter{
		w:   w,
		buf: make([]byte, bufSize),
	}
}

// WritePosting writes a docID and freq.
func (b *BatchPostingWriter) WritePosting(docID uint32, freq uint16) error {
	if b.bufPos+6 > len(b.buf) {
		if err := b.Flush(); err != nil {
			return err
		}
	}
	b.buf[b.bufPos] = byte(docID)
	b.buf[b.bufPos+1] = byte(docID >> 8)
	b.buf[b.bufPos+2] = byte(docID >> 16)
	b.buf[b.bufPos+3] = byte(docID >> 24)
	b.buf[b.bufPos+4] = byte(freq)
	b.buf[b.bufPos+5] = byte(freq >> 8)
	b.bufPos += 6
	return nil
}

// Flush writes buffered data.
func (b *BatchPostingWriter) Flush() error {
	if b.bufPos == 0 {
		return nil
	}
	_, err := b.w.Write(b.buf[:b.bufPos])
	b.bufPos = 0
	return err
}
