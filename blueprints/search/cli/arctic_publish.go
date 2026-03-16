package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	_ "net/http/pprof"
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
	cmd.Flags().StringVar(&fromStr, "from", "2005-12", "Start month YYYY-MM (inclusive)")
	cmd.Flags().StringVar(&toStr, "to", "", "End month YYYY-MM inclusive (default: current month)")
	cmd.Flags().IntVar(&minFreeGB, "min-free-gb", 30, "Minimum free disk GB required to continue")
	cmd.Flags().IntVar(&chunkLines, "chunk-lines", 0, "Lines per JSONL chunk (0 = use package default)")
	cmd.Flags().BoolVar(&private, "private", false, "Create HF repo as private if it does not exist")
	return cmd
}

func runArcticPublish(ctx context.Context, repoRoot, repoID, fromStr, toStr string, minFreeGB, chunkLines int, private bool) error {
	// Start pprof HTTP server for live heap profiling on :6060.
	go func() {
		fmt.Fprintf(os.Stderr, "arctic: pprof listening on :6060 (http://localhost:6060/debug/pprof/)\n")
		if err := http.ListenAndServe(":6060", nil); err != nil {
			fmt.Fprintf(os.Stderr, "arctic: pprof server: %v\n", err)
		}
	}()

	// Log memory stats every 30s to help diagnose OOM.

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
	// Automatically retries on 429 Too Many Requests, sleeping for the
	// server-requested Retry-After duration before each retry.
	// On "connection reset by peer" it waits 15s then verifies whether the
	// commit actually landed on HF (by checking if the uploaded parquet shard
	// files exist) before deciding to treat it as an error.
	hfCommitFn := func(ctx context.Context, ops []arctic.HFOp, message string) (string, error) {
		var hfOps []hfOperation
		for _, op := range ops {
			hfOps = append(hfOps, hfOperation{
				LocalPath:  op.LocalPath,
				PathInRepo: op.PathInRepo,
				Delete:     op.Delete,
			})
		}
		const maxRateLimitRetries = 3
		for attempt := 0; attempt < maxRateLimitRetries; attempt++ {
			url, err := hf.createCommit(ctx, repoID, "main", message, hfOps)
			if err == nil {
				return url, nil
			}

			// "Connection reset by peer" means the TCP connection dropped, but
			// HuggingFace may have already processed the commit.  Wait briefly
			// then check whether the uploaded files now exist in the repo.
			if isConnectionReset(err) {
				fmt.Fprintf(os.Stderr, "arctic: connection reset by peer during HF commit — waiting 15s to verify commit landed…\n")
				select {
				case <-ctx.Done():
					return "", ctx.Err()
				case <-time.After(15 * time.Second):
				}
				if verified, verifyErr := hfVerifyOpsExist(ctx, hf, repoID, hfOps); verifyErr == nil && verified {
					fmt.Fprintf(os.Stderr, "arctic: connection reset: commit verified on HF — continuing\n")
					return "", nil
				} else if verifyErr != nil {
					fmt.Fprintf(os.Stderr, "arctic: connection reset: verify check failed: %v — treating as commit error\n", verifyErr)
				} else {
					fmt.Fprintf(os.Stderr, "arctic: connection reset: commit not yet visible on HF — will retry\n")
				}
			}

			var rlErr *HFRateLimitError
			if !errors.As(err, &rlErr) || attempt == maxRateLimitRetries-1 {
				return "", err
			}
			wait := rlErr.RetryAfter + 30*time.Second
			if wait < time.Minute {
				wait = time.Minute
			}
			fmt.Fprintf(os.Stderr, "arctic: 429 rate limited — sleeping %s before retry %d/%d\n",
				wait.Round(time.Second), attempt+1, maxRateLimitRetries)
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(wait):
			}
		}
		return "", fmt.Errorf("unreachable")
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

	opts := arctic.PublishOptions{
		FromYear:  fromYear,
		FromMonth: fromMonth,
		ToYear:    toYear,
		ToMonth:   toMonth,
		HFCommit:  hfCommitFn,
	}

	// Auto-detect hardware and compute resource budget.
	task := arctic.NewPipelineTask(cfg, opts)
	hw := task.Hardware()
	budget := task.Budget()

	fmt.Println()
	fmt.Printf("  Hardware   %s\n", labelStyle.Render(hw.String()))
	fmt.Printf("  Budget     %s\n", infoStyle.Render(budget.String()))
	fmt.Println()

	metric, err := task.Run(ctx, func(s *arctic.PublishState) {
		switch s.Phase {
		case "skip":
			fmt.Printf("  [%s] %s  %s\n",
				labelStyle.Render(s.YM),
				labelStyle.Render(s.Type),
				labelStyle.Render("skipped (already committed)"))
		case "download":
			if s.Bytes > 0 && s.BytesTotal > 0 {
				pct := int(100 * s.Bytes / s.BytesTotal)
				line := fmt.Sprintf("%s / %s  %d%%",
					fmtArcticSize(s.Bytes), fmtArcticSize(s.BytesTotal), pct)
				if s.Message != "" {
					line += "  " + s.Message
				}
				fmt.Printf("\r  [%s] %s  %s  %-60s",
					labelStyle.Render(s.YM),
					labelStyle.Render(s.Type),
					infoStyle.Render("downloading"),
					line)
			} else if s.Bytes > 0 {
				fmt.Printf("\r  [%s] %s  %s  %s  %-40s",
					labelStyle.Render(s.YM),
					labelStyle.Render(s.Type),
					infoStyle.Render("downloading"),
					labelStyle.Render(fmtArcticSize(s.Bytes)),
					s.Message)
			} else {
				msg := "connecting…"
				if s.Message != "" {
					msg = s.Message
				}
				fmt.Printf("\r  [%s] %s  %s  %-60s",
					labelStyle.Render(s.YM),
					labelStyle.Render(s.Type),
					infoStyle.Render("downloading"),
					msg)
			}
		case "validate":
			fmt.Printf("\r  [%s] %s  %s  %-60s\n",
				labelStyle.Render(s.YM),
				labelStyle.Render(s.Type),
				infoStyle.Render("validating"),
				s.Message)
		case "process_start":
			// Shard just started — DuckDB is working; overwrite line with activity.
			fmt.Printf("\r  [%s] %s  %s  shard %d  %s rows so far  %-20s",
				labelStyle.Render(s.YM),
				labelStyle.Render(s.Type),
				infoStyle.Render("processing"),
				s.Shards,
				ccFmtInt64(s.Rows),
				"converting…")
		case "process":
			// Shard completed — overwrite with speed info.
			speed := ""
			if s.RowsPerSec >= 1000 {
				speed = fmt.Sprintf("  %.1fK rows/s", s.RowsPerSec/1000)
			} else if s.RowsPerSec > 0 {
				speed = fmt.Sprintf("  %.0f rows/s", s.RowsPerSec)
			}
			fmt.Printf("\r  [%s] %s  %s  shard %d  %s rows%s%-20s",
				labelStyle.Render(s.YM),
				labelStyle.Render(s.Type),
				infoStyle.Render("processing"),
				s.Shards,
				ccFmtInt64(s.Rows),
				speed,
				"")
		case "commit":
			fmt.Printf("\n  [%s] %s  %s  %s rows  %d shards\n",
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

// isConnectionReset returns true if err contains "connection reset by peer".
func isConnectionReset(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "connection reset by peer")
}

// hfVerifyOpsExist checks whether the non-delete, non-metadata ops in a batch
// (i.e. the new parquet shard files) now exist in the remote HF repo.
// Returns (true, nil)  — all shard files found: commit confirmed.
// Returns (false, nil) — at least one shard missing: commit not confirmed.
// Returns (false, err) — the existence check itself failed.
// If ops contain no shard files (heartbeat-only batch) it returns (true, nil)
// since such commits are idempotent and non-critical.
func hfVerifyOpsExist(ctx context.Context, hf *hfClient, repoID string, ops []hfOperation) (bool, error) {
	metaPaths := map[string]bool{
		"stats.csv":      true,
		"README.md":      true,
		"states.json":    true,
		"zst_sizes.json": true,
	}
	var checkPaths []string
	for _, op := range ops {
		if !op.Delete && !metaPaths[op.PathInRepo] {
			checkPaths = append(checkPaths, op.PathInRepo)
		}
	}
	if len(checkPaths) == 0 {
		// Only metadata — cannot distinguish updated from pre-existing; assume ok.
		return true, nil
	}
	existing, err := hf.pathsExist(ctx, repoID, checkPaths)
	if err != nil {
		return false, err
	}
	for _, p := range checkPaths {
		if !existing[p] {
			return false, nil
		}
	}
	return true, nil
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
