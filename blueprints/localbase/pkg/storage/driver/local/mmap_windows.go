//go:build windows

// File: driver/local/mmap_windows.go
package local

import (
	"io"
)

// openWithMmap is a fallback for Windows that returns an error.
// On Windows, we fall back to regular file I/O.
func openWithMmap(full string, offset, length int64) (io.ReadCloser, int64, error) {
	// Return an error to trigger fallback to regular file I/O
	return nil, 0, io.EOF
}

// mmapSupported returns false on Windows.
func mmapSupported() bool {
	return false
}
