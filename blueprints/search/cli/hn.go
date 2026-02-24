package cli

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/hn"
	"github.com/spf13/cobra"
)

func NewHN() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hn",
		Short: "Hacker News dataset download/import (parquet + API fallback)",
		Long: `Download and import all Hacker News data using a fast bulk parquet source
with an official HN API fallback.

Data is stored at $HOME/data/hn/ by default:
  raw/items.parquet        primary parquet snapshot (resumable HTTP download)
  raw/api/chunks/*.jsonl   chunked API fallback downloads (resumable by chunk)
  hn.duckdb                imported DuckDB database

Examples:
  search hn list
  search hn download
  search hn download --source parquet
  search hn download --source api --from-id 1 --to-id 5000
  search hn import
  search hn status
  search hn sync`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.PersistentFlags().String("dir", "", "Override HN data dir (default: $HOME/data/hn)")
	cmd.PersistentFlags().String("parquet-url", "", "Override HN parquet URL")
	cmd.PersistentFlags().String("api-base-url", "", "Override HN API base URL")
	cmd.PersistentFlags().String("clickhouse-url", "", "Override ClickHouse SQL playground HTTP URL")
	cmd.PersistentFlags().String("clickhouse-user", "", "Override ClickHouse HTTP user")
	cmd.PersistentFlags().String("clickhouse-database", "", "Override ClickHouse database")
	cmd.PersistentFlags().String("clickhouse-table", "", "Override ClickHouse table")
	_ = cmd.PersistentFlags().MarkHidden("dir")
	_ = cmd.PersistentFlags().MarkHidden("parquet-url")
	_ = cmd.PersistentFlags().MarkHidden("api-base-url")
	_ = cmd.PersistentFlags().MarkHidden("clickhouse-url")
	_ = cmd.PersistentFlags().MarkHidden("clickhouse-user")
	_ = cmd.PersistentFlags().MarkHidden("clickhouse-database")
	_ = cmd.PersistentFlags().MarkHidden("clickhouse-table")

	cmd.AddCommand(newHNList())
	cmd.AddCommand(newHNStatus())
	cmd.AddCommand(newHNDownload())
	cmd.AddCommand(newHNImport())
	cmd.AddCommand(newHNSync())
	cmd.AddCommand(newHNCompact())
	cmd.AddCommand(newHNExport())
	return cmd
}

func hnConfigFromCmd(cmd *cobra.Command) hn.Config {
	var cfg hn.Config
	if v, _ := cmd.Flags().GetString("dir"); strings.TrimSpace(v) != "" {
		cfg.DataDir = v
	}
	if v, _ := cmd.Flags().GetString("parquet-url"); strings.TrimSpace(v) != "" {
		cfg.ParquetURL = v
	}
	if v, _ := cmd.Flags().GetString("api-base-url"); strings.TrimSpace(v) != "" {
		cfg.APIBaseURL = v
	}
	if v, _ := cmd.Flags().GetString("clickhouse-url"); strings.TrimSpace(v) != "" {
		cfg.ClickHouseBaseURL = v
	}
	if v, _ := cmd.Flags().GetString("clickhouse-user"); strings.TrimSpace(v) != "" {
		cfg.ClickHouseUser = v
	}
	if v, _ := cmd.Flags().GetString("clickhouse-database"); strings.TrimSpace(v) != "" {
		cfg.ClickHouseDatabase = v
	}
	if v, _ := cmd.Flags().GetString("clickhouse-table"); strings.TrimSpace(v) != "" {
		cfg.ClickHouseTable = v
	}
	return cfg
}

func newHNList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Show remote HN source and local file/database status",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg := hnConfigFromCmd(cmd)
			noRemote, _ := cmd.Flags().GetBool("no-remote")
			showHNLocalStatus(ctx, cfg)
			if noRemote {
				return nil
			}
			fmt.Println()
			fmt.Println(infoStyle.Render("Remote ClickHouse source (latest):"))
			chInfo, err := cfg.ClickHouseInfo(ctx)
			if err != nil {
				fmt.Printf("  %s %v\n", warningStyle.Render("WARN"), err)
				fmt.Println(labelStyle.Render("Falling back to cached/static snapshot metadata if available."))
				if cached, cerr := cfg.ReadCachedParquetHead(); cerr == nil && cached != nil {
					fmt.Printf("  Cached size: %s\n", formatBytes(cached.Size))
					fmt.Printf("  Cached ETag: %s\n", cached.ETag)
					fmt.Printf("  Cached at:   %s\n", cached.CheckedAt.Format("2006-01-02 15:04:05 MST"))
				}
				return nil
			}
			fmt.Printf("  Endpoint:    %s\n", infoStyle.Render(chInfo.BaseURL))
			fmt.Printf("  Table:       %s.%s\n", successStyle.Render(chInfo.Database), successStyle.Render(chInfo.Table))
			fmt.Printf("  Rows:        %s\n", successStyle.Render(formatLargeNumber(chInfo.Count)))
			fmt.Printf("  Max ID:      %s\n", successStyle.Render(formatLargeNumber(chInfo.MaxID)))
			fmt.Printf("  Max Time:    %s\n", successStyle.Render(chInfo.MaxTime))
			fmt.Printf("  Checked:     %s\n", labelStyle.Render(chInfo.CheckedAt.Format("2006-01-02 15:04:05 MST")))

			fmt.Println()
			fmt.Println(labelStyle.Render("Static snapshot fallback (legacy):"))
			info, err := cfg.HeadParquet(ctx)
			if err != nil {
				fmt.Printf("  %s %v\n", warningStyle.Render("WARN"), err)
				return nil
			}
			fmt.Printf("  URL:         %s\n", labelStyle.Render(info.URL))
			fmt.Printf("  Size:        %s\n", labelStyle.Render(formatBytes(info.Size)))
			if info.LastModified != "" {
				fmt.Printf("  Updated:     %s\n", labelStyle.Render(info.LastModified))
			}
			return nil
		},
	}
	cmd.Flags().Bool("no-remote", false, "Skip remote HEAD request and show local status only")
	return cmd
}

