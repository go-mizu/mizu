package cli

import (
	"fmt"
	"strconv"
	"time"

	"github.com/go-mizu/mizu/blueprints/book/store/factory"
	"github.com/go-mizu/mizu/blueprints/book/types"
	"github.com/spf13/cobra"
)

func NewReview() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rate <book-id> <1-5>",
		Short: "Rate a book (1-5 stars)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			store, err := factory.Open(ctx, GetDatabasePath())
			if err != nil {
				return err
			}
			defer store.Close()
			if err := store.Ensure(ctx); err != nil {
				return err
			}

			bookID, _ := strconv.ParseInt(args[0], 10, 64)
			rating, _ := strconv.Atoi(args[1])
			if rating < 1 || rating > 5 {
				return fmt.Errorf("rating must be 1-5")
			}

			book, err := store.Book().Get(ctx, bookID)
			if err != nil {
				return err
			}
			if book == nil {
				return fmt.Errorf("book not found: %d", bookID)
			}

			// Check for existing review
			now := time.Now()
			existing, _ := store.Review().GetUserReview(ctx, bookID)
			if existing != nil {
				existing.Rating = rating
				if existing.FinishedAt == nil {
					existing.FinishedAt = &now
				}
				if err := store.Review().Update(ctx, existing); err != nil {
					return err
				}
				fmt.Printf("Updated rating for %q: %s\n", book.Title, Stars(rating))
			} else {
				review := &types.Review{BookID: bookID, Rating: rating, FinishedAt: &now}
				if err := store.Review().Create(ctx, review); err != nil {
					return err
				}
				fmt.Printf("Rated %q: %s\n", book.Title, Stars(rating))
			}
			return nil
		},
	}
	return cmd
}
