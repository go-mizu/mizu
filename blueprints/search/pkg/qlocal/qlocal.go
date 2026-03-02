package qlocal

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	_ "modernc.org/sqlite"

	"gopkg.in/yaml.v3"
)

const (
	DefaultGlob              = "**/*.md"
	DefaultMultiGetMaxBytes  = 10 * 1024
	DefaultChunkSizeChars    = 3600
	DefaultChunkOverlapChars = 540
	DefaultChunkWindowChars  = 800
	DefaultEmbedModel        = "qlocal-hash-256"
	StrongSignalMinScore     = 0.85
	StrongSignalMinGap       = 0.15
)

var (
	structuredQueryPrefixRe = regexp.MustCompile(`(?i)^(lex|vec|hyde):\s*`)
	structuredExpandPrefix  = regexp.MustCompile(`(?i)^expand:\s*`)
)

type OutputFormat string

const (
	OutputCLI   OutputFormat = "cli"
	OutputCSV   OutputFormat = "csv"
	OutputMD    OutputFormat = "md"
	OutputXML   OutputFormat = "xml"
	OutputFiles OutputFormat = "files"
	OutputJSON  OutputFormat = "json"
)

type Collection struct {
	Path             string            `yaml:"path" json:"path"`
	Pattern          string            `yaml:"pattern,omitempty" json:"pattern,omitempty"`
	Context          map[string]string `yaml:"context,omitempty" json:"context,omitempty"`
	Update           string            `yaml:"update,omitempty" json:"update,omitempty"`
	IncludeByDefault *bool             `yaml:"includeByDefault,omitempty" json:"includeByDefault,omitempty"`
}

type Config struct {
	GlobalContext string                 `yaml:"global_context,omitempty" json:"global_context,omitempty"`
	Collections   map[string]*Collection `yaml:"collections" json:"collections"`
}

type NamedCollection struct {
	Name string `json:"name"`
	Collection
}

type App struct {
	IndexName string
	DB        *sql.DB
	dbPath    string
	cfgPath   string
}

type OpenOptions struct {
	IndexName  string
	DBPath     string
	ConfigPath string
}

type Status struct {
	TotalDocuments int64              `json:"totalDocuments"`
	NeedsEmbedding int64              `json:"needsEmbedding"`
	HasVectorIndex bool               `json:"hasVectorIndex"`
	Collections    []CollectionStatus `json:"collections"`
	MCP            MCPDaemonStatus    `json:"mcp"`
	DBPath         string             `json:"dbPath"`
	ConfigPath     string             `json:"configPath"`
}

type CollectionStatus struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	Pattern    string `json:"pattern"`
	Documents  int64  `json:"documents"`
	LastUpdate string `json:"lastUpdated,omitempty"`
}

type SearchResult struct {
	Filepath    string  `json:"filepath"`
	DisplayPath string  `json:"displayPath"`
	Title       string  `json:"title"`
	Hash        string  `json:"hash,omitempty"`
	DocID       string  `json:"docid,omitempty"`
	Collection  string  `json:"collection,omitempty"`
	ModifiedAt  string  `json:"modifiedAt,omitempty"`
	BodyLength  int     `json:"bodyLength,omitempty"`
	Body        string  `json:"body,omitempty"`
	Context     string  `json:"context,omitempty"`
	Score       float64 `json:"score"`
	Source      string  `json:"source,omitempty"`
	ChunkPos    int     `json:"chunkPos,omitempty"`
	ChunkText   string  `json:"chunkText,omitempty"`
	ChunkSeq    int     `json:"chunkSeq,omitempty"`
	RerankScore float64 `json:"rerankScore,omitempty"`
	RRFScore    float64 `json:"rrfScore,omitempty"`
}

type SearchOptions struct {
	Limit       int
	MinScore    float64
	Collections []string
	IncludeBody bool
}

type HybridOptions struct {
	Limit       int
	MinScore    float64
	Collections []string
}

type OutputOptions struct {
	Format      OutputFormat
	Full        bool
	LineNumbers bool
	Query       string
}

type Document struct {
	Filepath    string `json:"filepath"`
	DisplayPath string `json:"displayPath"`
	Title       string `json:"title"`
	Hash        string `json:"hash"`
	DocID       string `json:"docid"`
	Collection  string `json:"collection"`
	ModifiedAt  string `json:"modifiedAt"`
	BodyLength  int    `json:"bodyLength"`
	Body        string `json:"body,omitempty"`
	Context     string `json:"context,omitempty"`
}

type MultiGetResult struct {
	Doc        Document `json:"doc"`
	Skipped    bool     `json:"skipped"`
	SkipReason string   `json:"skipReason,omitempty"`
}

type ListEntry struct {
	Filepath    string `json:"filepath"`
	DisplayPath string `json:"displayPath"`
	Title       string `json:"title"`
	Collection  string `json:"collection"`
	ModifiedAt  string `json:"modifiedAt"`
	BodyLength  int    `json:"bodyLength"`
	DocID       string `json:"docid"`
}

type UpdateOptions struct {
	Pull bool
}

type UpdateStats struct {
	Collections int   `json:"collections"`
	Scanned     int   `json:"scanned"`
	Added       int   `json:"added"`
	Updated     int   `json:"updated"`
	Unchanged   int   `json:"unchanged"`
	Deactivated int   `json:"deactivated"`
	Errors      int   `json:"errors"`
	DurationMS  int64 `json:"durationMs"`
}

type EmbedOptions struct {
	Force bool
	Model string
}

type EmbedStats struct {
	Documents  int   `json:"documents"`
	Chunks     int   `json:"chunks"`
	Errors     int   `json:"errors"`
	DurationMS int64 `json:"durationMs"`
}

type CleanupStats struct {
	LLMCacheDeleted int64 `json:"llmCacheDeleted"`
	InactiveDeleted int64 `json:"inactiveDeleted"`
	OrphanedContent int64 `json:"orphanedContent"`
	OrphanedVectors int64 `json:"orphanedVectors"`
}

type StructuredSubSearch struct {
	Type  string `json:"type"`
	Query string `json:"query"`
	Line  int    `json:"line,omitempty"`
}

