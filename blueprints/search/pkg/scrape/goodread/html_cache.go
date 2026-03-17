package goodread

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// HTMLCachePath returns the filesystem path for the gzipped HTML cache file.
// Layout: dataDir/html/{entityType}/{id}.html.gz
func HTMLCachePath(dataDir, entityType, id string) string {
	return filepath.Join(dataDir, "html", entityType, id+".html.gz")
}

// SaveHTML compresses html and writes it to path, creating parent dirs as needed.
func SaveHTML(path, html string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer f.Close()
	gz, err := gzip.NewWriterLevel(f, gzip.BestSpeed)
	if err != nil {
		return err
	}
	if _, err := io.WriteString(gz, html); err != nil {
		gz.Close()
		return err
	}
	return gz.Close()
}

// LoadHTML reads and decompresses a gzipped HTML cache file.
func LoadHTML(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer gz.Close()
	data, err := io.ReadAll(gz)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// DeleteHTML removes a cached HTML file. Ignores not-found errors.
func DeleteHTML(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// entityIDFromURL extracts the entity ID from a Goodreads URL.
// Examples:
//
//	/book/show/12345.The_Title → "12345"
//	/author/show/67890         → "67890"
//	/series/12345              → "12345"
func entityIDFromURL(rawURL string) string {
	if idx := strings.Index(rawURL, "?"); idx > 0 {
		rawURL = rawURL[:idx]
	}
	rawURL = strings.TrimSuffix(rawURL, "/")
	idx := strings.LastIndex(rawURL, "/")
	if idx < 0 {
		return rawURL
	}
	seg := rawURL[idx+1:]
	// GR URLs often look like /book/show/12345.Title — take the numeric prefix.
	if dotIdx := strings.Index(seg, "."); dotIdx > 0 {
		seg = seg[:dotIdx]
	}
	return seg
}
