package bench

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// LimitsConfig configures the physical limits benchmark.
type LimitsConfig struct {
	BenchTime time.Duration // target duration per benchmark
	OutputDir string        // output directory for reports
}

// LimitsResult holds the result of one physical limit benchmark.
type LimitsResult struct {
	Operation  string  `json:"operation"`
	Iterations int     `json:"iterations"`
	TotalNs    int64   `json:"total_time_ns"`
	Throughput float64 `json:"throughput_mbps,omitempty"` // MB/s for data ops
	OpsPerSec  float64 `json:"ops_per_sec"`
	AvgNs      int64   `json:"avg_latency_ns"`
	P50Ns      int64   `json:"p50_latency_ns"`
	P99Ns      int64   `json:"p99_latency_ns"`
	ObjectSize int     `json:"object_size,omitempty"`
}

// LimitsReport holds all physical limits results.
type LimitsReport struct {
	Timestamp string          `json:"timestamp"`
	Platform  string          `json:"platform"`
	CPU       string          `json:"cpu"`
	RAMGB     int             `json:"ram_gb"`
	Results   []*LimitsResult `json:"results"`
}

// RunLimits runs all physical limits benchmarks and saves reports.
func RunLimits(ctx context.Context, cfg LimitsConfig) error {
	if cfg.BenchTime == 0 {
		cfg.BenchTime = 1 * time.Second
	}
	if cfg.OutputDir == "" {
		cfg.OutputDir = "./report"
	}

	report := &LimitsReport{
		Timestamp: time.Now().Format(time.RFC3339),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		CPU:       detectCPU(),
		RAMGB:     detectRAMGB(),
	}

	fmt.Println("=== Physical Limits Benchmark ===")
	fmt.Printf("Platform: %s\n", report.Platform)
	fmt.Printf("CPU: %s\n", report.CPU)
	fmt.Printf("RAM: %d GB\n", report.RAMGB)
	fmt.Printf("BenchTime: %v per benchmark\n\n", cfg.BenchTime)

	// Create temp directory for file benchmarks
	tmpDir, err := os.MkdirTemp("", "bench-limits-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	type benchFn struct {
		name string
		fn   func(context.Context, time.Duration, string) *LimitsResult
	}

	benches := []benchFn{
		// Memory operations
		{"Memcpy/4KB", func(ctx context.Context, d time.Duration, dir string) *LimitsResult {
			return benchMemcpy(ctx, d, 4*1024)
		}},
		{"Memcpy/64KB", func(ctx context.Context, d time.Duration, dir string) *LimitsResult {
			return benchMemcpy(ctx, d, 64*1024)
		}},
		{"Memcpy/1MB", func(ctx context.Context, d time.Duration, dir string) *LimitsResult {
			return benchMemcpy(ctx, d, 1024*1024)
		}},
		{"Memcpy/10MB", func(ctx context.Context, d time.Duration, dir string) *LimitsResult {
			return benchMemcpy(ctx, d, 10*1024*1024)
		}},
		{"Memcpy/100MB", func(ctx context.Context, d time.Duration, dir string) *LimitsResult {
			return benchMemcpy(ctx, d, 100*1024*1024)
		}},

		// Mmap read (warm pages)
		{"MmapRead/4KB", func(ctx context.Context, d time.Duration, dir string) *LimitsResult {
			return benchMmapRead(ctx, d, dir, 4*1024)
		}},
		{"MmapRead/64KB", func(ctx context.Context, d time.Duration, dir string) *LimitsResult {
			return benchMmapRead(ctx, d, dir, 64*1024)
		}},
		{"MmapRead/1MB", func(ctx context.Context, d time.Duration, dir string) *LimitsResult {
			return benchMmapRead(ctx, d, dir, 1024*1024)
		}},

		// Mmap write (warm pages)
		{"MmapWrite/4KB", func(ctx context.Context, d time.Duration, dir string) *LimitsResult {
			return benchMmapWrite(ctx, d, dir, 4*1024)
		}},
		{"MmapWrite/64KB", func(ctx context.Context, d time.Duration, dir string) *LimitsResult {
			return benchMmapWrite(ctx, d, dir, 64*1024)
		}},

		// Mmap page faults (cold sparse pages)
		{"MmapPageFault/4KB", func(ctx context.Context, d time.Duration, dir string) *LimitsResult {
			return benchMmapPageFault(ctx, d, dir, 4*1024)
		}},
		{"MmapPageFault/64KB", func(ctx context.Context, d time.Duration, dir string) *LimitsResult {
			return benchMmapPageFault(ctx, d, dir, 64*1024)
		}},

		// Pwrite (pre-allocated file)
		{"Pwrite/4KB", func(ctx context.Context, d time.Duration, dir string) *LimitsResult {
			return benchPwrite(ctx, d, dir, 4*1024)
		}},
		{"Pwrite/64KB", func(ctx context.Context, d time.Duration, dir string) *LimitsResult {
			return benchPwrite(ctx, d, dir, 64*1024)
		}},
		{"Pwrite/1MB", func(ctx context.Context, d time.Duration, dir string) *LimitsResult {
			return benchPwrite(ctx, d, dir, 1024*1024)
		}},
		{"Pwrite/10MB", func(ctx context.Context, d time.Duration, dir string) *LimitsResult {
			return benchPwrite(ctx, d, dir, 10*1024*1024)
		}},
		{"Pwrite/100MB", func(ctx context.Context, d time.Duration, dir string) *LimitsResult {
			return benchPwrite(ctx, d, dir, 100*1024*1024)
		}},

		// Pread (warm file)
		{"Pread/4KB", func(ctx context.Context, d time.Duration, dir string) *LimitsResult {
			return benchPread(ctx, d, dir, 4*1024)
		}},
		{"Pread/64KB", func(ctx context.Context, d time.Duration, dir string) *LimitsResult {
			return benchPread(ctx, d, dir, 64*1024)
		}},
		{"Pread/1MB", func(ctx context.Context, d time.Duration, dir string) *LimitsResult {
			return benchPread(ctx, d, dir, 1024*1024)
		}},

		// Pwrite to sparse file
		{"PwriteSparse/1MB", func(ctx context.Context, d time.Duration, dir string) *LimitsResult {
			return benchPwriteSparse(ctx, d, dir, 1024*1024)
		}},
		{"PwriteSparse/10MB", func(ctx context.Context, d time.Duration, dir string) *LimitsResult {
			return benchPwriteSparse(ctx, d, dir, 10*1024*1024)
		}},

		// File operations
		{"FileCreate", func(ctx context.Context, d time.Duration, dir string) *LimitsResult {
			return benchFileCreate(ctx, d, dir)
		}},
		{"FileCreateWrite/1KB", func(ctx context.Context, d time.Duration, dir string) *LimitsResult {
			return benchFileCreateWrite(ctx, d, dir, 1024)
		}},
		{"FileCreateWrite/64KB", func(ctx context.Context, d time.Duration, dir string) *LimitsResult {
			return benchFileCreateWrite(ctx, d, dir, 64*1024)
		}},
		{"FileCreateWrite/1MB", func(ctx context.Context, d time.Duration, dir string) *LimitsResult {
			return benchFileCreateWrite(ctx, d, dir, 1024*1024)
		}},
		{"FileCreateWrite/10MB", func(ctx context.Context, d time.Duration, dir string) *LimitsResult {
			return benchFileCreateWrite(ctx, d, dir, 10*1024*1024)
		}},
		{"FileDelete", func(ctx context.Context, d time.Duration, dir string) *LimitsResult {
			return benchFileDelete(ctx, d, dir)
		}},
		{"FileStat", func(ctx context.Context, d time.Duration, dir string) *LimitsResult {
			return benchFileStat(ctx, d, dir)
		}},
		{"DirCreate", func(ctx context.Context, d time.Duration, dir string) *LimitsResult {
			return benchDirCreate(ctx, d, dir)
		}},

		// Hash operations
		{"CRC32/1MB", func(ctx context.Context, d time.Duration, dir string) *LimitsResult {
			return benchCRC32(ctx, d, 1024*1024)
		}},
		{"FNV32/1KB", func(ctx context.Context, d time.Duration, dir string) *LimitsResult {
			return benchFNV32(ctx, d, 1024)
		}},

		// Atomic operations
		{"AtomicAdd", func(ctx context.Context, d time.Duration, dir string) *LimitsResult {
			return benchAtomicAdd(ctx, d)
		}},
		{"AtomicCAS", func(ctx context.Context, d time.Duration, dir string) *LimitsResult {
			return benchAtomicCAS(ctx, d)
		}},
		{"MutexLock", func(ctx context.Context, d time.Duration, dir string) *LimitsResult {
			return benchMutexLock(ctx, d)
		}},
		{"RWMutexRLock", func(ctx context.Context, d time.Duration, dir string) *LimitsResult {
			return benchRWMutexRLock(ctx, d)
		}},

		// Allocation
		{"MakeSlice/1KB", func(ctx context.Context, d time.Duration, dir string) *LimitsResult {
			return benchMakeSlice(ctx, d, 1024)
		}},
		{"MakeSlice/1MB", func(ctx context.Context, d time.Duration, dir string) *LimitsResult {
			return benchMakeSlice(ctx, d, 1024*1024)
		}},
		{"SyncPoolGet", func(ctx context.Context, d time.Duration, dir string) *LimitsResult {
			return benchSyncPool(ctx, d)
		}},
	}

	for i, b := range benches {
		select {
		case <-ctx.Done():
			fmt.Println("\nBenchmark cancelled")
			break
		default:
		}

		fmt.Printf("  [%d/%d] %s...", i+1, len(benches), b.name)
		result := b.fn(ctx, cfg.BenchTime, tmpDir)
		if result != nil {
			result.Operation = b.name
			report.Results = append(report.Results, result)

			// Print inline result
			if result.Throughput > 0 {
				fmt.Printf(" %s @ %s\n", formatOps(result.OpsPerSec), formatThroughputValue(result.Throughput))
			} else {
				fmt.Printf(" %s\n", formatOps(result.OpsPerSec))
			}
		} else {
			fmt.Printf(" (skipped)\n")
		}
	}

	// Save reports
	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	// JSON
	jsonPath := filepath.Join(cfg.OutputDir, "limits.json")
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	if err := os.WriteFile(jsonPath, data, 0644); err != nil {
		return fmt.Errorf("write json: %w", err)
	}

	// Markdown
	mdPath := filepath.Join(cfg.OutputDir, "limits.md")
	md := generateLimitsMarkdown(report)
	if err := os.WriteFile(mdPath, []byte(md), 0644); err != nil {
		return fmt.Errorf("write markdown: %w", err)
	}

	fmt.Printf("\nReports saved to %s\n", cfg.OutputDir)
	return nil
}

