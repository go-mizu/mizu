package export

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// AssetCollector tracks unique asset URLs discovered during HTML rewriting.
// Thread-safe — called from parallel worker pool.
type AssetCollector struct {
	mu     sync.Mutex
	assets map[string]string // absolute URL → local relative path (from siteDir)
}

// NewAssetCollector creates a new asset collector.
func NewAssetCollector() *AssetCollector {
	return &AssetCollector{
		assets: make(map[string]string),
	}
}

// Add records an asset URL and returns the local relative path it will be saved to.
// The path is relative to the site directory (e.g., "_assets/openai.com/_next/static/css/foo.css").
// Thread-safe; deduplicates by URL.
func (ac *AssetCollector) Add(absoluteURL string) string {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	if local, ok := ac.assets[absoluteURL]; ok {
		return local
	}

	local := assetLocalPath(absoluteURL)
	ac.assets[absoluteURL] = local
	return local
}

// URLs returns all collected asset URLs.
func (ac *AssetCollector) URLs() map[string]string {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	cp := make(map[string]string, len(ac.assets))
	for k, v := range ac.assets {
		cp[k] = v
	}
	return cp
}

// Count returns the number of unique assets collected.
func (ac *AssetCollector) Count() int {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	return len(ac.assets)
}

// assetLocalPath converts an absolute URL to a local file path under _assets/.
// https://openai.com/_next/static/css/foo.css → _assets/openai.com/_next/static/css/foo.css
// URLs with query strings get a hash suffix to avoid collisions.
func assetLocalPath(absoluteURL string) string {
	u, err := url.Parse(absoluteURL)
	if err != nil {
		// Fallback: hash the URL
		h := sha256.Sum256([]byte(absoluteURL))
		return fmt.Sprintf("_assets/%x", h[:8])
	}

	host := u.Hostname()
	p := strings.TrimPrefix(u.Path, "/")
	if p == "" {
		p = "index"
	}

	local := filepath.Join("_assets", host, p)

	// If URL has query params, add short hash to avoid collisions
	if u.RawQuery != "" {
		h := sha256.Sum256([]byte(u.RawQuery))
		ext := path.Ext(p)
		base := strings.TrimSuffix(local, ext)
		local = fmt.Sprintf("%s_%x%s", base, h[:4], ext)
	}

	return local
}

// AssetDownloadStats tracks download progress.
type AssetDownloadStats struct {
	Total      int
	Downloaded int64
	Failed     int64
	Bytes      int64
	CSSExtra   int64 // extra assets discovered inside CSS files
}

// DownloadAssets downloads all collected assets to the site directory.
// Returns stats about the download. CSS files are parsed for url() references
// and those assets are downloaded too.
func DownloadAssets(ctx context.Context, ac *AssetCollector, siteDir string, workers int, progress func(AssetDownloadStats)) error {
	assets := ac.URLs()
	if len(assets) == 0 {
		return nil
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	var downloaded, failed, totalBytes, cssExtra atomic.Int64

	// Channel for download jobs.
	type dlJob struct {
		url   string
		local string
	}
	ch := make(chan dlJob, workers*4)

	// Worker pool.
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range ch {
				if ctx.Err() != nil {
					return
				}
				n, err := downloadAsset(ctx, client, job.url, filepath.Join(siteDir, job.local))
				if err != nil {
					failed.Add(1)
					continue
				}
				downloaded.Add(1)
				totalBytes.Add(n)

				// Parse CSS files for nested url() references.
				if isCSS(job.local) {
					extras := parseCSSForAssets(filepath.Join(siteDir, job.local), job.url)
					for _, extra := range extras {
						localPath := ac.Add(extra)
						// Check if already downloaded (Add deduplicates)
						fullPath := filepath.Join(siteDir, localPath)
						if _, err := os.Stat(fullPath); err == nil {
							continue // already exists
						}
						cssExtra.Add(1)
						n2, err := downloadAsset(ctx, client, extra, fullPath)
						if err != nil {
							failed.Add(1)
							continue
						}
						downloaded.Add(1)
						totalBytes.Add(n2)
					}
				}
			}
		}()
	}

	// Progress reporter.
	total := len(assets)
	stopProgress := make(chan struct{})
	progressDone := make(chan struct{})
	if progress != nil {
		go func() {
			defer close(progressDone)
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-stopProgress:
					return
				case <-ticker.C:
					progress(AssetDownloadStats{
						Total:      total,
						Downloaded: downloaded.Load(),
						Failed:     failed.Load(),
						Bytes:      totalBytes.Load(),
						CSSExtra:   cssExtra.Load(),
					})
				}
			}
		}()
	} else {
		close(progressDone)
	}

	// Feed download jobs.
	for absURL, localPath := range assets {
		select {
		case ch <- dlJob{url: absURL, local: localPath}:
		case <-ctx.Done():
			close(ch)
			wg.Wait()
			close(stopProgress)
			<-progressDone
			return ctx.Err()
		}
	}
	close(ch)
	wg.Wait()

	close(stopProgress)
	<-progressDone

	// Final progress.
	if progress != nil {
		progress(AssetDownloadStats{
			Total:      total + int(cssExtra.Load()),
			Downloaded: downloaded.Load(),
			Failed:     failed.Load(),
			Bytes:      totalBytes.Load(),
			CSSExtra:   cssExtra.Load(),
		})
	}

	return nil
}

