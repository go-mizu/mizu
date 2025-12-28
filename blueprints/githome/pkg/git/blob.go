package git

import (
	"bytes"
	"context"
	"io"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// Blob represents file content
type Blob struct {
	SHA      string `json:"sha"`
	Path     string `json:"path"`
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	Content  string `json:"content,omitempty"`
	IsBinary bool   `json:"is_binary"`
	Encoding string `json:"encoding"`
	Language string `json:"language"`
	Lines    int    `json:"lines"`
	SLOC     int    `json:"sloc"` // Source lines of code (non-empty, non-comment)
}

// MaxBlobSize is the maximum size for inline content (1MB)
const MaxBlobSize = 1024 * 1024

// GetBlob retrieves file content at the given ref and path
func (r *Repository) GetBlob(ctx context.Context, ref, path string) (*Blob, error) {
	if !IsValidPath(path) {
		return nil, ErrPathTraversal
	}

	sha, err := r.ResolveRef(ctx, ref)
	if err != nil {
		return nil, err
	}

	// Get object info
	objRef := sha + ":" + path
	out, err := r.git(ctx, "cat-file", "-s", objRef)
	if err != nil {
		return nil, ErrNotFound
	}

	var size int64
	if _, err := parseIntFromBytes(out, &size); err != nil {
		size = 0
	}

	// Get object SHA
	blobSHA, err := r.git(ctx, "rev-parse", objRef)
	if err != nil {
		return nil, ErrNotFound
	}

	blob := &Blob{
		SHA:      strings.TrimSpace(string(blobSHA)),
		Path:     path,
		Name:     filepath.Base(path),
		Size:     size,
		Language: DetectLanguage(filepath.Base(path)),
		Encoding: "utf-8",
	}

	// Read content if not too large
	if size <= MaxBlobSize {
		content, err := r.git(ctx, "cat-file", "blob", objRef)
		if err != nil {
			return nil, err
		}

		// Check if binary
		if isBinary(content) {
			blob.IsBinary = true
			blob.Encoding = "base64"
		} else {
			blob.Content = string(content)
			blob.Lines = countLines(content)
			blob.SLOC = countSLOC(content, blob.Language)
		}
	} else {
		// For large files, just check if binary
		sample, _ := r.git(ctx, "cat-file", "blob", objRef)
		if len(sample) > 8192 {
			sample = sample[:8192]
		}
		blob.IsBinary = isBinary(sample)
	}

	return blob, nil
}

// GetBlobRaw returns a reader for raw file content
func (r *Repository) GetBlobRaw(ctx context.Context, ref, path string) (io.ReadCloser, error) {
	if !IsValidPath(path) {
		return nil, ErrPathTraversal
	}

	sha, err := r.ResolveRef(ctx, ref)
	if err != nil {
		return nil, err
	}

	objRef := sha + ":" + path
	return r.gitPipe(ctx, "cat-file", "blob", objRef)
}

// isBinary checks if content appears to be binary
func isBinary(content []byte) bool {
	// Check for null bytes (common in binary files)
	if bytes.Contains(content, []byte{0}) {
		return true
	}

	// Check if valid UTF-8
	if !utf8.Valid(content) {
		return true
	}

	// Check for high ratio of non-printable characters
	if len(content) == 0 {
		return false
	}

	nonPrintable := 0
	sample := content
	if len(sample) > 512 {
		sample = sample[:512]
	}

	for _, b := range sample {
		if b < 0x20 && b != '\n' && b != '\r' && b != '\t' {
			nonPrintable++
		}
	}

	return float64(nonPrintable)/float64(len(sample)) > 0.1
}

// countLines counts the number of lines in content
func countLines(content []byte) int {
	if len(content) == 0 {
		return 0
	}

	lines := bytes.Count(content, []byte{'\n'})
	// Add 1 if doesn't end with newline
	if content[len(content)-1] != '\n' {
		lines++
	}

	return lines
}

// countSLOC counts source lines of code (non-empty, non-comment lines)
func countSLOC(content []byte, language string) int {
	lines := bytes.Split(content, []byte{'\n'})
	sloc := 0

	inBlockComment := false
	commentStart, commentEnd, lineComment := getCommentMarkers(language)

	for _, line := range lines {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 {
			continue
		}

		lineStr := string(trimmed)

		// Handle block comments
		if commentStart != "" && commentEnd != "" {
			if inBlockComment {
				if idx := strings.Index(lineStr, commentEnd); idx >= 0 {
					inBlockComment = false
					lineStr = lineStr[idx+len(commentEnd):]
					trimmed = []byte(strings.TrimSpace(lineStr))
				} else {
					continue
				}
			}

			for strings.Contains(lineStr, commentStart) {
				startIdx := strings.Index(lineStr, commentStart)
				endIdx := strings.Index(lineStr[startIdx+len(commentStart):], commentEnd)
				if endIdx >= 0 {
					lineStr = lineStr[:startIdx] + lineStr[startIdx+len(commentStart)+endIdx+len(commentEnd):]
				} else {
					inBlockComment = true
					lineStr = lineStr[:startIdx]
					break
				}
			}
			trimmed = []byte(strings.TrimSpace(lineStr))
		}

		if len(trimmed) == 0 {
			continue
		}

		// Skip line comments
		if lineComment != "" && strings.HasPrefix(string(trimmed), lineComment) {
			continue
		}

		sloc++
	}

	return sloc
}

// getCommentMarkers returns comment markers for a language
func getCommentMarkers(language string) (blockStart, blockEnd, line string) {
	switch strings.ToLower(language) {
	case "go", "java", "javascript", "typescript", "c", "c++", "rust", "swift", "kotlin":
		return "/*", "*/", "//"
	case "python", "ruby", "shell", "bash":
		return "", "", "#"
	case "html", "xml":
		return "<!--", "-->", ""
	case "css":
		return "/*", "*/", ""
	case "sql":
		return "/*", "*/", "--"
	default:
		return "", "", ""
	}
}

func parseIntFromBytes(b []byte, v *int64) (int, error) {
	s := strings.TrimSpace(string(b))
	n, err := parseInt64(s)
	if err != nil {
		return 0, err
	}
	*v = n
	return len(s), nil
}

func parseInt64(s string) (int64, error) {
	var n int64
	for _, c := range s {
		if c < '0' || c > '9' {
			return n, nil
		}
		n = n*10 + int64(c-'0')
	}
	return n, nil
}
