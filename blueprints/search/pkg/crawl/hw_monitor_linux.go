//go:build linux

package crawl

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"
)

// hwRaw is a raw snapshot of kernel I/O counters read from /proc.
type hwRaw struct {
	ts        time.Time
	diskRead  uint64 // cumulative sectors read across all whole-disk devices
	diskWrite uint64 // cumulative sectors written
	netRx     uint64 // cumulative bytes received across all non-loopback interfaces
	netTx     uint64 // cumulative bytes transmitted
}

// hwRawRead reads a point-in-time snapshot from /proc/diskstats and /proc/net/dev.
// Missing or unreadable files return zero counters (graceful degradation).
func hwRawRead() hwRaw {
	r := hwRaw{ts: time.Now()}
	hwReadDiskStats(&r)
	hwReadNetStats(&r)
	return r
}

// hwReadDiskStats sums read/write sectors across all whole-disk block devices.
// Partitions (sda1, nvme0n1p1) are skipped; whole devices (sda, nvme0n1) are included.
//
// /proc/diskstats field layout (0-indexed):
//
//	[0] major [1] minor [2] name
//	[3] reads_completed [4] reads_merged [5] sectors_read [6] ms_reading
//	[7] writes_completed [8] writes_merged [9] sectors_written [10] ms_writing ...
func hwReadDiskStats(r *hwRaw) {
	f, err := os.Open("/proc/diskstats")
	if err != nil {
		return
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 10 {
			continue
		}
		name := fields[2]
		if hwIsPartition(name) {
			continue
		}
		if v, err := strconv.ParseUint(fields[5], 10, 64); err == nil {
			r.diskRead += v
		}
		if v, err := strconv.ParseUint(fields[9], 10, 64); err == nil {
			r.diskWrite += v
		}
	}
}

// hwIsPartition returns true if name looks like a disk partition rather than a whole device.
//
//   - sda   → false (whole disk)
//   - sda1  → true  (partition)
//   - nvme0n1   → false (whole NVMe device)
//   - nvme0n1p1 → true  (NVMe partition — contains 'p' after n-prefix)
//   - mmcblk0   → false (whole SD card)
//   - mmcblk0p1 → true  (SD card partition)
//   - vda / vda1 → false / true
func hwIsPartition(name string) bool {
	if len(name) == 0 {
		return false
	}
	last := name[len(name)-1]
	if last < '0' || last > '9' {
		// Ends with a letter → whole-disk device (sda, nvme0n1, vda, …)
		return false
	}
	// Name ends with a digit.
	// NVMe and MMC: whole device ends in a digit but has no 'p' segment.
	if strings.HasPrefix(name, "nvme") || strings.HasPrefix(name, "mmcblk") {
		// nvme0n1 → no 'p' → whole device; nvme0n1p1 → has 'p' → partition
		return strings.ContainsRune(name, 'p')
	}
	// All other block devices ending in a digit are partitions (sda1, vda1, hda1, xvda1, …).
	return true
}

// hwReadNetStats sums received/transmitted bytes across all non-loopback interfaces.
//
// /proc/net/dev format (after the colon, 0-indexed fields):
//
//	[0] rx_bytes [1] rx_packets … [8] tx_bytes …
func hwReadNetStats(r *hwRaw) {
	f, err := os.Open("/proc/net/dev")
	if err != nil {
		return
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Scan() // header line 1 ("Inter-|…")
	sc.Scan() // header line 2 (" face |bytes …")
	for sc.Scan() {
		line := sc.Text()
		colon := strings.IndexByte(line, ':')
		if colon < 0 {
			continue
		}
		iface := strings.TrimSpace(line[:colon])
		if iface == "lo" {
			continue // skip loopback
		}
		fields := strings.Fields(line[colon+1:])
		if len(fields) < 9 {
			continue
		}
		if v, err := strconv.ParseUint(fields[0], 10, 64); err == nil {
			r.netRx += v
		}
		if v, err := strconv.ParseUint(fields[8], 10, 64); err == nil {
			r.netTx += v
		}
	}
}
