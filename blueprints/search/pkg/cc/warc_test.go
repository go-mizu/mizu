package cc

import (
	"bytes"
	"compress/gzip"
	"testing"
)

func gzipCompress(data []byte) []byte {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	gz.Write(data)
	gz.Close()
	return buf.Bytes()
}

func TestParseWARCRecord(t *testing.T) {
	// Build a minimal WARC response record
	raw := "WARC/1.1\r\n" +
		"WARC-Type: response\r\n" +
		"WARC-Date: 2026-01-15T23:13:59Z\r\n" +
		"WARC-Target-URI: https://www.example.org/page.html\r\n" +
		"WARC-Record-ID: <urn:uuid:12345678-1234-5678-1234-567812345678>\r\n" +
		"Content-Type: application/http;msgtype=response\r\n" +
		"Content-Length: 150\r\n" +
		"\r\n" +
		"HTTP/1.1 200 OK\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"Content-Length: 52\r\n" +
		"\r\n" +
		"<html><head><title>Test Page</title></head><body>Hello</body></html>\r\n\r\n"

	compressed := gzipCompress([]byte(raw))

	resp, err := ParseWARCRecord(compressed)
	if err != nil {
		t.Fatalf("ParseWARCRecord failed: %v", err)
	}

	if resp.WARCType != "response" {
		t.Errorf("WARCType = %q, want %q", resp.WARCType, "response")
	}
	if resp.TargetURI != "https://www.example.org/page.html" {
		t.Errorf("TargetURI = %q, want %q", resp.TargetURI, "https://www.example.org/page.html")
	}
	if resp.HTTPStatus != 200 {
		t.Errorf("HTTPStatus = %d, want 200", resp.HTTPStatus)
	}
	if resp.HTTPHeaders["Content-Type"] != "text/html; charset=UTF-8" {
		t.Errorf("Content-Type = %q", resp.HTTPHeaders["Content-Type"])
	}
	if !bytes.Contains(resp.Body, []byte("<title>Test Page</title>")) {
		t.Errorf("Body does not contain expected title tag, got: %q", string(resp.Body))
	}
}

func TestParseHTTPStatusLine(t *testing.T) {
	tests := []struct {
		line string
		want int
	}{
		{"HTTP/1.1 200 OK", 200},
		{"HTTP/1.0 301 Moved Permanently", 301},
		{"HTTP/1.1 404 Not Found", 404},
		{"HTTP/2 200", 200},
		{"garbage", 0},
	}
	for _, tt := range tests {
		got := parseHTTPStatusLine(tt.line)
		if got != tt.want {
			t.Errorf("parseHTTPStatusLine(%q) = %d, want %d", tt.line, got, tt.want)
		}
	}
}

func TestExtractPageInfo(t *testing.T) {
	html := []byte(`<html>
<head>
<title>My Page Title</title>
<meta name="description" content="This is a test description">
</head>
<body>Hello world</body>
</html>`)

	title, desc := ExtractPageInfo(html)
	if title != "My Page Title" {
		t.Errorf("title = %q, want %q", title, "My Page Title")
	}
	if desc != "This is a test description" {
		t.Errorf("description = %q, want %q", desc, "This is a test description")
	}
}

func TestExtractPageInfoNoMeta(t *testing.T) {
	html := []byte(`<html><head><title>Simple</title></head><body>text</body></html>`)
	title, desc := ExtractPageInfo(html)
	if title != "Simple" {
		t.Errorf("title = %q, want %q", title, "Simple")
	}
	if desc != "" {
		t.Errorf("description = %q, want empty", desc)
	}
}

func TestBuildIndexQuery(t *testing.T) {
	filter := IndexFilter{
		Languages:  []string{"eng"},
		TLDs:       []string{"com", "org"},
		MimeTypes:  []string{"text/html"},
		StatusCodes: []int{200},
		Limit:      100,
	}

	query, args := buildIndexQuery(filter)

	if !bytes.Contains([]byte(query), []byte("fetch_status IN")) {
		t.Error("query should contain fetch_status IN clause")
	}
	if !bytes.Contains([]byte(query), []byte("content_mime_detected IN")) {
		t.Error("query should contain content_mime_detected IN clause")
	}
	if !bytes.Contains([]byte(query), []byte("url_host_tld IN")) {
		t.Error("query should contain url_host_tld IN clause")
	}
	if !bytes.Contains([]byte(query), []byte("content_languages LIKE")) {
		t.Error("query should contain content_languages LIKE clause")
	}
	if !bytes.Contains([]byte(query), []byte("LIMIT 100")) {
		t.Error("query should contain LIMIT 100")
	}

	// Should have: 200, "text/html", "com", "org", "%eng%"
	if len(args) != 5 {
		t.Errorf("expected 5 args, got %d: %v", len(args), args)
	}
}

func TestBuildCountQuery(t *testing.T) {
	filter := IndexFilter{
		StatusCodes: []int{200},
		Limit:       100,
	}

	query, _ := buildCountQuery(filter)
	if !bytes.Contains([]byte(query), []byte("SELECT COUNT(*)")) {
		t.Error("count query should start with SELECT COUNT(*)")
	}
}
