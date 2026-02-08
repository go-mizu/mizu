package reddit

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anacrolix/torrent/metainfo"
)

const SubredditTorrentURL = "https://academictorrents.com/download/56aa49f9653ba545f48df2e33679f014d2829c10.torrent"

// SubredditMeta holds metadata for one subreddit from the torrent.
type SubredditMeta struct {
	Name            string `json:"name"`                        // case-sensitive name (e.g. "golang")
	CommentsPath    string `json:"comments_path,omitempty"`     // torrent path
	CommentsSize    int64  `json:"comments_size,omitempty"`     // compressed size
	SubmissionsPath string `json:"submissions_path,omitempty"`
	SubmissionsSize int64  `json:"submissions_size,omitempty"`
}

// SubredditMetaCache holds the cached torrent metadata.
type SubredditMetaCache struct {
	Subreddits  map[string]SubredditMeta `json:"subreddits"` // lowercased name → meta
	CachedAt    time.Time                `json:"cached_at"`
	InfoHash    string                   `json:"info_hash"`
	TotalFiles  int                      `json:"total_files"`
	TotalSize   int64                    `json:"total_size"`
	TorrentFile string                   `json:"torrent_file"` // path to saved .torrent file
}

// SubredditMetaCachePath returns the path to the metadata cache file.
func SubredditMetaCachePath() string {
	return filepath.Join(DataDir(), "subreddits_meta.json")
}

// SubredditTorrentFilePath returns the path to the saved .torrent file.
func SubredditTorrentFilePath() string {
	return filepath.Join(DataDir(), "subreddits.torrent")
}

// LoadSubredditMetaCache loads the cached subreddit metadata from disk.
func LoadSubredditMetaCache() *SubredditMetaCache {
	data, err := os.ReadFile(SubredditMetaCachePath())
	if err != nil {
		return nil
	}
	var cache SubredditMetaCache
	if json.Unmarshal(data, &cache) != nil {
		return nil
	}
	if cache.InfoHash != SubredditTorrentInfoHash || len(cache.Subreddits) == 0 {
		return nil
	}
	// Verify .torrent file exists
	if _, err := os.Stat(cache.TorrentFile); err != nil {
		return nil
	}
	return &cache
}

// LookupSubreddit checks if a subreddit is in the torrent.
func (c *SubredditMetaCache) LookupSubreddit(name string) (SubredditMeta, bool) {
	meta, ok := c.Subreddits[strings.ToLower(name)]
	return meta, ok
}

// FetchSubredditMeta downloads the .torrent file via HTTP, parses it,
// and caches the subreddit metadata. Much faster than magnet link metadata exchange.
func FetchSubredditMeta(onProgress func(string)) (*SubredditMetaCache, error) {
	torrentPath := SubredditTorrentFilePath()
	os.MkdirAll(filepath.Dir(torrentPath), 0o755)

	// Step 1: Download .torrent file via HTTP
	if onProgress != nil {
		onProgress("Downloading .torrent file from Academic Torrents...")
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(SubredditTorrentURL)
	if err != nil {
		return nil, fmt.Errorf("download .torrent: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("download .torrent: HTTP %d", resp.StatusCode)
	}

	f, err := os.Create(torrentPath)
	if err != nil {
		return nil, fmt.Errorf("create .torrent file: %w", err)
	}
	written, err := io.Copy(f, resp.Body)
	f.Close()
	if err != nil {
		os.Remove(torrentPath)
		return nil, fmt.Errorf("save .torrent file: %w", err)
	}

	if onProgress != nil {
		onProgress(fmt.Sprintf("Downloaded .torrent file (%.1f MB), parsing...", float64(written)/(1<<20)))
	}

	// Step 2: Parse .torrent file to extract all subreddit names and sizes
	mi, err := metainfo.LoadFromFile(torrentPath)
	if err != nil {
		return nil, fmt.Errorf("parse .torrent: %w", err)
	}

	info, err := mi.UnmarshalInfo()
	if err != nil {
		return nil, fmt.Errorf("unmarshal torrent info: %w", err)
	}

	// Step 3: Build subreddit index
	subs := make(map[string]SubredditMeta)
	var totalSize int64

	for _, file := range info.UpvertedFiles() {
		path := file.DisplayPath(&info)
		size := file.Length
		totalSize += size

		base := filepath.Base(path)

		for _, suffix := range []string{"_comments.zst", "_submissions.zst"} {
			if strings.HasSuffix(base, suffix) {
				subName := base[:len(base)-len(suffix)]
				key := strings.ToLower(subName)

				meta := subs[key]
				meta.Name = subName
				if suffix == "_comments.zst" {
					meta.CommentsPath = path
					meta.CommentsSize = size
				} else {
					meta.SubmissionsPath = path
					meta.SubmissionsSize = size
				}
				subs[key] = meta
			}
		}
	}

	// Step 4: Save cache
	cache := &SubredditMetaCache{
		Subreddits:  subs,
		CachedAt:    time.Now(),
		InfoHash:    SubredditTorrentInfoHash,
		TotalFiles:  len(info.UpvertedFiles()),
		TotalSize:   totalSize,
		TorrentFile: torrentPath,
	}

	data, _ := json.MarshalIndent(cache, "", "  ")
	os.WriteFile(SubredditMetaCachePath(), data, 0o644)

	if onProgress != nil {
		onProgress(fmt.Sprintf("Cached %d subreddits (%d files, %.1f TB)",
			len(subs), cache.TotalFiles, float64(totalSize)/(1<<40)))
	}

	return cache, nil
}
