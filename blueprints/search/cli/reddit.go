package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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
  import     Import downloaded files to DuckDB + parquet
  info       Show statistics for an imported file

Examples:
  search reddit list
  search reddit list --kind comments --year 2005
  search reddit download comments/RC_2005-12.zst
  search reddit download --kind submissions --year 2005
  search reddit import comments/RC_2005-12.zst
  search reddit import --all
  search reddit info comments/RC_2005-12`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newRedditList())
	cmd.AddCommand(newRedditDownload())
	cmd.AddCommand(newRedditImport())
	cmd.AddCommand(newRedditInfo())

	return cmd
}

// ── reddit list ──────────────────────────────────────────────

func newRedditList() *cobra.Command {
	var (
		kind   string
		year   int
		status string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all files in the Reddit torrent",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRedditList(cmd.Context(), kind, year, status)
		},
	}

	cmd.Flags().StringVar(&kind, "kind", "", "Filter by type: comments, submissions")
	cmd.Flags().IntVar(&year, "year", 0, "Filter by year (e.g. 2005)")
	cmd.Flags().StringVar(&status, "status", "", "Filter: downloaded, imported, pending")

	return cmd
}

func runRedditList(ctx context.Context, kindFilter string, yearFilter int, statusFilter string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Reddit Archive (Pushshift)"))
	fmt.Println()

	fmt.Println(labelStyle.Render("  Connecting to torrent peers for metadata..."))

	// Create torrent client to list files
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

	files, err := cl.Files(ctx)
	if err != nil {
		return fmt.Errorf("list torrent files: %w", err)
	}

	// Parse and filter
	var comments, submissions []reddit.DataFile
	for _, f := range files {
		df, ok := reddit.ParseTorrentPath(f.Path, f.Length)
		if !ok {
			continue
		}

		// Apply filters
		if kindFilter != "" && string(df.Kind) != kindFilter {
			continue
		}
		if yearFilter > 0 {
			if len(df.YearMonth) >= 4 {
				y := df.YearMonth[:4]
				if y != fmt.Sprintf("%d", yearFilter) {
					continue
				}
			}
		}

		// Check status
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

	// Clear the "connecting" line
	fmt.Print("\033[1A\033[2K")

	var totalSize int64
	var downloadedCount, importedCount int

	// Display comments
	if len(comments) > 0 && (kindFilter == "" || kindFilter == "comments") {
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

	// Display submissions
	if len(submissions) > 0 && (kindFilter == "" || kindFilter == "submissions") {
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
	fmt.Println(labelStyle.Render(fmt.Sprintf(
		"  Total: %s (%d files, %d downloaded, %d imported)",
		formatBytes(totalSize), total, downloadedCount, importedCount,
	)))
	fmt.Println()

	return nil
}

func printFileStatus(df reddit.DataFile) {
	downloaded := fileExists(df.ZstPath)
	imported := fileExists(df.DBPath)

	dlStatus := labelStyle.Render("○")
	if downloaded {
		dlStatus = successStyle.Render("✓")
	}
	imStatus := labelStyle.Render("○")
	if imported {
		imStatus = successStyle.Render("✓")
	}

	name := df.Name + ".zst"
	size := formatBytes(df.Size)

	fmt.Printf("  %s %s  %-18s %8s  %s downloaded  %s imported\n",
		dlStatus, imStatus, name, size,
		statusWord(downloaded), statusWord(imported))
}

func statusWord(ok bool) string {
	if ok {
		return successStyle.Render("✓")
	}
	return labelStyle.Render("○")
}

// ── reddit download ──────────────────────────────────────────

func newRedditDownload() *cobra.Command {
	var (
		kind string
		year int
	)

	cmd := &cobra.Command{
		Use:   "download [file...]",
		Short: "Download files from the Reddit torrent",
		Long: `Download specific files from the Reddit archive torrent.

Files can be specified by path (e.g. comments/RC_2005-12.zst) or by pattern.
Use --kind and --year flags to filter which files to download.

