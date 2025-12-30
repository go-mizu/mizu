package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-mizu/blueprints/cms/app/web"
	"github.com/spf13/cobra"
)

// NewServe creates the serve command.
func NewServe() *cobra.Command {
	var (
		port   int
		dbPath string
		secret string
		dev    bool
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the CMS server",
		Long:  "Start the Payload CMS compatible REST API server.",
		RunE: func(cmd *cobra.Command, args []string) error {
			srv, err := web.New(web.Config{
				Port:   port,
				DBPath: dbPath,
				Secret: secret,
				Dev:    dev,
			})
			if err != nil {
				return fmt.Errorf("create server: %w", err)
			}

			// Handle graceful shutdown
			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

			go func() {
				<-sigCh
				fmt.Println("\nShutting down...")
				srv.Shutdown(ctx)
				cancel()
			}()

			return srv.Run()
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 3000, "Server port")
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path (default: $HOME/data/blueprint/cms/cms.db)")
	cmd.Flags().StringVar(&secret, "secret", "", "JWT secret (required for production)")
	cmd.Flags().BoolVar(&dev, "dev", false, "Development mode")

	return cmd
}
