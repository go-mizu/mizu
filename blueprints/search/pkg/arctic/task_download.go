//go:build !windows

package arctic

import (
	"context"
	"fmt"
	"os"
	"time"
)

// reuseExistingZst checks whether a valid .zst file already exists on disk,
// allowing the download to be skipped. A file is reusable if it passes quick
// zstd validation and its size matches the catalog (zst_sizes.json).
func reuseExistingZst(cfg Config, zstSizes ZstSizes, job *PipelineJob) bool {
	prefix := zstPrefix(job.Type)
	zstPath := cfg.ZstPath(prefix, job.YM.String())
	fi, err := os.Stat(zstPath)
	if err != nil || fi.Size() == 0 {
		return false
	}
	if err := QuickValidateZst(zstPath); err != nil {
		os.Remove(zstPath)
		return false
	}
	if expected := zstSizes.Get(job.Type, job.YM.String()); expected > 0 && fi.Size() < expected {
		logf("[%s] %s partial file (%.1f MB of %.1f MB) — re-downloading",
			job.YM.String(), job.Type,
			float64(fi.Size())/(1024*1024), float64(expected)/(1024*1024))
		os.Remove(zstPath)
		return false
	}
	logf("[%s] %s reusing existing %s (%.1f MB)",
		job.YM.String(), job.Type, zstPath, float64(fi.Size())/(1024*1024))
	job.ZstPath = zstPath
	job.DurDown = 0
	return true
}

// downloadOne downloads a single .zst file via BitTorrent. On success,
// job.ZstPath and job.DurDown are populated. The progressFn callback
// receives download progress updates for the UI.
func downloadOne(ctx context.Context, cfg Config, zstSizes ZstSizes, job *PipelineJob, progressFn func(*PublishState)) error {
	prefix := zstPrefix(job.Type)
	zstPath := cfg.ZstPath(prefix, job.YM.String())
	job.ZstPath = zstPath

	if reuseExistingZst(cfg, zstSizes, job) {
		return nil
	}

	// Remove stale file before downloading.
	os.Remove(zstPath)

	if progressFn != nil {
		progressFn(&PublishState{Phase: "download", YM: job.YM.String(), Type: job.Type})
	}

	var expectedBytes int64
	durDown, err := DownloadZst(ctx, cfg, job.YM.Year, job.YM.Month, job.Type, func(p DownloadProgress) {
		if p.BytesTotal > 0 {
			expectedBytes = p.BytesTotal
		}
		if progressFn != nil {
			progressFn(&PublishState{
				Phase: "download", YM: job.YM.String(), Type: job.Type,
				Bytes: p.BytesDone, BytesTotal: p.BytesTotal, Message: p.Message,
			})
		}
	})
	if err != nil {
		return err
	}
	job.DurDown = durDown

	return validateZst(zstPath, expectedBytes)
}

// validateZst checks that a .zst file exists, is non-empty, matches the
// expected size (if known), and passes quick zstd header validation.
func validateZst(path string, expectedBytes int64) error {
	fi, err := os.Stat(path)
	if err != nil {
		return &ErrCorruption{Msg: "file missing after download"}
	}
	if fi.Size() == 0 {
		return &ErrCorruption{Msg: "file is empty"}
	}
	if expectedBytes > 0 && fi.Size() < expectedBytes {
		return &ErrCorruption{Msg: fmt.Sprintf("truncated (%d of %d bytes)", fi.Size(), expectedBytes)}
	}
	if err := QuickValidateZst(path); err != nil {
		return &ErrCorruption{Msg: fmt.Sprintf("validate: %v", err)}
	}
	return nil
}

// downloadWithRetry attempts to download a .zst file, retrying on failure with
// exponential backoff. On corruption, it renames the file to .part so the
// torrent client can resume from verified pieces. Returns nil on success.
func downloadWithRetry(ctx context.Context, cfg Config, zstSizes ZstSizes, job *PipelineJob, progressFn func(*PublishState)) error {
	// First attempt.
	err := downloadOne(ctx, cfg, zstSizes, job, progressFn)
	if err == nil {
		return nil
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}

	prefix := zstPrefix(job.Type)
	zstPath := cfg.ZstPath(prefix, job.YM.String())

	var lastErr error = err
	for attempt := 1; attempt < maxRetries; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Clean up before retry — delete both the partial .zst and any
		// .part file left by the previous torrent session. Keeping the .part
		// causes the new torrent client to resume from the same stuck
		// position with the same unresponsive peers; a clean start lets it
		// announce fresh and pick up different peers.
		os.Remove(zstPath)
		os.Remove(zstPath + ".part")

		backoff := time.Duration(10<<attempt) * time.Second
		if backoff > 5*time.Minute {
			backoff = 5 * time.Minute
		}
		logf("download retry %d/%d for [%s] %s in %s",
			attempt+1, maxRetries, job.YM.String(), job.Type, backoff)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}

		job.Attempt++
		lastErr = downloadOne(ctx, cfg, zstSizes, job, progressFn)
		if lastErr == nil {
			return nil
		}
		logf("download attempt %d/%d failed for [%s] %s: %v",
			attempt+1, maxRetries, job.YM.String(), job.Type, lastErr)
	}
	return fmt.Errorf("download failed after %d attempts (last: %v)", maxRetries, lastErr)
}

// downloadSpeed computes download throughput in Mbps from file size and duration.
func downloadSpeed(zstPath string, dur time.Duration) float64 {
	if dur <= 0 {
		return 0
	}
	fi, err := os.Stat(zstPath)
	if err != nil || fi.Size() == 0 {
		return 0
	}
	return float64(fi.Size()) / dur.Seconds() / (1024 * 1024) * 8
}