// downloadAsset fetches a URL and writes it to disk. Returns bytes written.
func downloadAsset(ctx context.Context, client *http.Client, assetURL, destPath string) (int64, error) {
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return 0, err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", assetURL, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Accept", "*/*")

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return 0, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	n, err := io.Copy(f, io.LimitReader(resp.Body, 10*1024*1024)) // 10MB max per asset
	return n, err
}

// cssAssetURLRe matches url() in CSS, capturing the URL inside.
var cssAssetURLRe = regexp.MustCompile(`url\(\s*['"]?([^'")]+)['"]?\s*\)`)

// parseCSSForAssets reads a downloaded CSS file and extracts absolute url() references.
func parseCSSForAssets(cssPath, cssURL string) []string {
	data, err := os.ReadFile(cssPath)
	if err != nil {
		return nil
	}

	baseURL, err := url.Parse(cssURL)
	if err != nil {
		return nil
	}

	var assets []string
	seen := make(map[string]bool)

	for _, match := range cssAssetURLRe.FindAllStringSubmatch(string(data), -1) {
		if len(match) < 2 {
			continue
		}
		raw := strings.TrimSpace(match[1])
		if isSpecialURL(raw) {
			continue
		}

		resolved := resolveURL(raw, baseURL)
		if resolved == nil || resolved.Scheme == "" {
			continue
		}

		absURL := resolved.String()
		if seen[absURL] {
			continue
		}
		seen[absURL] = true
		assets = append(assets, absURL)
	}

	return assets
}

// RewriteDownloadedCSS rewrites url() references inside a downloaded CSS file
// to point to local asset paths.
func RewriteDownloadedCSS(cssPath, cssURL, siteDir string, ac *AssetCollector) {
	data, err := os.ReadFile(cssPath)
	if err != nil {
		return
	}

	baseURL, err := url.Parse(cssURL)
	if err != nil {
		return
	}

	cssLocalPath := ac.Add(cssURL)
	cssDir := path.Dir(cssLocalPath)

	rewritten := cssAssetURLRe.ReplaceAllStringFunc(string(data), func(match string) string {
		sub := cssAssetURLRe.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		raw := strings.TrimSpace(sub[1])
		if isSpecialURL(raw) {
			return match
		}

		resolved := resolveURL(raw, baseURL)
		if resolved == nil || resolved.Scheme == "" {
			return match
		}

		absURL := resolved.String()
		localPath := ac.Add(absURL)

		// Make relative to this CSS file's location
		rel, err := filepath.Rel(cssDir, localPath)
		if err != nil {
			return match
		}
		rel = filepath.ToSlash(rel)
		return fmt.Sprintf("url(%s)", rel)
	})

	if rewritten != string(data) {
		if err := os.WriteFile(cssPath, []byte(rewritten), 0o644); err != nil {
			log.Printf("[export] ERROR rewrite CSS %s: %v", cssPath, err)
		}
	}
}

func isCSS(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".css"
}
