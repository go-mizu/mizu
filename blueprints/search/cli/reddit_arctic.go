package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/reddit"
	"github.com/spf13/cobra"
)

// newRedditSub creates the "reddit sub" command.
func newRedditSub() *cobra.Command {
	var (
		after    string
		before   string
		kind     string
		noImport bool
		resume   bool
	)

	cmd := &cobra.Command{
		Use:   "sub <subreddit>",
		Short: "Download subreddit data via Arctic Shift",
		Long: `Download all comments and submissions for a subreddit from the Arctic Shift API.

Data is stored at $HOME/data/reddit/arctic/subreddit/{name}/:
  comments.jsonl       Raw JSONL from API
  submissions.jsonl    Raw JSONL from API
  data.duckdb          Imported DuckDB database
  comments.parquet     Exported parquet
  submissions.parquet  Exported parquet

Examples:
  search reddit sub golang
  search reddit sub golang --after 2020-01-01 --before 2023-01-01
  search reddit sub golang --kind comments
  search reddit sub golang --no-import
  search reddit sub golang --resume`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := strings.TrimPrefix(args[0], "r/")
			target := reddit.ArcticTarget{Kind: "subreddit", Name: name}
			return runArcticDownload(cmd.Context(), target, after, before, kind, noImport, resume)
		},
	}

	cmd.Flags().StringVar(&after, "after", "", "Start date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&before, "before", "", "End date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&kind, "kind", "", "Download only: comments or submissions")
	cmd.Flags().BoolVar(&noImport, "no-import", false, "Skip DuckDB import and parquet export")
	cmd.Flags().BoolVar(&resume, "resume", false, "Resume interrupted download")

	// Add info subcommand
	cmd.AddCommand(newRedditSubInfo())

	return cmd
}

// newRedditUser creates the "reddit user" command.
func newRedditUser() *cobra.Command {
	var (
		after    string
		before   string
		kind     string
		noImport bool
		resume   bool
	)

	cmd := &cobra.Command{
		Use:   "user <username>",
		Short: "Download user data via Arctic Shift",
		Long: `Download all comments and submissions for a user from the Arctic Shift API.

Data is stored at $HOME/data/reddit/arctic/user/{name}/:
  comments.jsonl       Raw JSONL from API
  submissions.jsonl    Raw JSONL from API
  data.duckdb          Imported DuckDB database
  comments.parquet     Exported parquet
  submissions.parquet  Exported parquet

Examples:
  search reddit user spez
  search reddit user spez --kind comments --after 2024-01-01
  search reddit user spez --resume`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := strings.TrimPrefix(args[0], "u/")
			target := reddit.ArcticTarget{Kind: "user", Name: name}
			return runArcticDownload(cmd.Context(), target, after, before, kind, noImport, resume)
		},
	}

	cmd.Flags().StringVar(&after, "after", "", "Start date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&before, "before", "", "End date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&kind, "kind", "", "Download only: comments or submissions")
	cmd.Flags().BoolVar(&noImport, "no-import", false, "Skip DuckDB import and parquet export")
	cmd.Flags().BoolVar(&resume, "resume", false, "Resume interrupted download")

	// Add info subcommand
	cmd.AddCommand(newRedditUserInfo())

	return cmd
}

