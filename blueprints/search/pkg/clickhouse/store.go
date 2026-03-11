package clickhouse

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/duckdb/duckdb-go/v2"
)

const schema = `
CREATE TABLE IF NOT EXISTS accounts (
    id             VARCHAR PRIMARY KEY DEFAULT gen_random_uuid()::VARCHAR,
    email          VARCHAR NOT NULL UNIQUE,
    password       VARCHAR NOT NULL,
    org_id         VARCHAR DEFAULT '',
    api_key_id     VARCHAR DEFAULT '',
    api_key_secret VARCHAR DEFAULT '',
    created_at     TIMESTAMP DEFAULT now(),
    is_active      BOOLEAN DEFAULT true
);

CREATE TABLE IF NOT EXISTS services (
    id           VARCHAR PRIMARY KEY DEFAULT gen_random_uuid()::VARCHAR,
    account_id   VARCHAR NOT NULL REFERENCES accounts(id),
    cloud_id     VARCHAR DEFAULT '',
    name         VARCHAR NOT NULL,
    alias        VARCHAR NOT NULL UNIQUE,
    host         VARCHAR DEFAULT '',
    port         INTEGER DEFAULT 8443,
    db_user      VARCHAR DEFAULT 'default',
    db_password  VARCHAR DEFAULT '',
    provider     VARCHAR DEFAULT 'aws',
    region       VARCHAR DEFAULT 'us-east-1',
    is_default   BOOLEAN DEFAULT false,
    created_at   TIMESTAMP DEFAULT now(),
    last_used_at TIMESTAMP,
    notes        VARCHAR DEFAULT ''
);

CREATE TABLE IF NOT EXISTS query_log (
    id            VARCHAR PRIMARY KEY DEFAULT gen_random_uuid()::VARCHAR,
    service_id    VARCHAR REFERENCES services(id),
    sql           VARCHAR NOT NULL,
    rows_returned INTEGER,
    duration_ms   INTEGER,
    ran_at        TIMESTAMP DEFAULT now()
);
`

// DefaultDBPath returns ~/data/clickhouse/clickhouse.duckdb.
func DefaultDBPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "data", "clickhouse", "clickhouse.duckdb")
}

// Store manages local DuckDB state for ClickHouse accounts/services.
type Store struct {
	db *sql.DB
}

// NewStore opens the DuckDB file and runs schema migrations.
func NewStore(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("mkdir: %w", err)
	}
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, fmt.Errorf("open duckdb: %w", err)
	}
	db.SetMaxOpenConns(1)
	for _, stmt := range splitStatements(schema) {
		if _, err := db.Exec(stmt); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("schema: %w", err)
		}
	}
	return &Store{db: db}, nil
}

// Close closes the store.
func (s *Store) Close() error {
	return s.db.Close()
}

// AddAccount inserts an account + optional service from a RegisterResult atomically.
func (s *Store) AddAccount(r RegisterResult) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	// Insert account
	var accountID string
	err = tx.QueryRow(
		`INSERT INTO accounts (email, password, org_id, api_key_id, api_key_secret)
		 VALUES (?, ?, ?, ?, ?)
		 RETURNING id`,
		r.Email, r.Password, r.OrgID, r.APIKeyID, r.APIKeySecret,
	).Scan(&accountID)
	if err != nil {
		return fmt.Errorf("insert account: %w", err)
	}

	// Insert service if host is present
	if r.Host != "" && r.ServiceID != "" {
		alias := emailAlias(r.Email)
		port := r.Port
		if port == 0 {
			port = 8443
		}
		if _, err := tx.Exec(
			`INSERT INTO services (account_id, cloud_id, name, alias, host, port, db_user, db_password)
			 VALUES (?, ?, 'default-service', ?, ?, ?, 'default', ?)`,
			accountID, r.ServiceID, alias, r.Host, port, r.DBPassword,
		); err != nil {
			return fmt.Errorf("insert service: %w", err)
		}
		// Set as default
		if _, err := tx.Exec(`UPDATE services SET is_default = false`); err != nil {
			return fmt.Errorf("clear defaults: %w", err)
		}
		if _, err := tx.Exec(`UPDATE services SET is_default = true WHERE alias = ?`, alias); err != nil {
			return fmt.Errorf("set default: %w", err)
		}
	}

	return tx.Commit()
}

