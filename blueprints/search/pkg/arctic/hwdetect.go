package arctic

import (
	"fmt"
	"os"
	"runtime"
	"syscall"
)

// HardwareProfile describes the detected hardware capabilities.
type HardwareProfile struct {
	Hostname   string  `json:"hostname"`
	OS         string  `json:"os"`
	CPUCores   int     `json:"cpu_cores"`
	RAMTotalGB float64 `json:"ram_total_gb"`
	RAMAvailGB float64 `json:"ram_avail_gb"`
	DiskTotalGB float64 `json:"disk_total_gb"`
	DiskFreeGB  float64 `json:"disk_free_gb"`
	NetworkMbps float64 `json:"network_mbps,omitempty"` // estimated after first download
}

// DetectHardware probes the current machine and returns a HardwareProfile.
// diskPath is used for disk space detection (typically Config.WorkDir).
func DetectHardware(diskPath string) HardwareProfile {
	hostname, _ := os.Hostname()
	totalRAM, availRAM := detectRAM()

	var diskTotal, diskFree float64
	if diskPath != "" {
		var st syscall.Statfs_t
		if err := syscall.Statfs(diskPath, &st); err == nil {
			diskTotal = float64(st.Blocks) * float64(st.Bsize) / (1024 * 1024 * 1024)
			diskFree = float64(st.Bavail) * float64(st.Bsize) / (1024 * 1024 * 1024)
		}
	}

	return HardwareProfile{
		Hostname:    hostname,
		OS:          runtime.GOOS,
		CPUCores:    runtime.NumCPU(),
		RAMTotalGB:  totalRAM,
		RAMAvailGB:  availRAM,
		DiskTotalGB: diskTotal,
		DiskFreeGB:  diskFree,
	}
}

// String returns a human-readable summary line.
func (h HardwareProfile) String() string {
	return fmt.Sprintf("%s (%s): %d cores, %.0f GB RAM (%.0f avail), %.0f GB disk (%.0f free)",
		h.Hostname, h.OS, h.CPUCores,
		h.RAMTotalGB, h.RAMAvailGB,
		h.DiskTotalGB, h.DiskFreeGB)
}
