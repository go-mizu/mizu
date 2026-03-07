package web

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/cc"
)

// DownloadTask downloads WARC files from Common Crawl S3.
// It is a self-contained core.Task with no dependency on Manager.
type DownloadTask struct {
	CrawlDir string   `json:"crawl_dir"`
	Paths    []string `json:"paths"`    // manifest paths
	Selected []int    `json:"selected"` // indices into Paths
}

// DownloadState is emitted during download with per-file detail.
type DownloadState struct {
	FileIndex     int     `json:"file_index"`
	FileTotal     int     `json:"file_total"`
	FileName      string  `json:"file_name"`
	WARCIndex     string  `json:"warc_index"`
	BytesReceived int64   `json:"bytes_received"`
	BytesTotal    int64   `json:"bytes_total"`
	Progress      float64 `json:"progress"`
	BytesPerSec   float64 `json:"bytes_per_sec,omitempty"`
}

// DownloadMetric is the final result after all files are downloaded.
type DownloadMetric struct {
	Files   int           `json:"files"`
	Bytes   int64         `json:"bytes"`
	Elapsed time.Duration `json:"elapsed_ns"`
}

// NewDownloadTask creates a download task for the given WARC files.
func NewDownloadTask(crawlDir string, paths []string, selected []int) *DownloadTask {
	return &DownloadTask{CrawlDir: crawlDir, Paths: paths, Selected: selected}
}

func (t *DownloadTask) Run(ctx context.Context, emit func(*DownloadState)) (DownloadMetric, error) {
	start := time.Now()

	warcDir := filepath.Join(t.CrawlDir, "warc")
	if err := os.MkdirAll(warcDir, 0o755); err != nil {
		return DownloadMetric{}, fmt.Errorf("create warc dir: %w", err)
	}

	client := cc.NewClient("", 4)
	total := len(t.Selected)
	var totalBytes int64

	for i, idx := range t.Selected {
		if ctx.Err() != nil {
			return DownloadMetric{}, ctx.Err()
		}
		remotePath := t.Paths[idx]
		fileName := filepath.Base(remotePath)
		warcIdx := warcFileIndex(remotePath, idx)
		localPath := filepath.Join(warcDir, fileName)
		fileStart := time.Now()

		emitDownloadProgress(emit, i, total, warcIdx, fileName, 0, 0, fileStart)

		err := client.DownloadFile(ctx, remotePath, localPath, func(received, bytesTotal int64) {
			totalBytes = received
			emitDownloadProgress(emit, i, total, warcIdx, fileName, received, bytesTotal, fileStart)
		})
		if err != nil {
			return DownloadMetric{}, fmt.Errorf("download %s: %w", fileName, err)
		}
	}

	return DownloadMetric{
		Files:   total,
		Bytes:   totalBytes,
		Elapsed: time.Since(start),
	}, nil
}

// emitDownloadProgress emits a detailed download state snapshot.
func emitDownloadProgress(emit func(*DownloadState), fileIdx, fileTotal int, warcIdx, fileName string, received, bytesTotal int64, fileStart time.Time) {
	if emit == nil {
		return
	}
	filePct := downloadFraction(received, bytesTotal)
	overall := fileProgress(fileIdx, fileTotal, filePct)
	var bps float64
	if elapsed := time.Since(fileStart); elapsed > 0 && received > 0 {
		bps = float64(received) / elapsed.Seconds()
	}
	emit(&DownloadState{
		FileIndex:     fileIdx,
		FileTotal:     fileTotal,
		FileName:      fileName,
		WARCIndex:     warcIdx,
		BytesReceived: received,
		BytesTotal:    bytesTotal,
		Progress:      overall,
		BytesPerSec:   bps,
	})
}

// downloadFraction returns the fraction of bytes received.
func downloadFraction(received, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(received) / float64(total)
}

// fileProgress computes overall progress across a multi-file loop.
func fileProgress(fileIdx, fileTotal int, fileFraction float64) float64 {
	if fileTotal <= 0 {
		return 0
	}
	return (float64(fileIdx) + fileFraction) / float64(fileTotal)
}
