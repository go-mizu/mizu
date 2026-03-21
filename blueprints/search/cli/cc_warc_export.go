package cli

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/cc"
	warcpkg "github.com/go-mizu/mizu/blueprints/search/pkg/warc"
	warcmd "github.com/go-mizu/mizu/blueprints/search/pkg/warc_md"
	"github.com/google/uuid"
	"github.com/parquet-go/parquet-go"
	"github.com/parquet-go/parquet-go/compress/zstd"
	"github.com/spf13/cobra"
)

type ccWARCExportRow struct {
	DocID          string `parquet:"doc_id"`
	URL            string `parquet:"url"`
	Host           string `parquet:"host"`
	CrawlDate      string `parquet:"crawl_date"`
	WARCRecordID   string `parquet:"warc_record_id"`
	WARCRefersTo   string `parquet:"warc_refers_to"`
	HTMLLength     int64  `parquet:"html_length"`
	MarkdownLength int64  `parquet:"markdown_length"`
	Markdown       string `parquet:"markdown"`
}

func newCCWarcExport() *cobra.Command {
	var (
		crawlID string
		fileIdx string
		force   bool
	)

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export packed markdown WARC shards to parquet",
		Long: `Read warc_md/{shard}.md.warc.gz and write export/repo/data/{shard}.parquet.

The parquet schema preserves the markdown body and all WARC headers via
warc_headers_json, while also exposing the common WARC fields as columns.`,
		Example: `  search cc warc export --file 0
  search cc warc export --crawl CC-MAIN-2026-08 --file 0-9
  search cc warc export --file all --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCCWarcExport(cmd.Context(), crawlID, fileIdx, force)
		},
	}

	cmd.Flags().StringVar(&crawlID, "crawl", "", "Crawl ID (default: latest)")
	cmd.Flags().StringVar(&fileIdx, "file", "0", "File index, range (0-9), or all")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing parquet files")
	return cmd
}

func runCCWarcExport(ctx context.Context, crawlID, fileIdx string, force bool) error {
	resolvedID, note, err := ccResolveCrawlID(ctx, crawlID)
	if err != nil {
		return fmt.Errorf("resolving crawl: %w", err)
	}
	crawlID = resolvedID
	if note != "" {
		ccPrintDefaultCrawlResolution(crawlID, note)
	}

	client := cc.NewClient("", 4)
	paths, err := client.DownloadManifest(ctx, crawlID, "warc.paths.gz")
	if err != nil {
		return fmt.Errorf("manifest: %w", err)
	}
	selected, err := ccParseFileSelector(fileIdx, len(paths))
	if err != nil {
		return fmt.Errorf("--file: %w", err)
	}

	cfg := warcmd.DefaultConfig(crawlID)
	repoRoot := ccDefaultExportRepoRoot(crawlID)
	dataDir := filepath.Join(repoRoot, "data", crawlID)
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return fmt.Errorf("create export data dir: %w", err)
	}

	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Common Crawl WARC Export"))
	fmt.Println()
	fmt.Printf("  Crawl     %s\n", labelStyle.Render(crawlID))
	fmt.Printf("  Files     %s\n", infoStyle.Render(ccFmtInt64(int64(len(selected)))))
	fmt.Printf("  Input     %s\n", labelStyle.Render(cfg.WARCMdDir()))
	fmt.Printf("  Output    %s\n", labelStyle.Render(dataDir))
	fmt.Println()

	var exported, skipped int64
	var totalHTMLBytes, totalMdBytes int64
	for i, idx := range selected {
		shard := warcIndexFromPath(paths[idx], idx)
		inPath := filepath.Join(cfg.WARCMdDir(), shard+".md.warc.gz")
		outPath := filepath.Join(dataDir, shard+".parquet")

		if !fileExists(inPath) {
			return fmt.Errorf("input shard not found: %s (run `search cc warc pack --file %d` first)", inPath, idx)
		}
		if fileExists(outPath) && !force {
			fmt.Printf("  [%d/%d] %s  %s\n", i+1, len(selected), labelStyle.Render(shard), warningStyle.Render("skipped (exists)"))
			skipped++
			continue
		}

		rows, htmlBytes, mdBytes, err := exportWARCMdShardToParquet(inPath, outPath, nil)
		if err != nil {
			return fmt.Errorf("export shard %s: %w", shard, err)
		}
		totalHTMLBytes += htmlBytes
		totalMdBytes += mdBytes

		// Get parquet file size
		pqSize := int64(0)
		if fi, err := os.Stat(outPath); err == nil {
			pqSize = fi.Size()
		}
		warcSize := int64(0)
		if fi, err := os.Stat(inPath); err == nil {
			warcSize = fi.Size()
		}

		fmt.Printf("  [%d/%d] %s  %s rows  %s  warc %s -> parquet %s (-%s%%)\n",
			i+1, len(selected),
			labelStyle.Render(shard),
			infoStyle.Render(ccFmtInt64(rows)),
			successStyle.Render(filepath.Base(outPath)),
			infoStyle.Render(ccFmtBytes(warcSize)),
			infoStyle.Render(ccFmtBytes(pqSize)),
			infoStyle.Render(ccPctReduction(warcSize, pqSize)))
		exported++
	}

	fmt.Println()
	fmt.Printf("  Exported  %s\n", infoStyle.Render(ccFmtInt64(exported)))
	if skipped > 0 {
		fmt.Printf("  Skipped   %s\n", warningStyle.Render(ccFmtInt64(skipped)))
	}
	if totalHTMLBytes > 0 {
		reduction := float64(totalHTMLBytes-totalMdBytes) / float64(totalHTMLBytes) * 100
		fmt.Printf("  HTML      %s -> Markdown %s  (-%s%%)\n",
			infoStyle.Render(ccFmtBytes(totalHTMLBytes)),
			infoStyle.Render(ccFmtBytes(totalMdBytes)),
			infoStyle.Render(fmt.Sprintf("%.1f", reduction)))
	}
	fmt.Printf("  Repo root %s\n", labelStyle.Render(repoRoot))
	return nil
}

// exportProgressFn is called periodically during export with current row count and elapsed time.
type exportProgressFn func(rows int64, elapsed time.Duration)

func exportWARCMdShardToParquet(inPath, outPath string, progressFn exportProgressFn) (rows int64, htmlBytes int64, mdBytes int64, err error) {
	fail := func(e error) (int64, int64, int64, error) { return 0, 0, 0, e }

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fail(fmt.Errorf("mkdir parquet dir: %w", err))
	}

	in, err := os.Open(inPath)
	if err != nil {
		return fail(fmt.Errorf("open input: %w", err))
	}
	defer in.Close()

	tmpPath := outPath + ".tmp"
	_ = os.Remove(tmpPath)
	out, err := os.Create(tmpPath)
	if err != nil {
		return fail(fmt.Errorf("create parquet: %w", err))
	}

	pw := parquet.NewGenericWriter[ccWARCExportRow](out,
		parquet.Compression(&zstd.Codec{Level: zstd.SpeedBestCompression}),
		parquet.MaxRowsPerRowGroup(100_000),
		parquet.PageBufferSize(2*1024*1024), // 2 MB: saves ~54 MB vs 8 MB (9 cols × 6 MB)
	)

	reader := warcpkg.NewReader(in)
	batch := make([]ccWARCExportRow, 0, 1000)
	var rowsWritten, totalHTML, totalMd int64
	start := time.Now()

	// Progress ticker goroutine
	var stopTicker chan struct{}
	if progressFn != nil {
		stopTicker = make(chan struct{})
		go func() {
			tick := time.NewTicker(500 * time.Millisecond)
			defer tick.Stop()
			for {
				select {
				case <-tick.C:
					progressFn(rowsWritten, time.Since(start))
				case <-stopTicker:
					return
				}
			}
		}()
	}

	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		if _, err := pw.Write(batch); err != nil {
			return err
		}
		batch = batch[:0]
		return nil
	}

	cleanup := func() {
		if stopTicker != nil {
			close(stopTicker)
		}
		pw.Close()
		out.Close()
		_ = os.Remove(tmpPath)
	}

	for reader.Next() {
		rec := reader.Record()
		body, err := io.ReadAll(rec.Body)
		if err != nil {
			cleanup()
			return fail(fmt.Errorf("read record body: %w", err))
		}

		targetURL := strings.TrimSpace(rec.Header.TargetURI())
		crawlDate := ""
		if ts := rec.Header.Date(); !ts.IsZero() {
			crawlDate = ts.Format(time.RFC3339)
		}
		htmlLen := ccParseHTMLLength(rec.Header.Get("X-HTML-Length"))
		totalHTML += htmlLen
		totalMd += int64(len(body))
		batch = append(batch, ccWARCExportRow{
			DocID:          ccURLToDocID(targetURL),
			URL:            targetURL,
			Host:           ccHostFromURL(targetURL),
			CrawlDate:      crawlDate,
			WARCRecordID:   rec.Header.RecordID(),
			WARCRefersTo:   rec.Header.RefersTo(),
			HTMLLength:     htmlLen,
			MarkdownLength: int64(len(body)),
			Markdown:       sanitizeUTF8(string(body)),
		})
		rowsWritten++
		if len(batch) >= 1000 {
			if err := flush(); err != nil {
				cleanup()
				return fail(fmt.Errorf("write parquet: %w", err))
			}
		}
	}
	if err := reader.Err(); err != nil {
		cleanup()
		return fail(fmt.Errorf("read warc: %w", err))
	}
	if err := flush(); err != nil {
		cleanup()
		return fail(fmt.Errorf("write parquet: %w", err))
	}
	if stopTicker != nil {
		close(stopTicker)
		stopTicker = nil
	}
	if err := pw.Close(); err != nil {
		out.Close()
		_ = os.Remove(tmpPath)
		return fail(fmt.Errorf("close parquet writer: %w", err))
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fail(fmt.Errorf("close parquet file: %w", err))
	}
	if err := os.Rename(tmpPath, outPath); err != nil {
		_ = os.Remove(tmpPath)
		return fail(fmt.Errorf("finalize parquet: %w", err))
	}
	return rowsWritten, totalHTML, totalMd, nil
}

// ccURLToDocID returns a deterministic UUID v5 (SHA-1, URL namespace) for the
// canonical URL. This makes doc_id stable and deduplication-friendly: the same
// URL always produces the same doc_id regardless of crawl date or WARC shard.
//
// Formula: doc_id = UUID5(NamespaceURL, canonicalURL)
// Example: "https://example.com/page" → "5a3d2b1c-..."
func ccURLToDocID(rawURL string) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(rawURL)).String()
}

func ccParseHTMLLength(s string) int64 {
	if s == "" {
		return 0
	}
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0
	}
	return n
}

func ccHostFromURL(raw string) string {
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}

func sanitizeUTF8(s string) string {
	if strings.ToValidUTF8(s, "") == s {
		return s
	}
	return strings.ToValidUTF8(s, "\uFFFD")
}

func ccPctReduction(from, to int64) string {
	if from <= 0 {
		return "0"
	}
	return fmt.Sprintf("%.0f", float64(from-to)/float64(from)*100)
}
