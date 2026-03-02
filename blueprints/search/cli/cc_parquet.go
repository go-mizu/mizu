package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/cc"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

type ccParquetListLocalStat struct {
	cached bool
	size   int64
}

func newCCParquet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "parquet",
		Short: "List, download, and import columnar-index parquet files",
		Long: `Work with Common Crawl columnar-index parquet files directly.

This command can list parquet files in a crawl dump (defaults: latest crawl, subset=warc),
download specific files or samples, and import them into per-parquet DuckDB
databases with a catalog DuckDB view.`,
		Example: `  search cc parquet list
  search cc parquet list --subset all --limit 12
  search cc parquet download --part 0
  search cc parquet import --limit 5`,
	}

	cmd.AddCommand(newCCParquetList())
	cmd.AddCommand(newCCParquetDownload())
	cmd.AddCommand(newCCParquetImport())
	return cmd
}

func newCCParquetList() *cobra.Command {
	var (
		crawlID   string
		subset    string
		limit     int
		namesOnly bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List parquet files in a Common Crawl dump (manifest)",
		Long: `List parquet files from the Common Crawl columnar-index manifest.

Defaults:
  --crawl  latest crawl (from cache/API)
  --subset warc

Selectors shown in the table:
  Part      p:N   (from part-000NN filename; preferred for --part N downloads)
  Manifest  m:N   (global manifest index; explicit, stable across subset filters)
  Subset#   N     (ordinal within subset; recrawl --file N uses warc subset by default)`,
		Example: `  search cc parquet list
  search cc parquet list --subset all --limit 20
  search cc parquet list --names-only --limit 5
  search cc parquet list --crawl CC-MAIN-2026-08 --subset warc`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCCParquetList(cmd.Context(), crawlID, subset, limit, namesOnly)
		},
	}

	cmd.Flags().StringVar(&crawlID, "crawl", "", "Crawl ID (default: latest cached/latest available)")
	cmd.Flags().StringVar(&subset, "subset", "warc", "Subset filter (default: warc; use 'all' for every subset)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Max rows to display (0=all)")
	cmd.Flags().BoolVar(&namesOnly, "names-only", false, "Print only manifest index + remote path")

	return cmd
}

