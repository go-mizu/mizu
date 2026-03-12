//go:build linux || darwin

package pipeline

import "golang.org/x/sys/unix"

func diskUsage(path string) (total, free int64) {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return 0, 0
	}
	total = int64(stat.Blocks) * int64(stat.Bsize)
	free = int64(stat.Bavail) * int64(stat.Bsize)
	return
}
