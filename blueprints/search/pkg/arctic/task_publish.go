package arctic

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// PublishOptions configures the publish run.
type PublishOptions struct {
	FromYear  int
	FromMonth int
	ToYear    int
	ToMonth   int
	HFCommit  CommitFn
}

// PublishState is emitted on each significant event.
type PublishState struct {
	Phase      string // "skip"|"download"|"process_start"|"process"|"commit"|"committed"|"disk_check"|"done"
	YM         string // "YYYY-MM"
	Type       string // "comments"|"submissions"
	Shards     int
	Rows       int64
	Bytes      int64
	BytesTotal int64   // used during download phase
	RowsPerSec float64 // rows/s for the last completed shard
	DurDown    time.Duration
	DurProc    time.Duration
	DurComm    time.Duration
	Message    string
}

// PublishMetric is the aggregate result returned on completion.
type PublishMetric struct {
	Committed int
	Skipped   int
	Elapsed   time.Duration
}

// PublishTask orchestrates the full arctic publish pipeline.
type PublishTask struct {
	cfg  Config
	opts PublishOptions

	// heartbeat state — initialised in Run()
	ls           *LiveState
	commitMu     sync.Mutex   // serialises all HF API calls (data + heartbeat)
	lastHFCommit atomic.Int64 // unix nanos of last successful HF commit
}

// NewPublishTask constructs a PublishTask.
func NewPublishTask(cfg Config, opts PublishOptions) *PublishTask {
	return &PublishTask{cfg: cfg, opts: opts}
}

// Run executes the publish loop. Calls emit on state changes.
// Iterates months oldest-first, processes comments then submissions per month.
func (t *PublishTask) Run(ctx context.Context, emit func(*PublishState)) (PublishMetric, error) {
	start := time.Now()
	metric := PublishMetric{}

	t.cleanupWork()

	rows, err := ReadStatsCSV(t.cfg.StatsCSVPath())
	if err != nil {
		return metric, fmt.Errorf("read stats: %w", err)
	}
	committed := CommittedSet(rows)

	months := t.monthRange()

	// Initialise live state with total work count (months × 2 types).
	t.ls = NewLiveState(len(months) * 2)
	t.lastHFCommit.Store(time.Now().UnixNano())

	// Write initial states.json immediately.
	t.writeHeartbeatFiles()

	// Start background heartbeat goroutine.
	hbCtx, hbStop := context.WithCancel(ctx)
	defer hbStop()
	go t.runHeartbeat(hbCtx)

	for _, ym := range months {
		for _, typ := range []string{"comments", "submissions"} {
			if err := ctx.Err(); err != nil {
				return metric, err
			}

			key := ym.String() + "/" + typ

			if committed[key] {
				metric.Skipped++
				t.ls.Update(func(s *StateSnapshot) { s.Stats.Skipped++ })
				if emit != nil {
					emit(&PublishState{Phase: "skip", YM: ym.String(), Type: typ})
				}
				continue
			}

			free, err := t.cfg.FreeDiskGB()
			if err != nil {
				return metric, fmt.Errorf("disk check: %w", err)
			}
			if free < float64(t.cfg.MinFreeGB) {
				msg := fmt.Sprintf("only %.1f GB free, need %d GB — stopping", free, t.cfg.MinFreeGB)
				if emit != nil {
					emit(&PublishState{Phase: "disk_check", YM: ym.String(), Type: typ, Message: msg})
				}
				return metric, fmt.Errorf("disk full: %s", msg)
			}

			if err := t.processOneWithRetry(ctx, ym, typ, rows, emit); err != nil {
				return metric, fmt.Errorf("[%s] %s: %w", ym.String(), typ, err)
			}

			rows, _ = ReadStatsCSV(t.cfg.StatsCSVPath())
			committed = CommittedSet(rows)
			metric.Committed++
		}
	}

	// Mark done and do a final heartbeat commit.
	t.ls.Update(func(s *StateSnapshot) {
		s.Phase = PhaseDone
		s.Current = nil
	})
	t.writeHeartbeatFiles()
	t.commitHeartbeat(ctx, true /* force */)

	metric.Elapsed = time.Since(start)
	if emit != nil {
		emit(&PublishState{Phase: "done"})
	}
	return metric, nil
}

