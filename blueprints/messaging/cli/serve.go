package cli

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/go-mizu/blueprints/messaging/app/web"
)

// NewServe creates the serve command.
func NewServe() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the web server",
		Long: `Starts the HTTP server for the messaging application.

The server provides the REST API, WebSocket connections, and HTML pages.`,
		RunE: runServe,
	}
}

func runServe(cmd *cobra.Command, args []string) error {
	ui := NewUI()

	ui.Header(iconServer, "Starting Messaging Server")
	ui.Blank()

	cfg := web.Config{
		Addr:    addr,
		DataDir: dataDir,
		Dev:     dev,
	}

	ui.StartSpinner("Initializing server...")
	start := time.Now()

	server, err := web.New(cfg)
	if err != nil {
		ui.StopSpinnerError("Failed to create server")
		return err
	}
	defer server.Close()

	ui.StopSpinner("Server initialized", time.Since(start))

	ui.Summary([][2]string{
		{"Address", addr},
		{"Data Dir", dataDir},
		{"Mode", modeString(dev)},
	})

	ui.Blank()
	ui.Hint("Press Ctrl+C to stop the server")
	ui.Blank()

	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		ui.Blank()
		ui.Warn("Shutting down...")
		cancel()
		server.Close()
	}()

	ui.Step("Listening on " + addr)

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Run()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return nil
	}
}

func modeString(dev bool) string {
	if dev {
		return "development"
	}
	return "production"
}
