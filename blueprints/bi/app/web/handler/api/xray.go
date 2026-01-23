package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/bi/drivers"
	"github.com/go-mizu/blueprints/bi/store"
	"github.com/go-mizu/blueprints/bi/store/sqlite"
)

// XRay handles X-ray API endpoints for automatic insights.
type XRay struct {
	store *sqlite.Store
}

// NewXRay creates a new XRay handler.
func NewXRay(store *sqlite.Store) *XRay {
	return &XRay{store: store}
}

// XRayResult represents the generated X-ray dashboard.
type XRayResult struct {
	Title       string          `json:"title"`
	Description string          `json:"description"`
	TableID     string          `json:"table_id"`
	TableName   string          `json:"table_name"`
	GeneratedAt time.Time       `json:"generated_at"`
	Cards       []XRayCard      `json:"cards"`
	Navigation  []XRayNavLink   `json:"navigation"`
	Stats       XRayTableStats  `json:"stats"`
}

// XRayCard represents a single insight card.
type XRayCard struct {
	ID            string                 `json:"id"`
	Title         string                 `json:"title"`
	Description   string                 `json:"description"`
	Visualization string                 `json:"visualization"`
	Query         string                 `json:"query"`
	Data          *store.QueryResult     `json:"data,omitempty"`
	Width         int                    `json:"width"`
	Height        int                    `json:"height"`
	Row           int                    `json:"row"`
	Col           int                    `json:"col"`
	Settings      map[string]interface{} `json:"settings,omitempty"`
}

// XRayNavLink represents a navigation link to related X-rays.
type XRayNavLink struct {
	Type       string `json:"type"` // zoom_in, zoom_out, related
	Label      string `json:"label"`
	TargetType string `json:"target_type"` // table, field
	TargetID   string `json:"target_id"`
}

// XRayTableStats contains overall table statistics.
type XRayTableStats struct {
	RowCount      int64  `json:"row_count"`
	ColumnCount   int    `json:"column_count"`
	NullableCount int    `json:"nullable_count"`
	LastUpdated   string `json:"last_updated,omitempty"`
}

