//go:build !windows

package arctic

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

// PipelineJob represents a single (month, type) pair flowing through the pipeline.
type PipelineJob struct {
	YM         ymKey
	Type       string // "comments" | "submissions"
	ZstPath    string // populated after download
	WorkDir    string // per-job isolated work directory
	ProcResult ProcessResult
	DurDown    time.Duration
	DurProc    time.Duration
	Attempt    int // retry counter
}

// PipelineTask orchestrates the pipelined arctic publish pipeline.
type PipelineTask struct {
	cfg    Config
	opts   PublishOptions
	budget ResourceBudget
	hw     HardwareProfile

	ls           *LiveState
	commitMu     sync.Mutex
	lastHFCommit atomic.Int64

	// throughput tracking (sliding window of last 20 completed jobs)
	tpMu          sync.Mutex
	downloadSpeeds []float64 // MB/s per completed download
	processSpeeds  []float64 // rows/s per completed process
	commitSpeeds   []float64 // seconds per completed commit

	// disk space coordination
	diskMu   sync.Mutex
	diskCond *sync.Cond

	// zst size catalog loaded from zst_sizes.json (torrent metadata)
	zstSizes ZstSizes
}

// NewPipelineTask constructs a PipelineTask with auto-detected hardware.
func NewPipelineTask(cfg Config, opts PublishOptions) *PipelineTask {
	hw := DetectHardware(cfg.WorkDir)
	budget := ComputeBudget(hw, cfg)

	t := &PipelineTask{
		cfg:      cfg,
		opts:     opts,
		budget:   budget,
		hw:       hw,
		zstSizes: LoadZstSizesCached(cfg.ZstSizesPath()),
	}
	t.diskCond = sync.NewCond(&t.diskMu)
	return t
}

// Budget returns the computed resource budget.
func (t *PipelineTask) Budget() ResourceBudget { return t.budget }

// Hardware returns the detected hardware profile.
func (t *PipelineTask) Hardware() HardwareProfile { return t.hw }

// Run executes the pipelined publish. If budget indicates sequential mode,
// it delegates to the existing PublishTask.Run() for maximum stability.
func (t *PipelineTask) Run(ctx context.Context, emit func(*PublishState)) (PublishMetric, error) {
	if t.budget.Sequential {
		// Fall back to proven sequential path.
		seq := NewPublishTask(t.cfg, t.opts)
		return seq.Run(ctx, emit)
	}
	return t.runPipeline(ctx, emit)
}

