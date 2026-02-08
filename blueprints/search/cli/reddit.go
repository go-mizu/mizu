package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/reddit"
	"github.com/go-mizu/mizu/blueprints/search/pkg/torrent"
	"github.com/spf13/cobra"
)

// NewReddit creates the reddit command with subcommands.
func NewReddit() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reddit",
		Short: "Reddit archive data (Pushshift, 2005-2025)",
		Long: `Download and process the Pushshift Reddit archive via BitTorrent.

The archive contains 488 files (241 comments + 247 submissions) totaling 3.8 TB,
covering June 2005 through December 2025 in zst-compressed ndjson format.

Data is stored at $HOME/data/reddit/ with three tiers:
  raw/        Downloaded .zst files from torrent
  database/   Imported DuckDB files
  parquet/    Exported parquet files

Subcommands:
  list       List all files in the torrent with download/import status
  download   Download specific files from the torrent
  import     Import downloaded files to DuckDB + parquet (auto-downloads if missing)
  info       Show statistics for an imported file
  sub        Download subreddit data via Arctic Shift API
  user       Download user data via Arctic Shift API

File arguments are flexible — all of these work:
  comments/RC_2005-12.zst    (full torrent path)
  RC_2005-12                 (just the name, RC_=comments RS_=submissions)
  2005-12                    (bare date, use --kind to specify type)

Examples:
  search reddit list
  search reddit list --kind comments --year 2005
  search reddit download 2005-12 --kind comments
  search reddit download --last 3 --kind submissions
  search reddit download --from 2020-01 --to 2020-06
  search reddit import RC_2005-12
  search reddit import --kind comments --year 2005
  search reddit info RC_2005-12`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newRedditList())
	cmd.AddCommand(newRedditDownload())
	cmd.AddCommand(newRedditImport())
	cmd.AddCommand(newRedditInfo())
	cmd.AddCommand(newRedditSub())
	cmd.AddCommand(newRedditUser())

	return cmd
}

// ── shared: metadata loading with cache ──────────────────

// getRedditFiles returns all files from cache or torrent, caching for future calls.
func getRedditFiles(ctx context.Context, noCache bool) ([]reddit.DataFile, error) {
	// Try cache first
	if !noCache {
		if cache := reddit.LoadCache(); cache != nil {
			return reddit.CachedDataFiles(cache), nil
		}
	}

	// Fetch from torrent
	fmt.Println(labelStyle.Render("  Fetching torrent metadata from peers..."))

	cfg := torrent.Config{
		DataDir:  reddit.RawDir(),
		InfoHash: reddit.InfoHash,
		Trackers: reddit.Trackers,
		NoUpload: true,
	}

	cl, err := torrent.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("create torrent client: %w", err)
	}
	defer cl.Close()

	tFiles, err := cl.Files(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch torrent metadata: %w", err)
	}

	// Save to cache
	cached := make([]reddit.CachedFile, len(tFiles))
	for i, f := range tFiles {
		cached[i] = reddit.CachedFile{Path: f.Path, Size: f.Length}
	}
	if err := reddit.SaveCache(cached); err != nil {
		// Non-fatal, just skip caching
		fmt.Println(warningStyle.Render(fmt.Sprintf("  Warning: could not save metadata cache: %v", err)))
	}

	// Clear the "fetching" line
	fmt.Print("\033[1A\033[2K")

	// Convert to DataFiles
	var files []reddit.DataFile
	for _, f := range tFiles {
		path := f.Path
		if len(path) > 7 && path[:7] == "reddit/" {
			path = path[7:]
		}
		df, ok := reddit.ParseTorrentPath(path, f.Length)
		if !ok {
			continue
		}
		files = append(files, df)
	}
	return files, nil
}

