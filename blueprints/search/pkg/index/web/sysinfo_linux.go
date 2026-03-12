//go:build linux

package web

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

func init() {
	hostMemInfo = linuxMemInfo
	hostNetBytes = linuxNetBytes
	hostDiskBytes = linuxDiskBytes
}

// linuxMemInfo reads total and available memory from /proc/meminfo.
func linuxMemInfo() (total, avail int64) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			total = parseMemInfoKB(line)
		} else if strings.HasPrefix(line, "MemAvailable:") {
			avail = parseMemInfoKB(line)
		}
		if total > 0 && avail > 0 {
			break
		}
	}
	return total, avail
}

func parseMemInfoKB(line string) int64 {
	// Format: "MemTotal:    16384000 kB"
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0
	}
	v, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return 0
	}
	return v * 1024 // convert kB to bytes
}

// linuxNetBytes sums bytes received and sent across all non-loopback interfaces
// by reading /proc/net/dev.
func linuxNetBytes() (recv, sent int64) {
	f, err := os.Open("/proc/net/dev")
	if err != nil {
		return 0, 0
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	// Skip two header lines.
	sc.Scan()
	sc.Scan()
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		colonIdx := strings.Index(line, ":")
		if colonIdx < 0 {
			continue
		}
		iface := strings.TrimSpace(line[:colonIdx])
		if iface == "lo" {
			continue
		}
		fields := strings.Fields(line[colonIdx+1:])
		if len(fields) < 9 {
			continue
		}
		r, _ := strconv.ParseInt(fields[0], 10, 64)
		s, _ := strconv.ParseInt(fields[8], 10, 64)
		recv += r
		sent += s
	}
	return recv, sent
}

// linuxDiskBytes sums sectors read and written across all block devices
// by reading /proc/diskstats. Sector size is assumed to be 512 bytes.
func linuxDiskBytes() (read, written int64) {
	f, err := os.Open("/proc/diskstats")
	if err != nil {
		return 0, 0
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	seen := make(map[string]bool)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 14 {
			continue
		}
		dev := fields[2]
		// Skip partitions (e.g. sda1) — only count whole disks.
		if len(dev) > 0 && dev[len(dev)-1] >= '0' && dev[len(dev)-1] <= '9' {
			continue
		}
		if seen[dev] {
			continue
		}
		seen[dev] = true
		r, _ := strconv.ParseInt(fields[5], 10, 64)  // sectors read
		w, _ := strconv.ParseInt(fields[9], 10, 64)  // sectors written
		read += r * 512
		written += w * 512
	}
	return read, written
}
