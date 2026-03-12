package cc

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
	indexpack "github.com/go-mizu/mizu/blueprints/search/pkg/index/pack"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index/web/pipeline/util"
	warcpkg "github.com/go-mizu/mizu/blueprints/search/pkg/warc"
)

// Compile-time check.
var _ core.Task[IndexState, IndexMetric] = (*IndexTask)(nil)

// indexConcurrency is the number of shards indexed in parallel.
const indexConcurrency = 2

// IndexTask builds FTS indexes from packed or raw markdown data.
type IndexTask struct {
	crawlDir string
	paths    []string
	selected []int
	engine   string
	source   string
}

// IndexState is emitted during indexing with per-shard detail.
type IndexState struct {
	FileIndex   int     `json:"file_index"`
	FileTotal   int     `json:"file_total"`
	WARCIndex   string  `json:"warc_index"`
	Engine      string  `json:"engine"`
	Source      string  `json:"source"`
	DocsIndexed int64   `json:"docs_indexed"`
	DocsTotal   int64   `json:"docs_total"`
	Progress    float64 `json:"progress"`
	DocsPerSec  float64 `json:"docs_per_sec,omitempty"`
}

// IndexMetric is the final result after indexing completes.
type IndexMetric struct {
	Files   int           `json:"files"`
	Docs    int64         `json:"docs"`
	Elapsed time.Duration `json:"elapsed_ns"`
}

// NewIndexTask creates an index task for the given engine and source.
func NewIndexTask(crawlDir string, paths []string, selected []int, engine, source string) *IndexTask {
	if engine == "" {
		engine = "dahlia"
	}
	if source == "" {
		source = "files"
	}
	return &IndexTask{crawlDir: crawlDir, paths: paths, selected: selected, engine: engine, source: source}
}

