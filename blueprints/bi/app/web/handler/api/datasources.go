package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/bi/drivers"
	_ "github.com/go-mizu/blueprints/bi/drivers/mysql"
	_ "github.com/go-mizu/blueprints/bi/drivers/postgres"
	_ "github.com/go-mizu/blueprints/bi/drivers/sqlite"
	"github.com/go-mizu/blueprints/bi/store"
	"github.com/go-mizu/blueprints/bi/store/sqlite"
)

// DataSources handles data source API endpoints.
type DataSources struct {
	store *sqlite.Store
}

// NewDataSources creates a new DataSources handler.
func NewDataSources(store *sqlite.Store) *DataSources {
	return &DataSources{store: store}
}

// List returns all data sources.
func (h *DataSources) List(c *mizu.Ctx) error {
	sources, err := h.store.DataSources().List(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, sources)
}

// Create creates a new data source.
func (h *DataSources) Create(c *mizu.Ctx) error {
	var ds store.DataSource
	if err := c.BindJSON(&ds, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request body"})
	}

	if err := h.store.DataSources().Create(c.Request().Context(), &ds); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(201, ds)
}

// Get returns a data source by ID.
func (h *DataSources) Get(c *mizu.Ctx) error {
	id := c.Param("id")
	ds, err := h.store.DataSources().GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	if ds == nil {
		return c.JSON(404, map[string]string{"error": "Data source not found"})
	}
	return c.JSON(200, ds)
}

// Update updates a data source.
func (h *DataSources) Update(c *mizu.Ctx) error {
	id := c.Param("id")
	var ds store.DataSource
	if err := c.BindJSON(&ds, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request body"})
	}
	ds.ID = id

	if err := h.store.DataSources().Update(c.Request().Context(), &ds); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, ds)
}

// Delete deletes a data source.
func (h *DataSources) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.store.DataSources().Delete(c.Request().Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]string{"status": "deleted"})
}

// Test tests the connection to an existing data source.
func (h *DataSources) Test(c *mizu.Ctx) error {
	id := c.Param("id")
	ds, err := h.store.DataSources().GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	if ds == nil {
		return c.JSON(404, map[string]string{"error": "Data source not found"})
	}

	// Create driver config from data source
	config := dataSourceToConfig(ds)

	// Test connection with timeout
	ctx, cancel := context.WithTimeout(c.Request().Context(), 30*time.Second)
	defer cancel()

	start := time.Now()
	driver, err := drivers.Open(ctx, config)
	if err != nil {
		return c.JSON(200, map[string]any{
			"valid":       false,
			"error":       err.Error(),
			"error_code":  categorizeError(err),
			"suggestions": getSuggestions(err),
		})
	}
	defer driver.Close()

	latency := time.Since(start).Milliseconds()

	// Get version if available
	var version string
	if versioner, ok := driver.(interface{ Version(context.Context) (string, error) }); ok {
		version, _ = versioner.Version(ctx)
	}

	// Get schemas
	schemas, _ := driver.ListSchemas(ctx)

	return c.JSON(200, map[string]any{
		"valid":      true,
		"version":    version,
		"schemas":    schemas,
		"latency_ms": latency,
	})
}

// TestConnection tests a connection before creating a datasource (validate endpoint).
func (h *DataSources) TestConnection(c *mizu.Ctx) error {
	var req struct {
		Engine        string            `json:"engine"`
		Host          string            `json:"host"`
		Port          int               `json:"port"`
		Database      string            `json:"database"`
		Username      string            `json:"username"`
		Password      string            `json:"password"`
		SSL           bool              `json:"ssl"`
		SSLMode       string            `json:"ssl_mode"`
		SSLRootCert   string            `json:"ssl_root_cert"`
		SSLClientCert string            `json:"ssl_client_cert"`
		SSLClientKey  string            `json:"ssl_client_key"`
		Options       map[string]string `json:"options"`
	}
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request body"})
	}

	config := drivers.Config{
		Engine:        req.Engine,
		Host:          req.Host,
		Port:          req.Port,
		Database:      req.Database,
		Username:      req.Username,
		Password:      req.Password,
		SSL:           req.SSL,
		SSLMode:       req.SSLMode,
		SSLRootCert:   req.SSLRootCert,
		SSLClientCert: req.SSLClientCert,
		SSLClientKey:  req.SSLClientKey,
		Options:       req.Options,
	}

	// Test connection with timeout
	ctx, cancel := context.WithTimeout(c.Request().Context(), 30*time.Second)
	defer cancel()

	start := time.Now()
	driver, err := drivers.Open(ctx, config)
	if err != nil {
		return c.JSON(200, map[string]any{
			"valid":       false,
			"error":       err.Error(),
			"error_code":  categorizeError(err),
			"suggestions": getSuggestions(err),
		})
	}
	defer driver.Close()

	latency := time.Since(start).Milliseconds()

	// Get version if available
	var version string
	if versioner, ok := driver.(interface{ Version(context.Context) (string, error) }); ok {
		version, _ = versioner.Version(ctx)
	}

	// Get schemas
	schemas, _ := driver.ListSchemas(ctx)

	return c.JSON(200, map[string]any{
		"valid":      true,
		"version":    version,
		"schemas":    schemas,
		"latency_ms": latency,
	})
}