// =============================================================================
// Adaptive benchmark helper (shared across all limits benchmarks)
// =============================================================================

type limitsCollector struct {
	samples []time.Duration
	size    int
}

func newLimitsCollector(size int) *limitsCollector {
	return &limitsCollector{
		samples: make([]time.Duration, 0, 4096),
		size:    size,
	}
}

func (c *limitsCollector) record(d time.Duration) {
	c.samples = append(c.samples, d)
}

func (c *limitsCollector) result() *LimitsResult {
	n := len(c.samples)
	if n == 0 {
		return nil
	}

	sorted := make([]time.Duration, n)
	copy(sorted, c.samples)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	var total time.Duration
	for _, s := range sorted {
		total += s
	}

	avg := total / time.Duration(n)
	p50 := sorted[n*50/100]
	p99idx := n * 99 / 100
	if p99idx >= n {
		p99idx = n - 1
	}
	p99 := sorted[p99idx]

	r := &LimitsResult{
		Iterations: n,
		TotalNs:    total.Nanoseconds(),
		OpsPerSec:  float64(n) / total.Seconds(),
		AvgNs:      avg.Nanoseconds(),
		P50Ns:      p50.Nanoseconds(),
		P99Ns:      p99.Nanoseconds(),
		ObjectSize: c.size,
	}

	if c.size > 0 {
		totalBytes := int64(n) * int64(c.size)
		r.Throughput = float64(totalBytes) / (1024 * 1024) / total.Seconds()
	}

	return r
}

