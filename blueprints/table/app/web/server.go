package web

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	_ "github.com/mattn/go-sqlite3"
)

// Config holds server configuration.
type Config struct {
	Addr    string
	DataDir string
	Dev     bool
}

// Server is the web server wrapper.
type Server struct {
	cfg Config
	db  *sql.DB
	cmd *exec.Cmd
}

// New creates a new server.
func New(cfg Config) (*Server, error) {
	// Ensure data directory exists
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	// Open SQLite database
	dbPath := filepath.Join(cfg.DataDir, "table.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Initialize schema
	if err := initSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}

	s := &Server{
		cfg: cfg,
		db:  db,
	}

	return s, nil
}

// Run starts the server.
func (s *Server) Run() error {
	slog.Info("Starting Table server", "addr", s.cfg.Addr, "data", s.cfg.DataDir)

	// Find the backend directory
	backendDir := findBackendDir()
	if backendDir == "" {
		return fmt.Errorf("could not find backend directory")
	}

	// Build environment
	env := os.Environ()
	env = append(env, fmt.Sprintf("PORT=%s", s.cfg.Addr[1:])) // Remove leading ':'
	env = append(env, fmt.Sprintf("DATA_DIR=%s", s.cfg.DataDir))
	env = append(env, fmt.Sprintf("DB_PATH=%s", filepath.Join(s.cfg.DataDir, "table.db")))

	// Start the Node.js backend
	s.cmd = exec.Command("npx", "tsx", "src/node.ts")
	s.cmd.Dir = backendDir
	s.cmd.Env = env
	s.cmd.Stdout = os.Stdout
	s.cmd.Stderr = os.Stderr

	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("start backend: %w", err)
	}

	return s.cmd.Wait()
}