// downloadFiles downloads the given files via torrent with progress display.
// Returns nil if all files downloaded successfully.
func downloadFiles(ctx context.Context, files []reddit.DataFile) error {
	// Build torrent paths
	var paths []string
	var totalSize int64
	for _, df := range files {
		paths = append(paths, reddit.TorrentPath(df))
		totalSize += df.Size
	}

	os.MkdirAll(filepath.Join(reddit.RawDir(), "reddit", "comments"), 0o755)
	os.MkdirAll(filepath.Join(reddit.RawDir(), "reddit", "submissions"), 0o755)

	cfg := torrent.Config{
		DataDir:  reddit.RawDir(),
		InfoHash: reddit.InfoHash,
		Trackers: reddit.Trackers,
		NoUpload: true,
	}

	cl, err := torrent.New(cfg)
	if err != nil {
		return fmt.Errorf("create torrent client: %w", err)
	}
	defer cl.Close()

	fmt.Println(infoStyle.Render(fmt.Sprintf("  Downloading %d file(s), %s total",
		len(paths), formatBytes(totalSize))))
	fmt.Println()

	// Print 5 placeholder lines for progress
	for i := 0; i < 5; i++ {
		fmt.Println()
	}

	lastRender := time.Now()
	err = cl.Download(ctx, paths, func(p torrent.Progress) {
		if time.Since(lastRender) < 200*time.Millisecond {
			return
		}
		lastRender = time.Now()
		renderDownloadProgress(p)
	})

	fmt.Println()
	if err != nil {
		return err
	}
	fmt.Println(successStyle.Render("  Download complete!"))
	return nil
}

func renderDownloadProgress(p torrent.Progress) {
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

	fmt.Print("\033[5A\033[J")

	fmt.Printf("  Downloading %s\n", infoStyle.Render(p.File))
	if pct >= 100 {
		fmt.Printf("  %s  %s\n", successStyle.Render(bar), successStyle.Render(fmt.Sprintf("%.1f%%", pct)))
	} else {
		fmt.Printf("  %s  %.1f%%\n", infoStyle.Render(bar), pct)
	}
	fmt.Printf("  Speed: %s  Peak: %s  Peers: %d\n",
		infoStyle.Render(formatBytesPerSec(p.Speed)),
		labelStyle.Render(formatBytesPerSec(p.PeakSpeed)),
		p.Peers,
	)
	fmt.Printf("  Downloaded: %s / %s\n",
		infoStyle.Render(formatBytes(p.BytesCompleted)),
		labelStyle.Render(formatBytes(p.BytesTotal)),
	)
	fmt.Printf("  ETA: %s  Elapsed: %s\n",
		infoStyle.Render(formatDuration(p.ETA)),
		labelStyle.Render(formatDuration(p.Elapsed)),
	)
}

// ── shared: file filtering ──────────────────────────────

type fileFilter struct {
	kind   string // "comments" or "submissions"
	year   int
	from   string // "2005-06" inclusive
	to     string // "2020-12" inclusive
	last   int    // most recent N files
	args   []string
	noImported bool // skip already imported (for import command)
}

// matchFiles filters allFiles by the filter criteria and resolves positional args.
func matchFiles(allFiles []reddit.DataFile, f fileFilter) []reddit.DataFile {
	// If positional args given, resolve each one
	if len(f.args) > 0 {
		var result []reddit.DataFile
		for _, arg := range f.args {
			resolved := resolveFileArg(arg, f.kind, allFiles)
			result = append(result, resolved...)
		}
		return result
	}

	// Filter by kind, year, date range
	var filtered []reddit.DataFile
	for _, df := range allFiles {
		if f.kind != "" && string(df.Kind) != f.kind {
			continue
		}
		if f.year > 0 && !yearMatches(df.YearMonth, f.year) {
			continue
		}
		if f.from != "" && df.YearMonth < f.from {
			continue
		}
		if f.to != "" && df.YearMonth > f.to {
			continue
		}
		filtered = append(filtered, df)
	}

	// --last N: take the N most recent (they're sorted by name=date)
	if f.last > 0 && len(filtered) > f.last {
		filtered = filtered[len(filtered)-f.last:]
	}

	return filtered
}

