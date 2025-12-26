package cli

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/go-mizu/blueprints/kanban/app/web"
)

// NewServe creates the serve command
func NewServe() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the web server",
		Long: `Start the Kanban web server.

The server provides:
  - Web UI for project management
  - REST API for programmatic access
  - Real-time updates

Examples:
  kanban serve                    # Start on default port 8080
  kanban serve --addr :3000       # Start on port 3000
  kanban serve --dev              # Enable development mode`,
		RunE: runServe,
	}

	cmd.Flags().StringP("addr", "a", ":8080", "Address to listen on")

	return cmd
}

func runServe(cmd *cobra.Command, args []string) error {
	addr, _ := cmd.Flags().GetString("addr")
	dev, _ := cmd.Root().PersistentFlags().GetBool("dev")

	Blank()
	Header("", "Kanban Server")
	Blank()

	Summary(
		"Address", addr,
		"Data", dataDir,
		"Mode", modeString(dev),
		"Version", Version,
	)
	Blank()

	// Create server
	srv, err := web.New(web.Config{
		Addr:    addr,
		DataDir: dataDir,
		Dev:     dev,
	})
	if err != nil {
		Error(fmt.Sprintf("Failed to create server: %v", err))
		return err
	}
	defer srv.Close()

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		Step("", fmt.Sprintf("Server listening on http://localhost%s", addr))
		errCh <- srv.Run()
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		Error(fmt.Sprintf("Server error: %v", err))
		return err
	case <-quit:
		Blank()
		Step("", "Shutting down...")
		Success("Server stopped")
	}

	return nil
}
