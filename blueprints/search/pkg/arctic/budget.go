package arctic

import "fmt"

// ResourceBudget defines concurrency limits for the pipeline, computed from
// detected hardware capabilities.
type ResourceBudget struct {
	MaxDownloads      int `json:"max_downloads"`       // concurrent torrent downloads
	MaxProcess        int `json:"max_process"`         // concurrent DuckDB processing jobs
	MaxUploads        int `json:"max_uploads"`          // always 1 (HF serialized)
	MaxConvertWorkers int `json:"max_convert_workers"`  // concurrent DuckDB shard conversions per ProcessZst
	DownloadQueue     int `json:"download_queue"`       // buffered channel depth: download → process
	ProcessQueue      int `json:"process_queue"`        // buffered channel depth: process → upload
	DuckDBMemoryMB    int `json:"duckdb_memory_mb"`     // per-instance DuckDB memory limit
	Sequential        bool `json:"sequential"`          // true = fall back to sequential mode
}

// ComputeBudget derives a resource budget from a hardware profile.
//
// The pipeline always helps — even with 1 download + 1 process + 1 upload
// worker — because stages overlap: while uploading month N (which can take
// 5-10 minutes on HF), we download+process month N+1.
//
// On server2 (6 cores, 11 GB RAM, 128 GB disk), the budget is:
//   downloads=1, process=2, DuckDB=512MB → pipeline mode (overlap stages)
//   Note: a process-wide semaphore (zstdDecoderSem) ensures only one 2 GB
//   zstd decoder is open at a time regardless of MaxProcess.
//
// On a beefier server (20 cores, 256 GB RAM, 2 TB disk):
//   downloads=2, process=3, DuckDB=1024MB → full parallelism
func ComputeBudget(hw HardwareProfile, cfg Config) ResourceBudget {
	b := ResourceBudget{
		MaxUploads:     1,   // HF API is serialized via commitMu
		DuckDBMemoryMB: 512, // default per-instance
	}

	// Hard sequential fallback: only if truly starved.
	// 2 GB RAM is the absolute minimum for zstd decoder + DuckDB.
	// 30 GB disk needed for at least one .zst + work artifacts.
	if hw.RAMTotalGB < 2 || hw.DiskFreeGB < 30 {
		b.MaxDownloads = 1
		b.MaxProcess = 1
		b.MaxConvertWorkers = 1
		b.DownloadQueue = 1
		b.ProcessQueue = 1
		b.Sequential = true
		return b
	}

	// --- Downloads ---
	// Each .zst ranges from tiny (early months) to ~50 GB (recent months).
	// With 128 GB free we can safely have 1 downloading while 1 is processing.
	// Only allow 2 concurrent downloads if we have plenty of disk.
	b.MaxDownloads = 1
	if hw.DiskFreeGB >= 200 {
		b.MaxDownloads = 2
	}

	// --- Processing ---
	// Each ProcessZst needs: ~2 GB zstd decoder window + ~512 MB DuckDB + overhead.
	// The zstd decoder alone allocates a 2 GB buffer (WithDecoderMaxWindow(1<<31))
	// because the Reddit .zst archives use a 2 GB window size.
	// A process-wide semaphore (zstdDecoderSem) limits concurrent decoders to 1,
	// but each processing slot still needs headroom for DuckDB + scanner buffers.
	// Use total RAM (not just available — the OS will reclaim page cache).
	// Reserve 4 GB for OS + torrent client + upload + other processes.
	usableRAM := hw.RAMTotalGB - 4
	if usableRAM < 1.5 {
		usableRAM = 1.5
	}

	// Each processing slot needs ~3 GB headroom (2 GB decoder + 512 MB DuckDB + overhead).
	// With the decoder semaphore, only one slot holds the 2 GB decoder at a time,
	// but the budget still caps concurrency to prevent memory pressure.
	b.MaxProcess = int(usableRAM / 3.0)

	// Cap by CPU — each ProcessZst is ~1 core (DuckDB has some internal parallelism).
	cpuLimit := hw.CPUCores / 2
	if cpuLimit < 1 {
		cpuLimit = 1
	}
	if b.MaxProcess > cpuLimit {
		b.MaxProcess = cpuLimit
	}
	// Hard cap at 4.
	if b.MaxProcess > 4 {
		b.MaxProcess = 4
	}
	if b.MaxProcess < 1 {
		b.MaxProcess = 1
	}

	// --- Convert workers (async parquet conversion within ProcessZst) ---
	// Go engine: each worker needs ~400 MB (ZSTD SpeedBestCompression encoder
	//   ~256 MB + parquet-go writer buffers + 100K in-memory lines).
	// DuckDB engine: each worker needs ~600 MB (512 MB DuckDB + overhead).
	//   With explicit SET threads per instance, DuckDB workers don't oversubscribe
	//   the CPU, so the CPU cap can match core count (not cores/2).
	// Budget from remaining headroom above the 2.5 GB needed for decoder + scanner.
	convertRAM := usableRAM - 2.5
	if convertRAM < 0.5 {
		convertRAM = 0.5
	}
	perWorkerGB := 0.6 // DuckDB default
	if cfg.Engine != "duckdb" {
		perWorkerGB = 0.4 // Go parquet writer: ZSTD encoder + parquet buffers + chunk
	}
	b.MaxConvertWorkers = int(convertRAM / perWorkerGB)
	// CPU cap: DuckDB workers pin threads internally (SET threads = NumCPU/workers),
	// so we can use up to CPUCores workers without oversubscription.
	// Go engine workers are single-threaded, same logic applies.
	convertCPU := hw.CPUCores
	if convertCPU < 1 {
		convertCPU = 1
	}
	if b.MaxConvertWorkers > convertCPU {
		b.MaxConvertWorkers = convertCPU
	}
	if b.MaxConvertWorkers > 8 {
		b.MaxConvertWorkers = 8
	}
	if b.MaxConvertWorkers < 1 {
		b.MaxConvertWorkers = 1
	}

	// --- DuckDB memory per instance ---
	// Scale DuckDB memory with available RAM, but cap at 2 GB.
	// On 12 GB server: (12 - 3) / 1 / 2 = 4.5 GB → cap at 512 MB (plenty)
	// On 256 GB server: (256 - 3) / 3 / 2 = 42 GB → cap at 2048 MB
	if hw.RAMTotalGB >= 24 {
		perInstance := int(usableRAM / float64(b.MaxProcess) / 2 * 1024)
		if perInstance > b.DuckDBMemoryMB {
			b.DuckDBMemoryMB = perInstance
		}
		if b.DuckDBMemoryMB > 2048 {
			b.DuckDBMemoryMB = 2048
		}
	}

	// --- Queue depths ---
	// Download→process queue: enough to keep process workers fed.
	b.DownloadQueue = b.MaxProcess + 1
	if b.DownloadQueue < 2 {
		b.DownloadQueue = 2
	}
	// Process→upload queue: small buffer so upload picks up fast.
	b.ProcessQueue = 2

	// Pipeline mode is always enabled — even 1/1/1 benefits from stage overlap.
	// The only exception is the hard fallback above (RAM < 2 GB or disk < 30 GB).
	b.Sequential = false

	// Apply environment overrides.
	if v := envIntOr("MIZU_ARCTIC_MAX_DOWNLOADS", 0); v > 0 {
		b.MaxDownloads = v
	}
	if v := envIntOr("MIZU_ARCTIC_MAX_PROCESS", 0); v > 0 {
		b.MaxProcess = v
	}
	if v := envIntOr("MIZU_ARCTIC_DUCKDB_MB", 0); v > 0 {
		b.DuckDBMemoryMB = v
	}
	if v := envIntOr("MIZU_ARCTIC_MAX_CONVERT", 0); v > 0 {
		b.MaxConvertWorkers = v
	}
	if envOr("MIZU_ARCTIC_PIPELINE", "") == "0" {
		b.MaxDownloads = 1
		b.MaxProcess = 1
		b.MaxConvertWorkers = 1
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
	return fmt.Sprintf("pipeline: %d download, %d process (%d convert workers), %d upload, DuckDB %dMB/instance",
		b.MaxDownloads, b.MaxProcess, b.MaxConvertWorkers, b.MaxUploads, b.DuckDBMemoryMB)
}
