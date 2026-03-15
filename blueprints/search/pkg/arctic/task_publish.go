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
	Phase      string // "skip"|"download"|"process"|"commit"|"committed"|"disk_check"|"done"
	YM         string // "YYYY-MM"
	Type       string // "comments"|"submissions"
	Shards     int
	Rows       int64
	Bytes      int64
	BytesTotal int64 // used during download phase
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

// runHeartbeat runs in a goroutine: writes files every minute, commits every 5 minutes.
func (t *PublishTask) runHeartbeat(ctx context.Context) {
	writeTick := time.NewTicker(1 * time.Minute)
	commitTick := time.NewTicker(5 * time.Minute)
	defer writeTick.Stop()
	defer commitTick.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-writeTick.C:
			t.writeHeartbeatFiles()
		case <-commitTick.C:
			// Skip if a data commit just happened within the last 5 minutes.
			if time.Since(time.Unix(0, t.lastHFCommit.Load())) < 5*time.Minute {
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
		fmt.Fprintf(os.Stderr, "arctic: heartbeat: write states.json: %v\n", err)
	}

	rows, _ := ReadStatsCSV(t.cfg.StatsCSVPath())
	readme, err := GenerateREADMEWithLive(rows, &snap)
	if err != nil {
		fmt.Fprintf(os.Stderr, "arctic: heartbeat: generate readme: %v\n", err)
		return
	}
	if err := os.WriteFile(t.cfg.READMEPath(), readme, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "arctic: heartbeat: write readme: %v\n", err)
	}
}

// commitHeartbeat commits states.json + README.md to HuggingFace.
// force=true skips the cooldown check (used for the final done commit).
func (t *PublishTask) commitHeartbeat(ctx context.Context, force bool) {
	if !force {
		if time.Since(time.Unix(0, t.lastHFCommit.Load())) < 5*time.Minute {
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
		fmt.Fprintf(os.Stderr, "arctic: heartbeat commit: %v\n", err)
		return
	}
	t.lastHFCommit.Store(time.Now().UnixNano())
}

const maxRetries = 3

// processOneWithRetry wraps processOne with auto-heal: on failure it cleans up
// the corrupted/partial files for this (month, type) and retries up to maxRetries
// times. Context cancellation is not retried.
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
		fmt.Fprintf(os.Stderr, "arctic: attempt %d/%d failed for [%s] %s: %v — cleaning up and retrying\n",
			attempt, maxRetries, ym.String(), typ, lastErr)

		// Clean up the bad .zst and any partial shards so the next attempt starts fresh.
		zstPath := cfg.ZstPath(prefix, ym.String())
		os.Remove(zstPath)
		os.Remove(zstPath + ".part")
		shardDir := cfg.ShardLocalDir(typ, year, mm)
		os.RemoveAll(shardDir)
		// Remove chunk files.
		if matches, _ := filepath.Glob(filepath.Join(cfg.WorkDir, "chunk_*.jsonl")); matches != nil {
			for _, m := range matches {
				os.Remove(m)
			}
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
	durDown, err := DownloadZst(ctx, cfg, ym.Year, ym.Month, typ, func(p DownloadProgress) {
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

	// --- Process ---
	t.ls.Update(func(s *StateSnapshot) {
		s.Phase = PhaseProcessing
		s.Current = &LiveCurrent{YM: ym.String(), Type: typ, Phase: PhaseProcessing}
	})
	if emit != nil {
		emit(&PublishState{Phase: "process", YM: ym.String(), Type: typ})
	}
	t1 := time.Now()
	procResult, err := ProcessZst(ctx, cfg, zstPath, typ, year, mm, func(sr ShardResult) {
		t.ls.Update(func(s *StateSnapshot) {
			if s.Current != nil {
				s.Current.Shard = sr.Index + 1
				s.Current.Rows = sr.Rows
			}
		})
		if emit != nil {
			emit(&PublishState{Phase: "process", YM: ym.String(), Type: typ,
				Shards: sr.Index + 1, Rows: sr.Rows, Bytes: sr.SizeBytes})
		}
	})
	if err != nil {
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
		fmt.Fprintf(os.Stderr, "arctic: write states.json before commit: %v\n", err)
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
	const batchSize = 50
	t.commitMu.Lock()
	for i := 0; i < len(ops); i += batchSize {
		end := i + batchSize
		if end > len(ops) {
			end = len(ops)
		}
		msg := fmt.Sprintf("Add %s/%s (%d shards, %s rows)",
			typ, ym.String(), len(procResult.Shards),
			fmtCount(procResult.TotalRows))
		if _, err := t.opts.HFCommit(ctx, ops[i:end], msg); err != nil {
			t.commitMu.Unlock()
			return fmt.Errorf("hf commit batch %d: %w", i/batchSize, err)
		}
	}
	t.lastHFCommit.Store(time.Now().UnixNano())
	t.commitMu.Unlock()

	durComm := time.Since(t2)
	newRow.DurCommitS = durComm.Seconds()
	allRows[len(allRows)-1] = newRow
	if err := WriteStatsCSV(cfg.StatsCSVPath(), allRows); err != nil {
		fmt.Fprintf(os.Stderr, "arctic: warning: update stats.csv with commit duration: %v\n", err)
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
			fmt.Fprintf(os.Stderr, "arctic: cleanup: %v\n", err)
		}
	}
	for _, typ := range []string{"comments", "submissions"} {
		dir := filepath.Join(t.cfg.WorkDir, typ)
		if err := os.RemoveAll(dir); err != nil {
			fmt.Fprintf(os.Stderr, "arctic: cleanup: %v\n", err)
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
					fmt.Fprintf(os.Stderr, "arctic: cleanup: %v\n", err)
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
