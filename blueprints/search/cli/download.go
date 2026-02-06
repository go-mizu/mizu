package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	fw "github.com/go-mizu/mizu/blueprints/search/pkg/fineweb"
	"github.com/spf13/cobra"
)

// NewDownload creates the download command with subcommands.
func NewDownload() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download",
		Short: "Download datasets from HuggingFace",
		Long: `Download and manage FineWeb-2 dataset files from HuggingFace.

Subcommands:
  langs    List all available languages with size info
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

	cmd.PersistentFlags().Bool("no-cache", false, "Bypass API cache and fetch fresh data")

	cmd.AddCommand(newDownloadLangs())
	cmd.AddCommand(newDownloadFiles())
	cmd.AddCommand(newDownloadInfo())
	cmd.AddCommand(newDownloadGet())

	return cmd
}

// ── cache helpers ──────────────────────────────────────────────

func useCache(cmd *cobra.Command) bool {
	noCache, _ := cmd.Flags().GetBool("no-cache")
	return !noCache
}

// loadOrFetchConfigs returns configs+sizes, using cache when available.
func loadOrFetchConfigs(cmd *cobra.Command, client *fw.Client) ([]fw.DatasetConfig, *fw.DatasetSizeInfo, *fw.Cache, error) {
	ctx := cmd.Context()
	cache := fw.NewCache()

	if useCache(cmd) {
		if cd := cache.Load(); cd != nil && cd.Configs != nil && cd.Sizes != nil {
			return cd.Configs, cd.Sizes, cache, nil
		}
	}

	// Fetch both configs and sizes in sequence (both are single fast API calls)
	fmt.Print(mutedStyle.Render("  Fetching language list..."))
	t0 := time.Now()

	configs, err := client.ListConfigs(ctx)
	if err != nil {
		fmt.Println()
		return nil, nil, cache, fmt.Errorf("listing configs: %w", err)
	}

	sizes, err := client.GetDatasetSize(ctx, "")
	if err != nil {
		fmt.Println()
		return nil, nil, cache, fmt.Errorf("getting sizes: %w", err)
	}

	elapsed := time.Since(t0)
	fmt.Printf("\r\033[K  %s  Fetched in %s\n", successStyle.Render("OK"), elapsed.Round(time.Millisecond))

	// Update cache
	cd := cache.Load()
	if cd == nil {
		cd = &fw.CacheData{}
	}
	cd.Configs = configs
	cd.Sizes = sizes
	_ = cache.Save(cd)

	return configs, sizes, cache, nil
}

// loadOrFetchFiles returns file list for a lang/split, using cache when available.
func loadOrFetchFiles(cmd *cobra.Command, client *fw.Client, lang, split string) ([]fw.FileInfo, error) {
	ctx := cmd.Context()
	cache := fw.NewCache()
	key := lang + "/" + split

	if useCache(cmd) {
		if cd := cache.Load(); cd != nil && cd.Files != nil {
			if files, ok := cd.Files[key]; ok && len(files) > 0 {
				return files, nil
			}
		}
	}

	fmt.Print(mutedStyle.Render("  Fetching file list..."))
	t0 := time.Now()

	files, err := client.ListSplitFiles(ctx, lang, split)
	if err != nil {
		fmt.Println()
		return nil, fmt.Errorf("listing files: %w", err)
	}

	elapsed := time.Since(t0)
	fmt.Printf("\r\033[K  %s  Found %d files in %s\n",
		successStyle.Render("OK"), len(files), elapsed.Round(time.Millisecond))

	// Update cache
	cd := cache.Load()
	if cd == nil {
		cd = &fw.CacheData{}
	}
	if cd.Files == nil {
		cd.Files = make(map[string][]fw.FileInfo)
	}
	cd.Files[key] = files
	_ = cache.Save(cd)

	return files, nil
}

func cacheAgeLabel(cache *fw.Cache) string {
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

type localStatus struct {
	files     int
	totalSize int64
}

func scanLocalDir(lang, split string) localStatus {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, "data", "fineweb-2", lang, split)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return localStatus{}
	}
	var ls localStatus
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

func scanLocalLang(lang string) localStatus {
	var total localStatus
	for _, split := range []string{"train", "test"} {
		s := scanLocalDir(lang, split)
		total.files += s.files
		total.totalSize += s.totalSize
	}
	return total
}

// ── download langs ─────────────────────────────────────────────

func newDownloadLangs() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "langs",
		Short: "List all available languages in FineWeb-2",
		Long: `Fetches the full list of language configs from HuggingFace API.
