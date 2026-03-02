package warc

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ImportOptions controls which WARC records are imported.
type ImportOptions struct {
	// RecordTypes filters by WARC-Type; nil or empty = accept all.
	RecordTypes []string
	// MIMETypes filters HTTP response Content-Type; nil or empty = accept all.
	MIMETypes []string
	// StatusCodes filters HTTP response status; nil or empty = accept all.
	StatusCodes []int
	// MaxBodySize caps body storage in bytes; 0 = no limit.
	MaxBodySize int64
	// BatchSize is the DuckDB insert batch size; default 1000.
	BatchSize int
	// WARCFile is the filename tag stored in WARCRecord.WARCFile.
	WARCFile string
}

// ImportStats reports accumulated progress.
type ImportStats struct {
	Read     int64         // WARC records scanned
	Imported int64         // records stored in DuckDB
	Skipped  int64         // records filtered out
	Bytes    int64         // body bytes stored
	Elapsed  time.Duration
}

// Importer streams a Reader into a RecordDB.
type Importer struct {
	db   *RecordDB
	opts ImportOptions
}

// NewImporter creates an Importer that writes to db.
func NewImporter(db *RecordDB, opts ImportOptions) *Importer {
	if opts.BatchSize <= 0 {
		opts.BatchSize = 1000
	}
	return &Importer{db: db, opts: opts}
}

// ProcessRecord applies ImportOptions filtering to a single WARC record.
// Returns the extracted WARCRecord and whether it was accepted by the filters.
// If not accepted, the record's Body has been fully consumed.
func ProcessRecord(rec *Record, opts ImportOptions) (WARCRecord, bool) {
	im := &Importer{opts: opts}
	return im.processRecord(rec)
}

// Import reads all records from r and imports matching ones to the RecordDB.
// fn is called periodically with progress stats (may be nil).
func (im *Importer) Import(ctx context.Context, r *Reader, fn func(ImportStats)) error {
	start := time.Now()
	var stats ImportStats
	var batch []WARCRecord

	flush := func() {
		if len(batch) > 0 {
			im.db.Insert(batch)
			batch = batch[:0]
		}
	}

	for r.Next() {
		select {
		case <-ctx.Done():
			flush()
			return ctx.Err()
		default:
		}

		rec := r.Record()
		stats.Read++

		// Filter record type
		if !im.acceptType(rec.Header.Type()) {
			io.Copy(io.Discard, rec.Body)
			stats.Skipped++
			if fn != nil && stats.Read%1000 == 0 {
				stats.Elapsed = time.Since(start)
				fn(stats)
			}
			continue
		}

		wr, skip := im.processRecord(rec)
		if skip {
			stats.Skipped++
			if fn != nil && stats.Read%1000 == 0 {
				stats.Elapsed = time.Since(start)
				fn(stats)
			}
			continue
		}

		stats.Imported++
		stats.Bytes += int64(len(wr.Body))
		batch = append(batch, wr)

		if len(batch) >= im.opts.BatchSize {
			flush()
		}

		if fn != nil && stats.Read%1000 == 0 {
			stats.Elapsed = time.Since(start)
			fn(stats)
		}
	}

	flush()
	if err := r.Err(); err != nil {
		return err
	}

	if fn != nil {
		stats.Elapsed = time.Since(start)
		fn(stats)
	}
	return nil
}

func (im *Importer) acceptType(t string) bool {
	if len(im.opts.RecordTypes) == 0 {
		return true
	}
	for _, rt := range im.opts.RecordTypes {
		if t == rt {
			return true
		}
	}
	return false
}

