// Package fingerprint provides column fingerprinting (statistics) and value caching.
package fingerprint

import (
	"context"
	"fmt"
	"time"

	"github.com/go-mizu/blueprints/bi/drivers"
	"github.com/go-mizu/blueprints/bi/store"
)

const (
	// MaxSampleSize is the maximum number of rows to sample for fingerprinting.
	MaxSampleSize = 10000

	// MaxCachedValues is the maximum number of distinct values to cache.
	MaxCachedValues = 1000

	// MaxValueLength is the maximum length of cached values.
	MaxValueLength = 100
)

// Fingerprint contains statistical information about a column.
type Fingerprint struct {
	ColumnID      string    `json:"column_id"`
	DistinctCount int64     `json:"distinct_count"`
	NullCount     int64     `json:"null_count"`
	TotalCount    int64     `json:"total_count"`
	MinValue      string    `json:"min_value,omitempty"`
	MaxValue      string    `json:"max_value,omitempty"`
	AvgLength     float64   `json:"avg_length,omitempty"` // for string columns
	SampleSize    int64     `json:"sample_size"`
	ComputedAt    time.Time `json:"computed_at"`
}

// Service provides fingerprinting operations.
type Service struct {
	store store.Store
}

// NewService creates a new fingerprint service.
func NewService(store store.Store) *Service {
	return &Service{store: store}
}

// FingerprintColumn computes statistics for a single column.
func (s *Service) FingerprintColumn(ctx context.Context, driver drivers.Driver, schema, table, column, mappedType string) (*Fingerprint, error) {
	quotedCol := driver.QuoteIdentifier(column)
	quotedTable := driver.QuoteIdentifier(table)

	var tableRef string
	if schema != "" && driver.SupportsSchemas() {
		tableRef = driver.QuoteIdentifier(schema) + "." + quotedTable
	} else {
		tableRef = quotedTable
	}

	// Build query based on database capabilities
	query := buildFingerprintQuery(driver.Name(), tableRef, quotedCol, mappedType)

	result, err := driver.Execute(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("fingerprint query: %w", err)
	}

	if len(result.Rows) == 0 {
		return &Fingerprint{
			ComputedAt: time.Now(),
		}, nil
	}

	row := result.Rows[0]
	fp := &Fingerprint{
		ComputedAt: time.Now(),
	}

	// Parse results
	if v, ok := row["distinct_count"]; ok && v != nil {
		fp.DistinctCount = toInt64(v)
	}
	if v, ok := row["null_count"]; ok && v != nil {
		fp.NullCount = toInt64(v)
	}
	if v, ok := row["total_count"]; ok && v != nil {
		fp.TotalCount = toInt64(v)
	}
	if v, ok := row["sample_size"]; ok && v != nil {
		fp.SampleSize = toInt64(v)
	}
	if v, ok := row["min_value"]; ok && v != nil {
		fp.MinValue = toString(v)
	}
	if v, ok := row["max_value"]; ok && v != nil {
		fp.MaxValue = toString(v)
	}
	if v, ok := row["avg_length"]; ok && v != nil {
		fp.AvgLength = toFloat64(v)
	}

	return fp, nil
}

// FingerprintTable computes statistics for all columns in a table.
func (s *Service) FingerprintTable(ctx context.Context, driver drivers.Driver, schema, table string, columns []*store.Column) (map[string]*Fingerprint, error) {
	results := make(map[string]*Fingerprint)

	for _, col := range columns {
		fp, err := s.FingerprintColumn(ctx, driver, schema, table, col.Name, col.MappedType)
		if err != nil {
			// Log error but continue with other columns
			continue
		}
		fp.ColumnID = col.ID
		results[col.ID] = fp
	}

	return results, nil
}

// CacheFieldValues caches distinct values for a column (for filter dropdowns).
func (s *Service) CacheFieldValues(ctx context.Context, driver drivers.Driver, schema, table, column string) ([]string, error) {
	quotedCol := driver.QuoteIdentifier(column)
	quotedTable := driver.QuoteIdentifier(table)

	var tableRef string
	if schema != "" && driver.SupportsSchemas() {
		tableRef = driver.QuoteIdentifier(schema) + "." + quotedTable
	} else {
		tableRef = quotedTable
	}

	query := fmt.Sprintf(`
		SELECT DISTINCT %s AS value
		FROM %s
		WHERE %s IS NOT NULL
		ORDER BY %s
		LIMIT %d
	`, quotedCol, tableRef, quotedCol, quotedCol, MaxCachedValues)

	result, err := driver.Execute(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("cache values query: %w", err)
	}

	values := make([]string, 0, len(result.Rows))
	for _, row := range result.Rows {
		if v, ok := row["value"]; ok && v != nil {
			strVal := toString(v)
			if len(strVal) <= MaxValueLength {
				values = append(values, strVal)
			} else {
				// Truncate long values
				values = append(values, strVal[:MaxValueLength]+"...")
			}
		}
	}

	return values, nil
}