// runHeartbeat runs in a goroutine: writes files every minute, commits every 10 minutes.
func (t *PublishTask) runHeartbeat(ctx context.Context) {
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
			// Skip if a data commit just happened within the last 10 minutes.
			if time.Since(time.Unix(0, t.lastHFCommit.Load())) < 10*time.Minute {
				continue
			}
			t.writeHeartbeatFiles()
			t.commitHeartbeat(ctx, false)
		}
	}
}

// writeHeartbeatFiles writes states.json and README.md to local disk.
// Errors are logged but never fatal.
func (t *PublishTask) writeHeartbeatFiles() {
	snap := t.ls.Snapshot()

	if err := WriteStateJSON(t.cfg, snap); err != nil {
		logf("heartbeat: write states.json: %v", err)
	}

	rows, _ := ReadStatsCSV(t.cfg.StatsCSVPath())
	readme, err := GenerateREADMEWithLive(rows, &snap)
	if err != nil {
		logf("heartbeat: generate readme: %v", err)
		return
	}
	if err := os.WriteFile(t.cfg.READMEPath(), readme, 0o644); err != nil {
		logf("heartbeat: write readme: %v", err)
	}
}

// commitHeartbeat commits states.json + README.md to HuggingFace.
// force=true skips the cooldown check (used for the final done commit).
func (t *PublishTask) commitHeartbeat(ctx context.Context, force bool) {
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
		if snap.Current != nil {
			if snap.Current.Shard > 0 {
				msg = fmt.Sprintf("Progress: %s %s/%s shard %d",
					snap.Phase, snap.Current.YM, snap.Current.Type, snap.Current.Shard)
			} else {
				msg = fmt.Sprintf("Progress: %s %s/%s",
					snap.Phase, snap.Current.YM, snap.Current.Type)
			}
		} else {
			msg = fmt.Sprintf("Progress: %s", snap.Phase)
		}
	}

	ops := []HFOp{
		{LocalPath: t.cfg.StatesJSONPath(), PathInRepo: "states.json"},
		{LocalPath: t.cfg.READMEPath(), PathInRepo: "README.md"},
	}

	// states.json may not exist yet on first heartbeat if writeHeartbeatFiles failed.
	if _, err := os.Stat(t.cfg.StatesJSONPath()); err != nil {
		return
	}

	t.commitMu.Lock()
	defer t.commitMu.Unlock()

	if _, err := t.opts.HFCommit(ctx, ops, msg); err != nil {
		logf("heartbeat commit: %v", err)
		return
	}
	t.lastHFCommit.Store(time.Now().UnixNano())
}

const maxRetries = 5

// processOneWithRetry wraps processOne with auto-heal: on failure it classifies
// the error and applies appropriate cleanup before retrying.
//
// Error classification:
//   - Corruption errors (bad zstd, truncated, zero-filled): delete .zst and .part,
//     force a clean re-download.
//   - Transient errors (timeout, no peers, network): keep .part file so the torrent
//     client can resume from existing pieces on the next attempt.
//
// Uses exponential backoff: 10s, 20s, 40s, 80s, 160s between retries.
func (t *PublishTask) processOneWithRetry(ctx context.Context, ym ymKey, typ string,
	existingRows []StatsRow, emit func(*PublishState)) error {

	cfg := t.cfg
	prefix := zstPrefix(typ)
	year := fmt.Sprintf("%04d", ym.Year)
	mm := fmt.Sprintf("%02d", ym.Month)

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		lastErr = t.processOne(ctx, ym, typ, existingRows, emit)
		if lastErr == nil {
			return nil
		}
		if ctx.Err() != nil {
			return lastErr
		}

		corruption := IsCorruption(lastErr)
		kind := "transient"
		if corruption {
			kind = "corruption"
		}
		logf("attempt %d/%d failed [%s] for [%s] %s: %v — cleaning up and retrying",
			attempt, maxRetries, kind, ym.String(), typ, lastErr)

		// Update live state with retry info.
		t.ls.Update(func(s *StateSnapshot) {
			s.Phase = PhaseRetrying
			s.Stats.Retries++
			if s.Current != nil {
				s.Current.Phase = PhaseRetrying
			}
		})

		// Exponential backoff: 10s × 2^(attempt-1), capped at 5 min.
		backoff := time.Duration(10<<(attempt-1)) * time.Second
		if backoff > 5*time.Minute {
			backoff = 5 * time.Minute
		}
		logf("waiting %s before retry %d/%d", backoff, attempt+1, maxRetries)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}

		// Always clean up work artifacts (shards, chunks).
		shardDir := cfg.ShardLocalDir(typ, year, mm)
		os.RemoveAll(shardDir)
		if matches, _ := filepath.Glob(filepath.Join(cfg.WorkDir, "chunk_*.jsonl")); matches != nil {
			for _, m := range matches {
				os.Remove(m)
			}
		}

		zstPath := cfg.ZstPath(prefix, ym.String())
		if corruption {
			// Corruption (zero-filled mmap regions): rename .zst back to .part
			// so the torrent client re-verifies piece SHA-1 hashes and only
			// re-downloads the corrupt pieces instead of the entire file.
			partPath := zstPath + ".part"
			if _, statErr := os.Stat(zstPath); statErr == nil {
				os.Rename(zstPath, partPath)
			}
		} else {
			// Transient (timeout/network): keep .part for torrent resume.
			// Only delete the final .zst if it exists (it shouldn't for a
			// download timeout, but might for a processing error).
			os.Remove(zstPath)
		}
	}
	return fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}