// Sync syncs metadata from a data source.
func (h *DataSources) Sync(c *mizu.Ctx) error {
	id := c.Param("id")
	ds, err := h.store.DataSources().GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	if ds == nil {
		return c.JSON(404, map[string]string{"error": "Data source not found"})
	}

	// Parse request body for sync options
	var req struct {
		FullSync        bool `json:"full_sync"`
		ScanFieldValues bool `json:"scan_field_values"`
	}
	c.BindJSON(&req, 1<<20) // Ignore error, use defaults

	// Create driver config
	config := dataSourceToConfig(ds)

	// Open connection
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Minute)
	defer cancel()

	start := time.Now()
	driver, err := drivers.Open(ctx, config)
	if err != nil {
		// Update sync status
		now := time.Now()
		ds.LastSyncAt = &now
		ds.LastSyncStatus = "failed"
		ds.LastSyncError = err.Error()
		h.store.DataSources().Update(ctx, ds)

		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	defer driver.Close()

	var tablesSynced, columnsSynced int
	var syncErrors []string

	// Get schemas
	schemas, err := driver.ListSchemas(ctx)
	if err != nil {
		schemas = []string{""} // Use empty schema for schema-less DBs
	}

	// Apply schema filter
	if ds.SchemaFilterType == "inclusion" && len(ds.SchemaFilterPatterns) > 0 {
		schemas = filterSchemas(schemas, ds.SchemaFilterPatterns, true)
	} else if ds.SchemaFilterType == "exclusion" && len(ds.SchemaFilterPatterns) > 0 {
		schemas = filterSchemas(schemas, ds.SchemaFilterPatterns, false)
	}

	// Sync tables for each schema
	for _, schema := range schemas {
		tables, err := driver.ListTables(ctx, schema)
		if err != nil {
			syncErrors = append(syncErrors, fmt.Sprintf("schema %s: %v", schema, err))
			continue
		}

		for _, t := range tables {
			table := &store.Table{
				DataSourceID: ds.ID,
				Schema:       t.Schema,
				Name:         t.Name,
				DisplayName:  t.Name,
				Description:  t.Description,
				RowCount:     t.RowCount,
			}

			if err := h.store.Tables().Create(ctx, table); err != nil {
				// Try to update existing table
				continue
			}
			tablesSynced++

			// Sync columns for this table
			columns, err := driver.ListColumns(ctx, schema, t.Name)
			if err != nil {
				syncErrors = append(syncErrors, fmt.Sprintf("table %s.%s columns: %v", schema, t.Name, err))
				continue
			}

			for _, col := range columns {
				storeCol := &store.Column{
					TableID:       table.ID,
					Name:          col.Name,
					DisplayName:   col.Name,
					Type:          col.Type,
					MappedType:    col.MappedType,
					Description:   col.Description,
					Position:      col.Position,
					Nullable:      col.Nullable,
					PrimaryKey:    col.PrimaryKey,
					ForeignKey:    col.ForeignKey,
					Visibility:    "everywhere",
				}

				// Infer semantic type from column name and type
				storeCol.Semantic = inferSemanticType(col.Name, col.MappedType, col.PrimaryKey, col.ForeignKey)

				if err := h.store.Tables().CreateColumn(ctx, storeCol); err != nil {
					continue
				}
				columnsSynced++
			}
		}
	}

	// Update sync status
	now := time.Now()
	ds.LastSyncAt = &now
	if len(syncErrors) == 0 {
		ds.LastSyncStatus = "success"
		ds.LastSyncError = ""
	} else {
		ds.LastSyncStatus = "partial"
		ds.LastSyncError = strings.Join(syncErrors, "; ")
	}
	h.store.DataSources().Update(ctx, ds)

	return c.JSON(200, map[string]any{
		"status":          ds.LastSyncStatus,
		"duration_ms":     time.Since(start).Milliseconds(),
		"schemas_synced":  len(schemas),
		"tables_synced":   tablesSynced,
		"columns_synced":  columnsSynced,
		"errors":          syncErrors,
	})
}

// ListTables returns tables for a data source.
func (h *DataSources) ListTables(c *mizu.Ctx) error {
	id := c.Param("id")
	tables, err := h.store.Tables().ListByDataSource(c.Request().Context(), id)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, tables)
}

// ListColumns returns columns for a table.
func (h *DataSources) ListColumns(c *mizu.Ctx) error {
	tableID := c.Param("table")
	columns, err := h.store.Tables().ListColumns(c.Request().Context(), tableID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, columns)
}

// UpdateColumn updates metadata for a column.
func (h *DataSources) UpdateColumn(c *mizu.Ctx) error {
	columnID := c.Param("columnId")
	var update struct {
		DisplayName string `json:"display_name"`
		Description string `json:"description"`
		Semantic    string `json:"semantic"`
		Visibility  string `json:"visibility"`
	}
	if err := c.BindJSON(&update, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request body"})
	}

	col, err := h.store.Tables().GetColumn(c.Request().Context(), columnID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	if col == nil {
		return c.JSON(404, map[string]string{"error": "Column not found"})
	}

	if update.DisplayName != "" {
		col.DisplayName = update.DisplayName
	}
	if update.Description != "" {
		col.Description = update.Description
	}
	if update.Semantic != "" {
		col.Semantic = update.Semantic
	}
	if update.Visibility != "" {
		col.Visibility = update.Visibility
	}

	if err := h.store.Tables().UpdateColumn(c.Request().Context(), col); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, col)
}

