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
		INSERT INTO datasources (
			id, name, engine, host, port, database_name, username, password,
			ssl, ssl_mode, ssl_root_cert, ssl_client_cert, ssl_client_key,
			tunnel_enabled, tunnel_host, tunnel_port, tunnel_user, tunnel_auth_method,
			tunnel_password, tunnel_private_key, tunnel_passphrase,
			schema_filter_type, schema_filter_patterns,
			auto_sync, sync_schedule, last_sync_at, last_sync_status, last_sync_error,
			cache_ttl, max_open_conns, max_idle_conns, conn_max_lifetime, conn_max_idle_time,
			options, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		ds.ID, ds.Name, ds.Engine, ds.Host, ds.Port, ds.Database, ds.Username, ds.Password,
		ds.SSL, ds.SSLMode, ds.SSLRootCert, ds.SSLClientCert, ds.SSLClientKey,
		ds.TunnelEnabled, ds.TunnelHost, ds.TunnelPort, ds.TunnelUser, ds.TunnelAuthMethod,
		ds.TunnelPassword, ds.TunnelPrivateKey, ds.TunnelPassphrase,
		ds.SchemaFilterType, toJSON(ds.SchemaFilterPatterns),
		ds.AutoSync, ds.SyncSchedule, ds.LastSyncAt, ds.LastSyncStatus, ds.LastSyncError,
		ds.CacheTTL, ds.MaxOpenConns, ds.MaxIdleConns, ds.ConnMaxLifetime, ds.ConnMaxIdleTime,
		toJSON(ds.Options), ds.CreatedAt, ds.UpdatedAt,
	)
	return err
}

