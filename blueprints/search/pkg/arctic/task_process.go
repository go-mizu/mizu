//go:build !windows

package arctic

import (
	"context"
	"fmt"
	"os"
	"time"
)

// processOne runs ProcessZst on a downloaded .zst file, producing parquet shards
// in an isolated work directory. On success, job.ProcResult and job.DurProc are
// populated. The shardFn callback receives per-shard progress updates.
func processOne(ctx context.Context, cfg Config, budget ResourceBudget, job *PipelineJob, shardFn func(ShardResult)) (ProcessResult, error) {
	// Pre-flight: verify the .zst exists before committing to processing.
	if _, err := os.Stat(job.ZstPath); err != nil {
		return ProcessResult{}, &ErrCorruption{Msg: fmt.Sprintf("zst missing before process: %v", err)}
	}

	// Create per-job config with isolated work directory.
	jobCfg := cfg.ForJob(job.YM.String(), job.Type)
	jobCfg.DuckDBMemoryMB = budget.DuckDBMemoryMB
	jobCfg.MaxConvertWorkers = budget.MaxConvertWorkers
	if err := os.MkdirAll(jobCfg.WorkDir, 0o755); err != nil {
		return ProcessResult{}, fmt.Errorf("create job workdir: %w", err)
	}
	job.WorkDir = jobCfg.WorkDir

	year := fmt.Sprintf("%04d", job.YM.Year)
	mm := fmt.Sprintf("%02d", job.YM.Month)

	t0 := time.Now()
	result, err := ProcessZst(ctx, jobCfg, job.ZstPath, job.Type, year, mm, shardFn)
	if err != nil {
		os.RemoveAll(jobCfg.WorkDir)
		return ProcessResult{}, classifyProcessError(err, job.ZstPath)
	}

	// Delete .zst now that the stream is exhausted — parquet shards contain
	// everything needed for the HF upload.
	os.Remove(job.ZstPath)

	job.DurProc = time.Since(t0)
	job.ProcResult = result
	return result, nil
}

// classifyProcessError wraps a processing error with the appropriate type.
// Corruption errors (bad zstd, scan failure, missing file) force a re-download;
// other errors (context canceled, DuckDB failure) are transient.
func classifyProcessError(err error, zstPath string) error {
	errStr := err.Error()
	if containsAny(errStr, "zstd", "scan jsonl", "open zst", "no such file") {
		os.Remove(zstPath)
		return &ErrCorruption{Msg: fmt.Sprintf("process: %v", err)}
	}
	return err
}

// processWithRetry attempts to process a .zst file, retrying on failure.
// On corruption, it deletes the .zst and re-downloads before retrying.
// On transient errors, it retries with the existing .zst after a backoff.
// Returns nil on success. The downloadFn is called when re-download is needed.
func processWithRetry(ctx context.Context, cfg Config, budget ResourceBudget, zstSizes ZstSizes,
	job *PipelineJob, shardFn func(ShardResult), downloadFn func() error) error {

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		_, lastErr = processOne(ctx, cfg, budget, job, shardFn)
		if lastErr == nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}

		corruption := IsCorruption(lastErr)
		kind := "transient"
		if corruption {
			kind = "corruption"
		}
		logf("process attempt %d/%d failed [%s] for [%s] %s: %v",
			attempt+1, maxRetries, kind, job.YM.String(), job.Type, lastErr)

		if corruption && downloadFn != nil {
			// Corrupt .zst — delete and re-download.
			os.Remove(job.ZstPath)
			backoff := time.Duration(10<<attempt) * time.Second
			if backoff > 5*time.Minute {
				backoff = 5 * time.Minute
			}
			logf("re-downloading [%s] %s after corruption (in %s)",
				job.YM.String(), job.Type, backoff)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
			if err := downloadFn(); err != nil {
				logf("re-download failed: %v", err)
				lastErr = err
				continue
			}
		} else {
			// Transient — backoff then retry with existing .zst.
			backoff := time.Duration(10<<attempt) * time.Second
			if backoff > 5*time.Minute {
				backoff = 5 * time.Minute
			}
			logf("retrying process for [%s] %s in %s",
				job.YM.String(), job.Type, backoff)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}
	}
	return fmt.Errorf("process failed after %d attempts: %v", maxRetries, lastErr)
}

// processSpeed computes processing throughput in rows/sec.
func processSpeed(totalRows int64, dur time.Duration) float64 {
	if dur <= 0 {
		return 0
	}
	return float64(totalRows) / dur.Seconds()
}
