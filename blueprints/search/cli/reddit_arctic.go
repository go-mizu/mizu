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
		useAPI   bool
		workers  int
	)

	cmd := &cobra.Command{
		Use:   "sub <subreddit>",
		Short: "Download subreddit data via Arctic Shift",
		Long: `Download all comments and submissions for a subreddit.

By default, downloads from the top 40k subreddits torrent (2005-2023, fastest).
Falls back to the Arctic Shift API if the subreddit isn't in the torrent.
Use --api to force API download (slower but supports date ranges and more subreddits).

Data is stored at $HOME/data/reddit/arctic/subreddit/{name}/:
  comments.jsonl       Raw JSONL
  submissions.jsonl    Raw JSONL
  data.duckdb          Imported DuckDB database
  comments.parquet     Exported parquet
  submissions.parquet  Exported parquet

Examples:
  search reddit sub golang
  search reddit sub golang --api --after 2020-01-01
  search reddit sub golang --kind comments
  search reddit sub golang --no-import`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := strings.TrimPrefix(args[0], "r/")
			target := reddit.ArcticTarget{Kind: "subreddit", Name: name}
			return runArcticDownload(cmd.Context(), target, after, before, kind, noImport, useAPI, workers)
		},
	}

	cmd.Flags().StringVar(&after, "after", "", "Start date (YYYY-MM-DD), forces API mode")
	cmd.Flags().StringVar(&before, "before", "", "End date (YYYY-MM-DD), forces API mode")
	cmd.Flags().StringVar(&kind, "kind", "", "Download only: comments or submissions")
	cmd.Flags().BoolVar(&noImport, "no-import", false, "Skip DuckDB import and parquet export")
	cmd.Flags().BoolVar(&useAPI, "api", false, "Force API download (parallel, supports date ranges)")
	cmd.Flags().IntVar(&workers, "workers", 8, "Parallel API download workers")

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
		workers  int
	)

	cmd := &cobra.Command{
		Use:   "user <username>",
		Short: "Download user data via Arctic Shift API",
		Long: `Download all comments and submissions for a user from the Arctic Shift API.

Data is stored at $HOME/data/reddit/arctic/user/{name}/:
  comments.jsonl       Raw JSONL from API
  submissions.jsonl    Raw JSONL from API
  data.duckdb          Imported DuckDB database
  comments.parquet     Exported parquet
  submissions.parquet  Exported parquet

Examples:
  search reddit user spez
  search reddit user spez --kind comments --after 2024-01-01`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := strings.TrimPrefix(args[0], "u/")
			target := reddit.ArcticTarget{Kind: "user", Name: name}
			return runArcticDownload(cmd.Context(), target, after, before, kind, noImport, true, workers)
		},
	}

	cmd.Flags().StringVar(&after, "after", "", "Start date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&before, "before", "", "End date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&kind, "kind", "", "Download only: comments or submissions")
	cmd.Flags().BoolVar(&noImport, "no-import", false, "Skip DuckDB import and parquet export")
	cmd.Flags().IntVar(&workers, "workers", 8, "Parallel download workers")

	cmd.AddCommand(newRedditUserInfo())

	return cmd
}

func runArcticDownload(ctx context.Context, target reddit.ArcticTarget,
	after, before, kind string, noImport, forceAPI bool, workers int) error {

	fmt.Println(Banner())
	prefix := "r/"
	if target.Kind == "user" {
		prefix = "u/"
	}
	fmt.Println(subtitleStyle.Render(fmt.Sprintf("Arctic Shift — %s%s", prefix, target.Name)))
	fmt.Println()

	// Determine which kinds to download
	kinds := []reddit.FileKind{reddit.Comments, reddit.Submissions}
	if kind == "comments" {
		kinds = []reddit.FileKind{reddit.Comments}
	} else if kind == "submissions" {
		kinds = []reddit.FileKind{reddit.Submissions}
	}

	// Date filters force API mode
	if after != "" || before != "" {
		forceAPI = true
	}

	// Try torrent download first (subreddits only, no date filters)
	if !forceAPI && target.Kind == "subreddit" {
		downloaded, err := runTorrentDownload(ctx, target, kinds)
		if err != nil {
			return err
		}
		if downloaded {
			if !noImport {
				if err := runImportPhase(ctx, target, kinds); err != nil {
					return err
				}
			}
			fmt.Println(successStyle.Render("  Done!"))
			return nil
		}
		fmt.Println(labelStyle.Render("  Not in top 40k torrent, using API..."))
		fmt.Println()
	}

	// Parallel API download
	return runParallelAPIDownload(ctx, target, kinds, after, before, noImport, workers)
}

