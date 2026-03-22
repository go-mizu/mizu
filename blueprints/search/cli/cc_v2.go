package cli

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/cc_v2"
	warcmd "github.com/go-mizu/mizu/blueprints/search/pkg/warc_md"
	"github.com/spf13/cobra"
)

// NewCCV2 creates the cc_v2 command for the rewritten CC pipeline.
func NewCCV2() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cc_v2",
		Short: "Common Crawl pipeline v2 (simplified, race-free)",
		Long: `Rewritten CC pipeline with cleaner architecture:

  Pipeline:  Download WARC → pack → parquet (sequential, no prefetch races)
  Watcher:   Poll parquet dir → batch → commit to HuggingFace
  Scheduler: Manage pipeline screen sessions (auto-heal, adaptive scaling)

Key improvements over v1:
  - No WARC deletion races (only watcher deletes, after HF commit)
  - Distributed locking via Redis SETNX (prevents double downloads)
  - Single source of truth (Redis, with file-based fallback)
  - Simpler state machine: claimed → ready → committed

Subcommands:
  publish   Run pipeline/watcher/scheduler`,
	}

	cmd.AddCommand(newCCV2Publish())
	return cmd
}

func newCCV2Publish() *cobra.Command {
	var (
		crawlID        string
		fileIdx        string
		repoID         string
		pipeline       bool
		watch          bool
		schedule       bool
		list           bool
		gaps           bool
		skipErrors     bool
		private        bool
		commitInterval int
		schedStart     int
		schedEnd       int
		schedMaxSess   int
		schedChunk     int
	)

	cmd := &cobra.Command{
		Use:   "publish",
		Short: "Run CC v2 pipeline/watcher/scheduler",
		Long: `CC v2 publish modes:

  --pipeline   Download → pack → parquet (no HF push; use --watch for that)
  --watch      Poll parquet dir, push to HF in real-time
  --schedule   Manage pipeline screen sessions across a shard range
  --list       Show committed shards
  --gaps       Show/backfill uncommitted shards`,
		Example: `  # Full auto:
  search cc_v2 publish --watch --commit-interval 90 &
  search cc_v2 publish --schedule --start 0 --end 9999

  # Pipeline only:
  search cc_v2 publish --pipeline --file 0-49 --skip-errors

  # Check progress:
  search cc_v2 publish --list
  search cc_v2 publish --gaps --start 0 --end 99`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCCV2Publish(cmd.Context(), crawlID, fileIdx, repoID,
				pipeline, watch, schedule, list, gaps, skipErrors, private,
				commitInterval, schedStart, schedEnd, schedMaxSess, schedChunk)
		},
	}

	cmd.Flags().StringVar(&crawlID, "crawl", "", "Crawl ID (default: latest)")
	cmd.Flags().StringVar(&fileIdx, "file", "all", "File index, range, or comma list")
	cmd.Flags().StringVar(&repoID, "repo", "open-index/open-markdown", "HuggingFace dataset repo ID")
	cmd.Flags().BoolVar(&pipeline, "pipeline", false, "Download + pack + parquet")
	cmd.Flags().BoolVar(&watch, "watch", false, "Watch parquet dir → HF")
	cmd.Flags().BoolVar(&schedule, "schedule", false, "Manage pipeline screen sessions")
	cmd.Flags().BoolVar(&list, "list", false, "List committed shards")
	cmd.Flags().BoolVar(&gaps, "gaps", false, "Show/backfill gap shards")
	cmd.Flags().BoolVar(&skipErrors, "skip-errors", false, "Skip failed shards")
	cmd.Flags().BoolVar(&private, "private", false, "Create HF repo as private")
	cmd.Flags().IntVar(&commitInterval, "commit-interval", 30, "Min seconds between HF commits")
	cmd.Flags().IntVar(&schedStart, "start", 0, "First shard index")
	cmd.Flags().IntVar(&schedEnd, "end", 9999, "Last shard index")
	cmd.Flags().IntVar(&schedMaxSess, "max-sessions", 0, "Max concurrent sessions (0=auto)")
	cmd.Flags().IntVar(&schedChunk, "chunk-size", 50, "Shards per screen session")

	return cmd
}

