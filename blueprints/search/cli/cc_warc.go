package cli

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-mizu/mizu/blueprints/search/cli/ui"
	"github.com/go-mizu/mizu/blueprints/search/pkg/cc"
	"github.com/go-mizu/mizu/blueprints/search/pkg/warc"
	"github.com/spf13/cobra"

	_ "github.com/duckdb/duckdb-go/v2"
)

// newCCWarcParent returns the `cc warc` parent command with subcommands.
func newCCWarcParent() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "warc",
		Short: "Work with Common Crawl WARC files",
		Long: `Download, extract, and import Common Crawl WARC files.

WARC files are the raw crawl archives (~1GB each). Use these commands to work
with full WARC files rather than byte-range requests (cc fetch).

Storage layout:
  $HOME/data/common-crawl/{crawl}/warc/         downloaded .warc.gz files
  $HOME/data/common-crawl/{crawl}/warc-import/  imported DuckDB (8 shards)

Subcommands:
  get       Fetch and display a single WARC record by byte range
  list      List available WARC files for a crawl
  download  Download full .warc.gz files
  extract   Stream records from local files (output NDJSON/TSV)
  import    Import records from local files to DuckDB
  query     Query the imported WARC DuckDB
`,
		Example: `  search cc warc list
  search cc warc download --file 0
  search cc warc extract --file 0 --mime text/html --status 200
  search cc warc import --file 0 --mime text/html --status 200
  search cc warc query --domain example.com --limit 20
  search cc warc get --file crawl-data/.../CC-MAIN.warc.gz --offset 12345 --length 6789`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newCCWarcGet())
	cmd.AddCommand(newCCWarcList())
	cmd.AddCommand(newCCWarcDownload())
	cmd.AddCommand(newCCWarcExtract())
	cmd.AddCommand(newCCWarcImport())
	cmd.AddCommand(newCCWarcQuery())
	return cmd
}

// ── cc warc get ── (was: cc warc)

func newCCWarcGet() *cobra.Command {
	var file string
	var offset, length int64

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Fetch and display a single WARC record by byte range",
		Example: `  search cc warc get --file crawl-data/CC-MAIN-2026-08/.../CC-MAIN.warc.gz --offset 12345 --length 6789`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCCWarc(cmd.Context(), file, offset, length)
		},
	}
	cmd.Flags().StringVar(&file, "file", "", "WARC file path (relative to CC base URL)")
	cmd.Flags().Int64Var(&offset, "offset", 0, "Byte offset of the record")
	cmd.Flags().Int64Var(&length, "length", 0, "Byte length of the record")
	cmd.MarkFlagRequired("file")
	cmd.MarkFlagRequired("offset")
	cmd.MarkFlagRequired("length")
	return cmd
}

func runCCWarc(ctx context.Context, file string, offset, length int64) error {
	client := cc.NewClient("", 4)

	ptr := cc.WARCPointer{
		WARCFilename: file,
		RecordOffset: offset,
		RecordLength: length,
	}

	fmt.Println(infoStyle.Render(fmt.Sprintf("Fetching WARC record from %s [%d-%d]...", file, offset, offset+length-1)))

	data, err := client.FetchWARCRecord(ctx, 0, ptr)
	if err != nil {
		return fmt.Errorf("fetching record: %w", err)
	}

	resp, err := cc.ParseWARCRecord(data)
	if err != nil {
		return fmt.Errorf("parsing record: %w", err)
	}

	fmt.Println()
	fmt.Println(successStyle.Render("WARC Record:"))
	fmt.Printf("  Type:        %s\n", resp.WARCType)
	fmt.Printf("  Target URI:  %s\n", resp.TargetURI)
	fmt.Printf("  Date:        %s\n", resp.Date.Format(time.RFC3339))
	fmt.Printf("  Record ID:   %s\n", resp.RecordID)
	fmt.Printf("  HTTP Status: %d\n", resp.HTTPStatus)

	fmt.Println()
	fmt.Println(infoStyle.Render("HTTP Headers:"))
	for k, v := range resp.HTTPHeaders {
		fmt.Printf("  %s: %s\n", k, v)
	}

	fmt.Println()
	fmt.Printf("  %s\n", infoStyle.Render(fmt.Sprintf("Body (%d bytes):", len(resp.Body))))
	body := string(resp.Body)
	if len(body) > 2000 {
		body = body[:2000] + "\n... (truncated)"
	}
	fmt.Println(body)

	return nil
}

