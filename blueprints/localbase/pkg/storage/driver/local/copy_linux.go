//go:build linux

// File: driver/local/copy_linux.go
package local

import (
	"os"

	"golang.org/x/sys/unix"
)

// copyFileZeroCopyLinux uses copy_file_range for zero-copy file copying on Linux.
// This avoids copying data through user space, achieving up to 5x speedup.
func copyFileZeroCopyLinux(src, dst string) error {
	// #nosec G304 -- paths validated by caller
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}
	size := srcInfo.Size()

	// #nosec G304 -- paths validated by caller
	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, FilePermissions)
	if err != nil {
		return err
	}
	defer func() {
		dstFile.Close()
		if err != nil {
			os.Remove(dst)
		}
	}()

	srcFd := int(srcFile.Fd())
	dstFd := int(dstFile.Fd())

	// copy_file_range copies data in kernel space
	var written int64
	for written < size {
		n, err := unix.CopyFileRange(srcFd, nil, dstFd, nil, int(size-written), 0)
		if err != nil {
			// If copy_file_range fails, fall back to regular copy
			return fallbackCopy(srcFile, dstFile, written, size)
		}
		if n == 0 {
			break
		}
		written += int64(n)
	}

	// Optional fsync
	if !NoFsync {
		return dstFile.Sync()
	}
	return nil
}

// fallbackCopy completes the copy using regular read/write if copy_file_range fails.
func fallbackCopy(src, dst *os.File, offset, size int64) error {
	// Seek to the right position
	if _, err := src.Seek(offset, 0); err != nil {
		return err
	}
	if _, err := dst.Seek(offset, 0); err != nil {
		return err
	}

	buf := shardedLargePool.Get()
	defer shardedLargePool.Put(buf)

	remaining := size - offset
	for remaining > 0 {
		toRead := int64(len(buf))
		if toRead > remaining {
			toRead = remaining
		}

		n, err := src.Read(buf[:toRead])
		if err != nil {
			return err
		}
		if n == 0 {
			break
		}

		if _, err := dst.Write(buf[:n]); err != nil {
			return err
		}
		remaining -= int64(n)
	}

	return nil
}

// zeroCopySupported returns true on Linux where copy_file_range is available.
func zeroCopySupported() bool {
	return true
}

// copyFileZeroCopy is the Linux implementation using copy_file_range.
func copyFileZeroCopy(src, dst string) error {
	return copyFileZeroCopyLinux(src, dst)
}
