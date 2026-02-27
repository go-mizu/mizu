package crawl

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// SysInfo captures hardware and OS resource data used for auto-tuning crawl parameters.
type SysInfo struct {
	Hostname       string    `json:"hostname"`
	OS             string    `json:"os"`
	Arch           string    `json:"arch"`
	KernelVersion  string    `json:"kernel_version"`
	CPUCount       int       `json:"cpu_count"`
	GoMaxProcs     int       `json:"go_max_procs"`
	GoVersion      string    `json:"go_version"`
	MemTotalMB     int64     `json:"mem_total_mb"`
	MemAvailableMB int64     `json:"mem_available_mb"`
	FdSoftBefore   uint64    `json:"fd_soft_before"` // before raise attempt
	FdSoftAfter    uint64    `json:"fd_soft_after"`  // after raise attempt
	FdHard         uint64    `json:"fd_hard"`
	GatheredAt     time.Time `json:"gathered_at"`
	FromCache      bool      `json:"-"` // not persisted; set true when loaded from file
}

// GatherSysInfo collects hardware info and raises the fd soft limit to 65536.
// FdSoftBefore/FdSoftAfter reflect values before and after the raise attempt.
func GatherSysInfo() SysInfo {
	si := SysInfo{
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
		CPUCount:   runtime.NumCPU(),
		GoMaxProcs: runtime.GOMAXPROCS(0),
		GoVersion:  runtime.Version(),
		GatheredAt: time.Now(),
	}
	si.Hostname, _ = os.Hostname()
	gatherPlatformSysInfo(&si)
	return si
}

// DefaultSysInfoCachePath returns ~/.cache/search/sysinfo.json.
func DefaultSysInfoCachePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "search", "sysinfo.json")
}

// LoadOrGatherSysInfo loads SysInfo from cacheFile if it was gathered within ttl;
// otherwise gathers fresh data and saves the cache.
// raiseRlimit(65536) is always called regardless of cache hit.
func LoadOrGatherSysInfo(cacheFile string, ttl time.Duration) SysInfo {
	if si, err := loadSysInfoCache(cacheFile, ttl); err == nil {
		// Re-raise rlimit even on cache hit (idempotent; needed for fresh processes).
		_ = raiseRlimit(65536)
		si.FromCache = true
		return si
	}
	si := GatherSysInfo()
	_ = saveSysInfoCache(cacheFile, si)
	return si
}

// Table returns a human-readable hardware profile.
func (si SysInfo) Table() string {
	var sb strings.Builder

	memAvailPct := ""
	if si.MemTotalMB > 0 {
		memAvailPct = fmt.Sprintf(" (%.0f%% free)", float64(si.MemAvailableMB)/float64(si.MemTotalMB)*100)
	}

	fdRaise := ""
	if si.FdSoftAfter > si.FdSoftBefore {
		fdRaise = fmt.Sprintf(" → %s (raised)", formatSysNum(si.FdSoftAfter))
	} else if si.FdSoftAfter > 0 {
		fdRaise = fmt.Sprintf(" → %s", formatSysNum(si.FdSoftAfter))
	}

	cacheNote := ""
	if si.FromCache {
		age := time.Since(si.GatheredAt).Truncate(time.Second)
		cacheNote = fmt.Sprintf("  (cached %s ago)", age)
	}

	osStr := si.OS + "/" + si.Arch
	if si.KernelVersion != "" {
		osStr += "  │  kernel " + si.KernelVersion
	}

	fmt.Fprintf(&sb, "  ┌─ Hardware Profile%s ─────────────────────────────────────────\n", cacheNote)
	fmt.Fprintf(&sb, "  │  Hostname       %s\n", si.Hostname)
	fmt.Fprintf(&sb, "  │  OS             %s\n", osStr)
	fmt.Fprintf(&sb, "  │  CPUs           %d  │  GOMAXPROCS %d  │  %s\n",
		si.CPUCount, si.GoMaxProcs, si.GoVersion)
	if si.MemTotalMB > 0 {
		fmt.Fprintf(&sb, "  │  RAM total      %s\n", formatMB(si.MemTotalMB))
		fmt.Fprintf(&sb, "  │  RAM avail      %s%s\n", formatMB(si.MemAvailableMB), memAvailPct)
	}
	fmt.Fprintf(&sb, "  │  fd soft        %s%s  │  fd hard  %s\n",
		formatSysNum(si.FdSoftBefore), fdRaise, formatSysNum(si.FdHard))
	fmt.Fprintf(&sb, "  └───────────────────────────────────────────────────────────────\n")

	return sb.String()
}

// FormatMB formats a megabyte value as "X.X GB" or "X MB".
func FormatMB(mb int64) string {
	if mb >= 1024 {
		return fmt.Sprintf("%.1f GB", float64(mb)/1024)
	}
	return fmt.Sprintf("%d MB", mb)
}

// formatMB is the unexported alias used within the package.
func formatMB(mb int64) string { return FormatMB(mb) }

// formatSysNum formats a uint64 with comma separators.
func formatSysNum(n uint64) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}

func loadSysInfoCache(path string, ttl time.Duration) (SysInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return SysInfo{}, err
	}
	var si SysInfo
	if err := json.Unmarshal(data, &si); err != nil {
		return SysInfo{}, err
	}
	if time.Since(si.GatheredAt) > ttl {
		return SysInfo{}, fmt.Errorf("cache expired")
	}
	return si, nil
}

func saveSysInfoCache(path string, si SysInfo) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(si, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
