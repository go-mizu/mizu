package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/go-mizu/mizu/blueprints/lingo/app/web"
	"github.com/go-mizu/mizu/blueprints/lingo/store"
	"github.com/go-mizu/mizu/blueprints/lingo/store/postgres"
	"github.com/go-mizu/mizu/blueprints/lingo/store/sqlite"
	"github.com/spf13/cobra"
)

// defaultDBPath returns the default SQLite database path
func defaultDBPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "lingo.db"
	}
	return filepath.Join(home, "data", "blueprints", "lingo", "lingo.db")
}

// NewServe creates the serve command
func NewServe() *cobra.Command {
	var (
		port        int
		devMode     bool
		usePostgres bool
		dbPath      string
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the Lingo server",
		Long: `Start all Lingo services including:
  - HTTP API server
  - Learning path and exercises
  - Gamification system (XP, streaks, hearts)
  - Social features (friends, leaderboards)
  - Dashboard UI

The server runs on port 8080 by default.
Database defaults to SQLite at $HOME/data/blueprints/lingo/lingo.db.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServe(cmd.Context(), port, devMode, !usePostgres, dbPath)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 8080, "Port to listen on")
	cmd.Flags().BoolVar(&devMode, "dev", false, "Enable development mode")
	cmd.Flags().BoolVar(&usePostgres, "postgres", false, "Use PostgreSQL instead of SQLite")
	cmd.Flags().StringVar(&dbPath, "db", defaultDBPath(), "SQLite database path")

	return cmd
}

func runServe(ctx context.Context, port int, devMode, useSqlite bool, dbPath string) error {
	fmt.Println(Banner())

	// Connect to database
	var st store.Store
	var err error

	if useSqlite {
		// Ensure database directory exists
		dbDir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			return fmt.Errorf("failed to create database directory: %w", err)
		}
		fmt.Println(infoStyle.Render(fmt.Sprintf("Connecting to SQLite (%s)...", dbPath)))
		st, err = sqlite.New(ctx, dbPath)
	} else {
		fmt.Println(infoStyle.Render("Connecting to PostgreSQL..."))
		st, err = postgres.New(ctx, GetDatabaseURL())
	}
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer st.Close()
	fmt.Println(successStyle.Render("  Connected"))

	// Create server
	srv, err := web.NewServer(st, devMode)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      srv,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		fmt.Println()
		fmt.Println(boxStyle.Render(fmt.Sprintf(`%s

%s %s
%s %s

%s`,
			titleStyle.Render("Lingo is running"),
			labelStyle.Render("App:"),
			urlStyle.Render(fmt.Sprintf("http://localhost:%d", port)),
			labelStyle.Render("API:"),
			urlStyle.Render(fmt.Sprintf("http://localhost:%d/api/v1", port)),
			subtitleStyle.Render("Press Ctrl+C to stop"),
		)))
		fmt.Println()

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Wait for shutdown signal or error
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case <-sigCh:
		fmt.Println()
		fmt.Println(infoStyle.Render("Shutting down..."))
	case <-ctx.Done():
		fmt.Println()
		fmt.Println(infoStyle.Render("Shutting down..."))
	}

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown error: %w", err)
	}

	fmt.Println(successStyle.Render("Server stopped gracefully"))
	return nil
}
