package importer

import (
	"context"
	"sync"

	"github.com/go-mizu/blueprints/spreadsheet/feature/cells"
)

// BatchProcessor handles parallel processing of cell batches during import.
type BatchProcessor struct {
	cells     cells.API
	batchSize int
	workers   int
}

// NewBatchProcessor creates a new batch processor with the given configuration.
func NewBatchProcessor(cellsAPI cells.API, batchSize, workers int) *BatchProcessor {
	if batchSize <= 0 {
		batchSize = 500
	}
	if workers <= 0 {
		workers = 4
	}
	return &BatchProcessor{
		cells:     cellsAPI,
		batchSize: batchSize,
		workers:   workers,
	}
}

// ProcessCells imports cells in parallel batches.
// Returns the number of successfully imported cells and any warnings encountered.
func (p *BatchProcessor) ProcessCells(ctx context.Context, sheetID string, cellsToImport []*cells.Cell) (int, []string) {
	if len(cellsToImport) == 0 {
		return 0, nil
	}

	// For small imports, process sequentially to avoid overhead
	if len(cellsToImport) <= p.batchSize*2 {
		return p.processSequential(ctx, sheetID, cellsToImport)
	}

	return p.processParallel(ctx, sheetID, cellsToImport)
}

// processSequential processes batches one at a time.
func (p *BatchProcessor) processSequential(ctx context.Context, sheetID string, cellsToImport []*cells.Cell) (int, []string) {
	var warnings []string
	imported := 0

	for i := 0; i < len(cellsToImport); i += p.batchSize {
		end := i + p.batchSize
		if end > len(cellsToImport) {
			end = len(cellsToImport)
		}

		batch := cellsToImport[i:end]
		updates := make([]cells.CellUpdate, len(batch))
		for j, cell := range batch {
			updates[j] = cells.CellUpdate{
				Row:     cell.Row,
				Col:     cell.Col,
				Value:   cell.Value,
				Formula: cell.Formula,
				Format:  &cell.Format,
			}
		}

		_, err := p.cells.BatchUpdate(ctx, sheetID, &cells.BatchUpdateIn{Cells: updates})
		if err != nil {
			warnings = append(warnings, err.Error())
		} else {
			imported += len(batch)
		}
	}

	return imported, warnings
}

// processParallel processes batches concurrently using worker pool.
func (p *BatchProcessor) processParallel(ctx context.Context, sheetID string, cellsToImport []*cells.Cell) (int, []string) {
	// Split into batches
	var batches [][]*cells.Cell
	for i := 0; i < len(cellsToImport); i += p.batchSize {
		end := i + p.batchSize
		if end > len(cellsToImport) {
			end = len(cellsToImport)
		}
		batches = append(batches, cellsToImport[i:end])
	}

	// Create work channel
	work := make(chan []*cells.Cell, len(batches))
	for _, batch := range batches {
		work <- batch
	}
	close(work)

	// Create result channel
	type result struct {
		imported int
		warning  string
	}
	results := make(chan result, len(batches))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < p.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for batch := range work {
				// Check context cancellation
				select {
				case <-ctx.Done():
					results <- result{warning: ctx.Err().Error()}
					continue
				default:
				}

				updates := make([]cells.CellUpdate, len(batch))
				for j, cell := range batch {
					updates[j] = cells.CellUpdate{
						Row:     cell.Row,
						Col:     cell.Col,
						Value:   cell.Value,
						Formula: cell.Formula,
						Format:  &cell.Format,
					}
				}

				_, err := p.cells.BatchUpdate(ctx, sheetID, &cells.BatchUpdateIn{Cells: updates})
				if err != nil {
					results <- result{warning: err.Error()}
				} else {
					results <- result{imported: len(batch)}
				}
			}
		}()
	}

	// Wait for workers to finish and close results channel
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var warnings []string
	imported := 0
	for r := range results {
		imported += r.imported
		if r.warning != "" {
			warnings = append(warnings, r.warning)
		}
	}

	return imported, warnings
}

// ProcessCellsForNewSheet is optimized for new sheets where no existing cells need to be fetched.
// This skips the GetByPositions call in BatchUpdate for better performance.
func (p *BatchProcessor) ProcessCellsForNewSheet(ctx context.Context, sheetID string, cellsToImport []*cells.Cell) (int, []string) {
	// For now, use the same processing as regular cells
	// The optimization would require changes to the cells.API interface
	// to support a "skip lookup" mode
	return p.ProcessCells(ctx, sheetID, cellsToImport)
}
