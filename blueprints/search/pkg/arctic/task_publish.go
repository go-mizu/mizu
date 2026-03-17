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

// ErrCommitStall is returned when the pipeline has not made a successful HF
// data commit within Config.MaxCommitStall. The process should be restarted
// (it will resume from the last committed month via stats.csv).
var ErrCommitStall = fmt.Errorf("arctic: commit stall — no HF commit within max-commit-stall window")

const maxRetries = 5

// --- Types ---

// PublishOptions configures the publish run.
type PublishOptions struct {
	FromYear  int
	FromMonth int
	ToYear    int
	ToMonth   int
	HFCommit  CommitFn
	// Types filters which data types to process. If empty, both
	// "comments" and "submissions" are processed (default behaviour).
	Types []string
}

// PublishState is emitted on each significant event.
type PublishState struct {
	Phase      string // "skip"|"download"|"validate"|"process_start"|"process"|"commit"|"committed"|"disk_check"|"done"
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

// PipelineJob represents a single (month, type) pair flowing through the pipeline.
type PipelineJob struct {
	YM         ymKey
	Type       string // "comments" | "submissions"
	ZstPath    string // populated after download
	WorkDir    string // per-job isolated work directory
	ProcResult ProcessResult
	DurDown    time.Duration
	DurProc    time.Duration
	DurComm    time.Duration
	Attempt    int // retry counter
}

// ymKey is a (Year, Month) pair.
type ymKey struct {
	Year  int
	Month int
}

func (k ymKey) String() string { return fmt.Sprintf("%04d-%02d", k.Year, k.Month) }

func ymAfter(a, b ymKey) bool {
	if a.Year != b.Year {
		return a.Year > b.Year
	}
	return a.Month > b.Month
}

// --- PipelineTask ---

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
	downloadSpeeds []float64
	processSpeeds  []float64
	commitSpeeds   []float64

	// disk space coordination
	diskMu   sync.Mutex
	diskCond *sync.Cond

	// zst size catalog
	zstSizes ZstSizes

	// stall detection
	commitStalled atomic.Bool
	pipeCancelMu  sync.Mutex
	pipeCancelFn  context.CancelFunc
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

// Run executes the pipelined publish.
func (t *PipelineTask) Run(ctx context.Context, emit func(*PublishState)) (PublishMetric, error) {
	return t.runPipeline(ctx, emit)
}

// --- Pipeline orchestration ---

func (t *PipelineTask) runPipeline(ctx context.Context, emit func(*PublishState)) (PublishMetric, error) {
	start := time.Now()
	metric := PublishMetric{}

	cleanupWork(t.cfg)

	rows, err := ReadStatsCSV(t.cfg.StatsCSVPath())
	if err != nil {
		return metric, fmt.Errorf("read stats: %w", err)
	}
	committed := CommittedSet(rows)

	// Build job list — skip already committed.
	months := monthRange(t.opts)
	var jobs []*PipelineJob
	skipped := 0
	activeTypes := t.opts.Types
	if len(activeTypes) == 0 {
		activeTypes = []string{"comments", "submissions"}
	}
	for _, ym := range months {
		for _, typ := range activeTypes {
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

	writeHeartbeatFiles(t.cfg, t.ls, t.zstSizes)

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

	// Error collection — only for truly fatal errors (upload failures).
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

	pipeCtx, pipeCancel := context.WithCancel(ctx)
	defer pipeCancel()
	t.pipeCancelMu.Lock()
	t.pipeCancelFn = pipeCancel
	t.pipeCancelMu.Unlock()

	var wgDown, wgProc, wgUpload sync.WaitGroup

	// --- Download workers ---
	wgDown.Add(t.budget.MaxDownloads)
	for i := 0; i < t.budget.MaxDownloads; i++ {
		go func() {
			defer wgDown.Done()
			for job := range downloadCh {
				if pipeCtx.Err() != nil {
					continue
				}

				t.updatePipelineSlot("downloading", job.YM.String(), job.Type, func(slot *PipelineSlot) {
					slot.Phase = PhaseDownloading
				})

				progressFn := func(ps *PublishState) {
					t.updatePipelineSlot("downloading", job.YM.String(), job.Type, func(slot *PipelineSlot) {
						slot.BytesDone = ps.Bytes
						slot.BytesTotal = ps.BytesTotal
					})
					if emit != nil {
						emit(ps)
					}
				}

				err := downloadWithRetry(pipeCtx, t.cfg, t.zstSizes, job, progressFn)
				t.removePipelineSlot("downloading", job.YM.String(), job.Type)

				if err != nil {
					if pipeCtx.Err() != nil {
						continue
					}
					// NON-FATAL: skip this month, log, continue pipeline.
					logf("pipeline: SKIP [%s] %s — download failed: %v", job.YM.String(), job.Type, err)
					t.ls.Update(func(s *StateSnapshot) { s.Stats.Skipped++ })
					continue
				}

				// Record download speed.
				if mbps := downloadSpeed(job.ZstPath, job.DurDown); mbps > 0 {
					t.recordDownloadSpeed(mbps)
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

				t.updatePipelineSlot("processing", job.YM.String(), job.Type, func(slot *PipelineSlot) {
					slot.Phase = PhaseProcessing
				})

				var procTotalRows int64
				shardFn := func(sr ShardResult) {
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
				}

				// Re-download function for corruption recovery.
				redownload := func() error {
					return downloadOne(pipeCtx, t.cfg, t.zstSizes, job, 0, func(ps *PublishState) {
						if emit != nil {
							emit(ps)
						}
					})
				}

				err := processWithRetry(pipeCtx, t.cfg, t.budget, t.zstSizes, job, shardFn, redownload)
				t.removePipelineSlot("processing", job.YM.String(), job.Type)

				if err != nil {
					if pipeCtx.Err() != nil {
						continue
					}
					// NON-FATAL: skip this month, log, continue pipeline.
					logf("pipeline: SKIP [%s] %s — process failed: %v", job.YM.String(), job.Type, err)
					t.ls.Update(func(s *StateSnapshot) { s.Stats.Skipped++ })
					continue
				}

				// Record process speed.
				if rps := processSpeed(job.ProcResult.TotalRows, job.DurProc); rps > 0 {
					t.recordProcessSpeed(rps)
				}

				uploadCh <- job
			}
		}()
	}

	// --- Upload worker (always 1) ---
	var committedCount int64
	wgUpload.Add(1)
	go func() {
		defer wgUpload.Done()
		for job := range uploadCh {
			if pipeCtx.Err() != nil {
				continue
			}

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

			// Upload errors are fatal — they indicate HF API problems that
			// affect all months, so retrying other months won't help.
			var lastErr error
			for attempt := 0; attempt < maxRetries; attempt++ {
				if pipeCtx.Err() != nil {
					break
				}
				lastErr = commitToHF(pipeCtx, t.cfg, t.opts.HFCommit, job,
					t.ls, t.zstSizes, &t.commitMu, &t.lastHFCommit)
				if lastErr == nil {
					break
				}
				if pipeCtx.Err() != nil {
					break
				}
				backoff := time.Duration(30<<attempt) * time.Second
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

			t.removePipelineSlot("uploading", job.YM.String(), job.Type)

			if lastErr != nil {
				if pipeCtx.Err() == nil {
					setPipeErr(fmt.Errorf("[%s] %s: upload (after %d retries): %w",
						job.YM.String(), job.Type, maxRetries, lastErr))
					pipeCancel()
				}
				continue
			}

			cleanupAfterCommit(job)
			atomic.AddInt64(&committedCount, 1)
			t.recordCommitSpeed(job.DurComm.Seconds())

			// Update live state.
			t.ls.Update(func(s *StateSnapshot) {
				s.Stats.Committed++
				s.Stats.TotalRows += job.ProcResult.TotalRows
				s.Stats.TotalBytes += job.ProcResult.TotalSize
			})

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
					DurComm: job.DurComm,
				})
			}

			// Signal disk space may have freed up.
			t.diskCond.Broadcast()
		}
	}()

	// Feed jobs into download stage.
	go func() {
		for _, job := range jobs {
			waitForDisk(pipeCtx, t.cfg, t.diskCond)
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

	metric.Committed = int(atomic.LoadInt64(&committedCount))
	metric.Skipped = skipped

	if t.commitStalled.Load() {
		return metric, ErrCommitStall
	}
	if pipeErr != nil {
		return metric, pipeErr
	}

	t.finalize(ctx, emit)
	metric.Elapsed = time.Since(start)
	return metric, nil
}

// --- Heartbeat ---

func (t *PipelineTask) runHeartbeat(ctx context.Context) {
	writeTick := time.NewTicker(1 * time.Minute)
	commitTick := time.NewTicker(10 * time.Minute)
	stallTick := time.NewTicker(2 * time.Minute)
	defer writeTick.Stop()
	defer commitTick.Stop()
	defer stallTick.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-writeTick.C:
			writeHeartbeatFiles(t.cfg, t.ls, t.zstSizes)
		case <-commitTick.C:
			if time.Since(time.Unix(0, t.lastHFCommit.Load())) < 10*time.Minute {
				continue
			}
			writeHeartbeatFiles(t.cfg, t.ls, t.zstSizes)
			commitHeartbeatToHF(ctx, t.cfg, t.opts.HFCommit, t.ls, &t.commitMu, &t.lastHFCommit, false)
		case <-stallTick.C:
			t.checkCommitStall()
		}
	}
}

func (t *PipelineTask) checkCommitStall() {
	maxStall := t.cfg.MaxCommitStall
	if maxStall <= 0 {
		return
	}
	idle := time.Since(time.Unix(0, t.lastHFCommit.Load()))
	if idle <= maxStall {
		return
	}
	logf("pipeline: STALL DETECTED — no HF data commit for %s (limit: %s). Triggering restart.",
		idle.Round(time.Second), maxStall.Round(time.Second))
	t.commitStalled.Store(true)
	t.pipeCancelMu.Lock()
	if t.pipeCancelFn != nil {
		t.pipeCancelFn()
	}
	t.pipeCancelMu.Unlock()
}

func (t *PipelineTask) finalize(ctx context.Context, emit func(*PublishState)) {
	t.ls.Update(func(s *StateSnapshot) {
		s.Phase = PhaseDone
		s.Current = nil
		s.Pipeline = nil
	})
	writeHeartbeatFiles(t.cfg, t.ls, t.zstSizes)
	commitHeartbeatToHF(ctx, t.cfg, t.opts.HFCommit, t.ls, &t.commitMu, &t.lastHFCommit, true)
	if emit != nil {
		emit(&PublishState{Phase: "done"})
	}
}

// --- Pipeline slot management ---

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

		for i := range *slots {
			if (*slots)[i].YM == ym && (*slots)[i].Type == typ {
				fn(&(*slots)[i])
				return
			}
		}
		slot := PipelineSlot{YM: ym, Type: typ}
		fn(&slot)
		*slots = append(*slots, slot)

		updateCurrent(s)
	})
}

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

		updateCurrent(s)
	})
}

