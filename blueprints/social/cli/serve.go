package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/go-mizu/blueprints/social/app/web"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the web server",
	Long:  `Start the Social web server.`,
	RunE:  runServe,
}

func runServe(cmd *cobra.Command, args []string) error {
	ui := NewUI()
	ui.Header("Social", Version)

	cfg := web.Config{
		Addr:    addr,
		DataDir: dataDir,
		Dev:     dev,
	}

	ui.Info("Starting server...")
	ui.Item("Address", cfg.Addr)
	ui.Item("Data", cfg.DataDir)
	ui.Item("Mode", func() string {
		if cfg.Dev {
			return "development"
		}
		return "production"
	}())

	server, err := web.New(cfg)
	if err != nil {
		return fmt.Errorf("create server: %w", err)
	}
	defer server.Close()

	// Handle shutdown
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		ui.Info("Shutting down...")
		cancel()
		server.Close()
	}()

	ui.Success("Server started")
	ui.Item("URL", fmt.Sprintf("http://localhost%s", cfg.Addr))

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
