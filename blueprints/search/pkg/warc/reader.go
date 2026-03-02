package warc

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"strings"
)

const (
	gzipMagic1 = byte(0x1f)
	gzipMagic2 = byte(0x8b)
	bufSize    = 64 * 1024
)

// Reader iterates WARC records from any io.Reader.
// It auto-detects whether the stream is gzip-compressed.
//
// Usage:
//
//	r := warc.NewReader(f)
//	for r.Next() {
//	    rec := r.Record()
//	    // process rec.Header and rec.Body
//	    // Body must be drained before calling Next() again
//	    io.Copy(io.Discard, rec.Body)
//	}
//	if err := r.Err(); err != nil { ... }
type Reader struct {
	raw    io.Reader     // original source (buffered)
	gz     *gzip.Reader  // non-nil for gzip input
	buf    *bufio.Reader // reader for current (possibly decompressed) data
	rec    *Record
	err    error
	isGzip bool
	first  bool             // true until the first record has been read (gzip mode)
	body   *io.LimitedReader // tracks unconsumed body bytes for auto-drain
}

// NewReader creates a Reader from any io.Reader.
// If the stream starts with gzip magic bytes (0x1f 0x8b), it is treated as a
// concatenated-gzip WARC.GZ file. Otherwise treated as plain WARC text.
func NewReader(r io.Reader) *Reader {
	raw := bufio.NewReaderSize(r, bufSize)

	// Peek at first 2 bytes to detect gzip
	peek, _ := raw.Peek(2)
	if len(peek) >= 2 && peek[0] == gzipMagic1 && peek[1] == gzipMagic2 {
		gz, err := gzip.NewReader(raw)
		if err != nil {
			return &Reader{err: fmt.Errorf("warc: gzip init: %w", err)}
		}
		gz.Multistream(false) // each WARC record is its own gzip member
		return &Reader{
			raw:    raw,
			gz:     gz,
			buf:    bufio.NewReaderSize(gz, bufSize),
			isGzip: true,
			first:  true,
		}
	}

	return &Reader{
		raw: raw,
		buf: bufio.NewReaderSize(raw, bufSize),
	}
}

// Next advances to the next WARC record.
// Returns false at EOF or on error. Call Err() to distinguish.
func (r *Reader) Next() bool {
	if r.err != nil {
		return false
	}
	// Drain unconsumed body from previous record
	if r.body != nil && r.body.N > 0 {
		io.Copy(io.Discard, r.body)
	}
	r.body = nil
	r.rec = nil

	if r.isGzip {
		if r.first {
			// First call: gz already points at member 0; just use it.
			r.first = false
		} else {
			// Drain any remaining bytes in current member (e.g., trailing \r\n\r\n)
			io.Copy(io.Discard, r.buf)
			// Advance to the next gzip member
			if err := r.gz.Reset(r.raw); err != nil {
				if err == io.EOF {
					return false
				}
				r.err = fmt.Errorf("warc: gzip member reset: %w", err)
				return false
			}
			r.gz.Multistream(false)
			r.buf.Reset(r.gz)
		}
	} else {
		// For plain WARC: skip record separator (\r\n\r\n between records)
		// by reading until we see "WARC/" at the start of a line.
		// (The separator is already consumed if body was fully read.)
	}

	rec, body, err := readRecord(r.buf)
	if err != nil {
		if err == io.EOF {
			return false
		}
		r.err = fmt.Errorf("warc: reading record: %w", err)
		return false
	}
	r.rec = rec
	r.body = body
	return true
}

// Record returns the current record. Valid only after Next() returns true.
func (r *Reader) Record() *Record { return r.rec }

// Err returns the first non-EOF error encountered during iteration.
func (r *Reader) Err() error { return r.err }

// readRecord parses one WARC record from buf.
// Returns the Record and the LimitedReader tracking body bytes.
func readRecord(buf *bufio.Reader) (*Record, *io.LimitedReader, error) {
	// 1. Skip blank lines before record (handles \r\n\r\n separators in plain WARC)
	var versionLine string
	for {
		line, err := buf.ReadString('\n')
		if err != nil {
			return nil, nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			continue // skip blank separator lines
		}
		versionLine = line
		break
	}

	if !strings.HasPrefix(versionLine, "WARC/") {
		return nil, nil, fmt.Errorf("warc: expected WARC/ version line, got %q", versionLine)
	}

	// 2. Parse WARC headers (key: value pairs, terminated by blank line)
	hdr := make(Header)
	for {
		line, err := buf.ReadString('\n')
		if err != nil {
			return nil, nil, fmt.Errorf("warc: reading headers: %w", err)
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break // blank line ends WARC headers
		}
		k, v, ok := strings.Cut(line, ": ")
		if !ok {
			k, v, ok = strings.Cut(line, ":")
			if !ok {
				continue // skip malformed header
			}
			v = strings.TrimSpace(v)
		}
		hdr[k] = v
	}

	// 3. Body: exactly Content-Length bytes (0 if not specified)
	bodyLen := hdr.ContentLength()
	lim := &io.LimitedReader{R: buf, N: bodyLen}

	rec := &Record{
		Header: hdr,
		Body:   lim,
	}
	return rec, lim, nil
}