// ── cc warc list ──

func newCCWarcList() *cobra.Command {
	var (
		crawlID string
		limit   int
		asJSON  bool
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available WARC files for a crawl",
		Example: `  search cc warc list
  search cc warc list --crawl CC-MAIN-2026-08 --limit 20`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCCWarcList(cmd.Context(), crawlID, limit, asJSON)
		},
	}
	cmd.Flags().StringVar(&crawlID, "crawl", "", "Crawl ID (default: latest)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Max entries to show (0 = all)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Output as JSON")
	return cmd
}

func runCCWarcList(ctx context.Context, crawlID string, limit int, asJSON bool) error {
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
		return fmt.Errorf("downloading manifest: %w", err)
	}

	if limit > 0 && len(paths) > limit {
		paths = paths[:limit]
	}

	cfg := cc.DefaultConfig()
	cfg.CrawlID = crawlID
	warcDir := cfg.WARCDir()

	if asJSON {
		type entry struct {
			Index int    `json:"index"`
			Path  string `json:"path"`
			Local bool   `json:"local"`
		}
		out := make([]entry, len(paths))
		for i, p := range paths {
			local := fileExists(filepath.Join(warcDir, filepath.Base(p)))
			out[i] = entry{Index: i, Path: p, Local: local}
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	// Build table rows
	rows := make([][]string, len(paths))
	for i, p := range paths {
		localMark := ""
		if fileExists(filepath.Join(warcDir, filepath.Base(p))) {
			localMark = ccStatusChip("ok", "local")
		}
		rows[i] = []string{
			strconv.Itoa(i),
			filepath.Base(p),
			localMark,
		}
	}

	card := ccRenderKVCard("WARC Files — "+crawlID, [][2]string{
		{"Crawl", crawlID},
		{"Total files", ccFmtInt64(int64(len(paths)))},
		{"Local dir", warcDir},
	})
	fmt.Println(card)
	fmt.Println()
	fmt.Println(ccRenderTable(
		[]string{"#", "Filename", "Local"},
		rows,
		ccTableOptions{},
	))
	return nil
}

// ── cc warc download ──

func newCCWarcDownload() *cobra.Command {
	var (
		crawlID string
		fileIdx string
		workers int
		dir     string
	)
	cmd := &cobra.Command{
		Use:   "download",
		Short: "Download full .warc.gz files from Common Crawl",
		Long: `Download .warc.gz files to local disk.

--file accepts a single index (0), a range (0-9), or "all".
Files are downloaded to $HOME/data/common-crawl/{crawl}/warc/ by default.
Existing files with matching size are skipped.`,
		Example: `  search cc warc download --file 0
  search cc warc download --file 0-4 --workers 4
  search cc warc download --crawl CC-MAIN-2026-08 --file 0`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCCWarcDownload(cmd.Context(), crawlID, fileIdx, workers, dir)
		},
	}
	cmd.Flags().StringVar(&crawlID, "crawl", "", "Crawl ID (default: latest)")
	cmd.Flags().StringVar(&fileIdx, "file", "0", "File index, range (0-9), or all")
	cmd.Flags().IntVar(&workers, "workers", 2, "Concurrent download workers")
	cmd.Flags().StringVar(&dir, "dir", "", "Output directory (default: data dir/warc/)")
	return cmd
}

