package meta

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// extractPDFMetadata extracts metadata from PDF files.
func extractPDFMetadata(ctx context.Context, filePath string) (*DocumentMetadata, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	meta := &DocumentMetadata{}

	// Get file size for searching
	fileInfo, _ := file.Stat()
	fileSize := fileInfo.Size()

	// Read header to verify PDF
	header := make([]byte, 8)
	file.Read(header)
	if !bytes.HasPrefix(header, []byte("%PDF-")) {
		return nil, nil
	}

	// Extract PDF version
	meta.PDFVersion = string(header[5:8])

	// Count pages by searching for "/Type /Page" patterns
	file.Seek(0, io.SeekStart)
	meta.PageCount = countPDFPages(file, fileSize)

	// Look for document info dictionary in trailer or xref
	file.Seek(0, io.SeekStart)
	info := extractPDFInfo(file, fileSize)

	if info != nil {
		meta.Title = info.title
		meta.Author = info.author
		meta.Subject = info.subject
		meta.Keywords = info.keywords
		meta.Creator = info.creator
		meta.Producer = info.producer
		meta.CreatedAt = info.createdAt
		meta.ModifiedAt = info.modifiedAt
	}

	// Check for encryption
	file.Seek(0, io.SeekStart)
	meta.IsEncrypted = checkPDFEncryption(file)
	meta.HasPassword = meta.IsEncrypted

	return meta, nil
}

type pdfInfo struct {
	title      string
	author     string
	subject    string
	keywords   string
	creator    string
	producer   string
	createdAt  string
	modifiedAt string
}

// countPDFPages counts pages in a PDF by looking for page objects.
func countPDFPages(r io.ReadSeeker, fileSize int64) int {
	// Search for /Type /Page (or /Type/Page) pattern
	pagePattern := regexp.MustCompile(`/Type\s*/Page[^s]`)

	// Read in chunks
	const chunkSize = 64 * 1024
	buffer := make([]byte, chunkSize)
	count := 0

	for {
		n, err := r.Read(buffer)
		if n == 0 || err != nil {
			break
		}

		// Find all matches in this chunk
		matches := pagePattern.FindAll(buffer[:n], -1)
		count += len(matches)
	}

	// If no pages found, look for /Count in page tree
	if count == 0 {
		r.Seek(0, io.SeekStart)
		count = findPageCount(r)
	}

	return count
}

// findPageCount looks for /Count in the page tree.
func findPageCount(r io.ReadSeeker) int {
	countPattern := regexp.MustCompile(`/Count\s+(\d+)`)

	const chunkSize = 64 * 1024
	buffer := make([]byte, chunkSize)

	for {
		n, err := r.Read(buffer)
		if n == 0 || err != nil {
			break
		}

		if matches := countPattern.FindSubmatch(buffer[:n]); len(matches) > 1 {
			if count, err := strconv.Atoi(string(matches[1])); err == nil {
				return count
			}
		}
	}

	return 0
}

// extractPDFInfo extracts document information dictionary.
func extractPDFInfo(r io.ReadSeeker, fileSize int64) *pdfInfo {
	info := &pdfInfo{}

	// Search from end of file for trailer
	searchSize := int64(4096)
	if searchSize > fileSize {
		searchSize = fileSize
	}

	r.Seek(-searchSize, io.SeekEnd)
	trailer := make([]byte, searchSize)
	r.Read(trailer)

	// Look for Info reference in trailer
	infoPattern := regexp.MustCompile(`/Info\s+(\d+)\s+(\d+)\s+R`)
	matches := infoPattern.FindSubmatch(trailer)

	if len(matches) > 2 {
		objNum, _ := strconv.Atoi(string(matches[1]))
		// Find and parse the info object
		r.Seek(0, io.SeekStart)
		if obj := findPDFObject(r, objNum); obj != nil {
			parsePDFInfoDict(obj, info)
		}
	}

	// Also look for inline info in trailer
	parsePDFInfoDict(trailer, info)

	return info
}

// findPDFObject finds a specific object number in the PDF.
func findPDFObject(r io.ReadSeeker, objNum int) []byte {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	objPrefix := strconv.Itoa(objNum) + " 0 obj"
	var inObject bool
	var buffer bytes.Buffer

	for scanner.Scan() {
		line := scanner.Text()

		if !inObject && strings.Contains(line, objPrefix) {
			inObject = true
		}

		if inObject {
			buffer.WriteString(line)
			buffer.WriteByte('\n')

			if strings.Contains(line, "endobj") {
				return buffer.Bytes()
			}
		}
	}

	return nil
}

