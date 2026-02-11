package cli

import (
	"fmt"
	"time"

	"github.com/go-mizu/mizu/blueprints/book/store/sqlite"
	"github.com/spf13/cobra"
)

func NewStats() *cobra.Command {
	var year int
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Show reading statistics",
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

			if year == 0 {
				year = time.Now().Year()
			}

			stats, err := store.Stats().GetStats(ctx, year)
			if err != nil {
				return err
			}

			fmt.Println(titleStyle.Render(fmt.Sprintf("Reading Stats %d", year)))
			fmt.Println()
			fmt.Printf("  Books Read:     %d\n", stats.TotalBooks)
			fmt.Printf("  Pages Read:     %d\n", stats.TotalPages)
			if stats.AverageRating > 0 {
				fmt.Printf("  Avg Rating:     %.1f %s\n", stats.AverageRating, Stars(int(stats.AverageRating+0.5)))
			}
			fmt.Println()

			if len(stats.GenreBreakdown) > 0 {
				fmt.Println(titleStyle.Render("  Genre Breakdown:"))
				for genre, count := range stats.GenreBreakdown {
					fmt.Printf("    %-20s %d books\n", genre, count)
				}
				fmt.Println()
			}

			if len(stats.RatingDist) > 0 {
				fmt.Println(titleStyle.Render("  Rating Distribution:"))
				for r := 5; r >= 1; r-- {
					count := stats.RatingDist[r]
					bar := ""
					for i := 0; i < count; i++ {
						bar += "█"
					}
					fmt.Printf("    %s  %s %d\n", Stars(r), bar, count)
				}
			}

			if stats.ShortestBook != nil {
				fmt.Printf("\n  Shortest:  %s (%dp)\n", stats.ShortestBook.Title, stats.ShortestBook.PageCount)
			}
			if stats.LongestBook != nil {
				fmt.Printf("  Longest:   %s (%dp)\n", stats.LongestBook.Title, stats.LongestBook.PageCount)
			}

			return nil
		},
	}
	cmd.Flags().IntVar(&year, "year", 0, "Year to show stats for (default: current)")
	return cmd
}
