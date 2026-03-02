package index

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode/utf8"
	"sync"
	"sync/atomic"
	"time"
)

// PipelineConfig controls the indexing pipeline.
type PipelineConfig struct {
	SourceDir string // markdown/ directory path
	BatchSize int    // docs per Engine.Index call (default 5000)
	Workers   int    // parallel file readers (default 4)
}

// PipelineStats tracks pipeline progress.
type PipelineStats struct {
	TotalFiles  atomic.Int64
	DocsIndexed atomic.Int64
	Errors      atomic.Int64
	StartTime   time.Time
	PeakRSSMB   atomic.Int64 // peak RSS in MB
}

// ProgressFunc is called periodically with current stats.
type ProgressFunc func(stats *PipelineStats)

// RunPipeline indexes all markdown files from sourceDir into engine.
func RunPipeline(ctx context.Context, engine Engine, cfg PipelineConfig, progress ProgressFunc) (*PipelineStats, error) {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 5000
	}
	if cfg.Workers <= 0 {
		cfg.Workers = 4
	}

	stats := &PipelineStats{StartTime: time.Now()}

	// Start memory tracker
	memStop := make(chan struct{})
	go trackPeakMem(stats, memStop)
	defer close(memStop)

	// Start progress ticker
	if progress != nil {
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		go func() {
			for {
				select {
				case <-ticker.C:
					progress(stats)
				case <-ctx.Done():
					return
				case <-memStop:
					return
				}
			}
		}()
	}

	fileCh := make(chan string, 1000)
	docCh := make(chan Document, 5000)

	// Stage 1: Walker
	var walkErr error
	go func() {
		defer close(fileCh)
		walkErr = walkMarkdown(ctx, cfg.SourceDir, fileCh, stats)
	}()

	// Stage 2: Readers
	var readerWg sync.WaitGroup
	for i := 0; i < cfg.Workers; i++ {
		readerWg.Add(1)
		go func() {
			defer readerWg.Done()
			readFiles(ctx, fileCh, docCh, stats)
		}()
	}
	go func() {
		readerWg.Wait()
		close(docCh)
	}()

	// Stage 3: Batcher
	if err := batchIndex(ctx, engine, docCh, cfg.BatchSize, stats); err != nil {
		return stats, err
	}

	if walkErr != nil {
		return stats, walkErr
	}

	// Final progress call
	if progress != nil {
		progress(stats)
	}

	return stats, nil
}

func walkMarkdown(ctx context.Context, dir string, out chan<- string, stats *PipelineStats) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable dirs
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if strings.HasSuffix(name, ".md.gz") || strings.HasSuffix(name, ".md") {
			stats.TotalFiles.Add(1)
			select {
			case out <- path:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	})
}

func readFiles(ctx context.Context, in <-chan string, out chan<- Document, stats *PipelineStats) {
	for path := range in {
		if ctx.Err() != nil {
			return
		}
		doc, err := readMarkdownFile(path)
		if err != nil {
			stats.Errors.Add(1)
			continue
		}
		if doc.Text == "" {
			continue
		}
		select {
		case out <- doc:
		case <-ctx.Done():
			return
		}
	}
}

func readMarkdownFile(path string) (Document, error) {
	f, err := os.Open(path)
	if err != nil {
		return Document{}, err
	}
	defer f.Close()

	var r io.Reader = f
	if strings.HasSuffix(path, ".gz") {
		gr, err := gzip.NewReader(f)
		if err != nil {
			return Document{}, err
		}
		defer gr.Close()
		r = gr
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return Document{}, err
	}

	// Extract UUID from filename: strip directory and extensions
	base := filepath.Base(path)
	docID := strings.TrimSuffix(strings.TrimSuffix(base, ".gz"), ".md")

	text := string(data)
	if !utf8.ValidString(text) {
		text = strings.ToValidUTF8(text, "\uFFFD")
	}

	return Document{
		DocID: docID,
		Text:  text,
	}, nil
}

func batchIndex(ctx context.Context, engine Engine, docs <-chan Document, batchSize int, stats *PipelineStats) error {
	batch := make([]Document, 0, batchSize)
	for doc := range docs {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		batch = append(batch, doc)
		if len(batch) >= batchSize {
			if err := engine.Index(ctx, batch); err != nil {
				return fmt.Errorf("index batch: %w", err)
			}
			stats.DocsIndexed.Add(int64(len(batch)))
			batch = batch[:0]
		}
	}
	if len(batch) > 0 {
		if err := engine.Index(ctx, batch); err != nil {
			return fmt.Errorf("index final batch: %w", err)
		}
		stats.DocsIndexed.Add(int64(len(batch)))
	}
	return nil
}

func trackPeakMem(stats *PipelineStats, stop <-chan struct{}) {
	var m runtime.MemStats
	for {
		select {
		case <-stop:
			return
		case <-time.After(500 * time.Millisecond):
			runtime.ReadMemStats(&m)
			mb := int64(m.Sys / (1024 * 1024))
			for {
				cur := stats.PeakRSSMB.Load()
				if mb <= cur || stats.PeakRSSMB.CompareAndSwap(cur, mb) {
					break
				}
			}
		}
	}
}

// DirSizeBytes returns total size of files in dir.
func DirSizeBytes(dir string) int64 {
	var total int64
	filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		total += info.Size()
		return nil
	})
	return total
}
