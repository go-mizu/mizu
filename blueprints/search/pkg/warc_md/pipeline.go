package warc_md

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	mdpkg "github.com/go-mizu/mizu/blueprints/search/pkg/markdown"
	warcpkg "github.com/go-mizu/mizu/blueprints/search/pkg/warc"
	"golang.org/x/sync/errgroup"
)

// RunFilePipeline executes all three phases sequentially with intermediate
// files on disk. Temp directories (warc_single/ and markdown/) are removed
// after all phases succeed unless cfg.KeepTemp is set.
//
// p1Fn, p2Fn, p3Fn are per-phase progress callbacks (may be nil).
func RunFilePipeline(ctx context.Context, cfg Config, inputFiles []string,
	p1Fn, p2Fn, p3Fn ProgressFunc) (*PipelineResult, error) {

	start := time.Now()

	// ── Phase 1: Extract ────────────────────────────────────────────────────
	s1, err := RunExtract(ctx, ExtractConfig{
		InputFiles:  inputFiles,
		OutputDir:   cfg.WARCSingleDir(),
		Workers:     len(inputFiles),
		Force:       cfg.Force,
		StatusCode:  cfg.StatusCode,
		MIMEFilter:  cfg.MIMEFilter,
		MaxBodySize: cfg.MaxBodySize,
	}, p1Fn)
	if err != nil {
		return nil, fmt.Errorf("phase 1 extract: %w", err)
	}

	// ── Phase 2: Convert ────────────────────────────────────────────────────
	s2, err := RunConvert(ctx, ConvertConfig{
		InputDir:  cfg.WARCSingleDir(),
		OutputDir: cfg.MarkdownDir(),
		IndexPath: cfg.IndexPath(),
		Workers:   cfg.ConvertWorkers(),
		Force:     cfg.Force,
		BatchSize: 1000,
		Fast:      cfg.Fast,
	}, p2Fn)
	if err != nil {
		return nil, fmt.Errorf("phase 2 convert: %w", err)
	}

	// ── Phase 3: Compress ───────────────────────────────────────────────────
	s3, err := RunCompress(ctx, CompressConfig{
		InputDir:  cfg.MarkdownDir(),
		OutputDir: cfg.MarkdownGzDir(),
		Workers:   cfg.CompressWorkers(),
		Force:     cfg.Force,
	}, p3Fn)
	if err != nil {
		return nil, fmt.Errorf("phase 3 compress: %w", err)
	}

	result := &PipelineResult{
		Extract:  s1,
		Convert:  s2,
		Compress: s3,
		Duration: time.Since(start),
	}

	// ── Cleanup temp dirs ────────────────────────────────────────────────────
	if !cfg.KeepTemp {
		os.RemoveAll(cfg.WARCSingleDir())
		os.RemoveAll(cfg.MarkdownDir())
	}

	return result, nil
}