// resolveFileArg resolves a single argument to DataFile(s).
// Accepts: "comments/RC_2005-12.zst", "RC_2005-12", "2005-12", etc.
func resolveFileArg(arg string, kindHint string, allFiles []reddit.DataFile) []reddit.DataFile {
	// Try as torrent path: "comments/RC_2005-12.zst"
	if df, ok := reddit.ParseTorrentPath(arg, 0); ok {
		return fillSize([]reddit.DataFile{df}, allFiles)
	}

	// Try without extension: "comments/RC_2005-12"
	if df, ok := reddit.ParseTorrentPath(arg+".zst", 0); ok {
		return fillSize([]reddit.DataFile{df}, allFiles)
	}

	// Try just the name: "RC_2005-12" or "RS_2005-06"
	name := strings.TrimSuffix(arg, ".zst")
	if strings.HasPrefix(name, "RC_") {
		df, _ := reddit.ParseTorrentPath("comments/"+name+".zst", 0)
		return fillSize([]reddit.DataFile{df}, allFiles)
	}
	if strings.HasPrefix(name, "RS_") {
		df, _ := reddit.ParseTorrentPath("submissions/"+name+".zst", 0)
		return fillSize([]reddit.DataFile{df}, allFiles)
	}

	// Bare date: "2005-12" — use kindHint or both
	if looksLikeDate(name) {
		var result []reddit.DataFile
		if kindHint == "" || kindHint == "comments" {
			df, _ := reddit.ParseTorrentPath("comments/RC_"+name+".zst", 0)
			result = append(result, df)
		}
		if kindHint == "" || kindHint == "submissions" {
			df, _ := reddit.ParseTorrentPath("submissions/RS_"+name+".zst", 0)
			result = append(result, df)
		}
		return fillSize(result, allFiles)
	}

	return nil
}

// fillSize fills in the Size field from allFiles for resolved DataFiles.
func fillSize(resolved []reddit.DataFile, allFiles []reddit.DataFile) []reddit.DataFile {
	byName := make(map[string]int64)
	for _, f := range allFiles {
		byName[f.Name] = f.Size
	}
	for i := range resolved {
		if s, ok := byName[resolved[i].Name]; ok {
			resolved[i].Size = s
		}
	}
	return resolved
}

func yearMatches(yearMonth string, year int) bool {
	if len(yearMonth) < 4 {
		return false
	}
	y, err := strconv.Atoi(yearMonth[:4])
	return err == nil && y == year
}

func looksLikeDate(s string) bool {
	// "2005-12" pattern
	if len(s) != 7 || s[4] != '-' {
		return false
	}
	_, err1 := strconv.Atoi(s[:4])
	_, err2 := strconv.Atoi(s[5:])
	return err1 == nil && err2 == nil
}

// ── reddit list ──────────────────────────────────────────────

func newRedditList() *cobra.Command {
	var (
		kind    string
		year    int
		status  string
		noCache bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all files in the Reddit torrent",
		Long: `List all 488 files in the Reddit archive with download/import status.

Uses cached metadata for instant startup. First run fetches from torrent peers.

Examples:
  search reddit list
  search reddit list --kind comments
  search reddit list --year 2005
  search reddit list --status downloaded`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRedditList(cmd.Context(), kind, year, status, noCache)
		},
	}

	cmd.Flags().StringVar(&kind, "kind", "", "Filter: comments, submissions")
	cmd.Flags().IntVar(&year, "year", 0, "Filter by year")
	cmd.Flags().StringVar(&status, "status", "", "Filter: downloaded, imported, pending")
	cmd.Flags().BoolVar(&noCache, "no-cache", false, "Bypass metadata cache (re-fetch from peers)")

	return cmd
}