// ── Torrent download ─────────────────────────────────────────

func runTorrentDownload(ctx context.Context, target reddit.ArcticTarget, kinds []reddit.FileKind) (bool, error) {
	// Load or fetch metadata cache
	cache := reddit.LoadSubredditMetaCache()
	if cache == nil {
		fmt.Println(infoStyle.Render("  Fetching torrent metadata (first time, cached after)..."))

		// 2 placeholder lines
		fmt.Println()
		fmt.Println()

		var err error
		cache, err = reddit.FetchSubredditMeta(func(msg string) {
			fmt.Print("\033[2A\033[J")
			fmt.Printf("    %s\n", labelStyle.Render(msg))
			fmt.Println()
		})
		if err != nil {
			fmt.Print("\033[2A\033[J")
			fmt.Println(warningStyle.Render(fmt.Sprintf("  Warning: torrent metadata fetch failed: %v", err)))
			return false, nil // Fall back to API
		}
		fmt.Print("\033[2A\033[J")
		fmt.Printf("  Torrent metadata: %s subreddits cached\n",
			infoStyle.Render(formatNumber(int64(len(cache.Subreddits)))))
		fmt.Println()
	}

	// Lookup subreddit
	meta, found := cache.LookupSubreddit(target.Name)
	if !found {
		return false, nil
	}

	// Use actual cased name from torrent
	target.Name = meta.Name

	fmt.Printf("  Source:  %s\n", infoStyle.Render("Top 40k subreddits torrent"))
	if meta.CommentsSize > 0 {
		fmt.Printf("  Comments:    %s compressed\n", labelStyle.Render(formatBytes(meta.CommentsSize)))
	}
	if meta.SubmissionsSize > 0 {
		fmt.Printf("  Submissions: %s compressed\n", labelStyle.Render(formatBytes(meta.SubmissionsSize)))
	}
	fmt.Println()

	// 5 placeholder lines for torrent progress
	for i := 0; i < 5; i++ {
		fmt.Println()
	}

	err := reddit.DownloadSubredditTorrent(ctx, target, kinds, meta, cache.TorrentFile, func(p reddit.TorrentDownloadProgress) {
		renderTorrentProgress(p)
	})

	fmt.Print("\033[5A\033[J")

	if err != nil {
		// Check if it's a timeout (no peers) — fall back to API
		if strings.Contains(err.Error(), "torrent timeout") {
			fmt.Println(warningStyle.Render("  No torrent peers found (60s timeout), falling back to API..."))
			fmt.Println()
			return false, nil
		}
		return true, err
	}

	// Show downloaded files
	for _, k := range kinds {
		jsonlPath := target.JSONLPath(k)
		if st, err := os.Stat(jsonlPath); err == nil {
			kindLabel := "Comments"
			if k == reddit.Submissions {
				kindLabel = "Submissions"
			}
			fmt.Printf("  %s: %s (%s)\n",
				successStyle.Render(kindLabel),
				labelStyle.Render(shortenHome(jsonlPath)),
				infoStyle.Render(formatBytes(st.Size())))
		}
	}
	fmt.Println()

	return true, nil
}

