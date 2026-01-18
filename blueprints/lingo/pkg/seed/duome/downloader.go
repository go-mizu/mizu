package duome

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Downloader handles fetching HTML files from Duome
type Downloader struct {
	client    *http.Client
	baseDir   string
	userAgent string
	delay     time.Duration
	progress  ProgressCallback
}

// DownloaderOption configures a Downloader
type DownloaderOption func(*Downloader)

// WithTimeout sets the HTTP client timeout
func WithTimeout(d time.Duration) DownloaderOption {
	return func(dl *Downloader) {
		dl.client.Timeout = d
	}
}

// WithDelay sets the delay between requests
func WithDelay(d time.Duration) DownloaderOption {
	return func(dl *Downloader) {
		dl.delay = d
	}
}

// WithProgress sets the progress callback
func WithProgress(cb ProgressCallback) DownloaderOption {
	return func(dl *Downloader) {
		dl.progress = cb
	}
}

// WithUserAgent sets the User-Agent header
func WithUserAgent(ua string) DownloaderOption {
	return func(dl *Downloader) {
		dl.userAgent = ua
	}
}

// NewDownloader creates a new Downloader
func NewDownloader(baseDir string, opts ...DownloaderOption) *Downloader {
	d := &Downloader{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseDir:   baseDir,
		userAgent: "Lingo/1.0 (+https://github.com/go-mizu/mizu)",
		delay:     time.Second, // Rate limit: 1 request per second
	}

	for _, opt := range opts {
		opt(d)
	}

	return d
}

// BaseDir returns the base directory for downloads
func (d *Downloader) BaseDir() string {
	return d.baseDir
}

// RawDir returns the directory for raw HTML files
func (d *Downloader) RawDir() string {
	return filepath.Join(d.baseDir, "raw")
}

// JSONDir returns the directory for parsed JSON files
func (d *Downloader) JSONDir() string {
	return filepath.Join(d.baseDir, "json")
}

// ensureDirs creates necessary directories
func (d *Downloader) ensureDirs() error {
	dirs := []string{
		d.RawDir(),
		d.JSONDir(),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}
	return nil
}

// VocabularyPath returns the path for a vocabulary HTML file
func (d *Downloader) VocabularyPath(pair LanguagePair) string {
	return filepath.Join(d.RawDir(), fmt.Sprintf("vocabulary_%s_%s.html", pair.From, pair.To))
}

// TipsPath returns the path for a tips HTML file
func (d *Downloader) TipsPath(pair LanguagePair) string {
	return filepath.Join(d.RawDir(), fmt.Sprintf("tips_%s_%s.html", pair.From, pair.To))
}

// MetadataPath returns the path for the metadata file
func (d *Downloader) MetadataPath() string {
	return filepath.Join(d.baseDir, "metadata.json")
}

// loadMetadata loads the metadata file
func (d *Downloader) loadMetadata() (*Metadata, error) {
	path := d.MetadataPath()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Metadata{Downloads: make(map[string]DownloadInfo)}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read metadata: %w", err)
	}

	var m Metadata
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse metadata: %w", err)
	}
	if m.Downloads == nil {
		m.Downloads = make(map[string]DownloadInfo)
	}
	return &m, nil
}

// saveMetadata saves the metadata file
func (d *Downloader) saveMetadata(m *Metadata) error {
	m.UpdatedAt = time.Now()
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}
	return os.WriteFile(d.MetadataPath(), data, 0644)
}

// fetch downloads a URL and returns the content
func (d *Downloader) fetch(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", d.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	return body, nil
}

