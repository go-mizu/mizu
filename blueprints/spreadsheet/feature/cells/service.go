package cells

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/spreadsheet/pkg/formula"
	"github.com/go-mizu/blueprints/spreadsheet/pkg/ulid"
)

var (
	ErrNotFound     = errors.New("cell not found")
	ErrInvalidRange = errors.New("invalid range")
)

// Service implements the cells API.
type Service struct {
	store Store
}

// NewService creates a new cells service.
func NewService(store Store, secret string) *Service {
	return &Service{store: store}
}

// Get retrieves a cell by position.
func (s *Service) Get(ctx context.Context, sheetID string, row, col int) (*Cell, error) {
	return s.store.Get(ctx, sheetID, row, col)
}

// GetRange retrieves cells in a range.
func (s *Service) GetRange(ctx context.Context, sheetID string, startRow, startCol, endRow, endCol int) ([]*Cell, error) {
	if startRow > endRow || startCol > endCol {
		return nil, ErrInvalidRange
	}
	return s.store.GetRange(ctx, sheetID, startRow, startCol, endRow, endCol)
}

// Set sets a single cell value or formula.
func (s *Service) Set(ctx context.Context, sheetID string, row, col int, in *SetCellIn) (*Cell, error) {
	now := time.Now()

	cell := &Cell{
		ID:        ulid.New(),
		SheetID:   sheetID,
		Row:       row,
		Col:       col,
		UpdatedAt: now,
	}

	// Check if cell already exists to preserve existing data
	existing, err := s.store.Get(ctx, sheetID, row, col)
	if err == nil && existing != nil {
		cell.ID = existing.ID
		cell.Format = existing.Format
		cell.Hyperlink = existing.Hyperlink
		cell.Note = existing.Note
	}

	// Apply format if provided
	if in.Format != nil {
		cell.Format = mergeFormats(cell.Format, *in.Format)
	}

	if in.Formula != "" {
		cell.Formula = in.Formula
		cell.Type = CellTypeFormula

		// Evaluate formula
		value, display, err := s.evaluateFormula(ctx, sheetID, in.Formula, row, col)
		if err != nil {
			cell.Value = nil
			cell.Display = fmt.Sprintf("#ERROR: %v", err)
			cell.Type = CellTypeError
		} else {
			cell.Value = value
			cell.Display = display
		}
	} else {
		cell.Value = in.Value
		cell.Type = detectCellType(in.Value)
		cell.Display = formatDisplay(in.Value, cell.Format)
	}

	if err := s.store.Set(ctx, cell); err != nil {
		return nil, err
	}

	return cell, nil
}

// BatchUpdate updates multiple cells at once.
// Optimized to fetch all existing cells in a single query instead of N+1 queries.
func (s *Service) BatchUpdate(ctx context.Context, sheetID string, in *BatchUpdateIn) ([]*Cell, error) {
	cells := make([]*Cell, 0, len(in.Cells))
	now := time.Now()

	// Collect all positions we need to check - single query instead of N queries
	positions := make([]CellPosition, len(in.Cells))
	for i, update := range in.Cells {
		positions[i] = CellPosition{Row: update.Row, Col: update.Col}
	}

	// Fetch all existing cells in one query
	existingCells, err := s.store.GetByPositions(ctx, sheetID, positions)
	if err != nil {
		return nil, err
	}

	for _, update := range in.Cells {
		cell := &Cell{
			ID:        ulid.New(),
			SheetID:   sheetID,
			Row:       update.Row,
			Col:       update.Col,
			UpdatedAt: now,
		}

		// Check if cell already exists using the pre-fetched map
		pos := CellPosition{Row: update.Row, Col: update.Col}
		if existing, ok := existingCells[pos]; ok {
			cell.ID = existing.ID
			cell.Format = existing.Format
			cell.Hyperlink = existing.Hyperlink
			cell.Note = existing.Note
		}

		// Apply format if provided
		if update.Format != nil {
			cell.Format = mergeFormats(cell.Format, *update.Format)
		}

		if update.Formula != "" {
			cell.Formula = update.Formula
			cell.Type = CellTypeFormula

			value, display, err := s.evaluateFormula(ctx, sheetID, update.Formula, update.Row, update.Col)
			if err != nil {
				cell.Value = nil
				cell.Display = fmt.Sprintf("#ERROR: %v", err)
				cell.Type = CellTypeError
			} else {
				cell.Value = value
				cell.Display = display
			}
		} else {
			cell.Value = update.Value
			cell.Type = detectCellType(update.Value)
			cell.Display = formatDisplay(update.Value, cell.Format)
		}

		cells = append(cells, cell)
	}

	if err := s.store.BatchSet(ctx, cells); err != nil {
		return nil, err
	}

	return cells, nil
}

