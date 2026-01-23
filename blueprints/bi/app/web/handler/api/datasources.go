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