func (t *PipelineTask) runPipeline(ctx context.Context, emit func(*PublishState)) (PublishMetric, error) {
	start := time.Now()
	metric := PublishMetric{}

	t.cleanupWork()

	rows, err := ReadStatsCSV(t.cfg.StatsCSVPath())
	if err != nil {
		return metric, fmt.Errorf("read stats: %w", err)
	}
	committed := CommittedSet(rows)

	// Build job list — skip already committed.
	months := t.monthRange()
	var jobs []*PipelineJob
	skipped := 0
	for _, ym := range months {
		for _, typ := range []string{"comments", "submissions"} {
			key := ym.String() + "/" + typ
			if committed[key] {
				skipped++
				if emit != nil {
					emit(&PublishState{Phase: "skip", YM: ym.String(), Type: typ})
				}
				continue
			}
			jobs = append(jobs, &PipelineJob{YM: ym, Type: typ})
		}
	}

	totalPairs := len(months) * 2
	t.ls = NewLiveState(totalPairs)
	t.ls.Update(func(s *StateSnapshot) {
		s.Hardware = &t.hw
		s.Budget = &t.budget
		s.Pipeline = &PipelineState{}
		s.Throughput = &ThroughputStats{}
		s.Stats.Skipped = skipped
	})
	t.lastHFCommit.Store(time.Now().UnixNano())

	t.writeHeartbeatFiles()

	// Start heartbeat.
	hbCtx, hbStop := context.WithCancel(ctx)
	defer hbStop()
	go t.runHeartbeat(hbCtx)

	if len(jobs) == 0 {
		metric.Skipped = skipped
		t.finalize(ctx, emit)
		metric.Elapsed = time.Since(start)
		return metric, nil
	}

	// Pipeline channels.
	downloadCh := make(chan *PipelineJob, t.budget.DownloadQueue)
	processCh := make(chan *PipelineJob, t.budget.DownloadQueue)
	uploadCh := make(chan *PipelineJob, t.budget.ProcessQueue)

	// Error collection.
	var pipeErr error
	var pipeErrMu sync.Mutex
	setPipeErr := func(err error) {
		pipeErrMu.Lock()
		if pipeErr == nil {
			pipeErr = err
			logf("pipeline: FATAL — %v", err)
		} else {
			logf("pipeline: (suppressed) %v", err)
		}
		pipeErrMu.Unlock()
	}
	// Pipeline context — cancel all stages on first fatal error.
	pipeCtx, pipeCancel := context.WithCancel(ctx)
	defer pipeCancel()

	var wgDown, wgProc, wgUpload sync.WaitGroup

	// --- Download workers ---
	wgDown.Add(t.budget.MaxDownloads)
	for i := 0; i < t.budget.MaxDownloads; i++ {
		go func() {
			defer wgDown.Done()
			for job := range downloadCh {
				if pipeCtx.Err() != nil {
					continue // drain channel
				}
				if err := t.downloadJob(pipeCtx, job, emit); err != nil {
					if pipeCtx.Err() != nil {
						continue
					}
					// Retry with backoff.
					if retried := t.retryDownload(pipeCtx, job, emit); retried != nil {
						if pipeCtx.Err() != nil {
							// Pipeline was canceled by another goroutine while we
							// were retrying — don't overwrite the real root cause.
							continue
						}
						setPipeErr(fmt.Errorf("[%s] %s: download: %w", job.YM.String(), job.Type, retried))
						pipeCancel()
						continue
					}
				}
				processCh <- job
			}
		}()
	}

	// --- Process workers ---
	wgProc.Add(t.budget.MaxProcess)
	for i := 0; i < t.budget.MaxProcess; i++ {
		go func() {
			defer wgProc.Done()
			for job := range processCh {
				if pipeCtx.Err() != nil {
					continue
				}
				// Auto-heal: retry process failures with classification.
				// - Corruption: delete .zst (force re-download on next run),
				//   then re-download + re-process.
				// - Transient: keep .zst, retry processing after backoff.
				var lastErr error
				for attempt := 0; attempt < maxRetries; attempt++ {
					if pipeCtx.Err() != nil {
						break
					}
					lastErr = t.processJob(pipeCtx, job, emit)
					if lastErr == nil {
						break
					}
					if pipeCtx.Err() != nil {
						break
					}

					corruption := IsCorruption(lastErr)
					kind := "transient"
					if corruption {
						kind = "corruption"
					}
					logf("pipeline: process attempt %d/%d failed [%s] for [%s] %s: %v",
						attempt+1, maxRetries, kind, job.YM.String(), job.Type, lastErr)

					t.ls.Update(func(s *StateSnapshot) { s.Stats.Retries++ })

					if corruption {
						// Corrupt .zst — delete and re-download.
						os.Remove(job.ZstPath)
						backoff := time.Duration(10<<attempt) * time.Second
						if backoff > 5*time.Minute {
							backoff = 5 * time.Minute
						}
						logf("pipeline: re-downloading [%s] %s after corruption (in %s)",
							job.YM.String(), job.Type, backoff)
						select {
						case <-pipeCtx.Done():
							break
						case <-time.After(backoff):
						}
						if err := t.downloadJob(pipeCtx, job, emit); err != nil {
							logf("pipeline: re-download failed: %v", err)
							lastErr = err
							continue
						}
					} else {
						// Transient — backoff then retry with existing .zst.
						backoff := time.Duration(10<<attempt) * time.Second
						if backoff > 5*time.Minute {
							backoff = 5 * time.Minute
						}
						logf("pipeline: retrying process for [%s] %s in %s",
							job.YM.String(), job.Type, backoff)
						select {
						case <-pipeCtx.Done():
							break
						case <-time.After(backoff):
						}
					}
				}
				if lastErr != nil {
					if pipeCtx.Err() == nil {
						setPipeErr(fmt.Errorf("[%s] %s: process (after %d retries): %w",
							job.YM.String(), job.Type, maxRetries, lastErr))
						pipeCancel()
					}
					continue
				}
				uploadCh <- job
			}
		}()
	}

	// --- Upload worker (always 1) ---
	var committed_count int64
	wgUpload.Add(1)
	go func() {
		defer wgUpload.Done()
		for job := range uploadCh {
			if pipeCtx.Err() != nil {
				continue
			}
			// Auto-heal: retry upload failures with backoff.
			// Upload errors are always transient (network/rate-limit) since
			// the parquet shards are already validated locally.
			var lastErr error
			for attempt := 0; attempt < maxRetries; attempt++ {
				if pipeCtx.Err() != nil {
					break
				}
				lastErr = t.uploadJob(pipeCtx, job, emit)
				if lastErr == nil {
					break
				}
				if pipeCtx.Err() != nil {
					break
				}
				backoff := time.Duration(30<<attempt) * time.Second // 30s, 60s, 120s, 240s
				if backoff > 10*time.Minute {
					backoff = 10 * time.Minute
				}
				logf("pipeline: upload attempt %d/%d failed for [%s] %s: %v — retrying in %s",
					attempt+1, maxRetries, job.YM.String(), job.Type, lastErr, backoff)
				t.ls.Update(func(s *StateSnapshot) { s.Stats.Retries++ })
				select {
				case <-pipeCtx.Done():
				case <-time.After(backoff):
				}
			}
			if lastErr != nil {
				if pipeCtx.Err() == nil {
					setPipeErr(fmt.Errorf("[%s] %s: upload (after %d retries): %w",
						job.YM.String(), job.Type, maxRetries, lastErr))
					pipeCancel()
				}
				continue
			}
			atomic.AddInt64(&committed_count, 1)

			// Signal disk space may have freed up.
			t.diskCond.Broadcast()
		}
	}()

	// Feed jobs into download stage.
	go func() {
		for _, job := range jobs {
			// Disk space gate: wait if below threshold.
			t.waitForDisk(pipeCtx)
			if pipeCtx.Err() != nil {
				break
			}
			downloadCh <- job
		}
		close(downloadCh)
	}()

	// Cascade channel closes through pipeline stages.
	go func() {
		wgDown.Wait()
		close(processCh)
	}()
	go func() {
		wgProc.Wait()
		close(uploadCh)
	}()

	// Wait for everything.
	wgUpload.Wait()

	metric.Committed = int(atomic.LoadInt64(&committed_count))
	metric.Skipped = skipped

	if pipeErr != nil {
		return metric, pipeErr
	}

	t.finalize(ctx, emit)
	metric.Elapsed = time.Since(start)
	return metric, nil
}

