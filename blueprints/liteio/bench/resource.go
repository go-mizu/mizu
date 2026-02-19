package bench

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"
)

// ResourceSnapshot captures a point-in-time view of resource usage.
type ResourceSnapshot struct {
	Timestamp   time.Time `json:"timestamp"`
	Label       string    `json:"label,omitempty"`
	GoHeapMB    float64   `json:"go_heap_mb"`    // runtime.MemStats.HeapInuse
	GoSysMB     float64   `json:"go_sys_mb"`     // runtime.MemStats.Sys (total Go runtime)
	GoAllocMB   float64   `json:"go_alloc_mb"`   // runtime.MemStats.Alloc
	GoStackMB   float64   `json:"go_stack_mb"`   // runtime.MemStats.StackInuse
	NumGC       uint32    `json:"num_gc"`         // runtime.MemStats.NumGC
	PeakRSSMB   float64   `json:"peak_rss_mb"`   // OS-level peak RSS
	DiskUsageMB float64   `json:"disk_usage_mb"`  // Data directory size
}

// ResourceSummary holds per-driver resource summary for reports.
type ResourceSummary struct {
	PeakRSSMB   float64 `json:"peak_rss_mb"`
	PeakHeapMB  float64 `json:"peak_heap_mb"`
	PeakSysMB   float64 `json:"peak_sys_mb"`
	FinalDiskMB float64 `json:"final_disk_mb"`
	NumGC       uint32  `json:"num_gc"`
}

// ResourceTracker collects resource snapshots during benchmarks.
type ResourceTracker struct {
	dataPath  string
	snapshots []ResourceSnapshot
	mu        sync.Mutex
}

// NewResourceTracker creates a tracker for the given data directory.
func NewResourceTracker(dataPath string) *ResourceTracker {
	return &ResourceTracker{dataPath: dataPath}
}

// Snapshot captures the current resource state.
func (rt *ResourceTracker) Snapshot(label string) ResourceSnapshot {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	snap := ResourceSnapshot{
		Timestamp:   time.Now(),
		Label:       label,
		GoHeapMB:    float64(ms.HeapInuse) / (1024 * 1024),
		GoSysMB:     float64(ms.Sys) / (1024 * 1024),
		GoAllocMB:   float64(ms.Alloc) / (1024 * 1024),
		GoStackMB:   float64(ms.StackInuse) / (1024 * 1024),
		NumGC:       ms.NumGC,
		PeakRSSMB:   peakRSSMB(),
		DiskUsageMB: dirSizeMB(rt.dataPath),
	}

	rt.mu.Lock()
	rt.snapshots = append(rt.snapshots, snap)
	rt.mu.Unlock()

	return snap
}

// Summary returns a ResourceSummary from all collected snapshots.
func (rt *ResourceTracker) Summary() *ResourceSummary {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	if len(rt.snapshots) == 0 {
		return &ResourceSummary{}
	}

	var s ResourceSummary
	for _, snap := range rt.snapshots {
		if snap.PeakRSSMB > s.PeakRSSMB {
			s.PeakRSSMB = snap.PeakRSSMB
		}
		if snap.GoHeapMB > s.PeakHeapMB {
			s.PeakHeapMB = snap.GoHeapMB
		}
		if snap.GoSysMB > s.PeakSysMB {
			s.PeakSysMB = snap.GoSysMB
		}
		if snap.NumGC > s.NumGC {
			s.NumGC = snap.NumGC
		}
	}

	// Use the last snapshot's disk usage as final.
	last := rt.snapshots[len(rt.snapshots)-1]
	s.FinalDiskMB = last.DiskUsageMB

	return &s
}

// peakRSSMB returns the process peak RSS in megabytes.
// On macOS, ru_maxrss is in bytes. On Linux, it's in kilobytes.
func peakRSSMB() float64 {
	var rusage syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &rusage); err != nil {
		return 0
	}
	// macOS: ru_maxrss is in bytes
	if runtime.GOOS == "darwin" {
		return float64(rusage.Maxrss) / (1024 * 1024)
	}
	// Linux: ru_maxrss is in kilobytes
	return float64(rusage.Maxrss) / 1024
}

// dirSizeMB returns the total size of files in a directory, in megabytes.
func dirSizeMB(path string) float64 {
	if path == "" {
		return 0
	}
	var total int64
	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors
		}
		if !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	return float64(total) / (1024 * 1024)
}
