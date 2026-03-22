//go:build !linux

package warc_md

import "runtime"

// readRSSMB returns an approximation of RSS on non-Linux platforms using
// runtime.MemStats.Sys (mapped virtual memory). On macOS this overestimates
// but is the best we can do without cgo.
func readRSSMB() float64 {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return float64(ms.Sys >> 20)
}
