package bench

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strings"
	"time"
)

// ProfilerConfig holds configuration for profiling.
type ProfilerConfig struct {
	// Enabled enables profiler data collection.
	Enabled bool
	// Endpoint is the pprof server endpoint (e.g., "http://localhost:9200").
	Endpoint string
	// OutputDir is the directory to write profiler reports.
	OutputDir string
	// Duration is the CPU profile collection duration.
	Duration time.Duration
}

// DefaultProfilerConfig returns default profiler configuration.
func DefaultProfilerConfig() *ProfilerConfig {
	return &ProfilerConfig{
		Enabled:   true,
		Endpoint:  "http://localhost:9200",
		OutputDir: "./pkg/storage/report/profiler",
		Duration:  30 * time.Second,
	}
}

// Profiler captures and stores pprof data.
type Profiler struct {
	config *ProfilerConfig
	client *http.Client
	logger func(format string, args ...any)
}

// NewProfiler creates a new profiler instance.
func NewProfiler(cfg *ProfilerConfig) *Profiler {
	if cfg == nil {
		cfg = DefaultProfilerConfig()
	}
	return &Profiler{
		config: cfg,
		client: &http.Client{
			Timeout: 5 * time.Minute, // Long timeout for profile collection
		},
		logger: func(format string, args ...any) { fmt.Printf(format+"\n", args...) },
	}
}

// SetLogger sets a custom logger.
func (p *Profiler) SetLogger(fn func(format string, args ...any)) {
	p.logger = fn
}

// ProfileType represents different types of profiles.
type ProfileType string

const (
	ProfileCPU          ProfileType = "profile"
	ProfileHeap         ProfileType = "heap"
	ProfileGoroutine    ProfileType = "goroutine"
	ProfileBlock        ProfileType = "block"
	ProfileMutex        ProfileType = "mutex"
	ProfileAllocs       ProfileType = "allocs"
	ProfileThreadcreate ProfileType = "threadcreate"
	ProfileTrace        ProfileType = "trace"
)

