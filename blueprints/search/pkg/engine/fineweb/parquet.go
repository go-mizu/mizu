package fineweb

import (
	"context"
	"fmt"
	"io"
	"iter"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/parquet-go/parquet-go"
)

// ParquetReader reads documents from parquet files.
type ParquetReader struct {
	parquetDir string
	batchSize  int
}

// NewParquetReader creates a reader for the given parquet directory.
func NewParquetReader(parquetDir string) *ParquetReader {
	return &ParquetReader{
		parquetDir: parquetDir,
		batchSize:  10000, // Default batch size
	}
}

// WithBatchSize sets the batch size for reading.
func (r *ParquetReader) WithBatchSize(size int) *ParquetReader {
	r.batchSize = size
	return r
}

// ParquetDocument matches the parquet file schema.
type ParquetDocument struct {
	ID            string  `parquet:"id"`
	URL           string  `parquet:"url"`
	Text          string  `parquet:"text"`
	Dump          string  `parquet:"dump"`
	Date          string  `parquet:"date"`
	Language      string  `parquet:"language"`
	LanguageScore float64 `parquet:"language_score"`
}

// toDocument converts a parquet document to a Document.
func (p *ParquetDocument) toDocument() Document {
	return Document{
		ID:            p.ID,
		URL:           p.URL,
		Text:          p.Text,
		Dump:          p.Dump,
		Date:          p.Date,
		Language:      p.Language,
		LanguageScore: p.LanguageScore,
	}
}

// ListParquetFiles returns a sorted list of parquet files in the directory.
func (r *ParquetReader) ListParquetFiles() ([]string, error) {
	entries, err := os.ReadDir(r.parquetDir)
	if err != nil {
		return nil, fmt.Errorf("reading parquet directory: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".parquet") {
			files = append(files, filepath.Join(r.parquetDir, entry.Name()))
		}
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no parquet files found in %s", r.parquetDir)
	}

	sort.Strings(files)
	return files, nil
}

// CountDocuments returns the total number of documents across all parquet files.
func (r *ParquetReader) CountDocuments(ctx context.Context) (int64, error) {
	files, err := r.ListParquetFiles()
	if err != nil {
		return 0, err
	}

	var total int64
	for _, file := range files {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}

		f, err := os.Open(file)
		if err != nil {
			return 0, fmt.Errorf("opening %s: %w", file, err)
		}

		stat, err := f.Stat()
		if err != nil {
			f.Close()
			return 0, fmt.Errorf("stat %s: %w", file, err)
		}

		pf, err := parquet.OpenFile(f, stat.Size())
		if err != nil {
			f.Close()
			return 0, fmt.Errorf("opening parquet %s: %w", file, err)
		}

		total += pf.NumRows()
		f.Close()
	}

	return total, nil
}

// ReadAll returns an iterator over all documents in all parquet files.
func (r *ParquetReader) ReadAll(ctx context.Context) iter.Seq2[Document, error] {
	return func(yield func(Document, error) bool) {
		files, err := r.ListParquetFiles()
		if err != nil {
			yield(Document{}, err)
			return
		}

		for _, file := range files {
			select {
			case <-ctx.Done():
				yield(Document{}, ctx.Err())
				return
			default:
			}

			if !r.readFile(ctx, file, yield) {
				return
			}
		}
	}
}

// ReadFile returns an iterator over documents in a single parquet file.
func (r *ParquetReader) ReadFile(ctx context.Context, file string) iter.Seq2[Document, error] {
	return func(yield func(Document, error) bool) {
		r.readFile(ctx, file, yield)
	}
}

func (r *ParquetReader) readFile(ctx context.Context, file string, yield func(Document, error) bool) bool {
	f, err := os.Open(file)
	if err != nil {
		return yield(Document{}, fmt.Errorf("opening %s: %w", file, err))
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return yield(Document{}, fmt.Errorf("stat %s: %w", file, err))
	}

	pf, err := parquet.OpenFile(f, stat.Size())
	if err != nil {
		return yield(Document{}, fmt.Errorf("opening parquet %s: %w", file, err))
	}

	reader := parquet.NewGenericReader[ParquetDocument](pf)
	defer reader.Close()

	batch := make([]ParquetDocument, r.batchSize)
	for {
		select {
		case <-ctx.Done():
			return yield(Document{}, ctx.Err())
		default:
		}

		n, err := reader.Read(batch)
		if err != nil && err != io.EOF {
			return yield(Document{}, fmt.Errorf("reading parquet %s: %w", file, err))
		}

		for i := 0; i < n; i++ {
			if !yield(batch[i].toDocument(), nil) {
				return false
			}
		}

		if err == io.EOF || n == 0 {
			break
		}
	}

	return true
}

// ReadBatches returns an iterator over batches of documents.
func (r *ParquetReader) ReadBatches(ctx context.Context) iter.Seq2[[]Document, error] {
	return func(yield func([]Document, error) bool) {
		files, err := r.ListParquetFiles()
		if err != nil {
			yield(nil, err)
			return
		}

		for _, file := range files {
			select {
			case <-ctx.Done():
				yield(nil, ctx.Err())
				return
			default:
			}

			if !r.readFileBatches(ctx, file, yield) {
				return
			}
		}
	}
}

func (r *ParquetReader) readFileBatches(ctx context.Context, file string, yield func([]Document, error) bool) bool {
	f, err := os.Open(file)
	if err != nil {
		return yield(nil, fmt.Errorf("opening %s: %w", file, err))
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return yield(nil, fmt.Errorf("stat %s: %w", file, err))
	}

	pf, err := parquet.OpenFile(f, stat.Size())
	if err != nil {
		return yield(nil, fmt.Errorf("opening parquet %s: %w", file, err))
	}

	reader := parquet.NewGenericReader[ParquetDocument](pf)
	defer reader.Close()

	batch := make([]ParquetDocument, r.batchSize)
	for {
		select {
		case <-ctx.Done():
			return yield(nil, ctx.Err())
		default:
		}

		n, err := reader.Read(batch)
		if err != nil && err != io.EOF {
			return yield(nil, fmt.Errorf("reading parquet %s: %w", file, err))
		}

		if n > 0 {
			docs := make([]Document, n)
			for i := 0; i < n; i++ {
				docs[i] = batch[i].toDocument()
			}
			if !yield(docs, nil) {
				return false
			}
		}

		if err == io.EOF || n == 0 {
			break
		}
	}

	return true
}
