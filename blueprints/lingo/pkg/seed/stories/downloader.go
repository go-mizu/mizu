package stories

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Downloader handles downloading stories from GitHub
type Downloader struct {
	client   *http.Client
	baseDir  string
	progress ProgressCallback
}

// ProgressCallback is called to report download progress
type ProgressCallback func(current, total int, message string)

// DownloaderOption configures the downloader
type DownloaderOption func(*Downloader)

// WithProgress sets a progress callback
func WithProgress(cb ProgressCallback) DownloaderOption {
	return func(d *Downloader) {
		d.progress = cb
	}
}

// DefaultStoriesBaseDir returns the default stories data directory
func DefaultStoriesBaseDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "data/stories"
	}
	return filepath.Join(home, "data", "blueprints", "lingo", "stories")
}

// NewDownloader creates a new story downloader
func NewDownloader(baseDir string, opts ...DownloaderOption) *Downloader {
	d := &Downloader{
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		baseDir: baseDir,
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

// BaseDir returns the base directory for downloaded files
func (d *Downloader) BaseDir() string {
	return d.baseDir
}

// DownloadPair downloads all stories for a language pair
func (d *Downloader) DownloadPair(ctx context.Context, pair LanguagePair) error {
	// Get the list of available stories for this pair
	stories, err := d.listStories(ctx, pair)
	if err != nil {
		return fmt.Errorf("list stories: %w", err)
	}

	if len(stories) == 0 {
		return fmt.Errorf("no stories found for %s->%s", pair.From, pair.To)
	}

	// Create directory for this pair
	pairDir := filepath.Join(d.baseDir, "raw", pair.From+"-"+pair.To)
	if err := os.MkdirAll(pairDir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// Download each story
	total := len(stories)
	for i, storyPath := range stories {
		if d.progress != nil {
			d.progress(i+1, total, fmt.Sprintf("Downloading %s", filepath.Base(storyPath)))
		}

		if err := d.downloadStory(ctx, pair, storyPath, pairDir); err != nil {
			return fmt.Errorf("download story %s: %w", storyPath, err)
		}

		// Small delay to be respectful to the server
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}

	return nil
}

// DownloadAll downloads stories for all supported pairs
func (d *Downloader) DownloadAll(ctx context.Context, pairs []LanguagePair) error {
	for _, pair := range pairs {
		if err := d.DownloadPair(ctx, pair); err != nil {
			// Log but continue with other pairs
			fmt.Printf("Warning: failed to download %s->%s: %v\n", pair.From, pair.To, err)
		}
	}
	return nil
}

// listStories lists available story files for a language pair
func (d *Downloader) listStories(ctx context.Context, pair LanguagePair) ([]string, error) {
	// Fetch the directory listing from GitHub API
	apiURL := fmt.Sprintf("https://api.github.com/repos/rgerum/unofficial-duolingo-stories-content/contents/%s-%s", pair.From, pair.To)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "Lingo-Story-Downloader/1.0")

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("language pair not found: %s->%s", pair.From, pair.To)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error: %s", resp.Status)
	}

	var files []struct {
		Name string `json:"name"`
		Type string `json:"type"`
		Path string `json:"path"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&files); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	var stories []string
	for _, f := range files {
		if f.Type == "file" && strings.HasSuffix(f.Name, ".txt") {
			stories = append(stories, f.Path)
		}
	}

	return stories, nil
}

// downloadStory downloads a single story file
func (d *Downloader) downloadStory(ctx context.Context, pair LanguagePair, storyPath, destDir string) error {
	// Construct the raw content URL
	url := GitHubRepoURL + storyPath

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Lingo-Story-Downloader/1.0")

	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download error: %s", resp.Status)
	}

	// Read content
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	// Save to file
	filename := filepath.Base(storyPath)
	destPath := filepath.Join(destDir, filename)

	if err := os.WriteFile(destPath, content, 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

// GetDownloadedStories returns paths to all downloaded story files for a pair
func (d *Downloader) GetDownloadedStories(pair LanguagePair) ([]string, error) {
	pairDir := filepath.Join(d.baseDir, "raw", pair.From+"-"+pair.To)

	entries, err := os.ReadDir(pairDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var paths []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".txt") {
			paths = append(paths, filepath.Join(pairDir, entry.Name()))
		}
	}

	return paths, nil
}

// GetPrimaryPairs returns the primary language pairs for stories
func GetPrimaryPairs() []LanguagePair {
	return []LanguagePair{
		{From: "en", To: "es"},
		{From: "en", To: "fr"},
		{From: "en", To: "de"},
		{From: "en", To: "pt"},
		{From: "en", To: "it"},
		{From: "en", To: "ja"},
		{From: "en", To: "ko"},
		{From: "en", To: "zh"},
	}
}

// ReadStoryFile reads and returns the content of a story file
func ReadStoryFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}