func renderTorrentProgress(p reddit.TorrentDownloadProgress) {
	fmt.Print("\033[5A\033[J")

	switch p.Phase {
	case "metadata":
		fmt.Println(labelStyle.Render("    Loading torrent..."))
		fmt.Println()
		fmt.Println()
		fmt.Println()
		fmt.Println()

	case "download":
		pct := float64(0)
		if p.BytesTotal > 0 {
			pct = 100.0 * float64(p.BytesCompleted) / float64(p.BytesTotal)
		}
		barWidth := 40
		filled := int(pct / 100.0 * float64(barWidth))
		if filled > barWidth {
			filled = barWidth
		}
		bar := strings.Repeat("━", filled) + strings.Repeat("─", barWidth-filled)

		fmt.Printf("    Downloading %s\n", infoStyle.Render(p.File))
		if pct >= 100 {
			fmt.Printf("    %s  %s\n", successStyle.Render(bar), successStyle.Render(fmt.Sprintf("%.1f%%", pct)))
		} else {
			fmt.Printf("    %s  %.1f%%\n", infoStyle.Render(bar), pct)
		}
		fmt.Printf("    Speed: %s  Peers: %d\n",
			infoStyle.Render(formatBytesPerSec(p.Speed)), p.Peers)
		fmt.Printf("    Downloaded: %s / %s\n",
			infoStyle.Render(formatBytes(p.BytesCompleted)),
			labelStyle.Render(formatBytes(p.BytesTotal)))
		fmt.Printf("    ETA: %s  Elapsed: %s\n",
			infoStyle.Render(formatDuration(p.ETA)),
			labelStyle.Render(formatDuration(p.Elapsed)))

	case "decompress":
		fmt.Printf("    Decompressing %s\n", infoStyle.Render(p.File))
		if p.DecompressedBytes > 0 {
			fmt.Printf("    Decompressed: %s\n", infoStyle.Render(formatBytes(p.DecompressedBytes)))
		} else {
			fmt.Println()
		}
		fmt.Printf("    Elapsed: %s\n", labelStyle.Render(formatDuration(p.Elapsed)))
		fmt.Println()
		fmt.Println()

	case "done":
		fmt.Println(successStyle.Render("    Download complete!"))
		fmt.Println()
		fmt.Println()
		fmt.Println()
		fmt.Println()
	}
}

// ── Parallel API download ────────────────────────────────────

func runParallelAPIDownload(ctx context.Context, target reddit.ArcticTarget,
	kinds []reddit.FileKind, after, before string, noImport bool, workers int) error {

	client := reddit.NewArcticClient()

	var afterEpoch, beforeEpoch int64
	if after != "" {
		t, err := parseDate(after)
		if err != nil {
			return fmt.Errorf("invalid --after: %w", err)
		}
		afterEpoch = t.Unix()
	}
	if before != "" {
		t, err := parseDate(before)
		if err != nil {
			return fmt.Errorf("invalid --before: %w", err)
		}
		beforeEpoch = t.Unix()
	}

	prefix := "r/"
	if target.Kind == "user" {
		prefix = "u/"
	}

	// Get min date
	minDate, err := client.GetMinDate(ctx, target)
	if err != nil {
		return fmt.Errorf("get min date: %w", err)
	}

	fmt.Printf("  Earliest data: %s\n", infoStyle.Render(minDate.Format("2006-01-02")))
	fmt.Printf("  Data dir:      %s\n", labelStyle.Render(shortenHome(target.Dir())))
	fmt.Printf("  Method:        %s\n", labelStyle.Render(fmt.Sprintf("Arctic Shift API (%d parallel workers)", workers)))
	fmt.Println()

	for _, k := range kinds {
		kindLabel := "Comments"
		if k == reddit.Submissions {
			kindLabel = "Submissions"
		}

		fmt.Printf("  %s%s — %s\n", prefix, target.Name, infoStyle.Render(kindLabel))

		// 4 placeholder lines
		for i := 0; i < 4; i++ {
			fmt.Println()
		}

		err := client.ParallelDownload(ctx, target, k, afterEpoch, beforeEpoch, workers,
			func(p reddit.ArcticProgress) {
				renderArcticProgress(p, kindLabel)
			})
		if err != nil {
			fmt.Println(errorStyle.Render(fmt.Sprintf("  Error: %v", err)))
			return err
		}

		if st, err := os.Stat(target.JSONLPath(k)); err == nil {
			fmt.Print("\033[4A\033[J")
			fmt.Printf("    %s complete: %s\n", successStyle.Render(kindLabel), infoStyle.Render(formatBytes(st.Size())))
			fmt.Println()
		}
	}

	if !noImport {
		if err := runImportPhase(ctx, target, kinds); err != nil {
			return err
		}
	}

	fmt.Println(successStyle.Render("  Done!"))
	return nil
}

