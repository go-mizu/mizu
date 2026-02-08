// Package torrent provides a thin wrapper around anacrolix/torrent for
// selective file downloads with progress tracking.
package torrent

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
)

// Config configures the torrent client.
type Config struct {
	DataDir  string   // Where to save downloaded files
	InfoHash string   // Torrent info hash (hex)
	Trackers []string // Tracker URLs
	NoUpload bool     // Disable seeding (default true)
	MaxConns int      // Max connections per torrent (default 80)
}

// File represents a file in the torrent.
type File struct {
	Path   string // e.g. "comments/RC_2005-12.zst"
	Length int64  // Size in bytes
}

// Progress reports download progress.
type Progress struct {
	File           string
	BytesCompleted int64
	BytesTotal     int64
	Speed          float64 // bytes/sec (rolling 10s window)
	PeakSpeed      float64
	Peers          int
	ETA            time.Duration
	Elapsed        time.Duration
}

// ProgressCallback is called with download progress updates.
type ProgressCallback func(Progress)

// Client is a torrent client for selective file downloads.
type Client struct {
	cfg Config
	cl  *torrent.Client
	t   *torrent.Torrent
}

// New creates a new torrent client with the given config.
func New(cfg Config) (*Client, error) {
	if cfg.MaxConns <= 0 {
		cfg.MaxConns = 80
	}

	tcfg := torrent.NewDefaultClientConfig()
	tcfg.DataDir = cfg.DataDir
	tcfg.NoUpload = cfg.NoUpload
	tcfg.Seed = false
	tcfg.EstablishedConnsPerTorrent = cfg.MaxConns

	cl, err := torrent.NewClient(tcfg)
	if err != nil {
		return nil, fmt.Errorf("create torrent client: %w", err)
	}

	return &Client{cfg: cfg, cl: cl}, nil
}

// Close shuts down the torrent client.
func (c *Client) Close() {
	if c.cl != nil {
		c.cl.Close()
	}
}

// addTorrent adds the torrent and waits for metadata.
func (c *Client) addTorrent(ctx context.Context) error {
	if c.t != nil {
		return nil
	}

	var ih metainfo.Hash
	if err := ih.FromHexString(c.cfg.InfoHash); err != nil {
		return fmt.Errorf("parse info hash: %w", err)
	}

	spec := &torrent.TorrentSpec{
		InfoHash: ih,
		Trackers: make([][]string, len(c.cfg.Trackers)),
	}
	for i, tr := range c.cfg.Trackers {
		spec.Trackers[i] = []string{tr}
	}

	t, _, err := c.cl.AddTorrentSpec(spec)
	if err != nil {
		return fmt.Errorf("add torrent: %w", err)
	}

	// Wait for metadata
	select {
	case <-t.GotInfo():
	case <-ctx.Done():
		return ctx.Err()
	}

	c.t = t
	return nil
}

// Files lists all files in the torrent. Blocks until metadata is received.
func (c *Client) Files(ctx context.Context) ([]File, error) {
	if err := c.addTorrent(ctx); err != nil {
		return nil, err
	}

	tFiles := c.t.Files()
	files := make([]File, len(tFiles))
	for i, f := range tFiles {
		files[i] = File{
			Path:   f.DisplayPath(),
			Length: f.Length(),
		}
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})
	return files, nil
}

// Download downloads the specified files with progress reporting.
// Only files matching the given paths are downloaded; all others are skipped.
func (c *Client) Download(ctx context.Context, paths []string, cb ProgressCallback) error {
	if err := c.addTorrent(ctx); err != nil {
		return err
	}

	// Build lookup set
	want := make(map[string]bool, len(paths))
	for _, p := range paths {
		want[p] = true
	}

	// Select files for download
	var selected []*torrent.File
	for _, f := range c.t.Files() {
		if want[f.DisplayPath()] {
			f.Download()
			selected = append(selected, f)
		} else {
			f.SetPriority(torrent.PiecePriorityNone)
		}
	}

	if len(selected) == 0 {
		return fmt.Errorf("no matching files found in torrent")
	}

	// Download with progress tracking
	start := time.Now()
	var mu sync.Mutex
	var peakSpeed float64
	var lastBytes int64

	type speedSample struct {
		t     time.Time
		bytes int64
	}
	var samples []speedSample

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			allDone := true
			var totalCompleted, totalSize int64

			for _, f := range selected {
				completed := f.BytesCompleted()
				total := f.Length()
				totalCompleted += completed
				totalSize += total
				if completed < total {
					allDone = false
				}
			}

			// Rolling speed (10s window)
			now := time.Now()
			mu.Lock()
			samples = append(samples, speedSample{now, totalCompleted})
			cutoff := now.Add(-10 * time.Second)
			for len(samples) > 0 && samples[0].t.Before(cutoff) {
				samples = samples[1:]
			}
			var speed float64
			if len(samples) >= 2 {
				first := samples[0]
				last := samples[len(samples)-1]
				dt := last.t.Sub(first.t).Seconds()
				if dt > 0 {
					speed = float64(last.bytes-first.bytes) / dt
				}
			}
			if speed > peakSpeed {
				peakSpeed = speed
			}
			_ = lastBytes
			lastBytes = totalCompleted
			mu.Unlock()

			// ETA
			var eta time.Duration
			if speed > 0 {
				remaining := totalSize - totalCompleted
				eta = time.Duration(float64(remaining)/speed) * time.Second
			}

			if cb != nil {
				// Report per-file progress for the first selected file, aggregate for all
				fileName := ""
				if len(selected) == 1 {
					fileName = selected[0].DisplayPath()
				} else {
					fileName = fmt.Sprintf("%d files", len(selected))
				}
				cb(Progress{
					File:           fileName,
					BytesCompleted: totalCompleted,
					BytesTotal:     totalSize,
					Speed:          speed,
					PeakSpeed:      peakSpeed,
					Peers:          c.t.Stats().ActivePeers,
					ETA:            eta,
					Elapsed:        time.Since(start),
				})
			}

			if allDone {
				return nil
			}
		}
	}
}
