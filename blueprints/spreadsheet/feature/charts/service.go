package charts

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/go-mizu/blueprints/spreadsheet/pkg/ulid"
)

var (
	ErrNotFound       = errors.New("chart not found")
	ErrInvalidType    = errors.New("invalid chart type")
	ErrInvalidData    = errors.New("invalid data range")
	ErrEmptyDataRange = errors.New("data range is empty")
)

// DefaultColors provides default chart colors.
var DefaultColors = []string{
	"#4CAF50", // Green
	"#2196F3", // Blue
	"#FF9800", // Orange
	"#E91E63", // Pink
	"#9C27B0", // Purple
	"#00BCD4", // Cyan
	"#FFC107", // Amber
	"#795548", // Brown
	"#607D8B", // Blue Grey
	"#F44336", // Red
}

// Service implements the charts API.
type Service struct {
	store        Store
	cellProvider CellDataProvider
}

// NewService creates a new charts service.
func NewService(store Store, cellProvider CellDataProvider) *Service {
	return &Service{
		store:        store,
		cellProvider: cellProvider,
	}
}

// Create creates a new chart.
func (s *Service) Create(ctx context.Context, in *CreateIn) (*Chart, error) {
	if !isValidChartType(in.ChartType) {
		return nil, ErrInvalidType
	}

	if len(in.DataRanges) == 0 {
		return nil, ErrEmptyDataRange
	}

	now := time.Now()

	chart := &Chart{
		ID:         ulid.New(),
		SheetID:    in.SheetID,
		Name:       in.Name,
		ChartType:  in.ChartType,
		Position:   in.Position,
		Size:       in.Size,
		DataRanges: in.DataRanges,
		Title:      in.Title,
		Subtitle:   in.Subtitle,
		Legend:     in.Legend,
		Axes:       in.Axes,
		Series:     in.Series,
		Options:    in.Options,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// Apply defaults
	applyDefaults(chart)

	if err := s.store.Create(ctx, chart); err != nil {
		return nil, err
	}

	return chart, nil
}

// GetByID retrieves a chart by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Chart, error) {
	return s.store.GetByID(ctx, id)
}

// ListBySheet lists charts in a sheet.
func (s *Service) ListBySheet(ctx context.Context, sheetID string) ([]*Chart, error) {
	return s.store.ListBySheet(ctx, sheetID)
}

// Update updates a chart.
func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Chart, error) {
	chart, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Name != "" {
		chart.Name = in.Name
	}
	if in.ChartType != "" {
		if !isValidChartType(in.ChartType) {
			return nil, ErrInvalidType
		}
		chart.ChartType = in.ChartType
	}
	if in.Position != nil {
		chart.Position = *in.Position
	}
	if in.Size != nil {
		chart.Size = *in.Size
	}
	if len(in.DataRanges) > 0 {
		chart.DataRanges = in.DataRanges
	}
	if in.Title != nil {
		chart.Title = in.Title
	}
	if in.Subtitle != nil {
		chart.Subtitle = in.Subtitle
	}
	if in.Legend != nil {
		chart.Legend = in.Legend
	}
	if in.Axes != nil {
		chart.Axes = in.Axes
	}
	if len(in.Series) > 0 {
		chart.Series = in.Series
	}
	if in.Options != nil {
		chart.Options = in.Options
	}

	chart.UpdatedAt = time.Now()

	if err := s.store.Update(ctx, chart); err != nil {
		return nil, err
	}

	return chart, nil
}

