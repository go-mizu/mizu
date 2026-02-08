package reddit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// CachedFile is a file entry in the metadata cache.
type CachedFile struct {
	Path string `json:"path"` // torrent path e.g. "reddit/comments/RC_2005-12.zst"
	Size int64  `json:"size"` // bytes
}

// MetadataCache holds cached torrent metadata.
type MetadataCache struct {
	Files     []CachedFile `json:"files"`
	CachedAt  time.Time    `json:"cached_at"`
	InfoHash  string       `json:"info_hash"`
	TotalSize int64        `json:"total_size"`
}

// CachePath returns the path to the metadata cache file.
func CachePath() string {
	return filepath.Join(DataDir(), "metadata.json")
}

// LoadCache loads the metadata cache from disk. Returns nil if not found or invalid.
func LoadCache() *MetadataCache {
	data, err := os.ReadFile(CachePath())
	if err != nil {
		return nil
	}
	var cache MetadataCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil
	}
	if cache.InfoHash != InfoHash || len(cache.Files) == 0 {
		return nil
	}
	return &cache
}

// SaveCache saves the metadata cache to disk.
func SaveCache(files []CachedFile) error {
	var totalSize int64
	for _, f := range files {
		totalSize += f.Size
	}
	cache := MetadataCache{
		Files:     files,
		CachedAt:  time.Now(),
		InfoHash:  InfoHash,
		TotalSize: totalSize,
	}
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(CachePath()), 0o755); err != nil {
		return err
	}
	return os.WriteFile(CachePath(), data, 0o644)
}

// CachedDataFiles converts cached files to DataFile list, sorted by path.
func CachedDataFiles(cache *MetadataCache) []DataFile {
	var files []DataFile
	for _, cf := range cache.Files {
		// Strip "reddit/" prefix from torrent path to get parseable path
		path := cf.Path
		if len(path) > 7 && path[:7] == "reddit/" {
			path = path[7:]
		}
		df, ok := ParseTorrentPath(path, cf.Size)
		if !ok {
			continue
		}
		files = append(files, df)
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name < files[j].Name
	})
	return files
}

// TorrentPath returns the torrent-internal path for a DataFile.
// Torrent has root folder "reddit/", so: "reddit/comments/RC_2005-12.zst"
func TorrentPath(df DataFile) string {
	return "reddit/" + string(df.Kind) + "/" + df.Name + ".zst"
}