func runArcticDownload(ctx context.Context, target reddit.ArcticTarget, after, before, kind string, noImport, resume bool) error {
	fmt.Println(Banner())
	prefix := "r/"
	if target.Kind == "user" {
		prefix = "u/"
	}
	fmt.Println(subtitleStyle.Render(fmt.Sprintf("Arctic Shift — %s%s", prefix, target.Name)))
	fmt.Println()

	client := reddit.NewArcticClient()

	// Parse date flags
	var afterEpoch, beforeEpoch int64
	if after != "" {
		t, err := parseDate(after)
		if err != nil {
			return fmt.Errorf("invalid --after date: %w", err)
		}
		afterEpoch = t.Unix()
	}
	if before != "" {
		t, err := parseDate(before)
		if err != nil {
			return fmt.Errorf("invalid --before date: %w", err)
		}
		beforeEpoch = t.Unix()
	}

	// Determine which kinds to download
	kinds := []reddit.FileKind{reddit.Comments, reddit.Submissions}
	if kind == "comments" {
		kinds = []reddit.FileKind{reddit.Comments}
	} else if kind == "submissions" {
		kinds = []reddit.FileKind{reddit.Submissions}
	}

	// Get min date for display
	minDate, err := client.GetMinDate(ctx, target)
	if err != nil {
		return fmt.Errorf("get min date: %w", err)
	}
	fmt.Printf("  Earliest data: %s\n", infoStyle.Render(minDate.Format("2006-01-02")))
	fmt.Printf("  Data dir:      %s\n", labelStyle.Render(shortenHome(target.Dir())))
	fmt.Println()

	// Load resume progress
	var commentsAfter, submissionsAfter int64
	if resume {
		commentsAfter, submissionsAfter = reddit.LoadProgress(target)
		if commentsAfter > 0 || submissionsAfter > 0 {
			fmt.Println(infoStyle.Render("  Resuming from previous progress..."))
			if commentsAfter > 0 {
				fmt.Printf("    Comments:    from %s\n", labelStyle.Render(time.Unix(commentsAfter, 0).Format("2006-01-02 15:04:05")))
			}
			if submissionsAfter > 0 {
				fmt.Printf("    Submissions: from %s\n", labelStyle.Render(time.Unix(submissionsAfter, 0).Format("2006-01-02 15:04:05")))
			}
			fmt.Println()
		}
	}

	// Download each kind
	for _, k := range kinds {
		startAfter := afterEpoch
		if resume {
			if k == reddit.Comments && commentsAfter > startAfter {
				startAfter = commentsAfter
			} else if k == reddit.Submissions && submissionsAfter > startAfter {
				startAfter = submissionsAfter
			}
		}

		kindLabel := "Comments"
		if k == reddit.Submissions {
			kindLabel = "Submissions"
		}

		fmt.Printf("  %s%s — %s\n", prefix, target.Name, infoStyle.Render(kindLabel))

		// Print 4 placeholder lines for progress
		for i := 0; i < 4; i++ {
			fmt.Println()
		}

		err := client.Download(ctx, target, k, startAfter, beforeEpoch, func(p reddit.ArcticProgress) {
			renderArcticProgress(p, kindLabel)
		})
		if err != nil {
			fmt.Println(errorStyle.Render(fmt.Sprintf("  Error: %v", err)))
			return err
		}

		// Show final file size
		if st, err := os.Stat(target.JSONLPath(k)); err == nil {
			fmt.Print("\033[4A\033[J")
			fmt.Printf("    %s download complete\n", successStyle.Render(kindLabel))
			fmt.Printf("    File: %s (%s)\n", labelStyle.Render(shortenHome(target.JSONLPath(k))), infoStyle.Render(formatBytes(st.Size())))
			fmt.Println()
		}
	}

	// Import to DuckDB + parquet
	if !noImport {
		fmt.Println(infoStyle.Render("  Importing to DuckDB + parquet..."))
		fmt.Println()
		// 3 placeholder lines
		fmt.Println()
		fmt.Println()
		fmt.Println()

		err := reddit.ArcticImport(ctx, target, kinds, func(p reddit.ImportProgress) {
			fmt.Print("\033[3A\033[J")
			phase := labelStyle.Render(p.Phase)
			if p.Done {
				phase = successStyle.Render(p.Phase + " ✓")
			}
			fmt.Printf("    Phase: %s", phase)
			if p.Rows > 0 {
				fmt.Printf("  Rows: %s", infoStyle.Render(formatNumber(p.Rows)))
			}
			if p.Detail != "" {
				fmt.Printf("  [%s]", labelStyle.Render(p.Detail))
			}
			fmt.Println()
			fmt.Printf("    Elapsed: %s\n", labelStyle.Render(formatDuration(p.Elapsed)))
			fmt.Println()
		})
		if err != nil {
			return fmt.Errorf("import: %w", err)
		}

		// Show result sizes
		fmt.Print("\033[3A\033[J")
		if st, err := os.Stat(target.DBPath()); err == nil {
			fmt.Printf("    DuckDB:  %s (%s)\n",
				labelStyle.Render(shortenHome(target.DBPath())),
				infoStyle.Render(formatBytes(st.Size())))
		}
		for _, k := range kinds {
			pqPath := target.ParquetPath(k)
			if st, err := os.Stat(pqPath); err == nil {
				fmt.Printf("    Parquet: %s (%s)\n",
					labelStyle.Render(shortenHome(pqPath)),
					infoStyle.Render(formatBytes(st.Size())))
			}
		}
		fmt.Println()
	}

	fmt.Println(successStyle.Render("  Done!"))
	return nil
}