// UpdateColumnFingerprint updates a column's fingerprint data in the store.
func (s *Service) UpdateColumnFingerprint(ctx context.Context, columnID string, fp *Fingerprint, cachedValues []string) error {
	col, err := s.store.Tables().GetColumn(ctx, columnID)
	if err != nil {
		return err
	}
	if col == nil {
		return fmt.Errorf("column not found: %s", columnID)
	}

	// Update fingerprint data
	col.DistinctCount = fp.DistinctCount
	col.NullCount = fp.NullCount
	col.MinValue = fp.MinValue
	col.MaxValue = fp.MaxValue
	col.AvgLength = fp.AvgLength

	// Update cached values
	if len(cachedValues) > 0 {
		col.CachedValues = cachedValues
		now := time.Now()
		col.ValuesCachedAt = &now
	}

	return s.store.Tables().UpdateColumn(ctx, col)
}

// FingerprintDataSource fingerprints all columns in a data source.
func (s *Service) FingerprintDataSource(ctx context.Context, dataSourceID string) (*FingerprintResult, error) {
	result := &FingerprintResult{
		DataSourceID: dataSourceID,
		StartedAt:    time.Now(),
	}

	// Get data source
	ds, err := s.store.DataSources().GetByID(ctx, dataSourceID)
	if err != nil {
		result.Status = "failed"
		result.Errors = append(result.Errors, err.Error())
		return result, err
	}
	if ds == nil {
		result.Status = "failed"
		result.Errors = append(result.Errors, "data source not found")
		return result, fmt.Errorf("data source not found")
	}

	// Create driver config
	config := drivers.Config{
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
		Options:       ds.Options,
	}

	// Open connection
	driver, err := drivers.Open(ctx, config)
	if err != nil {
		result.Status = "failed"
		result.Errors = append(result.Errors, err.Error())
		return result, err
	}
	defer driver.Close()

	// Get all tables for this data source
	tables, err := s.store.Tables().ListByDataSource(ctx, dataSourceID)
	if err != nil {
		result.Status = "failed"
		result.Errors = append(result.Errors, err.Error())
		return result, err
	}

	for _, table := range tables {
		// Get columns for this table
		columns, err := s.store.Tables().ListColumns(ctx, table.ID)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("table %s: %v", table.Name, err))
			continue
		}

		for _, col := range columns {
			// Fingerprint the column
			fp, err := s.FingerprintColumn(ctx, driver, table.Schema, table.Name, col.Name, col.MappedType)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("column %s.%s: %v", table.Name, col.Name, err))
				continue
			}
			fp.ColumnID = col.ID

			// Cache values for categorical columns
			var cachedValues []string
			if col.MappedType == "string" && fp.DistinctCount <= MaxCachedValues {
				cachedValues, _ = s.CacheFieldValues(ctx, driver, table.Schema, table.Name, col.Name)
			}

			// Update in store
			if err := s.UpdateColumnFingerprint(ctx, col.ID, fp, cachedValues); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("update %s.%s: %v", table.Name, col.Name, err))
				continue
			}

			result.ColumnsFingerprinted++
			if len(cachedValues) > 0 {
				result.ValuesScanned++
			}
		}

		result.TablesProcessed++
	}

	result.CompletedAt = time.Now()
	result.DurationMs = result.CompletedAt.Sub(result.StartedAt).Milliseconds()

	if len(result.Errors) == 0 {
		result.Status = "success"
	} else if result.ColumnsFingerprinted > 0 {
		result.Status = "partial"
	} else {
		result.Status = "failed"
	}

	return result, nil
}

