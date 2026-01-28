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

// ReadN returns an iterator that yields at most n documents.
func (r *ParquetReader) ReadN(ctx context.Context, n int) iter.Seq2[Document, error] {
	return func(yield func(Document, error) bool) {
		count := 0
		for doc, err := range r.ReadAll(ctx) {
			if err != nil {
				yield(Document{}, err)
				return
			}
			if !yield(doc, nil) {
				return
			}
			count++
			if count >= n {
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

// ReadRowGroupsParallel reads row groups in parallel for higher throughput.
// numWorkers specifies how many row groups to read concurrently.
func (r *ParquetReader) ReadRowGroupsParallel(ctx context.Context, numWorkers int) iter.Seq2[[]Document, error] {
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

			if !r.readFileRowGroupsParallel(ctx, file, numWorkers, yield) {
				return
			}
		}
	}
}

func (r *ParquetReader) readFileRowGroupsParallel(ctx context.Context, file string, numWorkers int, yield func([]Document, error) bool) bool {
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

	// Get row groups
	rowGroups := pf.RowGroups()
	numRowGroups := len(rowGroups)
	if numRowGroups == 0 {
		return true
	}

	// Process row groups in parallel batches
	if numWorkers > numRowGroups {
		numWorkers = numRowGroups
	}

	// Use channels for parallel row group reading
	type rowGroupResult struct {
		idx  int
		docs []Document
		err  error
	}

	for startIdx := 0; startIdx < numRowGroups; startIdx += numWorkers {
		endIdx := startIdx + numWorkers
		if endIdx > numRowGroups {
			endIdx = numRowGroups
		}

		select {
		case <-ctx.Done():
			return yield(nil, ctx.Err())
		default:
		}

		// Read row groups in parallel
		results := make(chan rowGroupResult, endIdx-startIdx)
		for rgIdx := startIdx; rgIdx < endIdx; rgIdx++ {
			go func(idx int) {
				rg := rowGroups[idx]
				reader := parquet.NewGenericRowGroupReader[ParquetDocument](rg)
				defer reader.Close()

				batch := make([]ParquetDocument, rg.NumRows())
				n, err := reader.Read(batch)
				if err != nil && err != io.EOF {
					results <- rowGroupResult{idx: idx, err: err}
					return
				}

				docs := make([]Document, n)
				for i := 0; i < n; i++ {
					docs[i] = batch[i].toDocument()
				}
				results <- rowGroupResult{idx: idx, docs: docs}
			}(rgIdx)
		}

		// Collect results in order
		resultsMap := make(map[int]rowGroupResult)
		for i := startIdx; i < endIdx; i++ {
			res := <-results
			resultsMap[res.idx] = res
		}

		// Yield in order
		for rgIdx := startIdx; rgIdx < endIdx; rgIdx++ {
			res := resultsMap[rgIdx]
			if res.err != nil {
				return yield(nil, fmt.Errorf("reading row group %d: %w", rgIdx, res.err))
			}
			if len(res.docs) > 0 {
				if !yield(res.docs, nil) {
					return false
				}
			}
		}
	}

	return true
}

// ReadTextsOnlyBatches returns an iterator that only extracts ID and Text fields.
// This is much faster as it skips unnecessary field extraction.
func (r *ParquetReader) ReadTextsOnlyBatches(ctx context.Context) iter.Seq2[[]TextOnlyDoc, error] {
	return func(yield func([]TextOnlyDoc, error) bool) {
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

			if !r.readFileTextsOnly(ctx, file, yield) {
				return
			}
		}
	}
}

// TextOnlyDoc contains just ID and Text for fast indexing.
type TextOnlyDoc struct {
	ID   string
	Text string
}

// TextOnlyParquet is the minimal parquet struct for fast reading.
type TextOnlyParquet struct {
	ID   string `parquet:"id"`
	Text string `parquet:"text"`
}

func (r *ParquetReader) readFileTextsOnly(ctx context.Context, file string, yield func([]TextOnlyDoc, error) bool) bool {
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

	reader := parquet.NewGenericReader[TextOnlyParquet](pf)
	defer reader.Close()

	batch := make([]TextOnlyParquet, r.batchSize)
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
			docs := make([]TextOnlyDoc, n)
			for i := 0; i < n; i++ {
				docs[i] = TextOnlyDoc{ID: batch[i].ID, Text: batch[i].Text}
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

// ReadTextsOnlyParallel reads text-only documents with parallel row group decompression.
// numWorkers specifies how many row groups to read concurrently.
func (r *ParquetReader) ReadTextsOnlyParallel(ctx context.Context, numWorkers int) iter.Seq2[[]TextOnlyDoc, error] {
	return func(yield func([]TextOnlyDoc, error) bool) {
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

			if !r.readFileTextsOnlyParallel(ctx, file, numWorkers, yield) {
				return
			}
		}
	}
}

func (r *ParquetReader) readFileTextsOnlyParallel(ctx context.Context, file string, numWorkers int, yield func([]TextOnlyDoc, error) bool) bool {
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

	rowGroups := pf.RowGroups()
	numRowGroups := len(rowGroups)
	if numRowGroups == 0 {
		return true
	}

	if numWorkers > numRowGroups {
		numWorkers = numRowGroups
	}
	if numWorkers < 1 {
		numWorkers = 1
	}

	// Process row groups in parallel batches for ordered output
	type rowGroupResult struct {
		idx  int
		docs []TextOnlyDoc
		err  error
	}

	for startIdx := 0; startIdx < numRowGroups; startIdx += numWorkers {
		endIdx := startIdx + numWorkers
		if endIdx > numRowGroups {
			endIdx = numRowGroups
		}

		select {
		case <-ctx.Done():
			return yield(nil, ctx.Err())
		default:
		}

		// Read row groups in parallel
		results := make(chan rowGroupResult, endIdx-startIdx)
		for rgIdx := startIdx; rgIdx < endIdx; rgIdx++ {
			go func(idx int) {
				rg := rowGroups[idx]
				reader := parquet.NewGenericRowGroupReader[TextOnlyParquet](rg)
				defer reader.Close()

				batch := make([]TextOnlyParquet, rg.NumRows())
				n, err := reader.Read(batch)
				if err != nil && err != io.EOF {
					results <- rowGroupResult{idx: idx, err: err}
					return
				}

				docs := make([]TextOnlyDoc, n)
				for i := 0; i < n; i++ {
					docs[i] = TextOnlyDoc{ID: batch[i].ID, Text: batch[i].Text}
				}
				results <- rowGroupResult{idx: idx, docs: docs}
			}(rgIdx)
		}

		// Collect results in order
		resultsMap := make(map[int]rowGroupResult)
		for i := startIdx; i < endIdx; i++ {
			res := <-results
			resultsMap[res.idx] = res
		}

		// Yield in order
		for rgIdx := startIdx; rgIdx < endIdx; rgIdx++ {
			res := resultsMap[rgIdx]
			if res.err != nil {
				return yield(nil, fmt.Errorf("reading row group %d: %w", rgIdx, res.err))
			}
			if len(res.docs) > 0 {
				if !yield(res.docs, nil) {
					return false
				}
			}
		}
	}

	return true
}
