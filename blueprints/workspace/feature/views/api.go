// Package views provides database view management.
package views

import (
	"context"
	"time"

	"github.com/go-mizu/blueprints/workspace/feature/pages"
)

// ViewType represents the type of view.
type ViewType string

const (
	ViewTable    ViewType = "table"
	ViewBoard    ViewType = "board"
	ViewList     ViewType = "list"
	ViewCalendar ViewType = "calendar"
	ViewGallery  ViewType = "gallery"
	ViewTimeline ViewType = "timeline"
	ViewChart    ViewType = "chart"
)

// View represents a database view.
type View struct {
	ID         string      `json:"id"`
	DatabaseID string      `json:"database_id"`
	Name       string      `json:"name"`
	Type       ViewType    `json:"type"`
	Filter     *Filter     `json:"filter,omitempty"`
	Sorts      []Sort      `json:"sorts,omitempty"`
	Properties []ViewProp  `json:"properties,omitempty"`
	GroupBy    string      `json:"group_by,omitempty"`
	SubGroupBy string      `json:"sub_group_by,omitempty"`
	CalendarBy string      `json:"calendar_by,omitempty"`
	Config     *ViewConfig `json:"config,omitempty"`
	Position   int         `json:"position"`
	CreatedBy  string      `json:"created_by"`
	CreatedAt  time.Time   `json:"created_at"`
}

// ViewConfig holds view-specific configuration.
type ViewConfig struct {
	// Table view
	FrozenColumns int           `json:"frozen_columns,omitempty"`
	RowHeight     string        `json:"row_height,omitempty"` // small, medium, tall, extra_tall
	WrapCells     bool          `json:"wrap_cells,omitempty"`
	Calculations  []Calculation `json:"calculations,omitempty"`

	// Board view
	CardSize     string `json:"card_size,omitempty"` // small, medium, large
	CardPreview  string `json:"card_preview,omitempty"`
	ColorColumns bool   `json:"color_columns,omitempty"`

	// Timeline view
	TimeScale         string `json:"time_scale,omitempty"` // hours, days, weeks, months, quarters, years
	StartDateProperty string `json:"start_date_property,omitempty"`
	EndDateProperty   string `json:"end_date_property,omitempty"`
	ShowTablePanel    bool   `json:"show_table_panel,omitempty"`
	TablePanelWidth   int    `json:"table_panel_width,omitempty"`

	// Calendar view
	CalendarMode      string `json:"calendar_mode,omitempty"` // month, week
	StartWeekOnMonday bool   `json:"start_week_on_monday,omitempty"`

	// Gallery view
	GalleryCardSize string `json:"gallery_card_size,omitempty"` // small, medium, large
	PreviewSource   string `json:"preview_source,omitempty"`    // page_cover, page_content, files, none
	FilesPropertyID string `json:"files_property_id,omitempty"`
	FitImage        bool   `json:"fit_image,omitempty"`
	ShowTitle       bool   `json:"show_title,omitempty"`

	// Chart view
	ChartType  string       `json:"chart_type,omitempty"` // vertical_bar, horizontal_bar, line, donut
	ChartXAxis *AxisConfig  `json:"chart_x_axis,omitempty"`
	ChartYAxis *AxisConfig  `json:"chart_y_axis,omitempty"`
	ChartStyle *StyleConfig `json:"chart_style,omitempty"`
}

// Calculation represents a column calculation.
type Calculation struct {
	PropertyID string `json:"property_id"`
	Type       string `json:"type"` // count_all, sum, average, min, max, etc.
}

// AxisConfig holds chart axis configuration.
type AxisConfig struct {
	PropertyID     string   `json:"property_id"`
	Sort           string   `json:"sort,omitempty"`
	VisibleGroups  []string `json:"visible_groups,omitempty"`
	OmitZeroValues bool     `json:"omit_zero_values,omitempty"`
	Cumulative     bool     `json:"cumulative,omitempty"`
}

// StyleConfig holds chart style configuration.
type StyleConfig struct {
	Height          string `json:"height,omitempty"` // small, medium, large, extra_large
	GridLine        string `json:"grid_line,omitempty"`
	XAxisName       string `json:"x_axis_name,omitempty"`
	YAxisName       string `json:"y_axis_name,omitempty"`
	DataLabels      bool   `json:"data_labels,omitempty"`
	SmoothLine      bool   `json:"smooth_line,omitempty"`
	GradientArea    bool   `json:"gradient_area,omitempty"`
	ShowCenterValue bool   `json:"show_center_value,omitempty"`
	ShowLegend      bool   `json:"show_legend,omitempty"`
}

// ViewProp holds view-specific property configuration.
type ViewProp struct {
	PropertyID string `json:"property_id"`
	Visible    bool   `json:"visible"`
	Width      int    `json:"width,omitempty"`
}

// Filter represents a filter condition.
type Filter struct {
	And []Filter `json:"and,omitempty"`
	Or  []Filter `json:"or,omitempty"`

	PropertyID string      `json:"property_id,omitempty"`
	Operator   string      `json:"operator,omitempty"`
	Value      interface{} `json:"value,omitempty"`
}

// Sort represents a sort configuration.
type Sort struct {
	PropertyID string `json:"property_id"`
	Direction  string `json:"direction"` // asc, desc
}

// QueryResult holds the result of a view query.
type QueryResult struct {
	Items      []*pages.Page `json:"items"`
	NextCursor string        `json:"next_cursor,omitempty"`
	HasMore    bool          `json:"has_more"`
}

// CreateIn contains input for creating a view.
type CreateIn struct {
	DatabaseID string      `json:"database_id"`
	Name       string      `json:"name"`
	Type       ViewType    `json:"type"`
	Filter     *Filter     `json:"filter,omitempty"`
	Sorts      []Sort      `json:"sorts,omitempty"`
	GroupBy    string      `json:"group_by,omitempty"`
	SubGroupBy string      `json:"sub_group_by,omitempty"`
	CalendarBy string      `json:"calendar_by,omitempty"`
	Config     *ViewConfig `json:"config,omitempty"`
	CreatedBy  string      `json:"-"`
}

// UpdateIn contains input for updating a view.
type UpdateIn struct {
	Name       *string     `json:"name,omitempty"`
	Filter     *Filter     `json:"filter,omitempty"`
	Sorts      []Sort      `json:"sorts,omitempty"`
	Properties []ViewProp  `json:"properties,omitempty"`
	GroupBy    *string     `json:"group_by,omitempty"`
	SubGroupBy *string     `json:"sub_group_by,omitempty"`
	CalendarBy *string     `json:"calendar_by,omitempty"`
	Config     *ViewConfig `json:"config,omitempty"`
}

// API defines the views service contract.
type API interface {
	Create(ctx context.Context, in *CreateIn) (*View, error)
	GetByID(ctx context.Context, id string) (*View, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*View, error)
	Delete(ctx context.Context, id string) error

	ListByDatabase(ctx context.Context, databaseID string) ([]*View, error)
	Reorder(ctx context.Context, databaseID string, viewIDs []string) error
	Duplicate(ctx context.Context, id string, userID string) (*View, error)

	// Query
	Query(ctx context.Context, viewID string, cursor string, limit int) (*QueryResult, error)
}

// Store defines the data access contract for views.
type Store interface {
	Create(ctx context.Context, v *View) error
	GetByID(ctx context.Context, id string) (*View, error)
	Update(ctx context.Context, id string, in *UpdateIn) error
	Delete(ctx context.Context, id string) error
	ListByDatabase(ctx context.Context, databaseID string) ([]*View, error)
	Reorder(ctx context.Context, databaseID string, viewIDs []string) error
}
