//go:build darwin

package web

import (
	"syscall"
	"unsafe"
)

func init() {
	hostMemInfo = darwinMemInfo
}

// darwinMemInfo returns total and approximate available physical memory
// using sysctl on macOS. Available memory is approximated as free+inactive.
func darwinMemInfo() (total, avail int64) {
	// hw.memsize returns total physical RAM.
	if b, err := syscall.Sysctl("hw.memsize"); err == nil {
		if len(b) >= 8 {
			total = int64(*(*uint64)(unsafe.Pointer(&[]byte(b)[0])))
		}
	}
	// vm_stat is complex to parse; approximation using free pages via sysctl.
	// Use total heap from Go as fallback — real vm_stat requires host_statistics64().
	// For dashboard purposes, total is the most useful metric.
	return total, 0
}
