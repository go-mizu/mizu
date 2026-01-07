package importer

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// parseXLSX parses XLSX data.
func (s *Service) parseXLSX(reader io.Reader, opts *Options) ([]SheetImport, error) {
	f, err := excelize.OpenReader(reader)
	if err != nil {
		return nil, fmt.Errorf("open xlsx: %w", err)
	}
	defer f.Close()

	var sheetImports []SheetImport
	sheetList := f.GetSheetList()

	// Filter sheets if specified
	if opts.ImportSheet != "" {
		found := false
		for _, name := range sheetList {
			if name == opts.ImportSheet {
				sheetList = []string{name}
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("sheet not found: %s", opts.ImportSheet)
		}
	}

	for _, sheetName := range sheetList {
		sheetImport, err := s.parseXLSXSheet(f, sheetName, opts)
		if err != nil {
			return nil, fmt.Errorf("parse sheet %s: %w", sheetName, err)
		}
		sheetImport.Name = sheetName
		sheetImports = append(sheetImports, *sheetImport)
	}

	return sheetImports, nil
}

// parseXLSXSheet parses a single XLSX sheet.
func (s *Service) parseXLSXSheet(f *excelize.File, sheetName string, opts *Options) (*SheetImport, error) {
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("get rows: %w", err)
	}

	var cellImports []CellImport
	startRow := 0
	if opts.HasHeaders {
		startRow = 1
	}

	for rowIdx, row := range rows {
		if rowIdx < startRow {
			continue
		}

		targetRow := rowIdx - startRow + opts.StartRow

		if opts.SkipEmptyRows {
			empty := true
			for _, cell := range row {
				if strings.TrimSpace(cell) != "" {
					empty = false
					break
				}
			}
			if empty {
				continue
			}
		}

		for colIdx, value := range row {
			targetCol := colIdx + opts.StartCol

			if opts.TrimWhitespace {
				value = strings.TrimSpace(value)
			}

			if value == "" {
				continue
			}

			cellRef, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+1)

			cellImport := CellImport{
				Row: targetRow,
				Col: targetCol,
			}

			// Get formula if requested
			if opts.ImportFormulas {
				formula, _ := f.GetCellFormula(sheetName, cellRef)
				if formula != "" {
					cellImport.Formula = "=" + formula
					cellImport.Value = value // Store calculated value as well
				} else {
					cellImport.Value = s.parseXLSXValue(value, opts)
				}
			} else {
				cellImport.Value = s.parseXLSXValue(value, opts)
			}

			// Get formatting if requested
			if opts.ImportFormatting {
				styleID, _ := f.GetCellStyle(sheetName, cellRef)
				if styleID > 0 {
					cellImport.Format = s.parseXLSXStyle(f, styleID)
				}
			}

			cellImports = append(cellImports, cellImport)
		}
	}

	// Get merged cells
	var mergedRegions []MergedRegionImport
	merges, err := f.GetMergeCells(sheetName)
	if err == nil {
		for _, merge := range merges {
			startCell := merge.GetStartAxis()
			endCell := merge.GetEndAxis()

			startCol, startRow, _ := excelize.CellNameToCoordinates(startCell)
			endCol, endRow, _ := excelize.CellNameToCoordinates(endCell)

			// Adjust for 0-based indexing and options
			mergedRegions = append(mergedRegions, MergedRegionImport{
				StartRow: startRow - 1 + opts.StartRow,
				StartCol: startCol - 1 + opts.StartCol,
				EndRow:   endRow - 1 + opts.StartRow,
				EndCol:   endCol - 1 + opts.StartCol,
			})
		}
	}

	return &SheetImport{
		Cells:         cellImports,
		MergedRegions: mergedRegions,
	}, nil
}

// parseXLSXValue parses a cell value with type detection.
func (s *Service) parseXLSXValue(value string, opts *Options) interface{} {
	if !opts.AutoDetectTypes {
		return value
	}
	return s.detectType(value, opts.DateFormat)
}