func runRedditList(ctx context.Context, kindFilter string, yearFilter int, statusFilter string, noCache bool) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Reddit Archive (Pushshift)"))
	fmt.Println()

	allFiles, err := getRedditFiles(ctx, noCache)
	if err != nil {
		return err
	}

	// Filter
	var comments, submissions []reddit.DataFile
	for _, df := range allFiles {
		if kindFilter != "" && string(df.Kind) != kindFilter {
			continue
		}
		if yearFilter > 0 && !yearMatches(df.YearMonth, yearFilter) {
			continue
		}

		downloaded := fileExists(df.ZstPath)
		imported := fileExists(df.DBPath)
		if statusFilter == "downloaded" && !downloaded {
			continue
		}
		if statusFilter == "imported" && !imported {
			continue
		}
		if statusFilter == "pending" && downloaded {
			continue
		}

		if df.Kind == reddit.Comments {
			comments = append(comments, df)
		} else {
			submissions = append(submissions, df)
		}
	}

	var totalSize int64
	var downloadedCount, importedCount int

	if len(comments) > 0 {
		fmt.Println(infoStyle.Render(fmt.Sprintf("  Comments (%d files)", len(comments))))
		fmt.Println(labelStyle.Render("  " + strings.Repeat("─", 60)))
		for _, df := range comments {
			totalSize += df.Size
			printFileStatus(df)
			if fileExists(df.ZstPath) {
				downloadedCount++
			}
			if fileExists(df.DBPath) {
				importedCount++
			}
		}
		fmt.Println()
	}

	if len(submissions) > 0 {
		fmt.Println(infoStyle.Render(fmt.Sprintf("  Submissions (%d files)", len(submissions))))
		fmt.Println(labelStyle.Render("  " + strings.Repeat("─", 60)))
		for _, df := range submissions {
			totalSize += df.Size
			printFileStatus(df)
			if fileExists(df.ZstPath) {
				downloadedCount++
			}
			if fileExists(df.DBPath) {
				importedCount++
			}
		}
		fmt.Println()
	}

	total := len(comments) + len(submissions)
	cached := ""
	if !noCache && reddit.LoadCache() != nil {
		cached = " (cached)"
	}
	fmt.Println(labelStyle.Render(fmt.Sprintf(
		"  Total: %s (%d files, %d downloaded, %d imported)%s",
		formatBytes(totalSize), total, downloadedCount, importedCount, cached,
	)))
	fmt.Println()

	return nil
}

func printFileStatus(df reddit.DataFile) {
	downloaded := fileExists(df.ZstPath)
	imported := fileExists(df.DBPath)

	dlIcon := labelStyle.Render("○")
	if downloaded {
		dlIcon = successStyle.Render("✓")
	}
	imIcon := labelStyle.Render("○")
	if imported {
		imIcon = successStyle.Render("✓")
	}

	name := df.Name + ".zst"
	size := formatBytes(df.Size)

	fmt.Printf("  %s %s  %-18s %8s\n", dlIcon, imIcon, name, size)
}

// ── reddit download ──────────────────────────────────────────

func newRedditDownload() *cobra.Command {
	var (
		kind    string
		year    int
		last    int
		from    string
		to      string
		noCache bool
	)

	cmd := &cobra.Command{
		Use:   "download [file...]",
		Short: "Download files from the Reddit torrent",
		Long: `Download specific files from the Reddit archive torrent.

Files can be specified by name, date, or path:
  RC_2005-12                 Comment file for Dec 2005
  RS_2005-06                 Submission file for Jun 2005
  2005-12                    Both comment+submission for Dec 2005
  comments/RC_2005-12.zst    Full torrent path

Use flags to select ranges:
  --kind comments --year 2005     All 2005 comments
  --last 3                        3 most recent files
  --from 2020-01 --to 2020-06     Jan–Jun 2020

Examples:
  search reddit download RC_2005-12
  search reddit download 2005-12 --kind comments
  search reddit download --kind submissions --year 2005
  search reddit download --last 3 --kind comments
  search reddit download --from 2020-01 --to 2020-12`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRedditDownload(cmd.Context(), args, kind, year, last, from, to, noCache)
		},
	}

	cmd.Flags().StringVar(&kind, "kind", "", "Filter: comments, submissions")
	cmd.Flags().IntVar(&year, "year", 0, "Filter by year")
	cmd.Flags().IntVar(&last, "last", 0, "Download the N most recent files")
	cmd.Flags().StringVar(&from, "from", "", "Start date (YYYY-MM, inclusive)")
	cmd.Flags().StringVar(&to, "to", "", "End date (YYYY-MM, inclusive)")
	cmd.Flags().BoolVar(&noCache, "no-cache", false, "Bypass metadata cache")

	return cmd
}