// ListSchemas returns schemas for a data source.
func (h *DataSources) ListSchemas(c *mizu.Ctx) error {
	id := c.Param("id")
	ds, err := h.store.DataSources().GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	if ds == nil {
		return c.JSON(404, map[string]string{"error": "Data source not found"})
	}

	config := dataSourceToConfig(ds)
	ctx, cancel := context.WithTimeout(c.Request().Context(), 30*time.Second)
	defer cancel()

	driver, err := drivers.Open(ctx, config)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	defer driver.Close()

	schemas, err := driver.ListSchemas(ctx)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, schemas)
}

// GetStatus returns the connection status for a data source.
func (h *DataSources) GetStatus(c *mizu.Ctx) error {
	id := c.Param("id")
	ds, err := h.store.DataSources().GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	if ds == nil {
		return c.JSON(404, map[string]string{"error": "Data source not found"})
	}

	config := dataSourceToConfig(ds)
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	status := map[string]any{
		"last_sync_at":     ds.LastSyncAt,
		"last_sync_status": ds.LastSyncStatus,
		"last_sync_error":  ds.LastSyncError,
	}

	// Quick connection check
	start := time.Now()
	driver, err := drivers.Open(ctx, config)
	if err != nil {
		status["connected"] = false
		status["error"] = err.Error()
		return c.JSON(200, status)
	}
	defer driver.Close()

	status["connected"] = true
	status["latency_ms"] = time.Since(start).Milliseconds()

	// Get capabilities
	status["capabilities"] = driver.Capabilities()

	return c.JSON(200, status)
}

// Helper functions

// dataSourceToConfig converts a store.DataSource to a drivers.Config.
func dataSourceToConfig(ds *store.DataSource) drivers.Config {
	return drivers.Config{
		Engine:        ds.Engine,
		Host:          ds.Host,
		Port:          ds.Port,
		Database:      ds.Database,
		Username:      ds.Username,
		Password:      ds.Password,
		SSL:           ds.SSL,
		SSLMode:       ds.SSLMode,
		SSLRootCert:   ds.SSLRootCert,
		SSLClientCert: ds.SSLClientCert,
		SSLClientKey:  ds.SSLClientKey,
		MaxOpenConns:  ds.MaxOpenConns,
		MaxIdleConns:  ds.MaxIdleConns,
		ConnMaxLifetime: time.Duration(ds.ConnMaxLifetime) * time.Second,
		ConnMaxIdleTime: time.Duration(ds.ConnMaxIdleTime) * time.Second,
		Options:       ds.Options,
	}
}

// categorizeError returns an error code based on the error message.
func categorizeError(err error) string {
	msg := strings.ToLower(err.Error())

	switch {
	case strings.Contains(msg, "connection refused"):
		return "CONNECTION_REFUSED"
	case strings.Contains(msg, "password authentication failed") ||
		strings.Contains(msg, "access denied"):
		return "AUTH_FAILED"
	case strings.Contains(msg, "does not exist") ||
		strings.Contains(msg, "unknown database"):
		return "DATABASE_NOT_FOUND"
	case strings.Contains(msg, "ssl") || strings.Contains(msg, "tls"):
		return "SSL_ERROR"
	case strings.Contains(msg, "timeout"):
		return "TIMEOUT"
	case strings.Contains(msg, "permission denied"):
		return "PERMISSION_DENIED"
	default:
		return "UNKNOWN"
	}
}

// getSuggestions returns helpful suggestions based on the error.
func getSuggestions(err error) []string {
	code := categorizeError(err)

	switch code {
	case "CONNECTION_REFUSED":
		return []string{
			"Check that the database server is running",
			"Verify the host address is correct",
			"Ensure the port is correct and not blocked by a firewall",
		}
	case "AUTH_FAILED":
		return []string{
			"Verify the username and password are correct",
			"Check that the user has permission to connect",
		}
	case "DATABASE_NOT_FOUND":
		return []string{
			"Verify the database name is spelled correctly",
			"Ensure the database exists and the user has access",
		}
	case "SSL_ERROR":
		return []string{
			"Check SSL configuration settings",
			"Verify certificates are valid and properly configured",
			"Try disabling SSL to test connectivity",
		}
	case "TIMEOUT":
		return []string{
			"Check network connectivity to the database server",
			"Verify the host and port are correct",
			"Consider increasing the connection timeout",
		}
	default:
		return nil
	}
}

// filterSchemas filters schemas based on patterns.
func filterSchemas(schemas []string, patterns []string, include bool) []string {
	patternSet := make(map[string]bool)
	for _, p := range patterns {
		patternSet[strings.ToLower(p)] = true
	}

	var result []string
	for _, s := range schemas {
		_, matches := patternSet[strings.ToLower(s)]
		if include && matches {
			result = append(result, s)
		} else if !include && !matches {
			result = append(result, s)
		}
	}
	return result
}

