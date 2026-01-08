package importer

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/spreadsheet/feature/cells"
	"github.com/go-mizu/blueprints/spreadsheet/feature/sheets"
	"github.com/go-mizu/blueprints/spreadsheet/feature/workbooks"
)

// Service implements the import API.
type Service struct {
	workbooks workbooks.API
	sheets    sheets.API
	cells     cells.API
}

// NewService creates a new import service.
func NewService(wb workbooks.API, sh sheets.API, ce cells.API) *Service {
	return &Service{
		workbooks: wb,
		sheets:    sh,
		cells:     ce,
	}
}

// SupportedFormats returns supported import formats.
func (s *Service) SupportedFormats() []Format {
	return []Format{FormatCSV, FormatTSV, FormatXLSX, FormatJSON}
}

// DetectFormat detects the format from filename.
func (s *Service) DetectFormat(filename string) Format {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".csv":
		return FormatCSV
	case ".tsv", ".tab":
		return FormatTSV
	case ".xlsx", ".xlsm":
		return FormatXLSX
	case ".json":
		return FormatJSON
	default:
		return FormatCSV // Default to CSV
	}
}

// ValidateFile validates a file before import.
func (s *Service) ValidateFile(ctx context.Context, reader io.Reader, format Format) error {
	// Read first few bytes to validate
	buf := make([]byte, 512)
	n, err := reader.Read(buf)
	if err != nil && err != io.EOF {
		return fmt.Errorf("read file: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("file is empty")
	}

	switch format {
	case FormatCSV, FormatTSV:
		// CSV/TSV should be text
		for _, b := range buf[:n] {
			if b == 0 {
				return fmt.Errorf("file appears to be binary, not CSV/TSV")
			}
		}
	case FormatXLSX:
		// XLSX starts with PK (zip file)
		if n < 2 || buf[0] != 'P' || buf[1] != 'K' {
			return fmt.Errorf("file is not a valid XLSX file")
		}
	case FormatJSON:
		// JSON should start with { or [
		for i := 0; i < n; i++ {
			if buf[i] == ' ' || buf[i] == '\n' || buf[i] == '\r' || buf[i] == '\t' {
				continue
			}
			if buf[i] != '{' && buf[i] != '[' {
				return fmt.Errorf("file is not valid JSON")
			}
			break
		}
	}

	return nil
}

// ImportToWorkbook imports data to a workbook, creating a new sheet.
func (s *Service) ImportToWorkbook(ctx context.Context, workbookID string, reader io.Reader, filename string, format Format, opts *Options) (*Result, error) {
	if opts == nil {
		opts = &Options{}
	}

	// Detect format if not specified
	if format == "" {
		format = s.DetectFormat(filename)
	}

	// Parse the file
	sheetImports, err := s.parseFile(reader, format, opts)
	if err != nil {
		return nil, fmt.Errorf("parse file: %w", err)
	}

	if len(sheetImports) == 0 {
		return nil, fmt.Errorf("no data to import")
	}

	// Get sheet list to determine new sheet index
	existingSheets, err := s.sheets.List(ctx, workbookID)
	if err != nil {
		return nil, fmt.Errorf("list sheets: %w", err)
	}

	// Create new sheet for each imported sheet (or just the first if single sheet format)
	var result *Result
	for i, sheetImport := range sheetImports {
		sheetName := sheetImport.Name
		if sheetName == "" {
			if opts.SheetName != "" {
				sheetName = opts.SheetName
			} else {
				sheetName = strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
			}
			if i > 0 {
				sheetName = fmt.Sprintf("%s (%d)", sheetName, i+1)
			}
		}

		// Create new sheet
		sheet, err := s.sheets.Create(ctx, &sheets.CreateIn{
			WorkbookID: workbookID,
			Name:       sheetName,
			Index:      len(existingSheets) + i,
		})
		if err != nil {
			return nil, fmt.Errorf("create sheet: %w", err)
		}

		// Import cells
		sheetResult, err := s.importCellsToSheet(ctx, sheet.ID, sheetImport, opts)
		if err != nil {
			return nil, fmt.Errorf("import cells: %w", err)
		}

		if i == 0 {
			result = sheetResult
		} else {
			// Aggregate results
			result.RowsImported += sheetResult.RowsImported
			result.ColsImported = max(result.ColsImported, sheetResult.ColsImported)
			result.CellsImported += sheetResult.CellsImported
			result.Warnings = append(result.Warnings, sheetResult.Warnings...)
		}
	}

	return result, nil
}

// ImportToSheet imports data to an existing sheet.
func (s *Service) ImportToSheet(ctx context.Context, sheetID string, reader io.Reader, filename string, format Format, opts *Options) (*Result, error) {
	if opts == nil {
		opts = &Options{}
	}

	// Detect format if not specified
	if format == "" {
		format = s.DetectFormat(filename)
	}

	// Parse the file
	sheetImports, err := s.parseFile(reader, format, opts)
	if err != nil {
		return nil, fmt.Errorf("parse file: %w", err)
	}

	if len(sheetImports) == 0 {
		return nil, fmt.Errorf("no data to import")
	}

	// Import first sheet only when importing to existing sheet
	return s.importCellsToSheet(ctx, sheetID, sheetImports[0], opts)
}

// parseFile parses the input file based on format.
func (s *Service) parseFile(reader io.Reader, format Format, opts *Options) ([]SheetImport, error) {
	switch format {
	case FormatCSV:
		return s.parseCSV(reader, ',', opts)
	case FormatTSV:
		return s.parseCSV(reader, '\t', opts)
	case FormatXLSX:
		return s.parseXLSX(reader, opts)
	case FormatJSON:
		return s.parseJSON(reader, opts)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// parseCSV parses CSV/TSV data.
func (s *Service) parseCSV(reader io.Reader, delimiter rune, opts *Options) ([]SheetImport, error) {
	csvReader := csv.NewReader(reader)
	csvReader.Comma = delimiter
	csvReader.LazyQuotes = true
	csvReader.TrimLeadingSpace = opts.TrimWhitespace
	csvReader.FieldsPerRecord = -1 // Allow variable number of fields

	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse csv: %w", err)
	}

	var cellImports []CellImport
	startRow := 0
	if opts.HasHeaders {
		startRow = 1
	}

	for rowIdx, record := range records[startRow:] {
		row := rowIdx + opts.StartRow

		if opts.SkipEmptyRows {
			empty := true
			for _, cell := range record {
				if strings.TrimSpace(cell) != "" {
					empty = false
					break
				}
			}
			if empty {
				continue
			}
		}

		for colIdx, value := range record {
			col := colIdx + opts.StartCol

			if opts.TrimWhitespace {
				value = strings.TrimSpace(value)
			}

			if value == "" {
				continue
			}

			cellImport := CellImport{
				Row:   row,
				Col:   col,
				Value: value,
			}

			// Auto-detect types
			if opts.AutoDetectTypes {
				cellImport.Value = s.detectType(value, opts.DateFormat)
			}

			cellImports = append(cellImports, cellImport)
		}
	}

	// Return empty slice if no cells were imported (empty file)
	if len(cellImports) == 0 {
		return nil, nil
	}

	return []SheetImport{{Cells: cellImports}}, nil
}

// parseJSON parses JSON data.
func (s *Service) parseJSON(reader io.Reader, opts *Options) ([]SheetImport, error) {
	var data struct {
		Version string `json:"version"`
		Sheets  []struct {
			ID            string `json:"id"`
			Name          string `json:"name"`
			Cells         []struct {
				Row     int         `json:"row"`
				Col     int         `json:"col"`
				Value   interface{} `json:"value"`
				Formula string      `json:"formula"`
				Format  *struct {
					FontFamily      string `json:"fontFamily"`
					FontSize        int    `json:"fontSize"`
					FontColor       string `json:"fontColor"`
					Bold            bool   `json:"bold"`
					Italic          bool   `json:"italic"`
					Underline       bool   `json:"underline"`
					Strikethrough   bool   `json:"strikethrough"`
					BackgroundColor string `json:"backgroundColor"`
					HAlign          string `json:"hAlign"`
					VAlign          string `json:"vAlign"`
					WrapText        bool   `json:"wrapText"`
					NumberFormat    string `json:"numberFormat"`
				} `json:"format"`
			} `json:"cells"`
			MergedRegions []struct {
				StartRow int `json:"startRow"`
				StartCol int `json:"startCol"`
				EndRow   int `json:"endRow"`
				EndCol   int `json:"endCol"`
			} `json:"mergedRegions"`
		} `json:"sheets"`
	}

	if err := json.NewDecoder(reader).Decode(&data); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}

	var sheetImports []SheetImport

	for _, sheet := range data.Sheets {
		var cellImports []CellImport
		for _, cell := range sheet.Cells {
			ci := CellImport{
				Row:   cell.Row + opts.StartRow,
				Col:   cell.Col + opts.StartCol,
				Value: cell.Value,
			}
			if opts.ImportFormulas {
				ci.Formula = cell.Formula
			}
			if opts.ImportFormatting && cell.Format != nil {
				ci.Format = &CellFormat{
					FontFamily:      cell.Format.FontFamily,
					FontSize:        cell.Format.FontSize,
					FontColor:       cell.Format.FontColor,
					Bold:            cell.Format.Bold,
					Italic:          cell.Format.Italic,
					Underline:       cell.Format.Underline,
					Strikethrough:   cell.Format.Strikethrough,
					BackgroundColor: cell.Format.BackgroundColor,
					HAlign:          cell.Format.HAlign,
					VAlign:          cell.Format.VAlign,
					WrapText:        cell.Format.WrapText,
					NumberFormat:    cell.Format.NumberFormat,
				}
			}
			cellImports = append(cellImports, ci)
		}

		var mergedRegions []MergedRegionImport
		for _, mr := range sheet.MergedRegions {
			mergedRegions = append(mergedRegions, MergedRegionImport{
				StartRow: mr.StartRow + opts.StartRow,
				StartCol: mr.StartCol + opts.StartCol,
				EndRow:   mr.EndRow + opts.StartRow,
				EndCol:   mr.EndCol + opts.StartCol,
			})
		}

		sheetImports = append(sheetImports, SheetImport{
			Name:          sheet.Name,
			Cells:         cellImports,
			MergedRegions: mergedRegions,
		})
	}

	return sheetImports, nil
}

// importCellsToSheet imports cells to a sheet.
func (s *Service) importCellsToSheet(ctx context.Context, sheetID string, sheetImport SheetImport, opts *Options) (*Result, error) {
	result := &Result{
		SheetID: sheetID,
	}

	// Prepare cells for import
	var cellsToImport []*cells.Cell
	maxRow, maxCol := 0, 0

	for _, ci := range sheetImport.Cells {
		if ci.Row > maxRow {
			maxRow = ci.Row
		}
		if ci.Col > maxCol {
			maxCol = ci.Col
		}

		cell := &cells.Cell{
			SheetID: sheetID,
			Row:     ci.Row,
			Col:     ci.Col,
			Value:   ci.Value,
			Formula: ci.Formula,
		}

		// Set display value
		if ci.Formula != "" {
			cell.Type = cells.CellTypeFormula
		} else {
			cell.Display = fmt.Sprintf("%v", ci.Value)
			cell.Type = detectCellType(ci.Value)
		}

		// Apply formatting
		if ci.Format != nil {
			cell.Format = cells.Format{
				FontFamily:      ci.Format.FontFamily,
				FontSize:        ci.Format.FontSize,
				FontColor:       ci.Format.FontColor,
				Bold:            ci.Format.Bold,
				Italic:          ci.Format.Italic,
				Underline:       ci.Format.Underline,
				Strikethrough:   ci.Format.Strikethrough,
				BackgroundColor: ci.Format.BackgroundColor,
				HAlign:          ci.Format.HAlign,
				VAlign:          ci.Format.VAlign,
				WrapText:        ci.Format.WrapText,
				NumberFormat:    ci.Format.NumberFormat,
			}
		}

		cellsToImport = append(cellsToImport, cell)
	}

	// Configure batch processor
	batchSize := 500
	workers := 4
	if opts.BatchSize > 0 {
		batchSize = opts.BatchSize
	}
	if opts.Workers > 0 {
		workers = opts.Workers
	}

	// Use parallel processing for large imports when enabled
	if opts.ParallelProcessing && len(cellsToImport) > batchSize*2 {
		processor := NewBatchProcessor(s.cells, batchSize, workers)
		imported, warnings := processor.ProcessCells(ctx, sheetID, cellsToImport)
		result.Warnings = append(result.Warnings, warnings...)
		_ = imported // Count is tracked separately
	} else {
		// Sequential batch processing for small imports or when parallel is disabled
		for i := 0; i < len(cellsToImport); i += batchSize {
			end := i + batchSize
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

			_, err := s.cells.BatchUpdate(ctx, sheetID, &cells.BatchUpdateIn{Cells: updates})
			if err != nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf("Batch import failed at row %d: %v", i/batchSize, err))
			}
		}
	}

	// Import merged regions using batch operation for better performance
	if len(sheetImport.MergedRegions) > 0 {
		regions := make([]cells.MergedRegion, len(sheetImport.MergedRegions))
		for i, mr := range sheetImport.MergedRegions {
			regions[i] = cells.MergedRegion{
				StartRow: mr.StartRow,
				StartCol: mr.StartCol,
				EndRow:   mr.EndRow,
				EndCol:   mr.EndCol,
			}
		}
		_, err := s.cells.BatchMerge(ctx, sheetID, regions)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Failed to batch merge regions: %v", err))
		}
	}

	result.RowsImported = maxRow + 1
	result.ColsImported = maxCol + 1
	result.CellsImported = len(cellsToImport)

	return result, nil
}

