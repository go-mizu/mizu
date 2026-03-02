// Package warcstore writes HTTP crawl results as WARC 1.1 files.
//
// Each URL gets one uncompressed .warc file containing three records:
//  1. warcinfo  — crawl session metadata
//  2. request   — HTTP GET request headers
//  3. response  — HTTP response (status line + headers + body)
//
// The WARC-Record-ID for the response record is a deterministic UUIDv5
// derived from the RFC 3986 canonical form of the URL. This ID is stored
// in DuckDB as warc_id and used to locate the file on disk:
//
//	warc/{hex[0:2]}/{hex[2:4]}/{hex[4:6]}/{uuid}.warc
//
// where hex = uuid with hyphens stripped.
//
// # Namespace UUIDs (fixed, for reproducibility)
//
//	Root     = UUIDv5(DNS_NS, "go-mizu.search.warc")
//	Response = UUIDv5(Root,   "response")  ← stored in warc_id
//	Request  = UUIDv5(Root,   "request")   ← WARC-Concurrent-To
//	Warcinfo = UUIDv5(Root,   "warcinfo")  ← per-file warcinfo record
//
// DNS_NS = 6ba7b810-9dad-11d1-80b4-00c04fd430c8 (RFC 4122).
package warcstore

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Namespace UUIDs — fixed for reproducibility.
var (
	// dnsNS is the DNS namespace UUID from RFC 4122.
	dnsNS = uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")

	// rootNS is UUIDv5(DNS_NS, "go-mizu.search.warc").
	rootNS = uuid.NewSHA1(dnsNS, []byte("go-mizu.search.warc"))

	// nsResponse is the namespace for response record IDs (stored as warc_id).
	nsResponse = uuid.NewSHA1(rootNS, []byte("response"))

	// nsRequest is the namespace for request record IDs (WARC-Concurrent-To).
	nsRequest = uuid.NewSHA1(rootNS, []byte("request"))

	// nsWarcinfo is the namespace for warcinfo record IDs.
	nsWarcinfo = uuid.NewSHA1(rootNS, []byte("warcinfo"))
)

// Store is a WARC 1.1 store backed by the filesystem.
// Each URL gets one uncompressed .warc file containing three records
// (warcinfo, request, response). The file is placed at:
//
//	{dir}/{hex[0:2]}/{hex[2:4]}/{hex[4:6]}/{uuid}.warc
//
// where uuid is the deterministic WARC-Record-ID of the response record
// and hex = uuid.String() with hyphens removed.
type Store struct{ dir string }

// Open returns a Store backed by dir, creating it if needed.
func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("warcstore: mkdir %s: %w", dir, err)
	}
	return &Store{dir: dir}, nil
}

// Entry holds all the data needed to write one WARC file.
type Entry struct {
	URL         string
	Proto       string      // "HTTP/1.1" (empty → use "HTTP/1.1")
	Method      string      // "GET" (empty → use "GET")
	ReqHeaders  http.Header
	StatusCode  int
	StatusText  string      // "200 OK" (empty → "{StatusCode} OK")
	RespHeaders http.Header
	Body        []byte
	IP          string      // empty → omit WARC-IP-Address
	CrawledAt   time.Time   // zero → time.Now()
	RunID       string      // for warcinfo isPartOf
}

