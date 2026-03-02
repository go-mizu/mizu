package cli

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	crawl "github.com/go-mizu/mizu/blueprints/search/pkg/crawl"
	"github.com/go-mizu/mizu/blueprints/search/pkg/crawl/store"
	"github.com/go-mizu/mizu/blueprints/search/pkg/hn"
	"github.com/spf13/cobra"
)

func NewHN() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hn",
		Short: "Hacker News dataset sync/import (ClickHouse-only)",
		Long: `Download, incrementally update, and import the full Hacker News dataset
using ClickHouse as the only remote source.

Data is stored at $HOME/data/hn/ by default:
  raw/clickhouse/*.parquet        base checkpoint parquet chunks
  raw/clickhouse_delta/*.parquet  incremental delta parquet chunks (checkpoint-aligned)
  hn.duckdb                       imported DuckDB database

Recommended workflow:
  search hn sync                 # download delta (or full on first run) + import
  search hn sync --every 1m      # keep polling for new items
  search hn compact              # merge delta parquet back into base partitions
  search hn export               # export monthly parquet directly from ClickHouse

Other commands:
  search hn list
  search hn download
  search hn import
  search hn status`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.PersistentFlags().String("dir", "", "Override HN data dir (default: $HOME/data/hn)")
	cmd.PersistentFlags().String("clickhouse-url", "", "Override ClickHouse SQL playground HTTP URL")
	cmd.PersistentFlags().String("clickhouse-user", "", "Override ClickHouse HTTP user")
	cmd.PersistentFlags().String("clickhouse-database", "", "Override ClickHouse database")
	cmd.PersistentFlags().String("clickhouse-table", "", "Override ClickHouse table")
	_ = cmd.PersistentFlags().MarkHidden("dir")
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
	cmd.AddCommand(newHNDomains())
	cmd.AddCommand(newHNRecrawl())
	return cmd
}

func hnConfigFromCmd(cmd *cobra.Command) hn.Config {
	var cfg hn.Config
	// HN commands use a dedicated data directory by default. If the generic root
	// --data flag is not explicitly set, force the HN default ($HOME/data/hn).
	if dataFlag := cmd.Flags().Lookup("data"); dataFlag != nil {
		if dataFlag.Changed {
			if v, _ := cmd.Flags().GetString("data"); strings.TrimSpace(v) != "" {
				cfg.DataDir = v
			}
		} else if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
			cfg.DataDir = filepath.Join(home, "data", "hn")
		}
	}
	if v, _ := cmd.Flags().GetString("dir"); strings.TrimSpace(v) != "" {
		cfg.DataDir = v
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

func hnSignalContext(parent context.Context) (context.Context, context.CancelFunc) {
	ctx, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	return ctx, stop
}


func newHNList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Show remote HN source and local file/database status",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, stop := hnSignalContext(cmd.Context())
			defer stop()
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

			return nil
		},
	}
	cmd.Flags().Bool("no-remote", false, "Skip remote HEAD request and show local status only")
	return cmd
}

func newHNStatus() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show detailed local HN status and remote ClickHouse freshness",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, stop := hnSignalContext(cmd.Context())
			defer stop()
			return showHNStatus(ctx, hnConfigFromCmd(cmd))
		},
	}
}

func newHNDownload() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download",
		Short: "Download/update HN dataset from ClickHouse (auto full or delta)",
		Example: `  search hn download
  search hn download --full
  search hn download --from-id 47000001
  search hn download --parallel 8`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, stop := hnSignalContext(cmd.Context())
			defer stop()
			_, err := runHNDownload(ctx, cmd)
			return err
		},
	}
	cmd.Flags().Bool("full", false, "Force full ClickHouse chunk refresh into raw/clickhouse (otherwise auto delta to raw/clickhouse_delta when local data exists)")
	cmd.Flags().Bool("force", false, "Restart download and overwrite existing local target")
	cmd.Flags().Int64("chunk-id-span", 500000, "ClickHouse checkpoint size for base/delta parquet chunks")
	cmd.Flags().Int("parallel", 4, "Parallel ClickHouse parquet chunk downloads")
	cmd.Flags().Int64("from-id", 0, "Start item id (default: auto full=1, auto delta=local high-watermark+1)")
	cmd.Flags().Int64("to-id", 0, "End item id (default: remote max id)")
	return cmd
}

func newHNImport() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import local HN data into DuckDB",
		Example: `  search hn import
  search hn import --source clickhouse
  search hn import --source hybrid
  search hn import --rebuild`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, stop := hnSignalContext(cmd.Context())
			defer stop()
			cfg := hnConfigFromCmd(cmd)
			sourceStr, _ := cmd.Flags().GetString("source")
			dbPath, _ := cmd.Flags().GetString("db")
			rebuild, _ := cmd.Flags().GetBool("rebuild")
			source, err := parseHNImportSource(sourceStr)
			if err != nil {
				return err
			}
			fmt.Println(infoStyle.Render("Importing Hacker News data into DuckDB..."))
			if strings.TrimSpace(dbPath) == "" {
				dbPath = cfg.WithDefaults().DefaultDBPath()
			}
			fmt.Printf("  Source:      %s\n", labelStyle.Render(strings.ToLower(strings.TrimSpace(sourceStr))))
			fmt.Printf("  DuckDB:      %s\n", labelStyle.Render(dbPath))
			fmt.Printf("  Mode hint:   %s\n", labelStyle.Render(ternary(rebuild, "rebuild", "incremental if DB exists")))
			res, err := runHNImportWithProgress(ctx, cfg, hn.ImportOptions{
				Source:  source,
				DBPath:  dbPath,
				Rebuild: rebuild,
			})
			if err != nil {
				return err
			}
			printHNImportResult(res)
			return nil
		},
	}
	cmd.Flags().String("source", "auto", "Import source: auto|clickhouse|hybrid")
	cmd.Flags().String("db", "", "DuckDB output path (default: $HOME/data/hn/hn.duckdb)")
	cmd.Flags().Bool("rebuild", false, "Force full table rebuild instead of incremental merge when DB exists")
	return cmd
}

