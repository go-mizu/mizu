package cli

import (
	"fmt"
	"path/filepath"
	"strings"
)

// validatePath checks if a path is safe for template output.
// It rejects absolute paths, path traversal, and unsafe filenames.
func validatePath(path string) error {
	if path == "" {
		return fmt.Errorf("empty path")
	}

	// Reject absolute paths
	if filepath.IsAbs(path) {
		return fmt.Errorf("absolute path not allowed: %s", path)
	}

	// Clean the path
	clean := filepath.Clean(path)

	// Reject path traversal
	if strings.HasPrefix(clean, "..") || strings.Contains(clean, string(filepath.Separator)+"..") {
		return fmt.Errorf("path traversal not allowed: %s", path)
	}

	// Check each component
	parts := strings.Split(clean, string(filepath.Separator))
	for _, part := range parts {
		if err := validateFilename(part); err != nil {
			return fmt.Errorf("invalid path component %q: %w", part, err)
		}
	}

	return nil
}

// validateFilename checks if a filename is safe.
func validateFilename(name string) error {
	if name == "" || name == "." || name == ".." {
		return fmt.Errorf("invalid filename")
	}

	// Check for null bytes
	if strings.ContainsRune(name, 0) {
		return fmt.Errorf("null byte in filename")
	}

	// Check for reserved Windows filenames
	upper := strings.ToUpper(name)
	base := strings.TrimSuffix(upper, filepath.Ext(upper))
	reserved := []string{"CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9"}
	for _, r := range reserved {
		if base == r {
			return fmt.Errorf("reserved filename: %s", name)
		}
	}

	return nil
}
