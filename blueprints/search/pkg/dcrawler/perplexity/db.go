package perplexity

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// DB wraps a DuckDB database for storing Perplexity search results, accounts, and errors.
type DB struct {
	db   *sql.DB
	path string
}

// OpenDB opens or creates a DuckDB database at the given path.
func OpenDB(path string) (*DB, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, fmt.Errorf("open duckdb %s: %w", path, err)
	}

	d := &DB{db: db, path: path}
	if err := d.initSchema(); err != nil {
		db.Close()
		return nil, err
	}

	return d, nil
}

func (d *DB) initSchema() error {
	stmts := []string{
		`CREATE SEQUENCE IF NOT EXISTS search_seq START 1`,
		`CREATE SEQUENCE IF NOT EXISTS account_seq START 1`,
		`CREATE SEQUENCE IF NOT EXISTS apikey_seq START 1`,
		`CREATE SEQUENCE IF NOT EXISTS error_seq START 1`,
		`CREATE SEQUENCE IF NOT EXISTS thread_seq START 1`,
		`CREATE SEQUENCE IF NOT EXISTS thread_msg_seq START 1`,

		`CREATE TABLE IF NOT EXISTS searches (
			id           INTEGER PRIMARY KEY DEFAULT nextval('search_seq'),
			query        TEXT NOT NULL,
			answer       TEXT,
			citations    TEXT,
			web_results  TEXT,
			chunks       TEXT,
			media_items  TEXT,
			related      TEXT,
			backend_uuid TEXT,
			mode         TEXT,
			model        TEXT,
			source       TEXT DEFAULT 'sse',
			searched_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			account_id   INTEGER,
			api_key_id   INTEGER,
			tokens_used  INTEGER DEFAULT 0,
			duration_ms  INTEGER DEFAULT 0,
			error        TEXT
		)`,

		`CREATE TABLE IF NOT EXISTS accounts (
			id           INTEGER PRIMARY KEY DEFAULT nextval('account_seq'),
			email        TEXT NOT NULL,
			source       TEXT NOT NULL DEFAULT 'emailnator',
			session_data TEXT,
			pro_queries  INTEGER DEFAULT 5,
			file_uploads INTEGER DEFAULT 10,
			status       TEXT DEFAULT 'active',
			error_msg    TEXT,
			created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_used_at TIMESTAMP,
			use_count    INTEGER DEFAULT 0
		)`,

		`CREATE TABLE IF NOT EXISTS api_keys (
			id           INTEGER PRIMARY KEY DEFAULT nextval('apikey_seq'),
			api_key      TEXT NOT NULL,
			name         TEXT,
			status       TEXT DEFAULT 'active',
			error_msg    TEXT,
			tier         TEXT DEFAULT 'tier0',
			created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_used_at TIMESTAMP,
			use_count    INTEGER DEFAULT 0,
			total_tokens INTEGER DEFAULT 0
		)`,

		`CREATE TABLE IF NOT EXISTS errors (
			id            INTEGER PRIMARY KEY DEFAULT nextval('error_seq'),
			account_id    INTEGER,
			api_key_id    INTEGER,
			source        TEXT NOT NULL,
			operation     TEXT NOT NULL,
			query         TEXT,
			error_type    TEXT NOT NULL,
			error_msg     TEXT NOT NULL,
			http_status   INTEGER,
			response_body TEXT,
			created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS threads (
			id            INTEGER PRIMARY KEY DEFAULT nextval('thread_seq'),
			title         TEXT NOT NULL,
			mode          TEXT DEFAULT 'auto',
			model         TEXT,
			source        TEXT DEFAULT 'sse',
			account_id    INTEGER,
			api_key_id    INTEGER,
			message_count INTEGER DEFAULT 0,
			created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS thread_messages (
			id            INTEGER PRIMARY KEY DEFAULT nextval('thread_msg_seq'),
			thread_id     INTEGER NOT NULL,
			role          TEXT NOT NULL,
			content       TEXT NOT NULL,
			backend_uuid  TEXT,
			citations     TEXT,
			web_results   TEXT,
			related       TEXT,
			tokens_used   INTEGER DEFAULT 0,
			duration_ms   INTEGER DEFAULT 0,
			created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, stmt := range stmts {
		if _, err := d.db.Exec(stmt); err != nil {
			return fmt.Errorf("init schema: %w (stmt: %s)", err, stmt[:min(80, len(stmt))])
		}
	}

	// Migrate existing searches table if needed (add new columns)
	migrateCols := []string{
		`ALTER TABLE searches ADD COLUMN IF NOT EXISTS account_id INTEGER`,
		`ALTER TABLE searches ADD COLUMN IF NOT EXISTS api_key_id INTEGER`,
		`ALTER TABLE searches ADD COLUMN IF NOT EXISTS tokens_used INTEGER DEFAULT 0`,
		`ALTER TABLE searches ADD COLUMN IF NOT EXISTS duration_ms INTEGER DEFAULT 0`,
		`ALTER TABLE searches ADD COLUMN IF NOT EXISTS error TEXT`,
	}
	for _, stmt := range migrateCols {
		d.db.Exec(stmt) // ignore errors (column may already exist)
	}

	return nil
}

// --- Search Operations ---

// SaveSearch stores a search result.
func (d *DB) SaveSearch(r *SearchResult) error {
	citationsJSON, _ := json.Marshal(r.Citations)
	webResultsJSON, _ := json.Marshal(r.WebResults)
	chunksJSON, _ := json.Marshal(r.Chunks)
	mediaJSON, _ := json.Marshal(r.MediaItems)
	relatedJSON, _ := json.Marshal(r.RelatedQ)

	_, err := d.db.Exec(`
		INSERT INTO searches (query, answer, citations, web_results, chunks, media_items, related,
			backend_uuid, mode, model, source, searched_at, account_id, api_key_id, tokens_used, duration_ms)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		r.Query, r.Answer,
		string(citationsJSON), string(webResultsJSON),
		string(chunksJSON), string(mediaJSON),
		string(relatedJSON), r.BackendUUID,
		r.Mode, r.Model, r.Source, r.SearchedAt,
		nilIfZero(r.AccountID), nilIfZero(r.APIKeyID),
		r.TokensUsed, r.DurationMs,
	)
	return err
}