// downloadJob handles the download stage for a single job.
func (t *PipelineTask) downloadJob(ctx context.Context, job *PipelineJob, emit func(*PublishState)) error {
	cfg := t.cfg
	prefix := zstPrefix(job.Type)
	zstPath := cfg.ZstPath(prefix, job.YM.String())
	job.ZstPath = zstPath

	// Skip download if a valid .zst already exists (e.g. from a previous run
	// that processed successfully but failed during HF upload, or was OOM-killed
	// after processing). This avoids expensive re-downloads of multi-GB files.
	if fi, err := os.Stat(zstPath); err == nil && fi.Size() > 0 {
		if err := QuickValidateZst(zstPath); err == nil {
			logf("pipeline: [%s] %s reusing existing %s (%.1f MB)",
				job.YM.String(), job.Type, zstPath, float64(fi.Size())/(1024*1024))
			job.DurDown = 0
			return nil
		}
		// Existing file is corrupt — remove and re-download.
		os.Remove(zstPath)
	}

	// Update pipeline state.
	t.updatePipelineSlot("downloading", job.YM.String(), job.Type, func(slot *PipelineSlot) {
		slot.Phase = PhaseDownloading
	})

	if emit != nil {
		emit(&PublishState{Phase: "download", YM: job.YM.String(), Type: job.Type})
	}

	var expectedBytes int64
	durDown, err := DownloadZst(ctx, cfg, job.YM.Year, job.YM.Month, job.Type, func(p DownloadProgress) {
		if p.BytesTotal > 0 {
			expectedBytes = p.BytesTotal
		}
		t.updatePipelineSlot("downloading", job.YM.String(), job.Type, func(slot *PipelineSlot) {
			slot.BytesDone = p.BytesDone
			slot.BytesTotal = p.BytesTotal
		})
		if emit != nil {
			emit(&PublishState{Phase: "download", YM: job.YM.String(), Type: job.Type,
				Bytes: p.BytesDone, BytesTotal: p.BytesTotal, Message: p.Message})
		}
	})
	if err != nil {
		t.removePipelineSlot("downloading", job.YM.String(), job.Type)
		return err
	}
	job.DurDown = durDown

	// Validate.
	if fi, statErr := os.Stat(zstPath); statErr != nil {
		t.removePipelineSlot("downloading", job.YM.String(), job.Type)
		return &ErrCorruption{Msg: "file missing after download"}
	} else if fi.Size() == 0 {
		t.removePipelineSlot("downloading", job.YM.String(), job.Type)
		return &ErrCorruption{Msg: "file is empty"}
	} else if expectedBytes > 0 && fi.Size() < expectedBytes {
		t.removePipelineSlot("downloading", job.YM.String(), job.Type)
		return &ErrCorruption{Msg: fmt.Sprintf("truncated (%d of %d bytes)", fi.Size(), expectedBytes)}
	}
	if err := QuickValidateZst(zstPath); err != nil {
		t.removePipelineSlot("downloading", job.YM.String(), job.Type)
		return &ErrCorruption{Msg: fmt.Sprintf("validate: %v", err)}
	}

	// Record download speed.
	if durDown > 0 {
		fi, _ := os.Stat(zstPath)
		if fi != nil {
			mbps := float64(fi.Size()) / durDown.Seconds() / (1024 * 1024) * 8
			t.recordDownloadSpeed(mbps)
		}
	}

	t.removePipelineSlot("downloading", job.YM.String(), job.Type)
	return nil
}

