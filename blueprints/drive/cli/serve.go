package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/go-mizu/blueprints/drive/app/web"
)

func newServeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the HTTP server",
		RunE:  runServe,
	}
}

func runServe(cmd *cobra.Command, args []string) error {
	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}

	cfg := web.Config{
		Addr:    addr,
		DataDir: dataDir,
		Dev:     dev,
	}

	server, err := web.NewServer(cfg)
	if err != nil {
		return fmt.Errorf("create server: %w", err)
	}
	defer server.Close()

	fmt.Printf("Drive server starting on %s\n", addr)
	fmt.Printf("Data directory: %s\n", dataDir)

	return server.ListenAndServe()
}
