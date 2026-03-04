package index

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	kgzip "github.com/klauspost/compress/gzip"
)

// PipelineConfig controls the indexing pipeline.
type PipelineConfig struct {
	SourceDir string // markdown/ directory path
	BatchSize int    // docs per Engine.Index call (default 5000)
	Workers   int    // parallel file readers (default NumCPU)
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

// gzReaderPool pools klauspost gzip readers for reuse across workers.
var gzReaderPool sync.Pool

// RunPipeline indexes all markdown files from sourceDir into engine.
func RunPipeline(ctx context.Context, engine Engine, cfg PipelineConfig, progress ProgressFunc) (*PipelineStats, error) {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 5000
	}
	if cfg.Workers <= 0 {
		cfg.Workers = runtime.NumCPU()
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

	// Large channel buffers to keep all stages busy without blocking
	fileCh := make(chan string, 8192)
	docCh := make(chan Document, cfg.BatchSize*2)

	// Stage 1: Parallel walker
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

// walkMarkdown walks dir for .md and .md.gz files, sending paths to out.
// Top-level subdirectories are walked in parallel (up to NumCPU walkers).
func walkMarkdown(ctx context.Context, dir string, out chan<- string, stats *PipelineStats) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("readdir %s: %w", dir, err)
	}

	// Collect top-level files and subdirs in one pass
	var subdirs []string
	for _, e := range entries {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		name := e.Name()
		if e.IsDir() {
			subdirs = append(subdirs, filepath.Join(dir, name))
			continue
		}
		if strings.HasSuffix(name, ".md.gz") || strings.HasSuffix(name, ".md") {
			stats.TotalFiles.Add(1)
			select {
			case out <- filepath.Join(dir, name):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	if len(subdirs) == 0 {
		return nil
	}

	// Walk subdirs in parallel with up to NumCPU walkers
	walkerCount := runtime.NumCPU()
	if walkerCount > len(subdirs) {
		walkerCount = len(subdirs)
	}

	subdirCh := make(chan string, len(subdirs))
	for _, d := range subdirs {
		subdirCh <- d
	}
	close(subdirCh)

	var wg sync.WaitGroup
	var firstErrMu sync.Mutex
	var firstErr error

	for i := 0; i < walkerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for subdir := range subdirCh {
				if ctx.Err() != nil {
					return
				}
				walkErr := filepath.WalkDir(subdir, func(path string, d fs.DirEntry, err error) error {
					if err != nil {
						return nil // skip unreadable entries
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
				if walkErr != nil && ctx.Err() == nil {
					firstErrMu.Lock()
					if firstErr == nil {
						firstErr = walkErr
					}
					firstErrMu.Unlock()
				}
			}
		}()
	}
	wg.Wait()

	if firstErr != nil {
		return firstErr
	}
	return ctx.Err()
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
		if len(doc.Text) == 0 {
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
	var data []byte

	if strings.HasSuffix(path, ".gz") {
		f, err := os.Open(path)
		if err != nil {
			return Document{}, err
		}
		defer f.Close()

		// Try to reuse a pooled gzip reader
		var gr *kgzip.Reader
		if v := gzReaderPool.Get(); v != nil {
			gr = v.(*kgzip.Reader)
			if resetErr := gr.Reset(f); resetErr != nil {
				gr = nil // reader in bad state, allocate fresh
			}
		}
		if gr == nil {
			gr, err = kgzip.NewReader(f)
			if err != nil {
				return Document{}, err
			}
		}

		data, err = io.ReadAll(gr)
		gr.Close()
		if err == nil {
			gzReaderPool.Put(gr)
		}
		if err != nil {
			return Document{}, err
		}
	} else {
		// Plain .md file: fast path using os.ReadFile (preallocates from stat)
		var err error
		data, err = os.ReadFile(path)
		if err != nil {
			return Document{}, err
		}
	}

	// Extract UUID from filename: strip directory and extensions
	base := filepath.Base(path)
	docID := strings.TrimSuffix(strings.TrimSuffix(base, ".gz"), ".md")

	if !utf8.Valid(data) {
		data = []byte(strings.ToValidUTF8(string(data), "\uFFFD"))
	}

	return Document{
		DocID: docID,
		Text:  data,
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
