//go:build linux

package pipeline

import (
	"os"
	"strconv"
	"strings"
)

func init() {
	HostMemInfo = func() (total, avail int64) {
		data, err := os.ReadFile("/proc/meminfo")
		if err != nil {
			return 0, 0
		}
		for _, line := range strings.Split(string(data), "\n") {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			val, err := strconv.ParseInt(fields[1], 10, 64)
			if err != nil {
				continue
			}
			switch fields[0] {
			case "MemTotal:":
				total = val * 1024
			case "MemAvailable:":
				avail = val * 1024
			}
		}
		return
	}
}
