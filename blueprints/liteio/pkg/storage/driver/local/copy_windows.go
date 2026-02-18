//go:build windows

// File: driver/local/copy_windows.go
package local

import "errors"

// copyFileZeroCopy is a stub for Windows - always falls back to regular copy.
func copyFileZeroCopy(src, dst string) error {
	return errors.New("zero-copy not supported on windows")
}

// zeroCopySupported returns false on Windows.
func zeroCopySupported() bool {
	return false
}