func newHNStatus() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show local HN status (files + DuckDB)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return showHNLocalStatus(cmd.Context(), hnConfigFromCmd(cmd))
		},
	}
}

func newHNDownload() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download",
		Short: "Download HN dataset (parquet primary, API fallback)",
		Example: `  search hn download
  search hn download --source clickhouse
  search hn download --source delta
  search hn download --source parquet
  search hn download --source api --from-id 1 --to-id 5000
  search hn download --source auto --no-fallback`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := runHNDownload(cmd.Context(), cmd)
			return err
		},
	}
	cmd.Flags().String("source", "auto", "Download source: auto|clickhouse|delta|parquet|api")
	cmd.Flags().Bool("force", false, "Restart download and overwrite existing local target")
	cmd.Flags().Bool("no-fallback", false, "In auto mode, fail instead of falling back to API")
	cmd.Flags().Bool("no-delta", false, "Skip API delta after ClickHouse download (auto/clickhouse modes)")
	cmd.Flags().Int64("chunk-id-span", 500000, "ClickHouse mode: id range per parquet chunk")
	cmd.Flags().Int("parallel", 4, "ClickHouse mode: parallel parquet chunk downloads")
	cmd.Flags().Int("workers", 32, "API mode: concurrent item fetch workers")
	cmd.Flags().Int("chunk-size", 1000, "API mode: ids per chunk file")
	cmd.Flags().Int64("from-id", 0, "API mode: start item id (default 1)")
	cmd.Flags().Int64("to-id", 0, "API mode: end item id (default maxitem)")
	return cmd
}

func newHNImport() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import local HN data into DuckDB",
		Example: `  search hn import
  search hn import --source clickhouse
  search hn import --source hybrid
  search hn import --source parquet
  search hn import --source api`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := hnConfigFromCmd(cmd)
			sourceStr, _ := cmd.Flags().GetString("source")
			dbPath, _ := cmd.Flags().GetString("db")
			rebuild, _ := cmd.Flags().GetBool("rebuild")
			source, err := parseHNImportSource(sourceStr)
			if err != nil {
				return err
			}
			fmt.Println(infoStyle.Render("Importing Hacker News data into DuckDB..."))
			res, err := cfg.Import(cmd.Context(), hn.ImportOptions{Source: source, DBPath: dbPath, Rebuild: rebuild})
			if err != nil {
				return err
			}
			printHNImportResult(res)
			return nil
		},
	}
	cmd.Flags().String("source", "auto", "Import source: auto|clickhouse|hybrid|parquet|api")
	cmd.Flags().String("db", "", "DuckDB output path (default: $HOME/data/hn/hn.duckdb)")
	cmd.Flags().Bool("rebuild", false, "Force full table rebuild instead of incremental merge when DB exists")
	return cmd
}

