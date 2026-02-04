package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-mizu/mizu/blueprints/email/app/web"
	"github.com/go-mizu/mizu/blueprints/email/pkg/email"
	"github.com/go-mizu/mizu/blueprints/email/pkg/email/resend"
	"github.com/go-mizu/mizu/blueprints/email/store/sqlite"
	"github.com/spf13/cobra"
)

// NewServe creates the serve command
func NewServe() *cobra.Command {
	var (
		port       int
		devMode    bool
		driverName string
		fromAddr   string
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the Email server",
		Long: `Start the Email server including:
  - Email API (list, read, compose, search)
  - Label management
  - Contact autocomplete
  - Settings management
  - Dashboard UI

The server runs on port 8080 by default.

Email drivers:
  noop    - Accept sends without delivering (default)
  resend  - Send via Resend API (requires RESEND_API_KEY)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServe(cmd.Context(), port, devMode, driverName, fromAddr)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 8080, "Port to listen on")
	cmd.Flags().BoolVar(&devMode, "dev", false, "Enable development mode")
	cmd.Flags().StringVar(&driverName, "driver", envOrDefault("EMAIL_DRIVER", "noop"), "Email driver (noop, resend)")
	cmd.Flags().StringVar(&fromAddr, "from", os.Getenv("EMAIL_FROM"), "Default from address for outbound email")

	return cmd
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func runServe(ctx context.Context, port int, devMode bool, driverName, fromAddr string) error {
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

	// Initialize email driver
	var emailDriver email.Driver
	switch driverName {
	case "resend":
		d, err := resend.New(resend.Config{})
		if err != nil {
			return fmt.Errorf("failed to create resend driver: %w", err)
		}
		emailDriver = d
		fmt.Println(successStyle.Render("  Email driver: resend"))
	default:
		emailDriver = email.Noop()
		fmt.Println(infoStyle.Render("  Email driver: noop (emails won't be delivered)"))
	}

	// Create server
	srv, err := web.NewServer(store, emailDriver, fromAddr, devMode)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      srv,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
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

%s`,
			titleStyle.Render("Email is running"),
			labelStyle.Render("Dashboard:"),
			urlStyle.Render(fmt.Sprintf("http://localhost:%d", port)),
			labelStyle.Render("Email API:"),
			urlStyle.Render(fmt.Sprintf("http://localhost:%d/api/emails", port)),
			labelStyle.Render("Search API:"),
			urlStyle.Render(fmt.Sprintf("http://localhost:%d/api/emails/search", port)),
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
