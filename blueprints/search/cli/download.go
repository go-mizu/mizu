package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	fwdownloader "github.com/go-mizu/mizu/blueprints/search/pkg/fineweb"
	"github.com/spf13/cobra"
)

// NewDownload creates the download command with subcommands.
func NewDownload() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download",
		Short: "Download datasets from HuggingFace",
		Long: `Download and manage FineWeb-2 dataset files from HuggingFace.

Subcommands:
  langs    List all available languages
  files    List parquet files for a language
  info     Show dataset size and statistics
  get      Download parquet files with progress

Examples:
  search download langs
  search download langs --search vie
  search download info --lang vie_Latn
  search download files --lang vie_Latn --split train
  search download get --lang vie_Latn --split train --shards 2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newDownloadLangs())
	cmd.AddCommand(newDownloadFiles())
	cmd.AddCommand(newDownloadInfo())
	cmd.AddCommand(newDownloadGet())

	return cmd
}

// ── download langs ─────────────────────────────────────────────

func newDownloadLangs() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "langs",
		Short: "List all available languages in FineWeb-2",
		Long: `Fetches the full list of language configs from HuggingFace API.
Shows language code, split count, and matches against a search filter.

Examples:
  search download langs
  search download langs --search vie
  search download langs --search Latn | head -20`,
		RunE: runDownloadLangs,
	}

	cmd.Flags().String("search", "", "Filter languages by substring match")

	return cmd
}

func runDownloadLangs(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	search, _ := cmd.Flags().GetString("search")

	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("FineWeb-2 — Available Languages"))
	fmt.Println()

	client := fwdownloader.NewClient()

	fmt.Print(mutedStyle.Render("  Fetching language list from HuggingFace..."))
	t0 := time.Now()

	configs, err := client.ListConfigs(ctx)
	if err != nil {
		fmt.Println()
		return fmt.Errorf("listing configs: %w", err)
	}

	elapsed := time.Since(t0)
	fmt.Printf("\r\033[K  %s  Fetched in %s\n\n", successStyle.Render("OK"), elapsed.Round(time.Millisecond))

	// Deduplicate configs to get unique languages
	langSplits := make(map[string][]string)
	for _, c := range configs {
		langSplits[c.Config] = append(langSplits[c.Config], c.Split)
	}

	// Sort language codes
	langs := make([]string, 0, len(langSplits))
	for lang := range langSplits {
		langs = append(langs, lang)
	}
	sort.Strings(langs)

	// Filter
	searchLower := strings.ToLower(search)
	var filtered []string
	for _, lang := range langs {
		if search == "" || strings.Contains(strings.ToLower(lang), searchLower) {
			filtered = append(filtered, lang)
		}
	}

	// Print table header
	fmt.Printf("  %-30s %s\n", titleStyle.Render("Language"), titleStyle.Render("Splits"))
	fmt.Printf("  %-30s %s\n", "─────────────────────────────", "──────────────")

	for _, lang := range filtered {
		splits := langSplits[lang]
		sort.Strings(splits)
		fmt.Printf("  %-30s %s\n", lang, mutedStyle.Render(strings.Join(splits, ", ")))
	}

	fmt.Println()
	if search != "" {
		fmt.Printf("  %s matching %q: %s\n",
			infoStyle.Render("Found"),
			search,
			titleStyle.Render(fmt.Sprintf("%d", len(filtered))))
	} else {
		fmt.Printf("  %s: %s\n",
			infoStyle.Render("Total languages"),
			titleStyle.Render(fmt.Sprintf("%d", len(filtered))))
	}
	fmt.Println()

	return nil
}

// ── download info ──────────────────────────────────────────────

func newDownloadInfo() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show dataset size and statistics",
		Long: `Queries HuggingFace API for dataset size information.
Shows total rows, file sizes, and per-split breakdown.

