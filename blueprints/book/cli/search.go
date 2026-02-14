package cli

import (
	"fmt"

	"github.com/go-mizu/mizu/blueprints/book/pkg/openlibrary"
	"github.com/spf13/cobra"
)

func NewSearch() *cobra.Command {
	return &cobra.Command{
		Use:   "search <query>",
		Short: "Search books from Open Library",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			query := args[0]
			for i := 1; i < len(args); i++ {
				query += " " + args[i]
			}

			client := openlibrary.NewClient()
			results, err := client.Search(ctx, query, 10)
			if err != nil {
				return fmt.Errorf("search failed: %w", err)
			}

			fmt.Println(titleStyle.Render(fmt.Sprintf("Search: %q (%d results)", query, len(results))))
			fmt.Println()

			for i, book := range results {
				rating := ""
				if book.AverageRating > 0 {
					rating = fmt.Sprintf(" %.1f★", book.AverageRating)
				}
				pages := ""
				if book.PageCount > 0 {
					pages = fmt.Sprintf(" · %dp", book.PageCount)
				}
				year := ""
				if book.PublishYear > 0 {
					year = fmt.Sprintf(" (%d)", book.PublishYear)
				}
				isbn := ""
				if book.ISBN13 != "" {
					isbn = " · " + book.ISBN13
				}

				fmt.Printf("  %d. %s%s\n", i+1, titleStyle.Render(book.Title), year)
				fmt.Printf("     %s%s%s%s\n", book.AuthorNames, rating, pages, isbn)
				if book.OLKey != "" {
					fmt.Printf("     %s\n", dimStyle.Render(book.OLKey))
				}
				fmt.Println()
			}
			return nil
		},
	}
}