// Put writes a WARC file for the given entry and returns the response record's
// WARC-Record-ID (warc_id). Put is idempotent: if the file already exists it
// returns immediately without error.
func (s *Store) Put(e Entry) (warcID string, err error) {
	canonical := CanonicalURL(e.URL)

	respUUID := uuid.NewSHA1(nsResponse, []byte(canonical))
	reqUUID := uuid.NewSHA1(nsRequest, []byte(canonical))
	infoUUID := uuid.NewSHA1(nsWarcinfo, []byte(canonical))

	warcID = respUUID.String()
	path := s.uuidToPath(warcID)

	// Idempotent: skip if file already exists.
	if _, err := os.Stat(path); err == nil {
		return warcID, nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("warcstore: mkdir: %w", err)
	}

	// Resolve defaults.
	proto := e.Proto
	if proto == "" {
		proto = "HTTP/1.1"
	}
	method := e.Method
	if method == "" {
		method = "GET"
	}
	statusText := e.StatusText
	if statusText == "" {
		statusText = fmt.Sprintf("%d OK", e.StatusCode)
	}
	crawledAt := e.CrawledAt
	if crawledAt.IsZero() {
		crawledAt = time.Now()
	}
	warcDate := crawledAt.UTC().Format("2006-01-02T15:04:05Z")

	// Relative path stored in WARC-Filename (strip leading dir).
	relPath := s.relPath(warcID)

	// ---- Build each record block ----

	// 1. warcinfo block
	var warcinfoBlock bytes.Buffer
	fmt.Fprintf(&warcinfoBlock, "software: go-mizu/warcstore\r\n")
	fmt.Fprintf(&warcinfoBlock, "format: WARC File Format 1.1\r\n")
	fmt.Fprintf(&warcinfoBlock, "conformsTo: https://iipc.github.io/warc-specifications/specifications/warc-format/warc-1.1/\r\n")
	if e.RunID != "" {
		fmt.Fprintf(&warcinfoBlock, "isPartOf: %s\r\n", e.RunID)
	}

	// 2. request block
	var reqBlock bytes.Buffer
	// Request line — use actual path+query from the canonical URL.
	reqPath := "/"
	if u, err2 := url.Parse(canonical); err2 == nil {
		if p := u.RequestURI(); p != "" {
			reqPath = p
		}
		fmt.Fprintf(&reqBlock, "%s %s %s\r\n", method, reqPath, proto)
		fmt.Fprintf(&reqBlock, "Host: %s\r\n", u.Host)
	} else {
		fmt.Fprintf(&reqBlock, "%s %s %s\r\n", method, reqPath, proto)
	}
	// Additional request headers
	writeHeaders(&reqBlock, e.ReqHeaders)
	// Blank line (no request body for GET)
	reqBlock.WriteString("\r\n")

	// 3. response block
	var respBlock bytes.Buffer
	// Status line
	fmt.Fprintf(&respBlock, "%s %s\r\n", proto, statusText)
	// Response headers
	writeHeaders(&respBlock, e.RespHeaders)
	// Blank line
	respBlock.WriteString("\r\n")
	// Body
	respBlock.Write(e.Body)

	// ---- Assemble full WARC file ----
	var buf bytes.Buffer

	// Record 1: warcinfo
	writeWARCRecord(&buf, warcRecordParams{
		typ:      "warcinfo",
		id:       infoUUID.String(),
		date:     warcDate,
		filename: relPath,
		ip:       "",
		block:    warcinfoBlock.Bytes(),
		contentType: "application/warc-fields",
	})

	buf.WriteString("\r\n\r\n")

	// Record 2: request
	writeWARCRecord(&buf, warcRecordParams{
		typ:          "request",
		id:           reqUUID.String(),
		date:         warcDate,
		targetURI:    canonical,
		concurrentTo: respUUID.String(),
		ip:           e.IP,
		block:        reqBlock.Bytes(),
		contentType:  "application/http;msgtype=request",
	})

	buf.WriteString("\r\n\r\n")

	// Record 3: response — WARC-Concurrent-To links back to the request record.
	writeWARCRecord(&buf, warcRecordParams{
		typ:          "response",
		id:           respUUID.String(),
		date:         warcDate,
		targetURI:    canonical,
		concurrentTo: reqUUID.String(),
		ip:           e.IP,
		block:        respBlock.Bytes(),
		contentType:  "application/http;msgtype=response",
	})

	buf.WriteString("\r\n\r\n")

	// ---- Atomic write ----
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0o644); err != nil {
		return "", fmt.Errorf("warcstore: write tmp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return "", fmt.Errorf("warcstore: rename: %w", err)
	}

	return warcID, nil
}