func (t *IndexTask) Run(ctx context.Context, emit func(*IndexState)) (IndexMetric, error) {
	start := time.Now()
	total := len(t.selected)
	var totalDocs atomic.Int64

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(indexConcurrency)

	for i, idx := range t.selected {
		i, idx := i, idx
		g.Go(func() error {
			warcIdx := util.WARCFileIndex(t.paths[idx], idx)
			outputDir := filepath.Join(t.crawlDir, "fts", t.engine, warcIdx)

			eng, err := openEngine(gctx, t.engine, outputDir)
			if err != nil {
				return err
			}

			emitIndexProgress(emit, i, total, warcIdx, t.engine, t.source, 0, 0, start)

			var docs int64
			if t.source == "files" {
				docs, err = indexFromWARCMd(gctx, eng, t.crawlDir, warcIdx, func(done, docTotal int64, _ time.Duration) {
					totalDocs.Store(done)
					emitIndexProgress(emit, i, total, warcIdx, t.engine, t.source, done, docTotal, start)
				})
			} else {
				docs, err = indexFromPack(gctx, eng, t.crawlDir, t.source, warcIdx, func(done, docTotal int64, _ time.Duration) {
					totalDocs.Store(done)
					emitIndexProgress(emit, i, total, warcIdx, t.engine, t.source, done, docTotal, start)
				})
			}

			if err != nil {
				eng.Close()
				return err
			}
			totalDocs.Add(docs)

			if fin, ok := eng.(index.Finalizer); ok {
				if err := fin.Finalize(gctx); err != nil {
					eng.Close()
					return fmt.Errorf("finalize: %w", err)
				}
			}
			eng.Close()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return IndexMetric{}, err
	}

	return IndexMetric{
		Files:   total,
		Docs:    totalDocs.Load(),
		Elapsed: time.Since(start),
	}, nil
}

// IndexFromWARCMd indexes documents from a .md.warc.gz file into an engine.
// Exported for use by CLI (cli/cc_fts.go).
func IndexFromWARCMd(ctx context.Context, eng index.Engine, warcMdPath string, batchSize int, progress func(done, total int64, elapsed time.Duration)) (int64, error) {
	if _, err := os.Stat(warcMdPath); os.IsNotExist(err) {
		return 0, fmt.Errorf("warc_md file not found: %s", warcMdPath)
	}

	f, err := os.Open(warcMdPath)
	if err != nil {
		return 0, fmt.Errorf("open warc_md: %w", err)
	}
	defer f.Close()

	docCh := make(chan indexpack.Document, 256)
	errCh := make(chan error, 1)

	go func() {
		defer close(docCh)
		wr := warcpkg.NewReader(f)
		for wr.Next() {
			if ctx.Err() != nil {
				return
			}
			rec := wr.Record()
			if rec.Header.Type() != warcpkg.TypeConversion {
				io.Copy(io.Discard, rec.Body)
				continue
			}
			recordID := rec.Header.RecordID()
			docID := strings.TrimPrefix(recordID, "<urn:uuid:")
			docID = strings.TrimSuffix(docID, ">")
			docID = strings.TrimSpace(docID)
			if docID == "" || strings.ContainsAny(docID, ":<>") {
				io.Copy(io.Discard, rec.Body)
				continue
			}
			body, readErr := io.ReadAll(rec.Body)
			if readErr != nil {
				continue
			}
			select {
			case docCh <- indexpack.Document{DocID: docID, Text: body}:
			case <-ctx.Done():
				return
			}
		}
		if rErr := wr.Err(); rErr != nil {
			select {
			case errCh <- rErr:
			default:
			}
		}
	}()

	adaptProgress := func(done, total int64, elapsed time.Duration) {
		if progress != nil {
			progress(done, total, elapsed)
		}
	}
	if batchSize <= 0 {
		batchSize = 5000
	}
	stats, err := indexpack.RunPipelineFromChannel(ctx, eng, docCh, 0, batchSize, adaptProgress)
	select {
	case pErr := <-errCh:
		if err == nil {
			err = pErr
		}
	default:
	}
	if err != nil {
		return 0, err
	}
	return stats.DocsIndexed.Load(), nil
}

// openEngine creates and opens an FTS engine at the given directory.
func openEngine(ctx context.Context, name, dir string) (index.Engine, error) {
	eng, err := index.NewEngine(name)
	if err != nil {
		return nil, fmt.Errorf("create engine %s: %w", name, err)
	}
	if err := eng.Open(ctx, dir); err != nil {
		return nil, fmt.Errorf("open engine %s at %s: %w", name, dir, err)
	}
	return eng, nil
}

// indexFromWARCMd indexes documents from a warc_md/{shard}.md.warc.gz file into an engine.
func indexFromWARCMd(ctx context.Context, eng index.Engine, crawlDir, warcIdx string, progress func(done, total int64, elapsed time.Duration)) (int64, error) {
	warcMdPath := filepath.Join(crawlDir, "warc_md", warcIdx+".md.warc.gz")
	return IndexFromWARCMd(ctx, eng, warcMdPath, 5000, progress)
}

// indexFromPack indexes documents from a packed source file into an engine.
func indexFromPack(ctx context.Context, eng index.Engine, crawlDir, source, warcIdx string, progress func(done, total int64, elapsed time.Duration)) (int64, error) {
	packDir := filepath.Join(crawlDir, "pack")
	packFile, err := util.PackPath(packDir, source, warcIdx)
	if err != nil {
		return 0, err
	}
	if _, err := os.Stat(packFile); os.IsNotExist(err) {
		return 0, fmt.Errorf("pack file not found: %s", packFile)
	}

	adaptProgress := func(done, total int64, elapsed time.Duration) {
		if progress != nil {
			progress(done, total, elapsed)
		}
	}

	var stats *indexpack.PipelineStats
	switch source {
	case "parquet":
		stats, err = indexpack.RunPipelineFromParquet(ctx, eng, packFile, 5000, adaptProgress)
	case "bin":
		stats, err = indexpack.RunPipelineFromFlatBin(ctx, eng, packFile, 5000, adaptProgress)
	case "duckdb":
		stats, err = runPipelineFromDuckDBRaw(ctx, eng, packFile, 5000, adaptProgress)
	case "markdown":
		stats, err = indexpack.RunPipelineFromFlatBinGz(ctx, eng, packFile, 5000, adaptProgress)
	default:
		return 0, fmt.Errorf("unknown source %q (valid: files, parquet, bin, duckdb, markdown)", source)
	}
	if err != nil {
		return 0, err
	}
	return stats.DocsIndexed.Load(), nil
}

func emitIndexProgress(emit func(*IndexState), fileIdx, fileTotal int, warcIdx, engine, source string,
	docsIndexed, docsTotal int64, start time.Time) {
	if emit == nil {
		return
	}
	pct := util.PhaseProgress(docsIndexed, docsTotal)
	overall := util.FileProgress(fileIdx, fileTotal, pct)
	var dps float64
	if elapsed := time.Since(start); elapsed > 0 && docsIndexed > 0 {
		dps = float64(docsIndexed) / elapsed.Seconds()
	}
	emit(&IndexState{
		FileIndex:   fileIdx,
		FileTotal:   fileTotal,
		WARCIndex:   warcIdx,
		Engine:      engine,
		Source:      source,
		DocsIndexed: docsIndexed,
		DocsTotal:   docsTotal,
		Progress:    overall,
		DocsPerSec:  dps,
	})
}
