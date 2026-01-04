package export

import (
	"archive/zip"
	"bytes"
	"fmt"
	"html"
	"io"
	"net/http"
	"path"
	"strings"
	"time"
)

// Bundler creates ZIP archives for multi-file exports.
type Bundler struct {
	buffer  *bytes.Buffer
	writer  *zip.Writer
	files   map[string]bool
	created time.Time
}

// NewBundler creates a new ZIP bundler.
func NewBundler() *Bundler {
	buf := new(bytes.Buffer)
	return &Bundler{
		buffer:  buf,
		writer:  zip.NewWriter(buf),
		files:   make(map[string]bool),
		created: time.Now(),
	}
}

// AddFile adds a file to the bundle.
func (b *Bundler) AddFile(filePath string, content []byte) error {
	// Normalize path
	filePath = normalizePath(filePath)

	// Avoid duplicates
	if b.files[filePath] {
		return nil
	}

	header := &zip.FileHeader{
		Name:     filePath,
		Method:   zip.Deflate,
		Modified: b.created,
	}

	writer, err := b.writer.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("failed to create zip entry: %w", err)
	}

	if _, err := writer.Write(content); err != nil {
		return fmt.Errorf("failed to write zip content: %w", err)
	}

	b.files[filePath] = true
	return nil
}

// AddAsset downloads and adds an asset (image/file) to the bundle.
func (b *Bundler) AddAsset(assetPath, url string) error {
	// Normalize path
	assetPath = normalizePath(assetPath)

	// Avoid duplicates
	if b.files[assetPath] {
		return nil
	}

	// Download the asset
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download asset: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("asset download failed with status: %d", resp.StatusCode)
	}

	// Read content
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read asset content: %w", err)
	}

	return b.AddFile(assetPath, content)
}

// AddAssetBytes adds asset bytes directly to the bundle.
func (b *Bundler) AddAssetBytes(assetPath string, content []byte) error {
	return b.AddFile(assetPath, content)
}

// CreateSitemap creates an index.html sitemap for the export.
func (b *Bundler) CreateSitemap(pages []*ExportedPage, format Format) error {
	var sb strings.Builder

	sb.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n")
	sb.WriteString("<meta charset=\"UTF-8\">\n")
	sb.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	sb.WriteString("<title>Export Index</title>\n")
	sb.WriteString("<style>\n")
	sb.WriteString(`
body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Helvetica, Arial, sans-serif;
  max-width: 800px;
  margin: 40px auto;
  padding: 0 20px;
  line-height: 1.6;
}
h1 { font-size: 28px; margin-bottom: 24px; }
ul { list-style: none; padding: 0; }
li { margin: 8px 0; }
a { color: #2563eb; text-decoration: none; }
a:hover { text-decoration: underline; }
.page-item { display: flex; align-items: center; gap: 8px; }
.page-icon { font-size: 18px; }
.nested { margin-left: 24px; }
`)
	sb.WriteString("</style>\n</head>\n<body>\n")
	sb.WriteString("<h1>Export Index</h1>\n")
	sb.WriteString("<p>Exported on " + b.created.Format("January 2, 2006 at 3:04 PM") + "</p>\n")
	sb.WriteString("<ul>\n")

	b.writeSitemapPages(&sb, pages, format, 0)

	sb.WriteString("</ul>\n</body>\n</html>")

	return b.AddFile("index.html", []byte(sb.String()))
}

// writeSitemapPages recursively writes page links to the sitemap.
func (b *Bundler) writeSitemapPages(sb *strings.Builder, pages []*ExportedPage, format Format, depth int) {
	ext := ".html"
	if format == FormatMarkdown {
		ext = ".md"
	} else if format == FormatPDF {
		ext = ".pdf"
	}

	nestedClass := ""
	if depth > 0 {
		nestedClass = " class=\"nested\""
	}

	for _, page := range pages {
		icon := page.Icon
		if icon == "" {
			icon = "📄"
		}

		filename := page.Path
		if filename == "" {
			filename = sanitizeFilename(page.Title) + ext
		}

		sb.WriteString(fmt.Sprintf("<li%s>\n", nestedClass))
		sb.WriteString("<div class=\"page-item\">\n")
		sb.WriteString(fmt.Sprintf("<span class=\"page-icon\">%s</span>\n", icon))
		sb.WriteString(fmt.Sprintf("<a href=\"%s\">%s</a>\n", html.EscapeString(filename), html.EscapeString(page.Title)))
		sb.WriteString("</div>\n")

		if len(page.Children) > 0 {
			sb.WriteString("<ul>\n")
			b.writeSitemapPages(sb, page.Children, format, depth+1)
			sb.WriteString("</ul>\n")
		}

		sb.WriteString("</li>\n")
	}
}

// Close finalizes the ZIP archive and returns the bytes.
func (b *Bundler) Close() ([]byte, error) {
	if err := b.writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close zip: %w", err)
	}
	return b.buffer.Bytes(), nil
}

// FileCount returns the number of files in the bundle.
func (b *Bundler) FileCount() int {
	return len(b.files)
}