Shows language code, human name, row count, size, and local download status.
Results are cached for 24 hours (use --no-cache to refresh).

Examples:
  search download langs
  search download langs --search vie
  search download langs --search Latn | head -20`,
		RunE: runDownloadLangs,
	}

	cmd.Flags().String("search", "", "Filter languages by substring match")
	cmd.Flags().Bool("sort-size", false, "Sort by dataset size (largest first)")

	return cmd
}

func runDownloadLangs(cmd *cobra.Command, args []string) error {
	search, _ := cmd.Flags().GetString("search")
	sortSize, _ := cmd.Flags().GetBool("sort-size")

	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("FineWeb-2 — Available Languages"))
	fmt.Println()

	client := fw.NewClient()
	configs, sizes, cache, err := loadOrFetchConfigs(cmd, client)
	if err != nil {
		return err
	}

	// Build lang → splits map
	langSplits := make(map[string][]string)
	for _, c := range configs {
		langSplits[c.Config] = append(langSplits[c.Config], c.Split)
	}

	// Build lang → size map from sizes
	sizeMap := make(map[string]*fw.ConfigSize)
	if sizes != nil {
		for i := range sizes.Configs {
			sizeMap[sizes.Configs[i].Config] = &sizes.Configs[i]
		}
	}

	// Collect languages
	type langRow struct {
		code     string
		name     string
		splits   []string
		rows     int64
		size     int64
		local    localStatus
	}

	var rows []langRow
	searchLower := strings.ToLower(search)

	for lang, splits := range langSplits {
		if search != "" && !strings.Contains(strings.ToLower(lang), searchLower) {
			// Also search by human name
			if l, ok := fw.GetLanguage(lang); ok {
				if !strings.Contains(strings.ToLower(l.Name), searchLower) {
					continue
				}
			} else {
				continue
			}
		}

		sort.Strings(splits)

		var name string
		if l, ok := fw.GetLanguage(lang); ok {
			name = l.Name
		}

		var numRows, numBytes int64
		if cs, ok := sizeMap[lang]; ok {
			numRows = cs.NumRows
			numBytes = cs.NumBytes
		}

		rows = append(rows, langRow{
			code:   lang,
			name:   name,
			splits: splits,
			rows:   numRows,
			size:   numBytes,
			local:  scanLocalLang(lang),
		})
	}

	// Sort
	if sortSize {
		sort.Slice(rows, func(i, j int) bool { return rows[i].size > rows[j].size })
	} else {
		sort.Slice(rows, func(i, j int) bool { return rows[i].code < rows[j].code })
	}

	// Print table
	ageLabel := cacheAgeLabel(cache)
	if ageLabel != "" {
		fmt.Printf("  %s\n\n", mutedStyle.Render(ageLabel[1:]))
	}

	fmt.Printf("  %-4s %-18s %-14s %-7s %10s %10s  %s\n",
		titleStyle.Render("#"),
		titleStyle.Render("Language"),
		titleStyle.Render("Name"),
		titleStyle.Render("Splits"),
		titleStyle.Render("Rows"),
		titleStyle.Render("Size"),
		titleStyle.Render("Local"))
	fmt.Printf("  %-4s %-18s %-14s %-7s %10s %10s  %s\n",
		"──", "──────────────────", "──────────────", "──────", "──────────", "──────────", "──────────────")

	var totalRows, totalSize int64
	var totalLocalFiles int
	var totalLocalSize int64

	for i, r := range rows {
		totalRows += r.rows
		totalSize += r.size
		totalLocalFiles += r.local.files
		totalLocalSize += r.local.totalSize

		localStr := mutedStyle.Render("—")
		if r.local.files > 0 {
			localStr = successStyle.Render(fmt.Sprintf("%d files, %s", r.local.files, formatBytes(r.local.totalSize)))
		}

		nameStr := mutedStyle.Render("—")
		if r.name != "" {
			nameStr = r.name
		}

		rowsStr := mutedStyle.Render("—")
		if r.rows > 0 {
			rowsStr = formatLargeNumber(r.rows)
		}

		sizeStr := mutedStyle.Render("—")
		if r.size > 0 {
			sizeStr = formatBytes(r.size)
		}

		fmt.Printf("  %-4d %-18s %-14s %-7d %10s %10s  %s\n",
			i+1, r.code, nameStr, len(r.splits), rowsStr, sizeStr, localStr)
	}

	// Footer
	fmt.Printf("  %-4s %-18s %-14s %-7s %10s %10s  %s\n",
		"──", "──────────────────", "──────────────", "──────", "──────────", "──────────", "──────────────")

	localSummary := mutedStyle.Render("—")
	if totalLocalFiles > 0 {
		localSummary = successStyle.Render(fmt.Sprintf("%d files, %s", totalLocalFiles, formatBytes(totalLocalSize)))
	}

	totalRowsStr := ""
	if totalRows > 0 {
		totalRowsStr = formatLargeNumber(totalRows)
	}
	totalSizeStr := ""
	if totalSize > 0 {
		totalSizeStr = formatBytes(totalSize)
	}

	fmt.Printf("  %-4s %-18s %-14s %-7d %10s %10s  %s\n",
		"", titleStyle.Render("Total"),
		"",
		len(rows),
		titleStyle.Render(totalRowsStr),
		titleStyle.Render(totalSizeStr),
		localSummary)

	fmt.Println()
	if search != "" {
		fmt.Printf("  %s matching %q: %s\n",
			infoStyle.Render("Found"),
			search,
			titleStyle.Render(fmt.Sprintf("%d", len(rows))))
		fmt.Println()
	}

	fmt.Printf("  %s\n", mutedStyle.Render("Use --no-cache to refresh • --sort-size to sort by size"))
	fmt.Println()

	return nil
}

// ── download info ──────────────────────────────────────────────

func newDownloadInfo() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show dataset size and statistics",
		Long: `Queries HuggingFace API for dataset size information.
