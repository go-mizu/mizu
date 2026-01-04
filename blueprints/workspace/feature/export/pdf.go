package export

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"runtime"

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
	slog.Debug("starting PDF conversion", "title", exportedPage.Title)

	// First convert to HTML
	htmlContent, err := c.htmlConverter.Convert(exportedPage, opts)
	if err != nil {
		slog.Error("PDF: failed to generate HTML", "error", err)
		return nil, fmt.Errorf("failed to generate HTML: %w", err)
	}
	slog.Debug("PDF: HTML content generated", "size", len(htmlContent))

	// Use chromedp (headless Chrome) for PDF generation
	slog.Debug("PDF: starting chromedp conversion")
	pdfData, err := c.convertWithChromedp(htmlContent, opts)
	if err != nil {
		slog.Error("PDF: chromedp conversion failed", "error", err)
		return nil, fmt.Errorf("PDF generation failed: %w. "+
			"Please ensure Chrome or Chromium is installed (https://www.google.com/chrome/)", err)
	}

	slog.Info("PDF: generated using chromedp", "size", len(pdfData))
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

// findChrome attempts to find Chrome/Chromium executable.
func findChrome() (string, error) {
	// Check common Chrome paths based on OS
	var paths []string

	switch runtime.GOOS {
	case "darwin":
		paths = []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"/Applications/Google Chrome Canary.app/Contents/MacOS/Google Chrome Canary",
		}
	case "linux":
		paths = []string{
			"/usr/bin/google-chrome",
			"/usr/bin/google-chrome-stable",
			"/usr/bin/chromium",
			"/usr/bin/chromium-browser",
			"/snap/bin/chromium",
		}
	case "windows":
		paths = []string{
			`C:\Program Files\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		}
	}

	// Check each path
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	// Try PATH
	for _, name := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "chrome"} {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("Chrome/Chromium not found. Install Google Chrome or set CHROME_PATH environment variable")
}

// convertWithChromedp uses headless Chrome to generate PDF.
func (c *PDFConverter) convertWithChromedp(htmlContent []byte, opts *Request) ([]byte, error) {
	// Check if Chrome is available
	chromePath, err := findChrome()
	if err != nil {
		// Check environment variable
		if envPath := os.Getenv("CHROME_PATH"); envPath != "" {
			if _, statErr := os.Stat(envPath); statErr == nil {
				chromePath = envPath
			}
		}
		if chromePath == "" {
			slog.Error("Chrome not found",
				"error", err,
				"hint", "Install Google Chrome or set CHROME_PATH environment variable",
			)
			return nil, err
		}
	}
	slog.Debug("PDF: using Chrome", "path", chromePath)

	// Create allocator with explicit Chrome path
	allocCtx, allocCancel := chromedp.NewExecAllocator(
		context.Background(),
		chromedp.ExecPath(chromePath),
		chromedp.Headless,
		chromedp.DisableGPU,
		chromedp.NoSandbox, // Required for some environments
	)
	defer allocCancel()

	// Create context
	ctx, cancel := chromedp.NewContext(allocCtx)
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

	slog.Debug("PDF: starting Chrome",
		"temp_file", tmpFile.Name(),
		"page_size", opts.PageSize,
		"orientation", opts.Orientation,
	)

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
		slog.Error("PDF: chromedp execution failed",
			"error", err,
			"chrome_path", chromePath,
		)
		return nil, fmt.Errorf("chromedp PDF generation failed: %w", err)
	}

	slog.Debug("PDF: chromedp completed", "pdf_size", len(pdfData))
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

