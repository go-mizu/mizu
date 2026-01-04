package export

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// PDFConverter converts pages to PDF format.
type PDFConverter struct {
	htmlConverter *HTMLConverter
}

// NewPDFConverter creates a new PDF converter.
func NewPDFConverter() *PDFConverter {
	return &PDFConverter{
		htmlConverter: NewHTMLConverter(),
	}
}

// Convert converts an exported page to PDF.
func (c *PDFConverter) Convert(exportedPage *ExportedPage, opts *Request) ([]byte, error) {
	// First convert to HTML
	htmlContent, err := c.htmlConverter.Convert(exportedPage, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to generate HTML: %w", err)
	}

	// Try chromedp first, fall back to wkhtmltopdf
	pdfData, err := c.convertWithChromedp(htmlContent, opts)
	if err != nil {
		// Try wkhtmltopdf as fallback
		pdfData, err = c.convertWithWkhtmltopdf(htmlContent, opts)
		if err != nil {
			// Return HTML as fallback if PDF generation fails
			return htmlContent, fmt.Errorf("PDF generation failed, returning HTML: %w", err)
		}
	}

	return pdfData, nil
}

// ContentType returns the MIME type.
func (c *PDFConverter) ContentType() string {
	return "application/pdf"
}

// Extension returns the file extension.
func (c *PDFConverter) Extension() string {
	return ".pdf"
}

// convertWithChromedp uses headless Chrome to generate PDF.
func (c *PDFConverter) convertWithChromedp(htmlContent []byte, opts *Request) ([]byte, error) {
	// Create context
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Set timeout
	ctx, cancel = context.WithTimeout(ctx, 60000000000) // 60 seconds
	defer cancel()

	// Get page dimensions
	width, height := c.getPageDimensions(opts.PageSize)
	landscape := opts.Orientation == OrientationLandscape

	// Calculate scale
	scale := 1.0
	if opts.Scale > 0 && opts.Scale <= 200 {
		scale = float64(opts.Scale) / 100.0
	}

	// Create a temporary file for HTML
	tmpFile, err := os.CreateTemp("", "export-*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(htmlContent); err != nil {
		return nil, fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpFile.Close()

	var pdfData []byte

	// Navigate to HTML and print to PDF
	err = chromedp.Run(ctx,
		chromedp.Navigate("file://"+tmpFile.Name()),
		chromedp.WaitReady("body"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			pdfData, _, err = page.PrintToPDF().
				WithPaperWidth(width).
				WithPaperHeight(height).
				WithLandscape(landscape).
				WithScale(scale).
				WithPrintBackground(true).
				WithMarginTop(0.5).
				WithMarginBottom(0.5).
				WithMarginLeft(0.5).
				WithMarginRight(0.5).
				Do(ctx)
			return err
		}),
	)

	if err != nil {
		return nil, fmt.Errorf("chromedp PDF generation failed: %w", err)
	}

	return pdfData, nil
}

// convertWithWkhtmltopdf uses wkhtmltopdf as a fallback.
func (c *PDFConverter) convertWithWkhtmltopdf(htmlContent []byte, opts *Request) ([]byte, error) {
	// Check if wkhtmltopdf is available
	_, err := exec.LookPath("wkhtmltopdf")
	if err != nil {
		return nil, fmt.Errorf("wkhtmltopdf not found: %w", err)
	}

	// Create temporary files
	tmpDir, err := os.MkdirTemp("", "export-")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	htmlPath := filepath.Join(tmpDir, "input.html")
	pdfPath := filepath.Join(tmpDir, "output.pdf")

	if err := os.WriteFile(htmlPath, htmlContent, 0644); err != nil {
		return nil, fmt.Errorf("failed to write HTML file: %w", err)
	}

	// Build wkhtmltopdf command
	args := []string{
		"--enable-local-file-access",
		"--page-size", c.wkhtmlPageSize(opts.PageSize),
		"--orientation", c.wkhtmlOrientation(opts.Orientation),
		"--margin-top", "12.7mm",
		"--margin-bottom", "12.7mm",
		"--margin-left", "12.7mm",
		"--margin-right", "12.7mm",
		"--print-media-type",
	}

	if opts.Scale > 0 && opts.Scale <= 200 {
		args = append(args, "--zoom", fmt.Sprintf("%.2f", float64(opts.Scale)/100.0))
	}

	args = append(args, htmlPath, pdfPath)

	cmd := exec.Command("wkhtmltopdf", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("wkhtmltopdf failed: %w, output: %s", err, string(output))
	}

	pdfData, err := os.ReadFile(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF output: %w", err)
	}

	return pdfData, nil
}

// getPageDimensions returns paper width and height in inches.
func (c *PDFConverter) getPageDimensions(pageSize PageSize) (width, height float64) {
	switch pageSize {
	case PageSizeA4:
		return 8.27, 11.69
	case PageSizeA3:
		return 11.69, 16.54
	case PageSizeLetter:
		return 8.5, 11.0
	case PageSizeLegal:
		return 8.5, 14.0
	case PageSizeTabloid:
		return 11.0, 17.0
	default: // Auto - default to Letter
		return 8.5, 11.0
	}
}

// wkhtmlPageSize converts page size to wkhtmltopdf format.
func (c *PDFConverter) wkhtmlPageSize(pageSize PageSize) string {
	switch pageSize {
	case PageSizeA4:
		return "A4"
	case PageSizeA3:
		return "A3"
	case PageSizeLetter:
		return "Letter"
	case PageSizeLegal:
		return "Legal"
	case PageSizeTabloid:
		return "Tabloid"
	default:
		return "Letter"
	}
}

// wkhtmlOrientation converts orientation to wkhtmltopdf format.
func (c *PDFConverter) wkhtmlOrientation(orientation Orientation) string {
	if orientation == OrientationLandscape {
		return "Landscape"
	}
	return "Portrait"
}

// PDFConvertSimple provides a simple PDF conversion using just HTML + inline styles
// This is a fallback when neither chromedp nor wkhtmltopdf is available.
type PDFConvertSimple struct {
	htmlConverter *HTMLConverter
}

// NewPDFConverterSimple creates a simple PDF converter that returns HTML.
func NewPDFConverterSimple() *PDFConvertSimple {
	return &PDFConvertSimple{
		htmlConverter: NewHTMLConverter(),
	}
}

// Convert returns HTML content (fallback when no PDF tools available).
func (c *PDFConvertSimple) Convert(page *ExportedPage, opts *Request) ([]byte, error) {
	// Modify HTML to include print CSS
	htmlContent, err := c.htmlConverter.Convert(page, opts)
	if err != nil {
		return nil, err
	}

	// Inject print-specific CSS
	printCSS := `
<style>
@page {
  size: ` + c.getPageSizeCSS(opts.PageSize, opts.Orientation) + `;
  margin: 1in;
}
body {
  -webkit-print-color-adjust: exact;
  print-color-adjust: exact;
}
</style>
</head>`

	result := strings.Replace(string(htmlContent), "</head>", printCSS, 1)
	return []byte(result), nil
}

// getPageSizeCSS returns the CSS @page size value.
func (c *PDFConvertSimple) getPageSizeCSS(pageSize PageSize, orientation Orientation) string {
	var size string
	switch pageSize {
	case PageSizeA4:
		size = "A4"
	case PageSizeA3:
		size = "A3"
	case PageSizeLetter:
		size = "letter"
	case PageSizeLegal:
		size = "legal"
	case PageSizeTabloid:
		size = "ledger"
	default:
		size = "letter"
	}

	if orientation == OrientationLandscape {
		size += " landscape"
	}

	return size
}
