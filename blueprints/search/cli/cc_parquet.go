package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/cc"
	"github.com/spf13/cobra"
)

func newCCParquet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "parquet",
		Short: "List, download, and import columnar-index parquet files",
		Long: `Work with Common Crawl columnar-index parquet files directly.

This command can list every parquet file in a crawl dump (all subsets),
download specific files or samples, and import them into per-parquet DuckDB
databases with a catalog DuckDB view.`,
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
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCCParquetList(cmd.Context(), crawlID, subset, limit, namesOnly)
		},
	}

	cmd.Flags().StringVar(&crawlID, "crawl", "CC-MAIN-2026-08", "Crawl ID")
	cmd.Flags().StringVar(&subset, "subset", "", "Subset filter (e.g. warc, crawldiagnostics)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Max rows to display (0=all)")
	cmd.Flags().BoolVar(&namesOnly, "names-only", false, "Print only manifest index + remote path")

	return cmd
}

func runCCParquetList(ctx context.Context, crawlID, subset string, limit int, namesOnly bool) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("CC Columnar Parquet Manifest"))
	fmt.Println()

	cfg := cc.DefaultConfig()
	cfg.CrawlID = crawlID
	client := cc.NewClient(cfg.BaseURL, 4)

	fmt.Println(infoStyle.Render(fmt.Sprintf("Loading parquet manifest for %s...", crawlID)))
	start := time.Now()
	files, err := cc.ListParquetFiles(ctx, client, cfg, cc.ParquetListOptions{Subset: subset})
	if err != nil {
		return err
	}
	fmt.Println(successStyle.Render(fmt.Sprintf("  Loaded %s entries in %s",
		ccFmtInt64(int64(len(files))), time.Since(start).Truncate(time.Millisecond))))

	if len(files) == 0 {
		fmt.Println(warningStyle.Render("  No parquet files matched"))
		return nil
	}

	subsetCounts := make(map[string]int)
	for _, f := range files {
		key := f.Subset
		if key == "" {
			key = "(none)"
		}
		subsetCounts[key]++
	}

	fmt.Println()
	fmt.Println(infoStyle.Render("Subset counts:"))
	type subsetCount struct {
		Subset string
		Count  int
	}
	var pairs []subsetCount
	for k, v := range subsetCounts {
		pairs = append(pairs, subsetCount{Subset: k, Count: v})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].Count == pairs[j].Count {
			return pairs[i].Subset < pairs[j].Subset
		}
		return pairs[i].Count > pairs[j].Count
	})
	for _, p := range pairs {
		fmt.Printf("  %-20s %s\n", p.Subset, ccFmtInt64(int64(p.Count)))
	}

	display := files
	if limit > 0 && limit < len(display) {
		display = display[:limit]
	}

	fmt.Println()
	if namesOnly {
		for _, f := range display {
			fmt.Printf("  [%d] %s\n", f.ManifestIndex, f.RemotePath)
		}
	} else {
		fmt.Printf("  %-8s %-18s %-9s %-20s %s\n", "Manifest", "Subset", "Local", "Filename", "Remote Path")
		fmt.Println(strings.Repeat("─", 140))
		for _, f := range display {
			localPath := cc.LocalParquetPathForRemote(cfg, f.RemotePath)
			local := labelStyle.Render("missing")
			if st, statErr := os.Stat(localPath); statErr == nil && st.Size() > 0 {
				local = successStyle.Render("yes")
			}
			fmt.Printf("  %-8d %-18s %-9s %-20s %s\n",
				f.ManifestIndex, f.Subset, local, trimMiddle(f.Filename, 20), f.RemotePath)
		}
	}

	if len(display) < len(files) {
		fmt.Println()
		fmt.Println(labelStyle.Render(fmt.Sprintf("  Showing %d of %d entries", len(display), len(files))))
	}
	return nil
}

func newCCParquetDownload() *cobra.Command {
	var (
		crawlID string
		subset  string
		fileIdx int
		sample  int
		all     bool
		workers int
	)

	cmd := &cobra.Command{
		Use:   "download",
		Short: "Download parquet files from the Common Crawl columnar index",
		Long: `Download parquet files listed in cc-index-table.paths.gz.

Modes:
  --file N    Download one parquet by manifest index (all subsets)
  --sample N  Download N evenly spaced parquet files (after subset filter)
  --all       Download every parquet file (after subset filter)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCCParquetDownload(cmd.Context(), crawlID, subset, fileIdx, sample, all, workers)
		},
	}

	cmd.Flags().StringVar(&crawlID, "crawl", "CC-MAIN-2026-08", "Crawl ID")
	cmd.Flags().StringVar(&subset, "subset", "", "Subset filter (e.g. warc, crawldiagnostics)")
	cmd.Flags().IntVar(&fileIdx, "file", -1, "Manifest index to download (all-subset manifest index)")
	cmd.Flags().IntVar(&sample, "sample", 0, "Download N evenly spaced files (after subset filter)")
	cmd.Flags().BoolVar(&all, "all", false, "Download all files (after subset filter)")
	cmd.Flags().IntVar(&workers, "workers", 10, "Concurrent download workers")

	return cmd
}

func runCCParquetDownload(ctx context.Context, crawlID, subset string, fileIdx, sample int, all bool, workers int) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("CC Parquet Download"))
	fmt.Println()

	cfg := cc.DefaultConfig()
	cfg.CrawlID = crawlID
	cfg.IndexWorkers = workers
	client := cc.NewClient(cfg.BaseURL, cfg.TransportShards)

	reporter := newCCDownloadReporter()

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
		return fmt.Errorf("choose one mode: --file N, --sample N, or --all")
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
		limit   int
	)

	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import local parquet files into per-parquet DuckDB + catalog",
		Long: `Import local parquet files into one DuckDB database per parquet file, then
build a catalog DuckDB at index.duckdb containing metadata tables and a ccindex view.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCCParquetImport(cmd.Context(), crawlID, subset, file, limit)
		},
	}

	cmd.Flags().StringVar(&crawlID, "crawl", "CC-MAIN-2026-08", "Crawl ID")
	cmd.Flags().StringVar(&subset, "subset", "", "Subset filter for local parquet files")
	cmd.Flags().StringVar(&file, "file", "", "Import a specific local parquet file")
	cmd.Flags().IntVar(&limit, "limit", 0, "Import only the first N matching local parquet files (0=all)")

	return cmd
}

func runCCParquetImport(ctx context.Context, crawlID, subset, file string, limit int) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("CC Parquet Import"))
	fmt.Println()

	cfg := cc.DefaultConfig()
	cfg.CrawlID = crawlID

	var parquetPaths []string
	var err error

	if file != "" {
		if _, err := os.Stat(file); err != nil {
			return fmt.Errorf("parquet file not found: %s", file)
		}
		parquetPaths = []string{file}
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
