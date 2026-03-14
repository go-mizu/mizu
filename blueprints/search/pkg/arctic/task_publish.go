package arctic

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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
	Phase   string // "skip"|"download"|"process"|"commit"|"committed"|"disk_check"|"done"
	YM      string // "YYYY-MM"
	Type    string // "comments"|"submissions"
	Shards  int
	Rows    int64
	Bytes   int64
	DurDown time.Duration
	DurProc time.Duration
	DurComm time.Duration
	Message string
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

	for _, ym := range months {
		for _, typ := range []string{"comments", "submissions"} {
			if err := ctx.Err(); err != nil {
				return metric, err
			}

			key := ym.Key() + "/" + typ

			if committed[key] {
				metric.Skipped++
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

			if err := t.processOne(ctx, ym, typ, rows, emit); err != nil {
				return metric, fmt.Errorf("[%s] %s: %w", ym.String(), typ, err)
			}

			rows, _ = ReadStatsCSV(t.cfg.StatsCSVPath())
			committed = CommittedSet(rows)
			metric.Committed++
		}
	}

	metric.Elapsed = time.Since(start)
	if emit != nil {
		emit(&PublishState{Phase: "done"})
	}
	return metric, nil
}

func (t *PublishTask) processOne(ctx context.Context, ym ymKey, typ string,
	existingRows []StatsRow, emit func(*PublishState)) error {

	cfg := t.cfg
	prefix := zstPrefix(typ)
	zstPath := cfg.ZstPath(prefix, ym.String())
	year := fmt.Sprintf("%04d", ym.Year)
	mm := fmt.Sprintf("%02d", ym.Month)

	// --- Download ---
	if emit != nil {
		emit(&PublishState{Phase: "download", YM: ym.String(), Type: typ})
	}
	durDown, err := DownloadZst(ctx, cfg, ym.Year, ym.Month, typ, func(p DownloadProgress) {
		if emit != nil {
			emit(&PublishState{Phase: "download", YM: ym.String(), Type: typ,
				Bytes: p.BytesDone, Message: p.Phase})
		}
	})
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}

	// --- Process ---
	if emit != nil {
		emit(&PublishState{Phase: "process", YM: ym.String(), Type: typ})
	}
	t1 := time.Now()
	procResult, err := ProcessZst(ctx, cfg, zstPath, typ, year, mm, func(sr ShardResult) {
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
	allRows := append(existingRows, newRow)

	readme, err := GenerateREADME(allRows)
	if err != nil {
		return fmt.Errorf("readme: %w", err)
	}
	if err := os.WriteFile(cfg.READMEPath(), readme, 0o644); err != nil {
		return fmt.Errorf("write readme: %w", err)
	}
	if err := WriteStatsCSV(cfg.StatsCSVPath(), allRows); err != nil {
		return fmt.Errorf("write stats: %w", err)
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
	)

	// Batch commits ≤50 ops per call.
	const batchSize = 50
	for i := 0; i < len(ops); i += batchSize {
		end := i + batchSize
		if end > len(ops) {
			end = len(ops)
		}
		msg := fmt.Sprintf("add %s/%s %s/%s (%d shards, %s rows)",
			typ, ym.String(), year, mm, len(procResult.Shards),
			fmtCount(procResult.TotalRows))
		if _, err := t.opts.HFCommit(ctx, ops[i:end], msg); err != nil {
			return fmt.Errorf("hf commit batch %d: %w", i/batchSize, err)
		}
	}

	durComm := time.Since(t2)
	newRow.DurCommitS = durComm.Seconds()
	allRows[len(allRows)-1] = newRow
	_ = WriteStatsCSV(cfg.StatsCSVPath(), allRows)

	// Delete local shards after successful commit.
	for _, sr := range procResult.Shards {
		os.Remove(sr.LocalPath)
	}
	shardDir := cfg.ShardLocalDir(typ, year, mm)
	os.Remove(shardDir)
	os.Remove(filepath.Dir(shardDir)) // year dir — ignore error if not empty

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
		os.Remove(m)
	}
	for _, typ := range []string{"comments", "submissions"} {
		dir := filepath.Join(t.cfg.WorkDir, typ)
		os.RemoveAll(dir)
	}
	matches, _ = filepath.Glob(filepath.Join(t.cfg.RawDir, "R[CS]_*.zst"))
	for _, m := range matches {
		os.Remove(m)
	}
}

// ymKey is a (Year, Month) pair.
type ymKey struct {
	Year  int
	Month int
}

func (k ymKey) String() string { return fmt.Sprintf("%04d-%02d", k.Year, k.Month) }
func (k ymKey) Key() string    { return k.String() }

func (t *PublishTask) monthRange() []ymKey {
	from := ymKey{Year: t.opts.FromYear, Month: t.opts.FromMonth}
	to := ymKey{Year: t.opts.ToYear, Month: t.opts.ToMonth}
	if from.Year == 0 {
		from = ymKey{Year: 2005, Month: 6}
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