// FingerprintResult holds the result of a fingerprinting operation.
type FingerprintResult struct {
	DataSourceID         string    `json:"datasource_id"`
	Status               string    `json:"status"`
	StartedAt            time.Time `json:"started_at"`
	CompletedAt          time.Time `json:"completed_at"`
	DurationMs           int64     `json:"duration_ms"`
	TablesProcessed      int       `json:"tables_processed"`
	ColumnsFingerprinted int       `json:"columns_fingerprinted"`
	ValuesScanned        int       `json:"values_scanned"`
	Errors               []string  `json:"errors,omitempty"`
}

// buildFingerprintQuery builds a fingerprint query for a specific database.
func buildFingerprintQuery(driverName, tableRef, quotedCol, mappedType string) string {
	// Base statistics that work on most databases
	baseQuery := fmt.Sprintf(`
		SELECT
			COUNT(DISTINCT %s) AS distinct_count,
			SUM(CASE WHEN %s IS NULL THEN 1 ELSE 0 END) AS null_count,
			COUNT(*) AS total_count,
			%d AS sample_size
		FROM (SELECT %s FROM %s LIMIT %d) AS sample
	`, quotedCol, quotedCol, MaxSampleSize, quotedCol, tableRef, MaxSampleSize)

	// Add min/max for numeric and date types
	if mappedType == "number" || mappedType == "datetime" {
		return fmt.Sprintf(`
			SELECT
				COUNT(DISTINCT %s) AS distinct_count,
				SUM(CASE WHEN %s IS NULL THEN 1 ELSE 0 END) AS null_count,
				COUNT(*) AS total_count,
				%d AS sample_size,
				CAST(MIN(%s) AS TEXT) AS min_value,
				CAST(MAX(%s) AS TEXT) AS max_value
			FROM (SELECT %s FROM %s LIMIT %d) AS sample
		`, quotedCol, quotedCol, MaxSampleSize, quotedCol, quotedCol, quotedCol, tableRef, MaxSampleSize)
	}

	// Add avg_length for string types
	if mappedType == "string" {
		switch driverName {
		case "postgres", "postgresql":
			return fmt.Sprintf(`
				SELECT
					COUNT(DISTINCT %s) AS distinct_count,
					SUM(CASE WHEN %s IS NULL THEN 1 ELSE 0 END) AS null_count,
					COUNT(*) AS total_count,
					%d AS sample_size,
					MIN(%s) AS min_value,
					MAX(%s) AS max_value,
					AVG(LENGTH(%s)) AS avg_length
				FROM (SELECT %s FROM %s LIMIT %d) AS sample
			`, quotedCol, quotedCol, MaxSampleSize, quotedCol, quotedCol, quotedCol, quotedCol, tableRef, MaxSampleSize)

		case "mysql", "mariadb":
			return fmt.Sprintf(`
				SELECT
					COUNT(DISTINCT %s) AS distinct_count,
					SUM(CASE WHEN %s IS NULL THEN 1 ELSE 0 END) AS null_count,
					COUNT(*) AS total_count,
					%d AS sample_size,
					MIN(%s) AS min_value,
					MAX(%s) AS max_value,
					AVG(CHAR_LENGTH(%s)) AS avg_length
				FROM (SELECT %s FROM %s LIMIT %d) AS sample
			`, quotedCol, quotedCol, MaxSampleSize, quotedCol, quotedCol, quotedCol, quotedCol, tableRef, MaxSampleSize)

		case "sqlite":
			return fmt.Sprintf(`
				SELECT
					COUNT(DISTINCT %s) AS distinct_count,
					SUM(CASE WHEN %s IS NULL THEN 1 ELSE 0 END) AS null_count,
					COUNT(*) AS total_count,
					%d AS sample_size,
					MIN(%s) AS min_value,
					MAX(%s) AS max_value,
					AVG(LENGTH(%s)) AS avg_length
				FROM (SELECT %s FROM %s LIMIT %d)
			`, quotedCol, quotedCol, MaxSampleSize, quotedCol, quotedCol, quotedCol, quotedCol, tableRef, MaxSampleSize)
		}
	}

	return baseQuery
}

// Helper functions for type conversion

func toInt64(v any) int64 {
	switch val := v.(type) {
	case int64:
		return val
	case int:
		return int64(val)
	case int32:
		return int64(val)
	case float64:
		return int64(val)
	case float32:
		return int64(val)
	default:
		return 0
	}
}

func toFloat64(v any) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int64:
		return float64(val)
	case int:
		return float64(val)
	default:
		return 0
	}
}

func toString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	default:
		return fmt.Sprintf("%v", v)
	}
}