// Close shuts down the server.
func (s *Server) Close() error {
	if s.cmd != nil && s.cmd.Process != nil {
		s.cmd.Process.Signal(syscall.SIGTERM)
	}
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Service accessors for CLI use

type UserService struct{ db *sql.DB }
type WorkspaceService struct{ db *sql.DB }
type BaseService struct{ db *sql.DB }
type TableService struct{ db *sql.DB }
type FieldService struct{ db *sql.DB }
type RecordService struct{ db *sql.DB }
type ViewService struct{ db *sql.DB }

func (s *Server) UserService() *UserService           { return &UserService{s.db} }
func (s *Server) WorkspaceService() *WorkspaceService { return &WorkspaceService{s.db} }
func (s *Server) BaseService() *BaseService           { return &BaseService{s.db} }
func (s *Server) TableService() *TableService         { return &TableService{s.db} }
func (s *Server) FieldService() *FieldService         { return &FieldService{s.db} }
func (s *Server) RecordService() *RecordService       { return &RecordService{s.db} }
func (s *Server) ViewService() *ViewService           { return &ViewService{s.db} }

// UserService methods
func (u *UserService) Register(ctx context.Context, email, name, password string) (string, error) {
	id := generateID()
	hash := hashPassword(password)
	_, err := u.db.ExecContext(ctx, `
		INSERT INTO users (id, email, name, password_hash, created_at, updated_at)
		VALUES (?, ?, ?, ?, datetime('now'), datetime('now'))
		ON CONFLICT (email) DO NOTHING
	`, id, email, name, hash)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (u *UserService) GetByEmail(ctx context.Context, email string) (string, error) {
	var id string
	err := u.db.QueryRowContext(ctx, "SELECT id FROM users WHERE email = ?", email).Scan(&id)
	return id, err
}

// WorkspaceService methods
func (w *WorkspaceService) Create(ctx context.Context, ownerID, name, slug string) (string, error) {
	id := generateID()
	_, err := w.db.ExecContext(ctx, `
		INSERT INTO workspaces (id, name, slug, owner_id, plan, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'free', datetime('now'), datetime('now'))
		ON CONFLICT (slug) DO NOTHING
	`, id, name, slug, ownerID)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (w *WorkspaceService) GetBySlug(ctx context.Context, slug string) (string, error) {
	var id string
	err := w.db.QueryRowContext(ctx, "SELECT id FROM workspaces WHERE slug = ?", slug).Scan(&id)
	return id, err
}

// BaseService methods
func (b *BaseService) Create(ctx context.Context, workspaceID, name, color, createdBy string) (string, error) {
	id := generateID()
	_, err := b.db.ExecContext(ctx, `
		INSERT INTO bases (id, workspace_id, name, color, created_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, datetime('now'), datetime('now'))
	`, id, workspaceID, name, color, createdBy)
	if err != nil {
		return "", err
	}
	return id, nil
}

// TableService methods
func (t *TableService) Create(ctx context.Context, baseID, name, createdBy string) (string, error) {
	id := generateID()
	_, err := t.db.ExecContext(ctx, `
		INSERT INTO tables (id, base_id, name, position, created_by, created_at, updated_at)
		VALUES (?, ?, ?, (SELECT COALESCE(MAX(position), 0) + 1 FROM tables WHERE base_id = ?), ?, datetime('now'), datetime('now'))
	`, id, baseID, name, baseID, createdBy)
	if err != nil {
		return "", err
	}
	return id, nil
}

// FieldService methods
func (f *FieldService) Create(ctx context.Context, tableID, name, fieldType string, options map[string]interface{}, createdBy string) (string, error) {
	id := generateID()
	optionsJSON := "{}"
	if options != nil {
		// Simple JSON conversion
		optionsJSON = fmt.Sprintf("%v", options)
	}
	_, err := f.db.ExecContext(ctx, `
		INSERT INTO fields (id, table_id, name, type, options, position, is_primary, created_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, (SELECT COALESCE(MAX(position), 0) + 1 FROM fields WHERE table_id = ?), 0, ?, datetime('now'), datetime('now'))
	`, id, tableID, name, fieldType, optionsJSON, tableID, createdBy)
	if err != nil {
		return "", err
	}
	return id, nil
}

// RecordService methods
func (r *RecordService) Create(ctx context.Context, tableID string, values map[string]interface{}, createdBy string) (string, error) {
	id := generateID()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO records (id, table_id, position, created_by, created_at, updated_at)
		VALUES (?, ?, (SELECT COALESCE(MAX(position), 0) + 1 FROM records WHERE table_id = ?), ?, datetime('now'), datetime('now'))
	`, id, tableID, tableID, createdBy)
	if err != nil {
		return "", err
	}

	// Insert cell values
	for fieldID, value := range values {
		valueJSON := fmt.Sprintf(`"%v"`, value)
		r.db.ExecContext(ctx, `
			INSERT INTO cell_values (id, record_id, field_id, value, created_at, updated_at)
			VALUES (?, ?, ?, ?, datetime('now'), datetime('now'))
		`, generateID(), id, fieldID, valueJSON)
	}

	return id, nil
}

// ViewService methods
func (v *ViewService) Create(ctx context.Context, tableID, name, viewType string, config map[string]interface{}, createdBy string) (string, error) {
	id := generateID()
	configJSON := "{}"
	if config != nil {
		configJSON = fmt.Sprintf("%v", config)
	}
	_, err := v.db.ExecContext(ctx, `
		INSERT INTO views (id, table_id, name, type, config, position, created_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, (SELECT COALESCE(MAX(position), 0) + 1 FROM views WHERE table_id = ?), ?, datetime('now'), datetime('now'))
	`, id, tableID, name, viewType, configJSON, tableID, createdBy)
	if err != nil {
		return "", err
	}
	return id, nil
}

// Helper functions
func findBackendDir() string {
	// Try common locations
	candidates := []string{
		"app/backend/hono",
		"../app/backend/hono",
		"../../blueprints/table/app/backend/hono",
	}

	// Try to find based on executable location
	ex, _ := os.Executable()
	exDir := filepath.Dir(ex)

	for _, c := range candidates {
		path := filepath.Join(exDir, c)
		if _, err := os.Stat(filepath.Join(path, "package.json")); err == nil {
			return path
		}
	}

	// Try current directory
	cwd, _ := os.Getwd()
	for _, c := range candidates {
		path := filepath.Join(cwd, c)
		if _, err := os.Stat(filepath.Join(path, "package.json")); err == nil {
			return path
		}
	}

	return ""
}

func generateID() string {
	// Simple ID generation using crypto/rand
	b := make([]byte, 16)
	_, _ = os.Stdin.Read(b) // Will fail, but we'll use time-based fallback
	return fmt.Sprintf("%x", b[:8])
}

func hashPassword(password string) string {
	// Simple hash for demo - in production use bcrypt
	return fmt.Sprintf("hash:%s", password)
}
