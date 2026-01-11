// Package sqlite provides SQLite-based store implementations.
package sqlite

import (
	"context"
	"database/sql"
	_ "embed"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schema string

// Store wraps a SQLite connection and provides access to all stores.
type Store struct {
	db          *sql.DB
	users       *UsersStore
	workspaces  *WorkspacesStore
	bases       *BasesStore
	tables      *TablesStore
	fields      *FieldsStore
	records     *RecordsStore
	views       *ViewsStore
	operations  *OperationsStore
	shares      *SharesStore
	attachments *AttachmentsStore
	comments    *CommentsStore
	webhooks    *WebhooksStore
}

// Open opens a SQLite database at the given path and initializes all stores.
func Open(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	dbPath := filepath.Join(dataDir, "table.db")
	// SQLite connection string with pragmas for better performance
	// - WAL mode: enables concurrent reads during writes
	// - synchronous=NORMAL: good balance of safety and speed
	// - busy_timeout: wait up to 5s for locks
	// - foreign_keys: enforce referential integrity
	// - cache_size: 64MB page cache for faster reads
	// - mmap_size: 256MB memory-mapped I/O for large datasets
	// - temp_store=MEMORY: store temp tables in memory
	dsn := dbPath + "?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)&_pragma=cache_size(-64000)&_pragma=mmap_size(268435456)&_pragma=temp_store(MEMORY)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	// Set connection pool settings for SQLite
	db.SetMaxOpenConns(1) // SQLite works best with a single writer
	db.SetMaxIdleConns(1)

	// Initialize schema
	if _, err := db.ExecContext(context.Background(), schema); err != nil {
		db.Close()
		return nil, err
	}

	return &Store{
		db:          db,
		users:       NewUsersStore(db),
		workspaces:  NewWorkspacesStore(db),
		bases:       NewBasesStore(db),
		tables:      NewTablesStore(db),
		fields:      NewFieldsStore(db),
		records:     NewRecordsStore(db),
		views:       NewViewsStore(db),
		operations:  NewOperationsStore(db),
		shares:      NewSharesStore(db),
		attachments: NewAttachmentsStore(db),
		comments:    NewCommentsStore(db),
		webhooks:    NewWebhooksStore(db),
	}, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// DB returns the underlying database connection.
func (s *Store) DB() *sql.DB {
	return s.db
}

// Users returns the users store.
func (s *Store) Users() *UsersStore {
	return s.users
}

// Workspaces returns the workspaces store.
func (s *Store) Workspaces() *WorkspacesStore {
	return s.workspaces
}

// Bases returns the bases store.
func (s *Store) Bases() *BasesStore {
	return s.bases
}

// Tables returns the tables store.
func (s *Store) Tables() *TablesStore {
	return s.tables
}

// Fields returns the fields store.
func (s *Store) Fields() *FieldsStore {
	return s.fields
}

// Records returns the records store.
func (s *Store) Records() *RecordsStore {
	return s.records
}

// Views returns the views store.
func (s *Store) Views() *ViewsStore {
	return s.views
}

// Operations returns the operations store.
func (s *Store) Operations() *OperationsStore {
	return s.operations
}

// Shares returns the shares store.
func (s *Store) Shares() *SharesStore {
	return s.shares
}

// Attachments returns the attachments store.
func (s *Store) Attachments() *AttachmentsStore {
	return s.attachments
}

// Comments returns the comments store.
func (s *Store) Comments() *CommentsStore {
	return s.comments
}

// Webhooks returns the webhooks store.
func (s *Store) Webhooks() *WebhooksStore {
	return s.webhooks
}

// Tx executes a function within a transaction.
func (s *Store) Tx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}
