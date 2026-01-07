package export

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-mizu/blueprints/spreadsheet/feature/cells"
	"github.com/go-mizu/blueprints/spreadsheet/feature/sheets"
	"github.com/go-mizu/blueprints/spreadsheet/feature/workbooks"
)

// Service implements the export API.
type Service struct {
	workbooks workbooks.API
	sheets    sheets.API
	cells     cells.API
}

// NewService creates a new export service.
func NewService(wb workbooks.API, sh sheets.API, ce cells.API) *Service {
	return &Service{
		workbooks: wb,
		sheets:    sh,
		cells:     ce,
	}
}

// SupportedFormats returns supported export formats.
func (s *Service) SupportedFormats() []Format {
	return []Format{FormatCSV, FormatTSV, FormatXLSX, FormatJSON, FormatPDF, FormatHTML}
}

// ExportWorkbook exports an entire workbook.
func (s *Service) ExportWorkbook(ctx context.Context, workbookID string, format Format, opts *Options) (*Result, error) {
	if opts == nil {
		opts = &Options{}
	}

	// Load workbook
	wb, err := s.workbooks.GetByID(ctx, workbookID)
	if err != nil {
		return nil, fmt.Errorf("get workbook: %w", err)
	}

	// Load sheets
	sheetList, err := s.sheets.List(ctx, workbookID)
	if err != nil {
		return nil, fmt.Errorf("list sheets: %w", err)
	}

	// Build workbook data
	wbData := &WorkbookData{
		ID:   wb.ID,
		Name: wb.Name,
		Settings: WorkbookSettings{
			Locale:          wb.Settings.Locale,
			TimeZone:        wb.Settings.TimeZone,
			CalculationMode: wb.Settings.CalculationMode,
		},
		Sheets: make([]SheetData, 0, len(sheetList)),
	}

	for _, sheet := range sheetList {
		sheetData, err := s.loadSheetData(ctx, sheet)
		if err != nil {
			return nil, fmt.Errorf("load sheet %s: %w", sheet.ID, err)
		}
		wbData.Sheets = append(wbData.Sheets, *sheetData)
	}

	// Export based on format
	switch format {
	case FormatCSV:
		return s.exportWorkbookCSV(wbData, opts)
	case FormatTSV:
		opts.Delimiter = "\t"
		return s.exportWorkbookCSV(wbData, opts)
	case FormatXLSX:
		return s.exportWorkbookXLSX(wbData, opts)
	case FormatJSON:
		return s.exportWorkbookJSON(wbData, opts)
	case FormatPDF:
		return s.exportWorkbookPDF(wbData, opts)
	case FormatHTML:
		return s.exportWorkbookHTML(wbData, opts)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// ExportSheet exports a single sheet.
func (s *Service) ExportSheet(ctx context.Context, sheetID string, format Format, opts *Options) (*Result, error) {
	if opts == nil {
		opts = &Options{}
	}

	// Load sheet
	sheet, err := s.sheets.GetByID(ctx, sheetID)
	if err != nil {
		return nil, fmt.Errorf("get sheet: %w", err)
	}

	// Load sheet data
	sheetData, err := s.loadSheetData(ctx, sheet)
	if err != nil {
		return nil, fmt.Errorf("load sheet data: %w", err)
	}

	// Export based on format
	switch format {
	case FormatCSV:
		return s.exportSheetCSV(sheetData, opts)
	case FormatTSV:
		opts.Delimiter = "\t"
		return s.exportSheetCSV(sheetData, opts)
	case FormatXLSX:
		return s.exportSheetXLSX(sheetData, opts)
	case FormatJSON:
		return s.exportSheetJSON(sheetData, opts)
	case FormatPDF:
		return s.exportSheetPDF(sheetData, opts)
	case FormatHTML:
		return s.exportSheetHTML(sheetData, opts)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// loadSheetData loads all data for a sheet.
func (s *Service) loadSheetData(ctx context.Context, sheet *sheets.Sheet) (*SheetData, error) {
	// Load cells (get a large range to capture all data)
	cellList, err := s.cells.GetRange(ctx, sheet.ID, 0, 0, 10000, 256)
	if err != nil {
		return nil, fmt.Errorf("get cells: %w", err)
	}

	// Load merged regions
	merges, err := s.cells.GetMergedRegions(ctx, sheet.ID)
	if err != nil {
		return nil, fmt.Errorf("get merged regions: %w", err)
	}

	// Find bounds
	maxRow, maxCol := 0, 0
	cellDataList := make([]CellData, 0, len(cellList))
	for _, cell := range cellList {
		if cell.Row > maxRow {
			maxRow = cell.Row
		}
		if cell.Col > maxCol {
			maxCol = cell.Col
		}

		cd := CellData{
			Row:     cell.Row,
			Col:     cell.Col,
			Value:   cell.Value,
			Formula: cell.Formula,
			Display: cell.Display,
			Type:    string(cell.Type),
			Note:    cell.Note,
		}

		if cell.Hyperlink != nil {
			cd.Hyperlink = cell.Hyperlink.URL
		}

		// Convert format
		cd.Format = &CellFormat{
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
			BorderTop:       Border{Style: cell.Format.BorderTop.Style, Color: cell.Format.BorderTop.Color},
			BorderRight:     Border{Style: cell.Format.BorderRight.Style, Color: cell.Format.BorderRight.Color},
			BorderBottom:    Border{Style: cell.Format.BorderBottom.Style, Color: cell.Format.BorderBottom.Color},
			BorderLeft:      Border{Style: cell.Format.BorderLeft.Style, Color: cell.Format.BorderLeft.Color},
		}

		cellDataList = append(cellDataList, cd)
	}

	// Convert merged regions
	mergedRegions := make([]MergedRegion, len(merges))
	for i, m := range merges {
		mergedRegions[i] = MergedRegion{
			StartRow: m.StartRow,
			StartCol: m.StartCol,
			EndRow:   m.EndRow,
			EndCol:   m.EndCol,
		}
	}

	return &SheetData{
		ID:            sheet.ID,
		Name:          sheet.Name,
		Cells:         cellDataList,
		MergedRegions: mergedRegions,
		ColWidths:     sheet.ColWidths,
		RowHeights:    sheet.RowHeights,
		MaxRow:        maxRow,
		MaxCol:        maxCol,
	}, nil
}

// exportWorkbookCSV exports workbook as CSV or TSV (first sheet only).
func (s *Service) exportWorkbookCSV(wb *WorkbookData, opts *Options) (*Result, error) {
	if len(wb.Sheets) == 0 {
		return nil, fmt.Errorf("no sheets to export")
	}
	result, err := s.exportSheetCSV(&wb.Sheets[0], opts)
	if err != nil {
		return nil, err
	}
	// Set extension based on delimiter
	ext := ".csv"
	if opts.Delimiter == "\t" {
		ext = ".tsv"
	}
	result.Filename = sanitizeFilename(wb.Name) + ext
	return result, nil
}

// exportSheetCSV exports a sheet as CSV.
func (s *Service) exportSheetCSV(sheet *SheetData, opts *Options) (*Result, error) {
	delimiter := ','
	if opts.Delimiter != "" {
		delimiter = rune(opts.Delimiter[0])
	}

	// Build 2D grid
	grid := make([][]string, sheet.MaxRow+1)
	for i := range grid {
		grid[i] = make([]string, sheet.MaxCol+1)
	}

	// Fill grid with cell data
	for _, cell := range sheet.Cells {
		if cell.Row <= sheet.MaxRow && cell.Col <= sheet.MaxCol {
			if opts.ExportFormulas && cell.Formula != "" {
				grid[cell.Row][cell.Col] = cell.Formula
			} else if cell.Display != "" {
				grid[cell.Row][cell.Col] = cell.Display
			} else {
				grid[cell.Row][cell.Col] = fmt.Sprintf("%v", cell.Value)
			}
		}
	}

	// Write CSV
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	writer.Comma = delimiter

	// Add column headers if requested
	if opts.IncludeHeaders {
		headers := make([]string, sheet.MaxCol+1)
		for i := 0; i <= sheet.MaxCol; i++ {
			headers[i] = getColumnLabel(i)
		}
		writer.Write(headers)
	}

	for _, row := range grid {
		if opts.QuoteAll {
			// Quote all fields
			quotedRow := make([]string, len(row))
			for i, v := range row {
				quotedRow[i] = v
			}
			writer.Write(quotedRow)
		} else {
			writer.Write(row)
		}
	}
	writer.Flush()

	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("csv write: %w", err)
	}

	contentType := "text/csv; charset=utf-8"
	ext := ".csv"
	if delimiter == '\t' {
		contentType = "text/tab-separated-values; charset=utf-8"
		ext = ".tsv"
	}

	return &Result{
		ContentType: contentType,
		Filename:    sanitizeFilename(sheet.Name) + ext,
		Data:        bytes.NewReader(buf.Bytes()),
		Size:        int64(buf.Len()),
	}, nil
}

// exportWorkbookXLSX exports workbook as XLSX.
func (s *Service) exportWorkbookXLSX(wb *WorkbookData, opts *Options) (*Result, error) {
	data, err := generateXLSX(wb, opts)
	if err != nil {
		return nil, err
	}

	return &Result{
		ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		Filename:    sanitizeFilename(wb.Name) + ".xlsx",
		Data:        bytes.NewReader(data),
		Size:        int64(len(data)),
	}, nil
}

// exportSheetXLSX exports a single sheet as XLSX.
func (s *Service) exportSheetXLSX(sheet *SheetData, opts *Options) (*Result, error) {
	wb := &WorkbookData{
		Name:   sheet.Name,
		Sheets: []SheetData{*sheet},
	}
	return s.exportWorkbookXLSX(wb, opts)
}

// exportWorkbookJSON exports workbook as JSON.
func (s *Service) exportWorkbookJSON(wb *WorkbookData, opts *Options) (*Result, error) {
	data, err := generateJSON(wb, opts)
	if err != nil {
		return nil, err
	}

	return &Result{
		ContentType: "application/json; charset=utf-8",
		Filename:    sanitizeFilename(wb.Name) + ".json",
		Data:        bytes.NewReader(data),
		Size:        int64(len(data)),
	}, nil
}

// exportSheetJSON exports a single sheet as JSON.
func (s *Service) exportSheetJSON(sheet *SheetData, opts *Options) (*Result, error) {
	wb := &WorkbookData{
		Name:   sheet.Name,
		Sheets: []SheetData{*sheet},
	}
	return s.exportWorkbookJSON(wb, opts)
}

// exportWorkbookPDF exports workbook as PDF.
func (s *Service) exportWorkbookPDF(wb *WorkbookData, opts *Options) (*Result, error) {
	data, err := generatePDF(wb, opts)
	if err != nil {
		return nil, err
	}

	return &Result{
		ContentType: "application/pdf",
		Filename:    sanitizeFilename(wb.Name) + ".pdf",
		Data:        bytes.NewReader(data),
		Size:        int64(len(data)),
	}, nil
}

// exportSheetPDF exports a single sheet as PDF.
func (s *Service) exportSheetPDF(sheet *SheetData, opts *Options) (*Result, error) {
	wb := &WorkbookData{
		Name:   sheet.Name,
		Sheets: []SheetData{*sheet},
	}
	return s.exportWorkbookPDF(wb, opts)
}

// exportWorkbookHTML exports workbook as HTML.
func (s *Service) exportWorkbookHTML(wb *WorkbookData, opts *Options) (*Result, error) {
	data := generateHTML(wb, opts)

	return &Result{
		ContentType: "text/html; charset=utf-8",
		Filename:    sanitizeFilename(wb.Name) + ".html",
		Data:        bytes.NewReader(data),
		Size:        int64(len(data)),
	}, nil
}

// exportSheetHTML exports a single sheet as HTML.
func (s *Service) exportSheetHTML(sheet *SheetData, opts *Options) (*Result, error) {
	wb := &WorkbookData{
		Name:   sheet.Name,
		Sheets: []SheetData{*sheet},
	}
	return s.exportWorkbookHTML(wb, opts)
}

// getColumnLabel returns A, B, C, ..., Z, AA, AB, etc.
func getColumnLabel(index int) string {
	result := ""
	index++
	for index > 0 {
		index--
		result = string(rune('A'+(index%26))) + result
		index = index / 26
	}
	return result
}

// sanitizeFilename removes/replaces invalid filename characters.
func sanitizeFilename(name string) string {
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	return replacer.Replace(name)
}

// generateJSON generates JSON export.
func generateJSON(wb *WorkbookData, opts *Options) ([]byte, error) {
	type jsonSheet struct {
		ID            string        `json:"id"`
		Name          string        `json:"name"`
		Cells         []CellData    `json:"cells"`
		MergedRegions []MergedRegion `json:"mergedRegions,omitempty"`
		ColWidths     map[int]int   `json:"colWidths,omitempty"`
		RowHeights    map[int]int   `json:"rowHeights,omitempty"`
	}

	type jsonWorkbook struct {
		Version  string           `json:"version"`
		Workbook *WorkbookData    `json:"workbook,omitempty"`
		Sheets   []jsonSheet      `json:"sheets"`
	}

	output := jsonWorkbook{
		Version: "1.0",
		Sheets:  make([]jsonSheet, len(wb.Sheets)),
	}

	if opts.IncludeMetadata {
		output.Workbook = wb
	}

	for i, sheet := range wb.Sheets {
		output.Sheets[i] = jsonSheet{
			ID:            sheet.ID,
			Name:          sheet.Name,
			Cells:         sheet.Cells,
			MergedRegions: sheet.MergedRegions,
			ColWidths:     sheet.ColWidths,
			RowHeights:    sheet.RowHeights,
		}
	}

	if opts.Compact {
		return json.Marshal(output)
	}
	return json.MarshalIndent(output, "", "  ")
}

// generateHTML generates HTML table export.
func generateHTML(wb *WorkbookData, opts *Options) []byte {
	var buf bytes.Buffer

	buf.WriteString(`<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<title>` + wb.Name + `</title>
<style>
body { font-family: Arial, sans-serif; margin: 20px; }
table { border-collapse: collapse; margin-bottom: 30px; }
th, td { border: 1px solid #ccc; padding: 8px; text-align: left; min-width: 80px; }
th { background-color: #f5f5f5; font-weight: bold; }
.sheet-name { font-size: 18px; font-weight: bold; margin-bottom: 10px; }
.bold { font-weight: bold; }
.italic { font-style: italic; }
.underline { text-decoration: underline; }
.strikethrough { text-decoration: line-through; }
.align-left { text-align: left; }
.align-center { text-align: center; }
.align-right { text-align: right; }
</style>
</head>
<body>
<h1>` + wb.Name + `</h1>
`)

	for _, sheet := range wb.Sheets {
		buf.WriteString(`<div class="sheet">`)
		buf.WriteString(`<div class="sheet-name">` + sheet.Name + `</div>`)
		buf.WriteString(`<table>`)

		// Add header row
		if opts.IncludeHeaders {
			buf.WriteString(`<tr><th></th>`)
			for col := 0; col <= sheet.MaxCol; col++ {
				buf.WriteString(`<th>` + getColumnLabel(col) + `</th>`)
			}
			buf.WriteString(`</tr>`)
		}

		// Build cell map for quick lookup
		cellMap := make(map[string]CellData)
		for _, cell := range sheet.Cells {
			key := fmt.Sprintf("%d:%d", cell.Row, cell.Col)
			cellMap[key] = cell
		}

		// Generate rows
		for row := 0; row <= sheet.MaxRow; row++ {
			buf.WriteString(`<tr>`)
			if opts.IncludeHeaders {
				buf.WriteString(fmt.Sprintf(`<th>%d</th>`, row+1))
			}
			for col := 0; col <= sheet.MaxCol; col++ {
				key := fmt.Sprintf("%d:%d", row, col)
				cell, exists := cellMap[key]

				style := ""
				classes := ""

				if exists && cell.Format != nil {
					if cell.Format.Bold {
						classes += " bold"
					}
					if cell.Format.Italic {
						classes += " italic"
					}
					if cell.Format.Underline {
						classes += " underline"
					}
					if cell.Format.Strikethrough {
						classes += " strikethrough"
					}
					if cell.Format.HAlign != "" {
						classes += " align-" + cell.Format.HAlign
					}
					if cell.Format.BackgroundColor != "" {
						style += "background-color:" + cell.Format.BackgroundColor + ";"
					}
					if cell.Format.FontColor != "" {
						style += "color:" + cell.Format.FontColor + ";"
					}
				}

				value := ""
				if exists {
					if opts.ExportFormulas && cell.Formula != "" {
						value = cell.Formula
					} else if cell.Display != "" {
						value = cell.Display
					} else {
						value = fmt.Sprintf("%v", cell.Value)
					}
				}

				tdAttrs := ""
				if classes != "" {
					tdAttrs += ` class="` + strings.TrimSpace(classes) + `"`
				}
				if style != "" {
					tdAttrs += ` style="` + style + `"`
				}

				buf.WriteString(`<td` + tdAttrs + `>` + htmlEscape(value) + `</td>`)
			}
			buf.WriteString(`</tr>`)
		}

		buf.WriteString(`</table>`)
		buf.WriteString(`</div>`)
	}

	buf.WriteString(`</body></html>`)
	return buf.Bytes()
}

func htmlEscape(s string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
		"'", "&#39;",
	)
	return replacer.Replace(s)
}

// Ensure Service implements API
var _ API = (*Service)(nil)
