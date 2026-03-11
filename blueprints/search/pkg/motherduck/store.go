package motherduck

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/duckdb/duckdb-go/v2"
)

const schema = `
CREATE TABLE IF NOT EXISTS accounts (
    id         VARCHAR PRIMARY KEY DEFAULT gen_random_uuid()::VARCHAR,
    email      VARCHAR NOT NULL UNIQUE,
    password   VARCHAR NOT NULL,
    token      VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT now(),
    is_active  BOOLEAN DEFAULT true
);

CREATE TABLE IF NOT EXISTS databases (
    id           VARCHAR PRIMARY KEY DEFAULT gen_random_uuid()::VARCHAR,
    account_id   VARCHAR NOT NULL REFERENCES accounts(id),
    name         VARCHAR NOT NULL,
    alias        VARCHAR NOT NULL UNIQUE,
    is_default   BOOLEAN DEFAULT false,
    created_at   TIMESTAMP DEFAULT now(),
    last_used_at TIMESTAMP,
    notes        VARCHAR DEFAULT ''
);

CREATE TABLE IF NOT EXISTS query_log (
    id            VARCHAR PRIMARY KEY DEFAULT gen_random_uuid()::VARCHAR,
    db_id         VARCHAR,
    sql           VARCHAR NOT NULL,
    rows_returned INTEGER,
    duration_ms   INTEGER,
    ran_at        TIMESTAMP DEFAULT now()
);
`

// DefaultDBPath returns ~/data/motherduck/mother.duckdb.
func DefaultDBPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "data", "motherduck", "mother.duckdb")
}

// Store manages local DuckDB state for MotherDuck accounts/databases.
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

// AddAccount inserts an account from RegisterResult.
func (s *Store) AddAccount(r RegisterResult) error {
	_, err := s.db.Exec(
		`INSERT INTO accounts (email, password, token) VALUES (?, ?, ?)`,
		r.Email, r.Password, r.Token,
	)
	return err
}

// ListAccounts returns all accounts with database counts.
func (s *Store) ListAccounts() ([]Account, error) {
	rows, err := s.db.Query(
		`SELECT a.email,
		        (SELECT COUNT(*) FROM databases d WHERE d.account_id = a.id),
		        a.is_active, COALESCE(CAST(a.created_at AS VARCHAR), '')
		 FROM accounts a ORDER BY a.created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Account
	for rows.Next() {
		var a Account
		if err := rows.Scan(&a.Email, &a.DBCount, &a.IsActive, &a.CreatedAt); err != nil {
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

// GetFirstActiveAccount returns the first active account.
func (s *Store) GetFirstActiveAccount() (*Account, error) {
	var a Account
	err := s.db.QueryRow(
		`SELECT id, email, token FROM accounts WHERE is_active = true ORDER BY created_at LIMIT 1`,
	).Scan(&a.ID, &a.Email, &a.Token)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// AddDatabase inserts a database row.
func (s *Store) AddDatabase(accountID, name, alias string) error {
	_, err := s.db.Exec(
		`INSERT INTO databases (account_id, name, alias) VALUES (?, ?, ?)`,
		accountID, name, alias,
	)
	return err
}

// ListDatabases returns all databases with account email and query counts.
func (s *Store) ListDatabases() ([]Database, error) {
	rows, err := s.db.Query(
		`SELECT d.id, d.alias, d.name, a.email, d.is_default,
		        (SELECT COUNT(*) FROM query_log q WHERE q.db_id = d.id),
		        COALESCE(CAST(d.last_used_at AS VARCHAR), ''),
		        COALESCE(CAST(d.created_at AS VARCHAR), ''),
		        a.token
		 FROM databases d JOIN accounts a ON d.account_id = a.id
		 ORDER BY d.is_default DESC, d.created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Database
	for rows.Next() {
		var d Database
		if err := rows.Scan(
			&d.ID, &d.Alias, &d.Name, &d.Email, &d.IsDefault,
			&d.QueryCount, &d.LastUsedAt, &d.CreatedAt, &d.Token,
		); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// GetDatabaseByAlias returns a database by alias.
func (s *Store) GetDatabaseByAlias(alias string) (*Database, error) {
	var d Database
	err := s.db.QueryRow(
		`SELECT d.id, d.alias, d.name, a.email, d.is_default,
		        COALESCE(CAST(d.created_at AS VARCHAR), ''), a.token, a.id
		 FROM databases d JOIN accounts a ON d.account_id = a.id
		 WHERE d.alias = ?`, alias,
	).Scan(&d.ID, &d.Alias, &d.Name, &d.Email, &d.IsDefault, &d.CreatedAt, &d.Token, &d.AccountID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// GetDefaultDatabase returns the default database.
func (s *Store) GetDefaultDatabase() (*Database, error) {
	var alias string
	err := s.db.QueryRow(`SELECT alias FROM databases WHERE is_default = true LIMIT 1`).Scan(&alias)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s.GetDatabaseByAlias(alias)
}

// SetDefault sets alias as the default database.
func (s *Store) SetDefault(alias string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	if _, err := tx.Exec(`UPDATE databases SET is_default = false`); err != nil {
		return err
	}
	res, err := tx.Exec(`UPDATE databases SET is_default = true WHERE alias = ?`, alias)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("database not found: %s", alias)
	}
	return tx.Commit()
}

// RemoveDatabase deletes a database by alias.
func (s *Store) RemoveDatabase(alias string) error {
	res, err := s.db.Exec(`DELETE FROM databases WHERE alias = ?`, alias)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("database not found: %s", alias)
	}
	return nil
}

// TouchLastUsed updates last_used_at for a database.
func (s *Store) TouchLastUsed(alias string) error {
	_, err := s.db.Exec(`UPDATE databases SET last_used_at = now() WHERE alias = ?`, alias)
	return err
}

// LogQuery appends a query_log row.
func (s *Store) LogQuery(dbID, sqlStr string, rows, durationMS int) error {
	_, err := s.db.Exec(
		`INSERT INTO query_log (db_id, sql, rows_returned, duration_ms) VALUES (?, ?, ?, ?)`,
		dbID, sqlStr, rows, durationMS,
	)
	return err
}

// splitStatements splits a SQL string on semicolons.
func splitStatements(sqlStr string) []string {
	var out []string
	start := 0
	for i := 0; i < len(sqlStr); i++ {
		if sqlStr[i] == ';' {
			if s := trimWS(sqlStr[start:i]); s != "" {
				out = append(out, s)
			}
			start = i + 1
		}
	}
	if s := trimWS(sqlStr[start:]); s != "" {
		out = append(out, s)
	}
	return out
}

func trimWS(s string) string {
	i, j := 0, len(s)-1
	for i <= j && isWS(s[i]) {
		i++
	}
	for j >= i && isWS(s[j]) {
		j--
	}
	return s[i : j+1]
}

func isWS(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}
