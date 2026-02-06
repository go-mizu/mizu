package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	fw1 "github.com/go-mizu/mizu/blueprints/search/pkg/fw1"
	"github.com/spf13/cobra"
)

// NewFW1 creates the fw1 command with subcommands for FineWeb-1 (English) dataset.
func NewFW1() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fw1",
		Short: "Download FineWeb-1 (English) dataset from HuggingFace",
		Long: `Download and manage FineWeb (v1, English-only) dataset from HuggingFace.
Organized by CommonCrawl dump configs (e.g. CC-MAIN-2024-51).

Subcommands:
  dumps    List available CC dump configs with sizes
  files    List parquet files for a dump
  info     Show dataset size and statistics
  get      Download parquet files with progress

Examples:
  search fw1 dumps
  search fw1 dumps --search 2024
  search fw1 files --dump CC-MAIN-2024-51
  search fw1 get --dump CC-MAIN-2024-51 --last 1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.PersistentFlags().Bool("no-cache", false, "Bypass API cache and fetch fresh data")

	cmd.AddCommand(newFW1Dumps())
	cmd.AddCommand(newFW1Files())
	cmd.AddCommand(newFW1Info())
	cmd.AddCommand(newFW1Get())

	return cmd
}

// ── cache helpers ──────────────────────────────────────────────

func fw1UseCache(cmd *cobra.Command) bool {
	noCache, _ := cmd.Flags().GetBool("no-cache")
	return !noCache
}

func fw1LoadOrFetchDumps(cmd *cobra.Command, client *fw1.Client) ([]fw1.DatasetConfig, *fw1.DatasetSizeInfo, *fw1.Cache, error) {
	ctx := cmd.Context()
	cache := fw1.NewCache()

	if fw1UseCache(cmd) {
		if cd := cache.Load(); cd != nil && cd.Configs != nil && cd.Sizes != nil {
			return cd.Configs, cd.Sizes, cache, nil
		}
	}

	fmt.Print(mutedStyle.Render("  Fetching dump list..."))
	t0 := time.Now()

	configs, err := client.ListDumps(ctx)
	if err != nil {
		fmt.Println()
		return nil, nil, cache, fmt.Errorf("listing dumps: %w", err)
	}

	sizes, err := client.GetDatasetSize(ctx, "")
	if err != nil {
		fmt.Println()
		return nil, nil, cache, fmt.Errorf("getting sizes: %w", err)
	}

	elapsed := time.Since(t0)
	fmt.Printf("\r\033[K  %s  Fetched in %s\n", successStyle.Render("OK"), elapsed.Round(time.Millisecond))

	cd := cache.Load()
	if cd == nil {
		cd = &fw1.CacheData{}
	}
	cd.Configs = configs
	cd.Sizes = sizes
	_ = cache.Save(cd)

	return configs, sizes, cache, nil
}

func fw1LoadOrFetchFiles(cmd *cobra.Command, client *fw1.Client, dump string) ([]fw1.FileInfo, error) {
	ctx := cmd.Context()
	cache := fw1.NewCache()

	if fw1UseCache(cmd) {
		if cd := cache.Load(); cd != nil && cd.Files != nil {
			if files, ok := cd.Files[dump]; ok && len(files) > 0 {
				return files, nil
			}
		}
	}

	fmt.Print(mutedStyle.Render("  Fetching file list (may paginate)..."))
	t0 := time.Now()

	files, err := client.ListFiles(ctx, dump)
	if err != nil {
		fmt.Println()
		return nil, fmt.Errorf("listing files: %w", err)
	}

	elapsed := time.Since(t0)
	fmt.Printf("\r\033[K  %s  Found %d files in %s\n",
		successStyle.Render("OK"), len(files), elapsed.Round(time.Millisecond))

	cd := cache.Load()
	if cd == nil {
		cd = &fw1.CacheData{}
	}
	if cd.Files == nil {
		cd.Files = make(map[string][]fw1.FileInfo)
	}
	cd.Files[dump] = files
	_ = cache.Save(cd)

	return files, nil
}

func fw1CacheAgeLabel(cache *fw1.Cache) string {
	age := cache.Age()
	if age == 0 {
		return ""
	}
	if age < time.Minute {
		return " (cached just now)"
	}
	if age < time.Hour {
		return fmt.Sprintf(" (cached %dm ago)", int(age.Minutes()))
	}
	if age < 24*time.Hour {
		return fmt.Sprintf(" (cached %dh ago)", int(age.Hours()))
	}
	return fmt.Sprintf(" (cached %dd ago)", int(age.Hours()/24))
}

// ── local disk helpers ─────────────────────────────────────────

type fw1LocalStatus struct {
	files     int
	totalSize int64
}

func fw1ScanLocalDump(dump string) fw1LocalStatus {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, "data", "fineweb-1", dump)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fw1LocalStatus{}
	}
	var ls fw1LocalStatus
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".parquet") {
			ls.files++
			if info, err := e.Info(); err == nil {
				ls.totalSize += info.Size()
			}
		}
	}
	return ls
}

// ── fw1 dumps ──────────────────────────────────────────────────

func newFW1Dumps() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dumps",
		Short: "List available CC dump configs in FineWeb-1",
		Long: `Fetches the full list of CommonCrawl dump configs from HuggingFace API.
