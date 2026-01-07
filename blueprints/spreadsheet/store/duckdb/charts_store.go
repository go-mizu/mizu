package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/go-mizu/blueprints/spreadsheet/feature/charts"
)

// ChartsStore implements charts.Store.
type ChartsStore struct {
	db *sql.DB
}

// NewChartsStore creates a new charts store.
func NewChartsStore(db *sql.DB) *ChartsStore {
	return &ChartsStore{db: db}
}

// Create creates a new chart.
func (s *ChartsStore) Create(ctx context.Context, chart *charts.Chart) error {
	position, _ := json.Marshal(chart.Position)
	size, _ := json.Marshal(chart.Size)
	dataRanges, _ := json.Marshal(chart.DataRanges)
	title, _ := json.Marshal(chart.Title)
	subtitle, _ := json.Marshal(chart.Subtitle)
	legend, _ := json.Marshal(chart.Legend)
	axes, _ := json.Marshal(chart.Axes)
	series, _ := json.Marshal(chart.Series)
	options, _ := json.Marshal(chart.Options)

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO charts (id, sheet_id, name, chart_type, position, size, data_ranges,
			title, subtitle, legend, axes, series, options, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, chart.ID, chart.SheetID, chart.Name, string(chart.ChartType),
		string(position), string(size), string(dataRanges),
		string(title), string(subtitle), string(legend), string(axes),
		string(series), string(options), chart.CreatedAt, chart.UpdatedAt)
	return err
}

// GetByID retrieves a chart by ID.
func (s *ChartsStore) GetByID(ctx context.Context, id string) (*charts.Chart, error) {
	chart := &charts.Chart{}
	var chartType string
	var position, size, dataRanges sql.NullString
	var title, subtitle, legend, axes, series, options sql.NullString
	var name sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT id, sheet_id, name, chart_type,
			CAST(position AS VARCHAR), CAST(size AS VARCHAR), CAST(data_ranges AS VARCHAR),
			CAST(title AS VARCHAR), CAST(subtitle AS VARCHAR), CAST(legend AS VARCHAR),
			CAST(axes AS VARCHAR), CAST(series AS VARCHAR), CAST(options AS VARCHAR),
			created_at, updated_at
		FROM charts WHERE id = ?
	`, id).Scan(&chart.ID, &chart.SheetID, &name, &chartType,
		&position, &size, &dataRanges,
		&title, &subtitle, &legend, &axes, &series, &options,
		&chart.CreatedAt, &chart.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, charts.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	chart.ChartType = charts.ChartType(chartType)
	if name.Valid {
		chart.Name = name.String
	}

	// Unmarshal JSON fields
	if position.Valid {
		json.Unmarshal([]byte(position.String), &chart.Position)
	}
	if size.Valid {
		json.Unmarshal([]byte(size.String), &chart.Size)
	}
	if dataRanges.Valid {
		json.Unmarshal([]byte(dataRanges.String), &chart.DataRanges)
	}
	if title.Valid && title.String != "null" {
		json.Unmarshal([]byte(title.String), &chart.Title)
	}
	if subtitle.Valid && subtitle.String != "null" {
		json.Unmarshal([]byte(subtitle.String), &chart.Subtitle)
	}
	if legend.Valid && legend.String != "null" {
		json.Unmarshal([]byte(legend.String), &chart.Legend)
	}
	if axes.Valid && axes.String != "null" {
		json.Unmarshal([]byte(axes.String), &chart.Axes)
	}
	if series.Valid && series.String != "null" {
		json.Unmarshal([]byte(series.String), &chart.Series)
	}
	if options.Valid && options.String != "null" {
		json.Unmarshal([]byte(options.String), &chart.Options)
	}

	return chart, nil
}

// ListBySheet lists charts in a sheet.
func (s *ChartsStore) ListBySheet(ctx context.Context, sheetID string) ([]*charts.Chart, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, sheet_id, name, chart_type,
			CAST(position AS VARCHAR), CAST(size AS VARCHAR), CAST(data_ranges AS VARCHAR),
			CAST(title AS VARCHAR), CAST(subtitle AS VARCHAR), CAST(legend AS VARCHAR),
			CAST(axes AS VARCHAR), CAST(series AS VARCHAR), CAST(options AS VARCHAR),
			created_at, updated_at
		FROM charts WHERE sheet_id = ?
		ORDER BY created_at
	`, sheetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]*charts.Chart, 0)
	for rows.Next() {
		chart := &charts.Chart{}
		var chartType string
		var position, size, dataRanges sql.NullString
		var title, subtitle, legend, axes, series, options sql.NullString
		var name sql.NullString

		if err := rows.Scan(&chart.ID, &chart.SheetID, &name, &chartType,
			&position, &size, &dataRanges,
			&title, &subtitle, &legend, &axes, &series, &options,
			&chart.CreatedAt, &chart.UpdatedAt); err != nil {
			return nil, err
		}

		chart.ChartType = charts.ChartType(chartType)
		if name.Valid {
			chart.Name = name.String
		}

		// Unmarshal JSON fields
		if position.Valid {
			json.Unmarshal([]byte(position.String), &chart.Position)
		}
		if size.Valid {
			json.Unmarshal([]byte(size.String), &chart.Size)
		}
		if dataRanges.Valid {
			json.Unmarshal([]byte(dataRanges.String), &chart.DataRanges)
		}
		if title.Valid && title.String != "null" {
			json.Unmarshal([]byte(title.String), &chart.Title)
		}
		if subtitle.Valid && subtitle.String != "null" {
			json.Unmarshal([]byte(subtitle.String), &chart.Subtitle)
		}
		if legend.Valid && legend.String != "null" {
			json.Unmarshal([]byte(legend.String), &chart.Legend)
		}
		if axes.Valid && axes.String != "null" {
			json.Unmarshal([]byte(axes.String), &chart.Axes)
		}
		if series.Valid && series.String != "null" {
			json.Unmarshal([]byte(series.String), &chart.Series)
		}
		if options.Valid && options.String != "null" {
			json.Unmarshal([]byte(options.String), &chart.Options)
		}

		result = append(result, chart)
	}

	return result, nil
}

// Update updates a chart.
func (s *ChartsStore) Update(ctx context.Context, chart *charts.Chart) error {
	position, _ := json.Marshal(chart.Position)
	size, _ := json.Marshal(chart.Size)
	dataRanges, _ := json.Marshal(chart.DataRanges)
	title, _ := json.Marshal(chart.Title)
	subtitle, _ := json.Marshal(chart.Subtitle)
	legend, _ := json.Marshal(chart.Legend)
	axes, _ := json.Marshal(chart.Axes)
	series, _ := json.Marshal(chart.Series)
	options, _ := json.Marshal(chart.Options)

	_, err := s.db.ExecContext(ctx, `
		UPDATE charts SET name = ?, chart_type = ?, position = ?, size = ?,
			data_ranges = ?, title = ?, subtitle = ?, legend = ?, axes = ?,
			series = ?, options = ?, updated_at = ?
		WHERE id = ?
	`, chart.Name, string(chart.ChartType), string(position), string(size),
		string(dataRanges), string(title), string(subtitle), string(legend),
		string(axes), string(series), string(options), chart.UpdatedAt, chart.ID)
	return err
}

// Delete deletes a chart.
func (s *ChartsStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM charts WHERE id = ?`, id)
	return err
}