func newHNSync() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Download and then import HN data in one command",
		RunE: func(cmd *cobra.Command, args []string) error {
			every, _ := cmd.Flags().GetDuration("every")
			maxRuns, _ := cmd.Flags().GetInt("max-runs")
			if every <= 0 {
				return runHNSyncOnce(cmd.Context(), cmd, 1)
			}
			if maxRuns == 0 {
				maxRuns = -1
			}
			run := 0
			for {
				run++
				started := time.Now()
				fmt.Printf("%s HN sync tick #%d at %s\n", infoStyle.Render("Running"), run, labelStyle.Render(started.Format(time.RFC3339)))
				if err := runHNSyncOnce(cmd.Context(), cmd, run); err != nil {
					return err
				}
				if maxRuns > 0 && run >= maxRuns {
					return nil
				}
				wait := time.Until(started.Add(every))
				if wait <= 0 {
					continue
				}
				fmt.Printf("%s next tick in %s\n", labelStyle.Render("Waiting:"), labelStyle.Render(formatDuration(wait)))
				timer := time.NewTimer(wait)
				select {
				case <-cmd.Context().Done():
					timer.Stop()
					return cmd.Context().Err()
				case <-timer.C:
				}
			}
		},
	}
	cmd.Flags().String("source", "auto", "Download source: auto|clickhouse|delta|parquet|api")
	cmd.Flags().Bool("force", false, "Restart download and overwrite existing local target")
	cmd.Flags().Bool("no-fallback", false, "In auto mode, fail instead of falling back to API")
	cmd.Flags().Bool("no-delta", false, "Skip API delta after ClickHouse download (auto/clickhouse modes)")
	cmd.Flags().Int64("chunk-id-span", 500000, "ClickHouse mode: id range per parquet chunk")
	cmd.Flags().Int("parallel", 4, "ClickHouse mode: parallel parquet chunk downloads")
	cmd.Flags().Int("workers", 32, "API mode: concurrent item fetch workers")
	cmd.Flags().Int("chunk-size", 1000, "API mode: ids per chunk file")
	cmd.Flags().Int64("from-id", 0, "API mode: start item id (default 1)")
	cmd.Flags().Int64("to-id", 0, "API mode: end item id (default maxitem)")
	cmd.Flags().String("db", "", "DuckDB output path (default: $HOME/data/hn/hn.duckdb)")
	cmd.Flags().Bool("rebuild", false, "Force full table rebuild instead of incremental merge when DB exists")
	cmd.Flags().Duration("every", 0, "Run sync on a ticker interval (e.g. 1m, 30s)")
	cmd.Flags().Int("max-runs", 1, "Stop after N runs when --every is set (0 = run forever)")
	return cmd
}