Shows dump name, row count, size, and local download status.

Examples:
  search fw1 dumps
  search fw1 dumps --search 2024
  search fw1 dumps --sort-size`,
		RunE: runFW1Dumps,
	}

	cmd.Flags().String("search", "", "Filter dumps by substring match (e.g. '2024')")
	cmd.Flags().Bool("sort-size", false, "Sort by dataset size (largest first)")

	return cmd
}

func runFW1Dumps(cmd *cobra.Command, args []string) error {
	search, _ := cmd.Flags().GetString("search")
	sortSize, _ := cmd.Flags().GetBool("sort-size")

	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("FineWeb-1 (English) — Available Dumps"))
	fmt.Println()

	client := fw1.NewClient()
	configs, sizes, cache, err := fw1LoadOrFetchDumps(cmd, client)
	if err != nil {
		return err
	}

	// Build dump → size map
	sizeMap := make(map[string]*fw1.DumpSize)
	if sizes != nil {
		for i := range sizes.Configs {
			sizeMap[sizes.Configs[i].Config] = &sizes.Configs[i]
		}
	}

	// Deduplicate dump names
	dumpSet := make(map[string]bool)
	for _, c := range configs {
		dumpSet[c.Config] = true
	}

	type dumpRow struct {
		name  string
		rows  int64
		size  int64
		local fw1LocalStatus
	}

	var rows []dumpRow
	searchLower := strings.ToLower(search)

	for dump := range dumpSet {
		if search != "" && !strings.Contains(strings.ToLower(dump), searchLower) {
			continue
		}
		var numRows, numBytes int64
		if ds, ok := sizeMap[dump]; ok {
			numRows = ds.NumRows
			numBytes = ds.NumBytes
		}
		rows = append(rows, dumpRow{
			name:  dump,
			rows:  numRows,
			size:  numBytes,
			local: fw1ScanLocalDump(dump),
		})
	}

	if sortSize {
		sort.Slice(rows, func(i, j int) bool { return rows[i].size > rows[j].size })
	} else {
		sort.Slice(rows, func(i, j int) bool { return rows[i].name < rows[j].name })
	}

	ageLabel := fw1CacheAgeLabel(cache)
	if ageLabel != "" {
		fmt.Printf("  %s\n\n", mutedStyle.Render(ageLabel[1:]))
	}

	fmt.Printf("  %-4s %-22s %12s %12s  %s\n",
		titleStyle.Render("#"),
		titleStyle.Render("Dump"),
		titleStyle.Render("Rows"),
		titleStyle.Render("Size"),
		titleStyle.Render("Local"))
	fmt.Printf("  %-4s %-22s %12s %12s  %s\n",
		"──", "──────────────────────", "────────────", "────────────", "──────────────")

	for i, r := range rows {
		localStr := mutedStyle.Render("—")
		if r.local.files > 0 {
			localStr = successStyle.Render(fmt.Sprintf("%d files, %s", r.local.files, formatBytes(r.local.totalSize)))
		}

		rowsStr := mutedStyle.Render("—")
		if r.rows > 0 {
			rowsStr = formatLargeNumber(r.rows)
		}

		sizeStr := mutedStyle.Render("—")
		if r.size > 0 {
			sizeStr = formatBytes(r.size)
		}

		fmt.Printf("  %-4d %-22s %12s %12s  %s\n",
			i+1, r.name, rowsStr, sizeStr, localStr)
	}

	fmt.Println()
	fmt.Printf("  Total: %s dumps\n", titleStyle.Render(fmt.Sprintf("%d", len(rows))))
	fmt.Printf("  %s\n\n", mutedStyle.Render("Use --no-cache to refresh • --sort-size to sort by size"))

	return nil
}

// ── fw1 info ───────────────────────────────────────────────────

func newFW1Info() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show dataset size and statistics",
		Long: `Queries HuggingFace API for FineWeb-1 dataset size information.