func (t *PublishTask) processOne(ctx context.Context, ym ymKey, typ string,
	existingRows []StatsRow, emit func(*PublishState)) error {

	cfg := t.cfg
	prefix := zstPrefix(typ)
	zstPath := cfg.ZstPath(prefix, ym.String())
	year := fmt.Sprintf("%04d", ym.Year)
	mm := fmt.Sprintf("%02d", ym.Month)

	// --- Download ---
	t.ls.Update(func(s *StateSnapshot) {
		s.Phase = PhaseDownloading
		s.Current = &LiveCurrent{YM: ym.String(), Type: typ, Phase: PhaseDownloading}
	})
	if emit != nil {
		emit(&PublishState{Phase: "download", YM: ym.String(), Type: typ})
	}
	var expectedBytes int64
	durDown, err := DownloadZst(ctx, cfg, ym.Year, ym.Month, typ, func(p DownloadProgress) {
		if p.BytesTotal > 0 {
			expectedBytes = p.BytesTotal
		}
		t.ls.Update(func(s *StateSnapshot) {
			if s.Current != nil {
				s.Current.BytesDone = p.BytesDone
				s.Current.BytesTotal = p.BytesTotal
			}
		})
		if emit != nil {
			emit(&PublishState{Phase: "download", YM: ym.String(), Type: typ,
				Bytes: p.BytesDone, BytesTotal: p.BytesTotal, Message: p.Message})
		}
	})
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}

	// Validate the downloaded file before spending time processing it.
	if fi, statErr := os.Stat(zstPath); statErr != nil {
		return &ErrCorruption{Msg: "validate zst: file missing after download"}
	} else if fi.Size() == 0 {
		return &ErrCorruption{Msg: "validate zst: file is empty"}
	} else if expectedBytes > 0 && fi.Size() < expectedBytes {
		return &ErrCorruption{Msg: fmt.Sprintf("validate zst: truncated (%d of %d bytes)", fi.Size(), expectedBytes)}
	}
	if err := QuickValidateZst(zstPath); err != nil {
		return &ErrCorruption{Msg: fmt.Sprintf("validate zst: %v", err)}
	}

	// Skip deep validation — it re-reads the entire file to io.Discard which
	// doesn't scale to 50+ GB archives and risks OOM from the zstd window
	// allocation on memory-constrained servers.  QuickValidateZst catches
	// mmap corruption (zero-filled regions), and ProcessZst will catch any
	// remaining mid-stream corruption during the actual processing step,
	// triggering the retry logic.

	// --- Process ---
	t.ls.Update(func(s *StateSnapshot) {
		s.Phase = PhaseProcessing
		s.Current = &LiveCurrent{YM: ym.String(), Type: typ, Phase: PhaseProcessing}
	})
	if emit != nil {
		emit(&PublishState{Phase: "process", YM: ym.String(), Type: typ})
	}
	t1 := time.Now()
	var procTotalRows int64
	procResult, err := ProcessZst(ctx, cfg, zstPath, typ, year, mm, func(sr ShardResult) {
		if sr.Starting {
			// Shard just started — DuckDB is working; show activity.
			t.ls.Update(func(s *StateSnapshot) {
				if s.Current != nil {
					s.Current.Shard = sr.Index + 1
				}
			})
			if emit != nil {
				emit(&PublishState{Phase: "process_start", YM: ym.String(), Type: typ,
					Shards: sr.Index + 1, Rows: procTotalRows})
			}
			return
		}
		// Shard completed.
		procTotalRows += sr.Rows
		var rowsPerSec float64
		if sr.Duration > 0 {
			rowsPerSec = float64(sr.Rows) / sr.Duration.Seconds()
		}
		t.ls.Update(func(s *StateSnapshot) {
			if s.Current != nil {
				s.Current.Shard = sr.Index + 1
				s.Current.Rows = procTotalRows
			}
		})
		if emit != nil {
			emit(&PublishState{Phase: "process", YM: ym.String(), Type: typ,
				Shards: sr.Index + 1, Rows: procTotalRows, Bytes: sr.SizeBytes,
				RowsPerSec: rowsPerSec})
		}
	})
	if err != nil {
		// Classify processing errors. Corruption / missing file → force re-download.
		// "open zst" / "no such file" = .zst vanished between download and process
		// (e.g. previous successful process deleted it, then upload failed).
		errStr := err.Error()
		if strings.Contains(errStr, "zstd") || strings.Contains(errStr, "scan jsonl") ||
			strings.Contains(errStr, "open zst") || strings.Contains(errStr, "no such file") {
			return &ErrCorruption{Msg: fmt.Sprintf("process: %v", err)}
		}
		return fmt.Errorf("process: %w", err)
	}
	durProc := time.Since(t1)

	// Delete .zst now that stream is exhausted.
	os.Remove(zstPath)

	// --- HF Commit ---
	t.ls.Update(func(s *StateSnapshot) {
		s.Phase = PhaseCommitting
		if s.Current != nil {
			s.Current.Phase = PhaseCommitting
		}
	})
	if emit != nil {
		emit(&PublishState{Phase: "commit", YM: ym.String(), Type: typ,
			Shards: len(procResult.Shards), Rows: procResult.TotalRows, Bytes: procResult.TotalSize})
	}
	t2 := time.Now()

	newRow := StatsRow{
		Year:         ym.Year,
		Month:        ym.Month,
		Type:         typ,
		Shards:       len(procResult.Shards),
		Count:        procResult.TotalRows,
		SizeBytes:    procResult.TotalSize,
		DurDownloadS: durDown.Seconds(),
		DurProcessS:  durProc.Seconds(),
		CommittedAt:  time.Now().UTC(),
	}
	allRows := make([]StatsRow, len(existingRows)+1)
	copy(allRows, existingRows)
	allRows[len(existingRows)] = newRow

	// Write README with live state (committing phase) before committing.
	snap := t.ls.Snapshot()
	readme, err := GenerateREADMEWithLive(allRows, &snap)
	if err != nil {
		return fmt.Errorf("readme: %w", err)
	}
	if err := os.WriteFile(cfg.READMEPath(), readme, 0o644); err != nil {
		return fmt.Errorf("write readme: %w", err)
	}
	if err := WriteStatsCSV(cfg.StatsCSVPath(), allRows); err != nil {
		return fmt.Errorf("write stats: %w", err)
	}
	if err := WriteStateJSON(cfg, snap); err != nil {
		logf("write states.json before commit: %v", err)
	}

	var ops []HFOp
	for _, sr := range procResult.Shards {
		ops = append(ops, HFOp{
			LocalPath:  sr.LocalPath,
			PathInRepo: cfg.ShardHFPath(typ, year, mm, sr.Index),
		})
	}
	ops = append(ops,
		HFOp{LocalPath: cfg.StatsCSVPath(), PathInRepo: "stats.csv"},
		HFOp{LocalPath: cfg.READMEPath(), PathInRepo: "README.md"},
		HFOp{LocalPath: cfg.StatesJSONPath(), PathInRepo: "states.json"},
	)

	// Batch commits ≤50 ops per call; hold commitMu so heartbeat doesn't interleave.
	// Each batch is retried up to 3 times with exponential backoff on failure.
	const batchSize = 50
	const hfMaxRetries = 3
	t.commitMu.Lock()
	for i := 0; i < len(ops); i += batchSize {
		end := i + batchSize
		if end > len(ops) {
			end = len(ops)
		}
		msg := fmt.Sprintf("Add %s/%s (%d shards, %s rows)",
			typ, ym.String(), len(procResult.Shards),
			fmtCount(procResult.TotalRows))

		var commitErr error
		for retry := 0; retry < hfMaxRetries; retry++ {
			if ctx.Err() != nil {
				t.commitMu.Unlock()
				WriteStatsCSV(cfg.StatsCSVPath(), existingRows)
				return ctx.Err()
			}
			_, commitErr = t.opts.HFCommit(ctx, ops[i:end], msg)
			if commitErr == nil {
				break
			}
			if retry < hfMaxRetries-1 {
				wait := time.Duration(5<<retry) * time.Second // 5s, 10s
				logf("hf commit batch %d failed (retry %d/%d in %s): %v",
					i/batchSize, retry+1, hfMaxRetries, wait, commitErr)
				select {
				case <-ctx.Done():
					t.commitMu.Unlock()
					WriteStatsCSV(cfg.StatsCSVPath(), existingRows)
					return ctx.Err()
				case <-time.After(wait):
				}
			}
		}
		if commitErr != nil {
			t.commitMu.Unlock()
			// Revert stats.csv — remove the row we just added so it
			// doesn't falsely mark this month as committed on restart.
			WriteStatsCSV(cfg.StatsCSVPath(), existingRows)
			return fmt.Errorf("hf commit batch %d (after %d retries): %w", i/batchSize, hfMaxRetries, commitErr)
		}
	}
	t.lastHFCommit.Store(time.Now().UnixNano())
	t.commitMu.Unlock()

	durComm := time.Since(t2)
	newRow.DurCommitS = durComm.Seconds()
	allRows[len(allRows)-1] = newRow
	if err := WriteStatsCSV(cfg.StatsCSVPath(), allRows); err != nil {
		logf("warning: update stats.csv with commit duration: %v", err)
	}

	// Delete local shards after successful commit.
	for _, sr := range procResult.Shards {
		os.Remove(sr.LocalPath)
	}
	shardDir := cfg.ShardLocalDir(typ, year, mm)
	os.Remove(shardDir)
	os.Remove(filepath.Dir(shardDir)) // year dir — ignore error if not empty

	// Update live state: pair complete.
	t.ls.Update(func(s *StateSnapshot) {
		s.Phase = PhaseIdle
		s.Current = nil
		s.Stats.Committed++
		s.Stats.TotalRows += procResult.TotalRows
		s.Stats.TotalBytes += procResult.TotalSize
	})

	if emit != nil {
		emit(&PublishState{
			Phase:   "committed",
			YM:      ym.String(),
			Type:    typ,
			Shards:  len(procResult.Shards),
			Rows:    procResult.TotalRows,
			Bytes:   procResult.TotalSize,
			DurDown: durDown,
			DurProc: durProc,
			DurComm: durComm,
		})
	}

	return nil
}