// Count returns the total number of stored searches.
func (d *DB) Count() (int, error) {
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM searches").Scan(&count)
	return count, err
}

// RecentSearches returns the N most recent searches.
func (d *DB) RecentSearches(limit int) ([]SearchResult, error) {
	rows, err := d.db.Query(`
		SELECT query, answer, citations, web_results, related, backend_uuid, mode, model, source, searched_at
		FROM searches
		ORDER BY searched_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		var citationsStr, webResultsStr, relatedStr sql.NullString
		err := rows.Scan(
			&r.Query, &r.Answer, &citationsStr, &webResultsStr,
			&relatedStr, &r.BackendUUID, &r.Mode, &r.Model, &r.Source, &r.SearchedAt,
		)
		if err != nil {
			continue
		}
		if citationsStr.Valid {
			json.Unmarshal([]byte(citationsStr.String), &r.Citations)
		}
		if webResultsStr.Valid {
			json.Unmarshal([]byte(webResultsStr.String), &r.WebResults)
		}
		if relatedStr.Valid {
			json.Unmarshal([]byte(relatedStr.String), &r.RelatedQ)
		}
		results = append(results, r)
	}
	return results, nil
}

// --- Account Operations ---

// SaveAccount stores a new registered account.
func (d *DB) SaveAccount(a *Account) error {
	return d.db.QueryRow(`
		INSERT INTO accounts (email, source, session_data, pro_queries, file_uploads, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		RETURNING id
	`, a.Email, a.Source, a.SessionData, a.ProQueries, a.FileUploads, a.Status, time.Now(),
	).Scan(&a.ID)
}

// ListAccounts returns all accounts, optionally filtered by status.
func (d *DB) ListAccounts(status string) ([]Account, error) {
	query := `SELECT id, email, source, pro_queries, file_uploads, status, error_msg,
		created_at, COALESCE(last_used_at, created_at), use_count
		FROM accounts`
	var args []any
	if status != "" {
		query += ` WHERE status = ?`
		args = append(args, status)
	}
	query += ` ORDER BY id`

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []Account
	for rows.Next() {
		var a Account
		var errMsg sql.NullString
		err := rows.Scan(&a.ID, &a.Email, &a.Source, &a.ProQueries, &a.FileUploads,
			&a.Status, &errMsg, &a.CreatedAt, &a.LastUsedAt, &a.UseCount)
		if err != nil {
			continue
		}
		if errMsg.Valid {
			a.ErrorMsg = errMsg.String
		}
		accounts = append(accounts, a)
	}
	return accounts, nil
}

// GetAccountSession returns the session data for an account.
func (d *DB) GetAccountSession(id int) (string, error) {
	var data sql.NullString
	err := d.db.QueryRow(`SELECT session_data FROM accounts WHERE id = ?`, id).Scan(&data)
	if err != nil {
		return "", err
	}
	return data.String, nil
}

// UpdateAccountStatus updates an account's status and error message.
func (d *DB) UpdateAccountStatus(id int, status, errMsg string) error {
	_, err := d.db.Exec(`UPDATE accounts SET status = ?, error_msg = ? WHERE id = ?`, status, errMsg, id)
	return err
}

// UpdateAccountUsage increments use count and updates last_used_at.
func (d *DB) UpdateAccountUsage(id int, proQueriesLeft int) error {
	_, err := d.db.Exec(`
		UPDATE accounts SET use_count = use_count + 1, last_used_at = ?, pro_queries = ? WHERE id = ?
	`, time.Now(), proQueriesLeft, id)
	return err
}

// DeleteAccount removes an account by ID.
func (d *DB) DeleteAccount(id int) error {
	_, err := d.db.Exec(`DELETE FROM accounts WHERE id = ?`, id)
	return err
}

// CountAccounts returns account counts by status.
func (d *DB) CountAccounts() (total, active int, err error) {
	err = d.db.QueryRow(`SELECT COUNT(*) FROM accounts`).Scan(&total)
	if err != nil {
		return
	}
	err = d.db.QueryRow(`SELECT COUNT(*) FROM accounts WHERE status = 'active'`).Scan(&active)
	return
}

// --- API Key Operations ---

// SaveAPIKey stores a new API key.
func (d *DB) SaveAPIKey(k *APIKey) error {
	return d.db.QueryRow(`
		INSERT INTO api_keys (api_key, name, status, tier, created_at)
		VALUES (?, ?, ?, ?, ?)
		RETURNING id
	`, k.Key, k.Name, k.Status, k.Tier, time.Now(),
	).Scan(&k.ID)
}

// ListAPIKeys returns all API keys.
func (d *DB) ListAPIKeys() ([]APIKey, error) {
	rows, err := d.db.Query(`
		SELECT id, api_key, name, status, error_msg, tier, created_at,
			COALESCE(last_used_at, created_at), use_count, total_tokens
		FROM api_keys ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []APIKey
	for rows.Next() {
		var k APIKey
		var errMsg, name sql.NullString
		err := rows.Scan(&k.ID, &k.Key, &name, &k.Status, &errMsg, &k.Tier,
			&k.CreatedAt, &k.LastUsedAt, &k.UseCount, &k.TotalTokens)
		if err != nil {
			continue
		}
		if name.Valid {
			k.Name = name.String
		}
		if errMsg.Valid {
			k.ErrorMsg = errMsg.String
		}
		keys = append(keys, k)
	}
	return keys, nil
}

// UpdateAPIKeyStatus updates an API key's status.
func (d *DB) UpdateAPIKeyStatus(id int, status, errMsg string) error {
	_, err := d.db.Exec(`UPDATE api_keys SET status = ?, error_msg = ? WHERE id = ?`, status, errMsg, id)
	return err
}

// UpdateAPIKeyUsage increments use count and tokens.
func (d *DB) UpdateAPIKeyUsage(id, tokens int) error {
	_, err := d.db.Exec(`
		UPDATE api_keys SET use_count = use_count + 1, last_used_at = ?, total_tokens = total_tokens + ? WHERE id = ?
	`, time.Now(), tokens, id)
	return err
}

// DeleteAPIKey removes an API key by ID.
func (d *DB) DeleteAPIKey(id int) error {
	_, err := d.db.Exec(`DELETE FROM api_keys WHERE id = ?`, id)
	return err
}

// CountAPIKeys returns API key counts.
func (d *DB) CountAPIKeys() (total, active int, err error) {
	err = d.db.QueryRow(`SELECT COUNT(*) FROM api_keys`).Scan(&total)
	if err != nil {
		return
	}
	err = d.db.QueryRow(`SELECT COUNT(*) FROM api_keys WHERE status = 'active'`).Scan(&active)
	return
}

// --- Error Logging ---

// LogError stores an error in the database.
func (d *DB) LogError(e *ErrorLog) error {
	body := e.ResponseBody
	if len(body) > maxErrorBodyLen {
		body = body[:maxErrorBodyLen]
	}
	_, err := d.db.Exec(`
		INSERT INTO errors (account_id, api_key_id, source, operation, query, error_type, error_msg, http_status, response_body, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		nilIfZero(e.AccountID), nilIfZero(e.APIKeyID),
		e.Source, e.Operation, e.Query,
		e.ErrorType, e.ErrorMsg,
		nilIfZero(e.HTTPStatus), body,
		time.Now(),
	)
	return err
}

// RecentErrors returns recent errors, optionally filtered.
func (d *DB) RecentErrors(limit int, source, errType string) ([]ErrorLog, error) {
	query := `SELECT id, COALESCE(account_id, 0), COALESCE(api_key_id, 0), source, operation,
		COALESCE(query, ''), error_type, error_msg, COALESCE(http_status, 0),
		COALESCE(response_body, ''), created_at
		FROM errors WHERE 1=1`
	var args []any

	if source != "" {
		query += ` AND source = ?`
		args = append(args, source)
	}
	if errType != "" {
		query += ` AND error_type = ?`
		args = append(args, errType)
	}
	query += ` ORDER BY created_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var errors []ErrorLog
	for rows.Next() {
		var e ErrorLog
		err := rows.Scan(&e.ID, &e.AccountID, &e.APIKeyID, &e.Source, &e.Operation,
			&e.Query, &e.ErrorType, &e.ErrorMsg, &e.HTTPStatus, &e.ResponseBody, &e.CreatedAt)
		if err != nil {
			continue
		}
		errors = append(errors, e)
	}
	return errors, nil
}

// CountErrors returns total error count.
func (d *DB) CountErrors() (int, error) {
	var count int
	err := d.db.QueryRow(`SELECT COUNT(*) FROM errors`).Scan(&count)
	return count, err
}

// --- Thread Operations ---

// CreateThread creates a new conversation thread.
func (d *DB) CreateThread(t *Thread) error {
	now := time.Now()
	return d.db.QueryRow(`
		INSERT INTO threads (title, mode, model, source, account_id, api_key_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id
	`, t.Title, t.Mode, t.Model, t.Source,
		nilIfZero(t.AccountID), nilIfZero(t.APIKeyID), now, now,
	).Scan(&t.ID)
}

// GetThread returns a thread by ID.
func (d *DB) GetThread(id int) (*Thread, error) {
	var t Thread
	err := d.db.QueryRow(`
		SELECT id, title, mode, COALESCE(model, ''), source,
			COALESCE(account_id, 0), COALESCE(api_key_id, 0),
			message_count, created_at, updated_at
		FROM threads WHERE id = ?
	`, id).Scan(&t.ID, &t.Title, &t.Mode, &t.Model, &t.Source,
		&t.AccountID, &t.APIKeyID, &t.MessageCount, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// ListThreads returns recent threads ordered by last activity.
func (d *DB) ListThreads(limit int) ([]Thread, error) {
	rows, err := d.db.Query(`
		SELECT id, title, mode, COALESCE(model, ''), source,
			COALESCE(account_id, 0), COALESCE(api_key_id, 0),
			message_count, created_at, updated_at
		FROM threads
		ORDER BY updated_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var threads []Thread
	for rows.Next() {
		var t Thread
		err := rows.Scan(&t.ID, &t.Title, &t.Mode, &t.Model, &t.Source,
			&t.AccountID, &t.APIKeyID, &t.MessageCount, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			continue
		}
		threads = append(threads, t)
	}
	return threads, nil
}

// DeleteThread removes a thread and its messages.
func (d *DB) DeleteThread(id int) error {
	if _, err := d.db.Exec(`DELETE FROM thread_messages WHERE thread_id = ?`, id); err != nil {
		return err
	}
	_, err := d.db.Exec(`DELETE FROM threads WHERE id = ?`, id)
	return err
}

// CountThreads returns the total number of threads.
func (d *DB) CountThreads() (int, error) {
	var count int
	err := d.db.QueryRow(`SELECT COUNT(*) FROM threads`).Scan(&count)
	return count, err
}

// AddThreadMessage saves a message to a thread and updates the thread.
func (d *DB) AddThreadMessage(msg *ThreadMessage) error {
	citationsJSON, _ := json.Marshal(msg.Citations)
	webResultsJSON, _ := json.Marshal(msg.WebResults)
	relatedJSON, _ := json.Marshal(msg.RelatedQ)

	err := d.db.QueryRow(`
		INSERT INTO thread_messages (thread_id, role, content, backend_uuid,
			citations, web_results, related, tokens_used, duration_ms, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id
	`,
		msg.ThreadID, msg.Role, msg.Content, msg.BackendUUID,
		string(citationsJSON), string(webResultsJSON), string(relatedJSON),
		msg.TokensUsed, msg.DurationMs, time.Now(),
	).Scan(&msg.ID)
	if err != nil {
		return err
	}

	// Update thread metadata
	_, err = d.db.Exec(`
		UPDATE threads SET message_count = message_count + 1, updated_at = ? WHERE id = ?
	`, time.Now(), msg.ThreadID)
	return err
}

// GetThreadMessages returns all messages in a thread ordered by creation time.
func (d *DB) GetThreadMessages(threadID int) ([]ThreadMessage, error) {
	rows, err := d.db.Query(`
		SELECT id, thread_id, role, content, COALESCE(backend_uuid, ''),
			COALESCE(citations, '[]'), COALESCE(web_results, '[]'), COALESCE(related, '[]'),
			tokens_used, duration_ms, created_at
		FROM thread_messages
		WHERE thread_id = ?
		ORDER BY created_at ASC
	`, threadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []ThreadMessage
	for rows.Next() {
		var m ThreadMessage
		var citationsStr, webResultsStr, relatedStr string
		err := rows.Scan(&m.ID, &m.ThreadID, &m.Role, &m.Content, &m.BackendUUID,
			&citationsStr, &webResultsStr, &relatedStr,
			&m.TokensUsed, &m.DurationMs, &m.CreatedAt)
		if err != nil {
			continue
		}
		json.Unmarshal([]byte(citationsStr), &m.Citations)
		json.Unmarshal([]byte(webResultsStr), &m.WebResults)
		json.Unmarshal([]byte(relatedStr), &m.RelatedQ)
		messages = append(messages, m)
	}
	return messages, nil
}

// GetLastBackendUUID returns the most recent backend_uuid from a thread's assistant messages.
func (d *DB) GetLastBackendUUID(threadID int) (string, error) {
	var uuid sql.NullString
	err := d.db.QueryRow(`
		SELECT backend_uuid FROM thread_messages
		WHERE thread_id = ? AND role = 'assistant' AND backend_uuid IS NOT NULL AND backend_uuid != ''
		ORDER BY created_at DESC
		LIMIT 1
	`, threadID).Scan(&uuid)
	if err != nil {
		return "", err
	}
	return uuid.String, nil
}

// Close closes the database.
func (d *DB) Close() error {
	return d.db.Close()
}

func nilIfZero(n int) any {
	if n == 0 {
		return nil
	}
	return n
}