func runCCWarcDownload(ctx context.Context, crawlID, fileIdx string, workers int, outDir string) error {
	resolvedID, note, err := ccResolveCrawlID(ctx, crawlID)
	if err != nil {
		return fmt.Errorf("resolving crawl: %w", err)
	}
	crawlID = resolvedID
	if note != "" {
		ccPrintDefaultCrawlResolution(crawlID, note)
	}

	client := cc.NewClient("", workers*2)
	paths, err := client.DownloadManifest(ctx, crawlID, "warc.paths.gz")
	if err != nil {
		return fmt.Errorf("downloading manifest: %w", err)
	}

	// Parse file selector
	selected, err := ccParseFileSelector(fileIdx, len(paths))
	if err != nil {
		return fmt.Errorf("--file: %w", err)
	}

	cfg := cc.DefaultConfig()
	cfg.CrawlID = crawlID
	if outDir == "" {
		outDir = cfg.WARCDir()
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("creating output dir: %w", err)
	}

	labels := make([]string, len(selected))
	for i, idx := range selected {
		labels[i] = filepath.Base(paths[idx])
	}

	model := ui.NewProgressModel(
		fmt.Sprintf("Downloading WARC files — %s", crawlID),
		labels,
	)

	// Track overall progress
	var totalBytes int64
	var doneBytes atomic.Int64

	// Pre-fetch sizes
	sizes := make([]int64, len(selected))
	fmt.Printf("Fetching file sizes for %d files...\n", len(selected))
	for i, idx := range selected {
		sz, _ := client.HeadFileSize(ctx, paths[idx])
		sizes[i] = sz
		totalBytes += sz
	}
	model.Overall.Total = totalBytes

	p := tea.NewProgram(model, tea.WithInput(os.Stdin))
	go func() {
		for i, idx := range selected {
			i := i
			remotePath := paths[idx]
			localPath := filepath.Join(outDir, filepath.Base(remotePath))

			p.Send(ui.ItemUpdateMsg{Index: i, Total: sizes[i], Status: ui.StatusActive})

			var itemDone int64
			err := client.DownloadFile(ctx, remotePath, localPath, func(recv, total int64) {
				delta := recv - itemDone
				itemDone = recv
				doneBytes.Add(delta)
				p.Send(ui.ItemUpdateMsg{Index: i, Done: recv, Total: total, Status: ui.StatusActive})
				p.Send(ui.OverallUpdateMsg{Done: doneBytes.Load(), Total: totalBytes})
			})
			if err != nil {
				p.Send(ui.ItemUpdateMsg{Index: i, Status: ui.StatusError, Err: err})
			} else {
				p.Send(ui.ItemUpdateMsg{Index: i, Done: sizes[i], Total: sizes[i], Status: ui.StatusDone})
			}
		}
		p.Send(ui.DoneMsg{Msg: fmt.Sprintf("Downloaded %d file(s) to %s", len(selected), outDir)})
	}()

	_, err = p.Run()
	return err
}

// ── cc warc extract ──

func newCCWarcExtract() *cobra.Command {
	var (
		crawlID    string
		fileIdx    string
		mimeFilter string
		statusCode int
		domain     string
		outFormat  string
		limit      int64
		maxBody    int64
	)
	cmd := &cobra.Command{
		Use:   "extract",
		Short: "Stream WARC records to NDJSON or TSV (stdout)",
		Example: `  search cc warc extract --file 0 --mime text/html --status 200
  search cc warc extract --file 0 --out tsv --limit 10000`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCCWarcExtract(cmd.Context(), crawlID, fileIdx, mimeFilter, statusCode, domain, outFormat, limit, maxBody)
		},
	}
	cmd.Flags().StringVar(&crawlID, "crawl", "", "Crawl ID (default: latest)")
	cmd.Flags().StringVar(&fileIdx, "file", "0", "File index or range")
	cmd.Flags().StringVar(&mimeFilter, "mime", "", "MIME type filter (e.g. text/html)")
	cmd.Flags().IntVar(&statusCode, "status", 0, "HTTP status filter (e.g. 200)")
	cmd.Flags().StringVar(&domain, "domain", "", "Domain filter")
	cmd.Flags().StringVar(&outFormat, "out", "ndjson", "Output format: ndjson or tsv")
	cmd.Flags().Int64Var(&limit, "limit", 0, "Max records to output (0 = unlimited)")
	cmd.Flags().Int64Var(&maxBody, "max-body", 512*1024, "Max body bytes to include (0 = no limit)")
	return cmd
}