// parsePDFInfoDict extracts metadata from an info dictionary.
func parsePDFInfoDict(data []byte, info *pdfInfo) {
	// Define patterns for each field
	patterns := map[string]*regexp.Regexp{
		"title":    regexp.MustCompile(`/Title\s*\(([^)]*)\)|/Title\s*<([^>]*)>`),
		"author":   regexp.MustCompile(`/Author\s*\(([^)]*)\)|/Author\s*<([^>]*)>`),
		"subject":  regexp.MustCompile(`/Subject\s*\(([^)]*)\)|/Subject\s*<([^>]*)>`),
		"keywords": regexp.MustCompile(`/Keywords\s*\(([^)]*)\)|/Keywords\s*<([^>]*)>`),
		"creator":  regexp.MustCompile(`/Creator\s*\(([^)]*)\)|/Creator\s*<([^>]*)>`),
		"producer": regexp.MustCompile(`/Producer\s*\(([^)]*)\)|/Producer\s*<([^>]*)>`),
		"created":  regexp.MustCompile(`/CreationDate\s*\(([^)]*)\)|/CreationDate\s*<([^>]*)>`),
		"modified": regexp.MustCompile(`/ModDate\s*\(([^)]*)\)|/ModDate\s*<([^>]*)>`),
	}

	for key, pattern := range patterns {
		if matches := pattern.FindSubmatch(data); len(matches) > 1 {
			value := ""
			if len(matches[1]) > 0 {
				value = string(matches[1])
			} else if len(matches) > 2 && len(matches[2]) > 0 {
				value = decodeHexString(string(matches[2]))
			}

			value = decodePDFString(value)

			switch key {
			case "title":
				info.title = value
			case "author":
				info.author = value
			case "subject":
				info.subject = value
			case "keywords":
				info.keywords = value
			case "creator":
				info.creator = value
			case "producer":
				info.producer = value
			case "created":
				info.createdAt = formatPDFDate(value)
			case "modified":
				info.modifiedAt = formatPDFDate(value)
			}
		}
	}
}

// decodePDFString handles PDF string escaping.
func decodePDFString(s string) string {
	// Handle basic escape sequences
	s = strings.ReplaceAll(s, "\\n", "\n")
	s = strings.ReplaceAll(s, "\\r", "\r")
	s = strings.ReplaceAll(s, "\\t", "\t")
	s = strings.ReplaceAll(s, "\\\\", "\\")
	s = strings.ReplaceAll(s, "\\(", "(")
	s = strings.ReplaceAll(s, "\\)", ")")

	// Handle UTF-16BE BOM
	if len(s) >= 2 && s[0] == 0xFE && s[1] == 0xFF {
		// UTF-16BE encoded
		return decodeUTF16BE([]byte(s[2:]))
	}

	return s
}

func decodeUTF16BE(data []byte) string {
	var result strings.Builder
	for i := 0; i+1 < len(data); i += 2 {
		ch := uint16(data[i])<<8 | uint16(data[i+1])
		result.WriteRune(rune(ch))
	}
	return result.String()
}

func decodeHexString(s string) string {
	s = strings.ReplaceAll(s, " ", "")
	var result []byte
	for i := 0; i+1 < len(s); i += 2 {
		b, err := strconv.ParseUint(s[i:i+2], 16, 8)
		if err == nil {
			result = append(result, byte(b))
		}
	}
	return string(result)
}

// formatPDFDate converts PDF date format to readable format.
func formatPDFDate(s string) string {
	// PDF dates: D:YYYYMMDDHHmmSS+HH'mm'
	if len(s) < 2 || s[0:2] != "D:" {
		return s
	}

	s = s[2:]
	if len(s) < 8 {
		return s
	}

	year := s[0:4]
	month := "01"
	day := "01"
	hour := "00"
	min := "00"
	sec := "00"

	if len(s) >= 6 {
		month = s[4:6]
	}
	if len(s) >= 8 {
		day = s[6:8]
	}
	if len(s) >= 10 {
		hour = s[8:10]
	}
	if len(s) >= 12 {
		min = s[10:12]
	}
	if len(s) >= 14 {
		sec = s[12:14]
	}

	return year + "-" + month + "-" + day + " " + hour + ":" + min + ":" + sec
}

// checkPDFEncryption checks if PDF is encrypted.
func checkPDFEncryption(r io.ReadSeeker) bool {
	const searchSize = 8192
	buffer := make([]byte, searchSize)
	n, _ := r.Read(buffer)

	// Look for /Encrypt in the trailer or xref
	return bytes.Contains(buffer[:n], []byte("/Encrypt"))
}

// extractDocxMetadata extracts metadata from DOCX files.
func extractDocxMetadata(ctx context.Context, filePath string) (*DocumentMetadata, error) {
	// DOCX files are ZIP archives
	// Would need to read docProps/core.xml for metadata
	// For now, return basic info
	meta := &DocumentMetadata{}
	return meta, nil
}