// retryDownload retries a failed download with exponential backoff (max 5 attempts).
func (t *PipelineTask) retryDownload(ctx context.Context, job *PipelineJob, emit func(*PublishState)) error {
	prefix := zstPrefix(job.Type)
	zstPath := t.cfg.ZstPath(prefix, job.YM.String())

	var lastErr error
	for attempt := 1; attempt < maxRetries; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		corruption := job.Attempt > 0 // previous attempt had corruption
		if corruption {
			partPath := zstPath + ".part"
			if _, statErr := os.Stat(zstPath); statErr == nil {
				os.Rename(zstPath, partPath)
			}
		} else {
			os.Remove(zstPath)
		}

		backoff := time.Duration(10<<attempt) * time.Second
		if backoff > 5*time.Minute {
			backoff = 5 * time.Minute
		}
		logf("pipeline: download retry %d/%d for [%s] %s in %s",
			attempt+1, maxRetries, job.YM.String(), job.Type, backoff)

		t.ls.Update(func(s *StateSnapshot) { s.Stats.Retries++ })

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}

		job.Attempt++
		lastErr = t.downloadJob(ctx, job, emit)
		if lastErr == nil {
			return nil
		}
		logf("pipeline: download attempt %d/%d failed for [%s] %s: %v",
			attempt+1, maxRetries, job.YM.String(), job.Type, lastErr)
	}
	return fmt.Errorf("download failed after %d attempts (last: %v)", maxRetries, lastErr)
}

// processJob handles the processing stage for a single job.
func (t *PipelineTask) processJob(ctx context.Context, job *PipelineJob, emit func(*PublishState)) error {
	year := fmt.Sprintf("%04d", job.YM.Year)
	mm := fmt.Sprintf("%02d", job.YM.Month)

	// Pre-flight: verify the .zst file exists before committing to processing.
	// The file may have been deleted by a previous successful process (line ~557)
	// whose upload then failed, or by the corruption handler. Without this check,
	// ProcessZst returns "open zst: no such file" which gets misclassified as
	// transient, wasting all retry attempts on a guaranteed-to-fail operation.
	if _, err := os.Stat(job.ZstPath); err != nil {
		return &ErrCorruption{Msg: fmt.Sprintf("zst missing before process: %v", err)}
	}

	// Create per-job config with isolated work directory and budget-tuned DuckDB memory.
	jobCfg := t.cfg.ForJob(job.YM.String(), job.Type)
	jobCfg.DuckDBMemoryMB = t.budget.DuckDBMemoryMB
	jobCfg.MaxConvertWorkers = t.budget.MaxConvertWorkers
	if err := os.MkdirAll(jobCfg.WorkDir, 0o755); err != nil {
		return fmt.Errorf("create job workdir: %w", err)
	}
	job.WorkDir = jobCfg.WorkDir

	t.updatePipelineSlot("processing", job.YM.String(), job.Type, func(slot *PipelineSlot) {
		slot.Phase = PhaseProcessing
	})

	if emit != nil {
		emit(&PublishState{Phase: "process", YM: job.YM.String(), Type: job.Type})
	}

	t0 := time.Now()
	var procTotalRows int64
	procResult, err := ProcessZst(ctx, jobCfg, job.ZstPath, job.Type, year, mm, func(sr ShardResult) {
		if sr.Starting {
			t.updatePipelineSlot("processing", job.YM.String(), job.Type, func(slot *PipelineSlot) {
				slot.Shard = sr.Index + 1
				slot.Rows = procTotalRows
			})
			if emit != nil {
				emit(&PublishState{Phase: "process_start", YM: job.YM.String(), Type: job.Type,
					Shards: sr.Index + 1, Rows: procTotalRows})
			}
			return
		}
		procTotalRows += sr.Rows
		var rowsPerSec float64
		if sr.Duration > 0 {
			rowsPerSec = float64(sr.Rows) / sr.Duration.Seconds()
		}
		t.updatePipelineSlot("processing", job.YM.String(), job.Type, func(slot *PipelineSlot) {
			slot.Shard = sr.Index + 1
			slot.Rows = procTotalRows
			slot.RowsPerSec = rowsPerSec
		})
		if emit != nil {
			emit(&PublishState{Phase: "process", YM: job.YM.String(), Type: job.Type,
				Shards: sr.Index + 1, Rows: procTotalRows, Bytes: sr.SizeBytes,
				RowsPerSec: rowsPerSec})
		}
	})

	if err != nil {
		t.removePipelineSlot("processing", job.YM.String(), job.Type)
		os.RemoveAll(jobCfg.WorkDir)
		// Classify error for caller.
		errStr := err.Error()
		if containsAny(errStr, "zstd", "scan jsonl", "open zst", "no such file") {
			// Corruption or missing file — delete the .zst so a fresh download
			// is forced on retry. "open zst" / "no such file" means the .zst
			// vanished between download and process (e.g. deleted after a prior
			// successful process whose upload then failed).
			os.Remove(job.ZstPath)
			return &ErrCorruption{Msg: fmt.Sprintf("process: %v", err)}
		}
		// Non-corruption error (e.g. ctx canceled, DuckDB failure) — keep the
		// .zst so it doesn't need to be re-downloaded on retry.
		return err
	}

	// Processing succeeded — delete the .zst now. The parquet shards in
	// jobCfg.WorkDir contain everything needed for the HF upload.
	os.Remove(job.ZstPath)

	job.DurProc = time.Since(t0)
	job.ProcResult = procResult

	// Remap shard paths to use the job-specific work directory.
	// (ProcessZst already writes shards under jobCfg.WorkDir, so paths are correct.)

	// Record process speed.
	if job.DurProc > 0 {
		rps := float64(procResult.TotalRows) / job.DurProc.Seconds()
		t.recordProcessSpeed(rps)
	}

	t.removePipelineSlot("processing", job.YM.String(), job.Type)
	return nil
}

