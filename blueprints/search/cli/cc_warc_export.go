package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/cc"
	warcpkg "github.com/go-mizu/mizu/blueprints/search/pkg/warc"
	warcmd "github.com/go-mizu/mizu/blueprints/search/pkg/warc_md"
	"github.com/parquet-go/parquet-go"
	"github.com/spf13/cobra"
)

type ccWARCExportRow struct {
	DocID           string `parquet:"doc_id"`
	URL             string `parquet:"url"`
	Host            string `parquet:"host"`
	CrawlDate       string `parquet:"crawl_date"`
	WARCType        string `parquet:"warc_type"`
	WARCRecordID    string `parquet:"warc_record_id"`
	WARCRefersTo    string `parquet:"warc_refers_to"`
	ContentType     string `parquet:"content_type"`
	HTMLLength      int64  `parquet:"html_length"`
	MarkdownLength  int64  `parquet:"markdown_length"`
	WARCHeadersJSON string `parquet:"warc_headers_json"`
	MarkdownBody    string `parquet:"markdown_body"`
	SourceWARCFile  string `parquet:"source_warc_file"`
	SourceFileIndex int32  `parquet:"source_file_index"`
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

		rows, err := exportWARCMdShardToParquet(inPath, outPath, idx)
		if err != nil {
			return fmt.Errorf("export shard %s: %w", shard, err)
		}
		fmt.Printf("  [%d/%d] %s  %s rows  %s\n",
			i+1, len(selected),
			labelStyle.Render(shard),
			infoStyle.Render(ccFmtInt64(rows)),
			successStyle.Render(filepath.Base(outPath)))
		exported++
	}

	fmt.Println()
	fmt.Printf("  Exported  %s\n", infoStyle.Render(ccFmtInt64(exported)))
	if skipped > 0 {
		fmt.Printf("  Skipped   %s\n", warningStyle.Render(ccFmtInt64(skipped)))
	}
	fmt.Printf("  Repo root %s\n", labelStyle.Render(repoRoot))
	return nil
}

func exportWARCMdShardToParquet(inPath, outPath string, fileIndex int) (int64, error) {
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return 0, fmt.Errorf("mkdir parquet dir: %w", err)
	}

	in, err := os.Open(inPath)
	if err != nil {
		return 0, fmt.Errorf("open input: %w", err)
	}
	defer in.Close()

	tmpPath := outPath + ".tmp"
	_ = os.Remove(tmpPath)
	out, err := os.Create(tmpPath)
	if err != nil {
		return 0, fmt.Errorf("create parquet: %w", err)
	}

	pw := parquet.NewGenericWriter[ccWARCExportRow](out,
		parquet.Compression(&parquet.Zstd),
		parquet.MaxRowsPerRowGroup(100_000),
		parquet.PageBufferSize(8*1024*1024),
	)

	reader := warcpkg.NewReader(in)
	batch := make([]ccWARCExportRow, 0, 1000)
	var rowsWritten int64

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

	for reader.Next() {
		rec := reader.Record()
		body, err := io.ReadAll(rec.Body)
		if err != nil {
			pw.Close()
			out.Close()
			_ = os.Remove(tmpPath)
			return 0, fmt.Errorf("read record body: %w", err)
		}

		headersJSON, err := ccMarshalStableHeaderJSON(rec.Header)
		if err != nil {
			pw.Close()
			out.Close()
			_ = os.Remove(tmpPath)
			return 0, fmt.Errorf("marshal headers: %w", err)
		}

		targetURL := strings.TrimSpace(rec.Header.TargetURI())
		crawlDate := ""
		if ts := rec.Header.Date(); !ts.IsZero() {
			crawlDate = ts.Format(time.RFC3339)
		}
		batch = append(batch, ccWARCExportRow{
			DocID:           ccWARCRecordIDToDocID(rec.Header.RecordID()),
			URL:             targetURL,
			Host:            ccHostFromURL(targetURL),
			CrawlDate:       crawlDate,
			WARCType:        rec.Header.Type(),
			WARCRecordID:    rec.Header.RecordID(),
			WARCRefersTo:    rec.Header.RefersTo(),
			ContentType:     rec.Header.Get("Content-Type"),
			HTMLLength:      ccParseHTMLLength(rec.Header.Get("X-HTML-Length")),
			MarkdownLength:  int64(len(body)),
			WARCHeadersJSON: sanitizeUTF8(headersJSON),
			MarkdownBody:    sanitizeUTF8(string(body)),
			SourceWARCFile:  filepath.Base(inPath),
			SourceFileIndex: int32(fileIndex),
		})
		rowsWritten++
		if len(batch) >= 1000 {
			if err := flush(); err != nil {
				pw.Close()
				out.Close()
				_ = os.Remove(tmpPath)
				return 0, fmt.Errorf("write parquet: %w", err)
			}
		}
	}
	if err := reader.Err(); err != nil {
		pw.Close()
		out.Close()
		_ = os.Remove(tmpPath)
		return 0, fmt.Errorf("read warc: %w", err)
	}
	if err := flush(); err != nil {
		pw.Close()
		out.Close()
		_ = os.Remove(tmpPath)
		return 0, fmt.Errorf("write parquet: %w", err)
	}
	if err := pw.Close(); err != nil {
		out.Close()
		_ = os.Remove(tmpPath)
		return 0, fmt.Errorf("close parquet writer: %w", err)
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return 0, fmt.Errorf("close parquet file: %w", err)
	}
	if err := os.Rename(tmpPath, outPath); err != nil {
		_ = os.Remove(tmpPath)
		return 0, fmt.Errorf("finalize parquet: %w", err)
	}
	return rowsWritten, nil
}

func ccMarshalStableHeaderJSON(h warcpkg.Header) (string, error) {
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	ordered := make(map[string]string, len(keys))
	for _, k := range keys {
		ordered[k] = h[k]
	}
	b, err := json.Marshal(ordered)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func ccWARCRecordIDToDocID(recordID string) string {
	s := strings.TrimPrefix(strings.TrimSpace(recordID), "<urn:uuid:")
	s = strings.TrimSuffix(s, ">")
	if strings.ContainsAny(s, ":<>") {
		return ""
	}
	return s
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