// XRayTable generates an X-ray for a table.
func (h *XRay) XRayTable(c *mizu.Ctx) error {
	dsID := c.Param("datasourceId")
	tableID := c.Param("tableId")

	var req struct {
		IncludeData bool `json:"include_data"`
		Limit       int  `json:"limit"`
	}
	c.BindJSON(&req, 1<<20)
	if req.Limit == 0 {
		req.Limit = 10000
	}
	if !req.IncludeData {
		req.IncludeData = true // Default to including data
	}

	// Get data source
	ds, err := h.store.DataSources().GetByID(c.Request().Context(), dsID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	if ds == nil {
		return c.JSON(404, map[string]string{"error": "Data source not found"})
	}

	// Get table
	table, err := h.store.Tables().GetByID(c.Request().Context(), tableID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	if table == nil {
		return c.JSON(404, map[string]string{"error": "Table not found"})
	}

	// Get columns
	columns, err := h.store.Tables().ListColumns(c.Request().Context(), tableID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	// Connect to database
	config := dataSourceToConfig(ds)
	ctx, cancel := context.WithTimeout(c.Request().Context(), 2*time.Minute)
	defer cancel()

	driver, err := drivers.Open(ctx, config)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	defer driver.Close()

	// Generate X-ray
	xray := h.generateTableXRay(ctx, driver, ds, table, columns, req.IncludeData, req.Limit)

	return c.JSON(200, xray)
}

// generateTableXRay generates the X-ray result for a table.
func (h *XRay) generateTableXRay(
	ctx context.Context,
	driver drivers.Driver,
	_ *store.DataSource,
	table *store.Table,
	columns []*store.Column,
	includeData bool,
	_ int,
) *XRayResult {
	tableName := driver.QuoteIdentifier(table.Name)
	if table.Schema != "" && driver.SupportsSchemas() {
		tableName = driver.QuoteIdentifier(table.Schema) + "." + driver.QuoteIdentifier(table.Name)
	}

	// Categorize columns
	var dateColumns, numericColumns, categoryColumns, textColumns []*store.Column
	var nullableCount int

	for _, col := range columns {
		if col.Nullable {
			nullableCount++
		}

		switch col.Type {
		case "datetime", "date":
			dateColumns = append(dateColumns, col)
		case "number":
			numericColumns = append(numericColumns, col)
		default:
			// Check if it's a category (low cardinality)
			if col.DistinctCount > 0 && col.DistinctCount <= 20 {
				categoryColumns = append(categoryColumns, col)
			} else if col.Type == "string" {
				textColumns = append(textColumns, col)
			}
		}
	}

	xray := &XRayResult{
		Title:       fmt.Sprintf("X-ray: %s", table.DisplayName),
		Description: fmt.Sprintf("Automatic insights for the %s table", table.DisplayName),
		TableID:     table.ID,
		TableName:   table.Name,
		GeneratedAt: time.Now(),
		Cards:       []XRayCard{},
		Navigation:  []XRayNavLink{},
		Stats: XRayTableStats{
			RowCount:      table.RowCount,
			ColumnCount:   len(columns),
			NullableCount: nullableCount,
		},
	}

	// Card counter for positioning
	cardID := 0
	row := 0

	// 1. Overview Cards (Row 0)
	// Total row count
	cardID++
	rowCountCard := XRayCard{
		ID:            fmt.Sprintf("card-%d", cardID),
		Title:         "Total Records",
		Description:   "Count of all records in this table",
		Visualization: "number",
		Width:         4,
		Height:        2,
		Row:           row,
		Col:           0,
		Settings: map[string]interface{}{
			"compact": true,
		},
	}
	if includeData {
		query := fmt.Sprintf("SELECT COUNT(*) as count FROM %s", tableName)
		if result, err := driver.Execute(ctx, query); err == nil {
			rowCountCard.Data = convertResult(result)
		}
		rowCountCard.Query = query
	}
	xray.Cards = append(xray.Cards, rowCountCard)

	// Column count card
	cardID++
	colCountCard := XRayCard{
		ID:            fmt.Sprintf("card-%d", cardID),
		Title:         "Columns",
		Description:   "Number of columns in this table",
		Visualization: "number",
		Width:         4,
		Height:        2,
		Row:           row,
		Col:           4,
		Data: &store.QueryResult{
			Columns:  []store.ResultColumn{{Name: "count", DisplayName: "Columns", Type: "number"}},
			Rows:     []map[string]interface{}{{"count": len(columns)}},
			RowCount: 1,
		},
	}
	xray.Cards = append(xray.Cards, colCountCard)

	// If we have numeric columns, show sum of first one
	if len(numericColumns) > 0 {
		col := numericColumns[0]
		cardID++
		sumCard := XRayCard{
			ID:            fmt.Sprintf("card-%d", cardID),
			Title:         fmt.Sprintf("Total %s", col.DisplayName),
			Description:   fmt.Sprintf("Sum of %s", col.DisplayName),
			Visualization: "number",
			Width:         4,
			Height:        2,
			Row:           row,
			Col:           8,
			Settings: map[string]interface{}{
				"compact":  true,
				"decimals": 2,
			},
		}
		if includeData {
			query := fmt.Sprintf("SELECT SUM(%s) as total FROM %s", driver.QuoteIdentifier(col.Name), tableName)
			if result, err := driver.Execute(ctx, query); err == nil {
				sumCard.Data = convertResult(result)
			}
			sumCard.Query = query
		}
		xray.Cards = append(xray.Cards, sumCard)
	}

	// Avg of numeric column
	if len(numericColumns) > 0 {
		col := numericColumns[0]
		cardID++
		avgCard := XRayCard{
			ID:            fmt.Sprintf("card-%d", cardID),
			Title:         fmt.Sprintf("Average %s", col.DisplayName),
			Description:   fmt.Sprintf("Average of %s", col.DisplayName),
			Visualization: "number",
			Width:         3,
			Height:        2,
			Row:           row,
			Col:           12,
			Settings: map[string]interface{}{
				"compact":  true,
				"decimals": 2,
			},
		}
		if includeData {
			query := fmt.Sprintf("SELECT AVG(%s) as average FROM %s", driver.QuoteIdentifier(col.Name), tableName)
			if result, err := driver.Execute(ctx, query); err == nil {
				avgCard.Data = convertResult(result)
			}
			avgCard.Query = query
		}
		xray.Cards = append(xray.Cards, avgCard)
	}

	row += 2

	// 2. Time Series Analysis (if date columns exist)
	if len(dateColumns) > 0 {
		dateCol := dateColumns[0]

		// Records over time (full width line chart)
		cardID++
		timeSeriesCard := XRayCard{
			ID:            fmt.Sprintf("card-%d", cardID),
			Title:         fmt.Sprintf("Records by %s", dateCol.DisplayName),
			Description:   "Distribution of records over time",
			Visualization: "line",
			Width:         12,
			Height:        4,
			Row:           row,
			Col:           0,
			Settings: map[string]interface{}{
				"showPoints": false,
				"showTrend":  true,
			},
		}
		if includeData {
			// Group by date truncated to day/month depending on data range
			query := fmt.Sprintf(`
				SELECT DATE(%s) as date, COUNT(*) as count
				FROM %s
				WHERE %s IS NOT NULL
				GROUP BY DATE(%s)
				ORDER BY date
				LIMIT 100
			`, driver.QuoteIdentifier(dateCol.Name), tableName, driver.QuoteIdentifier(dateCol.Name), driver.QuoteIdentifier(dateCol.Name))
			if result, err := driver.Execute(ctx, query); err == nil {
				timeSeriesCard.Data = convertResult(result)
			}
			timeSeriesCard.Query = query
		}
		xray.Cards = append(xray.Cards, timeSeriesCard)

		// Records by day of week
		cardID++
		dowCard := XRayCard{
			ID:            fmt.Sprintf("card-%d", cardID),
			Title:         "Records by Day of Week",
			Description:   "Which days have the most activity",
			Visualization: "bar",
			Width:         6,
			Height:        4,
			Row:           row,
			Col:           12,
		}
		if includeData {
			// SQLite-specific day of week extraction
			query := fmt.Sprintf(`
				SELECT
					CASE CAST(strftime('%%w', %s) AS INTEGER)
						WHEN 0 THEN 'Sunday'
						WHEN 1 THEN 'Monday'
						WHEN 2 THEN 'Tuesday'
						WHEN 3 THEN 'Wednesday'
						WHEN 4 THEN 'Thursday'
						WHEN 5 THEN 'Friday'
						WHEN 6 THEN 'Saturday'
					END as day_of_week,
					COUNT(*) as count
				FROM %s
				WHERE %s IS NOT NULL
				GROUP BY strftime('%%w', %s)
				ORDER BY CAST(strftime('%%w', %s) AS INTEGER)
			`, driver.QuoteIdentifier(dateCol.Name), tableName, driver.QuoteIdentifier(dateCol.Name),
				driver.QuoteIdentifier(dateCol.Name), driver.QuoteIdentifier(dateCol.Name))
			if result, err := driver.Execute(ctx, query); err == nil {
				dowCard.Data = convertResult(result)
			}
			dowCard.Query = query
		}
		xray.Cards = append(xray.Cards, dowCard)

		row += 4
	}

	// 3. Category Analysis
	for i, catCol := range categoryColumns {
		if i >= 3 { // Limit to 3 category columns
			break
		}

		cardID++
		catCard := XRayCard{
			ID:            fmt.Sprintf("card-%d", cardID),
			Title:         fmt.Sprintf("Distribution by %s", catCol.DisplayName),
			Description:   fmt.Sprintf("Records grouped by %s", catCol.DisplayName),
			Visualization: "row",
			Width:         6,
			Height:        4,
			Row:           row,
			Col:           (i % 3) * 6,
		}
		if includeData {
			query := fmt.Sprintf(`
				SELECT %s as category, COUNT(*) as count
				FROM %s
				WHERE %s IS NOT NULL
				GROUP BY %s
				ORDER BY count DESC
				LIMIT 10
			`, driver.QuoteIdentifier(catCol.Name), tableName, driver.QuoteIdentifier(catCol.Name), driver.QuoteIdentifier(catCol.Name))
			if result, err := driver.Execute(ctx, query); err == nil {
				catCard.Data = convertResult(result)
			}
			catCard.Query = query
		}
		xray.Cards = append(xray.Cards, catCard)

		if (i+1)%3 == 0 || i == len(categoryColumns)-1 {
			row += 4
		}
	}

	// 4. Numeric Distribution
	for i, numCol := range numericColumns {
		if i >= 2 { // Limit to 2 numeric columns
			break
		}

		// Histogram (binned distribution)
		cardID++
		histCard := XRayCard{
			ID:            fmt.Sprintf("card-%d", cardID),
			Title:         fmt.Sprintf("%s Distribution", numCol.DisplayName),
			Description:   "Value distribution histogram",
			Visualization: "bar",
			Width:         9,
			Height:        4,
			Row:           row,
			Col:           0,
		}
		if includeData && numCol.MinValue != "" && numCol.MaxValue != "" {
			// Create binned histogram
			query := fmt.Sprintf(`
				SELECT
					CAST(%s AS INTEGER) as value,
					COUNT(*) as count
				FROM %s
				WHERE %s IS NOT NULL
				GROUP BY CAST(%s AS INTEGER)
				ORDER BY value
				LIMIT 20
			`, driver.QuoteIdentifier(numCol.Name), tableName, driver.QuoteIdentifier(numCol.Name), driver.QuoteIdentifier(numCol.Name))
			if result, err := driver.Execute(ctx, query); err == nil {
				histCard.Data = convertResult(result)
			}
			histCard.Query = query
		}
		xray.Cards = append(xray.Cards, histCard)

		// Stats summary for this column
		cardID++
		statsCard := XRayCard{
			ID:            fmt.Sprintf("card-%d", cardID),
			Title:         fmt.Sprintf("%s Statistics", numCol.DisplayName),
			Description:   "Summary statistics",
			Visualization: "table",
			Width:         9,
			Height:        4,
			Row:           row,
			Col:           9,
		}
		if includeData {
			query := fmt.Sprintf(`
				SELECT
					MIN(%s) as minimum,
					MAX(%s) as maximum,
					AVG(%s) as average,
					COUNT(*) as count
				FROM %s
				WHERE %s IS NOT NULL
			`, driver.QuoteIdentifier(numCol.Name), driver.QuoteIdentifier(numCol.Name),
				driver.QuoteIdentifier(numCol.Name), tableName, driver.QuoteIdentifier(numCol.Name))
			if result, err := driver.Execute(ctx, query); err == nil {
				statsCard.Data = convertResult(result)
			}
			statsCard.Query = query
		}
		xray.Cards = append(xray.Cards, statsCard)

		row += 4
	}

	// 5. Sample Data (always at the end)
	cardID++
	sampleCard := XRayCard{
		ID:            fmt.Sprintf("card-%d", cardID),
		Title:         "Sample Data",
		Description:   "First 10 records from the table",
		Visualization: "table",
		Width:         18,
		Height:        5,
		Row:           row,
		Col:           0,
	}
	if includeData {
		query := fmt.Sprintf("SELECT * FROM %s LIMIT 10", tableName)
		if result, err := driver.Execute(ctx, query); err == nil {
			sampleCard.Data = convertResult(result)
		}
		sampleCard.Query = query
	}
	xray.Cards = append(xray.Cards, sampleCard)

	// Navigation links
	// Zoom in links for each column
	for _, col := range columns {
		xray.Navigation = append(xray.Navigation, XRayNavLink{
			Type:       "zoom_in",
			Label:      fmt.Sprintf("X-ray %s", col.DisplayName),
			TargetType: "field",
			TargetID:   col.ID,
		})
	}

	// Related tables via foreign keys
	for _, col := range columns {
		if col.ForeignKey && col.ForeignTable != "" {
			xray.Navigation = append(xray.Navigation, XRayNavLink{
				Type:       "related",
				Label:      fmt.Sprintf("Explore %s", col.ForeignTable),
				TargetType: "table",
				TargetID:   col.ForeignTable,
			})
		}
	}

	return xray
}

// XRayField generates a detailed X-ray for a specific field.
func (h *XRay) XRayField(c *mizu.Ctx) error {
	dsID := c.Param("datasourceId")
	columnID := c.Param("columnId")

	var req struct {
		IncludeData bool `json:"include_data"`
	}
	c.BindJSON(&req, 1<<20)
	req.IncludeData = true

	// Get data source
	ds, err := h.store.DataSources().GetByID(c.Request().Context(), dsID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	if ds == nil {
		return c.JSON(404, map[string]string{"error": "Data source not found"})
	}

	// Get column
	col, err := h.store.Tables().GetColumn(c.Request().Context(), columnID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	if col == nil {
		return c.JSON(404, map[string]string{"error": "Column not found"})
	}

	// Get table
	table, err := h.store.Tables().GetByID(c.Request().Context(), col.TableID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	// Connect to database
	config := dataSourceToConfig(ds)
	ctx, cancel := context.WithTimeout(c.Request().Context(), 2*time.Minute)
	defer cancel()

	driver, err := drivers.Open(ctx, config)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	defer driver.Close()

	// Generate field X-ray
	xray := h.generateFieldXRay(ctx, driver, table, col, req.IncludeData)

	return c.JSON(200, xray)
}

// generateFieldXRay generates detailed insights for a single field.
func (h *XRay) generateFieldXRay(
	ctx context.Context,
	driver drivers.Driver,
	table *store.Table,
	col *store.Column,
	includeData bool,
) *XRayResult {
	tableName := driver.QuoteIdentifier(table.Name)
	if table.Schema != "" && driver.SupportsSchemas() {
		tableName = driver.QuoteIdentifier(table.Schema) + "." + driver.QuoteIdentifier(table.Name)
	}
	colName := driver.QuoteIdentifier(col.Name)

	xray := &XRayResult{
		Title:       fmt.Sprintf("X-ray: %s.%s", table.DisplayName, col.DisplayName),
		Description: fmt.Sprintf("Detailed analysis of the %s column", col.DisplayName),
		TableID:     table.ID,
		TableName:   table.Name,
		GeneratedAt: time.Now(),
		Cards:       []XRayCard{},
		Navigation: []XRayNavLink{
			{
				Type:       "zoom_out",
				Label:      fmt.Sprintf("Back to %s", table.DisplayName),
				TargetType: "table",
				TargetID:   table.ID,
			},
		},
	}

	cardID := 0
	row := 0

	// Stats cards
	cardID++
	distinctCard := XRayCard{
		ID:            fmt.Sprintf("card-%d", cardID),
		Title:         "Distinct Values",
		Visualization: "number",
		Width:         4,
		Height:        2,
		Row:           row,
		Col:           0,
	}
	if includeData {
		query := fmt.Sprintf("SELECT COUNT(DISTINCT %s) as distinct_count FROM %s", colName, tableName)
		if result, err := driver.Execute(ctx, query); err == nil {
			distinctCard.Data = convertResult(result)
		}
	}
	xray.Cards = append(xray.Cards, distinctCard)

	cardID++
	nullCard := XRayCard{
		ID:            fmt.Sprintf("card-%d", cardID),
		Title:         "Null Values",
		Visualization: "number",
		Width:         4,
		Height:        2,
		Row:           row,
		Col:           4,
	}
	if includeData {
		query := fmt.Sprintf("SELECT COUNT(*) as null_count FROM %s WHERE %s IS NULL", tableName, colName)
		if result, err := driver.Execute(ctx, query); err == nil {
			nullCard.Data = convertResult(result)
		}
	}
	xray.Cards = append(xray.Cards, nullCard)

	cardID++
	nonNullCard := XRayCard{
		ID:            fmt.Sprintf("card-%d", cardID),
		Title:         "Non-Null Values",
		Visualization: "number",
		Width:         4,
		Height:        2,
		Row:           row,
		Col:           8,
	}
	if includeData {
		query := fmt.Sprintf("SELECT COUNT(*) as non_null_count FROM %s WHERE %s IS NOT NULL", tableName, colName)
		if result, err := driver.Execute(ctx, query); err == nil {
			nonNullCard.Data = convertResult(result)
		}
	}
	xray.Cards = append(xray.Cards, nonNullCard)

	row += 2

	// Type-specific analysis
	switch col.Type {
	case "number":
		// Numeric statistics
		cardID++
		statsCard := XRayCard{
			ID:            fmt.Sprintf("card-%d", cardID),
			Title:         "Statistics",
			Visualization: "table",
			Width:         18,
			Height:        3,
			Row:           row,
			Col:           0,
		}
		if includeData {
			query := fmt.Sprintf(`
				SELECT
					MIN(%s) as min,
					MAX(%s) as max,
					AVG(%s) as avg,
					SUM(%s) as total
				FROM %s WHERE %s IS NOT NULL
			`, colName, colName, colName, colName, tableName, colName)
			if result, err := driver.Execute(ctx, query); err == nil {
				statsCard.Data = convertResult(result)
			}
		}
		xray.Cards = append(xray.Cards, statsCard)
		row += 3

		// Distribution histogram
		cardID++
		histCard := XRayCard{
			ID:            fmt.Sprintf("card-%d", cardID),
			Title:         "Value Distribution",
			Visualization: "bar",
			Width:         18,
			Height:        5,
			Row:           row,
			Col:           0,
		}
		if includeData {
			query := fmt.Sprintf(`
				SELECT CAST(%s AS INTEGER) as value, COUNT(*) as count
				FROM %s WHERE %s IS NOT NULL
				GROUP BY CAST(%s AS INTEGER)
				ORDER BY value
				LIMIT 30
			`, colName, tableName, colName, colName)
			if result, err := driver.Execute(ctx, query); err == nil {
				histCard.Data = convertResult(result)
			}
		}
		xray.Cards = append(xray.Cards, histCard)
		row += 5

	case "datetime", "date":
		// Time range
		cardID++
		rangeCard := XRayCard{
			ID:            fmt.Sprintf("card-%d", cardID),
			Title:         "Date Range",
			Visualization: "table",
			Width:         18,
			Height:        2,
			Row:           row,
			Col:           0,
		}
		if includeData {
			query := fmt.Sprintf(`
				SELECT MIN(%s) as earliest, MAX(%s) as latest
				FROM %s WHERE %s IS NOT NULL
			`, colName, colName, tableName, colName)
			if result, err := driver.Execute(ctx, query); err == nil {
				rangeCard.Data = convertResult(result)
			}
		}
		xray.Cards = append(xray.Cards, rangeCard)
		row += 2

		// Records over time
		cardID++
		timeCard := XRayCard{
			ID:            fmt.Sprintf("card-%d", cardID),
			Title:         "Records Over Time",
			Visualization: "line",
			Width:         18,
			Height:        5,
			Row:           row,
			Col:           0,
		}
		if includeData {
			query := fmt.Sprintf(`
				SELECT DATE(%s) as date, COUNT(*) as count
				FROM %s WHERE %s IS NOT NULL
				GROUP BY DATE(%s)
				ORDER BY date
				LIMIT 100
			`, colName, tableName, colName, colName)
			if result, err := driver.Execute(ctx, query); err == nil {
				timeCard.Data = convertResult(result)
			}
		}
		xray.Cards = append(xray.Cards, timeCard)
		row += 5

	default:
		// Text/category - value distribution
		cardID++
		distCard := XRayCard{
			ID:            fmt.Sprintf("card-%d", cardID),
			Title:         "Value Distribution",
			Visualization: "row",
			Width:         9,
			Height:        6,
			Row:           row,
			Col:           0,
		}
		if includeData {
			query := fmt.Sprintf(`
				SELECT %s as value, COUNT(*) as count
				FROM %s WHERE %s IS NOT NULL
				GROUP BY %s
				ORDER BY count DESC
				LIMIT 15
			`, colName, tableName, colName, colName)
			if result, err := driver.Execute(ctx, query); err == nil {
				distCard.Data = convertResult(result)
			}
		}
		xray.Cards = append(xray.Cards, distCard)

		// Pie chart
		cardID++
		pieCard := XRayCard{
			ID:            fmt.Sprintf("card-%d", cardID),
			Title:         "Proportions",
			Visualization: "donut",
			Width:         9,
			Height:        6,
			Row:           row,
			Col:           9,
		}
		if includeData {
			query := fmt.Sprintf(`
				SELECT %s as category, COUNT(*) as count
				FROM %s WHERE %s IS NOT NULL
				GROUP BY %s
				ORDER BY count DESC
				LIMIT 10
			`, colName, tableName, colName, colName)
			if result, err := driver.Execute(ctx, query); err == nil {
				pieCard.Data = convertResult(result)
			}
		}
		xray.Cards = append(xray.Cards, pieCard)
		row += 6
	}

	// Sample values
	cardID++
	sampleCard := XRayCard{
		ID:            fmt.Sprintf("card-%d", cardID),
		Title:         "Sample Values",
		Visualization: "table",
		Width:         18,
		Height:        4,
		Row:           row,
		Col:           0,
	}
	if includeData {
		query := fmt.Sprintf("SELECT DISTINCT %s as value FROM %s WHERE %s IS NOT NULL LIMIT 20", colName, tableName, colName)
		if result, err := driver.Execute(ctx, query); err == nil {
			sampleCard.Data = convertResult(result)
		}
	}
	xray.Cards = append(xray.Cards, sampleCard)

	return xray
}

// SaveXRay saves an X-ray as a dashboard.
func (h *XRay) SaveXRay(c *mizu.Ctx) error {
	var req struct {
		XRay         XRayResult `json:"xray"`
		Name         string     `json:"name"`
		CollectionID string     `json:"collection_id"`
	}
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request body"})
	}

	if req.Name == "" {
		req.Name = req.XRay.Title
	}

	// Create dashboard
	dashboard := &store.Dashboard{
		Name:        req.Name,
		Description: req.XRay.Description,
	}

	if err := h.store.Dashboards().Create(c.Request().Context(), dashboard); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	// Create cards - for X-ray we save the queries as text cards with visualization hints
	// In a full implementation, you'd create saved questions for each card
	for _, card := range req.XRay.Cards {
		dashCard := &store.DashboardCard{
			DashboardID: dashboard.ID,
			CardType:    "text",
			Row:         card.Row,
			Col:         card.Col,
			Width:       card.Width,
			Height:      card.Height,
			Settings: map[string]interface{}{
				"title":         card.Title,
				"description":   card.Description,
				"visualization": card.Visualization,
				"query":         card.Query,
				"xray_card_id":  card.ID,
			},
		}
		h.store.Dashboards().CreateCard(c.Request().Context(), dashCard)
	}

	return c.JSON(201, map[string]interface{}{
		"dashboard_id": dashboard.ID,
		"name":         dashboard.Name,
		"message":      "X-ray saved as dashboard",
	})
}

// Helper to convert driver result to store result
func convertResult(result *drivers.QueryResult) *store.QueryResult {
	if result == nil {
		return nil
	}

	cols := make([]store.ResultColumn, len(result.Columns))
	for i, c := range result.Columns {
		cols[i] = store.ResultColumn{
			Name:        c.Name,
			DisplayName: c.DisplayName,
			Type:        c.MappedType,
		}
	}

	return &store.QueryResult{
		Columns:  cols,
		Rows:     result.Rows,
		RowCount: int64(len(result.Rows)),
	}
}

// XRayCompare generates a comparison X-ray between two segments.
func (h *XRay) XRayCompare(c *mizu.Ctx) error {
	dsID := c.Param("datasourceId")
	tableID := c.Param("tableId")

	var req struct {
		Column       string `json:"column"`
		Value1       string `json:"value1"`
		Value2       string `json:"value2"`
		IncludeData  bool   `json:"include_data"`
	}
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request body"})
	}

	// Get data source
	ds, err := h.store.DataSources().GetByID(c.Request().Context(), dsID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	if ds == nil {
		return c.JSON(404, map[string]string{"error": "Data source not found"})
	}

	// Get table
	table, err := h.store.Tables().GetByID(c.Request().Context(), tableID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	if table == nil {
		return c.JSON(404, map[string]string{"error": "Table not found"})
	}

	// Get columns
	columns, err := h.store.Tables().ListColumns(c.Request().Context(), tableID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	// Connect to database
	config := dataSourceToConfig(ds)
	ctx, cancel := context.WithTimeout(c.Request().Context(), 2*time.Minute)
	defer cancel()

	driver, err := drivers.Open(ctx, config)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	defer driver.Close()

	tableName := driver.QuoteIdentifier(table.Name)
	if table.Schema != "" && driver.SupportsSchemas() {
		tableName = driver.QuoteIdentifier(table.Schema) + "." + driver.QuoteIdentifier(table.Name)
	}
	filterCol := driver.QuoteIdentifier(req.Column)

	xray := &XRayResult{
		Title:       fmt.Sprintf("Comparing %s: %s vs %s", req.Column, req.Value1, req.Value2),
		Description: "Side-by-side comparison of two segments",
		TableID:     table.ID,
		TableName:   table.Name,
		GeneratedAt: time.Now(),
		Cards:       []XRayCard{},
	}

	cardID := 0
	row := 0

	// Count comparison
	cardID++
	card1 := XRayCard{
		ID:            fmt.Sprintf("card-%d", cardID),
		Title:         fmt.Sprintf("%s = %s", req.Column, req.Value1),
		Visualization: "number",
		Width:         6,
		Height:        2,
		Row:           row,
		Col:           0,
	}
	query1 := fmt.Sprintf("SELECT COUNT(*) as count FROM %s WHERE %s = ?", tableName, filterCol)
	if result, err := driver.Execute(ctx, query1, req.Value1); err == nil {
		card1.Data = convertResult(result)
	}
	xray.Cards = append(xray.Cards, card1)

	cardID++
	card2 := XRayCard{
		ID:            fmt.Sprintf("card-%d", cardID),
		Title:         fmt.Sprintf("%s = %s", req.Column, req.Value2),
		Visualization: "number",
		Width:         6,
		Height:        2,
		Row:           row,
		Col:           6,
	}
	query2 := fmt.Sprintf("SELECT COUNT(*) as count FROM %s WHERE %s = ?", tableName, filterCol)
	if result, err := driver.Execute(ctx, query2, req.Value2); err == nil {
		card2.Data = convertResult(result)
	}
	xray.Cards = append(xray.Cards, card2)

	row += 2

	// Compare numeric columns
	for _, col := range columns {
		if col.Type == "number" && !strings.EqualFold(col.Name, req.Column) {
			cardID++
			compCard := XRayCard{
				ID:            fmt.Sprintf("card-%d", cardID),
				Title:         fmt.Sprintf("%s Comparison", col.DisplayName),
				Visualization: "bar",
				Width:         18,
				Height:        4,
				Row:           row,
				Col:           0,
			}

			colName := driver.QuoteIdentifier(col.Name)
			query := fmt.Sprintf(`
				SELECT
					%s as segment,
					AVG(%s) as average,
					SUM(%s) as total,
					COUNT(*) as count
				FROM %s
				WHERE %s IN (?, ?)
				GROUP BY %s
			`, filterCol, colName, colName, tableName, filterCol, filterCol)
			if result, err := driver.Execute(ctx, query, req.Value1, req.Value2); err == nil {
				compCard.Data = convertResult(result)
			}
			xray.Cards = append(xray.Cards, compCard)
			row += 4
			break // Just show first numeric comparison
		}
	}

	return c.JSON(200, xray)
}
