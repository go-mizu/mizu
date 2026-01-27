package fineweb

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Downloader manages FineWeb-2 dataset downloads.
type Downloader struct {
	config Config
	client *Client
}

// NewDownloader creates a new downloader.
func NewDownloader(config Config) *Downloader {
	// Merge with defaults
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
	return &Downloader{
		config: config,
		client: NewClient(),
	}
}

// Download downloads all parquet files for specified languages.
// Skips already downloaded files.
func (d *Downloader) Download(ctx context.Context, langs []string, progress ProgressFn) error {
	for _, lang := range langs {
		if err := d.downloadLanguage(ctx, lang, progress); err != nil {
			return fmt.Errorf("downloading %s: %w", lang, err)
		}
	}
	return nil
}

func (d *Downloader) downloadLanguage(ctx context.Context, lang string, progress ProgressFn) error {
	// List available files
	files, err := d.client.ListFiles(ctx, lang)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return fmt.Errorf("no parquet files found for language %s", lang)
	}

	// Sort files by name
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name < files[j].Name
	})

	// Download each file
	destDir := d.LocalPath(lang)
	for i, file := range files {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		destPath := filepath.Join(destDir, file.Name)

		// Check if already downloaded
		if info, err := os.Stat(destPath); err == nil && info.Size() == file.Size {
			if progress != nil {
				progress(DownloadProgress{
					Language:      lang,
					CurrentFile:   file.Name,
					FileIndex:     i + 1,
					TotalFiles:    len(files),
					BytesReceived: file.Size,
					TotalBytes:    file.Size,
				})
			}
			continue
		}

		// Report progress
		if progress != nil {
			progress(DownloadProgress{
				Language:    lang,
				CurrentFile: file.Name,
				FileIndex:   i + 1,
				TotalFiles:  len(files),
				TotalBytes:  file.Size,
			})
		}

		// Download with timeout
		downloadCtx, cancel := context.WithTimeout(ctx, d.config.Timeout)
		err := d.client.DownloadFile(downloadCtx, file, destPath)
		cancel()

		if err != nil {
			if progress != nil {
				progress(DownloadProgress{
					Language:    lang,
					CurrentFile: file.Name,
					FileIndex:   i + 1,
					TotalFiles:  len(files),
					Error:       err,
				})
			}
			return fmt.Errorf("downloading %s: %w", file.Name, err)
		}

		// Report completion
		if progress != nil {
			progress(DownloadProgress{
				Language:      lang,
				CurrentFile:   file.Name,
				FileIndex:     i + 1,
				TotalFiles:    len(files),
				BytesReceived: file.Size,
				TotalBytes:    file.Size,
			})
		}
	}

	// Final done
	if progress != nil {
		progress(DownloadProgress{
			Language:   lang,
			TotalFiles: len(files),
			Done:       true,
		})
	}

	return nil
}

// DownloadConcurrent downloads languages concurrently.
func (d *Downloader) DownloadConcurrent(ctx context.Context, langs []string, progress ProgressFn) error {
	var wg sync.WaitGroup
	errCh := make(chan error, len(langs))

	// Limit concurrency
	sem := make(chan struct{}, d.config.Concurrency)

	for _, lang := range langs {
		wg.Add(1)
		go func(lang string) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			if err := d.downloadLanguage(ctx, lang, progress); err != nil {
				errCh <- fmt.Errorf("%s: %w", lang, err)
			}
		}(lang)
	}

	wg.Wait()
	close(errCh)

	// Collect errors
	var errs []string
	for err := range errCh {
		errs = append(errs, err.Error())
	}
	if len(errs) > 0 {
		return fmt.Errorf("download errors: %s", strings.Join(errs, "; "))
	}

	return nil
}

// LocalPath returns the local directory for a language's parquet files.
func (d *Downloader) LocalPath(lang string) string {
	return filepath.Join(d.config.DataDir, lang, "train")
}

// IsDownloaded checks if a language has been downloaded.
func (d *Downloader) IsDownloaded(lang string) (bool, error) {
	dir := d.LocalPath(lang)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	// Check for at least one parquet file
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".parquet") {
			return true, nil
		}
	}
	return false, nil
}

// ListDownloaded returns list of downloaded languages.
func (d *Downloader) ListDownloaded() ([]string, error) {
	entries, err := os.ReadDir(d.config.DataDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var langs []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		lang := entry.Name()
		if downloaded, _ := d.IsDownloaded(lang); downloaded {
			langs = append(langs, lang)
		}
	}

	sort.Strings(langs)
	return langs, nil
}

// ListFiles returns the parquet files for a language.
func (d *Downloader) ListFiles(lang string) ([]string, error) {
	dir := d.LocalPath(lang)
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

// GetStatus returns download status for a language.
func (d *Downloader) GetStatus(ctx context.Context, lang string) (downloaded int, total int, err error) {
	// Get remote file count
	files, err := d.client.ListFiles(ctx, lang)
	if err != nil {
		return 0, 0, err
	}
	total = len(files)

	// Count local files
	localFiles, err := d.ListFiles(lang)
	if err != nil {
		return 0, total, err
	}
	downloaded = len(localFiles)

	return downloaded, total, nil
}