func runRedditDownload(ctx context.Context, args []string, kind string, year, last int, from, to string, noCache bool) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Reddit Archive Download"))
	fmt.Println()

	if len(args) == 0 && kind == "" && year == 0 && last == 0 && from == "" {
		return fmt.Errorf("specify files to download, or use --kind/--year/--last/--from flags\n\n" +
			"  Examples:\n" +
			"    search reddit download RC_2005-12\n" +
			"    search reddit download --kind comments --year 2005\n" +
			"    search reddit download --last 3")
	}

	allFiles, err := getRedditFiles(ctx, noCache)
	if err != nil {
		return err
	}

	matched := matchFiles(allFiles, fileFilter{
		kind: kind, year: year, from: from, to: to, last: last, args: args,
	})

	if len(matched) == 0 {
		return fmt.Errorf("no matching files found")
	}

	// Skip already downloaded
	var toDownload []reddit.DataFile
	for _, df := range matched {
		if fileExists(df.ZstPath) {
			fmt.Println(labelStyle.Render(fmt.Sprintf("  Skipping %s (already downloaded)", df.Name+".zst")))
		} else {
			toDownload = append(toDownload, df)
		}
	}

	if len(toDownload) == 0 {
		fmt.Println(successStyle.Render("  All requested files already downloaded!"))
		return nil
	}

	return downloadFiles(ctx, toDownload)
}

// ── reddit import ──────────────────────────────────────────

func newRedditImport() *cobra.Command {
	var (
		all       bool
		kind      string
		year      int
		last      int
		from      string
		to        string
		noCache   bool
	)

	cmd := &cobra.Command{
		Use:   "import [file...]",
		Short: "Import files to DuckDB + parquet (auto-downloads if missing)",
		Long: `Import Reddit archive files into DuckDB with parquet export.

If a file hasn't been downloaded yet, it will be downloaded automatically
from the torrent before importing.

Files can be specified by name, date, or path:
  RC_2005-12                 Comment file for Dec 2005
  RS_2005-06                 Submission file for Jun 2005
  2005-12                    Both comment+submission for Dec 2005

Use flags to select ranges:
  --kind comments --year 2005     All 2005 comments
  --last 3 --kind comments        3 most recent comment files
  --from 2020-01 --to 2020-06     Jan–Jun 2020
  --all                           All downloaded files

Examples:
  search reddit import RC_2005-12
  search reddit import 2005-12 --kind comments
  search reddit import --kind comments --year 2005
  search reddit import --all`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRedditImport(cmd.Context(), args, all, kind, year, last, from, to, noCache)
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "Import all downloaded files")
	cmd.Flags().StringVar(&kind, "kind", "", "Filter: comments, submissions")
	cmd.Flags().IntVar(&year, "year", 0, "Filter by year")
	cmd.Flags().IntVar(&last, "last", 0, "Import the N most recent files")
	cmd.Flags().StringVar(&from, "from", "", "Start date (YYYY-MM, inclusive)")
	cmd.Flags().StringVar(&to, "to", "", "End date (YYYY-MM, inclusive)")
	cmd.Flags().BoolVar(&noCache, "no-cache", false, "Bypass metadata cache")

	return cmd
}

