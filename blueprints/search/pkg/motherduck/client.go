package motherduck

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/duckdb/duckdb-go/v2"
)

// Query runs SQL against a MotherDuck database and returns rows + column names.
// dbName is the MotherDuck database name (e.g. "mydb"). Use "" for no specific db.
func Query(ctx context.Context, token, dbName, sqlStr string) ([]map[string]any, []string, error) {
	dsn := "md:"
	if dbName != "" {
		dsn += dbName
	}
	dsn += "?motherduck_token=" + token

	db, err := sql.Open("duckdb", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("open: %w", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	rows, err := db.QueryContext(ctx, sqlStr)
	if err != nil {
		return nil, nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, nil, err
	}

	var result []map[string]any
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, nil, err
		}
		row := make(map[string]any, len(cols))
		for i, col := range cols {
			row[col] = vals[i]
		}
		result = append(result, row)
	}
	return result, cols, rows.Err()
}

// CreateDB creates a MotherDuck database via DuckDB md: connection.
func CreateDB(ctx context.Context, token, name string) error {
	db, err := sql.Open("duckdb", "md:?motherduck_token="+token)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	_, err = db.ExecContext(ctx, "CREATE DATABASE IF NOT EXISTS "+name)
	return err
}