// Delete deletes a chart.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// Duplicate duplicates a chart.
func (s *Service) Duplicate(ctx context.Context, id string) (*Chart, error) {
	original, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	chart := &Chart{
		ID:         ulid.New(),
		SheetID:    original.SheetID,
		Name:       original.Name + " (Copy)",
		ChartType:  original.ChartType,
		Position:   Position{Row: original.Position.Row + 2, Col: original.Position.Col + 2, OffsetX: 0, OffsetY: 0},
		Size:       original.Size,
		DataRanges: copyDataRanges(original.DataRanges),
		Title:      copyTitle(original.Title),
		Subtitle:   copyTitle(original.Subtitle),
		Legend:     copyLegend(original.Legend),
		Axes:       copyAxes(original.Axes),
		Series:     copySeries(original.Series),
		Options:    copyOptions(original.Options),
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.store.Create(ctx, chart); err != nil {
		return nil, err
	}

	return chart, nil
}

// GetData retrieves resolved chart data from cell values.
func (s *Service) GetData(ctx context.Context, id string) (*ChartData, error) {
	chart, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if len(chart.DataRanges) == 0 {
		return nil, ErrEmptyDataRange
	}

	// Get the primary data range
	dr := chart.DataRanges[0]
	sheetID := dr.SheetID
	if sheetID == "" {
		sheetID = chart.SheetID
	}

	// Fetch cell values
	values, err := s.cellProvider.GetCellValues(ctx, sheetID, dr.StartRow, dr.StartCol, dr.EndRow, dr.EndCol)
	if err != nil {
		return nil, err
	}

	if len(values) == 0 {
		return &ChartData{Labels: []string{}, Datasets: []Dataset{}}, nil
	}

	// Parse data based on chart type
	return parseChartData(chart, values, dr.HasHeader)
}

// parseChartData parses cell values into chart data format.
func parseChartData(chart *Chart, values [][]interface{}, hasHeader bool) (*ChartData, error) {
	if len(values) == 0 || len(values[0]) == 0 {
		return &ChartData{Labels: []string{}, Datasets: []Dataset{}}, nil
	}

	data := &ChartData{
		Labels:   []string{},
		Datasets: []Dataset{},
	}

	startRow := 0
	numCols := len(values[0])

	// Extract labels from first column (after header if present)
	if hasHeader {
		startRow = 1
	}

	// First column is typically labels
	for i := startRow; i < len(values); i++ {
		if len(values[i]) > 0 {
			data.Labels = append(data.Labels, toString(values[i][0]))
		}
	}

	// Get series names from header row or generate defaults
	seriesNames := make([]string, numCols-1)
	if hasHeader && len(values) > 0 {
		for j := 1; j < len(values[0]); j++ {
			seriesNames[j-1] = toString(values[0][j])
		}
	} else {
		for j := 0; j < numCols-1; j++ {
			seriesNames[j] = "Series " + strconv.Itoa(j+1)
		}
	}

	// Create datasets for each column (except the first which is labels)
	for j := 1; j < numCols; j++ {
		dataset := Dataset{
			Label:       seriesNames[j-1],
			Data:        make([]float64, 0),
			BorderWidth: 2,
		}

		// Assign colors
		colorIdx := (j - 1) % len(DefaultColors)
		dataset.BackgroundColor = DefaultColors[colorIdx]
		dataset.BorderColor = DefaultColors[colorIdx]

		// For area/line charts with fill
		if chart.ChartType == ChartTypeArea || chart.ChartType == ChartTypeStackedArea {
			dataset.Fill = true
			// Make background semi-transparent
			dataset.BackgroundColor = DefaultColors[colorIdx] + "80"
		}

		// Extract numeric data
		for i := startRow; i < len(values); i++ {
			if j < len(values[i]) {
				dataset.Data = append(dataset.Data, toFloat64(values[i][j]))
			} else {
				dataset.Data = append(dataset.Data, 0)
			}
		}

		// Apply series config if available
		if j-1 < len(chart.Series) {
			sc := chart.Series[j-1]
			if sc.Name != "" {
				dataset.Label = sc.Name
			}
			if sc.Color != "" {
				dataset.BorderColor = sc.Color
			}
			if sc.BackgroundColor != "" {
				dataset.BackgroundColor = sc.BackgroundColor
			}
			if sc.BorderWidth > 0 {
				dataset.BorderWidth = sc.BorderWidth
			}
			dataset.Fill = sc.Fill
			dataset.Tension = sc.Tension
			if sc.PointRadius > 0 {
				dataset.PointRadius = sc.PointRadius
			}
			if sc.PointStyle != "" {
				dataset.PointStyle = sc.PointStyle
			}
		}

		data.Datasets = append(data.Datasets, dataset)
	}

	// Handle pie/doughnut charts - flatten to single dataset with multiple colors
	if chart.ChartType == ChartTypePie || chart.ChartType == ChartTypeDoughnut {
		if len(data.Datasets) > 0 {
			colors := make([]string, len(data.Labels))
			for i := range colors {
				colors[i] = DefaultColors[i%len(DefaultColors)]
			}
			data.Datasets[0].BackgroundColor = colors
		}
	}

	return data, nil
}

// toString converts a value to string.
func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}

