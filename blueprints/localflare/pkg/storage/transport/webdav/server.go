// File: lib/storage/transport/webdav/server.go

// Package webdav provides a WebDAV transport layer for storage.Storage backends.
//
// This package implements a WebDAV server (RFC 4918) that exposes any storage.Storage
// implementation over the WebDAV protocol, allowing standard WebDAV clients to interact
// with the storage backend.
//
// Path Mapping:
//
//	/                    → list buckets
//	/<bucket>/           → bucket root
//	/<bucket>/<key>      → object
//
// Example:
//
//	store, _ := storage.Open(ctx, "local:///data")
//
//	cfg := &webdav.Config{
//	    Prefix: "/webdav",
//	    Auth: webdav.AuthConfig{
//	        Type:  "basic",
//	        Realm: "WebDAV",
//	        BasicAuth: func(u, p string) bool { return u == "admin" && p == "secret" },
//	    },
//	}
//
//	server := webdav.New(store, cfg)
//	http.Handle("/webdav/", server)
//	http.ListenAndServe(":8080", nil)
package webdav

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-mizu/blueprints/localflare/pkg/storage"
	"github.com/go-mizu/mizu"
	"golang.org/x/net/webdav"
)

// Config controls WebDAV server behavior.
type Config struct {
	// Prefix is the URL path prefix (e.g., "/webdav").
	// Stripped from incoming paths before mapping to storage.
	Prefix string

	// Bucket restricts access to a single bucket.
	// If empty, all buckets are accessible.
	Bucket string

	// ReadOnly disables all write operations.
	ReadOnly bool

	// Auth configures authentication.
	Auth AuthConfig

	// LockSystem manages WebDAV locks.
	// If nil, an in-memory lock system is used.
	LockSystem webdav.LockSystem

	// Logger for request/error logging.
	Logger *slog.Logger

	// HideDotFiles hides files starting with '.' in listings.
	HideDotFiles bool

	// DefaultContentType for files without extension.
	DefaultContentType string

	// MaxUploadSize limits upload size in bytes (0 = unlimited).
	MaxUploadSize int64

	// WriteBufferSize is the max buffer size for uploads before spilling to disk.
	// Default is 32MB.
	WriteBufferSize int64

	// TempDir for buffering large uploads. Default is os.TempDir().
	TempDir string
}

func (c *Config) clone() *Config {
	if c == nil {
		return &Config{
			WriteBufferSize:    32 << 20, // 32MB
			DefaultContentType: "application/octet-stream",
		}
	}

	cp := *c
	if cp.WriteBufferSize == 0 {
		cp.WriteBufferSize = 32 << 20
	}
	if cp.DefaultContentType == "" {
		cp.DefaultContentType = "application/octet-stream"
	}
	if cp.Logger == nil {
		cp.Logger = slog.Default()
	}
	return &cp
}

// AuthConfig configures authentication.
type AuthConfig struct {
	// Type is the authentication method.
	// Supported: "none", "basic", "jwt"
	Type string

	// Realm for Basic authentication.
	Realm string

	// BasicAuth validates username/password for Basic auth.
	BasicAuth func(username, password string) bool

	// JWTSecret for JWT token validation.
	JWTSecret string

	// JWTClaims extracts user from validated JWT claims.
	JWTClaims func(claims map[string]any) (*User, error)
}

// User represents an authenticated user.
type User struct {
	ID       string
	Username string
	Roles    []string
	Metadata map[string]string
}

// Server handles WebDAV requests backed by storage.Storage.
type Server struct {
	store   storage.Storage
	cfg     *Config
	handler *webdav.Handler
}

// New creates a WebDAV server backed by storage.Storage.
func New(store storage.Storage, cfg *Config) *Server {
	if store == nil {
		panic("webdav: storage is nil")
	}

	cfg = cfg.clone()

	fs := &StorageFileSystem{
		store:              store,
		bucket:             cfg.Bucket,
		readOnly:           cfg.ReadOnly,
		hideDotFiles:       cfg.HideDotFiles,
		defaultContentType: cfg.DefaultContentType,
		maxUploadSize:      cfg.MaxUploadSize,
		writeBufferSize:    cfg.WriteBufferSize,
		tempDir:            cfg.TempDir,
		logger:             cfg.Logger,
	}

	lockSystem := cfg.LockSystem
	if lockSystem == nil {
		lockSystem = webdav.NewMemLS()
	}

	handler := &webdav.Handler{
		Prefix:     cfg.Prefix,
		FileSystem: fs,
		LockSystem: lockSystem,
		Logger: func(r *http.Request, err error) {
			if err != nil {
				cfg.Logger.Error("webdav request error",
					"method", r.Method,
					"path", r.URL.Path,
					"error", err,
				)
			} else {
				cfg.Logger.Debug("webdav request",
					"method", r.Method,
					"path", r.URL.Path,
				)
			}
		},
	}

	return &Server{
		store:   store,
		cfg:     cfg,
		handler: handler,
	}
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Handle authentication
	if s.cfg.Auth.Type != "" && s.cfg.Auth.Type != "none" {
		if !s.authenticate(w, r) {
			return
		}
	}

	// Handle read-only mode
	if s.cfg.ReadOnly && isWriteMethod(r.Method) {
		http.Error(w, "read-only mode", http.StatusForbidden)
		return
	}

	s.handler.ServeHTTP(w, r)
}

