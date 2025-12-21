package cli

import (
	"log"

	"github.com/spf13/cobra"

	"github.com/go-mizu/blueprints/microblog/app/web"
)

// NewServe creates the serve command.
func NewServe() *cobra.Command {
	var addr string
	var dev bool

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the web server",
		Long:  `Start the microblog web server.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := web.Config{
				Addr:    addr,
				DataDir: dataDir,
				Dev:     dev,
			}

			server, err := web.New(cfg)
			if err != nil {
				return err
			}
			defer server.Close()

			log.Printf("Server starting on http://localhost%s", addr)
			return server.Run()
		},
	}

	cmd.Flags().StringVar(&addr, "addr", ":8080", "HTTP listen address")
	cmd.Flags().BoolVar(&dev, "dev", false, "Enable development mode")

	return cmd
}