func runCCWarcExtract(ctx context.Context, crawlID, fileIdx, mimeFilter string, statusCode int, domain, outFormat string, limit, maxBody int64) error {
	resolvedID, _, err := ccResolveCrawlID(ctx, crawlID)
	if err != nil {
		return fmt.Errorf("resolving crawl: %w", err)
	}
	crawlID = resolvedID
	cfg := cc.DefaultConfig()
	cfg.CrawlID = crawlID

	client := cc.NewClient("", 4)
	paths, err := client.DownloadManifest(ctx, crawlID, "warc.paths.gz")
	if err != nil {
		return fmt.Errorf("manifest: %w", err)
	}

	selected, err := ccParseFileSelector(fileIdx, len(paths))
	if err != nil {
		return fmt.Errorf("--file: %w", err)
	}

	opts := warc.ImportOptions{
		RecordTypes: []string{"response"},
		MaxBodySize: maxBody,
	}
	if mimeFilter != "" {
		opts.MIMETypes = strings.Split(mimeFilter, ",")
	}
	if statusCode > 0 {
		opts.StatusCodes = []int{statusCode}
	}

	// Writer functions for NDJSON vs TSV
	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()

	var written int64
	writeRecord := func(wr warc.WARCRecord) bool {
		if domain != "" && !strings.Contains(wr.Domain, domain) {
			return true
		}
		if limit > 0 && written >= limit {
			return false
		}

		switch outFormat {
		case "tsv":
			fmt.Fprintf(out, "%s\t%s\t%d\t%s\t%s\t%s\n",
				wr.URL, wr.Domain, wr.HTTPStatus, wr.MIMEType, wr.Title, wr.CrawledAt.Format(time.RFC3339))
		default: // ndjson
			type row struct {
				URL        string `json:"url"`
				Domain     string `json:"domain"`
				HTTPStatus int    `json:"http_status"`
				MIMEType   string `json:"mime_type"`
				Title      string `json:"title,omitempty"`
				CrawledAt  string `json:"crawled_at"`
			}
			r := row{URL: wr.URL, Domain: wr.Domain, HTTPStatus: wr.HTTPStatus,
				MIMEType: wr.MIMEType, Title: wr.Title, CrawledAt: wr.CrawledAt.Format(time.RFC3339)}
			enc, _ := json.Marshal(r)
			out.Write(enc)
			out.WriteByte('\n')
		}
		written++
		return true
	}

	for _, idx := range selected {
		localPath := filepath.Join(cfg.WARCDir(), filepath.Base(paths[idx]))
		if err := streamWARCFile(ctx, localPath, opts, writeRecord); err != nil {
			return fmt.Errorf("streaming %s: %w", filepath.Base(localPath), err)
		}
		if limit > 0 && written >= limit {
			break
		}
	}
	return nil
}

// streamWARCFile opens a local .warc.gz file and calls fn for each matching record.
// fn returns false to stop early.
func streamWARCFile(ctx context.Context, path string, opts warc.ImportOptions, fn func(warc.WARCRecord) bool) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	r := warc.NewReader(f)
	return streamAndFilter(ctx, r, opts, fn)
}

// streamAndFilter iterates a Reader and calls fn for each accepted record.
func streamAndFilter(ctx context.Context, r *warc.Reader, opts warc.ImportOptions, fn func(warc.WARCRecord) bool) error {
	for r.Next() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		rec := r.Record()
		// type filter
		if len(opts.RecordTypes) > 0 && !sliceContains(opts.RecordTypes, rec.Header.Type()) {
			io.Copy(io.Discard, rec.Body)
			continue
		}
		wr, skip := extractWARCRecord(rec, opts)
		if skip {
			continue
		}
		if !fn(wr) {
			break
		}
	}
	return r.Err()
}

// ── cc warc import ──

func newCCWarcImport() *cobra.Command {
	var (
		crawlID    string
		fileIdx    string
		mimeFilter string
		statusCode int
		workers    int
		maxBody    int64
	)
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import WARC records into sharded DuckDB",
		Example: `  search cc warc import --file 0 --mime text/html --status 200
  search cc warc import --file 0-4 --workers 4`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCCWarcImport(cmd.Context(), crawlID, fileIdx, mimeFilter, statusCode, workers, maxBody)
		},
	}
	cmd.Flags().StringVar(&crawlID, "crawl", "", "Crawl ID (default: latest)")
	cmd.Flags().StringVar(&fileIdx, "file", "0", "File index or range")
	cmd.Flags().StringVar(&mimeFilter, "mime", "text/html", "MIME type filter")
	cmd.Flags().IntVar(&statusCode, "status", 200, "HTTP status filter")
	cmd.Flags().IntVar(&workers, "workers", 1, "Parallel file workers (each streams sequentially)")
	cmd.Flags().Int64Var(&maxBody, "max-body", 512*1024, "Max body bytes to store")
	return cmd
}