Shows total rows, file sizes, per-split breakdown, and local download progress.

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

	client := fw.NewClient()
	cache := fw.NewCache()

	// Try cache for non-lang-specific requests
	var info *fw.DatasetSizeInfo
	if useCache(cmd) && lang == "" {
		if cd := cache.Load(); cd != nil && cd.Sizes != nil {
			info = cd.Sizes
			age := cacheAgeLabel(cache)
			if age != "" {
				fmt.Printf("  %s\n\n", mutedStyle.Render(age[1:]))
			}
		}
	}

	if info == nil {
		fmt.Print(mutedStyle.Render("  Fetching size info from HuggingFace..."))
		t0 := time.Now()

		var err error
		info, err = client.GetDatasetSize(ctx, lang)
		if err != nil {
			fmt.Println()
			return fmt.Errorf("getting dataset size: %w", err)
		}

		elapsed := time.Since(t0)
		fmt.Printf("\r\033[K  %s  Fetched in %s\n\n", successStyle.Render("OK"), elapsed.Round(time.Millisecond))

		// Cache full-dataset sizes
		if lang == "" {
			cd := cache.Load()
			if cd == nil {
				cd = &fw.CacheData{}
			}
			cd.Sizes = info
			_ = cache.Save(cd)
		}
	}

	// Dataset summary
	fmt.Println(titleStyle.Render("  Dataset Summary"))
	fmt.Println()
	fmt.Printf("    %-25s %s\n", "Total rows:", titleStyle.Render(formatLargeNumber(info.TotalRows)))
	fmt.Printf("    %-25s %s\n", "Parquet size:", titleStyle.Render(formatBytes(info.TotalBytes)))
	fmt.Printf("    %-25s %s\n", "In-memory size:", titleStyle.Render(formatBytes(info.TotalBytesMemory)))
	fmt.Printf("    %-25s %s\n", "Configs:", titleStyle.Render(fmt.Sprintf("%d", len(info.Configs))))
	fmt.Println()

	if len(info.Configs) <= 30 {
		fmt.Println(titleStyle.Render("  Per-Language Breakdown"))
		fmt.Println()
		fmt.Printf("    %-24s %12s %12s %12s\n",
			titleStyle.Render("Language"),
			titleStyle.Render("Rows"),
			titleStyle.Render("Size"),
			titleStyle.Render("Memory"))
		fmt.Printf("    %-24s %12s %12s %12s\n",
			"────────────────────────", "────────────", "────────────", "────────────")

		for _, c := range info.Configs {
			fmt.Printf("    %-24s %12s %12s %12s\n",
				c.Config,
				formatLargeNumber(c.NumRows),
				formatBytes(c.NumBytes),
				formatBytes(c.NumBytesMemory))

			for _, s := range c.Splits {
				fmt.Printf("      %-22s %12s %12s %12s\n",
					mutedStyle.Render(s.Split),
					mutedStyle.Render(formatLargeNumber(s.NumRows)),
					mutedStyle.Render(formatBytes(s.NumBytes)),
					mutedStyle.Render(formatBytes(s.NumBytesMemory)))
			}
		}
		fmt.Println()
	}

	// Local download status with percentages
	fmt.Println(titleStyle.Render("  Local Downloads"))
	fmt.Println()

	langFilter := lang
	if langFilter == "" {
		// Show all downloaded languages
		showLocalStatusEnhanced(info)
	} else {
		showLocalStatusForLang(langFilter, info)
	}
	fmt.Println()

	return nil
}

