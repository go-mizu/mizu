package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/go-mizu/mizu/blueprints/forum/app/web"
	"github.com/go-mizu/mizu/blueprints/forum/store/duckdb"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the forum server",
	Long:  `Starts the HTTP server for the forum application.`,
	RunE:  runServe,
}

func runServe(cmd *cobra.Command, args []string) error {
	// Open database
	store, err := duckdb.Open(dataDir)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer store.Close()

	// Create server
	srv, err := web.NewServer(store, web.ServerConfig{
		Addr: addr,
		Dev:  dev,
	})
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("\nShutting down...")
		cancel()
	}()

	// Start server
	fmt.Printf("Forum server starting on %s\n", addr)
	if dev {
		fmt.Println("Running in development mode")
	}

	return srv.Start(ctx)
}
