package markdown

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"
)

// PipelineConfig configures the streaming in-memory pipeline.
// Bodies/*.gz → convert → md-gz/*.md.gz with no intermediate disk files.
type PipelineConfig struct {
	InputDir  string // bodies/ directory containing *.gz files
	OutputDir string // md-gz/ directory for *.md.gz output
	IndexPath string // DuckDB index path
	Workers   int    // goroutines per stage (0 = NumCPU)
	Fast      bool   // use go-readability instead of trafilatura
	Force     bool   // re-process even when output already exists
	BatchSize int    // DuckDB write batch size (0 = 1000)
	Compress  bool   // write .md.gz output (default: write .md)
}

// PipelineStats holds aggregate stats for the streaming pipeline.
type PipelineStats struct {
	Read       int64 // .gz files decompressed and sent into pipeline
	Converted  int64 // successful HTML → Markdown conversions
	Written    int64 // .md.gz files written to disk
	Skipped    int64 // files skipped (output already exists, incremental)
	Errors     int64 // extraction failures + conversion misses
	ReadBytes  int64 // uncompressed HTML bytes read
	WriteBytes int64 // compressed .md.gz bytes written
	PeakMemMB  float64
	Duration   time.Duration
}

// htmlItem is produced by Stage 1 readers and consumed by Stage 2 converters.
type htmlItem struct {
	relBase string // path relative to InputDir with .gz stripped
	html    []byte
}

// mdItem is produced by Stage 2 converters and consumed by Stage 3 writers.
type mdItem struct {
	relBase string
	cid     string
	md      string
}

// RunPipeline streams bodies/*.gz through HTML→Markdown conversion and gzip
// compression into md-gz/*.md.gz without writing any intermediate files.
//
// Architecture (3 fixed worker pools connected by buffered channels):
//
//	feeder → fileCh → N readers → htmlCh → N converters → mdCh → N writers
//
// Each stage uses exactly N goroutines that loop on a channel — no per-file
// goroutine creation, no scheduling overhead from errgroup.SetLimit.
// All three stages run concurrently: while stage 1 decompresses file N,
// stage 2 is converting N-1, and stage 3 is writing N-2.
func RunPipeline(ctx context.Context, cfg PipelineConfig, progressFn PhaseProgressFunc) (*PipelineStats, error) {
	workers := cfg.Workers
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 1000
	}

	// Collect all .gz input files upfront so we know totalFiles for progress.
	var files []string
	_ = filepath.WalkDir(cfg.InputDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".gz") {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if len(files) == 0 {
		return &PipelineStats{}, nil
	}
	totalFiles := int64(len(files))

	if err := os.MkdirAll(cfg.OutputDir, 0o755); err != nil {
		return nil, err
	}

	idx, err := OpenIndex(cfg.IndexPath, cfg.BatchSize)
	if err != nil {
		return nil, err
	}
	defer idx.Close()

	// Channel capacities: large enough to keep all stages busy, small enough
	// to bound peak memory. 256 slots × ~512 KB avg HTML ≈ 128 MB max.
	const chanCap = 256
	fileCh := make(chan string, workers*2)
	htmlCh := make(chan htmlItem, chanCap)
	mdCh := make(chan mdItem, chanCap)

	var (
		readCount  atomic.Int64
		convCount  atomic.Int64
		writeCount atomic.Int64
		skipCount  atomic.Int64
		errCount   atomic.Int64
		readBytes  atomic.Int64
		writeBytes atomic.Int64
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
					done := writeCount.Load() + skipCount.Load() + errCount.Load()
					progressFn(done, totalFiles, errCount.Load(),
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

	// ── Path feeder: sends file paths into fileCh ─────────────────────────────
	g.Go(func() error {
		defer close(fileCh)
		for _, f := range files {
			select {
			case fileCh <- f:
			case <-gctx.Done():
				return nil
			}
		}
		return nil
	})

	// ── Stage 1: N readers — decompress .gz → htmlCh ─────────────────────────
	var gzPool sync.Pool
	var readWg sync.WaitGroup
	readWg.Add(workers)
	for range workers {
		g.Go(func() error {
			defer readWg.Done()
			for {
				select {
				case fpath, ok := <-fileCh:
					if !ok {
						return nil
					}
					relPath, err := filepath.Rel(cfg.InputDir, fpath)
					if err != nil {
						errCount.Add(1)
						continue
					}
					relBase := strings.TrimSuffix(filepath.ToSlash(relPath), ".gz")

					// Incremental: skip if output already exists.
					if !cfg.Force {
						outExt := ".md"
						if cfg.Compress {
							outExt = ".md.gz"
						}
						outPath := filepath.Join(cfg.OutputDir, relBase+outExt)
						if _, err := os.Stat(outPath); err == nil {
							skipCount.Add(1)
							continue
						}
					}

					html, err := readGzFile(fpath, &gzPool)
					if err != nil {
						errCount.Add(1)
						continue
					}
					readBytes.Add(int64(len(html)))
					readCount.Add(1)

					select {
					case htmlCh <- htmlItem{relBase, html}:
					case <-gctx.Done():
						return nil
					}

				case <-gctx.Done():
					return nil
				}
			}
		})
	}
	// Close htmlCh once all readers finish.
	go func() { readWg.Wait(); close(htmlCh) }()

	// ── Stage 2: N converters — HTML → Markdown → mdCh ───────────────────────
	var convWg sync.WaitGroup
	convWg.Add(workers)
	for range workers {
		g.Go(func() error {
			defer convWg.Done()
			for {
				select {
				case item, ok := <-htmlCh:
					if !ok {
						return nil
					}
					var result Result
					if cfg.Fast {
						result = ConvertFast(item.html, "")
					} else {
						result = Convert(item.html, "")
					}

					cid := cidFromRelPath(item.relBase)
					ratio := float64(0)
					if result.HTMLSize > 0 {
						ratio = float64(result.MarkdownSize) / float64(result.HTMLSize)
					}
					idx.Add(IndexRecord{
						CID:              cid,
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
					convCount.Add(1)

					select {
					case mdCh <- mdItem{item.relBase, cid, result.Markdown}:
					case <-gctx.Done():
						return nil
					}

				case <-gctx.Done():
					return nil
				}
			}
		})
	}
	// Close mdCh once all converters finish.
	go func() { convWg.Wait(); close(mdCh) }()

	// ── Stage 3: N writers — write Markdown to disk ─────────────────────────────
	for range workers {
		g.Go(func() error {
			for {
				select {
				case item, ok := <-mdCh:
					if !ok {
						return nil
					}
					data := []byte(item.md)
					if cfg.Compress {
						outPath := filepath.Join(cfg.OutputDir, item.relBase+".md.gz")
						if err := compressToGz(outPath, data); err != nil {
							errCount.Add(1)
							continue
						}
						if fi, err := os.Stat(outPath); err == nil {
							writeBytes.Add(fi.Size())
						}
					} else {
						outPath := filepath.Join(cfg.OutputDir, item.relBase+".md")
						if err := writePlainMd(outPath, data); err != nil {
							errCount.Add(1)
							continue
						}
						writeBytes.Add(int64(len(data)))
					}
					writeCount.Add(1)

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

	return &PipelineStats{
		Read:       readCount.Load(),
		Converted:  convCount.Load(),
		Written:    writeCount.Load(),
		Skipped:    skipCount.Load(),
		Errors:     errCount.Load(),
		ReadBytes:  readBytes.Load(),
		WriteBytes: writeBytes.Load(),
		PeakMemMB:  getPeakMB(),
		Duration:   time.Since(pipeStart),
	}, gerr
}