func runCCParquetList(ctx context.Context, crawlID, subset string, limit int, namesOnly bool) error {
	if !namesOnly {
		fmt.Println(Banner())
		fmt.Println(subtitleStyle.Render("CC Columnar Parquet Manifest"))
		fmt.Println()
	}

	resolvedCrawlID, crawlNote, err := ccResolveCrawlID(ctx, crawlID)
	if err != nil {
		return fmt.Errorf("resolving crawl: %w", err)
	}
	crawlID = resolvedCrawlID
	subset = ccNormalizeParquetSubset(subset)
	if !namesOnly && (crawlNote != "" || subset == "warc") {
		fmt.Println(labelStyle.Render("Using defaults"))
		ccPrintDefaultCrawlResolution(crawlID, crawlNote)
		if subset == "warc" {
			fmt.Println(labelStyle.Render("  Using subset: warc (default)"))
		}
		fmt.Println()
	}

	cfg := cc.DefaultConfig()
	cfg.CrawlID = crawlID
	client := cc.NewClient(cfg.BaseURL, 4)

	if !namesOnly {
		fmt.Println(infoStyle.Render(fmt.Sprintf("Loading parquet manifest for %s...", crawlID)))
	}
	start := time.Now()
	files, err := cc.ListParquetFiles(ctx, client, cfg, cc.ParquetListOptions{Subset: subset})
	if err != nil {
		return err
	}
	manifestElapsed := time.Since(start).Truncate(time.Millisecond)
	if !namesOnly {
		fmt.Println(successStyle.Render(fmt.Sprintf("  Loaded %s entries in %s",
			ccFmtInt64(int64(len(files))), manifestElapsed)))
	}

	if len(files) == 0 {
		if !namesOnly {
			fmt.Println(warningStyle.Render("  No parquet files matched"))
		}
		return nil
	}

	cacheStore := cc.NewCache(cfg.DataDir)
	cacheData := cacheStore.Load()
	if cacheData == nil {
		cacheData = &cc.CacheData{}
	}

	subsetCounts := make(map[string]int)
	subsetLocalCounts := make(map[string]int)
	subsetLocalBytes := make(map[string]int64)
	subsetOrdinals := make([]int, len(files))
	nextSubsetOrdinal := make(map[string]int)
	for _, f := range files {
		key := f.Subset
		if key == "" {
			key = "(none)"
		}
		subsetCounts[key]++
	}
	for i, f := range files {
		key := f.Subset
		subsetOrdinals[i] = nextSubsetOrdinal[key]
		nextSubsetOrdinal[key]++
	}
	localStats := make([]ccParquetListLocalStat, len(files))
	metaByRemote := make(map[string]cc.ParquetMeta, len(files))
	cacheChanged := false
	var localCount int
	var localBytes int64
	now := time.Now()
	for i, f := range files {
		localPath := cc.LocalParquetPathForRemote(cfg, f.RemotePath)
		meta, _ := cacheStore.GetParquetMeta(cacheData, f.RemotePath)
		if st, statErr := os.Stat(localPath); statErr == nil && st.Size() > 0 {
			localStats[i] = ccParquetListLocalStat{cached: true, size: st.Size()}
			localCount++
			localBytes += st.Size()
			key := f.Subset
			if key == "" {
				key = "(none)"
			}
			subsetLocalCounts[key]++
			subsetLocalBytes[key] += st.Size()
			if meta.SizeUpdatedAt.IsZero() || meta.SizeBytes <= 0 {
				meta.SizeBytes = st.Size()
				meta.SizeUpdatedAt = now
				cacheStore.SetParquetMeta(cacheData, f.RemotePath, meta)
				cacheChanged = true
			}
		}
		metaByRemote[f.RemotePath] = meta
	}

	if !namesOnly {
		if err := ccEnsureParquetListMetadata(ctx, client, cfg, files, localStats, cacheStore, cacheData, metaByRemote); err != nil {
			fmt.Println(warningStyle.Render(fmt.Sprintf("  Metadata enrichment warning: %v", err)))
		}
	}
	if cacheChanged {
		_ = ccSaveCacheWithParquetMetaMerge(cacheStore, cacheData)
	}

	type subsetCount struct {
		Subset     string
		Count      int
		LocalCount int
		LocalBytes int64
		TotalSize  int64
		SizeKnown  int
		TotalURLs  int64
		URLsKnown  int
		TotalHosts int64
		HostsKnown int
	}
	var totalSizeBytes int64
	var totalSizeKnown int
	var totalURLs int64
	var totalURLsKnown int
	var totalHosts int64
	var totalHostsKnown int
	subsetSizeBytes := make(map[string]int64, len(subsetCounts))
	subsetSizeKnown := make(map[string]int, len(subsetCounts))
	subsetURLs := make(map[string]int64, len(subsetCounts))
	subsetURLsKnown := make(map[string]int, len(subsetCounts))
	subsetHosts := make(map[string]int64, len(subsetCounts))
	subsetHostsKnown := make(map[string]int, len(subsetCounts))
	for _, f := range files {
		key := f.Subset
		if key == "" {
			key = "(none)"
		}
		meta := metaByRemote[f.RemotePath]
		if !meta.SizeUpdatedAt.IsZero() && meta.SizeBytes > 0 {
			totalSizeBytes += meta.SizeBytes
			totalSizeKnown++
			subsetSizeBytes[key] += meta.SizeBytes
			subsetSizeKnown[key]++
		}
		if !meta.URLCountUpdated.IsZero() {
			totalURLs += meta.URLCount
			totalURLsKnown++
			subsetURLs[key] += meta.URLCount
			subsetURLsKnown[key]++
		}
		if !meta.HostCountUpdated.IsZero() && meta.HostCount >= 0 {
			totalHosts += meta.HostCount
			totalHostsKnown++
			subsetHosts[key] += meta.HostCount
			subsetHostsKnown[key]++
		}
	}
	var pairs []subsetCount
	for k, v := range subsetCounts {
		pairs = append(pairs, subsetCount{
			Subset:     k,
			Count:      v,
			LocalCount: subsetLocalCounts[k],
			LocalBytes: subsetLocalBytes[k],
			TotalSize:  subsetSizeBytes[k],
			SizeKnown:  subsetSizeKnown[k],
			TotalURLs:  subsetURLs[k],
			URLsKnown:  subsetURLsKnown[k],
			TotalHosts: subsetHosts[k],
			HostsKnown: subsetHostsKnown[k],
		})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].Count == pairs[j].Count {
			return pairs[i].Subset < pairs[j].Subset
		}
		return pairs[i].Count > pairs[j].Count
	})
	display := files
	if limit > 0 && limit < len(display) {
		display = display[:limit]
	}

	if !namesOnly {
		fmt.Println()
	}
	if namesOnly {
		for i, f := range display {
			subIdx := subsetOrdinals[i]
			fmt.Printf("p:%d\tm:%d\t%s:%d\t%s\n", ccParquetPartNumberOr(subIdx, f.Filename), f.ManifestIndex, f.Subset, subIdx, f.RemotePath)
		}
	} else {
		subsetLabelForUI := subset
		if subsetLabelForUI == "" {
			subsetLabelForUI = "all"
		}
		displayCount := len(display)
		summaryLeft := ccRenderKVCard("Manifest Summary", [][2]string{
			{"Crawl", infoStyle.Render(crawlID)},
			{"Subset", infoStyle.Render(subsetLabelForUI)},
			{"Files", ccFmtInt64(int64(len(files)))},
			{"Total size", ccFmtKnownTotalBytes(totalSizeBytes, totalSizeKnown, len(files))},
			{"Total URLs", ccFmtKnownTotalInt(totalURLs, totalURLsKnown, len(files))},
			{"Total hosts", ccFmtKnownTotalInt(totalHosts, totalHostsKnown, len(files))},
			{"Showing", ccFmtInt64(int64(displayCount))},
		})
		summaryRight := ccRenderKVCard("Local Cache", [][2]string{
			{"Cached files", fmt.Sprintf("%s (%s)", ccFmtInt64(int64(localCount)), ccPct(int64(localCount), int64(len(files))))},
			{"Local bytes", ccFmtBytes(localBytes)},
			{"Size cache", ccFmtCacheCoverage(totalSizeKnown, len(files))},
			{"URL cache", ccFmtCacheCoverage(totalURLsKnown, len(files))},
			{"Host cache", ccFmtCacheCoverage(totalHostsKnown, len(files))},
			{"Subsets", ccFmtInt64(int64(len(pairs)))},
			{"Manifest load", manifestElapsed.String()},
		})
		fmt.Println(ccRenderTwoCards(summaryLeft, summaryRight))
		fmt.Println()

		subsetRows := make([][]string, 0, len(pairs))
		for _, p := range pairs {
			localCell := ccStatusChip("muted", "none")
			if p.LocalCount > 0 {
				localCell = ccStatusChip("ok", fmt.Sprintf("%s / %s", ccFmtInt64(int64(p.LocalCount)), ccFmtInt64(int64(p.Count))))
			}
			subsetRows = append(subsetRows, []string{
				p.Subset,
				ccFmtInt64(int64(p.Count)),
				ccPct(int64(p.Count), int64(len(files))),
				ccFmtKnownTotalBytes(p.TotalSize, p.SizeKnown, p.Count),
				ccFmtKnownTotalInt(p.TotalURLs, p.URLsKnown, p.Count),
				ccFmtKnownTotalInt(p.TotalHosts, p.HostsKnown, p.Count),
				localCell + " " + labelStyle.Render(ccFmtBytes(p.LocalBytes)),
			})
		}
		fmt.Println(infoStyle.Render("Subset Breakdown"))
		fmt.Println(ccRenderTable([]string{"Subset", "Files", "%", "Total size", "Total URLs", "Total hosts", "Local cache"}, subsetRows, ccTableOptions{
			RightAlignCols: map[int]bool{1: true, 2: true, 3: true, 4: true, 5: true},
		}))
		fmt.Println()

		rows := make([][]string, 0, len(display))
		for i, f := range display {
			subIdx := subsetOrdinals[i]
			localChip := ccStatusChip("muted", "missing")
			if localStats[i].cached {
				localChip = ccStatusChip("ok", "cached")
			}
			subsetLabel := f.Subset
			if subsetLabel == "" {
				subsetLabel = "(none)"
			}
			meta := metaByRemote[f.RemotePath]
			sizeCell := "-"
			if !meta.SizeUpdatedAt.IsZero() && meta.SizeBytes > 0 {
				sizeCell = ccFmtBytes(meta.SizeBytes)
			}
			urlsCell := "-"
			if !meta.URLCountUpdated.IsZero() {
				urlsCell = ccFmtInt64(meta.URLCount)
			}
			hostsCell := "-"
			if !meta.HostCountUpdated.IsZero() {
				if meta.HostCount >= 0 {
					hostsCell = ccFmtInt64(meta.HostCount)
				} else {
					hostsCell = "n/a"
				}
			}
			rows = append(rows, []string{
				fmt.Sprintf("p:%d", ccParquetPartNumberOr(subIdx, f.Filename)),
				fmt.Sprintf("%d", subIdx),
				subsetLabel,
				fmt.Sprintf("m:%d", f.ManifestIndex),
				sizeCell,
				urlsCell,
				hostsCell,
				localChip,
			})
		}
		fmt.Println(infoStyle.Render("Parquet Files"))
		fmt.Println(ccRenderTable(
			[]string{"Part", "Subset#", "Subset", "Manifest", "Size", "URLs", "Hosts", "Local"},
			rows,
			ccTableOptions{RightAlignCols: map[int]bool{1: true, 4: true, 5: true, 6: true}},
		))
	}

	if !namesOnly && len(display) < len(files) {
		fmt.Println()
		fmt.Println(labelStyle.Render(fmt.Sprintf("Showing %d of %d entries (use --limit 0 for all)", len(display), len(files))))
	}
	if !namesOnly {
		fmt.Println()
	}
	hints := []string{
		"`p:N` matches the parquet part number (`part-00000...` → `p:0`) and maps to `cc parquet download --part N`",
		"`cc parquet import --part N` imports the local cached parquet for that part (download first if missing)",
		"`m:N` is the global Manifest selector; download with `cc parquet download --file N` (example row `m:600` → `--file 600`)",
		"`cc recrawl --file p:N` (or plain `N`) uses the warc part index; `m:N` also works",
	}
	if subset == "" || subset == "warc" {
		hints = append([]string{"`cc recrawl --file N` uses the warc `Subset#` column (first warc row is `0`)"}, hints...)
	}
	if !namesOnly {
		fmt.Println(ccRenderHintBox("Selector Cheat Sheet", hints))
	}
	return nil
}