Examples:
  search fw1 info
  search fw1 info --dump CC-MAIN-2024-51`,
		RunE: runFW1Info,
	}

	cmd.Flags().String("dump", "", "CC dump config (empty for full dataset)")

	return cmd
}

func runFW1Info(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	dump, _ := cmd.Flags().GetString("dump")

	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("FineWeb-1 (English) — Dataset Info"))
	fmt.Println()

	client := fw1.NewClient()

	fmt.Print(mutedStyle.Render("  Fetching size info..."))
	t0 := time.Now()

	info, err := client.GetDatasetSize(ctx, dump)
	if err != nil {
		fmt.Println()
		return fmt.Errorf("getting dataset size: %w", err)
	}

	elapsed := time.Since(t0)
	fmt.Printf("\r\033[K  %s  Fetched in %s\n\n", successStyle.Render("OK"), elapsed.Round(time.Millisecond))

	fmt.Println(titleStyle.Render("  Dataset Summary"))
	fmt.Println()
	fmt.Printf("    %-25s %s\n", "Total rows:", titleStyle.Render(formatLargeNumber(info.TotalRows)))
	fmt.Printf("    %-25s %s\n", "Parquet size:", titleStyle.Render(formatBytes(info.TotalBytes)))
	fmt.Printf("    %-25s %s\n", "In-memory size:", titleStyle.Render(formatBytes(info.TotalBytesMemory)))
	fmt.Printf("    %-25s %s\n", "Configs:", titleStyle.Render(fmt.Sprintf("%d", len(info.Configs))))
	fmt.Println()

	if len(info.Configs) <= 30 {
		fmt.Println(titleStyle.Render("  Per-Dump Breakdown"))
		fmt.Println()
		fmt.Printf("    %-24s %12s %12s\n",
			titleStyle.Render("Dump"), titleStyle.Render("Rows"), titleStyle.Render("Size"))
		fmt.Printf("    %-24s %12s %12s\n",
			"────────────────────────", "────────────", "────────────")

		for _, c := range info.Configs {
			fmt.Printf("    %-24s %12s %12s\n",
				c.Config, formatLargeNumber(c.NumRows), formatBytes(c.NumBytes))
		}
		fmt.Println()
	}

	return nil
}

// ── fw1 files ──────────────────────────────────────────────────

func newFW1Files() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "files",
		Short: "List parquet files for a CC dump",
		Long: `Lists all parquet files available on HuggingFace for a CC dump.

Examples:
  search fw1 files --dump CC-MAIN-2024-51
  search fw1 files --dump CC-MAIN-2023-50`,
		RunE: runFW1Files,
	}

	cmd.Flags().String("dump", "", "CC dump config name (required)")

	return cmd
}

func runFW1Files(cmd *cobra.Command, args []string) error {
	dump, _ := cmd.Flags().GetString("dump")
	if dump == "" {
		return fmt.Errorf("--dump is required (e.g. CC-MAIN-2024-51)")
	}

	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render(fmt.Sprintf("FineWeb-1 — Files for %s", dump)))
	fmt.Println()

	client := fw1.NewClient()
	files, err := fw1LoadOrFetchFiles(cmd, client, dump)
	if err != nil {
		return err
	}

	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })

	fmt.Println()

	home, _ := os.UserHomeDir()
	localDir := filepath.Join(home, "data", "fineweb-1", dump)

	var totalSize int64
	for _, f := range files {
		totalSize += f.Size
	}

	fmt.Printf("  %-4s %-28s %10s %7s  %s\n",
		titleStyle.Render("#"),
		titleStyle.Render("Filename"),
		titleStyle.Render("Size"),
		titleStyle.Render("% Tot"),
		titleStyle.Render("Status"))
	fmt.Printf("  %-4s %-28s %10s %7s  %s\n",
		"───", "────────────────────────────", "──────────", "──────", "──────────────")

	var dlCount int
	var dlSize int64
	for i, f := range files {
		pct := float64(0)
		if totalSize > 0 {
			pct = float64(f.Size) / float64(totalSize) * 100
		}

		localPath := filepath.Join(localDir, f.Name)
		status := mutedStyle.Render("—")
		if info, err := os.Stat(localPath); err == nil {
			if info.Size() == f.Size {
				status = successStyle.Render("✓ downloaded")
				dlCount++
				dlSize += f.Size
			} else {
				status = warningStyle.Render(fmt.Sprintf("⚠ partial (%s)", formatBytes(info.Size())))
			}
		}

		fmt.Printf("  %-4d %-28s %10s %6.1f%%  %s\n",
			i+1, f.Name, formatBytes(f.Size), pct, status)
	}

	fmt.Println()
	fmt.Printf("  Total: %s files, %s\n",
		titleStyle.Render(fmt.Sprintf("%d", len(files))),
		titleStyle.Render(formatBytes(totalSize)))
	if dlCount > 0 {
		fmt.Printf("  Downloaded: %s/%d files (%s)\n",
			successStyle.Render(fmt.Sprintf("%d", dlCount)),
			len(files), successStyle.Render(formatBytes(dlSize)))
	}
	fmt.Println()

	return nil
}

// ── fw1 get ────────────────────────────────────────────────────

func newFW1Get() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Download parquet files with progress bar",
		Long: `Downloads FineWeb-1 parquet files from HuggingFace.