func Open(opts OpenOptions) (*App, error) {
	indexName := sanitizeIndexName(strings.TrimSpace(opts.IndexName))
	if indexName == "" {
		indexName = "index"
	}
	dbPath := opts.DBPath
	if dbPath == "" {
		dbPath = defaultDBPath(indexName)
	}
	cfgPath := opts.ConfigPath
	if cfgPath == "" {
		cfgPath = defaultConfigPath(indexName)
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		return nil, fmt.Errorf("create config dir: %w", err)
	}
	dsn := fmt.Sprintf("%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	app := &App{
		IndexName: indexName,
		DB:        db,
		dbPath:    dbPath,
		cfgPath:   cfgPath,
	}
	if err := app.initSchema(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return app, nil
}

func (a *App) Close() error {
	if a == nil || a.DB == nil {
		return nil
	}
	return a.DB.Close()
}

func (a *App) DBPath() string     { return a.dbPath }
func (a *App) ConfigPath() string { return a.cfgPath }

func defaultDBPath(indexName string) string {
	cacheDir := os.Getenv("XDG_CACHE_HOME")
	if cacheDir == "" {
		home, _ := os.UserHomeDir()
		cacheDir = filepath.Join(home, ".cache")
	}
	return filepath.Join(cacheDir, "mizu", "qlocal", indexName+".sqlite")
}

func defaultConfigPath(indexName string) string {
	cfgDir := os.Getenv("XDG_CONFIG_HOME")
	if cfgDir == "" {
		home, _ := os.UserHomeDir()
		cfgDir = filepath.Join(home, ".config")
	}
	return filepath.Join(cfgDir, "mizu", "qlocal", indexName+".yml")
}

func sanitizeIndexName(name string) string {
	if name == "" {
		return ""
	}
	abs := name
	if strings.ContainsRune(name, filepath.Separator) || strings.Contains(name, "/") {
		if p, err := filepath.Abs(name); err == nil {
			abs = p
		}
	}
	abs = strings.ReplaceAll(abs, "\\", "_")
	abs = strings.ReplaceAll(abs, "/", "_")
	abs = strings.TrimLeft(abs, "_")
	return abs
}

func (a *App) initSchema() error {
	stmts := []string{
		`PRAGMA journal_mode=WAL`,
		`PRAGMA foreign_keys=ON`,
		`CREATE TABLE IF NOT EXISTS content (
			hash TEXT PRIMARY KEY,
			doc TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS documents (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			collection TEXT NOT NULL,
			path TEXT NOT NULL,
			title TEXT NOT NULL,
			hash TEXT NOT NULL,
			created_at TEXT NOT NULL,
			modified_at TEXT NOT NULL,
			active INTEGER NOT NULL DEFAULT 1,
			UNIQUE(collection, path),
			FOREIGN KEY(hash) REFERENCES content(hash) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_documents_collection_active ON documents(collection, active)`,
		`CREATE INDEX IF NOT EXISTS idx_documents_hash ON documents(hash)`,
		`CREATE INDEX IF NOT EXISTS idx_documents_path_active ON documents(path, active)`,
		`CREATE TABLE IF NOT EXISTS llm_cache (
			hash TEXT PRIMARY KEY,
			result TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS content_vectors (
			hash TEXT NOT NULL,
			seq INTEGER NOT NULL,
			pos INTEGER NOT NULL DEFAULT 0,
			model TEXT NOT NULL,
			dims INTEGER NOT NULL,
			vec BLOB NOT NULL,
			embedded_at TEXT NOT NULL,
			PRIMARY KEY(hash, seq)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_content_vectors_hash ON content_vectors(hash)`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS documents_fts USING fts5(filepath, title, body, tokenize='porter unicode61')`,
		`CREATE TRIGGER IF NOT EXISTS qlocal_documents_ai AFTER INSERT ON documents
		 WHEN new.active = 1
		 BEGIN
		   INSERT INTO documents_fts(rowid, filepath, title, body)
		   SELECT new.id, new.collection || '/' || new.path, new.title, c.doc
		   FROM content c WHERE c.hash = new.hash;
		 END`,
		`CREATE TRIGGER IF NOT EXISTS qlocal_documents_ad AFTER DELETE ON documents
		 BEGIN
		   DELETE FROM documents_fts WHERE rowid = old.id;
		 END`,
		`CREATE TRIGGER IF NOT EXISTS qlocal_documents_au AFTER UPDATE ON documents
		 BEGIN
		   DELETE FROM documents_fts WHERE rowid = old.id;
		   INSERT INTO documents_fts(rowid, filepath, title, body)
		   SELECT new.id, new.collection || '/' || new.path, new.title, c.doc
		   FROM content c WHERE c.hash = new.hash AND new.active = 1;
		 END`,
	}
	for _, stmt := range stmts {
		if _, err := a.DB.Exec(stmt); err != nil {
			return fmt.Errorf("init schema: %w (stmt=%s)", err, oneLine(stmt))
		}
	}
	return nil
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func (a *App) LoadConfig() (*Config, error) {
	data, err := os.ReadFile(a.cfgPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Config{Collections: map[string]*Collection{}}, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config yaml %s: %w", a.cfgPath, err)
	}
	if cfg.Collections == nil {
		cfg.Collections = map[string]*Collection{}
	}
	for _, c := range cfg.Collections {
		if c.Pattern == "" {
			c.Pattern = DefaultGlob
		}
	}
	return &cfg, nil
}

func (a *App) SaveConfig(cfg *Config) error {
	if cfg.Collections == nil {
		cfg.Collections = map[string]*Collection{}
	}
	for _, c := range cfg.Collections {
		if c.Pattern == "" {
			c.Pattern = DefaultGlob
		}
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(a.cfgPath, data, 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func IsValidCollectionName(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}

func (a *App) CollectionList() ([]NamedCollection, error) {
	cfg, err := a.LoadConfig()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(cfg.Collections))
	for name := range cfg.Collections {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]NamedCollection, 0, len(names))
	for _, name := range names {
		c := cfg.Collections[name]
		out = append(out, NamedCollection{Name: name, Collection: *cloneCollection(c)})
	}
	return out, nil
}

func cloneCollection(c *Collection) *Collection {
	if c == nil {
		return &Collection{Pattern: DefaultGlob}
	}
	out := *c
	if out.Context != nil {
		out.Context = map[string]string{}
		for k, v := range c.Context {
			out.Context[k] = v
		}
	}
	if out.Pattern == "" {
		out.Pattern = DefaultGlob
	}
	return &out
}

func (a *App) CollectionAdd(dirPath, name, pattern string) (NamedCollection, error) {
	if strings.TrimSpace(dirPath) == "" {
		return NamedCollection{}, errors.New("path is required")
	}
	abs, err := filepath.Abs(expandHome(dirPath))
	if err != nil {
		return NamedCollection{}, fmt.Errorf("resolve path: %w", err)
	}
	abs, err = filepath.EvalSymlinks(abs)
	if err != nil {
		abs = filepath.Clean(abs)
	}
	st, err := os.Stat(abs)
	if err != nil {
		return NamedCollection{}, fmt.Errorf("stat path: %w", err)
	}
	if !st.IsDir() {
		return NamedCollection{}, fmt.Errorf("not a directory: %s", abs)
	}
	if pattern == "" {
		pattern = DefaultGlob
	}
	cfg, err := a.LoadConfig()
	if err != nil {
		return NamedCollection{}, err
	}
	if name == "" {
		name = pathBaseStable(abs)
	}
	if !IsValidCollectionName(name) {
		return NamedCollection{}, fmt.Errorf("invalid collection name %q (use letters, numbers, -, _)", name)
	}
	if existing, ok := cfg.Collections[name]; ok {
		// qmd behavior is effectively upsert; preserve context/update/include flag
		preserved := cloneCollection(existing)
		preserved.Path = filepath.ToSlash(abs)
		preserved.Pattern = pattern
		cfg.Collections[name] = preserved
	} else {
		cfg.Collections[name] = &Collection{
			Path:    filepath.ToSlash(abs),
			Pattern: pattern,
		}
	}
	if err := a.SaveConfig(cfg); err != nil {
		return NamedCollection{}, err
	}
	return NamedCollection{Name: name, Collection: *cloneCollection(cfg.Collections[name])}, nil
}

func pathBaseStable(abs string) string {
	base := filepath.Base(abs)
	base = strings.TrimSpace(base)
	base = strings.ReplaceAll(base, " ", "-")
	base = strings.ReplaceAll(base, ".", "-")
	base = strings.ToLower(base)
	base = regexp.MustCompile(`[^a-z0-9_-]+`).ReplaceAllString(base, "-")
	base = strings.Trim(base, "-")
	if base == "" {
		base = "collection"
	}
	return base
}

func (a *App) CollectionRemove(name string) error {
	cfg, err := a.LoadConfig()
	if err != nil {
		return err
	}
	if _, ok := cfg.Collections[name]; !ok {
		return fmt.Errorf("collection not found: %s", name)
	}
	delete(cfg.Collections, name)
	return a.SaveConfig(cfg)
}

func (a *App) CollectionRename(oldName, newName string) error {
	if !IsValidCollectionName(newName) {
		return fmt.Errorf("invalid collection name %q", newName)
	}
	cfg, err := a.LoadConfig()
	if err != nil {
		return err
	}
	c, ok := cfg.Collections[oldName]
	if !ok {
		return fmt.Errorf("collection not found: %s", oldName)
	}
	if _, exists := cfg.Collections[newName]; exists {
		return fmt.Errorf("collection already exists: %s", newName)
	}
	cfg.Collections[newName] = c
	delete(cfg.Collections, oldName)
	// Rename database document rows for continuity.
	if _, err := a.DB.Exec(`UPDATE documents SET collection=? WHERE collection=?`, newName, oldName); err != nil {
		return fmt.Errorf("rename database rows: %w", err)
	}
	return a.SaveConfig(cfg)
}

func (a *App) CollectionSetUpdate(name string, cmd string) error {
	cfg, err := a.LoadConfig()
	if err != nil {
		return err
	}
	c, ok := cfg.Collections[name]
	if !ok {
		return fmt.Errorf("collection not found: %s", name)
	}
	if strings.TrimSpace(cmd) == "" {
		c.Update = ""
	} else {
		c.Update = cmd
	}
	return a.SaveConfig(cfg)
}

func (a *App) CollectionSetIncludeByDefault(name string, include bool) error {
	cfg, err := a.LoadConfig()
	if err != nil {
		return err
	}
	c, ok := cfg.Collections[name]
	if !ok {
		return fmt.Errorf("collection not found: %s", name)
	}
	// Match qmd semantics: true is default, omit field.
	if include {
		c.IncludeByDefault = nil
	} else {
		v := false
		c.IncludeByDefault = &v
	}
	return a.SaveConfig(cfg)
}

func (a *App) CollectionShow(name string) (*NamedCollection, error) {
	cfg, err := a.LoadConfig()
	if err != nil {
		return nil, err
	}
	c, ok := cfg.Collections[name]
	if !ok {
		return nil, fmt.Errorf("collection not found: %s", name)
	}
	out := NamedCollection{Name: name, Collection: *cloneCollection(c)}
	return &out, nil
}

func (a *App) DefaultCollectionNames() ([]string, error) {
	cfg, err := a.LoadConfig()
	if err != nil {
		return nil, err
	}
	var names []string
	for name, c := range cfg.Collections {
		if c.IncludeByDefault != nil && !*c.IncludeByDefault {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

type ContextItem struct {
	Collection string `json:"collection"`
	Path       string `json:"path"`
	Context    string `json:"context"`
}

func (a *App) ContextList() ([]ContextItem, error) {
	cfg, err := a.LoadConfig()
	if err != nil {
		return nil, err
	}
	var out []ContextItem
	if strings.TrimSpace(cfg.GlobalContext) != "" {
		out = append(out, ContextItem{Collection: "*", Path: "/", Context: cfg.GlobalContext})
	}
	names := make([]string, 0, len(cfg.Collections))
	for name := range cfg.Collections {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		c := cfg.Collections[name]
		var keys []string
		for k := range c.Context {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			out = append(out, ContextItem{Collection: name, Path: k, Context: c.Context[k]})
		}
	}
	return out, nil
}

func (a *App) ContextAdd(pathArg, text, cwd string) (string, error) {
	if strings.TrimSpace(text) == "" {
		return "", errors.New("context text is required")
	}
	cfg, err := a.LoadConfig()
	if err != nil {
		return "", err
	}
	collectionName, prefix, global, err := a.resolveContextTarget(cfg, pathArg, cwd)
	if err != nil {
		return "", err
	}
	if global {
		cfg.GlobalContext = text
		if err := a.SaveConfig(cfg); err != nil {
			return "", err
		}
		return "global", nil
	}
	c := cfg.Collections[collectionName]
	if c.Context == nil {
		c.Context = map[string]string{}
	}
	c.Context[prefix] = text
	if err := a.SaveConfig(cfg); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%s", collectionName, prefix), nil
}

func (a *App) ContextRemove(pathArg, cwd string) (string, error) {
	cfg, err := a.LoadConfig()
	if err != nil {
		return "", err
	}
	collectionName, prefix, global, err := a.resolveContextTarget(cfg, pathArg, cwd)
	if err != nil {
		return "", err
	}
	if global {
		cfg.GlobalContext = ""
		if err := a.SaveConfig(cfg); err != nil {
			return "", err
		}
		return "global", nil
	}
	c := cfg.Collections[collectionName]
	if c == nil || c.Context == nil {
		return "", fmt.Errorf("context not found for %s", pathArg)
	}
	if _, ok := c.Context[prefix]; !ok {
		return "", fmt.Errorf("context not found for %s", pathArg)
	}
	delete(c.Context, prefix)
	if len(c.Context) == 0 {
		c.Context = nil
	}
	if err := a.SaveConfig(cfg); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%s", collectionName, prefix), nil
}

func (a *App) resolveContextTarget(cfg *Config, pathArg, cwd string) (collectionName, prefix string, global bool, err error) {
	p := strings.TrimSpace(pathArg)
	if p == "" || p == "." {
		p = cwd
	}
	if p == "/" {
		return "", "/", true, nil
	}
	if strings.HasPrefix(p, "qmd://") {
		col, rel, ok := parseVirtualPath(p)
		if !ok {
			return "", "", false, fmt.Errorf("invalid virtual path: %s", p)
		}
		if _, ok := cfg.Collections[col]; !ok {
			return "", "", false, fmt.Errorf("collection not found: %s", col)
		}
		prefix = "/"
		if rel != "" {
			prefix = "/" + strings.TrimPrefix(filepath.ToSlash(rel), "/")
		}
		return col, prefix, false, nil
	}

	// Try explicit collection/path form.
	s := filepath.ToSlash(expandHome(p))
	if !strings.HasPrefix(s, "/") {
		parts := strings.SplitN(s, "/", 2)
		if len(parts) > 0 {
			if _, ok := cfg.Collections[parts[0]]; ok {
				col := parts[0]
				rel := ""
				if len(parts) == 2 {
					rel = parts[1]
				}
				prefix = "/"
				if rel != "" {
					prefix = "/" + strings.TrimPrefix(filepath.ToSlash(rel), "/")
				}
				return col, prefix, false, nil
			}
		}
	}

	abs, absErr := filepath.Abs(expandHome(p))
	if absErr != nil {
		return "", "", false, fmt.Errorf("resolve path: %w", absErr)
	}
	abs = filepath.Clean(abs)
	abs = filepath.ToSlash(abs)
	bestName := ""
	bestPath := ""
	for name, c := range cfg.Collections {
		cp := filepath.ToSlash(filepath.Clean(expandHome(c.Path)))
		if abs == cp || strings.HasPrefix(abs, cp+"/") {
			if len(cp) > len(bestPath) {
				bestName = name
				bestPath = cp
			}
		}
	}
	if bestName == "" {
		return "", "", false, fmt.Errorf("path is not inside any collection: %s", p)
	}
	rel := strings.TrimPrefix(strings.TrimPrefix(abs, bestPath), "/")
	prefix = "/"
	if rel != "" {
		prefix = "/" + rel
	}
	return bestName, prefix, false, nil
}

func parseVirtualPath(v string) (collection string, rel string, ok bool) {
	s := strings.TrimSpace(v)
	if !strings.HasPrefix(s, "qmd://") {
		return "", "", false
	}
	s = strings.TrimPrefix(s, "qmd://")
	s = strings.TrimPrefix(s, "/")
	parts := strings.SplitN(s, "/", 2)
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		return "", "", false
	}
	collection = parts[0]
	if len(parts) == 2 {
		rel = parts[1]
	}
	return collection, rel, true
}

func buildVirtualPath(collection, rel string) string {
	if rel == "" {
		return "qmd://" + collection
	}
	return "qmd://" + collection + "/" + strings.TrimPrefix(filepath.ToSlash(rel), "/")
}

func expandHome(p string) string {
	if strings.HasPrefix(p, "~/") || p == "~" {
		home, _ := os.UserHomeDir()
		if p == "~" {
			return home
		}
		return filepath.Join(home, p[2:])
	}
	return p
}

func (a *App) contextForPath(collection, rel string) (string, error) {
	cfg, err := a.LoadConfig()
	if err != nil {
		return "", err
	}
	if cfg.Collections == nil {
		return strings.TrimSpace(cfg.GlobalContext), nil
	}
	c := cfg.Collections[collection]
	filePath := "/" + strings.TrimPrefix(filepath.ToSlash(rel), "/")
	bestPrefix := ""
	bestCtx := ""
	if c != nil && c.Context != nil {
		for prefix, ctx := range c.Context {
			p := prefix
			if !strings.HasPrefix(p, "/") {
				p = "/" + p
			}
			if filePath == p || strings.HasPrefix(filePath, p+"/") || p == "/" {
				if len(p) > len(bestPrefix) {
					bestPrefix = p
					bestCtx = ctx
				}
			}
		}
	}
	if bestCtx != "" {
		return bestCtx, nil
	}
	return strings.TrimSpace(cfg.GlobalContext), nil
}

func (a *App) Status() (*Status, error) {
	cfg, err := a.LoadConfig()
	if err != nil {
		return nil, err
	}
	var totalDocs int64
	if err := a.DB.QueryRow(`SELECT COUNT(*) FROM documents WHERE active=1`).Scan(&totalDocs); err != nil {
		return nil, err
	}
	var needsEmbedding int64
	if err := a.DB.QueryRow(`
		SELECT COUNT(DISTINCT d.hash)
		FROM documents d
		LEFT JOIN content_vectors v ON v.hash=d.hash AND v.seq=0
		WHERE d.active=1 AND v.hash IS NULL
	`).Scan(&needsEmbedding); err != nil {
		return nil, err
	}
	var hasVectors bool
	var tableName string
	if err := a.DB.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='content_vectors'`).Scan(&tableName); err == nil && tableName != "" {
		var n int64
		if err2 := a.DB.QueryRow(`SELECT COUNT(*) FROM content_vectors LIMIT 1`).Scan(&n); err2 == nil {
			hasVectors = n > 0
		}
	}
	names := make([]string, 0, len(cfg.Collections))
	for name := range cfg.Collections {
		names = append(names, name)
	}
	sort.Strings(names)
	var collections []CollectionStatus
	for _, name := range names {
		c := cfg.Collections[name]
		if c == nil {
			continue
		}
		var count int64
		var last sql.NullString
		if err := a.DB.QueryRow(`SELECT COUNT(*), COALESCE(MAX(modified_at), '') FROM documents WHERE collection=? AND active=1`, name).Scan(&count, &last); err != nil {
			return nil, err
		}
		pattern := c.Pattern
		if pattern == "" {
			pattern = DefaultGlob
		}
		collections = append(collections, CollectionStatus{
			Name: name, Path: c.Path, Pattern: pattern, Documents: count, LastUpdate: last.String,
		})
	}
	return &Status{
		TotalDocuments: totalDocs,
		NeedsEmbedding: needsEmbedding,
		HasVectorIndex: hasVectors,
		Collections:    collections,
		MCP:            GetMCPDaemonStatus(a.IndexName),
		DBPath:         a.dbPath,
		ConfigPath:     a.cfgPath,
	}, nil
}

func (a *App) Update(ctx context.Context, opts UpdateOptions) (*UpdateStats, error) {
	cfg, err := a.LoadConfig()
	if err != nil {
		return nil, err
	}
	stats := &UpdateStats{}
	start := time.Now()
	tx, err := a.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	names := make([]string, 0, len(cfg.Collections))
	for name := range cfg.Collections {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		coll := cfg.Collections[name]
		if coll == nil {
			continue
		}
		stats.Collections++
		if opts.Pull && strings.TrimSpace(coll.Update) != "" {
			cmd := exec.CommandContext(ctx, "sh", "-lc", coll.Update)
			cmd.Dir = expandHome(coll.Path)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				stats.Errors++
			}
		}
		if err := a.updateCollectionTx(ctx, tx, name, coll, stats); err != nil {
			stats.Errors++
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	tx = nil
	stats.DurationMS = time.Since(start).Milliseconds()
	return stats, nil
}

func (a *App) updateCollectionTx(ctx context.Context, tx *sql.Tx, name string, coll *Collection, stats *UpdateStats) error {
	root := expandHome(coll.Path)
	root, _ = filepath.Abs(root)
	root = filepath.Clean(root)
	pattern := coll.Pattern
	if pattern == "" {
		pattern = DefaultGlob
	}
	re, err := globToRegexp(pattern)
	if err != nil {
		return fmt.Errorf("invalid glob %q: %w", pattern, err)
	}

	activeRows, err := tx.QueryContext(ctx, `SELECT path FROM documents WHERE collection=? AND active=1`, name)
	if err != nil {
		return err
	}
	existing := map[string]struct{}{}
	for activeRows.Next() {
		var p string
		if err := activeRows.Scan(&p); err == nil {
			existing[p] = struct{}{}
		}
	}
	_ = activeRows.Close()

	seen := map[string]struct{}{}
	now := time.Now().UTC().Format(time.RFC3339)
	err = filepath.WalkDir(root, func(full string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			stats.Errors++
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		namePart := d.Name()
		if d.IsDir() && strings.HasPrefix(namePart, ".") && namePart != "." {
			if full == root {
				return nil
			}
			return filepath.SkipDir
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, full)
		if err != nil {
			stats.Errors++
			return nil
		}
		rel = filepath.ToSlash(rel)
		if !re.MatchString(rel) {
			return nil
		}
		data, err := os.ReadFile(full)
		if err != nil {
			stats.Errors++
			return nil
		}
		stats.Scanned++
		hash := hashContent(data)
		title := extractTitle(rel, data)
		st, _ := os.Stat(full)
		modifiedAt := now
		if st != nil {
			modifiedAt = st.ModTime().UTC().Format(time.RFC3339)
		}

		if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO content(hash, doc, created_at) VALUES(?,?,?)`, hash, string(data), now); err != nil {
			stats.Errors++
			return nil
		}
		var id int64
		var oldHash, oldTitle string
		err = tx.QueryRowContext(ctx, `SELECT id, hash, title FROM documents WHERE collection=? AND path=?`, name, rel).Scan(&id, &oldHash, &oldTitle)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO documents(collection, path, title, hash, created_at, modified_at, active)
				VALUES(?,?,?,?,?,?,1)
			`, name, rel, title, hash, now, modifiedAt); err != nil {
				stats.Errors++
				return nil
			}
			stats.Added++
		case err != nil:
			stats.Errors++
			return nil
		default:
			if oldHash != hash || oldTitle != title {
				if _, err := tx.ExecContext(ctx, `
					UPDATE documents SET title=?, hash=?, modified_at=?, active=1 WHERE id=?
				`, title, hash, modifiedAt, id); err != nil {
					stats.Errors++
					return nil
				}
				// Invalidate stale vectors for changed content hash if doc now points elsewhere.
				stats.Updated++
			} else {
				if _, err := tx.ExecContext(ctx, `UPDATE documents SET modified_at=?, active=1 WHERE id=?`, modifiedAt, id); err != nil {
					stats.Errors++
					return nil
				}
				stats.Unchanged++
			}
		}
		seen[rel] = struct{}{}
		return nil
	})
	if err != nil {
		return err
	}
	for p := range existing {
		if _, ok := seen[p]; ok {
			continue
		}
		if _, err := tx.ExecContext(ctx, `UPDATE documents SET active=0 WHERE collection=? AND path=?`, name, p); err == nil {
			stats.Deactivated++
		}
	}
	return nil
}