// Delete deletes a cell.
func (s *Service) Delete(ctx context.Context, sheetID string, row, col int) error {
	return s.store.Delete(ctx, sheetID, row, col)
}

// SetFormat sets cell formatting.
func (s *Service) SetFormat(ctx context.Context, in *SetFormatIn) error {
	cell, err := s.store.Get(ctx, in.SheetID, in.Row, in.Col)
	if err != nil {
		// Create empty cell with format
		cell = &Cell{
			ID:        ulid.New(),
			SheetID:   in.SheetID,
			Row:       in.Row,
			Col:       in.Col,
			Type:      CellTypeText,
			UpdatedAt: time.Now(),
		}
	}

	cell.Format = mergeFormats(cell.Format, in.Format)
	cell.Display = formatDisplay(cell.Value, cell.Format)
	cell.UpdatedAt = time.Now()

	return s.store.Set(ctx, cell)
}

// SetRangeFormat sets formatting for a range.
// Optimized to fetch all cells once and batch update instead of N×M queries.
func (s *Service) SetRangeFormat(ctx context.Context, sheetID string, startRow, startCol, endRow, endCol int, format Format) error {
	// Fetch all existing cells in the range with a single query
	existingCells, err := s.store.GetRange(ctx, sheetID, startRow, startCol, endRow, endCol)
	if err != nil {
		return err
	}

	// Create a map for quick lookup
	existingMap := make(map[CellPosition]*Cell)
	for _, cell := range existingCells {
		pos := CellPosition{Row: cell.Row, Col: cell.Col}
		existingMap[pos] = cell
	}

	// Build list of cells to update
	now := time.Now()
	cellsToUpdate := make([]*Cell, 0, (endRow-startRow+1)*(endCol-startCol+1))

	for row := startRow; row <= endRow; row++ {
		for col := startCol; col <= endCol; col++ {
			pos := CellPosition{Row: row, Col: col}
			var cell *Cell

			if existing, ok := existingMap[pos]; ok {
				// Update existing cell
				cell = existing
			} else {
				// Create new empty cell with format
				cell = &Cell{
					ID:      ulid.New(),
					SheetID: sheetID,
					Row:     row,
					Col:     col,
					Type:    CellTypeText,
				}
			}

			cell.Format = mergeFormats(cell.Format, format)
			cell.Display = formatDisplay(cell.Value, cell.Format)
			cell.UpdatedAt = now

			cellsToUpdate = append(cellsToUpdate, cell)
		}
	}

	// Batch update all cells at once
	return s.store.BatchSet(ctx, cellsToUpdate)
}

// Clear clears a cell.
func (s *Service) Clear(ctx context.Context, sheetID string, row, col int) error {
	return s.store.Delete(ctx, sheetID, row, col)
}

// ClearRange clears a range of cells.
func (s *Service) ClearRange(ctx context.Context, sheetID string, startRow, startCol, endRow, endCol int) error {
	return s.store.DeleteRange(ctx, sheetID, startRow, startCol, endRow, endCol)
}

// SetNote sets a cell note.
func (s *Service) SetNote(ctx context.Context, sheetID string, row, col int, note string) error {
	cell, err := s.store.Get(ctx, sheetID, row, col)
	if err != nil {
		cell = &Cell{
			ID:        ulid.New(),
			SheetID:   sheetID,
			Row:       row,
			Col:       col,
			Type:      CellTypeText,
			UpdatedAt: time.Now(),
		}
	}

	cell.Note = note
	cell.UpdatedAt = time.Now()

	return s.store.Set(ctx, cell)
}

// SetHyperlink sets a cell hyperlink.
func (s *Service) SetHyperlink(ctx context.Context, sheetID string, row, col int, hyperlink *Hyperlink) error {
	cell, err := s.store.Get(ctx, sheetID, row, col)
	if err != nil {
		cell = &Cell{
			ID:        ulid.New(),
			SheetID:   sheetID,
			Row:       row,
			Col:       col,
			Type:      CellTypeText,
			UpdatedAt: time.Now(),
		}
	}

	cell.Hyperlink = hyperlink
	cell.UpdatedAt = time.Now()

	return s.store.Set(ctx, cell)
}

