//go:build !windows

// File: driver/local/copy_unix.go
package local

import (
	"errors"
)

// copyFileZeroCopy is a stub for non-Linux Unix systems.
// On Linux, this would use copy_file_range for zero-copy.
func copyFileZeroCopy(src, dst string) error {
	return errors.New("zero-copy not supported on this platform")
}

// zeroCopySupported returns false on non-Linux Unix systems.
func zeroCopySupported() bool {
	return false
}