func newHNSync() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Continuously sync HN data (download + import) until Ctrl-C",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, stop := hnSignalContext(cmd.Context())
			defer stop()
			once, _ := cmd.Flags().GetBool("once")
			every, _ := cmd.Flags().GetDuration("every")
			maxRuns, _ := cmd.Flags().GetInt("max-runs")
			if once {
				every = 0
				maxRuns = 1
			}
			if every <= 0 {
				if err := runHNSyncOnce(ctx, cmd, 1); err != nil {
					if ctx.Err() != nil {
						fmt.Println(warningStyle.Render("Interrupted HN sync."))
						return nil
					}
					return err
				}
				return nil
			}
			if maxRuns == 0 {
				maxRuns = -1
			}
			fmt.Printf("%s interval=%s runs=%s (Ctrl-C to stop)\n",
				infoStyle.Render("HN sync loop:"),
				labelStyle.Render(every.String()),
				labelStyle.Render(ternary(maxRuns < 0, "infinite", strconv.Itoa(maxRuns))),
			)
			run := 0
			for {
				run++
				started := time.Now()
				fmt.Printf("%s HN sync tick #%d at %s\n", infoStyle.Render("Running"), run, labelStyle.Render(started.Format(time.RFC3339)))
				if err := runHNSyncOnce(ctx, cmd, run); err != nil {
					if ctx.Err() != nil {
						fmt.Println(warningStyle.Render("Interrupted HN sync."))
						return nil
					}
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
				case <-ctx.Done():
					timer.Stop()
					fmt.Println(warningStyle.Render("Interrupted HN sync."))
					return nil
				case <-timer.C:
				}
			}
		},
	}
	cmd.Flags().Bool("full", false, "Force full ClickHouse chunk refresh into raw/clickhouse before import")
	cmd.Flags().Bool("force", false, "Restart download and overwrite existing local target")
	cmd.Flags().Int64("chunk-id-span", 500000, "ClickHouse checkpoint size for base/delta parquet chunks")
	cmd.Flags().Int("parallel", 4, "Parallel ClickHouse parquet chunk downloads")
	cmd.Flags().Int64("from-id", 0, "Start item id (default: auto full=1, auto delta=local high-watermark+1)")
	cmd.Flags().Int64("to-id", 0, "End item id (default: remote max id)")
	cmd.Flags().String("db", "", "DuckDB output path (default: $HOME/data/hn/hn.duckdb)")
	cmd.Flags().Bool("rebuild", false, "Force full table rebuild instead of incremental merge when DB exists")
	cmd.Flags().Bool("once", false, "Run one sync cycle and exit (disables the default continuous ticker)")
	cmd.Flags().Duration("every", time.Minute, "Ticker interval between sync runs (default: 1m)")
	cmd.Flags().Int("max-runs", 0, "Stop after N runs when ticker is enabled (0 = run forever)")
	return cmd
}

func newHNCompact() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compact",
		Short: "Merge local ClickHouse delta parquet chunks back into base parquet files",
		Long: `Reads local raw/clickhouse_delta/*.parquet and merges them into
raw/clickhouse/id_<start>_<end>.parquet partitions using DuckDB.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, stop := hnSignalContext(cmd.Context())
			defer stop()
			cfg := hnConfigFromCmd(cmd)
			fromID, _ := cmd.Flags().GetInt64("from-id")
			toID, _ := cmd.Flags().GetInt64("to-id")
			chunkSpan, _ := cmd.Flags().GetInt64("chunk-id-span")
			compLevel, _ := cmd.Flags().GetInt("compression-level")
			pruneDelta, _ := cmd.Flags().GetBool("prune-delta")

			fmt.Println(infoStyle.Render("Compacting ClickHouse delta parquet into base partitions..."))
			res, err := cfg.CompactDeltaToClickHouseParquet(ctx, hn.CompactOptions{
				FromID:           fromID,
				ToID:             toID,
				ChunkIDSpan:      chunkSpan,
				CompressionLevel: compLevel,
				PruneDelta:       pruneDelta,
			})
			if err != nil {
				return err
			}
			fmt.Printf("  %s  Dir: %s\n", successStyle.Render("OK"), labelStyle.Render(res.Dir))
			fmt.Printf("  Range:      %s\n", labelStyle.Render(formatHNRange(res.FromID, res.ToID)))
			fmt.Printf("  Chunk span: %s\n", labelStyle.Render(formatInt64Exact(res.ChunkIDSpan)))
			fmt.Printf("  Delta rows: %s (%s)\n", labelStyle.Render(formatLargeNumber(res.DeltaRows)), labelStyle.Render(formatInt64Exact(res.DeltaRows)))
			fmt.Printf("  Chunks:     touched=%d written=%d skipped=%d\n", res.ChunksTouched, res.ChunksWritten, res.ChunksSkipped)
			fmt.Printf("  Pruned:     parquet=%d delta_chunks=%d\n", res.FilesPruned, res.DeltaFilesPruned)
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
	cmd.Flags().Int64("from-id", 0, "Compact only delta rows >= this id (default: infer from download state/delta chunks)")
	cmd.Flags().Int64("to-id", 0, "Compact only delta rows <= this id (default: infer from download state/delta chunks)")
	cmd.Flags().Int64("chunk-id-span", 0, "ClickHouse chunk id span (default: auto-detect local chunk span)")
	cmd.Flags().Int("compression-level", 22, "Parquet zstd compression level for rewritten chunk files")
	cmd.Flags().Bool("prune-delta", false, "Delete ClickHouse delta parquet chunk files fully covered by the compacted id range")
	return cmd
}

func newHNExport() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export HN items by month from ClickHouse into parquet files",
		Example: `  search hn export
  search hn export --from-month 2006-10 --to-month 2006-12
  search hn export --out-dir ~/data/hn/export/monthly
  search hn export --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, stop := hnSignalContext(cmd.Context())
			defer stop()
			cfg := hnConfigFromCmd(cmd)
			outDir, _ := cmd.Flags().GetString("out-dir")
			fromMonth, _ := cmd.Flags().GetString("from-month")
			toMonth, _ := cmd.Flags().GetString("to-month")
			force, _ := cmd.Flags().GetBool("force")
			refreshLatest, _ := cmd.Flags().GetBool("refresh-latest")

			fmt.Println(infoStyle.Render("Exporting HN items by month to parquet..."))
			if strings.TrimSpace(outDir) == "" {
				outDir = filepath.Join(cfg.WithDefaults().BaseDir(), "export", "hn", "monthly")
			}
			fmt.Printf("  Output dir:  %s\n", labelStyle.Render(outDir))
			if strings.TrimSpace(fromMonth) != "" || strings.TrimSpace(toMonth) != "" {
				fmt.Printf("  Range:       %s .. %s\n",
					labelStyle.Render(ternary(strings.TrimSpace(fromMonth) != "", fromMonth, "(start)")),
					labelStyle.Render(ternary(strings.TrimSpace(toMonth) != "", toMonth, "(end)")),
				)
			}
			res, err := cfg.ExportMonthlyParquet(ctx, hn.ExportOptions{
				OutDir:        outDir,
				FromMonth:     fromMonth,
				ToMonth:       toMonth,
				Force:         force,
				RefreshLatest: refreshLatest,
				Progress: func(p hn.ExportProgress) {
					switch p.Stage {
					case "start":
						fmt.Printf("  Source:      %s (%s)\n", successStyle.Render(strings.ToUpper(p.SourceUsed)), labelStyle.Render(p.SourceDetail))
						fmt.Printf("  Export to:   %s\n", labelStyle.Render(p.OutDir))
					case "month_start":
						fmt.Printf("  [%d/%d] %s rows=%s -> %s\n",
							p.MonthIndex, p.MonthTotal,
							labelStyle.Render(p.Month),
							labelStyle.Render(formatInt64Exact(p.Rows)),
							labelStyle.Render(p.Path),
						)
					case "month_done":
						if p.Skipped {
							reason := p.SkipReason
							if strings.TrimSpace(reason) == "" {
								reason = "skipped"
							}
							fmt.Printf("      %s %s (%s)\n", labelStyle.Render("skip"), labelStyle.Render(p.Month), labelStyle.Render(reason))
						} else {
							fmt.Printf("      %s %s\n", successStyle.Render("wrote"), labelStyle.Render(p.Month))
						}
					}
				},
			})
			if err != nil {
				return err
			}
			fmt.Printf("  %s  Out dir: %s\n", successStyle.Render("OK"), labelStyle.Render(res.OutDir))
			fmt.Printf("  Source:     %s (%s)\n", successStyle.Render(strings.ToUpper(res.SourceUsed)), labelStyle.Render(res.SourceDetail))
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
				if m.Skipped && strings.TrimSpace(m.SkipReason) != "" {
					fmt.Printf("      reason=%s\n", labelStyle.Render(m.SkipReason))
				}
			}
			return nil
		},
	}
	cmd.Flags().String("out-dir", "", "Output directory (default: $HOME/data/hn/export/hn/monthly)")
	cmd.Flags().String("from-month", "", "Start month inclusive (YYYY-MM)")
	cmd.Flags().String("to-month", "", "End month inclusive (YYYY-MM)")
	cmd.Flags().Bool("force", false, "Rewrite all selected month parquet files even if they already exist")
	cmd.Flags().Bool("refresh-latest", true, "Rewrite the latest month file if it already exists (even when skipping older months)")
	return cmd
}

