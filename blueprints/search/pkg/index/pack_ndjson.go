package index

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// PackNDJSON packs all markdown files from markdownDir into a newline-delimited JSON file.
//
// Wire format: one JSON object per line — {"i":"<doc_id>","t":"<text>"}\n
// Newlines and special characters inside text are JSON-escaped.
func PackNDJSON(ctx context.Context, markdownDir, packPath string, workers, batchSize int, progress ProgressFunc) (*PipelineStats, error) {
	if err := os.MkdirAll(filepath.Dir(packPath), 0o755); err != nil {
		return nil, err
	}
	f, err := os.Create(packPath)
	if err != nil {
		return nil, err
	}

	bw := bufio.NewWriterSize(f, 1<<20) // 1 MB write buffer
	enc := json.NewEncoder(bw)
	enc.SetEscapeHTML(false)

	type ndjsonRec struct {
		I string `json:"i"`
		T string `json:"t"`
	}

	eng := &funcEngine{
		name: "ndjson-writer",
		indexFn: func(_ context.Context, docs []Document) error {
			for _, doc := range docs {
				if err := enc.Encode(ndjsonRec{I: doc.DocID, T: string(doc.Text)}); err != nil {
					return err
				}
			}
			return nil
		},
	}

	stats, pipeErr := RunPipeline(ctx, eng, PipelineConfig{
		SourceDir: markdownDir,
		BatchSize: batchSize,
		Workers:   workers,
	}, progress)

	flushErr := bw.Flush()
	closeErr := f.Close()

	if pipeErr != nil {
		os.Remove(packPath)
		return stats, pipeErr
	}
	if flushErr != nil {
		os.Remove(packPath)
		return stats, flushErr
	}
	return stats, closeErr
}

// RunPipelineFromNDJSON reads an NDJSON pack file and feeds documents into engine.
func RunPipelineFromNDJSON(ctx context.Context, engine Engine, packPath string, batchSize int, progress PackProgressFunc) (*PipelineStats, error) {
	f, err := os.Open(packPath)
	if err != nil {
		return nil, fmt.Errorf("open ndjson: %w", err)
	}
	defer f.Close()

	docCh := make(chan Document, max(batchSize*2, 1024))
	go func() {
		defer close(docCh)
		br := bufio.NewReaderSize(f, 1<<20)
		var rec struct {
			I string `json:"i"`
			T string `json:"t"`
		}
		for {
			if ctx.Err() != nil {
				return
			}
			line, err := br.ReadBytes('\n')
			if len(line) > 0 {
				line = bytes.TrimRight(line, "\r\n")
				if len(line) > 0 {
					if jsonErr := json.Unmarshal(line, &rec); jsonErr == nil {
						select {
						case docCh <- Document{DocID: rec.I, Text: []byte(rec.T)}:
						case <-ctx.Done():
							return
						}
					}
				}
			}
			if err == io.EOF {
				return
			}
			if err != nil {
				return
			}
		}
	}()

	return RunPipelineFromChannel(ctx, engine, docCh, 0, batchSize, progress)
}
