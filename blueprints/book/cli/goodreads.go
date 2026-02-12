package cli

import (
	"fmt"

	"github.com/go-mizu/mizu/blueprints/book/pkg/goodreads"
	"github.com/go-mizu/mizu/blueprints/book/store/factory"
	"github.com/go-mizu/mizu/blueprints/book/types"
	"github.com/spf13/cobra"
)

func NewGoodreads() *cobra.Command {
	return &cobra.Command{
		Use:   "goodreads <url_or_id>",
		Short: "Import a book from Goodreads",
		Long: `Fetch book data from Goodreads and import into the local database.

Accepts:
  - Goodreads URL: https://www.goodreads.com/book/show/112247
  - Goodreads ID:  112247`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			grID := goodreads.ParseGoodreadsURL(args[0])
			if grID == "" {
				return fmt.Errorf("invalid Goodreads URL or ID: %s", args[0])
			}

			store, err := factory.Open(ctx, GetDatabasePath())
			if err != nil {
				return err
			}
			defer store.Close()
			if err := store.Ensure(ctx); err != nil {
				return err
			}

			// Check if already imported
			if existing, _ := store.Book().GetByGoodreadsID(ctx, grID); existing != nil {
				fmt.Println(infoStyle.Render("Already imported: " + existing.Title))
				printBookDetail(existing)
				return nil
			}

			fmt.Println(dimStyle.Render("Fetching from Goodreads..."))
			client := goodreads.NewClient()
			grBook, err := client.GetBook(ctx, grID)
			if err != nil {
				return fmt.Errorf("fetch failed: %w", err)
			}

			book := grToBook(grBook)
			if err := store.Book().Create(ctx, &book); err != nil {
				return fmt.Errorf("save failed: %w", err)
			}

			// Import quotes
			for _, q := range grBook.Quotes {
				quote := types.Quote{
					BookID:     book.ID,
					AuthorName: q.AuthorName,
					Text:       q.Text,
					LikesCount: q.LikesCount,
				}
				store.Quote().Create(ctx, &quote) //nolint:errcheck
			}

			fmt.Println(successStyle.Render("Imported: " + book.Title))
			printBookDetail(&book)

			if len(grBook.Reviews) > 0 {
				fmt.Printf("\n  %s\n", subtitleStyle.Render(fmt.Sprintf("Reviews (%d):", grBook.ReviewsCount)))
				for i, r := range grBook.Reviews {
					if i >= 5 {
						fmt.Printf("  %s\n", dimStyle.Render(fmt.Sprintf("... and %d more", len(grBook.Reviews)-5)))
						break
					}
					name := r.ReviewerName
					if name == "" {
						name = "Anonymous"
					}
					fmt.Printf("  %s %s  %s\n", Stars(r.Rating), titleStyle.Render(name), dimStyle.Render(r.Date))
					if r.Text != "" {
						text := r.Text
						if len(text) > 120 {
							text = text[:120] + "..."
						}
						fmt.Printf("    %s\n", text)
					}
				}
			}

			return nil
		},
	}
}

func grToBook(gr *goodreads.GoodreadsBook) types.Book {
	return types.Book{
		GoodreadsID:      gr.GoodreadsID,
		Title:            gr.Title,
		AuthorNames:      gr.AuthorName,
		Description:      gr.Description,
		ISBN10:           gr.ISBN,
		ISBN13:           gr.ISBN13,
		ASIN:             gr.ASIN,
		PageCount:        gr.PageCount,
		Format:           gr.Format,
		Publisher:        gr.Publisher,
		PublishDate:      gr.PublishDate,
		FirstPublished:   gr.FirstPublished,
		Language:         gr.Language,
		CoverURL:         gr.CoverURL,
		Series:           gr.Series,
		AverageRating:    gr.AverageRating,
		RatingsCount:     gr.RatingsCount,
		ReviewsCount:     gr.ReviewsCount,
		CurrentlyReading: gr.CurrentlyReading,
		WantToRead:       gr.WantToRead,
		RatingDist:       gr.RatingDist,
		Subjects:         gr.Genres,
	}
}

func printBookDetail(book *types.Book) {
	fmt.Printf("  %s %s\n", labelStyle.Render("Author:"), book.AuthorNames)
	if book.ISBN13 != "" {
		fmt.Printf("  %s %s\n", labelStyle.Render("ISBN:"), book.ISBN13)
	}
	if book.PageCount > 0 {
		fmt.Printf("  %s %d pages, %s\n", labelStyle.Render("Format:"), book.PageCount, book.Format)
	}
	if book.Publisher != "" {
		fmt.Printf("  %s %s\n", labelStyle.Render("Publisher:"), book.Publisher)
	}
	if book.Series != "" {
		fmt.Printf("  %s %s\n", labelStyle.Render("Series:"), book.Series)
	}
	fmt.Printf("  %s %s %.2f (%d ratings, %d reviews)\n",
		labelStyle.Render("Rating:"),
		Stars(int(book.AverageRating+0.5)),
		book.AverageRating, book.RatingsCount, book.ReviewsCount)
	if book.CurrentlyReading > 0 || book.WantToRead > 0 {
		fmt.Printf("  %s %d currently reading, %d want to read\n",
			labelStyle.Render("Activity:"), book.CurrentlyReading, book.WantToRead)
	}
	if book.RatingDist != [5]int{} {
		fmt.Printf("  %s 5★ %d · 4★ %d · 3★ %d · 2★ %d · 1★ %d\n",
			labelStyle.Render("Dist:"),
			book.RatingDist[0], book.RatingDist[1], book.RatingDist[2],
			book.RatingDist[3], book.RatingDist[4])
	}
}
