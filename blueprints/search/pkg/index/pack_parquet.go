package index

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/parquet-go/parquet-go"
)

// packParquetDoc is the schema for the FTS pack parquet file.
type packParquetDoc struct {
	DocID string `parquet:"doc_id"`
	Text  string `parquet:"text"`
}

// PackParquet packs all markdown files from markdownDir into a Parquet file.
// Schema: doc_id STRING, text STRING.
func PackParquet(ctx context.Context, markdownDir, packPath string, workers, batchSize int, progress ProgressFunc) (*PipelineStats, error) {
	if err := os.MkdirAll(filepath.Dir(packPath), 0o755); err != nil {
		return nil, err
	}
	f, err := os.Create(packPath)
	if err != nil {
		return nil, err
	}

	pw := parquet.NewGenericWriter[packParquetDoc](f)

	eng := &funcEngine{
		name: "parquet-writer",
		indexFn: func(_ context.Context, docs []Document) error {
			rows := make([]packParquetDoc, len(docs))
			for i, d := range docs {
				rows[i] = packParquetDoc{DocID: d.DocID, Text: d.Text}
			}
			_, err := pw.Write(rows)
			return err
		},
	}

	stats, pipeErr := RunPipeline(ctx, eng, PipelineConfig{
		SourceDir: markdownDir,
		BatchSize: batchSize,
		Workers:   workers,
	}, progress)

	closeErr := pw.Close()
	fCloseErr := f.Close()

	if pipeErr != nil {
		os.Remove(packPath)
		return stats, pipeErr
	}
	if closeErr != nil {
		os.Remove(packPath)
		return stats, closeErr
	}
	return stats, fCloseErr
}

// RunPipelineFromParquet reads a Parquet pack file and feeds documents into engine.
// total is pre-read from the parquet footer so progress can show percentage.
func RunPipelineFromParquet(ctx context.Context, engine Engine, packPath string, batchSize int, progress PackProgressFunc) (*PipelineStats, error) {
	f, err := os.Open(packPath)
	if err != nil {
		return nil, fmt.Errorf("open parquet: %w", err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat parquet: %w", err)
	}

	pf, err := parquet.OpenFile(f, stat.Size())
	if err != nil {
		return nil, fmt.Errorf("parse parquet: %w", err)
	}

	total := pf.NumRows()

	docCh := make(chan Document, max(batchSize*2, 1024))
	go func() {
		defer close(docCh)
		r := parquet.NewGenericReader[packParquetDoc](pf)
		defer r.Close()
		batch := make([]packParquetDoc, batchSize)
		for {
			if ctx.Err() != nil {
				return
			}
			n, readErr := r.Read(batch)
			for i := range n {
				select {
				case docCh <- Document{DocID: batch[i].DocID, Text: batch[i].Text}:
				case <-ctx.Done():
					return
				}
			}
			if readErr == io.EOF || n == 0 {
				return
			}
			if readErr != nil {
				return
			}
		}
	}()

	return RunPipelineFromChannel(ctx, engine, docCh, total, batchSize, progress)
}
