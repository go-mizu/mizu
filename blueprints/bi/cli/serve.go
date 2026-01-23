package cli

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/go-mizu/blueprints/bi/app/web"
)

// NewServe creates the serve command
func NewServe() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the BI server",
		Long: `Start the BI server including:
  - Dashboard UI on :8080
  - REST API on :8080/api

Examples:
  bi serve                    # Start with defaults
  bi serve --addr :9000       # Custom port
  bi serve --dev              # Enable development mode`,
		RunE: runServe,
	}

	cmd.Flags().StringP("addr", "a", ":8080", "Server address")

	return cmd
}

func runServe(cmd *cobra.Command, args []string) error {
	addr, _ := cmd.Flags().GetString("addr")
	dev, _ := cmd.Root().PersistentFlags().GetBool("dev")

	Blank()
	Header("", "BI Server")
	Blank()

	Summary(
		"Dashboard", addr,
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
		Step("", fmt.Sprintf("Dashboard: http://localhost%s", addr))
		Blank()
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