// RunInMemoryPipeline runs a streaming 3-stage pipeline without any temp files.
// Stages are connected by buffered channels:
//
//	.warc.gz → producer → warcCh → N converters → mdCh → M writers → .md.gz
//
// Worker counts are adaptive via cfg.ConvertWorkers() and cfg.CompressWorkers().
// Each stage runs its own nested errgroup; the main errgroup manages lifecycle.
func RunInMemoryPipeline(ctx context.Context, cfg Config, inputFiles []string,
	progressFn ProgressFunc) (*PipelineResult, error) {

	convertWorkers := cfg.ConvertWorkers()
	compressWorkers := cfg.CompressWorkers()
	if cfg.StatusCode == 0 {
		cfg.StatusCode = 200
	}
	if cfg.MIMEFilter == "" {
		cfg.MIMEFilter = "text/html"
	}
	outDir := cfg.MarkdownGzDir()
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}

	// Open DuckDB index — or use a caller-provided shared one.
	// cfg.NoIndex disables index writes entirely (perf benchmarking / parallel runs).
	var idx *mdpkg.IndexDB
	switch {
	case cfg.NoIndex:
		// idx stays nil; all idx.Add calls below are guarded
	case cfg.SharedIndex != nil:
		idx = cfg.SharedIndex // caller owns; don't defer Close
	default:
		var err error
		idx, err = mdpkg.OpenIndex(cfg.IndexPath(), 1000)
		if err != nil {
			return nil, fmt.Errorf("open index: %w", err)
		}
		defer idx.Close()
	}

	// Channel capacities: each holds ~500 items in flight between stages.
	const chanCap = 500
	warcCh := make(chan WARCItem, chanCap)
	mdCh := make(chan MarkdownItem, chanCap)

	var (
		readBytes  atomic.Int64
		writeBytes atomic.Int64
		extracted  atomic.Int64
		converted  atomic.Int64
		compressed atomic.Int64
		errCount   atomic.Int64
	)

	pipeStart := time.Now()
	stopMem := make(chan struct{})
	getPeakMB := trackPeakMem(stopMem)

	var stopProgress chan struct{}
	if progressFn != nil {
		stopProgress = make(chan struct{})
		go func() {
			tick := time.NewTicker(500 * time.Millisecond)
			defer tick.Stop()
			for {
				select {
				case <-tick.C:
					progressFn(
						compressed.Load(), extracted.Load(), errCount.Load(),
						readBytes.Load(), writeBytes.Load(),
						time.Since(pipeStart), getPeakMB())
				case <-stopProgress:
					return
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	g, gctx := errgroup.WithContext(ctx)

	// ── Stage 1: Producer ──────────────────────────────────────────────────
	// One goroutine streams all input files sequentially into warcCh.
	// On exit (or error), warcCh is closed, unblocking the stage-2 bridge.
	g.Go(func() error {
		defer close(warcCh)
		for _, warcPath := range inputFiles {
			if gctx.Err() != nil {
				return nil
			}
			_ = produceFromWARCGz(gctx, warcPath, cfg, warcCh, &readBytes, &extracted)
		}
		return nil
	})

	// ── Stage 2 bridge: N converter goroutines ─────────────────────────────
	// A single bridge goroutine hosts its own nested errgroup of N converters.
	// Each converter uses "for range warcCh" (exits when warcCh closes) and
	// sends results to mdCh using a select to respect context cancellation.
	// When all converters finish, mdCh is closed, unblocking stage-3.
	g.Go(func() error {
		eg, ectx := errgroup.WithContext(gctx)
		for range convertWorkers {
			eg.Go(func() error {
				for item := range warcCh {
					var result mdpkg.Result
					if cfg.Fast {
						result = mdpkg.ConvertFast(item.HTMLBody, "")
					} else {
						result = mdpkg.Convert(item.HTMLBody, "")
					}
					converted.Add(1)

					ratio := float64(0)
					if result.HTMLSize > 0 {
						ratio = float64(result.MarkdownSize) / float64(result.HTMLSize)
					}
					if idx != nil {
						idx.Add(mdpkg.IndexRecord{
							CID:              item.RecordID,
							HTMLSize:         result.HTMLSize,
							MarkdownSize:     result.MarkdownSize,
							HTMLTokens:       result.HTMLTokens,
							MarkdownTokens:   result.MarkdownTokens,
							CompressionRatio: ratio,
							Title:            truncate(result.Title, 500),
							Language:         result.Language,
							HasContent:       result.HasContent,
							ConvertMs:        result.ConvertMs,
							Error:            result.Error,
						})
					}

					if result.Error != "" || !result.HasContent || result.Markdown == "" {
						errCount.Add(1)
						continue
					}

					select {
					case mdCh <- MarkdownItem{
						RecordID:   item.RecordID,
						Markdown:   result.Markdown,
						Title:      result.Title,
						Language:   result.Language,
						HasContent: result.HasContent,
					}:
					case <-ectx.Done():
						return ectx.Err()
					}
				}
				return nil
			})
		}
		err := eg.Wait()
		close(mdCh) // always close so writers can drain and exit
		return err
	})

	// ── Stage 3 bridge: M writer goroutines ───────────────────────────────
	// A single bridge goroutine hosts its own nested errgroup of M writers.
	// Each writer uses "for range mdCh" (exits when mdCh closes).
	// Context cancellation is checked at each record to allow fast exit.
	g.Go(func() error {
		eg, _ := errgroup.WithContext(gctx)
		for range compressWorkers {
			eg.Go(func() error {
				for item := range mdCh {
					if gctx.Err() != nil {
						return nil
					}
					outPath := MarkdownGzFilePath(outDir, item.RecordID)
					if !cfg.Force {
						if _, err := os.Stat(outPath); err == nil {
							compressed.Add(1)
							continue
						}
					}
					if err := compressToGz(outPath, []byte(item.Markdown)); err != nil {
						errCount.Add(1)
						continue
					}
					if fi, err := os.Stat(outPath); err == nil {
						writeBytes.Add(fi.Size())
					}
					compressed.Add(1)
				}
				return nil
			})
		}
		return eg.Wait()
	})

	gerr := g.Wait()

	if stopProgress != nil {
		close(stopProgress)
	}
	close(stopMem)
	duration := time.Since(pipeStart)
	peak := getPeakMB()

	return &PipelineResult{
		Extract: &PhaseStats{
			Files:     extracted.Load(),
			Errors:    errCount.Load(),
			ReadBytes: readBytes.Load(),
			Duration:  duration,
			PeakMemMB: peak,
		},
		Convert: &PhaseStats{
			Files:    converted.Load(),
			Errors:   errCount.Load(),
			Duration: duration,
			PeakMemMB: peak,
		},
		Compress: &PhaseStats{
			Files:      compressed.Load(),
			Errors:     errCount.Load(),
			WriteBytes: writeBytes.Load(),
			Duration:   duration,
			PeakMemMB:  peak,
		},
		Duration: duration,
	}, gerr
}

// produceFromWARCGz streams one .warc.gz file, extracts filtered HTML records,
// and sends them to warcCh. Returns on context cancellation or after EOF.
func produceFromWARCGz(ctx context.Context, warcPath string, cfg Config,
	warcCh chan<- WARCItem, readBytes *atomic.Int64, extracted *atomic.Int64) error {

	f, err := os.Open(warcPath)
	if err != nil {
		return err
	}
	defer f.Close()

	r := warcpkg.NewReader(f)
	for r.Next() {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		rec := r.Record()
		if rec.Header.Type() != warcpkg.TypeResponse {
			io.Copy(io.Discard, rec.Body)
			continue
		}

		var bodyReader io.Reader = rec.Body
		if cfg.MaxBodySize > 0 {
			bodyReader = io.LimitReader(rec.Body, cfg.MaxBodySize+8192)
		}
		bodyBytes, err := io.ReadAll(bodyReader)
		if err != nil {
			continue
		}
		readBytes.Add(int64(len(bodyBytes)))

		status, mime, htmlBody := parseHTTPResponse(bodyBytes)
		if cfg.StatusCode != 0 && status != cfg.StatusCode {
			continue
		}
		if cfg.MIMEFilter != "" && !strings.Contains(strings.ToLower(mime), strings.ToLower(strings.SplitN(cfg.MIMEFilter, "/", 2)[0])) {
			continue
		}
		if cfg.MaxBodySize > 0 && int64(len(htmlBody)) > cfg.MaxBodySize {
			htmlBody = htmlBody[:cfg.MaxBodySize]
		}

		recordID := rec.Header.RecordID()
		extracted.Add(1)

		select {
		case warcCh <- WARCItem{RecordID: recordID, HTMLBody: htmlBody}:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return r.Err()
}

// DiskUsageBytes sums the size of all regular files under path.
func DiskUsageBytes(path string) int64 {
	var total int64
	_ = filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if info, err := d.Info(); err == nil {
			total += info.Size()
		}
		return nil
	})
	return total
}

// DiskUsageMdGz sums only *.md.gz files under dir, excluding index.duckdb
// and any other metadata files that live in the same output directory.
func DiskUsageMdGz(dir string) int64 {
	var total int64
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".md.gz") {
			return nil
		}
		if info, err := d.Info(); err == nil {
			total += info.Size()
		}
		return nil
	})
	return total
}
