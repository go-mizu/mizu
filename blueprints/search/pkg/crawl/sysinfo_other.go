//go:build !linux

package crawl

import "syscall"

func gatherPlatformSysInfo(si *SysInfo) {
	// fd limits (works on macOS and other Unix via syscall)
	var rl syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rl); err == nil {
		si.FdSoftBefore = rl.Cur
		si.FdHard = rl.Max
	}
	if err := raiseRlimit(65536); err == nil {
		if err2 := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rl); err2 == nil {
			si.FdSoftAfter = rl.Cur
		}
	} else {
		si.FdSoftAfter = si.FdSoftBefore
	}
	// MemTotal/MemAvailable not available cross-platform without cgo; leave as 0.
	// KernelVersion not set.
}
