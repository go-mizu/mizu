// Package server provides a standalone S3-compatible storage server.
//
// This package wraps the S3 transport layer and storage drivers into a
// self-contained server that can be used for local development and testing.
//
// Example usage:
//
//	cfg := &server.Config{
//		Port:            9000,
//		DSN:             "local:///var/data/liteio",
//		AccessKeyID:     "liteio",
//		SecretAccessKey: "liteio123",
//	}
//	srv, err := server.New(cfg)
//	if err != nil {
//		log.Fatal(err)
//	}
//	if err := srv.Start(); err != nil {
//		log.Fatal(err)
//	}
package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"sync"
	"time"

	"github.com/go-mizu/mizu/blueprints/localbase/pkg/storage"
	"github.com/go-mizu/mizu/blueprints/localbase/pkg/storage/transport/s3"
	"github.com/go-mizu/mizu"

	// Register storage drivers
	_ "github.com/go-mizu/mizu/blueprints/localbase/pkg/storage/driver/local"
	_ "github.com/go-mizu/mizu/blueprints/localbase/pkg/storage/driver/memory"
)

// Config configures the S3-compatible server.
type Config struct {
	// Host to bind to. Default "0.0.0.0".
	Host string

	// Port to listen on. Default 9000.
	Port int

	// DSN for storage backend.
	// Examples:
	//   "local:///var/data/liteio"
	//   "memory://"
	// Default: "local://$HOME/data/liteio"
	DSN string

	// AccessKeyID for S3 authentication. Default "liteio".
	// If empty, authentication is disabled.
	AccessKeyID string

	// SecretAccessKey for S3 authentication. Default "liteio123".
	SecretAccessKey string

	// Region for S3 responses. Default "us-east-1".
	Region string

	// MaxObjectSize limits upload size. Default 5GB.
	MaxObjectSize int64

	// ReadTimeout for HTTP reads. Default 60s.
	ReadTimeout time.Duration

	// WriteTimeout for HTTP writes. Default 60s.
	WriteTimeout time.Duration

	// Logger for server logs. If nil, uses slog.Default().
	Logger *slog.Logger

	// EnablePprof enables pprof profiling endpoints. Default true.
	// When enabled, the following endpoints are available:
	//   /debug/pprof/
	//   /debug/pprof/cmdline
	//   /debug/pprof/profile
	//   /debug/pprof/symbol
	//   /debug/pprof/trace
	//   /debug/pprof/heap
	//   /debug/pprof/goroutine
	//   /debug/pprof/block
	//   /debug/pprof/mutex
	EnablePprof bool
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	dataDir := homeDir + "/data/liteio"

	return &Config{
		Host:            "0.0.0.0",
		Port:            9000,
		DSN:             "local://" + dataDir,
		AccessKeyID:     "liteio",
		SecretAccessKey: "liteio123",
		Region:          "us-east-1",
		MaxObjectSize:   5 * 1024 * 1024 * 1024, // 5GB
		ReadTimeout:     60 * time.Second,
		WriteTimeout:    60 * time.Second,
		EnablePprof:     true, // Enable pprof by default
	}
}

// applyDefaults fills in default values for unset fields.
func (c *Config) applyDefaults() {
	def := DefaultConfig()
	if c.Host == "" {
		c.Host = def.Host
	}
	if c.Port == 0 {
		c.Port = def.Port
	}
	if c.DSN == "" {
		c.DSN = def.DSN
	}
	if c.AccessKeyID == "" {
		c.AccessKeyID = def.AccessKeyID
	}
	if c.SecretAccessKey == "" {
		c.SecretAccessKey = def.SecretAccessKey
	}
	if c.Region == "" {
		c.Region = def.Region
	}
	if c.MaxObjectSize == 0 {
		c.MaxObjectSize = def.MaxObjectSize
	}
	if c.ReadTimeout == 0 {
		c.ReadTimeout = def.ReadTimeout
	}
	if c.WriteTimeout == 0 {
		c.WriteTimeout = def.WriteTimeout
	}
	if c.Logger == nil {
		c.Logger = slog.Default()
	}
}

// Server is an S3-compatible storage server.
type Server struct {
	config  *Config
	storage storage.Storage
	app     *mizu.App
	server  *http.Server

	mu       sync.Mutex
	running  bool
	addr     string
	listener net.Listener
}

// staticCredentialProvider implements s3.CredentialProvider.
type staticCredentialProvider struct {
	accessKey string
	secretKey string
}

func (p *staticCredentialProvider) Lookup(accessKeyID string) (*s3.Credential, error) {
	if accessKeyID != p.accessKey {
		return nil, errors.New("unknown access key")
	}
	return &s3.Credential{
		AccessKeyID:     p.accessKey,
		SecretAccessKey: p.secretKey,
	}, nil
}

// New creates a new S3-compatible server.
func New(cfg *Config) (*Server, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	} else {
		cfg.applyDefaults()
	}

	// Ensure data directory exists for local driver
	if err := ensureDataDir(cfg.DSN); err != nil {
		return nil, fmt.Errorf("ensure data dir: %w", err)
	}

	// Open storage backend
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stor, err := storage.Open(ctx, cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("open storage: %w", err)
	}

	// Create mizu app
	app := mizu.New()
	app.SetLogger(cfg.Logger)

	// Configure S3 transport
	s3Config := &s3.Config{
		Region:        cfg.Region,
		MaxObjectSize: cfg.MaxObjectSize,
	}

	// Set up authentication if credentials are provided
	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		s3Config.Credentials = &staticCredentialProvider{
			accessKey: cfg.AccessKeyID,
			secretKey: cfg.SecretAccessKey,
		}
		s3Config.Signer = &s3.SignerV4{}
	}

	// Register S3 API routes at root
	s3.Register(app, "/", stor, s3Config)

	return &Server{
		config:  cfg,
		storage: stor,
		app:     app,
	}, nil
}

