package cli

import (
	_ "embed"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/hn2"
	"github.com/spf13/cobra"
)

//go:embed embed/hn_readme.md.tmpl
var hnReadmeTmpl []byte

func newHNPublish() *cobra.Command {
	var (
		repoRoot string
		repoID   string
		live     bool
		interval time.Duration
		fromStr  string
		private  bool
	)

	cmd := &cobra.Command{
		Use:   "publish",
		Short: "Publish Hacker News dataset to Hugging Face",
		Long: `Publish the full Hacker News dataset to a Hugging Face dataset repo.

Historical mode (default): backfills all missing months from the first HN post
to last month. Already-committed months (tracked in stats.csv) are skipped —
safe to resume after interruption.

Live mode (--live): after historical backfill, polls every 5 minutes for new
items and commits them as today/YYYY-MM-DD_HH_MM.parquet blocks. At midnight,
today's blocks are merged into the monthly parquet and committed atomically.

Run both as separate screen sessions:
  screen -S hn-history    search hn publish
  screen -S hn-live       search hn publish --live`,
		Example: `  search hn publish
  search hn publish --live
  search hn publish --live --interval 5m
  search hn publish --from 2024-01`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHNPublish(cmd.Context(), repoRoot, repoID, live, interval, fromStr, private)
		},
	}

	cmd.Flags().StringVar(&repoRoot, "repo-root", "", "Local root directory (default: $HOME/data/hn/repo)")
	cmd.Flags().StringVar(&repoID, "repo", "open-index/hacker-news", "Hugging Face dataset repo ID")
	cmd.Flags().BoolVar(&live, "live", false, "Enable continuous 5-min live polling after backfill")
	cmd.Flags().DurationVar(&interval, "interval", 5*time.Minute, "Live poll interval (minimum 1m)")
	cmd.Flags().StringVar(&fromStr, "from", "", "Start month YYYY-MM (skip older months in historical backfill)")
	cmd.Flags().BoolVar(&private, "private", false, "Create HF repo as private if it does not exist")
	return cmd
}