func runCCV2Publish(ctx context.Context, crawlID, fileIdx, repoID string,
	pipeline, watch, schedule, list, gaps, skipErrors, private bool,
	commitInterval, schedStart, schedEnd, schedMaxSess, schedChunk int) error {

	// Resolve crawl ID.
	resolved, note, err := ccResolveCrawlID(ctx, crawlID)
	if err != nil {
		return fmt.Errorf("resolving crawl: %w", err)
	}
	crawlID = resolved
	if note != "" {
		fmt.Printf("  Using crawl: %s (%s)\n", crawlID, note)
	}

	dataDir := cc_v2.DefaultDataDir(crawlID)
	repoRoot := cc_v2.DefaultRepoRoot(crawlID)
	store := cc_v2.NewStore(dataDir, crawlID)
	defer store.Close()

	baseCfg := cc_v2.Config{
		CrawlID:  crawlID,
		RepoID:   repoID,
		DataDir:  dataDir,
		RepoRoot: repoRoot,
		Private:  private,
	}

	// ── List mode ────────────────────────────────────────────────────────
	if list {
		cc_v2.ListCommitted(ctx, store, crawlID)
		return nil
	}

	// ── Gap mode ─────────────────────────────────────────────────────────
	if gaps {
		gapIndices := cc_v2.ComputeGapIndices(ctx, store, schedStart, schedEnd)

		if schedule {
			if len(gapIndices) == 0 {
				fmt.Printf("  No gaps in %d–%d\n", schedStart, schedEnd)
				return nil
			}
			cfg := cc_v2.SchedulerConfig{
				Config:      baseCfg,
				Start:       schedStart,
				End:         schedEnd,
				MaxSessions: schedMaxSess,
				ChunkSize:   schedChunk,
				GapIndices:  gapIndices,
			}
			sched := cc_v2.NewScheduler(cfg, store)
			return sched.Run(ctx)
		}

		if pipeline {
			if len(gapIndices) == 0 {
				fmt.Printf("  No gaps in %d–%d\n", schedStart, schedEnd)
				return nil
			}
			cfg := cc_v2.PipelineConfig{
				Config:     baseCfg,
				SkipErrors: skipErrors,
				Indices:    gapIndices,
			}
			p := cc_v2.NewPipeline(cfg, store, ccV2PackFn)
			return p.Run(ctx)
		}

		cc_v2.PrintGaps(crawlID, schedStart, schedEnd, gapIndices)
		return nil
	}

	// ── Watch mode ───────────────────────────────────────────────────────
	if watch {
		token := strings.TrimSpace(os.Getenv("HF_TOKEN"))
		if token == "" {
			return fmt.Errorf("HF_TOKEN is not set")
		}
		cfg := cc_v2.WatcherConfig{
			Config:         baseCfg,
			PollInterval:   10 * time.Second,
			CommitInterval: time.Duration(commitInterval) * time.Second,
			MaxBatch:       30,
			ChartsEvery:    60 * time.Minute,
		}

		// Start pprof server for live profiling.
		go func() {
			fmt.Fprintf(os.Stderr, "  pprof        http://localhost:6060/debug/pprof/\n")
			http.ListenAndServe("localhost:6060", nil)
		}()

		hf := newCCV2HFCommitter(token)
		w := cc_v2.NewWatcher(cfg, store, hf)
		return w.Run(ctx)
	}

	// ── Schedule mode ────────────────────────────────────────────────────
	if schedule {
		cfg := cc_v2.SchedulerConfig{
			Config:      baseCfg,
			Start:       schedStart,
			End:         schedEnd,
			MaxSessions: schedMaxSess,
			ChunkSize:   schedChunk,
		}
		sched := cc_v2.NewScheduler(cfg, store)
		return sched.Run(ctx)
	}

	// ── Pipeline mode ────────────────────────────────────────────────────
	if pipeline {
		// Start pprof server for live profiling (different port per pipeline).
		pprofPort := 6061 + (schedStart % 100)
		go func() {
			http.ListenAndServe(fmt.Sprintf("localhost:%d", pprofPort), nil)
		}()
		indices, err := cc_v2.ParseFileSelector(fileIdx)
		if err != nil {
			return fmt.Errorf("--file: %w", err)
		}
		if len(indices) == 0 {
			return fmt.Errorf("no file indices specified")
		}
		cfg := cc_v2.PipelineConfig{
			Config:     baseCfg,
			SkipErrors: skipErrors,
			Indices:    indices,
		}
		p := cc_v2.NewPipeline(cfg, store, ccV2PackFn)
		return p.Run(ctx)
	}

	return fmt.Errorf("specify --pipeline, --watch, --schedule, --list, or --gaps")
}

// ccV2PackFn wraps the existing packDirectToParquet for use by cc_v2.Pipeline.
func ccV2PackFn(ctx context.Context, cfg warcmd.PackConfig, parquetPath string,
	progressFn warcmd.ProgressFunc) (int64, int64, int64, *warcmd.PackStats, error) {
	return packDirectToParquet(ctx, cfg, parquetPath, progressFn)
}

// ccV2HFCommitter adapts the existing hfClient to the cc_v2.HFCommitter interface.
type ccV2HFCommitter struct {
	hf *hfClient
}

func newCCV2HFCommitter(token string) *ccV2HFCommitter {
	return &ccV2HFCommitter{hf: newHFClient(token)}
}

func (c *ccV2HFCommitter) CreateRepo(ctx context.Context, repoID string, private bool) error {
	return c.hf.createDatasetRepo(ctx, repoID, private)
}

func (c *ccV2HFCommitter) Commit(ctx context.Context, repoID, branch, message string, ops []cc_v2.HFOp) (string, error) {
	hfOps := make([]hfOperation, len(ops))
	for i, op := range ops {
		hfOps[i] = hfOperation{LocalPath: op.LocalPath, PathInRepo: op.PathInRepo}
	}
	return c.hf.createCommit(ctx, repoID, branch, message, hfOps)
}

func (c *ccV2HFCommitter) DownloadFile(ctx context.Context, repoID, path string) ([]byte, error) {
	url := fmt.Sprintf("%s/api/datasets/%s/resolve/main/%s", hfHubURL, repoID, path)
	resp, err := c.hf.req(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d for %s", resp.StatusCode, path)
	}
	data := make([]byte, 0, 1024*1024)
	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}
		if err != nil {
			break
		}
	}
	return data, nil
}
