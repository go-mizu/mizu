package cc

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/textproto"
	"strconv"
	"strings"
	"time"
)

// ParseWARCRecord decompresses and parses a single WARC response record
// from gzip-compressed data (as returned by a byte-range request).
func ParseWARCRecord(data []byte) (*WARCResponse, error) {
	// Decompress gzip
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("gzip decompress: %w", err)
	}
	defer gz.Close()

	decompressed, err := io.ReadAll(gz)
	if err != nil {
		return nil, fmt.Errorf("reading decompressed data: %w", err)
	}

	return parseWARCFromBytes(decompressed)
}

// parseWARCFromBytes parses a decompressed WARC record.
func parseWARCFromBytes(data []byte) (*WARCResponse, error) {
	reader := bufio.NewReader(bytes.NewReader(data))

	// 1. Read WARC version line
	versionLine, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("reading WARC version: %w", err)
	}
	versionLine = strings.TrimSpace(versionLine)
	if !strings.HasPrefix(versionLine, "WARC/") {
		return nil, fmt.Errorf("not a WARC record: %q", versionLine)
	}

	// 2. Read WARC headers
	tp := textproto.NewReader(reader)
	warcHeaders, err := tp.ReadMIMEHeader()
	if err != nil {
		return nil, fmt.Errorf("reading WARC headers: %w", err)
	}

	resp := &WARCResponse{
		WARCType:  warcHeaders.Get("Warc-Type"),
		TargetURI: warcHeaders.Get("Warc-Target-Uri"),
		RecordID:  warcHeaders.Get("Warc-Record-Id"),
	}

	if dateStr := warcHeaders.Get("Warc-Date"); dateStr != "" {
		resp.Date, _ = time.Parse(time.RFC3339, dateStr)
		if resp.Date.IsZero() {
			resp.Date, _ = time.Parse("2006-01-02T15:04:05Z", dateStr)
		}
	}

	// For non-response records, return what we have
	if resp.WARCType != "response" {
		remaining, _ := io.ReadAll(reader)
		resp.Body = remaining
		return resp, nil
	}

	// 3. Parse HTTP response line
	httpLine, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("reading HTTP status line: %w", err)
	}
	httpLine = strings.TrimSpace(httpLine)
	resp.HTTPStatus = parseHTTPStatusLine(httpLine)

	// 4. Read HTTP headers
	tp2 := textproto.NewReader(reader)
	httpHeaders, err := tp2.ReadMIMEHeader()
	if err != nil {
		// Some records have malformed headers; try to continue
		resp.HTTPHeaders = make(map[string]string)
	} else {
		resp.HTTPHeaders = make(map[string]string, len(httpHeaders))
		for k := range httpHeaders {
			resp.HTTPHeaders[k] = httpHeaders.Get(k)
		}
	}

	// 5. Read HTTP body (remaining data)
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("reading HTTP body: %w", err)
	}
	// Trim trailing WARC record separator (\r\n\r\n)
	resp.Body = bytes.TrimRight(body, "\r\n")

	return resp, nil
}

// parseHTTPStatusLine extracts the status code from a line like "HTTP/1.1 200 OK".
func parseHTTPStatusLine(line string) int {
	parts := strings.SplitN(line, " ", 3)
	if len(parts) < 2 {
		return 0
	}
	code, _ := strconv.Atoi(parts[1])
	return code
}

// ExtractPageInfo extracts title, description, and content info from HTML body.
func ExtractPageInfo(body []byte) (title, description string) {
	s := string(body)

	// Extract title
	if idx := strings.Index(s, "<title"); idx >= 0 {
		// Find closing >
		closeTag := strings.Index(s[idx:], ">")
		if closeTag >= 0 {
			start := idx + closeTag + 1
			end := strings.Index(s[start:], "</title>")
			if end < 0 {
				end = strings.Index(s[start:], "</Title>")
			}
			if end < 0 {
				end = strings.Index(s[start:], "</TITLE>")
			}
			if end >= 0 {
				title = strings.TrimSpace(s[start : start+end])
				if len(title) > 500 {
					title = title[:500]
				}
			}
		}
	}

	// Extract meta description
	lower := strings.ToLower(s)
	if idx := strings.Index(lower, `name="description"`); idx >= 0 {
		description = extractMetaContent(s, idx)
	} else if idx := strings.Index(lower, `name='description'`); idx >= 0 {
		description = extractMetaContent(s, idx)
	} else if idx := strings.Index(lower, `property="og:description"`); idx >= 0 {
		description = extractMetaContent(s, idx)
	}

	return title, description
}

// extractMetaContent finds the content="" value near a meta tag attribute position.
func extractMetaContent(s string, attrPos int) string {
	// Search backward and forward for the <meta tag boundary
	start := attrPos - 200
	if start < 0 {
		start = 0
	}
	end := attrPos + 500
	if end > len(s) {
		end = len(s)
	}
	region := s[start:end]
	lower := strings.ToLower(region)

	if idx := strings.Index(lower, `content="`); idx >= 0 {
		valStart := idx + 9
		valEnd := strings.Index(lower[valStart:], `"`)
		if valEnd >= 0 {
			val := strings.TrimSpace(region[valStart : valStart+valEnd])
			if len(val) > 1000 {
				val = val[:1000]
			}
			return val
		}
	}
	return ""
}
