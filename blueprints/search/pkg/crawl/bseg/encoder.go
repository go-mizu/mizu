package bseg

import (
	"bufio"
	"os"
)

// Encoder writes Records to an *os.File in the .bseg2 binary format.
// On Close(), it seeks back to byte offset 6 and patches the uint32 record count.
type Encoder struct {
	f      *os.File
	w      *bufio.Writer
	count  uint32
	recBuf []byte // scratch buffer — grows as needed, avoids per-record alloc
}

// NewEncoder creates an Encoder that writes to f and uses a bufio.Writer of bufSize bytes.
// A 10-byte file header is written immediately. bufSize <= 0 defaults to 512 KB.
func NewEncoder(f *os.File, bufSize int) (*Encoder, error) {
	if bufSize <= 0 {
		bufSize = 512 * 1024
	}
	e := &Encoder{
		f:      f,
		w:      bufio.NewWriterSize(f, bufSize),
		recBuf: make([]byte, 0, 512),
	}

	// Write 10-byte file header.
	var hdr [hdrSize]byte
	copy(hdr[0:4], Magic)
	hdr[4] = Version
	hdr[5] = 0 // flags (reserved)
	putUint32(hdr[6:10], 0) // rec_count placeholder; patched on Close()
	if _, err := e.w.Write(hdr[:]); err != nil {
		return nil, err
	}
	return e, nil
}

// Encode serializes r and writes it to the underlying writer.
// rec_len is computed before any bytes are written, so a single write suffices.
func (e *Encoder) Encode(r *Record) error {
	// Compute rec_len = fixed portion + all 9 string fields.
	recLen := recFixedSize +
		strEncLen(r.URL) +
		strEncLen(r.ContentType) +
		strEncLen(r.BodyCID) +
		strEncLen(r.Title) +
		strEncLen(r.Description) +
		strEncLen(r.Language) +
		strEncLen(r.Domain) +
		strEncLen(r.RedirectURL) +
		strEncLen(r.Error)

	// Total bytes = 4 (rec_len field) + recLen.
	total := 4 + recLen

	// Grow scratch buffer if needed.
	if cap(e.recBuf) < total {
		e.recBuf = make([]byte, total)
	}
	b := e.recBuf[:total]

	// rec_len field (4 bytes).
	putUint32(b[0:4], uint32(recLen))

	// flags (1 byte): bit 0 = Failed.
	var flags uint8
	if r.Failed {
		flags |= 1
	}
	b[4] = flags

	// Fixed numeric fields.
	putInt32(b[5:9], r.StatusCode)
	putInt64(b[9:17], r.ContentLen)
	putInt64(b[17:25], r.FetchMs)
	putInt64(b[25:33], r.CrawledMs)

	// String fields starting at offset 33.
	tail := b[33:]
	tail = appendStr(tail[:0], r.URL)
	tail = appendStr(tail, r.ContentType)
	tail = appendStr(tail, r.BodyCID)
	tail = appendStr(tail, r.Title)
	tail = appendStr(tail, r.Description)
	tail = appendStr(tail, r.Language)
	tail = appendStr(tail, r.Domain)
	tail = appendStr(tail, r.RedirectURL)
	tail = appendStr(tail, r.Error)
	_ = tail

	if _, err := e.w.Write(b); err != nil {
		return err
	}
	e.count++
	return nil
}

// Close flushes the bufio.Writer, patches the rec_count field in the file header,
// and closes the underlying file. After Close(), the Encoder must not be used.
func (e *Encoder) Close() error {
	if err := e.w.Flush(); err != nil {
		_ = e.f.Close()
		return err
	}
	// Patch rec_count at byte offset 6 in the file.
	var buf [4]byte
	putUint32(buf[:], e.count)
	if _, err := e.f.WriteAt(buf[:], recCountOffset); err != nil {
		_ = e.f.Close()
		return err
	}
	return e.f.Close()
}
