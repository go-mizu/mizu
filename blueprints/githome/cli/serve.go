package cli

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/go-mizu/blueprints/githome/app/web"
	"github.com/spf13/cobra"
)

func newServeCmd() *cobra.Command {
	var (
		addr string
		dev  bool
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the GitHome server",
		Long: `Start the GitHome HTTP server.

The server provides:
  - Web UI for browsing repositories, issues, and pull requests
  - GitHub-compatible REST API at /api/v3/*

Press Ctrl+C to gracefully shut down the server.`,
		Example: `  githome serve
  githome serve --addr :3000
  githome serve --dev`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			level := slog.LevelInfo
			if dev {
				level = slog.LevelDebug
			}

			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level: level,
			}))
			slog.SetDefault(logger)

			cfg := web.Config{
				Addr:     addr,
				DataDir:  dataDir,
				ReposDir: reposDir,
				Dev:      dev,
			}

			srv, err := web.New(cfg)
			if err != nil {
				return fmt.Errorf("create server: %w", err)
			}
			defer srv.Close()

			url := fmt.Sprintf("http://localhost%s", addr)
			slog.Info("server started", "url", url, "dev", dev)
			return srv.RunContext(ctx)
		},
	}

	cmd.Flags().StringVarP(&addr, "addr", "a", ":8080", "Listen address (host:port)")
	cmd.Flags().BoolVarP(&dev, "dev", "d", false, "Enable development mode (verbose logging)")

	return cmd
}
