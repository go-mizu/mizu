package cli

import (
	"fmt"

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
			ui := NewUI()

			ui.Header(iconServer, "Microblog Server")
			ui.Blank()

			cfg := web.Config{
				Addr:    addr,
				DataDir: dataDir,
				Dev:     dev,
			}

			ui.Info("Address", fmt.Sprintf("http://localhost%s", addr))
			ui.Info("Data", dataDir)
			if dev {
				ui.Info("Mode", warnStyle.Render("development"))
			} else {
				ui.Info("Mode", "production")
			}
			ui.Blank()

			ui.StartSpinner("Starting server...")

			server, err := web.New(cfg)
			if err != nil {
				ui.StopSpinnerError("Failed to start server")
				return err
			}
			defer server.Close()

			ui.StopSpinnerError("") // Clear spinner line
			ui.Success(fmt.Sprintf("Server running at http://localhost%s", addr))
			ui.Hint("Press Ctrl+C to stop")
			ui.Blank()

			return server.Run()
		},
	}

	cmd.Flags().StringVar(&addr, "addr", ":8080", "HTTP listen address")
	cmd.Flags().BoolVar(&dev, "dev", false, "Enable development mode")

	return cmd
}
