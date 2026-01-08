package postgres

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
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
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
	var position, size, dataRanges []byte
	var title, subtitle, legend, axes, series, options []byte
	var name sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT id, sheet_id, name, chart_type,
			position, size, data_ranges,
			title, subtitle, legend, axes, series, options,
			created_at, updated_at
		FROM charts WHERE id = $1
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
	if len(position) > 0 {
		json.Unmarshal(position, &chart.Position)
	}
	if len(size) > 0 {
		json.Unmarshal(size, &chart.Size)
	}
	if len(dataRanges) > 0 {
		json.Unmarshal(dataRanges, &chart.DataRanges)
	}
	if len(title) > 0 && string(title) != "null" {
		json.Unmarshal(title, &chart.Title)
	}
	if len(subtitle) > 0 && string(subtitle) != "null" {
		json.Unmarshal(subtitle, &chart.Subtitle)
	}
	if len(legend) > 0 && string(legend) != "null" {
		json.Unmarshal(legend, &chart.Legend)
	}
	if len(axes) > 0 && string(axes) != "null" {
		json.Unmarshal(axes, &chart.Axes)
	}
	if len(series) > 0 && string(series) != "null" {
		json.Unmarshal(series, &chart.Series)
	}
	if len(options) > 0 && string(options) != "null" {
		json.Unmarshal(options, &chart.Options)
	}

	return chart, nil
}

// ListBySheet lists charts in a sheet.
func (s *ChartsStore) ListBySheet(ctx context.Context, sheetID string) ([]*charts.Chart, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, sheet_id, name, chart_type,
			position, size, data_ranges,
			title, subtitle, legend, axes, series, options,
			created_at, updated_at
		FROM charts WHERE sheet_id = $1
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
		var position, size, dataRanges []byte
		var title, subtitle, legend, axes, series, options []byte
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
		if len(position) > 0 {
			json.Unmarshal(position, &chart.Position)
		}
		if len(size) > 0 {
			json.Unmarshal(size, &chart.Size)
		}
		if len(dataRanges) > 0 {
			json.Unmarshal(dataRanges, &chart.DataRanges)
		}
		if len(title) > 0 && string(title) != "null" {
			json.Unmarshal(title, &chart.Title)
		}
		if len(subtitle) > 0 && string(subtitle) != "null" {
			json.Unmarshal(subtitle, &chart.Subtitle)
		}
		if len(legend) > 0 && string(legend) != "null" {
			json.Unmarshal(legend, &chart.Legend)
		}
		if len(axes) > 0 && string(axes) != "null" {
			json.Unmarshal(axes, &chart.Axes)
		}
		if len(series) > 0 && string(series) != "null" {
			json.Unmarshal(series, &chart.Series)
		}
		if len(options) > 0 && string(options) != "null" {
			json.Unmarshal(options, &chart.Options)
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
		UPDATE charts SET name = $1, chart_type = $2, position = $3, size = $4,
			data_ranges = $5, title = $6, subtitle = $7, legend = $8, axes = $9,
			series = $10, options = $11, updated_at = $12
		WHERE id = $13
	`, chart.Name, string(chart.ChartType), string(position), string(size),
		string(dataRanges), string(title), string(subtitle), string(legend),
		string(axes), string(series), string(options), chart.UpdatedAt, chart.ID)
	return err
}

// Delete deletes a chart.
func (s *ChartsStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM charts WHERE id = $1`, id)
	return err
}
