// Package util provides pure helper functions shared by pipeline subpackages.
// It has no dependencies on the pipeline orchestration layer, so both
// pipeline and its sub-packages (cc, scrape) can import it without cycles.
package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ParseFileSelector parses a file selector string into a list of indices.
// Supports: "0", "0-4", "all", "".
func ParseFileSelector(s string, total int) ([]int, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "all" {
		idx := make([]int, total)
		for i := range idx {
			idx[i] = i
		}
		return idx, nil
	}

	if strings.Contains(s, "-") {
		parts := strings.SplitN(s, "-", 2)
		lo, err1 := strconv.Atoi(parts[0])
		hi, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil {
			return nil, fmt.Errorf("invalid range %q", s)
		}
		if lo < 0 || hi >= total || lo > hi {
			return nil, fmt.Errorf("range %d-%d out of bounds (total: %d)", lo, hi, total)
		}
		idx := make([]int, hi-lo+1)
		for i := range idx {
			idx[i] = lo + i
		}
		return idx, nil
	}

	n, err := strconv.Atoi(s)
	if err != nil {
		return nil, fmt.Errorf("invalid file index %q", s)
	}
	if n < 0 || n >= total {
		return nil, fmt.Errorf("file index %d out of bounds (total: %d)", n, total)
	}
	return []int{n}, nil
}

// WARCFileIndex extracts the zero-padded 5-digit WARC file index from a path.
// Falls back to fmt.Sprintf("%05d", fallback) if not parseable.
func WARCFileIndex(warcPath string, fallback int) string {
	base := filepath.Base(warcPath)
	name := strings.TrimSuffix(strings.TrimSuffix(base, ".gz"), ".warc")
	parts := strings.Split(name, "-")
	if last := parts[len(parts)-1]; len(last) == 5 {
		if _, err := strconv.Atoi(last); err == nil {
			return last
		}
	}
	return fmt.Sprintf("%05d", fallback)
}

// PackPath returns the expected pack file path for the given format and WARC index.
func PackPath(packDir, format, warcIdx string) (string, error) {
	switch format {
	case "parquet":
		return filepath.Join(packDir, "parquet", warcIdx+".parquet"), nil
	case "bin":
		return filepath.Join(packDir, "bin", warcIdx+".bin"), nil
	case "duckdb":
		return filepath.Join(packDir, "duckdb", warcIdx+".duckdb"), nil
	case "markdown":
		return filepath.Join(packDir, "markdown", warcIdx+".bin.gz"), nil
	default:
		return "", fmt.Errorf("unknown format %q (valid: parquet, bin, duckdb, markdown)", format)
	}
}

// PhaseProgress returns fractional progress clamped to [0, 1].
func PhaseProgress(done, total int64) float64 {
	if total <= 0 {
		if done > 0 {
			return 0.95
		}
		return 0
	}
	p := float64(done) / float64(total)
	if p < 0 {
		return 0
	}
	if p > 1 {
		return 1
	}
	return p
}

// PhaseRate returns done/elapsed in items per second.
func PhaseRate(done int64, elapsed time.Duration) float64 {
	if elapsed <= 0 {
		return 0
	}
	return float64(done) / elapsed.Seconds()
}

// MBPerSec converts bytes and elapsed time to MB/s.
func MBPerSec(bytes int64, elapsed time.Duration) float64 {
	if elapsed <= 0 {
		return 0
	}
	return float64(bytes) / (1024 * 1024) / elapsed.Seconds()
}

// FileProgress computes overall progress across a multi-file loop.
func FileProgress(fileIdx, fileTotal int, fileFraction float64) float64 {
	if fileTotal <= 0 {
		return 0
	}
	return (float64(fileIdx) + fileFraction) / float64(fileTotal)
}

// FileExists reports whether the given path exists.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
