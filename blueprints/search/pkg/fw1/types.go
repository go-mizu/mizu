// Package fw1 provides a downloader for HuggingFace's FineWeb (v1) dataset.
// FineWeb is an English-only dataset organized by CommonCrawl dump configs.
package fw1

// Dump represents a CommonCrawl dump config in FineWeb.
type Dump struct {
	Name string // "CC-MAIN-2024-51"
}

// FileInfo represents a parquet file in the dataset.
type FileInfo struct {
	Name string // "000_00000.parquet"
	Path string // "data/CC-MAIN-2024-51/000_00000.parquet"
	URL  string // full download URL
	Size int64  // file size in bytes
	LFS  bool
	OID  string
}

// DownloadProgress reports per-file download progress.
type DownloadProgress struct {
	Dump          string
	CurrentFile   string
	FileIndex     int
	TotalFiles    int
	BytesReceived int64
	TotalBytes    int64
	Done          bool
	Error         error
}

// ProgressFn callback for progress updates.
type ProgressFn func(DownloadProgress)

// ByteProgressFn is called periodically during download with bytes downloaded so far.
type ByteProgressFn func(bytesDownloaded, totalBytes int64)

// DatasetConfig represents a dump config in the dataset.
type DatasetConfig struct {
	Config string // e.g. "CC-MAIN-2024-51"
	Split  string // always "train"
}

// DumpSize holds size info for a single dump config.
type DumpSize struct {
	Config          string
	NumRows         int64
	NumBytes        int64 // parquet file size
	NumBytesMemory  int64
	NumColumns      int
}

// DatasetSizeInfo holds aggregated size info.
type DatasetSizeInfo struct {
	TotalRows        int64
	TotalBytes       int64
	TotalBytesMemory int64
	Configs          []DumpSize
}
