//go:build !windows

package arctic

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

const (
	commitBatchSize = 10
	commitMaxRetries = 3
)

// buildCommitOps builds the HuggingFace upload operations for a completed job.
// Returns the data shard ops plus metadata ops (stats.csv, zst_sizes.json,
// README.md, states.json).
func buildCommitOps(cfg Config, job *PipelineJob) []HFOp {
	year := fmt.Sprintf("%04d", job.YM.Year)
	mm := fmt.Sprintf("%02d", job.YM.Month)

	var ops []HFOp
	for _, sr := range job.ProcResult.Shards {
		ops = append(ops, HFOp{
			LocalPath:  sr.LocalPath,
			PathInRepo: cfg.ShardHFPath(job.Type, year, mm, sr.Index),
		})
	}
	ops = append(ops,
		HFOp{LocalPath: cfg.StatsCSVPath(), PathInRepo: "stats.csv"},
		HFOp{LocalPath: cfg.ZstSizesPath(), PathInRepo: "zst_sizes.json"},
		HFOp{LocalPath: cfg.READMEPath(), PathInRepo: "README.md"},
		HFOp{LocalPath: cfg.StatesJSONPath(), PathInRepo: "states.json"},
	)
	return ops
}

// writeLocalCommitFiles writes stats.csv, README.md, and states.json to disk
// in preparation for an HF commit. Returns the full row set including the new row.
func writeLocalCommitFiles(cfg Config, job *PipelineJob, ls *LiveState, zstSizes ZstSizes) ([]StatsRow, StatsRow, error) {
	existingRows, _ := ReadStatsCSV(cfg.StatsCSVPath())
	newRow := StatsRow{
		Year:         job.YM.Year,
		Month:        job.YM.Month,
		Type:         job.Type,
		Shards:       len(job.ProcResult.Shards),
		Count:        job.ProcResult.TotalRows,
		SizeBytes:    job.ProcResult.TotalSize,
		ZstBytes:     zstSizes.Get(job.Type, job.YM.String()),
		DurDownloadS: job.DurDown.Seconds(),
		DurProcessS:  job.DurProc.Seconds(),
		CommittedAt:  time.Now().UTC(),
	}
	allRows := append(existingRows, newRow)

	snap := ls.Snapshot()
	readme, err := GenerateREADMEFull(allRows, &snap, zstSizes)
	if err != nil {
		return nil, StatsRow{}, fmt.Errorf("readme: %w", err)
	}
	if err := os.WriteFile(cfg.READMEPath(), readme, 0o644); err != nil {
		return nil, StatsRow{}, fmt.Errorf("write readme: %w", err)
	}
	if err := WriteStatsCSV(cfg.StatsCSVPath(), allRows); err != nil {
		return nil, StatsRow{}, fmt.Errorf("write stats: %w", err)
	}
	WriteStateJSON(cfg, snap)

	return allRows, newRow, nil
}

