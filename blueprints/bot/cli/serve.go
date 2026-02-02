package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-mizu/mizu/blueprints/bot/app/web"
	"github.com/go-mizu/mizu/blueprints/bot/store/sqlite"
	"github.com/spf13/cobra"
)

// NewServe creates the serve command
func NewServe() *cobra.Command {
	var (
		port    int
		devMode bool
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the Bot gateway server",
		Long: `Start the Bot gateway server including:
  - Gateway API (agents, channels, sessions, messages)
  - Message routing engine
  - Webhook endpoints
  - Web dashboard

The server runs on port 18789 by default.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServe(cmd.Context(), port, devMode)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 18789, "Port to listen on")
	cmd.Flags().BoolVar(&devMode, "dev", false, "Enable development mode")

	return cmd
}

func runServe(ctx context.Context, port int, devMode bool) error {
	fmt.Println(Banner())

	// Connect to database
	fmt.Println(infoStyle.Render("Opening SQLite database..."))
	store, err := sqlite.New(GetDatabasePath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer store.Close()

	// Ensure schema is up to date
	if err := store.Ensure(ctx); err != nil {
		return fmt.Errorf("failed to ensure schema: %w", err)
	}
	fmt.Println(successStyle.Render("  Connected"))

	// Create server
	srv := web.NewServer(store, devMode)
	defer srv.Close()

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      srv,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 0, // Disabled for SSE streams
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		fmt.Println()
		fmt.Println(boxStyle.Render(fmt.Sprintf(`%s

%s %s
%s %s
%s %s
%s %s

%s`,
			titleStyle.Render("Bot Gateway is running"),
			labelStyle.Render("Dashboard:"),
			urlStyle.Render(fmt.Sprintf("http://localhost:%d", port)),
			labelStyle.Render("Gateway API:"),
			urlStyle.Render(fmt.Sprintf("http://localhost:%d/api/status", port)),
			labelStyle.Render("Agents API:"),
			urlStyle.Render(fmt.Sprintf("http://localhost:%d/api/agents", port)),
			labelStyle.Render("Webhook:"),
			urlStyle.Render(fmt.Sprintf("http://localhost:%d/api/webhook/:id", port)),
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

	fmt.Println(successStyle.Render("Gateway stopped gracefully"))
	return nil
}
