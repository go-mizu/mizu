package cli

import (
	"fmt"

	"github.com/go-mizu/blueprints/cms/app/web"
	"github.com/spf13/cobra"
)

// NewInit creates the init command.
func NewInit() *cobra.Command {
	var dbPath string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize the database",
		Long:  "Create the database and initialize the schema.",
		RunE: func(cmd *cobra.Command, args []string) error {
			srv, err := web.New(web.Config{
				DBPath: dbPath,
			})
			if err != nil {
				return fmt.Errorf("create server: %w", err)
			}
			defer srv.Close()

			fmt.Println("Database initialized successfully!")
			return nil
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "", "Database path")

	return cmd
}
