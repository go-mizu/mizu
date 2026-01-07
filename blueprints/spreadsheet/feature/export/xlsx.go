package export

import (
	"bytes"
	"fmt"

	"github.com/xuri/excelize/v2"
)

// generateXLSX generates an XLSX file from workbook data.
func generateXLSX(wb *WorkbookData, opts *Options) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	// Remove default sheet
	f.DeleteSheet("Sheet1")

	for i, sheet := range wb.Sheets {
		sheetName := sheet.Name
		if sheetName == "" {
			sheetName = fmt.Sprintf("Sheet%d", i+1)
		}

		// Create sheet
		idx, err := f.NewSheet(sheetName)
		if err != nil {
			return nil, fmt.Errorf("create sheet %s: %w", sheetName, err)
		}

		if i == 0 {
			f.SetActiveSheet(idx)
		}

		// Build cell map for quick lookup
		cellMap := make(map[string]CellData)
		for _, cell := range sheet.Cells {
			key := fmt.Sprintf("%d:%d", cell.Row, cell.Col)
			cellMap[key] = cell
		}

		// Set cell values
		for _, cell := range sheet.Cells {
			cellRef, _ := excelize.CoordinatesToCellName(cell.Col+1, cell.Row+1)

			// Set value or formula
			if opts.ExportFormulas && cell.Formula != "" {
				f.SetCellFormula(sheetName, cellRef, cell.Formula)
			} else if cell.Value != nil {
				f.SetCellValue(sheetName, cellRef, cell.Value)
			} else if cell.Display != "" {
				f.SetCellValue(sheetName, cellRef, cell.Display)
			}

			// Apply formatting if requested
			if opts.ExportFormatting && cell.Format != nil {
				style := buildExcelStyle(cell.Format)
				if style != nil {
					styleID, err := f.NewStyle(style)
					if err == nil {
						f.SetCellStyle(sheetName, cellRef, cellRef, styleID)
					}
				}
			}

			// Add hyperlink
			if cell.Hyperlink != "" {
				f.SetCellHyperLink(sheetName, cellRef, cell.Hyperlink, "External")
			}

			// Add comment/note
			if cell.Note != "" {
				f.AddComment(sheetName, excelize.Comment{
					Cell:   cellRef,
					Author: "Spreadsheet",
					Text:   cell.Note,
				})
			}
		}

		// Set merged regions
		for _, merge := range sheet.MergedRegions {
			startRef, _ := excelize.CoordinatesToCellName(merge.StartCol+1, merge.StartRow+1)
			endRef, _ := excelize.CoordinatesToCellName(merge.EndCol+1, merge.EndRow+1)
			f.MergeCell(sheetName, startRef, endRef)
		}

		// Set column widths
		for col, width := range sheet.ColWidths {
			colName, _ := excelize.ColumnNumberToName(col + 1)
			f.SetColWidth(sheetName, colName, colName, float64(width)/7.0) // Approximate conversion
		}

		// Set row heights
		for row, height := range sheet.RowHeights {
			f.SetRowHeight(sheetName, row+1, float64(height))
		}
	}

	// Write to buffer
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("write xlsx: %w", err)
	}

	return buf.Bytes(), nil
}

// buildExcelStyle builds an excelize style from CellFormat.
func buildExcelStyle(format *CellFormat) *excelize.Style {
	if format == nil {
		return nil
	}

	style := &excelize.Style{}

	// Font
	if format.FontFamily != "" || format.FontSize > 0 || format.Bold || format.Italic ||
		format.Underline || format.Strikethrough || format.FontColor != "" {
		font := &excelize.Font{}
		if format.FontFamily != "" {
			font.Family = format.FontFamily
		}
		if format.FontSize > 0 {
			font.Size = float64(format.FontSize)
		}
		font.Bold = format.Bold
		font.Italic = format.Italic
		if format.Underline {
			font.Underline = "single"
		}
		font.Strike = format.Strikethrough
		if format.FontColor != "" {
			font.Color = format.FontColor
		}
		style.Font = font
	}

	// Fill (background color)
	if format.BackgroundColor != "" {
		style.Fill = excelize.Fill{
			Type:    "pattern",
			Pattern: 1,
			Color:   []string{format.BackgroundColor},
		}
	}

	// Alignment
	if format.HAlign != "" || format.VAlign != "" || format.WrapText {
		alignment := &excelize.Alignment{
			WrapText: format.WrapText,
		}
		switch format.HAlign {
		case "left":
			alignment.Horizontal = "left"
		case "center":
			alignment.Horizontal = "center"
		case "right":
			alignment.Horizontal = "right"
		}
		switch format.VAlign {
		case "top":
			alignment.Vertical = "top"
		case "middle":
			alignment.Vertical = "center"
		case "bottom":
			alignment.Vertical = "bottom"
		}
		style.Alignment = alignment
	}

	// Number format
	if format.NumberFormat != "" {
		style.NumFmt = getExcelNumFormat(format.NumberFormat)
	}

	// Borders
	var borders []excelize.Border
	if format.BorderTop.Style != "" && format.BorderTop.Style != "none" {
		borders = append(borders, excelize.Border{
			Type:  "top",
			Style: getBorderStyle(format.BorderTop.Style),
			Color: getColor(format.BorderTop.Color),
		})
	}
	if format.BorderRight.Style != "" && format.BorderRight.Style != "none" {
		borders = append(borders, excelize.Border{
			Type:  "right",
			Style: getBorderStyle(format.BorderRight.Style),
			Color: getColor(format.BorderRight.Color),
		})
	}
	if format.BorderBottom.Style != "" && format.BorderBottom.Style != "none" {
		borders = append(borders, excelize.Border{
			Type:  "bottom",
			Style: getBorderStyle(format.BorderBottom.Style),
			Color: getColor(format.BorderBottom.Color),
		})
	}
	if format.BorderLeft.Style != "" && format.BorderLeft.Style != "none" {
		borders = append(borders, excelize.Border{
			Type:  "left",
			Style: getBorderStyle(format.BorderLeft.Style),
			Color: getColor(format.BorderLeft.Color),
		})
	}
	if len(borders) > 0 {
		style.Border = borders
	}

	return style
}

// getBorderStyle converts border style string to excelize border style.
func getBorderStyle(style string) int {
	switch style {
	case "thin":
		return 1
	case "medium":
		return 2
	case "thick":
		return 5
	case "dashed":
		return 3
	case "dotted":
		return 4
	case "double":
		return 6
	default:
		return 1
	}
}

// getColor returns color or default black.
func getColor(color string) string {
	if color == "" {
		return "000000"
	}
	// Remove # prefix if present
	if len(color) > 0 && color[0] == '#' {
		return color[1:]
	}
	return color
}

// getExcelNumFormat converts number format string to Excel format code.
func getExcelNumFormat(format string) int {
	switch format {
	case "#,##0":
		return 3
	case "#,##0.00":
		return 4
	case "0%":
		return 9
	case "0.00%":
		return 10
	case "$#,##0":
		return 5
	case "$#,##0.00":
		return 7
	case "yyyy-mm-dd":
		return 14
	case "mm/dd/yyyy":
		return 14
	case "hh:mm:ss":
		return 21
	case "@": // Text
		return 49
	default:
		return 0 // General
	}
}
