//go:build linux

// File: driver/local/sync_linux.go
// Linux-specific sync using fdatasync for better write performance.
package local

import (
	"os"
	"syscall"
)

// syncFile uses fdatasync on Linux for better performance.
// fdatasync(2) flushes file data and only metadata that is required for
// subsequent data reads (i.e., file size). It skips flushing timestamp
// metadata (atime, mtime), which saves a journal write on ext4.
// This is 10-30% faster than fsync for typical write workloads.
func syncFile(f *os.File) error {
	return syscall.Fdatasync(int(f.Fd()))
}
