package duckdb

import (
	"context"
	"database/sql"
	_ "embed"
	"os"
	"path/filepath"

	_ "github.com/duckdb/duckdb-go/v2"
)

//go:embed schema.sql
var schema string

// Store wraps a DuckDB connection and provides access to all stores.
type Store struct {
	db          *sql.DB
	users       *UsersStore
	usermeta    *UsermetaStore
	sessions    *SessionsStore
	appPasswords *AppPasswordsStore
	posts       *PostsStore
	postmeta    *PostmetaStore
	terms       *TermsStore
	termTaxonomy *TermTaxonomyStore
	termRelationships *TermRelationshipsStore
	termmeta    *TermmetaStore
	comments    *CommentsStore
	commentmeta *CommentmetaStore
	options     *OptionsStore
	links       *LinksStore
	nonces      *NoncesStore
	cron        *CronStore
	transients  *TransientsStore
	seedMappings *SeedMappingsStore
}

// Open opens a DuckDB database at the given path and initializes all stores.
func Open(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	dbPath := filepath.Join(dataDir, "cms.db")
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, err
	}

	// Initialize schema
	if _, err := db.ExecContext(context.Background(), schema); err != nil {
		db.Close()
		return nil, err
	}

	return &Store{
		db:               db,
		users:            NewUsersStore(db),
		usermeta:         NewUsermetaStore(db),
		sessions:         NewSessionsStore(db),
		appPasswords:     NewAppPasswordsStore(db),
		posts:            NewPostsStore(db),
		postmeta:         NewPostmetaStore(db),
		terms:            NewTermsStore(db),
		termTaxonomy:     NewTermTaxonomyStore(db),
		termRelationships: NewTermRelationshipsStore(db),
		termmeta:         NewTermmetaStore(db),
		comments:         NewCommentsStore(db),
		commentmeta:      NewCommentmetaStore(db),
		options:          NewOptionsStore(db),
		links:            NewLinksStore(db),
		nonces:           NewNoncesStore(db),
		cron:             NewCronStore(db),
		transients:       NewTransientsStore(db),
		seedMappings:     NewSeedMappingsStore(db),
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

// Usermeta returns the usermeta store.
func (s *Store) Usermeta() *UsermetaStore {
	return s.usermeta
}

// Sessions returns the sessions store.
func (s *Store) Sessions() *SessionsStore {
	return s.sessions
}

// AppPasswords returns the application passwords store.
func (s *Store) AppPasswords() *AppPasswordsStore {
	return s.appPasswords
}

// Posts returns the posts store.
func (s *Store) Posts() *PostsStore {
	return s.posts
}

// Postmeta returns the postmeta store.
func (s *Store) Postmeta() *PostmetaStore {
	return s.postmeta
}

// Terms returns the terms store.
func (s *Store) Terms() *TermsStore {
	return s.terms
}

// TermTaxonomy returns the term taxonomy store.
func (s *Store) TermTaxonomy() *TermTaxonomyStore {
	return s.termTaxonomy
}

// TermRelationships returns the term relationships store.
func (s *Store) TermRelationships() *TermRelationshipsStore {
	return s.termRelationships
}

// Termmeta returns the termmeta store.
func (s *Store) Termmeta() *TermmetaStore {
	return s.termmeta
}

// Comments returns the comments store.
func (s *Store) Comments() *CommentsStore {
	return s.comments
}

// Commentmeta returns the commentmeta store.
func (s *Store) Commentmeta() *CommentmetaStore {
	return s.commentmeta
}

// Options returns the options store.
func (s *Store) Options() *OptionsStore {
	return s.options
}

// Links returns the links store.
func (s *Store) Links() *LinksStore {
	return s.links
}

// Nonces returns the nonces store.
func (s *Store) Nonces() *NoncesStore {
	return s.nonces
}

// Cron returns the cron store.
func (s *Store) Cron() *CronStore {
	return s.cron
}

// Transients returns the transients store.
func (s *Store) Transients() *TransientsStore {
	return s.transients
}

// SeedMappings returns the seed mappings store.
func (s *Store) SeedMappings() *SeedMappingsStore {
	return s.seedMappings
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