// Path returns the filesystem path for a warc_id without checking existence.
func (s *Store) Path(warcID string) string {
	return s.uuidToPath(warcID)
}

// CanonicalURL returns the RFC 3986 canonical form of rawURL:
//   - scheme and host are lowercased
//   - default ports (80 for http, 443 for https) are removed
//   - fragment is stripped
//   - path case is preserved
func CanonicalURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	u.Fragment = ""
	u.Scheme = strings.ToLower(u.Scheme)
	host := strings.ToLower(u.Hostname())
	port := u.Port()
	if (u.Scheme == "http" && port == "80") || (u.Scheme == "https" && port == "443") {
		port = ""
	}
	if port != "" {
		u.Host = host + ":" + port
	} else {
		u.Host = host
	}
	return u.String()
}

// uuidToPath converts a UUID string to a filesystem path under s.dir.
// hex = uuid with hyphens removed (32 chars).
// Path: {dir}/{hex[0:2]}/{hex[2:4]}/{hex[4:6]}/{uuid}.warc
func (s *Store) uuidToPath(warcID string) string {
	hex := strings.ReplaceAll(warcID, "-", "")
	return filepath.Join(s.dir, hex[0:2], hex[2:4], hex[4:6], warcID+".warc")
}

// relPath returns the path relative to s.dir (for WARC-Filename).
func (s *Store) relPath(warcID string) string {
	hex := strings.ReplaceAll(warcID, "-", "")
	return filepath.Join(hex[0:2], hex[2:4], hex[4:6], warcID+".warc")
}

// warcRecordParams holds the fields for a single WARC record.
type warcRecordParams struct {
	typ          string // WARC-Type
	id           string // UUID (without angle brackets / urn:uuid: prefix)
	date         string // WARC-Date
	targetURI    string // WARC-Target-URI (omitted for warcinfo)
	concurrentTo string // WARC-Concurrent-To UUID (omitted if empty)
	filename     string // WARC-Filename (warcinfo only)
	ip           string // WARC-IP-Address (omitted if empty)
	block        []byte
	contentType  string
}

// writeWARCRecord serialises one WARC record into buf.
// All header lines end with \r\n; the blank line after headers is \r\n.
func writeWARCRecord(buf *bytes.Buffer, p warcRecordParams) {
	buf.WriteString("WARC/1.1\r\n")
	fmt.Fprintf(buf, "WARC-Type: %s\r\n", p.typ)
	fmt.Fprintf(buf, "WARC-Date: %s\r\n", p.date)
	fmt.Fprintf(buf, "WARC-Record-ID: <urn:uuid:%s>\r\n", p.id)
	if p.targetURI != "" {
		fmt.Fprintf(buf, "WARC-Target-URI: %s\r\n", p.targetURI)
	}
	if p.filename != "" {
		fmt.Fprintf(buf, "WARC-Filename: %s\r\n", p.filename)
	}
	if p.concurrentTo != "" {
		fmt.Fprintf(buf, "WARC-Concurrent-To: <urn:uuid:%s>\r\n", p.concurrentTo)
	}
	if p.ip != "" {
		fmt.Fprintf(buf, "WARC-IP-Address: %s\r\n", p.ip)
	}
	fmt.Fprintf(buf, "Content-Type: %s\r\n", p.contentType)
	fmt.Fprintf(buf, "Content-Length: %d\r\n", len(p.block))
	buf.WriteString("\r\n") // blank line between WARC headers and block
	buf.Write(p.block)
}

// writeHeaders writes HTTP headers to w in a canonical sorted order,
// each line ending with \r\n.
func writeHeaders(buf *bytes.Buffer, h http.Header) {
	if h == nil {
		return
	}
	// Sort header names for determinism.
	names := make([]string, 0, len(h))
	for k := range h {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, k := range names {
		for _, v := range h[k] {
			fmt.Fprintf(buf, "%s: %s\r\n", k, v)
		}
	}
}
