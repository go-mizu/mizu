package cli

import (
	"fmt"

	"github.com/go-mizu/mizu/blueprints/book/pkg/openlibrary"
	"github.com/go-mizu/mizu/blueprints/book/store/sqlite"
	"github.com/spf13/cobra"
)

func NewFetch() *cobra.Command {
	return &cobra.Command{
		Use:   "fetch <isbn|ol_key>",
		Short: "Fetch a book from Open Library and add to database",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			key := args[0]

			store, err := sqlite.New(GetDatabasePath())
			if err != nil {
				return err
			}
			defer store.Close()
			if err := store.Ensure(ctx); err != nil {
				return err
			}

			client := openlibrary.NewClient()

			// Try as ISBN first, then as search query
			results, err := client.Search(ctx, key, 1)
			if err != nil {
				return fmt.Errorf("fetch failed: %w", err)
			}
			if len(results) == 0 {
				return fmt.Errorf("no results for %q", key)
			}

			book := results[0]
			if existing, _ := store.Book().GetByOLKey(ctx, book.OLKey); existing != nil {
				fmt.Println(infoStyle.Render("Book already in database: " + existing.Title))
				return nil
			}

			if err := store.Book().Create(ctx, &book); err != nil {
				return err
			}

			fmt.Println(successStyle.Render("Added: " + book.Title))
			fmt.Printf("  Author: %s\n", book.AuthorNames)
			if book.ISBN13 != "" {
				fmt.Printf("  ISBN:   %s\n", book.ISBN13)
			}
			if book.PageCount > 0 {
				fmt.Printf("  Pages:  %d\n", book.PageCount)
			}
			return nil
		},
	}
}