// cleanupWork removes leftover work files from an interrupted previous run.
func (t *PublishTask) cleanupWork() {
	matches, _ := filepath.Glob(filepath.Join(t.cfg.WorkDir, "chunk_*.jsonl"))
	for _, m := range matches {
		if err := os.Remove(m); err != nil {
			logf("cleanup: %v", err)
		}
	}
	for _, typ := range []string{"comments", "submissions"} {
		dir := filepath.Join(t.cfg.WorkDir, typ)
		if err := os.RemoveAll(dir); err != nil {
			logf("cleanup: %v", err)
		}
	}
	// Torrent saves files under RawDir/reddit/{comments,submissions}/.
	// Clean both the final .zst and any in-progress .part files.
	for _, sub := range []string{"comments", "submissions"} {
		dir := filepath.Join(t.cfg.RawDir, "reddit", sub)
		for _, glob := range []string{"R[CS]_*.zst", "R[CS]_*.zst.part"} {
			subMatches, _ := filepath.Glob(filepath.Join(dir, glob))
			for _, m := range subMatches {
				if err := os.Remove(m); err != nil {
					logf("cleanup: %v", err)
				}
			}
		}
	}
}

// ymKey is a (Year, Month) pair.
type ymKey struct {
	Year  int
	Month int
}

func (k ymKey) String() string { return fmt.Sprintf("%04d-%02d", k.Year, k.Month) }

func (t *PublishTask) monthRange() []ymKey {
	from := ymKey{Year: t.opts.FromYear, Month: t.opts.FromMonth}
	to := ymKey{Year: t.opts.ToYear, Month: t.opts.ToMonth}
	if from.Year == 0 {
		from = ymKey{Year: 2005, Month: 12} // earliest file in the bundle torrent
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

func ymAfter(a, b ymKey) bool {
	if a.Year != b.Year {
		return a.Year > b.Year
	}
	return a.Month > b.Month
}