func newHNDomains() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "domains",
		Short: "Build pages + domains analytics tables from local HN DuckDB",
		Long: `Builds a separate DuckDB database with:
  pages   - one row per HN item that has a URL (normalized host/domain fields)
  domains - aggregated stats per domain/host (counts, first/latest item, etc.)

It reads from the local HN database (hn.duckdb) and persists pages first, then
aggregates domains from pages for faster repeated runs.`,
		Example: `  search hn domains
  search hn domains --out-db ~/data/hn/hn_domains.duckdb
  search hn domains --force-pages`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, stop := hnSignalContext(cmd.Context())
			defer stop()
			cfg := hnConfigFromCmd(cmd)
			srcDB, _ := cmd.Flags().GetString("db")
			outDB, _ := cmd.Flags().GetString("out-db")
			forcePages, _ := cmd.Flags().GetBool("force-pages")
			if strings.TrimSpace(srcDB) == "" {
				srcDB = cfg.WithDefaults().DefaultDBPath()
			}
			if strings.TrimSpace(outDB) == "" {
				outDB = cfg.WithDefaults().DomainsDBPath()
			}
			fmt.Println(infoStyle.Render("Building HN domain analytics database..."))
			fmt.Printf("  Source DB:   %s\n", labelStyle.Render(srcDB))
			fmt.Printf("  Output DB:   %s\n", labelStyle.Render(outDB))
			fmt.Printf("  Strategy:    %s\n", labelStyle.Render("materialize pages first, then aggregate domains"))
			res, err := cfg.BuildDomains(ctx, hn.DomainsOptions{
				SourceDBPath: srcDB,
				OutDBPath:    outDB,
				ForcePages:   forcePages,
				Progress: func(p hn.DomainsProgress) {
					switch p.Stage {
					case "start", "attach":
						fmt.Printf("  %s %s\n", labelStyle.Render(p.Stage+":"), labelStyle.Render(p.Detail))
					case "pages":
						fmt.Printf("  %s %s\n", labelStyle.Render("pages:"), labelStyle.Render(p.Detail))
					case "domains":
						fmt.Printf("  %s %s\n", labelStyle.Render("domains:"), labelStyle.Render(p.Detail))
					case "done":
						fmt.Printf("  %s %s\n", successStyle.Render("done:"), labelStyle.Render(p.Detail))
					}
				},
			})
			if err != nil {
				return err
			}
			fmt.Printf("  %s  Output DB: %s\n", successStyle.Render("OK"), labelStyle.Render(res.OutDBPath))
			fmt.Printf("  Source rows:   %s (%s)\n", successStyle.Render(formatLargeNumber(res.SourceRows)), successStyle.Render(formatInt64Exact(res.SourceRows)))
			fmt.Printf("  Source max id: %s (%s)\n", successStyle.Render(formatLargeNumber(res.SourceMaxID)), successStyle.Render(formatInt64Exact(res.SourceMaxID)))
			if strings.TrimSpace(res.SourceMaxTime) != "" {
				fmt.Printf("  Source max time: %s\n", labelStyle.Render(res.SourceMaxTime))
			}
			fmt.Printf("  Link items:    %s (%s)\n", successStyle.Render(formatLargeNumber(res.SourceLinkItems)), successStyle.Render(formatInt64Exact(res.SourceLinkItems)))
			fmt.Printf("  Pages table:   %s rows (%s)\n", successStyle.Render(formatLargeNumber(res.PagesRows)), successStyle.Render(formatInt64Exact(res.PagesRows)))
			fmt.Printf("  Domains table: %s rows (%s)\n", successStyle.Render(formatLargeNumber(res.DomainsRows)), successStyle.Render(formatInt64Exact(res.DomainsRows)))
			switch {
			case res.PagesBuilt:
				fmt.Printf("  Pages build:   %s\n", successStyle.Render("rebuilt"))
			case res.PagesReused:
				fmt.Printf("  Pages build:   %s\n", labelStyle.Render("reused (source unchanged)"))
			default:
				fmt.Printf("  Pages build:   %s\n", labelStyle.Render("unknown"))
			}
			fmt.Printf("  Domains build: %s\n", successStyle.Render(ternary(res.DomainsBuilt, "rebuilt", "skipped")))
			fmt.Printf("  Elapsed:       %s\n", labelStyle.Render(formatDuration(res.Elapsed)))
			return nil
		},
	}
	cmd.Flags().String("db", "", "Source HN DuckDB path (default: $HOME/data/hn/hn.duckdb)")
	cmd.Flags().String("out-db", "", "Output domains DuckDB path (default: $HOME/data/hn/hn_domains.duckdb)")
	cmd.Flags().Bool("force-pages", false, "Rebuild pages table even if source HN DB appears unchanged")
	return cmd
}