// ── Import phase ─────────────────────────────────────────────

func runImportPhase(ctx context.Context, target reddit.ArcticTarget, kinds []reddit.FileKind) error {
	fmt.Println(infoStyle.Render("  Importing to DuckDB + parquet..."))
	fmt.Println()
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

	return nil
}

// ── Progress rendering ───────────────────────────────────────

func renderArcticProgress(p reddit.ArcticProgress, kindLabel string) {
	fmt.Print("\033[4A\033[J")

	if p.Done {
		fmt.Printf("    %s download complete\n", successStyle.Render(kindLabel))
	} else {
		// Show items with estimated total based on chunk progress
		itemsStr := formatNumber(p.Items)
		if p.ChunksDone > 0 && p.TotalChunks > 0 && p.ChunksDone < p.TotalChunks {
			estTotal := p.Items * int64(p.TotalChunks) / int64(p.ChunksDone)
			itemsStr += " / ~" + formatNumber(estTotal)
		}
		fmt.Printf("    Items: %s   Size: %s\n",
			infoStyle.Render(itemsStr),
			infoStyle.Render(formatBytes(p.Bytes)))
	}

	if !p.Oldest.IsZero() && !p.Newest.IsZero() {
		fmt.Printf("    Range: %s → %s\n",
			labelStyle.Render(p.Oldest.Format("2006-01-02")),
			labelStyle.Render(p.Newest.Format("2006-01-02")))
	} else {
		fmt.Println()
	}

	if p.Items > 0 && p.Elapsed > 0 {
		speed := float64(p.Items) / p.Elapsed.Seconds()
		// ETA based on chunk progress
		etaStr := "—"
		if p.ChunksDone > 0 && p.TotalChunks > 0 && p.ChunksDone < p.TotalChunks {
			remaining := p.TotalChunks - p.ChunksDone
			perChunk := p.Elapsed / time.Duration(p.ChunksDone)
			eta := perChunk * time.Duration(remaining)
			etaStr = formatDuration(eta)
		}
		fmt.Printf("    Speed: %s items/s   Chunks: %s   ETA: %s   Elapsed: %s\n",
			infoStyle.Render(fmt.Sprintf("%.0f", speed)),
			labelStyle.Render(fmt.Sprintf("%d/%d", p.ChunksDone, p.TotalChunks)),
			infoStyle.Render(etaStr),
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

// ── Info subcommands ─────────────────────────────────────────

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

	fmt.Println(infoStyle.Render("  Sizes"))
	fmt.Println(labelStyle.Render("  " + strings.Repeat("─", 50)))
	if info.CommentsJSONLSize > 0 {
		fmt.Printf("  Comments JSONL:      %s\n", formatBytes(info.CommentsJSONLSize))
	}
	if info.SubmissionsJSONLSize > 0 {
		fmt.Printf("  Submissions JSONL:   %s\n", formatBytes(info.SubmissionsJSONLSize))
	}
	fmt.Printf("  DuckDB:              %s\n", formatBytes(info.DBSize))
	if info.CommentsPQSize > 0 {
		fmt.Printf("  Comments Parquet:    %s\n", formatBytes(info.CommentsPQSize))
	}
	if info.SubmissionsPQSize > 0 {
		fmt.Printf("  Submissions Parquet: %s\n", formatBytes(info.SubmissionsPQSize))
	}
	fmt.Println()

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

	if len(info.TopAuthors) > 0 {
		fmt.Println(infoStyle.Render("  Top Authors"))
		fmt.Println(labelStyle.Render("  " + strings.Repeat("─", 50)))
		for _, a := range info.TopAuthors {
			fmt.Printf("  %-30s %s\n", a.Name, labelStyle.Render(formatNumber(a.Count)))
		}
		fmt.Println()
	}

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