// runAdaptive runs the benchmark function adaptively until benchTime is reached.
func runAdaptive(ctx context.Context, benchTime time.Duration, fn func()) *limitsCollector {
	c := newLimitsCollector(0)
	deadline := time.Now().Add(benchTime)

	// Warmup: 3 iterations
	for i := 0; i < 3; i++ {
		fn()
	}

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return c
		default:
		}

		start := time.Now()
		fn()
		c.record(time.Since(start))
	}
	return c
}

// runAdaptiveWithSize runs the benchmark and sets size for throughput calculation.
func runAdaptiveWithSize(ctx context.Context, benchTime time.Duration, size int, fn func()) *LimitsResult {
	c := newLimitsCollector(size)
	deadline := time.Now().Add(benchTime)

	// Warmup: 3 iterations
	for i := 0; i < 3; i++ {
		fn()
	}

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return c.result()
		default:
		}

		start := time.Now()
		fn()
		c.record(time.Since(start))
	}
	return c.result()
}

// =============================================================================
// Memory benchmarks
// =============================================================================

func benchMemcpy(ctx context.Context, benchTime time.Duration, size int) *LimitsResult {
	src := make([]byte, size)
	dst := make([]byte, size)
	// Fill src to ensure pages are faulted in
	for i := range src {
		src[i] = byte(i)
	}
	// Warm dst
	for i := range dst {
		dst[i] = 0
	}

	return runAdaptiveWithSize(ctx, benchTime, size, func() {
		copy(dst, src)
	})
}