// updateCurrent sets the Current field to the "most active" pipeline slot.
func updateCurrent(s *StateSnapshot) {
	if s.Pipeline == nil {
		s.Current = nil
		return
	}
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

// --- Throughput tracking ---

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
	t.ls.Update(func(s *StateSnapshot) {
		if s.Throughput == nil {
			s.Throughput = &ThroughputStats{}
		}
		s.Throughput.AvgDownloadMbps = avg(t.downloadSpeeds)
		s.Throughput.AvgProcessRowsPerSec = avg(t.processSpeeds)
		s.Throughput.AvgUploadSecPerCommit = avg(t.commitSpeeds)

		remaining := s.Stats.TotalMonths - s.Stats.Committed - s.Stats.Skipped
		if remaining > 0 && s.Stats.Committed > 0 {
			elapsed := time.Since(s.StartedAt)
			perPair := elapsed / time.Duration(s.Stats.Committed)
			eta := time.Now().Add(perPair * time.Duration(remaining))
			s.Throughput.EstimatedCompletion = &eta
		}
	})
}

// --- Helpers ---

func monthRange(opts PublishOptions) []ymKey {
	from := ymKey{Year: opts.FromYear, Month: opts.FromMonth}
	to := ymKey{Year: opts.ToYear, Month: opts.ToMonth}
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

// cleanupWork removes leftover work files from an interrupted previous run.
func cleanupWork(cfg Config) {
	// Clean pipeline job directories.
	matches, _ := filepath.Glob(filepath.Join(cfg.WorkDir, "pipeline_*"))
	for _, m := range matches {
		os.RemoveAll(m)
	}
	// Clean legacy chunk files.
	chunks, _ := filepath.Glob(filepath.Join(cfg.WorkDir, "chunk_*.jsonl"))
	for _, m := range chunks {
		os.Remove(m)
	}
	for _, typ := range []string{"comments", "submissions"} {
		dir := filepath.Join(cfg.WorkDir, typ)
		os.RemoveAll(dir)
	}
	// Clean incomplete .part downloads but keep completed .zst files.
	for _, sub := range []string{"comments", "submissions"} {
		dir := filepath.Join(cfg.RawDir, "reddit", sub)
		partMatches, _ := filepath.Glob(filepath.Join(dir, "R[CS]_*.zst.part"))
		for _, m := range partMatches {
			os.Remove(m)
		}
	}
}

// waitForDisk blocks until disk space is above MinFreeGB.
func waitForDisk(ctx context.Context, cfg Config, diskCond *sync.Cond) {
	diskCond.L.Lock()
	defer diskCond.L.Unlock()

	for {
		if ctx.Err() != nil {
			return
		}
		free, err := cfg.FreeDiskGB()
		if err != nil || free >= float64(cfg.MinFreeGB) {
			return
		}
		logf("pipeline: %.1f GB free (need %d GB) — waiting for uploads to free space",
			free, cfg.MinFreeGB)

		done := make(chan struct{})
		go func() {
			diskCond.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-ctx.Done():
			diskCond.Broadcast()
			return
		case <-time.After(30 * time.Second):
		}
	}
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
