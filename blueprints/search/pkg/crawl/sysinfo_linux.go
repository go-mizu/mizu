//go:build linux

package crawl

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"syscall"
)

func gatherPlatformSysInfo(si *SysInfo) {
	// Memory from /proc/meminfo
	if f, err := os.Open("/proc/meminfo"); err == nil {
		defer f.Close()
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := sc.Text()
			switch {
			case strings.HasPrefix(line, "MemTotal:"):
				si.MemTotalMB = parseMemInfoKB(line) / 1024
			case strings.HasPrefix(line, "MemAvailable:"):
				si.MemAvailableMB = parseMemInfoKB(line) / 1024
			}
		}
	}

	// Kernel version from /proc/version
	if data, err := os.ReadFile("/proc/version"); err == nil {
		parts := strings.Fields(string(data))
		if len(parts) >= 3 {
			si.KernelVersion = parts[2]
		}
	}

	// fd limits before raise
	var rl syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rl); err == nil {
		si.FdSoftBefore = rl.Cur
		si.FdHard = rl.Max
	}

	// Raise and record after value
	if err := raiseRlimit(65536); err == nil {
		if err2 := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rl); err2 == nil {
			si.FdSoftAfter = rl.Cur
		}
	} else {
		si.FdSoftAfter = si.FdSoftBefore
	}
}

// parseMemInfoKB parses a /proc/meminfo line like "MemTotal:    12345678 kB" → KB value.
func parseMemInfoKB(line string) int64 {
	fields := strings.Fields(line)
	if len(fields) >= 2 {
		v, _ := strconv.ParseInt(fields[1], 10, 64)
		return v
	}
	return 0
}
