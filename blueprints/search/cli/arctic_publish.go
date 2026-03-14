package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/arctic"
	"github.com/spf13/cobra"
)

func newArcticPublish() *cobra.Command {
	var (
		repoRoot   string
		repoID     string
		fromStr    string
		toStr      string
		minFreeGB  int
		chunkLines int
		private    bool
	)

	cmd := &cobra.Command{
		Use:   "publish",
		Short: "Publish Arctic Shift Reddit dataset to Hugging Face",
		Long: `Publish the full Arctic Shift Reddit dataset (comments + submissions) to
a Hugging Face dataset repo, month by month.

Already-committed months (tracked in stats.csv) are skipped — safe to
resume after interruption.

Requires HF_TOKEN environment variable to be set.`,
		Example: `  search arctic publish
  search arctic publish --from 2020-01
  search arctic publish --repo my-org/arctic-reddit --private
  search arctic publish --from 2023-01 --to 2023-12`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runArcticPublish(cmd.Context(), repoRoot, repoID, fromStr, toStr, minFreeGB, chunkLines, private)
		},
	}

	cmd.Flags().StringVar(&repoRoot, "repo-root", "", "Local root directory (default: $HOME/data/arctic/repo)")
	cmd.Flags().StringVar(&repoID, "repo", "open-index/arctic", "Hugging Face dataset repo ID")
	cmd.Flags().StringVar(&fromStr, "from", "2005-06", "Start month YYYY-MM (inclusive)")
	cmd.Flags().StringVar(&toStr, "to", "", "End month YYYY-MM inclusive (default: current month)")
	cmd.Flags().IntVar(&minFreeGB, "min-free-gb", 30, "Minimum free disk GB required to continue")
	cmd.Flags().IntVar(&chunkLines, "chunk-lines", 0, "Lines per JSONL chunk (0 = use package default)")
	cmd.Flags().BoolVar(&private, "private", false, "Create HF repo as private if it does not exist")
	return cmd
}

