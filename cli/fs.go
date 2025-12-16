package cli

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const (
	defaultDirMode  = 0755
	defaultFileMode = 0644
)

// ensureDir creates a directory and all parent directories.
func ensureDir(path string) error {
	return os.MkdirAll(path, defaultDirMode)
}

// atomicWrite writes data to a file atomically by writing to a temp file first.
func atomicWrite(path string, data []byte, mode fs.FileMode) error {
	if mode == 0 {
		mode = defaultFileMode
	}

	dir := filepath.Dir(path)
	if err := ensureDir(dir); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// Write to temp file in same directory
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, mode); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}

	// Rename to final path
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename temp file: %w", err)
	}

	return nil
}

// fileExists checks if a file exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// dirExists checks if a directory exists.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// readFileString reads a file as a string.
func readFileString(path string) (string, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path is validated by caller
	if err != nil {
		return "", err
	}
	return string(data), nil
}