// Merge merges cells.
func (s *Service) Merge(ctx context.Context, sheetID string, startRow, startCol, endRow, endCol int) (*MergedRegion, error) {
	region := &MergedRegion{
		ID:       ulid.New(),
		SheetID:  sheetID,
		StartRow: startRow,
		StartCol: startCol,
		EndRow:   endRow,
		EndCol:   endCol,
	}

	if err := s.store.CreateMerge(ctx, region); err != nil {
		return nil, err
	}

	return region, nil
}

// Unmerge unmerges cells.
func (s *Service) Unmerge(ctx context.Context, sheetID string, startRow, startCol, endRow, endCol int) error {
	return s.store.DeleteMerge(ctx, sheetID, startRow, startCol, endRow, endCol)
}

// GetMergedRegions gets merged regions in a sheet.
func (s *Service) GetMergedRegions(ctx context.Context, sheetID string) ([]*MergedRegion, error) {
	return s.store.GetMergedRegions(ctx, sheetID)
}

// CopyRange copies a range of cells.
func (s *Service) CopyRange(ctx context.Context, sourceSheetID string, sourceRange Range, destSheetID string, destRow, destCol int) error {
	cells, err := s.store.GetRange(ctx, sourceSheetID, sourceRange.StartRow, sourceRange.StartCol, sourceRange.EndRow, sourceRange.EndCol)
	if err != nil {
		return err
	}

	rowOffset := destRow - sourceRange.StartRow
	colOffset := destCol - sourceRange.StartCol

	newCells := make([]*Cell, 0, len(cells))
	now := time.Now()

	for _, cell := range cells {
		newCell := &Cell{
			ID:        ulid.New(),
			SheetID:   destSheetID,
			Row:       cell.Row + rowOffset,
			Col:       cell.Col + colOffset,
			Value:     cell.Value,
			Formula:   cell.Formula, // TODO: Adjust formula references
			Display:   cell.Display,
			Type:      cell.Type,
			Format:    cell.Format,
			UpdatedAt: now,
		}
		newCells = append(newCells, newCell)
	}

	return s.store.BatchSet(ctx, newCells)
}

// InsertRows inserts rows at the specified index.
func (s *Service) InsertRows(ctx context.Context, sheetID string, rowIndex, count int) error {
	return s.store.ShiftRows(ctx, sheetID, rowIndex, count)
}

// InsertCols inserts columns at the specified index.
func (s *Service) InsertCols(ctx context.Context, sheetID string, colIndex, count int) error {
	return s.store.ShiftCols(ctx, sheetID, colIndex, count)
}

// DeleteRows deletes rows starting at the specified index.
// Optimized to use single batch operation instead of N sequential operations.
func (s *Service) DeleteRows(ctx context.Context, sheetID string, startRow, count int) error {
	return s.store.DeleteRowsRange(ctx, sheetID, startRow, count)
}

// DeleteCols deletes columns starting at the specified index.
// Optimized to use single batch operation instead of N sequential operations.
func (s *Service) DeleteCols(ctx context.Context, sheetID string, startCol, count int) error {
	return s.store.DeleteColsRange(ctx, sheetID, startCol, count)
}

// EvaluateFormula evaluates a formula and returns the result.
func (s *Service) EvaluateFormula(ctx context.Context, sheetID, formulaStr string) (interface{}, error) {
	value, _, err := s.evaluateFormula(ctx, sheetID, formulaStr, 0, 0)
	return value, err
}

// evaluateFormula evaluates a formula string and returns value, display string, and error.
func (s *Service) evaluateFormula(ctx context.Context, sheetID, formulaStr string, row, col int) (interface{}, string, error) {
	// Skip leading = if present
	if strings.HasPrefix(formulaStr, "=") {
		formulaStr = formulaStr[1:]
	}

	// Parse formula
	lexer := formula.NewLexer(formulaStr)
	tokens, err := lexer.Tokenize()
	if err != nil {
		return nil, "", err
	}

	parser := formula.NewParser(tokens)
	ast, err := parser.Parse()
	if err != nil {
		return nil, "", err
	}

	// Create cell getter adapter
	cellGetter := &cellGetterAdapter{store: s.store}

	// Create evaluator context
	evalCtx := &formula.EvalContext{
		SheetID:    sheetID,
		CurrentRow: row,
		CurrentCol: col,
		CellGetter: cellGetter,
		Circular:   make(map[string]bool),
	}

	// Evaluate
	evaluator := formula.NewEvaluator(evalCtx)
	result, err := evaluator.Evaluate(ctx, ast)
	if err != nil {
		return nil, "", err
	}

	// Format display
	display := formatDisplay(result, Format{})

	return result, display, nil
}

