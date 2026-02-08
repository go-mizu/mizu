package dcrawler

import (
	"context"
	"database/sql"
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

	"github.com/cespare/xxhash/v2"
)

// DownloadImages downloads image URLs discovered during crawling.
// Reads image links from the result DB shards and downloads them
// concurrently to the images/ subdirectory.
func DownloadImages(ctx context.Context, cfg Config) error {
	imgDir := filepath.Join(cfg.DomainDir(), "images")
	if err := os.MkdirAll(imgDir, 0o755); err != nil {
		return fmt.Errorf("create image dir: %w", err)
	}

	urls := collectImageURLs(cfg.ResultDir(), cfg.ShardCount)
	if len(urls) == 0 {
		fmt.Println("  No image URLs found in crawl results")
		return nil
	}

	// Skip already downloaded
	var toDownload []string
	for _, u := range urls {
		fname := imageFilename(u)
		path := filepath.Join(imgDir, fname)
		if _, err := os.Stat(path); err == nil {
			continue
		}
		toDownload = append(toDownload, u)
	}

	existing := len(urls) - len(toDownload)
	if len(toDownload) == 0 {
		fmt.Printf("  All %s images already downloaded\n", fmtInt(len(urls)))
		return nil
	}
	fmt.Printf("  Images: %s to download (%s existing) → %s\n",
		fmtInt(len(toDownload)), fmtInt(existing), imgDir)

	ch := make(chan string, len(toDownload))
	for _, u := range toDownload {
		ch <- u
	}
	close(ch)

	var done, failed atomic.Int64
	var totalBytes atomic.Int64
	client := &http.Client{Timeout: 30 * time.Second}

	workers := 20
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for u := range ch {
				if ctx.Err() != nil {
					return
				}
				n, err := downloadImage(ctx, client, u, imgDir)
				if err != nil {
					failed.Add(1)
				} else {
					d := done.Add(1)
					totalBytes.Add(n)
					if d%20 == 0 {
						fmt.Printf("\r  Progress: %s / %s downloaded (%s)",
							fmtInt(int(d)), fmtInt(len(toDownload)),
							formatBytes(totalBytes.Load()))
					}
				}
			}
		}()
	}
	wg.Wait()

	fmt.Printf("\r  Done: %s downloaded, %s failed (%s total)          \n",
		fmtInt(int(done.Load())), fmtInt(int(failed.Load())),
		formatBytes(totalBytes.Load()))
	return nil
}

func collectImageURLs(resultDir string, shardCount int) []string {
	seen := make(map[string]struct{})
	var urls []string

	for i := range shardCount {
		path := fmt.Sprintf("%s/results_%03d.duckdb", resultDir, i)
		if _, err := os.Stat(path); err != nil {
			continue
		}
		db, err := sql.Open("duckdb", path+"?access_mode=READ_ONLY")
		if err != nil {
			continue
		}
		rows, err := db.Query("SELECT DISTINCT target_url FROM links WHERE rel IN ('image', 'image-srcset', 'og:image')")
		if err != nil {
			db.Close()
			continue
		}
		for rows.Next() {
			var u string
			if rows.Scan(&u) == nil {
				if _, ok := seen[u]; !ok {
					seen[u] = struct{}{}
					urls = append(urls, u)
				}
			}
		}
		rows.Close()
		db.Close()
	}
	return urls
}

func downloadImage(ctx context.Context, client *http.Client, imgURL, dir string) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", imgURL, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "image/webp,image/avif,image/*,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		io.Copy(io.Discard, resp.Body)
		return 0, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	fname := imageFilename(imgURL)
	path := filepath.Join(dir, fname)

	f, err := os.Create(path)
	if err != nil {
		return 0, err
	}
	n, err := io.Copy(f, io.LimitReader(resp.Body, 50*1024*1024))
	f.Close()
	if err != nil {
		os.Remove(path)
		return 0, err
	}
	return n, nil
}

func imageFilename(imgURL string) string {
	u, err := url.Parse(imgURL)
	if err != nil {
		return fmt.Sprintf("%016x.jpg", xxhash.Sum64String(imgURL))
	}
	base := filepath.Base(u.Path)
	if base == "" || base == "/" || base == "." {
		return fmt.Sprintf("%016x.jpg", xxhash.Sum64String(imgURL))
	}
	// Sanitize: only keep alphanumeric, dots, hyphens, underscores
	var sb strings.Builder
	for _, c := range base {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '.' || c == '-' || c == '_' {
			sb.WriteRune(c)
		}
	}
	clean := sb.String()
	if clean == "" {
		clean = "image"
	}
	// Prefix with short hash to avoid collisions
	hash := fmt.Sprintf("%08x", xxhash.Sum64String(imgURL))
	return hash + "_" + clean
}

func formatBytes(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