func runArcticPublish(ctx context.Context, repoRoot, repoID, fromStr, toStr string, minFreeGB, chunkLines int, private bool) error {
	token := strings.TrimSpace(os.Getenv("HF_TOKEN"))
	if token == "" {
		return fmt.Errorf("HF_TOKEN environment variable is not set")
	}

	cfg := arctic.Config{
		RepoRoot:   repoRoot,
		HFRepo:     repoID,
		MinFreeGB:  minFreeGB,
		ChunkLines: chunkLines,
	}
	cfg = cfg.WithDefaults()

	if err := cfg.EnsureDirs(); err != nil {
		return fmt.Errorf("ensure dirs: %w", err)
	}

	hf := newHFClient(token)
	if err := hf.createDatasetRepo(ctx, repoID, private); err != nil {
		fmt.Printf("  note: create repo: %v\n", err)
	}

	// hfCommitFn bridges pkg/arctic.HFOp → cli.hfOperation.
	hfCommitFn := func(ctx context.Context, ops []arctic.HFOp, message string) (string, error) {
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

	// Parse --from flag.
	var fromYear, fromMonth int
	if fromStr != "" {
		t, err := time.Parse("2006-01", fromStr)
		if err != nil {
			return fmt.Errorf("--from: expected YYYY-MM, got %q", fromStr)
		}
		fromYear, fromMonth = t.Year(), int(t.Month())
	}

	// Parse --to flag.
	var toYear, toMonth int
	if toStr != "" {
		t, err := time.Parse("2006-01", toStr)
		if err != nil {
			return fmt.Errorf("--to: expected YYYY-MM, got %q", toStr)
		}
		toYear, toMonth = t.Year(), int(t.Month())
	}

	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Arctic Publish → " + repoID))
	fmt.Println()
	fmt.Printf("  Repo root  %s\n", labelStyle.Render(cfg.RepoRoot))
	fmt.Printf("  HF repo    %s\n", infoStyle.Render(repoID))
	fmt.Printf("  From       %s\n", labelStyle.Render(fromStr))
	if toStr != "" {
		fmt.Printf("  To         %s\n", labelStyle.Render(toStr))
	}
	fmt.Printf("  Min free   %s GB\n", labelStyle.Render(fmt.Sprintf("%d", minFreeGB)))
	fmt.Println()

	task := arctic.NewPublishTask(cfg, arctic.PublishOptions{
		FromYear:  fromYear,
		FromMonth: fromMonth,
		ToYear:    toYear,
		ToMonth:   toMonth,
		HFCommit:  hfCommitFn,
	})

	metric, err := task.Run(ctx, func(s *arctic.PublishState) {
		switch s.Phase {
		case "skip":
			fmt.Printf("  [%s] %s  %s\n",
				labelStyle.Render(s.YM),
				labelStyle.Render(s.Type),
				labelStyle.Render("skipped (already committed)"))
		case "download":
			msg := "downloading…"
			if s.Message != "" {
				msg = s.Message
			}
			if s.Bytes > 0 {
				fmt.Printf("  [%s] %s  %s  %s\n",
					labelStyle.Render(s.YM),
					labelStyle.Render(s.Type),
					infoStyle.Render(msg),
					labelStyle.Render(fmtArcticSize(s.Bytes)))
			} else {
				fmt.Printf("  [%s] %s  %s\n",
					labelStyle.Render(s.YM),
					labelStyle.Render(s.Type),
					infoStyle.Render(msg))
			}
		case "process":
			if s.Shards > 0 {
				fmt.Printf("  [%s] %s  processing  shard %d  %s rows\n",
					labelStyle.Render(s.YM),
					labelStyle.Render(s.Type),
					s.Shards,
					ccFmtInt64(s.Rows))
			} else {
				fmt.Printf("  [%s] %s  %s\n",
					labelStyle.Render(s.YM),
					labelStyle.Render(s.Type),
					infoStyle.Render("processing…"))
			}
		case "commit":
			fmt.Printf("  [%s] %s  %s  %s rows  %d shards\n",
				labelStyle.Render(s.YM),
				labelStyle.Render(s.Type),
				infoStyle.Render("committing"),
				ccFmtInt64(s.Rows),
				s.Shards)
		case "committed":
			// [YYYY-MM] comments  ↓ Xs  ⚙ Xs  ↑ Xs  N shards  NM rows  N MB
			fmt.Printf("  [%s] %s  ↓ %s  ⚙ %s  ↑ %s  %d shards  %s rows  %s\n",
				successStyle.Render(s.YM),
				labelStyle.Render(s.Type),
				infoStyle.Render(fmtArcticDur(s.DurDown)),
				infoStyle.Render(fmtArcticDur(s.DurProc)),
				infoStyle.Render(fmtArcticDur(s.DurComm)),
				s.Shards,
				ccFmtInt64(s.Rows),
				labelStyle.Render(fmtArcticSize(s.Bytes)))
		case "disk_check":
			fmt.Printf("  %s %s\n", warningStyle.Render("disk:"), s.Message)
		case "done":
			fmt.Println()
			fmt.Println(successStyle.Render("  Done!"))
		}
	})
	if err != nil {
		return fmt.Errorf("arctic publish: %w", err)
	}

	fmt.Println()
	fmt.Printf("  Committed  %s months\n", infoStyle.Render(fmt.Sprintf("%d", metric.Committed)))
	fmt.Printf("  Skipped    %s months\n", labelStyle.Render(fmt.Sprintf("%d", metric.Skipped)))
	fmt.Printf("  Elapsed    %s\n", labelStyle.Render(metric.Elapsed.Round(time.Second).String()))
	fmt.Println()
	return nil
}

func fmtArcticDur(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

func fmtArcticSize(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(b)/(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(b)/(1<<20))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
