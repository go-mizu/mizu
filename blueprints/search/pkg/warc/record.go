// Package warc provides streaming read/write access to WARC 1.1 files.
// It auto-detects gzip compression and handles concatenated-gzip WARC.GZ
// (each WARC record in its own gzip member, as used by Common Crawl).
package warc

import (
	"io"
	"strconv"
	"strings"
	"time"
)

// Record-type constants (WARC-Type header values).
const (
	TypeWARCInfo     = "warcinfo"
	TypeResponse     = "response"
	TypeResource     = "resource"
	TypeRequest      = "request"
	TypeMetadata     = "metadata"
	TypeRevisit      = "revisit"
	TypeConversion   = "conversion"
	TypeContinuation = "continuation"
)

// Header holds WARC header fields as key-value pairs.
// Keys are stored in the original WARC canonical form (e.g., "WARC-Type").
type Header map[string]string

// Get returns the header value for the given key (case-insensitive lookup).
func (h Header) Get(key string) string {
	if v, ok := h[key]; ok {
		return v
	}
	// Fallback: case-insensitive search
	low := strings.ToLower(key)
	for k, v := range h {
		if strings.ToLower(k) == low {
			return v
		}
	}
	return ""
}

// Type returns the WARC-Type value.
func (h Header) Type() string { return h.Get("WARC-Type") }

// TargetURI returns the WARC-Target-URI value.
func (h Header) TargetURI() string { return h.Get("WARC-Target-URI") }

// Date parses and returns the WARC-Date value.
func (h Header) Date() time.Time {
	v := h.Get("WARC-Date")
	if v == "" {
		return time.Time{}
	}
	t, _ := time.Parse(time.RFC3339, v)
	if t.IsZero() {
		t, _ = time.Parse("2006-01-02T15:04:05Z", v)
	}
	return t
}

// ContentLength parses and returns the Content-Length value.
func (h Header) ContentLength() int64 {
	n, _ := strconv.ParseInt(h.Get("Content-Length"), 10, 64)
	return n
}

// RecordID returns the WARC-Record-ID value.
func (h Header) RecordID() string { return h.Get("WARC-Record-ID") }

// RefersTo returns the WARC-Refers-To value (used by revisit records).
func (h Header) RefersTo() string { return h.Get("WARC-Refers-To") }

// Record represents a single WARC record.
// Body must be fully consumed before calling Reader.Next().
// Unconsumed body bytes are drained automatically by the next Next() call.
type Record struct {
	Header Header
	Body   io.Reader
}