// Handler returns the underlying webdav.Handler for advanced configuration.
func (s *Server) Handler() *webdav.Handler {
	return s.handler
}

// ListenAndServe starts a standalone WebDAV server.
func (s *Server) ListenAndServe(addr string) error {
	s.cfg.Logger.Info("webdav server started", "addr", addr, "prefix", s.cfg.Prefix)
	return http.ListenAndServe(addr, s)
}

// authenticate checks credentials based on configured auth method.
func (s *Server) authenticate(w http.ResponseWriter, r *http.Request) bool {
	switch s.cfg.Auth.Type {
	case "basic":
		return s.authenticateBasic(w, r)
	case "jwt":
		return s.authenticateJWT(w, r)
	default:
		return true
	}
}

// authenticateBasic performs HTTP Basic authentication.
func (s *Server) authenticateBasic(w http.ResponseWriter, r *http.Request) bool {
	if s.cfg.Auth.BasicAuth == nil {
		return true
	}

	username, password, ok := r.BasicAuth()
	if !ok || !s.cfg.Auth.BasicAuth(username, password) {
		realm := s.cfg.Auth.Realm
		if realm == "" {
			realm = "WebDAV"
		}
		w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return false
	}

	return true
}

// authenticateJWT performs JWT token authentication.
func (s *Server) authenticateJWT(w http.ResponseWriter, r *http.Request) bool {
	if s.cfg.Auth.JWTSecret == "" {
		return true
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
		return false
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(w, "Invalid Authorization header", http.StatusUnauthorized)
		return false
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	claims, err := validateJWT(token, s.cfg.Auth.JWTSecret)
	if err != nil {
		http.Error(w, "Invalid token: "+err.Error(), http.StatusUnauthorized)
		return false
	}

	if s.cfg.Auth.JWTClaims != nil {
		_, err := s.cfg.Auth.JWTClaims(claims)
		if err != nil {
			http.Error(w, "Invalid claims: "+err.Error(), http.StatusForbidden)
			return false
		}
	}

	return true
}

// isWriteMethod returns true for methods that modify data.
func isWriteMethod(method string) bool {
	switch method {
	case "PUT", "POST", "DELETE", "MKCOL", "MOVE", "COPY", "PROPPATCH", "LOCK", "UNLOCK":
		return true
	default:
		return false
	}
}

// Register mounts the WebDAV server under basePath using mizu.
//
// Example:
//
//	store, _ := storage.Open(ctx, "local:///data")
//	cfg := &webdav.Config{
//	    Auth: webdav.AuthConfig{
//	        Type: "basic",
//	        Realm: "WebDAV",
//	        BasicAuth: func(u, p string) bool { return u == "admin" && p == "secret" },
//	    },
//	}
//	webdav.Register(app, "/dav", store, cfg)
func Register(app *mizu.App, basePath string, store storage.Storage, cfg *Config) *Server {
	if cfg == nil {
		cfg = &Config{}
	}

	// Normalize basePath
	basePath = strings.TrimSpace(basePath)
	if basePath == "" {
		basePath = "/"
	}
	if !strings.HasPrefix(basePath, "/") {
		basePath = "/" + basePath
	}
	basePath = strings.TrimSuffix(basePath, "/")

	// Set prefix to match basePath
	cfgCopy := *cfg
	cfgCopy.Prefix = basePath

	s := New(store, &cfgCopy)

	// Register handler for all WebDAV methods
	handler := func(c *mizu.Ctx) error {
		s.ServeHTTP(c.Writer(), c.Request())
		return nil
	}

	// Register routes for WebDAV methods
	if basePath == "" || basePath == "/" {
		// Root mount
		registerWebDAVMethods(app, "/", handler)
		registerWebDAVMethods(app, "/{path...}", handler)
	} else {
		registerWebDAVMethods(app, basePath, handler)
		registerWebDAVMethods(app, basePath+"/", handler)
		registerWebDAVMethods(app, basePath+"/{path...}", handler)
	}

	return s
}

// registerWebDAVMethods registers all WebDAV methods for a path.
func registerWebDAVMethods(app *mizu.App, path string, handler mizu.Handler) {
	app.Get(path, handler)
	app.Put(path, handler)
	app.Post(path, handler)
	app.Delete(path, handler)
	app.Handle("PROPFIND", path, handler)
	app.Handle("PROPPATCH", path, handler)
	app.Handle("MKCOL", path, handler)
	app.Handle("COPY", path, handler)
	app.Handle("MOVE", path, handler)
	app.Handle("LOCK", path, handler)
	app.Handle("UNLOCK", path, handler)
	app.Handle("OPTIONS", path, handler)
}
