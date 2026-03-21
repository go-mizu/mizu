//go:build linux

package warc_md

import (
	"os"
	"strconv"
	"strings"
)

// readRSSMB returns the current process resident set size in MB by reading
// /proc/self/status. This is the real memory footprint visible to the OS,
// unlike runtime.MemStats.Sys which includes mapped-but-unused pages.
func readRSSMB() float64 {
	data, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "VmRSS:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				kb, err := strconv.ParseInt(fields[1], 10, 64)
				if err == nil {
					return float64(kb) / 1024.0
				}
			}
		}
	}
	return 0
}
