package cli

import (
	"fmt"

	"github.com/go-mizu/mizu/blueprints/book/pkg/openlibrary"
	"github.com/go-mizu/mizu/blueprints/book/store/sqlite"
	"github.com/spf13/cobra"
)

func NewSeed() *cobra.Command {
	return &cobra.Command{
		Use:   "seed",
		Short: "Seed database with popular books from Open Library",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(Banner())
			ctx := cmd.Context()

			store, err := sqlite.New(GetDatabasePath())
			if err != nil {
				return err
			}
			defer store.Close()

			if err := store.Ensure(ctx); err != nil {
				return err
			}
			if err := store.Shelf().SeedDefaults(ctx); err != nil {
				return err
			}

			client := openlibrary.NewClient()

			// Seed popular books by searching various genres
			queries := []string{
				"harry potter", "lord of the rings", "1984 orwell",
				"pride and prejudice", "the great gatsby", "to kill a mockingbird",
				"the hobbit", "dune herbert", "brave new world",
				"fahrenheit 451", "the catcher in the rye", "animal farm",
				"crime and punishment", "jane eyre", "wuthering heights",
				"the alchemist coelho", "sapiens harari", "educated westover",
				"atomic habits", "thinking fast and slow",
			}

			total := 0
			for _, q := range queries {
				fmt.Printf("  Searching: %s... ", q)
				results, err := client.Search(ctx, q, 1)
				if err != nil {
					fmt.Println(errorStyle.Render("error: " + err.Error()))
					continue
				}
				if len(results) == 0 {
					fmt.Println(dimStyle.Render("no results"))
					continue
				}

				book := results[0]
				// Check if already exists
				if existing, _ := store.Book().GetByOLKey(ctx, book.OLKey); existing != nil {
					fmt.Println(dimStyle.Render("already exists"))
					continue
				}

				if err := store.Book().Create(ctx, &book); err != nil {
					fmt.Println(errorStyle.Render("error: " + err.Error()))
					continue
				}
				total++
				fmt.Println(successStyle.Render(fmt.Sprintf("added: %s", book.Title)))
			}

			fmt.Println()
			fmt.Println(successStyle.Render(fmt.Sprintf("Seeded %d books", total)))
			return nil
		},
	}
}