// uploadJob handles the upload stage for a single job.
func (t *PipelineTask) uploadJob(ctx context.Context, job *PipelineJob, emit func(*PublishState)) error {
	year := fmt.Sprintf("%04d", job.YM.Year)
	mm := fmt.Sprintf("%02d", job.YM.Month)
	cfg := t.cfg

	t.updatePipelineSlot("uploading", job.YM.String(), job.Type, func(slot *PipelineSlot) {
		slot.Phase = PhaseCommitting
		slot.Shards = len(job.ProcResult.Shards)
		slot.Rows = job.ProcResult.TotalRows
	})

	if emit != nil {
		emit(&PublishState{Phase: "commit", YM: job.YM.String(), Type: job.Type,
			Shards: len(job.ProcResult.Shards), Rows: job.ProcResult.TotalRows,
			Bytes: job.ProcResult.TotalSize})
	}

	t2 := time.Now()

	// Read existing stats and append new row.
	existingRows, _ := ReadStatsCSV(cfg.StatsCSVPath())
	newRow := StatsRow{
		Year:         job.YM.Year,
		Month:        job.YM.Month,
		Type:         job.Type,
		Shards:       len(job.ProcResult.Shards),
		Count:        job.ProcResult.TotalRows,
		SizeBytes:    job.ProcResult.TotalSize,
		ZstBytes:     t.zstSizes.Get(job.Type, job.YM.String()),
		DurDownloadS: job.DurDown.Seconds(),
		DurProcessS:  job.DurProc.Seconds(),
		CommittedAt:  time.Now().UTC(),
	}
	allRows := append(existingRows, newRow)

	// Write local files before commit.
	snap := t.ls.Snapshot()
	readme, err := GenerateREADMEFull(allRows, &snap, t.zstSizes)
	if err != nil {
		t.removePipelineSlot("uploading", job.YM.String(), job.Type)
		return fmt.Errorf("readme: %w", err)
	}
	if err := os.WriteFile(cfg.READMEPath(), readme, 0o644); err != nil {
		t.removePipelineSlot("uploading", job.YM.String(), job.Type)
		return fmt.Errorf("write readme: %w", err)
	}
	if err := WriteStatsCSV(cfg.StatsCSVPath(), allRows); err != nil {
		t.removePipelineSlot("uploading", job.YM.String(), job.Type)
		return fmt.Errorf("write stats: %w", err)
	}
	WriteStateJSON(cfg, snap)

	// Build HF ops.
	var ops []HFOp
	for _, sr := range job.ProcResult.Shards {
		ops = append(ops, HFOp{
			LocalPath:  sr.LocalPath,
			PathInRepo: cfg.ShardHFPath(job.Type, year, mm, sr.Index),
		})
	}
	ops = append(ops,
		HFOp{LocalPath: cfg.StatsCSVPath(),  PathInRepo: "stats.csv"},
		HFOp{LocalPath: cfg.ZstSizesPath(),  PathInRepo: "zst_sizes.json"},
		HFOp{LocalPath: cfg.READMEPath(),    PathInRepo: "README.md"},
		HFOp{LocalPath: cfg.StatesJSONPath(), PathInRepo: "states.json"},
	)

	// Batch commit — hold commitMu.
	const batchSize = 50
	const hfMaxRetries = 3
	logf("pipeline: [%s] %s uploading %d ops (%d shards) to HF…",
		job.YM.String(), job.Type, len(ops), len(job.ProcResult.Shards))
	t.commitMu.Lock()
	for i := 0; i < len(ops); i += batchSize {
		end := i + batchSize
		if end > len(ops) {
			end = len(ops)
		}
		msg := fmt.Sprintf("Add %s/%s (%d shards, %s rows)",
			job.Type, job.YM.String(), len(job.ProcResult.Shards),
			fmtCount(job.ProcResult.TotalRows))

		var commitErr error
		for retry := 0; retry < hfMaxRetries; retry++ {
			if ctx.Err() != nil {
				t.commitMu.Unlock()
				// Revert stats.csv — remove the row we just added so it
				// doesn't falsely mark this month as committed on restart.
				WriteStatsCSV(cfg.StatsCSVPath(), existingRows)
				t.removePipelineSlot("uploading", job.YM.String(), job.Type)
				return ctx.Err()
			}
			_, commitErr = t.opts.HFCommit(ctx, ops[i:end], msg)
			if commitErr == nil {
				break
			}
			if retry < hfMaxRetries-1 {
				wait := time.Duration(5<<retry) * time.Second
				logf("pipeline: hf commit retry %d/%d in %s: %v",
					retry+1, hfMaxRetries, wait, commitErr)
				select {
				case <-ctx.Done():
					t.commitMu.Unlock()
					WriteStatsCSV(cfg.StatsCSVPath(), existingRows)
					t.removePipelineSlot("uploading", job.YM.String(), job.Type)
					return ctx.Err()
				case <-time.After(wait):
				}
			}
		}
		if commitErr != nil {
			t.commitMu.Unlock()
			// Revert stats.csv — this month was NOT committed.
			WriteStatsCSV(cfg.StatsCSVPath(), existingRows)
			t.removePipelineSlot("uploading", job.YM.String(), job.Type)
			return fmt.Errorf("hf commit (after %d retries): %w", hfMaxRetries, commitErr)
		}
	}
	t.lastHFCommit.Store(time.Now().UnixNano())
	t.commitMu.Unlock()

	durComm := time.Since(t2)
	logf("pipeline: [%s] %s committed in %.1fs (%d shards, %s rows)",
		job.YM.String(), job.Type, durComm.Seconds(),
		len(job.ProcResult.Shards), fmtCount(job.ProcResult.TotalRows))
	t.recordCommitSpeed(durComm.Seconds())

	// Update stats.csv with commit duration.
	newRow.DurCommitS = durComm.Seconds()
	allRows[len(allRows)-1] = newRow
	WriteStatsCSV(cfg.StatsCSVPath(), allRows)

	// Cleanup: delete local shards and job work directory.
	for _, sr := range job.ProcResult.Shards {
		os.Remove(sr.LocalPath)
	}
	if job.WorkDir != "" {
		os.RemoveAll(job.WorkDir)
	}

	// Update live state.
	t.ls.Update(func(s *StateSnapshot) {
		s.Stats.Committed++
		s.Stats.TotalRows += job.ProcResult.TotalRows
		s.Stats.TotalBytes += job.ProcResult.TotalSize
	})
	t.removePipelineSlot("uploading", job.YM.String(), job.Type)

	if emit != nil {
		emit(&PublishState{
			Phase:   "committed",
			YM:      job.YM.String(),
			Type:    job.Type,
			Shards:  len(job.ProcResult.Shards),
			Rows:    job.ProcResult.TotalRows,
			Bytes:   job.ProcResult.TotalSize,
			DurDown: job.DurDown,
			DurProc: job.DurProc,
			DurComm: durComm,
		})
	}

	return nil
}