// =============================================================================
// Mmap benchmarks
// =============================================================================

func benchMmapRead(ctx context.Context, benchTime time.Duration, dir string, size int) *LimitsResult {
	path := filepath.Join(dir, fmt.Sprintf("mmap-read-%d", size))

	// Create and populate file
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return nil
	}
	defer os.Remove(path)

	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	// Mmap the file
	mapped, err := syscall.Mmap(int(f.Fd()), 0, size, syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return nil
	}
	defer syscall.Munmap(mapped)

	// Warm all pages
	dst := make([]byte, size)
	copy(dst, mapped)

	return runAdaptiveWithSize(ctx, benchTime, size, func() {
		copy(dst, mapped)
		runtime.KeepAlive(dst)
	})
}

func benchMmapWrite(ctx context.Context, benchTime time.Duration, dir string, size int) *LimitsResult {
	path := filepath.Join(dir, fmt.Sprintf("mmap-write-%d", size))

	// Create file of the right size
	f, err := os.Create(path)
	if err != nil {
		return nil
	}
	if err := f.Truncate(int64(size * 2)); err != nil { // 2x so we have room to write
		f.Close()
		return nil
	}

	// Mmap writable
	mapped, err := syscall.Mmap(int(f.Fd()), 0, size*2, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		f.Close()
		return nil
	}

	// Warm pages by writing once
	src := make([]byte, size)
	for i := range src {
		src[i] = byte(i)
	}
	copy(mapped[:size], src)

	defer func() {
		syscall.Munmap(mapped)
		f.Close()
		os.Remove(path)
	}()

	return runAdaptiveWithSize(ctx, benchTime, size, func() {
		copy(mapped[:size], src)
	})
}

