package crawl

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// BenchResult holds benchmark metrics for one chunk mode run.
type BenchResult struct {
	Mode            string  `json:"mode"`
	AvgRPS          float64 `json:"avg_rps"`
	PeakRPS         float64 `json:"peak_rps"`
	PeakHeapMB      uint64  `json:"peak_heap_mb"`
	GCCycles        uint32  `json:"gc_cycles"`
	DurationS       float64 `json:"duration_s"`
	OKCount         int64   `json:"ok_count"`
	BodyStoreWrites int64   `json:"body_store_writes"`
	BatchCount      int     `json:"batch_count"`
	FalseNegCount   int64   `json:"false_neg_count"` // URLs rescued by pass 2 retry
}

// BenchTracker collects runtime stats during a chunk mode crawl run.
type BenchTracker struct {
	mode       string
	start      time.Time
	peakHeapMB uint64
	gcBefore   uint32
	okCount    int64
	bsWrites   int64
	batchCount int
	peakRPS    float64
	falseNeg   int64
}

// NewBenchTracker starts a new benchmark tracker for the given chunk mode.
func NewBenchTracker(mode string) *BenchTracker {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return &BenchTracker{
		mode:     mode,
		start:    time.Now(),
		gcBefore: ms.NumGC,
	}
}

// RecordBatch records one completed batch: okDelta new OK results, elapsed time for the batch.
func (t *BenchTracker) RecordBatch(okDelta int64, elapsed time.Duration) {
	t.batchCount++
	t.okCount += okDelta
	if elapsed > 0 {
		rps := float64(okDelta) / elapsed.Seconds()
		if rps > t.peakRPS {
			t.peakRPS = rps
		}
	}
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	heapMB := ms.HeapAlloc / 1024 / 1024
	if heapMB > t.peakHeapMB {
		t.peakHeapMB = heapMB
	}
}

// RecordBodyStore adds n to the body store write count.
func (t *BenchTracker) RecordBodyStore(n int64) { t.bsWrites += n }

// RecordFalseNeg records the number of false negatives rescued by pass 2.
func (t *BenchTracker) RecordFalseNeg(n int64) { t.falseNeg = n }

// Save writes the benchmark result JSON to dataDir/bench_chunk_{mode}.json.
func (t *BenchTracker) Save(dataDir string) error {
	dur := time.Since(t.start)
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	result := BenchResult{
		Mode:            t.mode,
		AvgRPS:          float64(t.okCount) / dur.Seconds(),
		PeakRPS:         t.peakRPS,
		PeakHeapMB:      t.peakHeapMB,
		GCCycles:        ms.NumGC - t.gcBefore,
		DurationS:       dur.Seconds(),
		OKCount:         t.okCount,
		BodyStoreWrites: t.bsWrites,
		BatchCount:      t.batchCount,
		FalseNegCount:   t.falseNeg,
	}

	path := filepath.Join(dataDir, fmt.Sprintf("bench_chunk_%s.json", t.mode))
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
