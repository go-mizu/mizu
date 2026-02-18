//go:build !linux && !windows

// File: driver/local/sync_other.go
// Fallback sync using standard fsync for non-Linux platforms.
package local

import "os"

// syncFile falls back to Sync (fsync) on non-Linux platforms.
func syncFile(f *os.File) error {
	return f.Sync()
}