func showLocalStatusEnhanced(sizes *fw.DatasetSizeInfo) {
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, "data", "fineweb-2")

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

	// Build size map for percentage calculation
	sizeMap := make(map[string]int64) // "lang/split" → bytes
	if sizes != nil {
		for _, c := range sizes.Configs {
			for _, s := range c.Splits {
				sizeMap[c.Config+"/"+s.Split] = s.NumBytes
			}
		}
	}

	found := false
	var grandLocalSize int64
	var grandRemoteSize int64
	var grandLocalFiles int

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		langDir := entry.Name()
		for _, split := range []string{"train", "test"} {
			ls := scanLocalDir(langDir, split)
			if ls.files == 0 {
				continue
			}
			found = true
			grandLocalFiles += ls.files
			grandLocalSize += ls.totalSize

			remoteSize := sizeMap[langDir+"/"+split]
			grandRemoteSize += remoteSize

			pctStr := ""
			if remoteSize > 0 {
				pct := float64(ls.totalSize) / float64(remoteSize) * 100
				if pct >= 99.9 {
					pctStr = successStyle.Render(" (100%)")
				} else {
					pctStr = warningStyle.Render(fmt.Sprintf(" (%.1f%%)", pct))
				}
			}

			fmt.Printf("    %s/%s: %s files, %s%s\n",
				langDir, split,
				successStyle.Render(fmt.Sprintf("%d", ls.files)),
				formatBytes(ls.totalSize),
				pctStr)
		}
	}

	if !found {
		fmt.Printf("    %s\n", mutedStyle.Render("No data downloaded yet"))
		fmt.Printf("    %s\n", mutedStyle.Render("Use: search download get --lang vie_Latn"))
		return
	}

	// Grand total
	fmt.Printf("    %s\n", strings.Repeat("─", 45))
	pctTotal := ""
	if grandRemoteSize > 0 {
		pct := float64(grandLocalSize) / float64(grandRemoteSize) * 100
		pctTotal = fmt.Sprintf(" of %s (%.1f%%)", formatBytes(grandRemoteSize), pct)
	}
	fmt.Printf("    Total: %s files, %s%s\n",
		titleStyle.Render(fmt.Sprintf("%d", grandLocalFiles)),
		titleStyle.Render(formatBytes(grandLocalSize)),
		pctTotal)
}