By default downloads the last file (highest index = newest data) from the dump.
After download, automatically imports to DuckDB with derived columns for recrawling.

Examples:
  search fw1 get --dump CC-MAIN-2024-51
  search fw1 get --dump CC-MAIN-2024-51 --last 3
  search fw1 get --dump CC-MAIN-2024-51 --file 004_00049.parquet
  search fw1 get --dump CC-MAIN-2024-51 --shards 5
  search fw1 get --dump CC-MAIN-2024-51 --no-import`,
		RunE: runFW1Get,
	}

	cmd.Flags().String("dump", "", "CC dump config name (required)")
	cmd.Flags().Int("shards", 0, "Max files from start (0 = default: last 1)")
	cmd.Flags().Int("last", 1, "Download last N files (highest index = newest)")
	cmd.Flags().String("file", "", "Download specific file by name")
	cmd.Flags().Bool("no-import", false, "Skip auto-import to DuckDB")

	return cmd
}

func runFW1Get(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	dump, _ := cmd.Flags().GetString("dump")
	shards, _ := cmd.Flags().GetInt("shards")
	last, _ := cmd.Flags().GetInt("last")
	fileFilter, _ := cmd.Flags().GetString("file")
	noImport, _ := cmd.Flags().GetBool("no-import")

	if dump == "" {
		return fmt.Errorf("--dump is required (e.g. CC-MAIN-2024-51)")
	}

	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render(fmt.Sprintf("FineWeb-1 — Downloading %s", dump)))
	fmt.Println()

	client := fw1.NewClient()
	files, err := fw1LoadOrFetchFiles(cmd, client, dump)
	if err != nil {
		return err
	}

	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })

	// Filter
	if fileFilter != "" {
		var filtered []fw1.FileInfo
		for _, f := range files {
			if f.Name == fileFilter {
				filtered = append(filtered, f)
				break
			}
		}
		if len(filtered) == 0 {
			return fmt.Errorf("file %q not found in %s (%d files available)", fileFilter, dump, len(files))
		}
		files = filtered
	} else if shards > 0 && shards < len(files) {
		files = files[:shards]
	} else if last > 0 && last < len(files) {
		files = files[len(files)-last:]
	}

	var totalSize int64
	for _, f := range files {
		totalSize += f.Size
	}

	fmt.Println()

	home, _ := os.UserHomeDir()
	destDir := filepath.Join(home, "data", "fineweb-1", dump)

	// Scan local state
	var toDownload []fw1.FileInfo
	var skippedCount int
	var skippedSize int64
	var importOnlyFiles []fw1.FileInfo

	for _, file := range files {
		destPath := filepath.Join(destDir, file.Name)
		if info, err := os.Stat(destPath); err == nil && info.Size() == file.Size {
			skippedCount++
			skippedSize += file.Size
			if !noImport {
				dbPath := destPath + ".duckdb"
				if _, err := os.Stat(dbPath); os.IsNotExist(err) {
					importOnlyFiles = append(importOnlyFiles, file)
				}
			}
		} else {
			toDownload = append(toDownload, file)
		}
	}

	needSize := totalSize - skippedSize

	if skippedCount > 0 {
		fmt.Printf("  %s %d/%d files already downloaded (%s)\n",
			successStyle.Render("SKIP"),
			skippedCount, len(files), formatBytes(skippedSize))
	}

	// Import-only phase
	if len(importOnlyFiles) > 0 {
		fmt.Printf("  %s %d files need DuckDB import\n",
			infoStyle.Render("IMPORT"), len(importOnlyFiles))
		for _, file := range importOnlyFiles {
			destPath := filepath.Join(destDir, file.Name)
			dbPath := destPath + ".duckdb"
			fmt.Printf("    %s", mutedStyle.Render(fmt.Sprintf("Importing %s...", file.Name)))
			rows, dur, importErr := fw1.ImportParquetToDuckDB(destPath, dbPath)
			if importErr != nil {
				fmt.Printf("\r\033[K    %s %s: %v\n", warningStyle.Render("WARN"), file.Name, importErr)
			} else {
				fmt.Printf("\r\033[K    %s %s → %s rows (%s)\n",
					successStyle.Render("DB"),
					filepath.Base(dbPath),
					formatLargeNumber(rows), dur.Round(time.Millisecond))
			}
		}
		fmt.Println()
	}

	if len(toDownload) == 0 {
		fmt.Printf("\n  %s All %d files are already downloaded!\n\n",
			successStyle.Render("✓"), len(files))
		fmt.Printf("  Location: %s\n\n", destDir)
		return nil
	}

	fmt.Printf("  %s %d files to download (%s)\n\n",
		infoStyle.Render("GET"), len(toDownload), formatBytes(needSize))

	var totalDownloaded int64
	pipelineStart := time.Now()

	for i, file := range toDownload {
		destPath := filepath.Join(destDir, file.Name)

		overallPct := float64(0)
		if needSize > 0 {
			overallPct = float64(totalDownloaded) / float64(needSize) * 100
		}
		fmt.Printf("  [%d/%d] %s (%s) — overall %s\n",
			i+1, len(toDownload), file.Name, formatBytes(file.Size),
			mutedStyle.Render(fmt.Sprintf("%.0f%%", overallPct)))

		startTime := time.Now()
		var lastPrint time.Time

		progressFn := func(downloaded, total int64) {
			now := time.Now()
			if now.Sub(lastPrint) < 200*time.Millisecond && downloaded < total {
				return
			}
			lastPrint = now

			elapsed := now.Sub(startTime).Seconds()
			var speed float64
			if elapsed > 0 {
				speed = float64(downloaded) / elapsed
			}

			if total > 0 {
				pct := float64(downloaded) / float64(total) * 100
				bar := buildProgressBar(pct, 30)
				eta := formatDownloadETA(total-downloaded, speed)
				fmt.Printf("\r    %s %5.1f%% %s/%s  %s/s  ETA %s    ",
					bar, pct, formatBytes(downloaded), formatBytes(total),
					formatBytes(int64(speed)), eta)
			}
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
			successStyle.Render("OK"), formatBytes(file.Size),
			elapsed.Round(time.Millisecond), formatBytes(int64(avgSpeed)))

		// Auto-import to DuckDB
		if !noImport {
			dbPath := destPath + ".duckdb"
			fmt.Printf("    %s", mutedStyle.Render("Importing to DuckDB..."))
			rows, importDur, importErr := fw1.ImportParquetToDuckDB(destPath, dbPath)
			if importErr != nil {
				fmt.Printf("\r\033[K    %s import: %v\n", warningStyle.Render("WARN"), importErr)
			} else {
				fmt.Printf("\r\033[K    %s %s rows → %s (%s)\n",
					successStyle.Render("DB"), formatLargeNumber(rows),
					filepath.Base(dbPath), importDur.Round(time.Millisecond))
			}
		}
	}

	totalElapsed := time.Since(pipelineStart)
	avgSpeed := float64(totalDownloaded) / totalElapsed.Seconds()
	fmt.Println()
	fmt.Printf("  %s\n", strings.Repeat("─", 55))
	fmt.Printf("  %s  Downloaded %s in %s (%s/s avg)\n",
		successStyle.Render("✓"),
		titleStyle.Render(formatBytes(totalDownloaded)),
		totalElapsed.Round(time.Second),
		formatBytes(int64(avgSpeed)))
	if skippedCount > 0 {
		fmt.Printf("     Skipped %d files (%s already on disk)\n", skippedCount, formatBytes(skippedSize))
	}
	fmt.Printf("     Location: %s\n", destDir)
	fmt.Println()

	return nil
}
