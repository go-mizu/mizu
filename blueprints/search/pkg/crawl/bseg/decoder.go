package bseg

import (
	"bufio"
	"io"
)

// FileHeader holds the decoded .bseg2 file header fields.
type FileHeader struct {
	Version  uint8
	Flags    uint8
	RecCount uint32
}

// Decoder reads Records from an io.Reader in the .bseg2 binary format.
// Call NewDecoder to validate the file header before reading records.
type Decoder struct {
	r      *bufio.Reader
	hdr    FileHeader
	recBuf []byte // scratch buffer — grows as needed
}

// NewDecoder creates a Decoder that reads from r.
// It reads and validates the 10-byte file header. Returns ErrBadMagic or
// ErrBadVersion on a malformed file.
func NewDecoder(r io.Reader) (*Decoder, error) {
	br, ok := r.(*bufio.Reader)
	if !ok {
		br = bufio.NewReaderSize(r, 512*1024)
	}

	var hdrBuf [hdrSize]byte
	if _, err := io.ReadFull(br, hdrBuf[:]); err != nil {
		return nil, err
	}

	if string(hdrBuf[0:4]) != Magic {
		return nil, ErrBadMagic
	}
	version := hdrBuf[4]
	if version != Version {
		return nil, ErrBadVersion
	}

	hdr := FileHeader{
		Version:  version,
		Flags:    hdrBuf[5],
		RecCount: le.Uint32(hdrBuf[6:10]),
	}

	return &Decoder{
		r:      br,
		hdr:    hdr,
		recBuf: make([]byte, 0, 512),
	}, nil
}

// Header returns the decoded file header (Version, Flags, RecCount).
func (d *Decoder) Header() FileHeader { return d.hdr }

// Decode reads the next Record into r.
// Returns io.EOF when there are no more records.
// Returns io.ErrUnexpectedEOF or another error on a corrupt file.
func (d *Decoder) Decode(r *Record) error {
	// Read 4-byte rec_len prefix.
	var lenBuf [4]byte
	if _, err := io.ReadFull(d.r, lenBuf[:]); err != nil {
		// io.ReadFull returns io.ErrUnexpectedEOF on a partial read,
		// but io.EOF when zero bytes are read — which is the clean end-of-file.
		if err == io.EOF {
			return io.EOF
		}
		return err
	}

	recLen := int(le.Uint32(lenBuf[:]))
	if recLen < recFixedSize {
		return io.ErrUnexpectedEOF
	}

	// Grow scratch buffer if needed.
	if cap(d.recBuf) < recLen {
		d.recBuf = make([]byte, recLen)
	}
	buf := d.recBuf[:recLen]

	if _, err := io.ReadFull(d.r, buf); err != nil {
		return err
	}

	// Parse flags (byte 0 of the record body).
	flags := buf[0]
	r.Failed = (flags & 1) != 0

	// Fixed numeric fields.
	r.StatusCode = int32(le.Uint32(buf[1:5]))
	r.ContentLen = int64(le.Uint64(buf[5:13]))
	r.FetchMs = int64(le.Uint64(buf[13:21]))
	r.CrawledMs = int64(le.Uint64(buf[21:29]))

	// String fields starting at offset 29 within the record body.
	off := 29
	var err error
	if r.URL, off, err = readStr(buf, off); err != nil {
		return err
	}
	if r.ContentType, off, err = readStr(buf, off); err != nil {
		return err
	}
	if r.BodyCID, off, err = readStr(buf, off); err != nil {
		return err
	}
	if r.Title, off, err = readStr(buf, off); err != nil {
		return err
	}
	if r.Description, off, err = readStr(buf, off); err != nil {
		return err
	}
	if r.Language, off, err = readStr(buf, off); err != nil {
		return err
	}
	if r.Domain, off, err = readStr(buf, off); err != nil {
		return err
	}
	if r.RedirectURL, off, err = readStr(buf, off); err != nil {
		return err
	}
	if r.Error, _, err = readStr(buf, off); err != nil {
		return err
	}

	return nil
}