Examples:
  search reddit download comments/RC_2005-12.zst
  search reddit download submissions/RS_2005-06.zst
  search reddit download --kind comments --year 2005`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRedditDownload(cmd.Context(), args, kind, year)
		},
	}

	cmd.Flags().StringVar(&kind, "kind", "", "Download all files of type: comments, submissions")
	cmd.Flags().IntVar(&year, "year", 0, "Download all files for year")

	return cmd
}

func runRedditDownload(ctx context.Context, args []string, kindFilter string, yearFilter int) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Reddit Archive Download"))
	fmt.Println()

	if len(args) == 0 && kindFilter == "" && yearFilter == 0 {
		return fmt.Errorf("specify files to download, or use --kind/--year flags")
	}

	// Ensure raw directory exists
	os.MkdirAll(filepath.Join(reddit.RawDir(), "comments"), 0o755)
	os.MkdirAll(filepath.Join(reddit.RawDir(), "submissions"), 0o755)

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

	fmt.Println(labelStyle.Render("  Fetching torrent metadata..."))

	files, err := cl.Files(ctx)
	if err != nil {
		return fmt.Errorf("list torrent files: %w", err)
	}

	// Clear status line
	fmt.Print("\033[1A\033[2K")

	// Determine which files to download
	wantPaths := make(map[string]bool)
	for _, a := range args {
		wantPaths[a] = true
	}

	var downloadPaths []string
	var totalSize int64
	for _, f := range files {
		df, ok := reddit.ParseTorrentPath(f.Path, f.Length)
		if !ok {
			continue
		}

		match := false
		if wantPaths[f.Path] {
			match = true
		}
		if kindFilter != "" && string(df.Kind) == kindFilter {
			match = true
		}
		if yearFilter > 0 && len(df.YearMonth) >= 4 {
			y := df.YearMonth[:4]
			if y == fmt.Sprintf("%d", yearFilter) {
				if kindFilter == "" || string(df.Kind) == kindFilter {
					match = true
				}
			}
		}

		if match {
			// Skip already downloaded
			if fileExists(df.ZstPath) {
				fmt.Println(labelStyle.Render(fmt.Sprintf("  Skipping %s (already downloaded)", f.Path)))
				continue
			}
			downloadPaths = append(downloadPaths, f.Path)
			totalSize += f.Length
		}
	}

	if len(downloadPaths) == 0 {
		fmt.Println(successStyle.Render("  All requested files already downloaded!"))
		return nil
	}

	fmt.Println(infoStyle.Render(fmt.Sprintf("  Downloading %d file(s), %s total", len(downloadPaths), formatBytes(totalSize))))
	fmt.Println()

	// Download with progress
	lastRender := time.Now()
	err = cl.Download(ctx, downloadPaths, func(p torrent.Progress) {
		if time.Since(lastRender) < 200*time.Millisecond {
			return
		}
		lastRender = time.Now()

		pct := float64(0)
		if p.BytesTotal > 0 {
			pct = 100.0 * float64(p.BytesCompleted) / float64(p.BytesTotal)
		}

		// Progress bar
		barWidth := 40
		filled := int(pct / 100.0 * float64(barWidth))
		if filled > barWidth {
			filled = barWidth
		}
		bar := strings.Repeat("━", filled) + strings.Repeat("─", barWidth-filled)

		// Move up and clear 5 lines
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
	})

	fmt.Println()
	if err != nil {
		fmt.Println(errorStyle.Render(fmt.Sprintf("  Download failed: %v", err)))
		return err
	}
	fmt.Println(successStyle.Render("  Download complete!"))

	return nil
}

// ── reddit import ──────────────────────────────────────────

func newRedditImport() *cobra.Command {
	var (
		all bool
	)

	cmd := &cobra.Command{
		Use:   "import [file...]",
		Short: "Import downloaded files to DuckDB + parquet",
		Long: `Import zst-compressed ndjson files into DuckDB with parquet export.

DuckDB handles both zstd decompression and ndjson parsing natively,
so no intermediate files are created.

Files can be specified as:
  comments/RC_2005-12.zst    (full torrent path)
  RC_2005-12                 (just the name)
  comments/RC_2005-12        (without extension)

