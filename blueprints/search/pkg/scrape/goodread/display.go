package goodread

import (
	"fmt"
)

// PrintStats prints a summary of the goodread database stats.
func PrintStats(db *DB, stateDB *State) error {
	stats, err := db.GetStats()
	if err != nil {
		return err
	}

	fmt.Println("── Goodreads Database Stats ──")
	fmt.Printf("  Books:    %d\n", stats.Books)
	fmt.Printf("  Authors:  %d\n", stats.Authors)
	fmt.Printf("  Series:   %d\n", stats.Series)
	fmt.Printf("  Lists:    %d\n", stats.Lists)
	fmt.Printf("  Reviews:  %d\n", stats.Reviews)
	fmt.Printf("  Quotes:   %d\n", stats.Quotes)
	fmt.Printf("  Users:    %d\n", stats.Users)
	fmt.Printf("  Genres:   %d\n", stats.Genres)
	fmt.Printf("  Shelves:  %d\n", stats.Shelves)
	fmt.Printf("  DB size:  %.1f MB\n", float64(stats.DBSize)/(1024*1024))
	fmt.Printf("  DB path:  %s\n", db.Path())

	if stateDB != nil {
		pending, inProgress, done, failed := stateDB.QueueStats()
		fmt.Println("── Queue ──")
		fmt.Printf("  Pending:     %d\n", pending)
		fmt.Printf("  In progress: %d\n", inProgress)
		fmt.Printf("  Done:        %d\n", done)
		fmt.Printf("  Failed:      %d\n", failed)
	}

	// Show recent books
	books, err := db.RecentBooks(5)
	if err == nil && len(books) > 0 {
		fmt.Println("── Recent Books ──")
		for _, b := range books {
			fmt.Printf("  [%.2f ★  %d ratings] %s — %s\n",
				b.AvgRating, b.RatingsCount, b.Title, b.AuthorName)
		}
	}

	return nil
}

// PrintCrawlProgress prints a one-line crawl progress update.
func PrintCrawlProgress(state *CrawlState) {
	inflight := len(state.InFlight)
	fmt.Printf("\r  done=%-6d  pending=%-6d  failed=%-4d  in-flight=%-3d  rps=%.1f    ",
		state.Done, state.Pending, state.Failed, inflight, state.RPS)
}
