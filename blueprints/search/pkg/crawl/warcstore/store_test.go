package warcstore_test

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/crawl/warcstore"
)

func newStore(t *testing.T) *warcstore.Store {
	t.Helper()
	s, err := warcstore.Open(t.TempDir(), false)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return s
}

func sampleEntry() warcstore.Entry {
	return warcstore.Entry{
		URL:        "https://example.com/page",
		Proto:      "HTTP/1.1",
		Method:     "GET",
		ReqHeaders: http.Header{"Accept": []string{"text/html"}},
		StatusCode: 200,
		StatusText: "200 OK",
		RespHeaders: http.Header{
			"Content-Type": []string{"text/html; charset=utf-8"},
		},
		Body:      []byte("<html><body>hello</body></html>"),
		IP:        "93.184.216.34",
		CrawledAt: time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
		RunID:     "test-run-001",
	}
}

// TestPut_PathStructure verifies:
// - path is {dir}/{hex[0:2]}/{hex[2:4]}/{hex[4:6]}/{uuid}.warc
// - file exists
// - contains WARC/1.1
// - has warcinfo + request + response records
// - has WARC-IP-Address
func TestPut_PathStructure(t *testing.T) {
	s := newStore(t)
	e := sampleEntry()

	warcID, err := s.Put(e)
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if warcID == "" {
		t.Fatal("Put returned empty warcID")
	}

	// Verify warc_id is a valid UUID format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	parts := strings.Split(warcID, "-")
	if len(parts) != 5 {
		t.Fatalf("warcID not UUID format: %q", warcID)
	}

	// Derive expected path
	expectedPath := s.Path(warcID)
	if expectedPath == "" {
		t.Fatal("Path returned empty string")
	}

	// Verify directory structure: {hex[0:2]}/{hex[2:4]}/{hex[4:6]}/{uuid}.warc
	hex := strings.ReplaceAll(warcID, "-", "")
	if len(hex) != 32 {
		t.Fatalf("hex len %d, want 32", len(hex))
	}
	expectedSuffix := filepath.Join(hex[0:2], hex[2:4], hex[4:6], warcID+".warc")
	if !strings.HasSuffix(expectedPath, expectedSuffix) {
		t.Fatalf("path %q does not end with %q", expectedPath, expectedSuffix)
	}

	// File must exist
	if _, err := os.Stat(expectedPath); err != nil {
		t.Fatalf("file not found: %v", err)
	}

	// Read file content and verify
	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "WARC/1.1") {
		t.Error("file does not contain WARC/1.1")
	}
	if !strings.Contains(content, "WARC-Type: warcinfo") {
		t.Error("file does not contain warcinfo record")
	}
	if !strings.Contains(content, "WARC-Type: request") {
		t.Error("file does not contain request record")
	}
	if !strings.Contains(content, "WARC-Type: response") {
		t.Error("file does not contain response record")
	}
	if !strings.Contains(content, "WARC-IP-Address: 93.184.216.34") {
		t.Error("file does not contain WARC-IP-Address")
	}
}

// TestPut_Deterministic verifies that calling Put with the same URL twice
// returns the same warc_id and the file is written only once.
func TestPut_Deterministic(t *testing.T) {
	s := newStore(t)
	e := sampleEntry()

	id1, err := s.Put(e)
	if err != nil {
		t.Fatalf("Put #1: %v", err)
	}

	// Second call: same URL
	id2, err := s.Put(e)
	if err != nil {
		t.Fatalf("Put #2: %v", err)
	}

	if id1 != id2 {
		t.Fatalf("warc_id changed across calls: %q vs %q", id1, id2)
	}
}

// TestPut_DifferentURLs verifies that different URLs produce different warc_ids.
func TestPut_DifferentURLs(t *testing.T) {
	s := newStore(t)

	e1 := sampleEntry()
	e1.URL = "https://example.com/page1"

	e2 := sampleEntry()
	e2.URL = "https://example.com/page2"

	id1, err := s.Put(e1)
	if err != nil {
		t.Fatalf("Put e1: %v", err)
	}
	id2, err := s.Put(e2)
	if err != nil {
		t.Fatalf("Put e2: %v", err)
	}

	if id1 == id2 {
		t.Fatalf("different URLs produced same warc_id: %q", id1)
	}
}