func benchMmapPageFault(ctx context.Context, benchTime time.Duration, dir string, size int) *LimitsResult {
	// Each iteration: create sparse file, mmap, write (fault), munmap, close, remove
	// This measures the real cost of writing to a sparse mmap region.
	src := make([]byte, size)
	for i := range src {
		src[i] = byte(i)
	}

	c := newLimitsCollector(size)
	deadline := time.Now().Add(benchTime)

	var counter int64
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return c.result()
		default:
		}

		counter++
		path := filepath.Join(dir, fmt.Sprintf("mmap-fault-%d-%d", size, counter))

		f, err := os.Create(path)
		if err != nil {
			continue
		}
		// Sparse file: extend without writing
		if err := f.Truncate(int64(size)); err != nil {
			f.Close()
			os.Remove(path)
			continue
		}

		mapped, err := syscall.Mmap(int(f.Fd()), 0, size, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
		if err != nil {
			f.Close()
			os.Remove(path)
			continue
		}

		// Time only the write (which triggers page faults)
		start := time.Now()
		copy(mapped, src)
		c.record(time.Since(start))

		syscall.Munmap(mapped)
		f.Close()
		os.Remove(path)
	}

	return c.result()
}

// =============================================================================
// Syscall benchmarks
// =============================================================================

func benchPwrite(ctx context.Context, benchTime time.Duration, dir string, size int) *LimitsResult {
	path := filepath.Join(dir, fmt.Sprintf("pwrite-%d", size))

	// Pre-allocate file
	f, err := os.Create(path)
	if err != nil {
		return nil
	}
	defer func() {
		f.Close()
		os.Remove(path)
	}()

	// Pre-allocate by writing zeros
	buf := make([]byte, size)
	if _, err := f.Write(buf); err != nil {
		return nil
	}

	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i)
	}

	return runAdaptiveWithSize(ctx, benchTime, size, func() {
		syscall.Pwrite(int(f.Fd()), data, 0)
	})
}

func benchPread(ctx context.Context, benchTime time.Duration, dir string, size int) *LimitsResult {
	path := filepath.Join(dir, fmt.Sprintf("pread-%d", size))

	// Create and populate file
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return nil
	}
	defer os.Remove(path)

	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	// Warm by reading once
	buf := make([]byte, size)
	syscall.Pread(int(f.Fd()), buf, 0)

	return runAdaptiveWithSize(ctx, benchTime, size, func() {
		syscall.Pread(int(f.Fd()), buf, 0)
		runtime.KeepAlive(buf)
	})
}

func benchPwriteSparse(ctx context.Context, benchTime time.Duration, dir string, size int) *LimitsResult {
	// Each iteration writes to a NEW sparse region (simulating turtle's append to sparse file).
	path := filepath.Join(dir, fmt.Sprintf("pwrite-sparse-%d", size))

	f, err := os.Create(path)
	if err != nil {
		return nil
	}
	defer func() {
		f.Close()
		os.Remove(path)
	}()

	// Extend sparse file to large size
	totalSize := int64(10) * 1024 * 1024 * 1024 // 10 GB sparse
	if err := f.Truncate(totalSize); err != nil {
		return nil
	}

	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i)
	}

	c := newLimitsCollector(size)
	deadline := time.Now().Add(benchTime)
	var offset int64

	// Warmup
	for i := 0; i < 3; i++ {
		syscall.Pwrite(int(f.Fd()), data, offset)
		offset += int64(size)
	}

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return c.result()
		default:
		}

		if offset+int64(size) > totalSize {
			// Reset - truncate again
			offset = 0
			f.Truncate(totalSize)
		}

		start := time.Now()
		syscall.Pwrite(int(f.Fd()), data, offset)
		c.record(time.Since(start))
		offset += int64(size)
	}

	return c.result()
}

// =============================================================================
// File operation benchmarks
// =============================================================================

func benchFileCreate(ctx context.Context, benchTime time.Duration, dir string) *LimitsResult {
	subDir := filepath.Join(dir, "file-create")
	os.MkdirAll(subDir, 0755)
	defer os.RemoveAll(subDir)

	c := newLimitsCollector(0)
	deadline := time.Now().Add(benchTime)
	var counter int64

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return c.result()
		default:
		}

		counter++
		path := filepath.Join(subDir, fmt.Sprintf("f%d", counter))

		start := time.Now()
		f, err := os.Create(path)
		if err == nil {
			f.Close()
		}
		c.record(time.Since(start))
	}

	return c.result()
}

