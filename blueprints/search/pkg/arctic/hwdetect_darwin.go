//go:build darwin

package arctic

import (
	"os/exec"
	"strconv"
	"strings"
	"unsafe"

	"golang.org/x/sys/unix"
)

// detectRAM uses sysctl for total RAM and vm_stat for available on macOS.
func detectRAM() (total, avail float64) {
	// Total RAM via sysctl hw.memsize
	memsize, err := unix.SysctlUint64("hw.memsize")
	if err == nil {
		total = float64(memsize) / (1024 * 1024 * 1024)
	}

	// Available RAM: vm_stat gives page counts.
	// Available ≈ free + inactive + purgeable (macOS doesn't have MemAvailable).
	avail = estimateAvailDarwin()
	if avail == 0 {
		// Fallback: assume 70% of total is available (conservative).
		avail = total * 0.7
	}

	return total, avail
}

func estimateAvailDarwin() float64 {
	// Get page size.
	var pageSize int64
	psBuf, err := unix.SysctlRaw("hw.pagesize")
	if err == nil && len(psBuf) >= int(unsafe.Sizeof(int64(0))) {
		pageSize = *(*int64)(unsafe.Pointer(&psBuf[0]))
	}
	if pageSize == 0 {
		pageSize = 16384 // ARM64 macOS default
	}

	// Parse vm_stat output.
	out, err := exec.Command("vm_stat").Output()
	if err != nil {
		return 0
	}

	var freePages, inactivePages int64
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "Pages free:") {
			freePages = parseVMStatPages(line)
		} else if strings.HasPrefix(line, "Pages inactive:") {
			inactivePages = parseVMStatPages(line)
		}
	}

	availBytes := (freePages + inactivePages) * pageSize
	return float64(availBytes) / (1024 * 1024 * 1024)
}

func parseVMStatPages(line string) int64 {
	// Format: "Pages free:                              123456."
	parts := strings.SplitN(line, ":", 2)
	if len(parts) < 2 {
		return 0
	}
	s := strings.TrimSpace(parts[1])
	s = strings.TrimSuffix(s, ".")
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}