// hashContent returns SHA256 hash of content
func hashContent(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// DownloadVocabulary downloads the vocabulary page for a language pair
func (d *Downloader) DownloadVocabulary(ctx context.Context, pair LanguagePair) error {
	if err := d.ensureDirs(); err != nil {
		return err
	}

	path := d.VocabularyPath(pair)

	// Skip if file already exists and has content
	if info, err := os.Stat(path); err == nil && info.Size() > 0 {
		return nil // Already downloaded
	}

	url := pair.VocabularyURL()

	data, err := d.fetch(ctx, url)
	if err != nil {
		return fmt.Errorf("fetch vocabulary %s: %w", pair, err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write vocabulary %s: %w", pair, err)
	}

	// Update metadata
	meta, err := d.loadMetadata()
	if err != nil {
		return err
	}
	meta.Downloads["vocabulary_"+pair.String()] = DownloadInfo{
		URL:         url,
		FetchedAt:   time.Now(),
		ContentHash: hashContent(data),
		Size:        int64(len(data)),
	}
	return d.saveMetadata(meta)
}

// DownloadTips downloads the tips page for a language pair
func (d *Downloader) DownloadTips(ctx context.Context, pair LanguagePair) error {
	if err := d.ensureDirs(); err != nil {
		return err
	}

	path := d.TipsPath(pair)

	// Skip if file already exists and has content
	if info, err := os.Stat(path); err == nil && info.Size() > 0 {
		return nil // Already downloaded
	}

	url := pair.TipsURL()

	data, err := d.fetch(ctx, url)
	if err != nil {
		return fmt.Errorf("fetch tips %s: %w", pair, err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write tips %s: %w", pair, err)
	}

	// Update metadata
	meta, err := d.loadMetadata()
	if err != nil {
		return err
	}
	meta.Downloads["tips_"+pair.String()] = DownloadInfo{
		URL:         url,
		FetchedAt:   time.Now(),
		ContentHash: hashContent(data),
		Size:        int64(len(data)),
	}
	return d.saveMetadata(meta)
}

// DownloadPair downloads both vocabulary and tips for a language pair
func (d *Downloader) DownloadPair(ctx context.Context, pair LanguagePair) error {
	// Check if both files already exist
	vocabPath := d.VocabularyPath(pair)
	tipsPath := d.TipsPath(pair)
	vocabExists := false
	tipsExists := false

	if info, err := os.Stat(vocabPath); err == nil && info.Size() > 0 {
		vocabExists = true
	}
	if info, err := os.Stat(tipsPath); err == nil && info.Size() > 0 {
		tipsExists = true
	}

	// If both exist, skip entirely
	if vocabExists && tipsExists {
		return nil
	}

	// Download vocabulary (will skip if exists)
	if err := d.DownloadVocabulary(ctx, pair); err != nil {
		return fmt.Errorf("download vocabulary: %w", err)
	}

	// Rate limiting delay only if we actually downloaded something
	if !vocabExists {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(d.delay):
		}
	}

	// Download tips (will skip if exists)
	if err := d.DownloadTips(ctx, pair); err != nil {
		return fmt.Errorf("download tips: %w", err)
	}

	return nil
}

// DownloadAll downloads vocabulary and tips for all specified language pairs
func (d *Downloader) DownloadAll(ctx context.Context, pairs []LanguagePair) error {
	total := len(pairs)
	for i, pair := range pairs {
		if d.progress != nil {
			d.progress(i+1, total, fmt.Sprintf("Downloading %s", pair))
		}

		// Check if already downloaded before attempting
		alreadyDownloaded := d.IsPairDownloaded(pair)

		if err := d.DownloadPair(ctx, pair); err != nil {
			// Log error but continue with other pairs
			fmt.Printf("Warning: failed to download %s: %v\n", pair, err)
			continue
		}

		// Rate limiting between pairs only if we actually downloaded something
		if i < total-1 && !alreadyDownloaded {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(d.delay):
			}
		}
	}

	return nil
}

// IsPairDownloaded checks if both vocabulary and tips files exist for a pair
func (d *Downloader) IsPairDownloaded(pair LanguagePair) bool {
	vocabPath := d.VocabularyPath(pair)
	tipsPath := d.TipsPath(pair)

	vocabInfo, vocabErr := os.Stat(vocabPath)
	tipsInfo, tipsErr := os.Stat(tipsPath)

	return vocabErr == nil && vocabInfo.Size() > 0 &&
		tipsErr == nil && tipsInfo.Size() > 0
}

// DownloadAllSupported downloads all supported language pairs
func (d *Downloader) DownloadAllSupported(ctx context.Context) error {
	return d.DownloadAll(ctx, GetSupportedPairs())
}

// DownloadPrimary downloads the primary/most common language pairs
func (d *Downloader) DownloadPrimary(ctx context.Context) error {
	return d.DownloadAll(ctx, GetPrimaryPairs())
}

// IsDownloaded checks if a file has been downloaded
func (d *Downloader) IsDownloaded(fileKey string) bool {
	meta, err := d.loadMetadata()
	if err != nil {
		return false
	}
	_, exists := meta.Downloads[fileKey]
	return exists
}

// GetDownloadInfo returns download info for a file
func (d *Downloader) GetDownloadInfo(fileKey string) (*DownloadInfo, error) {
	meta, err := d.loadMetadata()
	if err != nil {
		return nil, err
	}
	info, exists := meta.Downloads[fileKey]
	if !exists {
		return nil, fmt.Errorf("file %s not found in metadata", fileKey)
	}
	return &info, nil
}

// NeedsUpdate checks if a file needs to be re-downloaded
func (d *Downloader) NeedsUpdate(fileKey string, maxAge time.Duration) bool {
	info, err := d.GetDownloadInfo(fileKey)
	if err != nil {
		return true // Not downloaded yet
	}
	return time.Since(info.FetchedAt) > maxAge
}
