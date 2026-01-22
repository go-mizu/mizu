package export

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

// Service implements the Export API.
type Service struct {
	questions  QuestionStore
	dashboards DashboardStore
	executor   QueryExecutor
	renderer   ScreenshotRenderer
}

// NewService creates a new Export service.
func NewService(
	questions QuestionStore,
	dashboards DashboardStore,
	executor QueryExecutor,
	renderer ScreenshotRenderer,
) *Service {
	return &Service{
		questions:  questions,
		dashboards: dashboards,
		executor:   executor,
		renderer:   renderer,
	}
}

// ExportQuestion exports a question's results.
func (s *Service) ExportQuestion(ctx context.Context, in *QuestionExportIn) (*ExportResult, error) {
	question, err := s.questions.GetByID(ctx, in.QuestionID)
	if err != nil {
		return nil, ErrQuestionNotFound
	}

	// Execute the question query
	columns, rows, err := s.executor.Execute(ctx, question.Query)
	if err != nil {
		return nil, fmt.Errorf("execute query: %w", err)
	}

	// Apply limit if specified
	if in.Limit > 0 && len(rows) > in.Limit {
		rows = rows[:in.Limit]
	}

	// Export based on format
	return s.ExportQueryResult(ctx, &QueryResultExportIn{
		Columns: columns,
		Rows:    rows,
		Format:  in.Format,
		Options: in.Options,
	})
}

// ExportDashboard exports a dashboard as PDF or PNG.
func (s *Service) ExportDashboard(ctx context.Context, in *DashboardExportIn) (*ExportResult, error) {
	dashboard, err := s.dashboards.GetByID(ctx, in.DashboardID)
	if err != nil {
		return nil, ErrDashboardNotFound
	}

	if s.renderer == nil {
		return nil, fmt.Errorf("screenshot renderer not configured")
	}

	var data []byte
	var contentType string
	var ext string

	switch in.Format {
	case FormatPNG:
		width := in.Width
		if width == 0 {
			width = 1200
		}
		height := in.Height
		if height == 0 {
			height = 800
		}
		data, err = s.renderer.RenderDashboard(ctx, in.DashboardID, width, height)
		contentType = "image/png"
		ext = "png"

	case FormatPDF:
		opts := &PDFOptions{}
		if in.Options != nil {
			if ps, ok := in.Options["page_size"].(string); ok {
				opts.PageSize = ps
			}
			if o, ok := in.Options["orientation"].(string); ok {
				opts.Orientation = o
			}
		}
		data, err = s.renderer.RenderDashboardPDF(ctx, in.DashboardID, opts)
		contentType = "application/pdf"
		ext = "pdf"

	default:
		return nil, fmt.Errorf("%w: %s (use png or pdf)", ErrUnsupportedFormat, in.Format)
	}

	if err != nil {
		return nil, fmt.Errorf("render dashboard: %w", err)
	}

	filename := sanitizeFilename(dashboard.Name) + "." + ext

	return &ExportResult{
		Data:        data,
		ContentType: contentType,
		Filename:    filename,
		Size:        int64(len(data)),
		CreatedAt:   time.Now(),
	}, nil
}