func renderArcticProgress(p reddit.ArcticProgress, kindLabel string) {
	fmt.Print("\033[4A\033[J")

	if p.Done {
		fmt.Printf("    %s download complete\n", successStyle.Render(kindLabel))
	} else {
		fmt.Printf("    Items: %s   Size: %s\n",
			infoStyle.Render(formatNumber(p.Items)),
			infoStyle.Render(formatBytes(p.Bytes)))
	}

	if !p.Oldest.IsZero() && !p.Newest.IsZero() {
		fmt.Printf("    Range: %s  →  %s\n",
			labelStyle.Render(p.Oldest.Format("2006-01-02")),
			labelStyle.Render(p.Newest.Format("2006-01-02")))
	} else {
		fmt.Println()
	}

	if p.Items > 0 && p.Elapsed > 0 {
		speed := float64(p.Items) / p.Elapsed.Seconds()
		fmt.Printf("    Speed: %s items/s   Batch: %d   Elapsed: %s\n",
			infoStyle.Render(fmt.Sprintf("%.0f", speed)),
			p.BatchSize,
			labelStyle.Render(formatDuration(p.Elapsed)))
	} else {
		fmt.Println()
	}
	fmt.Println()
}

func parseDate(s string) (time.Time, error) {
	formats := []string{
		"2006-01-02",
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized date format: %s (use YYYY-MM-DD)", s)
}

// ── reddit sub info ──────────────────────────────────────────

func newRedditSubInfo() *cobra.Command {
	return &cobra.Command{
		Use:   "info <subreddit>",
		Short: "Show statistics for a downloaded subreddit",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := strings.TrimPrefix(args[0], "r/")
			target := reddit.ArcticTarget{Kind: "subreddit", Name: name}
			return runArcticInfo(target)
		},
	}
}

func newRedditUserInfo() *cobra.Command {
	return &cobra.Command{
		Use:   "info <username>",
		Short: "Show statistics for a downloaded user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := strings.TrimPrefix(args[0], "u/")
			target := reddit.ArcticTarget{Kind: "user", Name: name}
			return runArcticInfo(target)
		},
	}
}

func runArcticInfo(target reddit.ArcticTarget) error {
	fmt.Println(Banner())
	prefix := "r/"
	if target.Kind == "user" {
		prefix = "u/"
	}
	fmt.Println(subtitleStyle.Render(fmt.Sprintf("Arctic Shift — %s%s", prefix, target.Name)))
	fmt.Println()

	info, err := reddit.GetArcticInfo(target)
	if err != nil {
		return err
	}

	// Sizes
	fmt.Println(infoStyle.Render("  Sizes"))
	fmt.Println(labelStyle.Render("  " + strings.Repeat("─", 50)))
	if info.CommentsJSONLSize > 0 {
		fmt.Printf("  Comments JSONL:     %s\n", formatBytes(info.CommentsJSONLSize))
	}
	if info.SubmissionsJSONLSize > 0 {
		fmt.Printf("  Submissions JSONL:  %s\n", formatBytes(info.SubmissionsJSONLSize))
	}
	fmt.Printf("  DuckDB:             %s\n", formatBytes(info.DBSize))
	if info.CommentsPQSize > 0 {
		fmt.Printf("  Comments Parquet:   %s\n", formatBytes(info.CommentsPQSize))
	}
	if info.SubmissionsPQSize > 0 {
		fmt.Printf("  Submissions Parquet: %s\n", formatBytes(info.SubmissionsPQSize))
	}
	fmt.Println()

	// Data
	fmt.Println(infoStyle.Render("  Data"))
	fmt.Println(labelStyle.Render("  " + strings.Repeat("─", 50)))
	fmt.Printf("  Comments:     %s\n", infoStyle.Render(formatNumber(info.CommentsRows)))
	fmt.Printf("  Submissions:  %s\n", infoStyle.Render(formatNumber(info.SubmissionsRows)))
	total := info.CommentsRows + info.SubmissionsRows
	fmt.Printf("  Total:        %s\n", infoStyle.Render(formatNumber(total)))
	if info.DateRange[0] != "" {
		fmt.Printf("  Date range:   %s → %s\n", info.DateRange[0], info.DateRange[1])
	}
	fmt.Println()

	// Top Authors
	if len(info.TopAuthors) > 0 {
		fmt.Println(infoStyle.Render("  Top Authors"))
		fmt.Println(labelStyle.Render("  " + strings.Repeat("─", 50)))
		for _, a := range info.TopAuthors {
			fmt.Printf("  %-30s %s\n", a.Name, labelStyle.Render(formatNumber(a.Count)))
		}
		fmt.Println()
	}

	// Top Subreddits (for user downloads)
	if len(info.TopSubreddits) > 0 {
		fmt.Println(infoStyle.Render("  Top Subreddits"))
		fmt.Println(labelStyle.Render("  " + strings.Repeat("─", 50)))
		for _, s := range info.TopSubreddits {
			fmt.Printf("  %-30s %s\n", s.Name, labelStyle.Render(formatNumber(s.Count)))
		}
		fmt.Println()
	}

	return nil
}