// inferSemanticType attempts to infer a semantic type from column name and type.
func inferSemanticType(name, mappedType string, isPK, isFK bool) string {
	lowerName := strings.ToLower(name)

	// Keys first
	if isPK {
		return store.SemanticPK
	}
	if isFK {
		return store.SemanticFK
	}

	// Date patterns
	if mappedType == "datetime" {
		switch {
		case strings.Contains(lowerName, "created") || strings.Contains(lowerName, "create"):
			return store.SemanticCreated
		case strings.Contains(lowerName, "updated") || strings.Contains(lowerName, "modified"):
			return store.SemanticUpdated
		case strings.Contains(lowerName, "joined") || strings.Contains(lowerName, "join"):
			return store.SemanticJoined
		case strings.Contains(lowerName, "birth"):
			return store.SemanticBirthday
		}
	}

	// Number patterns
	if mappedType == "number" {
		switch {
		case strings.Contains(lowerName, "price") || strings.Contains(lowerName, "cost") || strings.Contains(lowerName, "amount"):
			return store.SemanticPrice
		case strings.Contains(lowerName, "percent") || strings.Contains(lowerName, "rate"):
			return store.SemanticPercent
		case strings.Contains(lowerName, "quantity") || strings.Contains(lowerName, "qty") || strings.Contains(lowerName, "count"):
			return store.SemanticQuantity
		case strings.Contains(lowerName, "score") || strings.Contains(lowerName, "rating"):
			return store.SemanticScore
		case strings.Contains(lowerName, "lat"):
			return store.SemanticLatitude
		case strings.Contains(lowerName, "lng") || strings.Contains(lowerName, "lon"):
			return store.SemanticLongitude
		}
	}

	// String patterns
	if mappedType == "string" {
		switch {
		case lowerName == "name" || strings.HasSuffix(lowerName, "_name") || strings.HasSuffix(lowerName, "name"):
			return store.SemanticName
		case strings.Contains(lowerName, "title"):
			return store.SemanticTitle
		case strings.Contains(lowerName, "description") || strings.Contains(lowerName, "desc"):
			return store.SemanticDescription
		case strings.Contains(lowerName, "email"):
			return store.SemanticEmail
		case strings.Contains(lowerName, "phone") || strings.Contains(lowerName, "tel"):
			return store.SemanticPhone
		case strings.Contains(lowerName, "url") || strings.Contains(lowerName, "link") || strings.Contains(lowerName, "website"):
			return store.SemanticURL
		case strings.Contains(lowerName, "category") || strings.Contains(lowerName, "type") || strings.Contains(lowerName, "status"):
			return store.SemanticCategory
		case strings.Contains(lowerName, "zip") || strings.Contains(lowerName, "postal"):
			return store.SemanticZipCode
		case strings.Contains(lowerName, "city"):
			return store.SemanticCity
		case strings.Contains(lowerName, "state") || strings.Contains(lowerName, "province"):
			return store.SemanticState
		case strings.Contains(lowerName, "country"):
			return store.SemanticCountry
		case strings.Contains(lowerName, "address"):
			return store.SemanticAddress
		}
	}

	return ""
}

// Scan scans field values for a data source (populates cached values for filters).
func (h *DataSources) Scan(c *mizu.Ctx) error {
	id := c.Param("id")
	ds, err := h.store.DataSources().GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	if ds == nil {
		return c.JSON(404, map[string]string{"error": "Data source not found"})
	}

	var req struct {
		TableID  string `json:"table_id"`
		ColumnID string `json:"column_id"`
		Limit    int    `json:"limit"`
	}
	c.BindJSON(&req, 1<<20)
	if req.Limit == 0 {
		req.Limit = 1000
	}

	config := dataSourceToConfig(ds)
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Minute)
	defer cancel()

	start := time.Now()
	driver, err := drivers.Open(ctx, config)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	defer driver.Close()

	// Get tables to scan
	tables, err := h.store.Tables().ListByDataSource(ctx, id)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	var fieldsScanned, valuesCached int
	var scanErrors []string

	for _, table := range tables {
		if req.TableID != "" && table.ID != req.TableID {
			continue
		}

		columns, err := h.store.Tables().ListColumns(ctx, table.ID)
		if err != nil {
			scanErrors = append(scanErrors, fmt.Sprintf("table %s: %v", table.Name, err))
			continue
		}

		for _, col := range columns {
			if req.ColumnID != "" && col.ID != req.ColumnID {
				continue
			}

			// Skip non-scannable columns (long text, etc.)
			if col.Visibility == "hidden" {
				continue
			}

			// Scan distinct values
			query := fmt.Sprintf(
				"SELECT DISTINCT %s FROM %s WHERE %s IS NOT NULL ORDER BY %s LIMIT %d",
				driver.QuoteIdentifier(col.Name),
				driver.QuoteIdentifier(table.Name),
				driver.QuoteIdentifier(col.Name),
				driver.QuoteIdentifier(col.Name),
				req.Limit,
			)

			if table.Schema != "" && driver.SupportsSchemas() {
				query = fmt.Sprintf(
					"SELECT DISTINCT %s FROM %s.%s WHERE %s IS NOT NULL ORDER BY %s LIMIT %d",
					driver.QuoteIdentifier(col.Name),
					driver.QuoteIdentifier(table.Schema),
					driver.QuoteIdentifier(table.Name),
					driver.QuoteIdentifier(col.Name),
					driver.QuoteIdentifier(col.Name),
					req.Limit,
				)
			}

			result, err := driver.Execute(ctx, query)
			if err != nil {
				scanErrors = append(scanErrors, fmt.Sprintf("column %s.%s: %v", table.Name, col.Name, err))
				continue
			}

			// Extract values
			var values []string
			for _, row := range result.Rows {
				for _, v := range row {
					if v != nil {
						values = append(values, fmt.Sprintf("%v", v))
					}
				}
			}

			// Update column with cached values
			col.CachedValues = values
			now := time.Now()
			col.ValuesCachedAt = &now
			col.DistinctCount = int64(len(values))

			if err := h.store.Tables().UpdateColumn(ctx, col); err != nil {
				scanErrors = append(scanErrors, fmt.Sprintf("update column %s.%s: %v", table.Name, col.Name, err))
				continue
			}

			fieldsScanned++
			valuesCached += len(values)
		}
	}

	return c.JSON(200, map[string]any{
		"status":         "completed",
		"duration_ms":    time.Since(start).Milliseconds(),
		"fields_scanned": fieldsScanned,
		"values_cached":  valuesCached,
		"errors":         scanErrors,
	})
}