// ExportQueryResult exports raw query results.
func (s *Service) ExportQueryResult(ctx context.Context, in *QueryResultExportIn) (*ExportResult, error) {
	if len(in.Rows) == 0 {
		return nil, ErrNoData
	}

	var buf bytes.Buffer
	var contentType string
	var ext string

	switch in.Format {
	case FormatCSV:
		opts := &CSVOptions{IncludeHeader: true}
		if in.Options != nil {
			if d, ok := in.Options["delimiter"].(string); ok {
				opts.Delimiter = d
			}
			if h, ok := in.Options["include_header"].(bool); ok {
				opts.IncludeHeader = h
			}
		}
		if err := s.WriteCSV(ctx, &buf, in.Columns, in.Rows, opts); err != nil {
			return nil, fmt.Errorf("write csv: %w", err)
		}
		contentType = "text/csv"
		ext = "csv"

	case FormatXLSX:
		opts := &XLSXOptions{IncludeHeader: true, SheetName: "Sheet1"}
		if in.Options != nil {
			if sn, ok := in.Options["sheet_name"].(string); ok {
				opts.SheetName = sn
			}
			if h, ok := in.Options["include_header"].(bool); ok {
				opts.IncludeHeader = h
			}
		}
		if err := s.WriteXLSX(ctx, &buf, in.Columns, in.Rows, opts); err != nil {
			return nil, fmt.Errorf("write xlsx: %w", err)
		}
		contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
		ext = "xlsx"

	case FormatJSON:
		if err := s.WriteJSON(ctx, &buf, in.Columns, in.Rows); err != nil {
			return nil, fmt.Errorf("write json: %w", err)
		}
		contentType = "application/json"
		ext = "json"

	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedFormat, in.Format)
	}

	return &ExportResult{
		Data:        buf.Bytes(),
		ContentType: contentType,
		Filename:    fmt.Sprintf("export_%s.%s", time.Now().Format("20060102_150405"), ext),
		Size:        int64(buf.Len()),
		CreatedAt:   time.Now(),
	}, nil
}

// WriteCSV writes data as CSV to a writer.
func (s *Service) WriteCSV(ctx context.Context, w io.Writer, columns []Column, rows []map[string]any, opts *CSVOptions) error {
	if opts == nil {
		opts = &CSVOptions{IncludeHeader: true}
	}

	writer := csv.NewWriter(w)
	if opts.Delimiter != "" && len(opts.Delimiter) == 1 {
		writer.Comma = rune(opts.Delimiter[0])
	}

	// Write header
	if opts.IncludeHeader {
		headers := make([]string, len(columns))
		for i, col := range columns {
			if col.DisplayName != "" {
				headers[i] = col.DisplayName
			} else {
				headers[i] = col.Name
			}
		}
		if err := writer.Write(headers); err != nil {
			return err
		}
	}

	// Write rows
	for _, row := range rows {
		record := make([]string, len(columns))
		for i, col := range columns {
			record[i] = formatValue(row[col.Name])
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	writer.Flush()
	return writer.Error()
}

// WriteXLSX writes data as Excel to a writer.
// Note: This is a simplified implementation. For production, use excelize library.
func (s *Service) WriteXLSX(ctx context.Context, w io.Writer, columns []Column, rows []map[string]any, opts *XLSXOptions) error {
	if opts == nil {
		opts = &XLSXOptions{IncludeHeader: true, SheetName: "Sheet1"}
	}

	// For now, write as CSV (placeholder for xlsx library integration)
	// In production, use github.com/xuri/excelize/v2
	csvOpts := &CSVOptions{IncludeHeader: opts.IncludeHeader}
	return s.WriteCSV(ctx, w, columns, rows, csvOpts)
}

// WriteJSON writes data as JSON to a writer.
func (s *Service) WriteJSON(ctx context.Context, w io.Writer, columns []Column, rows []map[string]any) error {
	output := map[string]any{
		"columns": columns,
		"rows":    rows,
		"metadata": map[string]any{
			"row_count":  len(rows),
			"exported_at": time.Now().Format(time.RFC3339),
		},
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// Helper functions

func formatValue(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%g", val)
	case float32:
		if val == float32(int32(val)) {
			return fmt.Sprintf("%d", int32(val))
		}
		return fmt.Sprintf("%g", val)
	case int, int32, int64:
		return fmt.Sprintf("%d", val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case time.Time:
		return val.Format(time.RFC3339)
	default:
		return fmt.Sprintf("%v", val)
	}
}

func sanitizeFilename(name string) string {
	// Remove or replace characters that are unsafe in filenames
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
	result := replacer.Replace(name)
	// Trim to reasonable length
	if len(result) > 100 {
		result = result[:100]
	}
	return strings.TrimSpace(result)
}
