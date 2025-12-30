package cli

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/go-mizu/mizu/blueprints/cms/app/web"
	"github.com/go-mizu/mizu/blueprints/cms/store/duckdb"
)

// NewServe creates the serve command.
func NewServe() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the CMS server",
		Long: `Starts the HTTP server for the CMS application.

The server provides:
  - WordPress REST API v2 at /wp-json/wp/v2/
  - XML-RPC API at /xmlrpc.php
  - Admin dashboard at /wp-admin/
  - Frontend theme rendering`,
		RunE: runServe,
	}
}

func runServe(cmd *cobra.Command, args []string) error {
	ui := NewUI()

	ui.Header(iconServer, "Starting CMS Server")
	ui.Blank()

	// Open database
	start := time.Now()
	ui.StartSpinner("Opening database...")

	store, err := duckdb.Open(dataDir)
	if err != nil {
		ui.StopSpinnerError("Failed to open database")
		return err
	}
	defer store.Close()

	ui.StopSpinner("Database ready", time.Since(start))

	// Create server
	ui.StartSpinner("Initializing server...")
	start = time.Now()

	srv, err := web.NewServer(store, web.ServerConfig{
		Addr:    addr,
		Dev:     dev,
		DataDir: dataDir,
	})
	if err != nil {
		ui.StopSpinnerError("Failed to create server")
		return err
	}

	ui.StopSpinner("Server initialized", time.Since(start))

	// Print configuration
	ui.Summary([][2]string{
		{"Address", addr},
		{"Data Dir", dataDir},
		{"Mode", modeString(dev)},
		{"REST API", addr + "/wp-json/wp/v2/"},
		{"Admin", addr + "/wp-admin/"},
	})

	ui.Blank()
	ui.Hint("Press Ctrl+C to stop the server")
	ui.Blank()

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		ui.Blank()
		ui.Warn("Shutting down...")
		cancel()
	}()

	// Start server
	ui.Step("Listening on " + addr)

	return srv.Start(ctx)
}

func modeString(dev bool) string {
	if dev {
		return "development"
	}
	return "production"
}
