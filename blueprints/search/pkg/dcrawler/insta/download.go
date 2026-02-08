package insta

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// DownloadProgress reports download progress.
type DownloadProgress struct {
	Total      int
	Downloaded int
	Skipped    int
	Failed     int
	Bytes      int64
	Done       bool
}

// DownloadProgressCallback is called with download progress updates.
type DownloadProgressCallback func(DownloadProgress)

// DownloadMedia downloads media items to the specified directory.
func DownloadMedia(ctx context.Context, items []MediaItem, dir string, workers int, images, videos bool, cb DownloadProgressCallback) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create media dir: %w", err)
	}

	// Filter items by type
	var filtered []MediaItem
	for _, item := range items {
		if item.Type == "image" && !images {
			continue
		}
		if item.Type == "video" && !videos {
			continue
		}
		filtered = append(filtered, item)
	}

	if len(filtered) == 0 {
		if cb != nil {
			cb(DownloadProgress{Done: true})
		}
		return nil
	}

	var (
		downloaded atomic.Int32
		skipped    atomic.Int32
		failed     atomic.Int32
		totalBytes atomic.Int64
	)

	ch := make(chan MediaItem, len(filtered))
	for _, item := range filtered {
		ch <- item
	}
	close(ch)

	if workers <= 0 {
		workers = 8
	}

	client := &http.Client{Timeout: 60 * time.Second}

	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case item, ok := <-ch:
					if !ok {
						return
					}

					filename := mediaFilename(item)
					path := filepath.Join(dir, filename)

					// Skip if already exists
					if fi, err := os.Stat(path); err == nil && fi.Size() > 0 {
						skipped.Add(1)
						if cb != nil {
							cb(DownloadProgress{
								Total:      len(filtered),
								Downloaded: int(downloaded.Load()),
								Skipped:    int(skipped.Load()),
								Failed:     int(failed.Load()),
								Bytes:      totalBytes.Load(),
							})
						}
						continue
					}

					n, err := downloadFile(ctx, client, item.URL, path)
					if err != nil {
						failed.Add(1)
					} else {
						downloaded.Add(1)
						totalBytes.Add(n)
					}

					if cb != nil {
						cb(DownloadProgress{
							Total:      len(filtered),
							Downloaded: int(downloaded.Load()),
							Skipped:    int(skipped.Load()),
							Failed:     int(failed.Load()),
							Bytes:      totalBytes.Load(),
						})
					}
				}
			}
		}()
	}

	wg.Wait()

	if cb != nil {
		cb(DownloadProgress{
			Total:      len(filtered),
			Downloaded: int(downloaded.Load()),
			Skipped:    int(skipped.Load()),
			Failed:     int(failed.Load()),
			Bytes:      totalBytes.Load(),
			Done:       true,
		})
	}

	return nil
}

func downloadFile(ctx context.Context, client *http.Client, url, path string) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", DefaultUserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	f, err := os.Create(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	n, err := io.Copy(f, resp.Body)
	if err != nil {
		os.Remove(path) // clean up partial file
		return 0, err
	}

	return n, nil
}

func mediaFilename(item MediaItem) string {
	ext := ".jpg"
	if item.Type == "video" {
		ext = ".mp4"
	}

	// Try to get extension from URL
	if idx := strings.LastIndex(item.URL, "."); idx > 0 {
		urlExt := item.URL[idx:]
		if qIdx := strings.Index(urlExt, "?"); qIdx > 0 {
			urlExt = urlExt[:qIdx]
		}
		switch urlExt {
		case ".jpg", ".jpeg", ".png", ".webp", ".mp4", ".mov":
			ext = urlExt
		}
	}

	if item.Index > 0 {
		return fmt.Sprintf("%s_%d%s", item.Shortcode, item.Index, ext)
	}
	return item.Shortcode + ext
}
