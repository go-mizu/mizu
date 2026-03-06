package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/cc"
	warcmd "github.com/go-mizu/mizu/blueprints/search/pkg/warc_md"
	"github.com/spf13/cobra"
)

// newCCWarcPack returns the `cc warc pack` command.
func newCCWarcPack() *cobra.Command {
	var (
		crawlID    string
		fileIdx    string
		from       int
		to         int
		workers    int
		force      bool
		fast       bool
		statusCode int
		mimeFilter string
		maxBody    int64
	)

	cmd := &cobra.Command{
		Use:   "pack",
		Short: "Convert WARC HTML to seekable Markdown WARC (single-pass pipeline)",
		Long: `Single-pass pipeline: read .warc.gz → convert HTML → write .md.warc.gz

Each output record is a WARC conversion record with Content-Type: text/markdown.
Each record is wrapped in its own gzip member, producing a seekable concatenated-gzip
file identical to Common Crawl's format.

Output: warc_md/{warcIdx}.md.warc.gz (one file per input .warc.gz)

Pipeline architecture:
  reader (sequential) → N converter workers (parallel) → writer (sequential)

Converters:
  default:  trafilatura (quality, F1=0.91)   ~200–600 docs/s
  --fast:   go-readability (3–8× faster)     ~800–2,000 docs/s`,
		Example: `  search cc warc pack --file 0
  search cc warc pack --file 0 --fast
  search cc warc pack --file 0-4 --workers 8
  search cc warc pack --from 0 --to 9 --workers 16`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCCWarcPack(cmd.Context(),
				crawlID, fileIdx, from, to, workers, force, fast,
				statusCode, mimeFilter, maxBody)
		},
	}

	cmd.Flags().StringVar(&crawlID, "crawl", "", "Crawl ID (default: latest)")
	cmd.Flags().StringVar(&fileIdx, "file", "0", "File index, range (0-9), or all")
	cmd.Flags().IntVar(&from, "from", -1, "First file index (inclusive) for parallel range")
	cmd.Flags().IntVar(&to, "to", -1, "Last file index (inclusive) for parallel range")
	cmd.Flags().IntVar(&workers, "workers", 0, "Converter goroutines (0 = NumCPU)")
	cmd.Flags().BoolVar(&force, "force", false, "Re-process existing files")
	cmd.Flags().BoolVar(&fast, "fast", false, "Use go-readability instead of trafilatura")
	cmd.Flags().IntVar(&statusCode, "status", 200, "HTTP status filter (0 = all)")
	cmd.Flags().StringVar(&mimeFilter, "mime", "text/html", "MIME type filter")
	cmd.Flags().Int64Var(&maxBody, "max-body", 512*1024, "Max HTML body bytes per record")

	return cmd
}

