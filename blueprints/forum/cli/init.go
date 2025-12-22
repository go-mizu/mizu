package cli

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/duckdb/duckdb-go/v2"

	"github.com/go-mizu/blueprints/forum/store/duckdb"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the database",
	Long:  "Create the database and run migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		dbPath := cmd.Flag("db").Value.String()

		fmt.Printf("Initializing database at %s...\n", dbPath)

		db, err := sql.Open("duckdb", dbPath)
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer db.Close()

		store, err := duckdb.New(db)
		if err != nil {
			return fmt.Errorf("create store: %w", err)
		}

		if err := store.Ensure(context.Background()); err != nil {
			return fmt.Errorf("ensure schema: %w", err)
		}

		fmt.Println("Database initialized successfully!")
		return nil
	},
}

func init() {
	initCmd.Flags().String("db", "forum.db", "Database path")
}