// normalizePath cleans up a file path for ZIP archives.
func normalizePath(filePath string) string {
	// Remove leading slashes
	filePath = strings.TrimPrefix(filePath, "/")

	// Clean the path
	filePath = path.Clean(filePath)

	// Replace backslashes
	filePath = strings.ReplaceAll(filePath, "\\", "/")

	return filePath
}

// CreateFolder creates a folder entry in the ZIP (empty directory).
func (b *Bundler) CreateFolder(folderPath string) error {
	// Ensure trailing slash for directories
	if !strings.HasSuffix(folderPath, "/") {
		folderPath += "/"
	}

	folderPath = normalizePath(folderPath) + "/"

	if b.files[folderPath] {
		return nil
	}

	header := &zip.FileHeader{
		Name:     folderPath,
		Method:   zip.Store,
		Modified: b.created,
	}

	_, err := b.writer.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("failed to create folder: %w", err)
	}

	b.files[folderPath] = true
	return nil
}

// ExportBundle represents a complete export bundle ready for download.
type ExportBundle struct {
	Filename    string
	ContentType string
	Data        []byte
	PageCount   int
}

// CreateBundle creates a complete export bundle from a page tree.
func CreateBundle(rootPage *ExportedPage, opts *Request) (*ExportBundle, error) {
	// Single page, no subpages
	if !opts.IncludeSubpages || len(rootPage.Children) == 0 {
		return createSingleFileExport(rootPage, opts)
	}

	// Multi-page export - create ZIP
	return createZipExport(rootPage, opts)
}

// createSingleFileExport creates a single-file export.
func createSingleFileExport(page *ExportedPage, opts *Request) (*ExportBundle, error) {
	var content []byte
	var ext string
	var contentType string
	var err error

	switch opts.Format {
	case FormatMarkdown:
		converter := NewMarkdownConverter()
		content, err = converter.Convert(page, opts)
		ext = converter.Extension()
		contentType = converter.ContentType()

	case FormatHTML:
		converter := NewHTMLConverter()
		content, err = converter.Convert(page, opts)
		ext = converter.Extension()
		contentType = converter.ContentType()

	case FormatPDF:
		converter := NewPDFConverter()
		content, err = converter.Convert(page, opts)
		ext = converter.Extension()
		contentType = converter.ContentType()

	default:
		return nil, fmt.Errorf("unsupported format: %s", opts.Format)
	}

	if err != nil {
		return nil, err
	}

	filename := sanitizeFilename(page.Title) + ext

	return &ExportBundle{
		Filename:    filename,
		ContentType: contentType,
		Data:        content,
		PageCount:   1,
	}, nil
}

// createZipExport creates a ZIP export with multiple files.
func createZipExport(rootPage *ExportedPage, opts *Request) (*ExportBundle, error) {
	bundler := NewBundler()
	pageCount := 0

	// Process the root page and all children
	err := processPageForBundle(bundler, rootPage, "", opts, &pageCount)
	if err != nil {
		return nil, err
	}

	// Create sitemap
	if err := bundler.CreateSitemap([]*ExportedPage{rootPage}, opts.Format); err != nil {
		return nil, fmt.Errorf("failed to create sitemap: %w", err)
	}

	// Close and get ZIP data
	data, err := bundler.Close()
	if err != nil {
		return nil, err
	}

	filename := sanitizeFilename(rootPage.Title) + ".zip"

	return &ExportBundle{
		Filename:    filename,
		ContentType: "application/zip",
		Data:        data,
		PageCount:   pageCount,
	}, nil
}

// processPageForBundle processes a page and adds it to the bundle.
func processPageForBundle(bundler *Bundler, page *ExportedPage, basePath string, opts *Request, pageCount *int) error {
	var content []byte
	var ext string
	var err error

	switch opts.Format {
	case FormatMarkdown:
		converter := NewMarkdownConverter()
		content, err = converter.Convert(page, opts)
		ext = converter.Extension()

	case FormatHTML:
		converter := NewHTMLConverter()
		content, err = converter.Convert(page, opts)
		ext = converter.Extension()

	case FormatPDF:
		converter := NewPDFConverter()
		content, err = converter.Convert(page, opts)
		ext = converter.Extension()

	default:
		return fmt.Errorf("unsupported format: %s", opts.Format)
	}

	if err != nil {
		return err
	}

	// Determine file path
	filename := sanitizeFilename(page.Title) + ext
	filePath := filename
	if basePath != "" && opts.CreateFolders {
		filePath = path.Join(basePath, filename)
	}

	page.Path = filePath

	// Add file to bundle
	if err := bundler.AddFile(filePath, content); err != nil {
		return err
	}

	*pageCount++

	// Process children
	if len(page.Children) > 0 {
		childBasePath := ""
		if opts.CreateFolders {
			childBasePath = path.Join(basePath, sanitizeFilename(page.Title))
			bundler.CreateFolder(childBasePath)
		}

		for _, child := range page.Children {
			if err := processPageForBundle(bundler, child, childBasePath, opts, pageCount); err != nil {
				return err
			}
		}
	}

	return nil
}