func newHNRecrawl() *cobra.Command {
	var (
		domainsDB           string
		seedDB              string
		limit               int
		maxPerDomain        int
		domainLike          string
		forceSeeds          bool
		engine              string
		workers             int
		maxConnsPerDomain   int
		dnsWorkers          int
		dnsTimeoutMs        int
		timeoutMs           int
		statusOnly          bool
		batchSize           int
		slowDomainMs        int
		domainFailThreshold int
		domainTimeoutMs     int
		domainDeadProbe     int
		domainStallRatio    int
		retryTimeoutMs      int
		noRetry             bool
		writerMode          string
		chunkMode           string
		chunkSize           int
		pprofPort           int
		bodyStoreDir        string
		dbMemMB             int
		dbShards            int
		pass2Workers        int
		segSizeMB           int
		printAutoConfig     bool
	)
	cmd := &cobra.Command{
		Use:   "recrawl",
		Short: "Recrawl URLs extracted from HN domains/pages database",
		Long: `Builds a recrawl seed database from hn_domains.duckdb (pages table), then
runs the high-throughput keepalive recrawl engine with per-domain connection pooling
and adaptive timeouts.`,
		Example: `  search hn recrawl
  search hn recrawl --workers 1500 --max-conns-per-domain 4
  search hn recrawl --domain github.com --limit 1000
  search hn recrawl --engine swarm`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, stop := hnSignalContext(cmd.Context())
			defer stop()
			if pprofPort > 0 {
				go func() { _ = http.ListenAndServe(fmt.Sprintf(":%d", pprofPort), nil) }()
				fmt.Printf("  pprof:         http://localhost:%d/debug/pprof/\n", pprofPort)
			}
			cfg := hnConfigFromCmd(cmd)
			if strings.TrimSpace(domainsDB) == "" {
				domainsDB = cfg.WithDefaults().DomainsDBPath()
			}
			if strings.TrimSpace(seedDB) == "" {
				seedDB = cfg.WithDefaults().RecrawlSeedDBPath()
			}

			fmt.Println(infoStyle.Render("Preparing HN recrawl seed database..."))
			fmt.Printf("  HN domains DB: %s\n", labelStyle.Render(domainsDB))
			fmt.Printf("  Seed DB:       %s\n", labelStyle.Render(seedDB))
			if limit > 0 {
				fmt.Printf("  Limit:         %s\n", labelStyle.Render(formatInt64Exact(int64(limit))))
			}
			if strings.TrimSpace(domainLike) != "" {
				fmt.Printf("  Domain filter: %s\n", labelStyle.Render(domainLike))
			}
			if st, err := os.Stat(domainsDB); err != nil || st.Size() <= 0 {
				srcHNDB := cfg.WithDefaults().DefaultDBPath()
				fmt.Println(infoStyle.Render("HN domains DB not found. Building it first..."))
				fmt.Printf("  Source HN DB:  %s\n", labelStyle.Render(srcHNDB))
				domRes, derr := cfg.BuildDomains(ctx, hn.DomainsOptions{
					SourceDBPath: srcHNDB,
					OutDBPath:    domainsDB,
					ForcePages:   false,
					Progress: func(p hn.DomainsProgress) {
						switch p.Stage {
						case "start", "attach", "pages", "domains":
							fmt.Printf("  %s %s\n", labelStyle.Render("domains:"+p.Stage), labelStyle.Render(p.Detail))
						case "done":
							fmt.Printf("  %s pages=%s domains=%s elapsed=%s\n",
								successStyle.Render("domains:ready"),
								labelStyle.Render(formatInt64Exact(p.Rows)),
								labelStyle.Render(formatInt64Exact(p.Rows2)),
								labelStyle.Render(formatDuration(p.Elapsed)),
							)
						}
					},
				})
				if derr != nil {
					if ctx.Err() != nil {
						fmt.Println(warningStyle.Render("Interrupted while building HN domains database."))
						return nil
					}
					return derr
				}
				fmt.Printf("  Domains DB:    %s (%s rows, %s domains)\n",
					successStyle.Render(domRes.OutDBPath),
					successStyle.Render(formatLargeNumber(domRes.PagesRows)),
					successStyle.Render(formatLargeNumber(domRes.DomainsRows)),
				)
			}
			seedRes, err := cfg.BuildRecrawlSeedDB(ctx, hn.RecrawlSeedOptions{
				DomainsDBPath: domainsDB,
				OutDBPath:     seedDB,
				Limit:         limit,
				MaxPerDomain:  maxPerDomain,
				DomainLike:    domainLike,
				Force:         forceSeeds,
				Progress: func(p hn.RecrawlSeedProgress) {
					switch p.Stage {
					case "attach", "build":
						fmt.Printf("  %s %s\n", labelStyle.Render(p.Stage+":"), labelStyle.Render(p.Detail))
					case "done":
						fmt.Printf("  %s %s rows=%s elapsed=%s\n",
							successStyle.Render("seed:"),
							labelStyle.Render(p.Detail),
							labelStyle.Render(formatInt64Exact(p.Rows)),
							labelStyle.Render(formatDuration(p.Elapsed)),
						)
					}
				},
			})
			if err != nil {
				if ctx.Err() != nil {
					fmt.Println(warningStyle.Render("Interrupted during HN recrawl seed preparation."))
					return nil
				}
				return err
			}
			if maxPerDomain > 0 {
				fmt.Printf("  Max/domain:    %d (stratified sampling)\n", maxPerDomain)
			}
			fmt.Printf("  Seed rows:     %s (%s)\n", successStyle.Render(formatLargeNumber(seedRes.Rows)), successStyle.Render(formatInt64Exact(seedRes.Rows)))
			fmt.Printf("  Seed domains:  %s (%s)\n", successStyle.Render(formatLargeNumber(seedRes.UniqueDomains)), successStyle.Render(formatInt64Exact(seedRes.UniqueDomains)))
			fmt.Printf("  Seed build:    %s\n", labelStyle.Render(formatDuration(seedRes.Elapsed)))
			fmt.Println()

			return runHNRecrawlV3(ctx, cfg, seedRes,
				engine, workers, maxConnsPerDomain, timeoutMs, domainFailThreshold, domainTimeoutMs, domainDeadProbe, domainStallRatio, statusOnly, batchSize, int64(slowDomainMs),
				dnsWorkers, dnsTimeoutMs,
				retryTimeoutMs, noRetry, writerMode,
				chunkMode, chunkSize, bodyStoreDir,
				dbMemMB, dbShards, pass2Workers, segSizeMB, printAutoConfig)
		},
	}
	cmd.Flags().StringVar(&domainsDB, "domains-db", "", "Path to hn_domains DuckDB (default: $HOME/data/hn/hn_domains.duckdb)")
	cmd.Flags().StringVar(&seedDB, "seed-db", "", "Path to recrawl seed DuckDB (default: $HOME/data/hn/recrawl/hn_pages.duckdb)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Max URLs to recrawl from hn_domains.pages (0=all)")
	cmd.Flags().IntVar(&maxPerDomain, "max-per-domain", 10, "Max URLs per domain via stratified sampling (0=no limit)")
	cmd.Flags().StringVar(&domainLike, "domain", "", "Filter host/domain with ILIKE substring match")
	cmd.Flags().BoolVar(&forceSeeds, "force-seeds", false, "Rebuild seed DB from hn_domains.pages even if existing")
	cmd.Flags().StringVar(&engine, "engine", "keepalive", "Crawl engine: keepalive|epoll|rawhttp|swarm")

	cmd.Flags().IntVar(&workers, "workers", -1, "Concurrent domain workers (-1 = auto from hardware)")
	cmd.Flags().IntVar(&maxConnsPerDomain, "max-conns-per-domain", -1, "Max simultaneous connections per domain (-1 = auto from hardware)")
	cmd.Flags().IntVar(&timeoutMs, "timeout", 1000, "Per-request HTTP timeout in milliseconds (pass 1)")
	cmd.Flags().BoolVar(&statusOnly, "status-only", false, "Only check HTTP status, close body immediately (fastest)")
	cmd.Flags().IntVar(&batchSize, "batch-size", 5000, "DB write batch size")
	cmd.Flags().IntVar(&slowDomainMs, "slow-domain-ms", 30_000, "Highlight domains active for longer than this threshold (ms)")
	cmd.Flags().IntVar(&domainFailThreshold, "domain-fail-threshold", -1, "Abandon domain after this many timeout rounds (×conns); -1=engine default (3)")
	cmd.Flags().IntVar(&domainTimeoutMs, "domain-timeout", -1, "Per-domain context deadline in ms; 0=disabled, -1=adaptive (2×sweep time, clamped [30s,10min])")
	cmd.Flags().IntVar(&domainDeadProbe, "domain-dead-probe", 10, "Abandon domain after this many consecutive timeouts with 0 successes (0=disabled)")
	cmd.Flags().IntVar(&domainStallRatio, "domain-stall-ratio", 20, "Abandon domain when timeouts ≥ successes×ratio after dead-probe window (0=disabled; 20 = >95% timeout rate)")

	cmd.Flags().IntVar(&dnsWorkers, "dns-workers", 1000, "Concurrent DNS workers (0=skip DNS pre-resolution)")
	cmd.Flags().IntVar(&dnsTimeoutMs, "dns-timeout", 1500, "DNS lookup timeout in milliseconds")

	cmd.Flags().IntVar(&retryTimeoutMs, "retry-timeout", 15000, "Pass-2 timeout for retrying http_timeout URLs (ms); 0=disabled")
	cmd.Flags().BoolVar(&noRetry, "no-retry", false, "Skip pass-2 retry of timeout URLs (faster; may miss slow-but-live servers)")
	cmd.Flags().StringVar(&writerMode, "writer", "bin", "Result writer backend: bin (default, non-blocking segment→DuckDB drain), duckdb (direct), devnull (benchmark only)")
	cmd.Flags().StringVar(&chunkMode, "chunk-mode", "stream", "Chunk mode: stream|batch|pipeline (stream: sort-then-stream, lower memory; batch: N-domain chunks)")
	cmd.Flags().IntVar(&chunkSize, "chunk-size", 0, "Override batch domain count (0=auto)")
	cmd.Flags().IntVar(&pprofPort, "pprof-port", 0, "Enable pprof HTTP server on this port (0=off)")
	cmd.Flags().StringVar(&bodyStoreDir, "body-store", "", "Body CAS store dir (default: $dataDir/bodies)")
	cmd.Flags().IntVar(&dbMemMB, "db-mem-mb", 0, "DuckDB memory per shard in MB (0=auto: 15% avail RAM / shards)")
	cmd.Flags().IntVar(&dbShards, "db-shards", 0, "ResultDB shard count (0=auto: clamp(CPUs×2, 4, 16))")
	cmd.Flags().IntVar(&pass2Workers, "pass2-workers", 0, "Pass-2 worker count (0=same as pass 1)")
	cmd.Flags().IntVar(&segSizeMB, "seg-size-mb", 0, "Binary segment rotation threshold in MB (0=auto)")
	cmd.Flags().BoolVar(&printAutoConfig, "auto-config", false, "Print auto-configured values and exit without running")
	return cmd
}