// toFloat64 converts a value to float64.
func toFloat64(v interface{}) float64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	default:
		return 0
	}
}

// isValidChartType checks if a chart type is valid.
func isValidChartType(t ChartType) bool {
	switch t {
	case ChartTypeLine, ChartTypeBar, ChartTypeColumn, ChartTypePie, ChartTypeDoughnut,
		ChartTypeArea, ChartTypeScatter, ChartTypeCombo, ChartTypeStackedBar,
		ChartTypeStackedColumn, ChartTypeStackedArea, ChartTypeRadar, ChartTypeBubble,
		ChartTypeWaterfall, ChartTypeHistogram, ChartTypeTreemap, ChartTypeGauge,
		ChartTypeCandlestick:
		return true
	}
	return false
}

// applyDefaults applies default values to a chart.
func applyDefaults(chart *Chart) {
	// Default size
	if chart.Size.Width == 0 {
		chart.Size.Width = 600
	}
	if chart.Size.Height == 0 {
		chart.Size.Height = 400
	}

	// Default legend
	if chart.Legend == nil {
		chart.Legend = &LegendConfig{
			Enabled:  true,
			Position: "bottom",
		}
	}

	// Default options
	if chart.Options == nil {
		chart.Options = &ChartOptions{
			Animated:       true,
			Interactive:    true,
			TooltipEnabled: true,
		}
	}
}

// Copy functions for deep copying chart components
func copyDataRanges(drs []DataRange) []DataRange {
	if drs == nil {
		return nil
	}
	result := make([]DataRange, len(drs))
	copy(result, drs)
	return result
}

func copyTitle(t *ChartTitle) *ChartTitle {
	if t == nil {
		return nil
	}
	c := *t
	return &c
}

func copyLegend(l *LegendConfig) *LegendConfig {
	if l == nil {
		return nil
	}
	c := *l
	return &c
}

func copyAxes(a *AxesConfig) *AxesConfig {
	if a == nil {
		return nil
	}
	c := &AxesConfig{}
	if a.XAxis != nil {
		x := *a.XAxis
		c.XAxis = &x
	}
	if a.YAxis != nil {
		y := *a.YAxis
		c.YAxis = &y
	}
	if a.Y2Axis != nil {
		y2 := *a.Y2Axis
		c.Y2Axis = &y2
	}
	return c
}

func copySeries(s []SeriesConfig) []SeriesConfig {
	if s == nil {
		return nil
	}
	result := make([]SeriesConfig, len(s))
	for i, sc := range s {
		result[i] = sc
		if sc.DataRange != nil {
			dr := *sc.DataRange
			result[i].DataRange = &dr
		}
		if sc.DataLabels != nil {
			dl := *sc.DataLabels
			result[i].DataLabels = &dl
		}
		if sc.Trendline != nil {
			tl := *sc.Trendline
			result[i].Trendline = &tl
		}
	}
	return result
}

func copyOptions(o *ChartOptions) *ChartOptions {
	if o == nil {
		return nil
	}
	c := *o
	return &c
}