Examples:
  search download info
  search download info --lang vie_Latn`,
		RunE: runDownloadInfo,
	}

	cmd.Flags().String("lang", "", "Language code to get info for (empty for full dataset)")

	return cmd
}

func runDownloadInfo(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	lang, _ := cmd.Flags().GetString("lang")

	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("FineWeb-2 — Dataset Info"))
	fmt.Println()

	client := fwdownloader.NewClient()

	fmt.Print(mutedStyle.Render("  Fetching size info from HuggingFace..."))
	t0 := time.Now()

	info, err := client.GetDatasetSize(ctx, lang)
	if err != nil {
		fmt.Println()
		return fmt.Errorf("getting dataset size: %w", err)
	}

	elapsed := time.Since(t0)
	fmt.Printf("\r\033[K  %s  Fetched in %s\n\n", successStyle.Render("OK"), elapsed.Round(time.Millisecond))

	// Dataset summary
	fmt.Println(titleStyle.Render("  Dataset Summary"))
	fmt.Println()
	fmt.Printf("    %-25s %s\n", "Total rows:", titleStyle.Render(formatLargeNumber(info.TotalRows)))
	fmt.Printf("    %-25s %s\n", "Parquet size:", titleStyle.Render(formatBytes(info.TotalBytes)))
	fmt.Printf("    %-25s %s\n", "In-memory size:", titleStyle.Render(formatBytes(info.TotalBytesMemory)))
	fmt.Printf("    %-25s %s\n", "Configs:", titleStyle.Render(fmt.Sprintf("%d", len(info.Configs))))
	fmt.Println()

	if len(info.Configs) <= 20 {
		// Per-config breakdown
		fmt.Println(titleStyle.Render("  Per-Language Breakdown"))
		fmt.Println()
		fmt.Printf("    %-25s %12s %12s %12s\n",
			titleStyle.Render("Language"),
			titleStyle.Render("Rows"),
			titleStyle.Render("Size"),
			titleStyle.Render("Memory"))
		fmt.Printf("    %-25s %12s %12s %12s\n",
			"────────────────────────", "────────────", "────────────", "────────────")

		for _, c := range info.Configs {
			fmt.Printf("    %-25s %12s %12s %12s\n",
				c.Config,
				formatLargeNumber(c.NumRows),
				formatBytes(c.NumBytes),
				formatBytes(c.NumBytesMemory))

			for _, s := range c.Splits {
				fmt.Printf("      %-23s %12s %12s %12s\n",
					mutedStyle.Render(s.Split),
					mutedStyle.Render(formatLargeNumber(s.NumRows)),
					mutedStyle.Render(formatBytes(s.NumBytes)),
					mutedStyle.Render(formatBytes(s.NumBytesMemory)))
			}
		}
		fmt.Println()
	}

	// Local download status
	fmt.Println(titleStyle.Render("  Local Downloads"))
	fmt.Println()
	showLocalStatus(lang)
	fmt.Println()

	return nil
}

func showLocalStatus(lang string) {
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, "data", "fineweb-2")

	if lang != "" {
		// Show status for specific language
		for _, split := range []string{"train", "test"} {
			splitDir := filepath.Join(dataDir, lang, split)
			entries, err := os.ReadDir(splitDir)
			if os.IsNotExist(err) {
				fmt.Printf("    %s/%s: %s\n", lang, split, mutedStyle.Render("not downloaded"))
				continue
			}
			if err != nil {
				fmt.Printf("    %s/%s: %s\n", lang, split, errorStyle.Render(err.Error()))
				continue
			}

			var count int
			var totalSize int64
			for _, e := range entries {
				if strings.HasSuffix(e.Name(), ".parquet") {
					count++
					info, _ := e.Info()
					if info != nil {
						totalSize += info.Size()
					}
				}
			}
			fmt.Printf("    %s/%s: %s files, %s\n",
				lang, split,
				successStyle.Render(fmt.Sprintf("%d", count)),
				formatBytes(totalSize))
		}
		return
	}

	// Show all downloaded languages
	entries, err := os.ReadDir(dataDir)
	if os.IsNotExist(err) {
		fmt.Printf("    %s\n", mutedStyle.Render("No data downloaded yet"))
		fmt.Printf("    %s\n", mutedStyle.Render("Use: search download get --lang vie_Latn"))
		return
	}
	if err != nil {
		fmt.Printf("    %s\n", errorStyle.Render(err.Error()))
		return
	}

	found := false
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		langDir := entry.Name()
		for _, split := range []string{"train", "test"} {
			splitDir := filepath.Join(dataDir, langDir, split)
			splitEntries, err := os.ReadDir(splitDir)
			if err != nil {
				continue
			}
			var count int
			var totalSize int64
			for _, e := range splitEntries {
				if strings.HasSuffix(e.Name(), ".parquet") {
					count++
					info, _ := e.Info()
					if info != nil {
						totalSize += info.Size()
					}
				}
			}
			if count > 0 {
				found = true
				fmt.Printf("    %s/%s: %s files, %s\n",
					langDir, split,
					successStyle.Render(fmt.Sprintf("%d", count)),
					formatBytes(totalSize))
			}
		}
	}

	if !found {
		fmt.Printf("    %s\n", mutedStyle.Render("No data downloaded yet"))
		fmt.Printf("    %s\n", mutedStyle.Render("Use: search download get --lang vie_Latn"))
	}
}

// ── download files ─────────────────────────────────────────────

func newDownloadFiles() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "files",
		Short: "List parquet files for a language",
		Long: `Lists all parquet files available on HuggingFace for a language and split.
