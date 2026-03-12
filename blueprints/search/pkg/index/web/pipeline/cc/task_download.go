// Package cc implements Common Crawl pipeline tasks: download, markdown,
// pack, and index. Each task implements core.Task[State, Metric].
package cc

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
	ccpkg "github.com/go-mizu/mizu/blueprints/search/pkg/cc"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index/web/pipeline/util"
)

// Compile-time check.
var _ core.Task[DownloadState, DownloadMetric] = (*DownloadTask)(nil)

const downloadConcurrency = 3

// DownloadTask downloads WARC files from Common Crawl S3.
type DownloadTask struct {
	crawlDir string
	paths    []string
	selected []int
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
	return &DownloadTask{crawlDir: crawlDir, paths: paths, selected: selected}
}

func (t *DownloadTask) Run(ctx context.Context, emit func(*DownloadState)) (DownloadMetric, error) {
	start := time.Now()

	warcDir := filepath.Join(t.crawlDir, "warc")
	if err := os.MkdirAll(warcDir, 0o755); err != nil {
		return DownloadMetric{}, fmt.Errorf("create warc dir: %w", err)
	}

	client := ccpkg.NewClient("", 4)
	total := len(t.selected)
	var totalBytes atomic.Int64

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(downloadConcurrency)

	for i, idx := range t.selected {
		i, idx := i, idx
		g.Go(func() error {
			remotePath := t.paths[idx]
			fileName := filepath.Base(remotePath)
			warcIdx := util.WARCFileIndex(remotePath, idx)
			localPath := filepath.Join(warcDir, fileName)
			fileStart := time.Now()

			emitDownloadProgress(emit, i, total, warcIdx, fileName, 0, 0, fileStart)

			err := client.DownloadFile(gctx, remotePath, localPath, func(received, bytesTotal int64) {
				totalBytes.Store(received)
				emitDownloadProgress(emit, i, total, warcIdx, fileName, received, bytesTotal, fileStart)
			})
			if err != nil {
				return fmt.Errorf("download %s: %w", fileName, err)
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return DownloadMetric{}, err
	}

	return DownloadMetric{
		Files:   total,
		Bytes:   totalBytes.Load(),
		Elapsed: time.Since(start),
	}, nil
}

func emitDownloadProgress(emit func(*DownloadState), fileIdx, fileTotal int, warcIdx, fileName string, received, bytesTotal int64, fileStart time.Time) {
	if emit == nil {
		return
	}
	filePct := downloadFraction(received, bytesTotal)
	overall := util.FileProgress(fileIdx, fileTotal, filePct)
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

func downloadFraction(received, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(received) / float64(total)
}
