package fw1

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Downloader manages FineWeb (v1) dataset downloads.
type Downloader struct {
	config Config
	client *Client
}

// NewDownloader creates a new downloader.
func NewDownloader(config Config) *Downloader {
	defaults := DefaultConfig()
	if config.DataDir == "" {
		config.DataDir = defaults.DataDir
	}
	if config.Concurrency == 0 {
		config.Concurrency = defaults.Concurrency
	}
	if config.Timeout == 0 {
		config.Timeout = defaults.Timeout
	}
	return &Downloader{config: config, client: NewClient()}
}

// Client returns the underlying HuggingFace client.
func (d *Downloader) Client() *Client {
	return d.client
}

// LocalPath returns the local directory for a dump's parquet files.
func (d *Downloader) LocalPath(dump string) string {
	return filepath.Join(d.config.DataDir, dump)
}

// IsDownloaded checks if a dump has any parquet files locally.
func (d *Downloader) IsDownloaded(dump string) (bool, error) {
	dir := d.LocalPath(dump)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".parquet") {
			return true, nil
		}
	}
	return false, nil
}

// ListLocalFiles returns parquet files for a dump on disk.
func (d *Downloader) ListLocalFiles(dump string) ([]string, error) {
	dir := d.LocalPath(dump)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var files []string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".parquet") {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}
	sort.Strings(files)
	return files, nil
}

// Download downloads specified files to disk.
func (d *Downloader) Download(ctx context.Context, dump string, files []FileInfo, progress ProgressFn) error {
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })

	destDir := d.LocalPath(dump)
	for i, file := range files {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		destPath := filepath.Join(destDir, file.Name)

		// Skip if already downloaded
		if info, err := os.Stat(destPath); err == nil && info.Size() == file.Size {
			if progress != nil {
				progress(DownloadProgress{
					Dump: dump, CurrentFile: file.Name,
					FileIndex: i + 1, TotalFiles: len(files),
					BytesReceived: file.Size, TotalBytes: file.Size,
				})
			}
			continue
		}

		if progress != nil {
			progress(DownloadProgress{
				Dump: dump, CurrentFile: file.Name,
				FileIndex: i + 1, TotalFiles: len(files),
				TotalBytes: file.Size,
			})
		}

		downloadCtx, cancel := context.WithTimeout(ctx, d.config.Timeout)
		err := d.client.DownloadFile(downloadCtx, file, destPath)
		cancel()

		if err != nil {
			if progress != nil {
				progress(DownloadProgress{
					Dump: dump, CurrentFile: file.Name,
					FileIndex: i + 1, TotalFiles: len(files),
					Error: err,
				})
			}
			return fmt.Errorf("downloading %s: %w", file.Name, err)
		}

		if progress != nil {
			progress(DownloadProgress{
				Dump: dump, CurrentFile: file.Name,
				FileIndex: i + 1, TotalFiles: len(files),
				BytesReceived: file.Size, TotalBytes: file.Size,
			})
		}
	}

	if progress != nil {
		progress(DownloadProgress{Dump: dump, TotalFiles: len(files), Done: true})
	}
	return nil
}