// ensureDataDir ensures the data directory exists for local storage driver.
func ensureDataDir(dsn string) error {
	// Only handle local:// and file:// DSNs
	var path string
	switch {
	case len(dsn) > 8 && dsn[:8] == "local://":
		path = dsn[8:]
	case len(dsn) > 7 && dsn[:7] == "file://":
		path = dsn[7:]
	default:
		// Not a local path DSN, skip
		return nil
	}

	if path == "" {
		return nil
	}

	return os.MkdirAll(path, 0755)
}

// Start starts the server and blocks until it's stopped.
func (s *Server) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return errors.New("server already running")
	}

	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		s.mu.Unlock()
		return fmt.Errorf("listen: %w", err)
	}

	s.listener = listener
	s.addr = listener.Addr().String()
	s.running = true

	s.server = &http.Server{
		Handler:           s.handler(),
		ReadTimeout:       s.config.ReadTimeout,
		WriteTimeout:      s.config.WriteTimeout,
		IdleTimeout:       120 * time.Second, // Keep connections open for reuse
		MaxHeaderBytes:    1 << 16,           // 64KB max header size
		ReadHeaderTimeout: 10 * time.Second,  // Fast header parsing timeout
	}

	s.mu.Unlock()

	s.config.Logger.Info("liteio server started",
		"addr", s.addr,
		"region", s.config.Region,
	)

	err = s.server.Serve(listener)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve: %w", err)
	}

	return nil
}

// StartBackground starts the server in a goroutine and returns immediately.
// Use Stop() or Shutdown() to stop the server.
func (s *Server) StartBackground() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return errors.New("server already running")
	}

	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		s.mu.Unlock()
		return fmt.Errorf("listen: %w", err)
	}

	s.listener = listener
	s.addr = listener.Addr().String()
	s.running = true

	s.server = &http.Server{
		Handler:           s.handler(),
		ReadTimeout:       s.config.ReadTimeout,
		WriteTimeout:      s.config.WriteTimeout,
		IdleTimeout:       120 * time.Second, // Keep connections open for reuse
		MaxHeaderBytes:    1 << 16,           // 64KB max header size
		ReadHeaderTimeout: 10 * time.Second,  // Fast header parsing timeout
	}

	s.mu.Unlock()

	s.config.Logger.Info("liteio server started",
		"addr", s.addr,
		"region", s.config.Region,
	)

	go func() {
		if err := s.server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.config.Logger.Error("server error", "error", err)
		}
	}()

	return nil
}

func (s *Server) handler() http.Handler {
	// Build pprof mux if enabled
	var pprofMux *http.ServeMux
	if s.config.EnablePprof {
		pprofMux = http.NewServeMux()
		pprofMux.HandleFunc("/debug/pprof/", pprof.Index)
		pprofMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		pprofMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		pprofMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		pprofMux.HandleFunc("/debug/pprof/trace", pprof.Trace)
		pprofMux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
		pprofMux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
		pprofMux.Handle("/debug/pprof/block", pprof.Handler("block"))
		pprofMux.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))
		pprofMux.Handle("/debug/pprof/allocs", pprof.Handler("allocs"))
		pprofMux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle pprof endpoints if enabled
		if pprofMux != nil && len(r.URL.Path) >= 12 && r.URL.Path[:12] == "/debug/pprof" {
			pprofMux.ServeHTTP(w, r)
			return
		}

		if r.URL.Path == "/healthz/ready" && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if r.Method == http.MethodGet {
				_, _ = w.Write([]byte(`{"status":"ok","server":"liteio"}`))
			}
			return
		}

		// Normalize path by stripping trailing slash (except root)
		// S3 clients like warp often add trailing slashes to bucket paths
		// Store original path in context for signature verification
		if len(r.URL.Path) > 1 && r.URL.Path[len(r.URL.Path)-1] == '/' {
			ctx := context.WithValue(r.Context(), s3.OriginalPathContextKey{}, r.URL.Path)
			r = r.WithContext(ctx)
			r.URL.Path = r.URL.Path[:len(r.URL.Path)-1]
		}

		s.app.ServeHTTP(w, r)
	})
}

// Stop stops the server immediately.
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.running = false

	if s.server != nil {
		if err := s.server.Close(); err != nil {
			return fmt.Errorf("close server: %w", err)
		}
	}

	if s.storage != nil {
		if err := s.storage.Close(); err != nil {
			return fmt.Errorf("close storage: %w", err)
		}
	}

	s.config.Logger.Info("liteio server stopped")
	return nil
}

// Shutdown gracefully shuts down the server with the given timeout.
func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = false
	srv := s.server
	stor := s.storage
	s.mu.Unlock()

	var errs []error

	if srv != nil {
		if err := srv.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("shutdown server: %w", err))
		}
	}

	if stor != nil {
		if err := stor.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close storage: %w", err))
		}
	}

	s.config.Logger.Info("liteio server shutdown complete")

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// Addr returns the address the server is listening on.
// Returns empty string if not started.
func (s *Server) Addr() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.addr
}

// Running returns true if the server is running.
func (s *Server) Running() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// Storage returns the underlying storage backend.
func (s *Server) Storage() storage.Storage {
	return s.storage
}
