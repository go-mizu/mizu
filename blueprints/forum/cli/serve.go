package cli

import (
	"github.com/go-mizu/blueprints/forum/app/web"
	"github.com/spf13/cobra"
)

var (
	host string
	port int
	db   string
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the web server",
	Long:  "Start the forum web server",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := web.Default()
		cfg.Host = host
		cfg.Port = port
		cfg.DatabasePath = db

		server, err := web.New(cfg)
		if err != nil {
			return err
		}

		return server.Start()
	},
}

func init() {
	serveCmd.Flags().StringVar(&host, "host", "localhost", "Server host")
	serveCmd.Flags().IntVar(&port, "port", 8080, "Server port")
	serveCmd.Flags().StringVar(&db, "db", "forum.db", "Database path")
}