// waitForDisk blocks until disk space is above MinFreeGB.
func (t *PipelineTask) waitForDisk(ctx context.Context) {
	t.diskMu.Lock()
	defer t.diskMu.Unlock()

	for {
		if ctx.Err() != nil {
			return
		}
		free, err := t.cfg.FreeDiskGB()
		if err != nil || free >= float64(t.cfg.MinFreeGB) {
			return
		}
		logf("pipeline: %.1f GB free (need %d GB) — waiting for uploads to free space",
			free, t.cfg.MinFreeGB)

		// Wait with timeout so we don't block forever.
		done := make(chan struct{})
		go func() {
			t.diskCond.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-ctx.Done():
			t.diskCond.Broadcast() // wake up the goroutine
			return
		case <-time.After(30 * time.Second):
			// Re-check disk even without signal.
		}
	}
}

// updatePipelineSlot updates or inserts a pipeline slot for the given stage.
func (t *PipelineTask) updatePipelineSlot(stage, ym, typ string, fn func(*PipelineSlot)) {
	t.ls.Update(func(s *StateSnapshot) {
		s.Phase = "running"
		if s.Pipeline == nil {
			s.Pipeline = &PipelineState{}
		}

		var slots *[]PipelineSlot
		switch stage {
		case "downloading":
			slots = &s.Pipeline.Downloading
		case "processing":
			slots = &s.Pipeline.Processing
		case "uploading":
			slots = &s.Pipeline.Uploading
		default:
			return
		}

		// Find existing or create new.
		for i := range *slots {
			if (*slots)[i].YM == ym && (*slots)[i].Type == typ {
				fn(&(*slots)[i])
				return
			}
		}
		slot := PipelineSlot{YM: ym, Type: typ}
		fn(&slot)
		*slots = append(*slots, slot)

		// Update Current to reflect the most active job.
		t.updateCurrent(s)
	})
}