// runHNRecrawlV3 runs the v3 recrawl engine for HN seeds.
// It builds the job configuration from HN-specific settings and delegates to runRecrawlJob.
func runHNRecrawlV3(ctx context.Context,
	hnCfg hn.Config,
	seedRes *hn.RecrawlSeedResult,
	engineName string,
	workers, maxConnsPerDomain, timeoutMs, domainFailThreshold, domainTimeoutMs, domainDeadProbe, domainStallRatio int,
	statusOnly bool,
	batchSize int,
	slowDomainMs int64,
	dnsWorkers, dnsTimeoutMs int,
	retryTimeoutMs int,
	noRetry bool,
	writerMode string,
	chunkMode string,
	chunkSize int,
	bodyStoreDir string,
	dbMemMB, dbShards, pass2Workers, segSizeMB int,
	printAutoConfig bool,
) error {

	// ── Hardware profile ──────────────────────────────────────────────────────
	siCache := filepath.Join(hnCfg.WithDefaults().RecrawlDir(), ".sysinfo.json")
	si := crawl.LoadOrGatherSysInfo(siCache, 30*time.Minute)
	fmt.Print(infoStyle.Render("Hardware Profile") + "\n")
	fmt.Print(si.Table())

	// Set GOMEMLIMIT to 75% of available RAM.
	if autoMem := si.MemAvailableMB * 1024 * 1024 * 75 / 100; autoMem > 0 {
		debug.SetMemoryLimit(autoMem)
		fmt.Printf("  GOMEMLIMIT     %s (auto-set from avail RAM)\n", crawl.FormatMB(si.MemAvailableMB*75/100))
	}
	fmt.Println()

	// Auto-config workers and innerN from hardware when not explicitly provided.
	if workers <= 0 {
		autoCfg, reason := crawl.AutoConfigKeepAlive(si, !statusOnly)
		workers = autoCfg.Workers
		if maxConnsPerDomain <= 0 {
			maxConnsPerDomain = autoCfg.MaxConnsPerDomain
		}
		fmt.Printf("  %s  %s\n", infoStyle.Render("Auto-config:"), labelStyle.Render(reason))
	} else if maxConnsPerDomain <= 0 {
		innerN := si.CPUCount * 2
		if innerN < 4 {
			innerN = 4
		}
		if innerN > 16 {
			innerN = 16
		}
		maxConnsPerDomain = innerN
	}

	// Compute adaptive DB + writer settings (0 = auto).
	availMB := int(si.MemAvailableMB)
	if dbShards <= 0 {
		dbShards = crawl.AutoShardCount(si.CPUCount)
	}
	if dbMemMB <= 0 {
		dbMemMB = crawl.AutoDuckMemPerShard(availMB, dbShards)
	}
	if segSizeMB <= 0 {
		segSizeMB = crawl.AutoBinSegMB(availMB)
	}
	p2Workers := pass2Workers
	if p2Workers <= 0 {
		p2Workers = workers
	}
	chanCap := crawl.AutoBinChanCap(availMB, 256)
	fmt.Printf("  DB config:    shards=%d  mem/shard=%dMB  ckpt=%dMB\n", dbShards, dbMemMB, max(dbMemMB/40, 4))
	fmt.Printf("  Bin writer:   seg=%dMB  chan=%d\n", segSizeMB, chanCap)
	fmt.Printf("  Pass 2:       workers=%d  timeout=%dms  domain_timeout=%dms\n\n",
		p2Workers, retryTimeoutMs, retryTimeoutMs*3)

	if printAutoConfig {
		return nil
	}

	// ── Load seeds ────────────────────────────────────────────────────────────
	fmt.Println(infoStyle.Render("Loading seeds into memory..."))
	seeds, err := store.LoadSeedURLs(ctx, seedRes.OutDBPath, int(seedRes.Rows))
	if err != nil {
		return fmt.Errorf("load seed URLs: %w", err)
	}
	fmt.Printf("  Loaded %s seed URLs\n\n", labelStyle.Render(formatInt64Exact(int64(len(seeds)))))

	// ── DNS pre-resolution ────────────────────────────────────────────────────
	totalSeeds := int64(len(seeds)) // total before DNS filtering
	var dnsDeadCount int64
	var dnsCache crawl.DNSCache
	dnsCachePath := filepath.Join(hnCfg.WithDefaults().RecrawlDir(), "dns.duckdb")
	if dnsWorkers > 0 {
		resolver := store.NewDNSResolver(time.Duration(dnsTimeoutMs) * time.Millisecond)
		if cached, _ := resolver.LoadCache(dnsCachePath); cached > 0 {
			fmt.Printf("  DNS cache: loaded %d entries\n", cached)
		}
		hostSet := make(map[string]struct{}, seedRes.UniqueDomains)
		for _, s := range seeds {
			if h := s.Domain; h != "" {
				hostSet[h] = struct{}{}
			}
		}
		hostList := make([]string, 0, len(hostSet))
		for h := range hostSet {
			hostList = append(hostList, h)
		}
		cov := ccDNSCacheCoverage(resolver, hostList)
		if cov.Pending > 0 {
			fmt.Printf("  DNS resolving %s unique hosts (%d workers, %dms timeout)...\n",
				labelStyle.Render(formatInt64Exact(int64(cov.Pending))), dnsWorkers, dnsTimeoutMs)
			resolver.ResolveBatch(ctx, hostList, dnsWorkers, time.Duration(dnsTimeoutMs)*time.Millisecond,
				func(_ store.DNSProgress) {})
			if saveErr := resolver.SaveCache(dnsCachePath); saveErr == nil {
				fmt.Printf("  DNS saved: %s live  %s dead  %s timeout\n",
					labelStyle.Render(formatInt64Exact(resolver.LiveCount())),
					labelStyle.Render(formatInt64Exact(resolver.DeadCount())),
					labelStyle.Render(formatInt64Exact(resolver.TimeoutCount())),
				)
			}
		} else {
			fmt.Printf("  DNS cache: all %d hosts covered\n", len(hostList))
		}
		before := len(seeds)
		filtered := seeds[:0]
		for _, s := range seeds {
			if !resolver.IsDeadOrTimeout(s.Domain) {
				filtered = append(filtered, s)
			}
		}
		seeds = filtered
		dnsDeadCount = int64(before - len(seeds))
		if dnsDeadCount > 0 {
			fmt.Printf("  Filtered %s dead/timeout seeds → %s remaining\n\n",
				labelStyle.Render(formatInt64Exact(dnsDeadCount)),
				labelStyle.Render(formatInt64Exact(int64(len(seeds)))),
			)
		}
		dnsCache = resolver.Cache()
	}
	if dnsCache == nil {
		dnsCache = &crawl.NoopDNS{}
	}

	// ── Build job config ──────────────────────────────────────────────────────
	resultDir := filepath.Join(hnCfg.WithDefaults().RecrawlDir(), "results")
	failedDBPath := filepath.Join(hnCfg.WithDefaults().RecrawlDir(), "failed.duckdb")

	jcfg := crawl.JobConfig{
		Engine:              engineName,
		Workers:             workers,
		MaxConnsPerDomain:   maxConnsPerDomain,
		Timeout:             time.Duration(timeoutMs) * time.Millisecond,
		RetryTimeout:        time.Duration(retryTimeoutMs) * time.Millisecond,
		NoRetry:             noRetry,
		StatusOnly:          statusOnly,
		InsecureTLS:         true,
		DomainFailThreshold: domainFailThreshold,
		DomainDeadProbe:     domainDeadProbe,
		DomainStallRatio:    domainStallRatio,
		BatchSize:           batchSize,
		ChunkMode:           chunkMode,
		ChunkSize:           chunkSize,
		SeedPath:            seedRes.OutDBPath,
		Pass2Workers:        p2Workers,
	}
	if domainTimeoutMs != 0 {
		jcfg.DomainTimeout = time.Duration(domainTimeoutMs) * time.Millisecond // -1ms = adaptive
	}

	return runRecrawlJob(ctx, recrawlJobArgs{
		Seeds:        seeds,
		DNSCache:     dnsCache,
		JobCfg:       jcfg,
		ResultDir:    resultDir,
		FailedDBPath: failedDBPath,
		WriterMode:   writerMode,
		SlowDomainMs: slowDomainMs,
		SegSizeMB:    segSizeMB,
		BodyStoreDir: bodyStoreDir,
		DBShards:     dbShards,
		DBMemMB:      dbMemMB,
		SysInfo:      si,
		TotalSeeds:   totalSeeds,
		DNSDeadCount: dnsDeadCount,
	})
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
	fmt.Printf("  Import DB:   %s\n", labelStyle.Render(ternary(strings.TrimSpace(dbPath) != "", dbPath, cfg.WithDefaults().DefaultDBPath())))
	res, err := runHNImportWithProgress(ctx, cfg, hn.ImportOptions{
		Source:  importSource,
		DBPath:  dbPath,
		Rebuild: rebuild,
	})
	if err != nil {
		return err
	}
	printHNImportResult(res)
	return nil
}