func (s *DataSourceStore) GetByID(ctx context.Context, id string) (*store.DataSource, error) {
	var ds store.DataSource
	var options, schemaPatterns string
	var lastSyncAt sql.NullTime

	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, engine, host, port, database_name, username, password,
			ssl, COALESCE(ssl_mode, ''), COALESCE(ssl_root_cert, ''), COALESCE(ssl_client_cert, ''), COALESCE(ssl_client_key, ''),
			COALESCE(tunnel_enabled, 0), COALESCE(tunnel_host, ''), COALESCE(tunnel_port, 0), COALESCE(tunnel_user, ''), COALESCE(tunnel_auth_method, ''),
			COALESCE(tunnel_password, ''), COALESCE(tunnel_private_key, ''), COALESCE(tunnel_passphrase, ''),
			COALESCE(schema_filter_type, ''), COALESCE(schema_filter_patterns, '[]'),
			COALESCE(auto_sync, 0), COALESCE(sync_schedule, ''), last_sync_at, COALESCE(last_sync_status, ''), COALESCE(last_sync_error, ''),
			COALESCE(cache_ttl, 0), COALESCE(max_open_conns, 0), COALESCE(max_idle_conns, 0), COALESCE(conn_max_lifetime, 0), COALESCE(conn_max_idle_time, 0),
			COALESCE(options, '{}'), created_at, updated_at
		FROM datasources WHERE id = ?
	`, id).Scan(
		&ds.ID, &ds.Name, &ds.Engine, &ds.Host, &ds.Port, &ds.Database, &ds.Username, &ds.Password,
		&ds.SSL, &ds.SSLMode, &ds.SSLRootCert, &ds.SSLClientCert, &ds.SSLClientKey,
		&ds.TunnelEnabled, &ds.TunnelHost, &ds.TunnelPort, &ds.TunnelUser, &ds.TunnelAuthMethod,
		&ds.TunnelPassword, &ds.TunnelPrivateKey, &ds.TunnelPassphrase,
		&ds.SchemaFilterType, &schemaPatterns,
		&ds.AutoSync, &ds.SyncSchedule, &lastSyncAt, &ds.LastSyncStatus, &ds.LastSyncError,
		&ds.CacheTTL, &ds.MaxOpenConns, &ds.MaxIdleConns, &ds.ConnMaxLifetime, &ds.ConnMaxIdleTime,
		&options, &ds.CreatedAt, &ds.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	fromJSON(options, &ds.Options)
	fromJSON(schemaPatterns, &ds.SchemaFilterPatterns)
	if lastSyncAt.Valid {
		ds.LastSyncAt = &lastSyncAt.Time
	}

	return &ds, nil
}

func (s *DataSourceStore) List(ctx context.Context) ([]*store.DataSource, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, engine, host, port, database_name, username, password,
			ssl, COALESCE(ssl_mode, ''), COALESCE(ssl_root_cert, ''), COALESCE(ssl_client_cert, ''), COALESCE(ssl_client_key, ''),
			COALESCE(tunnel_enabled, 0), COALESCE(tunnel_host, ''), COALESCE(tunnel_port, 0), COALESCE(tunnel_user, ''), COALESCE(tunnel_auth_method, ''),
			COALESCE(tunnel_password, ''), COALESCE(tunnel_private_key, ''), COALESCE(tunnel_passphrase, ''),
			COALESCE(schema_filter_type, ''), COALESCE(schema_filter_patterns, '[]'),
			COALESCE(auto_sync, 0), COALESCE(sync_schedule, ''), last_sync_at, COALESCE(last_sync_status, ''), COALESCE(last_sync_error, ''),
			COALESCE(cache_ttl, 0), COALESCE(max_open_conns, 0), COALESCE(max_idle_conns, 0), COALESCE(conn_max_lifetime, 0), COALESCE(conn_max_idle_time, 0),
			COALESCE(options, '{}'), created_at, updated_at
		FROM datasources ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.DataSource
	for rows.Next() {
		var ds store.DataSource
		var options, schemaPatterns string
		var lastSyncAt sql.NullTime

		if err := rows.Scan(
			&ds.ID, &ds.Name, &ds.Engine, &ds.Host, &ds.Port, &ds.Database, &ds.Username, &ds.Password,
			&ds.SSL, &ds.SSLMode, &ds.SSLRootCert, &ds.SSLClientCert, &ds.SSLClientKey,
			&ds.TunnelEnabled, &ds.TunnelHost, &ds.TunnelPort, &ds.TunnelUser, &ds.TunnelAuthMethod,
			&ds.TunnelPassword, &ds.TunnelPrivateKey, &ds.TunnelPassphrase,
			&ds.SchemaFilterType, &schemaPatterns,
			&ds.AutoSync, &ds.SyncSchedule, &lastSyncAt, &ds.LastSyncStatus, &ds.LastSyncError,
			&ds.CacheTTL, &ds.MaxOpenConns, &ds.MaxIdleConns, &ds.ConnMaxLifetime, &ds.ConnMaxIdleTime,
			&options, &ds.CreatedAt, &ds.UpdatedAt,
		); err != nil {
			return nil, err
		}

		fromJSON(options, &ds.Options)
		fromJSON(schemaPatterns, &ds.SchemaFilterPatterns)
		if lastSyncAt.Valid {
			ds.LastSyncAt = &lastSyncAt.Time
		}

		result = append(result, &ds)
	}
	return result, rows.Err()
}

func (s *DataSourceStore) Update(ctx context.Context, ds *store.DataSource) error {
	ds.UpdatedAt = time.Now()
	_, err := s.db.ExecContext(ctx, `
		UPDATE datasources SET
			name=?, engine=?, host=?, port=?, database_name=?, username=?, password=?,
			ssl=?, ssl_mode=?, ssl_root_cert=?, ssl_client_cert=?, ssl_client_key=?,
			tunnel_enabled=?, tunnel_host=?, tunnel_port=?, tunnel_user=?, tunnel_auth_method=?,
			tunnel_password=?, tunnel_private_key=?, tunnel_passphrase=?,
			schema_filter_type=?, schema_filter_patterns=?,
			auto_sync=?, sync_schedule=?, last_sync_at=?, last_sync_status=?, last_sync_error=?,
			cache_ttl=?, max_open_conns=?, max_idle_conns=?, conn_max_lifetime=?, conn_max_idle_time=?,
			options=?, updated_at=?
		WHERE id=?
	`,
		ds.Name, ds.Engine, ds.Host, ds.Port, ds.Database, ds.Username, ds.Password,
		ds.SSL, ds.SSLMode, ds.SSLRootCert, ds.SSLClientCert, ds.SSLClientKey,
		ds.TunnelEnabled, ds.TunnelHost, ds.TunnelPort, ds.TunnelUser, ds.TunnelAuthMethod,
		ds.TunnelPassword, ds.TunnelPrivateKey, ds.TunnelPassphrase,
		ds.SchemaFilterType, toJSON(ds.SchemaFilterPatterns),
		ds.AutoSync, ds.SyncSchedule, ds.LastSyncAt, ds.LastSyncStatus, ds.LastSyncError,
		ds.CacheTTL, ds.MaxOpenConns, ds.MaxIdleConns, ds.ConnMaxLifetime, ds.ConnMaxIdleTime,
		toJSON(ds.Options), ds.UpdatedAt, ds.ID,
	)
	return err
}

func (s *DataSourceStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM datasources WHERE id=?`, id)
	return err
}