func newHNCompact() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compact",
		Short: "Merge local API delta chunks back into ClickHouse chunk parquet files",
		Long: `Reads local raw/api/chunks/*.jsonl, converts them to a ClickHouse-compatible parquet schema
and merges them into local raw/clickhouse/id_<start>_<end>.parquet partitions using DuckDB.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := hnConfigFromCmd(cmd)
			fromID, _ := cmd.Flags().GetInt64("from-id")
			toID, _ := cmd.Flags().GetInt64("to-id")
			chunkSpan, _ := cmd.Flags().GetInt64("chunk-id-span")
			compLevel, _ := cmd.Flags().GetInt("compression-level")
			pruneAPI, _ := cmd.Flags().GetBool("prune-api")

			fmt.Println(infoStyle.Render("Compacting API delta into ClickHouse parquet partitions..."))
			res, err := cfg.CompactDeltaToClickHouseParquet(cmd.Context(), hn.CompactOptions{
				FromID:           fromID,
				ToID:             toID,
				ChunkIDSpan:      chunkSpan,
				CompressionLevel: compLevel,
				PruneAPI:         pruneAPI,
			})
			if err != nil {
				return err
			}
			fmt.Printf("  %s  Dir: %s\n", successStyle.Render("OK"), labelStyle.Render(res.Dir))
			fmt.Printf("  Range:      %s\n", labelStyle.Render(formatHNRange(res.FromID, res.ToID)))
			fmt.Printf("  Chunk span: %s\n", labelStyle.Render(formatInt64Exact(res.ChunkIDSpan)))
			fmt.Printf("  API rows:   %s (%s)\n", labelStyle.Render(formatLargeNumber(res.APIRows)), labelStyle.Render(formatInt64Exact(res.APIRows)))
			fmt.Printf("  Chunks:     touched=%d written=%d skipped=%d\n", res.ChunksTouched, res.ChunksWritten, res.ChunksSkipped)
			fmt.Printf("  Pruned:     parquet=%d api_chunks=%d\n", res.FilesPruned, res.APIChunksPruned)
			fmt.Printf("  Elapsed:    %s\n", labelStyle.Render(formatDuration(res.Elapsed)))
			for _, ch := range res.Chunks {
				fmt.Printf("    %s rows=%s path=%s\n",
					labelStyle.Render(formatHNRange(ch.ChunkStart, ch.ChunkEnd)),
					labelStyle.Render(formatInt64Exact(ch.Rows)),
					labelStyle.Render(ch.Path),
				)
			}
			return nil
		},
	}
	cmd.Flags().Int64("from-id", 0, "Compact only API delta rows >= this id (default: infer from download state/API chunks)")
	cmd.Flags().Int64("to-id", 0, "Compact only API delta rows <= this id (default: infer from download state/API chunks)")
	cmd.Flags().Int64("chunk-id-span", 0, "ClickHouse chunk id span (default: auto-detect local chunk span)")
	cmd.Flags().Int("compression-level", 22, "Parquet zstd compression level for rewritten chunk files")
	cmd.Flags().Bool("prune-api", false, "Delete API jsonl chunk files fully covered by the compacted id range")
	return cmd
}

func newHNExport() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export HN items from DuckDB into monthly parquet files",
		Example: `  search hn export
  search hn export --from-month 2006-10 --to-month 2006-12
  search hn export --out-dir ~/data/hn/export/monthly
  search hn export --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := hnConfigFromCmd(cmd)
			dbPath, _ := cmd.Flags().GetString("db")
			outDir, _ := cmd.Flags().GetString("out-dir")
			fromMonth, _ := cmd.Flags().GetString("from-month")
			toMonth, _ := cmd.Flags().GetString("to-month")
			force, _ := cmd.Flags().GetBool("force")
			refreshLatest, _ := cmd.Flags().GetBool("refresh-latest")
			compLevel, _ := cmd.Flags().GetInt("compression-level")

			fmt.Println(infoStyle.Render("Exporting HN items by month to parquet..."))
			res, err := cfg.ExportMonthlyParquet(cmd.Context(), hn.ExportOptions{
				DBPath:           dbPath,
				OutDir:           outDir,
				FromMonth:        fromMonth,
				ToMonth:          toMonth,
				Force:            force,
				RefreshLatest:    refreshLatest,
				CompressionLevel: compLevel,
			})
			if err != nil {
				return err
			}
			fmt.Printf("  %s  Out dir: %s\n", successStyle.Render("OK"), labelStyle.Render(res.OutDir))
			fmt.Printf("  DB:         %s\n", labelStyle.Render(res.DBPath))
			if strings.TrimSpace(res.LatestMonth) != "" {
				fmt.Printf("  Latest:     %s\n", labelStyle.Render(res.LatestMonth))
			}
			fmt.Printf("  Months:     scanned=%d written=%d skipped=%d\n", res.MonthsScanned, res.MonthsWritten, res.MonthsSkipped)
			fmt.Printf("  Rows:       %s (%s)\n", successStyle.Render(formatLargeNumber(res.RowsWritten)), successStyle.Render(formatInt64Exact(res.RowsWritten)))
			fmt.Printf("  Bytes:      %s\n", successStyle.Render(formatBytes(res.BytesWritten)))
			fmt.Printf("  Elapsed:    %s\n", labelStyle.Render(formatDuration(res.Elapsed)))
			for _, m := range res.Months {
				status := "wrote"
				if m.Skipped {
					status = "skipped"
				} else if m.Refreshed {
					status = "refreshed"
				}
				fmt.Printf("    %s %s rows=%s size=%s %s\n",
					labelStyle.Render(m.Month),
					labelStyle.Render(status),
					labelStyle.Render(formatInt64Exact(m.Rows)),
					labelStyle.Render(formatBytes(m.Size)),
					labelStyle.Render(m.Path),
				)
			}
			return nil
		},
	}
	cmd.Flags().String("db", "", "DuckDB path (default: $HOME/data/hn/hn.duckdb)")
	cmd.Flags().String("out-dir", "", "Output directory (default: $HOME/data/hn/export/hn/monthly)")
	cmd.Flags().String("from-month", "", "Start month inclusive (YYYY-MM)")
	cmd.Flags().String("to-month", "", "End month inclusive (YYYY-MM)")
	cmd.Flags().Bool("force", false, "Rewrite all selected month parquet files even if they already exist")
	cmd.Flags().Bool("refresh-latest", true, "Rewrite the latest month file if it already exists (even when skipping older months)")
	cmd.Flags().Int("compression-level", 22, "Parquet zstd compression level (DuckDB COPY)")
	return cmd
}

func runHNSyncOnce(ctx context.Context, cmd *cobra.Command, run int) error {
	usedSource, err := runHNDownload(ctx, cmd)
	if err != nil {
		return err
	}
	cfg := hnConfigFromCmd(cmd)
	dbPath, _ := cmd.Flags().GetString("db")
	rebuild, _ := cmd.Flags().GetBool("rebuild")
	importSource := hn.ImportSource(usedSource)
	if importSource == "auto" {
		importSource = hn.ImportSourceAuto
	}
	fmt.Println()
	if run > 0 {
		fmt.Printf("%s tick #%d: importing downloaded data...\n", infoStyle.Render("HN sync"), run)
	} else {
		fmt.Println(infoStyle.Render("Importing downloaded data..."))
	}
	res, err := cfg.Import(ctx, hn.ImportOptions{Source: importSource, DBPath: dbPath, Rebuild: rebuild})
	if err != nil {
		return err
	}
	printHNImportResult(res)
	return nil
}