func showLocalStatusForLang(lang string, sizes *fw.DatasetSizeInfo) {
	sizeMap := make(map[string]int64)
	if sizes != nil {
		for _, c := range sizes.Configs {
			if c.Config == lang {
				for _, s := range c.Splits {
					sizeMap[s.Split] = s.NumBytes
				}
			}
		}
	}

	for _, split := range []string{"train", "test"} {
		ls := scanLocalDir(lang, split)
		remoteSize := sizeMap[split]

		if ls.files == 0 {
			fmt.Printf("    %s/%s: %s", lang, split, mutedStyle.Render("not downloaded"))
			if remoteSize > 0 {
				fmt.Printf(" (%s available)", formatBytes(remoteSize))
			}
			fmt.Println()
			continue
		}

		pctStr := ""
		if remoteSize > 0 {
			pct := float64(ls.totalSize) / float64(remoteSize) * 100
			if pct >= 99.9 {
				pctStr = successStyle.Render(" (100%)")
			} else {
				pctStr = warningStyle.Render(fmt.Sprintf(" (%.1f%%)", pct))
			}
		}

		fmt.Printf("    %s/%s: %s files, %s%s\n",
			lang, split,
			successStyle.Render(fmt.Sprintf("%d", ls.files)),
			formatBytes(ls.totalSize),
			pctStr)
	}
}

// ── download files ─────────────────────────────────────────────

