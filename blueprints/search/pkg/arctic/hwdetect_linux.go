//go:build linux

package arctic

import (
	"bufio"
	"os"
	"strings"
)

// detectRAM reads /proc/meminfo to get total and available RAM in GB.
func detectRAM() (total, avail float64) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var totalKB, availKB, freeKB, buffersKB, cachedKB int64
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "MemTotal:"):
			parseMeminfoKB(line, &totalKB)
		case strings.HasPrefix(line, "MemAvailable:"):
			parseMeminfoKB(line, &availKB)
		case strings.HasPrefix(line, "MemFree:"):
			parseMeminfoKB(line, &freeKB)
		case strings.HasPrefix(line, "Buffers:"):
			parseMeminfoKB(line, &buffersKB)
		case strings.HasPrefix(line, "Cached:"):
			// Only match "Cached:", not "SwapCached:"
			if !strings.HasPrefix(line, "CachedSwap") {
				parseMeminfoKB(line, &cachedKB)
			}
		}
	}

	total = float64(totalKB) / (1024 * 1024)

	// MemAvailable is the best metric (kernel 3.14+).
	// Fall back to Free + Buffers + Cached on older kernels.
	if availKB > 0 {
		avail = float64(availKB) / (1024 * 1024)
	} else {
		avail = float64(freeKB+buffersKB+cachedKB) / (1024 * 1024)
	}
	return total, avail
}

func parseMeminfoKB(line string, out *int64) {
	// Line format: "MemTotal:       16384000 kB"
	fields := strings.Fields(line)
	if len(fields) >= 2 {
		var v int64
		for _, c := range fields[1] {
			if c >= '0' && c <= '9' {
				v = v*10 + int64(c-'0')
			}
		}
		*out = v
	}
}