func newCCParquetDownload() *cobra.Command {
	var (
		crawlID string
		subset  string
		fileIdx int
		partIdx int
		sample  int
		all     bool
		workers int
	)

	cmd := &cobra.Command{
		Use:   "download",
		Short: "Download parquet files from the Common Crawl columnar index",
		Long: `Download parquet files listed in cc-index-table.paths.gz.

Modes:
  --part N    Download one parquet by part number (within --subset, default warc)
  --file N    Download one parquet by manifest index (all subsets)
  --sample N  Download N evenly spaced parquet files (after subset filter)
  --all       Download every parquet file (after subset filter)`,
		Example: `  search cc parquet download --part 0
  search cc parquet download -p 0
  search cc parquet download --file 600
  search cc parquet download --subset warc --sample 3
  search cc parquet download --subset all --sample 10
  search cc parquet download --all --subset warc`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCCParquetDownload(cmd.Context(), crawlID, subset, fileIdx, partIdx, sample, all, workers)
		},
	}

	cmd.Flags().StringVar(&crawlID, "crawl", "", "Crawl ID (default: latest cached/latest available)")
	cmd.Flags().StringVar(&subset, "subset", "warc", "Subset filter (default: warc; use 'all' for every subset)")
	cmd.Flags().IntVar(&fileIdx, "file", -1, "Manifest index to download (all-subset manifest index)")
	cmd.Flags().IntVarP(&partIdx, "part", "p", -1, "Parquet part number to download within --subset (preferred, e.g. --part 0)")
	cmd.Flags().IntVar(&sample, "sample", 0, "Download N evenly spaced files (after subset filter)")
	cmd.Flags().BoolVar(&all, "all", false, "Download all files (after subset filter)")
	cmd.Flags().IntVar(&workers, "workers", 10, "Concurrent download workers")

	return cmd
}

func runCCParquetDownload(ctx context.Context, crawlID, subset string, fileIdx, partIdx, sample int, all bool, workers int) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("CC Parquet Download"))
	fmt.Println()

	resolvedCrawlID, crawlNote, err := ccResolveCrawlID(ctx, crawlID)
	if err != nil {
		return fmt.Errorf("resolving crawl: %w", err)
	}
	crawlID = resolvedCrawlID
	subset = ccNormalizeParquetSubset(subset)
	if crawlNote != "" || subset == "warc" {
		fmt.Println(labelStyle.Render("Using defaults"))
		ccPrintDefaultCrawlResolution(crawlID, crawlNote)
		if subset == "warc" {
			fmt.Println(labelStyle.Render("  Using subset: warc (default)"))
		}
		fmt.Println()
	}

	cfg := cc.DefaultConfig()
	cfg.CrawlID = crawlID
	cfg.IndexWorkers = workers
	client := cc.NewClient(cfg.BaseURL, cfg.TransportShards)

	reporter := newCCDownloadReporter()

	if fileIdx >= 0 && partIdx >= 0 {
		return fmt.Errorf("choose only one selector: --part N or --file N")
	}

	if partIdx >= 0 {
		fmt.Println(infoStyle.Render(fmt.Sprintf("Loading %s parquet manifest for %s...", subset, crawlID)))
		manifestStart := time.Now()
		files, err := cc.ListParquetFiles(ctx, client, cfg, cc.ParquetListOptions{Subset: subset})
		if err != nil {
			return err
		}
		fmt.Println(successStyle.Render(fmt.Sprintf("  Manifest ready: %s files (%s)",
			ccFmtInt64(int64(len(files))), time.Since(manifestStart).Truncate(time.Millisecond))))
		if len(files) == 0 {
			if subset == "" {
				return fmt.Errorf("no parquet files matched (all subsets)")
			}
			return fmt.Errorf("no parquet files matched subset=%q", subset)
		}
		if partIdx < 0 || partIdx >= len(files) {
			return fmt.Errorf("part out of range: %d (available: 0..%d for subset=%s)", partIdx, len(files)-1, subset)
		}
		selected := files[partIdx]
		fmt.Println(infoStyle.Render(fmt.Sprintf("Downloading part p:%d (%s)...", partIdx, selected.Filename)))
		fmt.Println(labelStyle.Render(fmt.Sprintf("  Manifest: m:%d", selected.ManifestIndex)))
		fmt.Println(labelStyle.Render(fmt.Sprintf("  Remote: %s", selected.RemotePath)))
		fmt.Println(labelStyle.Render(fmt.Sprintf("  → %s", cfg.IndexDir())))
		start := time.Now()
		if err := cc.DownloadParquetFiles(ctx, client, cfg, []cc.ParquetFile{selected}, workers, reporter.Callback); err != nil {
			return err
		}
		fmt.Println(successStyle.Render(fmt.Sprintf("Download complete in %s", time.Since(start).Truncate(time.Second))))
		fmt.Println(labelStyle.Render(fmt.Sprintf("  Part: p:%d  Manifest: m:%d", partIdx, selected.ManifestIndex)))
		return nil
	}

	if fileIdx >= 0 {
		fmt.Println(infoStyle.Render(fmt.Sprintf("Downloading manifest file #%d for %s...", fileIdx, crawlID)))
		if subset != "" {
			fmt.Println(labelStyle.Render(fmt.Sprintf("  Note: --subset=%s is ignored in --file mode (manifest index is global)", subset)))
		}
		fmt.Println(labelStyle.Render(fmt.Sprintf("  → %s", cfg.IndexDir())))
		start := time.Now()
		localPath, err := cc.DownloadManifestParquetFile(ctx, client, cfg, fileIdx, reporter.Callback)
		if err != nil {
			return err
		}
		fmt.Println(successStyle.Render(fmt.Sprintf("Download complete in %s", time.Since(start).Truncate(time.Second))))
		fmt.Println(labelStyle.Render(fmt.Sprintf("  Local: %s", localPath)))
		return nil
	}

	if !all && sample <= 0 {
		return fmt.Errorf("choose one mode: --part N, --file N, --sample N, or --all")
	}

	fmt.Println(infoStyle.Render(fmt.Sprintf("Loading manifest for %s...", crawlID)))
	manifestStart := time.Now()
	files, err := cc.ListParquetFiles(ctx, client, cfg, cc.ParquetListOptions{Subset: subset})
	if err != nil {
		return err
	}
	fmt.Println(successStyle.Render(fmt.Sprintf("  Manifest ready: %s files (%s)",
		ccFmtInt64(int64(len(files))), time.Since(manifestStart).Truncate(time.Millisecond))))

	if len(files) == 0 {
		if subset == "" {
			return fmt.Errorf("no parquet files matched (all subsets)")
		}
		return fmt.Errorf("no parquet files matched subset=%q", subset)
	}
	selected := files
	if sample > 0 && sample < len(files) {
		selected = sampleParquetSelection(files, sample)
	}

	fmt.Println(infoStyle.Render(fmt.Sprintf("Downloading %s parquet file(s)...", ccFmtInt64(int64(len(selected))))))
	fmt.Println(labelStyle.Render(fmt.Sprintf("  → %s", cfg.IndexDir())))
	start := time.Now()
	if err := cc.DownloadParquetFiles(ctx, client, cfg, selected, workers, reporter.Callback); err != nil {
		return err
	}
	fmt.Println(successStyle.Render(fmt.Sprintf("Download complete in %s", time.Since(start).Truncate(time.Second))))
	return nil
}

func newCCParquetImport() *cobra.Command {
	var (
		crawlID string
		subset  string
		file    string
		partIdx int
		limit   int
	)

	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import local parquet files into per-parquet DuckDB + catalog",
		Long: `Import local parquet files into one DuckDB database per parquet file, then
build a catalog DuckDB at index.duckdb containing metadata tables and a ccindex view.`,
		Example: `  search cc parquet import --part 0
  search cc parquet import -p 0
  search cc parquet import --limit 5
  search cc parquet import --subset all --limit 20
  search cc parquet import --file ~/data/common-crawl/.../part-00000...parquet`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCCParquetImport(cmd.Context(), crawlID, subset, file, partIdx, limit)
		},
	}

	cmd.Flags().StringVar(&crawlID, "crawl", "", "Crawl ID (default: latest cached/latest available)")
	cmd.Flags().StringVar(&subset, "subset", "warc", "Subset filter for local parquet files (default: warc; use 'all' for every subset)")
	cmd.Flags().StringVar(&file, "file", "", "Import a specific local parquet file")
	cmd.Flags().IntVarP(&partIdx, "part", "p", -1, "Import one local parquet by part number within --subset (preferred, e.g. --part 0)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Import only the first N matching local parquet files (0=all)")

	return cmd
}

