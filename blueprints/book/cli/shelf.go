package cli

import (
	"fmt"
	"strings"

	"github.com/go-mizu/mizu/blueprints/book/store/sqlite"
	"github.com/go-mizu/mizu/blueprints/book/types"
	"github.com/spf13/cobra"
)

func NewShelf() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "shelf",
		Short: "Manage bookshelves",
	}

	cmd.AddCommand(newShelfList())
	cmd.AddCommand(newShelfAdd())
	cmd.AddCommand(newShelfBooks())
	cmd.AddCommand(newShelve())
	return cmd
}

func newShelfList() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all shelves",
		RunE: func(cmd *cobra.Command, args []string) error {
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

			shelves, err := store.Shelf().List(ctx)
			if err != nil {
				return err
			}

			fmt.Println(titleStyle.Render("Bookshelves"))
			fmt.Println()
			for _, sh := range shelves {
				exclusive := ""
				if sh.IsExclusive {
					exclusive = " (exclusive)"
				}
				fmt.Printf("  %s (%d books)%s\n", sh.Name, sh.BookCount, dimStyle.Render(exclusive))
			}
			return nil
		},
	}
}

func newShelfAdd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <name>",
		Short: "Create a custom shelf",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			store, err := sqlite.New(GetDatabasePath())
			if err != nil {
				return err
			}
			defer store.Close()
			if err := store.Ensure(ctx); err != nil {
				return err
			}

			name := args[0]
			slug := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
			shelf := &types.Shelf{Name: name, Slug: slug}
			if err := store.Shelf().Create(ctx, shelf); err != nil {
				return err
			}
			fmt.Println(successStyle.Render("Created shelf: " + name))
			return nil
		},
	}
}

func newShelfBooks() *cobra.Command {
	return &cobra.Command{
		Use:   "books <shelf-name>",
		Short: "List books in a shelf",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			store, err := sqlite.New(GetDatabasePath())
			if err != nil {
				return err
			}
			defer store.Close()
			if err := store.Ensure(ctx); err != nil {
				return err
			}

			slug := strings.ToLower(strings.ReplaceAll(args[0], " ", "-"))
			shelf, err := store.Shelf().GetBySlug(ctx, slug)
			if err != nil {
				return err
			}
			if shelf == nil {
				return fmt.Errorf("shelf not found: %s", args[0])
			}

			books, total, err := store.Shelf().GetBooks(ctx, shelf.ID, "date_added", 1, 50)
			if err != nil {
				return err
			}

			fmt.Println(titleStyle.Render(fmt.Sprintf("%s (%d books)", shelf.Name, total)))
			fmt.Println()
			for _, sb := range books {
				if sb.Book != nil {
					fmt.Printf("  • %s by %s\n", sb.Book.Title, sb.Book.AuthorNames)
				}
			}
			if len(books) == 0 {
				fmt.Println(dimStyle.Render("  No books in this shelf"))
			}
			return nil
		},
	}
}

func newShelve() *cobra.Command {
	return &cobra.Command{
		Use:   "shelve <book-id> <shelf-name>",
		Short: "Add a book to a shelf",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
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

			var bookID int64
			fmt.Sscanf(args[0], "%d", &bookID)

			slug := strings.ToLower(strings.ReplaceAll(args[1], " ", "-"))
			shelf, err := store.Shelf().GetBySlug(ctx, slug)
			if err != nil {
				return err
			}
			if shelf == nil {
				return fmt.Errorf("shelf not found: %s", args[1])
			}

			if err := store.Shelf().AddBook(ctx, shelf.ID, bookID); err != nil {
				return err
			}

			book, _ := store.Book().Get(ctx, bookID)
			title := fmt.Sprintf("#%d", bookID)
			if book != nil {
				title = book.Title
			}
			fmt.Println(successStyle.Render(fmt.Sprintf("Added %q to %q", title, shelf.Name)))
			return nil
		},
	}
}
