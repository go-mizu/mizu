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
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelDebug,
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

			slog.Info("starting GitHome server", "addr", addr)
			return srv.Run()
		},
	}

	cmd.Flags().StringVarP(&addr, "addr", "a", ":8080", "Listen address")
	cmd.Flags().BoolVar(&dev, "dev", false, "Development mode")

	return cmd
}