func benchFileCreateWrite(ctx context.Context, benchTime time.Duration, dir string, size int) *LimitsResult {
	subDir := filepath.Join(dir, fmt.Sprintf("file-create-write-%d", size))
	os.MkdirAll(subDir, 0755)
	defer os.RemoveAll(subDir)

	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i)
	}

	c := newLimitsCollector(size)
	deadline := time.Now().Add(benchTime)
	var counter int64

	// Warmup
	for i := 0; i < 3; i++ {
		counter++
		path := filepath.Join(subDir, fmt.Sprintf("w%d", counter))
		os.WriteFile(path, data, 0644)
	}

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return c.result()
		default:
		}

		counter++
		path := filepath.Join(subDir, fmt.Sprintf("w%d", counter))

		start := time.Now()
		os.WriteFile(path, data, 0644)
		c.record(time.Since(start))
	}

	return c.result()
}

func benchFileDelete(ctx context.Context, benchTime time.Duration, dir string) *LimitsResult {
	subDir := filepath.Join(dir, "file-delete")
	os.MkdirAll(subDir, 0755)
	defer os.RemoveAll(subDir)

	c := newLimitsCollector(0)
	deadline := time.Now().Add(benchTime)
	var counter int64

	// Pre-create a batch of files
	batchSize := 10000
	paths := make([]string, batchSize)
	for i := 0; i < batchSize; i++ {
		counter++
		paths[i] = filepath.Join(subDir, fmt.Sprintf("d%d", counter))
		f, _ := os.Create(paths[i])
		if f != nil {
			f.Close()
		}
	}
	idx := 0

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return c.result()
		default:
		}

		if idx >= len(paths) {
			// Create more files
			for i := 0; i < batchSize; i++ {
				counter++
				paths[i] = filepath.Join(subDir, fmt.Sprintf("d%d", counter))
				f, _ := os.Create(paths[i])
				if f != nil {
					f.Close()
				}
			}
			idx = 0
		}

		start := time.Now()
		os.Remove(paths[idx])
		c.record(time.Since(start))
		idx++
	}

	return c.result()
}

func benchFileStat(ctx context.Context, benchTime time.Duration, dir string) *LimitsResult {
	path := filepath.Join(dir, "stat-target")
	os.WriteFile(path, []byte("hello"), 0644)
	defer os.Remove(path)

	c := newLimitsCollector(0)
	deadline := time.Now().Add(benchTime)

	// Warmup
	for i := 0; i < 10; i++ {
		os.Stat(path)
	}

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return c.result()
		default:
		}

		start := time.Now()
		os.Stat(path)
		c.record(time.Since(start))
	}

	return c.result()
}

func benchDirCreate(ctx context.Context, benchTime time.Duration, dir string) *LimitsResult {
	subDir := filepath.Join(dir, "dir-create")
	os.MkdirAll(subDir, 0755)
	defer os.RemoveAll(subDir)

	c := newLimitsCollector(0)
	deadline := time.Now().Add(benchTime)
	var counter int64

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return c.result()
		default:
		}

		counter++
		path := filepath.Join(subDir, fmt.Sprintf("a/b/c/%d", counter))

		start := time.Now()
		os.MkdirAll(path, 0755)
		c.record(time.Since(start))
	}

	return c.result()
}

// =============================================================================
// Hash benchmarks
// =============================================================================

func benchCRC32(ctx context.Context, benchTime time.Duration, size int) *LimitsResult {
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i)
	}
	table := crc32.MakeTable(crc32.Castagnoli)

	var sink uint32
	r := runAdaptiveWithSize(ctx, benchTime, size, func() {
		sink = crc32.Update(0, table, data)
	})
	runtime.KeepAlive(sink)
	return r
}

func benchFNV32(ctx context.Context, benchTime time.Duration, size int) *LimitsResult {
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i)
	}

	var sink uint32
	r := runAdaptiveWithSize(ctx, benchTime, size, func() {
		const offset32 = 2166136261
		const prime32 = 16777619
		h := uint32(offset32)
		for i := 0; i < len(data); i++ {
			h ^= uint32(data[i])
			h *= prime32
		}
		sink = h
	})
	runtime.KeepAlive(sink)
	return r
}

