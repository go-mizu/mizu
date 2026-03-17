//go:build !windows

package arctic

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/torrent"
)

// ZstSizes maps "type/YYYY-MM" → .zst file size in bytes from torrent metadata.
// Populated once by FetchZstSizes and persisted in zst_sizes.json.
type ZstSizes map[string]int64

func (z ZstSizes) key(typ, ym string) string { return typ + "/" + ym }

// Get returns the .zst size for the given (type, "YYYY-MM") pair, or 0 if unknown.
func (z ZstSizes) Get(typ, ym string) int64 {
	if z == nil {
		return 0
	}
	return z[z.key(typ, ym)]
}

// Set records a size.
func (z ZstSizes) Set(typ, ym string, size int64) {
	z[z.key(typ, ym)] = size
}

// Len returns the number of known entries.
func (z ZstSizes) Len() int { return len(z) }

// LoadZstSizes reads zst_sizes.json from disk. Returns empty map if file doesn't exist.
func LoadZstSizes(path string) ZstSizes {
	data, err := os.ReadFile(path)
	if err != nil {
		return make(ZstSizes)
	}
	var sizes ZstSizes
	if err := json.Unmarshal(data, &sizes); err != nil {
		return make(ZstSizes)
	}
	return sizes
}

// SaveZstSizes atomically writes the catalog to disk.
func SaveZstSizes(path string, sizes ZstSizes) error {
	data, err := json.MarshalIndent(sizes, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// in-memory cache so we only read disk once per process.
var (
	zstCacheMu   sync.RWMutex
	zstCacheData ZstSizes
	zstCachePath string
)

// LoadZstSizesCached loads the catalog from disk on first call and caches it.
func LoadZstSizesCached(path string) ZstSizes {
	zstCacheMu.RLock()
	if zstCacheData != nil && zstCachePath == path {
		defer zstCacheMu.RUnlock()
		return zstCacheData
	}
	zstCacheMu.RUnlock()

	zstCacheMu.Lock()
	defer zstCacheMu.Unlock()
	zstCacheData = LoadZstSizes(path)
	zstCachePath = path
	return zstCacheData
}

// InvalidateZstSizesCache forces the next call to LoadZstSizesCached to re-read disk.
func InvalidateZstSizesCache() {
	zstCacheMu.Lock()
	zstCacheData = nil
	zstCacheMu.Unlock()
}

// FetchZstSizes queries the bundle torrent (all months ≤ 2023-12) and all
// individual monthly torrents (2024-01 → latest) for .zst file sizes, then
// saves the result to path. This is a one-time network operation; subsequent
// calls load from the cached file.
//
// Runs bundle query first (one torrent, ~2700 files), then each monthly torrent
// sequentially. Each query waits for metadata only — no data is downloaded.
func FetchZstSizes(ctx context.Context, cfg Config, path string) (ZstSizes, error) {
	sizes := make(ZstSizes, 512)

	// ── Bundle torrent (2005-12 → 2023-12) ──────────────────────────────────
	logf("catalog: connecting to bundle torrent for file sizes…")
	bundleCL, err := torrent.New(torrent.Config{
		DataDir:    cfg.RawDir,
		InfoHash:   bundleInfoHash,
		Trackers:   arcticTrackers,
		NoUpload:   true,
		ListenPort: 0, // ephemeral port — avoids conflict with running pipeline
	})
	if err != nil {
		return nil, fmt.Errorf("bundle torrent client: %w", err)
	}
	bundleCtx, bundleCancel := context.WithTimeout(ctx, 5*time.Minute)
	files, err := bundleCL.Files(bundleCtx)
	bundleCancel()
	bundleCL.Close()
	if err != nil {
		return nil, fmt.Errorf("bundle torrent files: %w", err)
	}
	for _, f := range files {
		typ, ym := parseTorrentFilePath(f.Path)
		if typ != "" && ym != "" {
			sizes.Set(typ, ym, f.Length)
		}
	}
	logf("catalog: bundle: %d files → %d .zst entries", len(files), sizes.Len())

	// ── Individual monthly torrents (2024-01 → latest) ───────────────────────
	logf("catalog: fetching %d monthly torrent sizes…", len(monthlyInfoHashes))
	for ym, infoHash := range monthlyInfoHashes {
		if ctx.Err() != nil {
			break
		}
		// Skip if already have both types from the bundle (shouldn't happen, but safe).
		if sizes.Get("comments", ym) > 0 && sizes.Get("submissions", ym) > 0 {
			continue
		}
		cl, err := torrent.New(torrent.Config{
			DataDir:    cfg.RawDir,
			InfoHash:   infoHash,
			Trackers:   arcticTrackers,
			NoUpload:   true,
			ListenPort: 0, // ephemeral port — avoids conflict with running pipeline
		})
		if err != nil {
			logf("catalog: monthly %s: client error: %v", ym, err)
			continue
		}
		tCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		mfiles, err := cl.Files(tCtx)
		cancel()
		cl.Close()
		if err != nil {
			logf("catalog: monthly %s: files error: %v", ym, err)
			continue
		}
		for _, f := range mfiles {
			typ, fym := parseTorrentFilePath(f.Path)
			if typ != "" && fym != "" {
				sizes.Set(typ, fym, f.Length)
			}
		}
		logf("catalog: monthly %s: %d files", ym, len(mfiles))
	}

	logf("catalog: total %d .zst size entries", sizes.Len())

	if err := SaveZstSizes(path, sizes); err != nil {
		return sizes, fmt.Errorf("save catalog: %w", err)
	}
	InvalidateZstSizesCache()
	return sizes, nil
}

// parseTorrentFilePath extracts (typ, "YYYY-MM") from a torrent file path.
// Handles both bundle paths ("reddit/comments/RC_2005-12.zst") and
// monthly paths ("comments/RC_2024-01.zst").
func parseTorrentFilePath(path string) (typ, ym string) {
	base := filepath.Base(path)
	if !strings.HasSuffix(base, ".zst") {
		return "", ""
	}
	// base: "RC_YYYY-MM.zst" or "RS_YYYY-MM.zst"
	stem := base[:len(base)-4] // strip ".zst"
	if len(stem) < 10 || stem[2] != '_' {
		return "", ""
	}
	prefix := stem[:2]
	ym = stem[3:] // "YYYY-MM"
	if len(ym) != 7 {
		return "", ""
	}
	switch prefix {
	case "RC":
		typ = "comments"
	case "RS":
		typ = "submissions"
	default:
		return "", ""
	}

	// Override type from directory name if available (more reliable).
	dir := filepath.Dir(path)
	last := filepath.Base(dir)
	if last == "comments" {
		typ = "comments"
	} else if last == "submissions" {
		typ = "submissions"
	}
	return typ, ym
}
