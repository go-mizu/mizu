package warc_md

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
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
	workers := cfg.Workers
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

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
		Workers:   workers,
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
		Workers:   workers,
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
//	.warc.gz → producer → warcCh → N converters → mdCh → N writers → .md.gz
func RunInMemoryPipeline(ctx context.Context, cfg Config, inputFiles []string,
	progressFn ProgressFunc) (*PipelineResult, error) {

	workers := cfg.Workers
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
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

	// Open DuckDB index
	idx, err := mdpkg.OpenIndex(cfg.IndexPath(), 1000)
	if err != nil {
		return nil, fmt.Errorf("open index: %w", err)
	}
	defer idx.Close()

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

	// ── Stage 2: Converters ────────────────────────────────────────────────
	var convWg sync.WaitGroup
	convWg.Add(workers)
	for range workers {
		g.Go(func() error {
			defer convWg.Done()
			for {
				select {
				case item, ok := <-warcCh:
					if !ok {
						return nil
					}
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
					case <-gctx.Done():
						return nil
					}

				case <-gctx.Done():
					return nil
				}
			}
		})
	}
	// Close mdCh once all converters are done
	go func() {
		convWg.Wait()
		close(mdCh)
	}()

	// ── Stage 3: Writers ───────────────────────────────────────────────────
	for range workers {
		g.Go(func() error {
			for {
				select {
				case item, ok := <-mdCh:
					if !ok {
						return nil
					}
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

				case <-gctx.Done():
					return nil
				}
			}
		})
	}

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
