package warc_test

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/blueprints/search/pkg/warc"
)

// buildWARCRecord creates a minimal WARC response record string.
func buildWARCRecord(targetURI, body string) string {
	httpBody := "HTTP/1.1 200 OK\r\nContent-Type: text/html\r\n\r\n" + body
	return "WARC/1.1\r\n" +
		"WARC-Type: response\r\n" +
		"WARC-Target-URI: " + targetURI + "\r\n" +
		"WARC-Date: 2026-01-15T12:00:00Z\r\n" +
		"WARC-Record-ID: <urn:uuid:test-123>\r\n" +
		"Content-Length: " + fmt.Sprintf("%d", len(httpBody)) + "\r\n" +
		"\r\n" +
		httpBody + "\r\n\r\n"
}

// gzipWARC gzip-compresses each WARC record in its own gzip member.
func gzipWARC(records []string) []byte {
	var buf bytes.Buffer
	for _, rec := range records {
		gz := gzip.NewWriter(&buf)
		gz.Write([]byte(rec))
		gz.Close()
	}
	return buf.Bytes()
}

func TestReader_PlainWARC_SingleRecord(t *testing.T) {
	raw := buildWARCRecord("https://example.com/", "<html>hello</html>")
	r := warc.NewReader(strings.NewReader(raw))

	if !r.Next() {
		t.Fatalf("expected Next()=true, err=%v", r.Err())
	}
	rec := r.Record()
	if rec.Header.Type() != warc.TypeResponse {
		t.Errorf("Type = %q, want %q", rec.Header.Type(), warc.TypeResponse)
	}
	if rec.Header.TargetURI() != "https://example.com/" {
		t.Errorf("TargetURI = %q", rec.Header.TargetURI())
	}
	body, _ := io.ReadAll(rec.Body)
	if !bytes.Contains(body, []byte("hello")) {
		t.Errorf("body missing expected content, got %q", body)
	}
	if r.Next() {
		t.Error("expected Next()=false after single record")
	}
	if err := r.Err(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestReader_PlainWARC_MultiRecord(t *testing.T) {
	rec1 := buildWARCRecord("https://a.com/", "page A")
	rec2 := buildWARCRecord("https://b.com/", "page B")
	r := warc.NewReader(strings.NewReader(rec1 + rec2))

	count := 0
	for r.Next() {
		count++
		io.Copy(io.Discard, r.Record().Body)
	}
	if err := r.Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 records, got %d", count)
	}
}

func TestReader_GzipWARC_MultiRecord(t *testing.T) {
	records := []string{
		buildWARCRecord("https://a.com/", "page A"),
		buildWARCRecord("https://b.com/", "page B"),
		buildWARCRecord("https://c.com/", "page C"),
	}
	data := gzipWARC(records)

	r := warc.NewReader(bytes.NewReader(data))
	count := 0
	var uris []string
	for r.Next() {
		uris = append(uris, r.Record().Header.TargetURI())
		io.Copy(io.Discard, r.Record().Body)
		count++
	}
	if err := r.Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 records, got %d: %v", count, uris)
	}
}

func TestReader_AutoDrainBody(t *testing.T) {
	// Call Next() without reading body — should not hang or error
	rec1 := buildWARCRecord("https://a.com/", strings.Repeat("X", 5000))
	rec2 := buildWARCRecord("https://b.com/", "second")
	r := warc.NewReader(strings.NewReader(rec1 + rec2))

	if !r.Next() {
		t.Fatal("expected first record")
	}
	// Do NOT read body — let Next() drain it
	if !r.Next() {
		t.Fatalf("expected second record, err=%v", r.Err())
	}
	if r.Record().Header.TargetURI() != "https://b.com/" {
		t.Error("wrong second record")
	}
}

func TestReader_Header_Date(t *testing.T) {
	raw := "WARC/1.1\r\nWARC-Type: warcinfo\r\nWARC-Date: 2026-01-15T12:00:00Z\r\nContent-Length: 0\r\n\r\n\r\n\r\n"
	r := warc.NewReader(strings.NewReader(raw))
	if !r.Next() {
		t.Fatalf("Next()=false, err=%v", r.Err())
	}
	d := r.Record().Header.Date()
	if d.IsZero() {
		t.Error("Date() returned zero time")
	}
	if d.Year() != 2026 {
		t.Errorf("Date year = %d, want 2026", d.Year())
	}
}