func newDownloadFiles() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "files",
		Short: "List parquet files for a language",
		Long: `Lists all parquet files available on HuggingFace for a language and split.
Shows filename, size, percentage of total, and local download status.
Results are cached for 24 hours.

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
	lang, _ := cmd.Flags().GetString("lang")
	split, _ := cmd.Flags().GetString("split")

	// Resolve language name
	langLabel := lang
	if l, ok := fw.GetLanguage(lang); ok {
		langLabel = fmt.Sprintf("%s (%s)", lang, l.Name)
	}

	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render(fmt.Sprintf("FineWeb-2 — Files for %s/%s", langLabel, split)))
	fmt.Println()

	client := fw.NewClient()
	files, err := loadOrFetchFiles(cmd, client, lang, split)
	if err != nil {
		return err
	}

	fmt.Println()

	// Check local status
	home, _ := os.UserHomeDir()
	localDir := filepath.Join(home, "data", "fineweb-2", lang, split)

	// Compute total size
	var totalSize int64
	for _, f := range files {
		totalSize += f.Size
	}

	// Print table
	fmt.Printf("  %-4s %-28s %10s %7s  %s\n",
		titleStyle.Render("#"),
		titleStyle.Render("Filename"),
		titleStyle.Render("Size"),
		titleStyle.Render("% Tot"),
		titleStyle.Render("Status"))
	fmt.Printf("  %-4s %-28s %10s %7s  %s\n",
		"───", "────────────────────────────", "──────────", "──────", "──────────────")

	var downloadedCount int
	var downloadedSize int64
	for i, f := range files {
		pct := float64(0)
		if totalSize > 0 {
			pct = float64(f.Size) / float64(totalSize) * 100
		}

		// Check if locally available
		localPath := filepath.Join(localDir, f.Name)
		status := mutedStyle.Render("—")
		if info, err := os.Stat(localPath); err == nil {
			if info.Size() == f.Size {
				status = successStyle.Render("✓ downloaded")
				downloadedCount++
				downloadedSize += f.Size
			} else {
				status = warningStyle.Render(fmt.Sprintf("⚠ partial (%s)", formatBytes(info.Size())))
			}
		}

		fmt.Printf("  %-4d %-28s %10s %6.1f%%  %s\n",
			i+1, f.Name, formatBytes(f.Size), pct, status)
	}

	// Summary
	fmt.Println()
	fmt.Printf("  %s\n", strings.Repeat("─", 70))
	fmt.Printf("  Total: %s files, %s\n",
		titleStyle.Render(fmt.Sprintf("%d", len(files))),
		titleStyle.Render(formatBytes(totalSize)))

	if downloadedCount > 0 {
		dlPct := float64(downloadedSize) / float64(totalSize) * 100
		fmt.Printf("  Downloaded: %s/%d files (%s / %s = %.1f%%)\n",
			successStyle.Render(fmt.Sprintf("%d", downloadedCount)),
			len(files),
			successStyle.Render(formatBytes(downloadedSize)),
			formatBytes(totalSize),
			dlPct)
	} else {
		fmt.Printf("  Downloaded: %s\n", mutedStyle.Render("none"))
	}

	remaining := len(files) - downloadedCount
	if remaining > 0 {
		fmt.Printf("  Remaining: %d files (%s)\n",
			remaining,
			formatBytes(totalSize-downloadedSize))
		fmt.Printf("\n  %s\n",
			mutedStyle.Render(fmt.Sprintf("Run: search download get --lang %s --split %s", lang, split)))
	}
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

	langLabel := lang
	if l, ok := fw.GetLanguage(lang); ok {
		langLabel = fmt.Sprintf("%s (%s)", lang, l.Name)
	}

	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render(fmt.Sprintf("FineWeb-2 — Downloading %s/%s", langLabel, split)))
	fmt.Println()

	client := fw.NewClient()
	files, err := loadOrFetchFiles(cmd, client, lang, split)
	if err != nil {
		return err
	}

	// Sort by name
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name < files[j].Name
	})

	// Limit shards
	if shards > 0 && shards < len(files) {
		files = files[:shards]
	}

	// Calculate totals
	var totalSize int64
	for _, f := range files {
		totalSize += f.Size
	}

	fmt.Println()

	home, _ := os.UserHomeDir()
	destDir := filepath.Join(home, "data", "fineweb-2", lang, split)

	// First pass: figure out what needs downloading
	var toDownload []fw.FileInfo
	var skippedCount int
	var skippedSize int64
	for _, file := range files {
		destPath := filepath.Join(destDir, file.Name)
		if info, err := os.Stat(destPath); err == nil && info.Size() == file.Size {
			skippedCount++
			skippedSize += file.Size
		} else {
			toDownload = append(toDownload, file)
		}
	}

	needSize := totalSize - skippedSize

	if skippedCount > 0 {
		fmt.Printf("  %s %d/%d files already downloaded (%s)\n",
			successStyle.Render("SKIP"),
			skippedCount, len(files),
			formatBytes(skippedSize))
	}

	if len(toDownload) == 0 {
		fmt.Printf("\n  %s All %d files are already downloaded!\n\n",
			successStyle.Render("✓"),
			len(files))
		fmt.Printf("  Location: %s\n\n", destDir)
		return nil
	}

	fmt.Printf("  %s %d files to download (%s)\n\n",
		infoStyle.Render("GET"),
		len(toDownload),
		formatBytes(needSize))

	// Download
	var totalDownloaded int64
	pipelineStart := time.Now()

	for i, file := range toDownload {
		destPath := filepath.Join(destDir, file.Name)

		// Overall progress header
		overallPct := float64(0)
		if needSize > 0 {
			overallPct = float64(totalDownloaded) / float64(needSize) * 100
		}
		fmt.Printf("  [%d/%d] %s (%s) — overall %s\n",
			i+1, len(toDownload),
			file.Name, formatBytes(file.Size),
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
	avgSpeed := float64(totalDownloaded) / totalElapsed.Seconds()
	fmt.Println()
	fmt.Printf("  %s\n", strings.Repeat("─", 55))
	fmt.Printf("  %s  Downloaded %s in %s (%s/s avg)\n",
		successStyle.Render("✓"),
		titleStyle.Render(formatBytes(totalDownloaded)),
		totalElapsed.Round(time.Second),
		formatBytes(int64(avgSpeed)))
	if skippedCount > 0 {
		fmt.Printf("     Skipped %s files (%s already on disk)\n",
			mutedStyle.Render(fmt.Sprintf("%d", skippedCount)),
			formatBytes(skippedSize))
	}
	fmt.Printf("     Location: %s\n", destDir)
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
