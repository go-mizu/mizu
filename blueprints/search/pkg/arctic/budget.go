package arctic

import "fmt"

// ResourceBudget defines concurrency limits for the pipeline, computed from
// detected hardware capabilities.
type ResourceBudget struct {
	MaxDownloads   int `json:"max_downloads"`    // concurrent torrent downloads
	MaxProcess     int `json:"max_process"`      // concurrent DuckDB processing jobs
	MaxUploads     int `json:"max_uploads"`       // always 1 (HF serialized)
	DownloadQueue  int `json:"download_queue"`    // buffered channel depth: download → process
	ProcessQueue   int `json:"process_queue"`     // buffered channel depth: process → upload
	DuckDBMemoryMB int `json:"duckdb_memory_mb"`  // per-instance DuckDB memory limit
	Sequential     bool `json:"sequential"`       // true = fall back to sequential mode
}

// ComputeBudget derives a resource budget from a hardware profile.
// Environment overrides are applied via the Config.
func ComputeBudget(hw HardwareProfile, cfg Config) ResourceBudget {
	b := ResourceBudget{
		MaxUploads:     1,       // HF API is serialized via commitMu
		DuckDBMemoryMB: 512,     // default per-instance
	}

	// Safety floor: if resources are too constrained, fall back to sequential.
	if hw.RAMAvailGB < 4 || hw.DiskFreeGB < 60 {
		b.MaxDownloads = 1
		b.MaxProcess = 1
		b.DownloadQueue = 1
		b.ProcessQueue = 1
		b.Sequential = true
		return b
	}

	// Downloads: each .zst can be up to 50 GB. Need disk headroom for
	// the .zst + work artifacts (~10 GB overhead per concurrent job).
	// Cap at 2 to avoid saturating network.
	b.MaxDownloads = int(hw.DiskFreeGB / 80)
	if b.MaxDownloads < 1 {
		b.MaxDownloads = 1
	}
	if b.MaxDownloads > 2 {
		b.MaxDownloads = 2
	}

	// Processing: each DuckDB instance uses ~512 MB + ~1 GB zstd/scanner overhead.
	// Reserve 4 GB for the OS/torrent/other.
	usableRAM := hw.RAMAvailGB - 4
	if usableRAM < 1.5 {
		usableRAM = 1.5
	}
	b.MaxProcess = int(usableRAM / 1.5)

	// Also cap by CPU cores — each ProcessZst is mostly single-threaded
	// (DuckDB uses some parallelism internally).
	cpuLimit := hw.CPUCores / 4
	if cpuLimit < 1 {
		cpuLimit = 1
	}
	if b.MaxProcess > cpuLimit {
		b.MaxProcess = cpuLimit
	}
	// Hard cap at 4 to avoid thrashing.
	if b.MaxProcess > 4 {
		b.MaxProcess = 4
	}
	if b.MaxProcess < 1 {
		b.MaxProcess = 1
	}

	// Tune DuckDB memory per instance if we have lots of RAM.
	if hw.RAMAvailGB > 32 {
		perInstance := int(usableRAM * 1024 / float64(b.MaxProcess) / 3)
		if perInstance > 512 {
			b.DuckDBMemoryMB = perInstance
		}
		// Cap at 2 GB — diminishing returns beyond that.
		if b.DuckDBMemoryMB > 2048 {
			b.DuckDBMemoryMB = 2048
		}
	}

	// Queue depths: keep stages fed without excessive buffering.
	b.DownloadQueue = b.MaxProcess + 1
	b.ProcessQueue = 2

	// If everything is 1, use sequential mode for simplicity.
	if b.MaxDownloads == 1 && b.MaxProcess == 1 {
		b.Sequential = true
	}

	// Apply environment overrides.
	if v := envIntOr("MIZU_ARCTIC_MAX_DOWNLOADS", 0); v > 0 {
		b.MaxDownloads = v
		b.Sequential = false
	}
	if v := envIntOr("MIZU_ARCTIC_MAX_PROCESS", 0); v > 0 {
		b.MaxProcess = v
		b.Sequential = false
	}
	if envOr("MIZU_ARCTIC_PIPELINE", "") == "0" {
		b.MaxDownloads = 1
		b.MaxProcess = 1
		b.DownloadQueue = 1
		b.ProcessQueue = 1
		b.Sequential = true
	}

	return b
}

// String returns a human-readable summary.
func (b ResourceBudget) String() string {
	if b.Sequential {
		return "sequential mode (1 download, 1 process, 1 upload)"
	}
	return fmt.Sprintf("pipeline: %d downloads, %d process, %d upload, DuckDB %d MB/instance",
		b.MaxDownloads, b.MaxProcess, b.MaxUploads, b.DuckDBMemoryMB)
}