func runHNPublish(ctx context.Context, repoRoot, repoID string, live bool, interval time.Duration, fromStr string, private bool) error {
	token := strings.TrimSpace(os.Getenv("HF_TOKEN"))
	if token == "" {
		return fmt.Errorf("HF_TOKEN environment variable is not set")
	}
	if interval < time.Minute {
		interval = time.Minute
	}

	cfg := hn2.Config{RepoRoot: repoRoot}
	cfg = cfg.WithDefaults()

	if err := cfg.EnsureDirs(); err != nil {
		return fmt.Errorf("ensure dirs: %w", err)
	}

	hf := newHFClient(token)
	if err := hf.createDatasetRepo(ctx, repoID, private); err != nil {
		fmt.Printf("  note: create repo: %v\n", err)
	}

	// hfCommitFn bridges pkg/hn2.HFOp to cli.hfOperation.
	hfCommitFn := func(ctx context.Context, ops []hn2.HFOp, message string) (string, error) {
		var hfOps []hfOperation
		for _, op := range ops {
			hfOps = append(hfOps, hfOperation{
				LocalPath:  op.LocalPath,
				PathInRepo: op.PathInRepo,
				Delete:     op.Delete,
			})
		}
		return hf.createCommit(ctx, repoID, "main", message, hfOps)
	}

	// hfVerifyFn batch-checks all given paths against HF via the paths-info API.
	// Called once at startup to detect stats.csv/HF mismatches (interrupted commits).
	hfVerifyFn := func(ctx context.Context, pathsInRepo []string) (map[string]bool, error) {
		return hf.pathsExist(ctx, repoID, pathsInRepo)
	}

	// Parse --from flag.
	var fromYear, fromMonth int
	if fromStr != "" {
		t, err := time.Parse("2006-01", fromStr)
		if err != nil {
			return fmt.Errorf("--from: expected YYYY-MM, got %q", fromStr)
		}
		fromYear, fromMonth = t.Year(), int(t.Month())
	}

	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("HN Publish → " + repoID))
	fmt.Println()
	fmt.Printf("  Repo root  %s\n", labelStyle.Render(cfg.RepoRoot))
	fmt.Printf("  HF repo    %s\n", infoStyle.Render(repoID))
	if fromStr != "" {
		fmt.Printf("  From       %s\n", labelStyle.Render(fromStr))
	}
	if live {
		fmt.Printf("  Live mode  every %s\n", infoStyle.Render(interval.String()))
	}
	fmt.Println()

	// Query rich analytics from source (best-effort; README still works without it).
	// Uses a 24h disk cache at {repo-root}/analytics_cache.json to avoid burning
	// the ClickHouse demo quota when both sessions restart simultaneously.
	fmt.Printf("  %s\n", labelStyle.Render("Querying dataset analytics…"))
	analytics, analyticsErr := cfg.QueryAnalyticsCached(ctx, 24*time.Hour)
	if analyticsErr != nil {
		fmt.Printf("  %s %v\n", warningStyle.Render("analytics:"), analyticsErr)
		analytics = nil
	} else {
		fmt.Printf("  %s %s stories, %s comments, %s contributors\n",
			successStyle.Render("analytics:"),
			ccFmtInt64(analytics.Stories),
			ccFmtInt64(analytics.Comments),
			ccFmtInt64(analytics.UniqueAuthors))
	}
	fmt.Println()

	// --- Historical backfill (skipped in --live mode) ---
	if !live {
		histTask := hn2.NewHistoricalTask(cfg, hn2.HistoricalTaskOptions{
			FromYear:   fromYear,
			FromMonth:  fromMonth,
			HFCommit:   hfCommitFn,
			HFVerify:   hfVerifyFn,
			ReadmeTmpl: hnReadmeTmpl,
			Analytics:  analytics,
		})

		metric, err := histTask.Run(ctx, func(s *hn2.HistoricalState) {
			switch s.Phase {
			case "skip":
				fmt.Printf("  [%s] %s\n", labelStyle.Render(s.Month), labelStyle.Render("skipped (already committed)"))
			case "fetch":
				fmt.Printf("  [%s] %s  [%d/%d]\n",
					labelStyle.Render(s.Month), infoStyle.Render("fetching…"), s.MonthIndex, s.MonthTotal)
			case "commit":
				fmt.Printf("  [%s] %s  %s rows\n",
					labelStyle.Render(s.Month), successStyle.Render("committing"), ccFmtInt64(s.Rows))
			}
		})
		if err != nil {
			return fmt.Errorf("historical backfill: %w", err)
		}

		fmt.Println()
		fmt.Printf("  Historical  %s months written, %s skipped\n",
			infoStyle.Render(fmt.Sprintf("%d", metric.MonthsWritten)),
			labelStyle.Render(fmt.Sprintf("%d", metric.MonthsSkipped)))
		fmt.Printf("  Rows        %s\n", infoStyle.Render(ccFmtInt64(metric.RowsWritten)))
		fmt.Printf("  Elapsed     %s\n", labelStyle.Render(metric.Elapsed.Round(time.Second).String()))
		fmt.Println()
		return nil
	}

	// --- Live mode ---
	fmt.Println(subtitleStyle.Render("Live mode — polling every " + interval.String()))
	fmt.Println()

	liveTask := hn2.NewLiveTask(cfg, hn2.LiveTaskOptions{
		Interval:   interval,
		HFCommit:   hfCommitFn,
		ReadmeTmpl: hnReadmeTmpl,
		Analytics:  analytics,
	})

	_, err := liveTask.Run(ctx, func(s *hn2.LiveState) {
		switch s.Phase {
		case "fetch":
			fmt.Printf("  [%s] fetching since id=%d…\n",
				labelStyle.Render(s.Block), s.HighestID)
		case "commit":
			fmt.Printf("  [%s] +%s items  committed\n",
				labelStyle.Render(s.Block), infoStyle.Render(ccFmtInt64(s.NewItems)))
		case "wait":
			fmt.Printf("  [%s] next fetch in %s\n",
				labelStyle.Render(s.Block), labelStyle.Render(s.NextFetchIn.Round(time.Second).String()))
		case "rollover":
			fmt.Printf("  %s day rollover for %s…\n",
				warningStyle.Render("↻"), labelStyle.Render(s.Block))
		}
	})
	return err
}