Examples:
  search reddit import comments/RC_2005-12.zst
  search reddit import RS_2005-06
  search reddit import --all`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRedditImport(cmd.Context(), args, all)
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "Import all downloaded files")

	return cmd
}

func runRedditImport(ctx context.Context, args []string, all bool) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Reddit Archive Import"))
	fmt.Println()

	if len(args) == 0 && !all {
		return fmt.Errorf("specify files to import, or use --all")
	}

	// Find files to import
	var files []reddit.DataFile
	if all {
		files = findDownloadedFiles()
	} else {
		for _, arg := range args {
			df := resolveFile(arg)
			if df == nil {
				fmt.Println(warningStyle.Render(fmt.Sprintf("  Cannot resolve: %s", arg)))
				continue
			}
			files = append(files, *df)
		}
	}

	if len(files) == 0 {
		fmt.Println(labelStyle.Render("  No files to import."))
		return nil
	}

	fmt.Println(infoStyle.Render(fmt.Sprintf("  Importing %d file(s)", len(files))))
	fmt.Println()

	for i, df := range files {
		if !fileExists(df.ZstPath) {
			fmt.Println(warningStyle.Render(fmt.Sprintf("  [%d/%d] Skipping %s (not downloaded)", i+1, len(files), df.Name)))
			continue
		}

		if fileExists(df.DBPath) && fileExists(df.PQPath) {
			fmt.Println(labelStyle.Render(fmt.Sprintf("  [%d/%d] Skipping %s (already imported)", i+1, len(files), df.Name)))
			continue
		}

		fmt.Printf("  [%d/%d] %s\n", i+1, len(files), infoStyle.Render(df.Name+".zst"))

		// Print 3 placeholder lines for progress updates
		fmt.Println()
		fmt.Println()
		fmt.Println()

		err := reddit.Import(ctx, df, func(p reddit.ImportProgress) {
			// Move up 3 lines and clear
			fmt.Print("\033[3A\033[J")

			phase := labelStyle.Render(p.Phase)
			if p.Done {
				phase = successStyle.Render(p.Phase + " ✓")
			}

			fmt.Printf("    Phase: %s", phase)
			if p.Rows > 0 {
				fmt.Printf("  Rows: %s", infoStyle.Render(formatNumber(p.Rows)))
			}
			fmt.Println()

			if p.Detail != "" {
				// Shorten home dir
				detail := p.Detail
				if home, err := os.UserHomeDir(); err == nil {
					detail = strings.Replace(detail, home, "~", 1)
				}
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

		// Show file sizes
		var dbSize, pqSize int64
		if st, err := os.Stat(df.DBPath); err == nil {
			dbSize = st.Size()
		}
		if st, err := os.Stat(df.PQPath); err == nil {
			pqSize = st.Size()
		}

		dbPath := df.DBPath
		pqPath := df.PQPath
		if home, err := os.UserHomeDir(); err == nil {
			dbPath = strings.Replace(dbPath, home, "~", 1)
			pqPath = strings.Replace(pqPath, home, "~", 1)
		}

		fmt.Printf("    DuckDB:  %s (%s)\n", labelStyle.Render(dbPath), infoStyle.Render(formatBytes(dbSize)))
		fmt.Printf("    Parquet: %s (%s)\n", labelStyle.Render(pqPath), infoStyle.Render(formatBytes(pqSize)))
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

Examples:
  search reddit info comments/RC_2005-12
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

	df := resolveFile(arg)
	if df == nil {
		return fmt.Errorf("cannot resolve file: %s", arg)
	}

	if !fileExists(df.DBPath) {
		return fmt.Errorf("not imported yet: %s (run: search reddit import %s)", df.Name, arg)
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

func resolveFile(arg string) *reddit.DataFile {
	// Try as torrent path: "comments/RC_2005-12.zst"
	if df, ok := reddit.ParseTorrentPath(arg, 0); ok {
		return &df
	}

	// Try without extension: "comments/RC_2005-12"
	if df, ok := reddit.ParseTorrentPath(arg+".zst", 0); ok {
		return &df
	}

	// Try just the name: "RC_2005-12" or "RS_2005-06"
	name := strings.TrimSuffix(arg, ".zst")
	if strings.HasPrefix(name, "RC_") {
		df, _ := reddit.ParseTorrentPath("comments/"+name+".zst", 0)
		return &df
	}
	if strings.HasPrefix(name, "RS_") {
		df, _ := reddit.ParseTorrentPath("submissions/"+name+".zst", 0)
		return &df
	}

	return nil
}

func findDownloadedFiles() []reddit.DataFile {
	var files []reddit.DataFile

	for _, kind := range []reddit.FileKind{reddit.Comments, reddit.Submissions} {
		dir := filepath.Join(reddit.RawDir(), string(kind))
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