func runCCParquetImport(ctx context.Context, crawlID, subset, file string, partIdx, limit int) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("CC Parquet Import"))
	fmt.Println()

	resolvedCrawlID, crawlNote, err := ccResolveCrawlID(ctx, crawlID)
	if err != nil {
		return fmt.Errorf("resolving crawl: %w", err)
	}
	crawlID = resolvedCrawlID
	subset = ccNormalizeParquetSubset(subset)
	if crawlNote != "" || subset == "warc" {
		fmt.Println(labelStyle.Render("Using defaults"))
		ccPrintDefaultCrawlResolution(crawlID, crawlNote)
		if subset == "warc" {
			fmt.Println(labelStyle.Render("  Using subset: warc (default)"))
		}
		fmt.Println()
	}

	cfg := cc.DefaultConfig()
	cfg.CrawlID = crawlID

	var parquetPaths []string

	if file != "" && partIdx >= 0 {
		return fmt.Errorf("choose only one selector: --part N or --file <path>")
	}

	if file != "" {
		if _, err := os.Stat(file); err != nil {
			return fmt.Errorf("parquet file not found: %s", file)
		}
		parquetPaths = []string{file}
	} else if partIdx >= 0 {
		fmt.Println(infoStyle.Render(fmt.Sprintf("Resolving local parquet for part p:%d (subset=%s)...", partIdx, subset)))
		start := time.Now()
		client := cc.NewClient(cfg.BaseURL, 4)
		files, err := cc.ListParquetFiles(ctx, client, cfg, cc.ParquetListOptions{Subset: subset})
		if err != nil {
			return err
		}
		fmt.Println(successStyle.Render(fmt.Sprintf("  Manifest ready: %s files (%s)",
			ccFmtInt64(int64(len(files))), time.Since(start).Truncate(time.Millisecond))))
		if len(files) == 0 {
			if subset == "" {
				return fmt.Errorf("no parquet files matched (all subsets)")
			}
			return fmt.Errorf("no parquet files matched subset=%q", subset)
		}
		if partIdx < 0 || partIdx >= len(files) {
			return fmt.Errorf("part out of range: %d (available: 0..%d for subset=%s)", partIdx, len(files)-1, subset)
		}
		selected := files[partIdx]
		localPath := cc.LocalParquetPathForRemote(cfg, selected.RemotePath)
		if _, err := os.Stat(localPath); err != nil {
			return fmt.Errorf("local parquet not found for p:%d (m:%d): %s (download first: `cc parquet download --part %d`)", partIdx, selected.ManifestIndex, localPath, partIdx)
		}
		fmt.Println(labelStyle.Render(fmt.Sprintf("  Part: p:%d  Manifest: m:%d", partIdx, selected.ManifestIndex)))
		fmt.Println(labelStyle.Render(fmt.Sprintf("  Local: %s", localPath)))
		parquetPaths = []string{localPath}
	} else {
		fmt.Println(infoStyle.Render("Scanning local parquet files..."))
		start := time.Now()
		parquetPaths, err = cc.LocalParquetFilesBySubset(cfg, subset)
		if err != nil {
			return err
		}
		fmt.Println(successStyle.Render(fmt.Sprintf("  Found %s parquet files in %s",
			ccFmtInt64(int64(len(parquetPaths))), time.Since(start).Truncate(time.Millisecond))))
	}

	if len(parquetPaths) == 0 {
		return fmt.Errorf("no local parquet files found (crawl=%s subset=%q)", crawlID, subset)
	}

	sort.Strings(parquetPaths)
	if limit > 0 && limit < len(parquetPaths) {
		parquetPaths = parquetPaths[:limit]
	}

	fmt.Println(infoStyle.Render("Importing parquet files into per-file DuckDB databases..."))
	fmt.Println(labelStyle.Render(fmt.Sprintf("  Parquet root: %s", cfg.IndexDir())))
	fmt.Println(labelStyle.Render(fmt.Sprintf("  Shards:      %s", cfg.IndexShardDir())))
	fmt.Println(labelStyle.Render(fmt.Sprintf("  Catalog:     %s", cfg.IndexDBPath())))

	reporter := newCCImportReporter()
	start := time.Now()
	rowCount, err := cc.ImportParquetPathsWithProgress(ctx, cfg, parquetPaths, reporter.Callback)
	if err != nil {
		return err
	}
	fmt.Println(successStyle.Render(fmt.Sprintf("Import complete: %s rows in %s",
		ccFmtInt64(rowCount), time.Since(start).Truncate(time.Second))))
	return nil
}

type ccDownloadReporter struct {
	mu        sync.Mutex
	files     map[string]*ccDownloadFileState
	doneCount int
}

type ccDownloadFileState struct {
	Name      string
	StartedAt time.Time
	LastPrint time.Time
	Bytes     int64
	Total     int64
}

func newCCDownloadReporter() *ccDownloadReporter {
	return &ccDownloadReporter{
		files: make(map[string]*ccDownloadFileState),
	}
}

func (r *ccDownloadReporter) Callback(p cc.DownloadProgress) {
	key := p.RemotePath
	if key == "" {
		key = p.File
	}

	now := time.Now()

	r.mu.Lock()
	defer r.mu.Unlock()

	st, ok := r.files[key]
	if !ok {
		st = &ccDownloadFileState{Name: p.File}
		r.files[key] = st
	}
	if st.Name == "" {
		st.Name = p.File
	}

	if p.Started {
		st.StartedAt = now
		fmt.Printf("  [%d/%d] start  %s\n", p.FileIndex, p.TotalFiles, p.RemotePath)
		return
	}

	if p.BytesReceived > 0 {
		st.Bytes = p.BytesReceived
		st.Total = p.TotalBytes
		if st.StartedAt.IsZero() {
			st.StartedAt = now
		}
		if now.Sub(st.LastPrint) >= 2*time.Second {
			st.LastPrint = now
			fmt.Printf("  [%d/%d] bytes  %s  (%s)\n",
				p.FileIndex, p.TotalFiles, st.Name, fmtProgressBytes(st.Bytes, st.Total))
		}
	}

	if p.Error != nil {
		r.doneCount++
		fmt.Println(warningStyle.Render(fmt.Sprintf("  [%d/%d] error  %s: %v",
			p.FileIndex, p.TotalFiles, p.File, p.Error)))
		return
	}

	if p.Done {
		r.doneCount++
		elapsed := time.Duration(0)
		if !st.StartedAt.IsZero() {
			elapsed = now.Sub(st.StartedAt)
		}
		label := "done"
		if p.Skipped {
			label = "skip"
		}
		sizeText := ""
		if st.Bytes > 0 || p.BytesReceived > 0 {
			b := st.Bytes
			if b == 0 {
				b = p.BytesReceived
			}
			sizeText = " " + ccFmtBytes(b)
		}
		fmt.Printf("  [%d/%d] %-5s %s%s (%s)  total=%d\n",
			p.FileIndex, p.TotalFiles, label, p.File, sizeText, elapsed.Truncate(time.Second), r.doneCount)
	}
}

type ccImportReporter struct {
	mu            sync.Mutex
	lastHeartbeat map[string]time.Time
}

func newCCImportReporter() *ccImportReporter {
	return &ccImportReporter{
		lastHeartbeat: make(map[string]time.Time),
	}
}