func runCCWarcPack(ctx context.Context,
	crawlID, fileIdx string, from, to, workers int, force, fast bool,
	statusCode int, mimeFilter string, maxBody int64) error {

	if from >= 0 && to >= 0 {
		fileIdx = fmt.Sprintf("%d-%d", from, to)
	}

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
	warcDir := cfg.WARCDir()

	// Auto-download missing files
	var inputFiles []string
	for _, idx := range selected {
		localPath := filepath.Join(warcDir, filepath.Base(paths[idx]))
		if !fileExists(localPath) {
			if err := downloadWithProgress(ctx, client, paths[idx], localPath); err != nil {
				return fmt.Errorf("downloading %s: %w", filepath.Base(localPath), err)
			}
		}
		inputFiles = append(inputFiles, localPath)
	}

	// Print header
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("  WARC → Markdown WARC   Pack Pipeline"))
	fmt.Println()

	extractor := "trafilatura (quality)"
	if fast {
		extractor = "go-readability (fast)"
	}

	fmt.Printf("  Crawl     %s\n", labelStyle.Render(crawlID))
	fmt.Printf("  Files     %d\n", len(inputFiles))
	fmt.Printf("  Engine    %s\n", infoStyle.Render(extractor))
	fmt.Printf("  Output    %s\n", labelStyle.Render(cfg.WARCMdDir()))
	fmt.Println()

	pipeStart := time.Now()
	var totalIn, totalOut, totalErr int64
	var totalRead, totalWrite int64

	for i, localPath := range inputFiles {
		warcIdx := warcIndexFromPath(paths[selected[i]], selected[i])
		fname := filepath.Base(localPath)
		outPath := filepath.Join(cfg.WARCMdDir(), warcIdx+".md.warc.gz")

		fmt.Printf("%s\n", subtitleStyle.Render(
			fmt.Sprintf("  [%d/%d]  %s  →  %s.md.warc.gz", i+1, len(inputFiles), fname, warcIdx)))

		packCfg := warcmd.PackConfig{
			InputFiles:  []string{localPath},
			OutputPath:  outPath,
			Workers:     workers,
			Force:       force,
			Fast:        fast,
			StatusCode:  statusCode,
			MIMEFilter:  mimeFilter,
			MaxBodySize: maxBody,
		}

		result, err := warcmd.RunPack(ctx, packCfg,
			func(done, total, errors, readBytes, writeBytes int64, elapsed time.Duration, peakMemMB float64) {
				rate := float64(0)
				if elapsed.Seconds() > 0 {
					rate = float64(done) / elapsed.Seconds()
				}
				fmt.Printf("\r\033[K  %s  in=%s  out=%s  err=%s  %.0f/s  %s",
					infoStyle.Render("packing"),
					ccFmtInt64(total),
					ccFmtInt64(done),
					ccFmtInt64(errors),
					rate,
					elapsed.Round(time.Millisecond))
			},
		)
		fmt.Printf("\r\033[K")
		if err != nil {
			return fmt.Errorf("pack %s: %w", fname, err)
		}

		if result.Skipped > 0 {
			fmt.Printf("  %s (output exists, use --force to re-process)\n", warningStyle.Render("skipped"))
		} else {
			var outSize int64
			if fi, err := os.Stat(outPath); err == nil {
				outSize = fi.Size()
			}
			rate := float64(0)
			if result.Duration.Seconds() > 0 {
				rate = float64(result.OutputRecords) / result.Duration.Seconds()
			}
			fmt.Printf("  %s  in=%s  out=%s  err=%s  %s  %.0f/s  peak=%s\n",
				successStyle.Render("done"),
				ccFmtInt64(result.InputRecords),
				ccFmtInt64(result.OutputRecords),
				ccFmtInt64(result.Errors),
				formatBytes(outSize),
				rate,
				formatBytes(int64(result.PeakMemMB*1024*1024)),
			)
		}

		totalIn += result.InputRecords
		totalOut += result.OutputRecords
		totalErr += result.Errors
		totalRead += result.ReadBytes
		totalWrite += result.WriteBytes
	}

	totalDuration := time.Since(pipeStart)

	fmt.Println()
	fmt.Println(successStyle.Render("  Pack complete!"))
	fmt.Println()
	overallRate := float64(0)
	if totalDuration.Seconds() > 0 {
		overallRate = float64(totalOut) / totalDuration.Seconds()
	}
	fmt.Printf("  Input      %s HTML records  (%s read)\n",
		infoStyle.Render(ccFmtInt64(totalIn)), formatBytes(totalRead))
	fmt.Printf("  Output     %s markdown records  (%s written)\n",
		infoStyle.Render(ccFmtInt64(totalOut)), formatBytes(totalWrite))
	if totalErr > 0 {
		fmt.Printf("  Errors     %s\n", warningStyle.Render(ccFmtInt64(totalErr)))
	}
	fmt.Printf("  Rate       %.0f docs/s\n", overallRate)
	fmt.Printf("  Time       %s\n", totalDuration.Round(time.Millisecond))
	fmt.Println()

	return nil
}
