package warc

import (
	"bufio"
	"fmt"
	"io"
	"time"
)

// Writer writes WARC 1.1 records to an io.Writer.
// Caller controls compression (wrap w with gzip.NewWriter if needed).
type Writer struct {
	w   *bufio.Writer
	err error
}

// NewWriter creates a Writer that writes to w.
func NewWriter(w io.Writer) *Writer {
	return &Writer{w: bufio.NewWriterSize(w, 64*1024)}
}

// WriteRecord writes a single WARC record.
// If rec.Header does not contain WARC-Date, the current time is used.
// If rec.Header does not contain Content-Length, it must be set by caller
// (or the body must implement io.Seeker for auto-detection).
// The record separator (\r\n\r\n) is written after the body.
func (w *Writer) WriteRecord(rec *Record) error {
	if w.err != nil {
		return w.err
	}

	// Ensure WARC-Date
	if rec.Header.Get("WARC-Date") == "" {
		if rec.Header == nil {
			rec.Header = make(Header)
		}
		rec.Header["WARC-Date"] = time.Now().UTC().Format(time.RFC3339)
	}

	// Write version line
	if _, err := fmt.Fprintf(w.w, "WARC/1.1\r\n"); err != nil {
		w.err = err
		return err
	}

	// Write headers
	for k, v := range rec.Header {
		if _, err := fmt.Fprintf(w.w, "%s: %s\r\n", k, v); err != nil {
			w.err = err
			return err
		}
	}

	// Blank line after headers
	if _, err := fmt.Fprintf(w.w, "\r\n"); err != nil {
		w.err = err
		return err
	}

	// Write body
	if rec.Body != nil {
		if _, err := io.Copy(w.w, rec.Body); err != nil {
			w.err = fmt.Errorf("warc: writing body: %w", err)
			return w.err
		}
	}

	// Record separator
	if _, err := fmt.Fprintf(w.w, "\r\n\r\n"); err != nil {
		w.err = err
		return err
	}

	return nil
}

// Close flushes any buffered data. Must be called after all records are written.
func (w *Writer) Close() error {
	if w.err != nil {
		return w.err
	}
	return w.w.Flush()
}
