//go:build !windows

package web

import "syscall"

// diskUsage returns total and free bytes for the filesystem containing path.
func diskUsage(path string) (total, free int64) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0
	}
	total = int64(stat.Blocks) * int64(stat.Bsize)
	free = int64(stat.Bavail) * int64(stat.Bsize)
	return total, free
}
