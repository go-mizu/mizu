package arctic

import (
	"context"
	"fmt"
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
}

type DownloadCallback func(DownloadProgress)

// DownloadZst downloads RC_YYYY-MM.zst or RS_YYYY-MM.zst to cfg.RawDir.
// Returns download duration. Uses bundle torrent for months <= 2023-12,
// individual torrent otherwise. Cancels after 60s with no peer progress.
func DownloadZst(ctx context.Context, cfg Config, year, month int, typ string,
	cb DownloadCallback) (time.Duration, error) {

	ym := fmt.Sprintf("%04d-%02d", year, month)
	prefix := zstPrefix(typ)
	fileInTorrent := fmt.Sprintf("%s_%s.zst", prefix, ym)

	start := time.Now()

	if cb != nil {
		cb(DownloadProgress{Phase: "metadata"})
	}

	infoHash := bundleInfoHash
	if h, ok := monthlyInfoHashes[ym]; ok {
		infoHash = h
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
	defer cl.Close()

	dlCtx, dlCancel := context.WithCancel(ctx)
	defer dlCancel()

	var lastBytes int64
	go func() {
		t := time.NewTicker(5 * time.Second)
		defer t.Stop()
		noProgress := time.Now()
		for {
			select {
			case <-dlCtx.Done():
				return
			case <-t.C:
				if lastBytes > 0 {
					noProgress = time.Now()
				} else if time.Since(noProgress) > 60*time.Second {
					dlCancel()
					return
				}
			}
		}
	}()

	err = cl.Download(dlCtx, []string{fileInTorrent}, func(p torrent.Progress) {
		lastBytes = p.BytesCompleted
		if cb != nil {
			cb(DownloadProgress{
				Phase:      "downloading",
				BytesDone:  p.BytesCompleted,
				BytesTotal: p.BytesTotal,
				SpeedBps:   p.Speed,
				Peers:      p.Peers,
				Elapsed:    p.Elapsed,
			})
		}
	})
	if err != nil {
		if dlCtx.Err() != nil && ctx.Err() == nil {
			return 0, fmt.Errorf("torrent timeout: no peers found after 60s for %s", fileInTorrent)
		}
		return 0, fmt.Errorf("torrent download %s: %w", fileInTorrent, err)
	}

	dur := time.Since(start)
	if cb != nil {
		cb(DownloadProgress{Phase: "done", Elapsed: dur})
	}
	return dur, nil
}
