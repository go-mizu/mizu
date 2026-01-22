package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/bi/store"
)

// DataSourceStore implements store.DataSourceStore.
type DataSourceStore struct {
	db *sql.DB
}

func (s *DataSourceStore) Create(ctx context.Context, ds *store.DataSource) error {
	if ds.ID == "" {
		ds.ID = generateID()
	}
	now := time.Now()
	ds.CreatedAt = now
	ds.UpdatedAt = now

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO datasources (id, name, engine, host, port, database_name, username, password, ssl, options, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, ds.ID, ds.Name, ds.Engine, ds.Host, ds.Port, ds.Database, ds.Username, ds.Password, ds.SSL, toJSON(ds.Options), ds.CreatedAt, ds.UpdatedAt)
	return err
}

func (s *DataSourceStore) GetByID(ctx context.Context, id string) (*store.DataSource, error) {
	var ds store.DataSource
	var options string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, engine, host, port, database_name, username, password, ssl, options, created_at, updated_at
		FROM datasources WHERE id = ?
	`, id).Scan(&ds.ID, &ds.Name, &ds.Engine, &ds.Host, &ds.Port, &ds.Database, &ds.Username, &ds.Password, &ds.SSL, &options, &ds.CreatedAt, &ds.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	fromJSON(options, &ds.Options)
	return &ds, nil
}

func (s *DataSourceStore) List(ctx context.Context) ([]*store.DataSource, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, engine, host, port, database_name, username, password, ssl, options, created_at, updated_at
		FROM datasources ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.DataSource
	for rows.Next() {
		var ds store.DataSource
		var options string
		if err := rows.Scan(&ds.ID, &ds.Name, &ds.Engine, &ds.Host, &ds.Port, &ds.Database, &ds.Username, &ds.Password, &ds.SSL, &options, &ds.CreatedAt, &ds.UpdatedAt); err != nil {
			return nil, err
		}
		fromJSON(options, &ds.Options)
		result = append(result, &ds)
	}
	return result, rows.Err()
}

func (s *DataSourceStore) Update(ctx context.Context, ds *store.DataSource) error {
	ds.UpdatedAt = time.Now()
	_, err := s.db.ExecContext(ctx, `
		UPDATE datasources SET name=?, engine=?, host=?, port=?, database_name=?, username=?, password=?, ssl=?, options=?, updated_at=?
		WHERE id=?
	`, ds.Name, ds.Engine, ds.Host, ds.Port, ds.Database, ds.Username, ds.Password, ds.SSL, toJSON(ds.Options), ds.UpdatedAt, ds.ID)
	return err
}

func (s *DataSourceStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM datasources WHERE id=?`, id)
	return err
}