// =============================================================================
// Atomic operation benchmarks
// =============================================================================

func benchAtomicAdd(ctx context.Context, benchTime time.Duration) *LimitsResult {
	var counter int64
	c := runAdaptive(ctx, benchTime, func() {
		atomic.AddInt64(&counter, 1)
	})
	runtime.KeepAlive(counter)
	return c.result()
}

func benchAtomicCAS(ctx context.Context, benchTime time.Duration) *LimitsResult {
	var counter int64
	c := runAdaptive(ctx, benchTime, func() {
		for {
			old := atomic.LoadInt64(&counter)
			if atomic.CompareAndSwapInt64(&counter, old, old+1) {
				break
			}
		}
	})
	runtime.KeepAlive(counter)
	return c.result()
}

func benchMutexLock(ctx context.Context, benchTime time.Duration) *LimitsResult {
	var mu sync.Mutex
	var counter int64
	c := runAdaptive(ctx, benchTime, func() {
		mu.Lock()
		counter++
		mu.Unlock()
	})
	runtime.KeepAlive(counter)
	return c.result()
}

func benchRWMutexRLock(ctx context.Context, benchTime time.Duration) *LimitsResult {
	var mu sync.RWMutex
	var counter int64
	c := runAdaptive(ctx, benchTime, func() {
		mu.RLock()
		_ = counter
		mu.RUnlock()
	})
	return c.result()
}

// =============================================================================
// Allocation benchmarks
// =============================================================================

//go:noinline
func escapingSlice(size int) []byte {
	return make([]byte, size)
}

func benchMakeSlice(ctx context.Context, benchTime time.Duration, size int) *LimitsResult {
	c := runAdaptive(ctx, benchTime, func() {
		b := escapingSlice(size)
		runtime.KeepAlive(b)
	})
	return c.result()
}

func benchSyncPool(ctx context.Context, benchTime time.Duration) *LimitsResult {
	pool := sync.Pool{
		New: func() any {
			b := make([]byte, 4096)
			return &b
		},
	}
	// Prime the pool
	for i := 0; i < 100; i++ {
		pool.Put(pool.New())
	}

	c := runAdaptive(ctx, benchTime, func() {
		bp := pool.Get().(*[]byte)
		_ = (*bp)[0]
		pool.Put(bp)
	})
	return c.result()
}

// =============================================================================
// Platform detection
// =============================================================================

func detectCPU() string {
	if runtime.GOOS == "darwin" {
		out, err := exec.Command("sysctl", "-n", "machdep.cpu.brand_string").Output()
		if err == nil {
			return strings.TrimSpace(string(out))
		}
	}
	if runtime.GOOS == "linux" {
		out, err := exec.Command("sh", "-c", "grep 'model name' /proc/cpuinfo | head -1 | cut -d: -f2").Output()
		if err == nil && len(out) > 0 {
			return strings.TrimSpace(string(out))
		}
	}
	return runtime.GOARCH
}

func detectRAMGB() int {
	if runtime.GOOS == "darwin" {
		out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
		if err == nil {
			val, err := strconv.ParseUint(strings.TrimSpace(string(out)), 10, 64)
			if err == nil {
				return int(val / (1024 * 1024 * 1024))
			}
		}
	}
	if runtime.GOOS == "linux" {
		out, err := exec.Command("sh", "-c", "grep MemTotal /proc/meminfo | awk '{print $2}'").Output()
		if err == nil {
			val, err := strconv.ParseUint(strings.TrimSpace(string(out)), 10, 64)
			if err == nil {
				return int(val / (1024 * 1024)) // MemTotal is in kB
			}
		}
	}
	return 0
}

// =============================================================================
// Markdown report generation
// =============================================================================