// commitToHF uploads parquet shards and metadata to HuggingFace, holding
// commitMu for the duration. On failure it reverts stats.csv to the pre-commit
// state. On success it updates DurCommitS and lastHFCommit.
func commitToHF(ctx context.Context, cfg Config, commitFn CommitFn, job *PipelineJob,
	ls *LiveState, zstSizes ZstSizes,
	commitMu *sync.Mutex, lastHFCommit *atomic.Int64) error {

	t0 := time.Now()

	allRows, newRow, err := writeLocalCommitFiles(cfg, job, ls, zstSizes)
	if err != nil {
		return err
	}
	existingRows := allRows[:len(allRows)-1]

	ops := buildCommitOps(cfg, job)
	msg := fmt.Sprintf("Add %s/%s (%d shards, %s rows)",
		job.Type, job.YM.String(), len(job.ProcResult.Shards),
		fmtCount(job.ProcResult.TotalRows))

	logf("[%s] %s uploading %d ops (%d shards) to HF…",
		job.YM.String(), job.Type, len(ops), len(job.ProcResult.Shards))

	commitMu.Lock()
	for i := 0; i < len(ops); i += commitBatchSize {
		end := i + commitBatchSize
		if end > len(ops) {
			end = len(ops)
		}

		var commitErr error
		for retry := 0; retry < commitMaxRetries; retry++ {
			if ctx.Err() != nil {
				commitMu.Unlock()
				WriteStatsCSV(cfg.StatsCSVPath(), existingRows)
				return ctx.Err()
			}
			_, commitErr = commitFn(ctx, ops[i:end], msg)
			if commitErr == nil {
				break
			}
			if retry < commitMaxRetries-1 {
				wait := time.Duration(5<<retry) * time.Second
				logf("hf commit retry %d/%d in %s: %v", retry+1, commitMaxRetries, wait, commitErr)
				select {
				case <-ctx.Done():
					commitMu.Unlock()
					WriteStatsCSV(cfg.StatsCSVPath(), existingRows)
					return ctx.Err()
				case <-time.After(wait):
				}
			}
		}
		if commitErr != nil {
			commitMu.Unlock()
			WriteStatsCSV(cfg.StatsCSVPath(), existingRows)
			return fmt.Errorf("hf commit (after %d retries): %w", commitMaxRetries, commitErr)
		}
		// Reset stall timer after each successful batch — large months
		// (e.g. 700+ shards) can take hours to upload; without this the
		// stall detector would kill the pipeline mid-upload.
		lastHFCommit.Store(time.Now().UnixNano())
	}

	// Write DurCommitS while still holding the lock.
	durComm := time.Since(t0)
	newRow.DurCommitS = durComm.Seconds()
	allRows[len(allRows)-1] = newRow
	WriteStatsCSV(cfg.StatsCSVPath(), allRows)
	lastHFCommit.Store(time.Now().UnixNano())
	commitMu.Unlock()

	logf("[%s] %s committed in %.1fs (%d shards, %s rows)",
		job.YM.String(), job.Type, durComm.Seconds(),
		len(job.ProcResult.Shards), fmtCount(job.ProcResult.TotalRows))

	job.DurComm = durComm
	return nil
}

// cleanupAfterCommit removes local shard files and the job work directory
// after a successful HF commit.
func cleanupAfterCommit(job *PipelineJob) {
	for _, sr := range job.ProcResult.Shards {
		os.Remove(sr.LocalPath)
	}
	if job.WorkDir != "" {
		os.RemoveAll(job.WorkDir)
	}
}

// writeHeartbeatFiles writes states.json and README.md to local disk.
func writeHeartbeatFiles(cfg Config, ls *LiveState, zstSizes ZstSizes) {
	snap := ls.Snapshot()
	if err := WriteStateJSON(cfg, snap); err != nil {
		logf("heartbeat: write states.json: %v", err)
	}
	rows, _ := ReadStatsCSV(cfg.StatsCSVPath())
	readme, err := GenerateREADMEFull(rows, &snap, zstSizes)
	if err != nil {
		logf("heartbeat: generate readme: %v", err)
		return
	}
	if err := os.WriteFile(cfg.READMEPath(), readme, 0o644); err != nil {
		logf("heartbeat: write readme: %v", err)
	}
}

// commitHeartbeatToHF commits states.json + README.md to HuggingFace.
// If force is false, the commit is skipped when a data commit happened
// within the last 10 minutes.
func commitHeartbeatToHF(ctx context.Context, cfg Config, commitFn CommitFn,
	ls *LiveState, commitMu *sync.Mutex, lastHFCommit *atomic.Int64, force bool) {

	if !force {
		if time.Since(time.Unix(0, lastHFCommit.Load())) < 10*time.Minute {
			return
		}
	}

	snap := ls.Snapshot()
	var msg string
	switch snap.Phase {
	case PhaseDone:
		msg = fmt.Sprintf("Progress: done (%d committed)", snap.Stats.Committed)
	case PhaseIdle:
		msg = fmt.Sprintf("Progress: idle (%d committed)", snap.Stats.Committed)
	default:
		msg = fmt.Sprintf("Progress: pipeline running (%d committed)", snap.Stats.Committed)
	}

	ops := []HFOp{
		{LocalPath: cfg.StatesJSONPath(), PathInRepo: "states.json"},
		{LocalPath: cfg.READMEPath(), PathInRepo: "README.md"},
	}
	if _, err := os.Stat(cfg.StatesJSONPath()); err != nil {
		return
	}

	commitMu.Lock()
	defer commitMu.Unlock()

	if _, err := commitFn(ctx, ops, msg); err != nil {
		logf("heartbeat commit: %v", err)
		return
	}
	lastHFCommit.Store(time.Now().UnixNano())
}