func runHNImportWithProgress(ctx context.Context, cfg hn.Config, opts hn.ImportOptions) (*hn.ImportResult, error) {
	started := time.Now()
	var mu sync.Mutex
	var last hn.ImportProgress
	haveLast := false
	downstream := opts.Progress
	opts.Progress = func(p hn.ImportProgress) {
		mu.Lock()
		last = p
		haveLast = true
		mu.Unlock()
		if downstream != nil {
			downstream(p)
		} else {
			printHNImportProgress(p)
		}
	}

	done := make(chan struct{})
	go func() {
		t := time.NewTicker(10 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case <-t.C:
				mu.Lock()
				p := last
				ok := haveLast
				mu.Unlock()
				if !ok || strings.EqualFold(strings.TrimSpace(p.Stage), "done") {
					continue
				}
				stage := strings.TrimSpace(p.Stage)
				if stage == "" {
					stage = "import"
				}
				msg := strings.TrimSpace(p.Detail)
				if msg == "" {
					msg = stage
				}
				fmt.Printf("  %s %-11s %s (%s)\n",
					labelStyle.Render("import:"),
					labelStyle.Render("waiting"),
					labelStyle.Render(fmt.Sprintf("%s: %s", stage, msg)),
					labelStyle.Render(formatDuration(time.Since(started))),
				)
			}
		}
	}()
	defer close(done)

	return cfg.Import(ctx, opts)
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