func runRedditImport(ctx context.Context, args []string, all bool, kind string, year, last int, from, to string, noCache bool) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Reddit Archive Import"))
	fmt.Println()

	if len(args) == 0 && !all && kind == "" && year == 0 && last == 0 && from == "" {
		return fmt.Errorf("specify files to import, or use --all/--kind/--year/--last/--from flags\n\n" +
			"  Examples:\n" +
			"    search reddit import RC_2005-12\n" +
			"    search reddit import --kind comments --year 2005\n" +
			"    search reddit import --all")
	}

	var files []reddit.DataFile

	if all {
		// --all: import all downloaded files
		files = findDownloadedFiles()
	} else {
		// Resolve from args/flags using metadata
		allFiles, err := getRedditFiles(ctx, noCache)
		if err != nil {
			return err
		}
		files = matchFiles(allFiles, fileFilter{
			kind: kind, year: year, from: from, to: to, last: last, args: args,
		})
	}

	if len(files) == 0 {
		fmt.Println(labelStyle.Render("  No matching files found."))
		return nil
	}

	// Separate into: need download, need import, already done
	var needDownload []reddit.DataFile
	var toImport []reddit.DataFile
	for _, df := range files {
		if fileExists(df.DBPath) && fileExists(df.PQPath) {
			fmt.Println(labelStyle.Render(fmt.Sprintf("  Skipping %s (already imported)", df.Name+".zst")))
			continue
		}
		if !fileExists(df.ZstPath) {
			needDownload = append(needDownload, df)
		}
		toImport = append(toImport, df)
	}

	if len(toImport) == 0 {
		fmt.Println(successStyle.Render("  All files already imported!"))
		return nil
	}

	// Auto-download missing files
	if len(needDownload) > 0 {
		fmt.Println(infoStyle.Render(fmt.Sprintf(
			"  %d file(s) not downloaded yet — downloading first...",
			len(needDownload))))
		fmt.Println()

		if err := downloadFiles(ctx, needDownload); err != nil {
			return fmt.Errorf("auto-download failed: %w", err)
		}
		fmt.Println()
	}

	// Import
	fmt.Println(infoStyle.Render(fmt.Sprintf("  Importing %d file(s)", len(toImport))))
	fmt.Println()

	for i, df := range toImport {
		if !fileExists(df.ZstPath) {
			fmt.Println(warningStyle.Render(fmt.Sprintf("  [%d/%d] Skipping %s (download failed)", i+1, len(toImport), df.Name)))
			continue
		}

		if fileExists(df.DBPath) && fileExists(df.PQPath) {
			fmt.Println(labelStyle.Render(fmt.Sprintf("  [%d/%d] Skipping %s (already imported)", i+1, len(toImport), df.Name)))
			continue
		}

		fmt.Printf("  [%d/%d] %s\n", i+1, len(toImport), infoStyle.Render(df.Name+".zst"))

		// Print 3 placeholder lines for progress updates
		fmt.Println()
		fmt.Println()
		fmt.Println()

		err := reddit.Import(ctx, df, func(p reddit.ImportProgress) {
			fmt.Print("\033[3A\033[J")

			phase := labelStyle.Render(p.Phase)
			if p.Done {
				phase = successStyle.Render(p.Phase + " ✓")
			}

			fmt.Printf("    Phase: %s", phase)
			if p.Rows > 0 {
				fmt.Printf("  Rows: %s", infoStyle.Render(formatNumber(p.Rows)))
			}
			if p.Bytes > 0 && !p.Done {
				fmt.Printf("  (%s)", infoStyle.Render(p.Detail))
			}
			fmt.Println()

			if p.Detail != "" && p.Bytes == 0 {
				detail := shortenHome(p.Detail)
				fmt.Printf("    %s\n", labelStyle.Render(detail))
			} else {
				fmt.Println()
			}

			fmt.Printf("    Elapsed: %s\n", labelStyle.Render(formatDuration(p.Elapsed)))
		})

		if err != nil {
			fmt.Println(errorStyle.Render(fmt.Sprintf("    Failed: %v", err)))
			continue
		}

		// Show result sizes
		var dbSize, pqSize int64
		if st, err := os.Stat(df.DBPath); err == nil {
			dbSize = st.Size()
		}
		if st, err := os.Stat(df.PQPath); err == nil {
			pqSize = st.Size()
		}

		fmt.Printf("    DuckDB:  %s (%s)\n",
			labelStyle.Render(shortenHome(df.DBPath)),
			infoStyle.Render(formatBytes(dbSize)))
		fmt.Printf("    Parquet: %s (%s)\n",
			labelStyle.Render(shortenHome(df.PQPath)),
			infoStyle.Render(formatBytes(pqSize)))
		fmt.Println()
	}

	fmt.Println(successStyle.Render("  Import complete!"))
	return nil
}

// ── reddit info ──────────────────────────────────────────

func newRedditInfo() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info <file>",
		Short: "Show statistics for an imported file",
		Long: `Display statistics for an imported Reddit data file.

Accepts flexible file names:
  RC_2005-12    Comment file for Dec 2005
  RS_2005-06    Submission file for Jun 2005
  2005-12       Both (if only one is imported, shows that one)

Examples:
  search reddit info RC_2005-12
  search reddit info RS_2005-06`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRedditInfo(cmd.Context(), args[0])
		},
	}
	return cmd
}

