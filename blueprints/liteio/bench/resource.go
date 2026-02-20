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
	Timestamp      time.Time `json:"timestamp"`
	Label          string    `json:"label,omitempty"`
	GoHeapMB       float64   `json:"go_heap_mb"`        // runtime.MemStats.HeapInuse
	GoSysMB        float64   `json:"go_sys_mb"`         // runtime.MemStats.Sys (total Go runtime)
	GoAllocMB      float64   `json:"go_alloc_mb"`       // runtime.MemStats.Alloc
	GoStackMB      float64   `json:"go_stack_mb"`       // runtime.MemStats.StackInuse
	NumGC          uint32    `json:"num_gc"`             // runtime.MemStats.NumGC
	PeakRSSMB      float64   `json:"peak_rss_mb"`       // OS-level peak RSS
	DiskUsageMB    float64   `json:"disk_usage_mb"`     // Data directory size
	TotalAllocMB   float64   `json:"total_alloc_mb"`    // runtime.MemStats.TotalAlloc (cumulative)
	MallocCount    uint64    `json:"malloc_count"`       // runtime.MemStats.Mallocs
	FreeCount      uint64    `json:"free_count"`         // runtime.MemStats.Frees
	LiveObjects    uint64    `json:"live_objects"`        // Mallocs - Frees
	GCPauseTotalMs float64   `json:"gc_pause_total_ms"`  // runtime.MemStats.PauseTotalNs / 1e6
	GCPauseLastMs  float64   `json:"gc_pause_last_ms"`   // last GC pause duration in ms
	NumGoroutines  int       `json:"num_goroutines"`      // runtime.NumGoroutine()
	HeapObjects    uint64    `json:"heap_objects"`        // runtime.MemStats.HeapObjects
}

// ResourceSummary holds per-driver resource summary for reports.
type ResourceSummary struct {
	PeakRSSMB      float64 `json:"peak_rss_mb"`
	PeakHeapMB     float64 `json:"peak_heap_mb"`
	PeakSysMB      float64 `json:"peak_sys_mb"`
	FinalDiskMB    float64 `json:"final_disk_mb"`
	NumGC          uint32  `json:"num_gc"`
	PeakAllocMB    float64 `json:"peak_alloc_mb"`     // peak Alloc (live allocations)
	TotalAllocMB   float64 `json:"total_alloc_mb"`    // cumulative allocations
	GCPauseTotalMs float64 `json:"gc_pause_total_ms"` // total GC pause time
	GCPauseMaxMs   float64 `json:"gc_pause_max_ms"`   // maximum single GC pause
	PeakGoroutines int     `json:"peak_goroutines"`   // peak goroutine count
	AllocRate      float64 `json:"alloc_rate_mbps"`   // MB allocated per second (computed)
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

	// Compute last GC pause duration.
	var gcPauseLastMs float64
	if ms.NumGC > 0 {
		// PauseNs is a circular buffer of recent GC pauses, indexed by NumGC.
		idx := (ms.NumGC + 255) % 256
		gcPauseLastMs = float64(ms.PauseNs[idx]) / 1e6
	}

	snap := ResourceSnapshot{
		Timestamp:      time.Now(),
		Label:          label,
		GoHeapMB:       float64(ms.HeapInuse) / (1024 * 1024),
		GoSysMB:        float64(ms.Sys) / (1024 * 1024),
		GoAllocMB:      float64(ms.Alloc) / (1024 * 1024),
		GoStackMB:      float64(ms.StackInuse) / (1024 * 1024),
		NumGC:          ms.NumGC,
		PeakRSSMB:      peakRSSMB(),
		DiskUsageMB:    dirSizeMB(rt.dataPath),
		TotalAllocMB:   float64(ms.TotalAlloc) / (1024 * 1024),
		MallocCount:    ms.Mallocs,
		FreeCount:      ms.Frees,
		LiveObjects:    ms.Mallocs - ms.Frees,
		GCPauseTotalMs: float64(ms.PauseTotalNs) / 1e6,
		GCPauseLastMs:  gcPauseLastMs,
		NumGoroutines:  runtime.NumGoroutine(),
		HeapObjects:    ms.HeapObjects,
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
		if snap.GoAllocMB > s.PeakAllocMB {
			s.PeakAllocMB = snap.GoAllocMB
		}
		if snap.GCPauseLastMs > s.GCPauseMaxMs {
			s.GCPauseMaxMs = snap.GCPauseLastMs
		}
		if snap.NumGoroutines > s.PeakGoroutines {
			s.PeakGoroutines = snap.NumGoroutines
		}
	}

	// Use the last snapshot's disk usage and cumulative metrics as final.
	last := rt.snapshots[len(rt.snapshots)-1]
	s.FinalDiskMB = last.DiskUsageMB
	s.TotalAllocMB = last.TotalAllocMB
	s.GCPauseTotalMs = last.GCPauseTotalMs

	// Compute allocation rate from first to last snapshot.
	first := rt.snapshots[0]
	if elapsed := last.Timestamp.Sub(first.Timestamp); elapsed > 0 {
		allocDeltaMB := last.TotalAllocMB - first.TotalAllocMB
		s.AllocRate = allocDeltaMB / elapsed.Seconds()
	}

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
