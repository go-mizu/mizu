package export

import (
	"bytes"
	"fmt"

	"github.com/jung-kurt/gofpdf"
)

// generatePDF generates a PDF file from workbook data.
func generatePDF(wb *WorkbookData, opts *Options) ([]byte, error) {
	// Determine orientation and paper size
	orientation := "P" // Portrait
	if opts.Orientation == "landscape" {
		orientation = "L"
	}

	paperSize := "Letter"
	switch opts.PaperSize {
	case "a4":
		paperSize = "A4"
	case "legal":
		paperSize = "Legal"
	}

	pdf := gofpdf.New(orientation, "mm", paperSize, "")
	pdf.SetMargins(10, 10, 10)

	for _, sheet := range wb.Sheets {
		pdf.AddPage()

		// Add sheet title
		pdf.SetFont("Arial", "B", 14)
		pdf.Cell(0, 10, sheet.Name)
		pdf.Ln(15)

		if len(sheet.Cells) == 0 {
			pdf.SetFont("Arial", "", 10)
			pdf.Cell(0, 10, "(Empty sheet)")
			continue
		}

		// Build cell map
		cellMap := make(map[string]CellData)
		for _, cell := range sheet.Cells {
			key := fmt.Sprintf("%d:%d", cell.Row, cell.Col)
			cellMap[key] = cell
		}

		// Calculate column widths
		pageWidth, _ := pdf.GetPageSize()
		marginLeft, _, marginRight, _ := pdf.GetMargins()
		availableWidth := pageWidth - marginLeft - marginRight

		numCols := sheet.MaxCol + 1
		if opts.IncludeHeaders {
			numCols++ // Extra column for row numbers
		}

		colWidth := availableWidth / float64(numCols)
		if colWidth < 15 {
			colWidth = 15
		}
		if colWidth > 50 {
			colWidth = 50
		}

		rowHeight := 7.0

		// Set table font
		pdf.SetFont("Arial", "", 9)

		// Draw header row if requested
		if opts.IncludeHeaders {
			pdf.SetFillColor(240, 240, 240)
			pdf.SetFont("Arial", "B", 9)

			// Row number header
			pdf.CellFormat(colWidth, rowHeight, "", "1", 0, "C", true, 0, "")

			// Column headers
			for col := 0; col <= sheet.MaxCol; col++ {
				label := getColumnLabel(col)
				pdf.CellFormat(colWidth, rowHeight, label, "1", 0, "C", true, 0, "")
			}
			pdf.Ln(-1)
		}

		// Draw data rows
		pdf.SetFont("Arial", "", 9)
		for row := 0; row <= sheet.MaxRow; row++ {
			// Check if we need a new page
			_, pageHeight := pdf.GetPageSize()
			_, _, _, marginBottom := pdf.GetMargins()
			if pdf.GetY() > pageHeight-marginBottom-rowHeight {
				pdf.AddPage()
				pdf.SetFont("Arial", "", 9)
			}

			// Row number
			if opts.IncludeHeaders {
				pdf.SetFillColor(240, 240, 240)
				pdf.SetFont("Arial", "B", 9)
				pdf.CellFormat(colWidth, rowHeight, fmt.Sprintf("%d", row+1), "1", 0, "C", true, 0, "")
				pdf.SetFont("Arial", "", 9)
			}

			// Data cells
			for col := 0; col <= sheet.MaxCol; col++ {
				key := fmt.Sprintf("%d:%d", row, col)
				cell, exists := cellMap[key]

				value := ""
				if exists {
					if opts.ExportFormulas && cell.Formula != "" {
						value = cell.Formula
					} else if cell.Display != "" {
						value = cell.Display
					} else if cell.Value != nil {
						value = fmt.Sprintf("%v", cell.Value)
					}
				}

				// Truncate long values
				maxLen := int(colWidth / 2)
				if len(value) > maxLen {
					value = value[:maxLen-3] + "..."
				}

				// Apply formatting if available
				fillColor := false
				if opts.ExportFormatting && exists && cell.Format != nil {
					if cell.Format.Bold {
						pdf.SetFont("Arial", "B", 9)
					} else if cell.Format.Italic {
						pdf.SetFont("Arial", "I", 9)
					}
					if cell.Format.BackgroundColor != "" {
						r, g, b := hexToRGB(cell.Format.BackgroundColor)
						pdf.SetFillColor(r, g, b)
						fillColor = true
					}
				}

				// Draw grid if requested
				border := ""
				if opts.IncludeGridlines {
					border = "1"
				}

				align := "L"
				if exists && cell.Format != nil {
					switch cell.Format.HAlign {
					case "center":
						align = "C"
					case "right":
						align = "R"
					}
				}

				pdf.CellFormat(colWidth, rowHeight, value, border, 0, align, fillColor, 0, "")

				// Reset font
				pdf.SetFont("Arial", "", 9)
				pdf.SetFillColor(255, 255, 255)
			}
			pdf.Ln(-1)
		}
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("generate pdf: %w", err)
	}

	return buf.Bytes(), nil
}

// hexToRGB converts hex color to RGB values.
func hexToRGB(hex string) (int, int, int) {
	if hex == "" {
		return 255, 255, 255
	}
	// Remove # prefix
	if len(hex) > 0 && hex[0] == '#' {
		hex = hex[1:]
	}
	if len(hex) != 6 {
		return 255, 255, 255
	}

	var r, g, b int
	fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	return r, g, b
}