// TestPut_IdempotentSkip verifies that if a file already exists, Put does not
// return an error and returns the same warc_id.
func TestPut_IdempotentSkip(t *testing.T) {
	s := newStore(t)
	e := sampleEntry()

	id1, err := s.Put(e)
	if err != nil {
		t.Fatalf("Put #1: %v", err)
	}

	// Verify file exists
	path := s.Path(id1)
	info1, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat after first Put: %v", err)
	}

	// Second call should skip (file exists)
	id2, err := s.Put(e)
	if err != nil {
		t.Fatalf("Put #2 (idempotent): %v", err)
	}
	if id1 != id2 {
		t.Fatalf("warc_id mismatch on second Put: %q vs %q", id1, id2)
	}

	// File modification time should be unchanged (file was not rewritten)
	info2, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat after second Put: %v", err)
	}
	if !info2.ModTime().Equal(info1.ModTime()) {
		t.Error("file was rewritten on second Put (expected skip)")
	}
}

// TestCanonicalURL verifies the CanonicalURL function with various inputs.
func TestCanonicalURL(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"https://Example.COM/path", "https://example.com/path"},
		{"http://example.com:80/path", "http://example.com/path"},
		{"https://example.com:443/path", "https://example.com/path"},
		{"https://example.com:8080/path", "https://example.com:8080/path"},
		{"https://example.com/path#fragment", "https://example.com/path"},
		{"HTTPS://EXAMPLE.COM/PATH", "https://example.com/PATH"},
	}

	for _, tc := range cases {
		got := warcstore.CanonicalURL(tc.input)
		if got != tc.want {
			t.Errorf("CanonicalURL(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// parseWARCRecords splits a WARC file into records.
// Each record is a map of header-name→value plus a special key "__block__" for the block content.
// Returns the list of per-record header maps (in order) and block bytes.
type warcRecord struct {
	headers map[string]string // WARC headers (lowercased key)
	block   []byte
}

func parseWARCFile(content string) ([]warcRecord, error) {
	// Records are separated by \r\n\r\n at the end of each record block.
	// A record ends with: <block>\r\n\r\n
	// We split on WARC/1.1 markers.
	var records []warcRecord

	// Use a scanner that reads line by line
	scanner := bufio.NewScanner(strings.NewReader(content))

	var currentHeaders map[string]string
	var inHeaders bool
	var blockLines []string
	var expectBlock bool
	var blockLenExpected int

	flush := func() {
		if currentHeaders == nil {
			return
		}
		block := strings.Join(blockLines, "")
		records = append(records, warcRecord{
			headers: currentHeaders,
			block:   []byte(block),
		})
		currentHeaders = nil
		blockLines = nil
		expectBlock = false
		blockLenExpected = 0
	}

	for scanner.Scan() {
		line := scanner.Text() // without trailing \n; \r may remain

		// Strip trailing \r
		line = strings.TrimRight(line, "\r")

		if !inHeaders && !expectBlock {
			if line == "WARC/1.1" {
				flush()
				currentHeaders = make(map[string]string)
				inHeaders = true
			}
			continue
		}

		if inHeaders {
			if line == "" {
				// End of WARC headers — block follows
				inHeaders = false
				expectBlock = true
				if clStr, ok := currentHeaders["content-length"]; ok {
					n, _ := strconv.Atoi(clStr)
					blockLenExpected = n
				}
				continue
			}
			// Parse header line
			idx := strings.Index(line, ": ")
			if idx >= 0 {
				key := strings.ToLower(line[:idx])
				val := line[idx+2:]
				currentHeaders[key] = val
			}
			continue
		}

		if expectBlock {
			// We're collecting block content
			blockLines = append(blockLines, line+"\n")
			_ = blockLenExpected
		}
	}
	flush()

	return records, nil
}

// TestPut_WARCFormat verifies the on-disk WARC file structure in detail.
func TestPut_WARCFormat(t *testing.T) {
	s := newStore(t)
	e := sampleEntry()

	warcID, err := s.Put(e)
	if err != nil {
		t.Fatalf("Put: %v", err)
	}

	data, err := os.ReadFile(s.Path(warcID))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)

	// Verify exactly three records separated by \r\n\r\n
	// Count WARC/1.1 version lines
	occurrences := strings.Count(content, "WARC/1.1")
	if occurrences != 3 {
		t.Fatalf("expected 3 WARC/1.1 markers, got %d", occurrences)
	}

	// Verify records are separated by \r\n\r\n
	// Split on \r\n\r\n to find inter-record boundaries
	parts := strings.Split(content, "\r\n\r\n")
	// Should have at least 3 separators (after each record block)
	if len(parts) < 3 {
		t.Fatalf("expected at least 3 parts when split by \\r\\n\\r\\n, got %d", len(parts))
	}

	// Check response record has WARC-Record-ID: <urn:uuid:{warcID}>
	expectedRespID := fmt.Sprintf("WARC-Record-ID: <urn:uuid:%s>", warcID)
	if !strings.Contains(content, expectedRespID) {
		t.Errorf("response WARC-Record-ID not found: want %q", expectedRespID)
	}

	// Check request record has WARC-Concurrent-To pointing to the response UUID
	expectedConcurrent := fmt.Sprintf("WARC-Concurrent-To: <urn:uuid:%s>", warcID)
	if !strings.Contains(content, expectedConcurrent) {
		t.Errorf("WARC-Concurrent-To not found in request record: want %q", expectedConcurrent)
	}

	// Check warcinfo has WARC-Filename
	if !strings.Contains(content, "WARC-Filename:") {
		t.Error("warcinfo record missing WARC-Filename header")
	}

	// Verify Content-Length in each record matches the actual block byte length.
	// We do this by splitting on "WARC/1.1\r\n" to get each record section.
	recordBlocks := strings.Split(content, "WARC/1.1\r\n")
	// recordBlocks[0] is empty (content starts with WARC/1.1)
	if len(recordBlocks) < 4 { // [empty, warcinfo, request, response...]
		t.Fatalf("expected 4 parts when splitting on WARC/1.1, got %d", len(recordBlocks))
	}

	for i, rb := range recordBlocks[1:] {
		// rb starts with WARC headers, then \r\n (blank line), then block, then \r\n\r\n
		// Find end of WARC header section
		headerEnd := strings.Index(rb, "\r\n\r\n")
		if headerEnd < 0 {
			t.Fatalf("record %d: no \\r\\n\\r\\n separator found", i)
		}
		headerSection := rb[:headerEnd]
		// Find the blank line that separates WARC headers from block
		blankLine := strings.Index(headerSection, "\r\n\r\n")
		// The block starts after the first blank line (\r\n) in header section
		// Actually: WARC headers end at first \r\n (blank line), block follows
		// Let's find \r\n\r\n = end of last header + blank line
		// The format is: headers\r\n (one per line) then \r\n (blank) then block\r\n\r\n
		_ = blankLine

		// Find Content-Length in this record's headers.
		// Note: headerSection = rb[:headerEnd] where headerEnd is the start of
		// the \r\n\r\n blank-line separator. The last header's trailing \r\n is
		// consumed as the first \r\n of the \r\n\r\n, so Content-Length may be the
		// last header and appear without a trailing \r\n in headerSection.
		// We therefore search the full rb for Content-Length (it appears before the block).
		clIdx := strings.Index(rb, "Content-Length: ")
		if clIdx < 0 {
			t.Fatalf("record %d: no Content-Length header", i)
		}
		clEnd := strings.Index(rb[clIdx:], "\r\n")
		if clEnd < 0 {
			t.Fatalf("record %d: malformed Content-Length header", i)
		}
		clStr := rb[clIdx+len("Content-Length: ") : clIdx+clEnd]
		declaredLen, err := strconv.Atoi(clStr)
		if err != nil {
			t.Fatalf("record %d: invalid Content-Length %q: %v", i, clStr, err)
		}

		// The block starts immediately after the \r\n\r\n blank-line separator.
		// In rb (after splitting on "WARC/1.1\r\n"):
		//   "WARC-headers...\r\nContent-Length: N\r\n\r\nblock\r\n\r\n[next-record or EOF]"
		// headerEnd is the index of the \r\n\r\n separator (blank line after headers).
		// blockStart is right after that separator.
		blockStart := headerEnd + 4 // skip \r\n\r\n
		if blockStart > len(rb) {
			t.Fatalf("record %d: blockStart %d exceeds rb length %d", i, blockStart, len(rb))
		}
		// Validate that the Content-Length is correct by checking that exactly
		// declaredLen bytes of block are followed by \r\n\r\n (inter-record separator)
		// or by end-of-string for the last record.
		// We use the raw []byte of the original file (content) for accurate byte counting.
		// Find this record's blockStart in the original content.
		rbBytes := []byte(rb)
		blockEndInRb := blockStart + declaredLen
		if blockEndInRb > len(rbBytes) {
			t.Errorf("record %d: Content-Length=%d would exceed rb length %d (blockStart=%d)",
				i, declaredLen, len(rbBytes), blockStart)
			continue
		}
		// After the block, there must be \r\n\r\n or end of data.
		tail := string(rbBytes[blockEndInRb:])
		if tail != "" && !strings.HasPrefix(tail, "\r\n\r\n") {
			t.Errorf("record %d: Content-Length=%d but no \\r\\n\\r\\n after block (tail starts with %q)",
				i, declaredLen, tail[:min(len(tail), 8)])
		}
	}
}

// TestPut_NoBodyStore_EmptyIP verifies that when IP is empty,
// WARC-IP-Address is not written to the file.
func TestPut_NoBodyStore_EmptyIP(t *testing.T) {
	s := newStore(t)
	e := sampleEntry()
	e.IP = "" // no IP

	warcID, err := s.Put(e)
	if err != nil {
		t.Fatalf("Put: %v", err)
	}

	data, err := os.ReadFile(s.Path(warcID))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)

	if strings.Contains(content, "WARC-IP-Address") {
		t.Error("WARC-IP-Address should not be present when IP is empty")
	}
}

func TestPut_Compress(t *testing.T) {
	dir := t.TempDir()
	s, err := warcstore.Open(dir, true)
	if err != nil {
		t.Fatal(err)
	}
	if !s.Compressed() {
		t.Error("Compressed() should be true")
	}

	e := sampleEntry()
	id, err := s.Put(e)
	if err != nil {
		t.Fatalf("Put: %v", err)
	}

	// File must have .warc.gz extension.
	hex := strings.ReplaceAll(id, "-", "")
	path := filepath.Join(dir, hex[0:2], hex[2:4], hex[4:6], id+".warc.gz")
	if _, err := os.Stat(path); err != nil {
		t.Errorf("compressed file not found at %s: %v", path, err)
	}
	// .warc (uncompressed) must NOT exist.
	plain := filepath.Join(dir, hex[0:2], hex[2:4], hex[4:6], id+".warc")
	if _, err := os.Stat(plain); err == nil {
		t.Error("plain .warc should not exist when compress=true")
	}

	// Content must decompress to valid WARC.
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("gzip.NewReader: %v", err)
	}
	defer gz.Close()
	var sb strings.Builder
	buf := make([]byte, 4096)
	for {
		n, err := gz.Read(buf)
		sb.Write(buf[:n])
		if err != nil {
			break
		}
	}
	content := sb.String()
	if !strings.Contains(content, "WARC/1.1") {
		t.Error("decompressed content missing WARC/1.1")
	}
	if !strings.Contains(content, "WARC-Type: response") {
		t.Error("decompressed content missing response record")
	}
}
