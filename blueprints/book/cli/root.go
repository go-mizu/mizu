package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

var dataDir string
var databasePath string

func Execute(ctx context.Context) error {
	root := &cobra.Command{
		Use:   "book",
		Short: "Book - Personal library manager",
		Long: `Book is a full-featured personal library manager.

Features:
  - Search books via Open Library & Google Books
  - Personal bookshelves (Want to Read, Currently Reading, Read)
  - Ratings, reviews, and reading progress
  - Reading challenges and stats
  - Import/Export CSV
  - Browse by genre

Get started:
  book init      Initialize the database
  book serve     Start the web server
  book seed      Seed with sample data`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	home, _ := os.UserHomeDir()
	dataDir = filepath.Join(home, "data", "blueprints", "book")
	databasePath = filepath.Join(dataDir, "book.duckdb")

	root.Version = Version
	root.PersistentFlags().StringVar(&dataDir, "data", dataDir, "Data directory")
	root.PersistentFlags().StringVar(&databasePath, "database", databasePath, "Database path")

	root.AddCommand(NewServe())
	root.AddCommand(NewInit())
	root.AddCommand(NewSeed())
	root.AddCommand(NewSearch())
	root.AddCommand(NewFetch())
	root.AddCommand(NewShelf())
	root.AddCommand(NewReview())
	root.AddCommand(NewImportExport())
	root.AddCommand(NewStats())
	root.AddCommand(NewChallenge())
	root.AddCommand(NewGoodreads())

	if err := root.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render("[ERROR] "+err.Error()))
		return err
	}
	return nil
}

func GetDataDir() string      { return dataDir }
func GetDatabasePath() string { return databasePath }