func runCCWarcImport(ctx context.Context, crawlID, fileIdx, mimeFilter string, statusCode, workers int, maxBody int64) error {
	resolvedID, note, err := ccResolveCrawlID(ctx, crawlID)
	if err != nil {
		return fmt.Errorf("resolving crawl: %w", err)
	}
	crawlID = resolvedID
	if note != "" {
		ccPrintDefaultCrawlResolution(crawlID, note)
	}

	cfg := cc.DefaultConfig()
	cfg.CrawlID = crawlID

	client := cc.NewClient("", 4)
	paths, err := client.DownloadManifest(ctx, crawlID, "warc.paths.gz")
	if err != nil {
		return fmt.Errorf("manifest: %w", err)
	}
	selected, err := ccParseFileSelector(fileIdx, len(paths))
	if err != nil {
		return fmt.Errorf("--file: %w", err)
	}

	db, err := warc.OpenRecordDB(cfg.WARCImportDir(), 1000)
	if err != nil {
		return fmt.Errorf("opening record db: %w", err)
	}
	defer db.Close()

	opts := warc.ImportOptions{
		RecordTypes: []string{"response"},
		MaxBodySize: maxBody,
	}
	if mimeFilter != "" {
		opts.MIMETypes = strings.Split(mimeFilter, ",")
	}
	if statusCode > 0 {
		opts.StatusCodes = []int{statusCode}
	}

	labels := make([]string, len(selected))
	for i, idx := range selected {
		labels[i] = filepath.Base(paths[idx])
	}

	model := ui.NewProgressModel(
		fmt.Sprintf("Importing WARC — %s", crawlID),
		labels,
	)
	p := tea.NewProgram(model, tea.WithInput(os.Stdin))

	go func() {
		var totalImported int64
		for i, idx := range selected {
			localPath := filepath.Join(cfg.WARCDir(), filepath.Base(paths[idx]))
			p.Send(ui.ItemUpdateMsg{Index: i, Status: ui.StatusActive})

			f, err := os.Open(localPath)
			if err != nil {
				p.Send(ui.ItemUpdateMsg{Index: i, Status: ui.StatusError, Err: err})
				continue
			}

			fileOpts := opts
			fileOpts.WARCFile = filepath.Base(localPath)
			reader := warc.NewReader(f)
			im := warc.NewImporter(db, fileOpts)

			im.Import(ctx, reader, func(s warc.ImportStats) {
				p.Send(ui.ItemUpdateMsg{Index: i, Done: s.Imported, Status: ui.StatusActive})
				p.Send(ui.OverallUpdateMsg{Done: totalImported + s.Imported})
			})
			f.Close()

			totalImported += db.FlushedCount()
			p.Send(ui.ItemUpdateMsg{Index: i, Status: ui.StatusDone})
		}
		p.Send(ui.DoneMsg{Msg: fmt.Sprintf("Imported to %s/", cfg.WARCImportDir())})
	}()

	_, err = p.Run()
	return err
}

// ── cc warc query ──

func newCCWarcQuery() *cobra.Command {
	var (
		crawlID    string
		domain     string
		mimeType   string
		statusCode int
		limit      int
		outFormat  string
	)
	cmd := &cobra.Command{
		Use:   "query",
		Short: "Query imported WARC DuckDB",
		Example: `  search cc warc query --domain example.com --limit 20
  search cc warc query --mime text/html --status 200 --limit 100`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCCWarcQuery(cmd.Context(), crawlID, domain, mimeType, statusCode, limit, outFormat)
		},
	}
	cmd.Flags().StringVar(&crawlID, "crawl", "", "Crawl ID (default: latest)")
	cmd.Flags().StringVar(&domain, "domain", "", "Filter by domain")
	cmd.Flags().StringVar(&mimeType, "mime", "", "Filter by MIME type")
	cmd.Flags().IntVar(&statusCode, "status", 0, "Filter by HTTP status")
	cmd.Flags().IntVar(&limit, "limit", 20, "Max results")
	cmd.Flags().StringVar(&outFormat, "out", "table", "Output format: table, json, or tsv")
	return cmd
}