func hashContent(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func extractTitle(rel string, data []byte) string {
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "#") {
			line = strings.TrimLeft(line, "#")
			line = strings.TrimSpace(line)
			if line != "" {
				return line
			}
		}
	}
	base := path.Base(filepath.ToSlash(rel))
	base = strings.TrimSuffix(base, path.Ext(base))
	if base == "" {
		base = rel
	}
	return base
}

func globToRegexp(glob string) (*regexp.Regexp, error) {
	if strings.TrimSpace(glob) == "" {
		glob = DefaultGlob
	}
	var b strings.Builder
	b.WriteString("^")
	runes := []rune(filepath.ToSlash(glob))
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		switch r {
		case '*':
			if i+1 < len(runes) && runes[i+1] == '*' {
				// `**/` should match zero or more path segments (including none),
				// so `**/*.md` matches `README.md` and `docs/README.md`.
				if i+2 < len(runes) && runes[i+2] == '/' {
					b.WriteString(`(?:.*/)?`)
					i += 2
					continue
				}
				b.WriteString(".*")
				i++
			} else {
				b.WriteString(`[^/]*`)
			}
		case '?':
			b.WriteString(`[^/]`)
		case '.', '+', '(', ')', '[', ']', '{', '}', '^', '$', '|', '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteString("$")
	return regexp.Compile(b.String())
}

func docID(hash string) string {
	if len(hash) < 6 {
		return hash
	}
	return hash[:6]
}

func (a *App) Embed(ctx context.Context, opts EmbedOptions) (*EmbedStats, error) {
	model := opts.Model
	if model == "" {
		model = DefaultEmbedModel
	}
	if opts.Force {
		if _, err := a.DB.ExecContext(ctx, `DELETE FROM content_vectors`); err != nil {
			return nil, err
		}
	}
	rows, err := a.DB.QueryContext(ctx, `
		SELECT d.hash, c.doc, d.title
		FROM documents d
		JOIN content c ON c.hash = d.hash
		LEFT JOIN content_vectors v ON v.hash = d.hash AND v.seq = 0
		WHERE d.active = 1 AND v.hash IS NULL
		GROUP BY d.hash
	`)
	if err != nil {
		return nil, err
	}
	type embedDoc struct {
		Hash  string
		Body  string
		Title string
	}
	var docs []embedDoc
	for rows.Next() {
		var d embedDoc
		if err := rows.Scan(&d.Hash, &d.Body, &d.Title); err != nil {
			_ = rows.Close()
			return nil, err
		}
		docs = append(docs, d)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	stats := &EmbedStats{}
	start := time.Now()
	tx, err := a.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()
	now := time.Now().UTC().Format(time.RFC3339)
	for _, d := range docs {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		stats.Documents++
		chunks := chunkDocument(d.Body)
		if len(chunks) == 0 {
			chunks = []Chunk{{Text: d.Body, Pos: 0}}
		}
		formatted := make([]string, 0, len(chunks))
		for _, ch := range chunks {
			formatted = append(formatted, formatDocForEmbedding(ch.Text, d.Title))
		}
		var backendVecs [][]float32
		effectiveModel := model
		if vecs, ok, err := a.embedTextsWithBackend(ctx, formatted); err == nil && ok && len(vecs) == len(chunks) {
			backendVecs = vecs
			if effectiveModel == DefaultEmbedModel {
				embedModelName := strings.TrimSpace(os.Getenv("QLOCAL_EMBED_MODEL"))
				if embedModelName == "" {
					embedModelName = "default"
				}
				effectiveModel = "llm:" + embedModelName
			}
		}
		for i, ch := range chunks {
			vec := hashEmbed(formatted[i], 256)
			if i < len(backendVecs) && len(backendVecs[i]) > 0 {
				vec = backendVecs[i]
			}
			if err := insertVectorRow(ctx, tx, d.Hash, i, ch.Pos, effectiveModel, vec, now); err != nil {
				stats.Errors++
				continue
			}
			stats.Chunks++
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	tx = nil
	stats.DurationMS = time.Since(start).Milliseconds()
	return stats, nil
}

func insertVectorRow(ctx context.Context, tx *sql.Tx, hash string, seq, pos int, model string, vec []float32, ts string) error {
	buf := encodeVector(vec)
	_, err := tx.ExecContext(ctx, `
		INSERT OR REPLACE INTO content_vectors(hash, seq, pos, model, dims, vec, embedded_at)
		VALUES(?,?,?,?,?,?,?)
	`, hash, seq, pos, model, len(vec), buf, ts)
	return err
}

func encodeVector(v []float32) []byte {
	buf := make([]byte, len(v)*4)
	for i, x := range v {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(x))
	}
	return buf
}

func decodeVector(b []byte) []float32 {
	if len(b)%4 != 0 {
		return nil
	}
	out := make([]float32, len(b)/4)
	for i := range out {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return out
}

func formatQueryForEmbedding(q string) string {
	return "task: search result | query: " + q
}

func formatDocForEmbedding(text, title string) string {
	if strings.TrimSpace(title) == "" {
		title = "none"
	}
	return "title: " + title + " | text: " + text
}

func hashEmbed(text string, dims int) []float32 {
	if dims <= 0 {
		dims = 256
	}
	vec := make([]float32, dims)
	tokens := tokenize(text)
	if len(tokens) == 0 {
		return vec
	}
	for _, tok := range tokens {
		h := fnv1a(tok)
		idx := int(h % uint64(dims))
		sign := float32(1.0)
		if (h>>8)&1 == 1 {
			sign = -1
		}
		vec[idx] += sign
	}
	var sum float64
	for _, x := range vec {
		sum += float64(x * x)
	}
	if sum == 0 {
		return vec
	}
	norm := float32(1 / math.Sqrt(sum))
	for i := range vec {
		vec[i] *= norm
	}
	return vec
}

func fnv1a(s string) uint64 {
	const offset uint64 = 1469598103934665603
	const prime uint64 = 1099511628211
	h := offset
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= prime
	}
	return h
}

func tokenize(s string) []string {
	var out []string
	var b strings.Builder
	flush := func() {
		if b.Len() == 0 {
			return
		}
		t := strings.ToLower(b.String())
		if len(t) >= 2 {
			out = append(out, t)
		}
		b.Reset()
	}
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '\'' {
			b.WriteRune(unicode.ToLower(r))
			continue
		}
		flush()
	}
	flush()
	return out
}

type Chunk struct {
	Text string
	Pos  int
}

func chunkDocument(text string) []Chunk {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	if len(text) <= DefaultChunkSizeChars {
		return []Chunk{{Text: text, Pos: 0}}
	}
	var chunks []Chunk
	start := 0
	for start < len(text) {
		end := start + DefaultChunkSizeChars
		if end >= len(text) {
			chunks = append(chunks, Chunk{Text: text[start:], Pos: start})
			break
		}
		cut := bestCutoff(text, start, end)
		if cut <= start {
			cut = end
		}
		chunks = append(chunks, Chunk{Text: text[start:cut], Pos: start})
		if cut >= len(text) {
			break
		}
		next := cut - DefaultChunkOverlapChars
		if next <= start {
			next = cut
		}
		start = next
	}
	return chunks
}

func bestCutoff(text string, start, target int) int {
	windowStart := target - DefaultChunkWindowChars
	if windowStart < start {
		windowStart = start
	}
	best := -1
	bestScore := -1
	for i := target; i >= windowStart; i-- {
		if i <= start || i >= len(text) {
			continue
		}
		score := 0
		switch text[i] {
		case '\n':
			score = 10
		case ' ', '\t':
			score = 2
		default:
			continue
		}
		if i > 0 && text[i-1] == '\n' {
			score += 10
		}
		if i+1 < len(text) && text[i+1] == '#' {
			score += 15
		}
		dist := target - i
		score -= dist / 50
		if score > bestScore {
			bestScore = score
			best = i
		}
	}
	if best > 0 {
		return best
	}
	return target
}

func (a *App) SearchFTS(query string, opts SearchOptions) ([]SearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	ftsQuery := buildFTS5Query(query)
	if ftsQuery == "" {
		return nil, nil
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = 20
	}
	collections := opts.Collections
	if len(collections) == 0 {
		var err error
		collections, err = a.DefaultCollectionNames()
		if err != nil {
			return nil, err
		}
	}
	baseSQL := `
		SELECT
		  d.collection,
		  d.path,
		  d.title,
		  d.hash,
		  d.modified_at,
		  content.doc,
		  bm25(documents_fts, 10.0, 1.0) as bm25_score
		FROM documents_fts f
		JOIN documents d ON d.id = f.rowid
		JOIN content ON content.hash = d.hash
		WHERE documents_fts MATCH ? AND d.active = 1
	`
	args := []any{ftsQuery}
	if len(collections) > 0 {
		ph := make([]string, len(collections))
		for i, c := range collections {
			ph[i] = "?"
			args = append(args, c)
		}
		baseSQL += ` AND d.collection IN (` + strings.Join(ph, ",") + `)`
	}
	baseSQL += ` ORDER BY bm25_score ASC LIMIT ?`
	args = append(args, limit)
	rows, err := a.DB.Query(baseSQL, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SearchResult
	for rows.Next() {
		var collection, rel, title, hash, modifiedAt, body string
		var bm25 float64
		if err := rows.Scan(&collection, &rel, &title, &hash, &modifiedAt, &body, &bm25); err != nil {
			return nil, err
		}
		ctxStr, _ := a.contextForPath(collection, rel)
		score := math.Abs(bm25) / (1 + math.Abs(bm25))
		if score < opts.MinScore {
			continue
		}
		out = append(out, SearchResult{
			Filepath:    buildVirtualPath(collection, rel),
			DisplayPath: collection + "/" + rel,
			Title:       title,
			Hash:        hash,
			DocID:       docID(hash),
			Collection:  collection,
			ModifiedAt:  modifiedAt,
			BodyLength:  len(body),
			Body:        conditionalBody(opts.IncludeBody, body),
			Context:     ctxStr,
			Score:       score,
			Source:      "fts",
		})
	}
	return out, nil
}

func conditionalBody(include bool, body string) string {
	if include {
		return body
	}
	return ""
}

func sanitizeFTS5Term(term string) string {
	var b strings.Builder
	for _, r := range term {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '\'' {
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}

func ValidateLexQuery(query string) error {
	if strings.ContainsAny(query, "\r\n") {
		return errors.New("lex queries must be single-line")
	}
	if strings.Count(query, `"`)%2 != 0 {
		return errors.New(`lex query has unmatched "`)
	}
	return nil
}

func ValidateSemanticQuery(query string) error {
	if regexp.MustCompile(`-\w|-"`).MatchString(query) {
		return errors.New("negation (-term) is not supported in vec/hyde queries")
	}
	return nil
}

func buildFTS5Query(query string) string {
	var positive []string
	var negative []string
	s := strings.TrimSpace(query)
	for i := 0; i < len(s); {
		for i < len(s) && unicode.IsSpace(rune(s[i])) {
			i++
		}
		if i >= len(s) {
			break
		}
		neg := false
		if s[i] == '-' {
			neg = true
			i++
		}
		if i < len(s) && s[i] == '"' {
			i++
			start := i
			for i < len(s) && s[i] != '"' {
				i++
			}
			phrase := strings.TrimSpace(s[start:i])
			if i < len(s) && s[i] == '"' {
				i++
			}
			parts := tokenize(phrase)
			if len(parts) == 0 {
				continue
			}
			t := `"` + strings.Join(parts, " ") + `"`
			if neg {
				negative = append(negative, t)
			} else {
				positive = append(positive, t)
			}
			continue
		}
		start := i
		for i < len(s) && !unicode.IsSpace(rune(s[i])) && s[i] != '"' {
			i++
		}
		term := sanitizeFTS5Term(s[start:i])
		if term == "" {
			continue
		}
		t := `"` + term + `"*`
		if neg {
			negative = append(negative, t)
		} else {
			positive = append(positive, t)
		}
	}
	if len(positive) == 0 {
		return ""
	}
	result := strings.Join(positive, " AND ")
	for _, n := range negative {
		result += " NOT " + n
	}
	return result
}

func (a *App) VectorSearch(query string, opts SearchOptions) ([]SearchResult, error) {
	return a.VectorSearchContext(context.Background(), query, opts)
}

func (a *App) VectorSearchContext(ctx context.Context, query string, opts SearchOptions) ([]SearchResult, error) {
	base, err := a.vectorSearchRaw(ctx, query, opts)
	if err != nil {
		return nil, err
	}
	expanded, ok, err := a.expandQueryCached(ctx, query)
	if err != nil || !ok {
		return base, err
	}
	best := make(map[string]SearchResult, len(base))
	for _, r := range base {
		best[r.Filepath] = r
	}
	for _, sub := range expanded {
		if sub.Type == "lex" {
			continue
		}
		subres, err := a.vectorSearchRaw(ctx, sub.Query, opts)
		if err != nil {
			continue
		}
		for _, r := range subres {
			if cur, exists := best[r.Filepath]; !exists || r.Score > cur.Score {
				best[r.Filepath] = r
			}
		}
	}
	out := make([]SearchResult, 0, len(best))
	for _, r := range best {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	limit := opts.Limit
	if limit <= 0 {
		limit = 10
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (a *App) vectorSearchRaw(ctx context.Context, query string, opts SearchOptions) ([]SearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = 10
	}
	collections := opts.Collections
	if len(collections) == 0 {
		var err error
		collections, err = a.DefaultCollectionNames()
		if err != nil {
			return nil, err
		}
	}
	qvec := hashEmbed(formatQueryForEmbedding(query), 256)
	if vecs, ok, err := a.embedTextsWithBackend(ctx, []string{formatQueryForEmbedding(query)}); err == nil && ok && len(vecs) == 1 && len(vecs[0]) > 0 {
		qvec = vecs[0]
	}
	rowsSQL := `
		SELECT d.collection, d.path, d.title, d.hash, d.modified_at, content.doc,
		       cv.seq, cv.pos, cv.vec
		FROM content_vectors cv
		JOIN documents d ON d.hash = cv.hash AND d.active=1
		JOIN content ON content.hash = d.hash
	`
	var args []any
	if len(collections) > 0 {
		ph := make([]string, len(collections))
		for i, c := range collections {
			ph[i] = "?"
			args = append(args, c)
		}
		rowsSQL += ` WHERE d.collection IN (` + strings.Join(ph, ",") + `)`
	}
	rows, err := a.DB.Query(rowsSQL, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	best := map[string]SearchResult{}
	for rows.Next() {
		var collection, rel, title, hash, modifiedAt, body string
		var seq, pos int
		var vecBlob []byte
		if err := rows.Scan(&collection, &rel, &title, &hash, &modifiedAt, &body, &seq, &pos, &vecBlob); err != nil {
			return nil, err
		}
		vec := decodeVector(vecBlob)
		if len(vec) == 0 {
			continue
		}
		score := cosine(qvec, vec)
		if score < opts.MinScore {
			continue
		}
		key := collection + "/" + rel
		ctxStr, _ := a.contextForPath(collection, rel)
		chunks := chunkDocument(body)
		chunkText := ""
		if seq >= 0 && seq < len(chunks) {
			chunkText = chunks[seq].Text
		}
		cur, ok := best[key]
		if !ok || score > cur.Score {
			best[key] = SearchResult{
				Filepath:    buildVirtualPath(collection, rel),
				DisplayPath: key,
				Title:       title,
				Hash:        hash,
				DocID:       docID(hash),
				Collection:  collection,
				ModifiedAt:  modifiedAt,
				BodyLength:  len(body),
				Body:        conditionalBody(opts.IncludeBody, body),
				Context:     ctxStr,
				Score:       score,
				Source:      "vec",
				ChunkPos:    pos,
				ChunkSeq:    seq,
				ChunkText:   chunkText,
			}
		}
	}
	out := make([]SearchResult, 0, len(best))
	for _, r := range best {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func cosine(a, b []float32) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	var dot float64
	for i := range a {
		dot += float64(a[i] * b[i])
	}
	// hashEmbed already normalizes.
	return dot
}

func (a *App) Query(query string, opts HybridOptions) ([]SearchResult, error) {
	return a.QueryContext(context.Background(), query, opts)
}

func (a *App) QueryContext(ctx context.Context, query string, opts HybridOptions) ([]SearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = 10
	}
	collections := opts.Collections
	if len(collections) == 0 {
		var err error
		collections, err = a.DefaultCollectionNames()
		if err != nil {
			return nil, err
		}
	}

	structured, serr := ParseStructuredQuery(query)
	if serr != nil {
		return nil, serr
	}
	if len(structured) == 0 {
		if normalized, ok, err := normalizeStandaloneExpandQuery(query); err != nil {
			return nil, err
		} else if ok {
			query = normalized
		}
	}
	type rankedItem struct {
		Result SearchResult
	}
	var resultLists [][]SearchResult
	if len(structured) == 0 {
		fts, err := a.SearchFTS(query, SearchOptions{Limit: 20, Collections: collections, IncludeBody: true})
		if err != nil {
			return nil, err
		}
		if len(fts) > 0 {
			resultLists = append(resultLists, fts)
		}
		vec, err := a.vectorSearchRaw(ctx, query, SearchOptions{Limit: 20, Collections: collections, IncludeBody: true})
		if err == nil && len(vec) > 0 {
			resultLists = append(resultLists, vec)
		}
		// qmd parity path: optional typed query expansion via configured LLM backend.
		hasStrongSignal := false
		if len(fts) > 0 {
			top := fts[0].Score
			second := 0.0
			if len(fts) > 1 {
				second = fts[1].Score
			}
			hasStrongSignal = top >= StrongSignalMinScore && (top-second) >= StrongSignalMinGap
		}
		if expanded, ok, err := a.expandQueryCached(ctx, query); err == nil && ok && !hasStrongSignal {
			for _, sub := range expanded {
				switch sub.Type {
				case "lex":
					fts, err := a.SearchFTS(sub.Query, SearchOptions{Limit: 20, Collections: collections, IncludeBody: true})
					if err == nil && len(fts) > 0 {
						resultLists = append(resultLists, fts)
					}
				case "vec", "hyde":
					vec, err := a.vectorSearchRaw(ctx, sub.Query, SearchOptions{Limit: 20, Collections: collections, IncludeBody: true})
					if err == nil && len(vec) > 0 {
						resultLists = append(resultLists, vec)
					}
				}
			}
		}
	} else {
		for _, sub := range structured {
			switch sub.Type {
			case "lex":
				if err := ValidateLexQuery(sub.Query); err != nil {
					return nil, fmt.Errorf("line %d (lex): %w", sub.Line, err)
				}
				fts, err := a.SearchFTS(sub.Query, SearchOptions{Limit: 20, Collections: collections, IncludeBody: true})
				if err != nil {
					return nil, err
				}
				if len(fts) > 0 {
					resultLists = append(resultLists, fts)
				}
			case "vec", "hyde":
				if err := ValidateSemanticQuery(sub.Query); err != nil {
					return nil, fmt.Errorf("line %d (%s): %w", sub.Line, sub.Type, err)
				}
				vec, err := a.vectorSearchRaw(ctx, sub.Query, SearchOptions{Limit: 20, Collections: collections, IncludeBody: true})
				if err != nil {
					return nil, err
				}
				if len(vec) > 0 {
					resultLists = append(resultLists, vec)
				}
			}
		}
	}
	if len(resultLists) == 0 {
		return nil, nil
	}
	fused := reciprocalRankFusion(resultLists)
	if len(fused) == 0 {
		return nil, nil
	}
	candidateLimit := 40
	if len(fused) < candidateLimit {
		candidateLimit = len(fused)
	}
	candidates := fused[:candidateLimit]
	primaryQuery := query
	if len(structured) > 0 {
		for _, s := range structured {
			if s.Type == "lex" || s.Type == "vec" {
				primaryQuery = s.Query
				break
			}
		}
		if primaryQuery == query && len(structured) > 0 {
			primaryQuery = structured[0].Query
		}
	}
	terms := tokenize(primaryQuery)
	queryVec := hashEmbed(formatQueryForEmbedding(primaryQuery), 256)
	if vecs, ok, err := a.embedTextsWithBackend(ctx, []string{formatQueryForEmbedding(primaryQuery)}); err == nil && ok && len(vecs) == 1 && len(vecs[0]) > 0 {
		queryVec = vecs[0]
	}
	final := make([]SearchResult, 0, len(candidates))
	rerankDocs := make([]RerankDoc, 0, len(candidates))
	rerankFileIndex := make([]int, 0, len(candidates))
	for rank, c := range candidates {
		body := c.Body
		if body == "" {
			doc, err := a.Get(c.Filepath, GetOptions{Full: true})
			if err == nil {
				body = doc.Body
			}
		}
		chunks := chunkDocument(body)
		bestChunkText := c.ChunkText
		bestChunkPos := c.ChunkPos
		bestChunkScore := 0.0
		if len(chunks) > 0 {
			for _, ch := range chunks {
				overlap := keywordOverlapScore(ch.Text, terms)
				sem := cosine(queryVec, hashEmbed(formatDocForEmbedding(ch.Text, c.Title), 256))
				score := 0.7*overlap + 0.3*sem
				if score > bestChunkScore {
					bestChunkScore = score
					bestChunkText = ch.Text
					bestChunkPos = ch.Pos
				}
			}
		}
		rrfRank := rank + 1
		var retrievalWeight float64
		switch {
		case rrfRank <= 3:
			retrievalWeight = 0.75
		case rrfRank <= 10:
			retrievalWeight = 0.60
		default:
			retrievalWeight = 0.40
		}
		rrfScore := 1.0 / float64(rrfRank)
		blended := retrievalWeight*rrfScore + (1-retrievalWeight)*bestChunkScore
		c.Score = blended
		c.RerankScore = bestChunkScore
		c.RRFScore = rrfScore
		c.ChunkText = bestChunkText
		c.ChunkPos = bestChunkPos
		final = append(final, c)
		if strings.TrimSpace(bestChunkText) != "" {
			rerankDocs = append(rerankDocs, RerankDoc{File: c.Filepath, Text: bestChunkText})
			rerankFileIndex = append(rerankFileIndex, len(final)-1)
		}
	}
	// Optional LLM reranking (cached) with qmd-like position-aware blending.
	if scores, ok, err := a.rerankCached(ctx, primaryQuery, rerankDocs); err == nil && ok {
		for i, score := range scores {
			if i >= len(rerankFileIndex) {
				break
			}
			idx := rerankFileIndex[i]
			if idx < 0 || idx >= len(final) {
				continue
			}
			if score < 0 {
				score = 0
			}
			if score > 1 {
				score = 1
			}
			rrfRank := idx + 1
			var retrievalWeight float64
			switch {
			case rrfRank <= 3:
				retrievalWeight = 0.75
			case rrfRank <= 10:
				retrievalWeight = 0.60
			default:
				retrievalWeight = 0.40
			}
			final[idx].RerankScore = score
			final[idx].Score = retrievalWeight*final[idx].RRFScore + (1-retrievalWeight)*score
		}
	}
	sort.Slice(final, func(i, j int) bool { return final[i].Score > final[j].Score })
	if opts.MinScore > 0 {
		filtered := final[:0]
		for _, r := range final {
			if r.Score >= opts.MinScore {
				filtered = append(filtered, r)
			}
		}
		final = filtered
	}
	if len(final) > limit {
		final = final[:limit]
	}
	return final, nil
}

func keywordOverlapScore(text string, terms []string) float64 {
	if len(terms) == 0 {
		return 0
	}
	lower := strings.ToLower(text)
	hits := 0
	for _, t := range terms {
		if len(t) < 3 {
			continue
		}
		if strings.Contains(lower, t) {
			hits++
		}
	}
	if hits == 0 {
		return 0
	}
	return float64(hits) / float64(len(terms))
}

func reciprocalRankFusion(lists [][]SearchResult) []SearchResult {
	type item struct {
		r       SearchResult
		score   float64
		topRank int
	}
	m := map[string]*item{}
	for li, list := range lists {
		weight := 1.0
		if li < 2 {
			weight = 2.0
		}
		for rank, r := range list {
			k := r.Filepath
			contrib := weight / float64(60+rank+1)
			if it, ok := m[k]; ok {
				it.score += contrib
				if rank < it.topRank {
					it.topRank = rank
				}
				// Keep higher-scoring body/source metadata as base.
				if r.Score > it.r.Score {
					keepBody := it.r.Body
					it.r = r
					if it.r.Body == "" {
						it.r.Body = keepBody
					}
				}
			} else {
				cp := r
				m[k] = &item{r: cp, score: contrib, topRank: rank}
			}
		}
	}
	out := make([]SearchResult, 0, len(m))
	for _, it := range m {
		if it.topRank == 0 {
			it.score += 0.05
		} else if it.topRank <= 2 {
			it.score += 0.02
		}
		it.r.Score = it.score
		out = append(out, it.r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	return out
}

func ParseStructuredQuery(input string) ([]StructuredSubSearch, error) {
	rawLines := nonEmptyStructuredQueryLines(input)
	if len(rawLines) == 0 {
		return nil, nil
	}
	var out []StructuredSubSearch
	for _, line := range rawLines {
		if structuredExpandPrefix.MatchString(line.Trimmed) {
			if len(rawLines) > 1 {
				return nil, fmt.Errorf("line %d starts with expand:, but query documents cannot mix expand with typed lines. submit a single expand query instead", line.Number)
			}
			text := strings.TrimSpace(structuredExpandPrefix.ReplaceAllString(line.Trimmed, ""))
			if text == "" {
				return nil, errors.New("expand: query must include text")
			}
			return nil, nil
		}
		if m := structuredQueryPrefixRe.FindStringSubmatch(line.Trimmed); len(m) == 2 {
			typ := strings.ToLower(m[1])
			text := strings.TrimSpace(line.Trimmed[len(m[0]):])
			if text == "" {
				return nil, fmt.Errorf("line %d (%s:) must include text", line.Number, typ)
			}
			if strings.ContainsAny(text, "\r\n") {
				return nil, fmt.Errorf("line %d (%s:) contains a newline; keep each query on a single line", line.Number, typ)
			}
			out = append(out, StructuredSubSearch{Type: typ, Query: text, Line: line.Number})
			continue
		}
		if len(rawLines) == 1 {
			return nil, nil
		}
		return nil, fmt.Errorf("line %d is missing a lex:/vec:/hyde: prefix; each line in a query document must start with one", line.Number)
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

type structuredQueryLine struct {
	Trimmed string
	Number  int
}

func nonEmptyStructuredQueryLines(input string) []structuredQueryLine {
	lines := strings.Split(input, "\n")
	out := make([]structuredQueryLine, 0, len(lines))
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		out = append(out, structuredQueryLine{Trimmed: trimmed, Number: i + 1})
	}
	return out
}

func normalizeStandaloneExpandQuery(input string) (string, bool, error) {
	rawLines := nonEmptyStructuredQueryLines(input)
	if len(rawLines) != 1 {
		return input, false, nil
	}
	line := rawLines[0].Trimmed
	if !structuredExpandPrefix.MatchString(line) {
		return input, false, nil
	}
	text := strings.TrimSpace(structuredExpandPrefix.ReplaceAllString(line, ""))
	if text == "" {
		return "", false, errors.New("expand: query must include text")
	}
	return text, true, nil
}

type GetOptions struct {
	FromLine    int
	MaxLines    int
	Full        bool
	LineNumbers bool
}

func (a *App) Get(ref string, opts GetOptions) (*Document, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, errors.New("file reference is required")
	}
	fromLineFromRef := 0
	if m := regexp.MustCompile(`:(\d+)$`).FindStringSubmatch(ref); len(m) == 2 {
		if n, _ := strconv.Atoi(m[1]); n > 0 {
			fromLineFromRef = n
			ref = strings.TrimSuffix(ref, ":"+m[1])
		}
	}
	if opts.FromLine == 0 && fromLineFromRef > 0 {
		opts.FromLine = fromLineFromRef
	}
	row, err := a.findDocumentRow(ref, true)
	if err != nil {
		return nil, err
	}
	doc := row.toDocument(a)
	if !opts.Full {
		// By default qmd get returns full file, but CLI handler can still request full.
	}
	if opts.FromLine > 0 || opts.MaxLines > 0 {
		doc.Body = sliceLines(doc.Body, opts.FromLine, opts.MaxLines)
	}
	if opts.LineNumbers {
		start := 1
		if opts.FromLine > 0 {
			start = opts.FromLine
		}
		doc.Body = addLineNumbers(doc.Body, start)
	}
	return doc, nil
}

type dbDocRow struct {
	Collection string
	RelPath    string
	Title      string
	Hash       string
	ModifiedAt string
	Body       string
}

func (r dbDocRow) toDocument(a *App) *Document {
	ctxStr, _ := a.contextForPath(r.Collection, r.RelPath)
	return &Document{
		Filepath:    buildVirtualPath(r.Collection, r.RelPath),
		DisplayPath: r.Collection + "/" + r.RelPath,
		Title:       r.Title,
		Hash:        r.Hash,
		DocID:       docID(r.Hash),
		Collection:  r.Collection,
		ModifiedAt:  r.ModifiedAt,
		BodyLength:  len(r.Body),
		Body:        r.Body,
		Context:     ctxStr,
	}
}

func (a *App) findDocumentRow(ref string, includeBody bool) (*dbDocRow, error) {
	colBody := "content.doc"
	if !includeBody {
		colBody = "''"
	}
	baseQuery := `
		SELECT d.collection, d.path, d.title, d.hash, d.modified_at, ` + colBody + `
		FROM documents d
		JOIN content ON content.hash=d.hash
		WHERE d.active=1
	`
	// docid lookup
	if isDocIDRef(ref) {
		prefix := strings.TrimPrefix(ref, "#")
		row := a.DB.QueryRow(baseQuery+` AND substr(d.hash,1,6)=? LIMIT 1`, prefix)
		var r dbDocRow
		if err := row.Scan(&r.Collection, &r.RelPath, &r.Title, &r.Hash, &r.ModifiedAt, &r.Body); err == nil {
			return &r, nil
		}
	}
	// virtual path
	if strings.HasPrefix(ref, "qmd://") {
		if col, rel, ok := parseVirtualPath(ref); ok {
			row := a.DB.QueryRow(baseQuery+` AND d.collection=? AND d.path=? LIMIT 1`, col, rel)
			var r dbDocRow
			if err := row.Scan(&r.Collection, &r.RelPath, &r.Title, &r.Hash, &r.ModifiedAt, &r.Body); err == nil {
				return &r, nil
			}
		}
	}
	// collection/path explicit
	s := filepath.ToSlash(ref)
	if !strings.HasPrefix(s, "/") {
		parts := strings.SplitN(s, "/", 2)
		if len(parts) == 2 {
			row := a.DB.QueryRow(baseQuery+` AND d.collection=? AND d.path=? LIMIT 1`, parts[0], parts[1])
			var r dbDocRow
			if err := row.Scan(&r.Collection, &r.RelPath, &r.Title, &r.Hash, &r.ModifiedAt, &r.Body); err == nil {
				return &r, nil
			}
		}
	}
	// absolute path lookup via collections
	cfg, _ := a.LoadConfig()
	if filepath.IsAbs(expandHome(ref)) {
		abs := filepath.ToSlash(filepath.Clean(expandHome(ref)))
		for name, c := range cfg.Collections {
			cp := filepath.ToSlash(filepath.Clean(expandHome(c.Path)))
			if abs == cp || strings.HasPrefix(abs, cp+"/") {
				rel := strings.TrimPrefix(strings.TrimPrefix(abs, cp), "/")
				row := a.DB.QueryRow(baseQuery+` AND d.collection=? AND d.path=? LIMIT 1`, name, rel)
				var r dbDocRow
				if err := row.Scan(&r.Collection, &r.RelPath, &r.Title, &r.Hash, &r.ModifiedAt, &r.Body); err == nil {
					return &r, nil
				}
			}
		}
	}
	// suffix match fallback
	row := a.DB.QueryRow(baseQuery+` AND (d.collection||'/'||d.path) LIKE ? LIMIT 1`, "%"+strings.TrimPrefix(filepath.ToSlash(ref), "/"))
	var r dbDocRow
	if err := row.Scan(&r.Collection, &r.RelPath, &r.Title, &r.Hash, &r.ModifiedAt, &r.Body); err == nil {
		return &r, nil
	}
	// fuzzy suggestions
	similar, _ := a.FindSimilar(ref, 5)
	if len(similar) > 0 {
		return nil, fmt.Errorf("document not found: %s (did you mean: %s?)", ref, strings.Join(similar, ", "))
	}
	return nil, fmt.Errorf("document not found: %s", ref)
}

func isDocIDRef(s string) bool {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) != 6 {
		return false
	}
	for _, r := range s {
		if !strings.ContainsRune("0123456789abcdefABCDEF", r) {
			return false
		}
	}
	return true
}

func (a *App) FindSimilar(ref string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 5
	}
	rows, err := a.DB.Query(`SELECT collection||'/'||path FROM documents WHERE active=1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type cand struct {
		s string
		d int
	}
	var all []cand
	target := strings.ToLower(filepath.ToSlash(strings.TrimSpace(strings.TrimPrefix(ref, "qmd://"))))
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			continue
		}
		all = append(all, cand{s: s, d: levenshtein(strings.ToLower(s), target)})
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].d == all[j].d {
			return all[i].s < all[j].s
		}
		return all[i].d < all[j].d
	})
	var out []string
	for _, c := range all {
		out = append(out, c.s)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func levenshtein(a, b string) int {
	ra := []rune(a)
	rb := []rune(b)
	if len(ra) == 0 {
		return len(rb)
	}
	if len(rb) == 0 {
		return len(ra)
	}
	prev := make([]int, len(rb)+1)
	cur := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 0
			if ra[i-1] != rb[j-1] {
				cost = 1
			}
			del := prev[j] + 1
			ins := cur[j-1] + 1
			sub := prev[j-1] + cost
			cur[j] = minInt(del, ins, sub)
		}
		prev, cur = cur, prev
	}
	return prev[len(rb)]
}

func minInt(vals ...int) int {
	m := vals[0]
	for _, v := range vals[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

func (a *App) MultiGet(pattern string, maxLines, maxBytes int, includeBody bool) ([]MultiGetResult, []string, error) {
	if maxBytes <= 0 {
		maxBytes = DefaultMultiGetMaxBytes
	}
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return nil, nil, errors.New("pattern is required")
	}
	var refs []string
	var errorsOut []string
	looksList := strings.Contains(pattern, ",") && !strings.ContainsAny(pattern, "*?")
	if looksList {
		for _, part := range strings.Split(pattern, ",") {
			ref := strings.TrimSpace(part)
			if ref == "" {
				continue
			}
			refs = append(refs, ref)
		}
	} else {
		matched, err := a.matchFilesByGlob(pattern)
		if err != nil {
			return nil, nil, err
		}
		if len(matched) == 0 {
			return nil, []string{fmt.Sprintf("No files matched pattern: %s", pattern)}, nil
		}
		refs = matched
	}
	var out []MultiGetResult
	for _, ref := range refs {
		doc, err := a.Get(ref, GetOptions{Full: true})
		if err != nil {
			errorsOut = append(errorsOut, err.Error())
			continue
		}
		if doc.BodyLength > maxBytes {
			out = append(out, MultiGetResult{
				Doc:        *doc,
				Skipped:    true,
				SkipReason: fmt.Sprintf("File too large (%d bytes > %d bytes)", doc.BodyLength, maxBytes),
			})
			continue
		}
		if maxLines > 0 {
			doc.Body = sliceLines(doc.Body, 1, maxLines)
		}
		if !includeBody {
			doc.Body = ""
		}
		out = append(out, MultiGetResult{Doc: *doc})
	}
	return out, errorsOut, nil
}

func (a *App) matchFilesByGlob(glob string) ([]string, error) {
	re, err := globToRegexp(glob)
	if err != nil {
		return nil, err
	}
	rows, err := a.DB.Query(`SELECT collection||'/'||path FROM documents WHERE active=1 ORDER BY collection, path`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			continue
		}
		if re.MatchString(s) {
			out = append(out, s)
		}
	}
	return out, nil
}

func (a *App) List(pathFilter string, limit int) ([]ListEntry, error) {
	pathFilter = strings.TrimSpace(pathFilter)
	sqlStr := `SELECT d.collection, d.path, d.title, d.hash, d.modified_at, LENGTH(content.doc)
	           FROM documents d JOIN content ON content.hash=d.hash WHERE d.active=1`
	var args []any
	if pathFilter != "" {
		if strings.HasPrefix(pathFilter, "qmd://") {
			if col, rel, ok := parseVirtualPath(pathFilter); ok {
				sqlStr += ` AND d.collection=? AND d.path LIKE ?`
				args = append(args, col, rel+"%")
			}
		} else if strings.Contains(pathFilter, "/") {
			parts := strings.SplitN(filepath.ToSlash(pathFilter), "/", 2)
			if len(parts) == 2 {
				sqlStr += ` AND d.collection=? AND d.path LIKE ?`
				args = append(args, parts[0], parts[1]+"%")
			}
		}
	}
	sqlStr += ` ORDER BY d.collection, d.path`
	if limit > 0 {
		sqlStr += ` LIMIT ?`
		args = append(args, limit)
	}
	rows, err := a.DB.Query(sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ListEntry
	for rows.Next() {
		var e ListEntry
		var hash string
		if err := rows.Scan(&e.Collection, &e.Filepath, &e.Title, &hash, &e.ModifiedAt, &e.BodyLength); err != nil {
			return nil, err
		}
		e.DisplayPath = e.Collection + "/" + e.Filepath
		e.Filepath = buildVirtualPath(e.Collection, e.Filepath)
		e.DocID = docID(hash)
		out = append(out, e)
	}
	return out, nil
}

func (a *App) Cleanup(ctx context.Context) (*CleanupStats, error) {
	stats := &CleanupStats{}
	if res, err := a.DB.ExecContext(ctx, `DELETE FROM llm_cache`); err == nil {
		stats.LLMCacheDeleted, _ = res.RowsAffected()
	}
	if res, err := a.DB.ExecContext(ctx, `DELETE FROM content_vectors WHERE hash NOT IN (SELECT DISTINCT hash FROM documents WHERE active=1)`); err == nil {
		stats.OrphanedVectors, _ = res.RowsAffected()
	}
	if res, err := a.DB.ExecContext(ctx, `DELETE FROM documents WHERE active=0`); err == nil {
		stats.InactiveDeleted, _ = res.RowsAffected()
	}
	if res, err := a.DB.ExecContext(ctx, `DELETE FROM content WHERE hash NOT IN (SELECT DISTINCT hash FROM documents WHERE active=1)`); err == nil {
		stats.OrphanedContent, _ = res.RowsAffected()
	}
	_, _ = a.DB.ExecContext(ctx, `VACUUM`)
	return stats, nil
}

func sliceLines(s string, from, max int) string {
	lines := strings.Split(s, "\n")
	start := 0
	if from > 1 {
		start = from - 1
		if start > len(lines) {
			start = len(lines)
		}
	}
	end := len(lines)
	if max > 0 && start+max < end {
		end = start + max
	}
	return strings.Join(lines[start:end], "\n")
}

func addLineNumbers(s string, start int) string {
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = fmt.Sprintf("%d: %s", start+i, lines[i])
	}
	return strings.Join(lines, "\n")
}

type snippet struct {
	Line    int
	Snippet string
}

func extractSnippet(body, query string, maxLen int, chunkPos int) snippet {
	if maxLen <= 0 {
		maxLen = 500
	}
	searchBody := body
	lineOffset := 0
	if chunkPos > 0 && chunkPos < len(body) {
		start := max(0, chunkPos-100)
		end := min(len(body), chunkPos+DefaultChunkSizeChars+100)
		searchBody = body[start:end]
		if start > 0 {
			lineOffset = strings.Count(body[:start], "\n")
		}
	}
	lines := strings.Split(searchBody, "\n")
	terms := tokenize(query)
	bestIdx, bestScore := 0, -1
	for i, line := range lines {
		lower := strings.ToLower(line)
		score := 0
		for _, t := range terms {
			if len(t) >= 2 && strings.Contains(lower, t) {
				score++
			}
		}
		if score > bestScore {
			bestScore = score
			bestIdx = i
		}
	}
	start := max(0, bestIdx-1)
	end := min(len(lines), bestIdx+3)
	txt := strings.Join(lines[start:end], "\n")
	if chunkPos > 0 && strings.TrimSpace(txt) == "" {
		return extractSnippet(body, query, maxLen, 0)
	}
	if len(txt) > maxLen {
		txt = txt[:maxLen-3] + "..."
	}
	absStartLine := lineOffset + start + 1
	header := fmt.Sprintf("@@ -%d,%d @@", absStartLine, end-start)
	return snippet{Line: lineOffset + bestIdx + 1, Snippet: header + "\n" + txt}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// FormatSearchResults renders search/query/vsearch results in qmd-like formats.
func FormatSearchResults(results []SearchResult, opts OutputOptions) (string, error) {
	if opts.Format == "" {
		opts.Format = OutputCLI
	}
	switch opts.Format {
	case OutputJSON:
		var arr []map[string]any
		for _, r := range results {
			body := r.Body
			if body == "" && r.ChunkText != "" {
				body = r.ChunkText
			}
			item := map[string]any{
				"docid": "#" + r.DocID,
				"score": round2(r.Score),
				"file":  r.DisplayPath,
				"title": r.Title,
			}
			if r.Context != "" {
				item["context"] = r.Context
			}
			if opts.Full {
				txt := body
				if opts.LineNumbers {
					txt = addLineNumbers(txt, 1)
				}
				item["body"] = txt
			} else {
				sn := extractSnippet(body, opts.Query, 300, r.ChunkPos)
				txt := sn.Snippet
				if opts.LineNumbers {
					txt = addLineNumbers(txt, sn.Line)
				}
				item["snippet"] = txt
			}
			arr = append(arr, item)
		}
		b, err := json.MarshalIndent(arr, "", "  ")
		return string(b), err
	case OutputCSV:
		var b strings.Builder
		w := csv.NewWriter(&b)
		_ = w.Write([]string{"docid", "score", "file", "title", "context", "line", "snippet"})
		for _, r := range results {
			body := r.Body
			if body == "" && r.ChunkText != "" {
				body = r.ChunkText
			}
			sn := extractSnippet(body, opts.Query, 500, r.ChunkPos)
			content := sn.Snippet
			if opts.Full {
				content = body
			}
			if opts.LineNumbers {
				content = addLineNumbers(content, sn.Line)
			}
			_ = w.Write([]string{
				"#" + r.DocID,
				fmt.Sprintf("%.4f", r.Score),
				r.DisplayPath,
				r.Title,
				r.Context,
				strconv.Itoa(sn.Line),
				content,
			})
		}
		w.Flush()
		return b.String(), w.Error()
	case OutputFiles:
		var lines []string
		for _, r := range results {
			if r.Context != "" {
				lines = append(lines, fmt.Sprintf("#%s,%.2f,%s,%q", r.DocID, r.Score, r.DisplayPath, r.Context))
			} else {
				lines = append(lines, fmt.Sprintf("#%s,%.2f,%s", r.DocID, r.Score, r.DisplayPath))
			}
		}
		return strings.Join(lines, "\n"), nil
	case OutputMD:
		var parts []string
		for _, r := range results {
			body := r.Body
			if body == "" && r.ChunkText != "" {
				body = r.ChunkText
			}
			content := body
			if !opts.Full {
				content = extractSnippet(body, opts.Query, 500, r.ChunkPos).Snippet
			}
			if opts.LineNumbers {
				content = addLineNumbers(content, 1)
			}
			var sb strings.Builder
			sb.WriteString("---\n# ")
			sb.WriteString(r.Title)
			sb.WriteString("\n\n**docid:** `#")
			sb.WriteString(r.DocID)
			sb.WriteString("`\n")
			if r.Context != "" {
				sb.WriteString("**context:** ")
				sb.WriteString(r.Context)
				sb.WriteString("\n")
			}
			sb.WriteString("\n")
			sb.WriteString(content)
			sb.WriteString("\n")
			parts = append(parts, sb.String())
		}
		return strings.Join(parts, "\n"), nil
	case OutputXML:
		type item struct {
			XMLName xml.Name `xml:"file"`
			DocID   string   `xml:"docid,attr"`
			Name    string   `xml:"name,attr"`
			Title   string   `xml:"title,attr,omitempty"`
			Context string   `xml:"context,attr,omitempty"`
			Content string   `xml:",chardata"`
		}
		var items []item
		for _, r := range results {
			body := r.Body
			if body == "" && r.ChunkText != "" {
				body = r.ChunkText
			}
			content := body
			if !opts.Full {
				content = extractSnippet(body, opts.Query, 500, r.ChunkPos).Snippet
			}
			if opts.LineNumbers {
				content = addLineNumbers(content, 1)
			}
			items = append(items, item{
				DocID: "#" + r.DocID, Name: r.DisplayPath, Title: r.Title, Context: r.Context, Content: content,
			})
		}
		var buf strings.Builder
		for i, it := range items {
			b, err := xml.MarshalIndent(it, "", "  ")
			if err != nil {
				return "", err
			}
			if i > 0 {
				buf.WriteString("\n\n")
			}
			buf.Write(b)
		}
		return buf.String(), nil
	default:
		var lines []string
		if len(results) == 0 {
			return "No results.", nil
		}
		for _, r := range results {
			body := r.Body
			if body == "" && r.ChunkText != "" {
				body = r.ChunkText
			}
			sn := extractSnippet(body, opts.Query, 240, r.ChunkPos).Snippet
			lines = append(lines, fmt.Sprintf("[%0.2f] #%s %s\n%s\n", r.Score, r.DocID, r.DisplayPath, sn))
		}
		return strings.TrimSpace(strings.Join(lines, "\n")), nil
	}
}

func round2(f float64) float64 {
	return math.Round(f*100) / 100
}

func FormatMultiGet(results []MultiGetResult, format OutputFormat, lineNumbers bool) (string, error) {
	if format == "" {
		format = OutputCLI
	}
	switch format {
	case OutputJSON:
		var arr []map[string]any
		for _, r := range results {
			item := map[string]any{
				"file":  r.Doc.DisplayPath,
				"title": r.Doc.Title,
			}
			if r.Doc.Context != "" {
				item["context"] = r.Doc.Context
			}
			if r.Skipped {
				item["skipped"] = true
				item["reason"] = r.SkipReason
			} else {
				body := r.Doc.Body
				if lineNumbers {
					body = addLineNumbers(body, 1)
				}
				item["body"] = body
			}
			arr = append(arr, item)
		}
		b, err := json.MarshalIndent(arr, "", "  ")
		return string(b), err
	case OutputCSV:
		var b strings.Builder
		w := csv.NewWriter(&b)
		_ = w.Write([]string{"file", "title", "context", "skipped", "body"})
		for _, r := range results {
			body := r.Doc.Body
			if lineNumbers {
				body = addLineNumbers(body, 1)
			}
			if r.Skipped {
				body = r.SkipReason
			}
			_ = w.Write([]string{r.Doc.DisplayPath, r.Doc.Title, r.Doc.Context, strconv.FormatBool(r.Skipped), body})
		}
		w.Flush()
		return b.String(), w.Error()
	case OutputFiles:
		var lines []string
		for _, r := range results {
			s := r.Doc.DisplayPath
			if r.Doc.Context != "" {
				s += fmt.Sprintf(",%q", r.Doc.Context)
			}
			if r.Skipped {
				s += ",[SKIPPED]"
			}
			lines = append(lines, s)
		}
		return strings.Join(lines, "\n"), nil
	case OutputMD:
		var parts []string
		for _, r := range results {
			var sb strings.Builder
			sb.WriteString("## " + r.Doc.DisplayPath + "\n\n")
			if r.Doc.Context != "" {
				sb.WriteString("**Context:** " + r.Doc.Context + "\n\n")
			}
			if r.Skipped {
				sb.WriteString("> " + r.SkipReason + "\n")
			} else {
				body := r.Doc.Body
				if lineNumbers {
					body = addLineNumbers(body, 1)
				}
				sb.WriteString("```\n" + body + "\n```\n")
			}
			parts = append(parts, sb.String())
		}
		return strings.Join(parts, "\n"), nil
	case OutputXML:
		type item struct {
			XMLName xml.Name `xml:"document"`
			File    string   `xml:"file"`
			Title   string   `xml:"title"`
			Context string   `xml:"context,omitempty"`
			Skipped bool     `xml:"skipped"`
			Reason  string   `xml:"reason,omitempty"`
			Body    string   `xml:"body,omitempty"`
		}
		var items []item
		for _, r := range results {
			body := r.Doc.Body
			if lineNumbers {
				body = addLineNumbers(body, 1)
			}
			items = append(items, item{
				File: r.Doc.DisplayPath, Title: r.Doc.Title, Context: r.Doc.Context,
				Skipped: r.Skipped, Reason: r.SkipReason, Body: body,
			})
		}
		type root struct {
			XMLName xml.Name `xml:"documents"`
			Items   []item   `xml:"document"`
		}
		b, err := xml.MarshalIndent(root{Items: items}, "", "  ")
		if err != nil {
			return "", err
		}
		return xml.Header + string(b), nil
	default:
		var lines []string
		for _, r := range results {
			lines = append(lines, r.Doc.DisplayPath)
			if r.Skipped {
				lines = append(lines, "  [SKIPPED] "+r.SkipReason)
				continue
			}
			body := r.Doc.Body
			if lineNumbers {
				body = addLineNumbers(body, 1)
			}
			lines = append(lines, body)
			lines = append(lines, "")
		}
		return strings.TrimRight(strings.Join(lines, "\n"), "\n"), nil
	}
}