// Fingerprint runs fingerprinting on columns (calculates statistics).
func (h *DataSources) Fingerprint(c *mizu.Ctx) error {
	id := c.Param("id")
	ds, err := h.store.DataSources().GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	if ds == nil {
		return c.JSON(404, map[string]string{"error": "Data source not found"})
	}

	var req struct {
		TableID    string `json:"table_id"`
		SampleSize int    `json:"sample_size"`
	}
	c.BindJSON(&req, 1<<20)
	if req.SampleSize == 0 {
		req.SampleSize = 10000
	}

	config := dataSourceToConfig(ds)
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Minute)
	defer cancel()

	start := time.Now()
	driver, err := drivers.Open(ctx, config)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	defer driver.Close()

	tables, err := h.store.Tables().ListByDataSource(ctx, id)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	var columnsFingerprinted int
	var fingerErrors []string

	for _, table := range tables {
		if req.TableID != "" && table.ID != req.TableID {
			continue
		}

		columns, err := h.store.Tables().ListColumns(ctx, table.ID)
		if err != nil {
			fingerErrors = append(fingerErrors, fmt.Sprintf("table %s: %v", table.Name, err))
			continue
		}

		for _, col := range columns {
			// Build fingerprint query based on column type
			tableName := driver.QuoteIdentifier(table.Name)
			if table.Schema != "" && driver.SupportsSchemas() {
				tableName = driver.QuoteIdentifier(table.Schema) + "." + driver.QuoteIdentifier(table.Name)
			}
			colName := driver.QuoteIdentifier(col.Name)

			query := fmt.Sprintf(`
				SELECT
					COUNT(DISTINCT %s) as distinct_count,
					COUNT(*) - COUNT(%s) as null_count,
					MIN(%s) as min_value,
					MAX(%s) as max_value
				FROM (SELECT %s FROM %s LIMIT %d) t
			`, colName, colName, colName, colName, colName, tableName, req.SampleSize)

			result, err := driver.Execute(ctx, query)
			if err != nil {
				fingerErrors = append(fingerErrors, fmt.Sprintf("column %s.%s: %v", table.Name, col.Name, err))
				continue
			}

			if len(result.Rows) > 0 {
				row := result.Rows[0]
				if v, ok := row["distinct_count"]; ok {
					col.DistinctCount = toInt64(v)
				}
				if v, ok := row["null_count"]; ok {
					col.NullCount = toInt64(v)
				}
				if v, ok := row["min_value"]; ok && v != nil {
					col.MinValue = fmt.Sprintf("%v", v)
				}
				if v, ok := row["max_value"]; ok && v != nil {
					col.MaxValue = fmt.Sprintf("%v", v)
				}

				if err := h.store.Tables().UpdateColumn(ctx, col); err != nil {
					fingerErrors = append(fingerErrors, fmt.Sprintf("update column %s.%s: %v", table.Name, col.Name, err))
					continue
				}
				columnsFingerprinted++
			}
		}
	}

	return c.JSON(200, map[string]any{
		"status":                 "completed",
		"duration_ms":            time.Since(start).Milliseconds(),
		"columns_fingerprinted":  columnsFingerprinted,
		"errors":                 fingerErrors,
	})
}

// GetSyncLog returns sync history for a data source.
func (h *DataSources) GetSyncLog(c *mizu.Ctx) error {
	id := c.Param("id")
	ds, err := h.store.DataSources().GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	if ds == nil {
		return c.JSON(404, map[string]string{"error": "Data source not found"})
	}

	// For now, return basic sync info from the data source itself
	// A full implementation would store sync logs in a separate table
	logs := []map[string]any{}
	if ds.LastSyncAt != nil {
		logs = append(logs, map[string]any{
			"id":           fmt.Sprintf("log_%s", ds.ID),
			"type":         "schema_sync",
			"status":       ds.LastSyncStatus,
			"completed_at": ds.LastSyncAt,
			"error":        ds.LastSyncError,
		})
	}

	return c.JSON(200, map[string]any{"logs": logs})
}

// UpdateTable updates table metadata.
func (h *DataSources) UpdateTable(c *mizu.Ctx) error {
	tableID := c.Param("table")
	var update struct {
		DisplayName string `json:"display_name"`
		Description string `json:"description"`
		Visible     *bool  `json:"visible"`
		FieldOrder  string `json:"field_order"`
	}
	if err := c.BindJSON(&update, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request body"})
	}

	table, err := h.store.Tables().GetByID(c.Request().Context(), tableID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	if table == nil {
		return c.JSON(404, map[string]string{"error": "Table not found"})
	}

	if update.DisplayName != "" {
		table.DisplayName = update.DisplayName
	}
	if update.Description != "" {
		table.Description = update.Description
	}
	if update.Visible != nil {
		table.Visible = *update.Visible
	}

	if err := h.store.Tables().Update(c.Request().Context(), table); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, table)
}

