package reddit

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/torrent"
	"github.com/klauspost/compress/zstd"
)

const (
	// SubredditTorrentInfoHash is the Academic Torrents info hash for the
	// top 40k subreddits archive (2005-06 to 2023-12, 2.64 TB, 79,895 files).
	SubredditTorrentInfoHash = "56aa49f9653ba545f48df2e33679f014d2829c10"

	// SubredditTorrentSubdir is the subdirectory inside the torrent.
	SubredditTorrentSubdir = "subreddits23"

	// SubredditTorrentRoot is the root folder name inside the torrent archive.
	// Files are stored as reddit/subreddits23/<name>_{comments,submissions}.zst.
	SubredditTorrentRoot = "reddit"
)

// SubredditTorrentTrackers for the top 40k subreddits torrent.
var SubredditTorrentTrackers = []string{
	"https://academictorrents.com/announce.php",
	"udp://tracker.opentrackr.org:1337/announce",
	"udp://tracker.openbittorrent.com:6969/announce",
	"udp://open.stealth.si:80/announce",
	"udp://exodus.desync.com:6969/announce",
	"udp://tracker.torrent.eu.org:451/announce",
	"udp://tracker.tiny-vps.com:6969/announce",
}

// TorrentDownloadProgress reports torrent download progress.
type TorrentDownloadProgress struct {
	Phase          string // "metadata", "download", "decompress", "done"
	File           string
	BytesCompleted int64
	BytesTotal     int64
	Speed          float64
	PeakSpeed      float64
	Peers          int
	ETA            time.Duration
	Elapsed        time.Duration

	// Decompress phase
	DecompressedBytes int64
}

// TorrentDownloadCallback is called with progress updates.
type TorrentDownloadCallback func(TorrentDownloadProgress)

// DownloadSubredditTorrent downloads a subreddit's data from the top 40k torrent
// using cached metadata and .torrent file for fast startup.
// Returns (found, error) — found=false means the subreddit isn't in the torrent.
func DownloadSubredditTorrent(ctx context.Context, target ArcticTarget, kinds []FileKind,
	meta SubredditMeta, torrentFile string, cb TorrentDownloadCallback) error {

	rawDir := filepath.Join(ArcticDir(), "raw")

	// Build list of files to download
	var paths []string
	var totalSize int64
	for _, k := range kinds {
		var path string
		var size int64
		if k == Comments && meta.CommentsPath != "" {
			path = meta.CommentsPath
			size = meta.CommentsSize
		} else if k == Submissions && meta.SubmissionsPath != "" {
			path = meta.SubmissionsPath
			size = meta.SubmissionsSize
		}
		if path == "" {
			continue
		}

		// Check if JSONL already exists (skip download+decompress)
		jsonlPath := target.JSONLPath(k)
		if _, err := os.Stat(jsonlPath); err == nil {
			continue
		}

		paths = append(paths, path)
		totalSize += size
	}

	if len(paths) == 0 {
		if cb != nil {
			cb(TorrentDownloadProgress{Phase: "done"})
		}
		return nil
	}

	// Create directories
	os.MkdirAll(filepath.Join(rawDir, SubredditTorrentRoot, SubredditTorrentSubdir), 0o755)
	os.MkdirAll(target.Dir(), 0o755)

	// Create torrent client using .torrent file (instant metadata)
	cfg := torrent.Config{
		DataDir:     rawDir,
		InfoHash:    SubredditTorrentInfoHash,
		Trackers:    SubredditTorrentTrackers,
		NoUpload:    true,
		TorrentFile: torrentFile,
	}

	if cb != nil {
		cb(TorrentDownloadProgress{Phase: "metadata"})
	}

	cl, err := torrent.New(cfg)
	if err != nil {
		return fmt.Errorf("create torrent client: %w", err)
	}
	defer cl.Close()

	// Download with timeout for peer discovery
	// If no progress after 60s, give up so caller can fall back to API.
	start := time.Now()
	var lastProgress int64

	// Create a timeout context: cancel if no bytes received for 60s
	dlCtx, dlCancel := context.WithCancel(ctx)
	defer dlCancel()

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		noProgressSince := time.Now()
		for {
			select {
			case <-dlCtx.Done():
				return
			case <-ticker.C:
				current := lastProgress
				if current > 0 {
					noProgressSince = time.Now()
				} else if time.Since(noProgressSince) > 60*time.Second {
					dlCancel() // Timeout — no peers found
					return
				}
			}
		}
	}()

	err = cl.Download(dlCtx, paths, func(p torrent.Progress) {
		lastProgress = p.BytesCompleted
		if cb != nil {
			cb(TorrentDownloadProgress{
				Phase:          "download",
				File:           p.File,
				BytesCompleted: p.BytesCompleted,
				BytesTotal:     p.BytesTotal,
				Speed:          p.Speed,
				PeakSpeed:      p.PeakSpeed,
				Peers:          p.Peers,
				ETA:            p.ETA,
				Elapsed:        p.Elapsed,
			})
		}
	})
	if err != nil {
		if dlCtx.Err() != nil && ctx.Err() == nil {
			// Our timeout fired, not user cancellation — return special error
			return fmt.Errorf("torrent timeout: no peers found after 60s")
		}
		return fmt.Errorf("torrent download: %w", err)
	}

	// Decompress each .zst to JSONL
	for _, p := range paths {
		var kind FileKind
		if strings.HasSuffix(p, "_comments.zst") {
			kind = Comments
		} else {
			kind = Submissions
		}

		zstPath := filepath.Join(rawDir, SubredditTorrentRoot, p)
		jsonlPath := target.JSONLPath(kind)
		os.MkdirAll(filepath.Dir(jsonlPath), 0o755)

		if cb != nil {
			cb(TorrentDownloadProgress{
				Phase:   "decompress",
				File:    filepath.Base(zstPath),
				Elapsed: time.Since(start),
			})
		}

		err := decompressZstToFile(ctx, zstPath, jsonlPath, func(written int64) {
			if cb != nil {
				cb(TorrentDownloadProgress{
					Phase:             "decompress",
					File:              filepath.Base(zstPath),
					DecompressedBytes: written,
					Elapsed:           time.Since(start),
				})
			}
		})
		if err != nil {
			return fmt.Errorf("decompress %s: %w", filepath.Base(zstPath), err)
		}
	}

	if cb != nil {
		cb(TorrentDownloadProgress{Phase: "done", Elapsed: time.Since(start)})
	}

	return nil
}

// decompressZstToFile decompresses a .zst file to an output file.
// Uses 2GB window for Reddit archive compatibility.
func decompressZstToFile(ctx context.Context, zstPath, outPath string, onProgress func(int64)) error {
	in, err := os.Open(zstPath)
	if err != nil {
		return fmt.Errorf("open zst: %w", err)
	}
	defer in.Close()

	dec, err := zstd.NewReader(in, zstd.WithDecoderMaxWindow(1<<31))
	if err != nil {
		return fmt.Errorf("create zstd decoder: %w", err)
	}
	defer dec.Close()

	out, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	defer out.Close()

	buf := make([]byte, 4*1024*1024)
	var written int64
	lastReport := time.Now()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, readErr := dec.Read(buf)
		if n > 0 {
			if _, err := out.Write(buf[:n]); err != nil {
				return fmt.Errorf("write: %w", err)
			}
			written += int64(n)

			if onProgress != nil && time.Since(lastReport) > 500*time.Millisecond {
				lastReport = time.Now()
				onProgress(written)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("read zst: %w", readErr)
		}
	}

	if onProgress != nil {
		onProgress(written)
	}
	return nil
}
