// Package fw2 provides a downloader for HuggingFace's FineWeb-2 dataset.
package fw2

// Language represents a FineWeb-2 language configuration.
type Language struct {
	Code   string // ISO 639-3 + script: "vie_Latn", "eng_Latn"
	Name   string // Human readable: "Vietnamese", "English"
	Script string // Script system: "Latin", "Han", "Cyrillic"
}

// FileInfo represents a parquet file in the dataset.
type FileInfo struct {
	Name string // Filename
	Path string // Full path in dataset (e.g., "vie_Latn/train/000000.parquet")
	URL  string // Download URL
	Size int64  // File size in bytes
	LFS  bool   // Is LFS tracked
	OID  string // Git LFS OID (for verification)
}

// DownloadProgress reports download progress.
type DownloadProgress struct {
	Language      string
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

// ImportProgress reports import progress.
type ImportProgress struct {
	Language    string
	CurrentFile string
	FileIndex   int
	TotalFiles  int
	RowsImported int64
	Done        bool
	Error       error
}

// ImportProgressFn callback for import progress updates.
type ImportProgressFn func(ImportProgress)

// DatasetConfig represents a language config in the dataset.
type DatasetConfig struct {
	Config string // e.g. "vie_Latn"
	Split  string // e.g. "train" or "test"
}

// SplitSize holds size info for a single split.
type SplitSize struct {
	Config       string
	Split        string
	NumRows      int64
	NumBytes     int64 // parquet file size
	NumBytesMemory int64 // in-memory size
	NumColumns   int
}

// DatasetSizeInfo holds aggregated size info.
type DatasetSizeInfo struct {
	TotalRows       int64
	TotalBytes      int64
	TotalBytesMemory int64
	Configs         []ConfigSize
}

// ConfigSize holds per-config size info.
type ConfigSize struct {
	Config         string
	NumRows        int64
	NumBytes       int64
	NumBytesMemory int64
	NumColumns     int
	Splits         []SplitSize
}