// removePipelineSlot removes a pipeline slot from the given stage.
func (t *PipelineTask) removePipelineSlot(stage, ym, typ string) {
	t.ls.Update(func(s *StateSnapshot) {
		if s.Pipeline == nil {
			return
		}

		var slots *[]PipelineSlot
		switch stage {
		case "downloading":
			slots = &s.Pipeline.Downloading
		case "processing":
			slots = &s.Pipeline.Processing
		case "uploading":
			slots = &s.Pipeline.Uploading
		default:
			return
		}

		for i := range *slots {
			if (*slots)[i].YM == ym && (*slots)[i].Type == typ {
				*slots = append((*slots)[:i], (*slots)[i+1:]...)
				break
			}
		}

		t.updateCurrent(s)
	})
}

// updateCurrent sets the Current field to the "most active" pipeline slot
// for backward compatibility with the single-job view.
func (t *PipelineTask) updateCurrent(s *StateSnapshot) {
	if s.Pipeline == nil {
		s.Current = nil
		return
	}

	// Priority: uploading > processing > downloading.
	if len(s.Pipeline.Uploading) > 0 {
		slot := s.Pipeline.Uploading[0]
		s.Current = &LiveCurrent{YM: slot.YM, Type: slot.Type, Phase: PhaseCommitting}
		return
	}
	if len(s.Pipeline.Processing) > 0 {
		slot := s.Pipeline.Processing[0]
		s.Current = &LiveCurrent{
			YM: slot.YM, Type: slot.Type, Phase: PhaseProcessing,
			Shard: slot.Shard, Rows: slot.Rows,
		}
		return
	}
	if len(s.Pipeline.Downloading) > 0 {
		slot := s.Pipeline.Downloading[0]
		s.Current = &LiveCurrent{
			YM: slot.YM, Type: slot.Type, Phase: PhaseDownloading,
			BytesDone: slot.BytesDone, BytesTotal: slot.BytesTotal,
		}
		return
	}
	s.Current = nil
}

// Throughput tracking.

func (t *PipelineTask) recordDownloadSpeed(mbps float64) {
	t.tpMu.Lock()
	defer t.tpMu.Unlock()
	t.downloadSpeeds = appendWindow(t.downloadSpeeds, mbps, 20)
	t.updateThroughput()
}

func (t *PipelineTask) recordProcessSpeed(rowsPerSec float64) {
	t.tpMu.Lock()
	defer t.tpMu.Unlock()
	t.processSpeeds = appendWindow(t.processSpeeds, rowsPerSec, 20)
	t.updateThroughput()
}

func (t *PipelineTask) recordCommitSpeed(secs float64) {
	t.tpMu.Lock()
	defer t.tpMu.Unlock()
	t.commitSpeeds = appendWindow(t.commitSpeeds, secs, 20)
	t.updateThroughput()
}

func (t *PipelineTask) updateThroughput() {
	// Must be called with tpMu held.
	t.ls.Update(func(s *StateSnapshot) {
		if s.Throughput == nil {
			s.Throughput = &ThroughputStats{}
		}
		s.Throughput.AvgDownloadMbps = avg(t.downloadSpeeds)
		s.Throughput.AvgProcessRowsPerSec = avg(t.processSpeeds)
		s.Throughput.AvgUploadSecPerCommit = avg(t.commitSpeeds)

		// Estimate completion from committed + remaining pairs.
		remaining := s.Stats.TotalMonths - s.Stats.Committed - s.Stats.Skipped
		if remaining > 0 && s.Stats.Committed > 0 {
			elapsed := time.Since(s.StartedAt)
			perPair := elapsed / time.Duration(s.Stats.Committed)
			eta := time.Now().Add(perPair * time.Duration(remaining))
			s.Throughput.EstimatedCompletion = &eta
		}
	})
}

func appendWindow(buf []float64, v float64, maxLen int) []float64 {
	buf = append(buf, v)
	if len(buf) > maxLen {
		buf = buf[len(buf)-maxLen:]
	}
	return buf
}