// detectType attempts to detect the type of a string value.
func (s *Service) detectType(value string, dateFormat string) interface{} {
	// Try boolean
	lower := strings.ToLower(value)
	if lower == "true" || lower == "yes" {
		return true
	}
	if lower == "false" || lower == "no" {
		return false
	}

	// Try integer
	if i, err := strconv.ParseInt(value, 10, 64); err == nil {
		return i
	}

	// Try float
	if f, err := strconv.ParseFloat(value, 64); err == nil {
		return f
	}

	// Try date
	dateFormats := []string{
		"2006-01-02",
		"01/02/2006",
		"02/01/2006",
		"2006/01/02",
		"Jan 2, 2006",
		"January 2, 2006",
	}
	if dateFormat != "" {
		dateFormats = append([]string{dateFormat}, dateFormats...)
	}
	for _, format := range dateFormats {
		if t, err := time.Parse(format, value); err == nil {
			return t
		}
	}

	// Return as string
	return value
}

// detectCellType detects the cell type from a value.
func detectCellType(value interface{}) cells.CellType {
	switch value.(type) {
	case bool:
		return cells.CellTypeBool
	case int, int64, float64:
		return cells.CellTypeNumber
	case time.Time:
		return cells.CellTypeDate
	default:
		return cells.CellTypeText
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Ensure Service implements API
var _ API = (*Service)(nil)
