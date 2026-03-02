//go:build linux

package crawl

import (
	"fmt"
	"syscall"
)

func raiseRlimit(n uint64) error {
	var rl syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rl); err != nil {
		return fmt.Errorf("getrlimit: %w", err)
	}
	if rl.Cur >= n {
		return nil
	}
	if n > rl.Max {
		n = rl.Max
	}
	rl.Cur = n
	return syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rl)
}