func runRedditInfo(_ context.Context, arg string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Reddit Archive Info"))
	fmt.Println()

	resolved := resolveFileArg(arg, "", nil)
	if len(resolved) == 0 {
		return fmt.Errorf("cannot resolve file: %s", arg)
	}

	// Pick the one that's imported (or first)
	var df *reddit.DataFile
	for i := range resolved {
		if fileExists(resolved[i].DBPath) {
			df = &resolved[i]
			break
		}
	}
	if df == nil {
		names := make([]string, len(resolved))
		for i, r := range resolved {
			names[i] = r.Name
		}
		return fmt.Errorf("not imported yet: %s (run: search reddit import %s)", strings.Join(names, ", "), arg)
	}

	info, err := reddit.GetInfo(*df)
	if err != nil {
		return fmt.Errorf("get info: %w", err)
	}

	fmt.Printf("  %s\n", infoStyle.Render(df.Name))
	fmt.Printf("  Type: %s\n", labelStyle.Render(string(df.Kind)))
	fmt.Printf("  Period: %s\n", labelStyle.Render(df.YearMonth))
	fmt.Println()

	fmt.Println(infoStyle.Render("  Sizes"))
	fmt.Println(labelStyle.Render("  " + strings.Repeat("─", 40)))
	fmt.Printf("  Raw (.zst):    %s\n", formatBytes(info.ZstSize))
	fmt.Printf("  DuckDB:        %s\n", formatBytes(info.DBSize))
	fmt.Printf("  Parquet:       %s\n", formatBytes(info.PQSize))
	fmt.Println()

	fmt.Println(infoStyle.Render("  Data"))
	fmt.Println(labelStyle.Render("  " + strings.Repeat("─", 40)))
	fmt.Printf("  Rows:          %s\n", infoStyle.Render(formatNumber(info.Rows)))
	fmt.Printf("  Columns:       %d\n", info.Columns)
	if info.DateRange[0] != "" {
		fmt.Printf("  Date range:    %s → %s\n", info.DateRange[0], info.DateRange[1])
	}
	fmt.Println()

	if len(info.TopSubreddits) > 0 {
		fmt.Println(infoStyle.Render("  Top Subreddits"))
		fmt.Println(labelStyle.Render("  " + strings.Repeat("─", 40)))
		for i, s := range info.TopSubreddits {
			if i >= 10 {
				break
			}
			fmt.Printf("  %-25s %s\n", s.Name, labelStyle.Render(formatNumber(s.Count)))
		}
		fmt.Println()
	}

	if len(info.TopAuthors) > 0 {
		fmt.Println(infoStyle.Render("  Top Authors"))
		fmt.Println(labelStyle.Render("  " + strings.Repeat("─", 40)))
		for i, a := range info.TopAuthors {
			if i >= 10 {
				break
			}
			fmt.Printf("  %-25s %s\n", a.Name, labelStyle.Render(formatNumber(a.Count)))
		}
		fmt.Println()
	}

	if len(info.ColumnNames) > 0 {
		fmt.Println(infoStyle.Render("  Columns"))
		fmt.Println(labelStyle.Render("  " + strings.Repeat("─", 40)))
		fmt.Printf("  %s\n", labelStyle.Render(strings.Join(info.ColumnNames, ", ")))
		fmt.Println()
	}

	return nil
}

// ── helpers ──────────────────────────────────────────────

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func findDownloadedFiles() []reddit.DataFile {
	var files []reddit.DataFile

	for _, kind := range []reddit.FileKind{reddit.Comments, reddit.Submissions} {
		dir := filepath.Join(reddit.RawDir(), "reddit", string(kind))
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".zst") {
				continue
			}
			path := string(kind) + "/" + e.Name()
			info, _ := e.Info()
			var size int64
			if info != nil {
				size = info.Size()
			}
			if df, ok := reddit.ParseTorrentPath(path, size); ok {
				files = append(files, df)
			}
		}
	}

	return files
}

func shortenHome(path string) string {
	if home, err := os.UserHomeDir(); err == nil {
		return strings.Replace(path, home, "~", 1)
	}
	return path
}

func formatBytesPerSec(bps float64) string {
	switch {
	case bps >= 1<<30:
		return fmt.Sprintf("%.1f GB/s", bps/(1<<30))
	case bps >= 1<<20:
		return fmt.Sprintf("%.1f MB/s", bps/(1<<20))
	case bps >= 1<<10:
		return fmt.Sprintf("%.1f KB/s", bps/(1<<10))
	default:
		return fmt.Sprintf("%.0f B/s", bps)
	}
}

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "—"
	}
	d = d.Round(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh%02dm%02ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm%02ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