func printHNImportProgress(p hn.ImportProgress) {
	stage := strings.TrimSpace(p.Stage)
	if stage == "" {
		stage = "import"
	}
	msg := strings.TrimSpace(p.Detail)
	if msg == "" {
		msg = stage
	}
	fmt.Printf("  %s %-11s %s (%s)\n",
		labelStyle.Render("import:"),
		labelStyle.Render(stage),
		labelStyle.Render(msg),
		labelStyle.Render(formatDuration(p.Elapsed)),
	)
	if p.Stage == "count" && p.Rows > 0 && strings.Contains(strings.ToLower(p.Detail), "ready") {
		fmt.Printf("  %s rows=%s (%s)\n",
			labelStyle.Render("import:"),
			successStyle.Render(formatLargeNumber(p.Rows)),
			labelStyle.Render(formatInt64Exact(p.Rows)),
		)
	}
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
	fmt.Printf("  CH base:     %d file(s), %s\n", st.CHParquetCount, formatBytes(st.CHParquetBytes))
	if st.CHParquetCount > 0 {
		if st.CHParquetSpan > 0 {
			fmt.Printf("    span:      %s ids/chunk\n", labelStyle.Render(formatInt64Exact(st.CHParquetSpan)))
		}
		fmt.Printf("    range:     %s\n", labelStyle.Render(formatHNRange(st.CHParquetMinID, st.CHParquetMaxID)))
		if st.CHParquetRows > 0 {
			fmt.Printf("    rows:      %s (%s)\n", successStyle.Render(formatLargeNumber(st.CHParquetRows)), labelStyle.Render(formatInt64Exact(st.CHParquetRows)))
		}
		if strings.TrimSpace(st.CHParquetMaxTime) != "" {
			fmt.Printf("    max time:  %s\n", labelStyle.Render(st.CHParquetMaxTime))
		}
	}
	fmt.Printf("  CH delta:    %d file(s), %s\n", st.CHDeltaCount, formatBytes(st.CHDeltaBytes))
	if st.CHDeltaCount > 0 {
		if st.CHDeltaSpan > 0 {
			fmt.Printf("    span:      %s ids/chunk\n", labelStyle.Render(formatInt64Exact(st.CHDeltaSpan)))
		}
		fmt.Printf("    range:     %s\n", labelStyle.Render(formatHNRange(st.CHDeltaMinID, st.CHDeltaMaxID)))
		fmt.Printf("    rows:      %s (%s)\n", successStyle.Render(formatLargeNumber(st.CHDeltaRows)), labelStyle.Render(formatInt64Exact(st.CHDeltaRows)))
		if strings.TrimSpace(st.CHDeltaMaxTime) != "" {
			fmt.Printf("    max time:  %s\n", labelStyle.Render(st.CHDeltaMaxTime))
		}
	}
	if st.DBExists {
		fmt.Printf("  DuckDB:      %s (%s)\n", successStyle.Render(filepath.Base(st.DBPath)), formatBytes(st.DBSize))
		if st.DBRows > 0 {
			fmt.Printf("    rows:      %s (%s)\n", successStyle.Render(formatLargeNumber(st.DBRows)), labelStyle.Render(formatInt64Exact(st.DBRows)))
		}
		if st.DBMaxID > 0 || st.DBMinID > 0 {
			fmt.Printf("    range:     %s\n", labelStyle.Render(formatHNRange(st.DBMinID, st.DBMaxID)))
		}
		if strings.TrimSpace(st.DBMaxTime) != "" {
			fmt.Printf("    max time:  %s\n", labelStyle.Render(st.DBMaxTime))
		}
		if len(st.DBTypes) > 0 {
			fmt.Printf("    types:     ")
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

func showHNStatus(ctx context.Context, cfg hn.Config) error {
	if err := showHNLocalStatus(ctx, cfg); err != nil {
		return err
	}
	fmt.Println()
	fmt.Println(infoStyle.Render("Remote ClickHouse status:"))
	remote, err := cfg.ClickHouseInfo(ctx)
	if err != nil {
		fmt.Printf("  %s %v\n", warningStyle.Render("WARN"), err)
		return nil
	}
	fmt.Printf("  Endpoint:    %s\n", labelStyle.Render(remote.BaseURL))
	fmt.Printf("  Table:       %s.%s\n", successStyle.Render(remote.Database), successStyle.Render(remote.Table))
	fmt.Printf("  Rows:        %s (%s)\n", successStyle.Render(formatLargeNumber(remote.Count)), successStyle.Render(formatInt64Exact(remote.Count)))
	fmt.Printf("  Max ID:      %s (%s)\n", successStyle.Render(formatLargeNumber(remote.MaxID)), successStyle.Render(formatInt64Exact(remote.MaxID)))
	fmt.Printf("  Max Time:    %s\n", labelStyle.Render(remote.MaxTime))
	fmt.Printf("  Checked:     %s\n", labelStyle.Render(remote.CheckedAt.Format("2006-01-02 15:04:05 MST")))

	hw, hwErr := cfg.LocalHighWatermark(ctx, "")
	if hwErr == nil && hw != nil {
		fmt.Println()
		fmt.Println(labelStyle.Render("Freshness:"))
		fmt.Printf("  Local max:   %s (db=%s base=%s delta=%s state=%s)\n",
			labelStyle.Render(formatInt64Exact(hw.MaxKnownID)),
			labelStyle.Render(formatInt64Exact(hw.FromDB)),
			labelStyle.Render(formatInt64Exact(hw.FromCHChunks)),
			labelStyle.Render(formatInt64Exact(hw.FromCHDelta)),
			labelStyle.Render(formatInt64Exact(hw.FromDownloadState)),
		)
		diff := remote.MaxID - hw.MaxKnownID
		switch {
		case diff > 0:
			fmt.Printf("  Status:      %s %s new item id(s) available on ClickHouse\n",
				warningStyle.Render("BEHIND"),
				warningStyle.Render(formatInt64Exact(diff)),
			)
		case diff == 0:
			fmt.Printf("  Status:      %s local data is up to date by max id\n", successStyle.Render("UP-TO-DATE"))
		default:
			fmt.Printf("  Status:      %s local max id is ahead of remote by %s (transient/source lag?)\n",
				warningStyle.Render("AHEAD"),
				labelStyle.Render(formatInt64Exact(-diff)),
			)
		}
	}
	return nil
}

func runHNDownload(ctx context.Context, cmd *cobra.Command) (string, error) {
	cfg := hnConfigFromCmd(cmd)
	full, _ := cmd.Flags().GetBool("full")
	force, _ := cmd.Flags().GetBool("force")
	chunkIDSpan, _ := cmd.Flags().GetInt64("chunk-id-span")
	parallel, _ := cmd.Flags().GetInt("parallel")
	fromID, _ := cmd.Flags().GetInt64("from-id")
	toID, _ := cmd.Flags().GetInt64("to-id")
	dbPath, _ := cmd.Flags().GetString("db")

	if !cmd.Flags().Changed("chunk-id-span") {
		if span, ok := cfg.DetectLocalClickHouseChunkSpan(); ok && span > 0 && span != chunkIDSpan {
			fmt.Printf("  %s using local ClickHouse chunk span %d (detected from existing files)\n", labelStyle.Render("Info:"), span)
			chunkIDSpan = span
		}
	}
	if chunkIDSpan <= 0 {
		chunkIDSpan = 500_000
	}
	fmt.Printf("  Download cfg: chunk_span=%s parallel=%d force=%t full=%t\n",
		labelStyle.Render(formatInt64Exact(chunkIDSpan)),
		parallel,
		force,
		full,
	)
	if fromID > 0 || toID > 0 {
		fmt.Printf("  Requested range: %s\n", labelStyle.Render(formatHNRange(fromID, toID)))
	}

	printCHProgress := func(prefix string) func(hn.ClickHouseDownloadProgress) {
		return func(p hn.ClickHouseDownloadProgress) {
			if p.Complete {
				fmt.Printf("  %s  chunks=%d/%d (skipped=%d) bytes=%s avg=%s elapsed=%s\n",
					successStyle.Render("OK"), p.ChunksDone, p.ChunksTotal, p.ChunksSkipped,
					formatBytes(p.BytesDone), formatBytesPerSec(p.OverallSpeedBPS), formatDuration(p.Elapsed))
				return
			}
			if p.Detail != "" {
				fmt.Printf("  %s chunk %d/%d [%d-%d] %s (%s) active=%d total=%s avg=%s\n",
					prefix, p.ChunksDone, p.ChunksTotal, p.ChunkStart, p.ChunkEnd, p.Detail,
					formatBytes(p.ChunkBytes), p.ChunksActive, formatBytes(p.BytesDone), formatBytesPerSec(p.OverallSpeedBPS))
				return
			}
			fmt.Printf("  %s chunk %d/%d [%d-%d] %s in %s (%s) active=%d total=%s avg=%s\n",
				prefix, p.ChunksDone, p.ChunksTotal, p.ChunkStart, p.ChunkEnd,
				formatBytes(p.ChunkBytes), formatDuration(p.ChunkElapsed), formatBytesPerSec(p.ChunkSpeedBPS),
				p.ChunksActive, formatBytes(p.BytesDone), formatBytesPerSec(p.OverallSpeedBPS))
		}
	}
	printCHResult := func(label string, res *hn.ClickHouseDownloadResult) {
		if res == nil {
			return
		}
		fmt.Printf("  %s dir:     %s\n", label, labelStyle.Render(res.Dir))
		fmt.Printf("  %s range:   %s\n", label, labelStyle.Render(formatHNRange(res.StartID, res.EndID)))
		if res.RemoteInfo != nil {
			fmt.Printf("  Remote rows: %s (%s) max_id=%s (%s) max_time=%s\n",
				successStyle.Render(formatLargeNumber(res.RemoteInfo.Count)),
				successStyle.Render(formatInt64Exact(res.RemoteInfo.Count)),
				successStyle.Render(formatLargeNumber(res.RemoteInfo.MaxID)),
				successStyle.Render(formatInt64Exact(res.RemoteInfo.MaxID)),
				labelStyle.Render(res.RemoteInfo.MaxTime))
		}
	}

	var baseRes, deltaRes *hn.ClickHouseDownloadResult
	localSt, _ := cfg.LocalStatus(ctx)
	hasBase := localSt != nil && localSt.CHParquetCount > 0

	if full || !hasBase {
		fmt.Println(infoStyle.Render("Downloading HN base parquet chunks from ClickHouse..."))
		res, err := cfg.DownloadClickHouseParquet(ctx, hn.ClickHouseDownloadOptions{
			FromID:      ternary(fromID > 0, fromID, int64(1)),
			ToID:        toID,
			ChunkIDSpan: chunkIDSpan,
			Parallelism: parallel,
			Force:       force || full,
		}, printCHProgress("base"))
		if err != nil {
			return "", err
		}
		baseRes = res
		printCHResult("Base", res)
	} else {
		deltaFrom, hw, err := cfg.SuggestClickHouseDeltaStartID(ctx, fromID, dbPath)
		if err != nil {
			return "", fmt.Errorf("suggest clickhouse delta start id: %w", err)
		}
		remote, err := cfg.ClickHouseInfo(ctx)
		if err != nil {
			return "", err
		}
		if hw != nil {
			fmt.Printf("  Local high-water mark: %s (db=%s base=%s delta=%s state=%s)\n",
				labelStyle.Render(formatInt64Exact(hw.MaxKnownID)),
				labelStyle.Render(formatInt64Exact(hw.FromDB)),
				labelStyle.Render(formatInt64Exact(hw.FromCHChunks)),
				labelStyle.Render(formatInt64Exact(hw.FromCHDelta)),
				labelStyle.Render(formatInt64Exact(hw.FromDownloadState)))
		}
		deltaTo := toID
		if deltaTo <= 0 || deltaTo > remote.MaxID {
			deltaTo = remote.MaxID
		}
		if deltaFrom > deltaTo {
			fmt.Printf("  %s  no new ClickHouse delta (local max=%s, remote max=%s)\n",
				successStyle.Render("OK"),
				labelStyle.Render(formatInt64Exact(deltaFrom-1)),
				labelStyle.Render(formatInt64Exact(deltaTo)))
			deltaRes = &hn.ClickHouseDownloadResult{Dir: cfg.ClickHouseDeltaParquetDir(), StartID: deltaFrom, EndID: deltaTo, RemoteMaxID: remote.MaxID, RemoteCount: remote.Count, ChunkIDSpan: chunkIDSpan, IncrementalFromID: deltaFrom, RemoteInfo: remote}
		} else {
			fmt.Println(infoStyle.Render("Downloading HN delta parquet chunks from ClickHouse (checkpoint-aligned)..."))
			res, err := cfg.DownloadClickHouseParquet(ctx, hn.ClickHouseDownloadOptions{
				FromID:            deltaFrom,
				ToID:              deltaTo,
				ChunkIDSpan:       chunkIDSpan,
				Parallelism:       parallel,
				RefreshTailChunks: 0,
				OutputDir:         cfg.ClickHouseDeltaParquetDir(),
				AlignCheckpoints:  true,
				Force:             force,
			}, printCHProgress("delta"))
			if err != nil {
				return "", err
			}
			deltaRes = res
			printCHResult("Delta", res)
		}
	}

	state := &hn.DownloadState{SourceUsed: "clickhouse"}
	if baseRes != nil {
		state.ClickHouse = &hn.ClickHouseRunState{StartID: baseRes.StartID, EndID: baseRes.EndID, RemoteMaxID: baseRes.RemoteMaxID, RemoteCount: baseRes.RemoteCount, ChunkIDSpan: baseRes.ChunkIDSpan, TailRefreshChunks: baseRes.TailRefreshed, IncrementalFromID: baseRes.IncrementalFromID}
	}
	if deltaRes != nil {
		state.Delta = &hn.ClickHouseRunState{StartID: deltaRes.StartID, EndID: deltaRes.EndID, RemoteMaxID: deltaRes.RemoteMaxID, RemoteCount: deltaRes.RemoteCount, ChunkIDSpan: deltaRes.ChunkIDSpan, TailRefreshChunks: deltaRes.TailRefreshed, IncrementalFromID: deltaRes.IncrementalFromID}
	}
	if err := cfg.WriteDownloadState(state); err != nil {
		fmt.Printf("  %s unable to write download state: %v\n", warningStyle.Render("WARN"), err)
	}
	finalSt, _ := cfg.LocalStatus(ctx)
	if finalSt != nil && finalSt.CHParquetCount > 0 && finalSt.CHDeltaCount > 0 {
		return "hybrid", nil
	}
	return "clickhouse", nil
}

func parseHNImportSource(s string) (hn.ImportSource, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "auto":
		return hn.ImportSourceAuto, nil
	case "clickhouse":
		return hn.ImportSourceClickHouse, nil
	case "hybrid":
		return hn.ImportSourceHybrid, nil
	default:
		return "", fmt.Errorf("invalid --source %q (want auto|clickhouse|hybrid)", s)
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