// ScanColumn scans values for a single column.
func (h *DataSources) ScanColumn(c *mizu.Ctx) error {
	dsID := c.Param("id")
	tableID := c.Param("table")
	columnID := c.Param("column")

	ds, err := h.store.DataSources().GetByID(c.Request().Context(), dsID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	if ds == nil {
		return c.JSON(404, map[string]string{"error": "Data source not found"})
	}

	table, err := h.store.Tables().GetByID(c.Request().Context(), tableID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	if table == nil {
		return c.JSON(404, map[string]string{"error": "Table not found"})
	}

	col, err := h.store.Tables().GetColumn(c.Request().Context(), columnID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	if col == nil {
		return c.JSON(404, map[string]string{"error": "Column not found"})
	}

	var req struct {
		Limit int `json:"limit"`
	}
	c.BindJSON(&req, 1<<20)
	if req.Limit == 0 {
		req.Limit = 1000
	}

	config := dataSourceToConfig(ds)
	ctx, cancel := context.WithTimeout(c.Request().Context(), 2*time.Minute)
	defer cancel()

	start := time.Now()
	driver, err := drivers.Open(ctx, config)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	defer driver.Close()

	tableName := driver.QuoteIdentifier(table.Name)
	if table.Schema != "" && driver.SupportsSchemas() {
		tableName = driver.QuoteIdentifier(table.Schema) + "." + driver.QuoteIdentifier(table.Name)
	}
	colName := driver.QuoteIdentifier(col.Name)

	query := fmt.Sprintf(
		"SELECT DISTINCT %s FROM %s WHERE %s IS NOT NULL ORDER BY %s LIMIT %d",
		colName, tableName, colName, colName, req.Limit,
	)

	result, err := driver.Execute(ctx, query)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	var values []string
	for _, row := range result.Rows {
		for _, v := range row {
			if v != nil {
				values = append(values, fmt.Sprintf("%v", v))
			}
		}
	}

	// Update column
	col.CachedValues = values
	now := time.Now()
	col.ValuesCachedAt = &now
	col.DistinctCount = int64(len(values))

	if err := h.store.Tables().UpdateColumn(ctx, col); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]any{
		"column_id":      col.ID,
		"values":         values,
		"total_distinct": len(values),
		"duration_ms":    time.Since(start).Milliseconds(),
		"cached_at":      now,
	})
}

// DiscardCachedValues clears cached field values for a table.
func (h *DataSources) DiscardCachedValues(c *mizu.Ctx) error {
	tableID := c.Param("table")

	columns, err := h.store.Tables().ListColumns(c.Request().Context(), tableID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	cleared := 0
	for _, col := range columns {
		if len(col.CachedValues) > 0 {
			col.CachedValues = nil
			col.ValuesCachedAt = nil
			if err := h.store.Tables().UpdateColumn(c.Request().Context(), col); err != nil {
				continue
			}
			cleared++
		}
	}

	return c.JSON(200, map[string]any{
		"status":          "cleared",
		"columns_cleared": cleared,
	})
}

// GetCacheStats returns cache statistics for a data source.
func (h *DataSources) GetCacheStats(c *mizu.Ctx) error {
	id := c.Param("id")
	ds, err := h.store.DataSources().GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	if ds == nil {
		return c.JSON(404, map[string]string{"error": "Data source not found"})
	}

	tables, err := h.store.Tables().ListByDataSource(c.Request().Context(), id)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	var totalCachedValues int
	var columnsWithCache int

	for _, table := range tables {
		columns, err := h.store.Tables().ListColumns(c.Request().Context(), table.ID)
		if err != nil {
			continue
		}
		for _, col := range columns {
			if len(col.CachedValues) > 0 {
				columnsWithCache++
				totalCachedValues += len(col.CachedValues)
			}
		}
	}

	return c.JSON(200, map[string]any{
		"datasource_id":       id,
		"columns_with_cache":  columnsWithCache,
		"total_cached_values": totalCachedValues,
		"cache_ttl":           ds.CacheTTL,
	})
}

// ClearCache clears the query cache for a data source.
func (h *DataSources) ClearCache(c *mizu.Ctx) error {
	id := c.Param("id")
	ds, err := h.store.DataSources().GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	if ds == nil {
		return c.JSON(404, map[string]string{"error": "Data source not found"})
	}

	// Clear cached field values
	tables, err := h.store.Tables().ListByDataSource(c.Request().Context(), id)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	cleared := 0
	for _, table := range tables {
		columns, err := h.store.Tables().ListColumns(c.Request().Context(), table.ID)
		if err != nil {
			continue
		}
		for _, col := range columns {
			if len(col.CachedValues) > 0 {
				col.CachedValues = nil
				col.ValuesCachedAt = nil
				h.store.Tables().UpdateColumn(c.Request().Context(), col)
				cleared++
			}
		}
	}

	return c.JSON(200, map[string]any{
		"status":          "cleared",
		"columns_cleared": cleared,
	})
}

// GetTable returns a single table by ID.
func (h *DataSources) GetTable(c *mizu.Ctx) error {
	tableID := c.Param("table")
	table, err := h.store.Tables().GetByID(c.Request().Context(), tableID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	if table == nil {
		return c.JSON(404, map[string]string{"error": "Table not found"})
	}
	return c.JSON(200, table)
}

// SyncTable syncs a single table.
func (h *DataSources) SyncTable(c *mizu.Ctx) error {
	dsID := c.Param("id")
	tableID := c.Param("table")

	ds, err := h.store.DataSources().GetByID(c.Request().Context(), dsID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	if ds == nil {
		return c.JSON(404, map[string]string{"error": "Data source not found"})
	}

	table, err := h.store.Tables().GetByID(c.Request().Context(), tableID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	if table == nil {
		return c.JSON(404, map[string]string{"error": "Table not found"})
	}

	config := dataSourceToConfig(ds)
	ctx, cancel := context.WithTimeout(c.Request().Context(), 2*time.Minute)
	defer cancel()

	start := time.Now()
	driver, err := drivers.Open(ctx, config)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	defer driver.Close()

	columns, err := driver.ListColumns(ctx, table.Schema, table.Name)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	var columnsSynced int
	for _, col := range columns {
		storeCol := &store.Column{
			TableID:     table.ID,
			Name:        col.Name,
			DisplayName: col.Name,
			Type:        col.Type,
			MappedType:  col.MappedType,
			Description: col.Description,
			Position:    col.Position,
			Nullable:    col.Nullable,
			PrimaryKey:  col.PrimaryKey,
			ForeignKey:  col.ForeignKey,
			Visibility:  "everywhere",
		}
		storeCol.Semantic = inferSemanticType(col.Name, col.MappedType, col.PrimaryKey, col.ForeignKey)

		if err := h.store.Tables().CreateColumn(ctx, storeCol); err != nil {
			continue
		}
		columnsSynced++
	}

	return c.JSON(200, map[string]any{
		"status":         "completed",
		"duration_ms":    time.Since(start).Milliseconds(),
		"columns_synced": columnsSynced,
	})
}

// ScanTable scans field values for a single table.
func (h *DataSources) ScanTable(c *mizu.Ctx) error {
	dsID := c.Param("id")
	tableID := c.Param("table")

	ds, err := h.store.DataSources().GetByID(c.Request().Context(), dsID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	if ds == nil {
		return c.JSON(404, map[string]string{"error": "Data source not found"})
	}

	table, err := h.store.Tables().GetByID(c.Request().Context(), tableID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	if table == nil {
		return c.JSON(404, map[string]string{"error": "Table not found"})
	}

	// Delegate to Scan with table filter
	c.Request().URL.RawQuery = fmt.Sprintf("table_id=%s", tableID)
	return h.Scan(c)
}

// Helper to convert interface to int64
func toInt64(v any) int64 {
	switch val := v.(type) {
	case int64:
		return val
	case int:
		return int64(val)
	case float64:
		return int64(val)
	default:
		return 0
	}
}

// TablePreviewRequest represents a table preview request with pagination and filtering.
type TablePreviewRequest struct {
	Page     int                `json:"page,omitempty"`
	PageSize int                `json:"page_size,omitempty"`
	OrderBy  []TableOrderBy     `json:"order_by,omitempty"`
	Filters  []TableFilter      `json:"filters,omitempty"`
}

// TableOrderBy represents a sort specification.
type TableOrderBy struct {
	Column    string `json:"column"`
	Direction string `json:"direction"` // asc or desc
}

// TableFilter represents a filter condition.
type TableFilter struct {
	Column   string `json:"column"`
	Operator string `json:"operator"`
	Value    any    `json:"value"`
}

// TablePreview returns a preview of table data with pagination, sorting, and filtering.
func (h *DataSources) TablePreview(c *mizu.Ctx) error {
	dsID := c.Param("id")
	tableID := c.Param("table")

	ds, err := h.store.DataSources().GetByID(c.Request().Context(), dsID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	if ds == nil {
		return c.JSON(404, map[string]string{"error": "Data source not found"})
	}

	table, err := h.store.Tables().GetByID(c.Request().Context(), tableID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	if table == nil {
		return c.JSON(404, map[string]string{"error": "Table not found"})
	}

	// Parse request body (optional)
	var req TablePreviewRequest
	c.BindJSON(&req, 1<<20) // Ignore error - body is optional

	// Set defaults
	if req.PageSize <= 0 {
		req.PageSize = 100
	}
	if req.PageSize > 1000 {
		req.PageSize = 1000
	}
	if req.Page <= 0 {
		req.Page = 1
	}

	config := dataSourceToConfig(ds)
	ctx, cancel := context.WithTimeout(c.Request().Context(), 30*time.Second)
	defer cancel()

	driver, err := drivers.Open(ctx, config)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	defer driver.Close()

	// Build table name with schema
	tableName := driver.QuoteIdentifier(table.Name)
	if table.Schema != "" && driver.SupportsSchemas() {
		tableName = driver.QuoteIdentifier(table.Schema) + "." + driver.QuoteIdentifier(table.Name)
	}

	// First, get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) as count FROM %s", tableName)
	var totalRows int64

	// Add filters to count query
	if len(req.Filters) > 0 {
		whereClause, filterParams := buildFilterClause(driver, req.Filters)
		if whereClause != "" {
			countQuery += " WHERE " + whereClause
		}
		countResult, err := driver.Execute(ctx, countQuery, filterParams...)
		if err == nil && len(countResult.Rows) > 0 {
			if cnt, ok := countResult.Rows[0]["count"].(int64); ok {
				totalRows = cnt
			} else if cnt, ok := countResult.Rows[0]["count"].(float64); ok {
				totalRows = int64(cnt)
			}
		}
	} else {
		countResult, err := driver.Execute(ctx, countQuery)
		if err == nil && len(countResult.Rows) > 0 {
			if cnt, ok := countResult.Rows[0]["count"].(int64); ok {
				totalRows = cnt
			} else if cnt, ok := countResult.Rows[0]["count"].(float64); ok {
				totalRows = int64(cnt)
			}
		}
	}

	// Build data query
	dataQuery := fmt.Sprintf("SELECT * FROM %s", tableName)
	var queryParams []any

	// Add filters
	if len(req.Filters) > 0 {
		whereClause, filterParams := buildFilterClause(driver, req.Filters)
		if whereClause != "" {
			dataQuery += " WHERE " + whereClause
			queryParams = append(queryParams, filterParams...)
		}
	}

	// Add ORDER BY
	if len(req.OrderBy) > 0 {
		var orderClauses []string
		for _, o := range req.OrderBy {
			dir := "ASC"
			if strings.ToUpper(o.Direction) == "DESC" {
				dir = "DESC"
			}
			orderClauses = append(orderClauses, driver.QuoteIdentifier(o.Column)+" "+dir)
		}
		dataQuery += " ORDER BY " + strings.Join(orderClauses, ", ")
	}

	// Add pagination
	offset := (req.Page - 1) * req.PageSize
	dataQuery += fmt.Sprintf(" LIMIT %d OFFSET %d", req.PageSize, offset)

	// Execute query
	start := time.Now()
	result, err := driver.Execute(ctx, dataQuery, queryParams...)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	duration := time.Since(start).Milliseconds()

	// Build response with pagination info
	storeResult := &store.QueryResult{
		Columns:    make([]store.ResultColumn, len(result.Columns)),
		Rows:       result.Rows,
		RowCount:   int64(len(result.Rows)),
		TotalRows:  totalRows,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: int((totalRows + int64(req.PageSize) - 1) / int64(req.PageSize)),
		Duration:   float64(duration),
	}

	for i, col := range result.Columns {
		storeResult.Columns[i] = store.ResultColumn{
			Name:        col.Name,
			DisplayName: col.DisplayName,
			Type:        col.MappedType,
		}
	}

	return c.JSON(200, storeResult)
}

// buildFilterClause builds a WHERE clause from filters.
func buildFilterClause(driver drivers.Driver, filters []TableFilter) (string, []any) {
	var clauses []string
	var params []any

	for _, f := range filters {
		col := driver.QuoteIdentifier(f.Column)
		op := strings.ToUpper(f.Operator)

		switch op {
		case "IS NULL", "IS_NULL":
			clauses = append(clauses, col+" IS NULL")
		case "IS NOT NULL", "IS_NOT_NULL":
			clauses = append(clauses, col+" IS NOT NULL")
		case "IN":
			if arr, ok := f.Value.([]any); ok {
				placeholders := make([]string, len(arr))
				for i, v := range arr {
					placeholders[i] = "?"
					params = append(params, v)
				}
				clauses = append(clauses, col+" IN ("+strings.Join(placeholders, ", ")+")")
			}
		case "NOT IN":
			if arr, ok := f.Value.([]any); ok {
				placeholders := make([]string, len(arr))
				for i, v := range arr {
					placeholders[i] = "?"
					params = append(params, v)
				}
				clauses = append(clauses, col+" NOT IN ("+strings.Join(placeholders, ", ")+")")
			}
		case "BETWEEN":
			if arr, ok := f.Value.([]any); ok && len(arr) == 2 {
				clauses = append(clauses, col+" BETWEEN ? AND ?")
				params = append(params, arr[0], arr[1])
			}
		case "LIKE", "CONTAINS":
			clauses = append(clauses, col+" LIKE ?")
			params = append(params, "%"+fmt.Sprintf("%v", f.Value)+"%")
		case "STARTS_WITH", "STARTS WITH":
			clauses = append(clauses, col+" LIKE ?")
			params = append(params, fmt.Sprintf("%v", f.Value)+"%")
		case "ENDS_WITH", "ENDS WITH":
			clauses = append(clauses, col+" LIKE ?")
			params = append(params, "%"+fmt.Sprintf("%v", f.Value))
		case "=", "EQUALS":
			clauses = append(clauses, col+" = ?")
			params = append(params, f.Value)
		case "!=", "<>", "NOT_EQUALS":
			clauses = append(clauses, col+" != ?")
			params = append(params, f.Value)
		case ">", "GREATER_THAN":
			clauses = append(clauses, col+" > ?")
			params = append(params, f.Value)
		case ">=", "GREATER_OR_EQUAL":
			clauses = append(clauses, col+" >= ?")
			params = append(params, f.Value)
		case "<", "LESS_THAN":
			clauses = append(clauses, col+" < ?")
			params = append(params, f.Value)
		case "<=", "LESS_OR_EQUAL":
			clauses = append(clauses, col+" <= ?")
			params = append(params, f.Value)
		default:
			// Default to equals
			clauses = append(clauses, col+" = ?")
			params = append(params, f.Value)
		}
	}

	return strings.Join(clauses, " AND "), params
}

// SearchTables searches tables by name in a data source.
func (h *DataSources) SearchTables(c *mizu.Ctx) error {
	dsID := c.Param("id")
	query := c.Query("q")

	ds, err := h.store.DataSources().GetByID(c.Request().Context(), dsID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	if ds == nil {
		return c.JSON(404, map[string]string{"error": "Data source not found"})
	}

	tables, err := h.store.Tables().ListByDataSource(c.Request().Context(), dsID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	// Filter by query if provided
	if query != "" {
		query = strings.ToLower(query)
		var filtered []*store.Table
		for _, t := range tables {
			if strings.Contains(strings.ToLower(t.Name), query) ||
				strings.Contains(strings.ToLower(t.DisplayName), query) ||
				strings.Contains(strings.ToLower(t.Schema), query) {
				filtered = append(filtered, t)
			}
		}
		tables = filtered
	}

	return c.JSON(200, tables)
}
