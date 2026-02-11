package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-mizu/mizu/blueprints/book/app/web"
	"github.com/go-mizu/mizu/blueprints/book/store/sqlite"
	"github.com/spf13/cobra"
)

func NewServe() *cobra.Command {
	var port int
	var devMode bool

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the Book Manager web server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServe(cmd.Context(), port, devMode)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 8080, "Port to listen on")
	cmd.Flags().BoolVar(&devMode, "dev", false, "Enable development mode")
	return cmd
}

func runServe(ctx context.Context, port int, devMode bool) error {
	fmt.Println(Banner())

	fmt.Println(infoStyle.Render("Opening SQLite database..."))
	store, err := sqlite.New(GetDatabasePath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer store.Close()

	if err := store.Ensure(ctx); err != nil {
		return fmt.Errorf("failed to ensure schema: %w", err)
	}

	// Seed default shelves
	if err := store.Shelf().SeedDefaults(ctx); err != nil {
		return fmt.Errorf("failed to seed defaults: %w", err)
	}
	fmt.Println(successStyle.Render("  Database ready"))

	srv, err := web.NewServer(store, devMode)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	httpServer := &http.Server{
		Addr:        fmt.Sprintf(":%d", port),
		Handler:     srv,
		ReadTimeout: 30 * time.Second,
		IdleTimeout: 120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		fmt.Println()
		fmt.Println(boxStyle.Render(fmt.Sprintf(`%s

%s %s
%s %s

%s`,
			titleStyle.Render("Book Manager is running"),
			labelStyle.Render("Dashboard:"),
			urlStyle.Render(fmt.Sprintf("http://localhost:%d", port)),
			labelStyle.Render("API:"),
			urlStyle.Render(fmt.Sprintf("http://localhost:%d/api/books", port)),
			subtitleStyle.Render("Press Ctrl+C to stop"),
		)))
		fmt.Println()

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case <-sigCh:
		fmt.Println(infoStyle.Render("\nShutting down..."))
	case <-ctx.Done():
		fmt.Println(infoStyle.Render("\nShutting down..."))
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return httpServer.Shutdown(shutdownCtx)
}
