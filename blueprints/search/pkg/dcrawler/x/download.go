package x

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// MediaItem represents a downloadable media item from a tweet.
type MediaItem struct {
	TweetID  string
	URL      string
	Type     string // "photo", "video", "gif"
	Index    int    // position within tweet (for multi-photo)
}

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

// ExtractMedia extracts all downloadable media items from tweets.
func ExtractMedia(tweets []Tweet, photos, videos, gifs bool) []MediaItem {
	var items []MediaItem
	for _, t := range tweets {
		if photos {
			for i, u := range t.Photos {
				items = append(items, MediaItem{
					TweetID: t.ID,
					URL:     u,
					Type:    "photo",
					Index:   i,
				})
			}
		}
		if videos {
			for i, u := range t.Videos {
				items = append(items, MediaItem{
					TweetID: t.ID,
					URL:     u,
					Type:    "video",
					Index:   i,
				})
			}
		}
		if gifs {
			for i, u := range t.GIFs {
				items = append(items, MediaItem{
					TweetID: t.ID,
					URL:     u,
					Type:    "gif",
					Index:   i,
				})
			}
		}
	}
	return items
}

// DownloadMedia downloads media items to the specified directory.
func DownloadMedia(ctx context.Context, items []MediaItem, dir string, workers int, cb DownloadProgressCallback) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create media dir: %w", err)
	}

	if len(items) == 0 {
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

	ch := make(chan MediaItem, len(items))
	for _, item := range items {
		ch <- item
	}
	close(ch)

	if workers <= 0 {
		workers = 8
	}

	client := &http.Client{Timeout: 120 * time.Second}

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

					filename := mediaItemFilename(item)
					path := filepath.Join(dir, filename)

					// Skip if already exists
					if fi, err := os.Stat(path); err == nil && fi.Size() > 0 {
						skipped.Add(1)
						if cb != nil {
							cb(DownloadProgress{
								Total:      len(items),
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
							Total:      len(items),
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
			Total:      len(items),
			Downloaded: int(downloaded.Load()),
			Skipped:    int(skipped.Load()),
			Failed:     int(failed.Load()),
			Bytes:      totalBytes.Load(),
			Done:       true,
		})
	}

	return nil
}

func downloadFile(ctx context.Context, client *http.Client, rawURL, path string) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return 0, err
	}

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
		os.Remove(path)
		return 0, err
	}

	return n, nil
}

func mediaItemFilename(item MediaItem) string {
	ext := extensionFromURL(item.URL, item.Type)
	if item.Index > 0 {
		return fmt.Sprintf("%s_%s_%d%s", item.TweetID, item.Type, item.Index, ext)
	}
	return fmt.Sprintf("%s_%s%s", item.TweetID, item.Type, ext)
}

func extensionFromURL(rawURL, mediaType string) string {
	defaults := map[string]string{
		"photo": ".jpg",
		"video": ".mp4",
		"gif":   ".mp4", // Twitter GIFs are actually mp4
	}

	// Try to extract from URL path
	parsed, err := url.Parse(rawURL)
	if err == nil {
		ext := filepath.Ext(parsed.Path)
		switch strings.ToLower(ext) {
		case ".jpg", ".jpeg", ".png", ".webp", ".mp4", ".mov":
			return ext
		}
	}

	if d, ok := defaults[mediaType]; ok {
		return d
	}
	return ".bin"
}