func avg(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	var sum float64
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if len(sub) > 0 && len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}

// Heartbeat — same pattern as PublishTask.

func (t *PipelineTask) runHeartbeat(ctx context.Context) {
	writeTick := time.NewTicker(1 * time.Minute)
	commitTick := time.NewTicker(10 * time.Minute)
	defer writeTick.Stop()
	defer commitTick.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-writeTick.C:
			t.writeHeartbeatFiles()
		case <-commitTick.C:
			if time.Since(time.Unix(0, t.lastHFCommit.Load())) < 10*time.Minute {
				continue
			}
			t.writeHeartbeatFiles()
			t.commitHeartbeat(ctx, false)
		}
	}
}

func (t *PipelineTask) writeHeartbeatFiles() {
	snap := t.ls.Snapshot()
	if err := WriteStateJSON(t.cfg, snap); err != nil {
		logf("pipeline heartbeat: write states.json: %v", err)
	}
	rows, _ := ReadStatsCSV(t.cfg.StatsCSVPath())
	readme, err := GenerateREADMEFull(rows, &snap, t.zstSizes)
	if err != nil {
		logf("pipeline heartbeat: generate readme: %v", err)
		return
	}
	if err := os.WriteFile(t.cfg.READMEPath(), readme, 0o644); err != nil {
		logf("pipeline heartbeat: write readme: %v", err)
	}
}

func (t *PipelineTask) commitHeartbeat(ctx context.Context, force bool) {
	if !force {
		if time.Since(time.Unix(0, t.lastHFCommit.Load())) < 10*time.Minute {
			return
		}
	}
	snap := t.ls.Snapshot()
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
		{LocalPath: t.cfg.StatesJSONPath(), PathInRepo: "states.json"},
		{LocalPath: t.cfg.READMEPath(), PathInRepo: "README.md"},
	}
	if _, err := os.Stat(t.cfg.StatesJSONPath()); err != nil {
		return
	}

	t.commitMu.Lock()
	defer t.commitMu.Unlock()

	if _, err := t.opts.HFCommit(ctx, ops, msg); err != nil {
		logf("pipeline heartbeat commit: %v", err)
		return
	}
	t.lastHFCommit.Store(time.Now().UnixNano())
}

func (t *PipelineTask) finalize(ctx context.Context, emit func(*PublishState)) {
	t.ls.Update(func(s *StateSnapshot) {
		s.Phase = PhaseDone
		s.Current = nil
		s.Pipeline = nil
	})
	t.writeHeartbeatFiles()
	t.commitHeartbeat(ctx, true)
	if emit != nil {
		emit(&PublishState{Phase: "done"})
	}
}

// cleanupWork removes leftover work files from interrupted previous runs.
func (t *PipelineTask) cleanupWork() {
	// Clean pipeline job directories.
	matches, _ := filepath.Glob(filepath.Join(t.cfg.WorkDir, "pipeline_*"))
	for _, m := range matches {
		os.RemoveAll(m)
	}

	// Also clean legacy chunk files.
	chunks, _ := filepath.Glob(filepath.Join(t.cfg.WorkDir, "chunk_*.jsonl"))
	for _, m := range chunks {
		os.Remove(m)
	}
	for _, typ := range []string{"comments", "submissions"} {
		dir := filepath.Join(t.cfg.WorkDir, typ)
		os.RemoveAll(dir)
	}
	// Clean incomplete .part downloads but keep completed .zst files — they
	// will be reused by downloadJob if they pass QuickValidateZst, avoiding
	// expensive re-downloads after OOM kills or upload failures.
	for _, sub := range []string{"comments", "submissions"} {
		dir := filepath.Join(t.cfg.RawDir, "reddit", sub)
		for _, glob := range []string{"R[CS]_*.zst.part"} {
			subMatches, _ := filepath.Glob(filepath.Join(dir, glob))
			for _, m := range subMatches {
				os.Remove(m)
			}
		}
	}
}

// monthRange duplicates the logic from PublishTask for consistency.
func (t *PipelineTask) monthRange() []ymKey {
	from := ymKey{Year: t.opts.FromYear, Month: t.opts.FromMonth}
	to := ymKey{Year: t.opts.ToYear, Month: t.opts.ToMonth}
	if from.Year == 0 {
		from = ymKey{Year: 2005, Month: 12}
	}
	if to.Year == 0 {
		now := time.Now().UTC()
		to = ymKey{Year: now.Year(), Month: int(now.Month())}
	}
	var keys []ymKey
	cur := from
	for !ymAfter(cur, to) {
		keys = append(keys, cur)
		cur.Month++
		if cur.Month > 12 {
			cur.Month = 1
			cur.Year++
		}
	}
	return keys
}