// CaptureAll captures all profile types and writes them to the output directory.
func (p *Profiler) CaptureAll(ctx context.Context, prefix string) error {
	if !p.config.Enabled {
		return nil
	}

	// Ensure output directory exists
	if err := os.MkdirAll(p.config.OutputDir, 0755); err != nil {
		return fmt.Errorf("create profiler output dir: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	baseDir := filepath.Join(p.config.OutputDir, fmt.Sprintf("%s_%s", prefix, timestamp))
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return fmt.Errorf("create profiler session dir: %w", err)
	}

	p.logger("Capturing profiler data to %s", baseDir)

	var results []ProfileResult

	// Capture CPU profile (this takes time)
	cpuResult := p.captureCPUProfile(ctx, baseDir)
	results = append(results, cpuResult)
	if cpuResult.Error == "" {
		p.logger("  CPU profile: %s (%.2f KB)", cpuResult.File, float64(cpuResult.Size)/1024)
	} else {
		p.logger("  CPU profile: failed (%s)", cpuResult.Error)
	}

	// Capture heap snapshot profiles (instant)
	heapProfiles := []ProfileType{
		ProfileHeap,
		ProfileGoroutine,
		ProfileBlock,
		ProfileMutex,
		ProfileAllocs,
		ProfileThreadcreate,
	}

	for _, pt := range heapProfiles {
		result := p.captureProfile(ctx, baseDir, pt)
		results = append(results, result)
		if result.Error == "" {
			p.logger("  %s profile: %s (%.2f KB)", pt, result.File, float64(result.Size)/1024)
		} else {
			p.logger("  %s profile: failed (%s)", pt, result.Error)
		}
	}

	// Write summary JSON
	summary := ProfileSummary{
		Timestamp: time.Now(),
		Endpoint:  p.config.Endpoint,
		Duration:  p.config.Duration,
		Profiles:  results,
	}

	summaryPath := filepath.Join(baseDir, "summary.json")
	summaryData, _ := json.MarshalIndent(summary, "", "  ")
	if err := os.WriteFile(summaryPath, summaryData, 0644); err != nil {
		p.logger("  Warning: failed to write summary: %v", err)
	}

	// Write human-readable report
	reportPath := filepath.Join(baseDir, "report.md")
	if err := p.writeHumanReport(reportPath, summary); err != nil {
		p.logger("  Warning: failed to write report: %v", err)
	}

	return nil
}

// CaptureCPU captures only CPU profile.
func (p *Profiler) CaptureCPU(ctx context.Context, outputPath string) error {
	if !p.config.Enabled {
		return nil
	}

	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	url := fmt.Sprintf("%s/debug/pprof/profile?seconds=%d", p.config.Endpoint, int(p.config.Duration.Seconds()))
	return p.fetchAndSave(ctx, url, outputPath)
}

// CaptureHeap captures a heap profile.
func (p *Profiler) CaptureHeap(ctx context.Context, outputPath string) error {
	if !p.config.Enabled {
		return nil
	}

	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	url := fmt.Sprintf("%s/debug/pprof/heap", p.config.Endpoint)
	return p.fetchAndSave(ctx, url, outputPath)
}

func (p *Profiler) captureCPUProfile(ctx context.Context, baseDir string) ProfileResult {
	result := ProfileResult{
		Type:      string(ProfileCPU),
		Timestamp: time.Now(),
	}

	outputPath := filepath.Join(baseDir, "cpu.pprof")
	url := fmt.Sprintf("%s/debug/pprof/profile?seconds=%d", p.config.Endpoint, int(p.config.Duration.Seconds()))

	if err := p.fetchAndSave(ctx, url, outputPath); err != nil {
		result.Error = err.Error()
		return result
	}

	info, err := os.Stat(outputPath)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	result.File = outputPath
	result.Size = info.Size()
	return result
}

func (p *Profiler) captureProfile(ctx context.Context, baseDir string, pt ProfileType) ProfileResult {
	result := ProfileResult{
		Type:      string(pt),
		Timestamp: time.Now(),
	}

	outputPath := filepath.Join(baseDir, fmt.Sprintf("%s.pprof", pt))
	url := fmt.Sprintf("%s/debug/pprof/%s", p.config.Endpoint, pt)

	// Add debug=0 for binary format (except trace)
	if pt != ProfileTrace {
		url += "?debug=0"
	}

	if err := p.fetchAndSave(ctx, url, outputPath); err != nil {
		result.Error = err.Error()
		return result
	}

	info, err := os.Stat(outputPath)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	result.File = outputPath
	result.Size = info.Size()
	return result
}

func (p *Profiler) fetchAndSave(ctx context.Context, url, outputPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

func (p *Profiler) writeHumanReport(path string, summary ProfileSummary) error {
	var sb strings.Builder

	sb.WriteString("# Profiler Report\n\n")
	sb.WriteString(fmt.Sprintf("**Generated:** %s\n\n", summary.Timestamp.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("**Endpoint:** %s\n\n", summary.Endpoint))
	sb.WriteString(fmt.Sprintf("**CPU Profile Duration:** %s\n\n", summary.Duration))

	sb.WriteString("## Captured Profiles\n\n")
	sb.WriteString("| Type | File | Size | Status |\n")
	sb.WriteString("|------|------|------|--------|\n")

	for _, profile := range summary.Profiles {
		status := "✅ OK"
		if profile.Error != "" {
			status = "❌ " + profile.Error
		}
		sizeStr := "-"
		if profile.Size > 0 {
			sizeStr = fmt.Sprintf("%.2f KB", float64(profile.Size)/1024)
		}
		fileName := filepath.Base(profile.File)
		if fileName == "" || fileName == "." {
			fileName = "-"
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", profile.Type, fileName, sizeStr, status))
	}

	sb.WriteString("\n## Usage\n\n")
	sb.WriteString("To analyze CPU profile:\n")
	sb.WriteString("```bash\n")
	sb.WriteString("go tool pprof -http=:8080 cpu.pprof\n")
	sb.WriteString("```\n\n")

	sb.WriteString("To analyze heap profile:\n")
	sb.WriteString("```bash\n")
	sb.WriteString("go tool pprof -http=:8080 heap.pprof\n")
	sb.WriteString("```\n\n")

	sb.WriteString("To compare two profiles:\n")
	sb.WriteString("```bash\n")
	sb.WriteString("go tool pprof -http=:8080 -base=before.pprof after.pprof\n")
	sb.WriteString("```\n")

	return os.WriteFile(path, []byte(sb.String()), 0644)
}

// ProfileResult holds the result of capturing a single profile.
type ProfileResult struct {
	Type      string    `json:"type"`
	File      string    `json:"file,omitempty"`
	Size      int64     `json:"size,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Error     string    `json:"error,omitempty"`
}

// ProfileSummary holds a summary of all captured profiles.
type ProfileSummary struct {
	Timestamp time.Time       `json:"timestamp"`
	Endpoint  string          `json:"endpoint"`
	Duration  time.Duration   `json:"duration"`
	Profiles  []ProfileResult `json:"profiles"`
}

// ---------------------------------------------------------------------------
// In-process profiler (for embedded drivers)
// ---------------------------------------------------------------------------

// InProcessProfiler captures pprof profiles directly in the current process.
// Unlike the HTTP-based Profiler, this works for embedded drivers without a pprof server.
type InProcessProfiler struct {
	outputDir string
	logger    func(format string, args ...any)
	cpuFile   *os.File
}

// NewInProcessProfiler creates a profiler that captures profiles in the current process.
func NewInProcessProfiler(outputDir string) *InProcessProfiler {
	return &InProcessProfiler{
		outputDir: outputDir,
		logger:    func(format string, args ...any) { fmt.Printf(format+"\n", args...) },
	}
}

// SetLogger sets a custom logger for the in-process profiler.
func (p *InProcessProfiler) SetLogger(fn func(format string, args ...any)) {
	p.logger = fn
}

// StartCPU begins CPU profiling to a file.
func (p *InProcessProfiler) StartCPU(driverName string) error {
	dir := filepath.Join(p.outputDir, driverName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create profile dir: %w", err)
	}

	f, err := os.Create(filepath.Join(dir, "cpu.pprof"))
	if err != nil {
		return fmt.Errorf("create cpu profile: %w", err)
	}
	p.cpuFile = f

	runtime.SetMutexProfileFraction(5)
	runtime.SetBlockProfileRate(1)

	if err := pprof.StartCPUProfile(f); err != nil {
		f.Close()
		p.cpuFile = nil
		return fmt.Errorf("start cpu profile: %w", err)
	}
	p.logger("  Profile: CPU profiling started → %s", f.Name())
	return nil
}

// StopAndCapture stops CPU profiling and captures heap/goroutine/mutex/block profiles.
// Returns a ProfileAnalysis with parsed top consumers from each profile.
func (p *InProcessProfiler) StopAndCapture(driverName string) *ProfileAnalysis {
	analysis := &ProfileAnalysis{Driver: driverName}
	dir := filepath.Join(p.outputDir, driverName)
	os.MkdirAll(dir, 0755)

	// Stop CPU profile
	if p.cpuFile != nil {
		pprof.StopCPUProfile()
		p.cpuFile.Close()
		analysis.CPUProfilePath = p.cpuFile.Name()
		if info, err := os.Stat(p.cpuFile.Name()); err == nil {
			analysis.CPUProfileSize = info.Size()
		}
		p.cpuFile = nil
		p.logger("  Profile: CPU profile saved (%.1f KB)", float64(analysis.CPUProfileSize)/1024)
	}

	// Capture heap profile
	if f, err := os.Create(filepath.Join(dir, "heap.pprof")); err == nil {
		runtime.GC() // get up-to-date statistics
		pprof.WriteHeapProfile(f)
		f.Close()
		analysis.HeapProfilePath = f.Name()
		if info, err := os.Stat(f.Name()); err == nil {
			analysis.HeapProfileSize = info.Size()
		}
		p.logger("  Profile: heap profile saved (%.1f KB)", float64(analysis.HeapProfileSize)/1024)
	}

	// Capture goroutine profile
	if f, err := os.Create(filepath.Join(dir, "goroutine.pprof")); err == nil {
		prof := pprof.Lookup("goroutine")
		if prof != nil {
			prof.WriteTo(f, 0)
		}
		f.Close()
		analysis.GoroutineProfilePath = f.Name()
		// Also get goroutine count
		analysis.GoroutineCount = runtime.NumGoroutine()
		p.logger("  Profile: goroutine profile saved (%d goroutines)", analysis.GoroutineCount)
	}

	// Capture mutex profile
	if f, err := os.Create(filepath.Join(dir, "mutex.pprof")); err == nil {
		prof := pprof.Lookup("mutex")
		if prof != nil {
			prof.WriteTo(f, 0)
		}
		f.Close()
		analysis.MutexProfilePath = f.Name()
		p.logger("  Profile: mutex profile saved")
	}

	// Capture block profile
	if f, err := os.Create(filepath.Join(dir, "block.pprof")); err == nil {
		prof := pprof.Lookup("block")
		if prof != nil {
			prof.WriteTo(f, 0)
		}
		f.Close()
		analysis.BlockProfilePath = f.Name()
		p.logger("  Profile: block profile saved")
	}

	// Capture allocs profile
	if f, err := os.Create(filepath.Join(dir, "allocs.pprof")); err == nil {
		prof := pprof.Lookup("allocs")
		if prof != nil {
			prof.WriteTo(f, 0)
		}
		f.Close()
		analysis.AllocsProfilePath = f.Name()
		p.logger("  Profile: allocs profile saved")
	}

	// Capture runtime memory stats
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	analysis.HeapInUse = ms.HeapInuse
	analysis.HeapAlloc = ms.HeapAlloc
	analysis.HeapSys = ms.HeapSys
	analysis.HeapObjects = ms.HeapObjects
	analysis.TotalAlloc = ms.TotalAlloc
	analysis.NumGC = ms.NumGC
	analysis.GCPauseTotalNs = ms.PauseTotalNs
	analysis.StackInUse = ms.StackInuse

	// Run go tool pprof to extract top consumers
	analysis.CPUTop = runPprofTop(filepath.Join(dir, "cpu.pprof"), 10)
	analysis.HeapTop = runPprofTop(filepath.Join(dir, "heap.pprof"), 10)
	analysis.AllocsTop = runPprofTop(filepath.Join(dir, "allocs.pprof"), 10)

	// Reset profiling rates
	runtime.SetMutexProfileFraction(0)
	runtime.SetBlockProfileRate(0)

	return analysis
}

// runPprofTop runs `go tool pprof -top` on a profile and returns the output.
func runPprofTop(profilePath string, n int) string {
	if _, err := os.Stat(profilePath); err != nil {
		return ""
	}
	cmd := exec.Command("go", "tool", "pprof", "-top", fmt.Sprintf("-nodecount=%d", n), profilePath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("(pprof error: %v)", err)
	}
	return string(out)
}

// ProfileAnalysis holds parsed profiling data for a driver.
type ProfileAnalysis struct {
	Driver string `json:"driver"`

	// Profile file paths
	CPUProfilePath       string `json:"cpu_profile_path,omitempty"`
	HeapProfilePath      string `json:"heap_profile_path,omitempty"`
	GoroutineProfilePath string `json:"goroutine_profile_path,omitempty"`
	MutexProfilePath     string `json:"mutex_profile_path,omitempty"`
	BlockProfilePath     string `json:"block_profile_path,omitempty"`
	AllocsProfilePath    string `json:"allocs_profile_path,omitempty"`

	// Profile sizes
	CPUProfileSize  int64 `json:"cpu_profile_size,omitempty"`
	HeapProfileSize int64 `json:"heap_profile_size,omitempty"`

	// Runtime stats
	GoroutineCount int    `json:"goroutine_count,omitempty"`
	HeapInUse      uint64 `json:"heap_in_use,omitempty"`
	HeapAlloc      uint64 `json:"heap_alloc,omitempty"`
	HeapSys        uint64 `json:"heap_sys,omitempty"`
	HeapObjects    uint64 `json:"heap_objects,omitempty"`
	TotalAlloc     uint64 `json:"total_alloc,omitempty"`
	NumGC          uint32 `json:"num_gc,omitempty"`
	GCPauseTotalNs uint64 `json:"gc_pause_total_ns,omitempty"`
	StackInUse     uint64 `json:"stack_in_use,omitempty"`

	// Parsed top consumers
	CPUTop    string `json:"cpu_top,omitempty"`
	HeapTop   string `json:"heap_top,omitempty"`
	AllocsTop string `json:"allocs_top,omitempty"`
}

// FormatMarkdown returns markdown-formatted profiling analysis.
func (pa *ProfileAnalysis) FormatMarkdown() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("### Profile: %s\n\n", pa.Driver))

	sb.WriteString("**Runtime Memory Stats:**\n\n")
	sb.WriteString("| Metric | Value |\n|--------|-------|\n")
	sb.WriteString(fmt.Sprintf("| Heap In Use | %.1f MB |\n", float64(pa.HeapInUse)/(1024*1024)))
	sb.WriteString(fmt.Sprintf("| Heap Alloc | %.1f MB |\n", float64(pa.HeapAlloc)/(1024*1024)))
	sb.WriteString(fmt.Sprintf("| Heap Sys | %.1f MB |\n", float64(pa.HeapSys)/(1024*1024)))
	sb.WriteString(fmt.Sprintf("| Heap Objects | %d |\n", pa.HeapObjects))
	sb.WriteString(fmt.Sprintf("| Total Alloc | %.1f MB |\n", float64(pa.TotalAlloc)/(1024*1024)))
	sb.WriteString(fmt.Sprintf("| Stack In Use | %.1f MB |\n", float64(pa.StackInUse)/(1024*1024)))
	sb.WriteString(fmt.Sprintf("| GC Cycles | %d |\n", pa.NumGC))
	sb.WriteString(fmt.Sprintf("| GC Pause Total | %.1f ms |\n", float64(pa.GCPauseTotalNs)/1e6))
	sb.WriteString(fmt.Sprintf("| Goroutines | %d |\n", pa.GoroutineCount))
	sb.WriteString("\n")

	if pa.CPUTop != "" {
		sb.WriteString("**CPU Profile (top consumers):**\n\n```\n")
		sb.WriteString(pa.CPUTop)
		sb.WriteString("```\n\n")
	}

	if pa.HeapTop != "" {
		sb.WriteString("**Heap Profile (top consumers):**\n\n```\n")
		sb.WriteString(pa.HeapTop)
		sb.WriteString("```\n\n")
	}

	if pa.AllocsTop != "" {
		sb.WriteString("**Allocs Profile (top consumers):**\n\n```\n")
		sb.WriteString(pa.AllocsTop)
		sb.WriteString("```\n\n")
	}

	sb.WriteString("**Profile Files:**\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString(fmt.Sprintf("go tool pprof -http=:8080 %s  # CPU\n", pa.CPUProfilePath))
	sb.WriteString(fmt.Sprintf("go tool pprof -http=:8080 %s  # Heap\n", pa.HeapProfilePath))
	sb.WriteString(fmt.Sprintf("go tool pprof -http=:8080 %s  # Allocs\n", pa.AllocsProfilePath))
	sb.WriteString("```\n\n")

	return sb.String()
}