Shows filename, size, and whether the file is downloaded locally.

Examples:
  search download files --lang vie_Latn
  search download files --lang vie_Latn --split test
  search download files --lang eng_Latn --split train`,
		RunE: runDownloadFiles,
	}

	cmd.Flags().String("lang", "vie_Latn", "Language code")
	cmd.Flags().String("split", "train", "Dataset split (train or test)")

	return cmd
}

func runDownloadFiles(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	lang, _ := cmd.Flags().GetString("lang")
	split, _ := cmd.Flags().GetString("split")

	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render(fmt.Sprintf("FineWeb-2 — Files for %s/%s", lang, split)))
	fmt.Println()

	client := fwdownloader.NewClient()

	fmt.Print(mutedStyle.Render("  Fetching file list..."))
	t0 := time.Now()

	files, err := client.ListSplitFiles(ctx, lang, split)
	if err != nil {
		fmt.Println()
		return fmt.Errorf("listing files: %w", err)
	}

	elapsed := time.Since(t0)
	fmt.Printf("\r\033[K  %s  Found %d files in %s\n\n",
		successStyle.Render("OK"),
		len(files),
		elapsed.Round(time.Millisecond))

	// Check local status
	home, _ := os.UserHomeDir()
	localDir := filepath.Join(home, "data", "fineweb-2", lang, split)

	// Print table
	fmt.Printf("  %-4s %-25s %12s %s\n",
		titleStyle.Render("#"),
		titleStyle.Render("Filename"),
		titleStyle.Render("Size"),
		titleStyle.Render("Status"))
	fmt.Printf("  %-4s %-25s %12s %s\n",
		"───", "─────────────────────────", "────────────", "──────────")

	var totalSize int64
	var downloadedCount int
	for i, f := range files {
		totalSize += f.Size

		// Check if locally available
		localPath := filepath.Join(localDir, f.Name)
		status := mutedStyle.Render("not downloaded")
		if info, err := os.Stat(localPath); err == nil {
			if info.Size() == f.Size {
				status = successStyle.Render("downloaded")
				downloadedCount++
			} else {
				status = warningStyle.Render(fmt.Sprintf("partial (%s)", formatBytes(info.Size())))
			}
		}

		fmt.Printf("  %-4d %-25s %12s %s\n",
			i+1, f.Name, formatBytes(f.Size), status)
	}

	fmt.Println()
	fmt.Printf("  Total: %s files, %s (downloaded: %d/%d)\n",
		titleStyle.Render(fmt.Sprintf("%d", len(files))),
		titleStyle.Render(formatBytes(totalSize)),
		downloadedCount,
		len(files))
	fmt.Println()

	return nil
}

// ── download get ───────────────────────────────────────────────

func newDownloadGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Download parquet files with progress bar",
		Long: `Downloads parquet files from HuggingFace with a progress bar showing
download speed, percentage, and ETA.

Skips already downloaded files. Use --shards to limit the number of
files to download (useful for large languages like English).