// processRecord extracts a WARCRecord from a WARC response record.
// Returns (record, skip=true) if the record should be filtered out.
func (im *Importer) processRecord(rec *Record) (WARCRecord, bool) {
	wr := WARCRecord{
		WARCFile:   im.opts.WARCFile,
		RecordID:   rec.Header.RecordID(),
		RecordType: rec.Header.Type(),
		URL:        rec.Header.TargetURI(),
		CrawledAt:  rec.Header.Date(),
	}

	// Extract domain from URL
	if u, err := url.Parse(wr.URL); err == nil {
		wr.Domain = u.Hostname()
	}

	// For non-response records, read body as-is
	if rec.Header.Type() != TypeResponse {
		body, _ := im.readBody(rec.Body)
		wr.Body = body
		wr.BodyLength = int64(len(body))
		return wr, false
	}

	// Parse HTTP response block
	bodyReader := bufio.NewReader(rec.Body)
	// Read HTTP status line
	statusLine, err := bodyReader.ReadString('\n')
	if err != nil {
		io.Copy(io.Discard, rec.Body)
		return wr, true
	}
	wr.HTTPStatus = parseHTTPStatus(strings.TrimRight(statusLine, "\r\n"))

	// Filter by status code
	if !im.acceptStatus(wr.HTTPStatus) {
		io.Copy(io.Discard, rec.Body)
		return wr, true
	}

	// Read HTTP headers
	tp := textproto.NewReader(bodyReader)
	httpHdrs, _ := tp.ReadMIMEHeader()

	// Extract MIME type (strip params)
	ct := httpHdrs.Get("Content-Type")
	wr.MIMEType = strings.SplitN(ct, ";", 2)[0]
	wr.MIMEType = strings.TrimSpace(wr.MIMEType)

	// Filter by MIME type
	if !im.acceptMIME(wr.MIMEType) {
		io.Copy(io.Discard, rec.Body)
		return wr, true
	}

	// Encode HTTP headers as JSON
	if len(httpHdrs) > 0 {
		m := make(map[string]string, len(httpHdrs))
		for k := range httpHdrs {
			m[k] = httpHdrs.Get(k)
		}
		if b, err := json.Marshal(m); err == nil {
			wr.HTTPHeaders = string(b)
		}
	}

	// Read body
	remaining, _ := im.readBody(bodyReader)
	remaining = bytes.TrimRight(remaining, "\r\n")

	wr.Body = remaining
	wr.BodyLength = int64(len(remaining))
	wr.Language = detectLanguageFromHeaders(httpHdrs)

	// Extract title/description from HTML
	if strings.Contains(wr.MIMEType, "html") {
		wr.Title, wr.Description = extractPageInfo(remaining)
	}

	return wr, false
}

func (im *Importer) readBody(r io.Reader) ([]byte, error) {
	if im.opts.MaxBodySize > 0 {
		return io.ReadAll(io.LimitReader(r, im.opts.MaxBodySize))
	}
	return io.ReadAll(r)
}

func (im *Importer) acceptStatus(code int) bool {
	if len(im.opts.StatusCodes) == 0 {
		return true
	}
	for _, c := range im.opts.StatusCodes {
		if c == code {
			return true
		}
	}
	return false
}

func (im *Importer) acceptMIME(mime string) bool {
	if len(im.opts.MIMETypes) == 0 {
		return true
	}
	for _, m := range im.opts.MIMETypes {
		if strings.Contains(mime, m) || strings.Contains(m, mime) {
			return true
		}
	}
	return false
}

func parseHTTPStatus(line string) int {
	parts := strings.SplitN(line, " ", 3)
	if len(parts) < 2 {
		return 0
	}
	n, _ := strconv.Atoi(parts[1])
	return n
}

func detectLanguageFromHeaders(h textproto.MIMEHeader) string {
	// e.g., Content-Language: en, de
	return h.Get("Content-Language")
}

func extractPageInfo(body []byte) (title, desc string) {
	s := string(body)
	lower := strings.ToLower(s)
	// Title: search in s directly (tags are ASCII, positions match)
	if i := strings.Index(s, "<title"); i >= 0 {
		if j := strings.Index(s[i:], ">"); j >= 0 {
			start := i + j + 1
			// Use lower[start:] for case-insensitive end search, clamp for safety
			sub := lower[min(start, len(lower)):]
			if end := strings.Index(sub, "</title>"); end >= 0 {
				safeEnd := min(start+end, len(s))
				title = strings.TrimSpace(s[start:safeEnd])
				if len(title) > 500 {
					title = title[:500]
				}
			}
		}
	}
	// Meta description: pass lower (not s) so attrPos indices stay consistent
	for _, attr := range []string{`name="description"`, `name='description'`, `property="og:description"`} {
		if idx := strings.Index(lower, attr); idx >= 0 {
			desc = extractMetaContent(lower, idx)
			if desc != "" {
				break
			}
		}
	}
	return
}

// extractMetaContent extracts the content="..." value from the region around attrPos.
// s must already be lowercased (pass lower from extractPageInfo).
func extractMetaContent(s string, attrPos int) string {
	if attrPos > len(s) {
		return ""
	}
	region := s[max(attrPos-200, 0):min(attrPos+500, len(s))]
	if i := strings.Index(region, `content="`); i >= 0 {
		start := i + 9
		if start >= len(region) {
			return ""
		}
		if end := strings.Index(region[start:], `"`); end >= 0 {
			v := strings.TrimSpace(region[start : start+end])
			if len(v) > 1000 {
				v = v[:1000]
			}
			return v
		}
	}
	return ""
}