func generateLimitsMarkdown(report *LimitsReport) string {
	var sb strings.Builder

	sb.WriteString("# Physical Limits Report\n\n")
	sb.WriteString(fmt.Sprintf("**Machine:** %s, %d GB RAM, %s\n", report.CPU, report.RAMGB, report.Platform))
	sb.WriteString(fmt.Sprintf("**Generated:** %s\n\n", report.Timestamp))

	// Group results by category
	groups := map[string][]*LimitsResult{
		"Memory":      {},
		"Mmap":        {},
		"Syscall":     {},
		"File":        {},
		"Hash":        {},
		"Atomic":      {},
		"Allocation":  {},
	}
	groupOrder := []string{"Memory", "Mmap", "Syscall", "File", "Hash", "Atomic", "Allocation"}

	for _, r := range report.Results {
		switch {
		case strings.HasPrefix(r.Operation, "Memcpy"):
			groups["Memory"] = append(groups["Memory"], r)
		case strings.HasPrefix(r.Operation, "Mmap"):
			groups["Mmap"] = append(groups["Mmap"], r)
		case strings.HasPrefix(r.Operation, "Pwrite") || strings.HasPrefix(r.Operation, "Pread"):
			groups["Syscall"] = append(groups["Syscall"], r)
		case strings.HasPrefix(r.Operation, "File") || strings.HasPrefix(r.Operation, "Dir"):
			groups["File"] = append(groups["File"], r)
		case strings.HasPrefix(r.Operation, "CRC") || strings.HasPrefix(r.Operation, "FNV"):
			groups["Hash"] = append(groups["Hash"], r)
		case strings.HasPrefix(r.Operation, "Atomic") || strings.HasPrefix(r.Operation, "Mutex") || strings.HasPrefix(r.Operation, "RWMutex"):
			groups["Atomic"] = append(groups["Atomic"], r)
		case strings.HasPrefix(r.Operation, "Make") || strings.HasPrefix(r.Operation, "SyncPool"):
			groups["Allocation"] = append(groups["Allocation"], r)
		}
	}

	for _, group := range groupOrder {
		results := groups[group]
		if len(results) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("## %s\n\n", group))
		sb.WriteString("| Operation | Throughput | ops/sec | Latency P50 | Latency P99 |\n")
		sb.WriteString("|-----------|-----------|---------|-------------|-------------|\n")

		for _, r := range results {
			throughput := "-"
			if r.Throughput > 0 {
				throughput = formatThroughputValue(r.Throughput)
			}

			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n",
				r.Operation,
				throughput,
				formatOps(r.OpsPerSec),
				formatNanos(r.P50Ns),
				formatNanos(r.P99Ns),
			))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("---\n\n")
	sb.WriteString("*Generated by storage benchmark physical limits tool*\n")

	return sb.String()
}

// =============================================================================
// Formatting helpers
// =============================================================================

func formatThroughputValue(mbps float64) string {
	if mbps >= 1000*1000 {
		return fmt.Sprintf("%.1f TB/s", mbps/(1000*1000))
	}
	if mbps >= 1000 {
		return fmt.Sprintf("%.1f GB/s", mbps/1000)
	}
	return fmt.Sprintf("%.1f MB/s", mbps)
}

func formatOps(ops float64) string {
	if ops >= 1000*1000*1000 {
		return fmt.Sprintf("%.1fG ops/s", ops/(1000*1000*1000))
	}
	if ops >= 1000*1000 {
		return fmt.Sprintf("%.1fM ops/s", ops/(1000*1000))
	}
	if ops >= 1000 {
		return fmt.Sprintf("%.1fK ops/s", ops/1000)
	}
	return fmt.Sprintf("%.0f ops/s", ops)
}

func formatNanos(ns int64) string {
	if ns >= 1000*1000*1000 {
		return fmt.Sprintf("%.2fs", float64(ns)/1e9)
	}
	if ns >= 1000*1000 {
		return fmt.Sprintf("%.1fms", float64(ns)/1e6)
	}
	if ns >= 1000 {
		return fmt.Sprintf("%.1fus", float64(ns)/1e3)
	}
	return fmt.Sprintf("%dns", ns)
}