func printHNImportResult(res *hn.ImportResult) {
	fmt.Printf("  %s  Source: %s\n", successStyle.Render("OK"), res.SourceUsed)
	fmt.Printf("  Mode:       %s\n", successStyle.Render(res.Mode))
	if res.ImportFromID > 0 {
		fmt.Printf("  From ID:    %s (%s)\n", successStyle.Render(formatLargeNumber(res.ImportFromID)), labelStyle.Render(formatInt64Exact(res.ImportFromID)))
	}
	fmt.Printf("  Rows prev:  %s (%s)\n", labelStyle.Render(formatLargeNumber(res.RowsBefore)), labelStyle.Render(formatInt64Exact(res.RowsBefore)))
	fmt.Printf("  Rows delta: %s (%s)\n", successStyle.Render(formatLargeNumber(res.RowsDelta)), successStyle.Render(formatInt64Exact(res.RowsDelta)))
	fmt.Printf("  Rows:       %s (%s)\n", successStyle.Render(formatLargeNumber(res.Rows)), successStyle.Render(formatInt64Exact(res.Rows)))
	fmt.Printf("  DB:         %s\n", labelStyle.Render(res.DBPath))
	fmt.Printf("  Indexes:    %d\n", res.IndexesMade)
}

func formatInt64Exact(n int64) string {
	s := strconv.FormatInt(n, 10)
	if n < 0 {
		return "-" + formatInt64Exact(-n)
	}
	if len(s) <= 3 {
		return s
	}
	var out []byte
	rem := len(s) % 3
	if rem == 0 {
		rem = 3
	}
	out = append(out, s[:rem]...)
	for i := rem; i < len(s); i += 3 {
		out = append(out, ',')
		out = append(out, s[i:i+3]...)
	}
	return string(out)
}

func formatHNRange(startID, endID int64) string {
	if startID <= 0 && endID <= 0 {
		return ""
	}
	if endID <= 0 {
		return formatInt64Exact(startID) + "-"
	}
	return formatInt64Exact(startID) + "-" + formatInt64Exact(endID)
}

func showHNLocalStatus(ctx context.Context, cfg hn.Config) error {
	st, err := cfg.LocalStatus(ctx)
	if err != nil {
		return err
	}
	fmt.Println(infoStyle.Render("Local HN status:"))
	fmt.Printf("  Data dir:    %s\n", labelStyle.Render(st.DataDir))
	if st.ParquetExists {
		fmt.Printf("  Parquet:     %s (%s)\n", successStyle.Render(filepath.Base(st.ParquetPath)), formatBytes(st.ParquetSize))
	} else {
		fmt.Printf("  Parquet:     %s\n", warningStyle.Render("missing"))
	}
	fmt.Printf("  CH chunks:   %d file(s), %s\n", st.CHParquetCount, formatBytes(st.CHParquetBytes))
	fmt.Printf("  API chunks:  %d file(s), %s\n", st.APIChunkCount, formatBytes(st.APIChunkBytes))
	if st.DBExists {
		fmt.Printf("  DuckDB:      %s (%s)\n", successStyle.Render(filepath.Base(st.DBPath)), formatBytes(st.DBSize))
		if st.DBRows > 0 {
			fmt.Printf("  Rows:        %s\n", successStyle.Render(formatLargeNumber(st.DBRows)))
		}
		if len(st.DBTypes) > 0 {
			fmt.Printf("  Types:       ")
			for i, tc := range st.DBTypes {
				if i > 0 {
					fmt.Print(", ")
				}
				name := tc.Type
				if strings.TrimSpace(name) == "" {
					name = "(empty)"
				}
				fmt.Printf("%s=%s", name, formatLargeNumber(tc.Count))
			}
			fmt.Println()
		}
	} else {
		fmt.Printf("  DuckDB:      %s\n", warningStyle.Render("missing"))
	}
	return nil
}

