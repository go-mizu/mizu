package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-mizu/blueprints/localflare/store"
)

// AnalyticsEngineStoreImpl implements store.AnalyticsEngineStore.
type AnalyticsEngineStoreImpl struct {
	db *sql.DB
}

func (s *AnalyticsEngineStoreImpl) CreateDataset(ctx context.Context, dataset *store.AnalyticsEngineDataset) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO analytics_engine_datasets (id, name, created_at)
		VALUES (?, ?, ?)`,
		dataset.ID, dataset.Name, dataset.CreatedAt)
	return err
}

func (s *AnalyticsEngineStoreImpl) GetDataset(ctx context.Context, name string) (*store.AnalyticsEngineDataset, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, created_at FROM analytics_engine_datasets WHERE name = ?`, name)
	var ds store.AnalyticsEngineDataset
	if err := row.Scan(&ds.ID, &ds.Name, &ds.CreatedAt); err != nil {
		return nil, err
	}
	return &ds, nil
}

func (s *AnalyticsEngineStoreImpl) ListDatasets(ctx context.Context) ([]*store.AnalyticsEngineDataset, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, created_at FROM analytics_engine_datasets ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var datasets []*store.AnalyticsEngineDataset
	for rows.Next() {
		var ds store.AnalyticsEngineDataset
		if err := rows.Scan(&ds.ID, &ds.Name, &ds.CreatedAt); err != nil {
			return nil, err
		}
		datasets = append(datasets, &ds)
	}
	return datasets, rows.Err()
}

func (s *AnalyticsEngineStoreImpl) DeleteDataset(ctx context.Context, name string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM analytics_engine_datasets WHERE name = ?`, name)
	return err
}

func (s *AnalyticsEngineStoreImpl) WriteDataPoint(ctx context.Context, point *store.AnalyticsEngineDataPoint) error {
	indexesJSON, _ := json.Marshal(point.Indexes)
	doublesJSON, _ := json.Marshal(point.Doubles)

	// Convert blobs to base64 strings for storage
	blobsBase64 := make([]string, len(point.Blobs))
	for i, b := range point.Blobs {
		blobsBase64[i] = string(b)
	}
	blobsJSON, _ := json.Marshal(blobsBase64)

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO analytics_engine_data (dataset, timestamp, indexes, doubles, blobs)
		VALUES (?, ?, ?, ?, ?)`,
		point.Dataset, point.Timestamp, string(indexesJSON), string(doublesJSON), string(blobsJSON))
	return err
}

func (s *AnalyticsEngineStoreImpl) WriteBatch(ctx context.Context, points []*store.AnalyticsEngineDataPoint) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO analytics_engine_data (dataset, timestamp, indexes, doubles, blobs)
		VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, point := range points {
		indexesJSON, _ := json.Marshal(point.Indexes)
		doublesJSON, _ := json.Marshal(point.Doubles)

		blobsBase64 := make([]string, len(point.Blobs))
		for i, b := range point.Blobs {
			blobsBase64[i] = string(b)
		}
		blobsJSON, _ := json.Marshal(blobsBase64)

		if _, err := stmt.ExecContext(ctx, point.Dataset, point.Timestamp,
			string(indexesJSON), string(doublesJSON), string(blobsJSON)); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *AnalyticsEngineStoreImpl) Query(ctx context.Context, sqlQuery string) ([]map[string]interface{}, error) {
	// Parse and rewrite the SQL query to work with our schema
	// This is a simplified implementation - a full implementation would need a proper SQL parser
	rewrittenSQL := s.rewriteSQL(sqlQuery)

	rows, err := s.db.QueryContext(ctx, rewrittenSQL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(cols))
		valuePtrs := make([]interface{}, len(cols))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}
		row := make(map[string]interface{})
		for i, col := range cols {
			// Handle JSON arrays stored as strings
			if strVal, ok := values[i].(string); ok {
				if strings.HasPrefix(strVal, "[") {
					var arr []interface{}
					if json.Unmarshal([]byte(strVal), &arr) == nil {
						row[col] = arr
						continue
					}
				}
			}
			row[col] = values[i]
		}
		results = append(results, row)
	}
	return results, rows.Err()
}

func (s *AnalyticsEngineStoreImpl) rewriteSQL(query string) string {
	// Simple SQL rewriting for Analytics Engine queries
	// This maps Cloudflare's schema to our SQLite schema

	// Replace index1-index20 with JSON extraction
	for i := 1; i <= 20; i++ {
		placeholder := fmt.Sprintf("index%d", i)
		replacement := fmt.Sprintf("json_extract(indexes, '$[%d]')", i-1)
		query = strings.ReplaceAll(query, placeholder, replacement)
	}

	// Replace double1-double20 with JSON extraction
	for i := 1; i <= 20; i++ {
		placeholder := fmt.Sprintf("double%d", i)
		replacement := fmt.Sprintf("json_extract(doubles, '$[%d]')", i-1)
		query = strings.ReplaceAll(query, placeholder, replacement)
	}

	// Replace blob1-blob20 with JSON extraction
	for i := 1; i <= 20; i++ {
		placeholder := fmt.Sprintf("blob%d", i)
		replacement := fmt.Sprintf("json_extract(blobs, '$[%d]')", i-1)
		query = strings.ReplaceAll(query, placeholder, replacement)
	}

	// Handle FROM clause - replace dataset name with actual table
	// This is a simplified approach
	query = strings.ReplaceAll(query, "FROM ", "FROM analytics_engine_data WHERE dataset = ")

	return query
}

// Schema for Analytics Engine
const analyticsEngineSchema = `
	-- Analytics Engine Datasets
	CREATE TABLE IF NOT EXISTS analytics_engine_datasets (
		id TEXT PRIMARY KEY,
		name TEXT UNIQUE NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Analytics Engine Data Points
	CREATE TABLE IF NOT EXISTS analytics_engine_data (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		dataset TEXT NOT NULL,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		indexes TEXT DEFAULT '[]',
		doubles TEXT DEFAULT '[]',
		blobs TEXT DEFAULT '[]'
	);
	CREATE INDEX IF NOT EXISTS idx_ae_data_dataset ON analytics_engine_data(dataset);
	CREATE INDEX IF NOT EXISTS idx_ae_data_timestamp ON analytics_engine_data(dataset, timestamp);
`
