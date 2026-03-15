package arctic

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/torrent"
)

const bundleInfoHash = "9c263fc85366c1ef8f5bb9da0203f4c8c8db75f4"

var arcticTrackers = []string{
	"https://academictorrents.com/announce.php",
	"udp://tracker.opentrackr.org:1337/announce",
	"udp://tracker.openbittorrent.com:6969/announce",
	"udp://open.stealth.si:80/announce",
	"udp://exodus.desync.com:6969/announce",
	"udp://tracker.torrent.eu.org:451/announce",
}

// monthlyInfoHashes maps "YYYY-MM" to infohash for individual monthly torrents (2024+).
var monthlyInfoHashes = map[string]string{
	// 2024
	"2024-01": "ac88546145ca3227e2b90e51ab477c4527dd8b90",
	"2024-02": "5969ae3e21bb481fea63bf649ec933c222c1f824",
	"2024-03": "deef710de36929e0aa77200fddda73c86142372c",
	"2024-04": "ad4617a3e9c1f52405197fc088b28a8018e12a7a",
	"2024-05": "4f60634d96d35158842cd58b495dc3b444d78b0d",
	"2024-06": "dcdecc93ca9a9d758c045345112771cef5b4989a",
	"2024-07": "6e5300446bd9b328d0b812cdb3022891e086d9ec",
	"2024-08": "8c2d4b00ce8ff9d45e335bed106fe9046c60adb0",
	"2024-09": "43a6e113d6ecacf38e58ecc6caa28d68892dd8af",
	"2024-10": "507dfcda29de9936dd77ed4f34c6442dc675c98f",
	"2024-11": "a1b490117808d9541ab9e3e67a3447e2f4f48f01",
	"2024-12": "eb2017da9f63a49460dde21a4ebe3b7c517f3ad9",
	// 2025
	"2025-01": "4fd14d4c3d792e0b1c5cf6b1d9516c48ba6c4a24",
	"2025-02": "2f873e0b15da5ee29b63e586c0ab1dedd3508870",
	"2025-03": "69d5e046e15c02182430879f50d62b18fe1404fb",
	"2025-04": "552f34df5b830d18f98b69541e7e84f2658346b9",
	"2025-05": "186a0f85a52ff4f1b08677cd312423ace9b34976",
	"2025-06": "bec5590bd3bc6c0f2d868f36ec92bec1aff4480e",
	"2025-07": "b6a7ccf72368a7d39c018c423e01bc15aa551122",
	"2025-08": "c71a97c1f7f676c56963c4e15a81f20afb0109be",
	"2025-09": "a92ce24b4180e4aa9295353f4d26f050031e3058",
	"2025-10": "cb4fa22ea76ea0a2bb38885b27323c94a5d9d16c",
	"2025-11": "2d056b22743718ac81915f25b094b6226668663f",
	"2025-12": "481bf2eac43172ae724fd6c75dbcb8e27de77734",
	// 2026
	"2026-01": "8412b89151101d88c915334c45d9c223169a1a60",
	"2026-02": "c5ba00048236b60f819dbf010e9034d24fc291fb",
}

// zstPrefix returns "RC" for comments, "RS" for submissions.
func zstPrefix(typ string) string {
	if typ == "comments" {
		return "RC"
	}
	return "RS"
}

type DownloadProgress struct {
	Phase      string // "metadata" | "downloading" | "done"
	BytesDone  int64
	BytesTotal int64
	SpeedBps   float64
	Peers      int
	Elapsed    time.Duration
	Message    string // formatted progress line, e.g. "12.3 MB/s  5 peers  ETA 2m30s"
}

type DownloadCallback func(DownloadProgress)

// ErrCorruption is returned when a downloaded file fails integrity checks.
// The caller should delete the file and retry from scratch.
type ErrCorruption struct{ Msg string }

func (e *ErrCorruption) Error() string { return e.Msg }

// ErrTransient is returned for timeout/network errors where the existing
// partial download can be safely resumed by the torrent client.
type ErrTransient struct{ Msg string }

func (e *ErrTransient) Error() string { return e.Msg }

// IsCorruption returns true if err (or its chain) is a corruption error.
func IsCorruption(err error) bool {
	var ce *ErrCorruption
	return errors.As(err, &ce)
}