func (r *ccImportReporter) Callback(p cc.ImportProgress) {
	r.mu.Lock()
	defer r.mu.Unlock()

	switch p.Stage {
	case "discover":
		fmt.Println(infoStyle.Render("  Discovering local parquet files..."))
	case "start":
		fmt.Printf("  [%d/%d] import  %s\n", p.FileIndex, p.TotalFiles, filepath.Base(p.File))
		fmt.Println(labelStyle.Render(fmt.Sprintf("           %s", p.File)))
	case "heartbeat":
		last := r.lastHeartbeat[p.File]
		if time.Since(last) < 4*time.Second {
			return
		}
		r.lastHeartbeat[p.File] = time.Now()
		fmt.Printf("  [%d/%d] ...     %s (%s)\n",
			p.FileIndex, p.TotalFiles, filepath.Base(p.File), p.Elapsed.Truncate(time.Second))
	case "indexes":
		fmt.Printf("  [%d/%d] index   %s (rows=%s, cols=%d)\n",
			p.FileIndex, p.TotalFiles, filepath.Base(p.File), ccFmtInt64(p.Rows), p.Columns)
	case "file_done":
		fmt.Printf("  [%d/%d] done    %s (rows=%s, cols=%d, %s)\n",
			p.FileIndex, p.TotalFiles, filepath.Base(p.File), ccFmtInt64(p.Rows), p.Columns, p.Elapsed.Truncate(time.Second))
	case "catalog":
		fmt.Println(infoStyle.Render(fmt.Sprintf("  Catalog: %s", p.Message)))
	case "done":
		fmt.Println(successStyle.Render(fmt.Sprintf("  Finalized %d file(s), %s rows (%s)",
			p.TotalFiles, ccFmtInt64(p.Rows), p.Elapsed.Truncate(time.Second))))
	default:
		if p.Message != "" {
			fmt.Println(labelStyle.Render("  " + p.Message))
		}
	}
}

func sampleParquetSelection(files []cc.ParquetFile, sampleSize int) []cc.ParquetFile {
	if sampleSize <= 0 || sampleSize >= len(files) {
		return files
	}
	sampled := make([]cc.ParquetFile, 0, sampleSize)
	step := float64(len(files)) / float64(sampleSize)
	for i := range sampleSize {
		idx := int(float64(i) * step)
		if idx >= len(files) {
			idx = len(files) - 1
		}
		sampled = append(sampled, files[idx])
	}
	return sampled
}

func fmtProgressBytes(received, total int64) string {
	if total > 0 {
		pct := float64(received) / float64(total) * 100
		return fmt.Sprintf("%s / %s (%.1f%%)", ccFmtBytes(received), ccFmtBytes(total), pct)
	}
	return ccFmtBytes(received)
}

func ccFmtBytes(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	units := []string{"B", "KB", "MB", "GB", "TB"}
	val := float64(n)
	i := 0
	for val >= 1024 && i < len(units)-1 {
		val /= 1024
		i++
	}
	return fmt.Sprintf("%.1f %s", val, units[i])
}

func trimMiddle(s string, max int) string {
	if max <= 3 || len(s) <= max {
		return s
	}
	keep := (max - 3) / 2
	return s[:keep] + "..." + s[len(s)-(max-3-keep):]
}

func ccParquetPartNumberOr(fallback int, filename string) int {
	name := strings.TrimSpace(filename)
	if !strings.HasPrefix(name, "part-") {
		return fallback
	}
	rest := strings.TrimPrefix(name, "part-")
	dash := strings.IndexByte(rest, '-')
	if dash <= 0 {
		return fallback
	}
	n, err := strconv.Atoi(rest[:dash])
	if err != nil {
		return fallback
	}
	return n
}

func ccFmtKnownTotalBytes(totalBytes int64, known, total int) string {
	switch {
	case total <= 0:
		return "-"
	case known <= 0:
		return "unknown"
	case known < total:
		return fmt.Sprintf("%s (%d/%d)", ccFmtBytes(totalBytes), known, total)
	default:
		return ccFmtBytes(totalBytes)
	}
}

func ccFmtKnownTotalInt(totalValue int64, known, total int) string {
	switch {
	case total <= 0:
		return "-"
	case known <= 0:
		return "unknown"
	case known < total:
		return fmt.Sprintf("%s (%d/%d)", ccFmtInt64(totalValue), known, total)
	default:
		return ccFmtInt64(totalValue)
	}
}

func ccFmtCacheCoverage(known, total int) string {
	if total <= 0 {
		return "-"
	}
	return fmt.Sprintf("%d/%d (%s)", known, total, ccPct(int64(known), int64(total)))
}

