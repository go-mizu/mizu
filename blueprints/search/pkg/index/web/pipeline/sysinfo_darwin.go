//go:build darwin

package pipeline

func init() {
	// macOS: leave HostMemInfo nil — host memory info not available without CGO syscalls.
	// The overview response will show 0 for mem_total_bytes/mem_avail_bytes on macOS.
}