// DownloadZst downloads RC_YYYY-MM.zst or RS_YYYY-MM.zst to cfg.RawDir.
// Returns download duration. Uses bundle torrent for months <= 2023-12,
// individual torrent otherwise. Cancels after 3 min with no peer progress.
//
// Returns *ErrCorruption for data integrity failures (caller should delete
// and retry) or *ErrTransient for timeout/network failures (caller can keep
// .part file for resume).
func DownloadZst(ctx context.Context, cfg Config, year, month int, typ string,
	cb DownloadCallback) (time.Duration, error) {

	ym := fmt.Sprintf("%04d-%02d", year, month)
	prefix := zstPrefix(typ)
	fileInTorrent := fmt.Sprintf("%s/%s_%s.zst", typ, prefix, ym)

	start := time.Now()

	if cb != nil {
		cb(DownloadProgress{Phase: "metadata"})
	}

	infoHash := bundleInfoHash
	if h, ok := monthlyInfoHashes[ym]; ok {
		infoHash = h
	}

	// For any year ≥ 2024 not covered by the bundle torrent, require a monthly hash.
	if year >= 2024 {
		if _, ok := monthlyInfoHashes[ym]; !ok {
			return 0, fmt.Errorf("no torrent hash for %s: add it to monthlyInfoHashes in torrent.go (see download_links.md)", ym)
		}
	}

	tcfg := torrent.Config{
		DataDir:  cfg.RawDir,
		InfoHash: infoHash,
		Trackers: arcticTrackers,
		NoUpload: true,
	}

	cl, err := torrent.New(tcfg)
	if err != nil {
		return 0, fmt.Errorf("torrent client: %w", err)
	}
	// Do NOT defer cl.Close() here. We must call cl.Close() explicitly BEFORE
	// renaming the .part file, because anacrolix/torrent uses memory-mapped
	// storage: cl.Download() returns when pieces are verified but bytes may
	// still be in OS page cache. cl.Close() triggers the mmap sync + fsync,
	// ensuring all data is on disk before we read the file.

	// Pre-check: query file size from torrent metadata and verify disk space.
	fileSize, sizeErr := cl.FileSize(ctx, fileInTorrent)
	if sizeErr == nil && fileSize > 0 {
		freeGB, diskErr := cfg.FreeDiskGB()
		if diskErr == nil {
			needGB := float64(fileSize) / (1024 * 1024 * 1024) * 2.0 // 2x: .zst + parquet shards
			if freeGB < needGB {
				cl.Close()
				return 0, fmt.Errorf("insufficient disk: need %.1f GB (file %.1f GB + processing), have %.1f GB free",
					needGB, float64(fileSize)/(1024*1024*1024), freeGB)
			}
		}
		if cb != nil {
			cb(DownloadProgress{Phase: "metadata", BytesTotal: fileSize,
				Message: fmt.Sprintf("file size: %.1f GB", float64(fileSize)/(1024*1024*1024))})
		}
	}

	dlCtx, dlCancel := context.WithCancel(ctx)
	defer dlCancel()

	// lastActivity tracks the last time any meaningful progress was observed.
	// Updated on every callback with peers or bytes, and before Download starts.
	var lastActivity atomic.Int64
	lastActivity.Store(time.Now().UnixNano())

	go func() {
		t := time.NewTicker(10 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-dlCtx.Done():
				return
			case <-t.C:
				idle := time.Since(time.Unix(0, lastActivity.Load()))
				if idle > 3*time.Minute {
					dlCancel()
					return
				}
			}
		}
	}()

	err = cl.Download(dlCtx, []string{fileInTorrent}, func(p torrent.Progress) {
		// Any peers or bytes counts as activity (alive, not stalled).
		if p.Peers > 0 || p.BytesCompleted > 0 {
			lastActivity.Store(time.Now().UnixNano())
		}
		if cb != nil {
			msg := fmt.Sprintf("%d peers  connecting…", p.Peers)
			if p.Speed > 0 {
				msg = fmt.Sprintf("%.1f MB/s  %d peers", p.Speed/1e6, p.Peers)
				if p.ETA > 0 && p.BytesTotal > 0 {
					msg += fmt.Sprintf("  ETA %s", p.ETA.Round(time.Second))
				}
			}
			cb(DownloadProgress{
				Phase:      "downloading",
				BytesDone:  p.BytesCompleted,
				BytesTotal: p.BytesTotal,
				SpeedBps:   p.Speed,
				Peers:      p.Peers,
				Elapsed:    p.Elapsed,
				Message:    msg,
			})
		}
	})
	if err != nil {
		cl.Close()
		if dlCtx.Err() != nil && ctx.Err() == nil {
			return 0, &ErrTransient{Msg: fmt.Sprintf("torrent timeout: no progress for 3 minutes on %s", fileInTorrent)}
		}
		return 0, &ErrTransient{Msg: fmt.Sprintf("torrent download %s: %v", fileInTorrent, err)}
	}

	// Close the client BEFORE renaming. This flushes mmap-buffered data to
	// disk (fsync) and triggers anacrolix/torrent's own internal rename.
	// Without this, the .part file may be renamed while bytes are still in
	// page cache, producing a truncated .zst on the next read.
	cl.Close()

	// If anacrolix/torrent renamed the file itself (via Close), we're done.
	// Otherwise look for the .part file and rename it ourselves.
	finalPath := cfg.ZstPath(prefix, ym)
	if _, statErr := os.Stat(finalPath); os.IsNotExist(statErr) {
		partPath := finalPath + ".part"
		if _, perr := os.Stat(partPath); perr == nil {
			if rerr := os.Rename(partPath, finalPath); rerr != nil {
				return 0, fmt.Errorf("rename .part to .zst: %w", rerr)
			}
		}
	}

	dur := time.Since(start)
	if cb != nil {
		cb(DownloadProgress{Phase: "done", Elapsed: dur})
	}
	return dur, nil
}