Examples:
  search download get --lang vie_Latn
  search download get --lang vie_Latn --split test
  search download get --lang vie_Latn --split train --shards 2
  search download get --lang eng_Latn --split train --shards 1`,
		RunE: runDownloadGet,
	}

	cmd.Flags().String("lang", "vie_Latn", "Language code")
	cmd.Flags().String("split", "train", "Dataset split (train or test)")
	cmd.Flags().Int("shards", 0, "Max number of files to download (0 = all)")

	return cmd
}

func runDownloadGet(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	lang, _ := cmd.Flags().GetString("lang")
	split, _ := cmd.Flags().GetString("split")
	shards, _ := cmd.Flags().GetInt("shards")

	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render(fmt.Sprintf("FineWeb-2 — Downloading %s/%s", lang, split)))
	fmt.Println()

	client := fwdownloader.NewClient()

	fmt.Print(mutedStyle.Render("  Fetching file list..."))
	files, err := client.ListSplitFiles(ctx, lang, split)
	if err != nil {
		fmt.Println()
		return fmt.Errorf("listing files: %w", err)
	}

	// Sort by name
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name < files[j].Name
	})

	// Limit shards
	if shards > 0 && shards < len(files) {
		files = files[:shards]
	}

	fmt.Printf("\r\033[K  %s  Found %d files to download\n\n",
		infoStyle.Render("OK"), len(files))

	home, _ := os.UserHomeDir()
	destDir := filepath.Join(home, "data", "fineweb-2", lang, split)

	var totalDownloaded int64
	var skippedCount int
	pipelineStart := time.Now()

	for i, file := range files {
		destPath := filepath.Join(destDir, file.Name)

		// Check if already downloaded
		if info, err := os.Stat(destPath); err == nil && info.Size() == file.Size {
			skippedCount++
			fmt.Printf("  [%d/%d] %s %s (%s) %s\n",
				i+1, len(files),
				successStyle.Render("SKIP"),
				file.Name,
				formatBytes(file.Size),
				mutedStyle.Render("already downloaded"))
			continue
		}

		fmt.Printf("  [%d/%d] Downloading %s (%s)\n",
			i+1, len(files), file.Name, formatBytes(file.Size))

		startTime := time.Now()
		var lastPrint time.Time
		var lastBytes int64

		progressFn := func(downloaded, total int64) {
			now := time.Now()
			if now.Sub(lastPrint) < 200*time.Millisecond && downloaded < total {
				return // throttle updates
			}
			lastPrint = now

			elapsed := now.Sub(startTime).Seconds()
			var speed float64
			if elapsed > 0 {
				speed = float64(downloaded-lastBytes) / now.Sub(startTime).Seconds()
				// Use overall speed instead
				speed = float64(downloaded) / elapsed
			}

			if total > 0 {
				pct := float64(downloaded) / float64(total) * 100
				bar := buildProgressBar(pct, 30)
				eta := formatDownloadETA(total-downloaded, speed)
				fmt.Printf("\r    %s %5.1f%% %s/%s  %s/s  ETA %s    ",
					bar, pct,
					formatBytes(downloaded),
					formatBytes(total),
					formatBytes(int64(speed)),
					eta)
			} else {
				fmt.Printf("\r    %s  %s/s    ",
					formatBytes(downloaded),
					formatBytes(int64(speed)))
			}
			lastBytes = downloaded
		}

		err := client.DownloadFileWithProgress(ctx, file, destPath, progressFn)
		elapsed := time.Since(startTime)
		avgSpeed := float64(file.Size) / elapsed.Seconds()

		if err != nil {
			fmt.Printf("\r\033[K    %s %v\n", errorStyle.Render("ERROR"), err)
			return fmt.Errorf("downloading %s: %w", file.Name, err)
		}

		totalDownloaded += file.Size
		fmt.Printf("\r\033[K    %s %s in %s (%s/s)\n",
			successStyle.Render("OK"),
			formatBytes(file.Size),
			elapsed.Round(time.Millisecond),
			formatBytes(int64(avgSpeed)))
	}

	totalElapsed := time.Since(pipelineStart)
	fmt.Println()
	fmt.Printf("  %s\n", strings.Repeat("─", 55))
	fmt.Printf("  Downloaded: %s in %s\n",
		titleStyle.Render(formatBytes(totalDownloaded)),
		totalElapsed.Round(time.Second))
	if skippedCount > 0 {
		fmt.Printf("  Skipped:    %s files (already downloaded)\n",
			mutedStyle.Render(fmt.Sprintf("%d", skippedCount)))
	}
	fmt.Printf("  Location:   %s\n", destDir)
	fmt.Println()

	return nil
}

// ── helpers ────────────────────────────────────────────────────

func buildProgressBar(pct float64, width int) string {
	filled := int(pct / 100 * float64(width))
	if filled > width {
		filled = width
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return fmt.Sprintf("[%s]", bar)
}

func formatDownloadETA(remaining int64, speed float64) string {
	if speed <= 0 {
		return "---"
	}
	secs := float64(remaining) / speed
	if secs < 60 {
		return fmt.Sprintf("%.0fs", secs)
	}
	if secs < 3600 {
		m := int(secs) / 60
		s := int(secs) % 60
		return fmt.Sprintf("%dm%02ds", m, s)
	}
	h := int(secs) / 3600
	m := (int(secs) % 3600) / 60
	return fmt.Sprintf("%dh%02dm", h, m)
}

func formatBytes(b int64) string {
	switch {
	case b >= 1<<40:
		return fmt.Sprintf("%.1f TB", float64(b)/(1<<40))
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(b)/(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(b)/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(b)/(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func formatLargeNumber(n int64) string {
	if n >= 1_000_000_000 {
		return fmt.Sprintf("%.2fB", float64(n)/1_000_000_000)
	}
	if n >= 1_000_000 {
		return fmt.Sprintf("%.2fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}