// cellGetterAdapter adapts Store to formula.CellGetter interface.
type cellGetterAdapter struct {
	store Store
}

func (a *cellGetterAdapter) GetCellValue(ctx context.Context, sheetID string, row, col int) (interface{}, error) {
	cell, err := a.store.Get(ctx, sheetID, row, col)
	if err == ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return cell.Value, nil
}

func (a *cellGetterAdapter) GetRangeValues(ctx context.Context, sheetID string, startRow, startCol, endRow, endCol int) ([][]interface{}, error) {
	cells, err := a.store.GetRange(ctx, sheetID, startRow, startCol, endRow, endCol)
	if err != nil {
		return nil, err
	}

	// Create 2D array
	rows := endRow - startRow + 1
	cols := endCol - startCol + 1
	result := make([][]interface{}, rows)
	for i := range result {
		result[i] = make([]interface{}, cols)
	}

	// Fill in values
	for _, cell := range cells {
		r := cell.Row - startRow
		c := cell.Col - startCol
		if r >= 0 && r < rows && c >= 0 && c < cols {
			result[r][c] = cell.Value
		}
	}

	return result, nil
}

func (a *cellGetterAdapter) GetNamedRange(ctx context.Context, name string) (sheetID string, startRow, startCol, endRow, endCol int, err error) {
	// Named ranges not implemented yet
	return "", 0, 0, 0, 0, fmt.Errorf("named range not found: %s", name)
}

// detectCellType detects the type of a cell value.
func detectCellType(value interface{}) CellType {
	if value == nil {
		return CellTypeText
	}

	switch v := value.(type) {
	case bool:
		return CellTypeBool
	case int, int32, int64, float32, float64:
		return CellTypeNumber
	case string:
		if _, err := strconv.ParseFloat(v, 64); err == nil {
			return CellTypeNumber
		}
		return CellTypeText
	default:
		return CellTypeText
	}
}

// formatDisplay formats the display value based on format settings.
func formatDisplay(value interface{}, format Format) string {
	if value == nil {
		return ""
	}

	if format.NumberFormat != "" {
		return formula.FormatValue(value, format.NumberFormat)
	}

	return fmt.Sprintf("%v", value)
}

// mergeFormats merges two formats, with the second taking precedence.
func mergeFormats(base, override Format) Format {
	result := base

	if override.FontFamily != "" {
		result.FontFamily = override.FontFamily
	}
	if override.FontSize > 0 {
		result.FontSize = override.FontSize
	}
	if override.FontColor != "" {
		result.FontColor = override.FontColor
	}
	if override.Bold {
		result.Bold = true
	}
	if override.Italic {
		result.Italic = true
	}
	if override.Underline {
		result.Underline = true
	}
	if override.Strikethrough {
		result.Strikethrough = true
	}
	if override.BackgroundColor != "" {
		result.BackgroundColor = override.BackgroundColor
	}
	if override.HAlign != "" {
		result.HAlign = override.HAlign
	}
	if override.VAlign != "" {
		result.VAlign = override.VAlign
	}
	if override.WrapText {
		result.WrapText = true
	}
	if override.TextRotation != 0 {
		result.TextRotation = override.TextRotation
	}
	if override.Indent > 0 {
		result.Indent = override.Indent
	}
	if override.BorderTop.Style != "" {
		result.BorderTop = override.BorderTop
	}
	if override.BorderRight.Style != "" {
		result.BorderRight = override.BorderRight
	}
	if override.BorderBottom.Style != "" {
		result.BorderBottom = override.BorderBottom
	}
	if override.BorderLeft.Style != "" {
		result.BorderLeft = override.BorderLeft
	}
	if override.NumberFormat != "" {
		result.NumberFormat = override.NumberFormat
	}

	return result
}