func ccEnsureParquetListMetadata(
	ctx context.Context,
	client *cc.Client,
	cfg cc.Config,
	files []cc.ParquetFile,
	localStats []ccParquetListLocalStat,
	cacheStore *cc.Cache,
	cacheData *cc.CacheData,
	metaByRemote map[string]cc.ParquetMeta,
) error {
	var missingSizes int
	var missingURLs int
	var missingHosts int
	for _, f := range files {
		meta := metaByRemote[f.RemotePath]
		if meta.SizeUpdatedAt.IsZero() || meta.SizeBytes <= 0 {
			missingSizes++
		}
		if meta.URLCountUpdated.IsZero() {
			missingURLs++
		}
		if meta.HostCountUpdated.IsZero() {
			missingHosts++
		}
	}
	if missingSizes == 0 && missingURLs == 0 && missingHosts == 0 {
		return nil
	}

	fmt.Println(infoStyle.Render("Preparing parquet metadata cache (size + URL counts + host counts)..."))
	fmt.Println(labelStyle.Render(fmt.Sprintf("  Missing size metadata: %s", ccFmtInt64(int64(missingSizes)))))
	fmt.Println(labelStyle.Render(fmt.Sprintf("  Missing URL counts:    %s", ccFmtInt64(int64(missingURLs)))))
	fmt.Println(labelStyle.Render(fmt.Sprintf("  Missing host counts:   %s", ccFmtInt64(int64(missingHosts)))))

	var errs []string
	if missingSizes > 0 {
		if err := ccEnrichParquetSizeMetadata(ctx, client, files, cacheStore, cacheData, metaByRemote); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if missingURLs > 0 {
		if err := ccEnrichParquetURLCountMetadata(ctx, cfg, files, localStats, cacheStore, cacheData, metaByRemote); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if missingHosts > 0 {
		if err := ccEnrichParquetHostCountMetadata(ctx, cfg, files, localStats, cacheStore, cacheData, metaByRemote); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("%s", strings.Join(errs, "; "))
}

func ccEnrichParquetSizeMetadata(
	ctx context.Context,
	client *cc.Client,
	files []cc.ParquetFile,
	cacheStore *cc.Cache,
	cacheData *cc.CacheData,
	metaByRemote map[string]cc.ParquetMeta,
) error {
	type item struct {
		RemotePath string
	}
	missing := make([]item, 0, len(files))
	for _, f := range files {
		meta := metaByRemote[f.RemotePath]
		if meta.SizeUpdatedAt.IsZero() || meta.SizeBytes <= 0 {
			missing = append(missing, item{RemotePath: f.RemotePath})
		}
	}
	if len(missing) == 0 {
		return nil
	}

	fmt.Println(infoStyle.Render("Size metadata (HTTP HEAD)..."))
	start := time.Now()
	workers := min(max(runtime.NumCPU()*2, 4), 24)
	fmt.Println(labelStyle.Render(fmt.Sprintf("  Files: %s  workers=%d", ccFmtInt64(int64(len(missing))), workers)))

	var done atomic.Int64
	var okCount atomic.Int64
	var errCount atomic.Int64
	var mu sync.Mutex
	var sampleErrs []string

	stopProgress := make(chan struct{})
	progressDone := make(chan struct{})
	go func() {
		defer close(progressDone)
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stopProgress:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				d := done.Load()
				fmt.Println(labelStyle.Render(fmt.Sprintf(
					"  Size progress: %s/%s (%s)  ok=%s err=%s  ETA %s",
					ccFmtInt64(d), ccFmtInt64(int64(len(missing))), ccPct(d, int64(len(missing))),
					ccFmtInt64(okCount.Load()), ccFmtInt64(errCount.Load()),
					ccFmtETA(start, d, int64(len(missing))),
				)))
			}
		}
	}()

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(workers)
	for _, it := range missing {
		it := it
		g.Go(func() error {
			size, err := client.HeadFileSize(gctx, it.RemotePath)
			if err != nil {
				errCount.Add(1)
				mu.Lock()
				if len(sampleErrs) < 5 {
					sampleErrs = append(sampleErrs, fmt.Sprintf("%s: %v", filepath.Base(it.RemotePath), err))
				}
				mu.Unlock()
				done.Add(1)
				return nil
			}
			now := time.Now()
			mu.Lock()
			meta := metaByRemote[it.RemotePath]
			meta.SizeBytes = size
			meta.SizeUpdatedAt = now
			metaByRemote[it.RemotePath] = meta
			cacheStore.SetParquetMeta(cacheData, it.RemotePath, meta)
			mu.Unlock()
			okCount.Add(1)
			done.Add(1)
			return nil
		})
	}
	_ = g.Wait()
	close(stopProgress)
	<-progressDone
	if saveErr := ccSaveCacheWithParquetMetaMerge(cacheStore, cacheData); saveErr != nil {
		fmt.Println(warningStyle.Render(fmt.Sprintf("  Cache save warning: %v", saveErr)))
	}

	if d := done.Load(); d > 0 {
		fmt.Println(successStyle.Render(fmt.Sprintf(
			"  Size metadata complete: ok=%s err=%s (%s)",
			ccFmtInt64(okCount.Load()), ccFmtInt64(errCount.Load()), time.Since(start).Truncate(time.Second),
		)))
	}
	mu.Lock()
	defer mu.Unlock()
	if len(sampleErrs) > 0 {
		fmt.Println(warningStyle.Render("  Size metadata errors (sample):"))
		for _, s := range sampleErrs {
			fmt.Println(labelStyle.Render("   - " + s))
		}
	}
	if errCount.Load() > 0 {
		return fmt.Errorf("size metadata missing for %d file(s)", errCount.Load())
	}
	return nil
}

func ccEnrichParquetURLCountMetadata(
	ctx context.Context,
	cfg cc.Config,
	files []cc.ParquetFile,
	localStats []ccParquetListLocalStat,
	cacheStore *cc.Cache,
	cacheData *cc.CacheData,
	metaByRemote map[string]cc.ParquetMeta,
) error {
	type countReq struct {
		RemotePath string
		QueryPath  string
		Subset     string
	}
	var localReqs []countReq
	var remoteReqs []countReq
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	for i, f := range files {
		meta := metaByRemote[f.RemotePath]
		if !meta.URLCountUpdated.IsZero() {
			continue
		}
		sub := f.Subset
		if sub == "" {
			sub = "(none)"
		}
		if i < len(localStats) && localStats[i].cached {
			localReqs = append(localReqs, countReq{
				RemotePath: f.RemotePath,
				QueryPath:  cc.LocalParquetPathForRemote(cfg, f.RemotePath),
				Subset:     sub,
			})
		} else {
			remoteReqs = append(remoteReqs, countReq{
				RemotePath: f.RemotePath,
				QueryPath:  baseURL + "/" + f.RemotePath,
				Subset:     sub,
			})
		}
	}
	totalMissing := len(localReqs) + len(remoteReqs)
	if totalMissing == 0 {
		return nil
	}

	fmt.Println(infoStyle.Render("URL count metadata (DuckDB parquet_metadata row counts)..."))
	fmt.Println(labelStyle.Render(fmt.Sprintf("  Targets: %s (%d local, %d remote)", ccFmtInt64(int64(totalMissing)), len(localReqs), len(remoteReqs))))
	start := time.Now()
	var done int
	var successCount int
	var failCount int
	var sampleErrs []string
	saveEvery := 16
	sinceSave := 0

	if len(localReqs) > 0 {
		subsetCounts := make(map[string]int)
		for _, r := range localReqs {
			subsetCounts[r.Subset]++
		}
		fmt.Println(labelStyle.Render(fmt.Sprintf("  Local URL counts: %d file(s), %d subset(s)", len(localReqs), len(subsetCounts))))
		for i, r := range localReqs {
			fmt.Println(labelStyle.Render(fmt.Sprintf("    [local %d/%d] counting %s", i+1, len(localReqs), filepath.Base(r.RemotePath))))
			counts, err := cc.QueryParquetRowCounts(ctx, []cc.ParquetRowCountRequest{{Key: r.RemotePath, Path: r.QueryPath}})
			done++
			if err != nil {
				failCount++
				if len(sampleErrs) < 5 {
					sampleErrs = append(sampleErrs, fmt.Sprintf("%s (local): %v", filepath.Base(r.RemotePath), err))
				}
			} else if n, ok := counts[r.RemotePath]; ok {
				meta := metaByRemote[r.RemotePath]
				meta.URLCount = n
				meta.URLCountUpdated = time.Now()
				metaByRemote[r.RemotePath] = meta
				cacheStore.SetParquetMeta(cacheData, r.RemotePath, meta)
				successCount++
			} else {
				failCount++
				if len(sampleErrs) < 5 {
					sampleErrs = append(sampleErrs, fmt.Sprintf("%s (local): missing row count", filepath.Base(r.RemotePath)))
				}
			}
			sinceSave++
			if sinceSave >= saveEvery {
				sinceSave = 0
				if saveErr := ccSaveCacheWithParquetMetaMerge(cacheStore, cacheData); saveErr != nil {
					fmt.Println(warningStyle.Render(fmt.Sprintf("    Cache save warning: %v", saveErr)))
				}
			}
			fmt.Println(labelStyle.Render(fmt.Sprintf(
				"    URL progress: %s/%s (%s) ok=%s err=%s ETA %s",
				ccFmtInt64(int64(done)), ccFmtInt64(int64(totalMissing)), ccPct(int64(done), int64(totalMissing)),
				ccFmtInt64(int64(successCount)), ccFmtInt64(int64(failCount)),
				ccFmtETA(start, int64(done), int64(totalMissing)),
			)))
		}
	}

	if len(remoteReqs) > 0 {
		subsetCounts := make(map[string]int)
		for _, r := range remoteReqs {
			subsetCounts[r.Subset]++
		}
		workers := min(max(runtime.NumCPU()/2, 2), 8)
		fmt.Println(labelStyle.Render(fmt.Sprintf("  Remote URL counts: %d file(s), %d subset(s), workers=%d", len(remoteReqs), len(subsetCounts), workers)))

		var doneA atomic.Int64
		var okA atomic.Int64
		var errA atomic.Int64
		baseDone := done
		baseOK := successCount
		baseErr := failCount
		var mu sync.Mutex

		stopProgress := make(chan struct{})
		progressDone := make(chan struct{})
		go func() {
			defer close(progressDone)
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-stopProgress:
					return
				case <-ctx.Done():
					return
				case <-ticker.C:
					dv := int64(baseDone) + doneA.Load()
					okv := int64(baseOK) + okA.Load()
					errv := int64(baseErr) + errA.Load()
					fmt.Println(labelStyle.Render(fmt.Sprintf(
						"    URL progress: %s/%s (%s) ok=%s err=%s ETA %s",
						ccFmtInt64(dv), ccFmtInt64(int64(totalMissing)), ccPct(dv, int64(totalMissing)),
						ccFmtInt64(okv), ccFmtInt64(errv), ccFmtETA(start, dv, int64(totalMissing)),
					)))
				}
			}
		}()

		g, gctx := errgroup.WithContext(ctx)
		g.SetLimit(workers)
		for idx, r := range remoteReqs {
			idx, r := idx, r
			g.Go(func() error {
				if idx < 3 || idx%25 == 0 {
					fmt.Println(labelStyle.Render(fmt.Sprintf("    [remote %d/%d] %s", idx+1, len(remoteReqs), filepath.Base(r.RemotePath))))
				}
				counts, err := cc.QueryParquetRowCounts(gctx, []cc.ParquetRowCountRequest{{Key: r.RemotePath, Path: r.QueryPath}})
				now := time.Now()
				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					errA.Add(1)
					if len(sampleErrs) < 5 {
						sampleErrs = append(sampleErrs, fmt.Sprintf("%s (remote): %v", filepath.Base(r.RemotePath), err))
					}
				} else if n, ok := counts[r.RemotePath]; ok {
					meta := metaByRemote[r.RemotePath]
					meta.URLCount = n
					meta.URLCountUpdated = now
					metaByRemote[r.RemotePath] = meta
					cacheStore.SetParquetMeta(cacheData, r.RemotePath, meta)
					okA.Add(1)
				} else {
					errA.Add(1)
					if len(sampleErrs) < 5 {
						sampleErrs = append(sampleErrs, fmt.Sprintf("%s (remote): missing row count", filepath.Base(r.RemotePath)))
					}
				}
				doneA.Add(1)
				sinceSave++
				if sinceSave >= saveEvery {
					sinceSave = 0
					if saveErr := ccSaveCacheWithParquetMetaMerge(cacheStore, cacheData); saveErr != nil {
						fmt.Println(warningStyle.Render(fmt.Sprintf("    Cache save warning: %v", saveErr)))
					}
				}
				return nil
			})
		}
		_ = g.Wait()
		close(stopProgress)
		<-progressDone
		done = baseDone + int(doneA.Load())
		successCount = baseOK + int(okA.Load())
		failCount = baseErr + int(errA.Load())
		fmt.Println(labelStyle.Render(fmt.Sprintf(
			"    URL progress: %s/%s (%s) ok=%s err=%s ETA %s",
			ccFmtInt64(int64(done)), ccFmtInt64(int64(totalMissing)), ccPct(int64(done), int64(totalMissing)),
			ccFmtInt64(int64(successCount)), ccFmtInt64(int64(failCount)),
			ccFmtETA(start, int64(done), int64(totalMissing)),
		)))
	}

	if saveErr := ccSaveCacheWithParquetMetaMerge(cacheStore, cacheData); saveErr != nil {
		fmt.Println(warningStyle.Render(fmt.Sprintf("  Cache save warning: %v", saveErr)))
	}
	fmt.Println(successStyle.Render(fmt.Sprintf(
		"  URL count metadata complete: ok=%s err=%s (%s)",
		ccFmtInt64(int64(successCount)), ccFmtInt64(int64(failCount)), time.Since(start).Truncate(time.Second),
	)))
	if len(sampleErrs) > 0 {
		fmt.Println(warningStyle.Render("  URL count metadata errors (sample):"))
		for _, s := range sampleErrs {
			fmt.Println(labelStyle.Render("   - " + s))
		}
	}
	if failCount > 0 {
		return fmt.Errorf("url count metadata missing for %d file(s)", failCount)
	}
	return nil
}