// ListAccounts returns all accounts with service counts.
func (s *Store) ListAccounts() ([]Account, error) {
	rows, err := s.db.Query(
		`SELECT a.email,
		        (SELECT COUNT(*) FROM services sv WHERE sv.account_id = a.id),
		        a.is_active, COALESCE(CAST(a.created_at AS VARCHAR), ''), a.org_id
		 FROM accounts a ORDER BY a.created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Account
	for rows.Next() {
		var a Account
		if err := rows.Scan(&a.Email, &a.SvcCount, &a.IsActive, &a.CreatedAt, &a.OrgID); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// DeactivateAccount marks an account as inactive.
func (s *Store) DeactivateAccount(email string) error {
	res, err := s.db.Exec(`UPDATE accounts SET is_active = false WHERE email = ?`, email)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("account not found: %s", email)
	}
	return nil
}

// ListServices returns all services with account email and query counts.
func (s *Store) ListServices() ([]Service, error) {
	rows, err := s.db.Query(
		`SELECT s.id, s.alias, s.name, a.email, s.is_default, s.host, s.port,
		        (SELECT COUNT(*) FROM query_log q WHERE q.service_id = s.id),
		        COALESCE(CAST(s.last_used_at AS VARCHAR), ''),
		        COALESCE(CAST(s.created_at AS VARCHAR), ''),
		        s.db_user, s.db_password, s.cloud_id
		 FROM services s JOIN accounts a ON s.account_id = a.id
		 ORDER BY s.is_default DESC, s.created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Service
	for rows.Next() {
		var sv Service
		if err := rows.Scan(
			&sv.ID, &sv.Alias, &sv.Name, &sv.Email, &sv.IsDefault, &sv.Host, &sv.Port,
			&sv.QueryCount, &sv.LastUsedAt, &sv.CreatedAt, &sv.DBUser, &sv.DBPassword, &sv.CloudID,
		); err != nil {
			return nil, err
		}
		out = append(out, sv)
	}
	return out, rows.Err()
}

// GetServiceByAlias returns a service by alias.
func (s *Store) GetServiceByAlias(alias string) (*Service, error) {
	var sv Service
	err := s.db.QueryRow(
		`SELECT s.id, s.alias, s.name, a.email, s.is_default, s.host, s.port,
		        s.db_user, s.db_password, s.cloud_id,
		        COALESCE(CAST(s.created_at AS VARCHAR), '')
		 FROM services s JOIN accounts a ON s.account_id = a.id
		 WHERE s.alias = ?`, alias,
	).Scan(
		&sv.ID, &sv.Alias, &sv.Name, &sv.Email, &sv.IsDefault, &sv.Host, &sv.Port,
		&sv.DBUser, &sv.DBPassword, &sv.CloudID, &sv.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &sv, nil
}

// GetDefaultService returns the default service.
func (s *Store) GetDefaultService() (*Service, error) {
	var alias string
	err := s.db.QueryRow(`SELECT alias FROM services WHERE is_default = true LIMIT 1`).Scan(&alias)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s.GetServiceByAlias(alias)
}

// SetDefault sets alias as the default service (transaction: clear all, set one).
func (s *Store) SetDefault(alias string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	if _, err := tx.Exec(`UPDATE services SET is_default = false`); err != nil {
		return err
	}
	res, err := tx.Exec(`UPDATE services SET is_default = true WHERE alias = ?`, alias)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("service not found: %s", alias)
	}
	return tx.Commit()
}

// RemoveService deletes a service by alias.
func (s *Store) RemoveService(alias string) error {
	res, err := s.db.Exec(`DELETE FROM services WHERE alias = ?`, alias)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("service not found: %s", alias)
	}
	return nil
}

// TouchLastUsed updates last_used_at for a service.
func (s *Store) TouchLastUsed(alias string) error {
	_, err := s.db.Exec(`UPDATE services SET last_used_at = now() WHERE alias = ?`, alias)
	return err
}

// LogQuery appends a query_log row.
func (s *Store) LogQuery(serviceID, sqlStr string, rows, durationMS int) error {
	_, err := s.db.Exec(
		`INSERT INTO query_log (service_id, sql, rows_returned, duration_ms) VALUES (?, ?, ?, ?)`,
		serviceID, sqlStr, rows, durationMS,
	)
	return err
}

// splitStatements splits a SQL string on semicolons (skipping empty).
func splitStatements(sql string) []string {
	var out []string
	for _, s := range splitOnSemicolon(sql) {
		if t := trimSpace(s); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func splitOnSemicolon(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ';' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

func trimSpace(s string) string {
	i, j := 0, len(s)-1
	for i <= j && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r') {
		i++
	}
	for j >= i && (s[j] == ' ' || s[j] == '\t' || s[j] == '\n' || s[j] == '\r') {
		j--
	}
	return s[i : j+1]
}

// emailAlias returns up to 20 chars of the email local part.
func emailAlias(email string) string {
	for i, c := range email {
		if c == '@' {
			if i > 20 {
				return email[:20]
			}
			return email[:i]
		}
	}
	if len(email) > 20 {
		return email[:20]
	}
	return email
}