func runHNDownload(ctx context.Context, cmd *cobra.Command) (string, error) {
	cfg := hnConfigFromCmd(cmd)
	sourceStr, _ := cmd.Flags().GetString("source")
	force, _ := cmd.Flags().GetBool("force")
	noFallback, _ := cmd.Flags().GetBool("no-fallback")
	noDelta, _ := cmd.Flags().GetBool("no-delta")
	chunkIDSpan, _ := cmd.Flags().GetInt64("chunk-id-span")
	parallel, _ := cmd.Flags().GetInt("parallel")
	workers, _ := cmd.Flags().GetInt("workers")
	chunkSize, _ := cmd.Flags().GetInt("chunk-size")
	fromID, _ := cmd.Flags().GetInt64("from-id")
	toID, _ := cmd.Flags().GetInt64("to-id")

	source := strings.ToLower(strings.TrimSpace(sourceStr))
	if source == "" {
		source = "auto"
	}
	if source != "auto" && source != "clickhouse" && source != "delta" && source != "parquet" && source != "api" {
		return "", fmt.Errorf("invalid --source %q (want auto|clickhouse|delta|parquet|api)", sourceStr)
	}
	if (source == "auto" || source == "clickhouse") && !cmd.Flags().Changed("chunk-id-span") {
		if span, ok := cfg.DetectLocalClickHouseChunkSpan(); ok && span > 0 && span != chunkIDSpan {
			fmt.Printf("  %s using local ClickHouse chunk span %d (detected from existing files)\n", labelStyle.Render("Info:"), span)
			chunkIDSpan = span
		}
	}

	var clickhouseRes *hn.ClickHouseDownloadResult
	var apiRes *hn.APIDownloadResult
	var apiDeltaFrom, apiDeltaTo int64
	var apiWasDelta bool
	apiDownloaded := false

	doClickHouse := func() error {
		fmt.Println(infoStyle.Render("Downloading HN data from ClickHouse SQL playground (chunked Parquet)..."))
		cb := func(p hn.ClickHouseDownloadProgress) {
			if p.Complete {
				fmt.Printf("  %s  chunks=%d/%d (skipped=%d) bytes=%s avg=%s elapsed=%s\n",
					successStyle.Render("OK"),
					p.ChunksDone, p.ChunksTotal, p.ChunksSkipped,
					formatBytes(p.BytesDone),
					formatBytesPerSec(p.OverallSpeedBPS),
					formatDuration(p.Elapsed),
				)
				return
			}
			if p.Detail != "" {
				fmt.Printf("  Chunk %d/%d [%d-%d] %s (%s) active=%d total=%s avg=%s\n",
					p.ChunksDone, p.ChunksTotal, p.ChunkStart, p.ChunkEnd, p.Detail,
					formatBytes(p.ChunkBytes),
					p.ChunksActive,
					formatBytes(p.BytesDone),
					formatBytesPerSec(p.OverallSpeedBPS),
				)
				return
			}
			fmt.Printf("  Chunk %d/%d [%d-%d] %s in %s (%s) active=%d total=%s avg=%s\n",
				p.ChunksDone, p.ChunksTotal, p.ChunkStart, p.ChunkEnd,
				formatBytes(p.ChunkBytes),
				formatDuration(p.ChunkElapsed),
				formatBytesPerSec(p.ChunkSpeedBPS),
				p.ChunksActive,
				formatBytes(p.BytesDone),
				formatBytesPerSec(p.OverallSpeedBPS),
			)
		}
		res, err := cfg.DownloadClickHouseParquet(ctx, hn.ClickHouseDownloadOptions{
			FromID:      fromID,
			ToID:        toID,
			ChunkIDSpan: chunkIDSpan,
			Parallelism: parallel,
			Force:       force,
		}, cb)
		if err != nil {
			return err
		}
		clickhouseRes = res
		fmt.Printf("  Dir:        %s\n", labelStyle.Render(res.Dir))
		fmt.Printf("  Range:      %d-%d\n", res.StartID, res.EndID)
		if res.RemoteInfo != nil {
			fmt.Printf("  Remote rows: %s  max_id=%s  max_time=%s\n",
				successStyle.Render(formatLargeNumber(res.RemoteInfo.Count)),
				successStyle.Render(formatLargeNumber(res.RemoteInfo.MaxID)),
				labelStyle.Render(res.RemoteInfo.MaxTime),
			)
		}
		return nil
	}

	doParquet := func() error {
		fmt.Println(infoStyle.Render("Downloading HN parquet snapshot..."))
		var lastLineLen int
		cb := func(p hn.ParquetDownloadProgress) {
			if p.Complete && p.Skipped {
				line := fmt.Sprintf("  %s  already complete (%s)", successStyle.Render("OK"), formatBytes(p.LocalSize))
				fmt.Println(line)
				return
			}
			if p.Complete {
				line := fmt.Sprintf("  %s  %s downloaded (%s total)", successStyle.Render("OK"), filepath.Base(p.Path), formatBytes(p.LocalSize))
				fmt.Printf("\r%s%s\n", line, padSpaces(lastLineLen-len(line)))
				lastLineLen = 0
				return
			}
			pct := 0.0
			if p.RemoteSize > 0 {
				pct = 100 * float64(p.LocalSize) / float64(p.RemoteSize)
			}
			line := fmt.Sprintf("  %.1f%%  %s / %s  %s",
				pct,
				formatBytes(p.LocalSize),
				formatBytes(p.RemoteSize),
				formatBytesPerSec(p.SpeedBPS),
			)
			fmt.Printf("\r%s%s", line, padSpaces(lastLineLen-len(line)))
			lastLineLen = len(line)
		}
		res, err := cfg.DownloadParquet(ctx, force, cb)
		if lastLineLen > 0 {
			fmt.Println()
		}
		if err != nil {
			return err
		}
		fmt.Printf("  Path:       %s\n", labelStyle.Render(res.Path))
		fmt.Printf("  Size:       %s\n", successStyle.Render(formatBytes(res.LocalSize)))
		if res.Remote != nil && res.Remote.ETag != "" {
			fmt.Printf("  ETag:       %s\n", labelStyle.Render(res.Remote.ETag))
		}
		return nil
	}

	doAPI := func() error {
		fmt.Println(infoStyle.Render("Downloading HN data from official API (fallback mode)..."))
		cb := func(p hn.APIDownloadProgress) {
			if p.Complete {
				fmt.Printf("  %s  chunks=%d (skipped=%d) ids=%d items=%d\n",
					successStyle.Render("OK"), p.ChunksDone, p.ChunksSkipped, p.IDsProcessed, p.ItemsWritten)
				return
			}
			if p.Detail != "" {
				fmt.Printf("  Chunk %d/%d [%d-%d] %s\n", p.ChunksDone, p.ChunksTotal, p.ChunkStart, p.ChunkEnd, p.Detail)
				return
			}
			fmt.Printf("  Chunk %d/%d [%d-%d] ids=%d items=%d\n",
				p.ChunksDone, p.ChunksTotal, p.ChunkStart, p.ChunkEnd, p.IDsProcessed, p.ItemsWritten)
		}
		res, err := cfg.DownloadAPI(ctx, hn.APIDownloadOptions{
			Workers:   workers,
			ChunkSize: chunkSize,
			FromID:    fromID,
			ToID:      toID,
			Force:     force,
		}, cb)
		if err != nil {
			return err
		}
		apiDownloaded = true
		apiRes = res
		apiWasDelta = false
		apiDeltaFrom, apiDeltaTo = 0, 0
		fmt.Printf("  Dir:        %s\n", labelStyle.Render(res.Dir))
		fmt.Printf("  Range:      %d-%d\n", res.StartID, res.EndID)
		return nil
	}

	doAPIDelta := func() error {
		if noDelta {
			return nil
		}
		if clickhouseRes == nil {
			return nil
		}
		maxItem, err := cfg.GetMaxItem(ctx)
		if err != nil {
			return fmt.Errorf("get HN maxitem for delta: %w", err)
		}
		deltaFrom := clickhouseRes.EndID + 1
		if deltaFrom <= 0 {
			deltaFrom = clickhouseRes.RemoteMaxID + 1
		}
		if deltaFrom > maxItem {
			fmt.Println(labelStyle.Render("No API delta needed (ClickHouse is already at or ahead of HN maxitem)."))
			return nil
		}
		fmt.Println(infoStyle.Render(fmt.Sprintf("Downloading API delta %d-%d to catch up after ClickHouse snapshot...", deltaFrom, maxItem)))
		cb := func(p hn.APIDownloadProgress) {
			if p.Complete {
				fmt.Printf("  %s  delta chunks=%d (skipped=%d) ids=%d items=%d\n",
					successStyle.Render("OK"), p.ChunksDone, p.ChunksSkipped, p.IDsProcessed, p.ItemsWritten)
				return
			}
		}
		res, err := cfg.DownloadAPI(ctx, hn.APIDownloadOptions{
			Workers:   workers,
			ChunkSize: chunkSize,
			FromID:    deltaFrom,
			ToID:      maxItem,
			Force:     false, // preserve resumability for delta chunks
		}, cb)
		if err != nil {
			return err
		}
		apiDownloaded = true
		apiRes = res
		apiWasDelta = true
		apiDeltaFrom, apiDeltaTo = deltaFrom, maxItem
		return nil
	}

	finalize := func(sourceUsed string) (string, error) {
		st := &hn.DownloadState{
			SourceUsed: sourceUsed,
		}
		if clickhouseRes != nil {
			st.ClickHouse = &hn.ClickHouseRunState{
				StartID:           clickhouseRes.StartID,
				EndID:             clickhouseRes.EndID,
				RemoteMaxID:       clickhouseRes.RemoteMaxID,
				RemoteCount:       clickhouseRes.RemoteCount,
				ChunkIDSpan:       clickhouseRes.ChunkIDSpan,
				TailRefreshChunks: clickhouseRes.TailRefreshed,
				IncrementalFromID: clickhouseRes.IncrementalFromID,
			}
		}
		if apiRes != nil {
			st.API = &hn.APIRunState{
				StartID: apiRes.StartID,
				EndID:   apiRes.EndID,
				MaxItem: apiRes.MaxItem,
				IsDelta: apiWasDelta || (apiDeltaFrom > 0 && apiDeltaTo > 0),
			}
		}
		if err := cfg.WriteDownloadState(st); err != nil {
			fmt.Printf("  %s unable to write download state: %v\n", warningStyle.Render("WARN"), err)
		}
		return sourceUsed, nil
	}

	doDelta := func() error {
		dbPath, _ := cmd.Flags().GetString("db")
		deltaFrom, hw, err := cfg.SuggestAPIDeltaStartID(ctx, fromID, dbPath)
		if err != nil {
			return fmt.Errorf("suggest local delta start id: %w", err)
		}
		maxItem, err := cfg.GetMaxItem(ctx)
		if err != nil {
			return fmt.Errorf("get HN maxitem for delta: %w", err)
		}
		deltaTo := toID
		if deltaTo <= 0 || deltaTo > maxItem {
			deltaTo = maxItem
		}
		if hw != nil {
			fmt.Printf("  Local high-water mark: %s (db=%s ch=%s api=%s state=%s)\n",
				labelStyle.Render(formatInt64Exact(hw.MaxKnownID)),
				labelStyle.Render(formatInt64Exact(hw.FromDB)),
				labelStyle.Render(formatInt64Exact(hw.FromCHChunks)),
				labelStyle.Render(formatInt64Exact(hw.FromAPIChunks)),
				labelStyle.Render(formatInt64Exact(hw.FromDownloadState)),
			)
		}
		if deltaFrom > deltaTo {
			fmt.Printf("  %s  no new API delta (local max=%s, remote maxitem=%s)\n",
				successStyle.Render("OK"),
				labelStyle.Render(formatInt64Exact(deltaFrom-1)),
				labelStyle.Render(formatInt64Exact(deltaTo)),
			)
			apiDownloaded = false
			apiWasDelta = true
			apiRes = &hn.APIDownloadResult{
				Dir:     cfg.APIChunksDir(),
				StartID: deltaFrom,
				EndID:   deltaTo,
				MaxItem: maxItem,
			}
			apiDeltaFrom, apiDeltaTo = deltaFrom, deltaTo
			return nil
		}
		fmt.Println(infoStyle.Render(fmt.Sprintf("Downloading HN API delta %d-%d...", deltaFrom, deltaTo)))
		cb := func(p hn.APIDownloadProgress) {
			if p.Complete {
				fmt.Printf("  %s  delta chunks=%d (skipped=%d) ids=%d items=%d\n",
					successStyle.Render("OK"), p.ChunksDone, p.ChunksSkipped, p.IDsProcessed, p.ItemsWritten)
				return
			}
		}
		res, err := cfg.DownloadAPI(ctx, hn.APIDownloadOptions{
			Workers:   workers,
			ChunkSize: chunkSize,
			FromID:    deltaFrom,
			ToID:      deltaTo,
			Force:     false,
		}, cb)
		if err != nil {
			return err
		}
		apiDownloaded = true
		apiRes = res
		apiWasDelta = true
		apiDeltaFrom, apiDeltaTo = deltaFrom, deltaTo
		fmt.Printf("  Dir:        %s\n", labelStyle.Render(res.Dir))
		fmt.Printf("  Range:      %s-%s\n", labelStyle.Render(formatInt64Exact(res.StartID)), labelStyle.Render(formatInt64Exact(res.EndID)))
		return nil
	}

	switch source {
	case "clickhouse":
		if err := doClickHouse(); err != nil {
			return "", err
		}
		if err := doAPIDelta(); err != nil {
			return "", err
		}
		if apiDownloaded {
			return finalize("hybrid")
		}
		return finalize("clickhouse")
	case "parquet":
		if err := doParquet(); err != nil {
			return "", err
		}
		return finalize("parquet")
	case "api":
		if err := doAPI(); err != nil {
			return "", err
		}
		return finalize("api")
	case "delta":
		if err := doDelta(); err != nil {
			return "", err
		}
		if st, err := cfg.LocalStatus(ctx); err == nil && st.CHParquetCount > 0 {
			return finalize("hybrid")
		}
		return finalize("api")
	case "auto":
		err := doClickHouse()
		if err == nil {
			if derr := doAPIDelta(); derr != nil {
				return "", derr
			}
			if apiDownloaded {
				return finalize("hybrid")
			}
			return finalize("clickhouse")
		}
		if noFallback {
			return "", err
		}
		fmt.Printf("%s clickhouse download failed: %v\n", warningStyle.Render("WARN"), err)
		fmt.Println(labelStyle.Render("Falling back to official Hacker News API."))
		if apiErr := doAPI(); apiErr != nil {
			return "", errors.Join(err, apiErr)
		}
		return finalize("api")
	default:
		return "", fmt.Errorf("invalid source %q", source)
	}
}

func parseHNImportSource(s string) (hn.ImportSource, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "auto":
		return hn.ImportSourceAuto, nil
	case "parquet":
		return hn.ImportSourceParquet, nil
	case "clickhouse":
		return hn.ImportSourceClickHouse, nil
	case "hybrid":
		return hn.ImportSourceHybrid, nil
	case "api":
		return hn.ImportSourceAPI, nil
	default:
		return "", fmt.Errorf("invalid --source %q (want auto|clickhouse|hybrid|parquet|api)", s)
	}
}

func padSpaces(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat(" ", n)
}

func ternary[T any](cond bool, yes, no T) T {
	if cond {
		return yes
	}
	return no
}