func ccEnrichParquetHostCountMetadata(
	ctx context.Context,
	cfg cc.Config,
	files []cc.ParquetFile,
	localStats []ccParquetListLocalStat,
	cacheStore *cc.Cache,
	cacheData *cc.CacheData,
	metaByRemote map[string]cc.ParquetMeta,
) error {
	type countReq struct {
		RemotePath string
		QueryPath  string
		Subset     string
	}
	var localReqs []countReq
	var remoteReqs []countReq
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	for i, f := range files {
		meta := metaByRemote[f.RemotePath]
		if !meta.HostCountUpdated.IsZero() {
			continue
		}
		sub := f.Subset
		if sub == "" {
			sub = "(none)"
		}
		if i < len(localStats) && localStats[i].cached {
			localReqs = append(localReqs, countReq{
				RemotePath: f.RemotePath,
				QueryPath:  cc.LocalParquetPathForRemote(cfg, f.RemotePath),
				Subset:     sub,
			})
		} else {
			remoteReqs = append(remoteReqs, countReq{
				RemotePath: f.RemotePath,
				QueryPath:  baseURL + "/" + f.RemotePath,
				Subset:     sub,
			})
		}
	}
	totalMissing := len(localReqs) + len(remoteReqs)
	if totalMissing == 0 {
		return nil
	}

	fmt.Println(infoStyle.Render("Host count metadata (DuckDB COUNT(DISTINCT url_host_name))..."))
	fmt.Println(labelStyle.Render(fmt.Sprintf("  Targets: %s (%d local, %d remote)", ccFmtInt64(int64(totalMissing)), len(localReqs), len(remoteReqs))))
	start := time.Now()
	var done int
	var successCount int
	var unsupportedCount int
	var failCount int
	var sampleErrs []string
	saveEvery := 8
	sinceSave := 0

	if len(localReqs) > 0 {
		subsetCounts := make(map[string]int)
		for _, r := range localReqs {
			subsetCounts[r.Subset]++
		}
		fmt.Println(labelStyle.Render(fmt.Sprintf("  Local host counts: %d file(s), %d subset(s)", len(localReqs), len(subsetCounts))))
		for i, r := range localReqs {
			fmt.Println(labelStyle.Render(fmt.Sprintf("    [local %d/%d] counting %s", i+1, len(localReqs), filepath.Base(r.RemotePath))))
			counts, err := cc.QueryParquetHostCounts(ctx, []cc.ParquetRowCountRequest{{Key: r.RemotePath, Path: r.QueryPath}})
			done++
			now := time.Now()
			if err != nil {
				if ccIsParquetHostColumnUnsupportedErr(err) {
					meta := metaByRemote[r.RemotePath]
					meta.HostCount = -1
					meta.HostCountUpdated = now
					metaByRemote[r.RemotePath] = meta
					cacheStore.SetParquetMeta(cacheData, r.RemotePath, meta)
					unsupportedCount++
				} else {
					failCount++
					if len(sampleErrs) < 5 {
						sampleErrs = append(sampleErrs, fmt.Sprintf("%s (local): %v", filepath.Base(r.RemotePath), err))
					}
				}
			} else if n, ok := counts[r.RemotePath]; ok {
				meta := metaByRemote[r.RemotePath]
				meta.HostCount = n
				meta.HostCountUpdated = now
				metaByRemote[r.RemotePath] = meta
				cacheStore.SetParquetMeta(cacheData, r.RemotePath, meta)
				successCount++
			} else {
				failCount++
				if len(sampleErrs) < 5 {
					sampleErrs = append(sampleErrs, fmt.Sprintf("%s (local): missing host count", filepath.Base(r.RemotePath)))
				}
			}
			sinceSave++
			if sinceSave >= saveEvery {
				sinceSave = 0
				if saveErr := ccSaveCacheWithParquetMetaMerge(cacheStore, cacheData); saveErr != nil {
					fmt.Println(warningStyle.Render(fmt.Sprintf("    Cache save warning: %v", saveErr)))
				}
			}
			fmt.Println(labelStyle.Render(fmt.Sprintf(
				"    Host progress: %s/%s (%s) ok=%s n/a=%s err=%s ETA %s",
				ccFmtInt64(int64(done)), ccFmtInt64(int64(totalMissing)), ccPct(int64(done), int64(totalMissing)),
				ccFmtInt64(int64(successCount)), ccFmtInt64(int64(unsupportedCount)), ccFmtInt64(int64(failCount)),
				ccFmtETA(start, int64(done), int64(totalMissing)),
			)))
		}
	}

	if len(remoteReqs) > 0 {
		subsetCounts := make(map[string]int)
		for _, r := range remoteReqs {
			subsetCounts[r.Subset]++
		}
		workers := min(max(runtime.NumCPU()/4, 1), 4)
		fmt.Println(labelStyle.Render(fmt.Sprintf("  Remote host counts: %d file(s), %d subset(s), workers=%d", len(remoteReqs), len(subsetCounts), workers)))

		var doneA atomic.Int64
		var okA atomic.Int64
		var unsupportedA atomic.Int64
		var errA atomic.Int64
		baseDone := done
		baseOK := successCount
		baseUnsupported := unsupportedCount
		baseErr := failCount
		var mu sync.Mutex

		stopProgress := make(chan struct{})
		progressDone := make(chan struct{})
		go func() {
			defer close(progressDone)
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-stopProgress:
					return
				case <-ctx.Done():
					return
				case <-ticker.C:
					dv := int64(baseDone) + doneA.Load()
					okv := int64(baseOK) + okA.Load()
					nav := int64(baseUnsupported) + unsupportedA.Load()
					errv := int64(baseErr) + errA.Load()
					fmt.Println(labelStyle.Render(fmt.Sprintf(
						"    Host progress: %s/%s (%s) ok=%s n/a=%s err=%s ETA %s",
						ccFmtInt64(dv), ccFmtInt64(int64(totalMissing)), ccPct(dv, int64(totalMissing)),
						ccFmtInt64(okv), ccFmtInt64(nav), ccFmtInt64(errv), ccFmtETA(start, dv, int64(totalMissing)),
					)))
				}
			}
		}()

		g, gctx := errgroup.WithContext(ctx)
		g.SetLimit(workers)
		for idx, r := range remoteReqs {
			idx, r := idx, r
			g.Go(func() error {
				if idx < 3 || idx%25 == 0 {
					fmt.Println(labelStyle.Render(fmt.Sprintf("    [remote %d/%d] %s", idx+1, len(remoteReqs), filepath.Base(r.RemotePath))))
				}
				counts, err := cc.QueryParquetHostCounts(gctx, []cc.ParquetRowCountRequest{{Key: r.RemotePath, Path: r.QueryPath}})
				now := time.Now()
				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					if ccIsParquetHostColumnUnsupportedErr(err) {
						meta := metaByRemote[r.RemotePath]
						meta.HostCount = -1
						meta.HostCountUpdated = now
						metaByRemote[r.RemotePath] = meta
						cacheStore.SetParquetMeta(cacheData, r.RemotePath, meta)
						unsupportedA.Add(1)
					} else {
						errA.Add(1)
						if len(sampleErrs) < 5 {
							sampleErrs = append(sampleErrs, fmt.Sprintf("%s (remote): %v", filepath.Base(r.RemotePath), err))
						}
					}
				} else if n, ok := counts[r.RemotePath]; ok {
					meta := metaByRemote[r.RemotePath]
					meta.HostCount = n
					meta.HostCountUpdated = now
					metaByRemote[r.RemotePath] = meta
					cacheStore.SetParquetMeta(cacheData, r.RemotePath, meta)
					okA.Add(1)
				} else {
					errA.Add(1)
					if len(sampleErrs) < 5 {
						sampleErrs = append(sampleErrs, fmt.Sprintf("%s (remote): missing host count", filepath.Base(r.RemotePath)))
					}
				}
				doneA.Add(1)
				sinceSave++
				if sinceSave >= saveEvery {
					sinceSave = 0
					if saveErr := ccSaveCacheWithParquetMetaMerge(cacheStore, cacheData); saveErr != nil {
						fmt.Println(warningStyle.Render(fmt.Sprintf("    Cache save warning: %v", saveErr)))
					}
				}
				return nil
			})
		}
		_ = g.Wait()
		close(stopProgress)
		<-progressDone
		done = baseDone + int(doneA.Load())
		successCount = baseOK + int(okA.Load())
		unsupportedCount = baseUnsupported + int(unsupportedA.Load())
		failCount = baseErr + int(errA.Load())
		fmt.Println(labelStyle.Render(fmt.Sprintf(
			"    Host progress: %s/%s (%s) ok=%s n/a=%s err=%s ETA %s",
			ccFmtInt64(int64(done)), ccFmtInt64(int64(totalMissing)), ccPct(int64(done), int64(totalMissing)),
			ccFmtInt64(int64(successCount)), ccFmtInt64(int64(unsupportedCount)), ccFmtInt64(int64(failCount)),
			ccFmtETA(start, int64(done), int64(totalMissing)),
		)))
	}

	if saveErr := ccSaveCacheWithParquetMetaMerge(cacheStore, cacheData); saveErr != nil {
		fmt.Println(warningStyle.Render(fmt.Sprintf("  Cache save warning: %v", saveErr)))
	}
	fmt.Println(successStyle.Render(fmt.Sprintf(
		"  Host count metadata complete: ok=%s n/a=%s err=%s (%s)",
		ccFmtInt64(int64(successCount)), ccFmtInt64(int64(unsupportedCount)), ccFmtInt64(int64(failCount)),
		time.Since(start).Truncate(time.Second),
	)))
	if len(sampleErrs) > 0 {
		fmt.Println(warningStyle.Render("  Host count metadata errors (sample):"))
		for _, s := range sampleErrs {
			fmt.Println(labelStyle.Render("   - " + s))
		}
	}
	if failCount > 0 {
		return fmt.Errorf("host count metadata missing for %d file(s)", failCount)
	}
	return nil
}

func ccIsParquetHostColumnUnsupportedErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "url_host_name") {
		return false
	}
	return strings.Contains(msg, "not found") ||
		strings.Contains(msg, "unknown column") ||
		strings.Contains(msg, "binder error")
}

func ccFmtETA(start time.Time, done, total int64) string {
	if done <= 0 {
		return "---"
	}
	if total <= 0 || total <= done {
		return "0s"
	}
	elapsed := time.Since(start)
	if elapsed <= 0 {
		return "---"
	}
	rate := float64(done) / elapsed.Seconds()
	if rate <= 0 {
		return "---"
	}
	remainSec := (float64(total) - float64(done)) / rate
	if remainSec < 0 {
		remainSec = 0
	}
	return (time.Duration(remainSec) * time.Second).Truncate(time.Second).String()
}

func ccSaveCacheWithParquetMetaMerge(cacheStore *cc.Cache, cacheData *cc.CacheData) error {
	if cacheStore == nil || cacheData == nil {
		return nil
	}
	disk := cacheStore.Load()
	if disk == nil {
		return cacheStore.Save(cacheData)
	}

	merged := *cacheData
	if merged.Crawls == nil && disk.Crawls != nil {
		merged.Crawls = disk.Crawls
	}
	if merged.LatestCrawlID == "" {
		merged.LatestCrawlID = disk.LatestCrawlID
	}
	if merged.FetchedAt.IsZero() {
		merged.FetchedAt = disk.FetchedAt
	}
	if merged.Manifests == nil && disk.Manifests != nil {
		merged.Manifests = disk.Manifests
	} else if merged.Manifests != nil && disk.Manifests != nil {
		for k, v := range disk.Manifests {
			if _, ok := merged.Manifests[k]; !ok {
				merged.Manifests[k] = v
			}
		}
	}

	if merged.ParquetMeta == nil && disk.ParquetMeta != nil {
		merged.ParquetMeta = disk.ParquetMeta
	} else if merged.ParquetMeta != nil && disk.ParquetMeta != nil {
		for k, dv := range disk.ParquetMeta {
			mv, ok := merged.ParquetMeta[k]
			if !ok {
				merged.ParquetMeta[k] = dv
				continue
			}
			if dv.SizeUpdatedAt.After(mv.SizeUpdatedAt) {
				mv.SizeBytes = dv.SizeBytes
				mv.SizeUpdatedAt = dv.SizeUpdatedAt
			}
			if dv.URLCountUpdated.After(mv.URLCountUpdated) {
				mv.URLCount = dv.URLCount
				mv.URLCountUpdated = dv.URLCountUpdated
			}
			if dv.HostCountUpdated.After(mv.HostCountUpdated) {
				mv.HostCount = dv.HostCount
				mv.HostCountUpdated = dv.HostCountUpdated
			}
			merged.ParquetMeta[k] = mv
		}
	}

	return cacheStore.Save(&merged)
}
