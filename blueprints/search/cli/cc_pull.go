package cli

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	warcpkg "github.com/go-mizu/mizu/blueprints/search/pkg/warc"
	"github.com/parquet-go/parquet-go"
	"github.com/spf13/cobra"
)

func newCCPull() *cobra.Command {
	var (
		crawlID     string
		repoID      string
		fileIdx     string
		deleteLocal bool
	)

	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Download parquet shards from Hugging Face and reconstruct md.warc.gz",
		Long: `Download parquet shards from a Hugging Face dataset repo and reconstruct
the md.warc.gz files locally, then optionally delete the local parquet.

This is the reverse of 'search cc publish'. Each row in the parquet contains
all 8 WARC headers needed to reconstruct the WARC conversion record.

Output: $HOME/data/common-crawl/{crawl}/warc_md/{shard}.md.warc.gz`,
		Example: `  search cc pull --file 0
  search cc pull --file 0 --delete-local
  search cc pull --crawl CC-MAIN-2026-04 --repo open-index/cc-main --file 0`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			token := strings.TrimSpace(os.Getenv("HF_TOKEN"))
			if token == "" {
				return fmt.Errorf("HF_TOKEN not set")
			}

			crawlID, _, err := ccResolveCrawlID(ctx, crawlID)
			if err != nil {
				return fmt.Errorf("resolve crawl: %w", err)
			}
			if repoID == "" {
				repoID = "open-index/" + strings.ToLower(crawlID)
			}

			indices, err := ccParseOpenFileSelector(fileIdx)
			if err != nil {
				return fmt.Errorf("--file: %w", err)
			}
			if len(indices) == 0 {
				return fmt.Errorf("--file is required (specify an index or range like 0 or 37-100)")
			}

			hf := newHFClient(token)
			dataHome, _ := os.UserHomeDir()
			baseDir := filepath.Join(dataHome, "data", "common-crawl", crawlID)
			warcMdDir := filepath.Join(baseDir, "warc_md")
			dataDir := filepath.Join(baseDir, "export", "repo", "data", crawlID)

			for _, idx := range indices {
				shard := fmt.Sprintf("%05d", idx)
				parquetRemote := fmt.Sprintf("data/%s/%s.parquet", crawlID, shard)
				localParquet := filepath.Join(dataDir, shard+".parquet")
				outWARC := filepath.Join(warcMdDir, shard+".md.warc.gz")

				// Download parquet if not already local.
				if _, statErr := os.Stat(localParquet); os.IsNotExist(statErr) {
					fmt.Printf("  [%s] downloading parquet from HF...\n", shard)
					if err := pullDownloadParquet(ctx, hf, repoID, parquetRemote, localParquet); err != nil {
						fmt.Printf("  [%s] download failed: %v\n", shard, err)
						continue
					}
				} else {
					fmt.Printf("  [%s] parquet already local\n", shard)
				}

				// Reconstruct md.warc.gz.
				if err := os.MkdirAll(warcMdDir, 0o755); err != nil {
					return err
				}
				fmt.Printf("  [%s] reconstructing md.warc.gz...\n", shard)
				rows, err := pullReconstructWARC(localParquet, outWARC)
				if err != nil {
					fmt.Printf("  [%s] reconstruct failed: %v\n", shard, err)
					continue
				}
				fi, _ := os.Stat(outWARC)
				fmt.Printf("  [%s] wrote %d records -> %s (%s)\n",
					shard, rows, outWARC, ccFmtBytes(fi.Size()))

				// Optionally delete local parquet.
				if deleteLocal {
					if err := os.Remove(localParquet); err == nil {
						fmt.Printf("  [%s] deleted local parquet\n", shard)
					}
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&crawlID, "crawl", "", "Crawl ID (default: latest)")
	cmd.Flags().StringVar(&repoID, "repo", "", "HF repo ID (default: open-index/<crawl-id-lower>)")
	cmd.Flags().StringVar(&fileIdx, "file", "", "File index or range (e.g. 0, 37-100)")
	cmd.Flags().BoolVar(&deleteLocal, "delete-local", false, "Delete local parquet after WARC reconstruction")
	_ = cmd.MarkFlagRequired("file")
	return cmd
}

// pullDownloadParquet downloads a single parquet shard from HF to a local path.
func pullDownloadParquet(ctx context.Context, hf *hfClient, repoID, pathInRepo, localPath string) error {
	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		return err
	}
	dlURL := fmt.Sprintf("https://huggingface.co/datasets/%s/resolve/main/%s", repoID, pathInRepo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, dlURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+hf.token)
	resp, err := hf.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d for %s", resp.StatusCode, dlURL)
	}
	f, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

// pullReconstructWARC reads a parquet shard and writes a concatenated-gzip md.warc.gz.
// Returns the number of records written.
func pullReconstructWARC(parquetPath, outPath string) (int64, error) {
	f, err := os.Open(parquetPath)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	fi, _ := f.Stat()
	pf, err := parquet.OpenFile(f, fi.Size())
	if err != nil {
		return 0, err
	}
	reader := parquet.NewGenericReader[ccWARCExportRow](pf)
	defer reader.Close()

	out, err := os.Create(outPath)
	if err != nil {
		return 0, err
	}
	defer out.Close()

	var written int64
	batch := make([]ccWARCExportRow, 256)
	for {
		n, readErr := reader.Read(batch)
		for i := 0; i < n; i++ {
			row := batch[i]
			body := []byte(row.Markdown)

			hdr := warcpkg.Header{
				"WARC-Type":       warcpkg.TypeConversion,
				"WARC-Target-URI": row.URL,
				"WARC-Date":       row.CrawlDate,
				"WARC-Record-ID":  row.WARCRecordID,
				"WARC-Refers-To":  row.WARCRefersTo,
				"Content-Type":    "text/markdown",
				"Content-Length":  strconv.Itoa(len(body)),
				"X-HTML-Length":   strconv.FormatInt(row.HTMLLength, 10),
			}
			rec := &warcpkg.Record{
				Header: hdr,
				Body:   strings.NewReader(row.Markdown),
			}

			// Each record in its own gzip member (concatenated gzip).
			gz, gzErr := gzip.NewWriterLevel(out, gzip.BestSpeed)
			if gzErr != nil {
				return written, gzErr
			}
			w := warcpkg.NewWriter(gz)
			if err := w.WriteRecord(rec); err != nil {
				gz.Close()
				return written, err
			}
			if err := w.Close(); err != nil {
				gz.Close()
				return written, err
			}
			if err := gz.Close(); err != nil {
				return written, err
			}
			written++
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return written, readErr
		}
	}
	return written, nil
}
