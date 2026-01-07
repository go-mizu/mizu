package api

import (
	"io"
	"net/http"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/spreadsheet/feature/export"
	"github.com/go-mizu/blueprints/spreadsheet/feature/importer"
)

// ImportExport handles import and export endpoints.
type ImportExport struct {
	exporter  export.API
	importer  importer.API
	getUserID func(*mizu.Ctx) string
}

// NewImportExport creates a new ImportExport handler.
func NewImportExport(exporter export.API, importer importer.API, getUserID func(*mizu.Ctx) string) *ImportExport {
	return &ImportExport{
		exporter:  exporter,
		importer:  importer,
		getUserID: getUserID,
	}
}

// ExportWorkbook exports a workbook.
func (h *ImportExport) ExportWorkbook(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	workbookID := c.Param("id")

	// Get format from query or body
	format := export.Format(c.Query("format"))
	if format == "" {
		format = export.FormatXLSX
	}

	// Parse options
	opts := &export.Options{
		ExportFormatting: c.Query("formatting") == "true",
		ExportFormulas:   c.Query("formulas") == "true",
		IncludeHeaders:   c.Query("headers") == "true",
		IncludeGridlines: c.Query("gridlines") == "true",
		Orientation:      c.Query("orientation"),
		PaperSize:        c.Query("paperSize"),
		Compact:          c.Query("compact") == "true",
		IncludeMetadata:  c.Query("metadata") == "true",
	}

	result, err := h.exporter.ExportWorkbook(c.Request().Context(), workbookID, format, opts)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Set response headers
	c.Writer().Header().Set("Content-Type", result.ContentType)
	c.Writer().Header().Set("Content-Disposition", "attachment; filename=\""+result.Filename+"\"")
	c.Writer().Header().Set("Content-Length", string(rune(result.Size)))

	// Stream response
	io.Copy(c.Writer(), result.Data)
	return nil
}

// ExportSheet exports a single sheet.
func (h *ImportExport) ExportSheet(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	sheetID := c.Param("id")

	// Get format from query
	format := export.Format(c.Query("format"))
	if format == "" {
		format = export.FormatCSV
	}

	// Parse options
	opts := &export.Options{
		ExportFormatting: c.Query("formatting") == "true",
		ExportFormulas:   c.Query("formulas") == "true",
		IncludeHeaders:   c.Query("headers") == "true",
		IncludeGridlines: c.Query("gridlines") == "true",
		Orientation:      c.Query("orientation"),
		PaperSize:        c.Query("paperSize"),
		Compact:          c.Query("compact") == "true",
	}

	result, err := h.exporter.ExportSheet(c.Request().Context(), sheetID, format, opts)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Set response headers
	c.Writer().Header().Set("Content-Type", result.ContentType)
	c.Writer().Header().Set("Content-Disposition", "attachment; filename=\""+result.Filename+"\"")

	// Stream response
	io.Copy(c.Writer(), result.Data)
	return nil
}

// ImportToWorkbook imports a file to a workbook.
func (h *ImportExport) ImportToWorkbook(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	workbookID := c.Param("id")

	// Parse multipart form (50MB max)
	if err := c.Request().ParseMultipartForm(50 << 20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid form data"})
	}

	// Get the file
	file, header, err := c.Request().FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "file is required"})
	}
	defer file.Close()

	// Parse options from form
	opts := &importer.Options{
		HasHeaders:       c.Request().FormValue("hasHeaders") == "true",
		SkipEmptyRows:    c.Request().FormValue("skipEmptyRows") == "true",
		TrimWhitespace:   c.Request().FormValue("trimWhitespace") == "true",
		OverwriteExisting: c.Request().FormValue("overwriteExisting") == "true",
		AutoDetectTypes:  c.Request().FormValue("autoDetectTypes") != "false", // Default true
		ImportFormatting: c.Request().FormValue("importFormatting") == "true",
		ImportFormulas:   c.Request().FormValue("importFormulas") == "true",
		ImportSheet:      c.Request().FormValue("importSheet"),
		SheetName:        c.Request().FormValue("sheetName"),
		CreateNewSheet:   true,
	}

	// Detect format
	format := importer.Format(c.Request().FormValue("format"))

	result, err := h.importer.ImportToWorkbook(c.Request().Context(), workbookID, file, header.Filename, format, opts)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"success": true,
		"data":    result,
	})
}

// ImportToSheet imports a file to an existing sheet.
func (h *ImportExport) ImportToSheet(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	sheetID := c.Param("id")

	// Parse multipart form (50MB max)
	if err := c.Request().ParseMultipartForm(50 << 20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid form data"})
	}

	// Get the file
	file, header, err := c.Request().FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "file is required"})
	}
	defer file.Close()

	// Parse options from form
	opts := &importer.Options{
		HasHeaders:       c.Request().FormValue("hasHeaders") == "true",
		SkipEmptyRows:    c.Request().FormValue("skipEmptyRows") == "true",
		TrimWhitespace:   c.Request().FormValue("trimWhitespace") == "true",
		OverwriteExisting: c.Request().FormValue("overwriteExisting") != "false", // Default true for existing sheet
		AutoDetectTypes:  c.Request().FormValue("autoDetectTypes") != "false",
		ImportFormatting: c.Request().FormValue("importFormatting") == "true",
		ImportFormulas:   c.Request().FormValue("importFormulas") == "true",
		ImportSheet:      c.Request().FormValue("importSheet"),
	}

	// Detect format
	format := importer.Format(c.Request().FormValue("format"))

	result, err := h.importer.ImportToSheet(c.Request().Context(), sheetID, file, header.Filename, format, opts)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"success": true,
		"data":    result,
	})
}

// SupportedFormats returns supported import/export formats.
func (h *ImportExport) SupportedFormats(c *mizu.Ctx) error {
	return c.JSON(http.StatusOK, map[string]any{
		"import": h.importer.SupportedFormats(),
		"export": h.exporter.SupportedFormats(),
	})
}