func runCCWarcQuery(ctx context.Context, crawlID, domain, mimeType string, statusCode, limit int, outFormat string) error {
	resolvedID, _, err := ccResolveCrawlID(ctx, crawlID)
	if err != nil {
		return fmt.Errorf("resolving crawl: %w", err)
	}
	crawlID = resolvedID
	cfg := cc.DefaultConfig()
	cfg.CrawlID = crawlID

	importDir := cfg.WARCImportDir()
	if _, err := os.Stat(importDir); err != nil {
		return fmt.Errorf("no import dir at %s — run 'cc warc import' first", importDir)
	}

	var conditions []string
	var args []any
	if domain != "" {
		conditions = append(conditions, "domain LIKE ?")
		args = append(args, "%"+domain+"%")
	}
	if mimeType != "" {
		conditions = append(conditions, "mime_type LIKE ?")
		args = append(args, "%"+mimeType+"%")
	}
	if statusCode > 0 {
		conditions = append(conditions, "http_status = ?")
		args = append(args, statusCode)
	}

	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}

	type resultRow struct {
		URL        string
		Domain     string
		HTTPStatus int
		MIMEType   string
		Title      string
		CrawledAt  string
	}
	var results []resultRow

	for i := range warc.DBShardCount {
		dbPath := filepath.Join(importDir, fmt.Sprintf("warc_%03d.duckdb", i))
		if _, err := os.Stat(dbPath); err != nil {
			continue
		}
		sdb, err := sql.Open("duckdb", dbPath+"?access_mode=read_only")
		if err != nil {
			continue
		}
		q := "SELECT url, domain, http_status, mime_type, title, CAST(crawled_at AS VARCHAR) FROM records" + where + " LIMIT ?"
		rows, err := sdb.QueryContext(ctx, q, append(args, limit)...)
		if err != nil {
			sdb.Close()
			continue
		}
		for rows.Next() && len(results) < limit {
			var r resultRow
			rows.Scan(&r.URL, &r.Domain, &r.HTTPStatus, &r.MIMEType, &r.Title, &r.CrawledAt)
			results = append(results, r)
		}
		rows.Close()
		sdb.Close()
		if len(results) >= limit {
			break
		}
	}

	if outFormat == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}
	if outFormat == "tsv" {
		for _, r := range results {
			fmt.Printf("%s\t%s\t%d\t%s\t%s\n", r.URL, r.Domain, r.HTTPStatus, r.MIMEType, r.Title)
		}
		return nil
	}

	// Table output
	tableRows := make([][]string, len(results))
	for i, r := range results {
		tableRows[i] = []string{r.URL, r.Domain, strconv.Itoa(r.HTTPStatus), r.MIMEType, r.Title}
	}
	fmt.Println(ccRenderKVCard("WARC Import Query — "+crawlID, [][2]string{
		{"Results", strconv.Itoa(len(results))},
		{"Store", importDir + "/"},
	}))
	fmt.Println()
	fmt.Println(ccRenderTable(
		[]string{"URL", "Domain", "Status", "MIME", "Title"},
		tableRows,
		ccTableOptions{},
	))
	return nil
}

// ── helpers ──

// ccParseFileSelector parses "--file" value: "0", "0-9", or "all".
func ccParseFileSelector(s string, total int) ([]int, error) {
	s = strings.TrimSpace(s)
	if s == "all" {
		idx := make([]int, total)
		for i := range idx {
			idx[i] = i
		}
		return idx, nil
	}

	if strings.Contains(s, "-") {
		parts := strings.SplitN(s, "-", 2)
		lo, err1 := strconv.Atoi(parts[0])
		hi, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil {
			return nil, fmt.Errorf("invalid range %q", s)
		}
		if lo < 0 || hi >= total || lo > hi {
			return nil, fmt.Errorf("range %d-%d out of bounds (total: %d)", lo, hi, total)
		}
		idx := make([]int, hi-lo+1)
		for i := range idx {
			idx[i] = lo + i
		}
		return idx, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return nil, fmt.Errorf("invalid file index %q", s)
	}
	if n < 0 || n >= total {
		return nil, fmt.Errorf("file index %d out of bounds (total: %d)", n, total)
	}
	return []int{n}, nil
}

func sliceContains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// extractWARCRecord converts a raw warc.Record into a WARCRecord using
// ImportOptions filtering. Returns (record, accepted).
func extractWARCRecord(rec *warc.Record, opts warc.ImportOptions) (warc.WARCRecord, bool) {
	return warc.ProcessRecord(rec, opts)
}
