// Package bseg implements the .bseg2 binary segment file format for the crawl bin writer.
// Format is faster than encoding/gob (no reflection, no type descriptors).
//
// # File Layout
//
//	[0:4]  magic      = "BSEG"  (4 bytes, ASCII)
//	[4]    version    = 2       (uint8)
//	[5]    flags      = 0       (uint8, reserved)
//	[6:10] rec_count  = uint32  (LE — placeholder, patched on Close())
//
// Per-record:
//
//	[0:4]   rec_len     (uint32 LE — byte length of everything after this field)
//	[4]     flags       (uint8 — bit 0: Failed=1)
//	[5:9]   status_code (int32 LE)
//	[9:17]  content_len (int64 LE)
//	[17:25] fetch_ms    (int64 LE)
//	[25:33] crawled_ms  (int64 LE)
//	then 9 string fields, each as: uint16 LE length + bytes (truncated at 65535)
//	  url, content_type, body_cid, title, description, language, domain, redirect_url, error
package bseg

import (
	"encoding/binary"
	"errors"
	"io"
)

const (
	// Magic is the 4-byte file magic identifier.
	Magic = "BSEG"
	// Version is the current format version.
	Version = 2
	// hdrSize is the total file header size in bytes.
	hdrSize = 10
	// recCountOffset is the byte offset of the record count field in the header.
	recCountOffset = 6
	// recFixedSize is the fixed-width portion of each record (after rec_len field):
	// flags(1) + status_code(4) + content_len(8) + fetch_ms(8) + crawled_ms(8) = 29
	recFixedSize = 29
	// maxStrLen is the maximum length of a single string field in bytes.
	maxStrLen = 65535
)

// ErrBadMagic is returned by NewDecoder when the file does not start with "BSEG".
var ErrBadMagic = errors.New("bseg: bad magic (not a .bseg2 file)")

// ErrBadVersion is returned by NewDecoder when the format version is not supported.
var ErrBadVersion = errors.New("bseg: unsupported version")

// Record holds one decoded/encoded crawl result record.
type Record struct {
	Failed      bool
	StatusCode  int32
	ContentLen  int64
	FetchMs     int64
	CrawledMs   int64
	URL         string
	ContentType string
	BodyCID     string
	Title       string
	Description string
	Language    string
	Domain      string
	RedirectURL string
	Error       string
}

// le is a shorthand for binary.LittleEndian.
var le = binary.LittleEndian

// putUint16 writes v as a 2-byte LE integer to buf.
func putUint16(buf []byte, v uint16) {
	le.PutUint16(buf, v)
}

// putUint32 writes v as a 4-byte LE integer to buf.
func putUint32(buf []byte, v uint32) {
	le.PutUint32(buf, v)
}

// putInt32 writes v as a 4-byte LE two's-complement integer to buf.
func putInt32(buf []byte, v int32) {
	le.PutUint32(buf, uint32(v))
}

// putInt64 writes v as an 8-byte LE two's-complement integer to buf.
func putInt64(buf []byte, v int64) {
	le.PutUint64(buf, uint64(v))
}

// strEncLen returns the encoded byte length of a string field: 2 (uint16 length) + min(len(s), maxStrLen).
func strEncLen(s string) int {
	n := len(s)
	if n > maxStrLen {
		n = maxStrLen
	}
	return 2 + n
}

// appendStr appends the encoded form of s (uint16 length + clamped bytes) to buf.
func appendStr(buf []byte, s string) []byte {
	n := len(s)
	if n > maxStrLen {
		n = maxStrLen
	}
	var tmp [2]byte
	putUint16(tmp[:], uint16(n))
	buf = append(buf, tmp[:]...)
	buf = append(buf, s[:n]...)
	return buf
}

// readStr reads a length-prefixed string from r. Returns io.ErrUnexpectedEOF on short read.
func readStr(buf []byte, off int) (string, int, error) {
	if off+2 > len(buf) {
		return "", off, io.ErrUnexpectedEOF
	}
	n := int(le.Uint16(buf[off:]))
	off += 2
	if off+n > len(buf) {
		return "", off, io.ErrUnexpectedEOF
	}
	s := string(buf[off : off+n])
	return s, off + n, nil
}
