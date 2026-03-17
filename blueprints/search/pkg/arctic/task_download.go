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
// receives download progress updates for the UI. stallTimeout controls
// how long to wait without byte progress before aborting (0 = 3 min default).
func downloadOne(ctx context.Context, cfg Config, zstSizes ZstSizes, job *PipelineJob, stallTimeout time.Duration, progressFn func(*PublishState)) error {
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
	durDown, err := DownloadZst(ctx, cfg, job.YM.Year, job.YM.Month, job.Type, stallTimeout, func(p DownloadProgress) {
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

// stallTimeouts defines the per-attempt stall patience. Later attempts wait
// longer, giving slow seeders more time to respond before giving up.
var stallTimeouts = []time.Duration{
	3 * time.Minute,  // attempt 1: quick probe
	3 * time.Minute,  // attempt 2: confirm stall position
	5 * time.Minute,  // attempt 3: more patience
	8 * time.Minute,  // attempt 4: generous
	10 * time.Minute, // attempt 5: last resort
}

// downloadWithRetry attempts to download a .zst file, retrying on failure with
// exponential backoff and progressive stall timeouts. Detects structural stalls
// (swarm lacks data beyond a fixed byte position) and fails fast after 2
// consecutive same-position stalls instead of wasting all retry attempts.
func downloadWithRetry(ctx context.Context, cfg Config, zstSizes ZstSizes, job *PipelineJob, progressFn func(*PublishState)) error {
	prefix := zstPrefix(job.Type)
	zstPath := cfg.ZstPath(prefix, job.YM.String())

	var lastStallBytes int64
	var sameStallCount int

	for attempt := 0; attempt < maxRetries; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Clean up before each attempt (including first) — delete both
		// the partial .zst and any .part file from previous sessions.
		if attempt > 0 {
			os.Remove(zstPath)
			os.Remove(zstPath + ".part")

			backoff := time.Duration(10<<attempt) * time.Second
			if backoff > 5*time.Minute {
				backoff = 5 * time.Minute
			}
			logf("download retry %d/%d for [%s] %s in %s (stall timeout %s)",
				attempt+1, maxRetries, job.YM.String(), job.Type, backoff,
				stallTimeoutFor(attempt))

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		// Track peak bytes received during this attempt.
		var peakBytes int64
		wrappedProgress := func(ps *PublishState) {
			if ps.Bytes > peakBytes {
				peakBytes = ps.Bytes
			}
			if progressFn != nil {
				progressFn(ps)
			}
		}

		job.Attempt = attempt
		err := downloadOne(ctx, cfg, zstSizes, job, stallTimeoutFor(attempt), wrappedProgress)
		if err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}

		logf("download attempt %d/%d failed for [%s] %s at %.1f MB: %v",
			attempt+1, maxRetries, job.YM.String(), job.Type,
			float64(peakBytes)/(1024*1024), err)

		// Detect structural stall: if 2 consecutive attempts stall at the
		// same byte position (within 1 MB), the swarm doesn't have data
		// beyond this point. More retries won't help.
		if peakBytes > 0 && lastStallBytes > 0 {
			diff := peakBytes - lastStallBytes
			if diff < 0 {
				diff = -diff
			}
			if diff < 1024*1024 { // within 1 MB
				sameStallCount++
				if sameStallCount >= 2 {
					logf("download: structural stall at %.1f MB for [%s] %s — "+
						"swarm lacks data beyond this point (%d consecutive same-position stalls), giving up",
						float64(peakBytes)/(1024*1024), job.YM.String(), job.Type, sameStallCount+1)
					return fmt.Errorf("structural stall at %.1f MB after %d attempts — "+
						"torrent swarm does not have data beyond this point",
						float64(peakBytes)/(1024*1024), attempt+1)
				}
			} else {
				sameStallCount = 0 // progress was made, reset
			}
		}
		lastStallBytes = peakBytes
	}
	return fmt.Errorf("download failed after %d attempts", maxRetries)
}

// stallTimeoutFor returns the stall timeout for the given attempt index.
func stallTimeoutFor(attempt int) time.Duration {
	if attempt < len(stallTimeouts) {
		return stallTimeouts[attempt]
	}
	return stallTimeouts[len(stallTimeouts)-1]
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
