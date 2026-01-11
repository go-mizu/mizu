package storebench

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-mizu/blueprints/table/feature/bases"
	"github.com/go-mizu/blueprints/table/feature/fields"
	"github.com/go-mizu/blueprints/table/feature/records"
	"github.com/go-mizu/blueprints/table/feature/tables"
	"github.com/go-mizu/blueprints/table/feature/users"
	"github.com/go-mizu/blueprints/table/feature/workspaces"
	"github.com/go-mizu/blueprints/table/store/duckdb"
	"github.com/go-mizu/blueprints/table/store/postgres"
	"github.com/go-mizu/blueprints/table/store/sqlite"
	"github.com/oklog/ulid/v2"
)

// Store interface abstracts the storage backends.
type Store interface {
	Users() users.Store
	Workspaces() workspaces.Store
	Bases() bases.Store
	Tables() tables.Store
	Fields() fields.Store
	Records() records.Store
	Close() error
}

// Runner orchestrates benchmark execution.
type Runner struct {
	cfg     *Config
	results *BenchmarkResults
}

// NewRunner creates a new benchmark runner.
func NewRunner(cfg *Config) *Runner {
	return &Runner{
		cfg:     cfg,
		results: NewBenchmarkResults(cfg),
	}
}

// Run executes all configured benchmarks.
func (r *Runner) Run() (*BenchmarkResults, error) {
	if err := r.cfg.Validate(); err != nil {
		return nil, err
	}

	// Ensure data directory exists
	if err := os.MkdirAll(r.cfg.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	for _, backend := range r.cfg.Backends {
		fmt.Printf("\n=== Benchmarking %s ===\n", backend)

		store, cleanup, err := r.openStore(backend)
		if err != nil {
			fmt.Printf("  SKIP: Failed to open %s: %v\n", backend, err)
			continue
		}

		if err := r.runBackendBenchmarks(backend, store); err != nil {
			fmt.Printf("  ERROR: %v\n", err)
		}

		cleanup()
	}

	r.results.Finish()
	return r.results, nil
}

func (r *Runner) openStore(backend string) (Store, func(), error) {
	switch backend {
	case "duckdb":
		dir := filepath.Join(r.cfg.DataDir, "duckdb_bench")
		os.RemoveAll(dir) // Clean start
		s, err := duckdb.Open(dir)
		if err != nil {
			return nil, nil, err
		}
		return &duckdbWrapper{s}, func() { s.Close(); os.RemoveAll(dir) }, nil

	case "postgres":
		if r.cfg.PostgresURL == "" {
			return nil, nil, fmt.Errorf("PostgreSQL URL not configured")
		}
		s, err := postgres.Open(r.cfg.PostgresURL)
		if err != nil {
			return nil, nil, err
		}
		return &postgresWrapper{s}, func() { s.Close() }, nil

	case "sqlite":
		dir := filepath.Join(r.cfg.DataDir, "sqlite_bench")
		os.RemoveAll(dir) // Clean start
		s, err := sqlite.Open(dir)
		if err != nil {
			return nil, nil, err
		}
		return &sqliteWrapper{s}, func() { s.Close(); os.RemoveAll(dir) }, nil

	default:
		return nil, nil, fmt.Errorf("unknown backend: %s", backend)
	}
}

func (r *Runner) runBackendBenchmarks(backend string, store Store) error {
	// Create test fixtures
	ctx := context.Background()
	testUser, testWorkspace, testBase, testTable, testFields, err := r.setupTestData(ctx, store)
	if err != nil {
		return fmt.Errorf("failed to setup test data: %w", err)
	}

	fixtures := &TestFixtures{
		User:      testUser,
		Workspace: testWorkspace,
		Base:      testBase,
		Table:     testTable,
		Fields:    testFields,
	}

	for _, scenario := range r.cfg.Scenarios {
		switch scenario {
		case "records":
			r.runRecordScenarios(backend, store, fixtures)
		case "batch":
			r.runBatchScenarios(backend, store, fixtures)
		case "query":
			r.runQueryScenarios(backend, store, fixtures)
		case "fields":
			r.runFieldScenarios(backend, store, fixtures)
		case "concurrent":
			r.runConcurrentScenarios(backend, store, fixtures)
		}
	}

	return nil
}

// TestFixtures holds test data used across scenarios.
type TestFixtures struct {
	User      *users.User
	Workspace *workspaces.Workspace
	Base      *bases.Base
	Table     *tables.Table
	Fields    []*fields.Field
}

func (r *Runner) setupTestData(ctx context.Context, store Store) (*users.User, *workspaces.Workspace, *bases.Base, *tables.Table, []*fields.Field, error) {
	// Create user
	user := &users.User{
		ID:           newID(),
		Email:        "bench@test.com",
		Name:         "Benchmark User",
		PasswordHash: "notused",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := store.Users().Create(ctx, user); err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("create user: %w", err)
	}

	// Create workspace
	ws := &workspaces.Workspace{
		ID:        newID(),
		Name:      "Benchmark Workspace",
		Slug:      "bench-ws",
		Plan:      "free",
		OwnerID:   user.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := store.Workspaces().Create(ctx, ws); err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("create workspace: %w", err)
	}

	// Create base
	base := &bases.Base{
		ID:          newID(),
		WorkspaceID: ws.ID,
		Name:        "Benchmark Base",
		Color:       "#3498db",
		CreatedBy:   user.ID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := store.Bases().Create(ctx, base); err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("create base: %w", err)
	}

	// Create table
	tbl := &tables.Table{
		ID:        newID(),
		BaseID:    base.ID,
		Name:      "Benchmark Table",
		Position:  1,
		CreatedBy: user.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := store.Tables().Create(ctx, tbl); err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("create table: %w", err)
	}

	// Create fields
	fieldDefs := []struct {
		name    string
		typ     string
		options string
	}{
		{"Name", fields.TypeSingleLineText, "{}"},
		{"Description", fields.TypeLongText, "{}"},
		{"Status", fields.TypeSingleSelect, "{}"},
		{"Priority", fields.TypeNumber, `{"precision":0}`},
		{"DueDate", fields.TypeDate, "{}"},
		{"Amount", fields.TypeCurrency, `{"symbol":"$"}`},
		{"Email", fields.TypeEmail, "{}"},
		{"URL", fields.TypeURL, "{}"},
		{"Rating", fields.TypeRating, `{"max":5}`},
		{"Active", fields.TypeCheckbox, "{}"},
	}

	createdFields := make([]*fields.Field, 0, len(fieldDefs))
	for i, fd := range fieldDefs {
		f := &fields.Field{
			ID:        newID(),
			TableID:   tbl.ID,
			Name:      fd.name,
			Type:      fd.typ,
			Options:   []byte(fd.options),
			Position:  i + 1,
			IsPrimary: i == 0,
			Width:     200,
			CreatedBy: user.ID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := store.Fields().Create(ctx, f); err != nil {
			return nil, nil, nil, nil, nil, fmt.Errorf("create field %s: %w", fd.name, err)
		}
		createdFields = append(createdFields, f)
	}

	// Set primary field
	if err := store.Tables().SetPrimaryField(ctx, tbl.ID, createdFields[0].ID); err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("set primary field: %w", err)
	}

	return user, ws, base, tbl, createdFields, nil
}

func newID() string {
	return ulid.Make().String()
}

// Wrappers to adapt concrete stores to the Store interface

type duckdbWrapper struct {
	s *duckdb.Store
}

func (w *duckdbWrapper) Users() users.Store         { return w.s.Users() }
func (w *duckdbWrapper) Workspaces() workspaces.Store { return w.s.Workspaces() }
func (w *duckdbWrapper) Bases() bases.Store         { return w.s.Bases() }
func (w *duckdbWrapper) Tables() tables.Store       { return w.s.Tables() }
func (w *duckdbWrapper) Fields() fields.Store       { return w.s.Fields() }
func (w *duckdbWrapper) Records() records.Store     { return w.s.Records() }
func (w *duckdbWrapper) Close() error               { return w.s.Close() }

type postgresWrapper struct {
	s *postgres.Store
}

func (w *postgresWrapper) Users() users.Store         { return w.s.Users() }
func (w *postgresWrapper) Workspaces() workspaces.Store { return w.s.Workspaces() }
func (w *postgresWrapper) Bases() bases.Store         { return w.s.Bases() }
func (w *postgresWrapper) Tables() tables.Store       { return w.s.Tables() }
func (w *postgresWrapper) Fields() fields.Store       { return w.s.Fields() }
func (w *postgresWrapper) Records() records.Store     { return w.s.Records() }
func (w *postgresWrapper) Close() error               { return w.s.Close() }

type sqliteWrapper struct {
	s *sqlite.Store
}

func (w *sqliteWrapper) Users() users.Store         { return w.s.Users() }
func (w *sqliteWrapper) Workspaces() workspaces.Store { return w.s.Workspaces() }
func (w *sqliteWrapper) Bases() bases.Store         { return w.s.Bases() }
func (w *sqliteWrapper) Tables() tables.Store       { return w.s.Tables() }
func (w *sqliteWrapper) Fields() fields.Store       { return w.s.Fields() }
func (w *sqliteWrapper) Records() records.Store     { return w.s.Records() }
func (w *sqliteWrapper) Close() error               { return w.s.Close() }