// parseXLSXStyle extracts formatting from an Excel style.
func (s *Service) parseXLSXStyle(f *excelize.File, styleID int) *CellFormat {
	style, err := f.GetStyle(styleID)
	if err != nil || style == nil {
		return nil
	}

	format := &CellFormat{}

	// Font
	if style.Font != nil {
		format.FontFamily = style.Font.Family
		format.FontSize = int(style.Font.Size)
		format.Bold = style.Font.Bold
		format.Italic = style.Font.Italic
		if style.Font.Underline != "" {
			format.Underline = true
		}
		format.Strikethrough = style.Font.Strike
		format.FontColor = style.Font.Color
	}

	// Fill (background)
	if len(style.Fill.Color) > 0 {
		format.BackgroundColor = style.Fill.Color[0]
	}

	// Alignment
	if style.Alignment != nil {
		switch style.Alignment.Horizontal {
		case "left":
			format.HAlign = "left"
		case "center":
			format.HAlign = "center"
		case "right":
			format.HAlign = "right"
		}
		switch style.Alignment.Vertical {
		case "top":
			format.VAlign = "top"
		case "center":
			format.VAlign = "middle"
		case "bottom":
			format.VAlign = "bottom"
		}
		format.WrapText = style.Alignment.WrapText
	}

	// Number format
	format.NumberFormat = getNumberFormatString(style.NumFmt)

	return format
}

// getNumberFormatString converts Excel number format ID to string.
func getNumberFormatString(numFmt int) string {
	switch numFmt {
	case 0:
		return ""
	case 1:
		return "0"
	case 2:
		return "0.00"
	case 3:
		return "#,##0"
	case 4:
		return "#,##0.00"
	case 5, 6, 7, 8:
		return "$#,##0.00"
	case 9:
		return "0%"
	case 10:
		return "0.00%"
	case 14, 15, 16, 17:
		return "yyyy-mm-dd"
	case 18, 19, 20, 21:
		return "hh:mm:ss"
	case 49:
		return "@"
	default:
		return ""
	}
}

// GetSheetNames returns sheet names from an XLSX file.
func GetSheetNames(reader io.Reader) ([]string, error) {
	f, err := excelize.OpenReader(reader)
	if err != nil {
		return nil, fmt.Errorf("open xlsx: %w", err)
	}
	defer f.Close()

	return f.GetSheetList(), nil
}

// GetXLSXPreview returns a preview of XLSX data.
func GetXLSXPreview(reader io.Reader, sheetName string, maxRows int) ([][]string, error) {
	f, err := excelize.OpenReader(reader)
	if err != nil {
		return nil, fmt.Errorf("open xlsx: %w", err)
	}
	defer f.Close()

	if sheetName == "" {
		sheets := f.GetSheetList()
		if len(sheets) == 0 {
			return nil, fmt.Errorf("no sheets in workbook")
		}
		sheetName = sheets[0]
	}

	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("get rows: %w", err)
	}

	if maxRows > 0 && len(rows) > maxRows {
		rows = rows[:maxRows]
	}

	return rows, nil
}

// parseExcelColumn converts Excel column letter to 0-based index.
func parseExcelColumn(col string) int {
	col = strings.ToUpper(col)
	result := 0
	for _, c := range col {
		result = result*26 + int(c-'A'+1)
	}
	return result - 1
}

// formatExcelColumn converts 0-based index to Excel column letter.
func formatExcelColumn(col int) string {
	result := ""
	col++
	for col > 0 {
		col--
		result = string(rune('A'+col%26)) + result
		col /= 26
	}
	return result
}

// parseExcelRef parses an Excel cell reference like "A1" or "B2".
func parseExcelRef(ref string) (row, col int, err error) {
	ref = strings.ToUpper(ref)

	// Find where numbers start
	numStart := 0
	for i, c := range ref {
		if c >= '0' && c <= '9' {
			numStart = i
			break
		}
	}

	if numStart == 0 {
		return 0, 0, fmt.Errorf("invalid cell reference: %s", ref)
	}

	colStr := ref[:numStart]
	rowStr := ref[numStart:]

	col = parseExcelColumn(colStr)
	row64, err := strconv.ParseInt(rowStr, 10, 32)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid row number: %s", rowStr)
	}
	row = int(row64) - 1 // Convert to 0-based

	return row, col, nil
}
