package cli

import (
	"fmt"

	"github.com/go-mizu/mizu/blueprints/book/store/factory"
	"github.com/spf13/cobra"
)

func NewInit() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize the database",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(Banner())
			fmt.Println(infoStyle.Render("Initializing database at " + GetDatabasePath()))
			ctx := cmd.Context()

			store, err := factory.Open(ctx, GetDatabasePath())
			if err != nil {
				return err
			}
			defer store.Close()

			if err := store.Ensure(cmd.Context()); err != nil {
				return fmt.Errorf("schema creation failed: %w", err)
			}

			if err := store.Shelf().SeedDefaults(cmd.Context()); err != nil {
				return fmt.Errorf("seed defaults failed: %w", err)
			}

			fmt.Println(successStyle.Render("Database initialized successfully"))
			return nil
		},
	}
}
