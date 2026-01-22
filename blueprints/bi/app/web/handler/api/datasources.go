package api

import (
	"database/sql"
	"fmt"

	"github.com/go-mizu/mizu"

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

// Test tests the connection to a data source.
func (h *DataSources) Test(c *mizu.Ctx) error {
	id := c.Param("id")
	ds, err := h.store.DataSources().GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	if ds == nil {
		return c.JSON(404, map[string]string{"error": "Data source not found"})
	}

	// Test connection based on engine type
	var testErr error
	switch ds.Engine {
	case "sqlite":
		db, err := sql.Open("sqlite3", ds.Database)
		if err != nil {
			testErr = err
		} else {
			testErr = db.Ping()
			db.Close()
		}
	default:
		testErr = fmt.Errorf("unsupported engine: %s", ds.Engine)
	}

	if testErr != nil {
		return c.JSON(200, map[string]interface{}{
			"success": false,
			"error":   testErr.Error(),
		})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
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

	// Sync tables based on engine type
	var tables []*store.Table
	switch ds.Engine {
	case "sqlite":
		db, err := sql.Open("sqlite3", ds.Database)
		if err != nil {
			return c.JSON(500, map[string]string{"error": err.Error()})
		}
		defer db.Close()

		// Get tables
		rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'`)
		if err != nil {
			return c.JSON(500, map[string]string{"error": err.Error()})
		}
		defer rows.Close()

		for rows.Next() {
			var name string
			rows.Scan(&name)
			table := &store.Table{
				DataSourceID: ds.ID,
				Name:         name,
				DisplayName:  name,
			}
			tables = append(tables, table)
		}
	default:
		return c.JSON(400, map[string]string{"error": "Unsupported engine"})
	}

	// Store tables and columns
	for _, table := range tables {
		if err := h.store.Tables().Create(c.Request().Context(), table); err != nil {
			continue // Ignore duplicates
		}

		// Get columns
		db, _ := sql.Open("sqlite3", ds.Database)
		defer db.Close()

		rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table.Name))
		if err != nil {
			continue
		}
		defer rows.Close()

		pos := 0
		for rows.Next() {
			var cid int
			var name, colType string
			var notNull, pk int
			var dflt interface{}
			rows.Scan(&cid, &name, &colType, &notNull, &dflt, &pk)

			col := &store.Column{
				TableID:     table.ID,
				Name:        name,
				DisplayName: name,
				Type:        mapSQLiteType(colType),
				Position:    pos,
			}
			h.store.Tables().CreateColumn(c.Request().Context(), col)
			pos++
		}
	}

	return c.JSON(200, map[string]interface{}{
		"tables_synced": len(tables),
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

func mapSQLiteType(sqlType string) string {
	switch sqlType {
	case "INTEGER", "INT", "REAL", "NUMERIC":
		return "number"
	case "TEXT", "VARCHAR", "CHAR":
		return "string"
	case "BLOB":
		return "string"
	case "DATETIME", "DATE", "TIMESTAMP":
		return "datetime"
	case "BOOLEAN":
		return "boolean"
	default:
		return "string"
	}
}
