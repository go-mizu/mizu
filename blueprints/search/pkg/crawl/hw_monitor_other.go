//go:build !linux

package crawl

import "time"

// hwRaw is a stub snapshot for non-Linux platforms.
// All counters are always zero.
type hwRaw struct {
	ts        time.Time
	diskRead  uint64
	diskWrite uint64
	netRx     uint64
	netTx     uint64
}

// hwRawRead returns a zero-filled snapshot on non-Linux platforms.
// The HWMonitor will consequently always report 0 MB/s for disk and network.
func hwRawRead() hwRaw {
	return hwRaw{ts: time.Now()}
}
