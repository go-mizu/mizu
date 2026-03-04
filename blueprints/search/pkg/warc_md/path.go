package warc_md

import (
	"path/filepath"
	"strings"
)

// RecordIDToRelPath converts a WARC-Record-ID to a sharded relative path
// (without extension).
//
// Input:  "<urn:uuid:5d0e2270-349c-4861-bf28-1234567890ab>"
//         or bare UUID "5d0e2270-349c-4861-bf28-1234567890ab"
// Output: "5d/0e/22/5d0e2270-349c-4861-bf28-1234567890ab"
//
// The first 3 two-char segments (6 hex characters) become directory levels,
// limiting each leaf directory to ~256 files.
func RecordIDToRelPath(recordID string) string {
	id := stripWARCRecordID(recordID)
	if len(id) < 6 {
		return filepath.Join("xx", "xx", "xx", id)
	}
	return filepath.Join(id[0:2], id[2:4], id[4:6], id)
}

// WARCSingleFilePath returns the full path for a Phase 1 extracted HTML file.
func WARCSingleFilePath(baseDir, recordID string) string {
	return filepath.Join(baseDir, RecordIDToRelPath(recordID)+".warc")
}

// MarkdownFilePath returns the full path for a Phase 2 markdown file.
func MarkdownFilePath(baseDir, recordID string) string {
	return filepath.Join(baseDir, RecordIDToRelPath(recordID)+".md")
}

// MarkdownGzFilePath returns the full path for a Phase 3 compressed file.
func MarkdownGzFilePath(baseDir, recordID string) string {
	return filepath.Join(baseDir, RecordIDToRelPath(recordID)+".md.gz")
}

// stripWARCRecordID strips the angle-bracket wrapping and urn:uuid: prefix.
// "<urn:uuid:5d0e...>" → "5d0e..."
func stripWARCRecordID(id string) string {
	id = strings.TrimPrefix(id, "<")
	id = strings.TrimSuffix(id, ">")
	id = strings.TrimPrefix(id, "urn:uuid:")
	return id
}

// recordIDFromWARCSinglePath extracts the record ID from a warc_single file path.
// e.g. ".../5d/0e/22/5d0e2270-349c-4861-bf28-1234567890ab.warc" → "5d0e2270-349c-4861-bf28-1234567890ab"
func recordIDFromWARCSinglePath(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, ".warc")
}

// recordIDFromMarkdownPath extracts the record ID from a markdown file path.
func recordIDFromMarkdownPath(path string) string {
	base := filepath.Base(path)
	base = strings.TrimSuffix(base, ".md.gz")
	base = strings.TrimSuffix(base, ".md")
	return base
}

// MarkdownWarcFilePath returns the full path for the final .md output
// under the per-WARC directory (baseDir = MarkdownWarcDir result).
func MarkdownWarcFilePath(baseDir, recordID string) string {
	return filepath.Join(baseDir, RecordIDToRelPath(recordID)+".md")
}
